package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Break(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.Character.IsInCombat() {
		user.Character.EndAggro()
		user.SendText(messaging.CategorySystem, `You break off combat.`)
		room.SendTextVisual(messaging.CategoryMobEmote, 
			fmt.Sprintf(`<ansi fg="username">%s</ansi> breaks off combat.`, user.Character.Name),
			user.UserId,
		)
	} else {
		user.SendText(messaging.CategorySystem, `You aren't in combat!`)
	}

	return true, nil
}
