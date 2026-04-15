package mobcommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Selljunk converts mob inventory items into gold (1 coin per item).
func Selljunk(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if args[0] == "random" {
		// select a random item
		if len(mob.Character.Items) > 0 {
			matchItem := mob.Character.Items[util.Rand(len(mob.Character.Items))]
			Selljunk(matchItem.Name(), mob, room)

		}
		return true, nil
	}

	if args[0] == "all" {

		iCopies := []items.Item{}
		for _, item := range mob.Character.Items {
			iCopies = append(iCopies, item)
		}

		for _, item := range iCopies {
			Selljunk(item.Name(), mob, room)
		}

		return true, nil
	}

	// Check whether the user has an item in their inventory that matches
	matchItem, found := mob.Character.FindInBackpack(rest)

	if found {
		mob.Character.RemoveItem(matchItem)

		events.AddToQueue(events.ItemOwnership{
			MobInstanceId: mob.InstanceId,
			Item:          matchItem,
			Gained:        false,
		})

		mob.Character.Gold += 1
		room.SendTextVisual(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> chants softly. Their <ansi fg="item">%s</ansi> slowly levitates in the air, trembles briefly and then in a flash of light becomes a gold coin!`, mob.Character.Name, matchItem.DisplayName()))
	}

	return true, nil
}
