package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Grapple(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use grapple
	if mob.Character.Aggro == nil {
		return true, nil
	}

	// Check shared special move cooldown (same as bash/trip/kick)
	cfg := configs.GetGamePlayConfig()
	if !mob.Character.Cooldowns.Try("special-move", fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return true, nil
	}

	// Get current target from aggro
	targetPlayerId := mob.Character.Aggro.UserId
	targetMobId := mob.Character.Aggro.MobInstanceId

	var targetChar *users.UserRecord
	var targetMob *mobs.Mob
	var targetName string

	if targetMobId > 0 {
		targetMob = mobs.GetInstance(targetMobId)
		if targetMob == nil {
			return true, nil
		}
		targetName = targetMob.Character.Name
	} else if targetPlayerId > 0 {
		targetChar = users.GetByUserId(targetPlayerId)
		if targetChar == nil {
			return true, nil
		}
		targetName = targetChar.Character.Name
	} else {
		return true, nil
	}

	// Attempt the grapple
	var result combat.GrappleResult
	if targetMob != nil {
		result = combat.AttemptGrapple(&mob.Character, &targetMob.Character)
	} else {
		result = combat.AttemptGrapple(&mob.Character, targetChar.Character)
	}

	// Apply result and send messages
	if result.Success {
		// Apply the grapple to both characters
		if targetMob != nil {
			combat.ApplyGrappleResult(&mob.Character, &targetMob.Character, result, 0)
		} else {
			combat.ApplyGrappleResult(&mob.Character, targetChar.Character, result, 0)
		}

		// Determine position description
		positionDesc := "clinched"
		if result.NewPosition.String() == "grounded" {
			positionDesc = "grounded"
		}

		// Success messages
		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> you, transitioning to <ansi fg="cyan">%s</ansi> position!`, mob.Character.Name, positionDesc))
		}

		room.SendText(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> <ansi fg="username">%s</ansi> into <ansi fg="cyan">%s</ansi> position!`, mob.Character.Name, targetName, positionDesc),
			targetPlayerId,
		)

		// Check for grapple crit disarm
		if result.AttackZScore > 2.0 &&
			(result.NewPosition.String() == "clinched" || result.NewPosition.String() == "grounded") {

			// 15% chance to trigger disarm
			var disarmResult combat.DisarmResult
			if targetMob != nil {
				disarmResult = combat.AttemptCritDisarm(&mob.Character, &targetMob.Character, 15.0)
			} else {
				disarmResult = combat.AttemptCritDisarm(&mob.Character, targetChar.Character, 15.0)
			}

			if disarmResult.Success {
				// Drop weapon to room
				room.AddItem(disarmResult.Weapon, false)

				// Send messages
				if targetChar != nil {
					targetChar.SendText(disarmResult.TargetMsg)
				}
				room.SendText(disarmResult.RoomMessage, targetPlayerId)
			}
		}
	} else {
		// Failure messages
		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to grapple you, but you slip away!`, mob.Character.Name))
		}

		room.SendText(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to grapple <ansi fg="username">%s</ansi>, but fails!`, mob.Character.Name, targetName),
			targetPlayerId,
		)

		// Failed grapple penalties

		// Simple failure (z < 0.5): Defense penalty
		if result.AttackZScore < 0.5 {
			mob.Character.AddCondition(characters.ConditionDefensePenalty, 1, 0.85, "failed grapple")
		}

		// Critical failure (z < -2.0): Fall prone + reversal opportunity
		if result.AttackZScore < -2.0 {
			var critResult combat.CritFailureResult
			if targetMob != nil {
				critResult = combat.HandleGrappleCritFailure(&mob.Character, &targetMob.Character)
			} else {
				critResult = combat.HandleGrappleCritFailure(&mob.Character, targetChar.Character)
			}
			if targetChar != nil {
				targetChar.SendText(critResult.TargetMessage)
			}
			room.SendText(critResult.RoomMessage, targetPlayerId)
		}
	}

	// Grapple costs the current combat round
	if mob.Character.Aggro != nil {
		mob.Character.Aggro.RoundsWaiting = 1
	}

	return true, nil
}
