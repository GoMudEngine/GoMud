package behaviortree

// caravan_reset.go — exported helper for admin-driven caravan state reset.
//
// Chunk 3.7: the legacy caravan_state BTreeState key is gone. Reset now
// means "rewind the leader's patrol to waypoint 0 and clear dwell/path
// counters so a fresh cycle starts at the Thornwall depot."

import (
	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// ResetAllCaravanStates iterates every live mob instance, finds any whose
// PatrolId == caravan.CaravanPatrolId (i.e. caravan leaders), and resets
// each one to waypoint 0. Returns the count of mobs reset.
func ResetAllCaravanStates() int {
	resetCount := 0
	for _, instId := range mobs.GetAllMobInstanceIds() {
		mob := mobs.GetInstance(instId)
		if mob == nil || mob.PatrolId != caravan.CaravanPatrolId {
			continue
		}
		prevIdx := 0
		if v, ok := mob.Character.GetMiscData("patrol_waypoint_idx").(int); ok {
			prevIdx = v
		}
		mob.Character.SetMiscData("patrol_waypoint_idx", 0)
		mob.Character.SetMiscData("patrol_direction", +1)
		mob.Character.SetMiscData("patrol_dwell_remaining", 0)
		mob.Character.SetMiscData("patrol_path_fail_count", 0)
		resetCount++
		mudlog.Info("caravan state reset by admin",
			"instanceId", instId,
			"mobName", mob.Character.Name,
			"prevWaypointIdx", prevIdx,
		)
	}
	return resetCount
}

// ResetCaravanStateByInstanceId resets the caravan state for a single mob
// instance. Returns true if the mob was found and is a caravan leader,
// false otherwise.
func ResetCaravanStateByInstanceId(instanceId int) bool {
	mob := mobs.GetInstance(instanceId)
	if mob == nil || mob.PatrolId != caravan.CaravanPatrolId {
		return false
	}
	prevIdx := 0
	if v, ok := mob.Character.GetMiscData("patrol_waypoint_idx").(int); ok {
		prevIdx = v
	}
	mob.Character.SetMiscData("patrol_waypoint_idx", 0)
	mob.Character.SetMiscData("patrol_direction", +1)
	mob.Character.SetMiscData("patrol_dwell_remaining", 0)
	mob.Character.SetMiscData("patrol_path_fail_count", 0)
	mudlog.Info("caravan state reset by admin (targeted)",
		"instanceId", instanceId,
		"mobName", mob.Character.Name,
		"prevWaypointIdx", prevIdx,
	)
	return true
}
