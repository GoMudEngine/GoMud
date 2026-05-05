package crafting

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestValidateRecipeIngredientTags_FlagsMissingTag(t *testing.T) {
	recipes := map[string]*RecipeSpec{
		"test-r": {
			RecipeId: "test-r",
			Skill:    "alchemy",
			Ingredients: []RecipeIngredient{
				{ItemTag: "lake-mint", Quantity: 1},
			},
		},
	}
	specs := map[int]*items.ItemSpec{
		1: {
			ItemId:           1,
			Name:             "lake mint",
			ComponentTag:     "lake-mint",
			VendorCategories: []string{}, // missing alchemy
		},
	}
	err := ValidateRecipeIngredientTags(recipes, specs)
	if err == nil {
		t.Fatalf("expected error for missing tag")
	}
	if !strings.Contains(err.Error(), "alchemy") {
		t.Errorf("error should mention 'alchemy': %v", err)
	}
}

func TestValidateRecipeIngredientTags_AcceptsCorrectTag(t *testing.T) {
	recipes := map[string]*RecipeSpec{
		"test-r": {
			RecipeId: "test-r",
			Skill:    "alchemy",
			Ingredients: []RecipeIngredient{
				{ItemTag: "lake-mint", Quantity: 1},
			},
		},
	}
	specs := map[int]*items.ItemSpec{
		1: {
			ItemId:           1,
			Name:             "lake mint",
			ComponentTag:     "lake-mint",
			VendorCategories: []string{"alchemy"},
		},
	}
	err := ValidateRecipeIngredientTags(recipes, specs)
	if err != nil {
		t.Errorf("expected no error: %v", err)
	}
}

func TestValidateRecipeIngredientTags_SkipsIngredientsWithoutItem(t *testing.T) {
	// Recipe references a tag with no canonical item; warning only, not panic.
	recipes := map[string]*RecipeSpec{
		"test-r": {
			RecipeId: "test-r",
			Skill:    "alchemy",
			Ingredients: []RecipeIngredient{
				{ItemTag: "ghost-tag", Quantity: 1},
			},
		},
	}
	specs := map[int]*items.ItemSpec{}
	err := ValidateRecipeIngredientTags(recipes, specs)
	if err != nil {
		t.Errorf("expected no error for ghost tag: %v", err)
	}
}
