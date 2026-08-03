package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// return_item used to mint a fresh copy of the event's item via items.New()
// and leave the original sitting in the mob's inventory. On a combat-capable
// mob (e.g. the guard_captain archetype) that original later dropped as corpse
// loot — give item, receive a pristine copy, kill the mob, loot the original.
// It also reset enchant tier / remaining uses to template defaults.
//
// These tests drive actReturnItem directly. That is the smallest real seam:
// the give command's full path (usercommands.Give) is exercised by
// internal/usercommands/give_test.go, and everything actReturnItem needs is
// reachable from a hand-built EvalContext.

const (
	riItemId     = 90501
	riTemplateId = 90500
	riInstanceId = 90599
	riRoomId     = 90590
	riUserId     = 905
)

func seedReturnItemFixture(t *testing.T) func() {
	t.Helper()
	cleanItems := items.SeedItemsForTest(map[int]*items.ItemSpec{
		riItemId: {
			ItemId:  riItemId,
			Name:    "brass token",
			Type:    items.Object,
			Subtype: items.Mundane,
			Uses:    5,
			Weight:  5.0, // non-zero so the carry-capacity path is exercised
		},
	})
	cleanRoom := seedTestRoom(t, riRoomId, "Testzone")
	cleanMob := seedTestMob(t, riTemplateId, riInstanceId, riRoomId, "test guard")
	cleanUser := seedTestUser(t, riUserId, "returnitem", "Returnitem", riRoomId)

	// seedTestMob leaves stats at zero, which means zero carry capacity —
	// the mob could not hold the item the fixture is about to hand it.
	if m := mobs.GetInstance(riInstanceId); m != nil {
		m.Character.Stats.Strength.ValueAdj = 100
	}

	return func() {
		cleanUser()
		cleanMob()
		cleanRoom()
		cleanItems()
	}
}

// countItemsById counts carried items (backpack + component bag + bandolier)
// matching a template id.
func countItemsById(items_ []items.Item, itemId int) int {
	n := 0
	for _, it := range items_ {
		if it.ItemId == itemId {
			n++
		}
	}
	return n
}

func TestReturnItem_ReturnsRealItem_NoDuplication(t *testing.T) {
	cleanup := seedReturnItemFixture(t)
	defer cleanup()

	mob := mobs.GetInstance(riInstanceId)
	if mob == nil {
		t.Fatal("setup: mob instance not seeded")
	}
	user := users.GetByUserId(riUserId)
	if user == nil {
		t.Fatal("setup: user not seeded")
	}

	// Simulate give.go: the real, stateful item is already on the mob by the
	// time the player_give handler runs. Mark it so we can tell the returned
	// object apart from a freshly-minted template copy.
	given := items.New(riItemId)
	given.Uses = 2
	given.EnchantType = "flaming"
	given.EnchantTier = 3
	if !mob.Character.StoreItem(given) {
		t.Fatal("setup: mob could not store the given item")
	}

	ctx := &EvalContext{
		InstanceId: riInstanceId,
		MobId:      riTemplateId,
		RoomId:     riRoomId,
		MobName:    "test guard",
		Event: EventContext{
			EventType: "player_give",
			UserId:    riUserId,
			ItemId:    riItemId,
			ItemUUID:  given.UUID,
			RoomId:    riRoomId,
		},
	}

	if res := actReturnItem(nil, ctx); res != Success {
		t.Fatalf("actReturnItem: want Success, got %v", res)
	}

	// The mob must no longer hold it — otherwise it drops as corpse loot.
	if got := countItemsById(mob.Character.Items, riItemId); got != 0 {
		t.Errorf("mob still holds %d copies of the item after return_item; want 0", got)
	}

	// The player must hold exactly one.
	got := countItemsById(user.Character.Items, riItemId)
	if got != 1 {
		t.Fatalf("player holds %d copies after return_item; want exactly 1", got)
	}

	// And it must be the ORIGINAL object, with its state intact.
	returned := user.Character.Items[0]
	if returned.UUID != given.UUID {
		t.Errorf("returned item UUID = %v, want the original %v (a copy was minted)", returned.UUID, given.UUID)
	}
	if returned.Uses != 2 || returned.EnchantType != "flaming" || returned.EnchantTier != 3 {
		t.Errorf("returned item lost state: uses=%d enchant=%q tier=%d; want 2/flaming/3",
			returned.Uses, returned.EnchantType, returned.EnchantTier)
	}
}

