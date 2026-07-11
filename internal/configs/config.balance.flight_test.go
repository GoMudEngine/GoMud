package configs

import "testing"

func TestFlightDefaults(t *testing.T) {
	b := GetBalanceConfig()
	if int(b.FlightOpposedEdge) != 25 {
		t.Fatalf("FlightOpposedEdge = %v, want 25", int(b.FlightOpposedEdge))
	}
	if float64(b.FlightMoveStaminaMult) != 0.5 {
		t.Fatalf("FlightMoveStaminaMult = %v, want 0.5", float64(b.FlightMoveStaminaMult))
	}
	if float64(b.FlightFleeStaminaMult) != 0.5 {
		t.Fatalf("FlightFleeStaminaMult = %v, want 0.5", float64(b.FlightFleeStaminaMult))
	}
}
