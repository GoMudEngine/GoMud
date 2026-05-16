package behaviortree

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func condMobInCombat(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.IsInCombat() {
		return Success
	}
	return Failure
}

func condMobHealthBelow(params map[string]any, ctx *EvalContext) Result {
	pct := getIntParam(params, "percent")
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	maxHP := mob.Character.HealthMax.Value
	if maxHP <= 0 {
		return Failure
	}
	currentPct := mob.Character.Health * 100 / maxHP
	if currentPct < pct {
		return Success
	}
	return Failure
}

func condMobAtHome(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if mob.Character.RoomId == mob.HomeRoomId {
		return Success
	}
	return Failure
}

func condMobHasBuff(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	buffId := getIntParam(params, "buff_id")
	if mob.Character.HasBuff(buffId) {
		return Success
	}
	return Failure
}

// condMobInRoom checks if a mob with the given template mob_id is present in
// the room.
func condMobInRoom(params map[string]any, ctx *EvalContext) Result {
	mobId := getIntParam(params, "mob_id")
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		m := mobs.GetInstance(instId)
		if m != nil && int(m.MobId) == mobId {
			return Success
		}
	}
	return Failure
}

// condTargetIsCasting returns Success if the mob's current aggro target
// is mid-cast. Used by archetypes that want to prioritize interrupts.
func condTargetIsCasting(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || !mob.Character.IsInCombat() {
		return Failure
	}
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return Failure
	}
	if target.Char.IsCasting() {
		return Success
	}
	return Failure
}

// condTargetAggroNotOnMe returns Success if the mob's current aggro
// target is NOT attacking the mob — either has no aggro, or aggros
// someone else. Used by tank archetypes to gate taunt so they only
// taunt when they're not already the focus.
func condTargetAggroNotOnMe(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || !mob.Character.IsInCombat() {
		return Failure
	}
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return Failure
	}
	if !target.Char.IsInCombat() {
		return Success
	}
	// Target's aggro is on me (mob) iff Aggro.MobInstanceId == mob.InstanceId
	// AND Aggro.UserId == 0 (mob target, not player target).
	if target.Char.CurrentCombatTarget().MobInstanceId == mob.InstanceId && target.Char.CurrentCombatTarget().UserId == 0 {
		return Failure
	}
	return Success
}

// condPackmateBelowHpRatio returns Success if any same-room packmate
// (per mobs.FindPackmatesInRoom) has HP ratio strictly below the
// `threshold` param (default 0.40 — matches pure_caster's
// opportunistic-heal gate).
//
// Used by the pure_caster and support_caster archetypes'
// packmate_hurt handlers to gate heal-wounded-packmate branches.
func condPackmateBelowHpRatio(params map[string]any, ctx *EvalContext) Result {
	self := mobs.GetInstance(ctx.InstanceId)
	if self == nil {
		return Failure
	}
	threshold := getFloatParam(params, "threshold", 0.40)
	for _, pm := range mobs.FindPackmatesInRoom(self) {
		maxHp := pm.Character.HealthMax.Value
		if maxHp <= 0 {
			continue
		}
		ratio := float64(pm.Character.Health) / float64(maxHp)
		if ratio < threshold {
			return Success
		}
	}
	return Failure
}

// condPackmateIsTanking returns Success if any same-room packmate has
// an active Aggro (is engaged in combat). Used by the support_caster
// archetype to gate "shield the tank" behavior — if a packmate is
// actively tanking, prioritize casting defensive buffs on them.
func condPackmateIsTanking(params map[string]any, ctx *EvalContext) Result {
	self := mobs.GetInstance(ctx.InstanceId)
	if self == nil {
		return Failure
	}
	for _, pm := range mobs.FindPackmatesInRoom(self) {
		if pm.Character.IsInCombat() {
			return Success
		}
	}
	return Failure
}

// condTargetNotStanding returns Success if the mob's current aggro target
// is in any non-standing position (prone / clinched / grounded).
// Used to gate bonus-damage kicks (stomp when prone, knee when clinched).
func condTargetNotStanding(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || !mob.Character.IsInCombat() {
		return Failure
	}
	target := actions.ResolveAggroTarget(mob.Character.Aggro)
	if !target.Found {
		return Failure
	}
	if !target.Char.IsStanding() {
		return Success
	}
	return Failure
}
