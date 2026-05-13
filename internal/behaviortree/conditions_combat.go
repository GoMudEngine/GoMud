package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// condTargetPowerRatioAbove returns Success when
// self_power / target_power > value.
func condTargetPowerRatioAbove(params map[string]any, ctx *EvalContext) Result {
	return targetPowerRatioCompare(params, ctx, true)
}

// condTargetPowerRatioBelow returns Success when
// self_power / target_power < value.
func condTargetPowerRatioBelow(params map[string]any, ctx *EvalContext) Result {
	return targetPowerRatioCompare(params, ctx, false)
}

// targetPowerRatioCompare implements the shared comparison body
// for the two power-ratio conditions. above=true means "ratio
// strictly greater than value"; above=false means "ratio strictly
// less than value". Missing/zero value → Failure (caller config
// error).
//
// Degenerate target power (<= 0) is treated as "infinitely
// weaker": above-comparison Succeeds, below-comparison Fails.
func targetPowerRatioCompare(params map[string]any, ctx *EvalContext, above bool) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	threshold := getFloatParam(params, "value", 0)
	if threshold == 0 {
		return Failure
	}

	targetPower, ok := resolveTargetPower(mob, ctx)
	if !ok {
		return Failure
	}

	selfPower := combat.PowerScore(mob.Character)
	if targetPower <= 0 {
		if above {
			return Success
		}
		return Failure
	}
	ratio := selfPower / targetPower
	if above && ratio > threshold {
		return Success
	}
	if !above && ratio < threshold {
		return Success
	}
	return Failure
}

// resolveTargetPower returns the PowerScore of the contextual
// target, with fallback chain:
//  1. ctx.Event.UserId → player
//  2. mob.Character.Aggro.MobInstanceId → mob
//  3. mob.Character.Aggro.UserId → player
//
// Returns (0, false) when no target resolvable.
func resolveTargetPower(mob *mobs.Mob, ctx *EvalContext) (float64, bool) {
	if ctx.Event.UserId > 0 {
		if u := users.GetByUserId(ctx.Event.UserId); u != nil {
			return combat.PowerScore(*u.Character), true
		}
	}
	// Aggro fallback: matches actions.ResolveAggroTarget priority —
	// mob target before player target when both fields are set.
	if mob.Character.Aggro != nil {
		if mob.Character.Aggro.MobInstanceId > 0 {
			if m := mobs.GetInstance(mob.Character.Aggro.MobInstanceId); m != nil {
				return combat.PowerScore(m.Character), true
			}
		}
		if mob.Character.Aggro.UserId > 0 {
			if u := users.GetByUserId(mob.Character.Aggro.UserId); u != nil {
				return combat.PowerScore(*u.Character), true
			}
		}
	}
	return 0, false
}
