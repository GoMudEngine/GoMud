package usercommands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Alias(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest != `` {
		if strings.Index(rest, `=`) != -1 {
			parts := strings.SplitN(rest, `=`, 2)
			aliasName := parts[0]
			aliasVal := ``
			if len(parts) > 1 {
				aliasVal = parts[1]
			}

			addedAlias, deletedAlias := user.AddCommandAlias(aliasName, aliasVal)

			if addedAlias != `` {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="yellow">Custom Alias Added:</ansi> <ansi fg="command">%s</ansi>=<ansi fg="command">%s</ansi>`, addedAlias, aliasVal))
			}
			if deletedAlias != `` {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="yellow">Custom Alias Removed:</ansi> <ansi fg="command">%s</ansi>`, deletedAlias))
			}

			events.AddToQueue(events.AutomationChanged{UserId: user.UserId})

			return true, nil

		}
	}

	// biuld array and look up table for sorting purposes
	allOutCmds := []string{}
	reverseLookup := map[string][]string{}

	for inCmd, outCmd := range keywords.GetAllCommandAliases() {
		if _, ok := reverseLookup[outCmd]; !ok {
			reverseLookup[outCmd] = []string{}
			allOutCmds = append(allOutCmds, outCmd)
		}
		reverseLookup[outCmd] = append(reverseLookup[outCmd], inCmd)
	}

	sort.Strings(allOutCmds)

	headers := []string{"Alias", "Command"}
	rows := [][]string{}

	for _, outCmd := range allOutCmds {
		inCmds := reverseLookup[outCmd]
		for _, inCmd := range inCmds {
			rows = append(rows, []string{inCmd, outCmd})
		}
	}

	tableFormatting := []string{`<ansi fg="yellow">%s</ansi>`, `<ansi fg="command">%s</ansi>`}

	aliasTableData := templates.GetTable(`Built in Aliases`, headers, rows, tableFormatting)
	aliasTxt, _ := templates.Process("tables/generic", aliasTableData, user.UserId)
	user.SendText(messaging.CategorySystem, aliasTxt)

	if len(user.Aliases) > 0 {

		headers = []string{"Alias", "Command"}
		rows = [][]string{}

		for inCmd, outCmd := range user.Aliases {
			rows = append(rows, []string{inCmd, outCmd})
		}

		aliasTableData := templates.GetTable(`Custom Aliases`, headers, rows, tableFormatting)
		aliasTxt, _ := templates.Process("tables/generic", aliasTableData, user.UserId)
		user.SendText(messaging.CategorySystem, aliasTxt)
	}

	user.SendText(messaging.CategorySystem, `<ansi fg="yellow"><ansi fg="command">help alias</ansi> for more information on setting custom aliases.</ansi>`)
	user.SendText(messaging.CategorySystem, ``)

	return true, nil
}
