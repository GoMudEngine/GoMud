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

	if err := CheckOwnComponents(recipe, []items.Item{mine, bulkForeign}, nil, "Megalomania"); err != nil {
		t.Fatalf("own component rejected: %v", err)
	}
	if err := CheckOwnComponents(recipe, []items.Item{theirs, bulkForeign}, nil, "Megalomania"); err == nil {
		t.Fatal("foreign component accepted")
	}
	if err := CheckOwnComponents(recipe, []items.Item{unmade, bulkForeign}, nil, "Megalomania"); err == nil {
		t.Fatal("maker-less component accepted")
	}

	recipe.RequireOwnComponents = false
	if err := CheckOwnComponents(recipe, []items.Item{theirs, bulkForeign}, nil, "Megalomania"); err != nil {
		t.Fatalf("flag off should not restrict: %v", err)
	}
}
