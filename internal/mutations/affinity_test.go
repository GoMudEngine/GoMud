package mutations

import "testing"

func TestDecayAffinity(t *testing.T) {
	aff := map[string]float64{"ravener": 10, "tiny": 0.005}
	DecayAffinity(aff, 0.5)
	if aff["ravener"] != 5 {
		t.Fatalf("ravener = %v, want 5", aff["ravener"])
	}
	if _, ok := aff["tiny"]; ok {
		t.Fatal("negligible affinity should be pruned")
	}
}

func TestPrereqsMet(t *testing.T) {
	spec := &MutationSpec{MutationId: "flight", Name: "Flight", Rarity: 8,
		Prerequisites: []MutationPrereq{{Id: "hollow-bones", MinLevel: 2}}}
	if PrereqsMet(map[string]int{"hollow-bones": 1}, spec) {
		t.Fatal("should be unmet at level 1")
	}
	if !PrereqsMet(map[string]int{"hollow-bones": 2}, spec) {
		t.Fatal("should be met at level 2")
	}
	if !PrereqsMet(map[string]int{}, &MutationSpec{MutationId: "x"}) {
		t.Fatal("no prerequisites -> always met")
	}
}

func TestEffectiveAffinity_AddsGravity(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		"claws": {MutationId: "claws", Name: "Claws", Rarity: 3, Clusters: []string{"ravener"}},
	})
	defer cleanup()
	eff := EffectiveAffinity(map[string]int{"claws": 2}, map[string]float64{"ravener": 1})
	if eff["ravener"] != 3 { // action 1 + gravity 2
		t.Fatalf("ravener = %v, want 3", eff["ravener"])
	}
}

func TestGetGraphPool_GatesByAffinityAndPrereqs(t *testing.T) {
	cleanup := SeedMutationsForTest(map[string]*MutationSpec{
		// universal enabler: no cluster -> always eligible
		"hollow-bones": {MutationId: "hollow-bones", Name: "HB", Rarity: 2},
		// ravener deep keystone: needs affinity >= rarity*1.0 = 5
		"apex": {MutationId: "apex", Name: "Apex", Rarity: 5, Clusters: []string{"ravener"}},
		// gated behind a prerequisite the owner lacks
		"flight": {MutationId: "flight", Name: "Flight", Rarity: 8, Clusters: []string{"generalist"},
			Prerequisites: []MutationPrereq{{Id: "hollow-bones", MinLevel: 1}}},
	})
	defer cleanup()

	// Low ravener affinity -> apex excluded, hollow-bones present, flight blocked (no prereq)
	pool := GetGraphPool(map[string]int{}, map[string]float64{"ravener": 1}, nil)
	if contains(pool, "apex") {
		t.Fatal("apex should be gated out at low affinity")
	}
	if !contains(pool, "hollow-bones") {
		t.Fatal("universal hollow-bones should always be eligible")
	}
	if contains(pool, "flight") {
		t.Fatal("flight should be blocked without its prerequisite")
	}

	// High ravener affinity -> apex now eligible
	pool2 := GetGraphPool(map[string]int{}, map[string]float64{"ravener": 10}, nil)
	if !contains(pool2, "apex") {
		t.Fatal("apex should be eligible once affinity clears threshold")
	}
}

func contains(s []string, want string) bool {
	for _, v := range s {
		if v == want {
			return true
		}
	}
	return false
}
