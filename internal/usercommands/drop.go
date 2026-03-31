package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Drop(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {
		user.SendText(`Drop what?`)

		return true, nil
	}

	if args[0] == "all" {

		// "drop all gold" — drop all gold explicitly
		if len(args) >= 2 && args[1] == "gold" && user.Character.Gold > 0 {
			Drop(fmt.Sprintf("%d gold", user.Character.Gold), user, room, flags)
			return true, nil
		}

		// "drop all" — drop all items but not gold
		iCopies := []items.Item{}
		iCopies = append(iCopies, user.Character.Items...)

		for _, item := range iCopies {
			Drop(item.Name(), user, room, flags)
		}

		return true, nil
	}

	// Drop 10 gold
	if len(args) >= 2 && args[1] == "gold" {
		g, _ := strconv.ParseInt(args[0], 10, 32)
		dropAmt := int(g)
		if dropAmt < 1 {
			user.SendText("Oops!")
			return true, nil
		}

		if dropAmt > user.Character.Gold {
			user.SendText(fmt.Sprintf("You don't have %d gold to drop.", dropAmt))
			return true, nil
		}

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		room.Gold += dropAmt
		user.Character.Gold -= dropAmt

		events.AddToQueue(events.EquipmentChange{
			UserId:     user.UserId,
			GoldChange: -dropAmt,
		})

		user.SendText(
			fmt.Sprintf(`You drop <ansi fg="gold">%d gold</ansi> on the floor.`, dropAmt),
		)
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> drops <ansi fg="gold">%d gold</ansi>.`, user.Character.Name, dropAmt),
			user.UserId,
		)

		return true, nil
	}

	// Parse match number for all.item support (e.g. "drop all.potion")
	itemName, matchNum := util.GetMatchNumber(rest)
	if matchNum == -1 {
		dropped := 0
		for {
			matchItem, found := user.Character.FindInBackpack(itemName)
			if !found {
				break
			}

			user.Character.CancelBuffsWithFlag(buffs.Hidden)

			user.Character.RemoveItem(matchItem)

			events.AddToQueue(events.ItemOwnership{
				UserId: user.UserId,
				Item:   matchItem,
				Gained: false,
			})

			room.AddItem(matchItem, false)
			dropped++
		}
		if dropped == 0 {
			user.SendText(fmt.Sprintf(`You don't have any "%s" to drop.`, itemName))
		} else {
			user.SendText(fmt.Sprintf(`You drop %d item(s).`, dropped))
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> drops some items.`, user.Character.Name),
				user.UserId,
			)
		}
		return true, nil
	}

	// Check whether the user has an item in their inventory that matches
	matchItem, found := user.Character.FindInBackpack(rest)

	if !found {
		user.SendText(fmt.Sprintf("You don't have a %s to drop.", rest))
	} else {

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		iSpec := matchItem.GetSpec()

		// Swap the item location
		user.Character.RemoveItem(matchItem)

		events.AddToQueue(events.ItemOwnership{
			UserId: user.UserId,
			Item:   matchItem,
			Gained: false,
		})

		user.SendText(
			fmt.Sprintf(`You drop the <ansi fg="item">%s</ansi>.`, matchItem.DisplayName()),
		)
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> drops their <ansi fg="item">%s</ansi>...`, user.Character.Name, matchItem.DisplayName()),
			user.UserId,
		)

		// If grenades are dropped, they explode and affect everyone in the room!
		if iSpec.Type == items.Grenade {

			user.SendText(`Todo. Grenades disabled for now.`)

		}

		room.AddItem(matchItem, false)

	}

	return true, nil
}
