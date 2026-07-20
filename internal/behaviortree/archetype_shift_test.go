package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

func TestArchetypeForSpec(t *testing.T) {
	cases := []struct {
		clusters []string
		want     string
	}{
		{[]string{"colossus"}, "tank_taunter"},
		{[]string{"ironhide"}, "tank_taunter"},
		{[]string{"ravener"}, "predator"},
		{[]string{"stalker"}, "ambusher"},
		{[]string{"ethereal"}, "pure_caster"},
		{[]string{"manifester"}, "defensive_caster"},
		{[]string{"zealot"}, "defensive_caster"},
		{[]string{"weaver"}, ""},
		{[]string{"trickster"}, ""},
		{[]string{"chrysifier"}, ""},
		{nil, ""},
		{[]string{"ironhide", "zealot"}, "tank_taunter"}, // bridge: first mapped cluster wins
	}
	for _, c := range cases {
		spec := &mutations.MutationSpec{Clusters: c.clusters}
		if got := archetypeForSpec(spec); got != c.want {
			t.Errorf("archetypeForSpec(%v) = %q, want %q", c.clusters, got, c.want)
		}
	}
}

// --- ReevaluateArchetypeShift ---
// Reuses buildMutationMob (actions_mutation_test.go) and
// seedMutationTestRoom/mutTestRoomId (actions_mutation_at_target_test.go).

func seedShiftSpecs(t *testing.T) {
	t.Helper()
	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"brawn": {MutationId: "brawn", Rarity: 9, Clusters: []string{"colossus"}}, // -> tank_taunter
		"fangs": {MutationId: "fangs", Rarity: 5, Clusters: []string{"ravener"}},  // -> predator
		"aura":  {MutationId: "aura", Rarity: 5, Clusters: []string{"zealot"}},    // -> defensive_caster
		"zeal":  {MutationId: "zeal", Rarity: 9, Clusters: []string{"colossus"}},  // -> tank_taunter
		"plain": {MutationId: "plain", Rarity: 10},                                // zero-cluster: no pull, rarity must NOT matter
	})
	t.Cleanup(cleanup)
}

func TestReevaluateArchetypeShift_IneligibleArchetypeNeverShifts(t *testing.T) {
	seedShiftSpecs(t)
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9401, 99901, mutTestRoomId)
	mob.BehaviorArchetype = "boss_soren"
	mob.Character.Mutations = map[string]int{"brawn": 1}

	ReevaluateArchetypeShift(mob)
	if mob.BehaviorArchetype != "boss_soren" {
		t.Fatalf("boss shifted to %q; specialists must never shift", mob.BehaviorArchetype)
	}
}

func TestReevaluateArchetypeShift_PerMobTreeBlocks(t *testing.T) {
	seedShiftSpecs(t)
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9402, 99902, mutTestRoomId)
	mob.BehaviorArchetype = "generic_fighter"
	mob.Character.Mutations = map[string]int{"fangs": 1}

	// Install a per-mob tree — it shadows archetypes, so no shift.
	node, err := LoadTreeFromBytes([]byte("tree:\n  type: selector\n  children:\n    - type: action\n      event: mob_idle\n      do: attack\n"))
	if err != nil {
		t.Fatalf("LoadTreeFromBytes: %v", err)
	}
	t.Cleanup(GetEngine().SetMobTreeForTest(99902, node))

	ReevaluateArchetypeShift(mob)
	if mob.BehaviorArchetype != "generic_fighter" {
		t.Fatalf("per-mob-tree mob shifted to %q; shadowed mobs must not shift", mob.BehaviorArchetype)
	}
}

func TestReevaluateArchetypeShift_RarestPullWins(t *testing.T) {
	seedShiftSpecs(t)
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9403, 99903, mutTestRoomId)
	mob.BehaviorArchetype = "prey"
	// aura (r5, combat_passive) sorts FIRST alphabetically; zeal (r9,
	// generic_fighter) is rarest with a pull; plain (r10, NO pull). An
	// alphabetical-min implementation would wrongly pick combat_passive.
	mob.Character.Mutations = map[string]int{"aura": 1, "zeal": 1, "plain": 1}

	ReevaluateArchetypeShift(mob)
	if mob.BehaviorArchetype != "tank_taunter" {
		t.Fatalf("got %q, want tank_taunter (rarest PULL wins; pull-less rarity ignored)", mob.BehaviorArchetype)
	}
}

