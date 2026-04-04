package shops

import (
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// BuyOffer represents what an NPC is willing to pay for an item.
type BuyOffer struct {
	Price  int
	Reason string // "craft_material", "potion", "general", ""
}

// EvaluateBuyRules determines what a ShopInventory-backed merchant will pay
// for an item offered by a player. Rules are checked in priority order:
//
//  1. Quest items — always rejected.
//  2. Craft materials — if the NPC has a crafter skill and the item has a
//     component tag, offer a dynamic buy price. Skip if at max stock.
//  3. Gear upgrade — if the item is an equipment upgrade over what the NPC
//     currently wears (or fills an empty slot), offer buy-ratio price.
//  4. Potions — if the item is a potion, reject declining or spoiled items.
//     Accept others at a dynamic buy price.
//  5. General goods — if buysGeneral is true, offer 25% of base value.
//
// Returns an empty BuyOffer (Price == 0) when the NPC refuses the item.
//
// wornItems is the NPC's currently equipped items (pass nil to skip the
// gear-upgrade rule).
func EvaluateBuyRules(
	item items.Item,
	shopInv *ShopInventory,
	crafterSkill string,
	buysGeneral bool,
	cfg PricingConfig,
	wornItems []items.Item,
) BuyOffer {
	spec := item.GetSpec()
	if spec.ItemId < 1 {
		return BuyOffer{}
	}

	// Rule 1: Quest items are never bought.
	if spec.QuestToken != "" {
		return BuyOffer{}
	}

	// Rule 2: Craft materials — buy if this component is used by any of the
	// NPC's known recipes (ingredient tag matches). This prevents cross-
	// profession buying while still accepting materials beyond the seed
	// stock list (e.g., steel ingots for a blacksmith with steel recipes).
	if crafterSkill != "" && spec.ComponentTag != "" && usesComponent(shopInv, spec.ItemId, spec.ComponentTag) {
		entry := shopInv.GetStock(spec.ItemId)
		if entry != nil && entry.MaxStock > 0 && entry.Current >= entry.MaxStock {
			// Already at capacity — skip this rule, fall through.
		} else {
			current := 0
			restock := 1
			if entry != nil {
				current = entry.Current
				if entry.RestockQty > 0 {
					restock = entry.RestockQty
				}
			}
			price := CalcBuyPrice(spec.Value, current, restock, cfg)
			return BuyOffer{Price: price, Reason: "craft_material"}
		}
	}

	// Rule 3: Gear upgrade — NPC wants equipment that improves their loadout.
	// Pass wornItems == nil to skip this rule entirely.
	// For paired slots (ring, wrist), checks all slots of that type: an
	// empty second slot counts as an upgrade opportunity.
	if wornItems != nil && isEquipType(spec.Type) {
		isUpgrade := false
		hasEmptySlot := false
		hasSlotAtAll := false
		candidatePower := items.ItemPower(spec)

		for _, worn := range wornItems {
			wornSpec := worn.GetSpec()
			if wornSpec.Type != spec.Type {
				continue
			}
			hasSlotAtAll = true
			if worn.ItemId == 0 {
				// Empty slot marker (from GetAllItemsWithEmptySlots).
				hasEmptySlot = true
				continue
			}
			if items.IsUpgrade(wornSpec, spec) {
				isUpgrade = true
				break
			}
		}

		// No items of this type at all = empty slot. Or explicit empty marker.
		if !isUpgrade && (!hasSlotAtAll || hasEmptySlot) && candidatePower > 0 {
			isUpgrade = true
		}

		if isUpgrade {
			price := int(float64(spec.Value) * cfg.BuyRatio)
			if price < 1 {
				price = 1
			}
			return BuyOffer{Price: price, Reason: "gear_upgrade"}
		}
	}

	// Rule 4: Potions — buy potions unless the NPC crafts them.
	if spec.Type == items.Potion && !canCraftItem(shopInv, spec.ItemId) {
		if spec.Aging.HasAging() && item.CraftedRound > 0 {
			currentRound := util.GetRoundCount()
			var elapsed uint64
			if currentRound >= item.CraftedRound {
				elapsed = currentRound - item.CraftedRound
			}
			bottleMult := item.BottleMultiplier
			if bottleMult <= 0 {
				bottleMult = spec.BottleAgingMultiplier
			}
			effectiveSpeed := items.CalcEffectiveAgingSpeed(bottleMult, item.CraftSkill)
			phase, _ := items.GetAgingPhase(elapsed, spec.Aging, effectiveSpeed)
			if phase == items.PhaseDeclining || phase == items.PhaseSpoiled {
				return BuyOffer{}
			}
		}

		entry := shopInv.GetStock(spec.ItemId)
		current := 0
		restock := 1
		if entry != nil {
			current = entry.Current
			if entry.RestockQty > 0 {
				restock = entry.RestockQty
			}
		}
		price := CalcBuyPrice(spec.Value, current, restock, cfg)
		return BuyOffer{Price: price, Reason: "potion"}
	}

	// Rule 5: General goods.
	if buysGeneral {
		price := int(float64(spec.Value) * 0.25)
		if price < 1 {
			price = 1
		}
		return BuyOffer{Price: price, Reason: "general"}
	}

	return BuyOffer{}
}

// canCraftItem returns true if any of the shop's known recipes produces the
// given item as output. Used to prevent NPCs from buying items they craft.
func canCraftItem(shopInv *ShopInventory, itemId int) bool {
	if shopInv == nil || itemId <= 0 {
		return false
	}
	for _, recipeId := range shopInv.KnownRecipes {
		recipe := crafting.GetRecipe(recipeId)
		if recipe != nil && recipe.Output.ItemId == itemId {
			return true
		}
	}
	return false
}

// usesComponent returns true if the NPC has a use for this component:
// either a known recipe uses it as an ingredient, or the item is already
// in the shop's stock list. This prevents cross-profession buying while
// accepting materials beyond the seed stock (e.g., steel ingots for a
// blacksmith who learns steel recipes).
func usesComponent(shopInv *ShopInventory, itemId int, componentTag string) bool {
	if shopInv == nil {
		return false
	}
	// Check if the item is already in the stock list by ID.
	if itemId > 0 && shopInv.GetStock(itemId) != nil {
		return true
	}
	// Check known recipes for ingredient tag match.
	for _, recipeId := range shopInv.KnownRecipes {
		recipe := crafting.GetRecipe(recipeId)
		if recipe == nil {
			continue
		}
		for _, ing := range recipe.Ingredients {
			if ing.ItemTag == componentTag {
				return true
			}
		}
	}
	// Check stock list by component tag (for restock mats not in recipes).
	for _, entry := range shopInv.Stock {
		matSpec := items.GetItemSpec(entry.ItemId)
		if matSpec != nil && matSpec.ComponentTag == componentTag {
			return true
		}
	}
	return false
}

// isEquipType returns true for item types that represent wearable/wieldable
// equipment slots — the kinds of items an NPC would consider as a personal
// gear upgrade.
func isEquipType(t items.ItemType) bool {
	switch t {
	case items.Weapon, items.Offhand, items.Head, items.Neck,
		items.Body, items.Belt, items.Gloves, items.Ring,
		items.Wrist, items.Back, items.Shoulders, items.Legs,
		items.Feet, items.Tail:
		return true
	}
	return false
}
