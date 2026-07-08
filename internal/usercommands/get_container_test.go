package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGet_SingleWordContainer_Regression locks the currently-working
// single-word-container get paths. These must stay green through the Stage 1
// refactor (behavior preservation for the case that already works).
func TestGet_SingleWordContainer_Regression(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	user, room := getTestUserAndRoom(t)

	seedChest := func() {
		room.Containers = map[string]rooms.Container{
			"chest": {Items: []items.Item{items.New(10001)}}, // Iron Sword
		}
	}

	t.Run("no_from", func(t *testing.T) {
		seedChest()
		user.Character.Items = nil
		handled, err := Get("sword chest", user, room, 0)
		assert.True(t, handled)
		assert.NoError(t, err)
		assert.Len(t, user.Character.Items, 1, "`get sword chest` should take from the chest")
	})

	t.Run("explicit_from", func(t *testing.T) {
		seedChest()
		user.Character.Items = nil
		handled, err := Get("iron sword from chest", user, room, 0)
		assert.True(t, handled)
		assert.NoError(t, err)
		assert.Len(t, user.Character.Items, 1)
	})
}

// TestGet_MultiWordContainer_Composition is the Stage 1 fix target: multi-word
// container names (and multi-word "from" splits) were broken before the parser
// migration because get.go stripped only the last word. Red before Task 3,
// green after.
func TestGet_MultiWordContainer_Composition(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	user, room := getTestUserAndRoom(t)

	seedChest := func() {
		room.Containers = map[string]rooms.Container{
			"wooden chest": {Items: []items.Item{items.New(10001)}}, // Iron Sword
		}
	}

	t.Run("no_from", func(t *testing.T) {
		seedChest()
		user.Character.Items = nil
		handled, err := Get("sword wooden chest", user, room, 0)
		assert.True(t, handled)
		assert.NoError(t, err)
		assert.Len(t, user.Character.Items, 1, "`get sword wooden chest` should take from the chest")
	})

	t.Run("explicit_from", func(t *testing.T) {
		seedChest()
		user.Character.Items = nil
		handled, err := Get("iron sword from wooden chest", user, room, 0)
		assert.True(t, handled)
		assert.NoError(t, err)
		assert.Len(t, user.Character.Items, 1)
	})
}

// TestGet_HiddenContainerGate_SingleWord locks that an undiscovered hidden
// container is NOT lootable — the discovery gate must survive the refactor.
// Single-word name so the container genuinely resolves (isolating the gate, not
// a resolution miss).
func TestGet_HiddenContainerGate_SingleWord(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	user, room := getTestUserAndRoom(t)

	room.Containers = map[string]rooms.Container{
		"cache": {Hidden: true, Items: []items.Item{items.New(10001)}},
	}
	user.Character.Items = nil

	handled, err := Get("sword cache", user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)
	require.Len(t, user.Character.Items, 0, "an undiscovered hidden container must not be lootable")
}
