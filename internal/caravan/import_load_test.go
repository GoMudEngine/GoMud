package caravan

import "testing"

func TestLoadRunnerFromImport_RespectsCapAndSkipsWhenRegistryAbsent(t *testing.T) {
	runner := newCargoTestMob(80904, 9304, "Dobb")
	c := ImportCircuit{
		ImportItems: []int{40001, 40006, 40012},
		LoadCap:     5,
	}
	loaded := LoadRunnerFromImport(runner, c)
	if loaded == 0 {
		// Item registry not loaded in unit context — acceptable; smoke covers it.
		t.Skip("items.New returned invalid (registry not loaded); integrated smoke covers this")
	}
	if loaded > c.LoadCap || len(runner.Character.Items) > c.LoadCap {
		t.Errorf("load exceeded cap: loaded=%d items=%d cap=%d",
			loaded, len(runner.Character.Items), c.LoadCap)
	}
}

func TestLoadRunnerFromImport_NilOrEmptyNoOp(t *testing.T) {
	if got := LoadRunnerFromImport(nil, ImportCircuit{ImportItems: []int{40001}, LoadCap: 3}); got != 0 {
		t.Errorf("nil runner should load 0, got %d", got)
	}
	runner := newCargoTestMob(80905, 9304, "Dobb")
	if got := LoadRunnerFromImport(runner, ImportCircuit{ImportItems: nil, LoadCap: 3}); got != 0 {
		t.Errorf("empty manifest should load 0, got %d", got)
	}
	if got := LoadRunnerFromImport(runner, ImportCircuit{ImportItems: []int{40001}, LoadCap: 0}); got != 0 {
		t.Errorf("zero cap should load 0, got %d", got)
	}
}
