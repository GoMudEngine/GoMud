package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Offer(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	item, found := user.Character.FindInBackpack(rest)
	if !found {
		user.SendText(messaging.CategorySystem, "You don't have that item.")
		return true, nil
	}

	itemSpec := item.GetSpec()
	if itemSpec.ItemId < 1 {
		return true, nil
	}

	for _, mobId := range room.GetMobs(rooms.FindMerchant) {

		mob := mobs.GetInstance(mobId)
		if mob == nil {
			continue
		}

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		if item.IsSpecial() {

			mob.Command(`say I'm afraid I don't buy those.`)

			continue
		}

		sellValue := mob.GetSellPrice(item)

		if sellValue <= 0 {

			mob.Command(`say I'm not interested in that.`)

			continue
		}

		mob.Command(fmt.Sprintf(`say I can give you <ansi fg="gold">%d gold</ansi> for that <ansi fg="itemname">%s</ansi>.`, sellValue, item.DisplayName()))

		break
	}

	return true, nil
}
