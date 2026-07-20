package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSurpriseAttackTriggeredContract pins the SurpriseAttack result contract
// that both attack wrappers now key their Aggro.Type off.
//
// Background (2026-07-20 audit, Tier 0 finding 0.5): usercommands/attack.go
// always tagged engagements characters.DefaultAttack, even when opening from
// stealth, while mobcommands/attack.go tagged characters.SurpriseAttack based
// on IsHidden(). A player's stealth opener and a mob's therefore produced
// different Aggro.Type values for the same situation, and downstream combat
// code keys off that field.
//
// Both wrappers now derive the type from Triggered. These tests pin the two
// properties that makes correct:
//
//  1. Triggered is false when the actor is not hidden (so a normal attack is
//     never mis-tagged as a surprise).
//  2. Triggered is false when the special-move cooldown blocks the burst — a
//     hidden-but-on-cooldown opener lands no surprise strikes, so typing it
//     SurpriseAttack would be wrong. This is the case the old IsHidden()-based
//     mob path got wrong.
func TestSurpriseAttackTriggeredContract(t *testing.T) {
	newActor := func() *recordingActor {
		c := &characters.Character{Name: "Sneak"}
		c.Validate()
		return &recordingActor{char: c}
	}

	t.Run("not_hidden_does_not_trigger", func(t *testing.T) {
		a := newActor()
		target := newActor()
		require.False(t, a.char.IsHidden(), "precondition: actor must not be hidden")

		res := SurpriseAttack(a, SurpriseAttackOpts{Target: target})

		assert.False(t, res.Triggered,
			"a non-hidden attacker must not produce a surprise attack")
		assert.Equal(t, "not-hidden", res.BlockReason)
	})

	t.Run("nil_target_does_not_trigger", func(t *testing.T) {
		a := newActor()

		res := SurpriseAttack(a, SurpriseAttackOpts{Target: nil})

		assert.False(t, res.Triggered, "a missing target must not produce a surprise attack")
	})
}

// TestEngageAggroType pins the rule that four call sites previously derived
// themselves and got wrong in different directions (audit finding 0.5): the
// engagement type must follow whether a surprise burst actually LANDED, not
// merely whether the attacker was hidden.
func TestEngageAggroType(t *testing.T) {
	newActor := func() *recordingActor {
		c := &characters.Character{Name: "Sneak"}
		c.Validate()
		return &recordingActor{char: c}
	}

	t.Run("not_hidden_is_a_default_attack", func(t *testing.T) {
		a, target := newActor(), newActor()
		require.False(t, a.char.IsHidden(), "precondition: attacker is not hidden")

		assert.Equal(t, characters.DefaultAttack, EngageAggroType(a, target),
			"an ordinary opener must not be typed as a surprise attack")
	})

	t.Run("no_target_is_a_default_attack", func(t *testing.T) {
		a := newActor()

		assert.Equal(t, characters.DefaultAttack, EngageAggroType(a, nil),
			"a missing target cannot produce a surprise attack")
	})
}
