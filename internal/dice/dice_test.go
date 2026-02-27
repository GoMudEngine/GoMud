package dice

import (
	"fmt"
	"math"
	"testing"
)

func TestRoll(t *testing.T) {
	mean := 50.0
	stdDev := 10.0

	// Test basic rolling
	result := Roll(mean, stdDev)

	if result.Mean != mean {
		t.Errorf("Expected mean %f, got %f", mean, result.Mean)
	}

	if result.StdDev != stdDev {
		t.Errorf("Expected stdDev %f, got %f", stdDev, result.StdDev)
	}

	t.Logf("Roll(50, 10): %s", result.String())
}

func TestRollDistribution(t *testing.T) {
	mean := 50.0
	stdDev := 10.0
	iterations := 10000

	sum := 0.0
	sumSquares := 0.0

	for i := 0; i < iterations; i++ {
		result := Roll(mean, stdDev)
		sum += result.Value
		sumSquares += result.Value * result.Value
	}

	actualMean := sum / float64(iterations)
	variance := (sumSquares / float64(iterations)) - (actualMean * actualMean)
	actualStdDev := math.Sqrt(variance)

	t.Logf("Over %d rolls:", iterations)
	t.Logf("  Expected mean: %.2f, Actual mean: %.2f", mean, actualMean)
	t.Logf("  Expected stdDev: %.2f, Actual stdDev: %.2f", stdDev, actualStdDev)

	// Allow 5% variance due to randomness
	if math.Abs(actualMean-mean) > mean*0.05 {
		t.Errorf("Mean %.2f is too far from expected %.2f", actualMean, mean)
	}

	if math.Abs(actualStdDev-stdDev) > stdDev*0.15 {
		t.Errorf("StdDev %.2f is too far from expected %.2f", actualStdDev, stdDev)
	}
}

func TestOpposedRoll(t *testing.T) {
	attackerStat := 60.0
	defenderStat := 50.0
	stdDev := 10.0

	success, margin, atkRoll, defRoll := OpposedRoll(attackerStat, defenderStat, stdDev)

	t.Logf("Opposed Roll: Attacker(%.0f)=%.2f vs Defender(%.0f)=%.2f | Success: %t, Margin: %.2f",
		attackerStat, atkRoll.Value, defenderStat, defRoll.Value, success, margin)

	if atkRoll.Success != success {
		t.Errorf("Attack roll success flag doesn't match result")
	}

	if math.Abs(margin-(atkRoll.Value-defRoll.Value)) > 0.001 {
		t.Errorf("Margin calculation incorrect")
	}
}

func TestOpposedRollStatistics(t *testing.T) {
	attackerStat := 60.0
	defenderStat := 50.0
	stdDev := 10.0
	iterations := 10000

	wins := 0
	for i := 0; i < iterations; i++ {
		success, _, _, _ := OpposedRoll(attackerStat, defenderStat, stdDev)
		if success {
			wins++
		}
	}

	winRate := float64(wins) / float64(iterations)
	expectedWinRate := OpposedSuccessChance(attackerStat, defenderStat, stdDev)

	t.Logf("Opposed roll statistics over %d iterations:", iterations)
	t.Logf("  Attacker: %.0f, Defender: %.0f, StdDev: %.0f", attackerStat, defenderStat, stdDev)
	t.Logf("  Win rate: %.1f%% (expected: %.1f%%)", winRate*100, expectedWinRate*100)

	if math.Abs(winRate-expectedWinRate) > 0.03 {
		t.Errorf("Win rate %.2f is too far from expected %.2f", winRate, expectedWinRate)
	}
}

func TestDifficultyCheck(t *testing.T) {
	stat := 60.0
	difficulty := 50.0
	stdDev := 10.0

	result := DifficultyCheck(stat, difficulty, stdDev)

	t.Logf("Difficulty Check: Stat %.0f vs DC %.0f = %.2f | Success: %t, Margin: %.2f",
		stat, difficulty, result.Value, result.Success, result.Margin)

	if result.Success && result.Margin < 0 {
		t.Errorf("Success should have non-negative margin")
	}

	if !result.Success && result.Margin >= 0 {
		t.Errorf("Failure should have negative margin")
	}
}

