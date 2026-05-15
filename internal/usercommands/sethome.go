package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func SetHome(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := strings.ToLower(strings.TrimSpace(rest))

	if args == "" {
		// Show current home and available options
		current := user.Character.GetSetting("home")
		if current == "" {
			current = "default"
		}
		currentName := characters.HomeLocationNames[current]
		if currentName == "" {
			currentName = characters.HomeLocationNames["default"]
		}

		user.SendText(fmt.Sprintf(
			`<ansi fg="yellow-bold">Current home:</ansi> %s`,
			currentName))
		user.SendText(``)
		user.SendText(`<ansi fg="yellow-bold">Available locations:</ansi>`)
		for key, name := range characters.HomeLocationNames {
			marker := ""
			if key == current {
				marker = ` <ansi fg="green">(current)</ansi>`
			}
			user.SendText(fmt.Sprintf(
				`  <ansi fg="command">sethome %s</ansi> - %s%s`,
				key, name, marker))
		}
		return true, nil
	}

	if _, valid := characters.HomeLocations[args]; !valid {
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">Unknown location "%s".</ansi> Type `+
				`<ansi fg="command">sethome</ansi> to see options.`,
			args))
		return true, nil
	}

	user.Character.SetSetting("home", args)
	user.SendText(fmt.Sprintf(
		`<ansi fg="green">Home set to %s.</ansi> `+
			`You will return here when you die.`,
		characters.HomeLocationNames[args]))

	return true, nil
}
