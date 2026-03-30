package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Grapple(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Must be in combat or specify a target to use grapple
	if user.Character.Aggro == nil {
		if rest == "" {
			user.SendText("Grapple whom?")
			return true, nil
		}
		targetPId, targetMId := room.FindByName(rest)
		if targetPId == user.UserId {
			user.SendText("You can't grapple yourself.")
			return true, nil
		}
		if targetPId == 0 && targetMId == 0 {
			user.SendText("You don't see them here.")
			return true, nil
		}
		if targetMId > 0 {
			user.Character.SetAggro(0, targetMId, characters.DefaultAttack)
		} else {
			user.Character.SetAggro(targetPId, 0, characters.DefaultAttack)
		}
	}

	// Check shared special move cooldown
	cfg := configs.GetBalanceConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	}

	// Resolve target
	targetPlayerId := user.Character.Aggro.UserId
	targetMobId := user.Character.Aggro.MobInstanceId

	var targetChar *users.UserRecord
	var targetMob *mobs.Mob
	var targetName string
	var defender *characters.Character

	if targetMobId > 0 {
		targetMob = mobs.GetInstance(targetMobId)
		if targetMob == nil {
			user.SendText("Your target is gone!")
			return true, nil
		}
		targetName = targetMob.Character.Name
		defender = &targetMob.Character
	} else if targetPlayerId > 0 {
		targetChar = users.GetByUserId(targetPlayerId)
		if targetChar == nil {
			user.SendText("Your target is gone!")
			return true, nil
		}
		targetName = targetChar.Character.Name
		defender = targetChar.Character
	} else {
		user.SendText("You have no target!")
		return true, nil
	}

	// Execute grapple move
	result := combat.ExecuteGrappleMove(user.Character, defender, user.UserId, room)

	// Send messages based on result
	if result.Success {
		user.SendText(fmt.Sprintf(`You <ansi fg="yellow-bold">grapple</ansi> <ansi fg="mobname">%s</ansi>, transitioning to <ansi fg="cyan">%s</ansi> position!`, targetName, result.PositionDesc))
		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> you, transitioning to <ansi fg="cyan">%s</ansi> position!`, user.Character.Name, result.PositionDesc))
		}
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> <ansi fg="mobname">%s</ansi> into <ansi fg="cyan">%s</ansi> position!`, user.Character.Name, targetName, result.PositionDesc),
			user.UserId, targetPlayerId,
		)

		// Flavor text for prone targets
		if result.PositionPenalty < 0 {
			user.SendText(fmt.Sprintf(`<ansi fg="yellow">%s was already prone - they had little chance to resist!</ansi>`, targetName))
		}

		// Disarm messaging
		if result.DisarmResult != nil {
			user.SendText(result.DisarmResult.Message)
			if targetChar != nil {
				targetChar.SendText(result.DisarmResult.TargetMsg)
			}
			room.SendText(result.DisarmResult.RoomMessage, user.UserId, targetPlayerId)
		}
	} else {
		user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">grapple</ansi> attempt against <ansi fg="mobname">%s</ansi> fails!`, targetName))
		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> tries to grapple you, but you slip away!`, user.Character.Name))
		}
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> tries to grapple <ansi fg="mobname">%s</ansi>, but fails!`, user.Character.Name, targetName),
			user.UserId, targetPlayerId,
		)

		// Defense penalty messaging
		if result.DefensePenalty {
			user.SendText(`<ansi fg="red">Your failed attempt leaves you exposed!</ansi>`)
		}

		// Critical failure messaging
		if result.CritFailure != nil {
			user.SendText(result.CritFailure.Message)
			if targetChar != nil {
				targetChar.SendText(result.CritFailure.TargetMessage)
			}
			room.SendText(result.CritFailure.RoomMessage, user.UserId, targetPlayerId)
		}
	}

	// Record combat analytics
	tgtType := combat.Mob
	if targetMob == nil {
		tgtType = combat.User
	}
	combat.RecordSpecialMove(combat.User, tgtType, "grapple", result.Success, 0, user.Character, defender, util.GetRoundCount())

	// Progress unarmed combat skill
	events.AddToQueue(events.SkillUsed{
		UserId:  user.UserId,
		Skill:   skills.UnarmedCombat,
		Details: "grapple",
	})

	// Grapple costs the current combat round
	if user.Character.Aggro != nil {
		user.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
