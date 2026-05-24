package mobcommands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBlindingFlash_RoutesToAction verifies that BlindingFlash delegates to
// actions.TriggerBlindingFlash without panicking. The mob from the seeded
// registry (instanceId 100) does not have the blinding-flash mutation, so the
// action blocks silently — but the wrapper must still return (true, nil).
func TestBlindingFlash_RoutesToAction(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)
	require.NotNil(t, mob)
	require.NotNil(t, room)

	handled, err := BlindingFlash("", mob, room)
	assert.True(t, handled, "wrapper must return handled=true")
	assert.NoError(t, err)
}

// TestBlindingFlashCommandRegistered verifies that "blinding-flash" is
// present in the mob command registry.
func TestBlindingFlashCommandRegistered(t *testing.T) {
	cmds := GetAllMobCommands()
	found := false
	for _, c := range cmds {
		if c == "blinding-flash" {
			found = true
			break
		}
	}
	assert.True(t, found, "blinding-flash must be registered in the mob command map")
}
