package catalog

import (
	"strconv"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func init() {
	goals.RegisterGoalType("protection-mob", goals.GoalTypeMeta{
		Predicate:     protectionMobPredicate,
		ContextScore:  protectionMobContextScore,
		AllowMultiple: true,
		DedupKey:      protectionMobDedupKey,
		Params: []goals.ParamSchema{
			{Key: "target_kind", Required: true, GoType: "string"},
			{Key: "target_id", Required: true, GoType: "int"},
		},
	})
}

func protectionMobDedupKey(g *goals.Goal) string {
	kind, _ := g.Params["target_kind"].(string)
	id := paramIntOr(g, "target_id", 0)
	if kind == "" || id == 0 {
		return ""
	}
	return kind + ":" + strconv.Itoa(id)
}

// protectionMobPredicate: never satisfied (ongoing). 4.6's pruning
// sweep removes the goal when the target has been dead for >= N rounds.
func protectionMobPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	return false
}

// protectionMobContextScore:
//   - 0   if target dead
//   - 2.5 if target currently in combat
//   - 1.5 if target in same room (not in combat)
//   - 0.8 if target in same zone
//   - 0.2 if target alive in different zone
func protectionMobContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	kind, _ := g.Params["target_kind"].(string)
	id := paramIntOr(g, "target_id", 0)
	if kind == "" || id == 0 {
		return 0
	}
	if !targetAlive(kind, id) {
		return 0
	}
	if targetInCombat(kind, id) {
		return 2.5
	}
	return targetProximityScoreProtection(mob, kind, id)
}

// targetProximityScoreProtection returns a step-down proximity score
// for protection goals — same magnitudes as the plan spec:
//
//	1.5 — same room (target not in combat)
//	0.8 — same zone
//	0.2 — different zone (target alive but far)
func targetProximityScoreProtection(mob *mobs.Mob, kind string, id int) float64 {
	targetRoomId := resolveTargetRoomId(kind, id)
	if targetRoomId == 0 {
		return 0.2
	}
	if targetRoomId == mob.Character.RoomId {
		return 1.5
	}
	if r := rooms.LoadRoom(targetRoomId); r != nil && r.Zone == mob.Character.Zone {
		return 0.8
	}
	return 0.2
}
