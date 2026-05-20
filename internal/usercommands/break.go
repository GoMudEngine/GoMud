package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Break(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.Character.IsInCombat() {
		user.Character.EndAggro()
		user.SendTextLegacy(`You break off combat.`)
		room.SendTextVisualLegacy(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> breaks off combat.`, user.Character.Name),
			user.UserId,
		)
	} else {
		user.SendTextLegacy(`You aren't in combat!`)
	}

	return true, nil
}
