package catalog

import (
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	goals.RegisterGoalType("hunt_bounty_target", goals.GoalTypeMeta{
		Predicate:    huntBountyTargetPredicate,
		ContextScore: huntBountyTargetContextScore,
		// No Params: the goal is a param-less template-level intent marker.
		// Per-hunter target lives in the hunter instance's MiscData
		// (bh_target_user_id, bh_bounty_id), stamped by spawnHunter.
	})
}

// huntBountyTargetPredicate always returns false — this goal is a
// template-level strategic-intent marker that is never self-satisfied.
// Per-hunter lifecycle (target acquired, bounty closed) is owned by the
// dispatch manager, which despawns each hunter instance when its bounty
// closes. Keeping the goal alive on the template means the shared goal
// is always selected whenever a hunter instance is running.
func huntBountyTargetPredicate(_ *goals.Goal, _ *mobs.Mob) bool {
	return false // always active; never self-pruned
}

// huntBountyTargetContextScore returns a high constant so the goal is
// always preferred when selected. The per-hunter target is read from
// instance MiscData (bh_target_user_id) by the planner, not from goal
// params, so no param lookup is needed here.
func huntBountyTargetContextScore(_ *goals.Goal, _ *mobs.Mob) float64 {
	return 100
}
