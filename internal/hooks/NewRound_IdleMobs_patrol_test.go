package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func registerTestPatrol(t *testing.T) {
	t.Helper()
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:        "test_patrol",
		LoopShape: "strict",
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 100, DwellRounds: 5},
			{Room: 101, DwellRounds: 0},
			{Room: 102, DwellRounds: 10},
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest("test_patrol") })
}

func TestPatrolTickPlan_WantsPathWhenNotAtTarget(t *testing.T) {
	registerTestPatrol(t)

	mob := &mobs.Mob{}
	mob.Character.RoomId = 999 // not at any waypoint

	plan := patrolTickPlan(mob, "test_patrol")

	if !plan.HasPatrol {
		t.Fatalf("expected HasPatrol=true")
	}
	if !plan.WantsPath {
		t.Errorf("expected WantsPath=true when away from target, got %+v", plan)
	}
	if plan.TargetRoom != 100 {
		t.Errorf("expected initial target=100 (waypoint[0]), got %d", plan.TargetRoom)
	}
}

func TestPatrolTickPlan_WantsDwellWaitAtTarget(t *testing.T) {
	registerTestPatrol(t)

	mob := &mobs.Mob{}
	mob.Character.RoomId = 100 // at waypoint[0]
	mob.Character.SetMiscData("patrol_dwell_remaining", 3)

	plan := patrolTickPlan(mob, "test_patrol")

	if !plan.WantsDwellWait {
		t.Errorf("expected WantsDwellWait=true at target with dwell>0, got %+v", plan)
	}
}

func TestPatrolTickPlan_WantsAdvanceAtTargetWithZeroDwell(t *testing.T) {
	registerTestPatrol(t)

	mob := &mobs.Mob{}
	mob.Character.RoomId = 101 // at waypoint[1] which has DwellRounds=0
	mob.Character.SetMiscData("patrol_waypoint_idx", 1)
	mob.Character.SetMiscData("patrol_dwell_remaining", 0)

	plan := patrolTickPlan(mob, "test_patrol")

	if !plan.WantsAdvance {
		t.Errorf("expected WantsAdvance=true at target with dwell=0, got %+v", plan)
	}
	if plan.NextWaypointIdx != 2 {
		t.Errorf("expected NextWaypointIdx=2, got %d", plan.NextWaypointIdx)
	}
}

func TestPatrolTickPlan_WantsHomeFallbackAfterMaxRetries(t *testing.T) {
	registerTestPatrol(t)

	mob := &mobs.Mob{}
	mob.Character.RoomId = 999 // not at target
	mob.Character.SetMiscData("patrol_path_fail_count", 99) // way past default 20

	plan := patrolTickPlan(mob, "test_patrol")

	if !plan.WantsHomeFallback {
		t.Errorf("expected WantsHomeFallback=true after max retries, got %+v", plan)
	}
}

func TestApplyPatrolPlan_IncrementsRetryOnWantsPath(t *testing.T) {
	registerTestPatrol(t)

	mob := &mobs.Mob{}
	mob.Character.RoomId = 999

	plan := patrolTickPlan(mob, "test_patrol")
	applyPatrolPlan(mob, plan)

	got := mob.Character.GetMiscData("patrol_path_fail_count")
	if got == nil || got.(int) != 1 {
		t.Errorf("expected patrol_path_fail_count=1 after WantsPath apply, got %v", got)
	}
}

func TestApplyPatrolPlan_ResetsRetryAtTarget(t *testing.T) {
	registerTestPatrol(t)

	mob := &mobs.Mob{}
	mob.Character.RoomId = 100 // at waypoint[0]
	mob.Character.SetMiscData("patrol_path_fail_count", 5)
	mob.Character.SetMiscData("patrol_dwell_remaining", 3) // will WantsDwellWait

	plan := patrolTickPlan(mob, "test_patrol")
	applyPatrolPlan(mob, plan)

	got := mob.Character.GetMiscData("patrol_path_fail_count")
	if got == nil || got.(int) != 0 {
		t.Errorf("expected patrol_path_fail_count reset to 0 at target, got %v", got)
	}
}

func TestPatrolTickPlan_RespectsPerPatrolMaxRetries(t *testing.T) {
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:             "test_high_retry_patrol",
		LoopShape:      "strict",
		MaxPathRetries: 40,
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 100, DwellRounds: 0},
			{Room: 200, DwellRounds: 0},
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest("test_high_retry_patrol") })

	mob := &mobs.Mob{PatrolId: "test_high_retry_patrol"}
	mob.Character.RoomId = 999 // not at waypoint 0 (room 100) — wants path

	// At fail count 39, should still want path (under custom 40 cap)
	mob.Character.SetMiscData("patrol_path_fail_count", 39)
	plan := patrolTickPlan(mob, "test_high_retry_patrol")
	if plan.WantsHomeFallback {
		t.Errorf("expected WantsHomeFallback=false at fails=39 with cap=40, got %+v", plan)
	}
	if !plan.WantsPath {
		t.Errorf("expected WantsPath=true at fails=39 under cap=40, got %+v", plan)
	}

	// At fail count 40, should trigger home fallback
	mob.Character.SetMiscData("patrol_path_fail_count", 40)
	plan = patrolTickPlan(mob, "test_high_retry_patrol")
	if !plan.WantsHomeFallback {
		t.Errorf("expected WantsHomeFallback=true at fails=40 with cap=40, got %+v", plan)
	}
}
