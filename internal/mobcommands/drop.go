package mobcommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Drop(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {
		return true, nil
	}

	if args[0] == "all" {

		iCopies := []items.Item{}

		if mob.Character.Gold > 0 {
			Drop(fmt.Sprintf("%d gold", mob.Character.Gold), mob, room)
		}

		for _, item := range mob.Character.Items {
			iCopies = append(iCopies, item)
		}

		for _, item := range iCopies {
			Drop(item.Name(), mob, room)
		}

		return true, nil
	}

	// Drop 10 gold
	if len(args) >= 2 && args[1] == "gold" {
		g, _ := strconv.ParseInt(args[0], 10, 32)
		dropAmt := int(g)
		if dropAmt < 1 {
			return true, nil
		}

		if dropAmt <= mob.Character.Gold {

			room.Gold += dropAmt
			mob.Character.Gold -= dropAmt

			room.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> drops <ansi fg="gold">%d gold</ansi>.`, mob.Character.Name, dropAmt))

			return true, nil
		}
	}

	if mob.Character.HasBuffFlag(buffs.PermaGear) {
		return true, nil
	}

	// Check whether the user has an item in their inventory that matches
	matchItem, found := mob.Character.FindInBackpack(rest)

	if found {

		// Swap the item location
		room.AddItem(matchItem, false)
		mob.Character.RemoveItem(matchItem)

		events.AddToQueue(events.ItemOwnership{
			MobInstanceId: mob.InstanceId,
			Item:          matchItem,
			Gained:        false,
		})

		room.SendText(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> drops their <ansi fg="item">%s</ansi>...`, mob.Character.Name, matchItem.DisplayName()))
	}

	return true, nil
}
