package mobs

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// saveShopFn is the shop-persistence hook used by executeCraft and the
// salvage path. Tests override this variable to avoid disk I/O while
// still asserting that persistence is called after a craft or salvage.
var saveShopFn = func(zone string, mobId int, roomId int) error {
	return shops.SaveShop(zone, mobId, roomId)
}

// CraftResult describes the outcome of a crafter mob's tick, so the calling
// hook can emit room messages and world events without import cycles.
type CraftResult struct {
	Success      bool
	RecipeName   string
	OutputItemId int
	SkillMinimum int
	MobName      string
	Zone         string
	Salvaged     bool // true when the action was a salvage rather than a craft
	Restocked    bool // true when the supply cart delivered materials this tick
}

// RegisterMobShop seeds a ShopInventory for a mob that has a legacy shop or
// crafter materials. It merges:
//   - legacy Character.Shop items (each gets RestockQty=5, MaxStock=20)
//   - CrafterRestockMaterials (RestockQty=3, MaxStock=10)
//   - CrafterRecipeIds → KnownRecipes
//
// If neither shop items nor crafter materials are present, it does nothing.
// The mob's starting gold from the YAML is respected; a floor of 500 is
// applied so merchants always have meaningful purchasing power.
func RegisterMobShop(mob *Mob) {
	hasSaleItems := len(mob.Character.Shop) > 0
	hasCrafter := mob.Crafter && (len(mob.CrafterRestockMaterials) > 0 || len(mob.CrafterRecipeIds) > 0)

	if !hasSaleItems && !hasCrafter {
		return
	}

	// Seed gold from the mob's YAML (mob.Character.Gold). Apply a 500g
	// floor so any merchant has meaningful purchasing power even if a
	// content edit accidentally drops it to zero. Per spec section 3
	// (per-vendor audit), specialists set 1000g and generals set 5000g
	// in their YAMLs — this is what flows through to StartingGold.
	startingGold := mob.Character.Gold
	if startingGold < 500 {
		startingGold = 500
	}

	template := shops.ShopInventory{
		Gold:         startingGold,
		StartingGold: startingGold,
		KnownRecipes: append([]string{}, mob.CrafterRecipeIds...),
	}

	seen := map[int]bool{}

	// Helper: derive MaxStock from rarity tier × mob multiplier; fall
	// back to the legacy constant when the item has no rarity_tier.
	maxStock := func(itemId, fallback int) int {
		if got := shops.EffectiveMaxStock(itemId, mob.StockMultiplier); got > 0 {
			return got
		}
		return fallback
	}

	// Seed from legacy shop items (unlimited stock → restocked each cycle).
	for _, si := range mob.Character.Shop {
		if si.ItemId <= 0 || seen[si.ItemId] {
			continue
		}
		seen[si.ItemId] = true
		template.Stock = append(template.Stock, shops.StockEntry{
			ItemId:     si.ItemId,
			RestockQty: 5,
			MaxStock:   maxStock(si.ItemId, 20),
		})
	}

	// Seed crafter restock materials (ingredients the supply cart delivers).
	for _, itemId := range mob.CrafterRestockMaterials {
		if itemId <= 0 || seen[itemId] {
			continue
		}
		seen[itemId] = true
		template.Stock = append(template.Stock, shops.StockEntry{
			ItemId:     itemId,
			RestockQty: 3,
			MaxStock:   maxStock(itemId, 10),
		})
	}

	template.CraftSupport = mob.ShopCraftSupport
	shops.RegisterShop(mob.Zone, int(mob.MobId), mob.HomeRoomId, template)
}

// PrewarmShopForSpawnPlacement pre-registers a shop in the cache for
// a (mob template, room) placement WITHOUT spawning the actual mob.
// Used at boot to seed the economy/health dashboard with every shop
// the world knows about, including ones in zones no player has
// visited yet.
//
// Builds a synthetic *Mob with the template's shop config but the
// supplied roomId, then delegates to RegisterMobShop. The synthetic
// mob is discarded after RegisterShop runs — only the cache entry
// persists. Idempotent: RegisterShop short-circuits on a cache hit
// (and its existing CraftSupport auto-migration handles the case
// where a real mob spawn later registers with the same key).
func PrewarmShopForSpawnPlacement(template *Mob, roomId int) {
	if template == nil || roomId <= 0 {
		return
	}
	synthetic := Mob{
		MobId:                   template.MobId,
		Zone:                    template.Zone,
		HomeRoomId:              roomId,
		Crafter:                 template.Crafter,
		CrafterRestockMaterials: template.CrafterRestockMaterials,
		CrafterRecipeIds:        template.CrafterRecipeIds,
		ShopCraftSupport:        template.ShopCraftSupport,
		StockMultiplier:         template.StockMultiplier,
	}
	synthetic.Character.Shop = template.Character.Shop
	synthetic.Character.Gold = template.Character.Gold
	RegisterMobShop(&synthetic)
}

