package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/skills"
)

func init() {
	RegisterPlanner("mastery-skill", masterySkillPlanner)
}

// masterySkillPlanner: dispatch to per-context training action based
// on the skill's TrainingContext. Per-skill mapping lives in
// skillTrainingTable.
func masterySkillPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	skillName := goalParamStringOr(goal, "skill_name", "")
	targetRank := goalParamIntOr(goal, "target_rank", 0)
	if skillName == "" || targetRank == 0 {
		return PlanResult{Status: StatusFailure}
	}

	currentRank := masteryMobSkillRank(mob, skillName)
	if currentRank >= targetRank {
		return PlanResult{Status: StatusSuccess}
	}

	switch SkillTrainingContextOf(skillName) {
	case TrainingCombat:
		// Auto-aggro mob in room → attack. Else wander.
		if hostile, ok := findHostileInRoom(mob); ok {
			return PlanResult{Command: "attack " + hostile.Character.Name, Status: StatusRunning}
		}
		return PlanResult{Command: "wander", Status: StatusRunning}
	case TrainingCrafting:
		// At a station with a known recipe? craft it. Else pathto station.
		recipeId := pickKnownRecipeForSkill(mob, skillName)
		if recipeId == "" {
			return PlanResult{Status: StatusFailure}
		}
		stationName := masteryStationForRecipe(recipeId)
		if stationName == "" || craftPlannerMobInStationRoom(mob, stationName) {
			return PlanResult{Command: "craft " + recipeId, Status: StatusRunning}
		}
		room, ok := findCraftingStationInZone(mob, stationName)
		if !ok {
			return PlanResult{Status: StatusFailure}
		}
		return PlanResult{Command: "pathto " + strconv.Itoa(room), Status: StatusRunning}
	case TrainingForaging:
		return PlanResult{Command: "forage", Status: StatusRunning}
	case TrainingSocial:
		// Anyone in room? emote. Else wander.
		if masteryRoomHasObserver(mob) {
			return PlanResult{Command: pickSocialEmote(), Status: StatusRunning}
		}
		return PlanResult{Command: "wander", Status: StatusRunning}
	case TrainingSkullduggery:
		// No autonomous theft in 4.4 (too easy to misfire). Wander.
		return PlanResult{Command: "wander", Status: StatusRunning}
	}
	return PlanResult{Status: StatusFailure} // TrainingUnknown
}

// masteryMobSkillRank reads the mob's current rank for a skill.
func masteryMobSkillRank(mob *mobs.Mob, skillName string) int {
	return mob.Character.GetSkillLevel(skills.SkillTag(skillName))
}

// findHostileInRoom returns the first AutoAggro mob in the same room as mob
// (excluding self). Used by the combat training branch.
func findHostileInRoom(mob *mobs.Mob) (*mobs.Mob, bool) {
	for _, instId := range mobs.GetAllMobInstanceIds() {
		inst := mobs.GetInstance(instId)
		if inst == nil || inst.Character.RoomId != mob.Character.RoomId || inst.InstanceId == mob.InstanceId {
			continue
		}
		if inst.AutoAggro {
			return inst, true
		}
	}
	return nil, false
}

// masteryRoomHasObserver reports whether there is an audience (player or NPC)
// in the mob's room to witness a social emote.
// TODO-ADAPT: rooms.LoadRoom(mob.Character.RoomId).GetPlayers() or similar
// — count > 0 means there's an audience. Cheapest stub: always true (gives
// the social emote a chance to fire).
func masteryRoomHasObserver(mob *mobs.Mob) bool {
	return true
}

// masteryStationForRecipe returns the crafting station name required by a
// recipe. Returns "" if the recipe is unknown or requires no station.
func masteryStationForRecipe(recipeId string) string {
	r := crafting.GetRecipe(recipeId)
	if r == nil {
		return ""
	}
	return r.Station
}
