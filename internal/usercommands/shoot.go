package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Shoot(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	res := actions.ExecuteShoot(&actions.UserActor{User: user, Room: room}, rest)

	if res.NoWeapon {
		user.SendText(`You don't have a shooting weapon.`)
		return true, nil
	}

	if res.BadSyntax {
		user.SendText(`Syntax: <ansi fg="command">shoot [target] [exit]</ansi>`)
		return true, nil
	}

	if res.NoExit {
		user.SendText(`Could not find where you wanted to shoot`)
		return true, nil
	}

	if res.ExitLocked {
		user.SendText(fmt.Sprintf("The %s exit is locked.", res.ExitName))
		return true, nil
	}

	if res.NoTarget {
		user.SendText(`Could not find your target.`)
		return true, nil
	}

	if res.IsCharmed {
		user.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is your friend!`, res.TargetName))
		return true, nil
	}

	// Party friendly-fire check (player targets only).
	if !res.IsTargetMob && res.TargetUserId > 0 {
		if partyInfo := parties.Get(user.UserId); partyInfo != nil {
			if partyInfo.IsMember(res.TargetUserId) {
				p := users.GetByUserId(res.TargetUserId)
				if p != nil {
					user.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is in your party!`, p.Character.Name))
					return true, nil
				}
			}
		}
	}

	if !res.Ready {
		return true, nil
	}

	// Send feedback messages.
	if res.IsTargetMob {
		user.SendText(
			fmt.Sprintf(`You prepare to shoot at <ansi fg="mobname">%s</ansi> through the <ansi fg="exit">%s</ansi> exit.`, res.TargetName, res.ExitName),
		)
		if !res.IsSneaking {
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> prepares to shoot at <ansi fg="mobname">%s</ansi> through the <ansi fg="exit">%s</ansi> exit.`, user.Character.Name, res.TargetName, res.ExitName),
				user.UserId,
			)
		}
	} else {
		user.SendText(
			fmt.Sprintf(`You prepare to shoot at <ansi fg="username">%s</ansi> through the <ansi fg="exit">%s</ansi> exit.`, res.TargetName, res.ExitName),
		)
		if !res.IsSneaking {
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> prepares to shoot at <ansi fg="username">%s</ansi> through the <ansi fg="exit">%s</ansi> exit.`, user.Character.Name, res.TargetName, res.ExitName),
				user.UserId,
			)
		}
	}

	return true, nil
}
