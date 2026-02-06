package dice

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/util"
)

// RollType represents the type of roll being made
type RollType int

const (
	Normal       RollType = iota // Standard roll
	Advantage                    // Roll twice, take higher
	Disadvantage                 // Roll twice, take lower
)

// RollResult contains detailed information about a dice roll
type RollResult struct {
	Total       int      // Final result after all modifiers
	RawRolls    []int    // Individual die results before modifiers
	Modifier    int      // Total modifier applied
	Success     bool     // Whether the roll succeeded (for checks)
	Margin      int      // Margin of success/failure
	Critical    bool     // Whether this was a critical success
	Fumble      bool     // Whether this was a critical failure
	Description string   // Human-readable description of the roll
	Rolls       []int    // All rolls made (for advantage/disadvantage)
}

// String returns a formatted string representation of the roll result
func (r RollResult) String() string {
	rollStr := fmt.Sprintf("%v", r.RawRolls)
	if r.Modifier != 0 {
		return fmt.Sprintf("%s %+d = %d", rollStr, r.Modifier, r.Total)
	}
	return fmt.Sprintf("%s = %d", rollStr, r.Total)
}

// Roll performs a standard dice roll (XdY+modifier)
// Returns the total and individual die results
// Checks for critical success (max roll) and fumble (min roll) on d20 and d100 rolls
func Roll(diceCount, diceSides, modifier int) RollResult {
	result := RollResult{
		Modifier: modifier,
		RawRolls: make([]int, diceCount),
	}

	for i := 0; i < diceCount; i++ {
		result.RawRolls[i] = util.Rand(diceSides) + 1
		result.Total += result.RawRolls[i]
	}

	// Check for critical success/failure on common roll types
	if diceCount == 1 {
		naturalRoll := result.RawRolls[0]
		if diceSides == 20 {
			// d20: 20 is critical, 1 is fumble
			if naturalRoll == 20 {
				result.Critical = true
			} else if naturalRoll == 1 {
				result.Fumble = true
			}
		} else if diceSides == 100 {
			// d100: 95+ is critical, 5- is fumble
			if naturalRoll >= 95 {
				result.Critical = true
			} else if naturalRoll <= 5 {
				result.Fumble = true
			}
		}
	}

	result.Total += modifier
	result.Description = fmt.Sprintf("%dd%d%+d", diceCount, diceSides, modifier)

	return result
}

// RollWithType performs a roll with advantage or disadvantage
// For advantage: rolls twice and takes the higher result
// For disadvantage: rolls twice and takes the lower result
func RollWithType(diceCount, diceSides, modifier int, rollType RollType) RollResult {
	if rollType == Normal {
		return Roll(diceCount, diceSides, modifier)
	}

	// Roll twice for advantage/disadvantage
	roll1 := Roll(diceCount, diceSides, modifier)
	roll2 := Roll(diceCount, diceSides, modifier)

	var selected RollResult
	if rollType == Advantage {
		if roll1.Total >= roll2.Total {
			selected = roll1
		} else {
			selected = roll2
		}
		selected.Description = fmt.Sprintf("%dd%d%+d [Advantage]", diceCount, diceSides, modifier)
	} else { // Disadvantage
		if roll1.Total <= roll2.Total {
			selected = roll1
		} else {
			selected = roll2
		}
		selected.Description = fmt.Sprintf("%dd%d%+d [Disadvantage]", diceCount, diceSides, modifier)
	}

	// Track both rolls for transparency
	selected.Rolls = []int{roll1.Total, roll2.Total}

	return selected
}

// OpposedRoll performs an opposed check between two values
// Returns true if the attacker wins, along with the margin of success
func OpposedRoll(attackerStat, defenderStat int) (bool, int, RollResult, RollResult) {
	attackRoll := Roll(1, 100, attackerStat)
	defenseRoll := Roll(1, 100, defenderStat)

	success := attackRoll.Total > defenseRoll.Total
	margin := attackRoll.Total - defenseRoll.Total

	attackRoll.Success = success
	attackRoll.Margin = margin
	defenseRoll.Success = !success
	defenseRoll.Margin = -margin

	return success, margin, attackRoll, defenseRoll
}

// DifficultyCheck performs a check against a difficulty number
// difficulty represents the target number to beat
// Returns whether the check succeeded and by how much
func DifficultyCheck(stat, difficulty int, rollType RollType) RollResult {
	result := RollWithType(1, 100, stat, rollType)
	result.Success = result.Total >= difficulty
	result.Margin = result.Total - difficulty

	// Check for critical success (natural 95+) or critical failure (natural 5 or less)
	if len(result.RawRolls) > 0 {
		naturalRoll := result.RawRolls[0]
		if naturalRoll >= 95 {
			result.Critical = true
			result.Success = true // Critical success overrides normal failure
		} else if naturalRoll <= 5 {
			result.Fumble = true
			result.Success = false // Fumble overrides normal success
		}
	}

	return result
}

