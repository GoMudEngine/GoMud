package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
)

// TestGet_CorpseLoot locks the corpse-as-container `get` syntaxes reported
// broken after the 2026-07-07 corpse-loot redesign. A mob corpse holding loot
// must be lootable with the same natural phrasing players use for bags and
// room containers:
//
//   - `get <item> <corpse>`  (trailing container word, no explicit "from")
//   - `get all <corpse>`     (sweep every item + gold)
//
// Before the fix, corpse loot was reachable ONLY via `get <item> from <corpse>`;
// the two forms below fell through to the floor lookup (item form printed
// "You don't see a X around.") or to the silent floor-sweep (`all` form did
// nothing). The explicit-"from" control subtest proves the loot container is
// wired up and the corpse is genuinely lootable.
func TestGet_CorpseLoot(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)

	// seedCorpse replaces room.Corpses with a single fresh, unowned (freely
	// lootable) mob corpse carrying the given gold + item ids.
	seedCorpse := func(gold int, itemIds ...int) {
		loot := rooms.Container{Gold: gold}
		for _, id := range itemIds {
			loot.Items = append(loot.Items, items.New(id))
		}
		room.Corpses = []rooms.Corpse{{
			MobId:     1,
			Character: characters.Character{Name: "Skeleton"},
			Loot:      loot,
		}}
	}

	// Control: the pre-existing explicit-"from" syntax must keep working.
	t.Run("item_from_corpse_explicit_from", func(t *testing.T) {
		seedCorpse(0, 10001)
		user.Character.Items = nil

		handled, err := Get("iron sword from skeleton corpse", user, room, 0)
		assert.True(t, handled)
		assert.NoError(t, err)
		assert.Len(t, user.Character.Items, 1, "explicit 'from' loot should reach the backpack")
		assert.Len(t, room.Corpses[0].Loot.Items, 0, "looted item should leave the corpse")
	})

	// Bug #2: `get <item> <corpse>` with no explicit "from".
	t.Run("item_from_corpse_no_from", func(t *testing.T) {
		seedCorpse(0, 10001)
		user.Character.Items = nil

		handled, err := Get("sword corpse", user, room, 0)
		assert.True(t, handled)
		assert.NoError(t, err)
		assert.Len(t, user.Character.Items, 1, "`get sword corpse` should loot the corpse")
		assert.Len(t, room.Corpses[0].Loot.Items, 0, "looted item should leave the corpse")
	})

	// Bug #1: `get all <corpse>` sweeps every item and the gold.
	t.Run("all_corpse_sweeps_items_and_gold", func(t *testing.T) {
		seedCorpse(30, 10001, 20001)
		user.Character.Items = nil
		user.Character.Gold = 0

		handled, err := Get("all corpse", user, room, 0)
		assert.True(t, handled)
		assert.NoError(t, err)
		assert.Len(t, user.Character.Items, 2, "`get all corpse` should sweep every item")
		assert.Equal(t, 30, user.Character.Gold, "`get all corpse` should take the gold too")
		assert.Len(t, room.Corpses[0].Loot.Items, 0, "corpse loot items should be emptied")
		assert.Equal(t, 0, room.Corpses[0].Loot.Gold, "corpse loot gold should be emptied")
	})
}
