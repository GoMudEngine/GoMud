package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/planners"
	"github.com/GoMudEngine/GoMud/internal/util"
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

// TestTryGoalPlanner_StampsGoalActedRound_WhenCommandIssued verifies the
// "planner owns the tick" stamp: when the dispatched planner returns a
// non-empty Command, actGoalPlanner records the current round in the
// "goalActedRound" TempData key. The idle handler reads this to suppress its
// legacy idle block on ticks the planner is actively acting on.
func TestTryGoalPlanner_StampsGoalActedRound_WhenCommandIssued(t *testing.T) {
	goals.ClearCache()
	util.SetRoundCountForTest(4242)
	const testGoalType = "test-goal-planner-stamp-9905"
	planners.RegisterPlanner(testGoalType, func(*mobs.Mob, *goals.Goal) planners.PlanResult {
		return planners.PlanResult{Command: "pathto 100", Status: planners.StatusRunning}
	})
	t.Cleanup(func() { planners.RegisterPlanner(testGoalType, nil) })

	mob := buildGoalMob(t, 9905, 99705, "planner_stamp_test", 1)
	t.Cleanup(func() { goals.ClearCache() })

	if _, err := goals.Add(int(mob.MobId), "planner_stamp_test",
		&goals.Goal{Type: testGoalType, Priority: 50}); err != nil {
		t.Fatalf("goals.Add: %v", err)
	}

	ctx := &EvalContext{
		InstanceId: 9905,
		RoomId:     1,
		MobState:   NewBehaviorState(),
	}
	res := actGoalPlanner(nil, ctx)
	if res != Running {
		t.Errorf("running planner with command: got %v, want Running", res)
	}

	v := mob.GetTempData("goalActedRound")
	got, ok := v.(uint64)
	if !ok {
		t.Fatalf("goalActedRound TempData not set as uint64: %v (%T)", v, v)
	}
	if got != 4242 {
		t.Errorf("goalActedRound = %d, want 4242", got)
	}
}

// TestTryGoalPlanner_NoStamp_WhenCommandEmpty verifies that when the planner
// returns an empty Command (idle-Running, e.g. nothing to buy), actGoalPlanner
// does NOT stamp goalActedRound — leaving legacy idle free to emit emotes.
func TestTryGoalPlanner_NoStamp_WhenCommandEmpty(t *testing.T) {
	goals.ClearCache()
	util.SetRoundCountForTest(5555)
	const testGoalType = "test-goal-planner-nostamp-9906"
	planners.RegisterPlanner(testGoalType, func(*mobs.Mob, *goals.Goal) planners.PlanResult {
		return planners.PlanResult{Command: "", Status: planners.StatusRunning}
	})
	t.Cleanup(func() { planners.RegisterPlanner(testGoalType, nil) })

	mob := buildGoalMob(t, 9906, 99706, "planner_nostamp_test", 1)
	t.Cleanup(func() { goals.ClearCache() })

	if _, err := goals.Add(int(mob.MobId), "planner_nostamp_test",
		&goals.Goal{Type: testGoalType, Priority: 50}); err != nil {
		t.Fatalf("goals.Add: %v", err)
	}

	ctx := &EvalContext{
		InstanceId: 9906,
		RoomId:     1,
		MobState:   NewBehaviorState(),
	}
	res := actGoalPlanner(nil, ctx)
	if res != Running {
		t.Errorf("running planner with empty command: got %v, want Running", res)
	}

	if v := mob.GetTempData("goalActedRound"); v != nil {
		t.Errorf("goalActedRound should be unset for empty command, got %v", v)
	}
}

// TestRunGoalPlanner_NilMob_Failure verifies the exported dispatcher returns
// Failure for a nil mob without panicking.
func TestRunGoalPlanner_NilMob_Failure(t *testing.T) {
	if got := RunGoalPlanner(nil, 100); got != Failure {
		t.Errorf("nil mob: got %v, want Failure", got)
	}
}

