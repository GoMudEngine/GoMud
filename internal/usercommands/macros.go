package usercommands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Macros(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if len(user.Macros) == 0 {
		user.SendTextLegacy("You have no macros set.")
		return true, nil
	}

	sortedKeys := make([]string, 0, len(user.Macros))

	for k := range user.Macros {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	user.SendTextLegacy(`<ansi fg="226">Your macros:</ansi>`)
	for _, macro := range sortedKeys {
		macroCommand := user.Macros[macro]
		user.SendTextLegacy(``)

		user.SendTextLegacy(fmt.Sprintf(`  <ansi fg="228">%s</ansi>:`, macro))

		commandParts := strings.Split(macroCommand, `;`)

		for i, cmd := range commandParts {

			cmdParts := strings.Split(cmd, ` `)
			cmdAlone := cmdParts[0]
			cmdRest := ``
			if len(cmdParts) > 1 {
				cmdRest = strings.Join(cmdParts[1:], ` `)
			}

			user.SendTextLegacy(fmt.Sprintf(`      %s) <ansi fg="command">%s</ansi> %s`, string(rune(97+i)), cmdAlone, cmdRest))
		}
	}
	user.SendTextLegacy(``)
	user.SendTextLegacy(`To use a macro, type <ansi fg="command">={num}</ansi>.`)
	user.SendTextLegacy(`Some terminals support pressing the associated F-Key (<ansi fg="228">F1</ansi>, <ansi fg="228">F2</ansi>, etc.)`)

	return true, nil
}
