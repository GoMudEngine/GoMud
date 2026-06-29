package usercommands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTutorial_LowProgressTeleports(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	user, room := getTestUserAndRoom(t)
	user.Character.QuestProgress = map[int]string{} // no progress

	handled, err := Tutorial("", user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)
}

func TestTutorial_HighProgressRefuses(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	user, room := getTestUserAndRoom(t)
	user.Character.QuestProgress = map[int]string{65: "end", 35: "end"} // real progress

	handled, err := Tutorial("", user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)
}
