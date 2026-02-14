package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Grapple(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Must be in combat to use grapple
	if user.Character.Aggro == nil {
		user.SendText("You must be in combat to grapple!")
		return true, nil
	}

	// Check grapple cooldown (5 rounds)
	if !user.Character.Cooldowns.Try("grapple", "5 rounds") {
		remaining := user.Character.Cooldowns["grapple"]
		roundWord := "round"
		if remaining > 1 {
			roundWord = "rounds"
		}
		user.SendText(fmt.Sprintf("You can't grapple again yet! (%d %s remaining)", remaining, roundWord))
		return true, nil
	}

	// Get current target from aggro
	targetPlayerId := user.Character.Aggro.UserId
	targetMobId := user.Character.Aggro.MobInstanceId

	var targetChar *users.UserRecord
	var targetMob *mobs.Mob
	var targetName string

	if targetMobId > 0 {
		targetMob = mobs.GetInstance(targetMobId)
		if targetMob == nil {
			user.SendText("Your target is gone!")
			return true, nil
		}
		targetName = targetMob.Character.Name
	} else if targetPlayerId > 0 {
		targetChar = users.GetByUserId(targetPlayerId)
		if targetChar == nil {
			user.SendText("Your target is gone!")
			return true, nil
		}
		targetName = targetChar.Character.Name
	} else {
		user.SendText("You have no target!")
		return true, nil
	}

	// Attempt the grapple
	var result combat.GrappleResult
	if targetMob != nil {
		result = combat.AttemptGrapple(user.Character, &targetMob.Character)
	} else {
		result = combat.AttemptGrapple(user.Character, targetChar.Character)
	}

	// Apply result and send messages
	if result.Success {
		// Apply the grapple to both characters
		if targetMob != nil {
			combat.ApplyGrappleResult(user.Character, &targetMob.Character, result, user.UserId)
		} else {
			combat.ApplyGrappleResult(user.Character, targetChar.Character, result, user.UserId)
		}

		// Determine position description
		positionDesc := "clinched"
		if result.NewPosition.String() == "grounded" {
			positionDesc = "grounded"
		}

		// Success messages
		user.SendText(fmt.Sprintf(`You <ansi fg="yellow-bold">grapple</ansi> <ansi fg="mobname">%s</ansi>, transitioning to <ansi fg="cyan">%s</ansi> position!`, targetName, positionDesc))

		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> you, transitioning to <ansi fg="cyan">%s</ansi> position!`, user.Character.Name, positionDesc))
		}

		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> <ansi fg="mobname">%s</ansi> into <ansi fg="cyan">%s</ansi> position!`, user.Character.Name, targetName, positionDesc),
			user.UserId, targetPlayerId,
		)

		// Add flavor text for prone targets
		if result.PositionPenalty < 0 {
			user.SendText(fmt.Sprintf(`<ansi fg="yellow">%s was already prone - they had little chance to resist!</ansi>`, targetName))
		}
	} else {
		// Failure messages
		user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">grapple</ansi> attempt against <ansi fg="mobname">%s</ansi> fails!`, targetName))

		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> tries to grapple you, but you slip away!`, user.Character.Name))
		}

		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> tries to grapple <ansi fg="mobname">%s</ansi>, but fails!`, user.Character.Name, targetName),
			user.UserId, targetPlayerId,
		)
	}

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
