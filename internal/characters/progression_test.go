package characters

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/skills"
)

// Test constants matching the default balance config values.
// These are used instead of pulling from configs to keep tests self-contained.
const (
	SkillSoftCap = 50
	StatSoftCap  = 150
)

func TestCalculateProgressionChance_RankZero(t *testing.T) {
	chance := CalculateProgressionChance(0, SkillSoftCap)
	if chance != 0.30 {
		t.Errorf("Expected 30%% at rank 0, got %.4f", chance)
	}
}

func TestCalculateProgressionChance_NegativeRank(t *testing.T) {
	chance := CalculateProgressionChance(-5, SkillSoftCap)
	if chance != 0.30 {
		t.Errorf("Expected 30%% at negative rank, got %.4f", chance)
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
	expected := 0.30 * math.Exp(-3.0)
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

	if rankZero != 0.30 {
		t.Errorf("Expected 30%% at rank 0 for stats, got %.4f", rankZero)
	}
	if midRange >= rankZero || midRange <= atCap {
		t.Errorf("Mid-range should be between rank 0 and cap: %.4f not between %.4f and %.4f", midRange, rankZero, atCap)
	}
	if atCap > 0.05 {
		t.Errorf("At stat soft cap should be < 5%%, got %.4f", atCap)
	}
}

func TestIncreaseSkill_Basic(t *testing.T) {
	c := New()
	// All skills start at rank 1 (Stage 3.5). Should increase from 1 to 2.
	if !c.IncreaseSkill("unarmed-combat") {
		t.Error("Expected IncreaseSkill to return true for level 1→2")
	}
	if c.Skills["unarmed-combat"] != 2 {
		t.Errorf("Expected skill level 2, got %d", c.Skills["unarmed-combat"])
	}
}

func TestIncreaseSkill_NoCap(t *testing.T) {
	c := New()
	c.Skills["unarmed-combat"] = 4
	// No hard cap — should increase past 4
	if !c.IncreaseSkill("unarmed-combat") {
		t.Error("Expected IncreaseSkill to return true (no hard cap)")
	}
	if c.Skills["unarmed-combat"] != 5 {
		t.Errorf("Expected skill level 5, got %d", c.Skills["unarmed-combat"])
	}
}

func TestIncreaseSkill_Incremental(t *testing.T) {
	c := New()
	// All skills start at rank 1 (Stage 3.5). Increase several times.
	for expected := 2; expected <= 6; expected++ {
		if !c.IncreaseSkill("cast") {
			t.Errorf("Expected IncreaseSkill to return true for level %d", expected)
		}
		if c.Skills["cast"] != expected {
			t.Errorf("Expected skill level %d, got %d", expected, c.Skills["cast"])
		}
	}
}

func TestIncreaseStat_Strength(t *testing.T) {
	c := New()
	before := c.Stats.Strength.Training
	ok := c.IncreaseStat("strength", 1)
	if !ok {
		t.Error("Expected IncreaseStat to return true for valid stat")
	}
	if c.Stats.Strength.Training != before+1 {
		t.Errorf("Expected Training to be %d, got %d", before+1, c.Stats.Strength.Training)
	}
}

func TestIncreaseStat_AllStats(t *testing.T) {
	statNames := []string{"strength", "dexterity", "perception", "vitality", "willpower", "charisma"}
	for _, name := range statNames {
		c := New()
		ok := c.IncreaseStat(name, 3)
		if !ok {
			t.Errorf("Expected IncreaseStat to return true for %s", name)
		}
	}
}

func TestIncreaseStat_InvalidStat(t *testing.T) {
	c := New()
	ok := c.IncreaseStat("nonexistent", 1)
	if ok {
		t.Error("Expected IncreaseStat to return false for invalid stat name")
	}
}

func TestResolveSkillName_PassThrough(t *testing.T) {
	// With empty skillNameMap, all names pass through unchanged
	result := resolveSkillName("weapon-combat")
	if result != "weapon-combat" {
		t.Errorf("Expected 'weapon-combat' to pass through, got '%s'", result)
	}
	result = resolveSkillName("cast")
	if result != "cast" {
		t.Errorf("Expected 'cast' to pass through as 'cast', got '%s'", result)
	}
}

func TestGetCombatSkillLevel_UnarmedWithUnarmedCombat(t *testing.T) {
	// No weapon equipped → uses UnarmedCombat skill
	c := New()
	c.Skills["unarmed-combat"] = 3
	if got := c.GetCombatSkillLevel(); got != 3 {
		t.Errorf("Expected combat skill 3 from unarmed-combat, got %d", got)
	}
}

func TestGetCombatSkillLevel_BrawlingFallback(t *testing.T) {
	// No combat skills → minimum 1 (brawling fallback removed)
	c := New()
	delete(c.Skills, "unarmed-combat")
	delete(c.Skills, "weapon-combat")
	delete(c.Skills, "ranged-combat")
	if got := c.GetCombatSkillLevel(); got != 1 {
		t.Errorf("Expected combat skill minimum 1 with no skills, got %d", got)
	}
}

func TestGetCombatSkillLevel_NoSkillsReturns1(t *testing.T) {
	// No combat skills at all → minimum 1
	c := New()
	delete(c.Skills, "unarmed-combat")
	delete(c.Skills, "brawling")
	delete(c.Skills, "weapon-combat")
	delete(c.Skills, "ranged-combat")
	if got := c.GetCombatSkillLevel(); got != 1 {
		t.Errorf("Expected combat skill 1 (minimum), got %d", got)
	}
}

func TestGetCombatSkillTag_NoWeapon(t *testing.T) {
	c := New()
	if got := c.GetCombatSkillTag(); got != skills.UnarmedCombat {
		t.Errorf("Expected UnarmedCombat with no weapon, got %s", got)
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
		{0, 50, 29.0, 31.0},  // ~30%
		{10, 50, 12.0, 20.0}, // ~16%
		{25, 50, 3.0, 9.0},   // ~6.7%
		{40, 50, 1.0, 4.0},   // ~2.7%
		{50, 50, 0.5, 2.5},   // ~1.5%
	}

	for _, tt := range tests {
		chance := CalculateProgressionChance(tt.rank, tt.softCap) * 100
		if chance < tt.minPct || chance > tt.maxPct {
			t.Errorf("Rank %d (cap %d): expected %.1f-%.1f%%, got %.2f%%",
				tt.rank, tt.softCap, tt.minPct, tt.maxPct, chance)
		}
	}
}

func TestGetProgressionMultiplier(t *testing.T) {
	tests := []struct {
		skill    string
		expected float64
	}{
		{"weapon-combat", 0.3},
		{"unarmed-combat", 0.3},
		{"ranged-combat", 0.3},
		{"spellcasting", 0.5},
		{"cast", 0.5},
		{"tracking", 2.0},
		{"bartering", 2.0},
		{"foraging", 2.0},
		{"first-aid", 2.0},
		{"stealth", 2.0},
		{"unknown-skill", 1.0},
	}

	for _, tt := range tests {
		got := skills.GetProgressionMultiplier(tt.skill)
		if got != tt.expected {
			t.Errorf("GetProgressionMultiplier(%q) = %.1f, want %.1f", tt.skill, got, tt.expected)
		}
	}
}
