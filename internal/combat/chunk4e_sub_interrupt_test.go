package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
)

func setupSubInterruptTest(t *testing.T) {
	t.Helper()
	// Force the threshold knob to a known value for tests.
	configs.AddOverlayOverrides(map[string]any{
		"Balance.SubInterruptDamageThresholdPct": 0.10,
	})
}

// buildGrappledTarget returns a target 'a' grappling with partner 'b', and a
// third-party attacker 'c'. Uses buildControllerAndPartner from
// chunk4e_outside_hit_test.go (same package).
func buildGrappledTarget(t *testing.T) (a, b, c *characters.Character) {
	t.Helper()
	return buildControllerAndPartner(t)
}

func TestChunk4e_SmallDamageDoesNotAccumulate(t *testing.T) {
	setupSubInterruptTest(t)

	a, _, c := buildGrappledTarget(t)
	a.HealthMax.Value = 100

	// Small non-crit hit from third party C on target A.
	// 5 damage = 5% of 100 max, below the 10% threshold.
	chunk4eAccumulateSubInterruptDamage(c, a, 5, false)

	if a.SubInterruptDamageThisRound != 0 {
		t.Errorf("expected no accumulation for sub-threshold damage; got %v", a.SubInterruptDamageThisRound)
	}
}

func TestChunk4e_ThresholdDamageAccumulates(t *testing.T) {
	setupSubInterruptTest(t)

	a, _, c := buildGrappledTarget(t)
	a.HealthMax.Value = 100

	// At-threshold (10% of 100 = 10) non-crit hit.
	chunk4eAccumulateSubInterruptDamage(c, a, 10, false)

	if a.SubInterruptDamageThisRound != 10 {
		t.Errorf("expected accumulation = 10; got %v", a.SubInterruptDamageThisRound)
	}
}

func TestChunk4e_CritAlwaysAccumulates(t *testing.T) {
	setupSubInterruptTest(t)

	a, _, c := buildGrappledTarget(t)
	a.HealthMax.Value = 100

	// Crit hit at tiny damage — should still accumulate.
	chunk4eAccumulateSubInterruptDamage(c, a, 3, true)

	if a.SubInterruptDamageThisRound != 3 {
		t.Errorf("expected accumulation = 3 (crit); got %v", a.SubInterruptDamageThisRound)
	}
}

func TestChunk4e_NilSafety(t *testing.T) {
	chunk4eAccumulateSubInterruptDamage(nil, nil, 100, true)
	// If we got here without panic, test passes.
}

func TestChunk4e_ThresholdDisabled(t *testing.T) {
	// With threshold = 0, only crit damage should accumulate.
	configs.AddOverlayOverrides(map[string]any{
		"Balance.SubInterruptDamageThresholdPct": 0.0,
	})

	a, _, c := buildGrappledTarget(t)
	a.HealthMax.Value = 100

	// Half-max damage non-crit — should NOT accumulate with threshold=0.
	chunk4eAccumulateSubInterruptDamage(c, a, 50, false)
	if a.SubInterruptDamageThisRound != 0 {
		t.Errorf("threshold=0 should disable threshold-path; got %v", a.SubInterruptDamageThisRound)
	}

	// Crit at any damage — should still accumulate.
	chunk4eAccumulateSubInterruptDamage(c, a, 5, true)
	if a.SubInterruptDamageThisRound != 5 {
		t.Errorf("crit should always accumulate; got %v", a.SubInterruptDamageThisRound)
	}
}
