package mobcommands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestToxicBite_RoutesToAction verifies that ToxicBite delegates to
// actions.TriggerToxicBite without panicking. The mob from the seeded
// registry (instanceId 100) does not have the toxic-bite mutation, so
// the action blocks silently — but the wrapper must still return (true, nil).
func TestToxicBite_RoutesToAction(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)
	require.NotNil(t, mob)
	require.NotNil(t, room)

	handled, err := ToxicBite("", mob, room)
	assert.True(t, handled, "wrapper must return handled=true")
	assert.NoError(t, err)
}

// TestToxicBite_CommandRegistered verifies that "toxic_bite" is
// present in the mob command registry.
func TestToxicBite_CommandRegistered(t *testing.T) {
	cmds := GetAllMobCommands()
	found := false
	for _, c := range cmds {
		if c == "toxic_bite" {
			found = true
			break
		}
	}
	assert.True(t, found, "toxic_bite must be registered in the mob command map")
}
