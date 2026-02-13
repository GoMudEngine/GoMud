package combat

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type SourceTarget string

const (
	User SourceTarget = "user"
	Mob  SourceTarget = "mob"
)

// Performs a combat round from a player to a mob
func AttackPlayerVsMob(user *users.UserRecord, mob *mobs.Mob) AttackResult {

	attackResult := calculateCombat(*user.Character, mob.Character, User, Mob)

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
		// Track dual-wield skill if using two weapons
		if user.Character.Equipment.Weapon.ItemId > 0 && user.Character.Equipment.Offhand.ItemId > 0 && user.Character.Equipment.Offhand.GetSpec().Type == items.Weapon {
			user.Character.OnSkillUse(string(skills.DualWield), user.UserId)
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

// Performs a combat round from a player to a player
func AttackPlayerVsPlayer(userAtk *users.UserRecord, userDef *users.UserRecord) AttackResult {

	attackResult := calculateCombat(*userAtk.Character, *userDef.Character, User, User)

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
		// Track dual-wield skill if using two weapons
		if userAtk.Character.Equipment.Weapon.ItemId > 0 && userAtk.Character.Equipment.Offhand.ItemId > 0 && userAtk.Character.Equipment.Offhand.GetSpec().Type == items.Weapon {
			userAtk.Character.OnSkillUse(string(skills.DualWield), userAtk.UserId)
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

// Performs a combat round from a mob to a player
func AttackMobVsPlayer(mob *mobs.Mob, user *users.UserRecord) AttackResult {

	attackResult := calculateCombat(mob.Character, *user.Character, Mob, User)

	// Deduct stamina for the attack
	mob.Character.DeductAttackStamina()

	mob.Character.ApplyHealthChange(attackResult.DamageToSource * -1)

	if attackResult.DamageToTarget != 0 {
		user.Character.ApplyHealthChange(attackResult.DamageToTarget * -1)
		user.WimpyCheck()

		// Check for low-health progression trigger
		if user.Character.HealthMax.Value > 0 {
			healthPct := float64(user.Character.Health) / float64(user.Character.HealthMax.Value)
			if healthPct > 0 && healthPct < 0.25 {
				user.Character.OnLowResource("health", "vitality", user.UserId)
			}
		}

		// Check for low-stamina progression trigger
		if user.Character.StaminaMax.Value > 0 {
			staminaPct := float64(user.Character.Stamina) / float64(user.Character.StaminaMax.Value)
			if staminaPct > 0 && staminaPct < 0.25 {
				user.Character.OnLowResource("stamina", "strength", user.UserId)
				user.Character.OnLowResource("stamina", "dexterity", user.UserId)
			}
		}

		// Check for low-conviction progression trigger
		if user.Character.ConvictionMax.Value > 0 {
			convictionPct := float64(user.Character.Conviction) / float64(user.Character.ConvictionMax.Value)
			if convictionPct > 0 && convictionPct < 0.25 {
				user.Character.OnLowResource("conviction", "willpower", user.UserId)
				user.Character.OnLowResource("conviction", "charisma", user.UserId)
			}
		}
	}

	// Track defender's dexterity use (reacting to attacks)
	user.Character.OnStatUse("dexterity", user.UserId)

	if attackResult.Hit {
		user.PlaySound(`hit-self`, `combat`)
	}

	return attackResult
}

// Performs a combat round from a mob to a mob
func AttackMobVsMob(mobAtk *mobs.Mob, mobDef *mobs.Mob) AttackResult {

	attackResult := calculateCombat(mobAtk.Character, mobDef.Character, Mob, User)

	// Deduct stamina for the attack
	mobAtk.Character.DeductAttackStamina()

	mobAtk.Character.ApplyHealthChange(attackResult.DamageToSource * -1)
	mobDef.Character.ApplyHealthChange(attackResult.DamageToTarget * -1)

	// If attacking mob was player charmed, attribute damage done to that player
	if charmedUserId := mobAtk.Character.GetCharmedUserId(); charmedUserId > 0 {
		// Remember who has hit him
		mobDef.Character.TrackPlayerDamage(charmedUserId, attackResult.DamageToTarget)
	}

	return attackResult
}

func GetWaitMessages(stepType items.Intensity, sourceChar *characters.Character, targetChar *characters.Character, sourceType SourceTarget, targetType SourceTarget) AttackResult {

	attackResult := AttackResult{}

	msgs := items.GetPreAttackMessage(sourceChar.Equipment.Weapon.GetSpec().Subtype, stepType)

	var toAttackerMsg, toDefenderMsg, toAttackerRoomMsg, toDefenderRoomMsg items.ItemMessage

	// zero means randomly selected, otherwise use the ItemId to consistently choose a message
	msgSeed := 0
	if configs.GetGamePlayConfig().ConsistentAttackMessages {
		msgSeed = sourceChar.Equipment.Weapon.ItemId
	}

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
	}

	if sourceChar.RoomId == targetChar.RoomId {
		toAttackerMsg = msgs.Together.ToAttacker.Get(msgSeed)
		toDefenderMsg = msgs.Together.ToDefender.Get(msgSeed)
		toAttackerRoomMsg = msgs.Together.ToRoom.Get(msgSeed)
		toDefenderRoomMsg = items.ItemMessage("")

	} else {

		toAttackerMsg = msgs.Separate.ToAttacker.Get(msgSeed)
		toDefenderMsg = msgs.Separate.ToDefender.Get(msgSeed)
		toAttackerRoomMsg = msgs.Separate.ToAttackerRoom.Get(msgSeed)
		toDefenderRoomMsg = msgs.Separate.ToDefenderRoom.Get(msgSeed)

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

func calculateCombat(sourceChar characters.Character, targetChar characters.Character, sourceType SourceTarget, targetType SourceTarget) AttackResult {

	attackResult := AttackResult{}

	// Base attack count from attacker's dexterity only (defender dex affects hit chance, not attack count)
	// Formula: 1 base attack + 1 per 50 dex (so Dex 100 = 3 attacks, Dex 150 = 4 attacks)
	attackCount := 1 + int(sourceChar.Stats.Dexterity.ValueAdj/50)
	if attackCount < 1 {
		attackCount = 1
	}

	// Statmods can add a damage bonus...
	statModDBonus := sourceChar.StatMod(`damage`)
	// Add any additional attacks
	attackCount += sourceChar.StatMod(`attacks`)

	// Apply stamina-based penalties
	if sourceChar.StaminaMax.Value > 0 {
		staminaRatio := float64(sourceChar.Stamina) / float64(sourceChar.StaminaMax.Value)

		// Reduce attack count when stamina is low (< 20%)
		if staminaRatio < 0.2 {
			attackCount = int(math.Ceil(float64(attackCount) * 0.5)) // 50% reduction
			if attackCount < 1 {
				attackCount = 1 // Always at least 1 attack
			}
		}
	}

	// Apply encumbrance penalty to attack count (weight-based)
	carriedWeight := sourceChar.GetCarriedWeight()
	capacity := sourceChar.CarryCapacity()
	if carriedWeight > capacity {
		// Overencumbered: reduce attacks based on how much over capacity
		overAmount := carriedWeight - capacity
		overRatio := overAmount / capacity
		// Penalty scales from 0% at capacity to 50% at 2x capacity
		encumbrancePenalty := math.Min(overRatio * 0.5, 0.5)
		attackCount = int(math.Ceil(float64(attackCount) * (1.0 - encumbrancePenalty)))
		if attackCount < 1 {
			attackCount = 1 // Always at least 1 attack
		}
	}

	for i := 0; i < attackCount; i++ {

		mudlog.Debug(`calculateCombat`, `Atk`, fmt.Sprintf(`%d/%d`, i+1, attackCount), `Source`, fmt.Sprintf(`%s (%s)`, sourceChar.Name, sourceType), `Target`, fmt.Sprintf(`%s (%s)`, targetChar.Name, targetType))

		attackWeapons := []items.Item{}

		dualWieldLevel := sourceChar.GetSkillLevel(skills.DualWield)

		if sourceChar.Equipment.Weapon.ItemId > 0 {
			attackWeapons = append(attackWeapons, sourceChar.Equipment.Weapon)
		}

		if sourceChar.Equipment.Offhand.ItemId > 0 && sourceChar.Equipment.Offhand.GetSpec().Type == items.Weapon {
			attackWeapons = append(attackWeapons, sourceChar.Equipment.Offhand)
		}

		// Put an empty weapon, so basically hands.
		if len(attackWeapons) == 0 {
			attackWeapons = append(attackWeapons, items.Item{
				ItemId: 0,
			})
		}

		// Dual wielding: If two weapons equipped, always allow dual wielding
		// Skill affects offhand attack count (via GetModifiedAttackCount) and hit penalty (below)
		// No need to remove weapons - let skill determine effectiveness

		attackMessagePrefix := ``
		// If they are backstabbing it's a free crit
		if sourceChar.Aggro.Type == characters.BackStab {
			attackResult.Crit = true
			attackMessagePrefix = `<ansi fg="magenta-bold">*[BACKSTAB]*</ansi> `
			// Failover to the default attack
			sourceChar.SetAggro(sourceChar.Aggro.UserId, sourceChar.Aggro.MobInstanceId, characters.DefaultAttack)
		}

		for weaponIdx, weapon := range attackWeapons {

			penalty := 0
			if len(attackWeapons) > 1 {
				// Dual wield hit penalty scales with skill: 50% at skill 0 → 10% at skill 50
				// Natural weapons (claws) ignore penalty
				if sourceChar.Equipment.Weapon.GetSpec().Subtype == items.Claws && sourceChar.Equipment.Offhand.GetSpec().Subtype == items.Claws {
					penalty = 0 // Natural dual wielding has no penalty
				} else {
					penaltyReduction := float64(dualWieldLevel) / 50.0 // 0.0 to 1.0
					penalty = int(50.0 - (penaltyReduction * 40.0))    // 50 → 10
					if penalty < 10 {
						penalty = 10 // Minimum 10% penalty even at max skill
					}
				}
			}

			// Set the default weapon info
			raceInfo := species.GetSpecies(sourceChar.SpeciesId)
			weaponName := raceInfo.UnarmedName
			weaponSubType := items.Generic

			// Get default unarmed distribution damage
			attacks, baseDmg, dmgVariance, critBuffs := sourceChar.GetDefaultDistributionDamage()

			// Determine if this is the offhand weapon
			isOffhand := weaponIdx > 0 && weapon.ItemId == sourceChar.Equipment.Offhand.ItemId

			weaponSpeed := 1.0 // Unarmed baseline

			if weapon.ItemId > 0 {

				itemSpec := weapon.GetSpec()

				weaponName = weapon.DisplayName()

				weaponSubType = itemSpec.Subtype
				attacks, baseDmg, dmgVariance, critBuffs = weapon.GetDistributionDamage()

				// Get weapon speed multiplier
				weaponSpeed = itemSpec.GetSpeedMultiplier()

				// If there is a bonus vs. a specific race, apply it
				baseDmg += float64(weapon.StatMod(string(statmods.RacialBonusPrefix) + strings.ToLower(targetChar.Species())))
			}

			// Apply speed multiplier, skill modifiers, and dual wielding bonuses
			attacks = sourceChar.GetModifiedAttackCount(attacks, weaponSpeed, isOffhand)

			// Add damage bonus due to statmods
			baseDmg += float64(statModDBonus)

			// Percentage-based strength scaling: Str/100 as multiplier
			strMultiplier := 1.0 + float64(sourceChar.Stats.Strength.ValueAdj)/100.0
			dmgMean := baseDmg * strMultiplier

			// Apply stamina-based damage penalty
			if sourceChar.StaminaMax.Value > 0 {
				staminaRatio := float64(sourceChar.Stamina) / float64(sourceChar.StaminaMax.Value)
				if staminaRatio < 0.25 {
					// 25% damage reduction when very low on stamina
					dmgMean *= 0.75
				}
			}

			// zero means randomly selected, otherwise use the ItemId to consistently choose a message
			msgSeed := 0
			if configs.GetGamePlayConfig().ConsistentAttackMessages {
				msgSeed = weapon.ItemId
			}

			mudlog.Debug("DistDamage", "attacks", attacks, "baseDmg", baseDmg, "variance", dmgVariance, "strMult", strMultiplier, "dmgMean", dmgMean, "critBuffs", critBuffs)

			// Individual weapons may get multiple attacks
			for j := 0; j < attacks; j++ {

				attackTargetDamage := 0
				attackTargetReduction := 0

				attackSourceDamage := 0
				attackSourceReduction := 0

				// Stage 7.1: Layered Defense System
				// Calculate attack score with penalties
				attackScore := float64(sourceChar.Stats.Dexterity.ValueAdj) + float64(sourceChar.GetCombatSkillLevel())
				attackScore -= float64(penalty) // dual wield penalty

				// Apply stamina-based hit chance penalty
				if sourceChar.StaminaMax.Value > 0 {
					staminaRatio := float64(sourceChar.Stamina) / float64(sourceChar.StaminaMax.Value)
					if staminaRatio < 0.5 {
						// Penalty scales from 0 at 50% stamina to 30 at 0% stamina
						staminaPenalty := (0.5 - staminaRatio) * 60.0
						attackScore -= staminaPenalty
					}
				}

				// Get defender's defense sequence based on equipment
				defenseSequence := targetChar.GetDefenseSequence()
				combatStdDev := 15.0
				hit := false
				var lastHitRoll dice.RollResult

				// Try each defense in sequence
				for _, defenseType := range defenseSequence {
					// Track defense attempt
					attackResult.DefenseAttempts = append(attackResult.DefenseAttempts, DefenseType(defenseType))

					// Check if defender has stamina for this defense
					if !targetChar.DeductDefenseStamina(defenseType) {
						// Insufficient stamina, skip this defense
						continue
					}

					// Calculate defense score for this defense type
					defenseScore := targetChar.GetDefenseScore(defenseType)

					// Opposed roll: attack vs this defense
					defenseSucceeded, _, hitRoll, _ := dice.OpposedRoll(attackScore, defenseScore, combatStdDev)
					lastHitRoll = hitRoll

					if !defenseSucceeded {
						// Defense failed, continue to next defense
						continue
					}

					// Defense succeeded! Attack is avoided
					attackResult.DefenseUsed = DefenseType(defenseType)
					hit = false

					// Add defense success messages (Stage 7.1)
					var defenseVerb string
					var skillToProgress string
					switch defenseType {
					case characters.DefenseDodge:
						defenseVerb = "dodge"
						skillToProgress = string(skills.UnarmedCombat)
					case characters.DefenseParry:
						defenseVerb = "parry"
						skillToProgress = string(skills.WeaponCombat)
					case characters.DefenseBlock:
						defenseVerb = "block"
						skillToProgress = string(skills.WeaponCombat)
					}

					// Trigger skill progression for successful defense
					targetChar.TrackSkillUse(skillToProgress)
					targetChar.CheckSkillProgression(skillToProgress, targetChar.GetUserId(), 1.0)

					attackResult.SendToSource(fmt.Sprintf(`<ansi fg="attack-bad">%s %ss your attack!</ansi>`, targetChar.Name, defenseVerb))
					attackResult.SendToTarget(fmt.Sprintf(`<ansi fg="defense-good">You %s %s's attack!</ansi>`, defenseVerb, sourceChar.Name))
					attackResult.SendToSourceRoom(fmt.Sprintf(`<ansi fg="combat">%s %ss %s's attack.</ansi>`, targetChar.Name, defenseVerb, sourceChar.Name))
					if sourceChar.RoomId != targetChar.RoomId {
						attackResult.SendToTargetRoom(fmt.Sprintf(`<ansi fg="combat">%s %ss an attack.</ansi>`, targetChar.Name, defenseVerb))
					}

					break
				}

				// If no defense succeeded (or insufficient stamina for all), attack hits
				if attackResult.DefenseUsed == DefenseNone {
					hit = true
				}

				// Dynamic threshold for crit/fumble detection
				critThreshold := 2.0 // ~2.5% chance
				if sourceChar.HasBuffFlag(buffs.Accuracy) {
					critThreshold = 1.5 // ~6.7% with Accuracy buff
				}
				if targetChar.HasBuffFlag(buffs.Blink) {
					critThreshold = 2.5 // ~0.6% against Blink
				}
				// Skill advantage shifts crit threshold
				skillDiff := sourceChar.GetCombatSkillLevel() - targetChar.GetCombatSkillLevel()
				critThreshold -= float64(skillDiff) * 0.05
				fumbleThreshold := -critThreshold // Mirror the crit threshold

				if hit {
					attackResult.Hit = true

					// Distribution-based damage
					damageResult := dice.Roll(dmgMean, dmgVariance)
					attackTargetDamage = int(math.Round(math.Max(0, damageResult.Value)))

					if lastHitRoll.ZScore >= critThreshold || attackResult.Crit {
						attackResult.Crit = true
						attackResult.BuffTarget = critBuffs
						attackTargetDamage += int(math.Round(dmgMean))
						mudlog.Debug("CritDetected", "zScore", fmt.Sprintf("%.2f", lastHitRoll.ZScore), "threshold", fmt.Sprintf("%.2f", critThreshold), "source", sourceChar.Name, "target", targetChar.Name)
					}
				} else {
					// Fumble detection on miss
					if lastHitRoll.ZScore <= fumbleThreshold {
						attackResult.Fumble = true
						mudlog.Debug("FumbleDetected", "zScore", fmt.Sprintf("%.2f", lastHitRoll.ZScore), "threshold", fmt.Sprintf("%.2f", fumbleThreshold), "source", sourceChar.Name, "target", targetChar.Name)
					}
				}

				// Stage 7.1: Passive defense removed - defense is now active via stamina-costing dodge/parry/block

				// Calculate actual damage vs. expected damage pct
				pctDamage := 0.0
				if dmgMean > 0 {
					pctDamage = math.Ceil(float64(attackTargetDamage) / dmgMean * 100)
				}

				// Use fumble messages when a fumble is detected
				var msgs items.AttackOptions
				if attackResult.Fumble {
					msgs = items.GetPreAttackMessage(weaponSubType, items.Fumble)
				} else {
					msgs = items.GetAttackMessage(weaponSubType, int(pctDamage))
				}

				var toAttackerMsg, toDefenderMsg, toAttackerRoomMsg, toDefenderRoomMsg items.ItemMessage

				tokenReplacements := map[items.TokenName]string{
					items.TokenItemName:     weaponName,
					items.TokenSource:       sourceChar.Name,
					items.TokenSourceType:   string(sourceType) + `name`,
					items.TokenTarget:       targetChar.Name,
					items.TokenTargetType:   string(targetType) + `name`,
					items.TokenUsesLeft:     `[Invalid]`,
					items.TokenDamage:       strconv.Itoa(attackTargetDamage),
					items.TokenEntranceName: `unknown`,
					items.TokenExitName:     `unknown`,
				}

				if sourceChar.RoomId == targetChar.RoomId {

					toAttackerMsg = msgs.Together.ToAttacker.Get(msgSeed)
					toDefenderMsg = msgs.Together.ToDefender.Get(msgSeed)
					toAttackerRoomMsg = msgs.Together.ToRoom.Get(msgSeed)
					toDefenderRoomMsg = items.ItemMessage("")

				} else {

					toAttackerMsg = msgs.Separate.ToAttacker.Get(msgSeed)
					toDefenderMsg = msgs.Separate.ToDefender.Get(msgSeed)
					toAttackerRoomMsg = msgs.Separate.ToAttackerRoom.Get(msgSeed)
					toDefenderRoomMsg = msgs.Separate.ToDefenderRoom.Get(msgSeed)

					// Find the exit that leads to the target from the source (if any)
					if atkRoom := rooms.LoadRoom(sourceChar.RoomId); atkRoom != nil {
						for exitName, exit := range atkRoom.Exits {
							if exit.RoomId == targetChar.RoomId {
								tokenReplacements[items.TokenExitName] = exitName
								break
							}
						}
					}
					// find the exit that leads to the source from the target (if any)
					if defRoom := rooms.LoadRoom(targetChar.RoomId); defRoom != nil {
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

				if attackResult.Crit {
					toAttackerMsg = items.ItemMessage(`<ansi fg="yellow-bold">***</ansi> ` + string(toAttackerMsg) + ` <ansi fg="yellow-bold">***</ansi>`)
					toDefenderMsg = items.ItemMessage(`<ansi fg="yellow-bold">***</ansi> ` + string(toDefenderMsg) + ` <ansi fg="yellow-bold">***</ansi>`)
					toAttackerRoomMsg = items.ItemMessage(`<ansi fg="yellow-bold">***</ansi> ` + string(toAttackerRoomMsg) + ` <ansi fg="yellow-bold">***</ansi>`)
					if len(string(toDefenderRoomMsg)) > 0 {
						toDefenderRoomMsg = items.ItemMessage(`<ansi fg="yellow-bold">***</ansi> ` + string(toDefenderRoomMsg) + ` <ansi fg="yellow-bold">***</ansi>`)
					}
				}

				if attackResult.Fumble {
					toAttackerMsg = items.ItemMessage(`<ansi fg="red-bold">!!!</ansi> ` + string(toAttackerMsg) + ` <ansi fg="red-bold">!!!</ansi>`)
					toDefenderMsg = items.ItemMessage(`<ansi fg="red-bold">!!!</ansi> ` + string(toDefenderMsg) + ` <ansi fg="red-bold">!!!</ansi>`)
					toAttackerRoomMsg = items.ItemMessage(`<ansi fg="red-bold">!!!</ansi> ` + string(toAttackerRoomMsg) + ` <ansi fg="red-bold">!!!</ansi>`)
					if len(string(toDefenderRoomMsg)) > 0 {
						toDefenderRoomMsg = items.ItemMessage(`<ansi fg="red-bold">!!!</ansi> ` + string(toDefenderRoomMsg) + ` <ansi fg="red-bold">!!!</ansi>`)
					}
				}

				if len(attackMessagePrefix) > 0 {
					toAttackerMsg = items.ItemMessage(attackMessagePrefix + string(toAttackerMsg))
					toDefenderMsg = items.ItemMessage(attackMessagePrefix + string(toDefenderMsg))
					toAttackerRoomMsg = items.ItemMessage(attackMessagePrefix + string(toAttackerRoomMsg))
					if len(string(toDefenderRoomMsg)) > 0 {
						toDefenderRoomMsg = items.ItemMessage(attackMessagePrefix + string(toDefenderRoomMsg))
					}
				}

				// Send to attacker
				attackerMsg := string(toAttackerMsg)
				if attackSourceDamage > 0 && attackSourceReduction > 0 {
					attackerMsg += fmt.Sprintf(` <ansi fg="white">[%d was blocked]</ansi>`, attackSourceReduction)
				}

				attackResult.SendToSource(
					string(attackerMsg),
				)

				// Send to victim
				defenderMsg := string(toDefenderMsg)
				if attackTargetDamage > 0 && attackTargetReduction > 0 {
					defenderMsg += fmt.Sprintf(` <ansi fg="red">[you blocked %d]</ansi>`, attackTargetReduction)
				}

				attackResult.SendToTarget(
					string(defenderMsg),
				)

				// Send to room
				attackResult.SendToSourceRoom(
					string(toAttackerRoomMsg.SetTokenValue(items.TokenTarget, targetChar.Name).
						SetTokenValue(items.TokenTargetType, string(targetType))),
				)

				// Send to defender room if separate
				if len(string(toDefenderRoomMsg)) > 0 {
					attackResult.SendToTargetRoom(
						string(toDefenderRoomMsg.SetTokenValue(items.TokenTarget, targetChar.Name).SetTokenValue(items.TokenTargetType, string(targetType))),
					)
				}

				attackResult.DamageToTarget += attackTargetDamage
				attackResult.DamageToTargetReduction += attackTargetReduction

				attackResult.DamageToSource += attackSourceDamage
				attackResult.DamageToSourceReduction += attackSourceReduction
			}

			if petJoins, _ := dice.Percentile(20); petJoins { // 20% chance to join
				if sourceChar.RoomId == targetChar.RoomId {
					if sourceChar.Pet.Exists() && (sourceChar.Pet.Damage.BaseDamage > 0 || sourceChar.Pet.Damage.DiceRoll != ``) {

						petDmg := sourceChar.Pet.Damage
						var petAttacks int
						var petBaseDmg, petVar float64
						if petDmg.BaseDamage > 0 {
							petAttacks = petDmg.Attacks
							if petAttacks < 1 {
								petAttacks = 1
							}
							petBaseDmg = float64(petDmg.BaseDamage)
							petVar = float64(petDmg.Variance)
						} else {
							petAttacks, _, _, _, _ = sourceChar.Pet.GetDiceRoll()
							petBaseDmg, petVar = dice.DiceToDistribution(petDmg.DiceCount, petDmg.SideCount, petDmg.BonusDamage)
						}

						for i := 0; i < petAttacks; i++ {

							attackTargetDamage := int(math.Round(math.Max(0, dice.Roll(petBaseDmg, petVar).Value)))

							attackResult.DamageToTarget += attackTargetDamage

							toAttackerMsg := fmt.Sprintf(`%s jumps into the fray and deals <ansi fg="damage">%d damage</ansi> to <ansi fg="%sname">%s</ansi>!`, sourceChar.Pet.DisplayName(), attackTargetDamage, string(targetType), targetChar.Name)
							attackResult.SendToSource(toAttackerMsg)

							toDefenderMsg := fmt.Sprintf(`%s jumps into the fray and deals <ansi fg="damage">%d damage</ansi> to you!`, sourceChar.Pet.DisplayName(), attackTargetDamage)
							attackResult.SendToTarget(toDefenderMsg)

							toAttackerRoomMsg := fmt.Sprintf(`%s jumps into the fray and deals <ansi fg="damage">%d damage</ansi> to <ansi fg="%sname">%s</ansi>!`, sourceChar.Pet.DisplayName(), attackTargetDamage, string(targetType), targetChar.Name)
							attackResult.SendToTargetRoom(toAttackerRoomMsg)

						}

					}
				}
			}

		}
	}
	return attackResult

}
