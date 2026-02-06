package characters

import (
	"math"
	"testing"
)

func TestCalculateProgressionChance_RankZero(t *testing.T) {
	chance := CalculateProgressionChance(0, SkillSoftCap)
	if chance != 0.50 {
		t.Errorf("Expected 50%% at rank 0, got %.4f", chance)
	}
}

func TestCalculateProgressionChance_NegativeRank(t *testing.T) {
	chance := CalculateProgressionChance(-5, SkillSoftCap)
	if chance != 0.50 {
		t.Errorf("Expected 50%% at negative rank, got %.4f", chance)
	}
}

func TestCalculateProgressionChance_Decreasing(t *testing.T) {
	prev := CalculateProgressionChance(0, SkillSoftCap)
	for rank := 1; rank <= SkillSoftCap+20; rank++ {
		curr := CalculateProgressionChance(rank, SkillSoftCap)
		if curr >= prev {
			t.Errorf("Chance should decrease: rank %d (%.4f) >= rank %d (%.4f)", rank, curr, rank-1, prev)
		}
		prev = curr
	}
}

func TestCalculateProgressionChance_AtSoftCap(t *testing.T) {
	chance := CalculateProgressionChance(SkillSoftCap, SkillSoftCap)
	// Should be around 2.5% (0.50 * exp(-3))
	expected := 0.50 * math.Exp(-3.0)
	if math.Abs(chance-expected) > 0.001 {
		t.Errorf("Expected ~%.4f at soft cap, got %.4f", expected, chance)
	}
}

func TestCalculateProgressionChance_AboveSoftCap(t *testing.T) {
	atCap := CalculateProgressionChance(SkillSoftCap, SkillSoftCap)
	aboveCap := CalculateProgressionChance(SkillSoftCap+10, SkillSoftCap)

	if aboveCap >= atCap {
		t.Errorf("Above soft cap should be harder: %f >= %f", aboveCap, atCap)
	}

	// Should be very small above cap
	if aboveCap > 0.05 {
		t.Errorf("Above soft cap should be < 5%%, got %.4f", aboveCap)
	}
}

func TestCalculateProgressionChance_VeryHighRank(t *testing.T) {
	chance := CalculateProgressionChance(SkillSoftCap*3, SkillSoftCap)
	if chance > 0.001 {
		t.Errorf("Very high rank should be < 0.1%%, got %.6f", chance)
	}
	if chance <= 0 {
		t.Errorf("Chance should always be positive, got %.6f", chance)
	}
}

func TestCalculateProgressionChance_ZeroSoftCap(t *testing.T) {
	// Should not panic with zero soft cap
	chance := CalculateProgressionChance(5, 0)
	if chance < 0 || chance > 1 {
		t.Errorf("Chance should be between 0 and 1, got %.4f", chance)
	}
}

func TestCalculateProgressionChance_StatSoftCap(t *testing.T) {
	// Verify the curve works with the stat soft cap too
	rankZero := CalculateProgressionChance(0, StatSoftCap)
	midRange := CalculateProgressionChance(StatSoftCap/2, StatSoftCap)
	atCap := CalculateProgressionChance(StatSoftCap, StatSoftCap)

	if rankZero != 0.50 {
		t.Errorf("Expected 50%% at rank 0 for stats, got %.4f", rankZero)
	}
	if midRange >= rankZero || midRange <= atCap {
		t.Errorf("Mid-range should be between rank 0 and cap: %.4f not between %.4f and %.4f", midRange, rankZero, atCap)
	}
	if atCap > 0.05 {
		t.Errorf("At stat soft cap should be < 5%%, got %.4f", atCap)
	}
}

func TestCalculateProgressionChance_SampleValues(t *testing.T) {
	// Verify the documented sample values are approximately correct
	tests := []struct {
		rank    int
		softCap int
		minPct  float64
		maxPct  float64
	}{
		{0, 50, 49.0, 51.0},   // ~50%
		{10, 50, 20.0, 35.0},  // ~27%
		{25, 50, 5.0, 15.0},   // ~11%
		{40, 50, 2.0, 7.0},    // ~4.5%
		{50, 50, 1.0, 4.0},    // ~2.5%
	}

	for _, tt := range tests {
		chance := CalculateProgressionChance(tt.rank, tt.softCap) * 100
		if chance < tt.minPct || chance > tt.maxPct {
			t.Errorf("Rank %d (cap %d): expected %.1f-%.1f%%, got %.2f%%",
				tt.rank, tt.softCap, tt.minPct, tt.maxPct, chance)
		}
	}
}
