package characters

import (
	"math/rand"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// seedBloomTestMutations registers two minimal MutationSpec entries for the
// duration of a single test. Both specs have no RequiresBodyParts so they
// pass CanApplyTo for any species (including nil — fail-open). Rarity 5 and 7
// are in the mid/high tier that Bloom's weighting favours.
// Usage: defer seedBloomTestMutations()()
func seedBloomTestMutations() func() {
	return mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"mut-a": {MutationId: "mut-a", Name: "Alpha Twitch", Rarity: 5},
		"mut-b": {MutationId: "mut-b", Name: "Beta Clench", Rarity: 7},
	})
}

// TestBloomAdvance_StrongestUnderCap verifies that BloomAdvanceMutation
// advances the mutation with the highest current level when that level is
// below the global cap.
//
// Setup: default MutationMaxLevel = 3 (config validation default when no
// config file is loaded). Character owns "mut-a"@1 and "mut-b"@2. "mut-b"
// has the higher level and is below cap → it must be advanced to 3; "mut-a"
// must remain at 1.
func TestBloomAdvance_StrongestUnderCap(t *testing.T) {
	defer seedBloomTestMutations()()

	c := &Character{
		Mutations: map[string]int{
			"mut-a": 1,
			"mut-b": 2,
		},
	}
	rng := rand.New(rand.NewSource(42))
	id, newLevel := c.BloomAdvanceMutation(rng)

	if id != "mut-b" {
		t.Errorf("expected strongest mutation %q to be advanced, got %q", "mut-b", id)
	}
	if newLevel != 3 {
		t.Errorf("expected new level 3, got %d", newLevel)
	}
	if c.Mutations["mut-b"] != 3 {
		t.Errorf("expected c.Mutations[mut-b] == 3, got %d", c.Mutations["mut-b"])
	}
	// Weaker mutation must be untouched.
	if c.Mutations["mut-a"] != 1 {
		t.Errorf("expected mut-a to remain at 1, got %d", c.Mutations["mut-a"])
	}
}

// TestBloomSeed_WhenNoMutations verifies that a character with no existing
// mutations receives a freshly seeded mutation at level 1 after a single
// call to BloomAdvanceMutation.
func TestBloomSeed_WhenNoMutations(t *testing.T) {
	defer seedBloomTestMutations()()

	c := &Character{} // Mutations is nil
	rng := rand.New(rand.NewSource(99))
	id, newLevel := c.BloomAdvanceMutation(rng)

	if id == "" {
		t.Fatal("expected a mutation to be seeded, got empty id")
	}
	if newLevel != 1 {
		t.Errorf("expected seeded mutation level 1, got %d", newLevel)
	}
	if len(c.Mutations) != 1 {
		t.Errorf("expected exactly 1 mutation in map, got %d", len(c.Mutations))
	}
	if c.Mutations[id] != 1 {
		t.Errorf("c.Mutations[%q] = %d, want 1", id, c.Mutations[id])
	}
	if id != "mut-a" && id != "mut-b" {
		t.Errorf("seeded id %q is not one of the known test mutations", id)
	}
}

// TestBloomAdvance_RespectsCap verifies that a mutation already at the
// global maximum level is never advanced beyond it. When all owned mutations
// are capped AND the registry contains no unowned candidates,
// BloomAdvanceMutation must return ("", 0) and leave the map unchanged.
func TestBloomAdvance_RespectsCap(t *testing.T) {
	// Registry contains only "mut-a"; character already owns it at cap (3).
	defer mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"mut-a": {MutationId: "mut-a", Name: "Alpha Twitch", Rarity: 5},
	})()

	capLevel := 3 // default MutationMaxLevel when no config file is loaded
	c := &Character{
		Mutations: map[string]int{
			"mut-a": capLevel,
		},
	}
	rng := rand.New(rand.NewSource(1))
	id, newLevel := c.BloomAdvanceMutation(rng)

	if id != "" || newLevel != 0 {
		t.Errorf(
			"expected ('', 0) — all mutations capped, no new candidates; got (%q, %d)",
			id, newLevel,
		)
	}
	if c.Mutations["mut-a"] != capLevel {
		t.Errorf("mut-a was modified beyond cap: got %d, want %d", c.Mutations["mut-a"], capLevel)
	}
}
