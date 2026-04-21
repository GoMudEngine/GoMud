package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"

	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* mute 				(All)
 */
func Mute(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		infoOutput, _ := templates.Process("admincommands/help/command.mute", nil, user.UserId)
		user.SendText(infoOutput)
		return true, nil
	}

	target, err := actions.ResolveTargetActor(room, rest)
	if err != nil || !target.IsPlayer() {
		user.SendText("Could not find user.")
		return true, nil
	}

	u := target.(*actions.UserActor).User
	u.Muted = true

	user.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> (<ansi fg="username">%s</ansi>) has been <ansi fg="alert-5">MUTED</ansi>`, u.Username, u.Character.Name))

	return true, nil
}

func UnMute(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		infoOutput, _ := templates.Process("admincommands/help/command.mute", nil, user.UserId)
		user.SendText(infoOutput)
		return true, nil
	}

	target, err := actions.ResolveTargetActor(room, rest)
	if err != nil || !target.IsPlayer() {
		user.SendText("Could not find user.")
		return true, nil
	}

	u := target.(*actions.UserActor).User
	u.Muted = false

	user.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> (<ansi fg="username">%s</ansi>) has been <ansi fg="alert-1">UNMUTED</ansi>`, u.Username, u.Character.Name))

	return true, nil
}
