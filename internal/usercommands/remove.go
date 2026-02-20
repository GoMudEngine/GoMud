package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Remove(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "all" {
		removedItems := []items.Item{}
		for _, item := range user.Character.Equipment.GetAllItems() {
			Remove(item.Name(), user, room, flags)
			removedItems = append(removedItems, item)
		}

		events.AddToQueue(events.EquipmentChange{
			UserId:       user.UserId,
			ItemsRemoved: removedItems,
		})

		return true, nil
	}

	// Check whether the user has an item in their inventory that matches
	matchItem, found := user.Character.FindOnBody(rest)

	if !found || matchItem.ItemId < 1 {
		user.SendText(fmt.Sprintf(`You don't appear to be using a "%s".`, rest))
	} else {

		if matchItem.IsCursed() && user.Character.Health > 0 {
			// Cursed items can no longer be removed by skill — use an uncurse spell or service
			if user.Character.GetSkillLevel(skills.Spellcasting) < 4 {
				user.SendText(
					fmt.Sprintf(`You can't seem to remove your <ansi fg="item">%s</ansi>... It's <ansi fg="red-bold">CURSED!</ansi>`, matchItem.DisplayName()),
				)

				return true, nil
			} else {
				user.SendText(
					`It's <ansi fg="red-bold">CURSED</ansi> but luckily your <ansi fg="skillname">enchant</ansi> skill level allows you to remove it.`,
				)
			}
		}

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		if user.Character.RemoveFromBody(matchItem) {
			user.SendText(
				fmt.Sprintf(`You remove your <ansi fg="item">%s</ansi> and return it to your backpack.`, matchItem.DisplayName()),
			)
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> removes their <ansi fg="item">%s</ansi> and stores it away.`, user.Character.Name, matchItem.DisplayName()),
				user.UserId,
			)

			user.Character.StoreItem(matchItem)

			events.AddToQueue(events.EquipmentChange{
				UserId:       user.UserId,
				ItemsRemoved: []items.Item{matchItem},
			})

		} else {
			user.SendText(
				fmt.Sprintf(`You can't seem to remove your <ansi fg="item">%s</ansi>.`, matchItem.DisplayName()),
			)
		}

		user.Character.Validate()

	}

	return true, nil
}
