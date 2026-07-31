package relationships

import "testing"

func allMobsExist(int) bool { return true }

// Every authored edge is auto-mirrored, and the mirror deliberately carries no
// subtype ("per-side"). But when BOTH sides author the relationship with their
// own subtype — which is how all three affected pairs in the dogmud world are
// written — the mirror created by the first side makes the second side's
// declaration look like a duplicate, so it is skipped and its subtype is lost.
//
// A later explicit declaration must be allowed to fill in a subtype the mirror
// left empty.
func TestLoadFromMobs_ExplicitDeclarationFillsMirrorSubtype(t *testing.T) {
	input := []MobEdges{
		{MobId: 9381, Edges: []EdgeInput{
			{To: 9320, Type: TypeFriend, Subtype: "lore_source"},
		}},
		{MobId: 9320, Edges: []EdgeInput{
			{To: 9381, Type: TypeFriend, Subtype: "lore_source"},
		}},
	}

	LoadFromMobs(input, allMobsExist)

	forward := findRelation(t, 9381, 9320)
	if forward.Subtype != "lore_source" {
		t.Errorf("forward edge lost its subtype: %q", forward.Subtype)
	}

	reverse := findRelation(t, 9320, 9381)
	if reverse.Subtype != "lore_source" {
		t.Errorf("reverse edge subtype was dropped by mirror dedup: got %q, want %q",
			reverse.Subtype, "lore_source")
	}
}

// The mirror must still be created when only one side authors the edge, and it
// must still carry no subtype — subtypes are per-side by design.
func TestLoadFromMobs_OneSidedEdgeStillMirrorsWithoutSubtype(t *testing.T) {
	input := []MobEdges{
		{MobId: 100, Edges: []EdgeInput{
			{To: 200, Type: TypeFriend, Subtype: "old_comrade"},
		}},
	}

	LoadFromMobs(input, allMobsExist)

	if got := findRelation(t, 100, 200).Subtype; got != "old_comrade" {
		t.Errorf("authored side lost its subtype: %q", got)
	}
	if got := findRelation(t, 200, 100).Subtype; got != "" {
		t.Errorf("mirror invented a subtype: %q", got)
	}
}

// A genuine duplicate — the same side declaring the same edge twice — must
// still be skipped, and must not overwrite the first subtype.
func TestLoadFromMobs_GenuineDuplicateStillFirstWins(t *testing.T) {
	input := []MobEdges{
		{MobId: 300, Edges: []EdgeInput{
			{To: 400, Type: TypeFriend, Subtype: "first"},
			{To: 400, Type: TypeFriend, Subtype: "second"},
		}},
	}

	LoadFromMobs(input, allMobsExist)

	if got := findRelation(t, 300, 400).Subtype; got != "first" {
		t.Errorf("a repeated declaration overwrote the first subtype: %q", got)
	}
}

func findRelation(t *testing.T, from, to int) Relation {
	t.Helper()
	graphMu.RLock()
	defer graphMu.RUnlock()
	for _, r := range graph[from] {
		if r.Other == to {
			return r
		}
	}
	t.Fatalf("no relation %d→%d in graph", from, to)
	return Relation{}
}
