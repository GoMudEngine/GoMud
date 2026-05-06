package health

import "github.com/GoMudEngine/GoMud/internal/configs"

// ScoringConfig is a small DI shim around the global balance config.
// Passing it explicitly to score functions keeps the math testable
// without spinning up the full game state (no global config reads
// inside the score functions themselves).
type ScoringConfig struct {
	// Integer division: RoundsPerDay is typically not divisible by 24.
	// Use RoundsPerDay/24 — the same truncation pattern as crafter.go.
	RoundsPerGameHour uint64

	// TtR targets per rarity tier.
	TtRTargetTier50Hours int
	TtRTargetTier40Hours int
	TtRTargetTier30Hours int
	TtRTargetTier20Days  int
	TtRTargetTier10Days  int

	// Rolling window for TtR history (game-days).
	TtRWindowGameDays int

	// Logistics health tunables.
	LogisticsStuckRounds     uint64
	LogisticsStuckMultiplier float64

	// Five-axis overall blend weights.
	ScoreWeightStock      float64
	ScoreWeightInput      float64
	ScoreWeightThroughput float64
	ScoreWeightShopGold   float64

	// Restock cadence per tier (hours). Used to derive InputRateScore
	// zone targets (expected_restocks_per_day = 24/cadence).
	RestockCadenceTier50Hours int
	RestockCadenceTier40Hours int
	RestockCadenceTier30Hours int
	RestockCadenceTier20Hours int
	RestockCadenceTier10Days  int // stored as days; multiply ×24 for hours
}

// ScoringConfigFromBalance builds a ScoringConfig from the live global
// balance + timing configs. Called once at the top of Score() so all
// score functions receive a consistent snapshot of config values.
func ScoringConfigFromBalance() ScoringConfig {
	b := configs.GetBalanceConfig()
	t := configs.GetTimingConfig()
	// Integer division: RoundsPerDay is typically not divisible by 24.
	// Truncation is fine; a few rounds' error per hour is inconsequential
	// for scoring. Matches the same pattern used in crafter.go.
	rph := uint64(t.RoundsPerDay) / 24
	if rph == 0 {
		rph = 1
	}
	return ScoringConfig{
		RoundsPerGameHour:         rph,
		TtRTargetTier50Hours:      int(b.TtRTargetTier50Hours),
		TtRTargetTier40Hours:      int(b.TtRTargetTier40Hours),
		TtRTargetTier30Hours:      int(b.TtRTargetTier30Hours),
		TtRTargetTier20Days:       int(b.TtRTargetTier20Days),
		TtRTargetTier10Days:       int(b.TtRTargetTier10Days),
		TtRWindowGameDays:         int(b.TtRWindowGameDays),
		LogisticsStuckRounds:      uint64(b.LogisticsStuckRounds),
		LogisticsStuckMultiplier:  float64(b.LogisticsStuckMultiplier),
		ScoreWeightStock:          float64(b.ScoreWeightStock),
		ScoreWeightInput:          float64(b.ScoreWeightInput),
		ScoreWeightThroughput:     float64(b.ScoreWeightThroughput),
		ScoreWeightShopGold:       float64(b.ScoreWeightShopGold),
		RestockCadenceTier50Hours: int(b.RestockCadenceTier50Hours),
		RestockCadenceTier40Hours: int(b.RestockCadenceTier40Hours),
		RestockCadenceTier30Hours: int(b.RestockCadenceTier30Hours),
		RestockCadenceTier20Hours: int(b.RestockCadenceTier20Hours),
		RestockCadenceTier10Days:  int(b.RestockCadenceTier10Days),
	}
}

// StockScore returns the per-shop weighted-fill-percentage. This is the
// dashboard's "Stock Score" — answers "is there material on the shelves
// right now?" Identical to the historical PerShopScore (kept as alias).
func StockScore(s ShopSnapshot) float64 {
	v, _ := StockScoreOpt(s)
	return v
}

// StockScoreOpt is the option-returning variant used when callers need
// to distinguish "no signal" from "0%."
func StockScoreOpt(s ShopSnapshot) (float64, bool) {
	return PerShopScoreOpt(s)
}

