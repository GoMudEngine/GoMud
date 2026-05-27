package behaviortree

// actions_goal.go — chunk-4.4 try_goal_planner btree action.
//
// try_goal_planner dispatches to the registered per-goal-type planner
// (internal/planners/) for the mob's current strategic goal (selected
// every tick by chunk 4.2's Recompute). It:
//   1. Resolves the live mob instance via ctx.InstanceId.
//   2. Looks up CurrentGoalOf from the goals package.
//   3. Looks up the registered PlanFn for the goal's Type.
//   4. Invokes the planner under panic recovery (mirrors invokeContextScore).
//   5. Executes the returned Command (if non-empty) via mob.Command().
//   6. Maps planners.BTreeStatus → behaviortree.Result.
//
// Returns Failure if:
//   - The mob instance is not found
//   - No current goal exists for this mob
//   - No planner is registered for the goal type
//   - The planner panics (logged at Warn, mapped to Failure)
//   - The planner returns StatusFailure
//
// Authors insert try_goal_planner explicitly in each archetype's behavior
// tree where goal-driven tactical behavior should fire. See Task 21 of
// the 4.4 implementation plan.

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/planners"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func init() {
	actionRegistry["try_goal_planner"] = actGoalPlanner
}

// actGoalPlanner is the chunk-4.4 btree action that dispatches per the
// mob's current goal to the registered per-type planner.
func actGoalPlanner(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	templateId := int(mob.MobId)
	name := util.ConvertForFilename(mob.Character.Name)
	currentGoal := goals.CurrentGoalOf(templateId, name)
	if currentGoal == nil {
		return Failure
	}
	fn := planners.LookupPlanner(currentGoal.Type)
	if fn == nil {
		return Failure
	}

	result := invokePlannerSafely(fn, mob, currentGoal)

	if result.Command != "" {
		mob.Command(result.Command)
	}
	return translatePlannerStatus(result.Status)
}

// invokePlannerSafely wraps the planner call in panic recovery. A panic
// logs a warn line with goal type + mob id and returns Failure.
// Mirrors invokeContextScore (4.2) and invokeDedupKey (4.3).
func invokePlannerSafely(fn planners.PlanFn, mob *mobs.Mob, goal *goals.Goal) (result planners.PlanResult) {
	defer func() {
		if r := recover(); r != nil {
			mudlog.Warn("planners.plan panic",
				"goal_type", goal.Type,
				"goal_id", goal.Id,
				"mob_id", mob.MobId,
				"panic", fmt.Sprintf("%v", r))
			result = planners.PlanResult{Status: planners.StatusFailure}
		}
	}()
	return fn(mob, goal)
}

// translatePlannerStatus maps planners.BTreeStatus → behaviortree.Result.
// Two separate enums by design (planners can't import behaviortree
// without forming a cycle). This one-line switch is the only place the
// translation happens.
func translatePlannerStatus(ps planners.BTreeStatus) Result {
	switch ps {
	case planners.StatusSuccess:
		return Success
	case planners.StatusRunning:
		return Running
	}
	return Failure
}
