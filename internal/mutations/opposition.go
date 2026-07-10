package mutations

import "github.com/GoMudEngine/GoMud/internal/configs"

// poleScale returns a multiplier in [1-maxDecay, 1.0] that shrinks as pole
// depth grows: scale = 1 - maxDecay * depth/(depth+ref). Flat near zero,
// asymptotic toward the floor at extreme depth.
func poleScale(depth, maxDecay, ref float64) float64 {
	if depth <= 0 {
		return 1.0
	}
	frac := depth / (depth + ref)
	return 1.0 - maxDecay*frac
}

// BodyConvictionScale is the multiplier applied to ConvictionMax based on how
// deep the character has committed to the Body pole (chokes spells, taunt,
// and summons together — all Conviction-fuelled).
func BodyConvictionScale(owned map[string]int) float64 {
	b := configs.GetBalanceConfig()
	return poleScale(PoleDepth(owned, "body"),
		float64(b.MutationBodyConvictionDecayMax), float64(b.MutationPoleDecayRef))
}

// BeliefGearScale is the multiplier applied to gear effectiveness based on how
// deep the character has committed to the Belief pole (weapons/armor grow
// ornamental — extends incorporeal's gear_effectiveness_loss to the whole pole).
func BeliefGearScale(owned map[string]int) float64 {
	b := configs.GetBalanceConfig()
	return poleScale(PoleDepth(owned, "belief"),
		float64(b.MutationBeliefGearDecayMax), float64(b.MutationPoleDecayRef))
}
