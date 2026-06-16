package hooks

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/state/activity"
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
//
// The cat parameter classifies the message so the central pipeline
// can apply category-appropriate color + normalization. Callers used
// to omit this; T11 added it as a required arg.
func sendVisualRoomText(room *rooms.Room, cat messaging.Category, visualMsg string, excludeUserIds ...int) {
	if room == nil {
		return
	}
	room.SendTextVisual(cat, visualMsg, excludeUserIds...)
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
			u.SendText(messaging.CategoryDefault, `<ansi fg="yellow">You hear the sounds of fighting nearby.</ansi>`)
		}
	}
}

// replaceDarknessMessages replaces detailed combat messages with generic
// darkness text for combatants who cannot see.
func replaceDarknessMessages(result *combat.AttackResult, sourceCanSee bool, targetCanSee bool) {
	if sourceCanSee && targetCanSee {
		return
	}

	// Build replacement messages based on swing events. Dark-room
	// substitutes carry the same per-line outcome category as the
	// equivalent sighted line — defense crit / dodge → CategoryDodge,
	// hits → CategoryHitMelee, miss/fumble → CategoryHitMelee.
	if !sourceCanSee {
		newMsgs := make([]combat.TaggedMessage, 0, len(result.SwingEvents))
		for _, se := range result.SwingEvents {
			if se.DoubleFumble {
				continue
			}
			switch {
			case se.Fumble:
				newMsgs = append(newMsgs, combat.TaggedMessage{Category: messaging.CategoryHitMelee, Text: `<ansi fg="fumble-text">!!!</ansi> <ansi fg="yellow">You stumble badly in the darkness!</ansi> <ansi fg="fumble-text">!!!</ansi>`})
			case se.Crit:
				newMsgs = append(newMsgs, combat.TaggedMessage{Category: messaging.CategoryHitMelee, Text: `<ansi fg="crit-text">***</ansi> <ansi fg="attack-good">You land a devastating blow in the dark!</ansi> <ansi fg="crit-text">***</ansi>`})
			case se.DefenseCrit || se.DefenseUsed != "":
				newMsgs = append(newMsgs, combat.TaggedMessage{Category: messaging.CategoryDodge, Text: `<ansi fg="attack-bad">Your attack is deflected by something!</ansi>`})
			case se.Hit:
				newMsgs = append(newMsgs, combat.TaggedMessage{Category: messaging.CategoryHitMelee, Text: `<ansi fg="attack-good">You strike blindly and connect!</ansi>`})
			default:
				newMsgs = append(newMsgs, combat.TaggedMessage{Category: messaging.CategoryHitMelee, Text: `<ansi fg="yellow">You swing wildly in the darkness!</ansi>`})
			}
		}
		if len(newMsgs) > 0 {
			result.MessagesToSource = newMsgs
		}
	}

	if !targetCanSee {
		newMsgs := make([]combat.TaggedMessage, 0, len(result.SwingEvents))
		for _, se := range result.SwingEvents {
			if se.DoubleFumble {
				continue
			}
			switch {
			case se.Fumble:
				newMsgs = append(newMsgs, combat.TaggedMessage{Category: messaging.CategoryHitMelee, Text: `<ansi fg="yellow">You hear your attacker stumble!</ansi>`})
			case se.Crit:
				newMsgs = append(newMsgs, combat.TaggedMessage{Category: messaging.CategoryHitMelee, Text: `<ansi fg="crit-text">***</ansi> <ansi fg="red">Something hits you hard in the dark!</ansi> <ansi fg="crit-text">***</ansi>`})
			case se.DefenseCrit || se.DefenseUsed != "":
				newMsgs = append(newMsgs, combat.TaggedMessage{Category: messaging.CategoryDodge, Text: `<ansi fg="defense-good">You fend off something in the dark!</ansi>`})
			case se.Hit:
				newMsgs = append(newMsgs, combat.TaggedMessage{Category: messaging.CategoryHitMelee, Text: `<ansi fg="red">Something strikes you in the dark!</ansi>`})
			default:
				newMsgs = append(newMsgs, combat.TaggedMessage{Category: messaging.CategoryHitMelee, Text: `<ansi fg="yellow">You hear something whoosh past!</ansi>`})
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
			user.SendText(messaging.CategoryBuffExpire, `<ansi fg="blue">Your Minor Shield dissipates.</ansi>`)
		} else {
			user.Character.DecrementCondition(characters.ConditionShield)
		}
	}
}

// castingTargetChar returns the first target character from a CastingData, or nil.
func castingTargetChar(cs activity.CastingData) *characters.Character {
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
	if user.Character.Activity == nil || !user.Character.Activity.IsCasting() {
		return false
	}

	// Capture state before processFoldRound clears it on terminal conditions.
	csBeforeProcess, _ := user.Character.Activity.CastingData()

	// Bleeding out = automatic concentration break (player-only check).
	if user.Character.IsDisabled() {
		recordConcentrationFailure(combat.User, combat.Mob, user.Character, castingTargetChar(csBeforeProcess))
		clearCastingActivity(user.Character, activity.TriggerConcentrationBreak)
		return true
	}

	result := processFoldRound(user.Character)

	switch {
	case result.ProneBroke:
		recordConcentrationFailure(combat.User, combat.Mob, user.Character, castingTargetChar(csBeforeProcess))
		user.SendText(messaging.CategorySpellDisruption, `<ansi fg="red">You lose your concentration as you hit the ground!</ansi>`)
		room := rooms.LoadRoom(user.Character.RoomId)
		if room != nil {
			sendVisualRoomText(room, messaging.CategorySpellDisruption, fmt.Sprintf(
				`<ansi fg="username">%s</ansi>'s concentration breaks.`, user.Character.Name), user.UserId)
		}

	case result.GrappleBroke:
		// Chunk 4e T4: grapple breaks concentration same as Prone (spec §4.2).
		recordConcentrationFailure(combat.User, combat.Mob, user.Character, castingTargetChar(csBeforeProcess))
		user.SendText(messaging.CategorySpellDisruption, `<ansi fg="red">Your concentration shatters — you cannot hold the fold while grappled!</ansi>`)
		room := rooms.LoadRoom(user.Character.RoomId)
		if room != nil {
			sendVisualRoomText(room, messaging.CategorySpellDisruption, fmt.Sprintf(
				`<ansi fg="username">%s</ansi>'s concentration breaks.`, user.Character.Name), user.UserId)
		}

	case result.TargetGone:
		recordConcentrationFailure(combat.User, combat.Mob, user.Character, castingTargetChar(csBeforeProcess))
		user.SendText(messaging.CategorySpellDisruption, `<ansi fg="red">Your spell fizzles — the target is gone.</ansi>`)

	case result.SpellDataMissing:
		user.SendText(messaging.CategorySpellDisruption, `<ansi fg="red">The spell dissipates — its data cannot be found.</ansi>`)

	case result.InsufficientConviction:
		recordConcentrationFailure(combat.User, combat.Mob, user.Character, castingTargetChar(csBeforeProcess))
		user.SendText(messaging.CategorySpellDisruption, `<ansi fg="red">Your conviction wavers — the fold collapses.</ansi>`)

	case result.CastComplete:
		cs := result.CastingData
		spellData := result.SpellData
		// Send YAML wait text (if defined).
		if spellData != nil && (spellData.WaitUserText != "" || spellData.WaitRoomText != "") {
			tCtx := textutil.TokenContext{
				SourceName:      user.Character.GetCharacterName(true),
				SourcePlainName: user.Character.GetCharacterName(false),
			}
			cfg := textutil.SendTextConfig{
				UserSendFunc: func(msg string) { user.SendText(messaging.CategorySpellFold, msg) },
				RoomSendFunc: func(msg string, skip ...int) {
					if r := rooms.LoadRoom(user.Character.RoomId); r != nil {
						r.SendText(messaging.CategorySpellFold, msg, skip...)
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
		perception := user.Character.Stats.Perception.ValueAdj
		traditionalChance := configs.DiscoveryChance(configs.DiscoveryParams{
			Base:       float64(bal.SpellDiscoveryBaseChance),
			Decay:      float64(bal.SpellDiscoveryDecayRate),
			Known:      knownCount,
			Perception: perception,
			Skill:      castSkillLevel,
		})
		if util.Rand(100) < int(traditionalChance) {
			eligible := spells.GetEligibleSpells(user.Character.SpellBook, castSkillLevel,
				spells.SchoolElemental, spells.SchoolEnhancement, spells.SchoolMental, spells.SchoolVital)
			if len(eligible) > 0 {
				pick := eligible[util.Rand(len(eligible))]
				if user.Character.LearnSpell(pick) {
					if newSpell := spells.GetSpell(pick); newSpell != nil {
						user.SendText(messaging.CategorySkillProgress, fmt.Sprintf(
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
			manifestChance := configs.DiscoveryChance(configs.DiscoveryParams{
				Base:       float64(bal.SpellDiscoveryBaseChance),
				Decay:      float64(bal.SpellDiscoveryDecayRate),
				Known:      knownCount,
				Perception: perception,
				Skill:      manifestSkillLevel,
			})
			if util.Rand(100) < int(manifestChance) {
				eligible := spells.GetEligibleSpells(user.Character.SpellBook, manifestSkillLevel,
					spells.SchoolManifestation)
				if len(eligible) > 0 {
					pick := eligible[util.Rand(len(eligible))]
					if user.Character.LearnSpell(pick) {
						if newSpell := spells.GetSpell(pick); newSpell != nil {
							user.SendText(messaging.CategorySkillProgress, fmt.Sprintf(
								`<ansi fg="magenta-bold">A manifestation reveals itself: <ansi fg="cyan-bold">%s</ansi></ansi>`,
								newSpell.Name))
						}
					}
				}
			}
		}

	case result.StillCasting:
		cs := result.CastingData
		// Send YAML wait text (if defined).
		waitSpellInfo := spells.GetSpell(cs.SpellId)
		if waitSpellInfo != nil && (waitSpellInfo.WaitUserText != "" || waitSpellInfo.WaitRoomText != "") {
			tCtx := textutil.TokenContext{
				SourceName:      user.Character.GetCharacterName(true),
				SourcePlainName: user.Character.GetCharacterName(false),
			}
			cfg := textutil.SendTextConfig{
				UserSendFunc: func(msg string) { user.SendText(messaging.CategorySpellFold, msg) },
				RoomSendFunc: func(msg string, skip ...int) {
					if r := rooms.LoadRoom(user.Character.RoomId); r != nil {
						r.SendText(messaging.CategorySpellFold, msg, skip...)
					}
				},
				ExcludeId: user.UserId,
			}
			textutil.SendPhaseText(waitSpellInfo.WaitUserText, waitSpellInfo.WaitRoomText, tCtx, "pink", cfg)
		}
		user.SendText(messaging.CategorySpellFold, spells.GetCastMessage("cast_started", cs.SpellId))
	}

	return true
}

// handleMobFoldCasting processes fold spell casting for a mob.
// Returns true if the mob is casting and should skip combat.
func handleMobFoldCasting(mob *mobs.Mob, mobRoom *rooms.Room) bool {
	if mob.Character.Activity == nil || !mob.Character.Activity.IsCasting() {
		return false
	}

	// Capture state before processFoldRound clears it on terminal conditions.
	csBeforeProcess, _ := mob.Character.Activity.CastingData()

	result := processFoldRound(&mob.Character)

	switch {
	case result.ProneBroke:
		recordConcentrationFailure(combat.Mob, combat.User, &mob.Character, castingTargetChar(csBeforeProcess))
		mobRoom.SendText(messaging.CategorySpellDisruption, fmt.Sprintf(
			`%s's concentration breaks.`, mobDisplayName(mob, mobRoom, 0)))

	case result.GrappleBroke:
		// Chunk 4e T4: grapple breaks concentration same as Prone (spec §4.2).
		recordConcentrationFailure(combat.Mob, combat.User, &mob.Character, castingTargetChar(csBeforeProcess))
		mobRoom.SendText(messaging.CategorySpellDisruption, fmt.Sprintf(
			`%s's concentration breaks.`, mobDisplayName(mob, mobRoom, 0)))

	case result.TargetGone:
		recordConcentrationFailure(combat.Mob, combat.User, &mob.Character, castingTargetChar(csBeforeProcess))
		mobRoom.SendText(messaging.CategorySpellDisruption, fmt.Sprintf(
			`%s's spell fizzles.`, mobDisplayName(mob, mobRoom, 0)))

	case result.SpellDataMissing:
		// Silent failure — no message for missing spell data on mobs.

	case result.InsufficientConviction:
		recordConcentrationFailure(combat.Mob, combat.User, &mob.Character, castingTargetChar(csBeforeProcess))
		mobRoom.SendText(messaging.CategorySpellDisruption, fmt.Sprintf(
			`%s's spell falters.`, mobDisplayName(mob, mobRoom, 0)))

	case result.CastComplete:
		cs := result.CastingData
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
			perception := mob.Character.Stats.Perception.ValueAdj
			traditionalChance := configs.DiscoveryChance(configs.DiscoveryParams{
				Base:       float64(bal.SpellDiscoveryBaseChance),
				Decay:      float64(bal.SpellDiscoveryDecayRate),
				Known:      knownCount,
				Perception: perception,
				Skill:      castSkillLevel,
			})
			// Traditional school discovery.
			if util.Rand(100) < int(traditionalChance) {
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
				manifestChance := configs.DiscoveryChance(configs.DiscoveryParams{
					Base:       float64(bal.SpellDiscoveryBaseChance),
					Decay:      float64(bal.SpellDiscoveryDecayRate),
					Known:      knownCount,
					Perception: perception,
					Skill:      manifestSkillLevel,
				})
				if util.Rand(100) < int(manifestChance) {
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
		mobRoom.SendText(messaging.CategorySpellFold, fmt.Sprintf(
			`%s weaves magic with focused intent.`, mobDisplayName(mob, mobRoom, 0)))
	}

	return true
}

// handlePlayerFlee processes a player's flee attempt.
// Returns true if the player is fleeing and should skip combat.
func handlePlayerFlee(user *users.UserRecord, uRoom *rooms.Room, userId int) bool {
	// Task 15: IsDisengaging() reads CombatPhase.State() == Disengaging,
	// set by TransitionToDisengaging in flee.go. This replaces the legacy
	// Aggro.Type == Flee sentinel check.
	// TODO Task 18: remove legacy Aggro.Type fallback once Aggro is gone.
	isFleeing := user.Character.IsDisengaging()
	if !isFleeing && user.Character.Aggro != nil && user.Character.Aggro.Type == characters.Flee {
		// Legacy path: Aggro-only set (no CombatPhase wired). Still handled.
		isFleeing = true
	}
	if !isFleeing {
		return false
	}

	// Revert to Default combat regardless of outcome (legacy path only).
	// When CombatPhase is wired, ResolveFlee(false) handles the revert.
	if user.Character.Aggro != nil && user.Character.Aggro.Type == characters.Flee {
		user.Character.SetAggro(user.Character.Aggro.UserId, user.Character.Aggro.MobInstanceId, characters.DefaultAttack)
	}

	// Can't flee while in any grapple state. CombatPhase position veto
	// also blocks TransitionToDisengaging, but the message still needs
	// to fire here for UX. Chunk 4b R3: FSM-driven — IsStandingGrapple
	// || IsGroundGrapple covers all 11 grapple states.
	if user.Character.IsStandingGrapple() || user.Character.IsGroundGrapple() {
		user.SendText(messaging.CategorySystem, `<ansi fg="red">You can't flee while grappled!</ansi>`)
		if user.Character.CombatPhase != nil {
			user.Character.CombatPhase.ResolveFlee(false)
		}
		return true
	}

	// Shared opposed-roll blocker resolution (combat.ResolveFleeBlockers).
	// Replaces two duplicated loops; also corrects the prior
	// variable-shadowing in the player-blockers loop (the inner
	// `for _, userId := range` shadowed the outer fleer's id, so PvP
	// players never blocked each other from fleeing). Perspective-
	// specific messaging stays here.
	if blocker := combat.ResolveFleeBlockers(user.Character, uRoom); blocker != nil {
		var targetTag string
		if blocker.IsPlayer() {
			targetTag = "username"
		} else {
			targetTag = "mobname"
		}
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="red-bold"><ansi fg="%s">%s</ansi> blocks you from fleeing!</ansi>`, targetTag, blocker.Name))
		excludes := []int{user.UserId}
		if blocker.IsPlayer() {
			excludes = append(excludes, blocker.UserId)
		}
		uRoom.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> is blocked from fleeing by <ansi fg="%s">%s</ansi>!`, user.Character.Name, targetTag, blocker.Name), excludes...)
		// Task 15: flee failure — restore Engaged state in CombatPhase.
		if user.Character.CombatPhase != nil {
			user.Character.CombatPhase.ResolveFlee(false)
		}
		return true
	}

	// Success!
	exitName, exitRoomId := uRoom.GetRandomExit()

	if exitName == `` {
		user.SendText(messaging.CategorySystem, `You can't find an exit!`)
		// No exit found — treat as blocked (flee failure).
		if user.Character.CombatPhase != nil {
			user.Character.CombatPhase.ResolveFlee(false)
		}
		return true
	}

	user.SendText(messaging.CategoryRoomExit, fmt.Sprintf(`You flee to the <ansi fg="exit">%s</ansi> exit!`, exitName))
	uRoom.SendText(messaging.CategoryRoomExit, fmt.Sprintf(`<ansi fg="username">%s</ansi> flees to the <ansi fg="exit">%s</ansi> exit!`, user.Character.Name, exitName), user.UserId)

	// Task 15: flee success — EndAggro clears legacy Aggro; ResolveFlee
	// transitions CombatPhase Disengaging → Idle.
	user.Character.EndAggro()
	if user.Character.CombatPhase != nil {
		user.Character.CombatPhase.ResolveFlee(true)
	}

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

	defUser.SendText(messaging.CategoryEquipment, `<ansi fg="202">***</ansi>`)
	defUser.SendText(messaging.CategoryEquipment, fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> Your <ansi fg="item">%s</ansi> breaks! <ansi fg="202">***</ansi></ansi>`, br.BrokenItemName))
	defUser.SendText(messaging.CategoryEquipment, `<ansi fg="202">***</ansi>`)

	defRoom.SendText(messaging.CategoryEquipment, fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> The <ansi fg="item">%s</ansi> <ansi fg="username">%s</ansi> was carrying breaks! <ansi fg="202">***</ansi></ansi>`, br.BrokenItemName, defUser.Character.Name), defUser.UserId)

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
		defRoom.SendText(messaging.CategoryEquipment, fmt.Sprintf(`<ansi fg="214"><ansi fg="202">***</ansi> The <ansi fg="item">%s</ansi> <ansi fg="mobname">%s</ansi> was carrying breaks! <ansi fg="202">***</ansi></ansi>`, br.BrokenItemName, defMob.Character.Name))
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

// handlePlayerConcentrationBreak checks if a caster's concentration breaks when hit,
// and hard-cancels any in-progress crafting or salvaging activity on the same damage hit.
// The two helpers are independent: a Casting character gets a willpower roll;
// a Crafting/Salvaging character is always interrupted (no roll).
func handlePlayerConcentrationBreak(defUser *users.UserRecord, roundResult combat.AttackResult, defRoom *rooms.Room) {
	// Hard-cancel craft/salvage on any damage (no roll needed).
	if roundResult.DamageToTarget > 0 {
		cancelCraftOrSalvageOnDamage(defUser.Character)
	}
	if checkConcentrationBreak(defUser.Character, roundResult.DamageToTarget) {
		csSnap, _ := defUser.Character.Activity.CastingData()
		recordConcentrationFailure(combat.User, combat.Mob, defUser.Character, castingTargetChar(csSnap))
		clearCastingActivity(defUser.Character, activity.TriggerConcentrationBreak)
		defUser.SendText(messaging.CategorySpellDisruption, `<ansi fg="red">The pain shatters your concentration!</ansi>`)
		defRoom.SendText(messaging.CategorySpellDisruption, fmt.Sprintf(
			`<ansi fg="username">%s</ansi>'s concentration breaks.`,
			defUser.Character.Name), defUser.UserId)
	}
}

// handleMobAIDecision processes mob AI decisions (spell casting, special moves, combat commands).
// Returns true if the mob executed an AI action and should skip normal combat.
func handleMobAIDecision(mob *mobs.Mob, c configs.Config) bool {
	// Aggro can be cleared mid-round before this AI pass runs — e.g. a kiting
	// archer whose target left the room (keep_distance repositions it, and the
	// unified combat handler drops a mob's aggro when its target isn't present).
	// The unified dispatch below guards on Aggro != nil; mirror that here so we
	// never deref a nil Aggro (was a server-crashing nil-pointer panic).
	if mob.Character.Aggro == nil {
		return false
	}
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
			if u.Character.Health > 0 && !u.Character.IsHidden() {
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
			mobRoom.SendText(messaging.CategoryMobEmote,
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
