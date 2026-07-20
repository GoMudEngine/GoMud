package rooms

import (
	"sort"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// visitedFrom runs ForEachAdjacentRoom over r and returns the visited room ids
// (sorted) alongside how many times each was visited.
func visitedFrom(r *Room) ([]int, map[int]int) {
	counts := map[int]int{}
	r.ForEachAdjacentRoom(func(otherRoom *Room, sourceExit string) {
		counts[otherRoom.RoomId]++
	})
	ids := make([]int, 0, len(counts))
	for id := range counts {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids, counts
}

// TestForEachAdjacentRoom covers the traversal that backs shout propagation.
// It exists because mobcommands/shout.go walked only room.Exits while
// usercommands/shout.go walked Exits + ExitsTemp + ActiveMutators — so mob
// shouts silently failed to reach neighbours connected by a temporary exit.
// (2026-07-20 audit, Tier 0 finding 0.3.)
func TestForEachAdjacentRoom(t *testing.T) {
	// origin(1) -- north --> 2 (standard exit)
	//           -- rift  --> 3 (temporary exit)
	// Both neighbours have a way back to 1.
	newWorld := func() (*Room, func()) {
		origin := &Room{RoomId: 1, Zone: "TestZone", Title: "Origin"}
		north := &Room{RoomId: 2, Zone: "TestZone", Title: "North"}
		rift := &Room{RoomId: 3, Zone: "TestZone", Title: "Rift"}

		origin.Exits = map[string]exit.RoomExit{
			"north": {RoomId: 2},
		}
		north.Exits = map[string]exit.RoomExit{
			"south": {RoomId: 1},
		}
		rift.Exits = map[string]exit.RoomExit{
			"back": {RoomId: 1},
		}

		cleanup := SeedRoomsForTest(
			map[int]*Room{1: origin, 2: north, 3: rift},
			map[string]*ZoneConfig{"TestZone": {}},
		)
		return origin, cleanup
	}

	t.Run("walks_standard_exits", func(t *testing.T) {
		origin, cleanup := newWorld()
		defer cleanup()

		ids, _ := visitedFrom(origin)
		assert.Equal(t, []int{2}, ids, "the standard north exit must be visited")
	})

	// The actual regression: a neighbour reachable only via a temporary exit.
	t.Run("walks_temporary_exits", func(t *testing.T) {
		origin, cleanup := newWorld()
		defer cleanup()

		origin.ExitsTemp = map[string]exit.TemporaryRoomExit{
			"rift": {RoomId: 3, Title: "rift"},
		}

		ids, _ := visitedFrom(origin)
		assert.Equal(t, []int{2, 3}, ids,
			"a room reachable only through a temporary exit must still be visited")
	})

	t.Run("reports_the_exit_leading_back", func(t *testing.T) {
		origin, cleanup := newWorld()
		defer cleanup()

		got := map[int]string{}
		origin.ForEachAdjacentRoom(func(otherRoom *Room, sourceExit string) {
			got[otherRoom.RoomId] = sourceExit
		})

		require.Contains(t, got, 2)
		assert.Equal(t, "south", got[2],
			"sourceExit must name the exit leading back to the origin, not the outbound one")
	})

	// A room wired through two sources must not be shouted at twice.
	t.Run("visits_each_neighbour_once", func(t *testing.T) {
		origin, cleanup := newWorld()
		defer cleanup()

		origin.ExitsTemp = map[string]exit.TemporaryRoomExit{
			"shortcut": {RoomId: 2, Title: "shortcut"}, // same room as the north exit
		}

		_, counts := visitedFrom(origin)
		assert.Equal(t, 1, counts[2], "a neighbour reachable twice must be visited once")
	})

	// Without a return path the far room has no direction to attribute the
	// sound to, so it is skipped.
	t.Run("skips_neighbours_with_no_way_back", func(t *testing.T) {
		origin, cleanup := newWorld()
		defer cleanup()

		oneWay := &Room{RoomId: 4, Zone: "TestZone", Title: "One Way"}
		cleanup2 := SeedRoomsForTest(
			map[int]*Room{1: origin, 2: LoadRoom(2), 4: oneWay},
			map[string]*ZoneConfig{"TestZone": {}},
		)
		defer cleanup2()

		origin.Exits["chute"] = exit.RoomExit{RoomId: 4}

		ids, _ := visitedFrom(origin)
		assert.NotContains(t, ids, 4, "a one-way exit with no return path must be skipped")
	})

	t.Run("skips_self_reference", func(t *testing.T) {
		origin, cleanup := newWorld()
		defer cleanup()

		origin.Exits["loop"] = exit.RoomExit{RoomId: 1}

		ids, _ := visitedFrom(origin)
		assert.NotContains(t, ids, 1, "a room must not visit itself")
	})
}
