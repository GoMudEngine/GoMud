package behaviortree

// caravan_reset.go — exported helper for admin-driven caravan state reset.
//
// Chunk 3.7: the legacy caravan_state BTreeState key is gone. Reset now
// means "snap the leader and crew back to the Thornwall depot with a
// fresh wp0 dwell so the cycle restarts cleanly." All the actual work
// lives in caravan.ResetLeaderToDepot; this file is the cross-package
// adapter so admincommands can call into it without importing
// internal/caravan directly.

import (
	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// ResetAllCaravanStates iterates every live mob instance, finds any whose
// PatrolId == caravan.CaravanPatrolId (i.e. caravan leaders), and resets
// each one to wp0 via caravan.ResetLeaderToDepot. Returns the count of
// mobs reset.
func ResetAllCaravanStates() int {
	resetCount := 0
	for _, instId := range mobs.GetAllMobInstanceIds() {
		mob := mobs.GetInstance(instId)
		if mob == nil || mob.PatrolId != caravan.CaravanPatrolId {
			continue
		}
		if !caravan.ResetLeaderToDepot(mob) {
			continue
		}
		resetCount++
		mudlog.Info("caravan state reset by admin",
			"instanceId", instId,
			"mobName", mob.Character.Name,
		)
	}
	return resetCount
}

// ResetCaravanStateByInstanceId resets the caravan state for a single mob
// instance. Returns true if the mob was found and is a caravan leader,
// false otherwise.
func ResetCaravanStateByInstanceId(instanceId int) bool {
	mob := mobs.GetInstance(instanceId)
	if mob == nil {
		return false
	}
	if !caravan.ResetLeaderToDepot(mob) {
		return false
	}
	mudlog.Info("caravan state reset by admin (targeted)",
		"instanceId", instanceId,
		"mobName", mob.Character.Name,
	)
	return true
}
