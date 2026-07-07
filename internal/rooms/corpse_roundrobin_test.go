package rooms

import "testing"

func TestRoundRobinOrder(t *testing.T) {
	assignees, cursor := RoundRobinOrder(3, []int{10, 20}, 0)
	want := []int{10, 20, 10}
	if len(assignees) != len(want) {
		t.Fatalf("len(assignees)=%d want %d", len(assignees), len(want))
	}
	for i := range want {
		if assignees[i] != want[i] {
			t.Fatalf("assignees[%d]=%d want %d", i, assignees[i], want[i])
		}
	}
	if cursor != 1 {
		t.Fatalf("endCursor=%d want 1", cursor)
	}
}

func TestRoundRobinOrder_EmptyMembers(t *testing.T) {
	assignees, cursor := RoundRobinOrder(3, nil, 5)
	if assignees != nil {
		t.Fatalf("expected nil assignees for empty members, got %v", assignees)
	}
	if cursor != 5 {
		t.Fatalf("endCursor=%d want 5 (unchanged)", cursor)
	}
}

func TestRoundRobinOrder_CursorWraps(t *testing.T) {
	// start mid-rotation: cursor 1, 2 members, 2 items -> [20,10], end cursor 1
	assignees, cursor := RoundRobinOrder(2, []int{10, 20}, 1)
	want := []int{20, 10}
	for i := range want {
		if assignees[i] != want[i] {
			t.Fatalf("assignees[%d]=%d want %d", i, assignees[i], want[i])
		}
	}
	if cursor != 1 {
		t.Fatalf("endCursor=%d want 1", cursor)
	}
}
