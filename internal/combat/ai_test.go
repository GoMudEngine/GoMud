package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

// ─── GetAIProfile ───────────────────────────────────────────────────────────

func TestGetAIProfile(t *testing.T) {
	tests := []struct {
		name        string
		profileName string
		custom      map[string]int
		wantKey     string
		wantValue   int
	}{
		{
			name:        "default profile",
			profileName: "default",
			custom:      nil,
			wantKey:     "bash",
			wantValue:   25,
		},
		{
			name:        "caster profile",
			profileName: "caster",
			custom:      nil,
			wantKey:     "kick",
			wantValue:   10,
		},
		{
			name:        "aggressive profile",
			profileName: "aggressive",
			custom:      nil,
			wantKey:     "bash",
			wantValue:   40,
		},
		{
			name:        "unknown profile falls back to default",
			profileName: "nonexistent",
			custom:      nil,
			wantKey:     "bash",
			wantValue:   25,
		},
		{
			name:        "custom overrides profile",
			profileName: "default",
			custom:      map[string]int{"bash": 99},
			wantKey:     "bash",
			wantValue:   99,
		},
		{
			name:        "custom adds new key",
			profileName: "default",
			custom:      map[string]int{"flee": 50},
			wantKey:     "flee",
			wantValue:   50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := GetAIProfile(tt.profileName, tt.custom)
			assert.Equal(t, tt.wantValue, profile[tt.wantKey])
		})
	}
}

func TestGetAIProfile_DoesNotMutateOriginal(t *testing.T) {
	origBash := aiProfiles["default"]["bash"]
	profile := GetAIProfile("default", map[string]int{"bash": 999})
	assert.Equal(t, 999, profile["bash"])
	assert.Equal(t, origBash, aiProfiles["default"]["bash"], "original should not be modified")
}

// ─── CanUseBash ─────────────────────────────────────────────────────────────

func TestCanUseBash(t *testing.T) {
	tests := []struct {
		name      string
		hasShield bool
		cooldown  bool
		want      bool
	}{
		{"has shield, no cooldown", true, false, true},
		{"no shield", false, false, false},
		{"has shield, on cooldown", true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := characters.New()
			if tt.hasShield {
				c.Equipment.Offhand = items.Item{
					ItemId: 1,
					Spec: &items.ItemSpec{
						Type:            items.Offhand,
						DamageReduction: 5,
						BlockRating:     10,
					},
				}
			}
			if tt.cooldown {
				c.Cooldowns["special-move"] = 3
			}
			assert.Equal(t, tt.want, CanUseBash(c))
		})
	}
}

// ─── CanUseTrip ─────────────────────────────────────────────────────────────

func TestCanUseTrip(t *testing.T) {
	tests := []struct {
		name     string
		pos position.State
		cooldown bool
		want     bool
	}{
		{"standing, no cooldown", position.Standing, false, true},
		{"standing, on cooldown", position.Standing, true, false},
		{"clinched", position.Clinch, false, false},
		{"grounded", position.Mount, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := characters.New()
			setCombatPositionParallel(c, tt.pos)
			if tt.cooldown {
				c.Cooldowns["special-move"] = 3
			}
			assert.Equal(t, tt.want, CanUseTrip(c))
		})
	}
}

// ─── CanUseKick ─────────────────────────────────────────────────────────────

func TestCanUseKick(t *testing.T) {
	t.Run("no cooldown", func(t *testing.T) {
		c := characters.New()
		assert.True(t, CanUseKick(c))
	})

	t.Run("on cooldown", func(t *testing.T) {
		c := characters.New()
		c.Cooldowns["special-move"] = 3
		assert.False(t, CanUseKick(c))
	})
}

// ─── CanUseGrapple ──────────────────────────────────────────────────────────

func TestCanUseGrapple(t *testing.T) {
	tests := []struct {
		name     string
		pos position.State
		cooldown bool
		want     bool
	}{
		{"standing, no cooldown", position.Standing, false, true},
		{"on cooldown", position.Standing, true, false},
		{"already clinched", position.Clinch, false, false},
		{"already grounded", position.Mount, false, false},
		{"prone (not a grapple position)", position.Prone, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := characters.New()
			setCombatPositionParallel(c, tt.pos)
			if tt.cooldown {
				c.Cooldowns["special-move"] = 3
			}
			assert.Equal(t, tt.want, CanUseGrapple(c))
		})
	}
}

