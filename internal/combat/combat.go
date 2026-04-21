package combat

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type SourceTarget string

const (
	User SourceTarget = "user"
	Mob  SourceTarget = "mob"
)

// canSeeInRoom returns true if the character has nightvision or the room is lit.
func canSeeInRoom(char *characters.Character, room *rooms.Room) bool {
	if room == nil {
		return true
	}
	return room.GetVisibility() >= 1 || char.HasFlagFromAnySource(buffs.NightVision)
}

// Performs a combat round from a player to a mob
func AttackPlayerVsMob(user *users.UserRecord, mob *mobs.Mob) AttackResult {

	room := rooms.LoadRoom(user.Character.RoomId)
	ctx := combatContext{
		sourceCanSee: canSeeInRoom(user.Character, room),
		targetCanSee: canSeeInRoom(&mob.Character, room),
	}
	attackResult := calculateCombat(*user.Character, mob.Character, User, Mob, ctx)

	// Deduct stamina for the attack
	user.Character.DeductAttackStamina()

	if attackResult.DamageToSource != 0 {
		user.Character.ApplyHealthChange(attackResult.DamageToSource * -1)
		user.WimpyCheck()
	}

	mob.Character.ApplyHealthChange(attackResult.DamageToTarget * -1)

	// Remember who has hit him
	mob.Character.TrackPlayerDamage(user.UserId, attackResult.DamageToTarget)

	// Track progression stats for the attacking player
	user.Character.OnStatUse("strength", user.UserId)
	user.Character.OnStatUse("dexterity", user.UserId)
	if attackResult.Hit {
		user.PlaySound(`hit-other`, `combat`)
		combatSkill := string(user.Character.GetCombatSkillTag())
		user.Character.OnSkillUse(combatSkill, user.UserId)
		if attackResult.Crit {
			user.Character.OnCriticalSuccess(combatSkill, user.UserId)
		}
		// Track weapon-combat when dual wielding (dual wield governed by weapon-combat).
		// Exception: dual-wielding unarmed weapons (fist/claws, e.g., knuckles in both
		// hands) stays on unarmed-combat only — weapon-combat progression would be
		// inappropriate for pure unarmed combat.
		if isDualWieldingWeaponCombat(user.Character) {
			user.Character.OnSkillUse(string(skills.WeaponCombat), user.UserId)
		}
	} else {
		user.PlaySound(`miss`, `combat`)
		if attackResult.Fumble {
			combatSkill := string(user.Character.GetCombatSkillTag())
			user.Character.OnCriticalFailure(combatSkill, user.UserId)
		}
	}

	return attackResult
}

// isDualWieldingWeaponCombat reports whether the character is wielding two
// weapons where at least one trains weapon-combat. Returns false when both
// hands are empty, when only one hand is armed, when the offhand is a shield
// (type != Weapon), or when both weapons are unarmed subtypes (fist/claws).
func isDualWieldingWeaponCombat(c *characters.Character) bool {
	if c.Equipment.Weapon.ItemId == 0 || c.Equipment.Offhand.ItemId == 0 {
		return false
	}
	if c.Equipment.Offhand.GetSpec().Type != items.Weapon {
		return false
	}
	mainTag := characters.CombatSkillTagForItem(c.Equipment.Weapon)
	offTag := characters.CombatSkillTagForItem(c.Equipment.Offhand)
	return mainTag == skills.WeaponCombat || offTag == skills.WeaponCombat
}

