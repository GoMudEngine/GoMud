package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/badinputtracker"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* badcommands 				(All)
 */
func BadCommands(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "clear" {
		badinputtracker.Clear()
	}

	// Now look for mobs?
	headers := []string{"Command", "Count"}
	rows := [][]string{}

	for cmd, ct := range badinputtracker.GetBadCommands() {
		rows = append(rows, []string{
			cmd,
			fmt.Sprintf(`%d`, ct),
		})
	}

	badCommandTableData := templates.GetTable(`Bad Commands`, headers, rows)
	tplTxt, _ := templates.Process("tables/generic", badCommandTableData, user.UserId, user.UserId)

	user.SendTextLegacy(tplTxt)

	return true, nil
}
