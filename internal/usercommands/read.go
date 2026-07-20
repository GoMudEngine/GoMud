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
		// Readable items (books/scrolls) carry their text in the blob.
		desc := ""
		if iSpec.Type == items.Readable {
			desc = string(readItem.GetBlob())
		}
		// Fall back to the item's description for anything else. Quest
		// notices/letters/signs are frequently authored as `object`/`holdable`
		// yet the game explicitly tells players to "read the notice on the
		// post" — restricting `read` to the Readable type dead-ends that beat.
		// Any held item that carries text can be read.
		if len(desc) == 0 {
			desc = iSpec.Description
		}
		if len(desc) > 0 {
			foundItemName = readItem.DisplayName()
			foundItemLongName = readItem.DisplayName()
			foundItemDescription = desc
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
