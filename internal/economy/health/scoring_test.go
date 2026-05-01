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

func TestScore_PerShop_ClampsAndFloors(t *testing.T) {
	// Pins three documented edge cases:
	// (a) RestockQty=0 still gets a vote (weight floor of 1)
	// (b) Current > Max clamps fill to 1.0
	// (c) Max <= 0 entries are skipped
	shop := health.ShopSnapshot{
		Stock: []health.StockSnapshot{
			{ItemId: 1, RestockQty: 0, Current: 5, Max: 10},   // 50%, weight 1
			{ItemId: 2, RestockQty: 1, Current: 15, Max: 10},  // clamp 100%, weight 1
			{ItemId: 3, RestockQty: 99, Current: 99, Max: 0},  // skipped (Max<=0)
		},
	}
	score, ok := health.PerShopScoreOpt(shop)
	if !ok {
		t.Fatalf("got (_, false), want (_, true) — entries 1+2 should score")
	}
	// weighted: (1*0.5 + 1*1.0) / (1+1) = 0.75 → 75
	if math.Abs(score-75) > 0.01 {
		t.Errorf("got %.2f, want 75 (RestockQty floor + Current>Max clamp + Max<=0 skip)", score)
	}
}

func TestScore_PerCraftSupport_EmptyTagRollsToBlankKey(t *testing.T) {
	// Documented behavior: shops with empty CraftSupport roll into key ""
	// (should never happen thanks to startup validator, but pinned here).
	snap := health.Snapshot{
		Shops: []health.ShopSnapshot{
			{CraftSupport: "blacksmithing", Stock: []health.StockSnapshot{{RestockQty: 1, Current: 3, Max: 10}}}, // 30
			{CraftSupport: "", Stock: []health.StockSnapshot{{RestockQty: 1, Current: 7, Max: 10}}},               // 70 → key ""
		},
	}
	scores := health.PerCraftSupportScores(snap)
	if math.Abs(scores[""]-70) > 0.01 {
		t.Errorf("empty-tag rollup: got %.2f, want 70", scores[""])
	}
	if math.Abs(scores["blacksmithing"]-30) > 0.01 {
		t.Errorf("blacksmithing: got %.2f, want 30", scores["blacksmithing"])
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

func TestScore_Caravan_CycleCount(t *testing.T) {
	// Build a 4-snapshot history that contains exactly one ThornwallDwell→
	// ThornwallDwell transition.
	hist := []*health.Snapshot{
		{Caravans: []health.CaravanSnapshot{{InstId: 1, State: "thornwall_dwell"}}},  // t-3
		{Caravans: []health.CaravanSnapshot{{InstId: 1, State: "outbound_transit"}}}, // t-2
		{Caravans: []health.CaravanSnapshot{{InstId: 1, State: "stillwater_dwell"}}}, // t-1
		{Caravans: []health.CaravanSnapshot{{InstId: 1, State: "thornwall_dwell"}}},  // now
	}
	cycles := health.CountCaravanCycles(1, hist)
	if cycles != 1 {
		t.Errorf("got %d cycles, want 1", cycles)
	}
}

func TestScore_Forager_CycleCount(t *testing.T) {
	hist := []*health.Snapshot{
		{Foragers: []health.ForagerSnapshot{{InstId: 7, State: "resting"}}},
		{Foragers: []health.ForagerSnapshot{{InstId: 7, State: "foraging"}}},
		{Foragers: []health.ForagerSnapshot{{InstId: 7, State: "resting"}}},
		{Foragers: []health.ForagerSnapshot{{InstId: 7, State: "foraging"}}},
		{Foragers: []health.ForagerSnapshot{{InstId: 7, State: "resting"}}},
	}
	cycles := health.CountForagerCycles(7, hist)
	if cycles != 2 {
		t.Errorf("got %d cycles, want 2", cycles)
	}
}

func TestScore_Caravan_InsufficientHistory(t *testing.T) {
	cur := health.Snapshot{Caravans: []health.CaravanSnapshot{{InstId: 1, State: "thornwall_dwell"}}}
	score, ok := health.PerCaravanScore(1, cur, nil) // no history
	if ok {
		t.Errorf("got (%v, true), want (_, false) for no history", score)
	}
}
