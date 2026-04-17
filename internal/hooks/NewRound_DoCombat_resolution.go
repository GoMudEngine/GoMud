package hooks

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobai"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// File: NewRound_DoCombat_resolution.go
//
// Phase helpers for the per-combatant round handlers (handlePlayerVsMob
// and handleMobVsPlayer). Extracted during 1.2a god-function refactor.

// applyCombatDamageBonuses_PvM applies the Conviction Surge, Adrenaline
// Surge, return damage, and lifesteal stages for a player-vs-mob swing.
// Mutation order matches the original inline block: bonus damage is added
// before return damage and lifesteal are computed.
func applyCombatDamageBonuses_PvM(roundResult *combat.AttackResult, user *users.UserRecord, defMob *mobs.Mob, uRoom *rooms.Room, defRoom *rooms.Room) {
	// Phase 25.3: Conviction Surge buff
	if roundResult.Hit && roundResult.DamageToTarget > 0 && user.Character.HasBuffFlag(buffs.DamageBonus) {
		bonusDmg := int(math.Round(float64(roundResult.DamageToTarget) * 0.15))
		if bonusDmg < 1 {
			bonusDmg = 1
		}
		defMob.Character.Health -= bonusDmg
		roundResult.DamageToTarget += bonusDmg
	}

	// Stage 12.2: Adrenaline Surge
	if roundResult.Hit && roundResult.DamageToTarget > 0 {
		if mutations.IsAdrenalSurgeActive(user.Character.Mutations, user.Character.Health, user.Character.HealthMax.Value) {
			if surgeBonus := mutations.GetAdrenalSurgeBonus(user.Character.Mutations); surgeBonus > 0 {
				bonusDmg := int(math.Round(float64(roundResult.DamageToTarget) * surgeBonus))
				if bonusDmg < 1 {
					bonusDmg = 1
				}
				defMob.Character.Health -= bonusDmg
				roundResult.DamageToTarget += bonusDmg
			}
		}
	}

	// Return damage — fire elementals, battlerager armor, etc.
	// Direct HP reduction; does NOT trigger another combat round (no recursion risk).
	if roundResult.Hit && roundResult.DamageToTarget > 0 {
		returnPct := defMob.Character.StatMod("return_damage")
		if sp := species.GetSpecies(defMob.Character.SpeciesId); sp != nil {
			returnPct += sp.ReturnDamage
		}
		if returnPct > 0 {
			returnDmg := int(float64(roundResult.DamageToTarget) * float64(returnPct) / 100.0)
			if returnDmg > 0 {
				user.Character.Health -= returnDmg
				dmgDesc := combat.GetDamageDescription(returnDmg, user.Character.HealthMax.Value)
				defMobName := mobDisplayName(defMob, defRoom, user.UserId)
				sendVisualRoomText(uRoom, fmt.Sprintf(
					`<ansi fg="red">%s recoils from striking %s! (%s)</ansi>`,
					user.Character.Name, defMobName, dmgDesc), user.UserId)
				user.SendText(fmt.Sprintf(
					`<ansi fg="red">You recoil from striking %s! (%s)</ansi>`,
					defMobName, dmgDesc))
			}
		}
	}

	// Lifesteal — Hungering Touch enchantment heals attacker on hit
	if roundResult.Hit && roundResult.DamageToTarget > 0 {
		lifestealPct := user.Character.StatMod("lifesteal_pct")
		if lifestealPct > 0 {
			healAmt := int(float64(roundResult.DamageToTarget) * float64(lifestealPct) / 100.0)
			if healAmt > 0 {
				user.Character.Health += healAmt
				if user.Character.Health > user.Character.HealthMax.Value {
					user.Character.Health = user.Character.HealthMax.Value
				}
				healDesc := combat.GetHealDescription(healAmt, user.Character.HealthMax.Value)
				user.SendText(fmt.Sprintf(
					`<ansi fg="green">Your weapon feeds on the blow! (%s)</ansi>`,
					healDesc))
			}
		}
	}
}

