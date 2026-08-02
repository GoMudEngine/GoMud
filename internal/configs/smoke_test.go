package configs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ─── Smoke Tests ─────────────────────────────────────────────────────────────
// These tests verify that config loading and validation don't panic and
// produce sane default values. They catch the most common startup panics
// caused by config changes.

// TestSmoke_ConfigValidatesWithoutPanic verifies that calling Validate()
// on a zero-value Config doesn't panic. This catches fields that assume
// non-nil maps, slices, or nested structs.
func TestSmoke_ConfigValidatesWithoutPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		c := Config{}
		c.Validate()
	}, "Config.Validate() must not panic on zero-value config")
}

// TestSmoke_BalanceConfigValid verifies that all critical balance knobs
// are positive and within reasonable ranges after validation.
func TestSmoke_BalanceConfigValid(t *testing.T) {
	bal := GetBalanceConfig()

	// Roll spread
	assert.GreaterOrEqual(t, float64(bal.RollSpread), 0.05,
		"RollSpread must be >= 0.05")
	assert.LessOrEqual(t, float64(bal.RollSpread), 0.50,
		"RollSpread must be <= 0.50")

	// Defense floor / attack floor
	assert.GreaterOrEqual(t, float64(bal.MinDefenseChance), 0.0,
		"MinDefenseChance must be non-negative")
	assert.LessOrEqual(t, float64(bal.MinDefenseChance), 0.50,
		"MinDefenseChance must be <= 0.50")
	assert.GreaterOrEqual(t, float64(bal.MinAttackHitChance), 0.0,
		"MinAttackHitChance must be non-negative")
	assert.LessOrEqual(t, float64(bal.MinAttackHitChance), 0.50,
		"MinAttackHitChance must be <= 0.50")

	// Skill multiplier curve
	assert.Greater(t, float64(bal.SkillMultiplierBase), 0.0,
		"SkillMultiplierBase must be positive")
	assert.Greater(t, float64(bal.SkillMultiplierMax), float64(bal.SkillMultiplierBase),
		"SkillMultiplierMax must exceed SkillMultiplierBase")

	// Soft caps
	assert.Greater(t, int(bal.SkillSoftCap), 0,
		"SkillSoftCap must be positive")
	assert.Greater(t, int(bal.StatProgressionSoftCap), 0,
		"StatProgressionSoftCap must be positive")

	// Mitigation caps — must be between 0 and 1
	assert.Greater(t, float64(bal.PhysicalMitigationCap), 0.0,
		"PhysicalMitigationCap must be positive")
	assert.LessOrEqual(t, float64(bal.PhysicalMitigationCap), 1.0,
		"PhysicalMitigationCap must be <= 1.0")
	assert.Greater(t, float64(bal.MagicalMitigationCap), 0.0,
		"MagicalMitigationCap must be positive")
	assert.LessOrEqual(t, float64(bal.MagicalMitigationCap), 1.0,
		"MagicalMitigationCap must be <= 1.0")
	assert.Greater(t, float64(bal.ConvictionMitigationCap), 0.0,
		"ConvictionMitigationCap must be positive")
	assert.LessOrEqual(t, float64(bal.ConvictionMitigationCap), 1.0,
		"ConvictionMitigationCap must be <= 1.0")

	// Damage scales — must be positive
	assert.Greater(t, float64(bal.MeleeDamageScale), 0.0,
		"MeleeDamageScale must be positive")
	assert.Greater(t, float64(bal.SpellDamageScale), 0.0,
		"SpellDamageScale must be positive")
	assert.Greater(t, float64(bal.RhetoricDamageScale), 0.0,
		"RhetoricDamageScale must be positive")

	// Resource penalty curve
	assert.Greater(t, float64(bal.ResourcePenaltyCurve), 0.0,
		"ResourcePenaltyCurve must be positive")

	// Regen rates — must be positive
	assert.Greater(t, float64(bal.PlayerHealthRegenPct), 0.0,
		"PlayerHealthRegenPct must be positive")
	assert.Greater(t, float64(bal.PlayerStaminaRegenPct), 0.0,
		"PlayerStaminaRegenPct must be positive")
	assert.Greater(t, float64(bal.PlayerConvictionRegenPct), 0.0,
		"PlayerConvictionRegenPct must be positive")

	// Progression
	assert.Greater(t, float64(bal.BaseProgressionChance), 0.0,
		"BaseProgressionChance must be positive")
	assert.Greater(t, int(bal.UsesPerRank), 0,
		"UsesPerRank must be positive")
}

// TestSmoke_GamePlayConfigValid verifies essential GamePlay config values.
func TestSmoke_GamePlayConfigValid(t *testing.T) {
	gp := GetGamePlayConfig()

	// Verify the config struct is accessible without panic
	// (tests the getter + validation pipeline)
	_ = gp.UseSkillProgression
}

// TestSmoke_GetConfigDoesNotPanic verifies that GetConfig() works without
// a loaded config file by falling back to validated defaults.
func TestSmoke_GetConfigDoesNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_ = GetConfig()
	}, "GetConfig() must not panic without a config file")
}
