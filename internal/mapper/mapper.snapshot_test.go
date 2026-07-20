package mapper

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// TestSnapshotSkipsEphemeralRooms guards that instance/ephemeral rooms (dungeon
// instances, the Mending Hut copy) that a crawl stepped into never leak their
// billion-range ids into the client Zone.Map snapshot — neither as room nodes
// nor as exit stubs.
func TestSnapshotSkipsEphemeralRooms(t *testing.T) {
	const ephID = 1_000_000_000
	if !rooms.IsEphemeralRoomId(ephID) {
		t.Fatalf("test setup: %d should be an ephemeral room id", ephID)
	}
	nodes := map[int]*mapNode{
		5: node(5, 0, 0, 0, map[string]nodeExit{
			"north": {RoomId: ephID, Direction: d(0, -1, 0)}, // exit toward the instance
		}),
		ephID: node(ephID, 0, -1, 0, map[string]nodeExit{}),
	}
	m := mkMapper(nodes)
	snap := m.Snapshot(map[int]struct{}{5: {}, ephID: {}}) // both "visited"

	var r5 *SnapshotRoom
	for i := range snap {
		if snap[i].RoomId == ephID {
			t.Errorf("ephemeral room %d must not appear in the snapshot", ephID)
		}
		if snap[i].RoomId == 5 {
			r5 = &snap[i]
		}
	}
	if r5 == nil {
		t.Fatal("real room 5 missing from snapshot")
	}
	for _, e := range r5.Exits {
		if e.ToRoomId == ephID {
			t.Errorf("exit toward ephemeral room %d must be filtered from the snapshot", ephID)
		}
	}
}

func TestSnapshotSymbolDefaultsWhenUnset(t *testing.T) {
	nodes := map[int]*mapNode{
		1: node(1, 0, 0, 0, map[string]nodeExit{}),
	}
	// Symbol deliberately left at zero value
	m := mkMapper(nodes)
	snap := m.Snapshot(map[int]struct{}{1: {}})
	if len(snap) != 1 {
		t.Fatalf("expected 1 room in snapshot, got %d", len(snap))
	}
	if snap[0].Symbol == "\x00" {
		t.Error("Symbol must not be NUL (\\x00) when node.Symbol is unset")
	}
	if snap[0].Symbol == "" {
		t.Error("Symbol must not be empty string when node.Symbol is unset")
	}
}

func TestSnapshotClassifiesLongExit(t *testing.T) {
	nodes := map[int]*mapNode{
		1: node(1, 0, 0, 0, map[string]nodeExit{"east-x3": {RoomId: 4, Direction: d(3, 0, 0)}}),
		4: node(4, 3, 0, 0, map[string]nodeExit{}),
	}
	m := mkMapper(nodes)
	snap := m.Snapshot(map[int]struct{}{1: {}, 4: {}})
	var r1 *SnapshotRoom
	for i := range snap {
		if snap[i].RoomId == 1 {
			r1 = &snap[i]
		}
	}
	if r1 == nil || len(r1.Exits) != 1 {
		t.Fatalf("expected room 1 with 1 exit, got %+v", snap)
	}
	if r1.Exits[0].Kind != ExitLong {
		t.Fatalf("expected ExitLong for east-x3, got %q", r1.Exits[0].Kind)
	}
}

func TestSnapshotFiltersVisitedAndClassifies(t *testing.T) {
	nodes := map[int]*mapNode{
		1: node(1, 0, 0, 0, map[string]nodeExit{
			"north":   {RoomId: 2, Direction: d(0, -1, 0)},
			"east-x3": {RoomId: 3, Direction: d(3, 0, 0)},
		}),
		2: node(2, 0, -1, 0, map[string]nodeExit{"south": {RoomId: 1, Direction: d(0, 1, 0)}}),
		3: node(3, 3, 0, 0, map[string]nodeExit{}),
	}
	nodes[1].Symbol = 'T'
	m := mkMapper(nodes)

	visited := map[int]struct{}{1: {}, 2: {}} // room 3 not visited

	snap := m.Snapshot(visited)
	if len(snap) != 2 {
		t.Fatalf("expected 2 visited rooms in snapshot, got %d", len(snap))
	}

	var r1 *SnapshotRoom
	for i := range snap {
		if snap[i].RoomId == 1 {
			r1 = &snap[i]
		}
	}
	if r1 == nil {
		t.Fatal("room 1 missing from snapshot")
	}
	if r1.Symbol != "T" {
		t.Errorf("room 1 symbol: got %q want T", r1.Symbol)
	}
	// north->2 is a full edge; east-x3->3 (unvisited) is now a stub.
	var normalExits, stubExits int
	for _, e := range r1.Exits {
		if e.Stub {
			stubExits++
			if e.Kind != ExitLong {
				t.Fatalf("expected the east-x3 stub to be ExitLong, got %q", e.Kind)
			}
		} else {
			normalExits++
			if e.ToRoomId != 2 || e.Kind != ExitNormal {
				t.Fatalf("expected non-stub exit to room 2 normal, got %+v", e)
			}
		}
	}
	if normalExits != 1 || stubExits != 1 {
		t.Fatalf("expected 1 normal + 1 stub exit, got %d/%d", normalExits, stubExits)
	}
}

func TestSnapshotExitFlagsAndStubs(t *testing.T) {
	nodes := map[int]*mapNode{
		1: node(1, 0, 0, 0, map[string]nodeExit{
			"north": {RoomId: 2, Direction: d(0, -1, 0)},                    // visited -> full edge
			"east":  {RoomId: 3, Direction: d(1, 0, 0), LockDifficulty: 25}, // visited, locked
			"south": {RoomId: 9, Direction: d(0, 1, 0)},                     // UNVISITED/uncrawled -> stub
			"up":    {RoomId: 4, Direction: d(0, 0, 1), Secret: true, OneWay: true, Gate: true},
		}),
		2: node(2, 0, -1, 0, map[string]nodeExit{}),
		3: node(3, 1, 0, 0, map[string]nodeExit{}),
		4: node(4, 0, 0, 1, map[string]nodeExit{}),
	}
	m := mkMapper(nodes)
	visited := map[int]struct{}{1: {}, 2: {}, 3: {}, 4: {}}

	snap := m.Snapshot(visited)
	var r1 *SnapshotRoom
	for i := range snap {
		if snap[i].RoomId == 1 {
			r1 = &snap[i]
		}
	}
	if r1 == nil {
		t.Fatal("room 1 missing")
	}
	byTo := map[int]SnapshotExit{}
	for _, e := range r1.Exits {
		byTo[e.ToRoomId] = e
	}
	if !byTo[3].Locked {
		t.Error("east exit to 3 should be Locked")
	}
	if e := byTo[4]; !e.Secret || !e.OneWay || !e.Gate {
		t.Errorf("up exit flags wrong: %+v", e)
	}
	if !byTo[9].Stub {
		t.Error("south exit to unvisited/uncrawled room 9 should be a Stub")
	}
	if byTo[2].Stub {
		t.Error("north exit to visited room 2 should NOT be a stub")
	}
	if byTo[3].Stub || byTo[4].Stub {
		t.Error("visited destinations (3,4) should NOT be stubs")
	}
}
