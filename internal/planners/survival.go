package planners

import (
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const (
	survivalDefaultSafePct = 60
	survivalDefaultFleePct = 25
)

func init() {
	RegisterPlanner("survival", survivalPlanner)
}

// survivalPlanner: flee combat, drink heal potion, or rest until HP is
// at safe threshold. Reactive — no MiscData state.
//
// Branches:
//   - Recovered (HP >= safePct AND not in combat) → StatusSuccess
//   - In combat → flee via a random exit (StatusRunning)
//   - Low HP, not in combat → drink healing potion if available (StatusRunning)
//   - Default → rest (StatusRunning)
func survivalPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil || mob.Character.HealthMax.Value <= 0 {
		return PlanResult{Status: StatusFailure}
	}

	safePct := goalParamIntOr(goal, "safe_threshold_pct", survivalDefaultSafePct)
	fleePct := goalParamIntOr(goal, "flee_threshold_pct", survivalDefaultFleePct)
	hpPct := (mob.Character.Health * 100) / mob.Character.HealthMax.Value

	// Recovered: HP at safe threshold and not in combat → goal satisfied.
	if hpPct >= safePct && mob.Character.Aggro == nil {
		return PlanResult{Status: StatusSuccess}
	}

	// In combat → flee via a random exit.
	if mob.Character.Aggro != nil {
		exit := pickRandomExit(mob)
		if exit == "" {
			return PlanResult{Status: StatusFailure}
		}
		return PlanResult{Command: "flee " + exit, Status: StatusRunning}
	}

	// Low HP, not in combat → drink a healing potion if one is available.
	if hpPct < fleePct {
		if potion := pickHealingPotion(mob); potion != "" {
			return PlanResult{Command: "drink " + potion, Status: StatusRunning}
		}
	}

	// Default: rest to recover.
	return PlanResult{Command: "rest", Status: StatusRunning}
}

// pickHealingPotion returns the name of a healing potion in the mob's
// inventory, or empty string if none is found.
//
// TODO-ADAPT: heuristic candidates for "is this item a healing potion":
//   - item.GetSpec().Type == "potion" AND effect targets HP regen
//   - explicit potion-effect inspection via items.Item.GetSpec() effect fields
//
// Verify via codegraph (`codegraph_node ItemSpec` for the effect field
// shape). Conservative stub: returns empty string so the planner falls
// through to "rest", which is the correct safe default.
func pickHealingPotion(mob *mobs.Mob) string {
	if mob == nil || len(mob.Character.Items) == 0 {
		return ""
	}
	// TODO-ADAPT: iterate mob.Character.Items; check spec type + effects for
	// heal-potion heuristic; return spec.NameSimple or spec.Name of first match.
	return ""
}
