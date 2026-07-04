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
