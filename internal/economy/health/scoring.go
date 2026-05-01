package health

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
