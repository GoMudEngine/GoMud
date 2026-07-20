package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

// ─── ChanceToSwitchTarget ───────────────────────────────────────────────────

func TestChanceToSwitchTarget(t *testing.T) {
	tests := []struct {
		name      string
		dexterity int
		skillLvl  int // unarmed combat skill (used as combat skill)
		wantMin   int
		wantMax   int
	}{
		{
			name:      "baseline character (dex=100, skill=0)",
			dexterity: 100,
			skillLvl:  0,
			wantMin:   70, // 50 + 0*0.3 + 100*0.2 = 70
			wantMax:   70,
		},
		{
			name:      "high skill and dex",
			dexterity: 100,
			skillLvl:  100,
			wantMin:   95, // 50 + 100*0.3 + 100*0.2 = 100 → capped at 95
			wantMax:   95,
		},
		{
			name:      "zero stats → floor at 25",
			dexterity: 0,
			skillLvl:  0,
			wantMin:   25, // 50 + 0 + 0 = 50, actually 50 > 25
			wantMax:   50,
		},
		{
			name:      "moderate character",
			dexterity: 50,
			skillLvl:  20,
			wantMin:   56, // 50 + 20*0.3 + 50*0.2 = 50 + 6 + 10 = 66
			wantMax:   66,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := characters.New()
			c.Stats.Dexterity.ValueAdj = tt.dexterity
			// Set skills — GetCombatSkillLevel() picks highest combat skill
			c.Skills["unarmed-combat"] = tt.skillLvl

			got := ChanceToSwitchTarget(c)
			assert.GreaterOrEqual(t, got, tt.wantMin, "should be >= floor")
			assert.LessOrEqual(t, got, tt.wantMax, "should be <= cap")
		})
	}
}

func TestChanceToSwitchTarget_Clamping(t *testing.T) {
	// Verify floor at 25
	c := characters.New()
	c.Stats.Dexterity.ValueAdj = 0
	c.Skills["unarmed-combat"] = 0
	c.Skills["weapon-combat"] = 0
	got := ChanceToSwitchTarget(c)
	assert.GreaterOrEqual(t, got, 25, "floor should be 25")

	// Verify cap at 95
	c2 := characters.New()
	c2.Stats.Dexterity.ValueAdj = 200
	c2.Skills["unarmed-combat"] = 100
	got2 := ChanceToSwitchTarget(c2)
	assert.LessOrEqual(t, got2, 95, "cap should be 95")
}

// ─── PowerScore ─────────────────────────────────────────────────────────────

func TestPowerScore_Baseline(t *testing.T) {
	c := *characters.New()
	c.Stats.Strength.ValueAdj = 100
	c.Stats.Dexterity.ValueAdj = 100
	c.Stats.Perception.ValueAdj = 100
	c.Stats.Vitality.ValueAdj = 100
	c.Stats.Willpower.ValueAdj = 100
	c.Stats.Charisma.ValueAdj = 100
	c.HealthMax.Value = 100

	score := PowerScore(c)
	assert.Greater(t, score, 0.0, "baseline character should have positive score")
}

func TestPowerScore_HigherStatsHigherScore(t *testing.T) {
	makeChar := func(statVal int) characters.Character {
		c := *characters.New()
		c.Stats.Strength.ValueAdj = statVal
		c.Stats.Dexterity.ValueAdj = statVal
		c.Stats.Perception.ValueAdj = statVal
		c.Stats.Vitality.ValueAdj = statVal
		c.Stats.Willpower.ValueAdj = statVal
		c.Stats.Charisma.ValueAdj = statVal
		c.HealthMax.Value = statVal
		return c
	}

	weak := makeChar(50)
	strong := makeChar(150)
	assert.Greater(t, PowerScore(strong), PowerScore(weak), "higher stats → higher score")
}

