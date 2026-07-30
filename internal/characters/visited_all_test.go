package characters

import (
	"sort"
	"testing"
)

// TestGetAllVisitedRoomsSpansZones is the character-side half of the
// cross-boundary map fix. The map is drawn per zone, but a zone boundary is an
// engine concept a player cannot perceive — walking over one used to blank the
// map of everything behind you, including the room you had just left.
func TestGetAllVisitedRoomsSpansZones(t *testing.T) {
	c := New()

	c.MarkRoomVisited("Watchers Crossing", 421)
	c.MarkRoomVisited("Watchers Crossing", 420)
	c.MarkRoomVisited("Dustwalk Road", 409)

	// Per-zone accessor stays scoped — other callers depend on that.
	if got := len(c.GetVisitedRooms("Dustwalk Road")); got != 1 {
		t.Errorf("GetVisitedRooms(Dustwalk Road) = %d rooms, want 1", got)
	}

	all := c.GetAllVisitedRooms()
	sort.Ints(all)
	want := []int{409, 420, 421}
	if len(all) != len(want) {
		t.Fatalf("GetAllVisitedRooms() = %v, want %v", all, want)
	}
	for i := range want {
		if all[i] != want[i] {
			t.Fatalf("GetAllVisitedRooms() = %v, want %v", all, want)
		}
	}

	// The point of the fix: standing in Dustwalk Road, room 420 (the Watchers
	// Crossing room you just stepped out of) is still in the set the snapshot
	// gets to consider. mapper.HasRoom then decides whether it is close enough
	// to draw.
	found := false
	for _, id := range all {
		if id == 420 {
			found = true
		}
	}
	if !found {
		t.Error("the room the player just came from is missing — the map would blank on crossing")
	}
}

func TestGetAllVisitedRoomsEmpty(t *testing.T) {
	c := New()
	if got := c.GetAllVisitedRooms(); len(got) != 0 {
		t.Errorf("fresh character = %v, want empty", got)
	}
	// No duplicates when the same room is marked twice.
	c.MarkRoomVisited("Z", 1)
	c.MarkRoomVisited("Z", 1)
	if got := c.GetAllVisitedRooms(); len(got) != 1 {
		t.Errorf("re-marking the same room = %v, want one entry", got)
	}
}
