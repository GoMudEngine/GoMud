package crafting

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/stretchr/testify/assert"
)

// ─── Integration: Full Crafting Loop ─────────────────────────────────────────
// Exercises: seedRegistry, HasIngredients(), ConsumeIngredients(),
// CalcSuccessChance(), GetStarterRecipes(), GetEligibleRecipes()

func TestIntegration_CraftingFullLoop(t *testing.T) {
	seedRegistry()

	recipe := GetRecipe("iron-dagger")
	assert.NotNil(t, recipe, "iron-dagger recipe should exist")

	// Simulate a character with blacksmithing skill and ingredients
	skillLevel := 5
	inv := []items.Item{
		makeItem("iron-ingot"),
		makeItem("leather-strip"),
		makeItem("other-junk"),
	}

	// Step 1: Check ingredients
	ok, missing := HasIngredients(inv, []items.Item{}, recipe)
	assert.True(t, ok, "Should have all ingredients")
	assert.Empty(t, missing, "No missing ingredients")

	// Step 2: Calculate success chance
	chance := CalcSuccessChance(skillLevel, recipe.SkillMinimum)
	assert.GreaterOrEqual(t, chance, 50, "Skill at or above min should give >= 50%%")
	assert.LessOrEqual(t, chance, 95, "Chance should not exceed 95%% cap")

	// With skill 5 above minimum 0: base 50 + 5*5 = 75%
	assert.Equal(t, 75, chance, "Skill 5 above min should give 75%%")

	// Step 3: Consume ingredients
	remaining, _ := ConsumeIngredients(inv, []items.Item{}, recipe)
	assert.Len(t, remaining, 1, "Only the junk item should remain")
	assert.Equal(t, "other-junk", remaining[0].GetSpec().ComponentTag,
		"Remaining item should be the non-ingredient")

	// Step 4: Verify ingredients are gone
	ok, _ = HasIngredients(remaining, []items.Item{}, recipe)
	assert.False(t, ok, "Should no longer have ingredients after consuming")
}

func TestIntegration_CraftingInsufficientSkill(t *testing.T) {
	seedRegistry()

	recipe := GetRecipe("iron-buckler") // SkillMinimum = 5

	// Character with low skill
	skillLevel := 1

	// Success chance with skill well below minimum
	chance := CalcSuccessChance(skillLevel, recipe.SkillMinimum)
	assert.Less(t, chance, 50,
		"Skill below minimum should give < 50%% chance")

	// Minimum floor is 5%
	veryLowChance := CalcSuccessChance(-10, recipe.SkillMinimum)
	assert.Equal(t, 5, veryLowChance,
		"Very low skill should hit the 5%% floor")
}

func TestIntegration_CraftingIngredientsNotConsumedOnCheck(t *testing.T) {
	seedRegistry()

	recipe := GetRecipe("iron-dagger")
	inv := []items.Item{
		makeItem("iron-ingot"),
		makeItem("leather-strip"),
	}

	// HasIngredients should NOT modify the inventory
	ok, _ := HasIngredients(inv, []items.Item{}, recipe)
	assert.True(t, ok)
	assert.Len(t, inv, 2, "HasIngredients should not modify inventory")

	// Call it again — should still pass
	ok, _ = HasIngredients(inv, []items.Item{}, recipe)
	assert.True(t, ok, "Repeated HasIngredients should still pass")
}

func TestIntegration_CraftingMultiQuantityIngredient(t *testing.T) {
	seedRegistry()

	buckler := GetRecipe("iron-buckler") // needs 2x iron-ingot + 1x wooden-plank

	// Only 1 iron-ingot — should fail
	inv := []items.Item{
		makeItem("iron-ingot"),
		makeItem("wooden-plank"),
	}
	ok, missing := HasIngredients(inv, []items.Item{}, buckler)
	assert.False(t, ok, "Should fail with only 1 iron-ingot")
	assert.Equal(t, "iron-ingot", missing, "Missing ingredient should be iron-ingot")

	// Add second iron-ingot — should pass
	inv = append(inv, makeItem("iron-ingot"))
	ok, _ = HasIngredients(inv, []items.Item{}, buckler)
	assert.True(t, ok, "Should pass with 2 iron-ingots")

	// Consume and verify
	remaining, _ := ConsumeIngredients(inv, []items.Item{}, buckler)
	assert.Len(t, remaining, 0, "All items should be consumed for buckler")
}

func TestIntegration_RecipeDiscovery(t *testing.T) {
	seedRegistry()

	// Starter recipes: SkillMinimum == 0
	starters := GetStarterRecipes()
	assert.Contains(t, starters, "iron-dagger",
		"iron-dagger (min=0) should be a starter")
	assert.Contains(t, starters, "healing-poultice",
		"healing-poultice (min=0) should be a starter")
	assert.NotContains(t, starters, "iron-buckler",
		"iron-buckler (min=5) should NOT be a starter")

	// Simulate a character who knows starters and has blacksmithing=5
	knownRecipes := map[string]int{
		"iron-dagger":      1,
		"healing-poultice": 1,
	}
	skills := map[string]int{
		"blacksmithing": 5,
		"alchemy":       0,
	}

	eligible := GetEligibleRecipes(knownRecipes, skills)

	// iron-buckler (min=5, blacksmithing) should now be discoverable
	found := false
	for _, id := range eligible {
		if id == "iron-buckler" {
			found = true
		}
		// Known recipes should never appear
		assert.NotEqual(t, "iron-dagger", id,
			"Already known recipes should not be eligible")
		assert.NotEqual(t, "healing-poultice", id,
			"Already known recipes should not be eligible")
	}
	assert.True(t, found, "iron-buckler should be eligible at skill 5")

	// If skill is too low, nothing should be eligible
	skills["blacksmithing"] = 3
	eligible = GetEligibleRecipes(knownRecipes, skills)
	for _, id := range eligible {
		assert.NotEqual(t, "iron-buckler", id,
			"iron-buckler should not be eligible with skill 3")
	}
}

func TestIntegration_CraftingSuccessChanceRange(t *testing.T) {
	// Verify the full range of success chances
	tests := []struct {
		name  string
		skill int
		min   int
		want  int
	}{
		{"at minimum", 5, 5, 50},
		{"1 above", 6, 5, 55},
		{"max cap", 20, 5, 95},
		{"way above max", 100, 5, 95},
		{"below minimum", 3, 5, 40},
		{"way below", -10, 5, 5},
		{"zero skill zero min", 0, 0, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcSuccessChance(tt.skill, tt.min)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIntegration_RecipeRegistry(t *testing.T) {
	seedRegistry()

	// GetAll should return all seeded recipes
	all := GetAll()
	assert.Len(t, all, 3, "Should have 3 seeded recipes")

	// GetRecipe for each
	assert.NotNil(t, GetRecipe("iron-dagger"))
	assert.NotNil(t, GetRecipe("iron-buckler"))
	assert.NotNil(t, GetRecipe("healing-poultice"))
	assert.Nil(t, GetRecipe("nonexistent"))

	// FindRecipeByName
	r := FindRecipeByName("poultice")
	assert.NotNil(t, r)
	assert.Equal(t, "healing-poultice", r.RecipeId)

	// GetAllForSkill
	blacksmithing := GetAllForSkill("blacksmithing")
	assert.Len(t, blacksmithing, 2, "Should have 2 blacksmithing recipes")

	alchemy := GetAllForSkill("alchemy")
	assert.Len(t, alchemy, 1, "Should have 1 alchemy recipe")
}
