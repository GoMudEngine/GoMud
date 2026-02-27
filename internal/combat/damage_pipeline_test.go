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

func TestCalcRawDamage_Physical(t *testing.T) {
	// Physical: stat * SkillMult * itemMult * MeleeDamageScale(0.30)
	// Stat 100, rank 0, mult 1.0 → 100 * 1.0 * 1.0 * 0.30 = 30
	raw := CalcRawDamage(100, 0, 1.0, ChannelPhysical)
	if math.Abs(raw-30.0) > 0.01 {
		t.Errorf("CalcRawDamage physical(100, 0, 1.0) = %f, want 30.0", raw)
	}

	// Stat 100, rank 50, mult 1.0 → 100 * 3.0 * 1.0 * 0.30 = 90
	raw2 := CalcRawDamage(100, 50, 1.0, ChannelPhysical)
	if math.Abs(raw2-90.0) > 0.01 {
		t.Errorf("CalcRawDamage physical(100, 50, 1.0) = %f, want 90.0", raw2)
	}

	// Zero item multiplier should fallback to 0.30
	// 100 * 1.0 * 0.30 * 0.30 = 9.0
	raw3 := CalcRawDamage(100, 0, 0, ChannelPhysical)
	if math.Abs(raw3-9.0) > 0.01 {
		t.Errorf("CalcRawDamage physical(100, 0, 0) = %f, want 9.0 (fallback)", raw3)
	}
}

func TestCalcRawDamage_Magical(t *testing.T) {
	// Magical: stat * SkillMult * itemMult * SpellDamageScale(1.0)
	// No stat normalization — flat multiplier
	// Stat 100, rank 0, mult 1.0 → 100 * 1.0 * 1.0 * 1.0 = 100
	raw := CalcRawDamage(100, 0, 1.0, ChannelMagical)
	if math.Abs(raw-100.0) > 0.01 {
		t.Errorf("CalcRawDamage magical(100, 0, 1.0) = %f, want 100.0", raw)
	}
}

func TestCalcRawDamage_Conviction(t *testing.T) {
	// Conviction: stat * SkillMult * itemMult * RhetoricDamageScale(1.0)
	// Stat 100, rank 0, mult 0.5 → 100 * 1.0 * 0.5 * 1.0 = 50
	raw := CalcRawDamage(100, 0, 0.5, ChannelConviction)
	if math.Abs(raw-50.0) > 0.01 {
		t.Errorf("CalcRawDamage conviction(100, 0, 0.5) = %f, want 50.0", raw)
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

func TestResourceMultiplier(t *testing.T) {
	// 100% resource → 1.0 (no penalty)
	m100 := ResourceMultiplier(100, 100, 0.28)
	if math.Abs(m100-1.0) > 0.001 {
		t.Errorf("ResourceMultiplier(100, 100, 0.28) = %f, want 1.0", m100)
	}

	// 0% resource → 1 - penaltyMax = 0.72
	m0 := ResourceMultiplier(0, 100, 0.28)
	if math.Abs(m0-0.72) > 0.01 {
		t.Errorf("ResourceMultiplier(0, 100, 0.28) = %f, want ~0.72", m0)
	}

	// 5% resource → ~0.747 (penalty ~25%)
	m5 := ResourceMultiplier(5, 100, 0.28)
	if m5 < 0.73 || m5 > 0.76 {
		t.Errorf("ResourceMultiplier(5, 100, 0.28) = %f, want ~0.747", m5)
	}

	// 50% resource → ~0.93 (barely noticeable)
	m50 := ResourceMultiplier(50, 100, 0.28)
	if m50 < 0.92 || m50 > 0.94 {
		t.Errorf("ResourceMultiplier(50, 100, 0.28) = %f, want ~0.93", m50)
	}

	// Monotonically decreasing as resource drops
	prev := ResourceMultiplier(100, 100, 0.28)
	for pct := 99; pct >= 0; pct-- {
		cur := ResourceMultiplier(pct, 100, 0.28)
		if cur > prev+0.001 { // small tolerance for float
			t.Errorf("ResourceMultiplier(%d, 100, 0.28) = %f > prev %f — not monotonic", pct, cur, prev)
		}
		prev = cur
	}

	// Edge case: max <= 0 → 1.0
	mZeroMax := ResourceMultiplier(50, 0, 0.28)
	if mZeroMax != 1.0 {
		t.Errorf("ResourceMultiplier(50, 0, 0.28) = %f, want 1.0", mZeroMax)
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
