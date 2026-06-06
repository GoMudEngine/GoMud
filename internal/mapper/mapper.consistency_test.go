package mapper

import "testing"

func d(x, y, z int) positionDelta { return positionDelta{x: x, y: y, z: z} }

func TestSamePos(t *testing.T) {
	if !samePos(positionDelta{1, 2, 3, '│'}, positionDelta{1, 2, 3, ' '}) {
		t.Fatal("samePos must ignore the arrow field")
	}
	if samePos(d(1, 0, 0), d(0, 1, 0)) {
		t.Fatal("different coords must not be samePos")
	}
}

func TestClassifyKind(t *testing.T) {
	cases := []struct {
		name            string
		nominal, actual positionDelta
		want            ExitKind
	}{
		{"normal", d(0, -1, 0), d(0, -1, 0), ExitNormal},
		{"long-x3", d(0, -3, 0), d(0, -3, 0), ExitLong},
		{"vertical-up", d(0, 0, 1), d(0, 0, 1), ExitVertical},
		{"wrap", d(0, 1, 0), d(0, -4, 0), ExitWrap},
		{"long-diagonal", d(-3, -3, 0), d(-3, -3, 0), ExitLong},
	}
	for _, c := range cases {
		if got := classifyKind(c.nominal, c.actual); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

func TestFindCollisions(t *testing.T) {
	// One collision group.
	nodes := map[int]*mapNode{
		1: {RoomId: 1, Pos: d(0, 0, 0)},
		2: {RoomId: 2, Pos: d(1, 0, 0)},
		3: {RoomId: 3, Pos: d(1, 0, 0)}, // collides with 2
	}
	groups := findCollisions(nodes)
	if len(groups) != 1 || len(groups[0]) != 2 || groups[0][0] != 2 || groups[0][1] != 3 {
		t.Fatalf("expected one sorted collision group [2 3], got %v", groups)
	}

	// No collisions.
	clean := map[int]*mapNode{
		1: {RoomId: 1, Pos: d(0, 0, 0)},
		2: {RoomId: 2, Pos: d(1, 0, 0)},
	}
	if g := findCollisions(clean); len(g) != 0 {
		t.Fatalf("expected no collisions, got %v", g)
	}

	// Two distinct collision groups, returned in deterministic (ascending) order.
	multi := map[int]*mapNode{
		10: {RoomId: 10, Pos: d(0, 0, 0)},
		11: {RoomId: 11, Pos: d(0, 0, 0)}, // group A @ (0,0,0)
		20: {RoomId: 20, Pos: d(5, 5, 0)},
		21: {RoomId: 21, Pos: d(5, 5, 0)}, // group B @ (5,5,0)
	}
	mg := findCollisions(multi)
	if len(mg) != 2 || mg[0][0] != 10 || mg[1][0] != 20 {
		t.Fatalf("expected two groups ordered [10..],[20..], got %v", mg)
	}
}

// helper: build a minimal *mapper with hand-placed nodes (no crawl).
func mkMapper(nodes map[int]*mapNode) *mapper {
	return &mapper{crawledRooms: nodes}
}

func node(id, x, y, z int, exits map[string]nodeExit) *mapNode {
	return &mapNode{RoomId: id, Pos: d(x, y, z), Exits: exits}
}

func TestCheckConsistency_CleanGrid(t *testing.T) {
	nodes := map[int]*mapNode{
		1: node(1, 0, 0, 0, map[string]nodeExit{"north": {RoomId: 2, Direction: d(0, -1, 0)}}),
		2: node(2, 0, -1, 0, map[string]nodeExit{"south": {RoomId: 1, Direction: d(0, 1, 0)}}),
	}
	if f := mkMapper(nodes).CheckConsistency("test", false); len(f) != 0 {
		t.Fatalf("clean grid should yield no findings, got %v", f)
	}
}

func TestCheckConsistency_MissingReciprocal(t *testing.T) {
	nodes := map[int]*mapNode{
		1: node(1, 0, 0, 0, map[string]nodeExit{"north": {RoomId: 2, Direction: d(0, -1, 0)}}),
		2: node(2, 0, -1, 0, map[string]nodeExit{}),
	}
	f := mkMapper(nodes).CheckConsistency("test", false)
	if len(f) != 1 || f[0].Kind != "noreciprocal" {
		t.Fatalf("expected one noreciprocal finding, got %v", f)
	}
}

func TestCheckConsistency_OnewaySuppressesReciprocal(t *testing.T) {
	nodes := map[int]*mapNode{
		1: node(1, 0, 0, 0, map[string]nodeExit{"north": {RoomId: 2, Direction: d(0, -1, 0), OneWay: true}}),
		2: node(2, 0, -1, 0, map[string]nodeExit{}),
	}
	if f := mkMapper(nodes).CheckConsistency("test", false); len(f) != 0 {
		t.Fatalf("oneway should suppress reciprocity finding, got %v", f)
	}
}

func TestCheckConsistency_WrapFlaggedInCartesian(t *testing.T) {
	nodes := map[int]*mapNode{
		1: node(1, 0, -2, 0, map[string]nodeExit{"south": {RoomId: 2, Direction: d(0, 1, 0)}}),
		2: node(2, 0, 2, 0, map[string]nodeExit{"north": {RoomId: 1, Direction: d(0, -1, 0)}}),
	}
	f := mkMapper(nodes).CheckConsistency("test", false)
	saw := false
	for _, x := range f {
		if x.Kind == "deltamismatch" {
			saw = true
		}
	}
	if !saw {
		t.Fatalf("expected a deltamismatch finding, got %v", f)
	}
}

func TestCheckConsistency_WrapAllowedInNonCartesian(t *testing.T) {
	nodes := map[int]*mapNode{
		1: node(1, 0, -2, 0, map[string]nodeExit{"south": {RoomId: 2, Direction: d(0, 1, 0)}}),
		2: node(2, 0, 2, 0, map[string]nodeExit{"north": {RoomId: 1, Direction: d(0, -1, 0)}}),
	}
	if f := mkMapper(nodes).CheckConsistency("test", true); len(f) != 0 {
		t.Fatalf("non_cartesian zone should suppress wrap/reciprocity findings, got %v", f)
	}
}

func TestCheckConsistency_Collision(t *testing.T) {
	nodes := map[int]*mapNode{
		1: node(1, 0, 0, 0, map[string]nodeExit{}),
		2: node(2, 0, 0, 0, map[string]nodeExit{}),
	}
	f := mkMapper(nodes).CheckConsistency("test", false)
	if len(f) != 1 || f[0].Kind != "collision" {
		t.Fatalf("expected one collision finding, got %v", f)
	}
}