// TickMobShopRestock fires per-tier restock cycles based on each
// rarity tier's configured cadence. A shop with stock entries across
// multiple tiers sees fast cycles for commons (tier 50) and slow
// cycles for rares (tier 10), matching the per-tier cadence config.
//
// Returns true if any tier fired this tick.
func TickMobShopRestock(mob *Mob) bool {
	if mob.Crafter {
		return false // crafters use TickMobCraft path
	}
	shopInv := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)
	if shopInv == nil {
		return false
	}
	if shopInv.LastRestockByTier == nil {
		shopInv.LastRestockByTier = map[int]uint64{}
	}

	b := configs.GetBalanceConfig()
	roundCount := util.GetRoundCount()
	// Integer division: RoundsPerDay is typically not divisible by 24
	// (e.g., 900/24=37, truncating from 37.5). The sub-round-per-hour
	// drift is negligible (<2% cadence error even at tier-10).
	roundsPerHour := uint64(configs.GetTimingConfig().RoundsPerDay) / 24

	anyFired := false
	for _, tier := range []int{50, 40, 30, 20, 10} {
		hours := shops.RestockCadenceHours(b, tier)
		if hours <= 0 {
			continue
		}
		cadence := uint64(hours) * roundsPerHour
		last := shopInv.LastRestockByTier[tier]
		if last == 0 {
			shopInv.LastRestockByTier[tier] = roundCount
			continue
		}
		if roundCount-last < cadence {
			continue
		}
		shopInv.LastRestockByTier[tier] = roundCount
		if shopInv.RestockTier(tier) {
			anyFired = true
		}
	}
	return anyFired
}

// TickMobCraft fires on every crafter mob idle tick. Per-tier
// restock cadences gate actual stock additions (commons hourly,
// rares every 5 days), but recipe evaluation and salvage logic
// run on every tick so the crafter can react to depletion quickly.
// Returns a CraftResult summarizing what (if anything) the crafter
// did on this tick. Returns nil when the mob is not a crafter, the
// crafting feature is disabled, or the mob is in combat.
func TickMobCraft(mob *Mob) *CraftResult {
	if !mob.Crafter {
		return nil
	}
	b := configs.GetBalanceConfig()
	if !bool(b.CrafterEnabled) {
		return nil
	}
	if mob.Character.Aggro != nil {
		return nil
	}

	roundCount := util.GetRoundCount()
	// Integer division: RoundsPerDay is typically not divisible by 24
	// (e.g., 900/24=37, truncating from 37.5). The sub-round-per-hour
	// drift is negligible (<2% cadence error even at tier-10).
	roundsPerHour := uint64(configs.GetTimingConfig().RoundsPerDay) / 24

	// ── ShopInventory path ─────────────────────────────────────────────────
	shopInv := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)

	if shopInv != nil {
		if shopInv.LastRestockByTier == nil {
			shopInv.LastRestockByTier = map[int]uint64{}
		}

		caravanServed := b.IsCaravanServedZone(mob.Zone)
		restocked := false
		for _, tier := range []int{50, 40, 30, 20, 10} {
			hours := shops.RestockCadenceHours(b, tier)
			if hours <= 0 {
				continue
			}
			cadence := uint64(hours) * roundsPerHour
			last := shopInv.LastRestockByTier[tier]
			if last == 0 {
				shopInv.LastRestockByTier[tier] = roundCount
				continue
			}
			if roundCount-last < cadence {
				continue
			}
			shopInv.LastRestockByTier[tier] = roundCount
			if caravanServed {
				if tier == 50 || tier == 40 {
					// Caravan-served zones: common tiers (50/40) still refill via
					// the ticker as a baseline. Rarer tiers (30/20/10) depend on
					// caravan/forager deliveries — the ticker advances their
					// timestamp above to prevent drift if zone-status changes,
					// but does not add stock here.
					if shopInv.RestockTier(tier) {
						restocked = true
					}
				}
				// Intentional no-op for tiers 30/20/10 in caravan-served zones.
			} else {
				if shopInv.RestockTier(tier) {
					restocked = true
				}
			}
		}

		cfg := shops.DefaultPricingConfig()
		// Per-ingredient reserve: the crafter will not consume an ingredient
		// if doing so would drop its stock below MaxStock×reservePct (floor 1).
		// This keeps at least 25% (by default) of each ingredient available
		// for players to buy rather than draining the shop to a single unit.
		reservePct := float64(b.CrafterIngredientReservePct)

		// Build full recipe list: mob's known recipes + shop's known recipes
		recipeIds := mergeRecipeIds(mob.CrafterRecipeIds, shopInv.KnownRecipes)

		// Helper to tag restock on any result.
		tagRestock := func(r *CraftResult) *CraftResult {
			if r != nil {
				r.Restocked = restocked
			}
			return r
		}

		// ── Priority 1: Self-gear upgrade ──────────────────────────────────
		if selfRecipe := pickSelfGearRecipe(mob, recipeIds, shopInv, reservePct); selfRecipe != nil {
			return tagRestock(executeCraft(mob, selfRecipe, shopInv))
		}

		// ── Priority 2: Profitable craft ──────────────────────────────────
		craftDecision := shops.EvaluateCraftOptions(recipeIds, shopInv, cfg, reservePct)
		if craftDecision != nil {
			recipe := crafting.GetRecipe(craftDecision.RecipeId)
			if recipe != nil {
				return tagRestock(executeCraft(mob, recipe, shopInv))
			}
		}

		// ── Priority 3: Profitable salvage ────────────────────────────────
		salvageDecision := shops.EvaluateSalvageOptions(shopInv, cfg)
		if salvageDecision != nil {
			shopInv.RemoveStockAtRound(salvageDecision.ItemId, 1, util.GetRoundCount())
			spec := items.GetItemSpec(salvageDecision.ItemId)
			if spec != nil {
				for _, ret := range spec.SalvageReturns {
					matSpec := items.FindSpecByComponentTag(ret.ItemTag)
					if matSpec != nil {
						shopInv.AddStockAtRound(matSpec.ItemId, ret.Quantity, util.GetRoundCount())
					}
				}
			}
			if err := saveShopFn(mob.Zone, int(mob.MobId), mob.HomeRoomId); err != nil {
				mudlog.Warn("TickMobCraft salvage save", "mob", mob.Character.Name, "error", err)
			}
			return &CraftResult{
				Success:      true,
				OutputItemId: salvageDecision.ItemId,
				MobName:      mob.Character.Name,
				Zone:         mob.Character.Zone,
				Salvaged:     true,
				Restocked:    restocked,
			}
		}

		// No craft or salvage this tick, but restock may have happened.
		if restocked {
			return &CraftResult{
				Restocked: true,
				MobName:   mob.Character.Name,
				Zone:      mob.Character.Zone,
			}
		}
		return nil
	}

	// ── Legacy path (no ShopInventory) ────────────────────────────────────
	// Restock materials into backpack — suppressed for caravan-served zones.
	if !b.IsCaravanServedZone(mob.Zone) {
		for _, itemId := range mob.CrafterRestockMaterials {
			itm := items.New(itemId)
			if itm.ItemId > 0 {
				mob.Character.StoreItem(itm)
			}
		}
	}

	// Pick a recipe and attempt it immediately
	recipe := pickEligibleRecipe(mob)
	if recipe == nil {
		return nil
	}

	return executeCraftLegacy(mob, recipe)
}

