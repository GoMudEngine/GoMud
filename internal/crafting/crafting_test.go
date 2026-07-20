package crafting

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// seedRegistry populates the in-memory registry for tests without disk access.
func seedRegistry() {
	allRecipes = map[string]*RecipeSpec{
		"iron-dagger": {
			RecipeId:     "iron-dagger",
			Name:         "Iron Dagger",
			Skill:        "blacksmithing",
			SkillMinimum: 0,
			Station:      "forge",
			TimeRounds:   3,
			Ingredients: []RecipeIngredient{
				{ItemTag: "iron-ingot", Quantity: 1},
				{ItemTag: "leather-strip", Quantity: 1},
			},
			Output:         RecipeOutput{ItemId: 10005, Quantity: 1},
			SuccessMessage: "You hammer the iron into shape.",
			FailureMessage: "The iron cracks.",
		},
		"iron-buckler": {
			RecipeId:     "iron-buckler",
			Name:         "Iron Buckler",
			Skill:        "blacksmithing",
			SkillMinimum: 5,
			Station:      "forge",
			TimeRounds:   4,
			Ingredients: []RecipeIngredient{
				{ItemTag: "iron-ingot", Quantity: 2},
				{ItemTag: "wooden-plank", Quantity: 1},
			},
			Output:         RecipeOutput{ItemId: 20010, Quantity: 1},
			SuccessMessage: "You beat the iron into a shield.",
			FailureMessage: "The iron warps badly.",
		},
		"healing-poultice": {
			RecipeId:     "healing-poultice",
			Name:         "Healing Poultice",
			Skill:        "alchemy",
			SkillMinimum: 1,
			Station:      "alchemy_bench",
			TimeRounds:   2,
			Ingredients: []RecipeIngredient{
				{ItemTag: "healers-root", Quantity: 2},
				{ItemTag: "cloth-strip", Quantity: 1},
			},
			Output:         RecipeOutput{ItemId: 30010, Quantity: 1},
			SuccessMessage: "You grind the roots into cloth.",
			FailureMessage: "The mixture crumbles.",
		},
		"healing-salve": {
			RecipeId:     "healing-salve",
			Name:         "Healing Salve",
			Skill:        "alchemy",
			SkillMinimum: 0,
			Station:      "alchemy_bench",
			TimeRounds:   3,
			Ingredients: []RecipeIngredient{
				{ItemTag: "healers-root", Quantity: 2},
				{ItemTag: "bottle", Quantity: 1},
			},
			Output:         RecipeOutput{ItemId: 30036, Quantity: 1},
			SuccessMessage: "You seal the salve in the bottle.",
			FailureMessage: "The mixture separates.",
		},
	}
}

// makeItem creates a test item with a given ComponentTag, without touching the global registry.
func makeItem(tag string) items.Item {
	return items.Item{Spec: &items.ItemSpec{ComponentTag: tag}}
}

// ─── GetRecipeByOutputItemId ─────────────────────────────────────────────────

func TestGetRecipeByOutputItemId(t *testing.T) {
	seedRegistry()
	recipeByOutputId = nil // force rebuild

	// Should find iron dagger recipe by its output item ID
	recipe := GetRecipeByOutputItemId(10005)
	if recipe == nil {
		t.Fatal("expected to find recipe for item 10005")
	}
	if recipe.RecipeId != "iron-dagger" {
		t.Errorf("expected iron-dagger, got %s", recipe.RecipeId)
	}

	// Should return nil for unknown item ID
	recipe = GetRecipeByOutputItemId(999999)
	if recipe != nil {
		t.Errorf("expected nil for unknown item ID, got %s", recipe.RecipeId)
	}
}

// ─── CalcSuccessChance ────────────────────────────────────────────────────────

