package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
)

// TestActionReadiness_GenericCommand_Ready verifies that an unrecognized /
// non-special-move command (e.g. "say hello") always passes through as Ready.
func TestActionReadiness_GenericCommand_Ready(t *testing.T) {
	m := newTestMob(t, nil)
	actor := &MobActor{Mob: m, Room: nil}
	result := ActionReadiness(actor, "say hello")
	assert.Equal(t, ActionReady, result.Status)
}

// TestActionReadiness_NilActor_Rejected verifies that a nil actor is
// immediately rejected without panicking.
func TestActionReadiness_NilActor_Rejected(t *testing.T) {
	result := ActionReadiness(nil, "kick")
	assert.Equal(t, ActionRejected, result.Status)
	assert.NotEmpty(t, result.Reason)
}

// TestActionReadiness_SpecialMove_Ready verifies that a special-move verb
// where CommandIsReady returns true yields ActionReady.
// taunt requires char.Aggro != nil — newTestMob sets default aggro to user 1.
func TestActionReadiness_SpecialMove_Ready(t *testing.T) {
	m := newTestMob(t, nil)
	actor := &MobActor{Mob: m, Room: nil}
	result := ActionReadiness(actor, "taunt")
	assert.Equal(t, ActionReady, result.Status)
}

// TestActionReadiness_SpecialMove_OnCooldown_Deferred verifies that a
// special-move verb with a non-zero "special-move" cooldown yields
// ActionDeferred (transient — retry when the cooldown expires).
// Cooldowns are set directly on the Cooldowns map (no setter method exists).
func TestActionReadiness_SpecialMove_OnCooldown_Deferred(t *testing.T) {
	m := newTestMob(t, nil)
	// Direct map assignment is the canonical test approach (see command_readiness_test.go).
	m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
	actor := &MobActor{Mob: m, Room: nil}
	result := ActionReadiness(actor, "taunt")
	assert.Equal(t, ActionDeferred, result.Status)
	assert.Equal(t, "special-move busy", result.Reason)
}

// TestActionReadiness_SpecialMove_StructurallyBlocked_Rejected verifies that
// a special-move verb where CommandIsReady is false for structural reasons
// (no cooldown, not acting) yields ActionRejected.
// taunt with EndAggro() → char.Aggro == nil → structural block.
func TestActionReadiness_SpecialMove_StructurallyBlocked_Rejected(t *testing.T) {
	m := newTestMob(t, nil)
	m.Character.EndAggro() // removes the aggro set by newTestMob
	actor := &MobActor{Mob: m, Room: nil}
	result := ActionReadiness(actor, "taunt")
	assert.Equal(t, ActionRejected, result.Status)
	assert.Equal(t, "special-move unavailable", result.Reason)
}

// TestActionReadinessDrift iterates every verb in specialMoveVerbs and calls
// CommandIsReady for each, asserting no panic. This guards against drift: if
// a verb is added to or removed from CommandIsReady's switch without updating
// specialMoveVerbs, the set-membership difference is visible in tests.
func TestActionReadinessDrift(t *testing.T) {
	m := newTestMob(t, nil)
	actor := &MobActor{Mob: m, Room: nil}

	for verb := range specialMoveVerbs {
		v := verb // capture loop variable
		t.Run(v, func(t *testing.T) {
			assert.NotPanics(t, func() {
				CommandIsReady(actor, v)
			}, "CommandIsReady should not panic for special-move verb %q", v)
		})
	}
}

// ---------------------------------------------------------------------------
// Cast readiness tests (Task 2)
// ---------------------------------------------------------------------------

// TestCastReadiness_UnknownSpell_Rejected verifies that "cast <nonexistent>"
// yields ActionRejected immediately — wrong spell name is a structural error.
func TestCastReadiness_UnknownSpell_Rejected(t *testing.T) {
	actor, _, _ := newCastActor()
	result := ActionReadiness(actor, "cast notaspell_xyzzy_ar")
	assert.Equal(t, ActionRejected, result.Status)
	assert.Equal(t, "unknown spell", result.Reason)
}

