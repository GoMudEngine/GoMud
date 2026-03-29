package hooks

import (
	"fmt"
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// mobDisplayName returns the formatted display name for a mob in combat text,
// including duplicate index coloring when multiple mobs share the same name.
func mobDisplayName(mob *mobs.Mob, room *rooms.Room, viewingUserId int) string {
	dupIdx := room.GetMobDuplicateIndex(mob.InstanceId)
	return mob.Character.GetMobNameIndexed(viewingUserId, dupIdx).String()
}

// sendCombatRoomText sends a visual combat message to room observers.
// In lit rooms, all players see the message. In dark rooms, only players
// with nightvision see the visual text; others receive nothing here
// (a one-time sound fallback is sent per round instead).
func sendCombatRoomText(room *rooms.Room, visualMsg string, excludeUserIds ...int) {
	if room == nil {
		return
	}
	if room.GetVisibility() >= 1 {
		room.SendText(visualMsg, excludeUserIds...)
		return
	}
	for _, uid := range room.GetPlayers() {
		if isExcludedUser(uid, excludeUserIds) {
			continue
		}
		u := users.GetByUserId(uid)
		if u != nil && u.Character.HasFlagFromAnySource(buffs.NightVision) {
			u.SendText(visualMsg)
		}
	}
}

// isExcludedUser checks if a userId is in the exclusion list.
func isExcludedUser(uid int, excludeIds []int) bool {
	for _, id := range excludeIds {
		if uid == id {
			return true
		}
	}
	return false
}

// sendDarkRoomCombatFallback sends a one-time "sounds of fighting" message
// to non-nightvision players in dark rooms.
func sendDarkRoomCombatFallback(room *rooms.Room, excludeUserIds ...int) {
	if room == nil || room.GetVisibility() >= 1 {
		return
	}
	for _, uid := range room.GetPlayers() {
		if isExcludedUser(uid, excludeUserIds) {
			continue
		}
		u := users.GetByUserId(uid)
		if u != nil && !u.Character.HasFlagFromAnySource(buffs.NightVision) {
			u.SendText(`<ansi fg="yellow">You hear the sounds of fighting nearby.</ansi>`)
		}
	}
}

// canSeeInRoom returns true if the character has nightvision or the room
// has enough visibility for sight.
func canSeeInRoom(char *characters.Character, room *rooms.Room) bool {
	if room == nil {
		return true
	}
	return room.GetVisibility() >= 1 || char.HasFlagFromAnySource(buffs.NightVision)
}

// replaceDarknessMessages replaces detailed combat messages with generic
// darkness text for combatants who cannot see.
func replaceDarknessMessages(result *combat.AttackResult, sourceCanSee bool, targetCanSee bool) {
	if sourceCanSee && targetCanSee {
		return
	}

	// Build replacement messages based on swing events
	if !sourceCanSee {
		newMsgs := make([]string, 0, len(result.SwingEvents))
		for _, se := range result.SwingEvents {
			if se.DoubleFumble {
				// Keep comedy double fumble text
				continue
			}
			if se.Fumble {
				newMsgs = append(newMsgs, `<ansi fg="fumble-text">!!!</ansi> <ansi fg="yellow">You stumble badly in the darkness!</ansi> <ansi fg="fumble-text">!!!</ansi>`)
			} else if se.Crit {
				newMsgs = append(newMsgs, `<ansi fg="crit-text">***</ansi> <ansi fg="attack-good">You land a devastating blow in the dark!</ansi> <ansi fg="crit-text">***</ansi>`)
			} else if se.DefenseCrit || se.DefenseUsed != "" {
				newMsgs = append(newMsgs, `<ansi fg="attack-bad">Your attack is deflected by something!</ansi>`)
			} else if se.Hit {
				newMsgs = append(newMsgs, `<ansi fg="attack-good">You strike blindly and connect!</ansi>`)
			} else {
				newMsgs = append(newMsgs, `<ansi fg="yellow">You swing wildly in the darkness!</ansi>`)
			}
		}
		if len(newMsgs) > 0 {
			result.MessagesToSource = newMsgs
		}
	}

	if !targetCanSee {
		newMsgs := make([]string, 0, len(result.SwingEvents))
		for _, se := range result.SwingEvents {
			if se.DoubleFumble {
				// Keep comedy double fumble text
				continue
			}
			if se.Fumble {
				newMsgs = append(newMsgs, `<ansi fg="yellow">You hear your attacker stumble!</ansi>`)
			} else if se.Crit {
				newMsgs = append(newMsgs, `<ansi fg="crit-text">***</ansi> <ansi fg="red">Something hits you hard in the dark!</ansi> <ansi fg="crit-text">***</ansi>`)
			} else if se.DefenseCrit || se.DefenseUsed != "" {
				newMsgs = append(newMsgs, `<ansi fg="defense-good">You fend off something in the dark!</ansi>`)
			} else if se.Hit {
				newMsgs = append(newMsgs, `<ansi fg="red">Something strikes you in the dark!</ansi>`)
			} else {
				newMsgs = append(newMsgs, `<ansi fg="yellow">You hear something whoosh past!</ansi>`)
			}
		}
		if len(newMsgs) > 0 {
			result.MessagesToTarget = newMsgs
		}
	}
}

// handlePlayerShieldDecay processes Minor Shield round expiry for a player.
func handlePlayerShieldDecay(user *users.UserRecord) {
	if user.Character.HasCondition(characters.ConditionShield) {
		if user.Character.GetConditionDuration(characters.ConditionShield) <= 1 {
			user.Character.RemoveCondition(characters.ConditionShield)
			user.SendText(`<ansi fg="blue">Your Minor Shield dissipates.</ansi>`)
		} else {
			user.Character.DecrementCondition(characters.ConditionShield)
		}
	}
}

// castingTargetChar returns the first target character from a CastingState, or nil.
func castingTargetChar(cs *characters.CastingState) *characters.Character {
	if cs == nil {
		return nil
	}
	for _, mobInstId := range cs.TargetMobInstanceIds {
		if m := mobs.GetInstance(mobInstId); m != nil {
			return &m.Character
		}
	}
	for _, uid := range cs.TargetUserIds {
		if u := users.GetByUserId(uid); u != nil {
			return u.Character
		}
	}
	return nil
}

// recordConcentrationFailure records a fizzle event for a broken spell.
func recordConcentrationFailure(src, tgt combat.SourceTarget, srcChar *characters.Character, tgtChar *characters.Character) {
	combat.RecordSpell(src, tgt, false, false, false, true, 0, 0, srcChar, tgtChar, util.GetRoundCount())
}

// handlePlayerFoldCasting processes fold spell casting for a player.
// Returns true if the player is casting and should skip combat.
func handlePlayerFoldCasting(user *users.UserRecord, userId int) bool {
	if user.Character.CastingState == nil {
		return false
	}

	// Bleeding out = automatic concentration break
	if user.Character.IsDisabled() {
		recordConcentrationFailure(combat.User, combat.Mob, user.Character, castingTargetChar(user.Character.CastingState))
		user.Character.CastingState = nil
		return true
	}

	// Stage 11.3: prone = automatic concentration break
	if user.Character.CombatPosition == characters.PositionProne {
		recordConcentrationFailure(combat.User, combat.Mob, user.Character, castingTargetChar(user.Character.CastingState))
		user.Character.CastingState = nil
		user.SendText(`<ansi fg="red">You lose your concentration as you hit the ground!</ansi>`)
		room := rooms.LoadRoom(user.Character.RoomId)
		if room != nil {
			room.SendText(fmt.Sprintf(
				`<ansi fg="username">%s</ansi>'s concentration breaks.`, user.Character.Name), user.UserId)
		}
		return true
	}

	cs := user.Character.CastingState

	// Check if the spell target is still alive
	targetGone := false
	for _, mobInstId := range cs.TargetMobInstanceIds {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil || mob.Character.Health < 1 {
			targetGone = true
			break
		}
	}
	for _, targetUserId := range cs.TargetUserIds {
		if u := users.GetByUserId(targetUserId); u == nil || u.Character.Health < 1 {
			targetGone = true
			break
		}
	}
	if targetGone {
		recordConcentrationFailure(combat.User, combat.Mob, user.Character, castingTargetChar(user.Character.CastingState))
		user.Character.CastingState = nil
		user.SendText(`<ansi fg="red">Your spell fizzles — the target is gone.</ansi>`)
		return true
	}

	spellData := spells.GetSpell(cs.SpellId)
	if spellData == nil {
		user.Character.CastingState = nil
		user.SendText(`<ansi fg="red">The spell dissipates — its data cannot be found.</ansi>`)
		return true
	}

	// Pre-flight: simulate fold advance to compute conviction cost
	foldDelta := simulateFoldRound(cs)
	roundCost := calcFoldConvictionCost(cs, foldDelta)

	if roundCost > 0 && user.Character.Conviction < roundCost {
		recordConcentrationFailure(combat.User, combat.Mob, user.Character, castingTargetChar(user.Character.CastingState))
		user.Character.CastingState = nil
		user.SendText(`<ansi fg="red">Your conviction wavers — the fold collapses.</ansi>`)
		return true
	}

	user.Character.Conviction -= roundCost
	cs.ConvictionSpent += roundCost

	// Advance folds — resolve spell if complete
	if advanceFolds(cs) {
		resolveRoom := rooms.LoadRoom(user.Character.RoomId)
		if resolveRoom != nil {
			resolveSpell(user, cs, spellData, resolveRoom)
		}
		user.Character.TrackSpellCast(cs.SpellId)
		user.Character.OnSkillUse(string(skills.Spellcasting), userId)
		user.Character.OnStatUse("willpower", userId)

		// Phase 25.1: Spell discovery
		castSkillLevel := user.Character.GetSkillLevel(skills.Spellcasting)
		knownCount := len(user.Character.SpellBook)
		bal := configs.GetBalanceConfig()
		discoveryChance := float64(bal.SpellDiscoveryBaseChance) / (1.0 + float64(knownCount)*float64(bal.SpellDiscoveryDecayRate))
		if util.Rand(100) < int(discoveryChance) {
			eligible := spells.GetEligibleSpells(user.Character.SpellBook, castSkillLevel)
			if len(eligible) > 0 {
				pick := eligible[util.Rand(len(eligible))]
				if user.Character.LearnSpell(pick) {
					if newSpell := spells.GetSpell(pick); newSpell != nil {
						user.SendText(fmt.Sprintf(
							`<ansi fg="magenta-bold">A new pattern crystallizes in your mind: <ansi fg="cyan-bold">%s</ansi></ansi>`,
							newSpell.Name))
					}
				}
			}
		}

		user.Character.CastingState = nil
	} else {
		user.SendText(`<ansi fg="cyan">` + spells.GetCastMessage("cast_started", cs.SpellId) + `</ansi>`)
	}

	return true
}

// handleMobFoldCasting processes fold spell casting for a mob.
// Returns true if the mob is casting and should skip combat.
func handleMobFoldCasting(mob *mobs.Mob, mobRoom *rooms.Room) bool {
	if mob.Character.CastingState == nil {
		return false
	}

	if mob.Character.CombatPosition == characters.PositionProne {
		recordConcentrationFailure(combat.Mob, combat.User, &mob.Character, castingTargetChar(mob.Character.CastingState))
		mob.Character.CastingState = nil
		mobRoom.SendText(fmt.Sprintf(
			`%s's concentration breaks.`, mobDisplayName(mob, mobRoom, 0)))
		return true
	}

	cs := mob.Character.CastingState

	// Check if the spell target is still alive
	mobTargetGone := false
	for _, targetUserId := range cs.TargetUserIds {
		if u := users.GetByUserId(targetUserId); u == nil || u.Character.Health < 1 {
			mobTargetGone = true
			break
		}
	}
	for _, mobInstId := range cs.TargetMobInstanceIds {
		tm := mobs.GetInstance(mobInstId)
		if tm == nil || tm.Character.Health < 1 {
			mobTargetGone = true
			break
		}
	}
	if mobTargetGone {
		recordConcentrationFailure(combat.Mob, combat.User, &mob.Character, castingTargetChar(mob.Character.CastingState))
		mob.Character.CastingState = nil
		mobRoom.SendText(fmt.Sprintf(
			`%s's spell fizzles.`, mobDisplayName(mob, mobRoom, 0)))
		return true
	}

	spellData := spells.GetSpell(cs.SpellId)
	if spellData == nil {
		mob.Character.CastingState = nil
		return true
	}

	// Pre-flight: simulate fold advance to compute conviction cost
	foldDelta := simulateFoldRound(cs)
	roundCost := calcFoldConvictionCost(cs, foldDelta)

	if roundCost > 0 && mob.Character.Conviction < roundCost {
		recordConcentrationFailure(combat.Mob, combat.User, &mob.Character, castingTargetChar(mob.Character.CastingState))
		mob.Character.CastingState = nil
		mobRoom.SendText(fmt.Sprintf(
			`%s's spell falters.`, mobDisplayName(mob, mobRoom, 0)))
		return true
	}
	mob.Character.Conviction -= roundCost

	// Advance folds — resolve spell if complete
	if advanceFolds(cs) {
		if resolveRoom := rooms.LoadRoom(mob.Character.RoomId); resolveRoom != nil {
			resolveMobSpell(mob, cs, spellData, resolveRoom)
		}
		mob.Character.CastingState = nil
		// Stage 38.3: Mob spellcasting progression
		mob.Character.OnSkillUse(string(skills.Spellcasting), 0)
		mob.Character.OnStatUse("willpower", 0)
	} else {
		mobRoom.SendText(fmt.Sprintf(
			`%s weaves magic with focused intent.`, mobDisplayName(mob, mobRoom, 0)))
	}
	return true
}

// handlePlayerFlee processes a player's flee attempt.
// Returns true if the player is fleeing and should skip combat.
func handlePlayerFlee(user *users.UserRecord, uRoom *rooms.Room, userId int) bool {
	if user.Character.Aggro.Type != characters.Flee {
		return false
	}

	// Revert to Default combat regardless of outcome
	user.Character.SetAggro(user.Character.Aggro.UserId, user.Character.Aggro.MobInstanceId, characters.DefaultAttack)

	blockedByMob := ``
	for _, mobInstId := range uRoom.GetMobs(rooms.FindFighting) {
		if mob := mobs.GetInstance(mobInstId); mob != nil {
			if mob.Character.Aggro == nil || mob.Character.Aggro.UserId != userId {
				continue
			}

			chanceIn100 := int(float64(user.Character.Stats.Dexterity.ValueAdj) / (float64(user.Character.Stats.Dexterity.ValueAdj) + float64(mob.Character.Stats.Dexterity.ValueAdj)) * 70)
			chanceIn100 += 30

			roll := util.Rand(100)
			util.LogRoll(`Flee`, roll, chanceIn100)

			if roll >= chanceIn100 {
				blockedByMob = mob.Character.Name
				break
			}
		}
	}

	blockedByPlayer := ``
	blockedByPlayerId := 0
	for _, userId := range uRoom.GetPlayers(rooms.FindFighting) {
		if u := users.GetByUserId(userId); u != nil {
			if u.Character.Aggro == nil || u.Character.Aggro.UserId != userId {
				continue
			}

			chanceIn100 := int(float64(user.Character.Stats.Dexterity.ValueAdj) / (float64(user.Character.Stats.Dexterity.ValueAdj) + float64(u.Character.Stats.Dexterity.ValueAdj)) * 70)
			chanceIn100 += 30

			roll := util.Rand(100)
			util.LogRoll(`Flee`, roll, chanceIn100)

			if roll < chanceIn100 {
				blockedByPlayer = u.Character.Name
				blockedByPlayerId = u.UserId
				break
			}
		}
	}

	if blockedByMob != `` {
		user.SendText(fmt.Sprintf(`<ansi fg="red-bold"><ansi fg="mobname">%s</ansi> blocks you from fleeing!</ansi>`, blockedByMob))
		uRoom.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is blocked from fleeing by <ansi fg="mobname">%s</ansi>!`, user.Character.Name, blockedByMob), user.UserId)
		return true
	}

	if blockedByPlayer != `` {
		user.SendText(fmt.Sprintf(`<ansi fg="red-bold"><ansi fg="username">%s</ansi> blocks you from fleeing!</ansi>`, blockedByPlayer))
		uRoom.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is blocked from fleeing by <ansi fg="username">%s</ansi>!`, user.Character.Name, blockedByPlayer), user.UserId, blockedByPlayerId)
		return true
	}

	// Success!
	exitName, exitRoomId := uRoom.GetRandomExit()

	if exitName == `` {
		user.SendText(`You can't find an exit!`)
		return true
	}

	user.SendText(fmt.Sprintf(`You flee to the <ansi fg="exit">%s</ansi> exit!`, exitName))
	uRoom.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> flees to the <ansi fg="exit">%s</ansi> exit!`, user.Character.Name, exitName), user.UserId)

	user.Character.Aggro = nil

	originRoomId := user.Character.RoomId
	if err := rooms.MoveToRoom(user.UserId, exitRoomId); err == nil {

		scripting.TryRoomScriptEvent(`onExit`, user.UserId, originRoomId)

		for _, instId := range uRoom.GetMobs(rooms.FindCharmed) {
			if mob := mobs.GetInstance(instId); mob != nil {
				if mob.Character.IsCharmed(userId) {
					mob.Command(exitName)
				}
			}
		}

		newRoom := rooms.LoadRoom(exitRoomId)

		if doLook, err := scripting.TryRoomScriptEvent(`onEnter`, user.UserId, exitRoomId); err != nil || doLook {
			usercommands.Look(``, user, newRoom, events.CmdSecretly)
		}
	}

	return true
}

// dispatchCritEffectsPvP routes crit effect messages for PvP combat.
func dispatchCritEffectsPvP(result CritEffectResult, atkUser *users.UserRecord, defUser *users.UserRecord, uRoom *rooms.Room) {
	if result.Disarmed {
		defUser.SendText(result.DisarmItem.Message)
		atkUser.SendText(result.DisarmItem.TargetMsg)
		uRoom.SendText(result.DisarmItem.RoomMessage, atkUser.UserId, defUser.UserId)
	}
	if result.GrappleSet {
		defUser.SendText(fmt.Sprintf(
			`<ansi fg="yellow">You slip inside %s's guard! [Grapple opportunity]</ansi>`,
			atkUser.Character.Name))
		uRoom.SendText(fmt.Sprintf(
			`<ansi fg="combat">%s slips inside %s's guard!</ansi>`,
			defUser.Character.Name, atkUser.Character.Name),
			atkUser.UserId, defUser.UserId)
	}
}

