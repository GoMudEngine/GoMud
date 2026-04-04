package shops

import (
	"math"
)

// PricingConfig holds the tunable knobs for dynamic pricing.
type PricingConfig struct {
	BuyRatio            float64 // Base buy/sell spread (default 0.50)
	PriceFloor          float64 // Min scarcity multiplier (default 0.25)
	PriceCeiling        float64 // Max scarcity multiplier (default 5.0)
	AbundanceThreshold  float64 // Stock/restock ratio for full abundance (default 3.0)
}

// DefaultPricingConfig returns sensible defaults.
func DefaultPricingConfig() PricingConfig {
	return PricingConfig{
		BuyRatio:           0.50,
		PriceFloor:         0.25,
		PriceCeiling:       5.0,
		AbundanceThreshold: 3.0,
	}
}

// ScarcityMultiplier computes the price multiplier based on current stock
// and the item's restock quantity (which normalizes the curve).
// Range: PriceFloor (overstocked) to PriceCeiling (out of stock).
// For NPC-crafted items with no restock, caller should pass an appropriate
// normalizer (e.g., MaxStock/2).
func ScarcityMultiplier(current int, restockQty int, cfg PricingConfig) float64 {
	if restockQty <= 0 {
		restockQty = 1 // Avoid division by zero
	}

	ratio := float64(current) / float64(restockQty)

	if ratio <= 0 {
		return cfg.PriceCeiling
	}
	if ratio >= cfg.AbundanceThreshold {
		return cfg.PriceFloor
	}

	// Inverse quadratic: prices rise sharply as stock approaches zero
	t := ratio / cfg.AbundanceThreshold // 0.0 to 1.0
	mult := cfg.PriceFloor + (cfg.PriceCeiling-cfg.PriceFloor)*math.Pow(1.0-t, 2)
	return mult
}

// CalcSellPrice computes what the NPC charges a player to buy an item.
func CalcSellPrice(baseValue int, current int, restockQty int, cfg PricingConfig) int {
	mult := ScarcityMultiplier(current, restockQty, cfg)
	price := math.Ceil(float64(baseValue) * mult)
	if price < 1 {
		price = 1
	}
	return int(price)
}

// CalcBuyPrice computes what the NPC offers a player for an item.
func CalcBuyPrice(baseValue int, current int, restockQty int, cfg PricingConfig) int {
	mult := ScarcityMultiplier(current, restockQty, cfg)
	price := math.Ceil(float64(baseValue) * cfg.BuyRatio * mult)
	if price < 1 {
		price = 1
	}
	return int(price)
}

// ApplyBarterSellDiscount reduces a sell price based on bartering skill.
// discount is 0.0–1.0 representing percentage reduction.
func ApplyBarterSellDiscount(price int, discount float64) int {
	adjusted := float64(price) * (1.0 - discount)
	if adjusted < 1 {
		adjusted = 1
	}
	return int(math.Ceil(adjusted))
}

// ApplyBarterBuyBonus increases a buy price based on bartering skill.
// bonus is 0.0–1.0 representing percentage increase.
func ApplyBarterBuyBonus(price int, bonus float64) int {
	adjusted := float64(price) * (1.0 + bonus)
	return int(math.Ceil(adjusted))
}
