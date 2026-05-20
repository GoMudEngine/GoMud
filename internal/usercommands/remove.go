package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Remove(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	actor := &actions.UserActor{User: user, Room: room}

	if rest == "all" {
		removedItems := []items.Item{}
		for _, item := range user.Character.Equipment.GetAllItems() {
			result := actions.RemoveEquipment(actor, item.Name())
			if result.Removed {
				removedItems = append(removedItems, result.Item)
			}
		}

		events.AddToQueue(events.EquipmentChange{
			UserId:       user.UserId,
			ItemsRemoved: removedItems,
		})

		return true, nil
	}

	// Check whether the user has an item equipped that matches
	matchItem, found := user.Character.FindOnBody(rest)

	if !found || matchItem.ItemId < 1 {
		user.SendTextLegacy(fmt.Sprintf(`You don't appear to be using a "%s".`, rest))
	} else {

		// Cursed item check — must happen BEFORE calling the shared action,
		// since the shared action will actually remove the item.
		if matchItem.IsCursed() && user.Character.Health > 0 {
			if user.Character.GetSkillLevel(skills.Spellcasting) < 4 {
				user.SendTextLegacy(
					fmt.Sprintf(`You can't seem to remove your <ansi fg="item">%s</ansi>... It's <ansi fg="red-bold">CURSED!</ansi>`, matchItem.DisplayName()),
				)

				return true, nil
			} else {
				user.SendTextLegacy(
					`It's <ansi fg="red-bold">CURSED</ansi> but luckily your <ansi fg="skillname">enchant</ansi> skill level allows you to remove it.`,
				)
			}
		}

		result := actions.RemoveEquipment(actor, rest)

		if result.Removed {
			user.SendTextLegacy(
				fmt.Sprintf(`You remove your <ansi fg="item">%s</ansi> and return it to your backpack.`, result.Item.DisplayName()),
			)
			room.SendTextVisualLegacy(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> removes their <ansi fg="item">%s</ansi> and stores it away.`, user.Character.Name, result.Item.DisplayName()),
				user.UserId,
			)
		} else if result.Found {
			user.SendTextLegacy(
				fmt.Sprintf(`You can't seem to remove your <ansi fg="item">%s</ansi>.`, matchItem.DisplayName()),
			)
		}

	}

	return true, nil
}