// dispatchCritEffectsPvM routes crit effect messages for PvM combat (player attacking mob).
func dispatchCritEffectsPvM(result CritEffectResult, atkUser *users.UserRecord, defMob *mobs.Mob, uRoom *rooms.Room) {
	if result.Disarmed {
		atkUser.SendText(result.DisarmItem.TargetMsg)
		uRoom.SendText(result.DisarmItem.RoomMessage, atkUser.UserId)
	}
	if result.GrappleSet {
		uRoom.SendText(fmt.Sprintf(
			`<ansi fg="combat"><ansi fg="mobname">%s</ansi> slips inside %s's guard!</ansi>`,
			defMob.Character.Name, atkUser.Character.Name),
			atkUser.UserId)
	}
}

// handleCharmedMobAssist triggers charmed mobs to assist their owner when attacked.
func handleCharmedMobAssist(room *rooms.Room, defId int, targetDesc string) {
	for _, instanceId := range room.GetMobs(rooms.FindCharmed) {
		if charmedMob := mobs.GetInstance(instanceId); charmedMob != nil {
			if charmedMob.Character.IsCharmed(defId) && charmedMob.Character.Aggro == nil {
				charmedMob.Character.Aggro = &characters.Aggro{
					Type: characters.DefaultAttack,
				}
				charmedMob.Command(fmt.Sprintf("attack %s", targetDesc))
			}
		}
	}
}

