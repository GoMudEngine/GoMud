package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Quit(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.Character.IsInCombat() {
		user.SendTextLegacy("You're too busy to quit right now!")
		return true, nil
	}
	user.AddBuff(0, `quitting`)

	return true, nil
}
