package mobcommands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealingGel_RoutesToAction verifies that HealingGel delegates to
// actions.TriggerHealingGel without panicking. The mob from the seeded
// registry (instanceId 100) does not have the healing-gel mutation, so
// the action blocks silently — but the wrapper must still return (true, nil).
func TestHealingGel_RoutesToAction(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)
	require.NotNil(t, mob)
	require.NotNil(t, room)

	handled, err := HealingGel("", mob, room)
	assert.True(t, handled, "wrapper must return handled=true")
	assert.NoError(t, err)
}

// TestHealingGel_CommandRegistered verifies that "healing_gel" is
// present in the mob command registry.
func TestHealingGel_CommandRegistered(t *testing.T) {
	cmds := GetAllMobCommands()
	found := false
	for _, c := range cmds {
		if c == "healing_gel" {
			found = true
			break
		}
	}
	assert.True(t, found, "healing_gel must be registered in the mob command map")
}
