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
)

func Grapple(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Must be in combat to use grapple
	if user.Character.Aggro == nil {
		user.SendText("You must be in combat to grapple!")
		return true, nil
	}

	// Check shared special move cooldown (same as bash/trip/kick)
	cfg := configs.GetGamePlayConfig()
	if !user.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		user.SendText("You need a moment to recover before attempting another special move.")
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

		// Stage 8.4: Check for grapple crit disarm
		// Only while in Clinched or Grounded position AND z > 2.0

		if result.AttackZScore > 2.0 &&
			(result.NewPosition.String() == "clinched" || result.NewPosition.String() == "grounded") {

			// 15% chance to trigger disarm
			var disarmResult combat.DisarmResult
			if targetMob != nil {
				disarmResult = combat.AttemptCritDisarm(user.Character, &targetMob.Character, 15.0)
			} else {
				disarmResult = combat.AttemptCritDisarm(user.Character, targetChar.Character, 15.0)
			}

			if disarmResult.Success {
				// Drop weapon to room
				room.AddItem(disarmResult.Weapon, false)

				// Send messages
				user.SendText(disarmResult.Message)
				if targetChar != nil {
					targetChar.SendText(disarmResult.TargetMsg)
				}
				room.SendText(disarmResult.RoomMessage, user.UserId, targetPlayerId)
			}
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

		// Stage 8.6: Failed grapple penalties

		// Simple failure (z < 0.5): Defense penalty
		if result.AttackZScore < 0.5 {
			user.Character.AddCondition(characters.ConditionDefensePenalty, 1, 0.85, "failed grapple")
			user.SendText(`<ansi fg="red">Your failed attempt leaves you exposed!</ansi>`)
		}

		// Critical failure (z < -2.0): Fall prone + reversal opportunity
		if result.AttackZScore < -2.0 {
			var critResult combat.CritFailureResult
			if targetMob != nil {
				critResult = combat.HandleGrappleCritFailure(user.Character, &targetMob.Character)
			} else {
				critResult = combat.HandleGrappleCritFailure(user.Character, targetChar.Character)
			}
			user.SendText(critResult.Message)
			if targetChar != nil {
				targetChar.SendText(critResult.TargetMessage)
			}
			room.SendText(critResult.RoomMessage, user.UserId, targetPlayerId)
		}
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