func TestCalcSuccessChance(t *testing.T) {
	tests := []struct {
		skillLevel   int
		skillMinimum int
		want         int
	}{
		{0, 0, 50},  // skill == min → 50%
		{9, 0, 95},  // min+9 → 50+45 = 95% (cap)
		{10, 0, 95}, // above cap → still 95%
		{-10, 0, 5}, // far below min → 5% floor
		{-2, 0, 40}, // below min → 50-10 = 40%
		{5, 5, 50},  // skill == min (non-zero) → 50%
		{6, 5, 55},  // one above min → 55%
	}
	for _, tt := range tests {
		got := CalcSuccessChance(tt.skillLevel, tt.skillMinimum)
		if got != tt.want {
			t.Errorf("CalcSuccessChance(%d, %d) = %d, want %d",
				tt.skillLevel, tt.skillMinimum, got, tt.want)
		}
	}
}

// ─── HasIngredients ───────────────────────────────────────────────────────────

func TestHasIngredients(t *testing.T) {
	seedRegistry()
	recipe := allRecipes["iron-dagger"] // needs 1x iron-ingot + 1x leather-strip

	// Empty inventory → missing iron-ingot
	ok, missing := HasIngredients([]items.Item{}, []items.Item{}, recipe)
	if ok {
		t.Error("empty inv: expected false")
	}
	if missing == "" {
		t.Error("empty inv: expected non-empty missing tag")
	}

	// Has iron-ingot but not leather-strip
	inv := []items.Item{makeItem("iron-ingot")}
	ok, missing = HasIngredients(inv, []items.Item{}, recipe)
	if ok {
		t.Error("missing leather-strip: expected false")
	}
	if missing != "leather-strip" {
		t.Errorf("missing tag: want leather-strip, got %q", missing)
	}

	// Has both → success
	inv = []items.Item{makeItem("iron-ingot"), makeItem("leather-strip")}
	ok, missing = HasIngredients(inv, []items.Item{}, recipe)
	if !ok {
		t.Errorf("exact match: expected true, missing=%q", missing)
	}

	// Short count for iron-buckler (needs 2x iron-ingot)
	buckler := allRecipes["iron-buckler"]
	inv = []items.Item{makeItem("iron-ingot"), makeItem("wooden-plank")}
	ok, missing = HasIngredients(inv, []items.Item{}, buckler)
	if ok {
		t.Error("only 1 iron-ingot for buckler: expected false")
	}
	if missing != "iron-ingot" {
		t.Errorf("missing tag: want iron-ingot, got %q", missing)
	}

	// Exactly 2x iron-ingot + 1x wooden-plank → success
	inv = []items.Item{makeItem("iron-ingot"), makeItem("iron-ingot"), makeItem("wooden-plank")}
	ok, _ = HasIngredients(inv, []items.Item{}, buckler)
	if !ok {
		t.Error("2x iron-ingot + wooden-plank: expected true")
	}
}

// ─── ConsumeIngredients ───────────────────────────────────────────────────────

func TestConsumeIngredients(t *testing.T) {
	seedRegistry()
	recipe := allRecipes["iron-dagger"] // consumes 1x iron-ingot + 1x leather-strip

	extra := makeItem("other-item")
	inv := []items.Item{
		makeItem("iron-ingot"),
		makeItem("leather-strip"),
		extra,
		makeItem("iron-ingot"), // extra iron-ingot should NOT be consumed
	}

	result, _ := ConsumeIngredients(inv, []items.Item{}, recipe)

	if len(result) != 2 {
		t.Errorf("expected 2 items remaining, got %d", len(result))
	}

	// The extra iron-ingot and the other-item should remain
	tagCount := make(map[string]int)
	for _, it := range result {
		tagCount[it.GetSpec().ComponentTag]++
	}
	if tagCount["iron-ingot"] != 1 {
		t.Errorf("expected 1 remaining iron-ingot, got %d", tagCount["iron-ingot"])
	}
	if tagCount["other-item"] != 1 {
		t.Errorf("expected 1 remaining other-item, got %d", tagCount["other-item"])
	}
	if tagCount["leather-strip"] != 0 {
		t.Errorf("leather-strip should have been consumed, got %d", tagCount["leather-strip"])
	}
}

