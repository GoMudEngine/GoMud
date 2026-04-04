package shops

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/crafting"
)

// ─── helpers ────────────────────────────────────────────────────────────────

func makeInv(stock []StockEntry) *ShopInventory {
	return &ShopInventory{Stock: stock}
}

func makeRecipe(id string, ingTag string, ingQty int, outItemId int, outQty int) *crafting.RecipeSpec {
	return &crafting.RecipeSpec{
		RecipeId: id,
		Name:     id,
		Skill:    "blacksmithing",
		Ingredients: []crafting.RecipeIngredient{
			{ItemTag: ingTag, Quantity: ingQty},
		},
		Output: crafting.RecipeOutput{ItemId: outItemId, Quantity: outQty},
	}
}

// ─── MaterialCost ────────────────────────────────────────────────────────────

func TestMaterialCost_NilInventory(t *testing.T) {
	recipe := makeRecipe("r1", "iron", 2, 100, 1)
	cfg := DefaultPricingConfig()
	if got := MaterialCost(recipe, nil, cfg); got != 0 {
		t.Errorf("expected 0 for nil shopInv, got %f", got)
	}
}

func TestMaterialCost_NilRecipe(t *testing.T) {
	inv := makeInv(nil)
	cfg := DefaultPricingConfig()
	if got := MaterialCost(nil, inv, cfg); got != 0 {
		t.Errorf("expected 0 for nil recipe, got %f", got)
	}
}

// ─── HasMaterialsWithReserve ─────────────────────────────────────────────────

func TestHasMaterialsWithReserve_NilInventory(t *testing.T) {
	recipe := makeRecipe("r1", "iron", 1, 100, 1)
	if HasMaterialsWithReserve(recipe, nil, 0) {
		t.Error("expected false for nil shopInv")
	}
}

func TestHasMaterialsWithReserve_NilRecipe(t *testing.T) {
	inv := makeInv(nil)
	if HasMaterialsWithReserve(nil, inv, 0) {
		t.Error("expected false for nil recipe")
	}
}

// ─── EvaluateCraftOptions ────────────────────────────────────────────────────

func TestEvaluateCraftOptions_NilInventory(t *testing.T) {
	if got := EvaluateCraftOptions([]string{"r1"}, nil, DefaultPricingConfig(), 0); got != nil {
		t.Errorf("expected nil for nil shopInv, got %+v", got)
	}
}

func TestEvaluateCraftOptions_EmptyRecipes(t *testing.T) {
	inv := makeInv(nil)
	if got := EvaluateCraftOptions([]string{}, inv, DefaultPricingConfig(), 0); got != nil {
		t.Errorf("expected nil for empty recipes, got %+v", got)
	}
}

func TestEvaluateCraftOptions_UnknownRecipeSkipped(t *testing.T) {
	// Recipes not in the registry return nil from GetRecipe; should be skipped.
	inv := makeInv(nil)
	if got := EvaluateCraftOptions([]string{"nonexistent-recipe-id"}, inv, DefaultPricingConfig(), 0); got != nil {
		t.Errorf("expected nil for unknown recipe, got %+v", got)
	}
}

// ─── EvaluateSalvageOptions ──────────────────────────────────────────────────

func TestEvaluateSalvageOptions_NilInventory(t *testing.T) {
	if got := EvaluateSalvageOptions(nil, DefaultPricingConfig()); got != nil {
		t.Errorf("expected nil for nil shopInv, got %+v", got)
	}
}

func TestEvaluateSalvageOptions_EmptyInventory(t *testing.T) {
	inv := makeInv(nil)
	if got := EvaluateSalvageOptions(inv, DefaultPricingConfig()); got != nil {
		t.Errorf("expected nil for empty inventory, got %+v", got)
	}
}

func TestEvaluateSalvageOptions_SkipsLowStock(t *testing.T) {
	// Current == 1 should not be salvaged (keep the last one)
	inv := makeInv([]StockEntry{
		{ItemId: 999, RestockQty: 0, MaxStock: 10, Current: 1},
	})
	if got := EvaluateSalvageOptions(inv, DefaultPricingConfig()); got != nil {
		t.Errorf("expected nil when Current <= 1, got %+v", got)
	}
}

func TestEvaluateSalvageOptions_SkipsRestockedItems(t *testing.T) {
	// Items with RestockQty > 0 are supply-cart items, not crafted goods
	inv := makeInv([]StockEntry{
		{ItemId: 999, RestockQty: 5, MaxStock: 20, Current: 10},
	})
	if got := EvaluateSalvageOptions(inv, DefaultPricingConfig()); got != nil {
		t.Errorf("expected nil for restocked item, got %+v", got)
	}
}

// ─── ProductValue ────────────────────────────────────────────────────────────

func TestProductValue_NilInventory(t *testing.T) {
	recipe := makeRecipe("r1", "iron", 1, 100, 1)
	cfg := DefaultPricingConfig()
	if got := ProductValue(recipe, nil, cfg); got != 0 {
		t.Errorf("expected 0 for nil shopInv, got %f", got)
	}
}

func TestProductValue_NilRecipe(t *testing.T) {
	inv := makeInv(nil)
	cfg := DefaultPricingConfig()
	if got := ProductValue(nil, inv, cfg); got != 0 {
		t.Errorf("expected 0 for nil recipe, got %f", got)
	}
}

func TestProductValue_ZeroOutputItemId(t *testing.T) {
	recipe := &crafting.RecipeSpec{
		RecipeId: "enc",
		Name:     "enchanting",
		Skill:    "enchanting",
		Output:   crafting.RecipeOutput{ItemId: 0, Quantity: 1},
	}
	inv := makeInv(nil)
	cfg := DefaultPricingConfig()
	if got := ProductValue(recipe, inv, cfg); got != 0 {
		t.Errorf("expected 0 for zero output item id, got %f", got)
	}
}