// PerShopScore returns a 0-100 health score for one shop, weighted by
// RestockQty. Shops with no stock entries return 0; callers that want
// to distinguish "no data" from "zero" should use PerShopScoreOpt.
func PerShopScore(s ShopSnapshot) float64 {
	v, _ := PerShopScoreOpt(s)
	return v
}

// PerShopScoreOpt returns (score, true) when the shop has stock data
// to score, or (0, false) when there is no signal.
func PerShopScoreOpt(s ShopSnapshot) (float64, bool) {
	if len(s.Stock) == 0 {
		return 0, false
	}
	var weightedSum float64
	var totalWeight float64
	for _, e := range s.Stock {
		if e.Max <= 0 {
			continue
		}
		fill := float64(e.Current) / float64(e.Max)
		if fill < 0 {
			fill = 0
		}
		if fill > 1 {
			fill = 1
		}
		weight := float64(e.RestockQty)
		if weight < 1 {
			weight = 1
		}
		weightedSum += weight * fill
		totalWeight += weight
	}
	if totalWeight == 0 {
		return 0, false
	}
	return 100 * weightedSum / totalWeight, true
}

// PerCraftSupportScores returns mean per-shop score grouped by the
// craft discipline each shop supports. Shops with empty CraftSupport
// roll into key "" (should never happen in production thanks to
// startup validation; surfaces clearly in the UI as "(uncategorized)"
// if it ever does).
func PerCraftSupportScores(snap Snapshot) map[string]float64 {
	type bucket struct {
		sum   float64
		count int
	}
	buckets := map[string]*bucket{}
	for _, s := range snap.Shops {
		score, ok := PerShopScoreOpt(s)
		if !ok {
			continue
		}
		b, exists := buckets[s.CraftSupport]
		if !exists {
			b = &bucket{}
			buckets[s.CraftSupport] = b
		}
		b.sum += score
		b.count++
	}
	out := map[string]float64{}
	for k, b := range buckets {
		if b.count > 0 {
			out[k] = b.sum / float64(b.count)
		}
	}
	return out
}

// CountCaravanCycles returns the number of ThornwallDwell→ThornwallDwell
// cycles for instId across history. History must be ordered oldest→
// newest. A cycle is one entry into thornwall_dwell preceded by a
// non-thornwall_dwell state.
func CountCaravanCycles(instId int, history []*Snapshot) int {
	return countStateReturns(history, "thornwall_dwell", func(s *Snapshot) string {
		for _, c := range s.Caravans {
			if c.InstId == instId {
				return c.State
			}
		}
		return ""
	})
}

// CountForagerCycles returns the number of Resting→Resting transitions
// for instId across history.
func CountForagerCycles(instId int, history []*Snapshot) int {
	return countStateReturns(history, "resting", func(s *Snapshot) string {
		for _, f := range s.Foragers {
			if f.InstId == instId {
				return f.State
			}
		}
		return ""
	})
}

func countStateReturns(history []*Snapshot, target string, lookup func(*Snapshot) string) int {
	cycles := 0
	prev := ""
	for _, s := range history {
		cur := lookup(s)
		if cur == target && prev != "" && prev != target {
			cycles++
		}
		if cur != "" {
			prev = cur
		}
	}
	return cycles
}

// PerCaravanScore returns (score, true) if there's enough history to
// compute. Insufficient history (fewer than minHistory entries) returns
// (_, false). Score = cycleScore - stuckPenalty, clamped to [0, 100].
//
// The hardcoded constants below (minHistoryForCycles, stuckThresholdRounds,
// stuckPenalty) and the per-second cycle cadences (24h for caravans, 8h
// for foragers) are MVP shortcuts — the design spec calls for these to
// be config-driven, but the underlying per-state expected durations
// aren't yet a single tunable knob. See plan Task 10 for context.
// Tracked in MEMORY.md under "Economy dashboard followups".
const (
	minHistoryForCycles  = 24   // ~24 hourly samples = 1 day baseline
	stuckThresholdRounds = 5000 // any state held longer than this triggers the penalty (MVP fixed)
	stuckPenalty         = 30   // points deducted when stuck
)

