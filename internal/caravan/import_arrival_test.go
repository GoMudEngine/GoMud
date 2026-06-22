package caravan

import "testing"

func TestClassifyImportArrival_RoutesByEventTag(t *testing.T) {
	c, _ := ImportCircuitFor("np_docks_runner_circuit")
	if got := classifyImportArrival(c, "np_runner_depot"); got != importLoad {
		t.Errorf("depot event → %v, want importLoad", got)
	}
	if got := classifyImportArrival(c, "np_runner_vendor"); got != importDeliver {
		t.Errorf("vendor event → %v, want importDeliver", got)
	}
	if got := classifyImportArrival(c, "something_else"); got != importNone {
		t.Errorf("unknown event → %v, want importNone", got)
	}
}
