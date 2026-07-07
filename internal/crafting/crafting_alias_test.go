package crafting

import "testing"

// TestRecipeAliasResolution verifies that FindRecipeByName resolves a recipe by
// one of its declared short aliases, and that the alias pass sits between the
// exact-Name pass and the loose substring pass.
func TestRecipeAliasResolution(t *testing.T) {
	allRecipes = map[string]*RecipeSpec{
		"anti-corrosion-quench": {
			RecipeId: "anti-corrosion-quench",
			Name:     "Anti-Corrosion Quench",
			Skill:    "blacksmithing",
			Aliases:  []string{"quench"},
			Output:   RecipeOutput{ItemId: 10001, Quantity: 1},
		},
	}

	if got := FindRecipeByName("quench"); got == nil || got.RecipeId != "anti-corrosion-quench" {
		t.Fatalf("FindRecipeByName(\"quench\") = %v, want recipe anti-corrosion-quench", got)
	}

	if got := FindRecipeByName("nope"); got != nil {
		t.Fatalf("FindRecipeByName(\"nope\") = %v, want nil", got)
	}
}

// TestRecipeAliasBeatsSubstring verifies the alias pass runs before the loose
// substring pass: an exact alias on one recipe wins over a substring hit on
// another recipe's Name.
func TestRecipeAliasBeatsSubstring(t *testing.T) {
	allRecipes = map[string]*RecipeSpec{
		"anti-corrosion-quench": {
			RecipeId: "anti-corrosion-quench",
			Name:     "Anti-Corrosion Quench",
			Skill:    "blacksmithing",
			Aliases:  []string{"quench"},
			Output:   RecipeOutput{ItemId: 10001, Quantity: 1},
		},
		"substring-holder": {
			RecipeId: "substring-holder",
			Name:     "Some quench-adjacent recipe", // contains "quench" as substring
			Skill:    "blacksmithing",
			Output:   RecipeOutput{ItemId: 10002, Quantity: 1},
		},
	}

	got := FindRecipeByName("quench")
	if got == nil || got.RecipeId != "anti-corrosion-quench" {
		t.Fatalf("FindRecipeByName(\"quench\") = %v, want exact-alias recipe anti-corrosion-quench (alias must beat substring)", got)
	}
}