// handleOffhandBreakUserDef handles offhand item breakage when a player defender is hit.
func handleOffhandBreakUserDef(roundResult combat.AttackResult, defUser *users.UserRecord, defRoom *rooms.Room) {
	br := tryWeaponBreak(defUser.Character, roundResult, defRoom)
	if !br.Broke {
		return
	}

	defUser.SendText(`<ansi fg="202">***</ansi>`)
	defUser.SendText(fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> Your <ansi fg="item">%s</ansi> breaks! <ansi fg="202">***</ansi></ansi>`, br.BrokenItemName))
	defUser.SendText(`<ansi fg="202">***</ansi>`)

	defRoom.SendText(fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> The <ansi fg="item">%s</ansi> <ansi fg="username">%s</ansi> was carrying breaks! <ansi fg="202">***</ansi></ansi>`, br.BrokenItemName, defUser.Character.Name), defUser.UserId)

	events.AddToQueue(events.ItemOwnership{
		UserId: defUser.UserId,
		Item:   br.BrokenItem,
		Gained: false,
	})

	events.AddToQueue(events.ItemOwnership{
		UserId: defUser.UserId,
		Item:   br.ReplacementItem,
		Gained: true,
	})
}

// handleOffhandBreakMobDef handles offhand item breakage when a mob defender is hit.
func handleOffhandBreakMobDef(roundResult combat.AttackResult, defMob *mobs.Mob) {
	defRoom := rooms.LoadRoom(defMob.Character.RoomId)
	br := tryWeaponBreak(&defMob.Character, roundResult, defRoom)
	if !br.Broke {
		return
	}

	if defRoom != nil {
		defRoom.SendText(fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> The <ansi fg="item">%s</ansi> <ansi fg="mobname">%s</ansi> was carrying breaks! <ansi fg="202">***</ansi></ansi>`, br.BrokenItemName, defMob.Character.Name))
	}

	events.AddToQueue(events.ItemOwnership{
		MobInstanceId: defMob.InstanceId,
		Item:          br.BrokenItem,
		Gained:        false,
	})

	events.AddToQueue(events.ItemOwnership{
		MobInstanceId: defMob.InstanceId,
		Item:          br.ReplacementItem,
		Gained:        true,
	})
}

// handleAutoRetargetPlayer auto-targets a new attacker when the current target dies.
func handleAutoRetargetPlayer(user *users.UserRecord, uRoom *rooms.Room) {
	// Check for mobs attacking this player
	for _, mobInstId := range uRoom.GetMobs(rooms.FindFighting) {
		if attackingMob := mobs.GetInstance(mobInstId); attackingMob != nil {
			if attackingMob.Character.Aggro != nil && attackingMob.Character.Aggro.UserId == user.UserId {
				user.Character.SetAggro(0, attackingMob.InstanceId, characters.DefaultAttack)
				user.SendText(fmt.Sprintf("You turn your attention to <ansi fg=\"mobname\">%s</ansi>!", attackingMob.Character.Name))
				return
			}
		}
	}

	// If no mobs attacking, check for players attacking
	for _, playerId := range uRoom.GetPlayers(rooms.FindFighting) {
		if attackingPlayer := users.GetByUserId(playerId); attackingPlayer != nil {
			if attackingPlayer.Character.Aggro != nil && attackingPlayer.Character.Aggro.UserId == user.UserId {
				user.Character.SetAggro(attackingPlayer.UserId, 0, characters.DefaultAttack)
				user.SendText(fmt.Sprintf("You turn your attention to <ansi fg=\"username\">%s</ansi>!", attackingPlayer.Character.Name))
				return
			}
		}
	}
}

// handlePlayerConcentrationBreak checks if a caster's concentration breaks when hit.
func handlePlayerConcentrationBreak(defUser *users.UserRecord, roundResult combat.AttackResult, defRoom *rooms.Room) {
	if checkConcentrationBreak(defUser.Character, roundResult.DamageToTarget) {
		recordConcentrationFailure(combat.User, combat.Mob, defUser.Character, castingTargetChar(defUser.Character.CastingState))
		defUser.Character.CastingState = nil
		defUser.SendText(`<ansi fg="red">The pain shatters your concentration!</ansi>`)
		defRoom.SendText(fmt.Sprintf(
			`<ansi fg="username">%s</ansi>'s concentration breaks.`,
			defUser.Character.Name), defUser.UserId)
	}
}

