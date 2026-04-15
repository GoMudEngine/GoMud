package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
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
			if m := mobs.GetInstance(targetMId); m != nil && m.IsNonCombatant() {
				user.SendText(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, m.Character.Name))
				return true, nil
			}
			user.Character.SetAggro(0, targetMId, characters.DefaultAttack)
		} else {
			if p := users.GetByUserId(targetPId); p != nil {
				if pvpErr := room.CanPvp(user, p); pvpErr != nil {
					user.SendText(pvpErr.Error())
					return true, nil
				}
			}
			user.Character.SetAggro(targetPId, 0, characters.DefaultAttack)
		}
	}

	res := actions.ExecuteGrapple(&actions.UserActor{User: user, Room: room})

	if res.OnCooldown {
		user.SendText("You need a moment to recover before attempting another special move.")
		return true, nil
	}

	if res.NoTarget {
		user.SendText("You have no target!")
		return true, nil
	}

	if res.GrappleImmune {
		user.SendText(fmt.Sprintf("You reach for %s but your hands pass right through!", res.Target.Name))
		return true, nil
	}

	if !res.Executed {
		return true, nil
	}

	target := res.Target
	result := res.MoveResult
	targetName := target.Name
	targetPlayerId := target.UserId

	// Resolve the target user record for direct messaging (player targets).
	var targetChar *users.UserRecord
	if target.UserId > 0 {
		targetChar = users.GetByUserId(target.UserId)
	}

	// Send messages based on result
	if result.Success {
		user.SendText(fmt.Sprintf(`You <ansi fg="yellow-bold">grapple</ansi> <ansi fg="mobname">%s</ansi>, transitioning to <ansi fg="cyan">%s</ansi> position!`, targetName, result.PositionDesc))
		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> you, transitioning to <ansi fg="cyan">%s</ansi> position!`, user.Character.Name, result.PositionDesc))
		}
		room.SendTextVisual(
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
			room.SendTextVisual(result.DisarmResult.RoomMessage, user.UserId, targetPlayerId)
		}
	} else {
		user.SendText(fmt.Sprintf(`Your <ansi fg="yellow-bold">grapple</ansi> attempt against <ansi fg="mobname">%s</ansi> fails!`, targetName))
		if targetChar != nil {
			targetChar.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> tries to grapple you, but you slip away!`, user.Character.Name))
		}
		room.SendTextVisual(
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
			room.SendTextVisual(result.CritFailure.RoomMessage, user.UserId, targetPlayerId)
		}
	}

	return true, nil
}
