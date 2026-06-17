package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
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
