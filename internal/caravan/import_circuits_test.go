package caravan

import "testing"

func TestImportCircuits_NPDocksRegistered(t *testing.T) {
	c, ok := ImportCircuitFor("np_docks_runner_circuit")
	if !ok {
		t.Fatal("np_docks_runner_circuit not registered")
	}
	if c.RunnerMobId != 9304 {
		t.Errorf("runner mob = %d, want 9304 (Dobb)", c.RunnerMobId)
	}
	if c.DepotEvent != "np_runner_depot" || c.VendorEvent != "np_runner_vendor" {
		t.Errorf("event tags = %q/%q", c.DepotEvent, c.VendorEvent)
	}
	if len(c.DeliveryBuckets) == 0 || len(c.ImportItems) == 0 || c.LoadCap <= 0 {
		t.Errorf("circuit under-specified: %+v", c)
	}
}

func TestImportCircuits_UnknownNotFound(t *testing.T) {
	if _, ok := ImportCircuitFor("nope"); ok {
		t.Error("unknown circuit should not resolve")
	}
}
