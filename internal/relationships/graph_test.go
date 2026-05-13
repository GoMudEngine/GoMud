package relationships

import (
	"testing"
)

// edgeSpec is a test-only authoring shape: "mob X declares this edge."
type edgeSpec struct {
	From    int
	To      int
	Type    Type
	Subtype string
}

func buildFromSpecs(specs []edgeSpec) {
	resetGraph()
	graphMu.Lock()
	defer graphMu.Unlock()
	for _, s := range specs {
		// Forward edge.
		graph[s.From] = append(graph[s.From], Relation{Other: s.To, Type: s.Type, Subtype: s.Subtype})
		// Mirror edge.
		mirror := Relation{Other: s.From, Type: InverseType(s.Type)}
		// Skip if mirror already exists (de-dup).
		exists := false
		for _, r := range graph[s.To] {
			if r.Other == s.From && r.Type == mirror.Type {
				exists = true
				break
			}
		}
		if !exists {
			graph[s.To] = append(graph[s.To], mirror)
		}
	}
}

func TestAutoMirror_Symmetric(t *testing.T) {
	buildFromSpecs([]edgeSpec{
		{From: 95, To: 96, Type: TypeFriend, Subtype: "drinking-companion"},
	})

	// 95 has friend → 96.
	if got := edgesOf(95); len(got) != 1 || got[0].Other != 96 || got[0].Type != TypeFriend {
		t.Errorf("95: got %v", got)
	}
	if got := edgesOf(95)[0].Subtype; got != "drinking-companion" {
		t.Errorf("95 subtype: got %q", got)
	}

	// 96 has friend → 95 (auto-mirrored, no subtype).
	if got := edgesOf(96); len(got) != 1 || got[0].Other != 95 || got[0].Type != TypeFriend {
		t.Errorf("96: got %v", got)
	}
	if got := edgesOf(96)[0].Subtype; got != "" {
		t.Errorf("96 subtype should be empty (per-side), got %q", got)
	}
}

func TestAutoMirror_Asymmetric(t *testing.T) {
	buildFromSpecs([]edgeSpec{
		{From: 95, To: 248, Type: TypeEmployer, Subtype: "priest-handyman"},
	})

	// 95: employer → 248.
	if got := edgesOf(95); got[0].Type != TypeEmployer {
		t.Errorf("95 should be employer: got %s", got[0].Type)
	}
	// 248: employee → 95 (auto-mirrored inverse).
	if got := edgesOf(248); got[0].Type != TypeEmployee {
		t.Errorf("248 should be employee: got %s", got[0].Type)
	}
}

// edgesOf is a test helper that takes the cache RLock.
func edgesOf(mobId int) []Relation {
	graphMu.RLock()
	defer graphMu.RUnlock()
	out := make([]Relation, len(graph[mobId]))
	copy(out, graph[mobId])
	return out
}

func TestLoadFromMobs_HappyPath(t *testing.T) {
	resetGraph()
	LoadFromMobs([]MobEdges{
		{MobId: 95, Edges: []EdgeInput{
			{To: 96, Type: TypeFriend, Subtype: "drinking-companion"},
			{To: 117, Type: TypeFamily, Subtype: "niece"},
			{To: 248, Type: TypeEmployer, Subtype: "priest-handyman"},
		}},
		{MobId: 96, Edges: []EdgeInput{}}, // mob with no relationships
	}, alwaysValid)

	if got := len(edgesOf(95)); got != 3 {
		t.Errorf("95 should have 3 edges, got %d", got)
	}
	// 96 mirror of 95's friend edge.
	if got := edgesOf(96); len(got) != 1 || got[0].Other != 95 || got[0].Type != TypeFriend {
		t.Errorf("96 mirror: %v", got)
	}
	// 248 mirror as employee.
	if got := edgesOf(248); len(got) != 1 || got[0].Other != 95 || got[0].Type != TypeEmployee {
		t.Errorf("248 mirror: %v", got)
	}
}

func TestLoadFromMobs_UnknownTargetSkipped(t *testing.T) {
	resetGraph()
	// Mob 99 doesn't exist (validate returns false).
	validate := func(id int) bool { return id == 95 }
	LoadFromMobs([]MobEdges{
		{MobId: 95, Edges: []EdgeInput{
			{To: 99, Type: TypeFriend},
		}},
	}, validate)

	if got := len(edgesOf(95)); got != 0 {
		t.Errorf("edge to unknown mob should be skipped, got %d edges", got)
	}
}

func TestLoadFromMobs_SelfEdgeSkipped(t *testing.T) {
	resetGraph()
	LoadFromMobs([]MobEdges{
		{MobId: 95, Edges: []EdgeInput{
			{To: 95, Type: TypeFriend},
		}},
	}, alwaysValid)

	if got := len(edgesOf(95)); got != 0 {
		t.Errorf("self-edge should be skipped, got %d edges", got)
	}
}

func TestLoadFromMobs_UnknownTypeSkipped(t *testing.T) {
	resetGraph()
	LoadFromMobs([]MobEdges{
		{MobId: 95, Edges: []EdgeInput{
			{To: 96, Type: Type("nonsense")},
		}},
	}, alwaysValid)

	if got := len(edgesOf(95)); got != 0 {
		t.Errorf("unknown-type edge should be skipped, got %d", got)
	}
}

func TestLoadFromMobs_DuplicateEdgesDeduped(t *testing.T) {
	resetGraph()
	LoadFromMobs([]MobEdges{
		{MobId: 95, Edges: []EdgeInput{
			{To: 96, Type: TypeFriend, Subtype: "first"},
			{To: 96, Type: TypeFriend, Subtype: "second"},
		}},
	}, alwaysValid)

	got := edgesOf(95)
	if len(got) != 1 {
		t.Errorf("duplicate edges should dedupe, got %d", len(got))
	}
	if got[0].Subtype != "first" {
		t.Errorf("first declaration wins for subtype, got %q", got[0].Subtype)
	}
}

// alwaysValid is a test helper that approves every mob id.
func alwaysValid(int) bool { return true }
