package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
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
	applyPatrolPlan(mob, plan, "test_patrol")

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
	applyPatrolPlan(mob, plan, "test_patrol")

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

// TestApplyPatrolPlan_EmitsArrivalEventOnFirstDwellTick verifies that
// PatrolWaypointArrival fires exactly once when a mob first enters a
// WantsDwellWait tick (dwell_remaining == authored DwellRounds). Subsequent
// ticks at the same waypoint must NOT re-emit. Chunk 3.7.
func TestApplyPatrolPlan_EmitsArrivalEventOnFirstDwellTick(t *testing.T) {
	const patrolId = "test_arrival_emit"
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:        patrolId,
		LoopShape: "strict",
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 100, DwellRounds: 5, ArrivalEvent: "test_marker"},
			{Room: 200, DwellRounds: 0},
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest(patrolId) })

	mob := &mobs.Mob{InstanceId: 9001}
	mob.Character.RoomId = 100 // at waypoint 0
	mob.Character.SetMiscData("patrol_waypoint_idx", 0)
	mob.Character.SetMiscData("patrol_dwell_remaining", 5) // equals authored DwellRounds

	// Clear any pre-existing events for this instance.
	events.DrainQueuedPatrolWaypointArrivalsForTest(9001)

	// First tick: dwell_remaining == DwellRounds → should emit.
	plan := patrolTickPlan(mob, patrolId)
	if !plan.WantsDwellWait {
		t.Fatalf("expected WantsDwellWait=true on first tick, got %+v", plan)
	}
	applyPatrolPlan(mob, plan, patrolId)

	arrivals := events.DrainQueuedPatrolWaypointArrivalsForTest(9001)
	if len(arrivals) != 1 {
		t.Fatalf("expected exactly 1 PatrolWaypointArrival on first dwell tick, got %d", len(arrivals))
	}
	got := arrivals[0]
	if got.ArrivalEvent != "test_marker" {
		t.Errorf("expected ArrivalEvent=%q, got %q", "test_marker", got.ArrivalEvent)
	}
	if got.WaypointIdx != 0 {
		t.Errorf("expected WaypointIdx=0, got %d", got.WaypointIdx)
	}
	if got.RoomId != 100 {
		t.Errorf("expected RoomId=100, got %d", got.RoomId)
	}
	if got.PatrolId != patrolId {
		t.Errorf("expected PatrolId=%q, got %q", patrolId, got.PatrolId)
	}

	// Second tick at same waypoint: dwell_remaining was decremented to 4,
	// which != authored 5 → must NOT re-emit.
	plan = patrolTickPlan(mob, patrolId)
	if !plan.WantsDwellWait {
		t.Fatalf("expected WantsDwellWait=true on second tick, got %+v", plan)
	}
	applyPatrolPlan(mob, plan, patrolId)

	arrivals = events.DrainQueuedPatrolWaypointArrivalsForTest(9001)
	if len(arrivals) != 0 {
		t.Errorf("expected no PatrolWaypointArrival on second dwell tick, got %d", len(arrivals))
	}
}

func TestApplyPatrolPlan_OneshotTerminal_EmitsCompletedAndClears(t *testing.T) {
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:        "test_oneshot_complete",
		LoopShape: "oneshot",
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 100, DwellRounds: 1},
			{Room: 200, DwellRounds: 1},
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest("test_oneshot_complete") })

	events.DrainQueuedPatrolCompletedForTest(0)

	mob := &mobs.Mob{InstanceId: 9100, PatrolId: "test_oneshot_complete"}
	mob.Character.RoomId = 200
	mob.Character.SetMiscData("patrol_waypoint_idx", 1)
	mob.Character.SetMiscData("patrol_dwell_remaining", 0)

	plan := patrolTickPlan(mob, "test_oneshot_complete")
	if !plan.WantsComplete {
		t.Fatalf("expected WantsComplete=true at oneshot terminal, got %+v", plan)
	}
	applyPatrolPlan(mob, plan, "test_oneshot_complete")

	if mob.PatrolId != "" {
		t.Errorf("expected PatrolId cleared, got %q", mob.PatrolId)
	}

	completed := events.DrainQueuedPatrolCompletedForTest(mob.InstanceId)
	if len(completed) != 1 {
		t.Fatalf("expected 1 PatrolCompleted event for instance %d, got %d",
			mob.InstanceId, len(completed))
	}
	if completed[0].PatrolId != "test_oneshot_complete" {
		t.Errorf("PatrolId in event = %q, want %q",
			completed[0].PatrolId, "test_oneshot_complete")
	}
	if completed[0].RoomId != 200 {
		t.Errorf("RoomId in event = %d, want 200", completed[0].RoomId)
	}
}

func TestApplyPatrolPlan_StrictPatrol_DoesNotComplete(t *testing.T) {
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:        "test_strict_no_complete",
		LoopShape: "strict",
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 100, DwellRounds: 1},
			{Room: 200, DwellRounds: 1},
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest("test_strict_no_complete") })

	mob := &mobs.Mob{InstanceId: 9101, PatrolId: "test_strict_no_complete"}
	mob.Character.RoomId = 200
	mob.Character.SetMiscData("patrol_waypoint_idx", 1)
	mob.Character.SetMiscData("patrol_dwell_remaining", 0)

	plan := patrolTickPlan(mob, "test_strict_no_complete")
	if plan.WantsComplete {
		t.Errorf("expected WantsComplete=false for strict patrol at last waypoint, got true")
	}
}

// TestApplyPatrolPlan_EmitsArrivalEventOnZeroDwellWaypoint verifies that
// PatrolWaypointArrival fires when the mob is at a zero-dwell waypoint and
// the executor takes the WantsAdvance path. Chunk 3.7.
func TestApplyPatrolPlan_EmitsArrivalEventOnZeroDwellWaypoint(t *testing.T) {
	const patrolId = "test_zero_dwell_emit"
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:        patrolId,
		LoopShape: "strict",
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 100, DwellRounds: 0, ArrivalEvent: "instant_pass"},
			{Room: 200, DwellRounds: 0},
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest(patrolId) })

	mob := &mobs.Mob{InstanceId: 9002}
	mob.Character.RoomId = 100 // at waypoint 0, zero-dwell
	mob.Character.SetMiscData("patrol_waypoint_idx", 0)
	mob.Character.SetMiscData("patrol_dwell_remaining", 0)

	// Clear any pre-existing events for this instance.
	events.DrainQueuedPatrolWaypointArrivalsForTest(9002)

	plan := patrolTickPlan(mob, patrolId)
	if !plan.WantsAdvance {
		t.Fatalf("expected WantsAdvance=true for zero-dwell waypoint, got %+v", plan)
	}
	applyPatrolPlan(mob, plan, patrolId)

	arrivals := events.DrainQueuedPatrolWaypointArrivalsForTest(9002)
	if len(arrivals) != 1 {
		t.Fatalf("expected exactly 1 PatrolWaypointArrival on zero-dwell arrival, got %d", len(arrivals))
	}
	got := arrivals[0]
	if got.ArrivalEvent != "instant_pass" {
		t.Errorf("expected ArrivalEvent=%q, got %q", "instant_pass", got.ArrivalEvent)
	}
	if got.WaypointIdx != 0 {
		t.Errorf("expected WaypointIdx=0, got %d", got.WaypointIdx)
	}
	if got.RoomId != 100 {
		t.Errorf("expected RoomId=100, got %d", got.RoomId)
	}
}
