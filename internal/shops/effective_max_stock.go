// Package shops — Stage 3.4 EffectiveMaxStock helper.
//
// Stock cap = ItemSpec.RarityTier × stockMultiplier. Returns 0 for
// untiered items (quest items, defer-to-3.0e items, items without
// rarity_tier set). Callers fall back to legacy hardcoded values in
// that case.
//
// The mob pointer is intentionally NOT taken here to avoid an import
// cycle (mobs imports shops; shops must not import mobs). Callers
// pass mob.StockMultiplier directly.
package shops

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// EffectiveMaxStock returns the derived stock cap for an item given
// the vendor's stock multiplier. Returns 0 if the item has no
// rarity_tier — caller decides the fallback policy.
//
// Final cap = RarityTier × stockMultiplier × ShopMaxStockMultiplier.
//
// A stockMultiplier of 0 (unset on the mob) is treated as 1.0.
// ShopMaxStockMultiplier is a global knob — chunk 3.8 bumped to 2.0
// so the caravan has room to build cross-city surplus without
// vendors hitting their cap immediately on a single restock cycle.
func EffectiveMaxStock(itemId int, stockMultiplier float64) int {
	spec := items.GetItemSpec(itemId)
	if spec == nil || spec.RarityTier <= 0 {
		return 0
	}
	mult := stockMultiplier
	if mult <= 0 {
		mult = 1.0
	}
	globalMult := float64(configs.GetBalanceConfig().ShopMaxStockMultiplier)
	if globalMult <= 0 {
		globalMult = 1.0
	}
	return int(float64(spec.RarityTier) * mult * globalMult)
}
