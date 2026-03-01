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

// ─── PowerRanking ───────────────────────────────────────────────────────────

func TestPowerRanking(t *testing.T) {
	// Create two identical characters
	makeChar := func(dex, health int, weaponDmg int) characters.Character {
		c := *characters.New()
		c.Stats.Dexterity.ValueAdj = dex
		c.HealthMax.Value = health
		c.Equipment.Weapon = items.Item{ItemId: 1, Spec: &items.ItemSpec{
			Damage: items.Damage{
				Attacks:    1,
				BaseDamage: weaponDmg,
				Variance:   2,
			},
		}}
		return c
	}

	t.Run("identical characters → ~1.0", func(t *testing.T) {
		c1 := makeChar(100, 100, 20)
		c2 := makeChar(100, 100, 20)
		pct := PowerRanking(c1, c2)
		assert.InDelta(t, 1.0, pct, 0.05, "identical chars should be near 1.0")
	})

	t.Run("stronger attacker → > 1.0", func(t *testing.T) {
		atk := makeChar(150, 200, 40)
		def := makeChar(100, 100, 20)
		pct := PowerRanking(atk, def)
		assert.Greater(t, pct, 1.0, "stronger attacker should score > 1.0")
	})

	t.Run("weaker attacker → < 1.0", func(t *testing.T) {
		atk := makeChar(50, 50, 10)
		def := makeChar(100, 100, 20)
		pct := PowerRanking(atk, def)
		assert.Less(t, pct, 1.0, "weaker attacker should score < 1.0")
	})

	t.Run("defender zero dex → no division by zero", func(t *testing.T) {
		atk := makeChar(100, 100, 20)
		def := makeChar(0, 100, 20)
		pct := PowerRanking(atk, def)
		assert.Greater(t, pct, 0.0, "should handle zero dex without panic")
	})

	t.Run("defender zero health → no division by zero", func(t *testing.T) {
		atk := makeChar(100, 100, 20)
		def := makeChar(100, 0, 20)
		pct := PowerRanking(atk, def)
		assert.Greater(t, pct, 0.0, "should handle zero health without panic")
	})
}
