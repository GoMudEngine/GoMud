package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	RegisterPlanner("revenge-faction", revengeFactionPlanner)
}

// revengeFactionPlanner: search for and attack hostile faction members
// in zone. (Counter is incremented by 4.5's reactive kill hook; predicate
// in 4.3 catalog checks it. This planner just drives the search+attack.)
func revengeFactionPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	factionId := goalParamStringOr(goal, "faction_id", "")
	if factionId == "" {
		return PlanResult{Status: StatusFailure}
	}

	// Find a faction member in zone (any combat state).
	target, ok := findFactionMemberInZone(mob, factionId, false)
	if !ok {
		return PlanResult{Command: "wander", Status: StatusRunning}
	}

	// Same room → attack.
	if target.Character.RoomId == mob.Character.RoomId {
		return PlanResult{Command: "attack " + target.Character.Name, Status: StatusRunning}
	}

	// Different room same zone → pathto.
	return PlanResult{Command: "pathto " + strconv.Itoa(target.Character.RoomId), Status: StatusRunning}
}
