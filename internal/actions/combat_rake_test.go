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
// ExecuteRake tests
// ---------------------------------------------------------------------------

// TestRake_NoAggro verifies that ExecuteRake returns NoTarget=true when the
// actor has no aggro set (not yet in combat).
func TestRake_NoAggro(t *testing.T) {
	char := characters.New()
	room := newTestRoom()
	actor := newStubActor(char, room)

	result := ExecuteRake(actor)

	assert.False(t, result.Executed, "rake with no aggro should not execute")
	assert.True(t, result.NoTarget, "rake with no aggro should set NoTarget")
	assert.False(t, result.OnCooldown, "NoTarget should take priority over cooldown reporting")
}

// TestRake_OnCooldown verifies that ExecuteRake returns OnCooldown=true when
// the special-move cooldown is active.
func TestRake_OnCooldown(t *testing.T) {
	char := characters.New()
	room := newTestRoom()
	actor := newStubActor(char, room)

	// Set aggro so we pass the nil check and reach the cooldown gate.
	char.Aggro = &characters.Aggro{MobInstanceId: 999999}

	// Burn the cooldown slot so the next Try call is blocked.
	char.Cooldowns.Try("special-move", "3 rounds")

	result := ExecuteRake(actor)

	assert.False(t, result.Executed, "rake should not execute when on cooldown")
	assert.True(t, result.OnCooldown, "rake should report OnCooldown")
}

// TestRake_NotClawed verifies the anatomy/identity gate: a non-clawed actor in
// combat with a valid target gets NotClawed=true and Executed=false, while a
// clawed actor passes the gate and executes the move.
func TestRake_NotClawed(t *testing.T) {
	// Seed two species: one fanged (not clawed), one clawed.
	cleanup := species.SeedSpeciesForTest(map[int]*species.Species{
		5001: {SpeciesId: 5001, Name: "wolf", BodyParts: []string{"legs", "mouth"}, NaturalAttack: items.Bite},
		5002: {SpeciesId: 5002, Name: "feline", BodyParts: []string{"legs", "mouth"}, NaturalAttack: items.Claws},
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

	t.Run("non-clawed actor returns NotClawed", func(t *testing.T) {
		char := characters.New()
		char.SpeciesId = 5001 // wolf — fanged, not clawed
		char.Aggro = &characters.Aggro{MobInstanceId: targetMob.InstanceId}

		result := ExecuteRake(newStubActor(char, newTestRoom()))

		assert.True(t, result.NotClawed, "non-clawed actor should return NotClawed=true")
		assert.False(t, result.Executed, "non-clawed actor should not execute the rake")
		assert.Equal(t, 0, result.MoveResult.Damage, "non-clawed rake should deal no damage")
	})

	t.Run("clawed actor passes the gate and executes", func(t *testing.T) {
		char := characters.New()
		char.SpeciesId = 5002 // feline — clawed
		char.Aggro = &characters.Aggro{MobInstanceId: targetMob.InstanceId}

		result := ExecuteRake(newStubActor(char, newTestRoom()))

		assert.False(t, result.NotClawed, "clawed actor should NOT return NotClawed")
		assert.True(t, result.Executed, "clawed actor should execute the rake")
	})
}

// TestRake_NoTargetAfterCooldown verifies that when aggro is set to an
// invalid mob instance ID (target gone), Executed is false and NoTarget
// is true (even after cooldown clears).
func TestRake_TargetGone(t *testing.T) {
	char := characters.New()
	room := newTestRoom()
	actor := newStubActor(char, room)

	// Aggro pointing at a nonexistent mob instance — target resolution fails.
	char.Aggro = &characters.Aggro{MobInstanceId: 999999}

	result := ExecuteRake(actor)

	// Cooldown Try fires first (setting it), so OnCooldown should be false.
	// Target resolution then fails → NoTarget.
	assert.False(t, result.Executed, "rake with missing target should not execute")
	assert.True(t, result.NoTarget, "rake should report NoTarget when the resolved target is gone")
	assert.False(t, result.OnCooldown, "cooldown should not be reported when target is gone")
}
