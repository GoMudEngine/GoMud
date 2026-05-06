package health_test

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/economy/health"
)

// testScoringCfg is a minimal ScoringConfig with spec-default TtR targets
// and a fixed RoundsPerGameHour=10 for deterministic test math.
var testScoringCfg = health.ScoringConfig{
	RoundsPerGameHour:         10,
	TtRTargetTier50Hours:      3,
	TtRTargetTier40Hours:      6,
	TtRTargetTier30Hours:      18,
	TtRTargetTier20Days:       3,
	TtRTargetTier10Days:       7,
	TtRWindowGameDays:         7,
	LogisticsStuckRounds:      3000,
	LogisticsStuckMultiplier:  0.4,
	ScoreWeightStock:          0.40,
	ScoreWeightInput:          0.30,
	ScoreWeightThroughput:     0.20,
	ScoreWeightShopGold:       0.10,
	RestockCadenceTier50Hours: 1,
	RestockCadenceTier40Hours: 2,
	RestockCadenceTier30Hours: 6,
	RestockCadenceTier20Hours: 24,
	RestockCadenceTier10Days:  5,
}

// roundsForHours converts game-hours to rounds using testScoringCfg.
func roundsForHours(h int) uint64 {
	return uint64(h) * testScoringCfg.RoundsPerGameHour
}

// roundsForDays converts game-days to rounds using testScoringCfg.
func roundsForDays(d int) uint64 {
	return uint64(d) * 24 * testScoringCfg.RoundsPerGameHour
}

// floatNear returns true if a and b are within tol of each other.
func floatNear(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}

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
	// Five-axis formula: overall = weighted(stock + throughput + input + gold),
	// logistics NOT blended in. With a single 50%-stocked shop and no history
	// (using testScoringCfg weights):
	//   MeanStock      = 50   (5/10 fill)
	//   MeanThroughput = 100  (no depletion events → "always available")
	//   MeanShopGold   = 100  (startingGold=0 → no-penalize path)
	//   MeanInput      = 100  (bootstrap — cur.Round=0, roundsPerDay=240 → no prev)
	// Weights: Stock=0.40, Input=0.30, Throughput=0.20, ShopGold=0.10, sum=1.0
	// OverallScore = (0.40*50 + 0.20*100 + 0.30*100 + 0.10*100) / 1.0 = 80
	const expectedOverall = 80.0

	cur := health.Snapshot{
		Shops: []health.ShopSnapshot{
			{Zone: "blacksmithing_zone", CraftSupport: "blacksmithing",
				Stock: []health.StockSnapshot{{ItemId: 1, Tier: 50, RestockQty: 1, Current: 5, Max: 10}}},
		},
		Caravans: []health.CaravanSnapshot{{InstId: 1, State: "thornwall_dwell"}},
		Foragers: []health.ForagerSnapshot{{InstId: 7, State: "resting"}},
	}

	// Build 24 history entries (satisfies minHistoryForCycles).
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

	// Use ScoreWithConfig so the test exercises testScoringCfg, not the live
	// global balance config. This is the test-isolation fix from Phase 3 review.
	scores := health.ScoreWithConfig(&cur, hist, testScoringCfg)

	// MeanShop is back-compat alias for MeanStock.
	if math.Abs(scores.MeanShop-scores.MeanStock) > 0.01 {
		t.Errorf("MeanShop %.2f != MeanStock %.2f — back-compat alias broken", scores.MeanShop, scores.MeanStock)
	}
	if math.Abs(scores.MeanStock-50) > 0.5 {
		t.Errorf("MeanStock: got %.2f, want ~50", scores.MeanStock)
	}
	// Assert against the exact predicted value (±1 point tolerance).
	if !floatNear(scores.OverallScore, expectedOverall, 1.0) {
		t.Errorf("OverallScore: got %.2f, want %.2f ±1 (0.40*50 + 0.20*100 + 0.30*100 + 0.10*100)",
			scores.OverallScore, expectedOverall)
	}
	// PerShop rows carry individual scores.
	if scores.PerShop[0].Score < 49 || scores.PerShop[0].Score > 51 {
		t.Errorf("PerShop[0].Score: got %.2f, want ~50", scores.PerShop[0].Score)
	}
}

// ─── ThroughputScore tests ────────────────────────────────────────────────────

