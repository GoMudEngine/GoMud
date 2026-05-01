package behaviortree

// caravan_reset.go — exported helper for admin-driven caravan state reset.
//
// ResetAllCaravanStates lives here (inside the behaviortree package) so
// that it can call the unexported resetCaravanState and access BTreeState
// without exposing internals to callers. The admin command in
// usercommands/admin.caravan.go calls this function; it never touches
// BTreeState directly.

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// ResetAllCaravanStates iterates every live mob instance, finds any whose
// BTreeState contains a "caravan_state" key (i.e. caravan leaders), and
// resets each one to ThornwallDwell. Returns the count of mobs reset.
//
// Safe to call at any time; mobs without a caravan state are untouched.
func ResetAllCaravanStates() int {
	resetCount := 0
	for _, instId := range mobs.GetAllMobInstanceIds() {
		mob := mobs.GetInstance(instId)
		if mob == nil {
			continue
		}
		bs, ok := mob.BTreeState.(*BehaviorState)
		if !ok || bs == nil {
			continue
		}
		if bs.GetString("caravan_state") == "" {
			continue
		}
		prevState := bs.GetString("caravan_state")
		resetCaravanState(bs)
		resetCount++
		mudlog.Info("caravan state reset by admin",
			"instanceId", instId,
			"mobName", mob.Character.Name,
			"prevState", prevState,
		)
	}
	return resetCount
}

// ResetCaravanStateByInstanceId resets the caravan state for a single mob
// instance. Returns true if the mob was found and had a caravan state,
// false if the instance ID was not found or the mob is not a caravan leader.
func ResetCaravanStateByInstanceId(instanceId int) bool {
	mob := mobs.GetInstance(instanceId)
	if mob == nil {
		return false
	}
	bs, ok := mob.BTreeState.(*BehaviorState)
	if !ok || bs == nil {
		return false
	}
	if bs.GetString("caravan_state") == "" {
		return false
	}
	prevState := bs.GetString("caravan_state")
	resetCaravanState(bs)
	mudlog.Info("caravan state reset by admin (targeted)",
		"instanceId", instanceId,
		"mobName", mob.Character.Name,
		"prevState", prevState,
	)
	return true
}
