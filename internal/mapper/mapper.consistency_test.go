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
	}
	for _, c := range cases {
		if got := classifyKind(c.nominal, c.actual); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

func TestFindCollisions(t *testing.T) {
	nodes := map[int]*mapNode{
		1: {RoomId: 1, Pos: d(0, 0, 0)},
		2: {RoomId: 2, Pos: d(1, 0, 0)},
		3: {RoomId: 3, Pos: d(1, 0, 0)}, // collides with 2
	}
	groups := findCollisions(nodes)
	if len(groups) != 1 || len(groups[0]) != 2 {
		t.Fatalf("expected one collision group of 2, got %v", groups)
	}
}
