package sim

import "testing"

func TestForceSpawn(t *testing.T) {
	g := coverageGraph()
	cfg := DefaultConfig()
	st := NewState(1)

	next, diff, ok := ForceSpawn(st, g, cfg, "storm", "B", 0.9, Clock{Round: 5})
	if !ok {
		t.Fatal("spawn into a known zone must succeed")
	}
	if len(next.Fronts) != 1 || next.Fronts[0].Type != "storm" || next.Fronts[0].Zone != "B" {
		t.Fatalf("front not created: %+v", next.Fronts)
	}
	if next.Fronts[0].Id != 1 || next.NextID != 2 {
		t.Errorf("front id accounting wrong: id=%d nextId=%d", next.Fronts[0].Id, next.NextID)
	}
	if next.Weather["B"] != "storm" {
		t.Errorf("zone B should be storm, got %s", next.Weather["B"])
	}
	if len(diff.Changes) == 0 {
		t.Error("diff should record the new weather")
	}
	if next.Round != 5 {
		t.Errorf("round not stamped: %d", next.Round)
	}
	// Original state untouched (pure function).
	if len(st.Fronts) != 0 || len(st.Weather) != 0 {
		t.Error("input state must not be mutated")
	}

	if _, _, ok := ForceSpawn(st, g, cfg, "storm", "nowhere", 0.9, Clock{}); ok {
		t.Error("unknown zone must fail")
	}
}

func TestForceSpawnDefaultIntensity(t *testing.T) {
	g := coverageGraph()
	next, _, ok := ForceSpawn(NewState(1), g, DefaultConfig(), "rain", "A", 0, Clock{})
	if !ok || next.Fronts[0].Intensity != 0.6 {
		t.Fatalf("zero intensity should default to 0.6, got %+v", next.Fronts)
	}
}

func TestClearZones(t *testing.T) {
	g := coverageGraph()
	cfg := DefaultConfig()
	st := NewState(1)
	st, _, _ = ForceSpawn(st, g, cfg, "storm", "A", 0.9, Clock{})
	st, _, _ = ForceSpawn(st, g, cfg, "fog", "D", 0.9, Clock{})

	// Clearing zone A removes the storm centered there (coverage trivially
	// reaches its own center) but not the fog at D, whose projection to A is
	// 3 hops away — beyond MaxFrontRadius 2, so it does not cover A.
	next, diff := ClearZones(st, g, cfg, []ZoneId{"A"}, Clock{Round: 9})
	if len(next.Fronts) != 1 || next.Fronts[0].Type != "fog" {
		t.Fatalf("only the fog front should survive: %+v", next.Fronts)
	}
	if next.Weather["A"] != Clear {
		t.Errorf("cleared zone should be clear, got %s", next.Weather["A"])
	}
	if len(diff.Changes) == 0 {
		t.Error("clearing should diff")
	}

	// No zones = clear everything.
	next, _ = ClearZones(st, g, cfg, nil, Clock{Round: 9})
	if len(next.Fronts) != 0 {
		t.Fatalf("clear-all should drop every front: %+v", next.Fronts)
	}
	for z, w := range next.Weather {
		if w != Clear {
			t.Errorf("zone %s should be clear, got %s", z, w)
		}
	}

	// Original state untouched (pure function).
	if st.Weather["A"] != "storm" || st.Weather["D"] != "fog" {
		t.Error("ClearZones must not mutate the input state's Weather map")
	}
	if len(st.Fronts) != 2 {
		t.Errorf("ClearZones must not mutate the input state's Fronts slice: %v", st.Fronts)
	}
}