func PerCaravanScore(instId int, cur Snapshot, history []*Snapshot) (float64, bool) {
	if len(history) < minHistoryForCycles {
		return 0, false
	}
	cycles := CountCaravanCycles(instId, history)
	expectedPerWindow := float64(len(history)) / 24.0 // MVP: 1 cycle/day; wire to config later
	if expectedPerWindow <= 0 {
		expectedPerWindow = 1
	}
	score := 100 * float64(cycles) / expectedPerWindow
	if score > 100 {
		score = 100
	}

	// Stuck penalty: any caravan whose current state has been held
	// longer than stuckThresholdRounds loses points.
	for _, c := range cur.Caravans {
		if c.InstId == instId && c.StateEnteredRound > 0 {
			if cur.Round > c.StateEnteredRound &&
				cur.Round-c.StateEnteredRound > stuckThresholdRounds {
				score -= stuckPenalty
			}
			break
		}
	}
	if score < 0 {
		score = 0
	}
	return score, true
}

// PerForagerScore mirrors PerCaravanScore for foragers (Resting→Resting
// cycle counting + same stuck-penalty logic).
func PerForagerScore(instId int, cur Snapshot, history []*Snapshot) (float64, bool) {
	if len(history) < minHistoryForCycles {
		return 0, false
	}
	cycles := CountForagerCycles(instId, history)
	expectedPerWindow := float64(len(history)) / 8.0 // MVP: ~3 cycles/day; wire to config later
	if expectedPerWindow <= 0 {
		expectedPerWindow = 1
	}
	score := 100 * float64(cycles) / expectedPerWindow
	if score > 100 {
		score = 100
	}

	for _, f := range cur.Foragers {
		if f.InstId == instId && f.StateEnteredRound > 0 {
			if cur.Round > f.StateEnteredRound &&
				cur.Round-f.StateEnteredRound > stuckThresholdRounds {
				score -= stuckPenalty
			}
			break
		}
	}
	if score < 0 {
		score = 0
	}
	return score, true
}

// ── ThroughputScore ─────────────────────────────────────────────────────────

// ttrTargetRounds returns the TtR target in rounds for a rarity tier,
// reading the configured per-tier values from cfg.
func ttrTargetRounds(tier int, cfg ScoringConfig) uint64 {
	rph := cfg.RoundsPerGameHour
	switch tier {
	case 50:
		return uint64(cfg.TtRTargetTier50Hours) * rph
	case 40:
		return uint64(cfg.TtRTargetTier40Hours) * rph
	case 30:
		return uint64(cfg.TtRTargetTier30Hours) * rph
	case 20:
		return uint64(cfg.TtRTargetTier20Days) * 24 * rph
	case 10:
		return uint64(cfg.TtRTargetTier10Days) * 24 * rph
	default:
		return 0
	}
}

