package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Drink(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Search bandolier first (oldest first), then backpack
	fromBandolier := false
	matchItem, found := user.Character.FindInPotions(rest)
	if found {
		fromBandolier = true
	} else {
		matchItem, found = user.Character.FindInBackpack(rest)
	}

	if !found {
		user.SendText(fmt.Sprintf(`You don't have a "%s" to drink.`, rest))
	} else {

		itemSpec := matchItem.GetSpec()

		if itemSpec.Subtype != items.Drinkable {
			user.SendText(
				fmt.Sprintf(`You can't drink <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()),
			)
			return true, nil
		}

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		if fromBandolier {
			user.Character.UseItemFromPotions(matchItem)
		} else {
			user.Character.UseItem(matchItem)
		}

		user.SendText(fmt.Sprintf(`You drink the <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()))
		room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> drinks <ansi fg="itemname">%s</ansi>.`, user.Character.Name, matchItem.DisplayName()), user.UserId)

		for _, buffId := range itemSpec.BuffIds {
			user.AddBuff(buffId, `drink`)
		}
	}

	return true, nil
}
