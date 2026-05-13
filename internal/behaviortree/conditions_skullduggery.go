package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// condMobIsHidden returns Success when the calling mob carries the Hidden
// buff (buff 9). Used to gate stealth-dependent branches in the thief
// archetype — only attempt to steal or flee-while-stealing when already
// concealed.
func condMobIsHidden(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.HasBuffFlag(buffs.Hidden) {
		return Success
	}
	return Failure
}

// condTargetIsHidden returns Success when the resolved target carries the
// Hidden buff. Resolution order: ctx.Event.UserId → mob.Aggro.UserId →
// mob.Aggro.MobInstanceId.
//
// Note: this uses aggro-user-before-aggro-mob priority, matching the
// thief's pick-target context. This differs from resolveTargetPower in
// conditions_combat.go, which uses aggro-mob-before-aggro-user to match
// the combat agro priority.
func condTargetIsHidden(params map[string]any, ctx *EvalContext) Result {
	u, m := resolveSkullduggeryTarget(ctx)
	if u != nil && u.Character.HasBuffFlag(buffs.Hidden) {
		return Success
	}
	if m != nil && m.Character.HasBuffFlag(buffs.Hidden) {
		return Success
	}
	return Failure
}

// condTargetHasGold returns Success when the resolved player target holds at
// least `min` gold (default 1). Mob targets always return Failure — mob gold
// is merchant reserve currency and is not directly stealable by the thief
// archetype's steal action.
func condTargetHasGold(params map[string]any, ctx *EvalContext) Result {
	min := getIntParam(params, "min")
	if min == 0 {
		min = 1
	}
	u, _ := resolveSkullduggeryTarget(ctx)
	if u == nil {
		return Failure
	}
	if u.Character.Gold >= min {
		return Success
	}
	return Failure
}

// resolveSkullduggeryTarget returns the contextual target as a user or mob
// using the fallback chain:
//
//  1. ctx.Event.UserId → player
//  2. mob.Character.Aggro.UserId → player
//  3. mob.Character.Aggro.MobInstanceId → mob
//
// Returns (nil, nil) when no target is resolvable.
//
// The aggro-user-before-aggro-mob priority matches the thief's context:
// the primary targets for steal/shadow are players. Compare with
// resolveTargetPower in conditions_combat.go which uses the reverse order
// to match melee-combat aggro semantics.
func resolveSkullduggeryTarget(ctx *EvalContext) (*users.UserRecord, *mobs.Mob) {
	if ctx.Event.UserId > 0 {
		if u := users.GetByUserId(ctx.Event.UserId); u != nil {
			return u, nil
		}
	}
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || mob.Character.Aggro == nil {
		return nil, nil
	}
	if mob.Character.Aggro.UserId > 0 {
		if u := users.GetByUserId(mob.Character.Aggro.UserId); u != nil {
			return u, nil
		}
	}
	if mob.Character.Aggro.MobInstanceId > 0 {
		if m := mobs.GetInstance(mob.Character.Aggro.MobInstanceId); m != nil {
			return nil, m
		}
	}
	return nil, nil
}
