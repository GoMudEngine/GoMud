package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* setmotd 				(All)
 */
func SetMotd(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		user.SendText(`Usage: <ansi fg="command">setmotd <message></ansi> - Set the message of the day.`)
		user.SendText(`Use <ansi fg="command">motd</ansi> to view the current message.`)
		return true, nil
	}

	newMotd := strings.TrimSpace(rest)

	if err := configs.SetVal("server.motd", newMotd); err != nil {
		user.SendText(fmt.Sprintf(`Error setting MOTD: %s`, err))
		return true, nil
	}

	user.SendText(`Message of the day has been updated.`)
	user.SendText(``)
	user.SendText(fmt.Sprintf(`New MOTD: %s`, newMotd))

	return true, nil
}