// dispatchCombatMessages sends buffs and combat messages to the appropriate targets.
func dispatchCombatMessages(roundResult combat.AttackResult, atkUser *users.UserRecord, defUser *users.UserRecord, atkRoom *rooms.Room, defRoom *rooms.Room) {
	// Apply darkness message replacement for blind combatants
	srcCanSee := canSeeInRoom(atkUser.Character, atkRoom)
	tgtCanSee := canSeeInRoom(defUser.Character, defRoom)
	replaceDarknessMessages(&roundResult, srcCanSee, tgtCanSee)

	for _, buffId := range roundResult.BuffSource {
		atkUser.AddBuff(buffId, `combat`)
	}

	for _, buffId := range roundResult.BuffTarget {
		defUser.AddBuff(buffId, `combat`)
	}

	for _, msg := range roundResult.MessagesToSource {
		atkUser.SendText(msg)
	}

	for _, msg := range roundResult.MessagesToTarget {
		defUser.SendText(msg)
	}

	for _, msg := range roundResult.MessagesToSourceRoom {
		sendCombatRoomText(atkRoom, msg, atkUser.UserId, defUser.UserId)
	}

	for _, msg := range roundResult.MessagesToTargetRoom {
		sendCombatRoomText(defRoom, msg, atkUser.UserId, defUser.UserId)
	}

	// One-time sound fallback for dark rooms
	sendDarkRoomCombatFallback(atkRoom, atkUser.UserId, defUser.UserId)
	if defRoom != atkRoom {
		sendDarkRoomCombatFallback(defRoom, atkUser.UserId, defUser.UserId)
	}
}