// applyCombatDamageBonuses_MvP applies the Conviction Surge, Adrenaline
// Surge, return damage, and lifesteal stages for a mob-vs-player swing.
// Minor Shield reduction is NOT handled here — it remains inline in the
// parent because ConditionShield is player-defender-only.
func applyCombatDamageBonuses_MvP(roundResult *combat.AttackResult, mob *mobs.Mob, defUser *users.UserRecord, mobRoom *rooms.Room) {
	// Conviction Surge buff: +15% damage bonus (mob attacker)
	if roundResult.Hit && roundResult.DamageToTarget > 0 && mob.Character.HasBuffFlag(buffs.DamageBonus) {
		bonusDmg := int(math.Round(float64(roundResult.DamageToTarget) * 0.15))
		if bonusDmg < 1 {
			bonusDmg = 1
		}
		defUser.Character.Health -= bonusDmg
		roundResult.DamageToTarget += bonusDmg
	}

	// Stage 12.2: Adrenaline Surge
	if roundResult.Hit && roundResult.DamageToTarget > 0 {
		if mutations.IsAdrenalSurgeActive(mob.Character.Mutations, mob.Character.Health, mob.Character.HealthMax.Value) {
			if surgeBonus := mutations.GetAdrenalSurgeBonus(mob.Character.Mutations); surgeBonus > 0 {
				bonusDmg := int(math.Round(float64(roundResult.DamageToTarget) * surgeBonus))
				if bonusDmg < 1 {
					bonusDmg = 1
				}
				defUser.Character.Health -= bonusDmg
				roundResult.DamageToTarget += bonusDmg
			}
		}
	}

	// Return damage — fire elementals, battlerager armor, etc.
	// Direct HP reduction; does NOT trigger another combat round (no recursion risk).
	if roundResult.Hit && roundResult.DamageToTarget > 0 {
		returnPct := defUser.Character.StatMod("return_damage")
		if sp := species.GetSpecies(defUser.Character.SpeciesId); sp != nil {
			returnPct += sp.ReturnDamage
		}
		if returnPct > 0 {
			returnDmg := int(float64(roundResult.DamageToTarget) * float64(returnPct) / 100.0)
			if returnDmg > 0 {
				mob.Character.Health -= returnDmg
				dmgDesc := combat.GetDamageDescription(returnDmg, mob.Character.HealthMax.Value)
				mvpMobName := mobDisplayName(mob, mobRoom, defUser.UserId)
				defUser.SendText(fmt.Sprintf(
					`<ansi fg="red">%s recoils from striking you! (%s)</ansi>`,
					mvpMobName, dmgDesc))
				sendVisualRoomText(mobRoom, fmt.Sprintf(
					`<ansi fg="red">%s recoils from striking %s! (%s)</ansi>`,
					mvpMobName, defUser.Character.Name, dmgDesc), defUser.UserId)
			}
		}
	}

	// Lifesteal — mob enchantment heals attacker on hit
	if roundResult.Hit && roundResult.DamageToTarget > 0 {
		lifestealPct := mob.Character.StatMod("lifesteal_pct")
		if lifestealPct > 0 {
			healAmt := int(float64(roundResult.DamageToTarget) * float64(lifestealPct) / 100.0)
			if healAmt > 0 {
				mob.Character.Health += healAmt
				if mob.Character.Health > mob.Character.HealthMax.Value {
					mob.Character.Health = mob.Character.HealthMax.Value
				}
			}
		}
	}
}

