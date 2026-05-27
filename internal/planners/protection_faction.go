package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	RegisterPlanner("protection-faction", protectionFactionPlanner)
}

// protectionFactionPlanner: defend faction members in zone.
//   - Member in combat in same room → attack their aggressor.
//   - Member in combat in zone (different room) → pathto member's room.
//   - Hostile mob in zone (no member-in-combat) → pathto hostile + attack.
//   - Zone calm → Success (no action; goal stays current — predicate never satisfies).
//   - No members in zone → Failure.
func protectionFactionPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	factionId := goalParamStringOr(goal, "faction_id", "")
	if factionId == "" {
		return PlanResult{Status: StatusFailure}
	}

	// Find a faction member currently in combat in zone.
	if member, ok := findFactionMemberInZone(mob, factionId, true); ok {
		aggressor := aggroToName(member.Character.Aggro)
		if member.Character.RoomId == mob.Character.RoomId && aggressor != "" {
			return PlanResult{Command: "attack " + aggressor, Status: StatusRunning}
		}
		return PlanResult{Command: "pathto " + strconv.Itoa(member.Character.RoomId), Status: StatusRunning}
	}

	// Any members in zone at all?
	if _, ok := findFactionMemberInZone(mob, factionId, false); !ok {
		return PlanResult{Status: StatusFailure}
	}

	// Hostile mob in zone but no member-in-combat → preempt.
	if hostile, ok := findHostileInZone(mob); ok {
		if hostile.Character.RoomId == mob.Character.RoomId {
			return PlanResult{Command: "attack " + hostile.Character.Name, Status: StatusRunning}
		}
		return PlanResult{Command: "pathto " + strconv.Itoa(hostile.Character.RoomId), Status: StatusRunning}
	}

	// Zone calm with members present → hold.
	return PlanResult{Status: StatusSuccess}
}
