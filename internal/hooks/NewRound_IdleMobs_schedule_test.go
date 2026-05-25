package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
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
