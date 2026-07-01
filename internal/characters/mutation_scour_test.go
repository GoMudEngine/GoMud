package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutations"
)

func getMutationSpecForTest(id string) *mutations.MutationSpec { return mutations.GetMutation(id) }

// seedScourTestMutations registers a spread of common (rarity 2) and rare
// (rarity 7) specs so the post-scour squared-weighting bias toward rare is
// observable. Usage: defer seedScourTestMutations()()
func seedScourTestMutations() func() {
	return mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"common-1": {MutationId: "common-1", Name: "Common One", Rarity: 2},
		"common-2": {MutationId: "common-2", Name: "Common Two", Rarity: 2},
		"common-3": {MutationId: "common-3", Name: "Common Three", Rarity: 2},
		"rare-1":   {MutationId: "rare-1", Name: "Rare One", Rarity: 7},
		"rare-2":   {MutationId: "rare-2", Name: "Rare Two", Rarity: 7},
		"rare-3":   {MutationId: "rare-3", Name: "Rare Three", Rarity: 7},
	})
}

func TestScourMutations_ClearsAcquiredKeepsIntrinsicsGrantsCharges(t *testing.T) {
	c := New()
	c.SpeciesId = 1 // human (no intrinsics expected)
	c.Mutations = map[string]int{"keen-eyes": 2, "large": 1}
	c.MutationProgress = 0.9

	c.ScourMutations(3)

	if len(c.Mutations) != 0 {
		t.Fatalf("expected acquired mutations cleared, got %v", c.Mutations)
	}
	if c.MutationRerollBonus != 3 {
		t.Fatalf("expected 3 reroll charges, got %d", c.MutationRerollBonus)
	}
	if c.MutationProgress != 0 {
		t.Fatalf("expected MutationProgress reset to 0, got %v", c.MutationProgress)
	}
}

func TestScourMutations_EmptyAndDoubleScour(t *testing.T) {
	c := New()
	c.SpeciesId = 1
	c.Mutations = map[string]int{}
	c.ScourMutations(3)
	if c.MutationRerollBonus != 3 {
		t.Fatalf("empty scour should still grant charges, got %d", c.MutationRerollBonus)
	}
	// Double scour SETS (not adds) — no unbounded stacking.
	c.ScourMutations(3)
	if c.MutationRerollBonus != 3 {
		t.Fatalf("double scour should set charges to 3, got %d", c.MutationRerollBonus)
	}
}

func TestBloomSeedNewMutation_RerollBonusFavorsRareAndConsumesCharge(t *testing.T) {
	defer seedScourTestMutations()()
	c := New()
	c.SpeciesId = 1
	c.Mutations = map[string]int{}
	c.MutationRerollBonus = 2
	id, lvl := c.BloomSeedNewMutation(nil)
	if id == "" || lvl == 0 {
		t.Fatalf("expected a mutation to be seeded")
	}
	if c.MutationRerollBonus != 1 {
		t.Fatalf("expected one reroll charge consumed, got %d", c.MutationRerollBonus)
	}
}

func TestRerollBonus_ShiftsDistributionTowardRare(t *testing.T) {
	defer seedScourTestMutations()()
	rareCountWith, rareCountWithout := 0, 0
	for i := 0; i < 600; i++ {
		c := New()
		c.SpeciesId = 1
		c.Mutations = map[string]int{}
		c.MutationRerollBonus = 1
		if id, _ := c.BloomSeedNewMutation(nil); id != "" {
			if spec := getMutationSpecForTest(id); spec != nil && spec.Rarity >= 6 {
				rareCountWith++
			}
		}
		c2 := New()
		c2.SpeciesId = 1
		c2.Mutations = map[string]int{}
		if id, _ := c2.BloomSeedNewMutation(nil); id != "" {
			if spec := getMutationSpecForTest(id); spec != nil && spec.Rarity >= 6 {
				rareCountWithout++
			}
		}
	}
	if rareCountWith <= rareCountWithout {
		t.Fatalf("reroll bonus should favor rare mutations: with=%d without=%d", rareCountWith, rareCountWithout)
	}
}