// itemTtRScore returns the 0..1 throughput score for one item.
// Combines completed events in window with any ongoing depletion.
// Items with no signal score 1.0 (always-available is the best case).
func itemTtRScore(itemId int, tier int, snap ShopSnapshot, currentRound uint64, cfg ScoringConfig) float64 {
	target := ttrTargetRounds(tier, cfg)
	if target == 0 {
		return 1.0 // unknown tier: don't penalize
	}
	windowRounds := uint64(cfg.TtRWindowGameDays) * 24 * cfg.RoundsPerGameHour
	var windowStart uint64
	if currentRound > windowRounds {
		windowStart = currentRound - windowRounds
	}

	var sumTtR uint64
	var n int
	for _, ev := range snap.StockEvents[itemId] {
		if ev.RefilledRound < windowStart {
			continue
		}
		sumTtR += ev.RefilledRound - ev.DepletedRound
		n++
	}
	// Currently depleted contributes ongoing duration as a partial event.
	if depRound, ok := snap.CurrentDepletion[itemId]; ok && currentRound > depRound {
		sumTtR += currentRound - depRound
		n++
	}
	if n == 0 {
		return 1.0
	}
	mean := float64(sumTtR) / float64(n)
	score := 1.0 - mean/float64(target)
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

// ThroughputScore returns the per-shop time-to-refill (TtR) score,
// 0-100. Items weighted by rarity_tier^2 so commons (tier 50) drive
// the score 25× harder than rares (tier 10). Items with no history
// and not currently depleted score 100 (always-available = best case).
func ThroughputScore(snap ShopSnapshot, currentRound uint64, cfg ScoringConfig) float64 {
	var weighted, totalWeight float64
	for _, e := range snap.Stock {
		w := float64(e.Tier * e.Tier)
		if w <= 0 {
			continue
		}
		s := itemTtRScore(e.ItemId, e.Tier, snap, currentRound, cfg)
		weighted += w * s
		totalWeight += w
	}
	if totalWeight == 0 {
		return 100
	}
	return 100 * weighted / totalWeight
}

// ── InputRateScore ───────────────────────────────────────────────────────────

// cadenceHoursForTier returns the configured restock cadence in hours
// for a rarity tier. Returns 0 for unrecognized tiers.
func cadenceHoursForTier(tier int, cfg ScoringConfig) int {
	switch tier {
	case 50:
		return cfg.RestockCadenceTier50Hours
	case 40:
		return cfg.RestockCadenceTier40Hours
	case 30:
		return cfg.RestockCadenceTier30Hours
	case 20:
		return cfg.RestockCadenceTier20Hours
	case 10:
		return cfg.RestockCadenceTier10Days * 24
	default:
		return 0
	}
}

// meanRarityTier returns the mean rarity tier across all stock entries.
// Returns 30 (default proxy) when the slice is empty.
func meanRarityTier(stock []StockSnapshot) int {
	if len(stock) == 0 {
		return 30
	}
	sum := 0
	for _, e := range stock {
		sum += e.Tier
	}
	return sum / len(stock)
}

// zoneInputTarget returns the expected weighted items/day for a zone,
// derived automatically from active shops and foragers.
func zoneInputTarget(zone string, cur *Snapshot, cfg ScoringConfig) float64 {
	var target float64
	for _, s := range cur.Shops {
		if s.Zone != zone {
			continue
		}
		for _, e := range s.Stock {
			ch := cadenceHoursForTier(e.Tier, cfg)
			if ch == 0 {
				continue
			}
			target += (24.0 / float64(ch)) * float64(e.Tier)
		}
	}
	// Forager: ~3 deliveries/day at cargo capacity, mean tier ~30.
	// Use Territory to approximate zone membership (foragers don't have
	// a Zone field on the snapshot — territory is the zone proxy).
	for _, f := range cur.Foragers {
		if f.Territory == "" {
			continue
		}
		// Only count active foragers.
		if f.State == "(despawned)" || f.State == "(not active)" {
			continue
		}
		// Territory zone matching: check if territory starts with zone prefix.
		// E.g. zone="stillwater" territory="stillwater_marsh".
		if !territoryMatchesZone(f.Territory, zone) {
			continue
		}
		if f.CargoCapacity > 0 {
			target += 3.0 * float64(f.CargoCapacity) * 30.0 / 100.0
		}
	}
	return target
}

// territoryMatchesZone checks if the forager's territory string belongs
// to a zone. Foragers don't have a Zone field, so we compare the
// territory prefix to the zone name (underscore-joined).
func territoryMatchesZone(territory, zone string) bool {
	if territory == zone {
		return true
	}
	// territory like "stillwater_marsh" should match zone "stillwater"
	if len(territory) > len(zone) && territory[:len(zone)] == zone &&
		territory[len(zone)] == '_' {
		return true
	}
	return false
}

// InputRateScore measures items entering a zone's supply per game-day,
// weighted linearly by rarity_tier (commons dominate volume naturally).
// Returns 100 on bootstrap (no history) — matches the spec migration note.
func InputRateScore(zone string, cur *Snapshot, history []*Snapshot, cfg ScoringConfig) float64 {
	if cur == nil || len(history) < 2 {
		return 100
	}
	// Find the snapshot ~24h back.
	roundsPerDay := uint64(24) * cfg.RoundsPerGameHour
	if cur.Round < roundsPerDay {
		return 100 // not enough history
	}
	cutoff := cur.Round - roundsPerDay
	var prev *Snapshot
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Round <= cutoff {
			prev = history[i]
			break
		}
	}
	if prev == nil {
		return 100
	}

	// Sum weighted input over the window.
	var inputWeighted float64
	for _, s := range cur.Shops {
		if s.Zone != zone {
			continue
		}
		var prevRestock int
		for _, p := range prev.Shops {
			if p.Zone == s.Zone && p.MobId == s.MobId {
				prevRestock = p.RestockCount
				break
			}
		}
		delta := s.RestockCount - prevRestock
		if delta < 0 {
			delta = 0
		}
		mt := meanRarityTier(s.Stock)
		inputWeighted += float64(delta) * float64(mt)
	}
	// Forager deliveries by tier.
	for _, f := range cur.Foragers {
		if !territoryMatchesZone(f.Territory, zone) {
			continue
		}
		var prevDeliv map[int]int
		for _, p := range prev.Foragers {
			if p.MobId == f.MobId {
				prevDeliv = p.DeliveriesByTier
				break
			}
		}
		for tier, n := range f.DeliveriesByTier {
			d := n - prevDeliv[tier]
			if d < 0 {
				d = 0
			}
			inputWeighted += float64(d) * float64(tier)
		}
	}

	target := zoneInputTarget(zone, cur, cfg)
	if target <= 0 {
		return 100
	}
	score := 100 * inputWeighted / target
	if score > 100 {
		return 100
	}
	if score < 0 {
		return 0
	}
	return score
}

