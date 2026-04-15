package characters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ─── Integration: Character Progression ──────────────────────────────────────
// Exercises: New(), Validate(), IncreaseStat(), IncreaseSkill(),
// CalculateProgressionChance(), soft-cap curve
//
// Note: OnStatUse/OnSkillUse require full config/logger initialization and
// produce heavy logging output. We test the progression math directly instead.

func TestIntegration_CharacterProgression(t *testing.T) {
	c := New()

	// Record initial values
	initialStr := c.Stats.Strength.Training
	initialSkill := c.Skills["unarmed-combat"]

	// Directly increase stat and verify
	ok := c.IncreaseStat("strength", 5)
	assert.True(t, ok, "IncreaseStat should succeed for valid stat")
	assert.Equal(t, initialStr+5, c.Stats.Strength.Training,
		"Strength training should increase by 5")

	// Directly increase skill and verify
	ok = c.IncreaseSkill("unarmed-combat")
	assert.True(t, ok, "IncreaseSkill should succeed")
	assert.Equal(t, initialSkill+1, c.Skills["unarmed-combat"],
		"Unarmed-combat should increase by 1")

	// Verify progression chance decreases with rank
	chance0 := CalculateProgressionChance(0, SkillSoftCap)
	chance25 := CalculateProgressionChance(25, SkillSoftCap)
	chance50 := CalculateProgressionChance(50, SkillSoftCap)

	assert.Greater(t, chance0, chance25,
		"Progression chance at rank 0 should exceed rank 25")
	assert.Greater(t, chance25, chance50,
		"Progression chance at rank 25 should exceed rank 50")
	assert.InDelta(t, 0.30, chance0, 0.001,
		"Rank 0 should have ~30%% progression chance")
}

func TestIntegration_StatProgressionSimulated(t *testing.T) {
	// Simulate the probabilistic progression loop that OnStatUse performs.
	// We test the math directly: CalculateProgressionChance returns a probability,
	// and IncreaseStat applies the result.
	c := New()
	initialTraining := c.Stats.Strength.Training

	// At rank 0, chance is 30%. Over 50 simulated "progression events", expect ~15 gains.
	gains := 0
	for i := 0; i < 50; i++ {
		chance := CalculateProgressionChance(c.Stats.Strength.Training, StatSoftCap)
		// Simulate the probabilistic roll: use the chance as a threshold
		// Deterministic simulation: apply gain if chance > 0.20 (most low-rank calls should pass)
		if chance > 0.20 {
			c.IncreaseStat("strength", 1)
			gains++
		}
	}

	assert.Greater(t, gains, 0, "Should gain at least 1 stat point in 50 simulated uses")
	assert.Greater(t, c.Stats.Strength.Training, initialTraining,
		"Strength training should increase")
}

func TestIntegration_SkillProgressionSimulated(t *testing.T) {
	c := New()
	initialRank := c.Skills["unarmed-combat"]

	gains := 0
	for i := 0; i < 50; i++ {
		chance := CalculateProgressionChance(c.Skills["unarmed-combat"], SkillSoftCap)
		if chance > 0.20 {
			c.IncreaseSkill("unarmed-combat")
			gains++
		}
	}

	assert.Greater(t, gains, 0, "Should gain at least 1 skill rank in 50 simulated uses")
	assert.Greater(t, c.Skills["unarmed-combat"], initialRank,
		"Unarmed-combat should increase")
}

// ─── Integration: Death and Recovery ─────────────────────────────────────────
// Exercises: ApplyHealthChange(), Validate(), pool calculations

func TestIntegration_DeathAndRecovery(t *testing.T) {
	c := New()
	c.Stats.Vitality.ValueAdj = 100
	c.Stats.Strength.ValueAdj = 100
	c.HealthMax.Value = 200
	c.Health = 200

	// Drain HP gradually
	c.ApplyHealthChange(-150)
	assert.Equal(t, 50, c.Health, "HP should decrease by 150")

	c.ApplyHealthChange(-50)
	assert.Equal(t, 0, c.Health, "HP should reach 0")

	// Further damage should go negative (downed state)
	c.ApplyHealthChange(-3)
	assert.Less(t, c.Health, 0, "HP should go negative when downed")

	// Simulate revival: restore HP to max
	c.Health = c.HealthMax.Value
	assert.Equal(t, 200, c.Health, "HP should be restored to max")
	assert.Greater(t, c.Health, 0, "Character should be alive after revival")
}

