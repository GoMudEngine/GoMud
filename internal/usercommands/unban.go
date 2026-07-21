package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/moderation"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Unban lifts an account ban (by name) or an IP ban.
// Usage: unban <name>  |  unban ip <ip>
func Unban(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	fields := strings.Fields(rest)
	if len(fields) == 0 {
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Usage: unban <name>  |  unban ip <ip></ansi>`)
		return true, nil
	}

	if strings.ToLower(fields[0]) == "ip" && len(fields) >= 2 {
		_ = moderation.UnbanIP(fields[1])
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="green">IP unbanned:</ansi> %s`, fields[1]))
		return true, nil
	}

	_ = moderation.Unban(fields[0])
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="green"><ansi fg="username">%s</ansi> has been unbanned.</ansi>`, fields[0]))
	return true, nil
}
