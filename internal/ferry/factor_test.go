package ferry

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/warehouse"
)

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

// TestFactorDecide_WaitingGuard_NotAtDockKeepsWaiting pins the THIRD clause
// of the FactorWaiting guard (vs.Docked && vs.PortIdx == st.PortIdx &&
// pos.AtPortIdx == st.PortIdx): the vessel is docked at the factor's own
// port, but the factor itself isn't standing in that dock room (AtPortIdx
// == -1, e.g. mid-recovery-walk). Without this clause a factor elsewhere
// in the world would spuriously "board" the vessel.
func TestFactorDecide_WaitingGuard_NotAtDockKeepsWaiting(t *testing.T) {
	c := validCircuit()
	d := factorDecide(c, VesselState{Docked: true, PortIdx: 0}, factorPos{AtPortIdx: -1},
		factorState{Phase: FactorWaiting, PortIdx: 0})
	if d.Kind != ActNone {
		t.Fatalf("expected ActNone when docked at the factor's port but the factor isn't there, got %+v", d)
	}
}

// TestFactorDecide_Delivering_WalksToCurrentStopWhenNotThereYet pins the
// ActWalkTo branch of FactorDelivering: when pos.InRoom is NOT the current
// stop (StopIdx's room), the action is ActWalkTo targeting the CURRENT
// stop (not the next one) — the factor hasn't arrived yet, so it must
// finish walking to stop 1 (5508) before delivery there can fire.
func TestFactorDecide_Delivering_WalksToCurrentStopWhenNotThereYet(t *testing.T) {
	c := validCircuit()
	d := factorDecide(c, VesselState{Docked: false, PortIdx: 0},
		factorPos{InRoom: 9999}, factorState{Phase: FactorDelivering, PortIdx: 1, StopIdx: 1})
	if d.Kind != ActWalkTo || d.NextStop != 5508 {
		t.Fatalf("expected ActWalkTo NextStop=5508 (current stop, not next), got %+v", d)
	}
}

// TestFactorPhaseName_MapsAllPhases pins the FactorPhase → dashboard
// string mapping used by economy/health.captureFerryFactors, using the
// _test.go hooks so this stays a pure unit test of the mapping (the
// capture-side integration is covered separately in
// internal/economy/health/capture_ferry_test.go).
func TestFactorPhaseName_MapsAllPhases(t *testing.T) {
	cases := []struct {
		phase FactorPhase
		want  string
	}{
		{FactorWaiting, "waiting_at_dock"},
		{FactorAboard, "aboard"},
		{FactorDelivering, "delivering"},
		{FactorReturning, "returning_to_dock"},
	}
	for _, tc := range cases {
		const instId = 999001
		SetFactorStateForTest(instId, tc.phase)
		got, ok := FactorPhaseName(instId)
		ClearFactorStateForTest(instId)
		if !ok {
			t.Fatalf("phase %d: FactorPhaseName reported ok=false", tc.phase)
		}
		if got != tc.want {
			t.Errorf("phase %d: got %q, want %q", tc.phase, got, tc.want)
		}
	}
}

// TestFactorPhaseName_UnknownInstanceNotOk verifies the ok=false path for
// an instance id that was never registered as a trade factor.
func TestFactorPhaseName_UnknownInstanceNotOk(t *testing.T) {
	if _, ok := FactorPhaseName(-1); ok {
		t.Fatalf("expected ok=false for an unregistered instance id")
	}
}

// TestPrioritizeByDemand_SortsDescStable pins the Stage 4 load-time
// shortage-prioritization sort: descending by demand, ties keeping
// manifest order. sort.SliceStable's Less receives INDICES into the
// output slice, not item ids — a naive `demand[a] > demand[b]` reads
// demand keyed by index instead of item id, which this test would catch.
func TestPrioritizeByDemand_SortsDescStable(t *testing.T) {
	items := []int{40056, 40057, 40058, 40059}
	demand := map[int]int{40056: 2, 40057: 9, 40058: 0, 40059: 9}
	got := prioritizeByDemand(items, demand)
	// Desc by demand; ties keep manifest order (40057 before 40059).
	want := []int{40057, 40059, 40056, 40058}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	// Input slice must not be mutated.
	if items[0] != 40056 {
		t.Fatal("input mutated")
	}
}

// TestLoadFromWarehouse_DrawsWarehouseStockFirst exercises the Stage 4
// glue directly: loadFromWarehouse is called with no toggle-checking of
// its own (that lives in ensureFactorLoaded), so the test can drive the
// mechanism without needing to fake config state. PortStops is left empty
// so exportDemand contributes no demand (clean fallback to manifest
// order, per the "cold boot" contract) — priority ordering itself is
// pinned separately by TestPrioritizeByDemand_SortsDescStable.
func TestLoadFromWarehouse_DrawsWarehouseStockFirst(t *testing.T) {
	warehouse.ResetForTest()
	warehouse.Deposit("The Confluence", 40123, 5)

	factor := &mobs.Mob{MobId: 9577, InstanceId: 80950}
	factor.Character.Name = "Test Factor"
	factor.Character.Buffs = buffs.New()

	c := TradeCircuit{
		RouteId:         "test_route",
		PortExports:     [2][]int{{40123}, {}},
		PortStops:       [2][]int{{}, {}},
		LoadCap:         3,
		NewSlotMaxStock: 6,
	}

	loaded := loadFromWarehouse(factor, c, 0, "The Confluence")
	if loaded == 0 {
		// Item registry not loaded in unit context — acceptable; smoke
		// covers it (mirrors caravan.TestLoadRunnerFromImport_RespectsCapAndSkipsWhenRegistryAbsent).
		t.Skip("items.New returned invalid (registry not loaded); integrated smoke covers this")
	}
	if got := warehouse.WarehouseFor("The Confluence").StockOf(40123); got != 5-loaded {
		t.Errorf("warehouse stock = %d, want %d (5 - %d loaded)", got, 5-loaded, loaded)
	}
	if len(factor.Character.Items) != loaded {
		t.Errorf("factor carries %d items, want %d", len(factor.Character.Items), loaded)
	}
	if warehouse.WarehouseFor("The Confluence").DrawnCount != loaded {
		t.Errorf("DrawnCount = %d, want %d", warehouse.WarehouseFor("The Confluence").DrawnCount, loaded)
	}
}
