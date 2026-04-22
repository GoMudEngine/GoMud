package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

// ValidateAggro must treat Flee-type aggro (with no UserId / MobInstanceId)
// as valid — a fleeing player's Aggro intentionally has no target.
// Without this, the next combat round's ValidateAggro returns false, the
// caller runs RetargetOrEnd which acquires a new target and clears the Flee
// type, and the player never actually flees. Regression for the flee bug
// reported 2026-04-22.
func TestValidateAggro_FleeTypeWithNoTarget_IsValid(t *testing.T) {
	c := characters.New()
	c.Aggro = &characters.Aggro{
		UserId:        0,
		MobInstanceId: 0,
		Type:          characters.Flee,
	}

	ok := ValidateAggro(c)
	assert.True(t, ok, "Flee aggro with no target should be valid")
	assert.NotNil(t, c.Aggro, "ValidateAggro must not end a Flee aggro")
	assert.Equal(t, characters.Flee, c.Aggro.Type, "aggro type must remain Flee")
}

// Parity check: SpellCast-type aggro with no target is already treated as
// valid (existing behavior). Locking it in.
func TestValidateAggro_SpellCastTypeWithNoTarget_IsValid(t *testing.T) {
	c := characters.New()
	c.Aggro = &characters.Aggro{
		UserId:        0,
		MobInstanceId: 0,
		Type:          characters.SpellCast,
	}

	ok := ValidateAggro(c)
	assert.True(t, ok, "SpellCast aggro with no target should be valid")
	assert.NotNil(t, c.Aggro, "ValidateAggro must not end a SpellCast aggro")
}

// DefaultAttack with no target is stale state — ValidateAggro should
// invalidate it.
func TestValidateAggro_DefaultAttackWithNoTarget_IsInvalid(t *testing.T) {
	c := characters.New()
	c.Aggro = &characters.Aggro{
		UserId:        0,
		MobInstanceId: 0,
		Type:          characters.DefaultAttack,
	}

	ok := ValidateAggro(c)
	assert.False(t, ok, "DefaultAttack with no target is stale, should be invalid")
	assert.Nil(t, c.Aggro, "EndAggro must have cleared aggro")
}

// Nil aggro returns false without side effects.
func TestValidateAggro_NilAggro(t *testing.T) {
	c := characters.New()
	c.Aggro = nil

	ok := ValidateAggro(c)
	assert.False(t, ok)
	assert.Nil(t, c.Aggro)
}
