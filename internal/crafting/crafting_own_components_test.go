package crafting

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestRequireOwnComponents(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		777701: {ItemId: 777701, Name: "hungering guard", Type: items.Object, ComponentTag: "hungering_guard", IsComponent: true},
		777702: {ItemId: 777702, Name: "iron ingot", Type: items.Object, ComponentTag: "iron-ingot", IsComponent: false},
	})()

	// "hungering_guard" must be a genuinely CRAFTABLE component tag (some
	// recipe outputs an item carrying it) for require_own_components to gate
	// on it — see isCraftableComponentTag in crafting.go. Register a recipe
	// producing 777701 so this test exercises the real gating path rather
	// than accidentally passing because the tag looks unclaimed.
	RegisterRecipeForTest(&RecipeSpec{
		RecipeId: "test-craft-hungering-guard",
		Name:     "craft hungering guard (test)",
		Skill:    "jewelcrafting",
		Output:   RecipeOutput{ItemId: 777701, Quantity: 1},
	})
	ResetCraftableComponentTagsForTest()

	recipe := &RecipeSpec{
		RecipeId: "test-assembly", Name: "test assembly", Skill: "blacksmithing", SkillMinimum: 65,
		RequireOwnComponents: true,
		Ingredients: []RecipeIngredient{
			{ItemTag: "hungering_guard", Quantity: 1},
			{ItemTag: "iron-ingot", Quantity: 1},
		},
	}

	mine := items.New(777701)
	mine.MakerName = "Megalomania"
	theirs := items.New(777701)
	theirs.MakerName = "SomeoneElse"
	unmade := items.New(777701)

	// Bulk (non-component) ingredient with a foreign MakerName is exempt —
	// only crafted components are checked.
	bulkForeign := items.New(777702)
	bulkForeign.MakerName = "SomeoneElse"

	if ok, name := CheckOwnComponents(recipe, []items.Item{mine, bulkForeign}, nil, "Megalomania"); !ok {
		t.Fatalf("own component rejected: offending=%q", name)
	}
	if ok, name := CheckOwnComponents(recipe, []items.Item{theirs, bulkForeign}, nil, "Megalomania"); ok {
		t.Fatal("foreign component accepted")
	} else if name != "hungering guard" {
		t.Errorf("offending component name = %q, want %q", name, "hungering guard")
	}
	if ok, _ := CheckOwnComponents(recipe, []items.Item{unmade, bulkForeign}, nil, "Megalomania"); ok {
		t.Fatal("maker-less component accepted")
	}

	recipe.RequireOwnComponents = false
	if ok, name := CheckOwnComponents(recipe, []items.Item{theirs, bulkForeign}, nil, "Megalomania"); !ok {
		t.Fatalf("flag off should not restrict: offending=%q", name)
	}
}

// TestRequireOwnComponents_ExemptsDropReagents reproduces the live pinnacle-
// assembly bug: a recipe with require_own_components: true that mixes a
// genuinely CRAFTED sub-assembly (e.g. reinforced-frame) with a drop/forage
// reagent (e.g. Folded-Space Silk — is_component:true for bag routing only,
// but never any recipe's output, since it's a boss drop) was rejecting the
// reagent for lacking a maker's mark it could never carry. The fix scopes
// the maker-mark check to isCraftableComponentTag, exempting reagents while
// still gating real crafted components.
func TestRequireOwnComponents_ExemptsDropReagents(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		// Crafted sub-assembly: some recipe (registered below) outputs this.
		777801: {ItemId: 777801, Name: "reinforced frame", Type: items.Object, ComponentTag: "reinforced-frame-test", IsComponent: true},
		// Drop/forage reagent: is_component for bag routing, but NO recipe
		// outputs it (mirrors the real Folded-Space Silk / Warden
		// Chassis-Loom boss-drop reagents).
		777802: {ItemId: 777802, Name: "folded-space silk", Type: items.Object, ComponentTag: "folded-space-silk-test", IsComponent: true},
	})()

	RegisterRecipeForTest(&RecipeSpec{
		RecipeId: "test-craft-reinforced-frame",
		Name:     "craft reinforced frame (test)",
		Skill:    "tailoring",
		Output:   RecipeOutput{ItemId: 777801, Quantity: 1},
	})
	ResetCraftableComponentTagsForTest()

	recipe := &RecipeSpec{
		RecipeId: "test-pinnacle-assembly", Name: "test pinnacle assembly", Skill: "tailoring", SkillMinimum: 65,
		RequireOwnComponents: true,
		Ingredients: []RecipeIngredient{
			{ItemTag: "reinforced-frame-test", Quantity: 1},
			{ItemTag: "folded-space-silk-test", Quantity: 1},
		},
	}

	const crafter = "Veyra"

	myFrame := items.New(777801)
	myFrame.MakerName = crafter
	// The reagent carries no maker's mark at all — exactly like a boss-drop
	// item straight off a corpse. It can never satisfy a maker-mark check.
	dropReagent := items.New(777802)

	// Case 1 (THE BUG REPRO): own crafted component + a maker-less drop
	// reagent must be ACCEPTED — the reagent is exempt from the check.
	if ok, name := CheckOwnComponents(recipe, []items.Item{myFrame, dropReagent}, nil, crafter); !ok {
		t.Fatalf("BUG REPRODUCED: drop reagent incorrectly gated: offending=%q", name)
	}

	// Same, but the reagent additionally carries a foreign MakerName (e.g. it
	// passed through another crafter's hands at some point) — still exempt.
	foreignReagent := items.New(777802)
	foreignReagent.MakerName = "SomeoneElse"
	if ok, name := CheckOwnComponents(recipe, []items.Item{myFrame, foreignReagent}, nil, crafter); !ok {
		t.Fatalf("drop reagent with foreign maker mark incorrectly gated: offending=%q", name)
	}

	// Case 2: the real crafted component is STILL gated when foreign — the
	// fix must not weaken the own-work check for genuine sub-assemblies.
	foreignFrame := items.New(777801)
	foreignFrame.MakerName = "SomeoneElse"
	ok, name := CheckOwnComponents(recipe, []items.Item{foreignFrame, dropReagent}, nil, crafter)
	if ok {
		t.Fatal("foreign crafted component incorrectly accepted")
	}
	if name != "reinforced frame" {
		t.Errorf("offending component name = %q, want %q", name, "reinforced frame")
	}

	// Case 3: a reagent-only recipe (require_own_components still true) is a
	// no-op gate — always accepted, since nothing in it is craftable.
	reagentOnlyRecipe := &RecipeSpec{
		RecipeId: "test-reagent-only", Name: "test reagent only", Skill: "tailoring",
		RequireOwnComponents: true,
		Ingredients: []RecipeIngredient{
			{ItemTag: "folded-space-silk-test", Quantity: 1},
		},
	}
	if ok, name := CheckOwnComponents(reagentOnlyRecipe, []items.Item{foreignReagent}, nil, crafter); !ok {
		t.Fatalf("reagent-only recipe incorrectly gated: offending=%q", name)
	}
}
