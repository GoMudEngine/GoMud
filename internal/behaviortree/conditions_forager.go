package behaviortree

// conditions_forager.go — btree conditions specific to forager NPCs.
//
// All three conditions read forager state via the same MobState +
// ForagerProfile lookup pattern as actions_forager.go.

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	conditionRegistry["mob_can_safely_engage"] = condMobCanSafelyEngage
	conditionRegistry["mob_inventory_at_threshold"] =
		condMobInventoryAtThreshold
	conditionRegistry["mob_hp_below_recall_threshold"] =
		condMobHPBelowRecallThreshold
}

// condMobCanSafelyEngage returns Success when the mob's current aggro
// target is on the forager's prey whitelist AND the target's total stat
// pool is at most 60% of the forager's stat pool AND the forager is
// above 75% HP. Keeps foragers opportunistic without being suicidal.
func condMobCanSafelyEngage(
	params map[string]any,
	ctx *EvalContext,
) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	profile := forager.ProfileFor(int(mob.MobId))
	if profile == nil {
		return Failure
	}
	if !mob.Character.IsInCombat() {
		return Failure
	}
	target := mobs.GetInstance(mob.Character.CurrentCombatTarget().MobInstanceId)
	if target == nil {
		return Failure
	}
	// Prey whitelist check.
	prey := false
	for _, id := range profile.PreyWhitelist {
		if int(target.MobId) == id {
			prey = true
			break
		}
	}
	if !prey {
		return Failure
	}
	// 60% stat-pool gate: only engage if target is clearly weaker.
	if npcStatSum(target) > int(float64(npcStatSum(mob))*0.6) {
		return Failure
	}
	// HP gate: don't engage when already hurt.
	if hpRatio(mob) < 0.75 {
		return Failure
	}
	return Success
}

// condMobInventoryAtThreshold returns Success when the forager's
// carried-weight ratio meets or exceeds ForagerCarryThresholdPct.
func condMobInventoryAtThreshold(
	params map[string]any,
	ctx *EvalContext,
) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	threshold := float64(configs.GetBalanceConfig().ForagerCarryThresholdPct)
	if carryRatio(mob) >= threshold {
		return Success
	}
	return Failure
}

// condMobHPBelowRecallThreshold returns Success when the forager's HP
// ratio is at or below ForagerHPRecallThresholdPct.
func condMobHPBelowRecallThreshold(
	params map[string]any,
	ctx *EvalContext,
) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	threshold := float64(
		configs.GetBalanceConfig().ForagerHPRecallThresholdPct)
	if hpRatio(mob) <= threshold {
		return Success
	}
	return Failure
}

// npcStatSum returns the sum of all six adjusted stat values for the
// given mob. Used by condMobCanSafelyEngage for the relative-power gate.
func npcStatSum(m *mobs.Mob) int {
	s := &m.Character.Stats
	return s.Strength.ValueAdj +
		s.Dexterity.ValueAdj +
		s.Vitality.ValueAdj +
		s.Perception.ValueAdj +
		s.Willpower.ValueAdj +
		s.Charisma.ValueAdj
}
