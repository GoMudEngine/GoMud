package mapper

import "testing"

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
	// Only exits to other *visited* rooms are included; east-x3 -> room 3 (unvisited) is dropped.
	if len(r1.Exits) != 1 || r1.Exits[0].ToRoomId != 2 || r1.Exits[0].Kind != ExitNormal {
		t.Fatalf("room 1 exits wrong: %+v", r1.Exits)
	}
}
