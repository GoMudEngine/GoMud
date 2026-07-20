package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/gamelock"
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

// TestGet_QuotedMultiWordFloorItem locks the F1 fix: a quoted multi-word item
// name must resolve off the floor. Before the fix, get.go's floor lookup used
// `rest`, which kept the literal quote characters (only `args` was de-quoted),
// so `get "iron sword"` fell through to a noun/failure.
func TestGet_QuotedMultiWordFloorItem(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	user, room := getTestUserAndRoom(t)

	room.AddItem(items.New(10001), false) // Iron Sword
	user.Character.Items = nil

	handled, err := Get(`"iron sword"`, user, room, 0)
	assert.True(t, handled)
	assert.NoError(t, err)
	assert.Len(t, user.Character.Items, 1, `get "iron sword" (quoted) should pick up the item`)
	_, stillOnFloor := room.FindOnFloor("iron sword", false)
	assert.False(t, stillOnFloor, "item should no longer be on the floor")
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

// TestRegression_GetLockedContainerGate locks the fix for the 2026-07-20 audit
// finding: get.go had no lock check at all, so a player could not `look` inside
// a locked container but could `get` its contents — bypassing picklock/unlock
// entirely. Every sibling command gates on Lock.IsLocked() (look.go:174,
// put.go:60, lock.go:43, unlock.go:38, picklock.go:75); get.go did not.
func TestRegression_GetLockedContainerGate(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	user, room := getTestUserAndRoom(t)

	// Difficulty > 0 with UnlockedRound == 0 => IsLocked() is true.
	seedLockedChest := func() {
		room.Containers = map[string]rooms.Container{
			"strongbox": {
				Lock:  gamelock.Lock{Difficulty: 5},
				Gold:  500,
				Items: []items.Item{items.New(10001)}, // Iron Sword
			},
		}
	}

	t.Run("item_not_lootable", func(t *testing.T) {
		seedLockedChest()
		user.Character.Items = nil

		handled, err := Get("sword strongbox", user, room, 0)
		assert.True(t, handled)
		assert.NoError(t, err)
		assert.Len(t, user.Character.Items, 0, "a locked container must not surrender items")
		assert.Len(t, room.Containers["strongbox"].Items, 1, "the item must remain in the locked container")
	})

	t.Run("gold_not_lootable", func(t *testing.T) {
		seedLockedChest()
		user.Character.Gold = 0

		handled, err := Get("gold strongbox", user, room, 0)
		assert.True(t, handled)
		assert.NoError(t, err)
		assert.Equal(t, 0, user.Character.Gold, "a locked container must not surrender gold")
		assert.Equal(t, 500, room.Containers["strongbox"].Gold, "the gold must remain in the locked container")
	})

	// Guard the other direction: the gate must key off lock state, not simply
	// refuse all container gets.
	//
	// NOTE: this uses an unlocked container (Difficulty 0) rather than calling
	// SetUnlocked() on a locked one. IsLocked() decides relock via
	// gametime.AddPeriod(), and gametime config is not seeded in unit tests —
	// AddPeriod("1 hour") returns the current round unchanged, so a lock
	// relocks instantly and SetUnlocked() can never be observed as unlocked
	// here. Exercising the pick/unlock -> loot transition needs gametime
	// seeding and belongs in an integration test.
	t.Run("unlocked_container_still_lootable", func(t *testing.T) {
		room.Containers = map[string]rooms.Container{
			"strongbox": {
				Lock:  gamelock.Lock{}, // Difficulty 0 => never locked
				Items: []items.Item{items.New(10001)},
			},
		}
		user.Character.Items = nil

		require.False(t, room.Containers["strongbox"].Lock.IsLocked(),
			"precondition: this container must read as unlocked")

		handled, err := Get("sword strongbox", user, room, 0)
		assert.True(t, handled)
		assert.NoError(t, err)
		assert.Len(t, user.Character.Items, 1, "an unlocked container must still be lootable")
	})
}