// handleMobAIDecision processes mob AI decisions (spell casting, special moves, combat commands).
// Returns true if the mob executed an AI action and should skip normal combat.
func handleMobAIDecision(mob *mobs.Mob, c configs.Config) bool {
	if mob.Character.Aggro.Type != characters.DefaultAttack {
		return false
	}

	// Stage 11.5: Caster AI decision - try spell first, then special move
	var chosenMove string
	if util.Rand(100) < mob.ActivityLevel {
		var targetChar *characters.Character
		if mob.Character.Aggro.UserId > 0 {
			if u := users.GetByUserId(mob.Character.Aggro.UserId); u != nil {
				targetChar = u.Character
			}
		} else if mob.Character.Aggro.MobInstanceId > 0 {
			if tm := mobs.GetInstance(mob.Character.Aggro.MobInstanceId); tm != nil {
				targetChar = &tm.Character
			}
		}
		if targetChar != nil {
			chosenMove = combat.ChooseCastAction(mob)
			if chosenMove == "" {
				chosenMove = combat.ChooseSpecialMove(mob, targetChar)
			}
		}
	}

	// Execute AI-chosen move or fall back to CombatCommands
	if chosenMove != "" {
		mob.Command(chosenMove, 0)
		return true
	}

	// If they have combat commands, maybe do one of them?
	cmdCt := len(mob.CombatCommands)
	if cmdCt > 0 {
		if util.Rand(100) < mob.ActivityLevel {
			combatAction := mob.CombatCommands[util.Rand(cmdCt)]

			if combatAction == `` {
				return true
			}

			var waitTime float64 = 0.0
			allCmds := strings.Split(combatAction, `;`)
			if len(allCmds) >= c.Timing.TurnsPerRound() {
				mob.Command(`say I have a CombatAction that is too long. Please notify an admin.`)
			} else {
				for _, action := range strings.Split(combatAction, `;`) {
					mob.Command(action, waitTime)
					waitTime += 0.1
				}
			}
			return true
		}
	}

	return false
}

// handleMobTargetSwitch processes mob target switching AI.
// Returns true if the mob switched targets and should skip this round.
func handleMobTargetSwitch(mob *mobs.Mob, mobRoom *rooms.Room) bool {
	if util.Rand(100) >= 10 || mob.Character.Aggro.Type != characters.DefaultAttack {
		return false
	}

	combatSkill := mob.Character.GetCombatSkillLevel()
	if combatSkill < 30 {
		return false
	}

	potentialTargets := []int{}
	for _, userId := range mobRoom.GetPlayers() {
		if userId == mob.Character.Aggro.UserId {
			continue
		}
		if u := users.GetByUserId(userId); u != nil {
			if u.Character.Health > 0 && !u.Character.HasBuffFlag(buffs.Hidden) {
				if u.Character.Aggro != nil && u.Character.Aggro.MobInstanceId == mob.InstanceId {
					potentialTargets = append(potentialTargets, userId)
				}
			}
		}
	}

	if len(potentialTargets) == 0 {
		return false
	}

	switchChance := combat.ChanceToSwitchTarget(&mob.Character)
	roll := util.Rand(100)
	util.LogRoll("Mob Target Switch", roll, switchChance)

	if roll < switchChance {
		newTargetId := potentialTargets[util.Rand(len(potentialTargets))]
		mob.Character.SetAggro(newTargetId, 0, mob.Character.Aggro.Type, 1)

		if newTarget := users.GetByUserId(newTargetId); newTarget != nil {
			mobRoom.SendText(
				fmt.Sprintf("%s shifts focus to <ansi fg=\"username\">%s</ansi>!", mobDisplayName(mob, mobRoom, 0), newTarget.Character.Name),
			)
		}
		return true
	}

	return false
}

// handleMobWeaponPickup tries to equip a weapon from inventory when disarmed.
func handleMobWeaponPickup(mob *mobs.Mob) {
	if mob.Character.Equipment.Weapon.ItemId != 0 || len(mob.Character.Items) == 0 {
		return
	}

	roll := util.Rand(100)
	util.LogRoll(`Look for weapon`, roll, mob.Character.Stats.Charisma.ValueAdj)

	if roll < mob.Character.Stats.Charisma.ValueAdj {
		possibleWeapons := []string{}
		for _, itm := range mob.Character.Items {
			iSpec := itm.GetSpec()
			if iSpec.Type == items.Weapon {
				possibleWeapons = append(possibleWeapons, itm.DisplayName())
			}
		}

		if len(possibleWeapons) > 0 {
			mob.Command(fmt.Sprintf("equip %s", possibleWeapons[util.Rand(len(possibleWeapons))]))
		}
	}
}

// handleMobDownedGrace processes the downed grace period for a mob attacking a disabled player.
// Returns true if the player is in grace period or was finished off (combat should stop).
func handleMobDownedGrace(mob *mobs.Mob, defUser *users.UserRecord, defRoom *rooms.Room, affectedPlayerIds *[]int) bool {
	if !defUser.Character.IsDisabled() {
		return false
	}

	bal := configs.GetBalanceConfig()
	graceRounds := int(bal.CoupDeGraceRounds)
	if graceRounds <= 0 {
		mob.Character.Aggro = nil
		return true
	}
	defUser.Character.DownedRounds++
	mobName := mobDisplayName(mob, defRoom, 0)
	if defUser.Character.DownedRounds <= graceRounds {
		defRoom.SendText(fmt.Sprintf(
			`%s circles <ansi fg="username">%s</ansi>'s fallen body...`,
			mobName, defUser.Character.Name))
		return true
	}
	// Coup de grâce — finishing blow
	defUser.Character.Health = -10
	defUser.SendText(fmt.Sprintf(
		`<ansi fg="red">%s delivers a final, merciless blow!</ansi>`,
		mobName))
	defRoom.SendText(fmt.Sprintf(
		`<ansi fg="red">%s delivers a finishing blow to <ansi fg="username">%s</ansi>!</ansi>`,
		mobName, defUser.Character.Name), defUser.UserId)
	mob.Character.EndAggro()
	defUser.Character.EndAggro()
	*affectedPlayerIds = append(*affectedPlayerIds, defUser.UserId)
	return true
}

// handlePartyAutoAttack triggers auto-attack for party members when one is attacked by a mob.
// Uses the persistent per-character "autoattack" setting instead of the party-level list.
func handlePartyAutoAttack(mob *mobs.Mob, defUser *users.UserRecord) {
	if party := parties.Get(defUser.UserId); party != nil {
		for _, memberId := range party.UserIds {
			if memberId == defUser.UserId {
				continue
			}
			if memberUser := users.GetByUserId(memberId); memberUser != nil {
				if memberUser.Character.RoomId == defUser.Character.RoomId &&
					memberUser.Character.GetSetting("autoattack") != "off" &&
					memberUser.Character.Aggro == nil {
					memberUser.Command(fmt.Sprintf(`attack #%d`, mob.InstanceId))
				}
			}
		}
	}
}

