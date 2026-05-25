package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// schedulePlan describes what the executor wants to do for a mob this tick.
// Extracted so the decision logic is unit-testable without driving the full
// IdleMobs loop.
type schedulePlan struct {
	HasSchedule       bool
	SegmentChanged    bool
	NewSegmentStart   int
	NewIdleCommands   []string
	WantsPath         bool
	TargetRoom        int
	WantsHomeFallback bool   // true after MaxPathRetries failures
	FailureMessage    string // set when WantsHomeFallback fires
}

// scheduleTickPlan computes the desired tick action for a scheduled mob. Pure
// over its inputs (mob.ScheduleId, mob.Character.RoomId, hour, MiscData) —
// safe to call from tests with stubbed registry state.
func scheduleTickPlan(mob *mobs.Mob, hour24 int) schedulePlan {
	plan := schedulePlan{}
	if mob == nil || mob.ScheduleId == "" {
		return plan
	}
	s := mobs.GetSchedule(mob.ScheduleId)
	if s == nil {
		return plan
	}
	seg := s.CurrentSegment(hour24)
	if seg == nil {
		return plan
	}
	plan.HasSchedule = true

	// schedule_last_seg_start defaults to -1 (sentinel) when unset so that a
	// freshly spawned mob always triggers a SegmentChanged on its first tick.
	// The existing getMiscDataInt helper returns 0 for unset keys, so we
	// subtract one from the raw result: an unset key returns 0, which we
	// treat as -1 by checking whether the MiscData key is nil directly.
	rawLast := mob.Character.GetMiscData("schedule_last_seg_start")
	var lastSegStart int
	if rawLast == nil {
		lastSegStart = -1 // sentinel: never been set, always trigger change
	} else {
		lastSegStart = getMiscDataInt(&mob.Character, "schedule_last_seg_start")
	}
	if seg.Start != lastSegStart {
		plan.SegmentChanged = true
		plan.NewSegmentStart = seg.Start
		plan.NewIdleCommands = seg.IdleCommands
	}

	if mob.Character.RoomId != seg.TargetRoom {
		fails := getMiscDataInt(&mob.Character, "schedule_path_fail_count")
		maxRetries := int(configs.GetBalanceConfig().ScheduleMaxPathRetries)
		if maxRetries > 0 && fails >= maxRetries {
			plan.WantsHomeFallback = true
			plan.FailureMessage = fmt.Sprintf(
				"scheduled mob %d (%s) unreachable target_room %d after %d retries; falling back to home",
				mob.MobId, mob.Character.Name, seg.TargetRoom, fails)
		} else {
			plan.WantsPath = true
			plan.TargetRoom = seg.TargetRoom
		}
	}
	return plan
}

// applySchedulePlan mutates the mob (clears path, swaps idle pool, queues
// pathto, increments retry counter) based on the plan. Side-effecting; not
// pure. Called only from the live IdleMobs hook.
func applySchedulePlan(mob *mobs.Mob, plan schedulePlan) {
	if !plan.HasSchedule {
		return
	}
	if plan.SegmentChanged {
		mob.Character.SetMiscData("schedule_last_seg_start", plan.NewSegmentStart)
		mob.Character.SetMiscData("schedule_path_fail_count", 0) // reset on transition
		mob.Path.Clear()
		mob.IdleCommands = plan.NewIdleCommands
	}
	if plan.WantsHomeFallback {
		mudlog.Warn("schedule", "msg", plan.FailureMessage)
		mob.Command("pathto home")
		mob.Character.SetMiscData("schedule_path_fail_count", 0)
		return
	}
	if plan.WantsPath {
		// Only queue a new pathto if no path is in flight; let the existing
		// walker consume the current one if it still resolves.
		if mob.Path.Len() == 0 && mob.Path.Current() == nil {
			mob.Command(fmt.Sprintf("pathto %d", plan.TargetRoom))
			// Increment retry counter. Reset to zero when the mob reaches the
			// target room (WantsPath=false branch below).
			fails := getMiscDataInt(&mob.Character, "schedule_path_fail_count")
			mob.Character.SetMiscData("schedule_path_fail_count", fails+1)
		}
	} else {
		// At target — reset retry counter.
		mob.Character.SetMiscData("schedule_path_fail_count", 0)
	}
	// Suppress wander while a schedule is active.
	mob.MaxWander = 0
}
