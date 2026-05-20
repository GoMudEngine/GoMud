package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func BlindingSpit(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !mutations.HasMutation(user.Character.Mutations, "blinding-spit") {
		user.SendTextLegacy("You don't have that ability.")
		return true, nil
	}

	if !user.Character.IsInCombat() {
		user.SendTextLegacy("You must be in combat to use blinding spit!")
		return true, nil
	}

	cfg := configs.GetBalanceConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		user.SendTextLegacy("You need a moment to recover before attempting another special move.")
		return true, nil
	}

	staminaCost := 10
	if user.Character.Stamina < staminaCost {
		user.SendTextLegacy("You're too exhausted!")
		return true, nil
	}
	user.Character.Stamina -= staminaCost

	targetMobId := user.Character.EngagedTarget().MobInstanceId
	targetPlayerId := user.Character.EngagedTarget().UserId

	var targetName string
	var targetMob *mobs.Mob
	var targetUser *users.UserRecord

	if targetMobId > 0 {
		targetMob = mobs.GetInstance(targetMobId)
		if targetMob == nil {
			user.SendTextLegacy("Your target is gone!")
			return true, nil
		}
		targetName = targetMob.Character.Name
	} else if targetPlayerId > 0 {
		targetUser = users.GetByUserId(targetPlayerId)
		if targetUser == nil {
			user.SendTextLegacy("Your target is gone!")
			return true, nil
		}
		targetName = targetUser.Character.Name
	} else {
		user.SendTextLegacy("You have no target!")
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

	if attackSuccess {
		if targetMob != nil {
			targetMob.Character.AddCondition(characters.ConditionBlinded, 3, 0.5, "blinding-spit")
		} else {
			targetUser.Character.AddCondition(characters.ConditionBlinded, 3, 0.5, "blinding-spit")
		}
		user.SendTextLegacy(fmt.Sprintf(`<ansi fg="yellow-bold">You spit a stream of caustic fluid into <ansi fg="mobname">%s</ansi>'s eyes!</ansi>`, targetName))
		if targetPlayerId > 0 {
			if tUser := users.GetByUserId(targetPlayerId); tUser != nil {
				tUser.SendTextLegacy(fmt.Sprintf(`<ansi fg="red"><ansi fg="username">%s</ansi> spits caustic fluid into your eyes! You can barely see!</ansi>`, user.Character.Name))
			}
		}
		room.SendTextVisualLegacy(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> spits a stream of fluid into <ansi fg="mobname">%s</ansi>'s eyes!`, user.Character.Name, targetName),
			user.UserId, targetPlayerId,
		)
	} else {
		user.SendTextLegacy(fmt.Sprintf(`<ansi fg="red">Your blinding spit misses <ansi fg="mobname">%s</ansi>!</ansi>`, targetName))
		if targetPlayerId > 0 {
			if tUser := users.GetByUserId(targetPlayerId); tUser != nil {
				tUser.SendTextLegacy(fmt.Sprintf(`<ansi fg="username">%s</ansi> spits at you, but you dodge the caustic stream!`, user.Character.Name))
			}
		}
		room.SendTextVisualLegacy(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> spits at <ansi fg="mobname">%s</ansi>, but misses!`, user.Character.Name, targetName),
			user.UserId, targetPlayerId,
		)
	}

	events.AddToQueue(events.SkillUsed{UserId: user.UserId, Skill: skills.UnarmedCombat, Details: "blinding-spit"})

	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
