package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func SetDesc(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		current := user.Character.GetDescription()
		user.SendTextLegacy(fmt.Sprintf("Your current description:\n%s", current))
		user.SendTextLegacy(``)
		user.SendTextLegacy(`To set a new description: <ansi fg="command">setdesc <your description></ansi>`)
		user.SendTextLegacy(`To clear: <ansi fg="command">setdesc clear</ansi>`)
		return true, nil
	}

	if rest == "clear" {
		user.Character.Description = "They seem thoroughly uninteresting."
		user.SendTextLegacy("Your description has been cleared.")
		return true, nil
	}

	// Store raw text — wrapping happens at display time in the template
	user.Character.Description = rest
	user.SendTextLegacy(fmt.Sprintf(
		"Your description has been set to:\n%s",
		util.SplitStringNL(rest, 72)))

	return true, nil
}
