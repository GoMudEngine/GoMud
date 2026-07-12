package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

func TestForagedSearchScore_ProvidentHandsBoost(t *testing.T) {
	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"forager-mut": {MutationId: "forager-mut", Name: "Forager", Rarity: 3,
			Pros: []mutations.MutationEffect{{Type: "forage_yield_multiplier", Value: 0.25}}},
	})
	defer cleanup()

	c := characters.New()
	base := CalcSearchScore(c)
	c.Mutations = map[string]int{"forager-mut": 1}
	boosted := foragedSearchScore(c)

	if boosted <= base {
		t.Fatalf("forage-yield mutation should raise search score: base=%v boosted=%v", base, boosted)
	}
}
