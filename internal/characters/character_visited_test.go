package characters

import "testing"

func TestMarkRoomVisited(t *testing.T) {
	c := &Character{}
	c.MarkRoomVisited("Stillwater Marsh", 4101)
	c.MarkRoomVisited("Stillwater Marsh", 4101) // dedup
	c.MarkRoomVisited("Stillwater Marsh", 4102)

	if !c.HasVisitedRoom("Stillwater Marsh", 4101) {
		t.Fatal("4101 should be visited")
	}
	if c.HasVisitedRoom("Stillwater Marsh", 9999) {
		t.Fatal("9999 should not be visited")
	}
	if got := c.GetVisitedRooms("Stillwater Marsh"); len(got) != 2 {
		t.Fatalf("expected 2 visited rooms, got %d (%v)", len(got), got)
	}
}

func TestVisitedRooms_NilSafe(t *testing.T) {
	// Fresh character, never moved: reads must be nil-safe.
	c := &Character{}
	if c.HasVisitedRoom("Nowhere", 1) {
		t.Fatal("unvisited room on a nil map must report false")
	}
	if got := c.GetVisitedRooms("Nowhere"); got != nil {
		t.Fatalf("expected nil slice for an unvisited zone, got %v", got)
	}
}

func TestVisitedRooms_ZoneIsolation(t *testing.T) {
	c := &Character{}
	c.MarkRoomVisited("Zone A", 1)
	c.MarkRoomVisited("Zone B", 2)
	if c.HasVisitedRoom("Zone A", 2) {
		t.Fatal("Zone B's room must not appear in Zone A")
	}
	if !c.HasVisitedRoom("Zone B", 2) {
		t.Fatal("Zone B should contain room 2")
	}
	if len(c.GetVisitedRooms("Zone A")) != 1 || len(c.GetVisitedRooms("Zone B")) != 1 {
		t.Fatalf("each zone should have exactly 1 room")
	}
}
