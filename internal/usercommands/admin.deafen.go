package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"

	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* deafen 				(All)
 */
func Deafen(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		infoOutput, _ := templates.Process("admincommands/help/command.deafen", nil, user.UserId)
		user.SendText(messaging.CategorySystem, infoOutput)
		return true, nil
	}

	target, err := actions.ResolveTargetActor(room, rest)
	if err != nil || !target.IsPlayer() {
		user.SendText(messaging.CategorySystem, "Could not find user.")
		return true, nil
	}

	u := target.(*actions.UserActor).User
	u.Deafened = true

	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> (<ansi fg="username">%s</ansi>) has been <ansi fg="alert-5">DEAFENED</ansi>`, u.Username, u.Character.Name))

	return true, nil
}

func UnDeafen(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		infoOutput, _ := templates.Process("admincommands/help/command.deafen", nil, user.UserId)
		user.SendText(messaging.CategorySystem, infoOutput)
		return true, nil
	}

	target, err := actions.ResolveTargetActor(room, rest)
	if err != nil || !target.IsPlayer() {
		user.SendText(messaging.CategorySystem, "Could not find user.")
		return true, nil
	}

	u := target.(*actions.UserActor).User
	u.Deafened = false

	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> (<ansi fg="username">%s</ansi>) has been <ansi fg="alert-1">UNDEAFENED</ansi>`, u.Username, u.Character.Name))

	return true, nil
}
