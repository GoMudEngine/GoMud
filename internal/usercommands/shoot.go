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
		user.SendTextLegacy(`You don't have a shooting weapon.`)
		return true, nil
	}

	if res.BadSyntax {
		user.SendTextLegacy(`Syntax: <ansi fg="command">shoot [target] [exit]</ansi>`)
		return true, nil
	}

	if res.NoExit {
		user.SendTextLegacy(`Could not find where you wanted to shoot`)
		return true, nil
	}

	if res.ExitLocked {
		user.SendTextLegacy(fmt.Sprintf("The %s exit is locked.", res.ExitName))
		return true, nil
	}

	if res.NoTarget {
		user.SendTextLegacy(`Could not find your target.`)
		return true, nil
	}

	if res.IsCharmed {
		user.SendTextLegacy(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is your friend!`, res.TargetName))
		return true, nil
	}

	if res.IsNonCombatant {
		user.SendTextLegacy(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, res.TargetName))
		return true, nil
	}

	// PvP check (player targets only).
	if !res.IsTargetMob && res.TargetUserId > 0 {
		if p := users.GetByUserId(res.TargetUserId); p != nil {
			if pvpErr := room.CanPvp(user, p); pvpErr != nil {
				user.SendTextLegacy(pvpErr.Error())
				return true, nil
			}
		}
		// Party friendly-fire check.
		if partyInfo := parties.Get(user.UserId); partyInfo != nil {
			if partyInfo.IsMember(res.TargetUserId) {
				p := users.GetByUserId(res.TargetUserId)
				if p != nil {
					user.SendTextLegacy(fmt.Sprintf(`<ansi fg="username">%s</ansi> is in your party!`, p.Character.Name))
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
		user.SendTextLegacy(
			fmt.Sprintf(`You prepare to shoot at <ansi fg="mobname">%s</ansi> through the <ansi fg="exit">%s</ansi> exit.`, res.TargetName, res.ExitName),
		)
		if !res.IsSneaking {
			room.SendTextVisualLegacy(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> prepares to shoot at <ansi fg="mobname">%s</ansi> through the <ansi fg="exit">%s</ansi> exit.`, user.Character.Name, res.TargetName, res.ExitName),
				user.UserId,
			)
		}
	} else {
		user.SendTextLegacy(
			fmt.Sprintf(`You prepare to shoot at <ansi fg="username">%s</ansi> through the <ansi fg="exit">%s</ansi> exit.`, res.TargetName, res.ExitName),
		)
		if !res.IsSneaking {
			room.SendTextVisualLegacy(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> prepares to shoot at <ansi fg="username">%s</ansi> through the <ansi fg="exit">%s</ansi> exit.`, user.Character.Name, res.TargetName, res.ExitName),
				user.UserId,
			)
		}
	}

	return true, nil
}
