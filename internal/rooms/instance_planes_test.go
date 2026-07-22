package rooms

import "testing"

func TestNextInstancePlane_DistinctAndClearOfWorld(t *testing.T) {
	a := nextInstancePlane()
	b := nextInstancePlane()
	if a == b {
		t.Errorf("successive instance planes must differ: %d == %d", a, b)
	}
	if a < instancePlaneBase || b < instancePlaneBase {
		t.Errorf("instance planes must stay clear of authored world planes (>= %d): %d, %d",
			instancePlaneBase, a, b)
	}
	if b <= a {
		t.Errorf("instance planes must increase monotonically: %d then %d", a, b)
	}
}
