package gametime

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Moon period multipliers — full cycle length as a multiple of RoundsPerDay.
// Matches world.md lore for the three Witnesses.
const (
	swiftmoonPeriod float64 = 4.7
	wandererPeriod  float64 = 10.6
	eyePeriod       float64 = 21.1
)

// moonPhasePercent returns the current phase position in [0.0, 1.0).
// 0.0 = new moon, 0.5 = full moon.
func moonPhasePercent(periodMult float64, roundNum uint64, rpd uint64) float64 {
	cycleLen := periodMult * float64(rpd)
	if cycleLen < 1 {
		return 0
	}
	return math.Mod(float64(roundNum), cycleLen) / cycleLen
}

// moonContribution converts a phase percent to a smooth [0.0, 1.0] value.
// 0.0 at new moon, 1.0 at full moon, 0.5 at the quarters.
func moonContribution(phasePercent float64) float64 {
	return (1 - math.Cos(2*math.Pi*phasePercent)) / 2
}

func currentPhases() (swift, wander, eye float64) {
	roundNum := util.GetRoundCount()
	rpd := uint64(configs.GetTimingConfig().RoundsPerDay)
	if rpd == 0 {
		return
	}
	swift = moonContribution(moonPhasePercent(swiftmoonPeriod, roundNum, rpd))
	wander = moonContribution(moonPhasePercent(wandererPeriod, roundNum, rpd))
	eye = moonContribution(moonPhasePercent(eyePeriod, roundNum, rpd))
	return
}

// GetSwiftmoonPhase returns Swiftmoon's current phase: 0.0 (new) → 1.0 (full).
// Swiftmoon influences Dexterity and Strength.
func GetSwiftmoonPhase() float64 {
	swift, _, _ := currentPhases()
	return swift
}

// GetWandererPhase returns The Wanderer's current phase: 0.0 (new) → 1.0 (full).
// The Wanderer influences Vitality and Willpower.
func GetWandererPhase() float64 {
	_, wander, _ := currentPhases()
	return wander
}

// GetEyePhase returns The Eye's current phase: 0.0 (new) → 1.0 (full).
// The Eye influences Perception and Charisma, and modulates mutation rate.
func GetEyePhase() float64 {
	_, _, eye := currentPhases()
	return eye
}

// GetAllPhases returns all three moon phases in one call — use this when you
// need more than one to avoid redundant computation.
func GetAllPhases() (swiftmoon, wanderer, eye float64) {
	return currentPhases()
}

// MoonStatDelta computes the integer stat delta a moon phase applies to a stat.
//
// phase:   0.0 = new moon (negative), 0.5 = quarter (zero), 1.0 = full (positive)
// maxMod:  maximum fractional modifier, e.g. 0.05 for ±5%
// base:    the stat's current ValueAdj — determines magnitude of the delta
//
// Returns a signed int that should be added to ValueAdj and subtracted after use.
func MoonStatDelta(phase, maxMod float64, base int) int {
	// Map [0,1] → [-maxMod, +maxMod], centred at 0 when phase == 0.5
	frac := (phase - 0.5) * 2 * maxMod
	return int(math.Round(float64(base) * frac))
}
