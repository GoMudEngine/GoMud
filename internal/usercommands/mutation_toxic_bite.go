package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func ToxicBite(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !mutations.HasMutation(user.Character.Mutations, "toxic-bite") {
		user.SendText("You don't have that ability.")
		return true, nil
	}

	if !user.Character.IsInCombat() {
		user.SendText("You must be in combat to use toxic bite!")
		return true, nil
	}

	cfg := configs.GetBalanceConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	}

	staminaCost := 12
	if user.Character.Stamina < staminaCost {
		user.SendText("You're too exhausted!")
		return true, nil
	}
	user.Character.Stamina -= staminaCost

	targetMobId := user.Character.EngagedTarget().MobInstanceId
	targetPlayerId := user.Character.EngagedTarget().UserId

	var targetName string
	var targetMob *mobs.Mob
	var targetUser *users.UserRecord
	var targetMaxHP int

	if targetMobId > 0 {
		targetMob = mobs.GetInstance(targetMobId)
		if targetMob == nil {
			user.SendText("Your target is gone!")
			return true, nil
		}
		targetName = targetMob.Character.Name
		targetMaxHP = targetMob.Character.HealthMax.Value
	} else if targetPlayerId > 0 {
		targetUser = users.GetByUserId(targetPlayerId)
		if targetUser == nil {
			user.SendText("Your target is gone!")
			return true, nil
		}
		targetName = targetUser.Character.Name
		targetMaxHP = targetUser.Character.HealthMax.Value
	} else {
		user.SendText("You have no target!")
		return true, nil
	}

	attackerScore := float64(user.Character.GetSkillLevel(skills.UnarmedCombat)) + float64(user.Character.Stats.Dexterity.ValueAdj)
	var defenderScore float64
	if targetMob != nil {
		defenderScore = float64(targetMob.Character.Stats.Dexterity.ValueAdj) + float64(targetMob.Character.GetCombatSkillLevel())
	} else {
		defenderScore = float64(targetUser.Character.Stats.Dexterity.ValueAdj) + float64(targetUser.Character.GetCombatSkillLevel())
	}
	attackSuccess, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)

	// Bite damage
	biteDamage := int(float64(user.Character.Stats.Strength.ValueAdj) * 0.06)
	if biteDamage < 1 {
		biteDamage = 1
	}

	if attackSuccess {
		if targetMob != nil {
			targetMob.Character.Health -= biteDamage
			if targetMob.Character.Health < 1 {
				targetMob.Character.Health = 0
			}
		} else {
			targetUser.Character.Health -= biteDamage
			if targetUser.Character.Health < 1 {
				targetUser.Character.Health = 0
			}
		}
		// Apply poison DoT: magnitude is damage per AutoHeal tick
		poisonDmg := float64(user.Character.Stats.Vitality.ValueAdj) * 0.04
		if poisonDmg < 1 {
			poisonDmg = 1
		}
		if targetMob != nil {
			targetMob.Character.AddCondition(characters.ConditionPoisoned, 9, poisonDmg, "toxic-bite")
		} else {
			targetUser.Character.AddCondition(characters.ConditionPoisoned, 9, poisonDmg, "toxic-bite")
		}

		user.SendText(fmt.Sprintf(`<ansi fg="green-bold">You sink your toxic fangs into <ansi fg="mobname">%s</ansi>! Venom courses into the wound. (<ansi fg="damage">%s</ansi>)</ansi>`,
			targetName, combat.GetDamageDescription(biteDamage, targetMaxHP)))
		if targetPlayerId > 0 {
			if tUser := users.GetByUserId(targetPlayerId); tUser != nil {
				tUser.SendText(fmt.Sprintf(`<ansi fg="red"><ansi fg="username">%s</ansi> bites you with venomous fangs! You feel poison spreading! (<ansi fg="damage">%s</ansi>)</ansi>`,
					user.Character.Name, combat.GetDamageDescription(biteDamage, targetMaxHP)))
			}
		}
		room.SendTextVisual(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> bites <ansi fg="mobname">%s</ansi> with venomous fangs!`, user.Character.Name, targetName),
			user.UserId, targetPlayerId,
		)
	} else {
		// Self-damage on miss
		selfDamage := biteDamage / 2
		if selfDamage < 1 {
			selfDamage = 1
		}
		user.Character.Health -= selfDamage
		if user.Character.Health < 1 {
			user.Character.Health = 0
		}
		user.SendText(fmt.Sprintf(`<ansi fg="red">Your toxic bite misses <ansi fg="mobname">%s</ansi> and you bite your own tongue! (<ansi fg="damage">%s</ansi>)</ansi>`,
			targetName, combat.GetDamageDescription(selfDamage, user.Character.HealthMax.Value)))
		if targetPlayerId > 0 {
			if tUser := users.GetByUserId(targetPlayerId); tUser != nil {
				tUser.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> lunges to bite you, but misses!`, user.Character.Name))
			}
		}
		room.SendTextVisual(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> lunges to bite <ansi fg="mobname">%s</ansi>, but misses and bites their own tongue!`, user.Character.Name, targetName),
			user.UserId, targetPlayerId,
		)
	}

	// Stage 30.1: Record combat analytics
	biteTgtType := combat.Mob
	var biteTgtChar *characters.Character
	if targetMob != nil {
		biteTgtChar = &targetMob.Character
	} else {
		biteTgtType = combat.User
		biteTgtChar = targetUser.Character
	}
	biteDmgRecorded := 0
	if attackSuccess {
		biteDmgRecorded = biteDamage
	}
	combat.RecordSpecialMove(combat.User, biteTgtType, "toxic_bite", attackSuccess, biteDmgRecorded, user.Character, biteTgtChar, util.GetRoundCount())

	events.AddToQueue(events.SkillUsed{UserId: user.UserId, Skill: skills.UnarmedCombat, Details: "toxic-bite"})

	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