// handleCombatWaitRound handles the RoundsWaiting > 0 short-circuit shared
// by handlePlayerVsMob and handleMobVsPlayer. Returns true if the caller
// should return immediately (i.e. the attacker is still waiting).
//
// attackerUser is non-nil when the attacker is a player (PvM).
// defenderUser is non-nil when the defender is a player (MvP).
// viewerUserId is the user ID passed to sendVisualRoomText and
// sendDarkRoomCombatFallback — always the user participant.
func handleCombatWaitRound(
	attackerChar *characters.Character,
	defenderChar *characters.Character,
	roleSource combat.SourceTarget,
	roleTarget combat.SourceTarget,
	attackerUser *users.UserRecord,
	defenderUser *users.UserRecord,
	attackerRoom *rooms.Room,
	defenderRoom *rooms.Room,
	viewerUserId int,
) bool {
	if attackerChar.Aggro.RoundsWaiting <= 0 {
		return false
	}
	mudlog.Debug(`RoundsWaiting`, `User`, attackerChar.Name, `Rounds`, attackerChar.Aggro.RoundsWaiting)
	attackerChar.Aggro.RoundsWaiting--

	roundResult := combat.GetWaitMessages(items.Wait, attackerChar, defenderChar, roleSource, roleTarget)

	for _, msg := range roundResult.MessagesToSource {
		if attackerUser != nil {
			attackerUser.SendText(msg)
		}
	}
	for _, msg := range roundResult.MessagesToTarget {
		if defenderUser != nil {
			defenderUser.SendText(msg)
		}
	}
	for _, msg := range roundResult.MessagesToSourceRoom {
		sendVisualRoomText(attackerRoom, msg, viewerUserId)
	}
	for _, msg := range roundResult.MessagesToTargetRoom {
		sendVisualRoomText(defenderRoom, msg, viewerUserId)
	}
	sendDarkRoomCombatFallback(attackerRoom, viewerUserId)
	if defenderRoom != attackerRoom {
		sendDarkRoomCombatFallback(defenderRoom, viewerUserId)
	}
	return true
}

// handleMvPCritAndMessaging runs the crit-effect + charmed-mob-assist +
// darkness-replacement + buff + per-round message-emission block for
// mob-vs-player. Must be called AFTER combat.RecordAttack and AFTER the
// damage-bonus helper; mutates defUser/mob buffs and emits every per-round
// message. The mid-block room re-lookup is intentional — the mob may have
// moved during crit effects.
func handleMvPCritAndMessaging(roundResult *combat.AttackResult, mob *mobs.Mob, defUser *users.UserRecord, mobRoom *rooms.Room, defRoom *rooms.Room) {
	// Crit effects (player defending)
	mvpCritResult := applyCritEffects(&mob.Character, defUser.Character, *roundResult, mobRoom)
	if mvpCritResult.DefenderMsg != "" {
		defUser.SendText(mvpCritResult.DefenderMsg)
	}
	if mvpCritResult.RoomMsg != "" {
		mobRoom.SendText(mvpCritResult.RoomMsg, defUser.UserId)
	}

	// Charmed mob assist
	roomId := mob.Character.RoomId
	room := rooms.LoadRoom(roomId)
	handleCharmedMobAssist(room, defUser.UserId, fmt.Sprintf("#%d", mob.InstanceId))

	// Apply darkness message replacement for blind defender
	mvpTgtCanSee := canSeeInRoom(defUser.Character, defRoom)
	replaceDarknessMessages(roundResult, true, mvpTgtCanSee) // mob source doesn't need text replacement

	for _, buffId := range roundResult.BuffSource {
		mob.AddBuff(buffId, `combat`)
	}
	for _, buffId := range roundResult.BuffTarget {
		defUser.AddBuff(buffId, `combat`)
	}
	for _, msg := range roundResult.MessagesToTarget {
		defUser.SendText(msg)
	}
	for _, msg := range roundResult.MessagesToSourceRoom {
		sendVisualRoomText(mobRoom, msg, defUser.UserId)
	}
	for _, msg := range roundResult.MessagesToTargetRoom {
		sendVisualRoomText(defRoom, msg, defUser.UserId)
	}
	sendDarkRoomCombatFallback(mobRoom, defUser.UserId)
	if defRoom != mobRoom {
		sendDarkRoomCombatFallback(defRoom, defUser.UserId)
	}
}

