package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Mutations lists all mutations the player has acquired from the Chrysalis.
func Mutations(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if len(user.Character.Mutations) == 0 {
		user.SendTextLegacy(`<ansi fg="magenta">The Chrysalis has not yet reshaped you. No mutations have emerged.</ansi>`)
		return true, nil
	}

	user.SendTextLegacy(``)
	user.SendTextLegacy(`<ansi fg="magenta"> .:. <ansi fg="yellow">Your Mutations</ansi> .:.</ansi>`)
	user.SendTextLegacy(``)

	for mutId, level := range user.Character.Mutations {
		spec := mutations.GetMutation(mutId)
		if spec == nil {
			user.SendTextLegacy(fmt.Sprintf(`  <ansi fg="yellow">%s</ansi> (Level %d)  <ansi fg="red">[data missing]</ansi>`, mutId, level))
			continue
		}
		user.SendTextLegacy(fmt.Sprintf(`  <ansi fg="yellow">%s</ansi> <ansi fg="magenta">(Level %d)</ansi>`, spec.Name, level))
		user.SendTextLegacy(fmt.Sprintf(`    <ansi fg="white">%s</ansi>`, spec.Description))
	}

	user.SendTextLegacy(``)
	return true, nil
}
