package catalog

import (
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	goals.RegisterGoalType("wealth-gold", goals.GoalTypeMeta{
		Predicate:    wealthGoldPredicate,
		ContextScore: wealthGoldContextScore,
		Params: []goals.ParamSchema{
			{Key: "target", Required: true, GoType: "int"},
		},
		// AllowMultiple: false — one gold target per mob.
	})
}

func wealthGoldPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	target := paramIntOr(g, "target", 0)
	return mob.Character.Gold >= target
}

// wealthGoldContextScore: 0 if satisfied; otherwise baseline 1.0 plus
// (target - gold) / target, capped at 2.0.
func wealthGoldContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	target := paramIntOr(g, "target", 0)
	if target <= 0 || mob.Character.Gold >= target {
		return 0
	}
	gap := float64(target-mob.Character.Gold) / float64(target)
	score := 1.0 + gap
	if score > 2.0 {
		return 2.0
	}
	return score
}