// ─── FindRecipeByName ─────────────────────────────────────────────────────────

func TestFindRecipeByName(t *testing.T) {
	seedRegistry()

	// Exact match
	r := FindRecipeByName("Iron Dagger")
	if r == nil || r.RecipeId != "iron-dagger" {
		t.Error("exact name: expected iron-dagger")
	}

	// Case-insensitive
	r = FindRecipeByName("iron dagger")
	if r == nil || r.RecipeId != "iron-dagger" {
		t.Error("lowercase: expected iron-dagger")
	}

	// Partial substring
	r = FindRecipeByName("poultice")
	if r == nil || r.RecipeId != "healing-poultice" {
		t.Error("partial 'poultice': expected healing-poultice")
	}

	// No match
	r = FindRecipeByName("nonexistent recipe xyz")
	if r != nil {
		t.Error("no match: expected nil")
	}
}

// ─── GetStarterRecipes ───────────────────────────────────────────────────────

func TestGetStarterRecipes(t *testing.T) {
	seedRegistry()

	starters := GetStarterRecipes()

	// iron-dagger (SkillMinimum=0) should be a starter
	if _, ok := starters["iron-dagger"]; !ok {
		t.Error("expected iron-dagger in starter recipes")
	}
	// healing-poultice (SkillMinimum=1) should NOT be a starter
	if _, ok := starters["healing-poultice"]; ok {
		t.Error("healing-poultice should not be a starter recipe (SkillMinimum=1)")
	}
	// iron-buckler (SkillMinimum=5) should NOT be a starter
	if _, ok := starters["iron-buckler"]; ok {
		t.Error("iron-buckler should not be a starter recipe (SkillMinimum=5)")
	}
	// All values should be 1
	for id, v := range starters {
		if v != 1 {
			t.Errorf("starter %q value = %d, want 1", id, v)
		}
	}
}

// ─── GetEligibleRecipes ──────────────────────────────────────────────────────

func TestGetEligibleRecipes(t *testing.T) {
	seedRegistry()

	// Player knows iron-dagger and healing-poultice, has blacksmithing=5, alchemy=0
	known := map[string]int{
		"iron-dagger":      1,
		"healing-poultice": 1,
	}
	skills := map[string]int{
		"blacksmithing": 5,
		"alchemy":       0,
	}

	eligible := GetEligibleRecipes(known, skills, "blacksmithing")

	// iron-buckler (SkillMinimum=5, blacksmithing) should be eligible
	found := false
	for _, id := range eligible {
		if id == "iron-buckler" {
			found = true
		}
		// Known recipes should never appear
		if id == "iron-dagger" || id == "healing-poultice" {
			t.Errorf("known recipe %q should not be eligible", id)
		}
	}
	if !found {
		t.Error("expected iron-buckler to be eligible")
	}

	// If skill is too low, nothing should be eligible
	skills["blacksmithing"] = 3
	eligible = GetEligibleRecipes(known, skills, "blacksmithing")
	for _, id := range eligible {
		if id == "iron-buckler" {
			t.Error("iron-buckler should not be eligible with blacksmithing=3")
		}
	}

	// Recipes from a different skill should never appear when filtering by blacksmithing
	skills["blacksmithing"] = 5
	eligible = GetEligibleRecipes(known, skills, "blacksmithing")
	for _, id := range eligible {
		if r := GetRecipe(id); r != nil && r.Skill != "blacksmithing" {
			t.Errorf("recipe %q (skill=%s) should not appear when filtering for blacksmithing", id, r.Skill)
		}
	}

	// If everything is known, nothing eligible
	allKnown := map[string]int{
		"iron-dagger":      1,
		"healing-poultice": 1,
		"iron-buckler":     1,
	}
	eligible = GetEligibleRecipes(allKnown, map[string]int{"blacksmithing": 50, "alchemy": 50}, "blacksmithing")
	if len(eligible) != 0 {
		t.Errorf("all known: expected 0 eligible, got %d", len(eligible))
	}
}
