package mobs

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

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
}

// TickMobCraft handles autonomous crafting for crafter mobs. Crafting only
// fires on the material restock tick (every CrafterMaterialRestockRate rounds):
// materials arrive, and the mob immediately attempts one craft from them.
// Returns a non-nil CraftResult only when a craft is attempted.
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

	// Initialize restock timer on first call
	restockRate := uint64(b.CrafterMaterialRestockRate)
	if restockRate == 0 {
		return nil
	}
	if mob.crafterLastRestockRound == 0 {
		mob.crafterLastRestockRound = roundCount
		return nil
	}

	// Only act on the restock tick
	if roundCount-mob.crafterLastRestockRound < restockRate {
		return nil
	}
	mob.crafterLastRestockRound = roundCount

	// ── ShopInventory path ─────────────────────────────────────────────────
	shopInv := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)

	if shopInv != nil {
		// Supply cart delivery
		shopInv.Restock()

		cfg := shops.DefaultPricingConfig()
		reserve := 1 // Keep at least 1 of each ingredient in stock

		// Build full recipe list: mob's known recipes + shop's known recipes
		recipeIds := mergeRecipeIds(mob.CrafterRecipeIds, shopInv.KnownRecipes)

		// ── Priority 1: Self-gear upgrade ──────────────────────────────────
		if selfRecipe := pickSelfGearRecipe(mob, recipeIds, shopInv, reserve); selfRecipe != nil {
			return executeCraft(mob, selfRecipe, true, shopInv)
		}

		// ── Priority 2: Profitable craft ──────────────────────────────────
		craftDecision := shops.EvaluateCraftOptions(recipeIds, shopInv, cfg, reserve)
		if craftDecision != nil {
			recipe := crafting.GetRecipe(craftDecision.RecipeId)
			if recipe != nil {
				return executeCraft(mob, recipe, false, shopInv)
			}
		}

		// ── Priority 3: Profitable salvage ────────────────────────────────
		salvageDecision := shops.EvaluateSalvageOptions(shopInv, cfg)
		if salvageDecision != nil {
			shopInv.RemoveStock(salvageDecision.ItemId, 1)
			spec := items.GetItemSpec(salvageDecision.ItemId)
			if spec != nil {
				for _, ret := range spec.SalvageReturns {
					matSpec := items.FindSpecByComponentTag(ret.ItemTag)
					if matSpec != nil {
						shopInv.AddStock(matSpec.ItemId, ret.Quantity)
					}
				}
			}
			return &CraftResult{
				Success:      true,
				OutputItemId: salvageDecision.ItemId,
				MobName:      mob.Character.Name,
				Zone:         mob.Character.Zone,
				Salvaged:     true,
			}
		}

		// Nothing to do this tick
		return nil
	}

	// ── Legacy path (no ShopInventory) ────────────────────────────────────
	// Restock materials into backpack
	for _, itemId := range mob.CrafterRestockMaterials {
		itm := items.New(itemId)
		if itm.ItemId > 0 {
			mob.Character.StoreItem(itm)
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
func pickSelfGearRecipe(mob *Mob, recipeIds []string, shopInv *shops.ShopInventory, reserve int) *crafting.RecipeSpec {
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
		if !shops.HasMaterialsWithReserve(recipe, shopInv, reserve) {
			continue
		}

		return recipe
	}
	return nil
}

// executeCraft performs a craft attempt using ShopInventory for material
// tracking. On success, the output is added directly to shopInv stock.
func executeCraft(mob *Mob, recipe *crafting.RecipeSpec, forSelf bool, shopInv *shops.ShopInventory) *CraftResult {
	// Consume ingredients from shop stock
	for _, ing := range recipe.Ingredients {
		spec := items.FindSpecByComponentTag(ing.ItemTag)
		if spec != nil {
			shopInv.RemoveStock(spec.ItemId, ing.Quantity)
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
		if forSelf {
			// Equip the crafted item directly
			if recipe.Output.ItemId > 0 {
				itm := items.New(recipe.Output.ItemId)
				if itm.ItemId > 0 {
					mob.Character.StoreItem(itm)
				}
			}
		} else if recipe.Output.ItemId > 0 {
			for i := 0; i < recipe.Output.Quantity; i++ {
				shopInv.AddStock(recipe.Output.ItemId, 1)
			}
		}
		mob.Character.OnSkillUse(recipe.Skill, 0)
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
