package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
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
	// Seed SpeciesId 0 (used by characters.New()) with arms so the anatomy
	// gate does not block the "can bash with shield" cases.
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		0: {SpeciesId: 0, Name: "human", BodyParts: []string{"arms", "hands", "legs"}},
	})
	defer cleanup()

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
	// Seed species 0 (characters.New()) with legs so the anatomy gate does not
	// block the "can trip" cases.
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		0: {SpeciesId: 0, Name: "human", BodyParts: []string{"legs"}},
	})
	defer cleanup()

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
	// Seed species 0 (characters.New()) with legs so the anatomy gate does not
	// block the "can kick" cases.
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		0: {SpeciesId: 0, Name: "human", BodyParts: []string{"legs"}},
	})
	defer cleanup()

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
	// Seed the default species (SpeciesId 0, used by characters.New()) with
	// arms so the anatomy gate does not block the "can grapple" cases.
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		0: {SpeciesId: 0, Name: "human", BodyParts: []string{"arms", "hands", "legs"}},
	})
	defer cleanup()

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
	// Seed SpeciesId 0 (characters.New default) with arms so these cases
	// exercise the position/controller/cooldown logic, not the anatomy gate.
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		0: {SpeciesId: 0, Name: "test", BodyParts: []string{"arms", "hands", "legs"}},
	})
	defer cleanup()

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
					position.GrappleData{Partner: state.ActorRef{UserId: 1}},
					state.TransitionReason{Trigger: position.TriggerGrappleEntry},
				)
				_ = c.Position.TransitionToGuard(
					position.GrappleData{Partner: state.ActorRef{UserId: 1}},
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

func TestCanUseHamstring_FangedOrClawedWithLegs(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		7501: {SpeciesId: 7501, Name: "wolf", BodyParts: []string{"legs", "mouth"}, NaturalAttack: items.Bite},
		7502: {SpeciesId: 7502, Name: "human", BodyParts: []string{"arms", "hands", "legs", "mouth"}},
		7503: {SpeciesId: 7503, Name: "serpent", BodyParts: []string{"mouth", "skin"}, NaturalAttack: items.Bite},
	})
	defer cleanup()

	wolf := &characters.Character{SpeciesId: 7501, Cooldowns: characters.Cooldowns{}}
	human := &characters.Character{SpeciesId: 7502, Cooldowns: characters.Cooldowns{}}
	serpent := &characters.Character{SpeciesId: 7503, Cooldowns: characters.Cooldowns{}}

	if !CanUseHamstring(wolf) {
		t.Error("fanged, legged wolf should hamstring")
	}
	if CanUseHamstring(human) {
		t.Error("plain humanoid (no natural fang/claw) should not hamstring")
	}
	if CanUseHamstring(serpent) {
		t.Error("legless serpent should not hamstring (no legs to cut)")
	}
}

