package relationships

import (
	"testing"
)

func seedSimpleGraph(t *testing.T) {
	t.Helper()
	resetGraph()
	LoadFromMobs([]MobEdges{
		{MobId: 95, Edges: []EdgeInput{
			{To: 96, Type: TypeFriend},
			{To: 117, Type: TypeFamily, Subtype: "niece"},
			{To: 113, Type: TypeRival},
			{To: 248, Type: TypeEmployer},
		}},
	}, alwaysValid)
}

func TestRelationsOf(t *testing.T) {
	seedSimpleGraph(t)
	if got := len(RelationsOf(95)); got != 4 {
		t.Errorf("RelationsOf(95): got %d, want 4", got)
	}
	if got := len(RelationsOf(404)); got != 0 {
		t.Errorf("RelationsOf(404 unknown): got %d, want 0", got)
	}
}

func TestRelationsOfType(t *testing.T) {
	seedSimpleGraph(t)
	if got := len(RelationsOfType(95, TypeFamily)); got != 1 {
		t.Errorf("family count: %d", got)
	}
	if got := len(RelationsOfType(95, TypeFamily, TypeFriend)); got != 2 {
		t.Errorf("family+friend count: %d", got)
	}
	// No types passed = all edges.
	if got := len(RelationsOfType(95)); got != 4 {
		t.Errorf("no-filter count: %d", got)
	}
}

func TestKinAlliesRivals(t *testing.T) {
	seedSimpleGraph(t)
	if got := len(KinOf(95)); got != 1 {
		t.Errorf("KinOf: %d", got)
	}
	if got := len(AlliesOf(95)); got != 2 {
		t.Errorf("AlliesOf (family+friend+lover): %d", got)
	}
	if got := len(RivalsOf(95)); got != 1 {
		t.Errorf("RivalsOf: %d", got)
	}
}

func TestRelationsBetween(t *testing.T) {
	seedSimpleGraph(t)
	got := RelationsBetween(95, 96)
	if len(got) != 1 || got[0].Other != 96 {
		t.Errorf("RelationsBetween(95, 96): %v", got)
	}
	if got := RelationsBetween(95, 999); len(got) != 0 {
		t.Errorf("RelationsBetween(95, 999): %v", got)
	}
}

func TestAreRelated(t *testing.T) {
	seedSimpleGraph(t)
	if !AreRelated(95, 96) {
		t.Errorf("95 and 96 should be related")
	}
	if AreRelated(95, 999) {
		t.Errorf("95 and 999 should not be related")
	}
}

func TestEmployerEmployedBy(t *testing.T) {
	seedSimpleGraph(t)
	emp := EmployerOf(95)
	if len(emp) != 1 || emp[0] != 248 {
		t.Errorf("EmployerOf(95): %v, want [248]", emp)
	}
	by := EmployedBy(248)
	if len(by) != 1 || by[0] != 95 {
		t.Errorf("EmployedBy(248): %v, want [95]", by)
	}
}

func TestAllRelations(t *testing.T) {
	seedSimpleGraph(t)
	all := AllRelations()
	// 4 forward edges from 95 + 4 auto-mirrored = 8 total entries.
	if len(all) != 8 {
		t.Errorf("AllRelations: got %d, want 8", len(all))
	}
}