func TestSuccessChance(t *testing.T) {
	stat := 60.0
	difficulty := 50.0
	stdDev := 10.0

	chance := SuccessChance(stat, difficulty, stdDev)

	t.Logf("Success chance for stat %.0f vs difficulty %.0f (stdDev %.0f): %.1f%%",
		stat, difficulty, stdDev, chance*100)

	// With stat 10 higher than difficulty and stdDev of 10,
	// we expect about 84% success rate (z-score = 1.0)
	if chance < 0.80 || chance > 0.88 {
		t.Errorf("Success chance %.2f outside expected range [0.80, 0.88]", chance)
	}

	// Verify empirically
	iterations := 10000
	successes := 0
	for i := 0; i < iterations; i++ {
		result := DifficultyCheck(stat, difficulty, stdDev)
		if result.Success {
			successes++
		}
	}

	empiricalChance := float64(successes) / float64(iterations)
	t.Logf("  Empirical success rate: %.1f%%", empiricalChance*100)

	if math.Abs(empiricalChance-chance) > 0.03 {
		t.Errorf("Empirical chance %.2f too far from calculated %.2f", empiricalChance, chance)
	}
}

func TestRollStatArray(t *testing.T) {
	mean := 12.0
	stdDev := 2.0
	min := 3.0
	max := 18.0

	stats := RollStatArray(6, mean, stdDev, min, max)

	if len(stats) != 6 {
		t.Errorf("Expected 6 stats, got %d", len(stats))
	}

	t.Logf("Stat array (mean: %.0f, stdDev: %.0f): %v", mean, stdDev, stats)

	for i, stat := range stats {
		if stat < int(min) || stat > int(max) {
			t.Errorf("Stat %d out of range: %d (expected %.0f-%.0f)", i, stat, min, max)
		}
	}

	// Check average
	sum := 0
	for _, stat := range stats {
		sum += stat
	}
	avg := float64(sum) / 6.0
	t.Logf("  Average: %.1f", avg)
}

func TestRollDamage(t *testing.T) {
	baseDamage := 50.0
	variance := 10.0
	minDamage := 10.0

	damage := RollDamage(baseDamage, variance, minDamage)

	if damage < minDamage {
		t.Errorf("Damage %.2f below minimum %.2f", damage, minDamage)
	}

	t.Logf("Damage roll (base: %.0f, variance: %.0f, min: %.0f): %.2f",
		baseDamage, variance, minDamage, damage)

	// Test integer version
	damageInt := RollDamageInt(baseDamage, variance, minDamage)
	if damageInt < int(minDamage) {
		t.Errorf("Damage int %d below minimum %.0f", damageInt, minDamage)
	}
	t.Logf("  Integer damage: %d", damageInt)
}

func TestCriticalCheck(t *testing.T) {
	iterations := 100000
	critThreshold := 2.0  // ~2.5% chance
	fumbleThreshold := -2.0

	critCount := 0
	fumbleCount := 0

	for i := 0; i < iterations; i++ {
		result := Roll(50.0, 10.0)
		isCrit, isFumble := CriticalCheck(result, critThreshold, fumbleThreshold)
		if isCrit {
			critCount++
		}
		if isFumble {
			fumbleCount++
		}
	}

	critRate := float64(critCount) / float64(iterations) * 100
	fumbleRate := float64(fumbleCount) / float64(iterations) * 100

	t.Logf("Critical check over %d rolls (threshold: ±%.1f):", iterations, critThreshold)
	t.Logf("  Criticals: %d (%.2f%%, expected ~2.28%%)", critCount, critRate)
	t.Logf("  Fumbles: %d (%.2f%%, expected ~2.28%%)", fumbleCount, fumbleRate)

	// Expected rate for z=2.0 is about 2.28%
	expectedRate := 2.28
	tolerance := 0.5 // Allow ±0.5% due to randomness with 100k samples
	if math.Abs(critRate-expectedRate) > tolerance {
		t.Errorf("Crit rate %.2f%% outside tolerance of expected %.2f%% (±%.1f%%)", critRate, expectedRate, tolerance)
	}
	if math.Abs(fumbleRate-expectedRate) > tolerance {
		t.Errorf("Fumble rate %.2f%% outside tolerance of expected %.2f%% (±%.1f%%)", fumbleRate, expectedRate, tolerance)
	}
}

