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
//  3. Potions — if the item is a potion, reject declining or spoiled items.
//     Accept others at a dynamic buy price.
//  4. General goods — if buysGeneral is true, offer 25% of base value.
//
// Returns an empty BuyOffer (Price == 0) when the NPC refuses the item.
func EvaluateBuyRules(
	item items.Item,
	shopInv *ShopInventory,
	crafterSkill string,
	buysGeneral bool,
	cfg PricingConfig,
) BuyOffer {
	spec := item.GetSpec()
	if spec.ItemId < 1 {
		return BuyOffer{}
	}

	// Rule 1: Quest items are never bought.
	if spec.QuestToken != "" {
		return BuyOffer{}
	}

	// Rule 2: Craft materials — only if NPC has a crafter skill.
	if crafterSkill != "" && spec.ComponentTag != "" {
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

	// Rule 3: Potions.
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

	// Rule 4: General goods.
	if buysGeneral {
		price := int(float64(spec.Value) * 0.25)
		if price < 1 {
			price = 1
		}
		return BuyOffer{Price: price, Reason: "general"}
	}

	return BuyOffer{}
}
