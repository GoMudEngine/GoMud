package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

// TestStand_GrappleNotStood locks the grapple move-collision fix: `stand` from a
// grapple lock must NOT transition the grappler to standing (previously it hit
// the "already standing" bail and no-op'd; now it reports the grapple explicitly
// but still must not break the grapple by standing).
func TestStand_GrappleNotStood(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	user, room := getTestUserAndRoom(t)

	setCombatPositionParallel(user.Character, position.Clinch)
	user.Character.Stamina = 100

	handled, err := Stand("", user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)
	assert.True(t, user.Character.IsGrappling(), "stand must not break a grapple lock")
	assert.False(t, user.Character.IsStanding(), "grappler must not become Standing via stand")

	setCombatPositionParallel(user.Character, position.Standing) // reset
}