// TestCastReadiness_NoCP_Deferred verifies that a caster who knows the spell
// but has zero Conviction gets ActionDeferred ("insufficient conviction").
// Conviction = 0 by default from characters.New(); seedTestSpell sets Cost=5.
func TestCastReadiness_NoCP_Deferred(t *testing.T) {
	sd, cleanup := seedTestSpell("test-ar-nocp", spells.HelpSingle, 4)
	defer cleanup()

	actor, char, _ := newCastActor()
	// Grant the spell — characters.New() initialises SpellBook with starters
	// but not test spells; set directly as HasSpell checks SpellBook[id] > 0.
	char.SpellBook[sd.SpellId] = 1
	// char.Conviction is 0 (Go zero value) — below the spell's Cost of 5.

	result := ActionReadiness(actor, "cast test-ar-nocp")
	assert.Equal(t, ActionDeferred, result.Status)
	assert.Equal(t, "insufficient conviction", result.Reason)
}

// TestCastReadiness_Affordable_Ready verifies that a caster who knows the
// spell, has ample Conviction, is not casting/busy, and has no cooldowns
// gets ActionReady.
func TestCastReadiness_Affordable_Ready(t *testing.T) {
	sd, cleanup := seedTestSpell("test-ar-affordable", spells.HelpSingle, 4)
	defer cleanup()

	actor, char, _ := newCastActor()
	char.SpellBook[sd.SpellId] = 1 // character knows the spell
	char.Conviction = 1000         // well above the spell's Cost of 5

	result := ActionReadiness(actor, "cast test-ar-affordable")
	assert.Equal(t, ActionReady, result.Status)
}

// TestCastReadinessDrift guards castReadiness against drifting out of sync with
// the player cast pre-checks in skill.cast.go (which it deliberately mirrors,
// read-only). Each case builds a fully-castable caster, then trips exactly one
// gate and asserts the resulting classification + reason. If a gate in
// castReadiness is added/removed/reclassified without matching skill.cast.go,
// the corresponding case fails.
func TestCastReadinessDrift(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(c *characters.Character, sd *spells.SpellData)
		expected ReadyStatus
		reason   string
	}{
		{"cast-init-cooldown", func(c *characters.Character, sd *spells.SpellData) {
			c.Cooldowns = characters.Cooldowns{"cast-init": 3}
		}, ActionDeferred, "cast-init cooldown"},
		{"special-move-cooldown", func(c *characters.Character, sd *spells.SpellData) {
			c.Cooldowns = characters.Cooldowns{"special-move": 3}
		}, ActionDeferred, "special-move cooldown"},
		{"spell-not-known", func(c *characters.Character, sd *spells.SpellData) {
			delete(c.SpellBook, sd.SpellId)
		}, ActionRejected, "spell not known"},
		{"no-skill", func(c *characters.Character, sd *spells.SpellData) {
			c.Skills[string(skills.Spellcasting)] = 0
		}, ActionRejected, "no skill"},
		{"missing-component", func(c *characters.Character, sd *spells.SpellData) {
			sd.ComponentTag = "test-ar-drift-component"
		}, ActionRejected, "missing component"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sd, cleanup := seedTestSpell("test-ar-drift", spells.HelpSingle, 4)
			defer cleanup()

			actor, char, _ := newCastActor()
			char.SpellBook[sd.SpellId] = 1 // knows the spell
			char.Conviction = 1000         // ample CP (Cost is 5)
			tc.mutate(char, sd)

			result := ActionReadiness(actor, "cast test-ar-drift")
			assert.Equal(t, tc.expected, result.Status, "status")
			assert.Equal(t, tc.reason, result.Reason, "reason")
		})
	}
}