// Performs a combat round from a player to a player
func AttackPlayerVsPlayer(userAtk *users.UserRecord, userDef *users.UserRecord) AttackResult {

	room := rooms.LoadRoom(userAtk.Character.RoomId)
	ctx := combatContext{
		sourceCanSee: canSeeInRoom(userAtk.Character, room),
		targetCanSee: canSeeInRoom(userDef.Character, room),
	}
	attackResult := calculateCombat(*userAtk.Character, *userDef.Character, User, User, ctx)

	// Deduct stamina for the attack
	userAtk.Character.DeductAttackStamina()

	if attackResult.DamageToSource != 0 {
		userAtk.Character.ApplyHealthChange(attackResult.DamageToSource * -1)
		userAtk.WimpyCheck()
	}

	if attackResult.DamageToTarget != 0 {
		userDef.Character.ApplyHealthChange(attackResult.DamageToTarget * -1)
		userDef.WimpyCheck()
	}

	// Track progression stats for the attacking player
	userAtk.Character.OnStatUse("strength", userAtk.UserId)
	userAtk.Character.OnStatUse("dexterity", userAtk.UserId)
	if attackResult.Hit {
		userAtk.PlaySound(`hit-other`, `combat`)
		userDef.PlaySound(`hit-self`, `combat`)
		combatSkill := string(userAtk.Character.GetCombatSkillTag())
		userAtk.Character.OnSkillUse(combatSkill, userAtk.UserId)
		if attackResult.Crit {
			userAtk.Character.OnCriticalSuccess(combatSkill, userAtk.UserId)
		}
		// Track weapon-combat when dual wielding real weapons (see helper below)
		if isDualWieldingWeaponCombat(userAtk.Character) {
			userAtk.Character.OnSkillUse(string(skills.WeaponCombat), userAtk.UserId)
		}
	} else {
		userAtk.PlaySound(`miss`, `combat`)
		if attackResult.Fumble {
			combatSkill := string(userAtk.Character.GetCombatSkillTag())
			userAtk.Character.OnCriticalFailure(combatSkill, userAtk.UserId)
		}
	}

	return attackResult
}

// trackMobAttackProgression mirrors the player progression calls in
// AttackPlayerVsMob / AttackPlayerVsPlayer for a mob attacker.
// MobActor cannot be used here because actions imports combat (cycle).
// We call character methods directly with userId=0 (mob convention).
func trackMobAttackProgression(mob *mobs.Mob, result AttackResult) {
	mob.Character.OnStatUse("strength", 0)
	mob.Character.OnStatUse("dexterity", 0)
	for _, wh := range result.WeaponHits {
		if wh.Hit {
			mob.Character.OnSkillUse(wh.SkillTag, 0)
			if wh.Crit {
				mob.Character.OnCriticalSuccess(wh.SkillTag, 0)
			}
		} else if wh.Fumble {
			mob.Character.OnCriticalFailure(wh.SkillTag, 0)
		}
	}
	if len(result.WeaponHits) == 0 && result.Hit {
		mob.Character.OnSkillUse(string(skills.UnarmedCombat), 0)
	}
}

// Performs a combat round from a mob to a player
func AttackMobVsPlayer(mob *mobs.Mob, user *users.UserRecord) AttackResult {

	room := rooms.LoadRoom(mob.Character.RoomId)
	ctx := combatContext{
		sourceCanSee: canSeeInRoom(&mob.Character, room),
		targetCanSee: canSeeInRoom(user.Character, room),
	}
	attackResult := calculateCombat(mob.Character, *user.Character, Mob, User, ctx)

	// Deduct stamina for the attack
	mob.Character.DeductAttackStamina()

	mob.Character.ApplyHealthChange(attackResult.DamageToSource * -1)

	if attackResult.DamageToTarget != 0 {
		user.Character.ApplyHealthChange(attackResult.DamageToTarget * -1)
		user.WimpyCheck()
	}

	// Track defender's dexterity use (reacting to attacks)
	user.Character.OnStatUse("dexterity", user.UserId)

	// Track progression for the attacking mob (mirrors player attacker logic)
	trackMobAttackProgression(mob, attackResult)

	if attackResult.Hit {
		user.PlaySound(`hit-self`, `combat`)
	}

	return attackResult
}

