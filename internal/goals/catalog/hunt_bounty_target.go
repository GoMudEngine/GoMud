package catalog

import (
	"github.com/GoMudEngine/GoMud/internal/bounties"
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func init() {
	goals.RegisterGoalType("hunt_bounty_target", goals.GoalTypeMeta{
		Predicate:    huntBountyTargetPredicate,
		ContextScore: huntBountyTargetContextScore,
		Params: []goals.ParamSchema{
			{Key: "target_user_id", Required: true, GoType: "int"},
			{Key: "bounty_id", Required: true, GoType: "int"},
		},
	})
}

// huntBountyTargetPredicate: satisfied (goal should be dropped) when the
// bounty no longer exists or is no longer open. While the bounty is open,
// the goal stays alive and eligible for selection.
func huntBountyTargetPredicate(g *goals.Goal, _ *mobs.Mob) bool {
	bountyId := paramIntOr(g, "bounty_id", 0)
	if bountyId == 0 {
		return true // malformed goal — treat as satisfied so it gets cleaned up
	}
	b := bounties.Get(bountyId)
	return b == nil || b.Status != bounties.StatusOpen
}

// huntBountyTargetContextScore: 100 when the target player is online,
// 0 otherwise. A high constant prioritises active pursuit while the
// target is reachable; falling to 0 when the target is offline
// effectively suspends the goal without removing it.
func huntBountyTargetContextScore(g *goals.Goal, _ *mobs.Mob) float64 {
	userId := paramIntOr(g, "target_user_id", 0)
	if userId == 0 {
		return 0
	}
	if users.GetByUserId(userId) != nil {
		return 100
	}
	return 0
}