func TestCriticalCheckDynamicThreshold(t *testing.T) {
	iterations := 100000

	// Test with a skill differential that shifts the threshold
	// Skilled attacker: threshold lowered to 1.5 (more crits, fewer fumbles)
	critThreshold := 1.5
	fumbleThreshold := -critThreshold

	critCount := 0
	fumbleCount := 0

	for i := 0; i < iterations; i++ {
		result := Roll(50.0, 10.0)
		isCrit, isFumble := CriticalCheck(result, critThreshold, fumbleThreshold)
		if isCrit {
			critCount++
		}
		if isFumble {
			fumbleCount++
		}
	}

	critRate := float64(critCount) / float64(iterations) * 100
	fumbleRate := float64(fumbleCount) / float64(iterations) * 100

	t.Logf("Dynamic threshold check (crit: %.1f, fumble: %.1f):", critThreshold, fumbleThreshold)
	t.Logf("  Criticals: %d (%.2f%%, expected ~6.68%%)", critCount, critRate)
	t.Logf("  Fumbles: %d (%.2f%%, expected ~6.68%%)", fumbleCount, fumbleRate)

	// z=1.5 gives ~6.68% rate
	expectedRate := 6.68
	tolerance := 1.0
	if math.Abs(critRate-expectedRate) > tolerance {
		t.Errorf("Crit rate %.2f%% outside tolerance of expected %.2f%% (±%.1f%%)", critRate, expectedRate, tolerance)
	}
	if math.Abs(fumbleRate-expectedRate) > tolerance {
		t.Errorf("Fumble rate %.2f%% outside tolerance of expected %.2f%% (±%.1f%%)", fumbleRate, expectedRate, tolerance)
	}
}

func TestPercentile(t *testing.T) {
	chance := 75.0
	iterations := 10000

	successes := 0
	for i := 0; i < iterations; i++ {
		success, _ := Percentile(chance)
		if success {
			successes++
		}
	}

	successRate := float64(successes) / float64(iterations) * 100

	t.Logf("Percentile check (%.0f%% chance) over %d iterations:", chance, iterations)
	t.Logf("  Success rate: %.1f%%", successRate)

	if math.Abs(successRate-chance) > 2.0 {
		t.Errorf("Success rate %.1f%% too far from expected %.1f%%", successRate, chance)
	}
}

func TestRollTable(t *testing.T) {
	weights := []int{10, 20, 30, 40}
	iterations := 10000

	counts := make([]int, len(weights))
	for i := 0; i < iterations; i++ {
		index := RollTable(weights)
		counts[index]++
	}

	t.Logf("Roll table over %d iterations:", iterations)
	for i, count := range counts {
		percentage := float64(count) / float64(iterations) * 100
		expected := float64(weights[i])
		t.Logf("  Index %d (weight %d): %d times (%.1f%%, expected %.1f%%)",
			i, weights[i], count, percentage, expected)
	}
}

func TestGetPercentile(t *testing.T) {
	mean := 50.0
	stdDev := 10.0

	// Test common percentiles
	percentiles := []float64{50, 75, 90, 95, 99}

	t.Logf("Percentile values for Normal(%.0f, %.0f):", mean, stdDev)
	for _, p := range percentiles {
		value := GetPercentile(mean, stdDev, p)
		t.Logf("  %3.0fth percentile: %.2f", p, value)
	}

	// Median should be very close to mean
	median := GetPercentile(mean, stdDev, 50)
	if math.Abs(median-mean) > 0.01 {
		t.Errorf("Median %.2f differs from mean %.2f", median, mean)
	}

	// 95th percentile should be about 1.645 standard deviations above mean
	p95 := GetPercentile(mean, stdDev, 95)
	expected95 := mean + 1.645*stdDev
	if math.Abs(p95-expected95) > 0.5 {
		t.Errorf("95th percentile %.2f differs from expected %.2f", p95, expected95)
	}
}

func TestRollClamped(t *testing.T) {
	mean := 50.0
	stdDev := 20.0
	min := 30.0
	max := 70.0
	iterations := 1000

	belowMin := 0
	aboveMax := 0

	for i := 0; i < iterations; i++ {
		result := RollClamped(mean, stdDev, min, max)

		if result.Value < min {
			t.Errorf("Clamped roll %.2f below min %.2f", result.Value, min)
		}
		if result.Value > max {
			t.Errorf("Clamped roll %.2f above max %.2f", result.Value, max)
		}

		// Count how many would have been out of range
		unclamped := Roll(mean, stdDev)
		if unclamped.Value < min {
			belowMin++
		}
		if unclamped.Value > max {
			aboveMax++
		}
	}

	t.Logf("Clamped rolls (mean: %.0f, stdDev: %.0f, range: %.0f-%.0f):", mean, stdDev, min, max)
	t.Logf("  Unclamped: %d below min, %d above max out of %d", belowMin, aboveMax, iterations)
}

func TestStandardDeviation(t *testing.T) {
	// Test the helper function for calculating stdDev
	statRange := 100.0

	tests := []struct {
		factor   float64
		expected float64
	}{
		{0.1, 10.0},
		{0.15, 15.0},
		{0.2, 20.0},
	}

	for _, tt := range tests {
		result := StandardDeviation(statRange, tt.factor)
		if result != tt.expected {
			t.Errorf("StandardDeviation(%.0f, %.2f) = %.2f, expected %.2f",
				statRange, tt.factor, result, tt.expected)
		}
	}

	t.Logf("Standard deviation helper:")
	t.Logf("  Stat range 0-100, factor 0.10 = %.1f (low randomness)", StandardDeviation(100, 0.10))
	t.Logf("  Stat range 0-100, factor 0.15 = %.1f (moderate randomness)", StandardDeviation(100, 0.15))
	t.Logf("  Stat range 0-100, factor 0.20 = %.1f (high randomness)", StandardDeviation(100, 0.20))
}

