package ferry

import "testing"

// factorDecide is pure: given the circuit, the vessel state for its route,
// which port the factor is at (or -1 if elsewhere), whether it's on the
// deck, its current machine state, and its delivery progress, return the
// action to take this round.
func TestFactorDecide_BoardsWhenDockedAtItsPort(t *testing.T) {
	c := validCircuit()
	d := factorDecide(c, VesselState{Docked: true, PortIdx: 0}, factorPos{AtPortIdx: 0, OnDeck: false},
		factorState{Phase: FactorWaiting})
	if d.Kind != ActBoard {
		t.Fatalf("expected ActBoard, got %+v", d)
	}
}

func TestFactorDecide_DisembarksOnArrival(t *testing.T) {
	c := validCircuit()
	d := factorDecide(c, VesselState{Docked: true, PortIdx: 1}, factorPos{AtPortIdx: -1, OnDeck: true},
		factorState{Phase: FactorAboard})
	if d.Kind != ActDisembark || d.PortIdx != 1 {
		t.Fatalf("expected ActDisembark at port 1, got %+v", d)
	}
}

// TestFactorDecide_StaysAboardAtBoardingPort pins the disembark guard:
// after boarding, the vessel is still docked at the boarding port for the
// rest of the layover. The factor must NOT step back ashore there — it
// waits aboard until the vessel reaches the OTHER port.
func TestFactorDecide_StaysAboardAtBoardingPort(t *testing.T) {
	c := validCircuit()
	d := factorDecide(c, VesselState{Docked: true, PortIdx: 0}, factorPos{AtPortIdx: -1, OnDeck: true},
		factorState{Phase: FactorAboard, PortIdx: 0})
	if d.Kind != ActNone {
		t.Fatalf("expected ActNone while docked at the boarding port, got %+v", d)
	}
}

// TestFactorDecide_DisembarksOnlyAtOtherPort is the positive twin: docked
// at the port OPPOSITE the boarding port → disembark there.
func TestFactorDecide_DisembarksOnlyAtOtherPort(t *testing.T) {
	c := validCircuit()
	d := factorDecide(c, VesselState{Docked: true, PortIdx: 1}, factorPos{AtPortIdx: -1, OnDeck: true},
		factorState{Phase: FactorAboard, PortIdx: 0})
	if d.Kind != ActDisembark || d.PortIdx != 1 {
		t.Fatalf("expected ActDisembark at port 1, got %+v", d)
	}
}

func TestFactorDecide_StaysAboardAtSea(t *testing.T) {
	c := validCircuit()
	d := factorDecide(c, VesselState{Docked: false, PortIdx: 1}, factorPos{AtPortIdx: -1, OnDeck: true},
		factorState{Phase: FactorAboard})
	if d.Kind != ActNone {
		t.Fatalf("expected ActNone at sea, got %+v", d)
	}
}

func TestFactorDecide_WalksStopsInOrderThenReturns(t *testing.T) {
	c := validCircuit()
	// Delivering at port 1, currently AT stop 0 (5505) → deliver + advance.
	d := factorDecide(c, VesselState{Docked: false, PortIdx: 0},
		factorPos{InRoom: 5505}, factorState{Phase: FactorDelivering, PortIdx: 1, StopIdx: 0})
	if d.Kind != ActDeliverHere || d.NextStop != 5508 {
		t.Fatalf("expected ActDeliverHere then next stop 5508, got %+v", d)
	}
	// Past the last stop → return to dock.
	d = factorDecide(c, VesselState{Docked: false, PortIdx: 0},
		factorPos{InRoom: 5508}, factorState{Phase: FactorDelivering, PortIdx: 1, StopIdx: 1})
	if d.Kind != ActDeliverHere || !d.LastStop {
		t.Fatalf("expected final ActDeliverHere with LastStop, got %+v", d)
	}
}

func TestFactorDecide_MissedBoatKeepsWaiting(t *testing.T) {
	c := validCircuit()
	// Vessel docked at the OTHER port → keep waiting, no action.
	d := factorDecide(c, VesselState{Docked: true, PortIdx: 1}, factorPos{AtPortIdx: 0},
		factorState{Phase: FactorWaiting})
	if d.Kind != ActNone {
		t.Fatalf("expected ActNone, got %+v", d)
	}
}