// handleMvPProgression runs the concentration break, offhand break,
// crit-received hook, mob attacker stat/skill progression, and defender
// progression block for mob-vs-player.
func handleMvPProgression(roundResult *combat.AttackResult, mob *mobs.Mob, defUser *users.UserRecord, mobRoom *rooms.Room, defRoom *rooms.Room) {
	handlePlayerConcentrationBreak(defUser, *roundResult, defRoom)
	handleOffhandBreakUserDef(*roundResult, defUser, defRoom)

	// Physical crit received → vitality progression for defender
	if roundResult.Hit && roundResult.Crit {
		defUser.Character.OnCritReceived("physical", defUser.UserId)
	}

	// Stage 38.3: Mob attacker progression — per-weapon skill tracking
	statMobName := mobDisplayName(mob, mobRoom, 0)
	if gained := mob.Character.OnStatUse("strength", 0); gained {
		if tmpl, ok := characters.MobStatGainMessages["strength"]; ok {
			mobRoom.SendText(fmt.Sprintf(tmpl, statMobName))
		}
	}
	if gained := mob.Character.OnStatUse("dexterity", 0); gained {
		if tmpl, ok := characters.MobStatGainMessages["dexterity"]; ok {
			mobRoom.SendText(fmt.Sprintf(tmpl, statMobName))
		}
	}
	for _, wh := range roundResult.WeaponHits {
		if wh.Hit {
			mob.Character.OnSkillUse(wh.SkillTag, 0)
			if wh.Crit {
				mob.Character.OnCriticalSuccess(wh.SkillTag, 0)
			}
		} else if wh.Fumble {
			mob.Character.OnCriticalFailure(wh.SkillTag, 0)
		}
	}
	if len(roundResult.WeaponHits) == 0 && roundResult.Hit {
		mob.Character.OnSkillUse(string(skills.UnarmedCombat), 0)
	}

	// Defender progression — dodge/parry/block train specific skills
	processDefenderProgression(defUser.Character, defUser.UserId, *roundResult)
}

// handleMvPRoundResolution handles the end-of-round health-check logic for
// mob-vs-player: if either combatant dropped, end the player's aggro and
// retarget or end the mob's; otherwise re-assert the mob's aggro on the
// current defender.
func handleMvPRoundResolution(mob *mobs.Mob, defUser *users.UserRecord, mobRoom *rooms.Room) {
	if mob.Character.Health <= 0 || defUser.Character.Health <= 0 {
		defUser.Character.EndAggro()
		if mob.Character.Health > 0 {
			RetargetOrEnd(&mob.Character, mobRoom, 0, mob.InstanceId)
		} else {
			mob.Character.EndAggro()
		}
	} else {
		mob.Character.SetAggro(defUser.UserId, 0, characters.DefaultAttack)
	}
}

