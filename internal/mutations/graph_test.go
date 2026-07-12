package mutations

import "testing"

func TestMutationSpec_GraphFields(t *testing.T) {
	spec := &MutationSpec{
		MutationId:    "rending-claws",
		Name:          "Rending Claws",
		Rarity:        3,
		Clusters:      []string{"ravener"},
		Pole:          "body",
		Prerequisites: []MutationPrereq{{Id: "hollow-bones", MinLevel: 2}},
	}
	if got := spec.Prerequisites[0].MinLevel; got != 2 {
		t.Fatalf("MinLevel = %d, want 2", got)
	}
	if spec.Pole != "body" {
		t.Fatalf("Pole = %q, want body", spec.Pole)
	}
}

func TestValidateGraph_UnknownPrereqPanics(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"a": {MutationId: "a", Name: "A", Rarity: 1,
			Prerequisites: []MutationPrereq{{Id: "does-not-exist"}}},
	})
	defer cleanup()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on unknown prerequisite id")
		}
	}()
	ValidateGraph()
}

func TestValidateGraph_UnknownPolePanics(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"a": {MutationId: "a", Name: "A", Rarity: 1, Pole: "spooky"},
	})
	defer cleanup()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on unknown pole")
		}
	}()
	ValidateGraph()
}

func TestValidateGraph_ValidPasses(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"hollow-bones": {MutationId: "hollow-bones", Name: "Hollow Bones", Rarity: 2,
			Clusters: []string{"generalist"}, Pole: ""},
		"winged-flight": {MutationId: "winged-flight", Name: "Winged Flight", Rarity: 8,
			Clusters:      []string{"generalist"},
			Prerequisites: []MutationPrereq{{Id: "hollow-bones", MinLevel: 1}}},
	})
	defer cleanup()
	ValidateGraph() // must not panic
}

func TestClustersForSkill(t *testing.T) {
	if got := ClustersForSkill("spellcasting"); len(got) != 1 || got[0] != "ethereal" {
		t.Fatalf("spellcasting -> %v, want [ethereal]", got)
	}
	if got := ClustersForSkill("cooking"); len(got) != 1 || got[0] != "chrysifier" {
		t.Fatalf("cooking -> %v, want [chrysifier]", got)
	}
	if got := ClustersForSkill("tracking"); got != nil {
		t.Fatalf("tracking -> %v, want nil (unmapped skill)", got)
	}
}

func TestOwnedGravity(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"claws": {MutationId: "claws", Name: "Claws", Rarity: 3, Clusters: []string{"ravener"}},
		"fangs": {MutationId: "fangs", Name: "Fangs", Rarity: 4, Clusters: []string{"ravener", "stalker"}},
	})
	defer cleanup()
	g := OwnedGravity(map[string]int{"claws": 1, "fangs": 2})
	if g["ravener"] != 3 { // 1 + 2
		t.Fatalf("ravener gravity = %v, want 3", g["ravener"])
	}
	if g["stalker"] != 2 {
		t.Fatalf("stalker gravity = %v, want 2", g["stalker"])
	}
}
