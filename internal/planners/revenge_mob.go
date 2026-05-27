package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func init() {
	RegisterPlanner("revenge-mob", revengeMobPlanner)
}

// revengeMobPlanner: pursue target across rooms in zone; attack when
// adjacent or same room. No cross-zone pursuit in 4.4.
func revengeMobPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	kind := goalParamStringOr(goal, "target_kind", "")
	id := goalParamIntOr(goal, "target_id", 0)
	if kind == "" || id == 0 {
		return PlanResult{Status: StatusFailure}
	}

	// Target dead → success (catalog predicate will satisfy next tick).
	if !targetExists(kind, id) {
		return PlanResult{Status: StatusSuccess}
	}

	targetRoom := resolveTargetRoomId(kind, id)
	if targetRoom == 0 {
		return PlanResult{Status: StatusFailure}
	}

	// Target out of zone → Failure (4.4 does not pursue cross-zone).
	if r := rooms.LoadRoom(targetRoom); r == nil || r.Zone != mob.Character.Zone {
		return PlanResult{Status: StatusFailure}
	}

	// Same room → attack. (Use bare 'attack' since the existing target
	// resolution in mob commands will find the right target. If the
	// command requires an explicit name, use targetName lookup below.)
	if targetRoom == mob.Character.RoomId {
		name := targetCommandName(kind, id)
		if name == "" {
			return PlanResult{Status: StatusFailure}
		}
		return PlanResult{Command: "attack " + name, Status: StatusRunning}
	}

	// Different room same zone → pathto.
	return PlanResult{Command: "pathto " + strconv.Itoa(targetRoom), Status: StatusRunning}
}

// targetExists returns true iff a live mob/player matching kind+id exists.
func targetExists(kind string, id int) bool {
	switch kind {
	case "mob":
		for _, instId := range mobs.GetAllMobInstanceIds() {
			if inst := mobs.GetInstance(instId); inst != nil && int(inst.MobId) == id {
				return true
			}
		}
		return false
	case "player":
		u := users.GetByUserId(id)
		return u != nil && u.Character.Health > 0
	}
	return false
}

// resolveTargetRoomId returns the current room of a target, or 0 if
// unresolvable. (Same pattern as 4.3 catalog's helpers.go — separate
// package, duplicated.)
func resolveTargetRoomId(kind string, id int) int {
	switch kind {
	case "mob":
		for _, instId := range mobs.GetAllMobInstanceIds() {
			if inst := mobs.GetInstance(instId); inst != nil && int(inst.MobId) == id {
				return inst.Character.RoomId
			}
		}
	case "player":
		if u := users.GetByUserId(id); u != nil {
			return u.Character.RoomId
		}
	}
	return 0
}

// targetCommandName returns the noun string for command targeting.
// For mobs: shorthand or name; for players: username.
func targetCommandName(kind string, id int) string {
	switch kind {
	case "mob":
		for _, instId := range mobs.GetAllMobInstanceIds() {
			if inst := mobs.GetInstance(instId); inst != nil && int(inst.MobId) == id {
				return inst.Character.Name
			}
		}
	case "player":
		if u := users.GetByUserId(id); u != nil {
			return u.Character.Name
		}
	}
	return ""
}
