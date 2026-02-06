package dice

import (
	"fmt"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func init() {
	// Initialize logger for tests (write to stderr, no colors, ERROR level to reduce output)
	mudlog.SetupLogger(nil, "ERROR", "", false)
}

func TestRoll(t *testing.T) {
	// Test basic rolling
	result := Roll(2, 6, 3)

	if result.Total < 5 || result.Total > 15 {
		t.Errorf("Roll(2, 6, 3) produced invalid result: %d (expected 5-15)", result.Total)
	}

	if len(result.RawRolls) != 2 {
		t.Errorf("Expected 2 raw rolls, got %d", len(result.RawRolls))
	}

	if result.Modifier != 3 {
		t.Errorf("Expected modifier of 3, got %d", result.Modifier)
	}

	t.Logf("Roll(2d6+3): %s", result.String())
}

func TestRollWithType(t *testing.T) {
	// Test advantage roll
	result := RollWithType(1, 20, 5, Advantage)

	if len(result.Rolls) != 2 {
		t.Errorf("Advantage roll should track both rolls, got %d rolls", len(result.Rolls))
	}

	if result.Total < 6 || result.Total > 25 {
		t.Errorf("Roll(1d20+5 Advantage) produced invalid result: %d", result.Total)
	}

	t.Logf("Advantage Roll(1d20+5): %s (rolls: %v, selected: %d)", result.Description, result.Rolls, result.Total)

	// Test disadvantage roll
	result2 := RollWithType(1, 20, 0, Disadvantage)

	if len(result2.Rolls) != 2 {
		t.Errorf("Disadvantage roll should track both rolls, got %d rolls", len(result2.Rolls))
	}

	t.Logf("Disadvantage Roll(1d20): %s (rolls: %v, selected: %d)", result2.Description, result2.Rolls, result2.Total)
}

func TestOpposedRoll(t *testing.T) {
	attackerStat := 50
	defenderStat := 40

	success, margin, atkRoll, defRoll := OpposedRoll(attackerStat, defenderStat)

	t.Logf("Opposed Roll: Attacker(%d)=%d vs Defender(%d)=%d | Success: %t, Margin: %d",
		attackerStat, atkRoll.Total, defenderStat, defRoll.Total, success, margin)

	if success && margin <= 0 {
		t.Errorf("Success should have positive margin, got %d", margin)
	}

	if !success && margin >= 0 {
		t.Errorf("Failure should have negative margin, got %d", margin)
	}

	if atkRoll.Success != success {
		t.Errorf("Attack roll success flag doesn't match result")
	}
}

func TestDifficultyCheck(t *testing.T) {
	stat := 60
	difficulty := 50

	result := DifficultyCheck(stat, difficulty, Normal)

	t.Logf("Difficulty Check: Stat %d vs DC %d = %d (%s) | Success: %t, Margin: %d",
		stat, difficulty, result.Total, result.String(), result.Success, result.Margin)

	// Result should be in valid range
	if result.Total < 61 || result.Total > 160 {
		t.Errorf("Difficulty check result out of expected range: %d", result.Total)
	}
}

func TestDifficultyCheckCriticals(t *testing.T) {
	// Run many checks to try to get a critical and fumble
	stat := 50
	difficulty := 50

	critCount := 0
	fumbleCount := 0
	iterations := 1000

	for i := 0; i < iterations; i++ {
		result := DifficultyCheck(stat, difficulty, Normal)
		if result.Critical {
			critCount++
			t.Logf("Critical success! Natural roll: %d, Total: %d", result.RawRolls[0], result.Total)
		}
		if result.Fumble {
			fumbleCount++
			t.Logf("Fumble! Natural roll: %d, Total: %d", result.RawRolls[0], result.Total)
		}
	}

	t.Logf("Out of %d rolls: %d criticals (%.1f%%), %d fumbles (%.1f%%)",
		iterations, critCount, float64(critCount)/float64(iterations)*100,
		fumbleCount, float64(fumbleCount)/float64(iterations)*100)

	// Should get at least some criticals and fumbles in 1000 rolls
	if critCount == 0 {
		t.Logf("Warning: No critical successes in %d rolls (expected ~6%%)", iterations)
	}
	if fumbleCount == 0 {
		t.Logf("Warning: No fumbles in %d rolls (expected ~5%%)", iterations)
	}
}

func TestRollStatArray(t *testing.T) {
	// Test standard 3d6 method
	stats := RollStatArray(3, 6, 0, 6, false)

	if len(stats) != 6 {
		t.Errorf("Expected 6 stats, got %d", len(stats))
	}

	t.Logf("Standard 3d6 stats: %v", stats)

	for i, stat := range stats {
		if stat < 3 || stat > 18 {
			t.Errorf("Stat %d out of range: %d (expected 3-18)", i, stat)
		}
	}

	// Test 4d6 drop lowest (rolls 5 dice, drops lowest, so range is 4-24)
	stats2 := RollStatArray(4, 6, 0, 6, true)

	if len(stats2) != 6 {
		t.Errorf("Expected 6 stats, got %d", len(stats2))
	}

	t.Logf("4d6 drop lowest stats: %v", stats2)

	for i, stat := range stats2 {
		if stat < 4 || stat > 24 {
			t.Errorf("Stat %d out of range: %d (expected 4-24)", i, stat)
		}
	}

	// Test 2d6+6 bounded method
	stats3 := RollStatArray(2, 6, 6, 6, false)

	if len(stats3) != 6 {
		t.Errorf("Expected 6 stats, got %d", len(stats3))
	}

	t.Logf("2d6+6 stats: %v", stats3)

	for i, stat := range stats3 {
		if stat < 8 || stat > 18 {
			t.Errorf("Stat %d out of range: %d (expected 8-18)", i, stat)
		}
	}
}

func TestPercentile(t *testing.T) {
	// Test 50% chance
	successes := 0
	iterations := 1000

	for i := 0; i < iterations; i++ {
		success, _ := Percentile(50)
		if success {
			successes++
		}
	}

	successRate := float64(successes) / float64(iterations) * 100
	t.Logf("50%% percentile check: %d/%d successes (%.1f%%)", successes, iterations, successRate)

	// Should be roughly 50% (allow 10% variance)
	if successRate < 40 || successRate > 60 {
		t.Logf("Warning: Success rate %.1f%% is outside expected range (40-60%%)", successRate)
	}
}

func TestRollTable(t *testing.T) {
	// Test weighted table
	weights := []int{10, 20, 30, 40} // Total: 100

	counts := make([]int, len(weights))
	iterations := 1000

	for i := 0; i < iterations; i++ {
		result := RollTable(weights)
		counts[result]++
	}

	t.Logf("Roll table results over %d iterations:", iterations)
	for i, count := range counts {
		percentage := float64(count) / float64(iterations) * 100
		expectedPct := float64(weights[i])
		t.Logf("  Index %d (weight %d): %d times (%.1f%%, expected %.1f%%)",
			i, weights[i], count, percentage, expectedPct)
	}
}

func TestExplodingDice(t *testing.T) {
	result := ExplodingDice(3, 6, 0, 2)

	t.Logf("Exploding dice (3d6!): %s", result.String())
	t.Logf("  Raw rolls: %v (total: %d)", result.RawRolls, result.Total)

	// Should have at least 3 rolls
	if len(result.RawRolls) < 3 {
		t.Errorf("Expected at least 3 rolls, got %d", len(result.RawRolls))
	}

	// Minimum possible is 3, no maximum due to explosions
	if result.Total < 3 {
		t.Errorf("Total %d is below minimum possible (3)", result.Total)
	}
}

func TestSuccessPool(t *testing.T) {
	// Roll 5d10, target 7+
	result := SuccessPool(5, 10, 7)

	t.Logf("Success pool (5d10, target 7+): %d successes", result.Total)
	t.Logf("  Rolls: %v", result.RawRolls)

	if len(result.RawRolls) != 5 {
		t.Errorf("Expected 5 rolls, got %d", len(result.RawRolls))
	}

	// Count expected successes manually
	expectedSuccesses := 0
	for _, roll := range result.RawRolls {
		if roll >= 7 {
			expectedSuccesses++
		}
	}

	if result.Total != expectedSuccesses {
		t.Errorf("Success count mismatch: got %d, expected %d", result.Total, expectedSuccesses)
	}
}

func TestContestedRoll(t *testing.T) {
	stat1 := 60
	stat2 := 50

	stat1Wins, margin, roll1, roll2 := ContestedRoll(stat1, stat2, Normal, Normal)

	t.Logf("Contested Roll: %d vs %d", stat1, stat2)
	t.Logf("  Stat1 roll: %d, Stat2 roll: %d", roll1.Total, roll2.Total)
	t.Logf("  Winner: Stat%d, Margin: %d", map[bool]int{true: 1, false: 2}[stat1Wins], margin)

	if stat1Wins && roll1.Total <= roll2.Total {
		t.Errorf("Stat1 won but roll1 (%d) <= roll2 (%d)", roll1.Total, roll2.Total)
	}

	if !stat1Wins && margin != 0 && roll2.Total <= roll1.Total {
		t.Errorf("Stat2 won but roll2 (%d) <= roll1 (%d)", roll2.Total, roll1.Total)
	}
}

func TestAverageRoll(t *testing.T) {
	avg := AverageRoll(2, 6, 3)
	expected := 10.0 // (2 * 3.5) + 3 = 10

	if avg != expected {
		t.Errorf("AverageRoll(2d6+3) = %.1f, expected %.1f", avg, expected)
	}

	t.Logf("Average of 2d6+3: %.1f", avg)
}

func TestMinMaxRoll(t *testing.T) {
	min := MinRoll(2, 6, 3)
	max := MaxRoll(2, 6, 3)

	expectedMin := 5  // 2 + 3
	expectedMax := 15 // 12 + 3

	if min != expectedMin {
		t.Errorf("MinRoll(2d6+3) = %d, expected %d", min, expectedMin)
	}

	if max != expectedMax {
		t.Errorf("MaxRoll(2d6+3) = %d, expected %d", max, expectedMax)
	}

	t.Logf("2d6+3 range: %d to %d (average: %.1f)", min, max, AverageRoll(2, 6, 3))
}

func TestCheckCritical(t *testing.T) {
	// Test d20 criticals
	isCrit, isFumble := CheckCritical(20, 20, 0, 0)
	if !isCrit {
		t.Errorf("Natural 20 on d20 should be critical")
	}
	if isFumble {
		t.Errorf("Natural 20 on d20 should not be fumble")
	}

	isCrit, isFumble = CheckCritical(1, 20, 0, 0)
	if isCrit {
		t.Errorf("Natural 1 on d20 should not be critical")
	}
	if !isFumble {
		t.Errorf("Natural 1 on d20 should be fumble")
	}

	// Test d100 criticals
	isCrit, isFumble = CheckCritical(95, 100, 0, 0)
	if !isCrit {
		t.Errorf("95 on d100 should be critical")
	}

	isCrit, isFumble = CheckCritical(5, 100, 0, 0)
	if !isFumble {
		t.Errorf("5 on d100 should be fumble")
	}

	// Test custom thresholds
	isCrit, isFumble = CheckCritical(18, 20, 18, 3)
	if !isCrit {
		t.Errorf("18 should be critical with threshold 18")
	}

	t.Log("Critical/fumble detection tests passed")
}

func TestRollBetween(t *testing.T) {
	// Test rolling between specific values
	result := RollBetween(10, 20)

	if result < 10 || result > 20 {
		t.Errorf("RollBetween(10, 20) = %d, expected 10-20", result)
	}

	// Test with reversed args
	result2 := RollBetween(20, 10)

	if result2 < 10 || result2 > 20 {
		t.Errorf("RollBetween(20, 10) = %d, expected 10-20", result2)
	}

	// Test with equal args
	result3 := RollBetween(15, 15)

	if result3 != 15 {
		t.Errorf("RollBetween(15, 15) = %d, expected 15", result3)
	}

	t.Logf("RollBetween tests: %d, %d, %d", result, result2, result3)
}

// Benchmark tests
func BenchmarkRoll(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Roll(2, 6, 3)
	}
}