// TestRunGoalPlanner_CommandIssued_StampsBothMarkers verifies that when a
// registered planner returns a non-empty Command, RunGoalPlanner issues it and
// stamps BOTH goalPlannerRanRound (dedup marker) and goalActedRound (planner-
// owns-the-tick marker) with the caller-supplied round.
func TestRunGoalPlanner_CommandIssued_StampsBothMarkers(t *testing.T) {
	goals.ClearCache()
	const testGoalType = "test-run-goal-planner-cmd-9910"
	planners.RegisterPlanner(testGoalType, func(*mobs.Mob, *goals.Goal) planners.PlanResult {
		return planners.PlanResult{Command: "pathto 100", Status: planners.StatusRunning}
	})
	t.Cleanup(func() { planners.RegisterPlanner(testGoalType, nil) })

	mob := buildGoalMob(t, 9910, 99710, "run_planner_cmd_test", 1)
	t.Cleanup(func() { goals.ClearCache() })

	if _, err := goals.Add(int(mob.MobId), "run_planner_cmd_test",
		&goals.Goal{Type: testGoalType, Priority: 50}); err != nil {
		t.Fatalf("goals.Add: %v", err)
	}

	res := RunGoalPlanner(mob, 7777)
	if res != Running {
		t.Errorf("running planner with command: got %v, want Running", res)
	}

	if v, ok := mob.GetTempData(goalPlannerRanRoundKey).(uint64); !ok || v != 7777 {
		t.Errorf("goalPlannerRanRound = %v (ok=%v), want 7777", v, ok)
	}
	if v, ok := mob.GetTempData("goalActedRound").(uint64); !ok || v != 7777 {
		t.Errorf("goalActedRound = %v (ok=%v), want 7777", v, ok)
	}
}

// TestRunGoalPlanner_EmptyCommand_StampsRanNotActed verifies that when the
// planner returns an empty Command (idle-Running), RunGoalPlanner stamps the
// dedup marker (goalPlannerRanRound) but NOT goalActedRound — so legacy idle
// stays free to emit flavor emotes.
func TestRunGoalPlanner_EmptyCommand_StampsRanNotActed(t *testing.T) {
	goals.ClearCache()
	const testGoalType = "test-run-goal-planner-empty-9911"
	planners.RegisterPlanner(testGoalType, func(*mobs.Mob, *goals.Goal) planners.PlanResult {
		return planners.PlanResult{Command: "", Status: planners.StatusRunning}
	})
	t.Cleanup(func() { planners.RegisterPlanner(testGoalType, nil) })

	mob := buildGoalMob(t, 9911, 99711, "run_planner_empty_test", 1)
	t.Cleanup(func() { goals.ClearCache() })

	if _, err := goals.Add(int(mob.MobId), "run_planner_empty_test",
		&goals.Goal{Type: testGoalType, Priority: 50}); err != nil {
		t.Fatalf("goals.Add: %v", err)
	}

	res := RunGoalPlanner(mob, 8888)
	if res != Running {
		t.Errorf("running planner with empty command: got %v, want Running", res)
	}

	if v, ok := mob.GetTempData(goalPlannerRanRoundKey).(uint64); !ok || v != 8888 {
		t.Errorf("goalPlannerRanRound = %v (ok=%v), want 8888", v, ok)
	}
	if v := mob.GetTempData("goalActedRound"); v != nil {
		t.Errorf("goalActedRound should be unset for empty command, got %v", v)
	}
}

// TestRunGoalPlanner_NoCurrentGoal_Failure verifies that a mob with no goal
// returns Failure and does NOT stamp the dedup marker.
func TestRunGoalPlanner_NoCurrentGoal_Failure(t *testing.T) {
	goals.ClearCache()
	mob := buildGoalMob(t, 9912, 99712, "run_planner_nogoal_test", 1)
	t.Cleanup(func() { goals.ClearCache() })

	if got := RunGoalPlanner(mob, 1234); got != Failure {
		t.Errorf("no current goal: got %v, want Failure", got)
	}
	if v := mob.GetTempData(goalPlannerRanRoundKey); v != nil {
		t.Errorf("goalPlannerRanRound should be unset when no goal, got %v", v)
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
