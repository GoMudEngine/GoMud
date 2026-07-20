package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegression_MobPutGoldConservation locks the fix for the 2026-07-20 audit
// finding: mobcommands/put.go credited gold to the container without ever
// debiting the mob, so any mob executing `put <n> gold <container>` minted
// currency out of nothing. The player path (usercommands/put.go) checks
// affordability and debits; the mob path did neither.
func TestRegression_MobPutGoldConservation(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob, room := getTestMobAndRoom(t)

	seedChest := func() {
		room.Containers = map[string]rooms.Container{
			"chest": {},
		}
	}

	t.Run("gold_is_debited_from_mob", func(t *testing.T) {
		seedChest()
		mob.Character.Gold = 100

		handled, err := Put("40 gold chest", mob, room)
		assert.True(t, handled)
		assert.NoError(t, err)

		assert.Equal(t, 60, mob.Character.Gold, "mob must be debited the deposited gold")
		assert.Equal(t, 40, room.Containers["chest"].Gold, "container must receive the gold")
	})

	t.Run("total_gold_is_conserved", func(t *testing.T) {
		seedChest()
		mob.Character.Gold = 100
		before := mob.Character.Gold + room.Containers["chest"].Gold

		handled, err := Put("40 gold chest", mob, room)
		assert.True(t, handled)
		assert.NoError(t, err)

		after := mob.Character.Gold + room.Containers["chest"].Gold
		assert.Equal(t, before, after, "gold must not be created or destroyed by put")
	})

	t.Run("mob_cannot_deposit_gold_it_does_not_have", func(t *testing.T) {
		seedChest()
		mob.Character.Gold = 10

		handled, err := Put("500 gold chest", mob, room)
		assert.True(t, handled)
		assert.NoError(t, err)

		assert.GreaterOrEqual(t, mob.Character.Gold, 0, "mob gold must never go negative")
		assert.LessOrEqual(t, room.Containers["chest"].Gold, 10,
			"container must not receive more gold than the mob had")
		assert.Equal(t, 10, mob.Character.Gold+room.Containers["chest"].Gold,
			"gold must be conserved even on an over-deposit attempt")
	})
}

// TestRegression_MobPutLockedContainerGate locks the second half of the same
// audit finding: the player path refuses to put into a locked container
// (usercommands/put.go:60), the mob path had no lock check at all.
func TestRegression_MobPutLockedContainerGate(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	mob, room := getTestMobAndRoom(t)

	seedLockedChest := func() {
		room.Containers = map[string]rooms.Container{
			"strongbox": {Lock: gamelock.Lock{Difficulty: 5}},
		}
	}

	t.Run("gold_rejected", func(t *testing.T) {
		seedLockedChest()
		mob.Character.Gold = 100

		handled, err := Put("40 gold strongbox", mob, room)
		assert.True(t, handled)
		assert.NoError(t, err)

		assert.Equal(t, 100, mob.Character.Gold, "mob keeps its gold when the container is locked")
		assert.Equal(t, 0, room.Containers["strongbox"].Gold, "locked container must not accept gold")
	})

	t.Run("item_rejected", func(t *testing.T) {
		seedLockedChest()
		mob.Character.Items = []items.Item{items.New(1)} // Short Sword
		require.Len(t, mob.Character.Items, 1)

		handled, err := Put("short sword strongbox", mob, room)
		assert.True(t, handled)
		assert.NoError(t, err)

		assert.Len(t, room.Containers["strongbox"].Items, 0, "locked container must not accept items")
		assert.Len(t, mob.Character.Items, 1, "mob keeps the item when the container is locked")
	})

	// Control: proves item_rejected above is not a vacuous pass. If the mob
	// item path did not resolve at all, this would fail and expose that the
	// locked-item assertion was passing for the wrong reason.
	t.Run("unlocked_container_accepts_item", func(t *testing.T) {
		room.Containers = map[string]rooms.Container{
			"openbox": {}, // Difficulty 0 => never locked
		}
		mob.Character.Items = []items.Item{items.New(1)}

		handled, err := Put("short sword openbox", mob, room)
		assert.True(t, handled)
		assert.NoError(t, err)

		assert.Len(t, room.Containers["openbox"].Items, 1, "an unlocked container must accept the item")
		assert.Len(t, mob.Character.Items, 0, "the item must leave the mob's backpack")
	})
}
