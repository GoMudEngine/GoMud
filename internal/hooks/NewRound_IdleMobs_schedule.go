package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
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

	// WantsSleep: current segment is activity: sleeping AND mob is at
	// segment target_room AND not within ScheduleWakeGraceRounds of a
	// recent wake event. Idempotent per-tick check; safe to fire even
	// if mob is already sleeping (applySchedulePlan checks).
	WantsSleep bool

	// WantsWake: transitioning OUT of an activity: sleeping segment
	// (the prior segment had activity sleeping; this tick's segment
	// does not). Triggers explicit CancelBuffsWithFlag(Sleeping).
	WantsWake bool

	// Chunk 3.4: current segment patrol context. Empty for non-patrol
	// segments. applySchedulePlan stamps `active_patrol_id` MiscData
	// so the patrol executor (NewRound_IdleMobs_patrol.go) can consume
	// it the same tick.
	ActivePatrolId string
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

	// Chunk 3.3: detect wake transition (prior segment was activity: sleeping;
	// new is not).
	if plan.SegmentChanged {
		if prior := s.SegmentByStart(lastSegStart); prior != nil &&
			prior.Activity == "sleeping" && seg.Activity != "sleeping" {
			plan.WantsWake = true
		}
	}

	// Chunk 3.3: detect sleep desire (current segment is activity: sleeping;
	// at target; past grace cooldown).
	if seg.Activity == "sleeping" && mob.Character.RoomId == seg.TargetRoom {
		lastWoken := getMiscDataInt(&mob.Character, "schedule_wake_round")
		grace := int(configs.GetBalanceConfig().ScheduleWakeGraceRounds)
		if lastWoken == 0 || int(util.GetRoundCount())-lastWoken >= grace {
			plan.WantsSleep = true
		}
	}

	// Skip pathing for patrol segments — the patrol executor (T8/T9)
	// handles all movement. TargetRoom is 0 for patrol segments; without
	// this guard we'd compute WantsPath=true with TargetRoom=0 and the
	// applier would queue `pathto 0` every tick.
	if seg.Activity != "patrol" && mob.Character.RoomId != seg.TargetRoom {
		var fails int
		if !plan.SegmentChanged {
			fails = getMiscDataInt(&mob.Character, "schedule_path_fail_count")
		}
		// On segment transition, fails is treated as 0 — the new segment
		// gets a clean retry budget against its new target_room.
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

	// Chunk 3.4: capture patrol context so applySchedulePlan can stamp it.
	if seg.Activity == "patrol" {
		plan.ActivePatrolId = seg.PatrolId
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

	// Chunk 3.3: wake first — clear stale sleep from a prior sleep segment.
	if plan.WantsWake {
		mob.Character.CancelBuffsWithFlag(buffs.Sleeping)
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
	// Chunk 3.3: sleep last — only if at target and not already sleeping.
	if plan.WantsSleep && !mob.Character.HasBuffFlag(buffs.Sleeping) {
		mob.Command("sleep")
	}

	// Suppress wander while a schedule is active.
	mob.MaxWander = 0

	// Chunk 3.4: stamp active_patrol_id for the patrol executor to consume.
	// Empty string is fine — patrol executor reads-and-clears each tick,
	// so an empty stamp simply means "no active patrol context this tick."
	mob.Character.SetMiscData("active_patrol_id", plan.ActivePatrolId)
}
