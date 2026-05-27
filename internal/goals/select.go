package goals

import (
	"fmt"
	"sort"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// SelectReason explains why the selector picked what it did. Surfaced
// in the structured switch log line and the `goal scores` admin output.
type SelectReason struct {
	// Kind is one of: "no_goals", "kept_no_candidates",
	// "kept_top_unchanged", "kept_hysteresis_margin",
	// "kept_hysteresis_min_hold", "switched", "switched_prev_invalid".
	Kind string
	// Detail is a free-form human-readable explanation, e.g.
	// "g3(80) beat current g1(70) by 10pts after min-hold".
	Detail string
}

// Select is the pure heart of chunk 4.2's strategic layer.
//
// Inputs:
//   - goals: the mob's current goal list (from GoalsOf or a snapshot)
//   - weights: per-archetype goal-type weight multipliers (may be nil; defaults to 1.0)
//   - mob: live mob instance — passed to registered ContextScore funcs
//   - prev: the mob's current goal as of last Recompute, or nil if none
//   - currentSinceRound: round when prev became current (0 if prev nil)
//   - lastSwitchRound: round of most recent switch
//   - nowRound: util.GetRoundCount() at the moment of selection
//
// Outputs:
//   - current: the chosen goal (may equal prev)
//   - switched: true iff current != prev (or prev was nil and current is not)
//   - reason: structured explanation
//
// Select is lock-free and side-effect free. Recompute is the orchestrator
// that calls Select and applies the side effects.
func Select(
	goals []*Goal,
	weights map[string]float64,
	mob *mobs.Mob,
	prev *Goal,
	currentSinceRound, lastSwitchRound, nowRound uint64,
) (current *Goal, switched bool, reason SelectReason) {
	now := nowTime()

	// Step 1: filter candidates.
	candidates := make([]*Goal, 0, len(goals))
	for _, g := range goals {
		if IsSatisfied(g, mob) {
			continue
		}
		if IsExpired(g, now) {
			continue
		}
		if effectiveContextMod(g, mob) == 0 {
			continue
		}
		candidates = append(candidates, g)
	}

	// Step 2 / 3: empty candidates branches.
	prevValid := prev != nil && goalInList(prev, goals) &&
		!IsSatisfied(prev, mob) && !IsExpired(prev, now)
	if len(candidates) == 0 {
		if prevValid {
			return prev, false, SelectReason{
				Kind: "kept_no_candidates",
				Detail: fmt.Sprintf("all %d goal(s) filtered out (satisfied/expired/contextMod=0); prev g%s still valid",
					len(goals), prev.Id),
			}
		}
		return nil, prev != nil, SelectReason{Kind: "no_goals"}
	}

	// Step 4: pick top scorer with stable tie-break.
	sort.SliceStable(candidates, func(i, j int) bool {
		si := effectiveScore(candidates[i], weights, mob)
		sj := effectiveScore(candidates[j], weights, mob)
		if si != sj {
			return si > sj
		}
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority > candidates[j].Priority
		}
		return candidates[i].Id < candidates[j].Id
	})
	top := candidates[0]

	// Step 5: prev nil / prev invalid → switch.
	if prev == nil {
		return top, true, SelectReason{
			Kind:   "switched",
			Detail: fmt.Sprintf("g%s won fresh selection (score=%.2f)", top.Id, effectiveScore(top, weights, mob)),
		}
	}
	if !prevValid {
		return top, true, SelectReason{
			Kind: "switched_prev_invalid",
			Detail: fmt.Sprintf("prev g%s no longer valid (removed/satisfied/expired); g%s wins (score=%.2f)",
				prev.Id, top.Id, effectiveScore(top, weights, mob)),
		}
	}

	// Step 6: top == prev → keep.
	if top == prev {
		return prev, false, SelectReason{
			Kind:   "kept_top_unchanged",
			Detail: fmt.Sprintf("g%s remains top (score=%.2f)", prev.Id, effectiveScore(prev, weights, mob)),
		}
	}

	// Step 7: hysteresis gates.
	heldRounds := uint64(0)
	if nowRound > currentSinceRound {
		heldRounds = nowRound - currentSinceRound
	}
	minHold := uint64(minHoldRoundsConfig())
	if heldRounds < minHold {
		return prev, false, SelectReason{
			Kind:   "kept_hysteresis_min_hold",
			Detail: fmt.Sprintf("g%s wants to displace g%s but held only %d/%d rounds", top.Id, prev.Id, heldRounds, minHold),
		}
	}
	topScore := effectiveScore(top, weights, mob)
	prevScore := effectiveScore(prev, weights, mob)
	scoreGap := topScore - prevScore
	margin := switchMarginConfig()
	if scoreGap < margin {
		return prev, false, SelectReason{
			Kind: "kept_hysteresis_margin",
			Detail: fmt.Sprintf("g%s(%.2f) beat g%s(%.2f) by only %.2fpts; margin %.2f required",
				top.Id, topScore, prev.Id, prevScore, scoreGap, margin),
		}
	}
	return top, true, SelectReason{
		Kind: "switched",
		Detail: fmt.Sprintf("g%s(%.2f) displaced g%s(%.2f) by %.2fpts after %d held rounds",
			top.Id, topScore, prev.Id, prevScore, scoreGap, heldRounds),
	}
}

// effectiveScore is the 4.2 scoring formula:
//
//	score = priority × archetypeWeight × contextMod
//
// archetypeWeight defaults to 1.0 when type is unlisted in weights map.
// contextMod defaults to 1.0 when no ContextScore is registered for type.
func effectiveScore(g *Goal, weights map[string]float64, mob *mobs.Mob) float64 {
	w := 1.0
	if v, ok := weights[g.Type]; ok {
		w = v
	}
	return float64(g.Priority) * w * effectiveContextMod(g, mob)
}

// effectiveContextMod looks up the registered ContextScore for g.Type
// (if any) and invokes it under panic recovery (Task 3 wires the
// recovery wrapper). Returns 1.0 if no hook is registered.
func effectiveContextMod(g *Goal, mob *mobs.Mob) float64 {
	meta, ok := lookupMeta(g.Type)
	if !ok || meta.ContextScore == nil {
		return 1.0
	}
	return invokeContextScore(meta.ContextScore, g, mob)
}

// goalInList reports whether g is present in the slice by id-equality.
func goalInList(g *Goal, slice []*Goal) bool {
	for _, x := range slice {
		if x.Id == g.Id {
			return true
		}
	}
	return false
}

// ── Stubs filled in by later tasks ──────────────────────────────────

// nowTime is the time-source seam. Tests can override via package-level
// var if a deterministic clock is needed (no test relies on this yet).
var nowTime = func() time.Time { return time.Now().UTC() }

// invokeContextScore is the panic-recovered wrapper around a registered
// ContextScoreFn. Task 3 fills it in; for now, call directly (no recovery).
func invokeContextScore(fn ContextScoreFn, g *Goal, mob *mobs.Mob) float64 {
	return fn(g, mob)
}

// switchMarginConfig reads GoalSelectSwitchMargin. Task 9 wires the
// real config knob; default mirrors the spec's default of 5.0.
func switchMarginConfig() float64 { return 5.0 }

// minHoldRoundsConfig reads GoalSelectMinHoldRounds. Task 9 wires the
// real config knob; default mirrors the spec's default of 100.
func minHoldRoundsConfig() int { return 100 }
