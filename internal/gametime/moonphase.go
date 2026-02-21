package gametime

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Moon period multipliers — full cycle length expressed as a multiple of RoundsPerDay.
// These match the lore in world.md: the three Witnesses (colony ships in slow orbit).
const (
	swiftmoonPeriod float64 = 4.7
	wandererPeriod  float64 = 10.6
	eyePeriod       float64 = 21.1
)

// moonPhasePercent returns the current phase position of a moon as a value in [0.0, 1.0).
// 0.0 = new moon, 0.5 = full moon, approaching 1.0 = waning back toward new.
func moonPhasePercent(periodMult float64, roundNum uint64, rpd uint64) float64 {
	cycleLen := periodMult * float64(rpd)
	if cycleLen < 1 {
		return 0
	}
	return math.Mod(float64(roundNum), cycleLen) / cycleLen
}

// moonContribution converts a phase percentage to a smooth cosine contribution in [0.0, 1.0].
// Returns 0.0 at new moon (phase 0.0), 1.0 at full moon (phase 0.5), and 0.0 again at the
// next new moon — producing a smooth sinusoidal curve.
func moonContribution(phasePercent float64) float64 {
	return (1 - math.Cos(2*math.Pi*phasePercent)) / 2
}

// GetFoldPressure returns the current Fold pressure as a value in [0.0, 1.0].
// Pressure is the weighted cosine sum of all three Witnesses:
//
//	Swiftmoon: 20%  (fast, minor influence)
//	Wanderer:  30%  (medium, moderate influence)
//	The Eye:   50%  (slowest and largest — dominant)
//
// 0.0 = all moons at new moon (minimum pressure)
// 1.0 = all moons at full moon simultaneously (maximum pressure)
func GetFoldPressure() float64 {
	roundNum := util.GetRoundCount()
	rpd := uint64(configs.GetTimingConfig().RoundsPerDay)
	if rpd == 0 {
		return 0
	}

	swift := moonContribution(moonPhasePercent(swiftmoonPeriod, roundNum, rpd))
	wander := moonContribution(moonPhasePercent(wandererPeriod, roundNum, rpd))
	eye := moonContribution(moonPhasePercent(eyePeriod, roundNum, rpd))

	return 0.20*swift + 0.30*wander + 0.50*eye
}

// GetMoonContributions returns each moon's individual cosine contribution [0.0, 1.0]
// before weighting. Intended for admin diagnostics (devtool pressure).
func GetMoonContributions() (swiftmoon, wanderer, eye float64) {
	roundNum := util.GetRoundCount()
	rpd := uint64(configs.GetTimingConfig().RoundsPerDay)
	if rpd == 0 {
		return
	}
	swiftmoon = moonContribution(moonPhasePercent(swiftmoonPeriod, roundNum, rpd))
	wanderer = moonContribution(moonPhasePercent(wandererPeriod, roundNum, rpd))
	eye = moonContribution(moonPhasePercent(eyePeriod, roundNum, rpd))
	return
}
