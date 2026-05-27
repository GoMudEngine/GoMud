package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestCraftItem_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("craft-item"); !ok {
		t.Fatalf("craft-item not registered")
	}
}

func TestCraftItem_DedupKey_ByRecipeId(t *testing.T) {
	meta, _ := goals.LookupGoalType("craft-item")
	g1 := &goals.Goal{Type: "craft-item", Params: map[string]any{"recipe_id": "iron-sword"}}
	g2 := &goals.Goal{Type: "craft-item", Params: map[string]any{"recipe_id": "steel-sword"}}
	if k1, k2 := meta.DedupKey(g1), meta.DedupKey(g2); k1 == k2 {
		t.Errorf("dedup keys collide: %s == %s", k1, k2)
	}
}

func TestCraftItem_ContextScore_RecipeUnknown_Zero(t *testing.T) {
	meta, _ := goals.LookupGoalType("craft-item")
	mob := &mobs.Mob{}
	// mob.Character.KnownRecipes empty — recipe unknown.
	g := &goals.Goal{Type: "craft-item", Params: map[string]any{"recipe_id": "iron-sword"}}
	if got := meta.ContextScore(g, mob); got != 0 {
		t.Errorf("score with unknown recipe: got %f, want 0 (filtered)", got)
	}
}

// Additional tests (skill-too-low → 0.3; known+skilled+missing materials → 1.0;
// known+skilled+materials on hand → 2.0) require a richer test fixture with
// the recipes/skills/items registry. Defer those to integration testing
// once the catalog package is wired (Task 23 smoke).
