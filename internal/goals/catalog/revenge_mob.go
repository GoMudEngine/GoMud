package catalog

import (
	"strconv"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func init() {
	goals.RegisterGoalType("revenge-mob", goals.GoalTypeMeta{
		Predicate:     revengeMobPredicate,
		ContextScore:  revengeMobContextScore,
		AllowMultiple: true,
		DedupKey:      revengeMobDedupKey,
		Params: []goals.ParamSchema{
			{Key: "target_kind", Required: true, GoType: "string"}, // "mob" | "player"
			{Key: "target_id", Required: true, GoType: "int"},
		},
	})
}

func revengeMobDedupKey(g *goals.Goal) string {
	kind, _ := g.Params["target_kind"].(string)
	id := paramIntOr(g, "target_id", 0)
	if kind == "" || id == 0 {
		return ""
	}
	return kind + ":" + strconv.Itoa(id)
}

// revengeMobPredicate: satisfied when the target is dead (or offline
// for players, since offline = removed from the live users map).
//
//   - "mob" kind: no live instance with matching template MobId → dead.
//   - "player" kind: user not in the live users map OR Character.Health <= 0.
func revengeMobPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	kind, _ := g.Params["target_kind"].(string)
	id := paramIntOr(g, "target_id", 0)
	if kind == "" || id == 0 {
		return false
	}
	switch kind {
	case "mob":
		for _, instId := range mobs.GetAllMobInstanceIds() {
			if inst := mobs.GetInstance(instId); inst != nil && inst.MobId == mobs.MobId(id) {
				return false // at least one instance alive
			}
		}
		return true // no instances found → template dead / despawned
	case "player":
		u := users.GetByUserId(id)
		return u == nil || u.Character.Health <= 0
	}
	return false
}

// revengeMobContextScore tiers by target proximity, gated on recent sight:
//
//	0   — target not seen recently per CombatMemory (> 1000 rounds)
//	2.0 — target in the same room
//	1.5 — target in an adjacent room (one exit hop)
//	0.5 — target elsewhere in the same zone
//	0.1 — target out of zone (still nonzero to allow long-range revenge)
func revengeMobContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	kind, _ := g.Params["target_kind"].(string)
	id := paramIntOr(g, "target_id", 0)
	if kind == "" || id == 0 {
		return 0
	}
	if !targetRecentlySeen(mob, kind, id) {
		return 0
	}
	return targetProximityScore(mob, kind, id)
}

// ─── Shared helpers — used by revenge-mob, protection-mob, befriend ──────────
// Extract to internal/goals/catalog/helpers.go when Tasks 13/15 need them.

// targetRecentlySeen returns true when mob's CombatMemory records a sighting
// of the given target within the last 1000 rounds (spec §ContextScore window).
func targetRecentlySeen(mob *mobs.Mob, kind string, id int) bool {
	if mob.CombatMemory == nil {
		return false
	}
	switch kind {
	case "mob":
		if mob.CombatMemory.TargetMobId != id {
			return false
		}
	case "player":
		if mob.CombatMemory.TargetUserId != id {
			return false
		}
	default:
		return false
	}
	const recentWindow uint64 = 1000
	return util.GetRoundCount()-mob.CombatMemory.LastSeenRound <= recentWindow
}

// targetProximityScore resolves the target's current room and returns a
// step-down score based on distance from mob:
//
//	2.0 — same room
//	1.5 — adjacent room (one exit hop via mob's current room exits)
//	0.5 — same zone
//	0.1 — different zone
func targetProximityScore(mob *mobs.Mob, kind string, id int) float64 {
	targetRoomId := resolveTargetRoomId(kind, id)
	if targetRoomId == 0 {
		return 0.1 // target location unknown; small but nonzero
	}

	mobRoomId := mob.Character.RoomId

	// Same room.
	if targetRoomId == mobRoomId {
		return 2.0
	}

	// Adjacent room: check exits from mob's current room.
	if mobRoom := rooms.LoadRoom(mobRoomId); mobRoom != nil {
		for _, ex := range mobRoom.Exits {
			if ex.RoomId == targetRoomId {
				return 1.5
			}
		}
	}

	// Same zone check.
	targetRoom := rooms.LoadRoom(targetRoomId)
	if targetRoom != nil && mob.Character.Zone != "" && targetRoom.Zone == mob.Character.Zone {
		return 0.5
	}

	return 0.1
}

// resolveTargetRoomId returns the room id of the given target, or 0 if
// the target cannot be located.
func resolveTargetRoomId(kind string, id int) int {
	switch kind {
	case "mob":
		for _, instId := range mobs.GetAllMobInstanceIds() {
			if inst := mobs.GetInstance(instId); inst != nil && inst.MobId == mobs.MobId(id) {
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