func TestThroughputScore_NoEventsEverIsHundred(t *testing.T) {
	// A shop with stock but no depletion history reads as "always
	// available" — score 100.
	snap := health.ShopSnapshot{
		Stock: []health.StockSnapshot{
			{ItemId: 100, Tier: 50, Current: 5, Max: 10},
		},
	}
	got := health.ThroughputScore(snap, 1000, testScoringCfg)
	if got < 99 {
		t.Errorf("no-history shop = %.1f, want ~100", got)
	}
}

func TestThroughputScore_FastRefillIsHundred(t *testing.T) {
	// One tier-50 item, ttr = 1h, target = 3h → score >=95.
	snap := health.ShopSnapshot{
		Stock: []health.StockSnapshot{{ItemId: 100, Tier: 50, Current: 5, Max: 10}},
		StockEvents: map[int][]health.StockEvent{
			100: {{DepletedRound: 1000, RefilledRound: 1000 + roundsForHours(1)}},
		},
	}
	got := health.ThroughputScore(snap, 9999, testScoringCfg)
	if got < 95 {
		t.Errorf("1h ttr vs 3h target = %.1f, want >=95", got)
	}
}

func TestThroughputScore_SlowRefillIsLow(t *testing.T) {
	// tier-50 item, ttr = 6h, target = 3h → score should be near 0.
	// currentRound must be within the 7-day window so the event is included.
	// Event: depleted=1000, refilled=1060 (6h later at 10 rounds/hr).
	// Window = 7*24*10 = 1680 rounds. Use currentRound=2000 so
	// windowStart=320 and refilled=1060 >= 320 (event in window).
	snap := health.ShopSnapshot{
		Stock: []health.StockSnapshot{{ItemId: 100, Tier: 50, Current: 0, Max: 10}},
		StockEvents: map[int][]health.StockEvent{
			100: {{DepletedRound: 1000, RefilledRound: 1000 + roundsForHours(6)}},
		},
	}
	got := health.ThroughputScore(snap, 2000, testScoringCfg)
	if got > 5 {
		t.Errorf("6h ttr vs 3h target = %.1f, want <=5", got)
	}
}

func TestThroughputScore_CurrentlyDepletedDragsScore(t *testing.T) {
	// item depleted, no completed events; ongoing 4h-and-counting
	// should drag score below 35 for tier-50.
	snap := health.ShopSnapshot{
		Stock:            []health.StockSnapshot{{ItemId: 100, Tier: 50, Current: 0, Max: 10}},
		CurrentDepletion: map[int]uint64{100: 1000},
	}
	now := uint64(1000) + roundsForHours(4)
	got := health.ThroughputScore(snap, now, testScoringCfg)
	if got > 35 {
		t.Errorf("ongoing 4h depletion vs 3h target tier-50 = %.1f, want <=35", got)
	}
}

// ─── InputRateScore tests ─────────────────────────────────────────────────────

func TestInputRateScore_BootstrapNoHistory(t *testing.T) {
	// Empty zone with no input → score 100 (bootstrap).
	got := health.InputRateScore("stillwater", &health.Snapshot{}, []*health.Snapshot{}, testScoringCfg)
	if got < 99 {
		t.Errorf("empty bootstrap = %.1f, want ~100", got)
	}
}

func TestInputRateScore_LowInputIsLow(t *testing.T) {
	// Snapshot history shows 1 day passing with very few items in.
	cur := &health.Snapshot{
		Round: roundsForDays(2),
		Shops: []health.ShopSnapshot{
			{Zone: "stillwater", RestockCount: 5,
				Stock: []health.StockSnapshot{{ItemId: 1, Tier: 50, Current: 5, Max: 10}}},
		},
	}
	prev := &health.Snapshot{
		Round: roundsForDays(1),
		Shops: []health.ShopSnapshot{
			{Zone: "stillwater", RestockCount: 0,
				Stock: []health.StockSnapshot{{ItemId: 1, Tier: 50, Current: 5, Max: 10}}},
		},
	}
	got := health.InputRateScore("stillwater", cur, []*health.Snapshot{prev, cur}, testScoringCfg)
	if got > 30 {
		t.Errorf("low input = %.1f, want <=30", got)
	}
}

