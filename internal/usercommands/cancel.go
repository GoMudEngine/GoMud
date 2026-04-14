package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Cancel aborts an in-progress fold-based cast, reporting conviction already spent.
func Cancel(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if user.Character.CastingState == nil {
		user.SendText(`You aren't casting anything.`)
		return true, nil
	}

	cs := user.Character.CastingState
	user.Character.CastingState = nil

	user.SendText(fmt.Sprintf(
		`<ansi fg="cyan">You release your held folds. %d conviction is lost.</ansi>`,
		cs.ConvictionSpent))
	room.SendTextVisual(fmt.Sprintf(
		`<ansi fg="username">%s</ansi> breaks their concentration.`,
		user.Character.Name), user.UserId)

	return true, nil
}
