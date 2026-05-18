package combat_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

func setupBalanceForSubmissionTests(t *testing.T) {
	// Ensure balance config is loaded so the sub roll picks up the
	// T3 knobs.
	cfg := configs.GetBalanceConfig()
	if float64(cfg.SubmissionAttemptAlpha) == 0 {
		t.Helper()
		t.Skip("balance config not initialized for tests")
	}
}

func newCharFor(t *testing.T, str, vit, unarmedSkill int) *characters.Character {
	t.Helper()
	c := characters.New()
	c.Stats.Strength.Base = str
	c.Stats.Strength.ValueAdj = str
	c.Stats.Vitality.Base = vit
	c.Stats.Vitality.ValueAdj = vit
	if c.Skills == nil {
		c.Skills = map[string]int{}
	}
	c.Skills[string(skills.UnarmedCombat)] = unarmedSkill
	return c
}

func TestRollSubmissionAttempt_Structure(t *testing.T) {
	setupBalanceForSubmissionTests(t)
	atk := newCharFor(t, 120, 100, 30)
	def := newCharFor(t, 100, 100, 10)
	res := combat.RollSubmissionAttempt(atk, def, position.SubArmbar)
	assert.Equal(t, position.SubArmbar, res.SubType)
	assert.NotZero(t, res.AttackerScore)
	assert.NotZero(t, res.DefenderScore)
	// Tier is one of the four enum values
	assert.True(t, res.Tier >= combat.SubTierBad && res.Tier <= combat.SubTierCrit)
}

func TestRollSubmissionAttempt_BadTierOnVeryLowZ(t *testing.T) {
	setupBalanceForSubmissionTests(t)
	cfg := configs.GetBalanceConfig()
	assert.Equal(t, combat.SubTierBad,
		combat.ClassifySubmissionTier(false, float64(cfg.SubBadZThreshold)-0.5))
}

func TestClassifySubmissionTier_BoundaryConditions(t *testing.T) {
	setupBalanceForSubmissionTests(t)
	cfg := configs.GetBalanceConfig()
	bad := float64(cfg.SubBadZThreshold)
	crit := float64(cfg.SubCritZThreshold)

	// Strictly less than bad threshold → Bad (atk roll didn't succeed
	// here either way; we use bad-tier when z < bad)
	assert.Equal(t, combat.SubTierBad, combat.ClassifySubmissionTier(false, bad-0.1))
	// Equal to bad threshold and failed → Neutral
	assert.Equal(t, combat.SubTierNeutral, combat.ClassifySubmissionTier(false, bad))
	// Failed and above bad threshold → Neutral
	assert.Equal(t, combat.SubTierNeutral, combat.ClassifySubmissionTier(false, 0.0))
	// Success below crit threshold → Success
	assert.Equal(t, combat.SubTierSuccess, combat.ClassifySubmissionTier(true, crit-0.1))
	// Success at or above crit threshold → Crit
	assert.Equal(t, combat.SubTierCrit, combat.ClassifySubmissionTier(true, crit))
	assert.Equal(t, combat.SubTierCrit, combat.ClassifySubmissionTier(true, crit+1.0))
}
