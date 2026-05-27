package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// TestSeedMaterialsForRecipe_NilMob_NoPanic guards the nil-mob early
// return. No recipe lookup happens, so no panic regardless of recipe id.
func TestSeedMaterialsForRecipe_NilMob_NoPanic(t *testing.T) {
	SeedMaterialsForRecipe(nil, "any-recipe")
}

// TestSeedMaterialsForRecipe_EmptyRecipeId_NoPanic guards the empty
// recipe id early return.
func TestSeedMaterialsForRecipe_EmptyRecipeId_NoPanic(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.Name = "empty_recipe_test"
	SeedMaterialsForRecipe(mob, "")
}

// TestSeedMaterialsForRecipe_UnknownRecipe_NoOp verifies that an
// unknown recipe id causes an early return without panicking. The
// crafting registry returns nil for unknown ids; the function must
// handle that gracefully.
func TestSeedMaterialsForRecipe_UnknownRecipe_NoOp(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.Name = "unknown_recipe_test"
	mob.MobId = mobs.MobId(99100)
	SeedMaterialsForRecipe(mob, "does-not-exist-recipe-zzz")
}

// Integration test (real recipe + missing materials → wealth-item
// seeded) requires a live recipe registry and goals store. Deferred to
// Task 15 smoke per spec §6.3.
