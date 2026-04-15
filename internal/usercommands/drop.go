package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
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

	// Drop N gold
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

		if err := actions.FloorDropGold(dropAmt, user.Character, room); err != nil {
			user.SendText("Oops!")
			return true, nil
		}

		events.AddToQueue(events.EquipmentChange{
			UserId:     user.UserId,
			GoldChange: -dropAmt,
		})

		user.SendText(
			fmt.Sprintf(`You drop <ansi fg="gold">%d gold</ansi> on the floor.`, dropAmt),
		)
		room.SendTextVisual(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> drops <ansi fg="gold">%d gold</ansi>.`, user.Character.Name, dropAmt),
			user.UserId,
		)

		return true, nil
	}

	// Parse match number for all.item support (e.g. "drop all.potion")
	itemName, matchNum := util.GetMatchNumber(rest)
	if matchNum == -1 {
		actor := &actions.UserActor{User: user, Room: room}
		dropped := 0
		for {
			result := actions.DropItem(actor, itemName)
			if !result.Found {
				break
			}
			user.Character.CancelBuffsWithFlag(buffs.Hidden)
			dropped++
		}
		if dropped == 0 {
			user.SendText(fmt.Sprintf(`You don't have any "%s" to drop.`, itemName))
		} else {
			user.SendText(fmt.Sprintf(`You drop %d item(s).`, dropped))
			room.SendTextVisual(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> drops some items.`, user.Character.Name),
				user.UserId,
			)
		}
		return true, nil
	}

	// Single-item drop path
	actor := &actions.UserActor{User: user, Room: room}
	result := actions.DropItem(actor, rest)

	if !result.Found {
		user.SendText(fmt.Sprintf("You don't have a %s to drop.", rest))
	} else {
		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		iSpec := result.Item.GetSpec()

		user.SendText(
			fmt.Sprintf(`You drop the <ansi fg="item">%s</ansi>.`, result.Item.DisplayName()),
		)
		room.SendTextVisual(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> drops their <ansi fg="item">%s</ansi>...`, user.Character.Name, result.Item.DisplayName()),
			user.UserId,
		)

		// If grenades are dropped, they explode and affect everyone in the room!
		if iSpec.Type == items.Grenade {
			user.SendText(`Todo. Grenades disabled for now.`)
		}
	}

	return true, nil
}
