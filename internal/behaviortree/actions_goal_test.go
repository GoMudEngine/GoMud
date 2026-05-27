package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/planners"
)

// buildGoalMob creates a minimal mob instance for goal-action tests.
// Uses the same pattern as buildMutationMob in actions_mutation_test.go.
func buildGoalMob(t *testing.T, instanceId int, mobId mobs.MobId, name string, roomId int) *mobs.Mob {
	t.Helper()
	mob := &mobs.Mob{
		MobId:      mobId,
		InstanceId: instanceId,
		HomeRoomId: roomId,
	}
	mob.Character.RoomId = roomId
	mob.Character.Name = name
	mob.Character.Health = 100
	mob.Character.HealthMax.Value = 100
	mob.Character.Stamina = 100
	mob.Character.StaminaMax.Value = 100
	mob.Character.Buffs = buffs.New()
	mob.Character.Stats.Strength.ValueAdj = 100
	mob.Character.Stats.Dexterity.ValueAdj = 100
	mob.Character.Stats.Vitality.ValueAdj = 100
	mob.Character.Stats.Perception.ValueAdj = 100
	mob.Character.Stats.Willpower.ValueAdj = 100
	mob.Character.Stats.Charisma.ValueAdj = 100
	mobs.SetInstanceForTest(instanceId, mob)
	t.Cleanup(func() { mobs.SetInstanceForTest(instanceId, nil) })
	return mob
}

// TestTryGoalPlanner_RegisteredInActionRegistry verifies that the action
// was registered in the actionRegistry by the init() function.
// Mirrors TestTryMutationActive_RegisteredInActionRegistry.
func TestTryGoalPlanner_RegisteredInActionRegistry(t *testing.T) {
	if _, ok := actionRegistry["try_goal_planner"]; !ok {
		t.Fatal("try_goal_planner not registered in actionRegistry")
	}
}