// RollStatArray generates an array of stats for character creation
// Each stat is rolled using the specified method
// Common methods:
//   - 3d6: standard (10-11 average)
//   - 4d6 drop lowest: more generous (12-13 average)
//   - 2d6+6: bounded higher (13 average, min 8, max 18)
func RollStatArray(diceCount, diceSides, modifier int, statCount int, dropLowest bool) []int {
	stats := make([]int, statCount)

	for i := 0; i < statCount; i++ {
		if dropLowest && diceCount > 1 {
			// Roll one extra die and drop the lowest
			rolls := make([]int, diceCount+1)
			lowest := diceSides + 1
			lowestIdx := 0
			total := 0

			for j := 0; j < diceCount+1; j++ {
				rolls[j] = util.Rand(diceSides) + 1
				total += rolls[j]
				if rolls[j] < lowest {
					lowest = rolls[j]
					lowestIdx = j
				}
			}

			// Drop the lowest
			total -= rolls[lowestIdx]
			stats[i] = total + modifier
		} else {
			result := Roll(diceCount, diceSides, modifier)
			stats[i] = result.Total
		}
	}

	return stats
}

// Percentile performs a percentile roll (1d100)
// Useful for chance checks - returns true if roll is less than chance
func Percentile(chance int) (bool, int) {
	roll := util.Rand(100) + 1
	success := roll <= chance
	return success, roll
}

// RollTable rolls on a weighted table
// weights should sum to 100 for percentile, but can be any total
// Returns the index of the selected item
func RollTable(weights []int) int {
	if len(weights) == 0 {
		return -1
	}

	total := 0
	for _, w := range weights {
		total += w
	}

	roll := util.Rand(total)
	cumulative := 0

	for i, weight := range weights {
		cumulative += weight
		if roll < cumulative {
			return i
		}
	}

	// Fallback to last item (shouldn't happen with proper weights)
	return len(weights) - 1
}

// BellCurve performs a bell curve roll (3d6 style)
// This produces a more predictable distribution centered around the middle values
// With 3d6: average is 10-11, range is 3-18
func BellCurve(modifier int) RollResult {
	return Roll(3, 6, modifier)
}

// PointBuy converts a point-buy cost to a stat value
// Common D&D style point buy where higher stats cost exponentially more
func PointBuy(points int) int {
	// Standard 3-18 range with exponential cost
	// Cost: 8=0, 9=1, 10=2, 11=3, 12=4, 13=5, 14=7, 15=9, 16=12, 17=15, 18=19
	pointCosts := map[int]int{
		0: 8, 1: 9, 2: 10, 3: 11, 4: 12, 5: 13, 7: 14, 9: 15, 12: 16, 15: 17, 19: 18,
	}

	if stat, ok := pointCosts[points]; ok {
		return stat
	}

	// If not exact match, find the highest stat we can afford
	maxStat := 8
	for cost, stat := range pointCosts {
		if cost <= points && stat > maxStat {
			maxStat = stat
		}
	}

	return maxStat
}

// AverageRoll calculates the average result of a dice roll
// Useful for balancing and expected damage calculations
func AverageRoll(diceCount, diceSides, modifier int) float64 {
	avgPerDie := float64(diceSides+1) / 2.0
	return float64(diceCount)*avgPerDie + float64(modifier)
}

// MaxRoll calculates the maximum possible result of a dice roll
func MaxRoll(diceCount, diceSides, modifier int) int {
	return (diceCount * diceSides) + modifier
}

// MinRoll calculates the minimum possible result of a dice roll
func MinRoll(diceCount, diceSides, modifier int) int {
	return diceCount + modifier
}

// CompareRolls compares two roll results and returns which is better
// Returns 1 if roll1 is better, -1 if roll2 is better, 0 if tied
func CompareRolls(roll1, roll2 RollResult) int {
	if roll1.Total > roll2.Total {
		return 1
	} else if roll1.Total < roll2.Total {
		return -1
	}
	return 0
}

// RollBetween generates a random number between min and max (inclusive)
// This is a convenience function for simple range generation
func RollBetween(min, max int) int {
	if min > max {
		min, max = max, min
	}
	if min == max {
		return min
	}
	return util.Rand(max-min+1) + min
}

