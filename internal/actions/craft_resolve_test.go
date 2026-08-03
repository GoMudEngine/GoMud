package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/crafting"
)

// resolveCraftRecipe carries the known-recipe preference for `craft <name>`:
// a query matching several recipes must prefer the one the player KNOWS, and
// only prompt for disambiguation when they know more than one (2026-08-03;
// the pre-fix behavior was a random map-order pick).

func seedCraftResolveFixture(t *testing.T) (*characters.Character, func()) {
	t.Helper()
	for _, r := range []*crafting.RecipeSpec{
		{RecipeId: "cattail-down-cloak", Name: "cattail-down-cloak", Skill: "tailoring"},
		{RecipeId: "wool-cloak", Name: "wool-cloak", Skill: "tailoring"},
	} {
		crafting.RegisterRecipeForTest(r)
	}
	char := characters.New()
	char.KnownRecipes = map[string]int{}
	return char, func() {
		crafting.UnregisterRecipeForTest("cattail-down-cloak")
		crafting.UnregisterRecipeForTest("wool-cloak")
	}
}

func TestResolveCraftRecipe_PrefersKnownOverTighterUnknown(t *testing.T) {
	char, cleanup := seedCraftResolveFixture(t)
	defer cleanup()

	// wool-cloak is the tighter name match, but the player only knows the
	// cattail one — resolution must follow what they know.
	char.KnownRecipes["cattail-down-cloak"] = 1

	recipe, ambiguous := resolveCraftRecipe(char, "cloak")
	if len(ambiguous) != 0 {
		t.Fatalf("single known match must not be ambiguous: %v", ambiguous)
	}
	if recipe == nil || recipe.RecipeId != "cattail-down-cloak" {
		t.Fatalf("want cattail-down-cloak (the known one), got %+v", recipe)
	}
}

func TestResolveCraftRecipe_AmbiguousWhenMultipleKnown(t *testing.T) {
	char, cleanup := seedCraftResolveFixture(t)
	defer cleanup()

	char.KnownRecipes["cattail-down-cloak"] = 1
	char.KnownRecipes["wool-cloak"] = 1

	recipe, ambiguous := resolveCraftRecipe(char, "cloak")
	if recipe != nil {
		t.Fatalf("two known matches must not silently pick one, got %s", recipe.RecipeId)
	}
	if len(ambiguous) != 2 || ambiguous[0] != "wool-cloak" || ambiguous[1] != "cattail-down-cloak" {
		t.Fatalf("want [wool-cloak cattail-down-cloak] (tightest first), got %v", ambiguous)
	}
}

func TestResolveCraftRecipe_UnknownFallsBackToTightest(t *testing.T) {
	char, cleanup := seedCraftResolveFixture(t)
	defer cleanup()

	// Knows neither: resolve deterministically to the tightest match so the
	// downstream known-recipe gate can produce the discovery message.
	recipe, ambiguous := resolveCraftRecipe(char, "cloak")
	if len(ambiguous) != 0 {
		t.Fatalf("unknown-only match must not prompt: %v", ambiguous)
	}
	if recipe == nil || recipe.RecipeId != "wool-cloak" {
		t.Fatalf("want the tightest candidate wool-cloak, got %+v", recipe)
	}
}

func TestResolveCraftRecipe_NoMatch(t *testing.T) {
	char, cleanup := seedCraftResolveFixture(t)
	defer cleanup()

	recipe, ambiguous := resolveCraftRecipe(char, "zzz")
	if recipe != nil || len(ambiguous) != 0 {
		t.Fatalf("want nothing, got %+v / %v", recipe, ambiguous)
	}
}