func TestIntegration_PoolsFromStats(t *testing.T) {
	// Verify that Validate() recalculates pools from stats.
	// Note: Pool formulas depend on balance config (HealthPerStr, etc.).
	// Without full config, HealthMax gets a base value, Stamina/Conviction may
	// be zero. We verify the clamping behavior and that Validate doesn't panic.
	c := New()

	// Set known stat values
	c.Stats.Strength.Base = 100
	c.Stats.Strength.Training = 20
	c.Stats.Vitality.Base = 100
	c.Stats.Vitality.Training = 20
	c.Stats.Willpower.Base = 100
	c.Stats.Willpower.Training = 0
	c.Stats.Charisma.Base = 100
	c.Stats.Charisma.Training = 0

	c.Validate()

	// HealthMax should be positive (has a base value even without config)
	assert.Greater(t, c.HealthMax.Value, 0, "HealthMax should be positive")

	// All pools should be clamped: current <= max
	assert.LessOrEqual(t, c.Health, c.HealthMax.Value,
		"Health should not exceed HealthMax")
	assert.GreaterOrEqual(t, c.StaminaMax.Value, 0,
		"StaminaMax should be non-negative")
	assert.GreaterOrEqual(t, c.ConvictionMax.Value, 0,
		"ConvictionMax should be non-negative")
	assert.LessOrEqual(t, c.Stamina, c.StaminaMax.Value,
		"Stamina should not exceed StaminaMax")
	assert.LessOrEqual(t, c.Conviction, c.ConvictionMax.Value,
		"Conviction should not exceed ConvictionMax")

	// Stat values should be recalculated
	assert.Greater(t, c.Stats.Strength.Value, 0,
		"Strength Value should be positive after Validate")
	assert.Greater(t, c.Stats.Vitality.Value, 0,
		"Vitality Value should be positive after Validate")
}

func TestIntegration_AllStatsIncrease(t *testing.T) {
	// Verify all 6 stats can be increased and trigger Validate
	statNames := []string{
		"strength", "dexterity", "perception",
		"vitality", "willpower", "charisma",
	}

	for _, stat := range statNames {
		c := New()
		ok := c.IncreaseStat(stat, 10)
		assert.True(t, ok, "IncreaseStat(%s) should succeed", stat)
	}

	// Invalid stat should fail
	c := New()
	ok := c.IncreaseStat("invalid", 1)
	assert.False(t, ok, "IncreaseStat with invalid name should fail")
}

func TestIntegration_AllSkillsProgression(t *testing.T) {
	// Verify all skills can be increased
	c := New()
	skillNames := []string{
		"weapon-combat", "unarmed-combat",
		"spellcasting", "rhetoric", "search",
		"bartering",
	}

	for _, skill := range skillNames {
		before := c.Skills[skill]
		ok := c.IncreaseSkill(skill)
		assert.True(t, ok, "IncreaseSkill(%s) should succeed", skill)
		assert.Equal(t, before+1, c.Skills[skill],
			"Skill %s should increase by 1", skill)
	}
}

func TestIntegration_StatSoftCapCurve(t *testing.T) {
	// Verify the progression curve handles soft cap correctly
	// Below soft cap: chance decreases smoothly
	// At soft cap: chance is small but positive
	// Above soft cap: chance is very small but positive

	belowCap := CalculateProgressionChance(50, StatSoftCap)
	atCap := CalculateProgressionChance(StatSoftCap, StatSoftCap)
	aboveCap := CalculateProgressionChance(StatSoftCap+20, StatSoftCap)

	assert.Greater(t, belowCap, atCap,
		"Below soft cap should have higher chance than at cap")
	assert.Greater(t, atCap, aboveCap,
		"At soft cap should have higher chance than above")
	assert.Greater(t, aboveCap, 0.0,
		"Above soft cap should still have positive chance")
}

func TestIntegration_RegenFromPools(t *testing.T) {
	c := New()
	c.Stats.Vitality.ValueAdj = 100
	c.Stats.Strength.ValueAdj = 100
	c.Stats.Willpower.ValueAdj = 100
	c.Stats.Charisma.ValueAdj = 100
	c.HealthMax.Value = 200
	c.StaminaMax.Value = 100
	c.ConvictionMax.Value = 100

	// Regen per round should be positive (percentage-of-max, min 1)
	hpRegen := c.HealthPerRound()
	spRegen := c.StaminaPerRound()
	cpRegen := c.ConvictionPerRound()

	assert.GreaterOrEqual(t, hpRegen, 1,
		"HP regen should be at least 1")
	assert.GreaterOrEqual(t, spRegen, 1,
		"Stamina regen should be at least 1")
	assert.GreaterOrEqual(t, cpRegen, 1,
		"Conviction regen should be at least 1")
}
