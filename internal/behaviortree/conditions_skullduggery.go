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
// Hidden buff. Resolution order: SoftTarget → Event.UserId → CombatPhase.
//
// SoftTarget priority ensures the thief archetype's steal-and-flee sequence
// checks the same player picked by target_random_player_in_room.
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
//  1. ctx.SoftTarget → non-combat target set by target_random_player_in_room
//  2. ctx.Event.UserId → triggering player
//  3. Combat Phase target (mob.Character.CurrentCombatTarget)
//
// Returns (nil, nil) when no target is resolvable.
//
// SoftTarget has priority because skullduggery is a non-combat path: the
// upstream target-picker (target_random_player_in_room) sets SoftTarget
// specifically for the downstream condition + action chain to consume. This
// ensures conditions like target_has_gold evaluate against the same player
// that try_steal will act on, even when no combat has started.
func resolveSkullduggeryTarget(ctx *EvalContext) (*users.UserRecord, *mobs.Mob) {
	// Priority 1: SoftTarget (non-combat target set by target-picker actions).
	if !ctx.SoftTarget.IsZero() {
		if ctx.SoftTarget.IsPlayer() {
			if u := users.GetByUserId(ctx.SoftTarget.UserId); u != nil {
				return u, nil
			}
		}
		if ctx.SoftTarget.IsMob() {
			if m := mobs.GetInstance(ctx.SoftTarget.MobInstanceId); m != nil {
				return nil, m
			}
		}
		// SoftTarget set but couldn't resolve — fall through.
	}
	// Priority 2: Triggering player from the event.
	if ctx.Event.UserId > 0 {
		if u := users.GetByUserId(ctx.Event.UserId); u != nil {
			return u, nil
		}
	}
	// Priority 3: Combat Phase target (only when mob is actually in combat).
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return nil, nil
	}
	t := mob.Character.CurrentCombatTarget()
	if t.IsPlayer() {
		if u := users.GetByUserId(t.UserId); u != nil {
			return u, nil
		}
	}
	if t.IsMob() {
		if m := mobs.GetInstance(t.MobInstanceId); m != nil {
			return nil, m
		}
	}
	return nil, nil
}
