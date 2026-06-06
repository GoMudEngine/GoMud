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
