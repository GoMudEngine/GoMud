package catalog

import (
	"github.com/GoMudEngine/GoMud/internal/crafting"
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

func init() {
	goals.RegisterGoalType("craft-item", goals.GoalTypeMeta{
		Predicate:     craftItemPredicate,
		ContextScore:  craftItemContextScore,
		AllowMultiple: true,
		DedupKey:      craftItemDedupKey,
		Params: []goals.ParamSchema{
			{Key: "recipe_id", Required: true, GoType: "string"},
		},
	})
}

func craftItemDedupKey(g *goals.Goal) string {
	if rid, ok := g.Params["recipe_id"].(string); ok {
		return rid
	}
	return ""
}

// craftItemPredicate: satisfied when the item produced by the recipe
// is in mob's inventory or equipment.
func craftItemPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	rid, _ := g.Params["recipe_id"].(string)
	if rid == "" {
		return false
	}
	outputId, ok := craftingRecipeOutputId(rid)
	if !ok {
		return false
	}
	return mobHasItem(mob, "", outputId)
}

// craftItemContextScore tiers:
//   - Recipe unknown to mob → 0 (filtered)
//   - Skill rank below recipe's required minimum → 0.3 (let mastery-skill win)
//   - Known + skilled + materials missing → 1.0
//   - Known + skilled + materials on hand → 2.0
func craftItemContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	rid, _ := g.Params["recipe_id"].(string)
	if rid == "" {
		return 0
	}
	if !mobKnowsRecipe(mob, rid) {
		return 0
	}
	if !mobMeetsRecipeSkill(mob, rid) {
		return 0.3
	}
	if mobHasRecipeMaterials(mob, rid) {
		return 2.0
	}
	return 1.0
}

// ─── Adapters to the crafting registry ──────────────────────────────────────

// craftingRecipeOutputId returns the item id produced by a recipe.
// Returns (0, false) if the recipe is unknown or has no output item
// (e.g. enchanting recipes).
func craftingRecipeOutputId(recipeId string) (int, bool) {
	r := crafting.GetRecipe(recipeId)
	if r == nil || r.Output.ItemId <= 0 {
		return 0, false
	}
	return r.Output.ItemId, true
}

// mobKnowsRecipe reports whether the mob's character has learned the recipe.
// Delegates to Character.HasRecipe which guards the nil-map case.
func mobKnowsRecipe(mob *mobs.Mob, recipeId string) bool {
	return mob.Character.HasRecipe(recipeId)
}

// mobMeetsRecipeSkill reports whether the mob's skill level meets the
// recipe's required minimum. Uses Character.GetSkillLevel which folds
// in equipment/buff StatMod bonuses.
func mobMeetsRecipeSkill(mob *mobs.Mob, recipeId string) bool {
	r := crafting.GetRecipe(recipeId)
	if r == nil {
		return false
	}
	return mob.Character.GetSkillLevel(skills.SkillTag(r.Skill)) >= r.SkillMinimum
}

// mobHasRecipeMaterials reports whether the mob has all ingredients for
// the recipe. Checks both the backpack (Character.Items) and the
// component bag (Character.ComponentItems).
func mobHasRecipeMaterials(mob *mobs.Mob, recipeId string) bool {
	r := crafting.GetRecipe(recipeId)
	if r == nil {
		return false
	}
	ok, _ := crafting.HasIngredients(mob.Character.Items, mob.Character.ComponentItems, r)
	return ok
}
