package combat

import (
	"math"
	"testing"
)

func TestSkillMultiplier(t *testing.T) {
	// Rank 0 should return base (1.0)
	m0 := SkillMultiplier(0)
	if m0 != 1.0 {
		t.Errorf("SkillMultiplier(0) = %f, want 1.0", m0)
	}

	// Negative rank should return base
	mNeg := SkillMultiplier(-5)
	if mNeg != 1.0 {
		t.Errorf("SkillMultiplier(-5) = %f, want 1.0", mNeg)
	}

	// Rank 50 (soft cap) should return max (3.0)
	m50 := SkillMultiplier(50)
	if math.Abs(m50-3.0) > 0.01 {
		t.Errorf("SkillMultiplier(50) = %f, want ~3.0", m50)
	}

	// Rank 25 should be ~2.41 (1 + 2*sqrt(0.5))
	m25 := SkillMultiplier(25)
	expected25 := 1.0 + 2.0*math.Sqrt(0.5)
	if math.Abs(m25-expected25) > 0.01 {
		t.Errorf("SkillMultiplier(25) = %f, want ~%f", m25, expected25)
	}

	// Monotonically increasing
	prev := SkillMultiplier(0)
	for r := 1; r <= 50; r++ {
		cur := SkillMultiplier(r)
		if cur < prev {
			t.Errorf("SkillMultiplier(%d) = %f < SkillMultiplier(%d) = %f", r, cur, r-1, prev)
		}
		prev = cur
	}

	// Above soft cap should be clamped to max
	m100 := SkillMultiplier(100)
	if math.Abs(m100-3.0) > 0.01 {
		t.Errorf("SkillMultiplier(100) = %f, want ~3.0 (capped at soft cap)", m100)
	}
}

func TestCalcRawDamage(t *testing.T) {
	// Stat 100, rank 0, multiplier 1.0 → 100 * 1.0 * 1.0 = 100
	raw := CalcRawDamage(100, 0, 1.0)
	if math.Abs(raw-100.0) > 0.01 {
		t.Errorf("CalcRawDamage(100, 0, 1.0) = %f, want 100.0", raw)
	}

	// Stat 100, rank 50, multiplier 1.0 → 100 * 3.0 * 1.0 = 300
	raw2 := CalcRawDamage(100, 50, 1.0)
	if math.Abs(raw2-300.0) > 0.01 {
		t.Errorf("CalcRawDamage(100, 50, 1.0) = %f, want 300.0", raw2)
	}

	// Zero item multiplier should fallback to 0.30
	raw3 := CalcRawDamage(100, 0, 0)
	if math.Abs(raw3-30.0) > 0.01 {
		t.Errorf("CalcRawDamage(100, 0, 0) = %f, want 30.0 (fallback)", raw3)
	}
}

func TestApplyMitigation(t *testing.T) {
	// No mitigation
	result := ApplyMitigation(100, 0, 0.75)
	if math.Abs(result-100.0) > 0.01 {
		t.Errorf("ApplyMitigation(100, 0, 0.75) = %f, want 100.0", result)
	}

	// 50% mitigation
	result2 := ApplyMitigation(100, 0.50, 0.75)
	if math.Abs(result2-50.0) > 0.01 {
		t.Errorf("ApplyMitigation(100, 0.50, 0.75) = %f, want 50.0", result2)
	}

	// 90% mitigation capped at 75%
	result3 := ApplyMitigation(100, 0.90, 0.75)
	if math.Abs(result3-25.0) > 0.01 {
		t.Errorf("ApplyMitigation(100, 0.90, 0.75) = %f, want 25.0 (capped)", result3)
	}

	// Negative mitigation treated as 0
	result4 := ApplyMitigation(100, -0.5, 0.75)
	if math.Abs(result4-100.0) > 0.01 {
		t.Errorf("ApplyMitigation(100, -0.5, 0.75) = %f, want 100.0", result4)
	}
}

func TestGetConvictionDamageDescription(t *testing.T) {
	// Should not panic on zero max
	desc := GetConvictionDamageDescription(10, 0)
	if desc == "" {
		t.Error("GetConvictionDamageDescription returned empty string")
	}

	// Various thresholds
	desc1 := GetConvictionDamageDescription(2, 100)
	if desc1 != "a feeble jab at their resolve" {
		t.Errorf("expected feeble jab, got: %s", desc1)
	}

	desc2 := GetConvictionDamageDescription(60, 100)
	if desc2 != "a devastating attack on their will" {
		t.Errorf("expected devastating attack, got: %s", desc2)
	}
}
