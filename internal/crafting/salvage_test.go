package crafting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcSalvageChance(t *testing.T) {
	tests := []struct {
		name   string
		skill  int
		minPct float64
		maxPct float64
	}{
		{"skill 1 — near minimum", 1, 0.14, 0.25},
		{"skill 10 — low-mid", 10, 0.40, 0.52},
		{"skill 25 — mid", 25, 0.55, 0.70},
		{"skill 50 — at soft cap", 50, 0.84, 0.86},
		{"skill 100 — above cap still capped", 100, 0.84, 0.86},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chance := CalcSalvageChance(tt.skill, 0.15, 0.85, 50)
			assert.GreaterOrEqual(t, chance, tt.minPct)
			assert.LessOrEqual(t, chance, tt.maxPct)
		})
	}
}

func TestCalcSalvageRounds(t *testing.T) {
	assert.Equal(t, 1, CalcSalvageRounds(5, 10, 5))   // 5g = 1 round (minimum)
	assert.Equal(t, 2, CalcSalvageRounds(20, 10, 5))  // 20g = 2 rounds
	assert.Equal(t, 5, CalcSalvageRounds(200, 10, 5)) // 200g = capped at 5
	assert.Equal(t, 1, CalcSalvageRounds(0, 10, 5))   // 0g = 1 round (minimum)
}
