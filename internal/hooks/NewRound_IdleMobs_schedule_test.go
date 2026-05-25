package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func TestScheduleTick_NoScheduleReturnsEmptyPlan(t *testing.T) {
	mob := &mobs.Mob{ScheduleId: ""}

	plan := scheduleTickPlan(mob, 10)
	if plan.HasSchedule {
		t.Errorf("expected HasSchedule=false for empty schedule_id, got %+v", plan)
	}
}

func TestScheduleTick_UnknownScheduleReturnsEmptyPlan(t *testing.T) {
	mob := &mobs.Mob{ScheduleId: "definitely_not_real"}

	plan := scheduleTickPlan(mob, 10)
	if plan.HasSchedule {
		t.Errorf("expected HasSchedule=false for unknown schedule_id, got %+v", plan)
	}
}

func TestScheduleTick_WantsPathWhenAwayFromTarget(t *testing.T) {
	registerKerraScheduleForTest(t)

	mob := &mobs.Mob{ScheduleId: "thornwall_smith"}
	mob.Character.RoomId = 1111 // not at any segment target

	plan := scheduleTickPlan(mob, 10 /* forge hour */)
	if !plan.HasSchedule {
		t.Fatalf("expected HasSchedule=true, got %+v", plan)
	}
	if !plan.WantsPath {
		t.Errorf("expected WantsPath=true when away from target, got %+v", plan)
	}
	if plan.TargetRoom != 5678 {
		t.Errorf("expected target=5678, got %d", plan.TargetRoom)
	}
}

func TestScheduleTick_NoPathWhenAtTarget(t *testing.T) {
	registerKerraScheduleForTest(t)

	mob := &mobs.Mob{ScheduleId: "thornwall_smith"}
	mob.Character.RoomId = 5678 // at forge

	plan := scheduleTickPlan(mob, 10)
	if plan.WantsPath {
		t.Errorf("expected WantsPath=false when at target, got %+v", plan)
	}
}

func TestScheduleTick_SegmentTransitionDetected(t *testing.T) {
	registerKerraScheduleForTest(t)

	mob := &mobs.Mob{ScheduleId: "thornwall_smith"}
	mob.Character.RoomId = 5678
	// Simulate last tick being the forge segment (Start=9). Now hour=19 is tavern.
	mob.Character.SetMiscData("schedule_last_seg_start", 9)

	plan := scheduleTickPlan(mob, 19 /* tavern hour */)
	if !plan.SegmentChanged {
		t.Errorf("expected SegmentChanged=true on transition, got %+v", plan)
	}
	if plan.NewSegmentStart != 18 {
		t.Errorf("expected NewSegmentStart=18, got %d", plan.NewSegmentStart)
	}
}

func TestScheduleTick_SegmentTransitionResetsFailCount(t *testing.T) {
	registerKerraScheduleForTest(t)

	mob := &mobs.Mob{ScheduleId: "thornwall_smith"}
	mob.Character.RoomId = 1111 // not at any target

	// Pretend last tick was forge segment with 99 path failures (max retries
	// would be 20 in production config).
	mob.Character.SetMiscData("schedule_last_seg_start", 9)
	mob.Character.SetMiscData("schedule_path_fail_count", 99)

	// Now hour=19 (tavern) — segment transition. The 99-fail count from the
	// previous segment must NOT trigger WantsHomeFallback; the new segment
	// should get WantsPath against its target room (9012).
	plan := scheduleTickPlan(mob, 19)

	if !plan.SegmentChanged {
		t.Fatalf("expected SegmentChanged=true, got %+v", plan)
	}
	if plan.WantsHomeFallback {
		t.Errorf("expected WantsHomeFallback=false on segment transition with stale fail count, got %+v", plan)
	}
	if !plan.WantsPath {
		t.Errorf("expected WantsPath=true to new segment target, got %+v", plan)
	}
	if plan.TargetRoom != 9012 {
		t.Errorf("expected target=9012 (tavern), got %d", plan.TargetRoom)
	}
}

// registerKerraScheduleForTest injects the Kerra fixture and registers cleanup.
func registerKerraScheduleForTest(t *testing.T) {
	t.Helper()
	mobs.RegisterScheduleForTest(&mobs.Schedule{
		Id: "thornwall_smith",
		Segments: []mobs.ScheduleSegment{
			{Start: 6, End: 9, TargetRoom: 1234, IdleCommands: []string{"wake"}},
			{Start: 9, End: 18, TargetRoom: 5678, Activity: "craft", IdleCommands: []string{"hammer"}},
			{Start: 18, End: 22, TargetRoom: 9012, IdleCommands: []string{"sip"}},
			{Start: 22, End: 6, TargetRoom: 1234, IdleCommands: []string{"sleep"}},
		},
	})
	t.Cleanup(func() { mobs.UnregisterScheduleForTest("thornwall_smith") })
}

func TestScheduleTick_WantsSleep_AtTargetDuringSleepSegment(t *testing.T) {
	registerSleepyScheduleForTest(t)

	mob := &mobs.Mob{ScheduleId: "sleepy_test"}
	mob.Character.RoomId = 1234

	plan := scheduleTickPlan(mob, 23)
	if !plan.HasSchedule {
		t.Fatalf("expected HasSchedule=true")
	}
	if !plan.WantsSleep {
		t.Errorf("expected WantsSleep=true at target during sleep segment, got %+v", plan)
	}
}

