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
// ExecuteMaul tests
// ---------------------------------------------------------------------------

// TestMaul_NoAggro verifies that ExecuteMaul returns NoTarget=true when the
// actor has no aggro set (not yet in combat).
func TestMaul_NoAggro(t *testing.T) {
	char := characters.New()
	room := newTestRoom()
	actor := newStubActor(char, room)

	result := ExecuteMaul(actor)

	assert.False(t, result.Executed, "maul with no aggro should not execute")
	assert.True(t, result.NoTarget, "maul with no aggro should set NoTarget")
	assert.False(t, result.OnCooldown, "NoTarget should take priority over cooldown reporting")
}

// TestMaul_OnCooldown verifies that ExecuteMaul returns OnCooldown=true when
// the special-move cooldown is active.
func TestMaul_OnCooldown(t *testing.T) {
	char := characters.New()
	room := newTestRoom()
	actor := newStubActor(char, room)

	// Set aggro so we pass the nil check and reach the cooldown gate.
	char.Aggro = &characters.Aggro{MobInstanceId: 999999}

	// Burn the cooldown slot so the next Try call is blocked.
	char.Cooldowns.Try("special-move", "3 rounds")

	result := ExecuteMaul(actor)

	assert.False(t, result.Executed, "maul should not execute when on cooldown")
	assert.True(t, result.OnCooldown, "maul should report OnCooldown")
}

// TestMaul_NotFanged verifies the anatomy/identity gate: a non-fanged actor in
// combat with a valid target gets NotFanged=true and Executed=false, while a
// fanged actor passes the gate and executes the move.
func TestMaul_NotFanged(t *testing.T) {
	// Seed two species: one clawed (not fanged), one fanged.
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		5001: {SpeciesId: 5001, Name: "feline", BodyParts: []string{"legs", "paws"}, NaturalAttack: items.Claws},
		5002: {SpeciesId: 5002, Name: "wolf", BodyParts: []string{"legs", "mouth"}, NaturalAttack: items.Bite},
	})
	defer cleanup()

	// Register a target mob so ResolveAggroTarget returns Found=true.
	targetMob := &mobs.Mob{InstanceId: 5099}
	targetMob.Character.Name = "Target"
	targetMob.Character.HealthMax.Value = 100
	targetMob.Character.Health = 100
	setCombatPositionParallel(&targetMob.Character, position.Standing)
	mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
	defer mobs.SetInstanceForTest(targetMob.InstanceId, nil)

	t.Run("non-fanged actor returns NotFanged", func(t *testing.T) {
		char := characters.New()
		char.SpeciesId = 5001 // feline — clawed, not fanged
		char.Aggro = &characters.Aggro{MobInstanceId: targetMob.InstanceId}

		result := ExecuteMaul(newStubActor(char, newTestRoom()))

		assert.True(t, result.NotFanged, "non-fanged actor should return NotFanged=true")
		assert.False(t, result.Executed, "non-fanged actor should not execute the maul")
		assert.Equal(t, 0, result.MoveResult.Damage, "non-fanged maul should deal no damage")
	})

	t.Run("fanged actor passes the gate and executes", func(t *testing.T) {
		char := characters.New()
		char.SpeciesId = 5002 // wolf — fanged
		char.Aggro = &characters.Aggro{MobInstanceId: targetMob.InstanceId}

		result := ExecuteMaul(newStubActor(char, newTestRoom()))

		assert.False(t, result.NotFanged, "fanged actor should NOT return NotFanged")
		assert.True(t, result.Executed, "fanged actor should execute the maul")
	})
}

// TestMaul_TargetGone verifies that when aggro is set to an invalid mob
// instance ID (target gone), Executed is false and NoTarget is true.
func TestMaul_TargetGone(t *testing.T) {
	char := characters.New()
	room := newTestRoom()
	actor := newStubActor(char, room)

	// Aggro pointing at a nonexistent mob instance — target resolution fails.
	char.Aggro = &characters.Aggro{MobInstanceId: 999999}

	result := ExecuteMaul(actor)

	// Cooldown Try fires first (setting it), so OnCooldown should be false.
	// Target resolution then fails → NoTarget.
	assert.False(t, result.Executed, "maul with missing target should not execute")
	assert.True(t, result.NoTarget, "maul should report NoTarget when the resolved target is gone")
	assert.False(t, result.OnCooldown, "cooldown should not be reported when target is gone")
}
