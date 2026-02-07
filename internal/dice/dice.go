package dice

import (
	"fmt"
	"math"
	"math/rand"
)

// RollResult contains detailed information about a roll
type RollResult struct {
	Value       float64 // The rolled value
	Mean        float64 // The mean of the distribution
	StdDev      float64 // The standard deviation used
	Success     bool    // Whether the roll succeeded (for checks)
	Margin      float64 // Margin of success/failure
	ZScore      float64 // How many standard deviations from mean
	Percentile  float64 // Approximate percentile (0-100)
	Description string  // Human-readable description
}

// String returns a formatted string representation of the roll result
func (r RollResult) String() string {
	return fmt.Sprintf("%.2f (mean: %.2f, σ: %.2f, z: %.2f)", r.Value, r.Mean, r.StdDev, r.ZScore)
}

// Roll performs a normal distribution roll
// mean: center of the distribution (typically the character's stat value)
// stdDev: spread of the distribution (controls randomness/consistency)
// Returns a RollResult with the value and statistical information
func Roll(mean, stdDev float64) RollResult {
	value := rand.NormFloat64()*stdDev + mean
	zScore := (value - mean) / stdDev
	percentile := normalCDF(zScore) * 100

	return RollResult{
		Value:      value,
		Mean:       mean,
		StdDev:     stdDev,
		ZScore:     zScore,
		Percentile: percentile,
		Description: fmt.Sprintf("Normal(%.2f, %.2f)", mean, stdDev),
	}
}

// OpposedRoll performs a contested check between two stats
// Each side rolls using their stat as the mean with the given standard deviation
// Returns true if attacker wins, the margin, and both roll results
func OpposedRoll(attackerStat, defenderStat, stdDev float64) (bool, float64, RollResult, RollResult) {
	attackRoll := Roll(attackerStat, stdDev)
	defenseRoll := Roll(defenderStat, stdDev)

	success := attackRoll.Value > defenseRoll.Value
	margin := attackRoll.Value - defenseRoll.Value

	attackRoll.Success = success
	attackRoll.Margin = margin
	defenseRoll.Success = !success
	defenseRoll.Margin = -margin

	return success, margin, attackRoll, defenseRoll
}

// DifficultyCheck performs a check against a target value
// stat: the character's stat (used as mean)
// difficulty: the target value to beat
// stdDev: standard deviation (higher = more random)
// Returns a RollResult with success/failure information
func DifficultyCheck(stat, difficulty, stdDev float64) RollResult {
	result := Roll(stat, stdDev)
	result.Success = result.Value >= difficulty
	result.Margin = result.Value - difficulty

	return result
}

// RollInt performs a normal distribution roll and returns an integer
// Useful when you need whole number results
func RollInt(mean, stdDev float64) int {
	return int(math.Round(Roll(mean, stdDev).Value))
}

// RollClamped performs a roll and clamps the result to a range
// Useful for ensuring rolls don't exceed logical bounds
func RollClamped(mean, stdDev, min, max float64) RollResult {
	result := Roll(mean, stdDev)
	if result.Value < min {
		result.Value = min
	} else if result.Value > max {
		result.Value = max
	}
	return result
}

// RollIntClamped performs a roll, clamps it, and returns an integer
func RollIntClamped(mean, stdDev, min, max float64) int {
	return int(math.Round(RollClamped(mean, stdDev, min, max).Value))
}

// SuccessChance calculates the probability of beating a difficulty
// Returns a value from 0.0 to 1.0
func SuccessChance(stat, difficulty, stdDev float64) float64 {
	zScore := (stat - difficulty) / stdDev
	return normalCDF(zScore)
}

// ExpectedMargin calculates the expected margin of success
// Positive means expected success, negative means expected failure
func ExpectedMargin(stat, difficulty float64) float64 {
	return stat - difficulty
}

// OpposedSuccessChance calculates the probability of winning an opposed roll
// Returns a value from 0.0 to 1.0
func OpposedSuccessChance(attackerStat, defenderStat, stdDev float64) float64 {
	// The difference of two normal distributions is also normal
	// with mean = difference of means and variance = sum of variances
	meanDiff := attackerStat - defenderStat
	combinedStdDev := math.Sqrt(2) * stdDev // Both use same stdDev

	// Probability that difference > 0
	zScore := meanDiff / combinedStdDev
	return normalCDF(zScore)
}

// RollStatArray generates character stats using normal distribution
// count: number of stats to generate
// mean: target average for stats
// stdDev: variation in stats
// min, max: bounds for stat values
func RollStatArray(count int, mean, stdDev, min, max float64) []int {
	stats := make([]int, count)
	for i := 0; i < count; i++ {
		stats[i] = RollIntClamped(mean, stdDev, min, max)
	}
	return stats
}

// RollDamage performs a damage roll with potential variance
// baseDamage: the expected damage value
// variance: how much the damage can vary (standard deviation)
// minDamage: minimum damage that can be dealt
func RollDamage(baseDamage, variance, minDamage float64) float64 {
	damage := Roll(baseDamage, variance).Value
	if damage < minDamage {
		damage = minDamage
	}
	return damage
}

// RollDamageInt performs a damage roll and returns an integer
func RollDamageInt(baseDamage, variance, minDamage float64) int {
	return int(math.Round(RollDamage(baseDamage, variance, minDamage)))
}

// CriticalCheck determines if a roll is a critical based on z-score
// criticalThreshold: how many standard deviations above mean (typically 2.0 for ~2.5% chance)
// fumbleThreshold: how many standard deviations below mean (typically -2.0 for ~2.5% chance)
// Returns (isCritical, isFumble)
func CriticalCheck(result RollResult, criticalThreshold, fumbleThreshold float64) (bool, bool) {
	isCritical := result.ZScore >= criticalThreshold
	isFumble := result.ZScore <= fumbleThreshold
	return isCritical, isFumble
}

