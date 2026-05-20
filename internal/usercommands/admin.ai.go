package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* ai-flag 				(All)
* ai-list				(All)
 */
func AiFlag(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		user.SendTextLegacy(`Usage: <ansi fg="command">ai-flag &lt;username&gt;</ansi> - Toggle AI flag on a user account.`)
		return true, nil
	}

	rest = strings.TrimSpace(rest)

	// Try to find the user by character name (in room first, then globally)
	target, err := actions.ResolveTargetActor(room, rest)
	var targetUser *users.UserRecord
	if err == nil && target.IsPlayer() {
		targetUser = target.(*actions.UserActor).User
	}

	// If not found in room, try by character name globally
	if targetUser == nil {
		targetUser = users.GetByCharacterName(rest)
	}

	if targetUser == nil {
		user.SendTextLegacy("Could not find user.")
		return true, nil
	}

	targetUser.IsAI = !targetUser.IsAI

	if targetUser.IsAI {
		user.SendTextLegacy(fmt.Sprintf(`<ansi fg="username">%s</ansi> (<ansi fg="username">%s</ansi>) has been flagged as <ansi fg="cyan-bold">AI</ansi>.`, targetUser.Username, targetUser.Character.Name))
	} else {
		user.SendTextLegacy(fmt.Sprintf(`<ansi fg="username">%s</ansi> (<ansi fg="username">%s</ansi>) AI flag has been <ansi fg="alert-1">removed</ansi>.`, targetUser.Username, targetUser.Character.Name))
	}

	return true, nil
}

func AiList(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	aiCount := 0
	for _, uid := range users.GetOnlineUserIds() {
		u := users.GetByUserId(uid)
		if u == nil {
			continue
		}

		isAIConn := false
		if cd := connections.Get(u.ConnectionId()); cd != nil {
			isAIConn = cd.ConnType() == connections.ConnAI
		}

		if u.IsAI || isAIConn {
			aiCount++
			connTag := ""
			if isAIConn {
				connTag = " <ansi fg=\"cyan\">[AI port]</ansi>"
			}
			flagTag := ""
			if u.IsAI {
				flagTag = " <ansi fg=\"yellow\">[flagged]</ansi>"
			}
			user.SendTextLegacy(fmt.Sprintf(`  <ansi fg="username">%s</ansi> (<ansi fg="username">%s</ansi>)%s%s`, u.Username, u.Character.Name, flagTag, connTag))
		}
	}

	if aiCount == 0 {
		user.SendTextLegacy("No AI accounts currently online.")
	} else {
		user.SendTextLegacy(fmt.Sprintf("%d AI account(s) online.", aiCount))
	}

	return true, nil
}
