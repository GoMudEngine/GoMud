package crafting

import "testing"

// craft <name> resolution must be deterministic and tiered. The old
// FindRecipeByName substring pass iterated the allRecipes MAP, so
// `craft cloak` non-deterministically returned either matching recipe
// (2026-08-03 verification). FindRecipesByName returns every match from
// the highest-priority tier that has any, in a stable order (tightest
// name first, then alphabetical), and FindRecipeByName is its
// deterministic first-pick wrapper.

func seedFindFixture(t *testing.T) func() {
	t.Helper()
	saved := allRecipes
	allRecipes = map[string]*RecipeSpec{}
	for _, r := range []*RecipeSpec{
		{RecipeId: "cattail-down-cloak", Name: "cattail-down-cloak", Skill: "tailoring"},
		{RecipeId: "wool-cloak", Name: "wool-cloak", Skill: "tailoring"},
		{RecipeId: "iron-ingot", Name: "smelt-iron", Skill: "blacksmithing",
			Aliases: []string{"ingot"}},
		{RecipeId: "leather-vest", Name: "leather-vest", Skill: "tailoring"},
	} {
		RegisterRecipeForTest(r)
	}
	return func() { allRecipes = saved; recipeByOutputId = nil }
}

func TestFindRecipesByName_ExactBeatsSubstring(t *testing.T) {
	defer seedFindFixture(t)()
	got := FindRecipesByName("wool-cloak")
	if len(got) != 1 || got[0].RecipeId != "wool-cloak" {
		t.Fatalf("exact match must return only wool-cloak, got %v", names(got))
	}
}

func TestFindRecipesByName_SubstringReturnsAllSorted(t *testing.T) {
	defer seedFindFixture(t)()
	got := FindRecipesByName("cloak")
	if len(got) != 2 {
		t.Fatalf("want both cloak recipes, got %v", names(got))
	}
	// Tightest name (shortest containing the query) first, so the wrapper's
	// single pick is the closest match; alphabetical breaks ties.
	if got[0].RecipeId != "wool-cloak" || got[1].RecipeId != "cattail-down-cloak" {
		t.Errorf("want [wool-cloak cattail-down-cloak], got %v", names(got))
	}
}

func TestFindRecipesByName_Deterministic(t *testing.T) {
	defer seedFindFixture(t)()
	first := names(FindRecipesByName("cloak"))
	for i := 0; i < 20; i++ {
		if again := names(FindRecipesByName("cloak")); !equalStrings(first, again) {
			t.Fatalf("iteration %d returned a different order: %v vs %v", i, again, first)
		}
	}
	// And the single-pick wrapper follows the same order.
	for i := 0; i < 20; i++ {
		if r := FindRecipeByName("cloak"); r == nil || r.RecipeId != "wool-cloak" {
			t.Fatalf("FindRecipeByName must deterministically pick wool-cloak")
		}
	}
}

func TestFindRecipesByName_AliasTier(t *testing.T) {
	defer seedFindFixture(t)()
	got := FindRecipesByName("ingot")
	if len(got) != 1 || got[0].RecipeId != "iron-ingot" {
		t.Fatalf("alias must resolve smelt-iron, got %v", names(got))
	}
}

func TestFindRecipesByName_NoMatch(t *testing.T) {
	defer seedFindFixture(t)()
	if got := FindRecipesByName("zzz"); len(got) != 0 {
		t.Fatalf("want no matches, got %v", names(got))
	}
	if FindRecipeByName("zzz") != nil {
		t.Fatal("wrapper must return nil on no match")
	}
}

func names(rs []*RecipeSpec) []string {
	out := make([]string, 0, len(rs))
	for _, r := range rs {
		out = append(out, r.RecipeId)
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