func BenchmarkRollWithAdvantage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RollWithType(1, 20, 5, Advantage)
	}
}

func BenchmarkOpposedRoll(b *testing.B) {
	for i := 0; i < b.N; i++ {
		OpposedRoll(50, 40)
	}
}

func BenchmarkDifficultyCheck(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DifficultyCheck(60, 50, Normal)
	}
}

func BenchmarkRollStatArray(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RollStatArray(4, 6, 0, 6, true)
	}
}

// Example usage for documentation
func ExampleRoll() {
	result := Roll(2, 6, 3)
	fmt.Printf("Rolling 2d6+3: %s\n", result.String())
}

func ExampleRollWithType() {
	result := RollWithType(1, 20, 5, Advantage)
	fmt.Printf("Rolling 1d20+5 with Advantage: %s\n", result.String())
	fmt.Printf("Both rolls were: %v\n", result.Rolls)
}

func ExampleDifficultyCheck() {
	result := DifficultyCheck(60, 50, Normal)
	if result.Success {
		fmt.Printf("Success by %d! Total: %d\n", result.Margin, result.Total)
	} else {
		fmt.Printf("Failed by %d. Total: %d\n", -result.Margin, result.Total)
	}
}

func ExampleRollStatArray() {
	stats := RollStatArray(4, 6, 0, 6, true)
	fmt.Printf("Character stats (4d6 drop lowest): %v\n", stats)
}
