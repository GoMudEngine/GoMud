package rooms

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
)

// TestRoomHasEssentialMob_ForagerPinsRoom verifies that a room containing a
// forager mob is not queued for unload in RoomMaintenance and is not destroyed
// by removeRoomFromMemory, even when it has been idle well past the unload
// threshold.
func TestRoomHasEssentialMob_ForagerPinsRoom(t *testing.T) {
	// ── mob setup ────────────────────────────────────────────────────────────
	cleanupMobs := mobs.SeedMobsForTest(map[int]*mobs.Mob{}, map[int]*mobs.Mob{})
	defer cleanupMobs()

	essentialMob := &mobs.Mob{
		Groups: []string{"forager"},
	}
	const essentialInstId = 1001
	mobs.SetInstanceForTest(essentialInstId, essentialMob)

	// ── room setup ────────────────────────────────────────────────────────────
	cleanupRooms := SeedRoomsForTest(
		map[int]*Room{
			9001: {
				RoomId:      9001,
				Zone:        "TestZone",
				Title:       "Fernway Clearing",
				Description: "A forest clearing.",
				mobs:        []int{essentialInstId},
				// lastVisited far in the past — would normally trigger unload.
				lastVisited: 0,
			},
		},
		map[string]*ZoneConfig{
			"TestZone": {Name: "TestZone", RoomId: 9001, RoomIds: map[int]struct{}{9001: {}}},
		},
	)
	defer cleanupRooms()

	// ── roomHasEssentialMob helper ────────────────────────────────────────────
	r := roomManager.rooms[9001]
	assert.True(t, roomHasEssentialMob(r), "room with forager mob should be essential")

	// ── removeRoomFromMemory must not unload ──────────────────────────────────
	removeRoomFromMemory(r)
	_, stillLoaded := roomManager.rooms[9001]
	assert.True(t, stillLoaded, "removeRoomFromMemory must not unload a room containing an essential mob")

	// ── RoomMaintenance must not queue it ────────────────────────────────────
	// Set round count high enough that threshold > 0, ensuring the room's
	// lastVisited=0 would normally qualify.
	util.SetRoundCount(10000)
	removed := RoomMaintenance()
	for _, id := range removed {
		assert.NotEqual(t, 9001, id, "RoomMaintenance must not unload a room containing an essential mob")
	}
	_, stillLoadedAfterMaint := roomManager.rooms[9001]
	assert.True(t, stillLoadedAfterMaint, "room with essential mob must survive RoomMaintenance")
}

// TestRoomHasEssentialMob_NonEssentialUnloads verifies that a room whose mobs
// are all non-essential IS eligible for unload when idle past the threshold.
func TestRoomHasEssentialMob_NonEssentialUnloads(t *testing.T) {
	// ── mob setup ────────────────────────────────────────────────────────────
	cleanupMobs := mobs.SeedMobsForTest(map[int]*mobs.Mob{}, map[int]*mobs.Mob{})
	defer cleanupMobs()

	regularMob := &mobs.Mob{
		Groups: []string{"bandit"},
	}
	const regularInstId = 2001
	mobs.SetInstanceForTest(regularInstId, regularMob)

	// ── room setup ────────────────────────────────────────────────────────────
	cleanupRooms := SeedRoomsForTest(
		map[int]*Room{
			9002: {
				RoomId:      9002,
				Zone:        "TestZone",
				Title:       "Dark Alley",
				Description: "A shadowy alley.",
				mobs:        []int{regularInstId},
				lastVisited: 0,
			},
		},
		map[string]*ZoneConfig{
			"TestZone": {Name: "TestZone", RoomId: 9002, RoomIds: map[int]struct{}{9002: {}}},
		},
	)
	defer cleanupRooms()

	r := roomManager.rooms[9002]
	assert.False(t, roomHasEssentialMob(r), "room with bandit mob should NOT be essential")

	// removeRoomFromMemory should unload this room.
	removeRoomFromMemory(r)
	_, stillLoaded := roomManager.rooms[9002]
	assert.False(t, stillLoaded, "removeRoomFromMemory should unload a room containing only non-essential mobs")
}

// TestIsEssential confirms the Mob.IsEssential method returns correct values
// for all relevant group strings.
func TestIsEssential(t *testing.T) {
	cases := []struct {
		groups []string
		want   bool
	}{
		{[]string{"forager"}, true},
		{[]string{"caravan"}, true},
		{[]string{"merchant_train"}, false},
		{[]string{"bandit"}, false},
		{[]string{}, false},
		{[]string{"caravan", "merchant_train"}, true},
		{[]string{"forager", "scout"}, true},
	}

	for _, tc := range cases {
		m := &mobs.Mob{Groups: tc.groups}
		got := m.IsEssential()
		assert.Equal(t, tc.want, got,
			"IsEssential() wrong for groups %v", tc.groups)
	}
}

