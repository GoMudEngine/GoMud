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
// ExecuteGore tests
// ---------------------------------------------------------------------------

// TestGore_NoAggro verifies that ExecuteGore returns NoTarget=true when
// the actor has no aggro set (not yet in combat).
func TestGore_NoAggro(t *testing.T) {
	char := characters.New()
	room := newTestRoom()
	actor := newStubActor(char, room)

	result := ExecuteGore(actor)

	assert.False(t, result.Executed, "gore with no aggro should not execute")
	assert.True(t, result.NoTarget, "gore with no aggro should set NoTarget")
	assert.False(t, result.OnCooldown, "NoTarget should take priority over cooldown reporting")
}

// TestGore_OnCooldown verifies that ExecuteGore returns OnCooldown=true
// when the special-move cooldown is active.
func TestGore_OnCooldown(t *testing.T) {
	char := characters.New()
	room := newTestRoom()
	actor := newStubActor(char, room)

	// Set aggro so we pass the nil check and reach the cooldown gate.
	char.Aggro = &characters.Aggro{MobInstanceId: 999999}

	// Burn the cooldown slot so the next Try call is blocked.
	char.Cooldowns.Try("special-move", "3 rounds")

	result := ExecuteGore(actor)

	assert.False(t, result.Executed, "gore should not execute when on cooldown")
	assert.True(t, result.OnCooldown, "gore should report OnCooldown")
}

// TestGore_NotHorned verifies the horned gate: a non-horned actor in combat
// with a valid target gets NotHorned=true and Executed=false.
func TestGore_NotHorned(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		7401: {SpeciesId: 7401, Name: "wolf", BodyParts: []string{"legs", "mouth"}, NaturalAttack: items.Bite},
		7402: {SpeciesId: 7402, Name: "human", BodyParts: []string{"arms", "hands", "legs"}},
	})
	defer cleanup()

	// Register a target mob so ResolveAggroTarget returns Found=true.
	targetMob := &mobs.Mob{InstanceId: 7499}
	targetMob.Character.Name = "Target"
	targetMob.Character.HealthMax.Value = 100
	targetMob.Character.Health = 100
	setCombatPositionParallel(&targetMob.Character, position.Standing)
	mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
	defer mobs.SetInstanceForTest(targetMob.InstanceId, nil)

	t.Run("fanged non-horned actor returns NotHorned", func(t *testing.T) {
		char := characters.New()
		char.SpeciesId = 7401 // wolf — fanged but not horned
		char.Aggro = &characters.Aggro{MobInstanceId: targetMob.InstanceId}

		result := ExecuteGore(newStubActor(char, newTestRoom()))

		assert.True(t, result.NotHorned, "fanged non-horned actor should return NotHorned=true")
		assert.False(t, result.Executed, "non-horned actor should not execute gore")
		assert.Equal(t, 0, result.MoveResult.Damage, "non-horned gore should deal no damage")
	})

	t.Run("humanoid without natural attack returns NotHorned", func(t *testing.T) {
		char := characters.New()
		char.SpeciesId = 7402 // human — no natural attack
		char.Aggro = &characters.Aggro{MobInstanceId: targetMob.InstanceId}

		result := ExecuteGore(newStubActor(char, newTestRoom()))

		assert.True(t, result.NotHorned, "humanoid (no natural attack) should return NotHorned=true")
		assert.False(t, result.Executed, "humanoid should not execute gore")
	})
}

// TestGore_Executed verifies that a horned actor in combat with a valid
// target executes the gore (Executed=true, NotHorned=false).
func TestGore_Executed(t *testing.T) {
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		7501: {SpeciesId: 7501, Name: "boar", BodyParts: []string{"legs", "mouth", "horns"}, NaturalAttack: items.Gore},
	})
	defer cleanup()

	targetMob := &mobs.Mob{InstanceId: 7599}
	targetMob.Character.Name = "Target"
	targetMob.Character.HealthMax.Value = 100
	targetMob.Character.Health = 100
	setCombatPositionParallel(&targetMob.Character, position.Standing)
	mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
	defer mobs.SetInstanceForTest(targetMob.InstanceId, nil)

	char := characters.New()
	char.SpeciesId = 7501 // boar — horned
	char.Aggro = &characters.Aggro{MobInstanceId: targetMob.InstanceId}

	result := ExecuteGore(newStubActor(char, newTestRoom()))

	assert.False(t, result.NotHorned, "horned boar should NOT return NotHorned")
	assert.True(t, result.Executed, "horned boar should execute gore")
}