// ─── CanUseSubmit ───────────────────────────────────────────────────────────

func TestCanUseSubmit(t *testing.T) {
	tests := []struct {
		name       string
		pos        position.State
		controller bool
		cooldown   bool
		want       bool
	}{
		{"grounded + controller", position.Mount, true, false, true},
		{"grounded but not controller", position.Mount, false, false, false},
		{"standing + controller", position.Standing, true, false, false},
		{"grounded + controller + cooldown", position.Mount, true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := characters.New()
			// For PositionGrounded the helper writes the Mount FSM state
			// with InControl; the bottom-side case uses Guard with
			// Controlled instead so IsController() returns false. Skip
			// the helper for that case and wire the FSM directly.
			if tt.pos == position.Mount && !tt.controller {
				_ = c.Position.TransitionToClinch(
					position.GrappleData{Partner: state.ActorRef{UserId: 1}, IsControllerRole: false},
					state.TransitionReason{Trigger: position.TriggerGrappleEntry},
				)
				_ = c.Position.TransitionToGuard(
					position.GrappleData{Partner: state.ActorRef{UserId: 1}, IsControllerRole: false},
					state.TransitionReason{Trigger: position.TriggerGuardPull},
				)
			} else {
				setCombatPositionParallel(c, tt.pos)
				// For the controller case, setCombatPositionParallel sets
				// both Position (Mount) and Control (Controlling), so
				// IsController() returns true via Control.State().
			}
			if tt.cooldown {
				c.Cooldowns["special-move"] = 3
			}
			assert.Equal(t, tt.want, CanUseSubmit(c))
		})
	}
}

// ─── CanUseCast ─────────────────────────────────────────────────────────────

func TestCanUseCast(t *testing.T) {
	tests := []struct {
		name       string
		spellBook  map[string]int
		conviction int
		casting    bool
		want       bool
	}{
		{"has spells + conviction", map[string]int{"sparks": 1}, 10, false, true},
		{"no spells", map[string]int{}, 10, false, false},
		{"low conviction", map[string]int{"sparks": 1}, 2, false, false},
		{"already casting", map[string]int{"sparks": 1}, 10, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := characters.New()
			c.SpellBook = tt.spellBook
			c.Conviction = tt.conviction
			if tt.casting {
				c.Activity = activity.NewMachine()
				_ = c.Activity.TransitionToCasting(
					activity.CastingData{SpellId: "sparks"},
					state.TransitionReason{Trigger: activity.TriggerCastBegin},
				)
			}
			assert.Equal(t, tt.want, CanUseCast(c))
		})
	}
}

// ─── ScoreBash ──────────────────────────────────────────────────────────────

func TestScoreBash(t *testing.T) {
	makeMob := func(hasShield bool, combatSkill int) *mobs.Mob {
		m := &mobs.Mob{}
		m.Character = *characters.New()
		if hasShield {
			m.Character.Equipment.Offhand = items.Item{
				ItemId: 1,
				Spec: &items.ItemSpec{
					Type:            items.Offhand,
					DamageReduction: 5,
					BlockRating:     10,
				},
			}
		}
		m.Character.Skills[string(skills.WeaponCombat)] = combatSkill
		return m
	}

	makeTarget := func(healthPct float64, prone bool) *characters.Character {
		c := characters.New()
		c.HealthMax.Value = 100
		c.Health = int(healthPct)
		if prone {
			setCombatPositionParallel(c, position.Prone)
		}
		return c
	}

	t.Run("no shield → 0", func(t *testing.T) {
		score := ScoreBash(makeMob(false, 0), makeTarget(100, false))
		assert.Equal(t, 0, score)
	})

	t.Run("shield + healthy target → high score", func(t *testing.T) {
		score := ScoreBash(makeMob(true, 60), makeTarget(80, false))
		assert.Greater(t, score, 50)
	})

	t.Run("target already prone → reduced score", func(t *testing.T) {
		score := ScoreBash(makeMob(true, 0), makeTarget(80, true))
		assert.Less(t, score, 30)
	})
}

