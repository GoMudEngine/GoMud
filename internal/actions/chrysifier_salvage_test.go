package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

func TestSalvageChanceWithMutations_ProvidentHands(t *testing.T) {
	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"salv-mut": {MutationId: "salv-mut", Name: "Salv", Rarity: 3,
			Pros: []mutations.MutationEffect{{Type: "salvage_yield_bonus", Value: 0.30}}},
	})
	defer cleanup()

	c := characters.New()
	// No mutation → unchanged.
	if got := salvageChanceWithMutations(c, 0.5); got != 0.5 {
		t.Fatalf("no mutation: got %v, want 0.5", got)
	}
	// With mutation → boosted (0.5 * 1.30 = 0.65).
	c.Mutations = map[string]int{"salv-mut": 1}
	if got := salvageChanceWithMutations(c, 0.5); got <= 0.5 {
		t.Fatalf("with mutation: got %v, want > 0.5", got)
	}
	// Cap at 1.0.
	if got := salvageChanceWithMutations(c, 0.9); got > 1.0 {
		t.Fatalf("cap: got %v, want <= 1.0", got)
	}
}
