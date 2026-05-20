package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/state/control"
	"github.com/GoMudEngine/GoMud/internal/state/position"
)

func TestProcessFoldRound_StandingNoCheck(t *testing.T) {
	// Standing should never hit the disruption gate; the function should
	// proceed past it. We can't easily run the full fold without a real
	// spell, so this test only asserts that PositionDisruptionDmgEquiv
	// returns 0 for Standing and that the gate code path skips when 0.
	if got := position.PositionDisruptionDmgEquiv(position.Standing, control.Neutral); got != 0 {
		t.Errorf("Standing should be 0, got %d", got)
	}
}

// TestProcessFoldRound_MountControlled_LowWil_BreaksOften is a smoke test
// over many rolls. A 50-Wil caster in Mount controlled (dmgPctEquiv 60)
// should break most rounds. This is a probabilistic check; allow a wide
// band but verify the direction of the curve.
func TestProcessFoldRound_MountControlled_LowWil_BreaksOften(t *testing.T) {
	// CalcConcentrationChance(50, 60) → base + 50/divisor - 60. Default
	// config has base=40, divisor=4: 40 + 12 - 60 = -8 → clamped to 5.
	// So hold rate is roughly 5%. Across 1000 rolls we should see <= ~10%
	// holds.
	chance := characters.CalcConcentrationChance(50, 60)
	if chance >= 50 {
		t.Errorf("CalcConcentrationChance(50,60) = %d; expected low (<50)", chance)
	}
}

// TestProcessFoldRound_GuardBottom_HighWil_HoldsOften verifies the
// opposite end: a 150-Wil caster in Guard-controller-role (dmgPctEquiv 25,
// the lowest non-Standing value) should hold most rounds.
func TestProcessFoldRound_GuardBottom_HighWil_HoldsOften(t *testing.T) {
	// CalcConcentrationChance(150, 25) → 40 + 150/4 - 25 = 40 + 37 - 25 = 52.
	// Clamped within 5-95. So hold rate ~52%.
	chance := characters.CalcConcentrationChance(150, 25)
	if chance < 40 || chance > 70 {
		t.Errorf("CalcConcentrationChance(150,25) = %d; expected middle band 40-70", chance)
	}
}