func TestDiceToDistribution(t *testing.T) {
	tests := []struct {
		dCount, dSides, bonus int
		expectedMean          float64
		expectedStdDev        float64
	}{
		{1, 6, 0, 3.5, 1.7078},   // 1d6: mean=3.5, stddev≈1.71
		{2, 6, 0, 7.0, 2.4152},   // 2d6: mean=7.0, stddev≈2.42
		{2, 6, 3, 10.0, 2.4152},  // 2d6+3: mean=10.0, stddev≈2.42
		{1, 4, 0, 2.5, 1.1180},   // 1d4: mean=2.5, stddev≈1.12
		{1, 20, 0, 10.5, 5.7662}, // 1d20: mean=10.5, stddev≈5.77
		{3, 8, 2, 15.5, 3.9686},  // 3d8+2: mean=15.5, stddev≈3.97
	}

	for _, tt := range tests {
		mean, stdDev := DiceToDistribution(tt.dCount, tt.dSides, tt.bonus)

		if math.Abs(mean-tt.expectedMean) > 0.01 {
			t.Errorf("DiceToDistribution(%dd%d+%d) mean = %.4f, want %.4f",
				tt.dCount, tt.dSides, tt.bonus, mean, tt.expectedMean)
		}

		if math.Abs(stdDev-tt.expectedStdDev) > 0.01 {
			t.Errorf("DiceToDistribution(%dd%d+%d) stdDev = %.4f, want %.4f",
				tt.dCount, tt.dSides, tt.bonus, stdDev, tt.expectedStdDev)
		}

		t.Logf("%dd%d+%d → mean=%.2f, stdDev=%.4f", tt.dCount, tt.dSides, tt.bonus, mean, stdDev)
	}
}

func TestDiceToDistributionDamageOutput(t *testing.T) {
	// Verify that rolling damage via DiceToDistribution produces
	// results with the expected mean and spread
	dCount, dSides, bonus := 2, 6, 3
	mean, stdDev := DiceToDistribution(dCount, dSides, bonus)
	iterations := 10000

	sum := 0.0
	for i := 0; i < iterations; i++ {
		result := Roll(mean, stdDev)
		val := math.Max(0, result.Value)
		sum += val
	}

	actualMean := sum / float64(iterations)
	t.Logf("2d6+3 distribution damage over %d rolls: mean=%.2f (expected %.2f)",
		iterations, actualMean, mean)

	if math.Abs(actualMean-mean) > 0.5 {
		t.Errorf("Actual mean %.2f too far from expected %.2f", actualMean, mean)
	}
}

// Benchmark tests
func BenchmarkRoll(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Roll(50.0, 10.0)
	}
}

func BenchmarkOpposedRoll(b *testing.B) {
	for i := 0; i < b.N; i++ {
		OpposedRoll(60.0, 50.0, 10.0)
	}
}

func BenchmarkDifficultyCheck(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DifficultyCheck(60.0, 50.0, 10.0)
	}
}

func BenchmarkSuccessChance(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SuccessChance(60.0, 50.0, 10.0)
	}
}

func BenchmarkRollStatArray(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RollStatArray(6, 12.0, 2.0, 3.0, 18.0)
	}
}

// Example usage
func ExampleRoll() {
	result := Roll(50.0, 10.0)
	fmt.Printf("Result: %.2f (z-score: %.2f)\n", result.Value, result.ZScore)
}

func ExampleOpposedRoll() {
	success, margin, _, _ := OpposedRoll(60.0, 50.0, 10.0)
	if success {
		fmt.Printf("Attacker wins by %.2f\n", margin)
	} else {
		fmt.Printf("Defender wins by %.2f\n", -margin)
	}
}

func ExampleDifficultyCheck() {
	result := DifficultyCheck(60.0, 50.0, 10.0)
	if result.Success {
		fmt.Printf("Success! Margin: %.2f\n", result.Margin)
	} else {
		fmt.Printf("Failure. Margin: %.2f\n", result.Margin)
	}
}

func ExampleRollStatArray() {
	stats := RollStatArray(6, 12.0, 2.0, 3.0, 18.0)
	fmt.Printf("Character stats: %v\n", stats)
}