// mergeRecipeIds combines two slices of recipe IDs, deduplicating them.
func mergeRecipeIds(a, b []string) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	result := make([]string, 0, len(a)+len(b))
	for _, id := range a {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	for _, id := range b {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}
	return result
}

// pickSelfGearRecipe finds a recipe whose output is an equipment upgrade for
// the mob. Returns nil if no upgrade recipe is craftable with current stock.
// reservePct is the per-ingredient reserve fraction; see HasMaterialsWithReservePct.
func pickSelfGearRecipe(mob *Mob, recipeIds []string, shopInv *shops.ShopInventory, reservePct float64) *crafting.RecipeSpec {
	wornItems := mob.Character.Equipment.GetAllItems()

	for _, recipeId := range recipeIds {
		recipe := crafting.GetRecipe(recipeId)
		if recipe == nil || recipe.Output.ItemId <= 0 {
			continue
		}
		if crafting.IsEnchantingRecipe(recipe) {
			continue
		}

		candidateSpec := items.GetItemSpec(recipe.Output.ItemId)
		if candidateSpec == nil {
			continue
		}

		// Check if this is an upgrade over any currently worn item of the same type
		isUpgrade := false
		wornSameType := false
		for _, worn := range wornItems {
			wornSpec := worn.GetSpec()
			if wornSpec.Type == candidateSpec.Type {
				wornSameType = true
				if items.IsUpgrade(wornSpec, *candidateSpec) {
					isUpgrade = true
					break
				}
			}
		}
		// Also an upgrade if the slot is empty (nothing worn of that type)
		if !wornSameType && items.ItemPower(*candidateSpec) > 0 {
			isUpgrade = true
		}

		if !isUpgrade {
			continue
		}

		// Check materials are available (backpack for legacy; shopInv for shop path)
		if !shops.HasMaterialsWithReservePct(recipe, shopInv, reservePct) {
			continue
		}

		return recipe
	}
	return nil
}

