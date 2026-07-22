package mapper

import "testing"

// TestPlaceByAuthoredCoords_PreservesAuthored proves the authored-placement
// path positions each node at its stored coordinate rather than re-deriving a
// position by crawling exit deltas. The two rooms are wired north/south, but
// their *authored* coordinates are deliberately inconsistent with that exit
// delta (room 2 sits at (5,5,0), not the (0,-1,0) a north crawl would give).
// After placement, node.Pos must equal the authored coords, and GetRoomId must
// resolve each authored cell — behaviour the old exit-delta crawl could never
// produce.
func TestPlaceByAuthoredCoords_PreservesAuthored(t *testing.T) {
	m := NewMapper(1)
	m.crawledRooms = map[int]*mapNode{
		1: {RoomId: 1, Plane: 2, Pos: d(0, 0, 0), Exits: map[string]nodeExit{
			"north": {RoomId: 2, Direction: d(0, -1, 0)},
		}},
		2: {RoomId: 2, Plane: 2, Pos: d(5, 5, 0), Exits: map[string]nodeExit{
			"south": {RoomId: 1, Direction: d(0, 1, 0)},
		}},
	}

	m.placeByAuthoredCoords()

	if got := m.crawledRooms[1].Pos; got != d(0, 0, 0) {
		t.Errorf("room 1 Pos: got %+v want (0,0,0)", got)
	}
	if got := m.crawledRooms[2].Pos; got != d(5, 5, 0) {
		t.Errorf("room 2 Pos: got %+v want (5,5,0) — placement must not re-crawl", got)
	}

	if id, err := m.GetRoomId(0, 0, 0); err != nil || id != 1 {
		t.Errorf("GetRoomId(0,0,0): got id=%d err=%v want id=1", id, err)
	}
	if id, err := m.GetRoomId(5, 5, 0); err != nil || id != 2 {
		t.Errorf("GetRoomId(5,5,0): got id=%d err=%v want id=2", id, err)
	}
}

// TestAuthoredLayoutClean_DetectsCellCollision guards the fallback trigger: two
// rooms authored into the same (plane,x,y,z) cell — the signature of an
// un-migrated straggler defaulting to the origin — must mark the layout unclean
// so Start() falls back to the historical exit-delta crawl.
func TestAuthoredLayoutClean_DetectsCellCollision(t *testing.T) {
	clean := map[int]*mapNode{
		1: {RoomId: 1, Plane: 0, Pos: d(0, 0, 0)},
		2: {RoomId: 2, Plane: 0, Pos: d(1, 0, 0)},
	}
	if !mkMapper(clean).authoredLayoutClean() {
		t.Error("distinct authored cells should be clean")
	}

	// Same plane + same cell = collision.
	collide := map[int]*mapNode{
		1: {RoomId: 1, Plane: 0, Pos: d(0, 0, 0)},
		2: {RoomId: 2, Plane: 0, Pos: d(0, 0, 0)},
	}
	if mkMapper(collide).authoredLayoutClean() {
		t.Error("two rooms in the same authored cell must be flagged unclean")
	}

	// Same cell coords but *different planes* is legal (overworld vs interior).
	crossPlane := map[int]*mapNode{
		1: {RoomId: 1, Plane: 0, Pos: d(0, 0, 0)},
		2: {RoomId: 2, Plane: 1, Pos: d(0, 0, 0)},
	}
	if !mkMapper(crossPlane).authoredLayoutClean() {
		t.Error("identical coords on different planes must remain clean")
	}
}
