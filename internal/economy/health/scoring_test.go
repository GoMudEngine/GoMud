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

func TestScore_Caravan_StuckPenalty(t *testing.T) {
	// Pin the stuck-penalty branch: a caravan whose state has been held
	// longer than stuckThresholdRounds (5000) loses 30 points. With 0
	// completed cycles in history, score is already 0; the penalty just
	// confirms the clamp doesn't push it negative.
	hist := make([]*health.Snapshot, 24) // satisfy minHistoryForCycles
	for i := range hist {
		hist[i] = &health.Snapshot{}
	}
	cur := health.Snapshot{
		Round: 10000,
		Caravans: []health.CaravanSnapshot{
			{InstId: 1, State: "outbound_transit", StateEnteredRound: 1000},
		},
	}
	score, ok := health.PerCaravanScore(1, cur, hist)
	if !ok {
		t.Fatal("got (_, false), want (_, true) — sufficient history")
	}
	if score != 0 {
		t.Errorf("got %v, want 0 (stuck-penalty clamp)", score)
	}
}

func TestScore_OverallWeightsShopsHeaviest(t *testing.T) {
	cur := health.Snapshot{
		Shops: []health.ShopSnapshot{
			{CraftSupport: "blacksmithing", Stock: []health.StockSnapshot{{RestockQty: 1, Current: 5, Max: 10}}}, // 50
		},
		Caravans: []health.CaravanSnapshot{{InstId: 1, State: "thornwall_dwell"}},
		Foragers: []health.ForagerSnapshot{{InstId: 7, State: "resting"}},
	}

	// Build 24 history entries that yield ~24 caravan cycles and ~24
	// forager cycles (each entry alternates state). With the default
	// expected cadence, both score = 100.
	hist := make([]*health.Snapshot, 0, 24)
	caravanThornwall := &health.Snapshot{Caravans: []health.CaravanSnapshot{{InstId: 1, State: "thornwall_dwell"}}, Foragers: []health.ForagerSnapshot{{InstId: 7, State: "resting"}}}
	caravanTransit := &health.Snapshot{Caravans: []health.CaravanSnapshot{{InstId: 1, State: "outbound_transit"}}, Foragers: []health.ForagerSnapshot{{InstId: 7, State: "foraging"}}}
	for i := 0; i < 24; i++ {
		if i%2 == 0 {
			hist = append(hist, caravanThornwall)
		} else {
			hist = append(hist, caravanTransit)
		}
	}

	scores := health.Score(&cur, hist)

	if scores.PerShop[0].Score != 50 {
		t.Errorf("PerShop[0]: got %.2f, want 50", scores.PerShop[0].Score)
	}
	// With shop=50, caravan~100, forager~100 and weights 0.6/0.2/0.2:
	// overall = (0.6*50 + 0.2*100 + 0.2*100) / 1.0 = 70.
	// Allow ±15 for variation in cycle counting against the chosen pattern.
	if scores.OverallScore < 55 || scores.OverallScore > 85 {
		t.Errorf("OverallScore: got %.2f, want in [55, 85] (shops weighted heaviest)", scores.OverallScore)
	}
	// Shops weighted heaviest sanity: the weighted overall should be pulled
	// toward MeanShop vs the unweighted mean. Concretely: overall must be
	// closer to MeanShop than the unweighted mean is to MeanShop.
	unweightedMean := (scores.MeanShop + scores.MeanCaravan + scores.MeanForager) / 3
	if math.Abs(scores.OverallScore-scores.MeanShop) >= math.Abs(unweightedMean-scores.MeanShop) {
		t.Errorf("Overall %.2f not pulled toward MeanShop %.2f vs unweighted mean %.2f — shops aren't weighted heaviest",
			scores.OverallScore, scores.MeanShop, unweightedMean)
	}
}