// resolvePlayerVsMobTarget validates the user's current combat target is
// reachable, loads its room, emits combat_start for first-round defender
// mobs (BEFORE the attack roll so round-1 dies still fire reactive AI),
// cancels combat-cancel buffs, and guards Health < 1.
//
// Returns (defMob, defRoom, ok). On ok == false the helper has already
// called user.Character.EndAggro() and/or sent the "can't be found"
// message; caller should return immediately.
func resolvePlayerVsMobTarget(user *users.UserRecord, uRoom *rooms.Room, evt events.NewRound) (defMob *mobs.Mob, defRoom *rooms.Room, ok bool) {
	defMob = mobs.GetInstance(user.Character.Aggro.MobInstanceId)

	targetFound := true
	if defMob == nil {
		targetFound = false
	} else if defMob.Character.RoomId != user.Character.RoomId {
		if user.Character.Aggro.ExitName == `` {
			targetFound = false
		} else {
			if uRoom == nil {
				user.Character.EndAggro()
				return nil, nil, false
			}
			if _, exitRoomId := uRoom.FindExitByName(user.Character.Aggro.ExitName); exitRoomId != defMob.Character.RoomId {
				targetFound = false
			}
		}
	}

	if !targetFound {
		user.SendText("Your target can't be found.")
		user.Character.EndAggro()
		return nil, nil, false
	}

	defRoom = rooms.LoadRoom(defMob.Character.RoomId)
	if defRoom == nil {
		user.Character.EndAggro()
		return nil, nil, false
	}

	// Emit combat_start for the defender mob if this is its first round
	// in combat. We do this here (before the attack roll) so mobs that
	// die in round 1 still get their reactive AI signal fired.
	if defMob.CombatMemory == nil && defMob.Character.Aggro != nil {
		mudlog.Debug("MobAI", "emit", "combat_start", "mob", defMob.Character.Name, "mobId", defMob.InstanceId, "source", "handlePlayerVsMob")
		defMob.CombatMemory = mobai.SetMemory(
			defMob.Character.Aggro.UserId,
			defMob.Character.Aggro.MobInstanceId,
			defMob.Character.RoomId,
			evt.RoundNumber,
		)
		events.AddToQueue(events.MobAISignal{
			MobInstanceId: defMob.InstanceId,
			SignalType:    "combat_start",
			RoomId:        defMob.Character.RoomId,
		})
	}

	defMob.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

	if defMob.Character.Health < 1 {
		user.Character.EndAggro()
		return nil, nil, false
	}

	return defMob, defRoom, true
}

// handlePvMCritAndMessaging runs the crit-effect + darkness-replacement +
// buff + per-round message-emission block for player-vs-mob. Must be
// called AFTER combat.RecordAttack and AFTER the damage-bonus helper.
func handlePvMCritAndMessaging(roundResult *combat.AttackResult, user *users.UserRecord, defMob *mobs.Mob, uRoom *rooms.Room, defRoom *rooms.Room, roundNumber uint64) {
	// Stage 30.1: Record combat analytics
	pvmAtkType := "unarmed"
	if user.Character.Equipment.Weapon.ItemId > 0 {
		pvmAtkType = "weapon"
	}
	combat.RecordAttack(*roundResult, combat.User, combat.Mob, pvmAtkType, user.Character, &defMob.Character, roundNumber)

	pvmCritResult := applyCritEffects(user.Character, &defMob.Character, *roundResult, uRoom)
	dispatchCritEffectsPvM(pvmCritResult, user, defMob, uRoom)

	// Apply darkness message replacement for blind attacker
	pvmSrcCanSee := canSeeInRoom(user.Character, uRoom)
	replaceDarknessMessages(roundResult, pvmSrcCanSee, true) // mobs don't receive text messages

	for _, buffId := range roundResult.BuffSource {
		user.AddBuff(buffId, `combat`)
	}
	for _, buffId := range roundResult.BuffTarget {
		defMob.AddBuff(buffId, `combat`)
	}
	for _, msg := range roundResult.MessagesToSource {
		user.SendText(msg)
	}
	for _, msg := range roundResult.MessagesToSourceRoom {
		sendVisualRoomText(uRoom, msg, user.UserId)
	}
	for _, msg := range roundResult.MessagesToTargetRoom {
		sendVisualRoomText(defRoom, msg, user.UserId)
	}
	sendDarkRoomCombatFallback(uRoom, user.UserId)
	if defRoom != uRoom {
		sendDarkRoomCombatFallback(defRoom, user.UserId)
	}
}

