package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func init() {
	RegisterPlanner("protection-mob", protectionMobPlanner)
}

// protectionMobPlanner: defend named ally. Attack their aggressor if
// in combat; else close distance; if target safe in same room, hold.
func protectionMobPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	kind := goalParamStringOr(goal, "target_kind", "")
	id := goalParamIntOr(goal, "target_id", 0)
	if kind == "" || id == 0 {
		return PlanResult{Status: StatusFailure}
	}

	if !targetExists(kind, id) {
		return PlanResult{Status: StatusFailure} // target dead — 4.6 prunes
	}
	targetRoom := resolveTargetRoomId(kind, id)
	if targetRoom == 0 {
		return PlanResult{Status: StatusFailure}
	}
	// Cross-zone protection not supported in 4.4.
	if r := rooms.LoadRoom(targetRoom); r == nil || r.Zone != mob.Character.Zone {
		return PlanResult{Status: StatusFailure}
	}

	aggressor := targetAggressorName(kind, id)

	// Target in combat in same room → attack the aggressor.
	if targetRoom == mob.Character.RoomId && aggressor != "" {
		return PlanResult{Command: "attack " + aggressor, Status: StatusRunning}
	}

	// Target in combat in another room → close the distance.
	if aggressor != "" && targetRoom != mob.Character.RoomId {
		return PlanResult{Command: "pathto " + strconv.Itoa(targetRoom), Status: StatusRunning}
	}

	// Target safe in same room → hold (success — no action this tick).
	if targetRoom == mob.Character.RoomId {
		return PlanResult{Status: StatusSuccess}
	}

	// Target safe in another room same zone → close distance (ready
	// posture).
	return PlanResult{Command: "pathto " + strconv.Itoa(targetRoom), Status: StatusRunning}
}

// targetAggressorName returns the name of whoever is currently attacking
// the target, or empty if target isn't in combat. Inspects the target's
// Aggro pointer's UserId/MobInstanceId.
func targetAggressorName(kind string, id int) string {
	switch kind {
	case "mob":
		for _, instId := range mobs.GetAllMobInstanceIds() {
			inst := mobs.GetInstance(instId)
			if inst == nil || int(inst.MobId) != id {
				continue
			}
			if inst.Character.Aggro == nil {
				return ""
			}
			return aggroToName(inst.Character.Aggro)
		}
	case "player":
		u := users.GetByUserId(id)
		if u == nil || u.Character.Aggro == nil {
			return ""
		}
		return aggroToName(u.Character.Aggro)
	}
	return ""
}

// aggroToName resolves an Aggro pointer to a command-targetable name.
// Returns "" if aggro is nil. Checks UserId first (player aggressor),
// then MobInstanceId (mob aggressor).
func aggroToName(a *characters.Aggro) string {
	if a == nil {
		return ""
	}
	if a.UserId > 0 {
		if u := users.GetByUserId(a.UserId); u != nil {
			return u.Character.Name
		}
	}
	if a.MobInstanceId > 0 {
		if inst := mobs.GetInstance(a.MobInstanceId); inst != nil {
			return inst.Character.Name
		}
	}
	return ""
}
