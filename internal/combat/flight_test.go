package combat

import "testing"

func TestFlightEdge(t *testing.T) {
	if e := flightEdge(true, false, 25); e != 25 {
		t.Fatalf("flyer vs grounded = 25, got %d", e)
	}
	if e := flightEdge(false, true, 25); e != -25 {
		t.Fatalf("grounded vs flyer = -25, got %d", e)
	}
	if e := flightEdge(true, true, 25); e != 0 {
		t.Fatalf("flyer vs flyer = 0, got %d", e)
	}
	if e := flightEdge(false, false, 25); e != 0 {
		t.Fatalf("grounded vs grounded = 0, got %d", e)
	}
}
