package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* questdebug 				(Admin only)
 */
func QuestDebug(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	rest = strings.TrimSpace(rest)

	if rest == "" {
		user.SendTextLegacy(`<ansi fg="command">questdebug <player></ansi> - Enable verbose quest logging for a player.`)
		user.SendTextLegacy(`<ansi fg="command">questdebug <player> off</ansi> - Disable quest debug for a player.`)
		return true, nil
	}

	parts := strings.Fields(rest)
	playerName := parts[0]
	disable := len(parts) > 1 && strings.ToLower(parts[1]) == "off"

	target := users.GetByCharacterName(playerName)
	if target == nil {
		user.SendTextLegacy(fmt.Sprintf(`<ansi fg="red">Player "%s" not found or not online.</ansi>`, playerName))
		return true, nil
	}

	if disable {
		questengine.SetPlayerDebug(target.UserId, false)
		user.SendTextLegacy(fmt.Sprintf(
			`<ansi fg="yellow">Quest debug disabled for %s.</ansi>`, target.Character.Name))
	} else {
		questengine.SetPlayerDebug(target.UserId, true)
		user.SendTextLegacy(fmt.Sprintf(
			`<ansi fg="green">Quest debug enabled for %s. All quest evaluations will log at verbose.</ansi>`,
			target.Character.Name))
	}

	return true, nil
}
