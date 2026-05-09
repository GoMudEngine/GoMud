package bounties

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Test seams.
var (
	roundForTest          func() uint64
	goldMultiplierForTest func() float64
	goldFloorForTest      func() int
)

func currentRound() uint64 {
	if roundForTest != nil {
		return roundForTest()
	}
	return util.GetRoundCount()
}

func goldMultiplier() float64 {
	if goldMultiplierForTest != nil {
		return goldMultiplierForTest()
	}
	return float64(configs.GetBalanceConfig().BountyGoldDefaultMultiplier)
}

func goldFloor() int {
	if goldFloorForTest != nil {
		return goldFloorForTest()
	}
	return int(configs.GetBalanceConfig().BountyGoldFloor)
}

// computeDefaultGold returns floor(statpool * multiplier), with a
// floor of `BountyGoldFloor` so trivial mobs still pay a meaningful
// amount.
func computeDefaultGold(statpool int) int {
	g := int(float64(statpool) * goldMultiplier())
	if floor := goldFloor(); g < floor {
		return floor
	}
	return g
}

// computeDefaultRep returns max(1, floor(statpool / 100)).
func computeDefaultRep(statpool int) int {
	r := statpool / 100
	if r < 1 {
		return 1
	}
	return r
}