func TestScheduleTick_WantsSleep_FalseWhenAwayFromTarget(t *testing.T) {
	registerSleepyScheduleForTest(t)

	mob := &mobs.Mob{ScheduleId: "sleepy_test"}
	mob.Character.RoomId = 5678

	plan := scheduleTickPlan(mob, 23)
	if plan.WantsSleep {
		t.Errorf("expected WantsSleep=false away from target, got %+v", plan)
	}
}

func TestScheduleTick_WantsSleep_FalseDuringGraceCooldown(t *testing.T) {
	registerSleepyScheduleForTest(t)
	util.SetRoundCountForTest(1000)
	defer util.SetRoundCountForTest(0)

	mob := &mobs.Mob{ScheduleId: "sleepy_test"}
	mob.Character.RoomId = 1234
	mob.Character.SetMiscData("schedule_wake_round", 980)

	plan := scheduleTickPlan(mob, 23)
	if plan.WantsSleep {
		t.Errorf("expected WantsSleep=false during grace cooldown, got %+v", plan)
	}
}

func TestScheduleTick_WantsSleep_TrueAfterGraceCooldown(t *testing.T) {
	registerSleepyScheduleForTest(t)
	util.SetRoundCountForTest(1100)
	defer util.SetRoundCountForTest(0)

	mob := &mobs.Mob{ScheduleId: "sleepy_test"}
	mob.Character.RoomId = 1234
	mob.Character.SetMiscData("schedule_wake_round", 1000)

	plan := scheduleTickPlan(mob, 23)
	if !plan.WantsSleep {
		t.Errorf("expected WantsSleep=true after grace cooldown, got %+v", plan)
	}
}

func TestScheduleTick_WantsWake_OnExitFromSleepSegment(t *testing.T) {
	registerSleepyScheduleForTest(t)

	mob := &mobs.Mob{ScheduleId: "sleepy_test"}
	mob.Character.RoomId = 1234
	mob.Character.SetMiscData("schedule_last_seg_start", 22)

	plan := scheduleTickPlan(mob, 6)
	if !plan.WantsWake {
		t.Errorf("expected WantsWake=true on exit from sleep segment, got %+v", plan)
	}
}

func registerSleepyScheduleForTest(t *testing.T) {
	t.Helper()
	mobs.RegisterScheduleForTest(&mobs.Schedule{
		Id: "sleepy_test",
		Segments: []mobs.ScheduleSegment{
			{Start: 6, End: 22, TargetRoom: 1234, Activity: "",
				IdleCommands: []string{"emote wakes."}},
			{Start: 22, End: 6, TargetRoom: 1234, Activity: "sleeping",
				IdleCommands: []string{"emote snores."}},
		},
	})
	t.Cleanup(func() { mobs.UnregisterScheduleForTest("sleepy_test") })
}

func TestApplySchedulePlan_StampsActivePatrolId_OnPatrolSegment(t *testing.T) {
	mobs.RegisterScheduleForTest(&mobs.Schedule{
		Id: "guard_sched",
		Segments: []mobs.ScheduleSegment{
			{Start: 0, End: 12, Activity: "patrol", PatrolId: "guard_patrol",
				IdleCommands: []string{"watches."}},
			{Start: 12, End: 24, TargetRoom: 9999, Activity: "",
				IdleCommands: []string{"sleeps."}},
		},
	})
	defer mobs.UnregisterScheduleForTest("guard_sched")

	mob := &mobs.Mob{ScheduleId: "guard_sched"}
	mob.Character.RoomId = 1000

	plan := scheduleTickPlan(mob, 8) // patrol segment

	applySchedulePlan(mob, plan)

	got := mob.Character.GetMiscData("active_patrol_id")
	if got == nil || got.(string) != "guard_patrol" {
		t.Errorf("expected active_patrol_id=guard_patrol, got %v", got)
	}
}

func TestApplySchedulePlan_StampsEmptyOnNonPatrolSegment(t *testing.T) {
	mobs.RegisterScheduleForTest(&mobs.Schedule{
		Id: "non_patrol_sched",
		Segments: []mobs.ScheduleSegment{
			{Start: 0, End: 24, TargetRoom: 9999, Activity: "",
				IdleCommands: []string{"x"}},
		},
	})
	defer mobs.UnregisterScheduleForTest("non_patrol_sched")

	mob := &mobs.Mob{ScheduleId: "non_patrol_sched"}
	mob.Character.RoomId = 1000

	plan := scheduleTickPlan(mob, 8)

	applySchedulePlan(mob, plan)

	// Stamp should be empty string (or unset == empty when read). Patrol
	// executor reads-and-clears so empty/unset both mean "no patrol".
	got := mob.Character.GetMiscData("active_patrol_id")
	if got != nil {
		if s, ok := got.(string); ok && s != "" {
			t.Errorf("expected empty active_patrol_id on non-patrol segment, got %q", s)
		}
	}
}