// RollWithCriticals performs a roll and checks for criticals
func RollWithCriticals(mean, stdDev, critThreshold, fumbleThreshold float64) (RollResult, bool, bool) {
	result := Roll(mean, stdDev)
	isCrit, isFumble := CriticalCheck(result, critThreshold, fumbleThreshold)
	return result, isCrit, isFumble
}

// Percentile performs a simple percentile check (0-100)
// chance: the percentage chance of success (0-100)
// Returns (success, actualRoll)
func Percentile(chance float64) (bool, float64) {
	roll := rand.Float64() * 100
	return roll <= chance, roll
}

// RollBetween generates a random float between min and max
func RollBetween(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

// RollBetweenInt generates a random integer between min and max (inclusive)
func RollBetweenInt(min, max int) int {
	if min > max {
		min, max = max, min
	}
	if min == max {
		return min
	}
	return min + rand.Intn(max-min+1)
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

	roll := rand.Intn(total)
	cumulative := 0

	for i, weight := range weights {
		cumulative += weight
		if roll < cumulative {
			return i
		}
	}

	return len(weights) - 1
}

// normalCDF approximates the cumulative distribution function of the standard normal distribution
// This is the probability that a standard normal random variable is less than or equal to z
func normalCDF(z float64) float64 {
	// Using the approximation from Abramowitz and Stegun
	// Maximum error: 7.5e-8

	if z < 0 {
		return 1 - normalCDF(-z)
	}

	const (
		p  = 0.2316419
		b1 = 0.319381530
		b2 = -0.356563782
		b3 = 1.781477937
		b4 = -1.821255978
		b5 = 1.330274429
	)

	t := 1.0 / (1.0 + p*z)
	t2 := t * t
	t3 := t2 * t
	t4 := t3 * t
	t5 := t4 * t

	// Probability density function
	phi := math.Exp(-z*z/2) / math.Sqrt(2*math.Pi)

	// Cumulative probability
	return 1 - phi*(b1*t+b2*t2+b3*t3+b4*t4+b5*t5)
}

// GetPercentile returns the value at a given percentile for a distribution
// percentile: 0-100 (e.g., 50 = median, 95 = 95th percentile)
func GetPercentile(mean, stdDev, percentile float64) float64 {
	// Convert percentile to z-score using inverse normal CDF approximation
	z := inverseNormalCDF(percentile / 100.0)
	return mean + z*stdDev
}

// inverseNormalCDF approximates the inverse of the normal CDF
// Returns the z-score for a given probability
func inverseNormalCDF(p float64) float64 {
	// Beasley-Springer-Moro algorithm
	// Accurate to about 1e-9

	if p <= 0 {
		return math.Inf(-1)
	}
	if p >= 1 {
		return math.Inf(1)
	}

	const (
		a0 = 2.50662823884
		a1 = -18.61500062529
		a2 = 41.39119773534
		a3 = -25.44106049637
		b0 = -8.47351093090
		b1 = 23.08336743743
		b2 = -21.06224101826
		b3 = 3.13082909833
		c0 = 0.3374754822726147
		c1 = 0.9761690190917186
		c2 = 0.1607979714918209
		c3 = 0.0276438810333863
		c4 = 0.0038405729373609
		c5 = 0.0003951896511919
		c6 = 0.0000321767881768
		c7 = 0.0000002888167364
		c8 = 0.0000003960315187
	)

	y := p - 0.5
	var r, x float64

	if math.Abs(y) < 0.42 {
		// Central region
		r = y * y
		x = y * (((a3*r+a2)*r+a1)*r + a0) / ((((b3*r+b2)*r+b1)*r+b0)*r + 1)
	} else {
		// Tail region
		if y > 0 {
			r = 1 - p
		} else {
			r = p
		}
		r = math.Log(-math.Log(r))
		x = c0 + r*(c1+r*(c2+r*(c3+r*(c4+r*(c5+r*(c6+r*(c7+r*c8)))))))
		if y < 0 {
			x = -x
		}
	}

	return x
}

// CompareRolls compares two roll results
// Returns 1 if roll1 is higher, -1 if roll2 is higher, 0 if equal
func CompareRolls(roll1, roll2 RollResult) int {
	if roll1.Value > roll2.Value {
		return 1
	} else if roll1.Value < roll2.Value {
		return -1
	}
	return 0
}

// AverageResult returns the expected value for a given distribution
func AverageResult(mean, stdDev float64) float64 {
	return mean // For normal distribution, mean is the expected value
}

// DiceToDistribution converts NdM+bonus dice notation to normal distribution parameters.
// dCount: number of dice, dSides: sides per die, bonus: flat damage bonus
// Returns (mean, stdDev) suitable for use with Roll() and RollDamage().
func DiceToDistribution(dCount, dSides, bonus int) (mean float64, stdDev float64) {
	mean = float64(dCount)*(float64(dSides)+1)/2 + float64(bonus)
	variance := float64(dCount) * (float64(dSides)*float64(dSides) - 1) / 12
	stdDev = math.Sqrt(variance)
	return mean, stdDev
}

// StandardDeviation calculates a reasonable standard deviation based on stat range
// This is a helper for determining how much randomness to apply
// statRange: the typical range of the stat (e.g., 100 if stats range 0-100)
// randomnessFactor: 0.0 = no randomness, 0.5 = very random (typically 0.1-0.2)
func StandardDeviation(statRange, randomnessFactor float64) float64 {
	return statRange * randomnessFactor
}
