package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Boot force-disconnects an online player by name (server-wide, not room-scoped).
// A booted player may reconnect within the zombie window unless also banned.
// Usage: boot <name> [reason]
func Boot(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	fields := strings.Fields(rest)
	if len(fields) == 0 {
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Usage: boot <name> [reason]</ansi>`)
		return true, nil
	}
	name := fields[0]
	reason := strings.Join(fields[1:], " ")
	if reason == "" {
		reason = "disconnected by staff"
	}

	target := users.GetByCharacterName(name)
	if target == nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="red">No online player named "%s".</ansi>`, name))
		return true, nil
	}

	target.EventLog.Add(`conn`, fmt.Sprintf("Booted by %s: %s", user.Username, reason))
	target.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="alert-5">You have been disconnected by staff.</ansi> %s`, reason))

	connections.Kick(target.ConnectionId(), fmt.Sprintf("Booted by %s: %s", user.Username, reason))

	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> has been <ansi fg="alert-5">booted</ansi>.`, target.Username))
	return true, nil
}
