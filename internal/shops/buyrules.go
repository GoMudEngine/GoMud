package shops

import (
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

	// Rule 2: Craft materials — only if NPC stocks this specific item
	// (i.e., it's in their restock list). This prevents cross-profession
	// buying (blacksmith buying herbs, apothecary buying ingots).
	if crafterSkill != "" && spec.ComponentTag != "" {
		entry := shopInv.GetStock(spec.ItemId)
		if entry != nil {
			if entry.MaxStock > 0 && entry.Current >= entry.MaxStock {
				// Already at capacity — skip this rule, fall through.
			} else {
				current := entry.Current
				restock := entry.RestockQty
				if restock < 1 {
					restock = 1
				}
				price := CalcBuyPrice(spec.Value, current, restock, cfg)
				return BuyOffer{Price: price, Reason: "craft_material"}
			}
		}
		// Item not in this NPC's stock list — they don't use it.
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

	// Rule 4: Potions.
	if spec.Type == items.Potion {
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