// Performs a combat round from a mob to a mob
func AttackMobVsMob(mobAtk *mobs.Mob, mobDef *mobs.Mob) AttackResult {

	room := rooms.LoadRoom(mobAtk.Character.RoomId)
	ctx := combatContext{
		sourceCanSee: canSeeInRoom(&mobAtk.Character, room),
		targetCanSee: canSeeInRoom(&mobDef.Character, room),
	}
	attackResult := calculateCombat(mobAtk.Character, mobDef.Character, Mob, Mob, ctx)

	// Deduct stamina for the attack
	mobAtk.Character.DeductAttackStamina()

	mobAtk.Character.ApplyHealthChange(attackResult.DamageToSource * -1)
	mobDef.Character.ApplyHealthChange(attackResult.DamageToTarget * -1)

	// If attacking mob was player charmed, attribute damage done to that player
	if charmedUserId := mobAtk.Character.GetCharmedUserId(); charmedUserId > 0 {
		// Remember who has hit him
		mobDef.Character.TrackPlayerDamage(charmedUserId, attackResult.DamageToTarget)
	}

	// Track progression for both mobs
	trackMobAttackProgression(mobAtk, attackResult)
	// Defender dexterity (mirrors player defender tracking in AttackMobVsPlayer)
	mobDef.Character.OnStatUse("dexterity", 0)

	return attackResult
}

func GetWaitMessages(stepType items.Intensity, sourceChar *characters.Character, targetChar *characters.Character, sourceType SourceTarget, targetType SourceTarget) AttackResult {

	attackResult := AttackResult{}

	msgs := items.GetPreAttackMessage(sourceChar.Equipment.Weapon.GetSpec().Subtype, stepType)

	var toAttackerMsg, toDefenderMsg, toAttackerRoomMsg, toDefenderRoomMsg items.ItemMessage

	// zero means randomly selected, otherwise use the ItemId to consistently choose a message
	msgSeed := 0
	if configs.GetBalanceConfig().ConsistentAttackMessages {
		msgSeed = sourceChar.Equipment.Weapon.ItemId
	}

	// Stage 9.4: Track attack for stance calculation
	sourceChar.IncrementAttackCount()

	tokenReplacements := map[items.TokenName]string{
		items.TokenItemName:     species.GetSpecies(sourceChar.SpeciesId).UnarmedName,
		items.TokenSource:       sourceChar.Name,
		items.TokenSourceType:   string(sourceType) + `name`,
		items.TokenTarget:       targetChar.Name,
		items.TokenTargetType:   string(targetType) + `name`,
		items.TokenUsesLeft:     `[Invalid]`,
		items.TokenDamage:       `[Invalid]`,
		items.TokenEntranceName: `unknown`,
		items.TokenExitName:     `unknown`,
		items.TokenStance:       sourceChar.CalculateStanceString(),
		items.TokenPosition:     sourceChar.CalculatePositionString(),
		items.TokenMomentum:     sourceChar.CalculateMomentumString(),
	}

	// Get source character's weapon skill level for message selection
	skillLevel := sourceChar.GetCombatSkillLevel()

	if sourceChar.RoomId == targetChar.RoomId {
		toAttackerMsg = msgs.Together.ToAttacker.GetForSkillLevel(skillLevel, msgSeed)
		toDefenderMsg = msgs.Together.ToDefender.GetForSkillLevel(skillLevel, msgSeed)
		toAttackerRoomMsg = msgs.Together.ToRoom.GetForSkillLevel(skillLevel, msgSeed)
		toDefenderRoomMsg = items.ItemMessage("")

	} else {

		toAttackerMsg = msgs.Separate.ToAttacker.GetForSkillLevel(skillLevel, msgSeed)
		toDefenderMsg = msgs.Separate.ToDefender.GetForSkillLevel(skillLevel, msgSeed)
		toAttackerRoomMsg = msgs.Separate.ToAttackerRoom.GetForSkillLevel(skillLevel, msgSeed)
		toDefenderRoomMsg = msgs.Separate.ToDefenderRoom.GetForSkillLevel(skillLevel, msgSeed)

		// Find the exit that leads to the target from the source (if any)
		if atkRoom := rooms.LoadRoom(sourceChar.RoomId); atkRoom != nil {
			tokenReplacements[items.TokenExitName] = `unknown`
			for exitName, exit := range atkRoom.Exits {
				if exit.RoomId == targetChar.RoomId {
					tokenReplacements[items.TokenExitName] = exitName
					break
				}
			}
		}
		// find the exit that leads to the source from the target (if any)
		if defRoom := rooms.LoadRoom(targetChar.RoomId); defRoom != nil {
			tokenReplacements[items.TokenEntranceName] = `unknown`
			for exitName, exit := range defRoom.Exits {
				if exit.RoomId == sourceChar.RoomId {
					tokenReplacements[items.TokenEntranceName] = exitName
					break
				}
			}
		}
	}

	if sourceChar.Equipment.Weapon.ItemId > 0 {
		tokenReplacements[items.TokenItemName] = sourceChar.Equipment.Weapon.DisplayName()
	}

	if sourceType == Mob {
		tokenReplacements[items.TokenSource] = sourceChar.GetMobName(0).String()
	}

	if targetType == Mob {
		tokenReplacements[items.TokenTarget] = targetChar.GetMobName(0).String()
	}

	for tokenName, tokenValue := range tokenReplacements {
		toAttackerMsg = toAttackerMsg.SetTokenValue(tokenName, tokenValue)
		toDefenderMsg = toDefenderMsg.SetTokenValue(tokenName, tokenValue)
		toAttackerRoomMsg = toAttackerRoomMsg.SetTokenValue(tokenName, tokenValue)
		if len(string(toDefenderRoomMsg)) > 0 {
			toDefenderRoomMsg = toDefenderRoomMsg.SetTokenValue(tokenName, tokenValue)
		}
	}

	if string(toAttackerMsg) != `` {
		attackResult.SendToSource(string(toAttackerMsg))
	}

	if !sourceChar.HasBuffFlag(buffs.Hidden) {

		if string(toDefenderMsg) != `` {
			attackResult.SendToTarget(string(toDefenderMsg))
		}

		if string(toAttackerRoomMsg) != `` {
			attackResult.SendToSourceRoom(string(toAttackerRoomMsg))
		}

		if sourceChar.RoomId != targetChar.RoomId {
			if string(toDefenderRoomMsg) != `` {
				attackResult.SendToTargetRoom(string(toDefenderRoomMsg))
			}
		}

	}

	return attackResult
}

