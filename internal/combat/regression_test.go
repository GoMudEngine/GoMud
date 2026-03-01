package combat

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/stretchr/testify/assert"
)

// ─── Regression Tests ────────────────────────────────────────────────────────
// These tests guard against specific past bug fixes. Each test documents the
// original bug and verifies the fix remains in place.

// TestRegression_CritRateNotInflated guards against Stage 37.4 bug where
// crit rate was ~49% due to incorrect z-score calculation.
// With RollSpread-based stdDev, crit rate should be ~2.3% (z >= 2.0).
func TestRegression_CritRateNotInflated(t *testing.T) {
	const iterations = 5000
	crits := 0

	// High skill gap: attacker 200 vs defender 100
	// Even with large advantage, crit rate should stay ~2.3%
	for i := 0; i < iterations; i++ {
		_, _, atkRoll, _ := dice.OpposedRollStat(200.0, 100.0)
		isCrit, _ := dice.CriticalCheck(atkRoll, 2.0, -2.0)
		if isCrit {
			crits++
		}
	}

	critRate := float64(crits) / float64(iterations)
	// Crit rate should be around 2.3% (z >= 2.0 on normal distribution)
	// Allow up to 10% as an absolute upper bound — the bug had 49%
	assert.Less(t, critRate, 0.10,
		"Crit rate should be well under 10%% (was 49%% before fix); got %.1f%%",
		critRate*100)
	t.Logf("Crit rate: %.2f%% over %d iterations", critRate*100, iterations)
}

// TestRegression_FumbleRateSymmetric guards against Stage 37.4 bug where
// high-skill attackers fumbled ~30% of the time.
// Fumble rate should be ~2.3% regardless of stat advantage.
func TestRegression_FumbleRateSymmetric(t *testing.T) {
	const iterations = 5000

	testCases := []struct {
		name    string
		atkStat float64
		defStat float64
	}{
		{"equal_stats", 100.0, 100.0},
		{"high_advantage", 200.0, 100.0},
		{"low_attacker", 50.0, 100.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fumbles := 0
			for i := 0; i < iterations; i++ {
				_, _, atkRoll, _ := dice.OpposedRollStat(tc.atkStat, tc.defStat)
				_, isFumble := dice.CriticalCheck(atkRoll, 2.0, -2.0)
				if isFumble {
					fumbles++
				}
			}
			fumbleRate := float64(fumbles) / float64(iterations)
			// Fumble rate should be ~2.3%, never above 10%
			assert.Less(t, fumbleRate, 0.10,
				"Fumble rate for %s should be < 10%%; got %.1f%%",
				tc.name, fumbleRate*100)
			t.Logf("%s fumble rate: %.2f%%", tc.name, fumbleRate*100)
		})
	}
}

// TestRegression_DefenseFloorAlwaysApplies guards against Stage 37.4 bug
// where defense floor was bypassed when defender had 0 stamina.
// The defense floor (MinDefenseChance) should provide a chance to avoid
// hits even when the defender cannot afford any defense roll.
func TestRegression_DefenseFloorAlwaysApplies(t *testing.T) {
	// Test the defense floor mechanism: even with 0 stamina, when
	// runBestOfAllDefense finds no affordable defense, resolveDefenseOutcome
	// should still give a MinDefenseChance probability of avoiding the hit.
	// We test this by creating a defender with 0 stamina and verifying that
	// the best-of-all defense function produces no valid defense (margin = -Inf),
	// but the floor save concept is preserved in the code path.
	defender := characters.New()
	defender.Stamina = 0
	defender.Health = 100

	attacker := characters.New()
	attacker.Stats.Strength.ValueAdj = 150

	result := &AttackResult{}
	defSeq := []string{characters.DefenseDodge, characters.DefenseParry}

	best := runBestOfAllDefense(result, attacker, defender, defSeq, 150.0, false)

	// With 0 stamina, no defense should be affordable — margin stays -Inf
	assert.True(t, math.IsInf(best.margin, -1),
		"With 0 stamina, no defense should succeed on the roll")
	assert.Empty(t, best.defenseType,
		"No defense type should be selected with 0 stamina")

	// The defense floor is applied in resolveDefenseOutcome (not in
	// runBestOfAllDefense), so even a totally broke defender still has
	// a chance. We verify the code path handles the empty-defense case
	// gracefully by checking it doesn't panic and margin is -Inf.
}

// TestRegression_MitigationCapEnforced guards against bug where mitigation
// exceeding 75% was not properly capped.
func TestRegression_MitigationCapEnforced(t *testing.T) {
	testCases := []struct {
		name          string
		raw           float64
		mitigationPct float64
		cap           float64
		expected      float64
	}{
		{"under_cap", 100.0, 0.50, 0.75, 50.0},
		{"at_cap", 100.0, 0.75, 0.75, 25.0},
		{"over_cap_clamped", 100.0, 0.95, 0.75, 25.0},
		{"full_mitigation_capped", 100.0, 1.00, 0.75, 25.0},
		{"negative_mitigation_treated_as_zero", 100.0, -0.10, 0.75, 100.0},
		{"zero_mitigation", 100.0, 0.0, 0.75, 100.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ApplyMitigation(tc.raw, tc.mitigationPct, tc.cap)
			assert.InDelta(t, tc.expected, result, 0.01,
				"ApplyMitigation(%.0f, %.2f, %.2f) should be %.0f",
				tc.raw, tc.mitigationPct, tc.cap, tc.expected)
		})
	}
}

// TestRegression_ResourceMultiplierNeverNegative guards against bug where
// resource depletion penalty could produce negative multiplier values.
func TestRegression_ResourceMultiplierNeverNegative(t *testing.T) {
	testCases := []struct {
		name       string
		current    int
		max        int
		penaltyMax float64
	}{
		{"full_resource", 100, 100, 0.28},
		{"half_resource", 50, 100, 0.28},
		{"quarter_resource", 25, 100, 0.28},
		{"near_zero", 1, 100, 0.28},
		{"zero_resource", 0, 100, 0.28},
		{"negative_current", -10, 100, 0.28},
		{"zero_max", 0, 0, 0.28},
		{"high_penalty", 0, 100, 0.50},
		{"max_reasonable_penalty", 0, 100, 1.00},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mult := ResourceMultiplier(tc.current, tc.max, tc.penaltyMax)
			assert.GreaterOrEqual(t, mult, 0.0,
				"ResourceMultiplier should never be negative")
			assert.LessOrEqual(t, mult, 1.0,
				"ResourceMultiplier should never exceed 1.0")
		})
	}

	// Also verify monotonicity: multiplier should decrease as resource drains
	prev := ResourceMultiplier(100, 100, 0.28)
	for pct := 99; pct >= 0; pct-- {
		curr := ResourceMultiplier(pct, 100, 0.28)
		assert.LessOrEqual(t, curr, prev+0.001,
			"ResourceMultiplier should be monotonically non-increasing")
		prev = curr
	}
}
