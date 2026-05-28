package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const ruleNameCraftMaterialsSeed = "craft_materials_to_wealth_item"
const craftMaterialsSeedPriority = 60

// SeedMaterialsForRecipe is the public entry point called from
// internal/planners/craft_item.go's Failure branch when materials are
// missing. Walks the recipe ingredients and seeds a wealth-item goal
// for each ingredient tag the mob doesn't already have. The 4.3
// catalog's wealth-item DedupKey collapses repeat seedings for the
// same tag, so calling this every failure-tick is safe.
//
// NOT registered with the event dispatcher — no clean "planner failed
// because materials missing" world event exists, so the planner calls
// this function directly. This is the architectural exception noted in
// the chunk 4.5 spec §2.3.
func SeedMaterialsForRecipe(mob *mobs.Mob, recipeId string) {
	if mob == nil || recipeId == "" {
		return
	}
	r := crafting.GetRecipe(recipeId)
	if r == nil {
		return
	}
	mobId := int(mob.MobId)
	name := util.ConvertForFilename(mob.Character.Name)

	for _, ing := range r.Ingredients {
		tag := ing.ItemTag
		if tag == "" {
			continue
		}
		if mobHasIngredientTag(mob, tag) {
			continue
		}

		g := &goals.Goal{
			Type:     "wealth-item",
			Priority: craftMaterialsSeedPriority,
			Params:   map[string]any{"item_tag": tag},
		}
		_, err := goals.Add(mobId, name, g)
		if err != nil {
			mudlog.Debug("seeders.SeedMaterialsForRecipe: Add",
				"mob_id", mobId, "recipe", recipeId, "tag", tag, "error", err)
		}
	}
}

// mobHasIngredientTag reports whether the mob holds any item with the
// given ComponentTag in its backpack or component bag. Mirrors the
// crafting.HasIngredients lookup so the seeder's "already have it"
// check is consistent with the planner's craftPlannerMobHasMaterials
// check (which also scans ComponentItems).
func mobHasIngredientTag(mob *mobs.Mob, tag string) bool {
	for i := range mob.Character.Items {
		if mob.Character.Items[i].GetSpec().ComponentTag == tag {
			return true
		}
	}
	for i := range mob.Character.ComponentItems {
		if mob.Character.ComponentItems[i].GetSpec().ComponentTag == tag {
			return true
		}
	}
	return false
}
