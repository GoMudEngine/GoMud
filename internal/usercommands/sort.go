package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Sort(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.Character.Equipment.ComponentBag.ItemId < 1 {
		user.SendText(`You don't have a component bag equipped.`)
		return true, nil
	}

	moved := user.Character.SortComponentItems()

	if moved == 0 {
		user.SendText(`No crafting materials found to sort.`)
		return true, nil
	}

	user.SendText(fmt.Sprintf(
		`<ansi fg="green">You sort your materials into your %s. (%d items moved)</ansi>`,
		user.Character.Equipment.ComponentBag.DisplayName(), moved))

	return true, nil
}
