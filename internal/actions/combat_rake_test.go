package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
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