// ─── ScoreTrip ──────────────────────────────────────────────────────────────

func TestScoreTrip(t *testing.T) {
	makeMob := func(dex int, unarmedSkill int, pos position.State) *mobs.Mob {
		m := &mobs.Mob{}
		m.Character = *characters.New()
		m.Character.Stats.Dexterity.ValueAdj = dex
		m.Character.Skills[string(skills.UnarmedCombat)] = unarmedSkill
		setCombatPositionParallel(&m.Character, pos)
		return m
	}

	t.Run("target already prone → 0", func(t *testing.T) {
		target := characters.New()
		setCombatPositionParallel(target, position.Prone)
		target.HealthMax.Value = 100
		target.Health = 100
		score := ScoreTrip(makeMob(100, 0, position.Standing), target)
		assert.Equal(t, 0, score)
	})

	t.Run("high dex mob + low dex target → high score", func(t *testing.T) {
		target := characters.New()
		target.Stats.Dexterity.ValueAdj = 5
		target.HealthMax.Value = 100
		target.Health = 100
		score := ScoreTrip(makeMob(100, 50, position.Standing), target)
		assert.Greater(t, score, 60)
	})
}

// ─── ScoreKick ──────────────────────────────────────────────────────────────

func TestScoreKick(t *testing.T) {
	makeMob := func(str int, unarmedSkill int) *mobs.Mob {
		m := &mobs.Mob{}
		m.Character = *characters.New()
		m.Character.Stats.Strength.ValueAdj = str
		m.Character.Skills[string(skills.UnarmedCombat)] = unarmedSkill
		return m
	}

	t.Run("base score", func(t *testing.T) {
		target := characters.New()
		target.HealthMax.Value = 100
		target.Health = 100
		score := ScoreKick(makeMob(10, 0), target)
		assert.Equal(t, 45, score) // base only
	})

	t.Run("high strength + skill + low target HP", func(t *testing.T) {
		target := characters.New()
		target.HealthMax.Value = 100
		target.Health = 20
		score := ScoreKick(makeMob(100, 50), target)
		assert.Greater(t, score, 70) // 45 + 20 + 15 + 10 = 90
	})
}

// ─── ScoreGrapple ───────────────────────────────────────────────────────────

func TestScoreGrapple(t *testing.T) {
	makeMob := func(str int, unarmedSkill int, health int, maxHealth int) *mobs.Mob {
		m := &mobs.Mob{}
		m.Character = *characters.New()
		m.Character.Stats.Strength.ValueAdj = str
		m.Character.Skills[string(skills.UnarmedCombat)] = unarmedSkill
		m.Character.Health = health
		m.Character.HealthMax.Value = maxHealth
		return m
	}

	t.Run("already in grapple → 0", func(t *testing.T) {
		mob := makeMob(100, 50, 100, 100)
		setCombatPositionParallel(&mob.Character, position.Clinch)
		target := characters.New()
		target.HealthMax.Value = 100
		target.Health = 100
		score := ScoreGrapple(mob, target)
		assert.Equal(t, 0, score)
	})

	t.Run("strong mob + skilled + weak target", func(t *testing.T) {
		target := characters.New()
		target.Stats.Strength.ValueAdj = 50
		target.HealthMax.Value = 100
		target.Health = 20
		score := ScoreGrapple(makeMob(150, 50, 100, 100), target)
		assert.Greater(t, score, 80) // 50 + 30 + 20 + 15 = 115
	})

	t.Run("mob low health → penalized", func(t *testing.T) {
		target := characters.New()
		target.Stats.Strength.ValueAdj = 200 // stronger than mob
		target.HealthMax.Value = 100
		target.Health = 100
		score := ScoreGrapple(makeMob(100, 0, 10, 100), target)
		assert.Equal(t, 0, score) // 50 (base) - 50 (low hp) = 0, no str bonus since target stronger
	})
}
