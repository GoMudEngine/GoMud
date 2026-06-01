package planners

import (
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/itemvalue"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

// upgradeCandidate is the best in-stock upgrade the evaluator found.
type upgradeCandidate struct {
	ShopRoom int
	ItemId   int
	ItemName string  // canonical item name, used for the `buy <name>` command
	Price    int     // what the mob pays to buy from shop (CalcSellPrice = buyer-side price)
	Delta    float64 // itemvalue swap delta (gain)
}

// stockRestockNorm mirrors the normalizer used across the shops package
// (buyrules.go / craftdecision.go): RestockQty when > 0, else MaxStock/2
// (min 1), else 1. Normalizes the scarcity-pricing curve.
func stockRestockNorm(e shops.StockEntry) int {
	if e.RestockQty > 0 {
		return e.RestockQty
	}
	if e.MaxStock > 0 {
		n := e.MaxStock / 2
		if n < 1 {
			n = 1
		}
		return n
	}
	return 1
}

// scanZoneUpgrades walks every in-stock entry across all shops in the mob's
// zone, scores each as a swap for this mob via itemvalue, and prices it under
// dynamic pricing. It returns the best positive-delta upgrade (highest delta,
// tie-break lower price).
//
//   - onlyAffordable=true filters to price <= budget (used for "buy now").
//   - onlyAffordable=false ignores budget (used to decide whether to save up).
//   - minDelta gates out marginal upgrades.
//
// Rescans on each call (zone shop count is small); callers do not cache the
// chosen item, only optionally the sell-shop room.
func scanZoneUpgrades(mob *mobs.Mob, profile itemvalue.WeightProfile, budget int, onlyAffordable bool, minDelta float64) (upgradeCandidate, bool) {
	if mob == nil {
		return upgradeCandidate{}, false
	}
	cfg := shops.PricingConfigFromBalance()
	var best upgradeCandidate
	found := false

	for _, shop := range shops.AllShops() {
		if shop.Zone != mob.Character.Zone {
			continue
		}
		for _, e := range shop.Stock {
			if e.Current <= 0 {
				continue
			}
			spec := items.GetItemSpec(e.ItemId)
			if spec == nil {
				continue
			}
			// Only consider items `gearup` will actually wear. Otherwise the
			// mob buys an item it can't equip, the target slot never changes,
			// and it re-buys the same item every tick (a gold-drain loop).
			// Mirrors the bare-gearup wear filter in mobcommands/gearup.go.
			if spec.Type != items.Weapon && spec.Subtype != items.Wearable {
				continue
			}
			candidate := items.New(e.ItemId)
			delta := itemvalue.ItemValueDelta(&mob.Character, profile, candidate)
			if delta.Slot == "" || delta.Score <= minDelta {
				continue
			}
			price := shops.CalcSellPrice(spec.Value, e.Current, stockRestockNorm(e), cfg)
			if onlyAffordable && price > budget {
				continue
			}
			better := !found ||
				delta.Score > best.Delta ||
				(delta.Score == best.Delta && price < best.Price)
			if better {
				best = upgradeCandidate{
					ShopRoom: shop.RoomId,
					ItemId:   e.ItemId,
					ItemName: candidate.Name(),
					Price:    price,
					Delta:    delta.Score,
				}
				found = true
			}
		}
	}
	return best, found
}
