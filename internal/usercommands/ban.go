package usercommands

import (
	"fmt"
	"net"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/moderation"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Ban permanently bans an account (by name) or an IP.
// Usage: ban <name> [reason]          — account ban (boots if online)
//
//	ban ip <name|ip> [reason]    — IP ban
func Ban(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	fields := strings.Fields(rest)
	if len(fields) == 0 {
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Usage: ban <name> [reason]  |  ban ip <name|ip> [reason]</ansi>`)
		return true, nil
	}

	// IP ban subcommand.
	if strings.ToLower(fields[0]) == "ip" && len(fields) >= 2 {
		targetArg := fields[1]
		reason := strings.TrimSpace(strings.TrimPrefix(rest, fields[0]+" "+targetArg))
		if reason == "" {
			reason = "banned by staff"
		}
		ip := targetArg
		// If it isn't a literal IP, treat it as an online player's name.
		if net.ParseIP(targetArg) == nil {
			target := users.GetByCharacterName(targetArg)
			if target == nil {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="red">No online player named "%s" (and "%s" is not an IP).</ansi>`, targetArg, targetArg))
				return true, nil
			}
			cd := connections.Get(target.ConnectionId())
			if cd == nil {
				user.SendText(messaging.CategorySystem, `<ansi fg="red">Could not read that player's connection.</ansi>`)
				return true, nil
			}
			host, _, err := net.SplitHostPort(cd.RemoteAddr().String())
			if err != nil {
				host = cd.RemoteAddr().String()
			}
			ip = host
			connections.Kick(target.ConnectionId(), "Banned by staff: "+reason)
		}
		_ = moderation.BanIP(ip, reason, user.Username)
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="alert-5">IP banned:</ansi> %s`, ip))
		return true, nil
	}

	// Account ban.
	name := fields[0]
	reason := strings.TrimSpace(strings.TrimPrefix(rest, name))
	if reason == "" {
		reason = "banned by staff"
	}
	_ = moderation.BanAccount(name, reason, user.Username)

	// Boot if online.
	if target := users.GetByCharacterName(name); target != nil {
		target.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="alert-5">You have been banned.</ansi> %s`, reason))
		connections.Kick(target.ConnectionId(), "Banned by staff: "+reason)
	}

	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> has been <ansi fg="alert-5">banned</ansi>. Reason: %s`, name, reason))
	return true, nil
}
