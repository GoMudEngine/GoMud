package catalog

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const (
	upgradeGearFloorScore  = 1.0 // never 0 → 4.6 dormancy never abandons this standing drive
	upgradeGearActiveScore = 2.5 // idle + a plausible path to a purchase
)

func init() {
	goals.RegisterGoalType("upgrade-gear", goals.GoalTypeMeta{
		Predicate:    upgradeGearPredicate,
		ContextScore: upgradeGearContextScore,
		// AllowMultiple: false — one upgrade-gear drive per mob.
		Params: []goals.ParamSchema{
			{Key: "reserve", Required: false, GoType: "int"},
		},
	})
}

// upgradeGearPredicate always returns false: a mob can always want better
// gear, so this drive has no terminal satisfied state. Activation is governed
// entirely by ContextScore + the planner.
func upgradeGearPredicate(_ *goals.Goal, _ *mobs.Mob) bool {
	return false
}

// upgradeGearReserve returns the gold floor a shopping mob keeps, from the
// goal's optional "reserve" param or the MobUpgradeGoldReserve config default.
func upgradeGearReserve(g *goals.Goal) int {
	def := int(configs.GetBalanceConfig().MobUpgradeGoldReserve)
	return paramIntOr(g, "reserve", def)
}

// upgradeGearContextScore:
//   - 0 for nil mob.
//   - floor (1.0) while in combat (Aggro != nil) — survival/combat goals win;
//     staying at the floor keeps the drive from 4.6 dormancy abandonment.
//   - active (2.5) when idle AND there is a plausible path to a purchase:
//     spendable gold above the reserve, OR sellable loot to fund saving.
//   - floor (1.0) otherwise (idle but broke and nothing to sell).
//
// Deliberately cheap and self-contained — no shop scan. The planner is the
// authority on whether anything is actually in stock.
func upgradeGearContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	if mob.Character.Aggro != nil {
		return upgradeGearFloorScore
	}
	spendable := mob.Character.Gold - upgradeGearReserve(g)
	if spendable > 0 || mobHasAnySellableLoot(mob) {
		return upgradeGearActiveScore
	}
	return upgradeGearFloorScore
}

// mobHasAnySellableLoot reports whether the mob carries at least one
// non-quest item with positive vendor value. Mirrors the planners-package
// mobHasSellableItems; duplicated to keep catalog free of a planners import
// (the same intentional duplication noted across the goals/catalog and
// planners packages).
func mobHasAnySellableLoot(mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	for i := range mob.Character.Items {
		spec := mob.Character.Items[i].GetSpec()
		if spec.QuestToken != "" {
			continue
		}
		if spec.Value > 0 {
			return true
		}
	}
	return false
}
