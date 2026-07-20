package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Grapple(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use grapple
	if !mob.Character.IsInCombat() {
		return true, nil
	}

	res := actions.ExecuteGrapple(&actions.MobActor{Mob: mob, Room: room})

	if res.OnCooldown || res.NoTarget || res.GrappleImmune || !res.Executed {
		return true, nil
	}

	target := res.Target
	result := res.MoveResult
	mobName := mob.Character.Name
	targetName := target.Name
	targetPlayerId := target.UserId

	// Resolve the target user record for direct messaging (player targets).
	var targetChar *users.UserRecord
	if target.UserId > 0 {
		targetChar = users.GetByUserId(target.UserId)
	}

	canSee := targetChar == nil || canSeeInDark(targetChar, room)

	// Send messages based on result
	if result.Success {
		if targetChar != nil {
			if canSee {
				targetChar.SendText(messaging.CategoryGrappleFlow, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> you, transitioning to <ansi fg="cyan">%s</ansi> position!`, mobName, result.PositionDesc))
			} else {
				targetChar.SendText(messaging.CategoryGrappleFlow, fmt.Sprintf(`Something <ansi fg="yellow-bold">grapples</ansi> you, transitioning to <ansi fg="cyan">%s</ansi> position!`, result.PositionDesc))
			}
		}
		room.SendTextVisual(messaging.CategoryGrappleFlow,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> <ansi fg="username">%s</ansi> into <ansi fg="cyan">%s</ansi> position!`, mobName, targetName, result.PositionDesc),
			targetPlayerId)

		// Disarm messaging
		if result.DisarmResult != nil {
			if targetChar != nil {
				targetChar.SendText(messaging.CategoryGrappleFlow, result.DisarmResult.TargetMsg)
			}
			room.SendTextVisual(messaging.CategoryGrappleFlow, result.DisarmResult.RoomMessage, targetPlayerId)
		}
	} else {
		if targetChar != nil {
			if canSee {
				targetChar.SendText(messaging.CategoryGrappleFlow, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to grapple you, but you slip away!`, mobName))
			} else {
				targetChar.SendText(messaging.CategoryGrappleFlow, `Something tries to grapple you, but you slip away!`)
			}
		}
		room.SendTextVisual(messaging.CategoryGrappleFlow,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to grapple <ansi fg="username">%s</ansi>, but fails!`, mobName, targetName),
			targetPlayerId)

		// Critical failure messaging
		if result.CritFailure != nil {
			if targetChar != nil {
				targetChar.SendText(messaging.CategoryGrappleFlow, result.CritFailure.TargetMessage)
			}
			room.SendTextVisual(messaging.CategoryGrappleFlow, result.CritFailure.RoomMessage, targetPlayerId)
		}
	}

	return true, nil
}
