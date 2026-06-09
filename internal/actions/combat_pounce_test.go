package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// ExecutePounce tests
// ---------------------------------------------------------------------------

// TestPounce_NoAggro verifies that ExecutePounce returns NoTarget=true when
// the actor has no aggro set (not yet in combat).
func TestPounce_NoAggro(t *testing.T) {
	char := characters.New()
	room := newTestRoom()
	actor := newStubActor(char, room)

	result := ExecutePounce(actor)

	assert.False(t, result.Executed, "pounce with no aggro should not execute")
	assert.True(t, result.NoTarget, "pounce with no aggro should set NoTarget")
	assert.False(t, result.OnCooldown, "NoTarget should take priority over cooldown reporting")
}

// TestPounce_OnCooldown verifies that ExecutePounce returns OnCooldown=true
// when the special-move cooldown is active.
func TestPounce_OnCooldown(t *testing.T) {
	char := characters.New()
	room := newTestRoom()
	actor := newStubActor(char, room)

	// Set aggro so we pass the nil check and reach the cooldown gate.
	char.Aggro = &characters.Aggro{MobInstanceId: 999999}

	// Burn the cooldown slot so the next Try call is blocked.
	char.Cooldowns.Try("special-move", "3 rounds")

	result := ExecutePounce(actor)

	assert.False(t, result.Executed, "pounce should not execute when on cooldown")
	assert.True(t, result.OnCooldown, "pounce should report OnCooldown")
}

// TestPounce_NotPredator verifies the compound predator gate: a non-predator
// actor in combat with a valid target gets NotPredator=true and Executed=false,
// while a legged fanged actor passes the gate and executes the move.
func TestPounce_NotPredator(t *testing.T) {
	// Seed three species: no-legs fanged (not predator), legged no-attack
	// (not predator), and legged fanged (predator).
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		6001: {SpeciesId: 6001, Name: "serpent", BodyParts: []string{"mouth"}, NaturalAttack: items.Bite},
		6002: {SpeciesId: 6002, Name: "human", BodyParts: []string{"arms", "hands", "legs"}},
		6003: {SpeciesId: 6003, Name: "wolf", BodyParts: []string{"legs", "mouth"}, NaturalAttack: items.Bite},
	})
	defer cleanup()

	// Register a target mob so ResolveAggroTarget returns Found=true.
	targetMob := &mobs.Mob{InstanceId: 6099}
	targetMob.Character.Name = "Target"
	targetMob.Character.HealthMax.Value = 100
	targetMob.Character.Health = 100
	setCombatPositionParallel(&targetMob.Character, position.Standing)
	mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
	defer mobs.SetInstanceForTest(targetMob.InstanceId, nil)

	t.Run("legless fanged actor returns NotPredator", func(t *testing.T) {
		char := characters.New()
		char.SpeciesId = 6001 // serpent — fanged but no legs
		char.Aggro = &characters.Aggro{MobInstanceId: targetMob.InstanceId}

		result := ExecutePounce(newStubActor(char, newTestRoom()))

		assert.True(t, result.NotPredator, "legless fanged actor should return NotPredator=true")
		assert.False(t, result.Executed, "legless fanged actor should not execute pounce")
		assert.Equal(t, 0, result.MoveResult.Damage, "non-predator pounce should deal no damage")
	})

	t.Run("legged no-attack actor returns NotPredator", func(t *testing.T) {
		char := characters.New()
		char.SpeciesId = 6002 // human — legged but no claws or fangs
		char.Aggro = &characters.Aggro{MobInstanceId: targetMob.InstanceId}

		result := ExecutePounce(newStubActor(char, newTestRoom()))

		assert.True(t, result.NotPredator, "human (legged, no natural attack) should return NotPredator=true")
		assert.False(t, result.Executed, "non-predator human should not execute pounce")
	})

	t.Run("legged fanged actor passes the gate and executes", func(t *testing.T) {
		char := characters.New()
		char.SpeciesId = 6003 // wolf — legged + fanged
		char.Aggro = &characters.Aggro{MobInstanceId: targetMob.InstanceId}

		result := ExecutePounce(newStubActor(char, newTestRoom()))

		assert.False(t, result.NotPredator, "legged fanged wolf should NOT return NotPredator")
		assert.True(t, result.Executed, "legged fanged wolf should execute pounce")
	})
}

// TestPounce_Executed verifies that a legged clawed actor in combat with a
// valid target executes the pounce (Executed=true, NotPredator=false).
func TestPounce_Executed(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		6101: {SpeciesId: 6101, Name: "feline", BodyParts: []string{"legs", "paws"}, NaturalAttack: items.Claws},
	})
	defer cleanup()

	targetMob := &mobs.Mob{InstanceId: 6199}
	targetMob.Character.Name = "Target"
	targetMob.Character.HealthMax.Value = 100
	targetMob.Character.Health = 100
	setCombatPositionParallel(&targetMob.Character, position.Standing)
	mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
	defer mobs.SetInstanceForTest(targetMob.InstanceId, nil)

	char := characters.New()
	char.SpeciesId = 6101 // feline — legged + clawed
	char.Aggro = &characters.Aggro{MobInstanceId: targetMob.InstanceId}

	result := ExecutePounce(newStubActor(char, newTestRoom()))

	assert.False(t, result.NotPredator, "legged clawed feline should NOT return NotPredator")
	assert.True(t, result.Executed, "legged clawed feline should execute pounce")
}
