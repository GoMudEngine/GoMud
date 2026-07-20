package playtest

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func (m *PlaytestModule) registerCommands() {
	m.plug.AddUserCommand(`ai-flag`, m.cmdAIFlag, false, true) // admin only
	m.plug.AddUserCommand(`ai-list`, m.cmdAIList, false, true)
}

// ai-flag <username> [on|off]
func (m *PlaytestModule) cmdAIFlag(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	parts := strings.Fields(rest)
	if len(parts) == 0 {
		user.SendText(messaging.CategorySystem, `Usage: ai-flag <username> [on|off]`)
		return true, nil
	}
	name := parts[0]
	on := true
	if len(parts) > 1 && strings.EqualFold(parts[1], "off") {
		on = false
	}
	target, err := users.LoadUser(name)
	if err != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("No such account: %s", name))
		return true, nil
	}
	target.IsAI = on
	if err := users.SaveUser(target); err != nil {
		user.SendText(messaging.CategorySystem, "Failed to save: "+err.Error())
		return true, nil
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf("%s IsAI = %v", name, on))
	return true, nil
}

// ai-list lists all IsAI-flagged accounts (online and offline).
func (m *PlaytestModule) cmdAIList(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	var lines []string
	for _, u := range users.GetAllActiveUsers() {
		if u.IsAI {
			lines = append(lines, fmt.Sprintf("  %s (online)", u.Username))
		}
	}
	users.SearchOfflineUsers(func(u *users.UserRecord) bool {
		if u.IsAI {
			lines = append(lines, fmt.Sprintf("  %s (offline)", u.Username))
		}
		return true
	})
	if len(lines) == 0 {
		user.SendText(messaging.CategorySystem, `No AI-flagged accounts.`)
		return true, nil
	}
	user.SendText(messaging.CategorySystem, `AI-flagged accounts:`)
	for _, l := range lines {
		user.SendText(messaging.CategorySystem, l)
	}
	return true, nil
}
