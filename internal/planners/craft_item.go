package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/seeders"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

const craftItemStationRoomKey = "plan:craft-item:target_station_room"

func init() {
	RegisterPlanner("craft-item", craftItemPlanner)
}

// craftItemPlanner: produce a specific recipe. Branches per tick:
//   - recipe_id param missing / recipe unknown → Failure
//   - output item already in inventory/equipment → Success
//   - recipe unknown to mob → Failure (mob hasn't learned it yet)
//   - skill rank below recipe minimum → Failure (let coexisting
//     mastery-skill goal win next recompute)
//   - materials missing → Failure (4.5 will reactively seed a
//     wealth-item goal to acquire them)
//   - materials on hand + at station (or no station required) → craft
//   - materials on hand + need station + not at one → resolve sticky
//     target_station_room MiscData + pathto
func craftItemPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	rid := goalParamStringOr(goal, "recipe_id", "")
	if rid == "" {
		return PlanResult{Status: StatusFailure}
	}
	r := crafting.GetRecipe(rid)
	if r == nil {
		return PlanResult{Status: StatusFailure}
	}

	// Output already in inventory/equipment → success.
	if r.Output.ItemId > 0 && mobHasItem(mob, "", r.Output.ItemId) {
		return PlanResult{Status: StatusSuccess}
	}

	// Recipe not yet learned → Failure.
	if !craftPlannerMobKnowsRecipe(mob, rid) {
		return PlanResult{Status: StatusFailure}
	}

	// Skill rank below recipe minimum → Failure (mastery-skill goal wins).
	if !craftPlannerMobMeetsRecipeSkill(mob, r) {
		return PlanResult{Status: StatusFailure}
	}

	// Materials missing → seed wealth-item goals for the missing
	// ingredient tags so the mob has a productive next-tick goal
	// instead of spinning, then return Failure. The seeders package's
	// DedupKey on wealth-item collapses repeat seedings.
	if !craftPlannerMobHasMaterials(mob, r) {
		seeders.SeedMaterialsForRecipe(mob, rid)
		return PlanResult{Status: StatusFailure}
	}

	// At station OR no station required → craft.
	if r.Station == "" || craftPlannerMobInStationRoom(mob, r.Station) {
		return PlanResult{Command: "craft " + rid, Status: StatusRunning}
	}

	// Need station + not at one → pathto (sticky MiscData).
	stationRoom := mobMiscIntOr(mob, craftItemStationRoomKey, 0)
	if stationRoom == 0 {
		room, ok := findCraftingStationInZone(mob, r.Station)
		if !ok {
			return PlanResult{Status: StatusFailure}
		}
		stationRoom = room
		mobSetMisc(mob, craftItemStationRoomKey, stationRoom)
	}
	return PlanResult{Command: "pathto " + strconv.Itoa(stationRoom), Status: StatusRunning}
}

// ─── Local adapters (mirror 4.3 catalog/craft_item.go, same APIs) ────────────

// craftPlannerMobKnowsRecipe reports whether the mob has learned the recipe.
// Delegates to Character.HasRecipe which guards the nil-map case.
func craftPlannerMobKnowsRecipe(mob *mobs.Mob, recipeId string) bool {
	return mob.Character.HasRecipe(recipeId)
}

// craftPlannerMobMeetsRecipeSkill reports whether the mob's skill level meets
// the recipe's required minimum. Uses Character.GetSkillLevel which folds in
// equipment/buff StatMod bonuses.
func craftPlannerMobMeetsRecipeSkill(mob *mobs.Mob, r *crafting.RecipeSpec) bool {
	return mob.Character.GetSkillLevel(skills.SkillTag(r.Skill)) >= r.SkillMinimum
}

// craftPlannerMobHasMaterials reports whether the mob holds all ingredients
// for the recipe. Checks both the backpack (Character.Items) and the component
// bag (Character.ComponentItems).
func craftPlannerMobHasMaterials(mob *mobs.Mob, r *crafting.RecipeSpec) bool {
	ok, _ := crafting.HasIngredients(mob.Character.Items, mob.Character.ComponentItems, r)
	return ok
}

// craftPlannerMobInStationRoom reports whether the mob's current room has the
// named crafting station. Returns false for empty station names or nil mobs.
func craftPlannerMobInStationRoom(mob *mobs.Mob, stationName string) bool {
	if mob == nil || stationName == "" {
		return false
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return false
	}
	return room.Station == stationName
}
