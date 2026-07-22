package rooms

import "testing"

func TestPlaneRegistry_NonEuclidean(t *testing.T) {
	reg := NewPlaneRegistry()
	reg.Mark(0, false, "overworld")
	reg.Mark(7, true, "labyrinth")
	if reg.IsNonEuclidean(0) {
		t.Error("plane 0 should be Euclidean")
	}
	if !reg.IsNonEuclidean(7) {
		t.Error("plane 7 should be non-Euclidean")
	}
	// Unknown planes default to Euclidean (safe: enforce by default).
	if reg.IsNonEuclidean(99) {
		t.Error("unknown plane should default Euclidean")
	}
	// Mark wins if ANY contributor is non-Euclidean (idempotent OR).
	reg.Mark(0, true, "overworld")
	if !reg.IsNonEuclidean(0) {
		t.Error("plane 0 should flip non-Euclidean after a non-Euclidean mark")
	}
}