// executeCraft performs a craft attempt using ShopInventory for material
// tracking. On success, the output is added directly to shopInv stock.
func executeCraft(mob *Mob, recipe *crafting.RecipeSpec, shopInv *shops.ShopInventory) *CraftResult {
	round := util.GetRoundCount()

	// Consume ingredients from shop stock (round-aware so depletion events
	// fire and the dashboard's throughput scoring can see crafter demand).
	for _, ing := range recipe.Ingredients {
		spec := items.FindSpecByComponentTag(ing.ItemTag)
		if spec != nil {
			removed := shopInv.RemoveStockAtRound(spec.ItemId, ing.Quantity, round)
			shopInv.ConsumedByCrafterCount += removed
		}
	}

	skillLevel := mob.Character.GetSkillLevel(skills.SkillTag(recipe.Skill))
	chance := crafting.CalcSuccessChance(skillLevel, recipe.SkillMinimum)

	result := &CraftResult{
		RecipeName:   recipe.Name,
		OutputItemId: recipe.Output.ItemId,
		SkillMinimum: recipe.SkillMinimum,
		MobName:      mob.Character.Name,
		Zone:         mob.Character.Zone,
	}

	if util.Rand(100) < chance {
		result.Success = true
		if recipe.Output.ItemId > 0 {
			// All crafts land in shop stock — including gear-upgrade
			// crafts. The Priority-1 selector (pickSelfGearRecipe)
			// still fires preferentially for player-relevant gear even
			// when narrowly unprofitable, ensuring shopkeepers stock
			// daggers/bucklers/etc.; routing those to the shop instead
			// of the mob's backpack makes them buyable. Round-aware so
			// refill events fire when previously-depleted output slots
			// are restocked by a craft (Kerra TtR scoring).
			for i := 0; i < recipe.Output.Quantity; i++ {
				shopInv.AddStockAtRound(recipe.Output.ItemId, 1, round)
			}
		}
		mob.Character.OnSkillUse(recipe.Skill, 0)
	}

	// Persist shop state after any craft attempt (success or failure both
	// consume ingredients; failing to save loses that consumption and any
	// output that was added this tick).
	if shopInv != nil {
		if err := saveShopFn(mob.Zone, int(mob.MobId), mob.HomeRoomId); err != nil {
			mudlog.Warn("executeCraft save", "mob", mob.Character.Name, "error", err)
		}
	}

	return result
}

// executeCraftLegacy performs a craft attempt using the mob's backpack for
// ingredient tracking. Used when no ShopInventory is registered.
func executeCraftLegacy(mob *Mob, recipe *crafting.RecipeSpec) *CraftResult {
	skillLevel := mob.Character.GetSkillLevel(skills.SkillTag(recipe.Skill))
	chance := crafting.CalcSuccessChance(skillLevel, recipe.SkillMinimum)

	result := &CraftResult{
		RecipeName:   recipe.Name,
		OutputItemId: recipe.Output.ItemId,
		SkillMinimum: recipe.SkillMinimum,
		MobName:      mob.Character.Name,
		Zone:         mob.Character.Zone,
	}

	// Consume ingredients regardless of success
	backpack := mob.Character.GetAllBackpackItems()
	remaining, _ := crafting.ConsumeIngredients(backpack, []items.Item{}, recipe)
	mob.Character.Items = remaining

	if util.Rand(100) < chance {
		result.Success = true
		if recipe.Output.ItemId > 0 {
			for i := 0; i < recipe.Output.Quantity; i++ {
				mob.Character.Shop.StockItem(recipe.Output.ItemId)
			}
		}
		mob.Character.OnSkillUse(recipe.Skill, 0)
	}

	return result
}

// pickEligibleRecipe finds a random recipe the mob can attempt (legacy path).
func pickEligibleRecipe(mob *Mob) *crafting.RecipeSpec {
	backpack := mob.Character.GetAllBackpackItems()
	var eligible []*crafting.RecipeSpec

	for _, recipeId := range mob.CrafterRecipeIds {
		recipe := crafting.GetRecipe(recipeId)
		if recipe == nil {
			continue
		}
		// Must match the mob's craft skill
		if recipe.Skill != mob.CrafterSkill {
			continue
		}
		// Skill minimum check
		skillLevel := mob.Character.GetSkillLevel(skills.SkillTag(recipe.Skill))
		if skillLevel < recipe.SkillMinimum {
			continue
		}
		// Ingredient check (mobs don't have a component bag)
		ok, _ := crafting.HasIngredients(backpack, []items.Item{}, recipe)
		if !ok {
			continue
		}
		eligible = append(eligible, recipe)
	}

	if len(eligible) == 0 {
		return nil
	}
	return eligible[util.Rand(len(eligible))]
}
