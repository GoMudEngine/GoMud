// Package-level file for chunk 4b-fixup outcome resolution.
// Replaces the chunk-4b ControlLevel drift-needle model with a
// per-round dispatcher that maps the drift roll z-score directly
// to a position change.
package position

// OutcomeTier represents the magnitude bucket of a drift roll's
// |z-score|. Sign of z determines whether the outcome is controller-
// favorable (Advance) or defender-favorable (Degrade / Reverse /
// Escape).
type OutcomeTier int

const (
	TierHold      OutcomeTier = iota // |z| < 0.5
	TierOneStep                      // 0.5 <= |z| < 1.0
	TierTwoStep                      // 1.0 <= |z| < 2.0
	TierThreeStep                    // |z| >= 2.0
)

// Z-score thresholds for outcome bucketing. Match spec §5.
const (
	holdThreshold    = 0.5
	oneStepThreshold = 1.0
	twoStepThreshold = 2.0
	subWindowAlpha   = 1.5
)

// OutcomeTierFromAbsZ buckets a z-magnitude into an OutcomeTier per
// the spec §5 table. Caller is responsible for sign-dispatching to
// the advance vs degrade/reverse/escape branch.
func OutcomeTierFromAbsZ(absZ float64) OutcomeTier {
	switch {
	case absZ < holdThreshold:
		return TierHold
	case absZ < oneStepThreshold:
		return TierOneStep
	case absZ < twoStepThreshold:
		return TierTwoStep
	default:
		return TierThreeStep
	}
}

// SubWindowOpens returns true if |z| meets the independent sub-gate
// threshold (chunk 4d composition). Spec §5: |z| >= 1.5 on controller
// side opens a sub window from the post-advance position.
func SubWindowOpens(absZ float64) bool {
	return absZ >= subWindowAlpha
}

// AdvancementTarget returns the position state to transition to when
// the controller wins drift at the given tier, plus a `hold` flag
// indicating "no position change" (status quo). Implements spec §6.1
// table. For Clinch, defenderPosture determines the 1-step target.
// defenderPosture is ignored for non-Clinch sources.
//
// For terminal-apex positions (Crucifix, BackGround) and the striking
// apex (Mount at 1/2 step), returns (source, true). Sub-gate is the
// caller's responsibility.
func AdvancementTarget(source State, tier OutcomeTier, defenderPosture State) (target State, hold bool) {
	if tier == TierHold {
		return source, true
	}

	switch source {
	case Clinch:
		return clinchAdvancementTarget(tier, defenderPosture), false

	case BackStanding:
		return BackGround, false

	case Mount:
		if tier == TierThreeStep {
			return BackGround, false
		}
		// 1-step + 2-step: striking apex Hold
		return Mount, true

	case SideControl, KneeOnBelly:
		if tier == TierThreeStep {
			return BackGround, false
		}
		return Mount, false

	case NorthSouth:
		switch tier {
		case TierOneStep:
			return SideControl, false
		case TierTwoStep:
			return Mount, false
		default: // TierThreeStep
			return BackGround, false
		}

	case Crucifix, BackGround:
		// Terminal apex — sub-only, no position advance.
		return source, true

	case HalfGuard, Guard:
		switch tier {
		case TierOneStep:
			return SideControl, false
		case TierTwoStep:
			return Mount, false
		default: // TierThreeStep
			return BackGround, false
		}

	case Turtle:
		return BackGround, false
	}

	// Unexpected source state — treat as Hold to be safe.
	return source, true
}

func clinchAdvancementTarget(tier OutcomeTier, defenderPosture State) State {
	switch tier {
	case TierOneStep:
		switch defenderPosture {
		case Prone:
			return SideControl
		case Supine:
			return Mount
		case Turtle:
			return BackGround
		default:
			// BackStanding is also valid but reached only when the
			// defender turned away mid-clinch; default fallthrough is
			// Mount per spec §6.1.
			return Mount
		}
	case TierTwoStep:
		return Mount
	default: // TierThreeStep
		return BackGround
	}
}