func TestReturnItem_MobDoesNotHoldItem_FailsWithoutMinting(t *testing.T) {
	cleanup := seedReturnItemFixture(t)
	defer cleanup()

	user := users.GetByUserId(riUserId)
	if user == nil {
		t.Fatal("setup: user not seeded")
	}

	ctx := &EvalContext{
		InstanceId: riInstanceId,
		MobId:      riTemplateId,
		RoomId:     riRoomId,
		MobName:    "test guard",
		Event: EventContext{
			EventType: "player_give",
			UserId:    riUserId,
			ItemId:    riItemId,
			RoomId:    riRoomId,
		},
	}

	if res := actReturnItem(nil, ctx); res != Failure {
		t.Fatalf("actReturnItem with an empty-handed mob: want Failure, got %v", res)
	}
	if got := countItemsById(user.Character.Items, riItemId); got != 0 {
		t.Errorf("player received %d fabricated items; want 0", got)
	}
}

func TestReturnItem_UnknownMobInstance_Fails(t *testing.T) {
	cleanup := seedReturnItemFixture(t)
	defer cleanup()

	user := users.GetByUserId(riUserId)
	if user == nil {
		t.Fatal("setup: user not seeded")
	}

	ctx := &EvalContext{
		InstanceId: 999999, // not seeded
		MobId:      riTemplateId,
		RoomId:     riRoomId,
		MobName:    "test guard",
		Event: EventContext{
			EventType: "player_give",
			UserId:    riUserId,
			ItemId:    riItemId,
			RoomId:    riRoomId,
		},
	}

	if res := actReturnItem(nil, ctx); res != Failure {
		t.Fatalf("actReturnItem with an unresolvable mob: want Failure, got %v", res)
	}
	if got := countItemsById(user.Character.Items, riItemId); got != 0 {
		t.Errorf("player received %d fabricated items; want 0", got)
	}
}

// A player too loaded to carry the item must not destroy it: the transfer
// rolls back and the mob keeps holding it.
func TestReturnItem_PlayerOverloaded_ItemStaysWithMob(t *testing.T) {
	cleanup := seedReturnItemFixture(t)
	defer cleanup()

	mob := mobs.GetInstance(riInstanceId)
	user := users.GetByUserId(riUserId)
	if mob == nil || user == nil {
		t.Fatal("setup: fixture not seeded")
	}

	given := items.New(riItemId)
	if !mob.Character.StoreItem(given) {
		t.Fatal("setup: mob could not store the given item")
	}

	// Crush the player: carry capacity is Strength * CarryCapacityMultiplier,
	// and StoreItem refuses past 2x that. Zeroing Strength makes capacity 0.
	user.Character.Stats.Strength.ValueAdj = 0

	ctx := &EvalContext{
		InstanceId: riInstanceId,
		MobId:      riTemplateId,
		RoomId:     riRoomId,
		MobName:    "test guard",
		Event: EventContext{
			EventType: "player_give",
			UserId:    riUserId,
			ItemId:    riItemId,
			ItemUUID:  given.UUID,
			RoomId:    riRoomId,
		},
	}

	res := actReturnItem(nil, ctx)

	playerHas := countItemsById(user.Character.Items, riItemId)
	mobHas := countItemsById(mob.Character.Items, riItemId)

	if playerHas+mobHas != 1 {
		t.Fatalf("item was duplicated or destroyed: player=%d mob=%d (want exactly 1 total)", playerHas, mobHas)
	}
	if playerHas == 0 && res != Failure {
		t.Errorf("transfer rolled back to the mob but action returned %v; want Failure", res)
	}
}
