package behaviortree

// actions_combat.go — combat actions:
// actAttack, actFlee, actCast, actAddBuff, actRemoveBuff

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func actAttack(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	targetUserId := ctx.Event.UserId
	// If no specific target, pick a random player in the room
	if targetUserId == 0 {
		room := rooms.LoadRoom(ctx.RoomId)
		if room == nil {
			return Failure
		}
		players := room.GetPlayers()
		if len(players) == 0 {
			return Failure
		}
		targetUserId = players[util.Rand(len(players))]
	}
	// Promote to SurpriseAttack when hidden so the combat pipeline
	// applies the backstab crit + "*[SURPRISE ATTACK]*" prefix.
	// Mirrors the pattern in mobcommands/attack.go.
	aggroType := characters.DefaultAttack
	if mob.Character.IsHidden() {
		aggroType = characters.SurpriseAttack
	}
	mob.Character.SetAggro(targetUserId, 0, aggroType)
	return Success
}

func actFlee(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	mob.Command("flee")
	return Success
}

func actCast(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	spell := getStringParam(params, "spell")
	if spell == "" {
		return Failure
	}
	mob.Command("cast " + spell)
	return Success
}

// actAddBuff applies a buff to the acting mob.
// params: buff_id (int)
func actAddBuff(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	buffId := getIntParam(params, "buff_id")
	if buffId == 0 {
		return Failure
	}
	mob.AddBuff(buffId, "behaviortree")
	return Success
}

// actRemoveBuff removes a buff from the triggering player by buff ID.
// params: buff_id (int)
func actRemoveBuff(params map[string]any, ctx *EvalContext) Result {
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}
	buffId := getIntParam(params, "buff_id")
	user.Character.RemoveBuff(buffId)
	return Success
}

// actTargetRandomPlayerInRoom picks a random player in the
// caller's current room and stashes them as the EvalContext's
// SoftTarget. This is the non-combat target-picker primitive
// used by skullduggery archetypes (thief, future shadow/plant
// variants).
//
// CRITICAL: this action does NOT call SetAggro or transition
// Combat Phase. The picked player is a "soft target" — the
// caller's NEXT action (try_steal, try_plant, etc.) consumes
// SoftTarget without triggering combat. This is the structural
// fix for the chunk 2.7 thief-archetype bug; non-combat target
// picking must not silently engage combat.
//
// Returns Failure when no players are present in the room.
func actTargetRandomPlayerInRoom(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	playerIds := room.GetPlayers()
	if len(playerIds) == 0 {
		return Failure
	}
	idx := util.Rand(len(playerIds))
	pickedId := playerIds[idx]

	// CRITICAL: Stash in SoftTarget. Do NOT call SetAggro.
	// Combat target lives on Combat Phase's Engaged state ONLY.
	ctx.SoftTarget = state.ActorRef{UserId: pickedId}
	return Success
}

// actTargetWeakestMobInRoom scans room.GetMobs(), computes
// PowerScore(target) / PowerScore(self) for each candidate that
// passes mob.HatesMob, picks the lowest ratio strictly below the
// ratio_below ceiling (default 1.0), and sets it as Aggro.
// Returns Success on a successful target pick, Failure otherwise.
//
// Skips: self, dead mobs, non-combatant mobs, mobs the caller's
// HatesMob returns false for, and (if caller is itself charmed)
// fellow companions of the same owner.
//
// Players are NOT scanned — predation is a mob-vs-mob action.
// Player aggression continues through the standard hostile-mob
// attack chain.
func actTargetWeakestMobInRoom(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || mob.IsNonCombatant() {
		return Failure
	}
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}

	selfPower := combat.PowerScore(mob.Character)
	if selfPower <= 0 {
		return Failure
	}
	// ratio_below defaults to 1.0 (engage anyone strictly weaker).
	ceiling := getFloatParam(params, "ratio_below", 1.0)

	callerCharmedBy := mob.Character.GetCharmedUserId()
	var bestId int
	bestRatio := ceiling
	for _, otherId := range room.GetMobs() {
		if otherId == mob.InstanceId {
			continue
		}
		other := mobs.GetInstance(otherId)
		if other == nil || other.IsNonCombatant() {
			continue
		}
		if other.Character.Health <= 0 {
			continue
		}
		// Companion-allegiance skip: if caller is itself charmed,
		// skip fellow companions of the same owner. A wild caller
		// can still prey on a player's companion if HatesMob says
		// so — companions of an enemy are still enemies.
		if callerCharmedBy > 0 && other.Character.IsCharmed(callerCharmedBy) {
			continue
		}
		if !mob.HatesMob(other) {
			continue
		}
		targetPower := combat.PowerScore(other.Character)
		if targetPower <= 0 {
			continue
		}
		ratio := targetPower / selfPower
		if ratio < bestRatio {
			bestRatio = ratio
			bestId = otherId
		}
	}
	if bestId == 0 {
		return Failure
	}
	mob.Character.SetAggro(0, bestId, characters.DefaultAttack)
	return Success
}
