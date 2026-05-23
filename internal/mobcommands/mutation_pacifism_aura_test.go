package mobcommands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPacifismAura_RoutesToAction verifies that PacifismAura delegates to
// actions.TriggerPacifismAura without panicking. The mob from the seeded
// registry (instanceId 100) does not have the pacifism-aura mutation, so the
// action blocks silently — but the wrapper must still return (true, nil).
func TestPacifismAura_RoutesToAction(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	mob, room := getTestMobAndRoom(t)
	require.NotNil(t, mob)
	require.NotNil(t, room)

	handled, err := PacifismAura("", mob, room)
	assert.True(t, handled, "wrapper must return handled=true")
	assert.NoError(t, err)
}

// TestPacifismAuraCommandRegistered verifies that "pacifism-aura" is
// present in the mob command registry.
func TestPacifismAuraCommandRegistered(t *testing.T) {
	cmds := GetAllMobCommands()
	found := false
	for _, c := range cmds {
		if c == "pacifism-aura" {
			found = true
			break
		}
	}
	assert.True(t, found, "pacifism-aura must be registered in the mob command map")
}
