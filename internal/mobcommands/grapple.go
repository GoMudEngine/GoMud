package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Grapple(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Must be in combat to use grapple
	if mob.Character.Aggro == nil {
		return true, nil
	}

	res := actions.ExecuteGrapple(&actions.MobActor{Mob: mob, Room: room})

	if res.OnCooldown || res.NoTarget || res.GrappleImmune || !res.Executed {
		return true, nil
	}

	// Fire skill progression for the executed special move.
	mob.Character.OnSkillUse(string(skills.UnarmedCombat), 0)

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
				targetChar.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> you, transitioning to <ansi fg="cyan">%s</ansi> position!`, mobName, result.PositionDesc))
			} else {
				targetChar.SendText(fmt.Sprintf(`Something <ansi fg="yellow-bold">grapples</ansi> you, transitioning to <ansi fg="cyan">%s</ansi> position!`, result.PositionDesc))
			}
		}
		sendRoomText(room,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> <ansi fg="yellow-bold">grapples</ansi> <ansi fg="username">%s</ansi> into <ansi fg="cyan">%s</ansi> position!`, mobName, targetName, result.PositionDesc),
			targetPlayerId)

		// Disarm messaging
		if result.DisarmResult != nil {
			if targetChar != nil {
				targetChar.SendText(result.DisarmResult.TargetMsg)
			}
			sendRoomText(room, result.DisarmResult.RoomMessage, targetPlayerId)
		}
	} else {
		if targetChar != nil {
			if canSee {
				targetChar.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to grapple you, but you slip away!`, mobName))
			} else {
				targetChar.SendText(`Something tries to grapple you, but you slip away!`)
			}
		}
		sendRoomText(room,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tries to grapple <ansi fg="username">%s</ansi>, but fails!`, mobName, targetName),
			targetPlayerId)

		// Critical failure messaging
		if result.CritFailure != nil {
			if targetChar != nil {
				targetChar.SendText(result.CritFailure.TargetMessage)
			}
			sendRoomText(room, result.CritFailure.RoomMessage, targetPlayerId)
		}
	}

	return true, nil
}
