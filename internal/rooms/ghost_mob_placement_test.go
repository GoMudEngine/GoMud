package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ─── placementRoomFor ───────────────────────────────────────────────────────
//
// Regression coverage for the "ghost guard" bug: a SpawnInfo mob with a
// schedule_id has its Character.RoomId moved to a different room by
// applyScheduleSpawnOverride inside NewMobById. Prepare() must list the
// instance in the OVERRIDE room, not the spawn room.

func TestPlacementRoomFor_NoOverride_ListsSpawnRoom(t *testing.T) {
	// mobRoomId == spawnRoomId: ordinary mob, no schedule override.
	allExist := func(int) bool { return true }
	d := placementRoomFor(465, 465, allExist)
	assert.Equal(t, 465, d.listRoomId)
	assert.False(t, d.resetRoomIdToSpawn)
}

func TestPlacementRoomFor_ZeroRoomId_ListsSpawnRoom(t *testing.T) {
	// Defensive: a 0 RoomId means "unset"; treat as no override.
	allExist := func(int) bool { return true }
	d := placementRoomFor(465, 0, allExist)
	assert.Equal(t, 465, d.listRoomId)
	assert.False(t, d.resetRoomIdToSpawn)
}

func TestPlacementRoomFor_OverrideToLoadableRoom_ListsOverrideRoom(t *testing.T) {
	// The concrete ghost case: city guard spawns from room 465 but its
	// dayshift patrol's first waypoint is room 460. Override moved RoomId to
	// 460. The instance must be listed in 460, NOT 465.
	allExist := func(int) bool { return true }
	d := placementRoomFor(465, 460, allExist)
	assert.Equal(t, 460, d.listRoomId, "must list in override room, not spawn room")
	assert.False(t, d.resetRoomIdToSpawn)
}

func TestPlacementRoomFor_OverrideToUnloadableRoom_FallsBackToSpawn(t *testing.T) {
	// If the override room cannot be loaded, fall back to the spawn room and
	// reset the mob's RoomId so it isn't stranded in a non-existent room.
	noneExist := func(int) bool { return false }
	d := placementRoomFor(465, 460, noneExist)
	assert.Equal(t, 465, d.listRoomId)
	assert.True(t, d.resetRoomIdToSpawn, "must signal RoomId reset on fallback")
}

// ─── listMobInRoom ──────────────────────────────────────────────────────────

func TestListMobInRoom_AppendsAndCounts(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[2]
	listMobInRoom(r, 9999)

	assert.Contains(t, r.mobs, 9999)
	assert.Equal(t, len(r.mobs), roomManager.roomsWithMobs[r.RoomId])
}

func TestListMobInRoom_NoDuplicate(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	r := roomManager.rooms[2]
	listMobInRoom(r, 9999)
	listMobInRoom(r, 9999)

	count := 0
	for _, id := range r.mobs {
		if id == 9999 {
			count++
		}
	}
	assert.Equal(t, 1, count, "listMobInRoom must not double-list")
}

func TestListMobInRoom_NilTargetNoPanic(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()
	// Must not panic.
	listMobInRoom(nil, 1)
}
