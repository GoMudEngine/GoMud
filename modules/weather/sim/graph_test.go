package sim

import (
	"reflect"
	"testing"
)

func TestGraphJSONRoundTrip(t *testing.T) {
	g := &Graph{
		Version:      GraphVersion,
		BuiltAtRound: 12345,
		Nodes: map[string]ZoneNode{
			"Frostfang": {Zone: "Frostfang", Biome: "tundra", Rooms: 24, HasOutdoor: true},
			"Saltmarsh": {Zone: "Saltmarsh", Biome: "swamp", Rooms: 18, HasOutdoor: true},
		},
		Edges:      []Edge{{A: "Frostfang", B: "Saltmarsh", Weight: 3}},
		Components: 1,
	}

	b, err := g.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	got, err := FromJSON(b)
	if err != nil {
		t.Fatalf("FromJSON: %v", err)
	}

	// adj is unexported and never populated in this test, so DeepEqual is safe.
	if !reflect.DeepEqual(g, got) {
		t.Errorf("round trip mismatch:\n want %+v\n got  %+v", g, got)
	}
}

func TestGraphNeighbors(t *testing.T) {
	g := &Graph{
		Edges: []Edge{
			{A: "A", B: "B", Weight: 2},
			{A: "B", B: "C", Weight: 1},
		},
	}

	got := map[string]int{}
	for _, e := range g.Neighbors("B") {
		if e.A != "B" {
			t.Errorf("neighbor edge should be oriented from the queried zone; got A=%q", e.A)
		}
		got[e.B] = e.Weight
	}

	want := map[string]int{"A": 2, "C": 1}
	if len(got) != len(want) {
		t.Fatalf("want %d neighbors, got %d (%v)", len(want), len(got), got)
	}
	for z, w := range want {
		if got[z] != w {
			t.Errorf("neighbor %q: want weight %d, got %d", z, w, got[z])
		}
	}

	if n := g.Neighbors("Z"); len(n) != 0 {
		t.Errorf("unknown zone should have no neighbors, got %v", n)
	}
}

func TestFromJSONError(t *testing.T) {
	if _, err := FromJSON([]byte("{not valid json")); err == nil {
		t.Error("FromJSON should return an error for malformed JSON")
	}
}

func TestNeighborsUsesStableIndex(t *testing.T) {
	g := &Graph{
		Nodes: map[string]ZoneNode{"A": {Zone: "A"}, "B": {Zone: "B"}, "C": {Zone: "C"}},
		Edges: []Edge{{A: "A", B: "B", Weight: 2}, {A: "B", B: "C", Weight: 1}},
	}
	first := g.Neighbors("B")
	second := g.Neighbors("B")
	if len(first) != 2 || len(second) != 2 {
		t.Fatalf("B should have 2 neighbors, got %d then %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("Neighbors must be stable across calls: %v vs %v", first, second)
		}
	}
	// Orientation: every returned edge is oriented from the queried zone.
	for _, e := range first {
		if e.A != "B" {
			t.Errorf("edge not oriented from queried zone: %+v", e)
		}
	}
	if g.Neighbors("missing") != nil {
		t.Error("unknown zone should return nil")
	}

	g2 := &Graph{Nodes: map[string]ZoneNode{"L": {Zone: "L"}}}
	if g2.Neighbors("L") != nil {
		t.Error("known zone with no edges should return nil")
	}
}

func TestNeighborsIndexRebuiltAfterDecode(t *testing.T) {
	g := &Graph{
		Nodes: map[string]ZoneNode{"A": {Zone: "A"}, "B": {Zone: "B"}},
		Edges: []Edge{{A: "A", B: "B", Weight: 1}},
	}
	_ = g.Neighbors("A") // force index build
	b, err := g.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	g2, err := FromJSON(b)
	if err != nil {
		t.Fatal(err)
	}
	n := g2.Neighbors("B")
	if len(n) != 1 || n[0].B != "A" || n[0].Weight != 1 {
		t.Fatalf("decoded graph must answer Neighbors correctly, got %v", n)
	}
}

func TestFindZone(t *testing.T) {
	g := &Graph{Nodes: map[string]ZoneNode{"Frostfang": {Zone: "Frostfang"}}}
	if z, ok := g.FindZone("Frostfang"); !ok || z != "Frostfang" {
		t.Errorf("exact match failed: %q %v", z, ok)
	}
	if z, ok := g.FindZone("frostFANG"); !ok || z != "Frostfang" {
		t.Errorf("case-insensitive match failed: %q %v", z, ok)
	}
	if _, ok := g.FindZone("nowhere"); ok {
		t.Error("missing zone must not match")
	}
}

func TestGraphZones(t *testing.T) {
	g := &Graph{Nodes: map[string]ZoneNode{
		"B": {Zone: "B"}, "A": {Zone: "A"}, "C": {Zone: "C"},
	}}
	got := g.Zones()
	want := []string{"A", "B", "C"} // sorted for determinism
	if len(got) != len(want) {
		t.Fatalf("want %d zones, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Zones()[%d] = %q, want %q (must be sorted)", i, got[i], want[i])
		}
	}
}
