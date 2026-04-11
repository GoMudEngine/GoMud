package buffs

import (
	"math"
	"math/rand"
)

// ComputeTickAmount calculates the per-tick heal/damage amount.
// maxPool: target's max HP/SP/CP for the relevant pool.
// percent: base percentage (positive=heal, negative=damage).
// variance: random variance added to percent (0=none).
// minAmount: minimum absolute value (1 if not specified).
// scalingMult: spell skill multiplier (1.0 for potions/non-spell).
// Returns the signed tick amount (positive=heal, negative=damage).
func ComputeTickAmount(maxPool int, percent float64, variance float64, minAmount int, scalingMult float64) int {
	if percent == 0 {
		return 0
	}
	if minAmount < 1 {
		minAmount = 1
	}

	effectivePercent := percent
	if variance > 0 {
		effectivePercent += rand.Float64() * variance
	}

	base := float64(maxPool) * effectivePercent
	scaled := base * scalingMult
	amount := int(math.Round(math.Abs(scaled)))

	if amount < minAmount {
		amount = minAmount
	}

	if percent < 0 {
		return -amount
	}
	return amount
}
