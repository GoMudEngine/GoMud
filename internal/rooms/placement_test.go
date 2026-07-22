package rooms

import "testing"

func TestValidatePlacement(t *testing.T) {
	// occupied(plane,x,y,z) is injected so the test doesn't need loaded rooms.
	occ := map[[4]int]int{{0, 1, 1, 0}: 42}
	lookup := func(p, x, y, z int) (int, bool) { id, ok := occ[[4]int{p, x, y, z}]; return id, ok }

	// Free cell on a Euclidean plane: ok.
	if err := validatePlacement(0, 2, 2, 0, 0, false, lookup); err != nil {
		t.Errorf("free cell should validate: %v", err)
	}
	// Occupied cell by another room on a Euclidean plane: rejected.
	if err := validatePlacement(0, 1, 1, 0, 0, false, lookup); err == nil {
		t.Error("occupied Euclidean cell must be rejected")
	}
	// Occupied cell but it's the SAME room being moved (excludeRoomId): ok.
	if err := validatePlacement(0, 1, 1, 0, 42, false, lookup); err != nil {
		t.Errorf("self-occupied cell should validate: %v", err)
	}
	// Occupied cell on a non-Euclidean plane: ok (no grid enforcement).
	if err := validatePlacement(0, 1, 1, 0, 0, true, lookup); err != nil {
		t.Errorf("non-Euclidean overlap should be allowed: %v", err)
	}
}
