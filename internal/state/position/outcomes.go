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
	holdThreshold     = 0.5
	oneStepThreshold  = 1.0
	twoStepThreshold  = 2.0
	subWindowAlpha    = 1.5
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
