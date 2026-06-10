package sim

import "testing"

func coverageGraph() *Graph {
	// A - B - C - D chain.
	return &Graph{
		Nodes: map[string]ZoneNode{
			"A": {Zone: "A", Biome: "plains"}, "B": {Zone: "B", Biome: "plains"},
			"C": {Zone: "C", Biome: "plains"}, "D": {Zone: "D", Biome: "plains"},
		},
		Edges: []Edge{{A: "A", B: "B", Weight: 1}, {A: "B", B: "C", Weight: 1}, {A: "C", B: "D", Weight: 1}},
	}
}

func TestCovering(t *testing.T) {
	g := coverageGraph()
	cfg := DefaultConfig() // falloff 0.5, min 0.15, radius 2
	fronts := []Front{
		{Id: 1, Type: "storm", Zone: "A", Intensity: 0.8},
		{Id: 2, Type: "fog", Zone: "C", Intensity: 0.3},
	}
	// Zone B: storm projects 0.8*0.5=0.40 (1 hop), fog projects 0.3*0.5=0.15 (1 hop).
	got := Covering(g, fronts, cfg, "B")
	if len(got) != 2 {
		t.Fatalf("expected 2 covering fronts at B, got %d: %+v", len(got), got)
	}
	if got[0].Front.Id != 1 || got[0].Hops != 1 || got[0].Effective != 0.4 {
		t.Errorf("strongest first: %+v", got[0])
	}
	if got[1].Front.Id != 2 || got[1].Effective != 0.15 {
		t.Errorf("weaker second: %+v", got[1])
	}
	// Zone D: storm is 3 hops away (out of radius); fog projects 0.15 at 1 hop.
	got = Covering(g, fronts, cfg, "D")
	if len(got) != 1 || got[0].Front.Id != 2 {
		t.Fatalf("expected only fog at D: %+v", got)
	}
	// Unknown zone: nothing.
	if got := Covering(g, fronts, cfg, "nope"); len(got) != 0 {
		t.Errorf("unknown zone should have no coverage: %+v", got)
	}
}

func TestCoveringMatchesResolveWeather(t *testing.T) {
	g := coverageGraph()
	cfg := DefaultConfig()
	fronts := []Front{
		{Id: 1, Type: "storm", Zone: "A", Intensity: 0.8},
		{Id: 2, Type: "rain", Zone: "B", Intensity: 0.9},
	}
	resolved := resolveWeather(g, fronts, cfg)
	for _, z := range g.Zones() {
		covers := Covering(g, fronts, cfg, z)
		want := Clear
		if len(covers) > 0 {
			want = covers[0].Front.Type
		}
		if resolved[z] != want {
			t.Errorf("zone %s: resolveWeather=%s but Covering says %s", z, resolved[z], want)
		}
	}
}
