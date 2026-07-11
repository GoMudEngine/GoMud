package mutations

import "testing"

// TestDescribeEffect_Wave1Types ensures the effect types introduced/surfaced by
// the Wave 1 proof slice all produce a non-empty player-facing description, so
// no mutation renders with a blank effect list in the `mutations` command.
func TestDescribeEffect_Wave1Types(t *testing.T) {
	cases := []MutationEffect{
		{Type: "spell_power", Value: 0.3},
		{Type: "dodge_modifier", Value: 15},
		{Type: "dodge_modifier", Value: -15},
		{Type: "health_regen_multiplier", Value: 0.15},
	}
	for _, e := range cases {
		if got := DescribeEffect(e); got == "" {
			t.Errorf("DescribeEffect(%q, %v) = empty, want a phrase", e.Type, e.Value)
		}
	}
}
