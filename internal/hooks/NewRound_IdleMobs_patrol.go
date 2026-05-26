package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// patrolPlan describes what the patrol executor wants to do for a
// mob this tick. Extracted so the decision logic is unit-testable
// without driving the full IdleMobs loop.
type patrolPlan struct {
	HasPatrol         bool
	WantsDwellWait    bool // mob at current target waypoint AND dwell > 0
	WantsPath         bool // mob not at current target waypoint
	TargetRoom        int
	WantsAdvance      bool // dwell expired (or 0); advance this tick
	NextWaypointIdx   int
	NextDirection     int // +1 / -1; only meaningful for yo-yo
	NextDwellRounds   int // dwell for the new waypoint after advance
	WantsHomeFallback bool // after MaxPathRetries
	FailureMessage    string
}

// patrolTickPlan computes the desired tick action for a patrol mob.
// Pure over its inputs (mob.Character.RoomId, MiscData, patrolId) —
// safe to call from tests with stubbed registry state.
func patrolTickPlan(mob *mobs.Mob, patrolId string) patrolPlan {
	plan := patrolPlan{}
	if mob == nil || patrolId == "" {
		return plan
	}
	p := mobs.GetPatrol(patrolId)
	if p == nil || len(p.Waypoints) == 0 {
		return plan
	}
	plan.HasPatrol = true

	idx := getMiscDataInt(&mob.Character, "patrol_waypoint_idx")
	if idx < 0 || idx >= len(p.Waypoints) {
		idx = 0 // first-tick or stale-after-patrol-shrink: reset to start
	}
	dir := getMiscDataInt(&mob.Character, "patrol_direction")
	if dir == 0 {
		dir = +1
	}
	dwellRemaining := getMiscDataInt(&mob.Character, "patrol_dwell_remaining")
	failCount := getMiscDataInt(&mob.Character, "patrol_path_fail_count")

	currentWaypoint := &p.Waypoints[idx]

	if mob.Character.RoomId == currentWaypoint.Room {
		// At target. Dwell or advance?
		if dwellRemaining > 0 {
			plan.WantsDwellWait = true
			return plan
		}
		// Advance.
		nextIdx, nextDir := p.NextWaypoint(idx, dir)
		plan.WantsAdvance = true
		plan.NextWaypointIdx = nextIdx
		plan.NextDirection = nextDir
		plan.NextDwellRounds = p.Waypoints[nextIdx].DwellRounds
		return plan
	}

	// Not at target — path or fallback.
	maxRetries := int(configs.GetBalanceConfig().ScheduleMaxPathRetries)
	if maxRetries > 0 && failCount >= maxRetries {
		plan.WantsHomeFallback = true
		plan.FailureMessage = fmt.Sprintf(
			"patrol mob %d (%s) unreachable waypoint room %d after %d retries; falling back to home",
			mob.MobId, mob.Character.Name, currentWaypoint.Room, failCount)
		return plan
	}
	plan.WantsPath = true
	plan.TargetRoom = currentWaypoint.Room
	return plan
}

// applyPatrolPlan mutates the mob (updates MiscData, queues commands)
// based on the plan. Side-effecting; not pure.
func applyPatrolPlan(mob *mobs.Mob, plan patrolPlan) {
	if !plan.HasPatrol {
		return
	}

	switch {
	case plan.WantsHomeFallback:
		mudlog.Warn("patrol", "msg", plan.FailureMessage)
		mob.Command("pathto home")
		mob.Character.SetMiscData("patrol_path_fail_count", 0)
		return

	case plan.WantsAdvance:
		mob.Character.SetMiscData("patrol_waypoint_idx", plan.NextWaypointIdx)
		mob.Character.SetMiscData("patrol_direction", plan.NextDirection)
		mob.Character.SetMiscData("patrol_dwell_remaining", plan.NextDwellRounds)
		mob.Character.SetMiscData("patrol_path_fail_count", 0)
		return

	case plan.WantsDwellWait:
		current := getMiscDataInt(&mob.Character, "patrol_dwell_remaining")
		if current > 0 {
			mob.Character.SetMiscData("patrol_dwell_remaining", current-1)
		}
		// At-target → reset retry counter.
		mob.Character.SetMiscData("patrol_path_fail_count", 0)
		return

	case plan.WantsPath:
		// Queue pathto if no path is in flight (matches schedule executor pattern).
		if mob.Path.Len() == 0 && mob.Path.Current() == nil {
			mob.Command(fmt.Sprintf("pathto %d", plan.TargetRoom))
		}
		fails := getMiscDataInt(&mob.Character, "patrol_path_fail_count")
		mob.Character.SetMiscData("patrol_path_fail_count", fails+1)
		return
	}
}
