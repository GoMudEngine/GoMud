package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Read(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Check whether the user has an item in their inventory that matches

	foundItemName := ""
	foundItemLongName := ""
	foundItemDescription := ""
	// Search for an exact match first
	if readItem, found := user.Character.FindInBackpack(rest); found {
		iSpec := readItem.GetSpec()
		if iSpec.Type == items.Readable {
			foundItemName = readItem.DisplayName()
			foundItemLongName = readItem.DisplayName()
			foundItemDescription = string(readItem.GetBlob())
			if len(foundItemDescription) == 0 {
				foundItemDescription = iSpec.Description
			}
		}
	}

	isSneaking := user.Character.IsHidden()

	if len(foundItemName) == 0 {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You don't have a "%s" that can be read.`, rest))
	} else {
		user.SendText(messaging.CategorySystem, 
			fmt.Sprintf(`You look at <ansi fg="item">%s</ansi>...`, foundItemLongName),
		)

		if !isSneaking {
			room.SendTextVisual(messaging.CategoryMobEmote, 
				fmt.Sprintf(`<ansi fg="username">%s</ansi> looks at their <ansi fg="item">%s</ansi>...`, user.Character.Name, foundItemName),
				user.UserId,
			)
		}

		user.SendText(messaging.CategorySystem, foundItemDescription)
	}

	return true, nil
}