// ── LogisticsHealth ─────────────────────────────────────────────────────────

// LogisticsArgs is the input bundle for LogisticsHealth, decoupled from
// caravan/forager structs so the same function scores both entities.
type LogisticsArgs struct {
	Cycles         int
	ExpectedCycles int
	LbsDelivered   uint64
	TargetLbs      uint64
	Stuck          bool
	Despawned      bool
}

// LogisticsHealth returns a 0-100 composite of cycle rate and cargo
// flow, with a hard multiplier for stuck (×LogisticsStuckMultiplier,
// default 0.4) and 0 for despawned. Used identically for caravans and
// foragers so the caller constructs the args and this function stays
// entity-agnostic.
func LogisticsHealth(a LogisticsArgs, cfg ScoringConfig) float64 {
	if a.Despawned {
		return 0
	}
	cycleScore := 0.0
	if a.ExpectedCycles > 0 {
		cycleScore = 100.0 * float64(a.Cycles) / float64(a.ExpectedCycles)
		if cycleScore > 100 {
			cycleScore = 100
		}
	}
	cargoScore := 0.0
	if a.TargetLbs > 0 {
		cargoScore = 100.0 * float64(a.LbsDelivered) / float64(a.TargetLbs)
		if cargoScore > 100 {
			cargoScore = 100
		}
	}
	base := 0.5*cycleScore + 0.5*cargoScore
	if a.Stuck {
		base *= cfg.LogisticsStuckMultiplier
	}
	if base < 0 {
		return 0
	}
	if base > 100 {
		return 100
	}
	return base
}

// ── ShopGoldScore ────────────────────────────────────────────────────────────

// ShopGoldScore returns 0-100 based on merchant liquidity.
// gold/starting ratio clamped to [0, 1.5], then mapped linearly so
// 1.5× starting gold = 100, 1.0× = ~67, 0 = 0. Captures the "merchant
// can't buy from players" failure mode invisible in the stock score.
func ShopGoldScore(s ShopSnapshot) float64 {
	if s.StartingGold <= 0 {
		return 100 // no baseline → don't penalize
	}
	ratio := float64(s.Gold) / float64(s.StartingGold)
	if ratio > 1.5 {
		ratio = 1.5
	}
	if ratio < 0 {
		ratio = 0
	}
	return ratio / 1.5 * 100
}