// ExplodingDice rolls dice that "explode" on max value
// When you roll the maximum value, you roll again and add it
// maxExplosions limits how many times a single die can explode
func ExplodingDice(diceCount, diceSides, modifier int, maxExplosions int) RollResult {
	result := RollResult{
		Modifier: modifier,
		RawRolls: make([]int, 0, diceCount),
	}

	for i := 0; i < diceCount; i++ {
		diceTotal := 0
		explosions := 0

		for {
			roll := util.Rand(diceSides) + 1
			diceTotal += roll
			result.RawRolls = append(result.RawRolls, roll)

			// Check if we should explode
			if roll == diceSides && explosions < maxExplosions {
				explosions++
				continue
			}
			break
		}

		result.Total += diceTotal
	}

	result.Total += modifier
	result.Description = fmt.Sprintf("%dd%d!%+d", diceCount, diceSides, modifier)

	return result
}

// SuccessPool counts successes in a pool of dice
// Each die that meets or exceeds the target is a success
// Used in systems like World of Darkness or Shadowrun
func SuccessPool(diceCount, diceSides, targetNumber int) RollResult {
	result := RollResult{
		RawRolls: make([]int, diceCount),
	}

	successes := 0
	for i := 0; i < diceCount; i++ {
		result.RawRolls[i] = util.Rand(diceSides) + 1
		if result.RawRolls[i] >= targetNumber {
			successes++
		}
	}

	result.Total = successes
	result.Success = successes > 0
	result.Description = fmt.Sprintf("%dd%d (target %d)", diceCount, diceSides, targetNumber)

	return result
}

// CriticalRange checks if a roll is within the critical range
// critRange is the number from the top of the die range (e.g., 20 on d20, or 19-20 for range of 2)
func CriticalRange(roll, diceSides, critRange int) bool {
	threshold := diceSides - critRange + 1
	return roll >= threshold
}

// CheckCritical determines if a roll result is a critical success or fumble
// Can specify custom thresholds, or use defaults for d20 (20/1) and d100 (95+/5-)
// Returns (isCritical, isFumble)
func CheckCritical(naturalRoll, diceSides int, critThreshold, fumbleThreshold int) (bool, bool) {
	// If no custom thresholds provided, use defaults based on die type
	if critThreshold == 0 {
		if diceSides == 20 {
			critThreshold = 20
		} else if diceSides == 100 {
			critThreshold = 95
		} else {
			critThreshold = diceSides // Max roll is critical for other dice
		}
	}

	if fumbleThreshold == 0 {
		if diceSides == 20 {
			fumbleThreshold = 1
		} else if diceSides == 100 {
			fumbleThreshold = 5
		} else {
			fumbleThreshold = 1 // Min roll is fumble for other dice
		}
	}

	isCritical := naturalRoll >= critThreshold
	isFumble := naturalRoll <= fumbleThreshold

	return isCritical, isFumble
}

// ApplyCriticalToResult updates a RollResult with critical/fumble information
// Uses CheckCritical internally with optional custom thresholds
func ApplyCriticalToResult(result *RollResult, diceSides, critThreshold, fumbleThreshold int) {
	if len(result.RawRolls) == 1 {
		result.Critical, result.Fumble = CheckCritical(result.RawRolls[0], diceSides, critThreshold, fumbleThreshold)
	}
}

// ContestedRoll performs a contested roll between two stats
// Both sides roll, highest wins
// In case of tie, returns false with 0 margin
func ContestedRoll(stat1, stat2 int, rollType1, rollType2 RollType) (stat1Wins bool, margin int, roll1, roll2 RollResult) {
	roll1 = RollWithType(1, 100, stat1, rollType1)
	roll2 = RollWithType(1, 100, stat2, rollType2)

	if roll1.Total > roll2.Total {
		stat1Wins = true
		margin = roll1.Total - roll2.Total
		roll1.Success = true
		roll2.Success = false
	} else if roll2.Total > roll1.Total {
		stat1Wins = false
		margin = roll2.Total - roll1.Total
		roll1.Success = false
		roll2.Success = true
	} else {
		// Tie - neither wins
		stat1Wins = false
		margin = 0
		roll1.Success = false
		roll2.Success = false
	}

	roll1.Margin = margin
	roll2.Margin = -margin

	return
}

// GaussianRoll produces a Gaussian (normal) distribution approximation
// Uses central limit theorem by rolling multiple dice and averaging
// More rolls = tighter distribution around the mean
func GaussianRoll(mean, stdDev float64, rolls int) int {
	sum := 0
	for i := 0; i < rolls; i++ {
		sum += util.Rand(100) + 1
	}
	avg := float64(sum) / float64(rolls)

	// Map [1,100] average to Gaussian with mean and stdDev
	normalized := (avg - 50.5) / 28.87 // Normalize to ~N(0,1)
	result := int(math.Round(normalized*stdDev + mean))

	return result
}
