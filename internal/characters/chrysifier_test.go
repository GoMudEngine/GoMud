package characters

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutations"
)

func TestCraftMaterialsSaved_Bounds(t *testing.T) {
	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"thrifty-none": {MutationId: "thrifty-none", Name: "None", Rarity: 3},
		"thrifty-full": {MutationId: "thrifty-full", Name: "Full", Rarity: 3,
			Pros: []mutations.MutationEffect{{Type: "craft_material_discount", Value: 1.0}}},
	})
	defer cleanup()

	c := New()
	for i := 0; i < 30; i++ {
		if c.CraftMaterialsSaved() {
			t.Fatal("no mutation should never save materials")
		}
	}
	c.Mutations = map[string]int{"thrifty-full": 1}
	for i := 0; i < 30; i++ {
		if !c.CraftMaterialsSaved() {
			t.Fatal("100% discount should always save materials")
		}
	}
}

func TestCraftQualityLevel_Faithwrought(t *testing.T) {
	cleanup := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"faith": {MutationId: "faith", Name: "Faith", Rarity: 6,
			Pros: []mutations.MutationEffect{{Type: "craft_quality_bonus", Value: 0.20}}},
	})
	defer cleanup()

	c := New()
	if got := c.CraftQualityLevel(50); got != 50 {
		t.Fatalf("no mutation: got %d, want 50", got)
	}
	c.Mutations = map[string]int{"faith": 1}
	if got := c.CraftQualityLevel(50); got != 60 { // 50 + round(50*0.20)
		t.Fatalf("faithwrought: got %d, want 60", got)
	}
}
