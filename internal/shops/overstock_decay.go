package shops

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// DecayedUnit records the item and quantity removed by a single TickOverstockDecay
// sweep for a given stock entry. Callers that need to convert decayed units into
// other resources (e.g. enchanting materials) consume this slice.
type DecayedUnit struct {
	ItemId int
	Qty    int
}

// TickOverstockDecay drains unsold non-material overstock. For each stock entry
// whose Current exceeds its restock baseline (RestockQty) and is not a crafting
// material, if at least decayRounds have elapsed since it last grew, remove
// decayQty units (never below the baseline) and re-stamp LastGrewRound so decay
// paces out. Crafting materials (is_component) are never decayed.
//
// Baseline = RestockQty: NPC-dumped/backfilled items (RestockQty 0) drain fully
// to 0 when unsold; staples drain only to the level natural restock maintains.
func TickOverstockDecay(si *ShopInventory, round uint64) []DecayedUnit {
	b := configs.GetBalanceConfig()
	return TickOverstockDecayWith(si, round, isComponentItem, uint64(b.ShopOverstockDecayRounds), int(b.ShopOverstockDecayQty))
}

// TickOverstockDecayWith is the testable core; isComponent + thresholds are
// injected so unit tests need no loaded item specs.
func TickOverstockDecayWith(si *ShopInventory, round uint64, isComponent func(itemId int) bool, decayRounds uint64, decayQty int) []DecayedUnit {
	if si == nil || decayRounds == 0 || decayQty <= 0 {
		return nil
	}
	var decayed []DecayedUnit
	for i := range si.Stock {
		e := &si.Stock[i]
		baseline := e.RestockQty
		if e.Current <= baseline {
			continue
		}
		if isComponent(e.ItemId) {
			continue
		}
		if e.LastGrewRound != 0 && round < e.LastGrewRound+decayRounds {
			continue
		}
		drop := decayQty
		if e.Current-drop < baseline {
			drop = e.Current - baseline
		}
		e.Current -= drop
		e.LastGrewRound = round // pace subsequent decays
		decayed = append(decayed, DecayedUnit{ItemId: e.ItemId, Qty: drop})
	}
	return decayed
}

func isComponentItem(itemId int) bool {
	spec := items.GetItemSpec(itemId)
	return spec != nil && spec.IsComponent
}
