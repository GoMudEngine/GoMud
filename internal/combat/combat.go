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

	attackCount := int(math.Ceil(float64(sourceChar.Stats.Dexterity.ValueAdj-targetChar.Stats.Dexterity.ValueAdj) / 25))
	if attackCount < 1 {
		attackCount = 1
	}

	// Statmods can add a damage bonus...
	statModDBonus := sourceChar.StatMod(`damage`)
	// Add any additional attacks
	attackCount += sourceChar.StatMod(`attacks`)

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

		if len(attackWeapons) > 1 {

			maxWeapons := 1
			if dualWieldLevel == 1 {
				maxWeapons = 1
			}

			if dualWieldLevel == 2 {
				if success, _ := dice.Percentile(50); success {
					maxWeapons = 2
				}
			}

			if dualWieldLevel >= 3 {
				maxWeapons = 2
			}

			// If two martial weapons are equipped, allow dual wielding even without the stat.
			if sourceChar.Equipment.Weapon.GetSpec().Subtype == items.Claws && sourceChar.Equipment.Offhand.GetSpec().Subtype == items.Claws {
				maxWeapons = 2
			}

			for len(attackWeapons) > maxWeapons {
				// Remove a random position
				rnd := dice.RollBetweenInt(0, len(attackWeapons)-1)
				attackWeapons = append(attackWeapons[:rnd], attackWeapons[rnd+1:]...)
			}

		}

		attackMessagePrefix := ``
		// If they are backstabbing it's a free crit
		if sourceChar.Aggro.Type == characters.BackStab {
			attackResult.Crit = true
			attackMessagePrefix = `<ansi fg="magenta-bold">*[BACKSTAB]*</ansi> `
			// Failover to the default attack
			sourceChar.SetAggro(sourceChar.Aggro.UserId, sourceChar.Aggro.MobInstanceId, characters.DefaultAttack)
		}

		for _, weapon := range attackWeapons {

			penalty := 0
			if len(attackWeapons) > 1 {
				if dualWieldLevel < 4 {
					penalty = 35 //35% penalty to hit
				} else {
					penalty = 25 //25% penalty to hit
				}
			}

			// Set the default weapon info
			raceInfo := species.GetSpecies(sourceChar.SpeciesId)
			weaponName := raceInfo.UnarmedName
			weaponSubType := items.Generic

			// Get default unarmed distribution damage
			attacks, baseDmg, dmgVariance, critBuffs := sourceChar.GetDefaultDistributionDamage()

			if weapon.ItemId > 0 {

				itemSpec := weapon.GetSpec()

				weaponName = weapon.DisplayName()

				weaponSubType = itemSpec.Subtype
				attacks, baseDmg, dmgVariance, critBuffs = weapon.GetDistributionDamage()

				// If there is a bonus vs. a specific race, apply it
				baseDmg += float64(weapon.StatMod(string(statmods.RacialBonusPrefix) + strings.ToLower(targetChar.Species())))
			}

			// Add damage bonus due to statmods
			baseDmg += float64(statModDBonus)

			// Percentage-based strength scaling: Str/100 as multiplier
			strMultiplier := 1.0 + float64(sourceChar.Stats.Strength.ValueAdj)/100.0
			dmgMean := baseDmg * strMultiplier

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

				// Opposed roll: attacker dex+skill vs defender dex+skill
				attackScore := float64(sourceChar.Stats.Dexterity.ValueAdj) + float64(sourceChar.GetCombatSkillLevel())
				defendScore := float64(targetChar.Stats.Dexterity.ValueAdj) + float64(targetChar.GetCombatSkillLevel())
				attackScore -= float64(penalty) // dual wield penalty

				combatStdDev := 15.0
				hit, _, hitRoll, _ := dice.OpposedRoll(attackScore, defendScore, combatStdDev)

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

					if hitRoll.ZScore >= critThreshold || attackResult.Crit {
						attackResult.Crit = true
						attackResult.BuffTarget = critBuffs
						attackTargetDamage += int(math.Round(dmgMean))
						mudlog.Debug("CritDetected", "zScore", fmt.Sprintf("%.2f", hitRoll.ZScore), "threshold", fmt.Sprintf("%.2f", critThreshold), "source", sourceChar.Name, "target", targetChar.Name)
					}
				} else {
					// Fumble detection on miss
					if hitRoll.ZScore <= fumbleThreshold {
						attackResult.Fumble = true
						mudlog.Debug("FumbleDetected", "zScore", fmt.Sprintf("%.2f", hitRoll.ZScore), "threshold", fmt.Sprintf("%.2f", fumbleThreshold), "source", sourceChar.Name, "target", targetChar.Name)
					}
				}

				defenseAmt := dice.RollBetweenInt(0, targetChar.GetDefense())
				if defenseAmt > 0 {
					attackTargetReduction = int(math.Round((float64(defenseAmt) / 100) * float64(attackTargetDamage)))
					attackTargetDamage -= attackTargetReduction
				}

				defenseAmt = dice.RollBetweenInt(0, sourceChar.GetDefense())
				if defenseAmt > 0 {
					attackSourceReduction = int(math.Round((float64(defenseAmt) / 100) * float64(attackSourceDamage)))
					attackSourceDamage -= attackSourceReduction
				}

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