// handlePvMProgressionAndAggro runs the defender concentration break,
// mob_hurt behavior, attacker stat/skill progression, defender
// progression, hostility-group marking, mob-aggro-on-attack (with
// exit-walk if the mob is in a different room), and companion-owner
// assist block for player-vs-mob.
func handlePvMProgressionAndAggro(roundResult *combat.AttackResult, user *users.UserRecord, defMob *mobs.Mob, uRoom *rooms.Room, cfg *configs.Config) {
	// Stage 11.5: Mob concentration break when hit
	if checkConcentrationBreak(&defMob.Character, roundResult.DamageToTarget) {
		recordConcentrationFailure(combat.Mob, combat.User, &defMob.Character, castingTargetChar(defMob.Character.CastingState))
		defMob.Character.CastingState = nil
		uRoom.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s concentration breaks.`, defMob.Character.Name))
	}

	// Handle any scripted behavior now.
	if roundResult.Hit {
		// Behavior tree: fire-and-forget
		behaviortree.TryMobBehavior(defMob.InstanceId, behaviortree.EventContext{
			EventType: "mob_hurt",
			UserId:    user.UserId,
			RoomId:    defMob.Character.RoomId,
		})
	}

	// Stage 38.3: Player attacker progression — per-weapon skill tracking
	user.Character.OnStatUse("strength", user.UserId)
	user.Character.OnStatUse("dexterity", user.UserId)
	for _, wh := range roundResult.WeaponHits {
		if wh.Hit {
			user.Character.OnSkillUse(wh.SkillTag, user.UserId)
			if wh.Crit {
				user.Character.OnCriticalSuccess(wh.SkillTag, user.UserId)
			}
		} else if wh.Fumble {
			user.Character.OnCriticalFailure(wh.SkillTag, user.UserId)
		}
	}
	// Unarmed progression when fighting with no weapons at all
	if len(roundResult.WeaponHits) == 0 && roundResult.Hit {
		user.Character.OnSkillUse(string(skills.UnarmedCombat), user.UserId)
	}

	// Defender progression — dodge/parry/block train specific skills
	processDefenderProgression(&defMob.Character, 0, *roundResult)

	// Hostility
	for _, groupName := range defMob.Groups {
		mobs.MakeHostile(groupName, user.UserId, cfg.Timing.MinutesToRounds(2)-user.Character.Stats.Charisma.ValueAdj)
	}

	// Mobs get aggro when attacked
	if defMob.Character.Aggro == nil {
		defMob.PreventIdle = true
		if user.Character.RoomId != defMob.Character.RoomId {
			if mobRoom := rooms.LoadRoom(defMob.Character.RoomId); mobRoom != nil {
				for exitName, exitInfo := range mobRoom.Exits {
					if exitInfo.RoomId == user.Character.RoomId {
						defMob.Command(fmt.Sprintf(`go %s`, exitName))
						if actionStr := defMob.GetAngryCommand(); actionStr != `` {
							defMob.Command(actionStr)
						}
						break
					}
				}
			}
		}
		defMob.Command(fmt.Sprintf("attack @%d", user.UserId))
	}

	// Bidirectional autoassist: companion owner fights back when their companion is attacked.
	handleCompanionOwnerAssist(defMob, fmt.Sprintf("@%d", user.UserId))
}

// handlePvMRoundResolution handles the end-of-round health-check logic
// for player-vs-mob: if either combatant dropped, end the mob's aggro
// and retarget the player (with a "You turn your attention to..."
// message) or end the player's; otherwise re-assert the player's aggro
// on the current defender.
func handlePvMRoundResolution(user *users.UserRecord, defMob *mobs.Mob, uRoom *rooms.Room) {
	if user.Character.Health <= 0 || defMob.Character.Health <= 0 {
		defMob.Character.EndAggro()
		if user.Character.Health > 0 {
			if RetargetOrEnd(user.Character, uRoom, user.UserId, 0) {
				if mob := mobs.GetInstance(user.Character.Aggro.MobInstanceId); mob != nil {
					user.SendText(fmt.Sprintf("You turn your attention to <ansi fg=\"mobname\">%s</ansi>!", mob.Character.Name))
				} else if newDef := users.GetByUserId(user.Character.Aggro.UserId); newDef != nil {
					user.SendText(fmt.Sprintf("You turn your attention to <ansi fg=\"username\">%s</ansi>!", newDef.Character.Name))
				}
			}
		} else {
			user.Character.EndAggro()
		}
	} else {
		user.Character.SetAggro(0, defMob.InstanceId, characters.DefaultAttack)
	}
}