func TestInputRateScore_HighInputIsHigh(t *testing.T) {
	// Healthy zone: one game-day has elapsed, shops restocked frequently,
	// and a forager made deliveries.  Score should reach >=80.
	//
	// testScoringCfg: RoundsPerGameHour=10 → roundsPerDay=240.
	// Tier-50 cadence=1h → target contribution per item = 24/1 * 50 = 1200/day.
	// We place 1 shop with 1 tier-50 item.  delta restocks = 30 (30 cycles in
	// one game-day is plausible for a busy common-goods vendor).
	// inputWeighted = 30 * 50 = 1500   (shop contribution)
	// A forager with CargoCapacity=10 in the zone contributes
	//   target += 3 * 10 * 30/100 = 9 (to target, not to actual deliveries)
	// For deliveries: tier-50, 24 actual deliveries → +24*50 = 1200.
	//
	// target = 1200 (shop) + 9 (forager passive target) = 1209
	// actual = 1500 (shop restocks) + 1200 (forager deliveries) = 2700
	// score  = min(100, 2700/1209*100) = 100
	//
	// Even with the forager target contribution, actual easily exceeds it.
	// Assert >=80 to tolerate slight rounding in zoneInputTarget.

	prevRound := roundsForDays(1)
	curRound := roundsForDays(2)

	cur := &health.Snapshot{
		Round: curRound,
		Shops: []health.ShopSnapshot{
			{Zone: "stillwater", MobId: 1, RestockCount: 30,
				Stock: []health.StockSnapshot{{ItemId: 1, Tier: 50, Current: 5, Max: 10}}},
		},
		Foragers: []health.ForagerSnapshot{
			{MobId: 99, Territory: "stillwater", CargoCapacity: 10,
				State: "resting",
				DeliveriesByTier: map[int]int{50: 24}},
		},
	}
	prev := &health.Snapshot{
		Round: prevRound,
		Shops: []health.ShopSnapshot{
			{Zone: "stillwater", MobId: 1, RestockCount: 0,
				Stock: []health.StockSnapshot{{ItemId: 1, Tier: 50, Current: 5, Max: 10}}},
		},
		Foragers: []health.ForagerSnapshot{
			{MobId: 99, Territory: "stillwater", CargoCapacity: 10,
				State: "resting",
				DeliveriesByTier: map[int]int{50: 0}},
		},
	}

	got := health.InputRateScore("stillwater", cur, []*health.Snapshot{prev, cur}, testScoringCfg)
	if got < 80 {
		t.Errorf("high-input healthy zone = %.1f, want >=80", got)
	}
}

// ─── LogisticsHealth tests ────────────────────────────────────────────────────

func TestLogisticsHealth_DespawnedIsZero(t *testing.T) {
	got := health.LogisticsHealth(health.LogisticsArgs{Despawned: true}, testScoringCfg)
	if got != 0 {
		t.Errorf("despawned = %.1f, want 0", got)
	}
}

func TestLogisticsHealth_StuckMultiplier(t *testing.T) {
	args := health.LogisticsArgs{
		Cycles: 1, ExpectedCycles: 1,
		LbsDelivered: 5000, TargetLbs: 5000,
		Stuck: true,
	}
	got := health.LogisticsHealth(args, testScoringCfg)
	// base = 100; stuck multiplier 0.4 → 40
	if got < 38 || got > 42 {
		t.Errorf("stuck-with-perfect-flow = %.1f, want ~40", got)
	}
}

func TestLogisticsHealth_HealthyComposite(t *testing.T) {
	args := health.LogisticsArgs{
		Cycles: 1, ExpectedCycles: 1,
		LbsDelivered: 5000, TargetLbs: 5000,
	}
	got := health.LogisticsHealth(args, testScoringCfg)
	if got < 99 {
		t.Errorf("perfect = %.1f, want ~100", got)
	}
}

// ─── ShopGoldScore tests ──────────────────────────────────────────────────────

func TestShopGoldScore(t *testing.T) {
	cases := []struct {
		gold, starting int
		want           float64
	}{
		{0, 500, 0},
		{250, 500, 33.3},
		{500, 500, 66.6},
		{750, 500, 100},
		{1000, 500, 100}, // capped at 1.5×
	}
	for _, c := range cases {
		got := health.ShopGoldScore(health.ShopSnapshot{Gold: c.gold, StartingGold: c.starting})
		if !floatNear(got, c.want, 1.0) {
			t.Errorf("gold=%d/start=%d: got %.1f, want %.1f", c.gold, c.starting, got, c.want)
		}
	}
}