// DegradeTarget returns the position to step down to when the
// defender wins drift moderately (|z| in [0.5, 1.0)). Implements
// spec §6.2. For symmetric or terminal positions (Clinch, Guard,
// Turtle), returns (source, true) — defender can't degrade further
// from these and must escape or reverse instead.
func DegradeTarget(source State) (target State, hold bool) {
	switch source {
	case Clinch:
		// Symmetric — no degrade target. Stamina-drain round.
		return Clinch, true
	case BackStanding:
		return Clinch, false
	case Mount:
		return SideControl, false
	case SideControl:
		return HalfGuard, false
	case KneeOnBelly:
		return SideControl, false
	case NorthSouth:
		return SideControl, false
	case Crucifix:
		return BackGround, false
	case BackGround:
		return Mount, false
	case HalfGuard:
		return Guard, false
	case Guard, Turtle:
		// Terminal — defender can only escape or reverse from here.
		return source, true
	}
	return source, true
}

// ReversalTarget returns the position to transition to when the
// defender wins drift big (|z| in [1.0, 2.0)) — a reversal. Roles
// always swap (returned as `swap=true`). Two realism exceptions per
// spec §6.3 land in a different position:
//   - Mount        → Guard  (defender bridges up into former
//     controller's guard; former controller
//     is now the Guard-controlled top)
//   - BackGround   → Mount  (defender turns into former controller;
//     former controller is now Mount-controlled
//     bottom)
//
// All other sources return the same state with roles swapped.
//
// Caller is responsible for performing the role swap when applying
// the transition.
func ReversalTarget(source State) (target State, swap bool) {
	switch source {
	case Mount:
		return Guard, true
	case BackGround:
		return Mount, true
	default:
		return source, true
	}
}

// OutcomeKind enumerates the five round-outcome categories for a
// grapple per spec §4. Combined with Outcome.Target and
// Outcome.SubWindow, fully describes what happens this round.
type OutcomeKind int

const (
	OutcomeHold     OutcomeKind = iota // No position change
	OutcomeAdvance                     // Controller wins; advance to Target
	OutcomeDegrade                     // Defender wins moderate; position regresses to Target
	OutcomeReversal                    // Defender wins big; roles swap, position is Target
	OutcomeEscape                      // Defender wins decisive; both → Standing
)

// Outcome is the resolved per-round result of a drift roll, ready
// for Position_GrappleTick to apply via TransitionPair and pass to
// the messaging layer.
type Outcome struct {
	Kind      OutcomeKind
	Source    State // pre-transition position (for messaging context)
	Target    State // post-transition position (= Source for Hold)
	SubWindow bool  // true if |z| >= 1.5 (chunk 4d composition)
}

// ResolveOutcome dispatches the drift roll's z-score to one of the
// five outcome categories per spec §4 + §5.
//
//   - z is signed: positive = controller won, negative = defender won.
//   - source is the controller's current grapple position.
//   - defenderPosture matters only when source is Clinch (Clinch-1-step
//     posture-based dispatch); ignored for non-Clinch sources.
func ResolveOutcome(source State, z float64, defenderPosture State) Outcome {
	absZ := z
	if absZ < 0 {
		absZ = -absZ
	}
	tier := OutcomeTierFromAbsZ(absZ)
	subOpens := SubWindowOpens(absZ)

	// Hold tier (|z| < 0.5): no position change regardless of sign.
	if tier == TierHold {
		return Outcome{
			Kind:      OutcomeHold,
			Source:    source,
			Target:    source,
			SubWindow: subOpens, // Always false at this tier, but consistent
		}
	}

	// Controller-favored (positive z).
	if z > 0 {
		target, hold := AdvancementTarget(source, tier, defenderPosture)
		if hold {
			return Outcome{
				Kind:      OutcomeHold,
				Source:    source,
				Target:    source,
				SubWindow: subOpens,
			}
		}
		return Outcome{
			Kind:      OutcomeAdvance,
			Source:    source,
			Target:    target,
			SubWindow: subOpens,
		}
	}

	// Defender-favored (negative z). Branch on tier.
	switch tier {
	case TierOneStep:
		target, hold := DegradeTarget(source)
		if hold {
			return Outcome{
				Kind:      OutcomeHold,
				Source:    source,
				Target:    source,
				SubWindow: subOpens, // Never true at TierOneStep
			}
		}
		return Outcome{
			Kind:      OutcomeDegrade,
			Source:    source,
			Target:    target,
			SubWindow: subOpens,
		}
	case TierTwoStep:
		target, _ := ReversalTarget(source)
		return Outcome{
			Kind:      OutcomeReversal,
			Source:    source,
			Target:    target,
			SubWindow: subOpens,
		}
	default: // TierThreeStep
		return Outcome{
			Kind:      OutcomeEscape,
			Source:    source,
			Target:    Standing,
			SubWindow: subOpens,
		}
	}
}