// ── Scores struct and Score() ────────────────────────────────────────────────

// Scores is the bundle returned by Score(). Each per-entity score
// has a HasScore flag for "insufficient history" cases.
type Scores struct {
	OverallScore float64

	PerShop         []ShopScoreRow
	PerCraftSupport map[string]float64
	PerCaravan      []EntityScoreRow
	PerForager      []EntityScoreRow

	// Five-axis component aggregates surfaced as the five top cards.
	MeanStock      float64
	MeanThroughput float64
	MeanInput      float64
	MeanShopGold   float64

	// Back-compat aliases — legacy dashboard JS and tests reference
	// MeanShop/MeanCaravan/MeanForager; keep them until Phase 4 lands.
	MeanShop    float64 // == MeanStock
	MeanCaravan float64
	MeanForager float64
}

type ShopScoreRow struct {
	Zone            string
	MobId           int
	RoomId          int
	Name            string
	CraftSupport    string
	Score           float64 // back-compat alias for StockScore
	StockScore      float64
	ThroughputScore float64
	GoldScore       float64
	HasScore        bool
}

type EntityScoreRow struct {
	InstId   int
	Name     string
	Score    float64
	HasScore bool
}

// Score is the dashboard's main scoring entry point. It builds a
// ScoringConfig once and threads it through all five score axes.
func Score(cur *Snapshot, history []*Snapshot) Scores {
	out := Scores{}
	if cur == nil {
		return out
	}

	cfg := ScoringConfigFromBalance()

	// ── Per-shop: stock, throughput, gold ───────────────────────────────────
	out.PerShop = make([]ShopScoreRow, 0, len(cur.Shops))
	var stockSum, throughputSum, goldSum float64
	var shopCount int
	for _, s := range cur.Shops {
		stockScore, ok := PerShopScoreOpt(s)
		thrScore := ThroughputScore(s, cur.Round, cfg)
		gldScore := ShopGoldScore(s)
		out.PerShop = append(out.PerShop, ShopScoreRow{
			Zone:            s.Zone,
			MobId:           s.MobId,
			RoomId:          s.RoomId,
			Name:            s.Name,
			CraftSupport:    s.CraftSupport,
			Score:           stockScore, // back-compat alias
			StockScore:      stockScore,
			ThroughputScore: thrScore,
			GoldScore:       gldScore,
			HasScore:        ok,
		})
		if ok {
			stockSum += stockScore
			throughputSum += thrScore
			goldSum += gldScore
			shopCount++
		}
	}
	if shopCount > 0 {
		out.MeanStock = stockSum / float64(shopCount)
		out.MeanThroughput = throughputSum / float64(shopCount)
		out.MeanShopGold = goldSum / float64(shopCount)
	}
	// Back-compat: MeanShop == MeanStock.
	out.MeanShop = out.MeanStock

	out.PerCraftSupport = PerCraftSupportScores(*cur)

	// ── Per-zone input rate ──────────────────────────────────────────────────
	// Collect unique zones represented in current shops.
	zoneSet := map[string]struct{}{}
	for _, s := range cur.Shops {
		if s.Zone != "" {
			zoneSet[s.Zone] = struct{}{}
		}
	}
	var inputSum float64
	var inputZones int
	for zone := range zoneSet {
		score := InputRateScore(zone, cur, history, cfg)
		inputSum += score
		inputZones++
	}
	if inputZones > 0 {
		out.MeanInput = inputSum / float64(inputZones)
	}

	// ── Per-caravan: LogisticsHealth ─────────────────────────────────────────
	out.PerCaravan = make([]EntityScoreRow, 0, len(cur.Caravans))
	var caravanSum float64
	var caravanCount int
	for _, c := range cur.Caravans {
		cycles := CountCaravanCycles(c.InstId, history)
		expected := len(history) / 24
		if expected < 1 {
			expected = 1
		}

		// LbsDelivered delta over the window (guard uint64 underflow).
		var deliveredInWindow uint64
		if len(history) > 0 {
			earliest := history[0]
			for _, ec := range earliest.Caravans {
				if ec.InstId == c.InstId {
					if c.LbsDelivered >= ec.LbsDelivered {
						deliveredInWindow = c.LbsDelivered - ec.LbsDelivered
					}
					break
				}
			}
		}
		targetLbs := uint64(expected) * uint64(c.CargoCapacity)

		stuck := cur.Round > c.StateEnteredRound &&
			c.StateEnteredRound > 0 &&
			cur.Round-c.StateEnteredRound > cfg.LogisticsStuckRounds

		despawned := c.State == "(not active)" || c.State == "(despawned)"

		args := LogisticsArgs{
			Cycles:         cycles,
			ExpectedCycles: expected,
			LbsDelivered:   deliveredInWindow,
			TargetLbs:      targetLbs,
			Stuck:          stuck,
			Despawned:      despawned,
		}
		score := LogisticsHealth(args, cfg)
		ok := len(history) >= minHistoryForCycles
		out.PerCaravan = append(out.PerCaravan, EntityScoreRow{
			InstId: c.InstId, Name: c.Name, Score: score, HasScore: ok,
		})
		if ok {
			caravanSum += score
			caravanCount++
		}
	}
	if caravanCount > 0 {
		out.MeanCaravan = caravanSum / float64(caravanCount)
	}

	// ── Per-forager: LogisticsHealth ─────────────────────────────────────────
	out.PerForager = make([]EntityScoreRow, 0, len(cur.Foragers))
	var foragerSum float64
	var foragerCount int
	for _, f := range cur.Foragers {
		cycles := CountForagerCycles(f.InstId, history)
		expected := len(history) / 8
		if expected < 1 {
			expected = 1
		}

		var deliveredInWindow uint64
		if len(history) > 0 {
			earliest := history[0]
			for _, ef := range earliest.Foragers {
				if ef.InstId == f.InstId {
					if f.LbsDelivered >= ef.LbsDelivered {
						deliveredInWindow = f.LbsDelivered - ef.LbsDelivered
					}
					break
				}
			}
		}
		targetLbs := uint64(expected) * uint64(f.CargoCapacity)

		stuck := cur.Round > f.StateEnteredRound &&
			f.StateEnteredRound > 0 &&
			cur.Round-f.StateEnteredRound > cfg.LogisticsStuckRounds

		despawned := f.State == "(despawned)" || f.State == "(not active)"

		args := LogisticsArgs{
			Cycles:         cycles,
			ExpectedCycles: expected,
			LbsDelivered:   deliveredInWindow,
			TargetLbs:      targetLbs,
			Stuck:          stuck,
			Despawned:      despawned,
		}
		score := LogisticsHealth(args, cfg)
		ok := len(history) >= minHistoryForCycles
		out.PerForager = append(out.PerForager, EntityScoreRow{
			InstId: f.InstId, Name: f.Name, Score: score, HasScore: ok,
		})
		if ok {
			foragerSum += score
			foragerCount++
		}
	}
	if foragerCount > 0 {
		out.MeanForager = foragerSum / float64(foragerCount)
	}

	// ── Overall: five-axis weighted blend ────────────────────────────────────
	// Renormalize over components with data (bootstrap-safe).
	// Logistics is NOT included in overall — failures propagate downstream
	// through input rate already.
	var weightedSum, weightSum float64
	if shopCount > 0 {
		weightedSum += cfg.ScoreWeightStock * out.MeanStock
		weightSum += cfg.ScoreWeightStock
		weightedSum += cfg.ScoreWeightThroughput * out.MeanThroughput
		weightSum += cfg.ScoreWeightThroughput
		weightedSum += cfg.ScoreWeightShopGold * out.MeanShopGold
		weightSum += cfg.ScoreWeightShopGold
	}
	if inputZones > 0 {
		weightedSum += cfg.ScoreWeightInput * out.MeanInput
		weightSum += cfg.ScoreWeightInput
	}
	if weightSum > 0 {
		out.OverallScore = weightedSum / weightSum
	}

	return out
}