// TestIsEssential_ShopMob confirms that a mob with a non-empty Shop is
// considered essential regardless of its Groups, so shopkeepers in unvisited
// zones survive the idle-unload cycle.
func TestIsEssential_ShopMob(t *testing.T) {
	shopMob := &mobs.Mob{
		Character: characters.Character{
			Shop: characters.Shop{
				{ItemId: 1, Quantity: 10},
			},
		},
	}
	assert.True(t, shopMob.IsEssential(),
		"a mob with a shop must be essential")

	// A mob with no shop entries should remain non-essential (control case).
	emptyMob := &mobs.Mob{}
	assert.False(t, emptyMob.IsEssential(),
		"a mob with no shop must not be essential by default")
}

// TestRoomHasEssentialMob_ShopMobPinsRoom verifies that a room containing a
// shopkeeper mob is pinned from unload (roomHasEssentialMob returns true) and
// is not destroyed by removeRoomFromMemory.
func TestRoomHasEssentialMob_ShopMobPinsRoom(t *testing.T) {
	cleanupMobs := mobs.SeedMobsForTest(map[int]*mobs.Mob{}, map[int]*mobs.Mob{})
	defer cleanupMobs()

	shopkeeperMob := &mobs.Mob{
		Character: characters.Character{
			Shop: characters.Shop{
				{ItemId: 5, Quantity: 3},
			},
		},
	}
	const shopInstId = 3001
	mobs.SetInstanceForTest(shopInstId, shopkeeperMob)

	cleanupRooms := SeedRoomsForTest(
		map[int]*Room{
			9003: {
				RoomId:      9003,
				Zone:        "TestZone",
				Title:       "Market Stall",
				Description: "A vendor's stall.",
				mobs:        []int{shopInstId},
				// lastVisited far in the past — would normally trigger unload.
				lastVisited: 0,
			},
		},
		map[string]*ZoneConfig{
			"TestZone": {Name: "TestZone", RoomId: 9003, RoomIds: map[int]struct{}{9003: {}}},
		},
	)
	defer cleanupRooms()

	r := roomManager.rooms[9003]
	assert.True(t, roomHasEssentialMob(r),
		"room with a shopkeeper mob should be essential")

	// removeRoomFromMemory must not evict it.
	removeRoomFromMemory(r)
	_, stillLoaded := roomManager.rooms[9003]
	assert.True(t, stillLoaded,
		"removeRoomFromMemory must not unload a room containing a shopkeeper mob")

	// RoomMaintenance must also skip it.
	util.SetRoundCount(10000)
	removed := RoomMaintenance()
	for _, id := range removed {
		assert.NotEqual(t, 9003, id,
			"RoomMaintenance must not unload a room containing a shopkeeper mob")
	}
}

func TestRemoveRoomFromMemory_SavesChangedMob(t *testing.T) {
	cleanupMobs := mobs.SeedMobsForTest(map[int]*mobs.Mob{}, map[int]*mobs.Mob{})
	defer cleanupMobs()

	// Enable mob progression so SaveMobInstance writes.
	configs.AddOverlayOverrides(map[string]any{"Balance.MobProgressionEnabled": true})
	t.Cleanup(func() {
		configs.AddOverlayOverrides(map[string]any{"Balance.MobProgressionEnabled": false})
	})

	const instId = 2002
	mob := &mobs.Mob{
		MobId:      50,
		Zone:       "TestZone",
		HomeRoomId: 9100,
		Groups:     []string{"bandit"}, // non-essential → room unloads
	}
	mob.Character.Name = "Testmob"
	mob.Character.RoomId = 9100 // mob is actually in the room being unloaded
	mob.Character.Stats.Strength.Training = 1 // trips hasPersistableState
	mob.Character.Gold = 999
	mobs.SetInstanceForTest(instId, mob)

	cleanupRooms := SeedRoomsForTest(
		map[int]*Room{
			9100: {
				RoomId:      9100,
				Zone:        "TestZone",
				Title:       "Dark Alley",
				Description: "A shadowy alley.",
				mobs:        []int{instId},
				lastVisited: 0,
			},
		},
		map[string]*ZoneConfig{
			"TestZone": {Name: "TestZone", RoomId: 9100, RoomIds: map[int]struct{}{9100: {}}},
		},
	)
	defer cleanupRooms()

	// Clean any stale save, and ensure cleanup after.
	mobs.DeleteMobInstance(50, "TestZone", "Testmob", 9100)
	t.Cleanup(func() { mobs.DeleteMobInstance(50, "TestZone", "Testmob", 9100) })

	r := roomManager.rooms[9100]
	removeRoomFromMemory(r)

	// Room unloaded AND the mob's current gold was saved on the way out.
	_, stillLoaded := roomManager.rooms[9100]
	assert.False(t, stillLoaded, "non-essential room must unload")

	saved := mobs.LoadMobInstance(50, "TestZone", "Testmob", 9100)
	assert.NotNil(t, saved, "removeRoomFromMemory must save the mob before destroying it")
	assert.NotNil(t, saved.Gold)
	assert.Equal(t, 999, *saved.Gold, "saved gold must reflect the mob's current gold")
}