func TestStrongestArchetypePull_TiebreakDeterministic(t *testing.T) {
	seedShiftSpecs(t)
	owned := map[string]int{"fangs": 1, "aura": 1} // both r5
	for i := 0; i < 50; i++ {
		if got := strongestArchetypePull(owned); got != "defensive_caster" {
			t.Fatalf("iteration %d: got %q, want defensive_caster (alphabetical tiebreak must be deterministic)", i, got)
		}
	}
}

func TestReevaluateArchetypeShift_RarityTieAlphabeticalKey(t *testing.T) {
	seedShiftSpecs(t)
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9404, 99904, mutTestRoomId)
	mob.BehaviorArchetype = "generic_fighter"
	// aura + fangs both r5 → "aura" < "fangs" alphabetically → defensive_caster.
	mob.Character.Mutations = map[string]int{"fangs": 1, "aura": 1}

	ReevaluateArchetypeShift(mob)
	if mob.BehaviorArchetype != "defensive_caster" {
		t.Fatalf("got %q, want defensive_caster (alphabetical tiebreak)", mob.BehaviorArchetype)
	}
}

func TestReevaluateArchetypeShift_SameTargetNoOp(t *testing.T) {
	seedShiftSpecs(t)
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9405, 99905, mutTestRoomId)
	mob.BehaviorArchetype = "predator"
	mob.Character.Mutations = map[string]int{"fangs": 1}
	sentinel := NewBehaviorState()
	mob.BTreeState = sentinel

	ReevaluateArchetypeShift(mob)
	if mob.BTreeState != sentinel {
		t.Fatal("same-target shift must be a silent no-op; BTreeState was reset")
	}
}

func TestReevaluateArchetypeShift_SwapResetsStateAndPolicies(t *testing.T) {
	seedShiftSpecs(t)
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9406, 99906, mutTestRoomId)
	mob.BehaviorArchetype = "prey"
	mob.Character.Mutations = map[string]int{"brawn": 1}
	mob.BTreeState = NewBehaviorState()

	ReevaluateArchetypeShift(mob)
	if mob.BehaviorArchetype != "tank_taunter" {
		t.Fatalf("got %q, want tank_taunter", mob.BehaviorArchetype)
	}
	if mob.BTreeState != nil {
		t.Fatal("BTreeState must be reset to nil on shift")
	}
	wantSub := characters.DefaultSubmissionPolicyForArchetype("tank_taunter")
	if mob.Character.SubmissionPolicy != wantSub {
		t.Errorf("SubmissionPolicy = %v, want re-derived %v", mob.Character.SubmissionPolicy, wantSub)
	}
	wantSur := characters.DefaultSurrenderPolicyForArchetype("tank_taunter")
	if mob.Character.SurrenderPolicy != wantSur {
		t.Errorf("SurrenderPolicy = %v, want re-derived %v", mob.Character.SurrenderPolicy, wantSur)
	}
}

func TestReevaluateArchetypeShift_AuthoredPolicyPreserved(t *testing.T) {
	seedShiftSpecs(t)
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9407, 99907, mutTestRoomId)
	mob.BehaviorArchetype = "prey"
	mob.Character.Mutations = map[string]int{"brawn": 1}
	// Authored YAML override (any non-empty value engages the guard).
	mob.SubmissionPolicy = "authored-value"
	prior := mob.Character.SubmissionPolicy

	ReevaluateArchetypeShift(mob)
	if mob.Character.SubmissionPolicy != prior {
		t.Error("authored submission_policy must not be re-derived on shift")
	}
}

func TestReevaluateArchetypeShift_ArchetypeLessMobGainsArchetype(t *testing.T) {
	seedShiftSpecs(t)
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9408, 99908, mutTestRoomId)
	mob.BehaviorArchetype = "" // archetype-less — in the FROM set
	mob.Character.Mutations = map[string]int{"fangs": 1}
	mob.BTreeState = NewBehaviorState()

	ReevaluateArchetypeShift(mob)
	if mob.BehaviorArchetype != "predator" {
		t.Fatalf("got %q, want predator (archetype-less mobs gain an archetype)", mob.BehaviorArchetype)
	}
	if mob.BTreeState != nil {
		t.Fatal("BTreeState must be reset to nil on shift")
	}
	wantSub := characters.DefaultSubmissionPolicyForArchetype("predator")
	if mob.Character.SubmissionPolicy != wantSub {
		t.Errorf("SubmissionPolicy = %v, want re-derived %v", mob.Character.SubmissionPolicy, wantSub)
	}
	wantSur := characters.DefaultSurrenderPolicyForArchetype("predator")
	if mob.Character.SurrenderPolicy != wantSur {
		t.Errorf("SurrenderPolicy = %v, want re-derived %v", mob.Character.SurrenderPolicy, wantSur)
	}
}
