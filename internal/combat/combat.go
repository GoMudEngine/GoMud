package combat

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
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

	if attackResult.Hit {
		user.PlaySound(`hit-other`, `combat`)
	} else {
		user.PlaySound(`miss`, `combat`)
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

	if attackResult.Hit {
		userAtk.PlaySound(`hit-other`, `combat`)
		userDef.PlaySound(`hit-self`, `combat`)
	} else {
		userAtk.PlaySound(`miss`, `combat`)
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
	}

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

				roll := util.Rand(100)

				util.LogRoll(`Both Weapons`, roll, 50)

				if roll < 50 {
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
				rnd := util.Rand(len(attackWeapons))
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

			// Get default racial dice rolls
			attacks, dCount, dSides, dBonus, critBuffs := sourceChar.GetDefaultDiceRoll()

			// Add damage bonus due to statmods
			dBonus += statModDBonus

			if weapon.ItemId > 0 {

				itemSpec := weapon.GetSpec()

				weaponName = weapon.DisplayName()

				weaponSubType = itemSpec.Subtype
				attacks, dCount, dSides, dBonus, critBuffs = weapon.GetDiceRoll()

				// If there is a bonus vs. a specific race, apply it
				dBonus += weapon.StatMod(string(statmods.RacialBonusPrefix) + strings.ToLower(targetChar.Species()))
			}

			// zero means randomly selected, otherwise use the ItemId to consistently choose a message
			msgSeed := 0
			if configs.GetGamePlayConfig().ConsistentAttackMessages {
				msgSeed = weapon.ItemId
			}

			mudlog.Debug("DiceRolls", "attacks", attacks, "dCount", dCount, "dSides", dSides, "dBonus", dBonus, "critBuffs", critBuffs)

			// Individual weapons may get multiple attacks
			for j := 0; j < attacks; j++ {

				attackTargetDamage := 0
				attackTargetReduction := 0

				attackSourceDamage := 0
				attackSourceReduction := 0

				if Hits(sourceChar.Stats.Dexterity.ValueAdj, targetChar.Stats.Dexterity.ValueAdj, penalty) {
					attackResult.Hit = true
					attackTargetDamage = util.RollDice(dCount, dSides) + dBonus

					if attackResult.Crit || Crits(sourceChar, targetChar) {
						attackResult.Crit = true
						attackResult.BuffTarget = critBuffs
						attackTargetDamage += dCount*dSides + dBonus
					}
				}

				defenseAmt := util.Rand(targetChar.GetDefense())
				if defenseAmt > 0 {
					attackTargetReduction = int(math.Round((float64(defenseAmt) / 100) * float64(attackTargetDamage)))
					attackTargetDamage -= attackTargetReduction
				}

				defenseAmt = util.Rand(sourceChar.GetDefense())
				if defenseAmt > 0 {
					attackSourceReduction = int(math.Round((float64(defenseAmt) / 100) * float64(attackSourceDamage)))
					attackSourceDamage -= attackSourceReduction
				}

				// Calculate actual damage vs. possible damage pct
				pctDamage := math.Ceil(float64(attackTargetDamage) / float64(dCount*dSides+dBonus) * 100)

				msgs := items.GetAttackMessage(weaponSubType, int(pctDamage))

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

			if util.RollDice(1, 5) == 1 { // 20% chance to join
				if sourceChar.RoomId == targetChar.RoomId {
					if sourceChar.Pet.Exists() && sourceChar.Pet.Damage.DiceRoll != `` {

						attacks, dCount, dSides, dBonus, critBuffs = sourceChar.Pet.GetDiceRoll()

						for i := 0; i < attacks; i++ {

							attackTargetDamage := util.RollDice(dCount, dSides) + dBonus

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

// hit chance will be between 30 and 100
func hitChance(attackSpd, defendSpd int) int {
	atkPlusDef := float64(attackSpd + defendSpd)
	if atkPlusDef < 1 {
		atkPlusDef = 1
	}
	return 30 + int(float64(attackSpd)/atkPlusDef*70)
}

// Chance to hit
func Hits(attackSpd, defendSpd, hitModifier int) bool {
	// Attack speeds affect 90% of the hit chance
	toHit := hitChance(attackSpd, defendSpd)
	if hitModifier != 0 {
		toHit += hitModifier
	}

	// Always at leat a 5% chance
	if toHit < 5 {
		toHit = 5
	}

	// Always at most a 95% chance
	if toHit > 95 {
		toHit = 95
	}
	hitRoll := util.Rand(100)

	util.LogRoll(`Hits`, hitRoll, toHit)

	return hitRoll < toHit
}

// Whether they crit
func Crits(sourceChar characters.Character, targetChar characters.Character) bool {

	levelDiff := sourceChar.Level - targetChar.Level
	if levelDiff < 1 {
		levelDiff = 1
	}
	critChance := 5 + int(math.Round(float64(sourceChar.Stats.Strength.ValueAdj+sourceChar.Stats.Dexterity.ValueAdj)/float64(levelDiff)))

	if sourceChar.HasBuffFlag(buffs.Accuracy) {
		critChance *= 2
	}

	if targetChar.HasBuffFlag(buffs.Blink) {
		critChance /= 2
	}

	// Minimum 5% chance
	if critChance < 5 {
		critChance = 5
	}

	critRoll := util.Rand(100)

	util.LogRoll(`Crits`, critRoll, critChance)

	return critRoll < critChance
}
