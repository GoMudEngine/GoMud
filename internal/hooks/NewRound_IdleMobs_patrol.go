package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
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
	NextDirection     int  // +1 / -1; only meaningful for yo-yo
	NextDwellRounds   int  // dwell for the new waypoint after advance
	WantsHomeFallback bool // after MaxPathRetries
	FailureMessage    string
	WantsComplete     bool // chunk 3.8: oneshot patrol reached its terminal waypoint
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
		// Chunk 3.8: oneshot terminal — at the last waypoint of a
		// oneshot patrol, with dwell expired, signal completion
		// instead of advancing.
		if p.LoopShape == "oneshot" && idx == len(p.Waypoints)-1 {
			plan.WantsComplete = true
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
	// Chunk 3.7: honor per-patrol MaxPathRetries override; 0 = use global default.
	maxRetries := p.MaxPathRetries
	if maxRetries == 0 {
		maxRetries = int(configs.GetBalanceConfig().ScheduleMaxPathRetries)
	}
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
// activePatrolId is threaded in so the WantsDwellWait and WantsAdvance
// branches can emit PatrolWaypointArrival events (chunk 3.7).
func applyPatrolPlan(mob *mobs.Mob, plan patrolPlan, activePatrolId string) {
	if !plan.HasPatrol {
		return
	}

	switch {
	case plan.WantsComplete:
		events.AddToQueue(events.PatrolCompleted{
			MobInstanceId: mob.InstanceId,
			PatrolId:      activePatrolId,
			RoomId:        mob.Character.RoomId,
		})
		mobs.ClearOneshotPatrol(mob)
		return

	case plan.WantsHomeFallback:
		mudlog.Warn("patrol", "msg", plan.FailureMessage)
		mob.Command("pathto home")
		mob.Character.SetMiscData("patrol_path_fail_count", 0)
		return

	case plan.WantsAdvance:
		// Chunk 3.7: emit arrival event for zero-dwell waypoints, which
		// skip the WantsDwellWait branch. Guard on RoomId == waypoint room
		// so we only fire when actually at the waypoint, not on a path-walk
		// step that happened to call applyPatrolPlan with WantsAdvance.
		p := mobs.GetPatrol(activePatrolId)
		if p != nil {
			idx := getMiscDataInt(&mob.Character, "patrol_waypoint_idx")
			if idx >= 0 && idx < len(p.Waypoints) &&
				p.Waypoints[idx].DwellRounds == 0 &&
				mob.Character.RoomId == p.Waypoints[idx].Room {
				events.AddToQueue(events.PatrolWaypointArrival{
					MobInstanceId: mob.InstanceId,
					PatrolId:      activePatrolId,
					WaypointIdx:   idx,
					RoomId:        mob.Character.RoomId,
					ArrivalEvent:  p.Waypoints[idx].ArrivalEvent,
				})
			}
		}
		mob.Character.SetMiscData("patrol_waypoint_idx", plan.NextWaypointIdx)
		mob.Character.SetMiscData("patrol_direction", plan.NextDirection)
		mob.Character.SetMiscData("patrol_dwell_remaining", plan.NextDwellRounds)
		mob.Character.SetMiscData("patrol_path_fail_count", 0)
		return

	case plan.WantsDwellWait:
		current := getMiscDataInt(&mob.Character, "patrol_dwell_remaining")
		// Chunk 3.7: emit the per-waypoint arrival event on the first dwell
		// tick. Detected by comparing current dwell to the authored value;
		// they match only on the tick immediately after arrival (before
		// the decrement below). Idempotent: subsequent ticks have a smaller
		// current value and won't re-emit.
		p := mobs.GetPatrol(activePatrolId)
		if p != nil {
			idx := getMiscDataInt(&mob.Character, "patrol_waypoint_idx")
			if idx >= 0 && idx < len(p.Waypoints) && current == p.Waypoints[idx].DwellRounds {
				events.AddToQueue(events.PatrolWaypointArrival{
					MobInstanceId: mob.InstanceId,
					PatrolId:      activePatrolId,
					WaypointIdx:   idx,
					RoomId:        mob.Character.RoomId,
					ArrivalEvent:  p.Waypoints[idx].ArrivalEvent,
				})
			}
		}
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
