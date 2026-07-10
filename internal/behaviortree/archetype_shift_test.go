package behaviortree

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

func TestValidateArchetypePullsCore(t *testing.T) {
	exists := func(path string) bool {
		return strings.Contains(path, "generic_fighter") || strings.Contains(path, "predator")
	}

	// Valid pulls: whitelisted + file exists.
	good := []*mutations.MutationSpec{
		{MutationId: "no-pull", Rarity: 3},
		{MutationId: "brawn", Rarity: 9, ArchetypePull: "generic_fighter"},
		{MutationId: "fangs", Rarity: 5, ArchetypePull: "predator"},
	}
	if err := validateArchetypePulls(good, exists); err != nil {
		t.Fatalf("valid pulls: unexpected error %v", err)
	}

	// Non-whitelisted target (boss archetype).
	bad := []*mutations.MutationSpec{
		{MutationId: "hubris", Rarity: 9, ArchetypePull: "boss_soren"},
	}
	if err := validateArchetypePulls(bad, exists); err == nil {
		t.Fatal("non-whitelisted pull: expected error, got nil")
	}

	// Whitelisted but no archetype file on disk.
	missing := []*mutations.MutationSpec{
		{MutationId: "ghost", Rarity: 9, ArchetypePull: "pure_caster"},
	}
	if err := validateArchetypePulls(missing, exists); err == nil {
		t.Fatal("missing archetype file: expected error, got nil")
	}
}

// --- ReevaluateArchetypeShift ---
// Reuses buildMutationMob (actions_mutation_test.go) and
// seedMutationTestRoom/mutTestRoomId (actions_mutation_at_target_test.go).

func seedShiftSpecs(t *testing.T) {
	t.Helper()
	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"brawn": {MutationId: "brawn", Rarity: 9, ArchetypePull: "generic_fighter"},
		"fangs": {MutationId: "fangs", Rarity: 5, ArchetypePull: "predator"},
		"aura":  {MutationId: "aura", Rarity: 5, ArchetypePull: "combat_passive"},
		"plain": {MutationId: "plain", Rarity: 10}, // no pull — rarity must NOT matter
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
	// fangs (r5, predator) + brawn (r9, generic_fighter) + plain (r10, NO pull)
	mob.Character.Mutations = map[string]int{"fangs": 1, "brawn": 1, "plain": 1}

	ReevaluateArchetypeShift(mob)
	if mob.BehaviorArchetype != "generic_fighter" {
		t.Fatalf("got %q, want generic_fighter (rarest PULL wins; pull-less rarity ignored)", mob.BehaviorArchetype)
	}
}

func TestReevaluateArchetypeShift_RarityTieAlphabeticalKey(t *testing.T) {
	seedShiftSpecs(t)
	seedMutationTestRoom(t)
	mob := buildMutationMob(t, 9404, 99904, mutTestRoomId)
	mob.BehaviorArchetype = "generic_fighter"
	// aura + fangs both r5 → "aura" < "fangs" alphabetically → combat_passive.
	mob.Character.Mutations = map[string]int{"fangs": 1, "aura": 1}

	ReevaluateArchetypeShift(mob)
	if mob.BehaviorArchetype != "combat_passive" {
		t.Fatalf("got %q, want combat_passive (alphabetical tiebreak)", mob.BehaviorArchetype)
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
	if mob.BehaviorArchetype != "generic_fighter" {
		t.Fatalf("got %q, want generic_fighter", mob.BehaviorArchetype)
	}
	if mob.BTreeState != nil {
		t.Fatal("BTreeState must be reset to nil on shift")
	}
	wantSub := characters.DefaultSubmissionPolicyForArchetype("generic_fighter")
	if mob.Character.SubmissionPolicy != wantSub {
		t.Errorf("SubmissionPolicy = %v, want re-derived %v", mob.Character.SubmissionPolicy, wantSub)
	}
	wantSur := characters.DefaultSurrenderPolicyForArchetype("generic_fighter")
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
