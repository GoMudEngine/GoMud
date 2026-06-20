package shops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPricingConfig_BaselineQty(t *testing.T) {
	assert.Equal(t, 3, DefaultPricingConfig().DefaultBaselineQty)
}

func TestScarcityMultiplier_OutOfStock(t *testing.T) {
	cfg := DefaultPricingConfig()
	mult := ScarcityMultiplier(0, 5, cfg)
	assert.Equal(t, 5.0, mult)
}

func TestScarcityMultiplier_FullyAbundant(t *testing.T) {
	cfg := DefaultPricingConfig()
	mult := ScarcityMultiplier(15, 5, cfg) // ratio = 3.0 = threshold
	assert.Equal(t, 0.25, mult)
}

func TestScarcityMultiplier_OverAbundant(t *testing.T) {
	cfg := DefaultPricingConfig()
	mult := ScarcityMultiplier(20, 5, cfg) // ratio = 4.0 > threshold
	assert.Equal(t, 0.25, mult)
}

func TestScarcityMultiplier_AtRestock(t *testing.T) {
	cfg := DefaultPricingConfig()
	mult := ScarcityMultiplier(5, 5, cfg) // ratio = 1.0
	assert.Greater(t, mult, cfg.PriceFloor)
	assert.Less(t, mult, cfg.PriceCeiling)
}

func TestScarcityMultiplier_Monotonic(t *testing.T) {
	cfg := DefaultPricingConfig()
	prev := ScarcityMultiplier(0, 5, cfg)
	for stock := 1; stock <= 15; stock++ {
		curr := ScarcityMultiplier(stock, 5, cfg)
		assert.LessOrEqual(t, curr, prev, "stock=%d should have lower or equal mult than stock=%d", stock, stock-1)
		prev = curr
	}
}

func TestScarcityMultiplier_ZeroRestock(t *testing.T) {
	cfg := DefaultPricingConfig()
	// Should not panic, treats as restockQty=1
	mult := ScarcityMultiplier(3, 0, cfg)
	assert.Greater(t, mult, 0.0)
}

func TestCalcSellPrice_OutOfStock(t *testing.T) {
	cfg := DefaultPricingConfig()
	assert.Equal(t, 50, CalcSellPrice(10, 0, 5, cfg))
}

func TestCalcBuyPrice_OutOfStock(t *testing.T) {
	cfg := DefaultPricingConfig()
	assert.Equal(t, 25, CalcBuyPrice(10, 0, 5, cfg))
}

func TestCalcBuyPrice_Abundant(t *testing.T) {
	cfg := DefaultPricingConfig()
	price := CalcBuyPrice(10, 15, 5, cfg)
	assert.LessOrEqual(t, price, 2)
}

func TestCalcSellPrice_MinimumOne(t *testing.T) {
	cfg := DefaultPricingConfig()
	// Even a 0-value item should cost at least 1
	price := CalcSellPrice(0, 10, 5, cfg)
	assert.GreaterOrEqual(t, price, 1)
}

func TestBarterSellDiscount(t *testing.T) {
	assert.Equal(t, 85, ApplyBarterSellDiscount(100, 0.15))
}

func TestBarterBuyBonus(t *testing.T) {
	assert.Equal(t, 115, ApplyBarterBuyBonus(100, 0.15))
}

func TestBarterSellDiscount_MinimumOne(t *testing.T) {
	assert.Equal(t, 1, ApplyBarterSellDiscount(1, 0.99))
}

func TestSpread_BuyAlwaysLessThanSell(t *testing.T) {
	cfg := DefaultPricingConfig()
	for stock := 0; stock <= 20; stock++ {
		sell := CalcSellPrice(10, stock, 5, cfg)
		buy := CalcBuyPrice(10, stock, 5, cfg)
		assert.LessOrEqual(t, buy, sell, "buy should be <= sell at stock=%d", stock)
	}
}
