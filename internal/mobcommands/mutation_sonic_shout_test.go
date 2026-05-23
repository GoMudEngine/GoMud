package mobcommands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSonicShout_RoutesToAction verifies that SonicShout delegates to
// actions.TriggerSonicShout without panicking. The mob from the seeded
// registry (instanceId 100) does not have the sonic-shout mutation, so the
// action blocks silently — but the wrapper must still return (true, nil).
func TestSonicShout_RoutesToAction(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)
	require.NotNil(t, mob)
	require.NotNil(t, room)

	handled, err := SonicShout("", mob, room)
	assert.True(t, handled, "wrapper must return handled=true")
	assert.NoError(t, err)
}

// TestSonicShoutCommandRegistered verifies that "sonic-shout" is present
// in the mob command registry.
func TestSonicShoutCommandRegistered(t *testing.T) {
	cmds := GetAllMobCommands()
	found := false
	for _, c := range cmds {
		if c == "sonic-shout" {
			found = true
			break
		}
	}
	assert.True(t, found, "sonic-shout must be registered in the mob command map")
}
