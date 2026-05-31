package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func init() {
	RegisterPlanner("hunt_bounty_target", huntBountyTargetPlanner)
}

// huntDecision is the pure pursuit decision (unit-testable):
//
//	jailed target → "" (hold: loiter; never enter cell or engage)
//	hunter in target's room → "attack @<uid>"
//	else → "pathto <targetRoom>" (closing chase)
func huntDecision(targetJailed bool, hunterRoom, targetRoom, targetUserId int) (string, BTreeStatus) {
	if targetJailed {
		return "", StatusRunning
	}
	if hunterRoom == targetRoom {
		return "attack @" + strconv.Itoa(targetUserId), StatusRunning
	}
	return "pathto " + strconv.Itoa(targetRoom), StatusRunning
}

func huntBountyTargetPlanner(mob *mobs.Mob, _ *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	// Read per-hunter target from instance MiscData, not from goal params.
	// The goal is a param-less template-level intent marker; spawnHunter
	// stamps bh_target_user_id onto each hunter instance's MiscData so
	// concurrent hunters can each pursue a different target.
	uid := mobMiscIntOr(mob, "bh_target_user_id", 0)
	if uid == 0 {
		// No target stamped yet — hold until the dispatch manager stamps one.
		return PlanResult{Status: StatusRunning}
	}
	u := users.GetByUserId(uid)
	if u == nil {
		// target offline — hold; the dispatch manager suspends/ends the hunt
		return PlanResult{Status: StatusRunning}
	}
	// Jailed detection: no buffs.Jailed flag constant exists; detect via the
	// jail_until_round MiscData key stamped by justice.ExecuteArrest.
	jailed := u.Character.GetMiscData("jail_until_round") != nil
	cmd, status := huntDecision(jailed, mob.Character.RoomId, u.Character.RoomId, uid)
	return PlanResult{Command: cmd, Status: status}
}
