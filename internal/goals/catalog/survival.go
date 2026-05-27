package catalog

import (
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const (
	survivalDefaultSafeThresholdPct = 60
	survivalDefaultFleeThresholdPct = 25
)

func init() {
	goals.RegisterGoalType("survival", goals.GoalTypeMeta{
		Predicate:    survivalPredicate,
		ContextScore: survivalContextScore,
		Params: []goals.ParamSchema{
			{Key: "safe_threshold_pct", Required: false, GoType: "int"},
			{Key: "flee_threshold_pct", Required: false, GoType: "int"},
		},
		// AllowMultiple: false — only one survival goal per mob.
	})
}

// survivalPredicate: satisfied (mob has recovered) when HP is at or above
// safe_threshold_pct AND mob is not in combat.
func survivalPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil || mob.Character.HealthMax.Value <= 0 {
		return false
	}
	safe := paramIntOr(g, "safe_threshold_pct", survivalDefaultSafeThresholdPct)
	hpPct := (mob.Character.Health * 100) / mob.Character.HealthMax.Value
	return hpPct >= safe && mob.Character.Aggro == nil
}

// survivalContextScore:
//   - 0 if HP at/above safe threshold (filtered — predicate fires next tick)
//   - 5.0 if HP at/below flee threshold (critical)
//   - linear interpolation from 1.0 (at safe) to 3.0 (at flee) in between
func survivalContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil || mob.Character.HealthMax.Value <= 0 {
		return 0
	}
	safe := paramIntOr(g, "safe_threshold_pct", survivalDefaultSafeThresholdPct)
	flee := paramIntOr(g, "flee_threshold_pct", survivalDefaultFleeThresholdPct)
	hpPct := (mob.Character.Health * 100) / mob.Character.HealthMax.Value
	if hpPct >= safe {
		return 0
	}
	if hpPct <= flee {
		return 5.0
	}
	// Linear: at safe → 1.0, at flee → 3.0
	fraction := float64(safe-hpPct) / float64(safe-flee)
	return 1.0 + fraction*(3.0-1.0)
}

// paramIntOr reads an int param from g.Params with a fallback default.
// Tolerates the int64-from-YAML case (matches ValidateParams widening).
func paramIntOr(g *goals.Goal, key string, def int) int {
	if g == nil || g.Params == nil {
		return def
	}
	raw, ok := g.Params[key]
	if !ok {
		return def
	}
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	}
	return def
}
