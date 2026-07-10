package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// TestRecalculateStats_BodyPoleShrinksConviction verifies that deep Body-pole
// mutation commitment shrinks ConvictionMax via the graph opposition. Uses the
// same character construction as the sibling RecalculateStats pool tests;
// ConvictionMax derives above the floor (ConvictionBase/PerWilCha default to
// 5/2), so shrinkage is observable.
func TestRecalculateStats_BodyPoleShrinksConviction(t *testing.T) {
	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"brute": {MutationId: "brute", Name: "Brute", Rarity: 10, Pole: "body"},
	})
	defer cleanup()

	base := &Character{
		SpeciesId: 1,
		Stats:     validStats(),
		Mutations: map[string]int{},
		Buffs:     newTestBuffs(),
	}
	// Seed a non-floored Conviction pool (config pool multipliers are unset in
	// the unit-test env, so ConvictionMax would otherwise floor to 1). Base
	// survives RecalculateStats (Value = Base + Mods), leaving room to observe decay.
	base.ConvictionMax.Base = 200
	base.RecalculateStats()
	full := base.ConvictionMax.Value

	base.Mutations = map[string]int{"brute": 4}
	base.RecalculateStats()
	shrunk := base.ConvictionMax.Value

	if !(shrunk < full) {
		t.Fatalf("deep Body should shrink ConvictionMax: full=%d shrunk=%d", full, shrunk)
	}
	if shrunk < 1 {
		t.Fatalf("ConvictionMax must stay floored at 1, got %d", shrunk)
	}
}
