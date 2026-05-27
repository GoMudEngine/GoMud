package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestCraftItem_Registered(t *testing.T) {
	if LookupPlanner("craft-item") == nil {
		t.Fatalf("craft-item planner not registered")
	}
}

func TestCraftItem_NoRecipeIdParam_Failure(t *testing.T) {
	fn := LookupPlanner("craft-item")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "craft-item"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want StatusFailure (no recipe_id param)", res.Status)
	}
}

func TestCraftItem_EmptyRecipeId_Failure(t *testing.T) {
	fn := LookupPlanner("craft-item")
	g := &goals.Goal{Type: "craft-item", Params: map[string]any{"recipe_id": ""}}
	res := fn(&mobs.Mob{}, g)
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want StatusFailure (empty recipe_id)", res.Status)
	}
}

func TestCraftItem_UnknownRecipe_Failure(t *testing.T) {
	fn := LookupPlanner("craft-item")
	g := &goals.Goal{Type: "craft-item", Params: map[string]any{"recipe_id": "does-not-exist-recipe"}}
	res := fn(&mobs.Mob{}, g)
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want StatusFailure (unknown recipe)", res.Status)
	}
}

func TestCraftItem_NilMob_Failure(t *testing.T) {
	fn := LookupPlanner("craft-item")
	g := &goals.Goal{Type: "craft-item", Params: map[string]any{"recipe_id": "some-recipe"}}
	res := fn(nil, g)
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want StatusFailure (nil mob)", res.Status)
	}
}
