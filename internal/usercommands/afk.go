package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func AFK(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Toggle off if already AFK
	if user.ManualAFK && rest == "" {
		user.ManualAFK = false
		user.AFKMessage = ""
		user.SendText(`You are no longer AFK.`)
		room.SendText(fmt.Sprintf(
			`<ansi fg="username">%s</ansi> is back.`,
			user.Character.Name), user.UserId)
		return true, nil
	}

	// Set AFK with optional message
	user.ManualAFK = true
	user.AFKMessage = strings.TrimSpace(rest)

	if user.AFKMessage != "" {
		user.SendText(fmt.Sprintf(`You are now AFK: %s`, user.AFKMessage))
		room.SendText(fmt.Sprintf(
			`<ansi fg="username">%s</ansi> goes AFK: %s`,
			user.Character.Name, user.AFKMessage), user.UserId)
	} else {
		user.SendText(`You are now AFK. Type <ansi fg="command">afk</ansi> again to return.`)
		room.SendText(fmt.Sprintf(
			`<ansi fg="username">%s</ansi> goes AFK.`,
			user.Character.Name), user.UserId)
	}

	return true, nil
}