// handlePlayerVsPlayer processes a player attacking another player.
func handlePlayerVsPlayer(user *users.UserRecord, uRoom *rooms.Room, evt events.NewRound, affectedPlayerIds *[]int) {
	defUser := users.GetByUserId(user.Character.Aggro.UserId)

	if uRoom == nil {
		user.Character.Aggro = nil
		return
	}

	targetFound := true
	if defUser == nil {
		targetFound = false
	} else if defUser.Character.RoomId != user.Character.RoomId {
		if user.Character.Aggro.ExitName == `` {
			targetFound = false
		} else {
			if _, exitRoomId := uRoom.FindExitByName(user.Character.Aggro.ExitName); exitRoomId != defUser.Character.RoomId {
				targetFound = false
			}
		}
	}

	if !targetFound {
		user.SendText(`Your target can't be found.`)
		user.Character.Aggro = nil
		return
	}

	defRoom := rooms.LoadRoom(defUser.Character.RoomId)
	if defRoom == nil {
		user.Character.Aggro = nil
		return
	}

	defUser.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

	if defUser.Character.Health < 1 {
		user.SendText(`Your rage subsides.`)
		user.Character.Aggro = nil
		return
	}

	if user.Character.Aggro.RoundsWaiting > 0 {
		mudlog.Debug(`RoundsWaiting`, `User`, user.Character.Name, `Rounds`, user.Character.Aggro.RoundsWaiting)
		user.Character.Aggro.RoundsWaiting--

		roundResult := combat.GetWaitMessages(items.Wait, user.Character, defUser.Character, combat.User, combat.User)

		for _, msg := range roundResult.MessagesToSource {
			user.SendText(msg)
		}
		for _, msg := range roundResult.MessagesToTarget {
			defUser.SendText(msg)
		}
		if len(roundResult.MessagesToSourceRoom) > 0 {
			for _, msg := range roundResult.MessagesToSourceRoom {
				sendCombatRoomText(uRoom, msg, user.UserId, defUser.UserId)
			}
		}
		if len(roundResult.MessagesToTargetRoom) > 0 {
			for _, msg := range roundResult.MessagesToTargetRoom {
				sendCombatRoomText(defRoom, msg, user.UserId, defUser.UserId)
			}
		}
		sendDarkRoomCombatFallback(uRoom, user.UserId, defUser.UserId)
		if defRoom != uRoom {
			sendDarkRoomCombatFallback(defRoom, user.UserId, defUser.UserId)
		}
		return
	}

	if defUser.Character.HasBuffFlag(buffs.Hidden) {
		user.SendText("You can't seem to find your target.")
		return
	}

	*affectedPlayerIds = append(*affectedPlayerIds, user.Character.Aggro.UserId)

	processGrappleProgression(user.Character, defUser.Character, user.Character.Name, defUser.Character.Name, uRoom, user.UserId, defUser.UserId)

	roundResult := combat.AttackPlayerVsPlayer(user, defUser)

	// Stage 30.1: Record combat analytics
	atkType := "unarmed"
	if user.Character.Equipment.Weapon.ItemId > 0 {
		atkType = "weapon"
	}
	combat.RecordAttack(roundResult, combat.User, combat.User, atkType, user.Character, defUser.Character, evt.RoundNumber)

	critResult := applyCritEffects(user.Character, defUser.Character, roundResult, uRoom)
	dispatchCritEffectsPvP(critResult, user, defUser, uRoom)

	// Charmed mob assist
	room := rooms.LoadRoom(user.Character.RoomId)
	handleCharmedMobAssist(room, defUser.UserId, fmt.Sprintf("@%d", user.UserId))

	dispatchCombatMessages(roundResult, user, defUser, uRoom, defRoom)

	if roundResult.Hit {
		defUser.Character.TrackPlayerDamage(user.UserId, roundResult.DamageToTarget)
	}
	handleOffhandBreakUserDef(roundResult, defUser, defRoom)

	if user.Character.Health <= 0 || defUser.Character.Health <= 0 {
		defUser.Character.EndAggro()
		user.Character.EndAggro()

		if user.Character.Health > 0 && defUser.Character.Health <= 0 {
			handleAutoRetargetPlayer(user, uRoom)
		}
	} else {
		user.Character.SetAggro(defUser.UserId, 0, characters.DefaultAttack)
	}
}

