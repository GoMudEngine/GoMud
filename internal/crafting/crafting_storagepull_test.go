package crafting

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// TestPlanStoragePull exercises the all-or-nothing storage-pull planner.
func TestPlanStoragePull(t *testing.T) {
	recipe := &RecipeSpec{
		Ingredients: []RecipeIngredient{
			{ItemTag: "iron", Quantity: 3},
		},
	}

	t.Run("storage covers the shortfall", func(t *testing.T) {
		// Player holds 1 iron in the component bag; needs 2 more.
		componentInv := []items.Item{makeItem("iron")}
		storage := []items.Item{makeItem("iron"), makeItem("iron"), makeItem("iron")}

		pull, complete := PlanStoragePull(recipe, nil, componentInv, storage)
		if !complete {
			t.Fatalf("expected complete=true, got false")
		}
		if len(pull) != 2 {
			t.Fatalf("expected to pull 2 items, got %d", len(pull))
		}
		for _, it := range pull {
			if componentTagOf(it) != "iron" {
				t.Fatalf("pulled a non-iron item with tag %q", componentTagOf(it))
			}
		}
	})

	t.Run("storage cannot cover the shortfall pulls nothing", func(t *testing.T) {
		// Player holds 1 iron; needs 2 more but storage has only 1.
		componentInv := []items.Item{makeItem("iron")}
		storage := []items.Item{makeItem("iron")}

		pull, complete := PlanStoragePull(recipe, nil, componentInv, storage)
		if complete {
			t.Fatalf("expected complete=false, got true")
		}
		if len(pull) != 0 {
			t.Fatalf("expected to pull nothing, got %d", len(pull))
		}
	})

	t.Run("storage exactly covers the shortfall", func(t *testing.T) {
		// Player holds nothing; storage has exactly 3.
		storage := []items.Item{makeItem("iron"), makeItem("iron"), makeItem("iron")}

		pull, complete := PlanStoragePull(recipe, nil, nil, storage)
		if !complete {
			t.Fatalf("expected complete=true, got false")
		}
		if len(pull) != 3 {
			t.Fatalf("expected to pull 3 items, got %d", len(pull))
		}
	})

	t.Run("backpack holdings count toward the requirement", func(t *testing.T) {
		// 2 in backpack + 1 pulled from storage completes the 3 needed;
		// the extra irrelevant storage item is left behind.
		inv := []items.Item{makeItem("iron"), makeItem("iron")}
		storage := []items.Item{makeItem("iron"), makeItem("copper")}

		pull, complete := PlanStoragePull(recipe, inv, nil, storage)
		if !complete {
			t.Fatalf("expected complete=true, got false")
		}
		if len(pull) != 1 {
			t.Fatalf("expected to pull 1 item, got %d", len(pull))
		}
		if componentTagOf(pull[0]) != "iron" {
			t.Fatalf("expected to pull an iron item, got %q", componentTagOf(pull[0]))
		}
	})
}
