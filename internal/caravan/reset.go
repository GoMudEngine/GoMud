package caravan

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// ResetLeaderToDepot performs a hard reset of a caravan leader's
// patrol state: teleports the leader to wp0 (the Thornwall depot),
// restores the wp0 dwell counter so the leader actually pauses
// there, and force-regroups the rest of the crew to the leader's
// new room.
//
// Used by the admin `caravan reset` command (via the thin wrapper
// in internal/behaviortree/caravan_reset.go).
//
// Returns true if the reset was performed, false if the mob is not
// a caravan leader (PatrolId mismatch) or the patrol is not loaded.
//
// Chunk 3.7 follow-up FUC: pre-fix the reset only mutated MiscData,
// leaving the leader mid-cycle and crew scattered until the next
// natural wp0 arrival. Now reset is "everything snaps back to depot
// immediately."
func ResetLeaderToDepot(leader *mobs.Mob) bool {
	if leader == nil || leader.PatrolId != CaravanPatrolId {
		return false
	}
	p := mobs.GetPatrol(CaravanPatrolId)
	if p == nil || len(p.Waypoints) == 0 {
		return false
	}
	wp0 := p.Waypoints[0]

	// 1. Teleport the leader to wp0 if not already there.
	if leader.Character.RoomId != wp0.Room {
		oldRoomId := leader.Character.RoomId
		oldRoom := rooms.LoadRoom(oldRoomId)
		newRoom := rooms.LoadRoom(wp0.Room)
		if newRoom != nil {
			if oldRoom != nil {
				oldRoom.RemoveMob(leader.InstanceId)
			}
			newRoom.AddMob(leader.InstanceId) // sets RoomId internally
			mudlog.Info("caravan leader reset teleport",
				"leader", leader.Character.Name,
				"from_room", oldRoomId,
				"to_room", wp0.Room,
			)
		}
	}

	// 2. Reset patrol MiscData. Crucially, dwell_remaining is set
	//    to the authored wp0 value (not 0) — without this the
	//    leader would arrive and immediately advance to wp1 instead
	//    of pausing at the depot.
	leader.Character.SetMiscData("patrol_waypoint_idx", 0)
	leader.Character.SetMiscData("patrol_direction", +1)
	leader.Character.SetMiscData("patrol_dwell_remaining", wp0.DwellRounds)
	leader.Character.SetMiscData("patrol_path_fail_count", 0)

	// 3. Regroup the rest of the crew to the leader's new (depot) room.
	ForceRegroupCrew(leader)

	return true
}