// handlePlayerVsMob processes a player attacking a mob.
func handlePlayerVsMob(user *users.UserRecord, uRoom *rooms.Room, evt events.NewRound, moonMod float64, affectedPlayerIds *[]int, affectedMobInstanceIds *[]int) {
	c := configs.GetConfig()
	roomId := user.Character.RoomId

	*affectedMobInstanceIds = append(*affectedMobInstanceIds, user.Character.Aggro.MobInstanceId)

	defMob := mobs.GetInstance(user.Character.Aggro.MobInstanceId)

	targetFound := true
	if defMob == nil {
		targetFound = false
	} else if defMob.Character.RoomId != user.Character.RoomId {
		if user.Character.Aggro.ExitName == `` {
			targetFound = false
		} else {
			if uRoom == nil {
				user.Character.Aggro = nil
				return
			}
			if _, exitRoomId := uRoom.FindExitByName(user.Character.Aggro.ExitName); exitRoomId != defMob.Character.RoomId {
				targetFound = false
			}
		}
	}

	if !targetFound {
		user.SendText("Your target can't be found.")
		user.Character.Aggro = nil
		return
	}

	defRoom := rooms.LoadRoom(defMob.Character.RoomId)
	if defRoom == nil {
		user.Character.Aggro = nil
		return
	}

	defMob.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

	if defMob.Character.Health < 1 {
		user.SendText("Your rage subsides.")
		user.Character.Aggro = nil
		return
	}

	if user.Character.Aggro.RoundsWaiting > 0 {
		mudlog.Debug(`RoundsWaiting`, `User`, user.Character.Name, `Rounds`, user.Character.Aggro.RoundsWaiting)
		user.Character.Aggro.RoundsWaiting--

		roundResult := combat.GetWaitMessages(items.Wait, user.Character, &defMob.Character, combat.User, combat.Mob)

		for _, msg := range roundResult.MessagesToSource {
			user.SendText(msg)
		}
		for _, msg := range roundResult.MessagesToSourceRoom {
			sendCombatRoomText(uRoom, msg, user.UserId)
		}
		for _, msg := range roundResult.MessagesToTargetRoom {
			sendCombatRoomText(defRoom, msg, user.UserId)
		}
		sendDarkRoomCombatFallback(uRoom, user.UserId)
		if defRoom != uRoom {
			sendDarkRoomCombatFallback(defRoom, user.UserId)
		}
		return
	}

	if defMob.Character.HasBuffFlag(buffs.Hidden) {
		user.SendText("You can't seem to find your target.")
		return
	}

	*affectedPlayerIds = append(*affectedPlayerIds, user.Character.Aggro.UserId)

	processGrappleProgression(user.Character, &defMob.Character, user.Character.Name, defMob.Character.Name, uRoom, user.UserId, 0)

	var roundResult combat.AttackResult

	// Stage 17.2: Moon phase stat modifiers
	restore := applyMoonMods(user.Character, moonMod)
	roundResult = combat.AttackPlayerVsMob(user, defMob)
	restore()

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

	// Stage 30.1: Record combat analytics
	pvmAtkType := "unarmed"
	if user.Character.Equipment.Weapon.ItemId > 0 {
		pvmAtkType = "weapon"
	}
	combat.RecordAttack(roundResult, combat.User, combat.Mob, pvmAtkType, user.Character, &defMob.Character, evt.RoundNumber)

	pvmCritResult := applyCritEffects(user.Character, &defMob.Character, roundResult, uRoom)
	dispatchCritEffectsPvM(pvmCritResult, user, defMob, uRoom)

	// Apply darkness message replacement for blind attacker
	pvmSrcCanSee := canSeeInRoom(user.Character, uRoom)
	replaceDarknessMessages(&roundResult, pvmSrcCanSee, true) // mobs don't receive text messages

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
		sendCombatRoomText(uRoom, msg, user.UserId)
	}
	for _, msg := range roundResult.MessagesToTargetRoom {
		sendCombatRoomText(defRoom, msg, user.UserId)
	}
	sendDarkRoomCombatFallback(uRoom, user.UserId)
	if defRoom != uRoom {
		sendDarkRoomCombatFallback(defRoom, user.UserId)
	}

	// Stage 11.5: Mob concentration break when hit
	if checkConcentrationBreak(&defMob.Character, roundResult.DamageToTarget) {
		recordConcentrationFailure(combat.Mob, combat.User, &defMob.Character, castingTargetChar(defMob.Character.CastingState))
		defMob.Character.CastingState = nil
		uRoom.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s concentration breaks.`, defMob.Character.Name))
	}

	// Handle any scripted behavior now.
	if roundResult.Hit {
		scripting.TryMobScriptEvent(`onHurt`, defMob.InstanceId, user.UserId, `user`, map[string]any{`damage`: roundResult.DamageToTarget, `crit`: roundResult.Crit})
	}

	// Hostility
	for _, groupName := range defMob.Groups {
		mobs.MakeHostile(groupName, user.UserId, c.Timing.MinutesToRounds(2)-user.Character.Stats.Charisma.ValueAdj)
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

	if user.Character.Health <= 0 || defMob.Character.Health <= 0 {
		defMob.Character.EndAggro()
		user.Character.EndAggro()

		if user.Character.Health > 0 && defMob.Character.Health <= 0 {
			handleAutoRetargetPlayer(user, uRoom)
		}
	} else {
		user.Character.SetAggro(0, defMob.InstanceId, characters.DefaultAttack)
	}

	_ = roomId // used implicitly via uRoom
}

// handleMobVsPlayer processes a mob attacking a player.
func handleMobVsPlayer(mob *mobs.Mob, mobRoom *rooms.Room, evt events.NewRound, moonMod float64, affectedPlayerIds *[]int) {
	defUser := users.GetByUserId(mob.Character.Aggro.UserId)
	if defUser == nil || mob.Character.RoomId != defUser.Character.RoomId {
		mob.Character.Aggro = nil
		return
	}

	defRoom := rooms.LoadRoom(defUser.Character.RoomId)
	if defRoom == nil {
		mob.Character.Aggro = nil
		return
	}

	defUser.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

	// Downed grace period
	if handleMobDownedGrace(mob, defUser, defRoom, affectedPlayerIds) {
		return
	}

	if defUser.Character.HasBuffFlag(buffs.Hidden) {
		return
	}

	*affectedPlayerIds = append(*affectedPlayerIds, mob.Character.Aggro.UserId)

	// Reciprocal aggro — skip dead/downed players to prevent stale aggro in Shadow Realm
	if defUser.Character.Health > 0 && defUser.Character.Aggro == nil {
		defUser.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
	}

	handlePartyAutoAttack(mob, defUser)

	processGrappleProgression(&mob.Character, defUser.Character, mob.Character.Name, defUser.Character.Name, mobRoom, 0, defUser.UserId)

	// Target switch AI
	if handleMobTargetSwitch(mob, mobRoom) {
		return
	}

	handleMobWeaponPickup(mob)

	if mob.Character.Aggro.RoundsWaiting > 0 {
		mudlog.Debug(`RoundsWaiting`, `User`, mob.Character.Name, `Rounds`, mob.Character.Aggro.RoundsWaiting)
		mob.Character.Aggro.RoundsWaiting--

		roundResult := combat.GetWaitMessages(items.Wait, &mob.Character, defUser.Character, combat.Mob, combat.User)

		for _, msg := range roundResult.MessagesToTarget {
			defUser.SendText(msg)
		}
		for _, msg := range roundResult.MessagesToSourceRoom {
			sendCombatRoomText(mobRoom, msg, defUser.UserId)
		}
		for _, msg := range roundResult.MessagesToTargetRoom {
			sendCombatRoomText(defRoom, msg, defUser.UserId)
		}
		sendDarkRoomCombatFallback(mobRoom, defUser.UserId)
		if defRoom != mobRoom {
			sendDarkRoomCombatFallback(defRoom, defUser.UserId)
		}
		return
	}

	var roundResult combat.AttackResult

	// Stage 17.2: Moon phase stat modifiers (apply to mob attacker, symmetric with PvM)
	restore := applyMoonMods(&mob.Character, moonMod)
	roundResult = combat.AttackMobVsPlayer(mob, defUser)
	restore()

	// Stage 11.4: Minor Shield reduces physical weapon damage
	if roundResult.Hit && defUser.Character.HasCondition(characters.ConditionShield) {
		reduction := int(defUser.Character.GetConditionMagnitude(characters.ConditionShield)) / 2
		if roundResult.DamageToTarget > reduction+1 {
			roundResult.DamageToTarget -= reduction
			roundResult.DamageToTargetReduction += reduction
		}
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

	// Stage 30.1: Record combat analytics
	mvpAtkType := "unarmed"
	if mob.Character.Equipment.Weapon.ItemId > 0 {
		mvpAtkType = "weapon"
	}
	combat.RecordAttack(roundResult, combat.Mob, combat.User, mvpAtkType, &mob.Character, defUser.Character, evt.RoundNumber)

	// Crit effects (player defending)
	mvpCritResult := applyCritEffects(&mob.Character, defUser.Character, roundResult, mobRoom)
	if mvpCritResult.Disarmed {
		defUser.SendText(mvpCritResult.DisarmItem.Message)
		mobRoom.SendText(mvpCritResult.DisarmItem.RoomMessage, defUser.UserId)
	}
	if mvpCritResult.GrappleSet {
		mvpMobName := mobDisplayName(mob, mobRoom, defUser.UserId)
		defUser.SendText(fmt.Sprintf(
			`<ansi fg="yellow">You slip inside %s's guard! [Grapple opportunity]</ansi>`,
			mob.Character.Name))
		mobRoom.SendText(fmt.Sprintf(
			`<ansi fg="combat">%s slips inside %s's guard!</ansi>`,
			defUser.Character.Name, mvpMobName),
			defUser.UserId)
	}

	// Charmed mob assist
	roomId := mob.Character.RoomId
	room := rooms.LoadRoom(roomId)
	handleCharmedMobAssist(room, defUser.UserId, fmt.Sprintf("#%d", mob.InstanceId))

	// Apply darkness message replacement for blind defender
	mvpTgtCanSee := canSeeInRoom(defUser.Character, defRoom)
	replaceDarknessMessages(&roundResult, true, mvpTgtCanSee) // mob source doesn't need text replacement

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
		sendCombatRoomText(mobRoom, msg, defUser.UserId)
	}
	for _, msg := range roundResult.MessagesToTargetRoom {
		sendCombatRoomText(defRoom, msg, defUser.UserId)
	}
	sendDarkRoomCombatFallback(mobRoom, defUser.UserId)
	if defRoom != mobRoom {
		sendDarkRoomCombatFallback(defRoom, defUser.UserId)
	}

	handlePlayerConcentrationBreak(defUser, roundResult, defRoom)
	handleOffhandBreakUserDef(roundResult, defUser, defRoom)

	// Stage 38.3: Mob attacker progression
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
	if roundResult.Hit {
		combatSkill := string(mob.Character.GetCombatSkillTag())
		mob.Character.OnSkillUse(combatSkill, 0)
		if roundResult.Crit {
			mob.Character.OnCriticalSuccess(combatSkill, 0)
		}
	} else if roundResult.Fumble {
		combatSkill := string(mob.Character.GetCombatSkillTag())
		mob.Character.OnCriticalFailure(combatSkill, 0)
	}

	if mob.Character.Health <= 0 || defUser.Character.Health <= 0 {
		mob.Character.EndAggro()
		defUser.Character.EndAggro()
	} else {
		mob.Character.SetAggro(defUser.UserId, 0, characters.DefaultAttack)
	}
}