// TestTranslatePlannerStatus verifies all three planners.BTreeStatus →
// behaviortree.Result mappings. Pure function, no fixtures needed.
func TestTranslatePlannerStatus(t *testing.T) {
	tests := []struct {
		in   planners.BTreeStatus
		want Result
	}{
		{planners.StatusSuccess, Success},
		{planners.StatusRunning, Running},
		{planners.StatusFailure, Failure},
		// Out-of-range values should also map to Failure.
		{planners.BTreeStatus(99), Failure},
	}
	for _, tc := range tests {
		got := translatePlannerStatus(tc.in)
		if got != tc.want {
			t.Errorf("translatePlannerStatus(%v) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestInvokePlannerSafely_PanicReturnsFailure verifies that a planner
// that panics does NOT propagate the panic and returns Failure + empty
// Command. Pure function — no server state needed.
func TestInvokePlannerSafely_PanicReturnsFailure(t *testing.T) {
	panicFn := planners.PlanFn(func(*mobs.Mob, *goals.Goal) planners.PlanResult {
		panic("test planner boom")
	})
	mob := &mobs.Mob{}
	goal := &goals.Goal{Type: "test-panicky", Id: "g1"}

	result := invokePlannerSafely(panicFn, mob, goal)
	if result.Status != planners.StatusFailure {
		t.Errorf("panic: Status=%v, want StatusFailure", result.Status)
	}
	if result.Command != "" {
		t.Errorf("panic: Command=%q, want empty", result.Command)
	}
}

// TestTryGoalPlanner_NilMob_Failure verifies that a missing mob instance
// returns Failure without panicking.
func TestTryGoalPlanner_NilMob_Failure(t *testing.T) {
	// No mob registered for this instance ID — mobs.GetInstance returns nil.
	ctx := &EvalContext{
		InstanceId: 9900,
		RoomId:     1,
		MobState:   NewBehaviorState(),
	}
	res := actGoalPlanner(nil, ctx)
	if res != Failure {
		t.Errorf("nil mob: got %v, want Failure", res)
	}
}

// TestTryGoalPlanner_NoCurrentGoal_Failure verifies that a mob with no
// goals returns Failure.
func TestTryGoalPlanner_NoCurrentGoal_Failure(t *testing.T) {
	goals.ClearCache()
	buildGoalMob(t, 9901, 99701, "no_goal_test", 1)

	ctx := &EvalContext{
		InstanceId: 9901,
		RoomId:     1,
		MobState:   NewBehaviorState(),
	}
	res := actGoalPlanner(nil, ctx)
	if res != Failure {
		t.Errorf("no current goal: got %v, want Failure", res)
	}
}

// TestTryGoalPlanner_NoRegisteredPlanner_Failure verifies that a mob with
// a current goal but no registered planner for that type returns Failure.
func TestTryGoalPlanner_NoRegisteredPlanner_Failure(t *testing.T) {
	goals.ClearCache()
	mob := buildGoalMob(t, 9902, 99702, "no_planner_test", 1)
	t.Cleanup(func() { goals.ClearCache() })

	if _, err := goals.Add(int(mob.MobId), "no_planner_test",
		&goals.Goal{Type: "unregistered-goal-type-9902", Priority: 50}); err != nil {
		t.Fatalf("goals.Add: %v", err)
	}

	ctx := &EvalContext{
		InstanceId: 9902,
		RoomId:     1,
		MobState:   NewBehaviorState(),
	}
	res := actGoalPlanner(nil, ctx)
	if res != Failure {
		t.Errorf("no registered planner: got %v, want Failure", res)
	}
}

// TestTryGoalPlanner_RegisteredPlanner_PropagatesSuccess verifies the
// dispatch path: a registered planner returning StatusSuccess maps to
// btree Success.
func TestTryGoalPlanner_RegisteredPlanner_PropagatesSuccess(t *testing.T) {
	goals.ClearCache()
	const testGoalType = "test-goal-planner-success-9903"
	planners.RegisterPlanner(testGoalType, func(*mobs.Mob, *goals.Goal) planners.PlanResult {
		return planners.PlanResult{Status: planners.StatusSuccess}
	})
	t.Cleanup(func() { planners.RegisterPlanner(testGoalType, nil) })

	mob := buildGoalMob(t, 9903, 99703, "planner_success_test", 1)
	t.Cleanup(func() { goals.ClearCache() })

	if _, err := goals.Add(int(mob.MobId), "planner_success_test",
		&goals.Goal{Type: testGoalType, Priority: 50}); err != nil {
		t.Fatalf("goals.Add: %v", err)
	}

	ctx := &EvalContext{
		InstanceId: 9903,
		RoomId:     1,
		MobState:   NewBehaviorState(),
	}
	res := actGoalPlanner(nil, ctx)
	if res != Success {
		t.Errorf("planner Success: got %v, want Success", res)
	}
}

// TestTryGoalPlanner_PlannerPanic_RecoveredFailure verifies that a panic
// inside the planner is recovered and maps to Failure, not a crash.
func TestTryGoalPlanner_PlannerPanic_RecoveredFailure(t *testing.T) {
	goals.ClearCache()
	const testGoalType = "test-goal-planner-panic-9904"
	planners.RegisterPlanner(testGoalType, func(*mobs.Mob, *goals.Goal) planners.PlanResult {
		panic("planner boom in dispatch test")
	})
	t.Cleanup(func() { planners.RegisterPlanner(testGoalType, nil) })

	mob := buildGoalMob(t, 9904, 99704, "planner_panic_test", 1)
	t.Cleanup(func() { goals.ClearCache() })

	if _, err := goals.Add(int(mob.MobId), "planner_panic_test",
		&goals.Goal{Type: testGoalType, Priority: 50}); err != nil {
		t.Fatalf("goals.Add: %v", err)
	}

	ctx := &EvalContext{
		InstanceId: 9904,
		RoomId:     1,
		MobState:   NewBehaviorState(),
	}
	// Must not panic.
	res := actGoalPlanner(nil, ctx)
	if res != Failure {
		t.Errorf("panic planner: got %v, want Failure", res)
	}
}
