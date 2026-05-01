package rooms

import (
	"testing"

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