func TestCanUseSubmit_RequiresArms(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		7401: {SpeciesId: 7401, Name: "humanoid", BodyParts: []string{"arms", "hands", "legs"}},
		7402: {SpeciesId: 7402, Name: "noarms", BodyParts: []string{"legs", "mouth"}},
	})
	defer cleanup()

	armed := characters.New()
	armed.SpeciesId = 7401
	setCombatPositionParallel(armed, position.Mount) // grounded + controller

	beast := characters.New()
	beast.SpeciesId = 7402
	setCombatPositionParallel(beast, position.Mount)

	if !CanUseSubmit(armed) {
		t.Error("armed grappler-controller should be able to submit")
	}
	if CanUseSubmit(beast) {
		t.Error("no-arms creature must NOT submit (submission holds need arms)")
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

// ─── CanUseBash (shield-or-naturalBash AND arms-or-naturalBash) ─────────────

func TestCanUseBash_ShieldOrNaturalAndArms(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		7201: {SpeciesId: 7201, Name: "humanoid", BodyParts: []string{"arms", "hands"}},
		7202: {SpeciesId: 7202, Name: "golem", BodyParts: []string{"arms"}, NaturalBash: true},
		7203: {SpeciesId: 7203, Name: "noarms-elemental", BodyParts: []string{}, NaturalBash: true},
		7204: {SpeciesId: 7204, Name: "wolf", BodyParts: []string{"legs", "mouth"}},
		7205: {SpeciesId: 7205, Name: "shielded-noarms-beast", BodyParts: []string{"tentacles"}},
	})
	defer cleanup()

	// Humanoid (has arms, no shield, not natural) — must NOT bash.
	withNoShield := &characters.Character{SpeciesId: 7201}
	if CanUseBash(withNoShield) {
		t.Error("humanoid with no shield and not natural must NOT bash")
	}

	// Humanoid with shield equipped — CAN bash.
	withShield := &characters.Character{SpeciesId: 7201}
	withShield.Equipment.Offhand = items.Item{
		ItemId: 1,
		Spec: &items.ItemSpec{
			Type:            items.Offhand,
			DamageReduction: 5,
			BlockRating:     10,
		},
	}
	if !CanUseBash(withShield) {
		t.Error("armed humanoid with shield should be able to bash")
	}

	// Golem: naturalBash + has arms — CAN bash without shield.
	// Note: HasShield() already returns true for NaturalBash creatures; this
	// tests that the arms gate passes too.
	golem := &characters.Character{SpeciesId: 7202}
	if !CanUseBash(golem) {
		t.Error("golem (naturalBash) should bash without a shield")
	}

	// No-arms elemental: naturalBash + no arms — CAN bash (naturalBash bypasses both gates).
	elemental := &characters.Character{SpeciesId: 7203}
	if !CanUseBash(elemental) {
		t.Error("no-arms elemental (naturalBash) should bash")
	}

	// Wolf: no arms, not natural, no shield — must NOT bash.
	wolf := &characters.Character{SpeciesId: 7204}
	if CanUseBash(wolf) {
		t.Error("no-arms non-natural wolf must NOT bash")
	}

	// Shielded non-natural beast without arms — has a shield but cannot
	// brace/drive forward without arms; must NOT bash. This is the case
	// that the arms gate adds beyond the existing shield check.
	shieldedNoArms := &characters.Character{SpeciesId: 7205}
	shieldedNoArms.Equipment.Offhand = items.Item{
		ItemId: 2,
		Spec: &items.ItemSpec{
			Type:            items.Offhand,
			DamageReduction: 5,
			BlockRating:     10,
		},
	}
	if CanUseBash(shieldedNoArms) {
		t.Error("shielded non-natural no-arms beast must NOT bash (lacks arms gate)")
	}
}

// ─── CanUseTrip / CanUseKick (anatomy gate) ─────────────────────────────────

func TestCanUseTrip_RequiresLegs(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		7301: {SpeciesId: 7301, Name: "wolf", BodyParts: []string{"legs", "mouth"}},
		7302: {SpeciesId: 7302, Name: "serpent", BodyParts: []string{"mouth"}},
	})
	defer cleanup()
	wolf := &characters.Character{SpeciesId: 7301}
	serpent := &characters.Character{SpeciesId: 7302}
	if !CanUseTrip(wolf) {
		t.Error("legged wolf should trip")
	}
	if CanUseTrip(serpent) {
		t.Error("legless serpent must NOT trip")
	}
	if !CanUseKick(wolf) {
		t.Error("legged wolf should kick")
	}
	if CanUseKick(serpent) {
		t.Error("legless serpent must NOT kick")
	}
}

// ─── CanUseGrapple (anatomy gate) ───────────────────────────────────────────

func TestCanUseGrapple_RequiresArms(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		7101: {SpeciesId: 7101, Name: "humanoid", BodyParts: []string{"arms", "hands", "legs"}},
		7102: {SpeciesId: 7102, Name: "wolf", BodyParts: []string{"legs", "mouth", "tail"}},
	})
	defer cleanup()

	humanoid := &characters.Character{SpeciesId: 7101}
	wolf := &characters.Character{SpeciesId: 7102}

	if !CanUseGrapple(humanoid) {
		t.Error("armed humanoid should be able to grapple")
	}
	if CanUseGrapple(wolf) {
		t.Error("no-arms wolf must NOT be able to grapple")
	}
}
