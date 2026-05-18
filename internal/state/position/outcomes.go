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