// handleMobVsMob processes a mob attacking another mob.
func handleMobVsMob(mob *mobs.Mob, mobRoom *rooms.Room, evt events.NewRound, affectedMobInstanceIds *[]int) {
	*affectedMobInstanceIds = append(*affectedMobInstanceIds, mob.Character.Aggro.MobInstanceId)

	defMob := mobs.GetInstance(mob.Character.Aggro.MobInstanceId)

	if defMob == nil || mob.Character.RoomId != defMob.Character.RoomId {
		mob.Character.Aggro = nil
		return
	}

	defRoom := rooms.LoadRoom(defMob.Character.RoomId)
	if defRoom == nil {
		mob.Character.Aggro = nil
		return
	}

	defMob.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

	if defMob.Character.Health < 1 {
		mob.Character.Aggro = nil
		return
	}

	if mob.Character.Aggro.RoundsWaiting > 0 {
		mudlog.Debug(`RoundsWaiting`, `User`, mob.Character.Name, `Rounds`, mob.Character.Aggro.RoundsWaiting)
		mob.Character.Aggro.RoundsWaiting--

		roundResult := combat.GetWaitMessages(items.Wait, &mob.Character, &defMob.Character, combat.Mob, combat.Mob)

		for _, msg := range roundResult.MessagesToSourceRoom {
			sendCombatRoomText(mobRoom, msg)
		}
		for _, msg := range roundResult.MessagesToTargetRoom {
			sendCombatRoomText(defRoom, msg)
		}
		sendDarkRoomCombatFallback(mobRoom)
		if defRoom != mobRoom {
			sendDarkRoomCombatFallback(defRoom)
		}
		return
	}

	if defMob.Character.HasBuffFlag(buffs.Hidden) {
		return
	}

	processGrappleProgression(&mob.Character, &defMob.Character, mob.Character.Name, defMob.Character.Name, mobRoom, 0, 0)

	var roundResult combat.AttackResult
	roundResult = combat.AttackMobVsMob(mob, defMob)

	// Stage 12.2: Adrenaline Surge
	if roundResult.Hit && roundResult.DamageToTarget > 0 {
		if mutations.IsAdrenalSurgeActive(mob.Character.Mutations, mob.Character.Health, mob.Character.HealthMax.Value) {
			if surgeBonus := mutations.GetAdrenalSurgeBonus(mob.Character.Mutations); surgeBonus > 0 {
				bonusDmg := int(math.Round(float64(roundResult.DamageToTarget) * surgeBonus))
				if bonusDmg < 1 {
					bonusDmg = 1
				}
				defMob.Character.Health -= bonusDmg
				roundResult.DamageToTarget += bonusDmg
			}
		}
	}

	// Stage 30.1: Record combat analytics
	mmAtkType := "unarmed"
	if mob.Character.Equipment.Weapon.ItemId > 0 {
		mmAtkType = "weapon"
	}
	combat.RecordAttack(roundResult, combat.Mob, combat.Mob, mmAtkType, &mob.Character, &defMob.Character, evt.RoundNumber)

	for _, buffId := range roundResult.BuffSource {
		mob.AddBuff(buffId, `combat`)
	}
	for _, buffId := range roundResult.BuffTarget {
		defMob.AddBuff(buffId, `combat`)
	}
	for _, msg := range roundResult.MessagesToSourceRoom {
		sendCombatRoomText(mobRoom, msg)
	}
	for _, msg := range roundResult.MessagesToTargetRoom {
		sendCombatRoomText(defRoom, msg)
	}
	sendDarkRoomCombatFallback(mobRoom)
	if defRoom != mobRoom {
		sendDarkRoomCombatFallback(defRoom)
	}

	// Handle any scripted behavior now.
	if roundResult.Hit {
		scripting.TryMobScriptEvent(`onHurt`, defMob.InstanceId, mob.InstanceId, `mob`, map[string]any{`damage`: roundResult.DamageToTarget, `crit`: roundResult.Crit})
	}

	// Mobs get aggro when attacked
	if defMob.Character.Aggro == nil {
		defMob.PreventIdle = true
		defMob.Character.Aggro = &characters.Aggro{
			Type: characters.DefaultAttack,
		}
		defMob.Command(fmt.Sprintf("attack #%d", mob.InstanceId))
	}

	handleOffhandBreakMobDef(roundResult, defMob)

	// Stage 38.3: Mob attacker progression (skip room messages for MvM)
	mob.Character.OnStatUse("strength", 0)
	mob.Character.OnStatUse("dexterity", 0)
	if roundResult.Hit {
		combatSkill := string(mob.Character.GetCombatSkillTag())
		mob.Character.OnSkillUse(combatSkill, 0)
	}

	if mob.Character.Health <= 0 || defMob.Character.Health <= 0 {
		mob.Character.EndAggro()
		defMob.Character.EndAggro()
	} else {
		mob.Character.SetAggro(0, defMob.InstanceId, characters.DefaultAttack)
	}
}
