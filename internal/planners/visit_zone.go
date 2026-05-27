package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

const visitZoneNextHopKey = "plan:visit-zone:next_hop_zone"

func init() {
	RegisterPlanner("visit-zone", visitZonePlanner)
}

// visitZonePlanner: walk to an exit-room leading toward target zone.
// Multi-hop uses a simple "any unvisited adjacent zone" heuristic.
func visitZonePlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	target := goalParamStringOr(goal, "target_zone", "")
	if target == "" {
		return PlanResult{Status: StatusFailure}
	}

	// Already visited / currently in target → Success (predicate will fire).
	if mob.VisitedZones[target] {
		return PlanResult{Status: StatusSuccess}
	}
	if mob.Character.Zone == target {
		return PlanResult{Status: StatusSuccess}
	}

	// Target is adjacent? Walk into it directly.
	for _, adj := range zoneAdjacentTo(mob.Character.Zone) {
		if adj == target {
			exitRoom, ok := exitRoomToward(mob, target)
			if !ok {
				return PlanResult{Status: StatusFailure}
			}
			return PlanResult{Command: "pathto " + strconv.Itoa(exitRoom), Status: StatusRunning}
		}
	}

	// Multi-hop: pick "any unvisited adjacent zone" heuristic.
	hop := mobMiscStringOr(mob, visitZoneNextHopKey, "")
	if hop == "" {
		for _, adj := range zoneAdjacentTo(mob.Character.Zone) {
			if !mob.VisitedZones[adj] {
				hop = adj
				break
			}
		}
		if hop == "" {
			return PlanResult{Status: StatusFailure}
		}
		mobSetMisc(mob, visitZoneNextHopKey, hop)
	}
	exitRoom, ok := exitRoomToward(mob, hop)
	if !ok {
		return PlanResult{Status: StatusFailure}
	}
	return PlanResult{Command: "pathto " + strconv.Itoa(exitRoom), Status: StatusRunning}
}

// exitRoomToward returns the room id in mob's current zone that has an
// exit leading INTO targetZone. ok=false if no such room exists.
//
// Walk every room in mob's zone (via ZoneConfig.RoomIds), inspect each
// room's Exits map, and return the source room id for the first exit
// whose destination room belongs to targetZone.
func exitRoomToward(mob *mobs.Mob, targetZone string) (int, bool) {
	if mob == nil || targetZone == "" {
		return 0, false
	}
	zc := rooms.GetZoneConfig(mob.Character.Zone)
	if zc == nil {
		return 0, false
	}
	for rid := range zc.RoomIds {
		room := rooms.LoadRoom(rid)
		if room == nil {
			continue
		}
		for _, ex := range room.Exits {
			dest := rooms.LoadRoom(ex.RoomId)
			if dest == nil {
				continue
			}
			if dest.Zone == targetZone {
				return rid, true
			}
		}
	}
	return 0, false
}
