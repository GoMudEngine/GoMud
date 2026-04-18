package usercommands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGive_QuestEngineInterceptsBeforeBtreePlayerGive locks the architecture
// established by f7a647b3 (feat: quest engine intercepts item_give before
// transfer in give.go) and reinforced by c3c48a7c (fix: smoke test bugfixes
// for Phase 4b/4c, which removed duplicate grant_quest from the affected
// behavior trees because the quest engine handles advancement).
//
// Two scoped subtests exercise both branches:
//
//   - "consume_branch_blocks_btree": quest engine returns
//     {Handled:true, ConsumeItem:true} → item is removed from the player,
//     NOT transferred to the mob, and the mob's player_give btree handler
//     is NOT invoked (Approach A: positive observable absence).
//   - "no_quest_btree_fires": empty quest engine → item transfers to the
//     mob and the mob's player_give btree handler DOES fire. This is the
//     control case proving the btree handler in subtest 1 is observable.
//
// Future refactors that re-order give.go's consume-then-btree flow, or
// re-introduce double-handling, will fail this test.
func TestGive_QuestEngineInterceptsBeforeBtreePlayerGive(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)

	// Locate the seeded mob (Skeleton, MobId 1, InstanceId 100 from
	// seedAllRegistries). FindByName lower-cases its arg; "skeleton"
	// matches the seeded character name.
	_, mobInstanceId := room.FindByName("skeleton")
	require.NotZero(t, mobInstanceId, "test mob 'skeleton' must be in the room")
	mob := mobs.GetInstance(mobInstanceId)
	require.NotNil(t, mob, "mob instance must exist")

	const questItemId = 10001
	mobTemplateId := int(mob.MobId)

	// Approach A: seed a behavior tree on the mob template with a
	// player_give handler that mutates observable mob state. Both subtests
	// share this tree; subtest 1 expects btree_fired stays 0, subtest 2
	// expects btree_fired = 1.
	//
	// Tree shape: an event-filtered action that sets state when (and only
	// when) the player_give event reaches the tree.
	btreeYAML := `tree:
  type: action
  event: player_give
  do: set_state
  key: btree_fired
  value: 1
`
	tmpDir := t.TempDir()
	btreePath := filepath.Join(tmpDir, "1-skeleton.yaml")
	require.NoError(t, os.WriteFile(btreePath, []byte(btreeYAML), 0644),
		"failed to write temp btree YAML")
	require.NoError(t,
		behaviortree.GetEngine().LoadTree(mobTemplateId, btreePath),
		"failed to load btree for test mob")
	// NOTE: behaviortree.Engine has no exported tree-eviction API. The cached
	// tree on mobTemplateId 1 will leak into other tests if another test
	// relies on mob 1 having NO tree. Today no other test in this package
	// does (verified across usercommands_test.go on 2026-04-17). If that
	// changes, add an exported behaviortree.RemoveTreeForTest helper.

	t.Run("consume_branch_blocks_btree", func(t *testing.T) {
		// Reset the global quest engine so the test's seeded quest is the
		// only trigger registered for "item_give". Restored on cleanup.
		cleanupEngine := questengine.ResetEngineForTest()
		defer cleanupEngine()

		// Seed the player's backpack with the quest item.
		require.True(t, user.Character.StoreItem(items.New(questItemId)),
			"failed to seed quest item in player's backpack")
		defer cleanupTestItems(user.Character.Items, mob.Character.Items)
		require.Equal(t, 1, countItemsById(user.Character.Items, questItemId),
			"pre-condition: player should have exactly 1 quest item in backpack")
		require.Equal(t, 0, countItemsById(mob.Character.Items, questItemId),
			"pre-condition: mob should not yet have the quest item")

		// Register a quest with a single trigger: item_give for our
		// mob+item, with a ConsumeItem action. Drives give.go's early
		// return.
		quest := &questengine.QuestDef{
			QuestId:     999001,
			Name:        "Give Test Quest",
			Description: "Locks give.go quest-engine vs btree handoff.",
			Steps:       []questengine.QuestStep{{Id: "start"}},
			Triggers: []questengine.TriggerDef{
				{
					Event: "item_give",
					Mob:   mobTemplateId,
					Item:  questItemId,
					Actions: []questengine.ActionDef{
						{ConsumeItem: questItemId},
					},
				},
			},
		}
		questengine.GetEngine().RegisterQuest(quest)

		// Initialize per-instance btree state so we can read GetInt later
		// regardless of whether the handler ran. (GetInt on missing key
		// returns 0, so this is belt-and-suspenders.)
		state := behaviortree.EnsureBTreeState(mob)
		state.Set("btree_fired", 0)

		// Act: give the item. The string is "<item> <target>"; give.go
		// takes the LAST whitespace-separated token as the receiver name
		// and joins the rest as the item name.
		handled, err := Give("iron sword skeleton", user, room, 0)
		assert.True(t, handled, "Give should return handled=true")
		assert.NoError(t, err)

		// Assertion 1: item removed from the player's backpack.
		assert.Equal(t, 0, countItemsById(user.Character.Items, questItemId),
			"after Give: quest item should be removed from player's backpack "+
				"(quest engine consumed it)")

		// Assertion 2: item was NOT transferred to the mob — the consume
		// branch in give.go returns BEFORE actions.GiveItemToChar.
		assert.Equal(t, 0, countItemsById(mob.Character.Items, questItemId),
			"after Give: mob must NOT have the quest item (consume branch ran)")

		// Assertion 3 (Approach A): the player_give btree handler did NOT
		// fire. If give.go's early return were removed (regression), the
		// handler would set btree_fired=1.
		assert.Equal(t, 0, state.GetInt("btree_fired"),
			"after Give: player_give btree handler must NOT have fired "+
				"(quest engine intercept short-circuits before TryMobBehavior)")
	})

	t.Run("no_quest_btree_fires", func(t *testing.T) {
		// Control case: with no quest registered for item_give, give.go
		// must transfer the item to the mob AND fire the player_give btree
		// handler. This proves Approach A's observable side-effect is real.
		cleanupEngine := questengine.ResetEngineForTest()
		defer cleanupEngine()

		// Re-seed the player's backpack (subtest 1's cleanup already
		// emptied it via the consume branch).
		require.True(t, user.Character.StoreItem(items.New(questItemId)),
			"failed to seed quest item in player's backpack")
		defer cleanupTestItems(user.Character.Items, mob.Character.Items)
		require.Equal(t, 1, countItemsById(user.Character.Items, questItemId),
			"pre-condition: player should have 1 quest item in backpack")

		// Reset state so we can observe the handler firing.
		state := behaviortree.EnsureBTreeState(mob)
		state.Set("btree_fired", 0)

		handled, err := Give("iron sword skeleton", user, room, 0)
		assert.True(t, handled, "Give should return handled=true")
		assert.NoError(t, err)

		// Item left the player's backpack.
		assert.Equal(t, 0, countItemsById(user.Character.Items, questItemId),
			"after Give: item should leave player's backpack")

		// Item transferred to the mob (no quest engine consume).
		assert.Equal(t, 1, countItemsById(mob.Character.Items, questItemId),
			"after Give: mob should have received the item (no quest consume)")

		// Btree handler DID fire — proves the assertion in subtest 1 is
		// observable, not a false negative from a broken fixture.
		assert.Equal(t, 1, state.GetInt("btree_fired"),
			"after Give: player_give btree handler MUST have fired "+
				"(no quest engine intercept; control case for subtest 1)")
	})
}

// countItemsById returns the number of items in the slice whose ItemId
// matches the given id. Used to verify presence/absence of a known item
// without coupling to slice ordering.
func countItemsById(itemSlice []items.Item, id int) int {
	n := 0
	for _, it := range itemSlice {
		if it.ItemId == id {
			n++
		}
	}
	return n
}

// cleanupTestItems is a no-op marker for explicit-cleanup intent in
// give_test subtests. The seedAllRegistries cleanup func re-creates the
// user / mob registries between top-level tests, but inside one parent
// test we share the same user/mob across subtests; this function exists
// as documentation that subtest 1 leaves the player's backpack empty
// (consume) and subtest 2 leaves the mob holding the item — both are
// scoped to this parent test and torn down by the outer defer cleanup().
func cleanupTestItems(_, _ []items.Item) {}

