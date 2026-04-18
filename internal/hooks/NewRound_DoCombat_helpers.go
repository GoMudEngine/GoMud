package hooks

import (
	"fmt"
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobai"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/textutil"
	"github.com/GoMudEngine/GoMud/internal/usercommands"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// processDefenderProgression fires skill and stat progression for a defender
// based on which defense types were used across all swings in the round.
// Dodge → unarmed-combat + dexterity, Parry → weapon-combat + dexterity + strength,
// Block → weapon-combat + strength.
func processDefenderProgression(c *characters.Character, userId int, result combat.AttackResult) {
	dodged, parried, blocked := false, false, false
	for _, se := range result.SwingEvents {
		switch se.DefenseUsed {
		case combat.DefenseDodge:
			dodged = true
		case combat.DefenseParry:
			parried = true
		case combat.DefenseBlock:
			blocked = true
		}
	}

	if dodged {
		c.OnSkillUse(string(skills.UnarmedCombat), userId)
		c.OnStatUse("dexterity", userId)
	}
	if parried {
		c.OnSkillUse(string(skills.WeaponCombat), userId)
		c.OnStatUse("dexterity", userId)
		c.OnStatUse("strength", userId)
	}
	if blocked {
		c.OnSkillUse(string(skills.WeaponCombat), userId)
		c.OnStatUse("strength", userId)
	}
}

// mobDisplayName returns the formatted display name for a mob in combat text,
// including duplicate index coloring when multiple mobs share the same name.
func mobDisplayName(mob *mobs.Mob, room *rooms.Room, viewingUserId int) string {
	dupIdx := room.GetMobDuplicateIndex(mob.InstanceId)
	return mob.Character.GetMobNameIndexed(viewingUserId, dupIdx).String()
}

// sendVisualRoomText sends a visual message that requires sight.
// Delegates to Room.SendTextVisual which handles darkness filtering.
func sendVisualRoomText(room *rooms.Room, visualMsg string, excludeUserIds ...int) {
	if room == nil {
		return
	}
	room.SendTextVisual(visualMsg, excludeUserIds...)
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

	// Bleeding out = automatic concentration break (player-only check).
	if user.Character.IsDisabled() {
		recordConcentrationFailure(combat.User, combat.Mob, user.Character, castingTargetChar(user.Character.CastingState))
		user.Character.CastingState = nil
		return true
	}

	// Capture state before processFoldRound clears it on terminal conditions.
	csBeforeProcess := user.Character.CastingState

	result := processFoldRound(user.Character)

	switch {
	case result.ProneBroke:
		recordConcentrationFailure(combat.User, combat.Mob, user.Character, castingTargetChar(csBeforeProcess))
		user.SendText(`<ansi fg="red">You lose your concentration as you hit the ground!</ansi>`)
		room := rooms.LoadRoom(user.Character.RoomId)
		if room != nil {
			sendVisualRoomText(room, fmt.Sprintf(
				`<ansi fg="username">%s</ansi>'s concentration breaks.`, user.Character.Name), user.UserId)
		}

	case result.TargetGone:
		recordConcentrationFailure(combat.User, combat.Mob, user.Character, castingTargetChar(csBeforeProcess))
		user.SendText(`<ansi fg="red">Your spell fizzles — the target is gone.</ansi>`)

	case result.SpellDataMissing:
		user.SendText(`<ansi fg="red">The spell dissipates — its data cannot be found.</ansi>`)

	case result.InsufficientConviction:
		recordConcentrationFailure(combat.User, combat.Mob, user.Character, castingTargetChar(csBeforeProcess))
		user.SendText(`<ansi fg="red">Your conviction wavers — the fold collapses.</ansi>`)

	case result.CastComplete:
		cs := result.CastingState
		spellData := result.SpellData
		// Send YAML wait text (if defined).
		if spellData != nil && (spellData.WaitUserText != "" || spellData.WaitRoomText != "") {
			tCtx := textutil.TokenContext{
				SourceName:      user.Character.GetCharacterName(true),
				SourcePlainName: user.Character.GetCharacterName(false),
			}
			cfg := textutil.SendTextConfig{
				UserSendFunc: func(msg string) { user.SendText(msg) },
				RoomSendFunc: func(msg string, skip ...int) {
					if r := rooms.LoadRoom(user.Character.RoomId); r != nil {
						r.SendText(msg, skip...)
					}
				},
				ExcludeId: user.UserId,
			}
			textutil.SendPhaseText(spellData.WaitUserText, spellData.WaitRoomText, tCtx, "pink", cfg)
		}

		resolveRoom := rooms.LoadRoom(user.Character.RoomId)
		if resolveRoom != nil {
			resolveSpell(user, cs, spellData, resolveRoom)
		}
		user.Character.TrackSpellCast(cs.SpellId)
		// Fire progression for the correct skill based on spell school.
		// Difficulty scaling: harder spells give proportionally more progression.
		spellBonus := 1.0
		if spellData != nil {
			bal := configs.GetBalanceConfig()
			spellBonus = 1.0 + float64(spellData.Difficulty)*float64(bal.SpellDifficultyProgressionScale)

			// Self-cast penalty: HelpSingle targeting only self gets reduced progression
			if spellData.Type == spells.HelpSingle &&
				len(cs.TargetMobInstanceIds) == 0 &&
				len(cs.TargetUserIds) == 1 && cs.TargetUserIds[0] == userId {
				spellBonus *= float64(bal.SelfCastProgressionMultiplier)
			}

			// AoE guard: HarmArea/HarmMulti with no targets hit skips progression
			if (spellData.Type == spells.HarmArea || spellData.Type == spells.HarmMulti) &&
				len(cs.TargetUserIds) == 0 && len(cs.TargetMobInstanceIds) == 0 {
				spellBonus = 0
			}
		}

		if spellBonus > 0 {
			if spellData != nil && spellData.HasSchool(spells.SchoolManifestation) {
				user.Character.OnSkillUseScaled(string(skills.Manifestation), userId, spellBonus)
				user.Character.OnStatUse("charisma", userId)
			} else {
				user.Character.OnSkillUseScaled(string(skills.Spellcasting), userId, spellBonus)
				user.Character.OnStatUse("willpower", userId)
			}
		}

		// Phase 25.1: Spell discovery — traditional schools.
		castSkillLevel := user.Character.GetSkillLevel(skills.Spellcasting)
		knownCount := len(user.Character.SpellBook)
		bal := configs.GetBalanceConfig()
		discoveryChance := float64(bal.SpellDiscoveryBaseChance) / (1.0 + float64(knownCount)*float64(bal.SpellDiscoveryDecayRate))
		if util.Rand(100) < int(discoveryChance) {
			eligible := spells.GetEligibleSpells(user.Character.SpellBook, castSkillLevel,
				spells.SchoolElemental, spells.SchoolEnhancement, spells.SchoolMental, spells.SchoolVital)
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
		// Phase 25.1: Spell discovery — manifestation school.
		// Only runs if the player has any manifestation skill.
		manifestSkillLevel := user.Character.GetSkillLevel(skills.Manifestation)
		if manifestSkillLevel > 0 {
			if util.Rand(100) < int(discoveryChance) {
				eligible := spells.GetEligibleSpells(user.Character.SpellBook, manifestSkillLevel,
					spells.SchoolManifestation)
				if len(eligible) > 0 {
					pick := eligible[util.Rand(len(eligible))]
					if user.Character.LearnSpell(pick) {
						if newSpell := spells.GetSpell(pick); newSpell != nil {
							user.SendText(fmt.Sprintf(
								`<ansi fg="magenta-bold">A manifestation reveals itself: <ansi fg="cyan-bold">%s</ansi></ansi>`,
								newSpell.Name))
						}
					}
				}
			}
		}

	case result.StillCasting:
		cs := result.CastingState
		// Send YAML wait text (if defined).
		waitSpellInfo := spells.GetSpell(cs.SpellId)
		if waitSpellInfo != nil && (waitSpellInfo.WaitUserText != "" || waitSpellInfo.WaitRoomText != "") {
			tCtx := textutil.TokenContext{
				SourceName:      user.Character.GetCharacterName(true),
				SourcePlainName: user.Character.GetCharacterName(false),
			}
			cfg := textutil.SendTextConfig{
				UserSendFunc: func(msg string) { user.SendText(msg) },
				RoomSendFunc: func(msg string, skip ...int) {
					if r := rooms.LoadRoom(user.Character.RoomId); r != nil {
						r.SendText(msg, skip...)
					}
				},
				ExcludeId: user.UserId,
			}
			textutil.SendPhaseText(waitSpellInfo.WaitUserText, waitSpellInfo.WaitRoomText, tCtx, "pink", cfg)
		}
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

	// Capture state before processFoldRound clears it on terminal conditions.
	csBeforeProcess := mob.Character.CastingState

	result := processFoldRound(&mob.Character)

	switch {
	case result.ProneBroke:
		recordConcentrationFailure(combat.Mob, combat.User, &mob.Character, castingTargetChar(csBeforeProcess))
		mobRoom.SendText(fmt.Sprintf(
			`%s's concentration breaks.`, mobDisplayName(mob, mobRoom, 0)))

	case result.TargetGone:
		recordConcentrationFailure(combat.Mob, combat.User, &mob.Character, castingTargetChar(csBeforeProcess))
		mobRoom.SendText(fmt.Sprintf(
			`%s's spell fizzles.`, mobDisplayName(mob, mobRoom, 0)))

	case result.SpellDataMissing:
		// Silent failure — no message for missing spell data on mobs.

	case result.InsufficientConviction:
		recordConcentrationFailure(combat.Mob, combat.User, &mob.Character, castingTargetChar(csBeforeProcess))
		mobRoom.SendText(fmt.Sprintf(
			`%s's spell falters.`, mobDisplayName(mob, mobRoom, 0)))

	case result.CastComplete:
		cs := result.CastingState
		spellData := result.SpellData
		if resolveRoom := rooms.LoadRoom(mob.Character.RoomId); resolveRoom != nil {
			resolveMobSpell(mob, cs, spellData, resolveRoom)
		}
		// Stage 38.3: Mob spellcasting progression — difficulty-scaled
		spellBonus := 1.0
		if spellData != nil {
			bal := configs.GetBalanceConfig()
			spellBonus = 1.0 + float64(spellData.Difficulty)*float64(bal.SpellDifficultyProgressionScale)
		}
		if spellData != nil && spellData.HasSchool(spells.SchoolManifestation) {
			mob.Character.OnSkillUseScaled(string(skills.Manifestation), 0, spellBonus)
			mob.Character.OnStatUse("charisma", 0)
		} else {
			mob.Character.OnSkillUseScaled(string(skills.Spellcasting), 0, spellBonus)
			mob.Character.OnStatUse("willpower", 0)
		}

		// Task 6: Spell discovery for caster mobs.
		// Only mobs that started with spells or have archetype="casting" can discover new ones.
		isCaster := mob.Archetype == "casting" || len(mob.Character.SpellBook) > 0
		if isCaster {
			castSkillLevel := mob.Character.GetSkillLevel(skills.Spellcasting)
			knownCount := len(mob.Character.SpellBook)
			bal := configs.GetBalanceConfig()
			discoveryChance := float64(bal.SpellDiscoveryBaseChance) / (1.0 + float64(knownCount)*float64(bal.SpellDiscoveryDecayRate))
			// Traditional school discovery.
			if util.Rand(100) < int(discoveryChance) {
				eligible := spells.GetEligibleSpells(mob.Character.SpellBook, castSkillLevel,
					spells.SchoolElemental, spells.SchoolEnhancement, spells.SchoolMental, spells.SchoolVital)
				if len(eligible) > 0 {
					pick := eligible[util.Rand(len(eligible))]
					mob.Character.LearnSpell(pick)
				}
			}
			// Manifestation school discovery — only if mob has manifestation skill.
			manifestSkillLevel := mob.Character.GetSkillLevel(skills.Manifestation)
			if manifestSkillLevel > 0 {
				if util.Rand(100) < int(discoveryChance) {
					eligible := spells.GetEligibleSpells(mob.Character.SpellBook, manifestSkillLevel,
						spells.SchoolManifestation)
					if len(eligible) > 0 {
						pick := eligible[util.Rand(len(eligible))]
						mob.Character.LearnSpell(pick)
					}
				}
			}
		}

	case result.StillCasting:
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

	// Can't flee while in a grapple position (clinched or grounded)
	if user.Character.CombatPosition == characters.PositionClinched ||
		user.Character.CombatPosition == characters.PositionGrounded {
		user.SendText(`<ansi fg="red">You can't flee while grappled!</ansi>`)
		return true
	}

	blockedByMob := ``
	for _, mobInstId := range uRoom.GetMobs(rooms.FindFighting) {
		if mob := mobs.GetInstance(mobInstId); mob != nil {
			if mob.Character.Aggro == nil || mob.Character.Aggro.UserId != userId {
				continue
			}

			// Flee: Dex + Skullduggery vs blocker's Dex + UnarmedCombat
			fleeScore := float64(user.Character.Stats.Dexterity.ValueAdj +
				user.Character.GetSkillLevel(skills.Skullduggery)*25)

			// Prone penalty — halve flee score when knocked down
			if user.Character.CombatPosition == characters.PositionProne {
				fleeScore *= 0.5
			}
			blockScore := float64(mob.Character.Stats.Dexterity.ValueAdj +
				mob.Character.GetSkillLevel(skills.UnarmedCombat)*25)
			success, _, _, _ := dice.OpposedRollStat(fleeScore, blockScore)
			if !success {
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

			// Flee: Dex + Skullduggery vs blocker's Dex + UnarmedCombat
			fleeScore := float64(user.Character.Stats.Dexterity.ValueAdj +
				user.Character.GetSkillLevel(skills.Skullduggery)*25)
			blockScore := float64(u.Character.Stats.Dexterity.ValueAdj +
				u.Character.GetSkillLevel(skills.UnarmedCombat)*25)
			success, _, _, _ := dice.OpposedRollStat(fleeScore, blockScore)
			if !success {
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

	user.Character.EndAggro()

	if err := rooms.MoveToRoom(user.UserId, exitRoomId); err == nil {

		for _, instId := range uRoom.GetMobs(rooms.FindCharmed) {
			if mob := mobs.GetInstance(instId); mob != nil {
				if mob.Character.IsCharmed(userId) {
					mob.Command(exitName)
				}
			}
		}

		newRoom := rooms.LoadRoom(exitRoomId)
		usercommands.Look(``, user, newRoom, events.CmdSecretly)
	}

	return true
}

// dispatchCritEffectsPvP routes crit effect messages for PvP combat.
func dispatchCritEffectsPvP(result CritEffectResult, atkUser *users.UserRecord, defUser *users.UserRecord, uRoom *rooms.Room) {
	if result.DefenderMsg != "" {
		defUser.SendText(result.DefenderMsg)
	}
	if result.AttackerMsg != "" {
		atkUser.SendText(result.AttackerMsg)
	}
	if result.RoomMsg != "" {
		uRoom.SendText(result.RoomMsg, atkUser.UserId, defUser.UserId)
	}
}

// dispatchCritEffectsPvM routes crit effect messages for PvM combat (player attacking mob).
func dispatchCritEffectsPvM(result CritEffectResult, atkUser *users.UserRecord, defMob *mobs.Mob, uRoom *rooms.Room) {
	if result.AttackerMsg != "" {
		atkUser.SendText(result.AttackerMsg)
	}
	if result.RoomMsg != "" {
		uRoom.SendText(result.RoomMsg, atkUser.UserId)
	}
}

// handleCompanionOwnerAssist triggers a companion's owner (and the owner's other
// companions) to fight back when the companion is attacked.
// attackerDesc is the attack-command argument that identifies the attacker
// (e.g. "#42" for a mob instance or "@7" for a player).
func handleCompanionOwnerAssist(defMob *mobs.Mob, attackerDesc string) {
	ownerId := defMob.Character.GetCharmedUserId()
	if ownerId == 0 {
		return
	}
	owner := users.GetByUserId(ownerId)
	if owner == nil {
		return
	}

	// Find the companion entry to check AutoAssist.
	comp := owner.Character.GetCompanionByInstanceId(defMob.InstanceId)
	if comp == nil || !comp.AutoAssist {
		return
	}

	// Owner fights back if not already in combat.
	if owner.Character.Aggro == nil {
		owner.Command(fmt.Sprintf("attack %s", attackerDesc))
	}

	// Other companions of the same owner also assist.
	ownerRoom := rooms.LoadRoom(owner.Character.RoomId)
	if ownerRoom != nil {
		handleCharmedMobAssist(ownerRoom, ownerId, attackerDesc)
	}
}

// handleCharmedMobAssist triggers charmed mobs to assist their owner when attacked.
// Only engages companions that have AutoAssist enabled.
func handleCharmedMobAssist(room *rooms.Room, defId int, targetDesc string) {
	defUser := users.GetByUserId(defId)
	if defUser == nil {
		return
	}
	for _, instanceId := range room.GetMobs(rooms.FindCharmed) {
		if charmedMob := mobs.GetInstance(instanceId); charmedMob != nil {
			if charmedMob.Character.IsCharmed(defId) && charmedMob.Character.Aggro == nil {
				comp := defUser.Character.GetCompanionByInstanceId(instanceId)
				if comp != nil && comp.AutoAssist {
					charmedMob.Command(fmt.Sprintf("attack %s", targetDesc))
				}
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
		sendVisualRoomText(atkRoom, msg, atkUser.UserId, defUser.UserId)
	}

	for _, msg := range roundResult.MessagesToTargetRoom {
		sendVisualRoomText(defRoom, msg, atkUser.UserId, defUser.UserId)
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

	// If mob has reactive AI tactics and recently reacted, skip legacy AI
	resolvedTactics := mobai.ResolveTactics(mob.TacticPreset, mob.Tactics)
	if len(resolvedTactics) > 0 {
		bal := configs.GetBalanceConfig()
		delay := mobai.GetEffectiveReactionDelay(
			mob.ReactionDelay,
			float64(bal.MobReactionDelayMin),
			float64(bal.MobReactionDelayMax),
		)
		cooldownTurns := uint64(delay * float64(configs.GetTimingConfig().TurnsPerSecond()))
		lastReaction := mob.GetLastReactionTurn()
		currentTurn := util.GetTurnCount()
		if lastReaction > 0 && currentTurn-lastReaction < cooldownTurns*2 {
			mudlog.Debug("MobAI", "decision", "skip_legacy", "mob", mob.Character.Name, "lastReaction", lastReaction, "current", currentTurn)
			return true // Reactive AI is handling this mob
		}
		mudlog.Debug("MobAI", "decision", "fallthrough_to_legacy", "mob", mob.Character.Name, "tactics", len(resolvedTactics), "lastReaction", lastReaction)
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
		user.Character.EndAggro()
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
		user.Character.EndAggro()
		return
	}

	defRoom := rooms.LoadRoom(defUser.Character.RoomId)
	if defRoom == nil {
		user.Character.EndAggro()
		return
	}

	defUser.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

	if defUser.Character.Health < 1 {
		user.Character.EndAggro()
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
				sendVisualRoomText(uRoom, msg, user.UserId, defUser.UserId)
			}
		}
		if len(roundResult.MessagesToTargetRoom) > 0 {
			for _, msg := range roundResult.MessagesToTargetRoom {
				sendVisualRoomText(defRoom, msg, user.UserId, defUser.UserId)
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

	// Conviction Surge buff: +15% damage bonus
	if roundResult.Hit && roundResult.DamageToTarget > 0 && user.Character.HasBuffFlag(buffs.DamageBonus) {
		bonusDmg := int(math.Round(float64(roundResult.DamageToTarget) * 0.15))
		if bonusDmg < 1 {
			bonusDmg = 1
		}
		defUser.Character.Health -= bonusDmg
		roundResult.DamageToTarget += bonusDmg
	}

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
		// Physical crit received → vitality progression for defender
		if roundResult.Crit {
			defUser.Character.OnCritReceived("physical", defUser.UserId)
		}
	}
	handleOffhandBreakUserDef(roundResult, defUser, defRoom)

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
	if len(roundResult.WeaponHits) == 0 && roundResult.Hit {
		user.Character.OnSkillUse(string(skills.UnarmedCombat), user.UserId)
	}

	// Defender progression — dodge/parry/block train specific skills
	processDefenderProgression(defUser.Character, defUser.UserId, roundResult)

	if user.Character.Health <= 0 || defUser.Character.Health <= 0 {
		defUser.Character.EndAggro()
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
		user.Character.SetAggro(defUser.UserId, 0, characters.DefaultAttack)
	}
}

// handlePlayerVsMob processes a player attacking a mob.
func handlePlayerVsMob(user *users.UserRecord, uRoom *rooms.Room, evt events.NewRound, moonMod float64, affectedPlayerIds *[]int, affectedMobInstanceIds *[]int) {
	c := configs.GetConfig()

	*affectedMobInstanceIds = append(*affectedMobInstanceIds, user.Character.Aggro.MobInstanceId)

	defMob, defRoom, ok := resolvePlayerVsMobTarget(user, uRoom, evt)
	if !ok {
		return
	}

	if handleCombatWaitRound(
		user.Character, &defMob.Character,
		combat.User, combat.Mob,
		user, nil,
		uRoom, defRoom,
		user.UserId,
	) {
		return
	}

	if defMob.Character.HasBuffFlag(buffs.Hidden) {
		user.SendText("You can't seem to find your target.")
		return
	}

	*affectedPlayerIds = append(*affectedPlayerIds, user.Character.Aggro.UserId)

	processGrappleProgression(user.Character, &defMob.Character, user.Character.Name, defMob.Character.Name, uRoom, user.UserId, 0)

	// Stage 17.2: Moon phase stat modifiers
	restore := applyMoonMods(user.Character, moonMod)
	roundResult := combat.AttackPlayerVsMob(user, defMob)
	restore()

	applyCombatDamageBonuses_PvM(&roundResult, user, defMob, uRoom, defRoom)
	handlePvMCritAndMessaging(&roundResult, user, defMob, uRoom, defRoom, evt.RoundNumber)
	handlePvMProgressionAndAggro(&roundResult, user, defMob, uRoom, &c)
	handlePvMRoundResolution(user, defMob, uRoom)
}

// handleMobVsPlayer processes a mob attacking a player.
func handleMobVsPlayer(mob *mobs.Mob, mobRoom *rooms.Room, evt events.NewRound, moonMod float64, affectedPlayerIds *[]int) {
	defUser := users.GetByUserId(mob.Character.Aggro.UserId)
	if defUser == nil || mob.Character.RoomId != defUser.Character.RoomId {
		mob.Character.EndAggro()
		return
	}

	defRoom := rooms.LoadRoom(defUser.Character.RoomId)
	if defRoom == nil {
		mob.Character.EndAggro()
		return
	}

	defUser.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

	// Defender already dead (Health <= 0): end mob aggro, skip the round.
	// Under the post-bleedout-removal rule there's no "downed but alive"
	// state — IsDisabled() now implies the player is dead and will be
	// processed by the round-end suicide path.
	if defUser.Character.IsDisabled() {
		mob.Character.EndAggro()
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

	if handleCombatWaitRound(
		&mob.Character, defUser.Character,
		combat.Mob, combat.User,
		nil, defUser,
		mobRoom, defRoom,
		defUser.UserId,
	) {
		return
	}

	var roundResult combat.AttackResult

	// Stage 17.2: Moon phase stat modifiers (apply to mob attacker, symmetric with PvM)
	restore := applyMoonMods(&mob.Character, moonMod)
	roundResult = combat.AttackMobVsPlayer(mob, defUser)
	restore()

	// Stage 11.4: Minor Shield reduces physical weapon damage (MvP-only —
	// ConditionShield lives on player defenders).
	if roundResult.Hit && defUser.Character.HasCondition(characters.ConditionShield) {
		reduction := int(defUser.Character.GetConditionMagnitude(characters.ConditionShield)) / 2
		if roundResult.DamageToTarget > reduction+1 {
			roundResult.DamageToTarget -= reduction
			roundResult.DamageToTargetReduction += reduction
		}
	}

	applyCombatDamageBonuses_MvP(&roundResult, mob, defUser, mobRoom)

	// Stage 30.1: Record combat analytics
	mvpAtkType := "unarmed"
	if mob.Character.Equipment.Weapon.ItemId > 0 {
		mvpAtkType = "weapon"
	}
	combat.RecordAttack(roundResult, combat.Mob, combat.User, mvpAtkType, &mob.Character, defUser.Character, evt.RoundNumber)

	handleMvPCritAndMessaging(&roundResult, mob, defUser, mobRoom, defRoom)
	handleMvPProgression(&roundResult, mob, defUser, mobRoom, defRoom)
	handleMvPRoundResolution(mob, defUser, mobRoom)
}

// handleMobVsMob processes a mob attacking another mob.
func handleMobVsMob(mob *mobs.Mob, mobRoom *rooms.Room, evt events.NewRound, affectedMobInstanceIds *[]int) {
	*affectedMobInstanceIds = append(*affectedMobInstanceIds, mob.Character.Aggro.MobInstanceId)

	defMob := mobs.GetInstance(mob.Character.Aggro.MobInstanceId)

	if defMob == nil || mob.Character.RoomId != defMob.Character.RoomId {
		mob.Character.EndAggro()
		return
	}

	defRoom := rooms.LoadRoom(defMob.Character.RoomId)
	if defRoom == nil {
		mob.Character.EndAggro()
		return
	}

	defMob.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)

	if defMob.Character.Health < 1 {
		mob.Character.EndAggro()
		return
	}

	if mob.Character.Aggro.RoundsWaiting > 0 {
		mudlog.Debug(`RoundsWaiting`, `User`, mob.Character.Name, `Rounds`, mob.Character.Aggro.RoundsWaiting)
		mob.Character.Aggro.RoundsWaiting--

		roundResult := combat.GetWaitMessages(items.Wait, &mob.Character, &defMob.Character, combat.Mob, combat.Mob)

		for _, msg := range roundResult.MessagesToSourceRoom {
			sendVisualRoomText(mobRoom, msg)
		}
		for _, msg := range roundResult.MessagesToTargetRoom {
			sendVisualRoomText(defRoom, msg)
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

	// Conviction Surge buff: +15% damage bonus (mob attacker)
	if roundResult.Hit && roundResult.DamageToTarget > 0 && mob.Character.HasBuffFlag(buffs.DamageBonus) {
		bonusDmg := int(math.Round(float64(roundResult.DamageToTarget) * 0.15))
		if bonusDmg < 1 {
			bonusDmg = 1
		}
		defMob.Character.Health -= bonusDmg
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
				mob.Character.Health -= returnDmg
				dmgDesc := combat.GetDamageDescription(returnDmg, mob.Character.HealthMax.Value)
				atkMobName := mobDisplayName(mob, mobRoom, 0)
				defMobName := mobDisplayName(defMob, defRoom, 0)
				sendVisualRoomText(mobRoom, fmt.Sprintf(
					`<ansi fg="red">%s recoils from striking %s! (%s)</ansi>`,
					atkMobName, defMobName, dmgDesc))
			}
		}
	}

	// Lifesteal — attacking mob heals on hit
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
		sendVisualRoomText(mobRoom, msg)
	}
	for _, msg := range roundResult.MessagesToTargetRoom {
		sendVisualRoomText(defRoom, msg)
	}
	sendDarkRoomCombatFallback(mobRoom)
	if defRoom != mobRoom {
		sendDarkRoomCombatFallback(defRoom)
	}

	// Handle any scripted behavior now.
	if roundResult.Hit {
		// Behavior tree: fire-and-forget
		behaviortree.TryMobBehavior(defMob.InstanceId, behaviortree.EventContext{
			EventType: "mob_hurt",
			MobId:     mob.InstanceId,
			RoomId:    defMob.Character.RoomId,
		})
	}

	// Mobs get aggro when attacked
	if defMob.Character.Aggro == nil {
		defMob.PreventIdle = true
		defMob.Character.Aggro = &characters.Aggro{
			Type: characters.DefaultAttack,
		}
		defMob.Command(fmt.Sprintf("attack #%d", mob.InstanceId))
	}

	// Bidirectional autoassist: companion owner fights back when companion is attacked.
	handleCompanionOwnerAssist(defMob, fmt.Sprintf("#%d", mob.InstanceId))

	handleOffhandBreakMobDef(roundResult, defMob)

	// Stage 38.3: Mob attacker progression (skip room messages for MvM)
	mob.Character.OnStatUse("strength", 0)
	mob.Character.OnStatUse("dexterity", 0)
	for _, wh := range roundResult.WeaponHits {
		if wh.Hit {
			mob.Character.OnSkillUse(wh.SkillTag, 0)
		}
	}
	if len(roundResult.WeaponHits) == 0 && roundResult.Hit {
		mob.Character.OnSkillUse(string(skills.UnarmedCombat), 0)
	}

	// Defender progression — dodge/parry/block train specific skills
	processDefenderProgression(&defMob.Character, 0, roundResult)

	if mob.Character.Health <= 0 || defMob.Character.Health <= 0 {
		defMob.Character.EndAggro()
		if mob.Character.Health > 0 {
			RetargetOrEnd(&mob.Character, mobRoom, 0, mob.InstanceId)
		} else {
			mob.Character.EndAggro()
		}
	} else {
		mob.Character.SetAggro(0, defMob.InstanceId, characters.DefaultAttack)
	}
}
