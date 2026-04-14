package mobcommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Get(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {
		return true, nil
	}

	if args[0] == "all" {
		if room.Gold > 0 {
			Get("gold", mob, room)
		}

		if len(room.Items) > 0 {
			iCopies := []items.Item{}
			for _, item := range room.Items {
				iCopies = append(iCopies, item)
			}

			for _, item := range iCopies {
				Get(item.Name(), mob, room)
			}
		}

		return true, nil
	}

	if args[0] == "gold" {

		if room.Gold > 0 {

			mob.Character.CancelBuffsWithFlag(buffs.Hidden) // No longer sneaking

			actor := &actions.MobActor{Mob: mob, Room: room}
			goldAmt := room.Gold
			if err := actions.GetGoldFromFloor(actor, goldAmt); err == nil {
				room.SendTextVisual(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> picks up <ansi fg="gold">%d gold</ansi>.`, mob.Character.Name, goldAmt))
			}
		}

		return true, nil
	}

	getFromStash := false

	if len(args) >= 2 {
		// Detect "stash" or "from stash" at end and remove it
		if args[len(args)-1] == "stash" {
			getFromStash = true
			if args[len(args)-2] == "from" {
				rest = strings.Join(args[0:len(args)-2], " ")
			} else {
				rest = strings.Join(args[0:len(args)-1], " ")
			}
		}

		if args[len(args)-1] == "ground" {
			getFromStash = false
			if args[len(args)-2] == "from" {
				rest = strings.Join(args[0:len(args)-2], " ")
			} else {
				rest = strings.Join(args[0:len(args)-1], " ")
			}
		}

	}

	// Check whether the mob has an item on the floor that matches
	actor := &actions.MobActor{Mob: mob, Room: room}
	result := actions.GetItemFromFloor(actor, rest, getFromStash)

	if result.Found && result.Err == nil {
		mob.Character.CancelBuffsWithFlag(buffs.Hidden) // No longer sneaking

		room.SendTextVisual(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> picks up the <ansi fg="itemname">%s</ansi>...`, mob.Character.Name, result.Item.DisplayName()))
	}

	return true, nil
}