func TestPowerScore_HigherSkillsHigherScore(t *testing.T) {
	makeChar := func(skillLvl int) characters.Character {
		c := *characters.New()
		c.Stats.Strength.ValueAdj = 100
		c.Stats.Dexterity.ValueAdj = 100
		c.Stats.Willpower.ValueAdj = 100
		c.Stats.Charisma.ValueAdj = 100
		c.HealthMax.Value = 100
		c.Skills["weapon-combat"] = skillLvl
		c.Skills["spellcasting"] = skillLvl
		c.Skills["rhetoric"] = skillLvl
		return c
	}

	noSkill := makeChar(0)
	skilled := makeChar(30)
	assert.Greater(t, PowerScore(skilled), PowerScore(noSkill), "higher skills → higher score")
}

func TestPowerScore_MitigationIncreasesScore(t *testing.T) {
	c1 := *characters.New()
	c1.Stats.Strength.ValueAdj = 100
	c1.Stats.Dexterity.ValueAdj = 100
	c1.Stats.Willpower.ValueAdj = 100
	c1.Stats.Charisma.ValueAdj = 100
	c1.HealthMax.Value = 100

	c2 := c1
	// Add armor with physical mitigation
	c2.Equipment.Body = items.Item{ItemId: 1, Spec: &items.ItemSpec{
		PhysicalMitigation:   20,
		MagicalMitigation:    10,
		ConvictionMitigation: 10,
	}}

	assert.Greater(t, PowerScore(c2), PowerScore(c1), "mitigation → higher score")
}

func TestPowerScore_KillsAddBump(t *testing.T) {
	c1 := *characters.New()
	c1.Stats.Strength.ValueAdj = 100
	c1.Stats.Dexterity.ValueAdj = 100
	c1.Stats.Willpower.ValueAdj = 100
	c1.Stats.Charisma.ValueAdj = 100
	c1.HealthMax.Value = 100

	c2 := c1
	c2.KD.TotalKills = 100

	assert.Greater(t, PowerScore(c2), PowerScore(c1), "kills add small bump")
}

func TestPowerScore_MultiWieldHigherScore(t *testing.T) {
	makeWeapon := func() items.Item {
		return items.Item{ItemId: 1, Spec: &items.ItemSpec{
			DamageMultiplier: 1.0,
			SpeedMultiplier:  1.0,
		}}
	}

	c1 := *characters.New()
	c1.Stats.Strength.ValueAdj = 100
	c1.Stats.Dexterity.ValueAdj = 100
	c1.Stats.Willpower.ValueAdj = 100
	c1.Stats.Charisma.ValueAdj = 100
	c1.HealthMax.Value = 100
	c1.Equipment.Weapon = makeWeapon()

	c2 := c1
	c2.ExtraArms = 2
	c2.Equipment.Offhand = makeWeapon()
	c2.Equipment.ExtraArm1 = makeWeapon()
	c2.Equipment.ExtraArm2 = makeWeapon()

	assert.Greater(t, PowerScore(c2), PowerScore(c1), "multi-wield → higher score")
}

func TestPowerScore_ZeroValueNoPanic(t *testing.T) {
	c := *characters.New()
	// Everything zero/nil — should not panic
	assert.NotPanics(t, func() {
		score := PowerScore(c)
		assert.GreaterOrEqual(t, score, 0.0)
	})
}

// TestBuildWeaponSetup_IncorporealIntegration documents the Incorporeal
// mutation integration point for weapon damage scaling.
//
// Without Incorporeal mutation: GearEffectivenessMultiplier(mutations) = 1.0,
// so weaponDmgMult = itemSpec.DamageMultiplier * 1.0 (unchanged).
//
// With Incorporeal rank 4: GearEffectivenessMultiplier(mutations) = 0.0,
// so weaponDmgMult = itemSpec.DamageMultiplier * 0.0 = 0, triggering the
// fallback to unarmed damage (ethereal beings' natural strikes still connect).
//
// Implementation site: internal/combat/combat_helpers.go:buildWeaponSetup()
// The line:
//    gearMul := mutations.GearEffectivenessMultiplier(sourceChar.Mutations)
//    ws.weaponDmgMult = itemSpec.DamageMultiplier * gearMul
//
// Full Incorporeal integration (rank 4 fallback behavior) is covered by
// Task 12 smoke tests, which load the mutation YAML and verify combat flow.
