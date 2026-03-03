package gametime

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ─── Integration: Moon Stat Modifiers ────────────────────────────────────────
// Exercises: MoonStatDelta(), moonContribution(), moonPhasePercent()

func TestIntegration_MoonStatModifiers(t *testing.T) {
	baseStat := 100
	maxMod := 0.05 // ±5%

	// phase=0.0 (new moon) → negative delta
	deltaN := MoonStatDelta(0.0, maxMod, baseStat)
	assert.Less(t, deltaN, 0, "New moon should give negative stat delta")
	assert.Equal(t, -5, deltaN, "New moon delta: 100 * (0.0-0.5)*2*0.05 = -5")

	// phase=0.5 (quarter) → zero delta
	deltaQ := MoonStatDelta(0.5, maxMod, baseStat)
	assert.Equal(t, 0, deltaQ, "Quarter moon should give zero delta")

	// phase=1.0 (full moon) → positive delta
	deltaF := MoonStatDelta(1.0, maxMod, baseStat)
	assert.Greater(t, deltaF, 0, "Full moon should give positive stat delta")
	assert.Equal(t, 5, deltaF, "Full moon delta: 100 * (1.0-0.5)*2*0.05 = 5")

	// Symmetry: |delta at new| == |delta at full|
	assert.Equal(t, -deltaN, deltaF, "New and full moon deltas should be symmetric")

	// phase=0.25 → halfway between new and quarter
	delta25 := MoonStatDelta(0.25, maxMod, baseStat)
	assert.Less(t, delta25, 0, "Phase 0.25 should be negative")
	assert.Greater(t, delta25, deltaN, "Phase 0.25 should be less negative than new moon")

	// phase=0.75 → halfway between quarter and full
	delta75 := MoonStatDelta(0.75, maxMod, baseStat)
	assert.Greater(t, delta75, 0, "Phase 0.75 should be positive")
	assert.Less(t, delta75, deltaF, "Phase 0.75 should be less positive than full moon")
}

func TestIntegration_MoonStatScaling(t *testing.T) {
	maxMod := 0.05

	// Delta should scale with base stat
	delta100 := MoonStatDelta(1.0, maxMod, 100)
	delta200 := MoonStatDelta(1.0, maxMod, 200)

	assert.Equal(t, 5, delta100, "Base 100 at full moon → +5")
	assert.Equal(t, 10, delta200, "Base 200 at full moon → +10")

	// Zero base → zero delta
	deltaZero := MoonStatDelta(1.0, maxMod, 0)
	assert.Equal(t, 0, deltaZero, "Zero base should give zero delta")

	// Higher maxMod → larger deltas
	delta10pct := MoonStatDelta(1.0, 0.10, 100)
	assert.Equal(t, 10, delta10pct, "10%% maxMod at full moon → +10")
}

func TestIntegration_MoonContribution(t *testing.T) {
	// moonContribution maps phase percent [0,1) to smooth [0,1]
	// 0.0 → new moon → contribution 0.0
	// 0.5 → full moon → contribution 1.0
	// 0.25 → waxing quarter → contribution 0.5

	c0 := moonContribution(0.0)
	assert.InDelta(t, 0.0, c0, 0.001, "New moon contribution should be 0")

	c50 := moonContribution(0.5)
	assert.InDelta(t, 1.0, c50, 0.001, "Full moon contribution should be 1.0")

	c25 := moonContribution(0.25)
	assert.InDelta(t, 0.5, c25, 0.001, "Waxing quarter contribution should be 0.5")

	c75 := moonContribution(0.75)
	assert.InDelta(t, 0.5, c75, 0.001, "Waning quarter contribution should be 0.5")

	// Smooth and always in [0,1]
	for i := 0; i <= 100; i++ {
		phase := float64(i) / 100.0
		c := moonContribution(phase)
		assert.GreaterOrEqual(t, c, 0.0,
			"Contribution should be >= 0 at phase %.2f", phase)
		assert.LessOrEqual(t, c, 1.0,
			"Contribution should be <= 1 at phase %.2f", phase)
	}
}

func TestIntegration_MoonPhasePercent(t *testing.T) {
	rpd := uint64(100) // 100 rounds per day for simplicity

	// At round 0, all phases should be 0.0
	p := moonPhasePercent(swiftmoonPeriod, 0, rpd)
	assert.InDelta(t, 0.0, p, 0.001, "Phase at round 0 should be 0")

	// Phase should be in [0, 1) range
	for round := uint64(0); round < 5000; round += 50 {
		p := moonPhasePercent(swiftmoonPeriod, round, rpd)
		assert.GreaterOrEqual(t, p, 0.0, "Phase should be >= 0")
		assert.Less(t, p, 1.0, "Phase should be < 1.0")
	}

	// Zero RPD should return 0
	p = moonPhasePercent(swiftmoonPeriod, 100, 0)
	assert.Equal(t, 0.0, p, "Zero RPD should return 0")

	// Full cycle: at round = period * rpd, should wrap back to ~0
	cycleLen := swiftmoonPeriod * float64(rpd)
	pEnd := moonPhasePercent(swiftmoonPeriod, uint64(math.Round(cycleLen)), rpd)
	assert.InDelta(t, 0.0, pEnd, 0.01,
		"Phase should wrap back to ~0 at end of cycle")
}
