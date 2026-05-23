package mobcommands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBlindingSpit_RoutesToAction verifies that BlindingSpit delegates to
// actions.TriggerBlindingSpit without panicking. The mob from the seeded
// registry (instanceId 100) does not have the blinding-spit mutation, so
// the action blocks silently — but the wrapper must still return (true, nil).
func TestBlindingSpit_RoutesToAction(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)
	require.NotNil(t, mob)
	require.NotNil(t, room)

	handled, err := BlindingSpit("", mob, room)
	assert.True(t, handled, "wrapper must return handled=true")
	assert.NoError(t, err)
}

// TestBlindingSpit_CommandRegistered verifies that "blinding_spit" is
// present in the mob command registry.
func TestBlindingSpit_CommandRegistered(t *testing.T) {
	cmds := GetAllMobCommands()
	found := false
	for _, c := range cmds {
		if c == "blinding_spit" {
			found = true
			break
		}
	}
	assert.True(t, found, "blinding_spit must be registered in the mob command map")
}
