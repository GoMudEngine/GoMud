package health_test

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/economy/health"
)

func TestScore_PerShop_WeightedByRestockQty(t *testing.T) {
	shop := health.ShopSnapshot{
		Stock: []health.StockSnapshot{
			{ItemId: 1, RestockQty: 5, Current: 5, Max: 10},   // 50% fill, weight 5
			{ItemId: 2, RestockQty: 10, Current: 10, Max: 10}, // 100% fill, weight 10
		},
	}
	score := health.PerShopScore(shop)
	// weighted: (5*0.5 + 10*1.0) / (5+10) = 12.5/15 = 0.8333 → 83.33
	if math.Abs(score-83.33) > 0.5 {
		t.Errorf("got %.2f, want ~83.33", score)
	}
}

func TestScore_PerShop_NoStockReturnsNil(t *testing.T) {
	shop := health.ShopSnapshot{}
	if v, ok := health.PerShopScoreOpt(shop); ok {
		t.Errorf("got %v, want (_, false) for empty shop", v)
	}
}

func TestScore_PerCraftSupport_MeanOfShops(t *testing.T) {
	snap := health.Snapshot{
		Shops: []health.ShopSnapshot{
			{CraftSupport: "blacksmithing", Stock: []health.StockSnapshot{{RestockQty: 1, Current: 4, Max: 10}}}, // 40
			{CraftSupport: "blacksmithing", Stock: []health.StockSnapshot{{RestockQty: 1, Current: 8, Max: 10}}}, // 80
			{CraftSupport: "cooking", Stock: []health.StockSnapshot{{RestockQty: 1, Current: 5, Max: 10}}},        // 50
		},
	}
	scores := health.PerCraftSupportScores(snap)
	if math.Abs(scores["blacksmithing"]-60) > 0.01 {
		t.Errorf("blacksmithing: got %.2f, want 60", scores["blacksmithing"])
	}
	if math.Abs(scores["cooking"]-50) > 0.01 {
		t.Errorf("cooking: got %.2f, want 50", scores["cooking"])
	}
}
