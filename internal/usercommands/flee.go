package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Flee(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !user.Character.IsDisengaging() {
		// Fleeing costs stamina
		const fleeStaminaCost = 10
		if !user.Character.DeductStamina(fleeStaminaCost) {
			user.SendText(`You're too exhausted to flee! You need to stand and fight.`)
			return true, nil
		}

		user.SendText(`You attempt to flee...`)

		user.Character.Aggro = &characters.Aggro{}
		user.Character.Aggro.Type = characters.Flee
	}

	return true, nil
}
