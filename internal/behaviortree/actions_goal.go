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

// goalPlannerRanRoundKey is the TempData key stamped by RunGoalPlanner every
// round the planner is dispatched (regardless of whether it issued a command).
// The MobIdle handler reads it (via mobGoalPlannerRanRound) to dedup: archetype
// mobs whose btree already ran try_goal_planner this round must not have the
// idle handler re-dispatch the planner. Distinct from "goalActedRound" (Fix B),
// which is stamped ONLY when the planner issues a command.
const goalPlannerRanRoundKey = "goalPlannerRanRound"

func init() {
	actionRegistry["try_goal_planner"] = actGoalPlanner
}

// RunGoalPlanner dispatches the mob's current-goal planner: looks up the
// current goal + its registered planner, runs it under panic recovery,
// executes any returned command, and stamps round markers. Returns the
// translated btree Result. Used by the try_goal_planner btree action AND by
// the MobIdle handler (so mobs whose per-mob tree lacks the try_goal_planner
// node still pursue goals). nowRound is util.GetRoundCount() from the caller.
func RunGoalPlanner(mob *mobs.Mob, nowRound uint64) Result {
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
	// Mark that the planner ran this round (dedup so the idle handler doesn't
	// re-dispatch for mobs whose btree already ran it).
	mob.SetTempData(goalPlannerRanRoundKey, nowRound)
	result := invokePlannerSafely(fn, mob, currentGoal)
	if result.Command != "" {
		mob.Command(result.Command)
		// "Planner owns the tick" stamp: record the round on which the
		// planner actually ISSUED a command. The idle handler reads this
		// (goalActedRound == GetRoundCount()) to suppress its legacy idle
		// block (WanderCount home-pull + idle emote) so it doesn't fight the
		// command the planner just queued (e.g. a pathto-to-shop). It also
		// reads it one round stale (== GetRoundCount()-1) in the displacement
		// guard so a mid-journey MaxWander==0 mob isn't yanked home. We do NOT
		// stamp on an empty Command (idle-Running) — in that case legacy idle
		// SHOULD still fire so the mob emits its flavor emotes.
		mob.SetTempData("goalActedRound", nowRound)
	}
	return translatePlannerStatus(result.Status)
}

// actGoalPlanner is the chunk-4.4 btree action that dispatches per the
// mob's current goal to the registered per-type planner. Delegates to
// RunGoalPlanner so the same logic backs the MobIdle-handler fallback.
func actGoalPlanner(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	return RunGoalPlanner(mob, util.GetRoundCount())
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