func calculateCombat(sourceChar characters.Character, targetChar characters.Character, sourceType SourceTarget, targetType SourceTarget, ctx combatContext) AttackResult {

	attackResult := AttackResult{}

	// Statmods can add a damage bonus
	statModDBonus := sourceChar.StatMod(`damage`)
	extraAttacks := sourceChar.StatMod(`attacks`)

	attackWeapons := collectAttackWeapons(&sourceChar)

	attackMessagePrefix := ``
	backstabCrit := false
	if sourceChar.Aggro.Type == characters.SurpriseAttack {
		backstabCrit = true
		attackMessagePrefix = `<ansi fg="magenta-bold">*[SURPRISE ATTACK]*</ansi> `
		sourceChar.SetAggro(sourceChar.Aggro.UserId, sourceChar.Aggro.MobInstanceId, characters.DefaultAttack)
	}

	attackResult.DefenderWasAttacked = len(attackWeapons) > 0

	for weaponIdx, weapon := range attackWeapons {

		ws := buildWeaponSetup(&sourceChar, &targetChar, weapon, weaponIdx, len(attackWeapons))
		sdp := buildDamageParams(&sourceChar, &targetChar, ws, statModDBonus, sourceType)
		sdp.critBuffs = ws.critBuffs

		// Track per-weapon hits for skill progression
		weaponHit := WeaponHitInfo{
			SkillTag: string(characters.CombatSkillTagForItem(weapon)),
		}

		// Single merged swing count per weapon
		swingCount := calcSwingCount(&sourceChar, ws.weaponSpeed, extraAttacks, ws.isOffhand)

		mudlog.Debug("DistDamage", "swings", swingCount, "baseDmg", ws.baseDmg, "variance", sdp.dmgVariance, "dmgMean", sdp.dmgMean, "weaponMult", ws.weaponDmgMult, "critBuffs", ws.critBuffs)

		critThreshold := calcCritThreshold(&sourceChar, &targetChar)

		for j := 0; j < swingCount; j++ {

			mudlog.Debug(`calculateCombat`, `Swing`, fmt.Sprintf(`%d/%d`, j+1, swingCount), `Weapon`, ws.weaponName, `Source`, fmt.Sprintf(`%s (%s)`, sourceChar.Name, sourceType), `Target`, fmt.Sprintf(`%s (%s)`, targetChar.Name, targetType))

			// Reset per-swing flags
			attackResult.Crit = false
			attackResult.Fumble = false
			attackResult.DoubleFumble = false

			attackTargetDamage := 0
			attackTargetReduction := 0
			attackSourceDamage := 0
			attackSourceReduction := 0

			attackScore := calcAttackScore(&sourceChar, &targetChar, ws.penalty, ctx)

			defenseSequence := targetChar.GetDefenseSequence()

			// Third-party grapple vulnerability
			defenseSequence, isThirdParty := filterDefensesForThirdParty(&attackResult, &sourceChar, &targetChar, defenseSequence)

			// Roll attack once via best-of-all defense
			best := runBestOfAllDefense(&attackResult, &sourceChar, &targetChar, defenseSequence, attackScore, isThirdParty, ctx)

			// New resolution order: fumbles → crits → normal → floors
			res := resolveDefenseOutcome(&attackResult, best, &sourceChar, &targetChar, critThreshold, isThirdParty)

			sourceChar.UpdateMomentum(res.hit)

			if res.hit {
				attackResult.Hit = true
				weaponHit.Hit = true
				if res.crit {
					weaponHit.Crit = true
				}
				attackTargetDamage, backstabCrit = calcHitDamage(&attackResult, res.crit, backstabCrit, sdp)
			}

			if res.fumble {
				attackResult.Fumble = true
				weaponHit.Fumble = true
			}

			// Determine per-swing attack type for analytics
			swingAtkType := "unarmed"
			if weaponHit.SkillTag == string(skills.WeaponCombat) {
				swingAtkType = "weapon"
			}

			// Record per-swing analytics
			attackResult.SwingEvents = append(attackResult.SwingEvents, SwingEvent{
				Hit:           res.hit,
				Crit:          res.crit,
				Fumble:        res.fumble,
				DoubleFumble:  res.doubleFumble,
				DefenseCrit:   res.defenseCrit,
				Damage:        attackTargetDamage,
				DamageReduced: attackTargetReduction,
				DefenseUsed:   attackResult.DefenseUsed,
				AttackZScore:  attackResult.AttackZScore,
				DefenseZScore: attackResult.DefenseZScore,
				AttackType:    swingAtkType,
			})

			// Only build attack messages for non-double-fumble (double fumble already sent)
			if !res.doubleFumble {
				buildAttackMessages(&attackResult, &sourceChar, &targetChar, ws, sdp,
					attackTargetDamage, attackTargetReduction, attackSourceDamage, attackSourceReduction,
					sourceType, targetType, attackMessagePrefix)
			}

			attackResult.DamageToTarget += attackTargetDamage
			attackResult.DamageToTargetReduction += attackTargetReduction
			attackResult.DamageToSource += attackSourceDamage
			attackResult.DamageToSourceReduction += attackSourceReduction
		}

		attackResult.WeaponHits = append(attackResult.WeaponHits, weaponHit)
		applyPetDamage(&attackResult, &sourceChar, &targetChar, targetType)
	}

	// If unarmed (no weapons at all), add unarmed entry
	if len(attackWeapons) == 0 {
		attackResult.DefenderWasAttacked = true
	}

	return attackResult

}
