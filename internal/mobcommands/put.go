package mobcommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Put(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 2 {
		return true, nil
	}

	containerName := ``
	nameSearch := ``
	for i := len(args) - 1; i >= 1; i-- {
		if len(nameSearch) > 0 {
			nameSearch = ` ` + nameSearch
		}
		nameSearch = args[i] + nameSearch

		containerName = room.FindContainerByName(nameSearch)
		if containerName != `` {
			args = args[:i]
			break
		}
	}

	if containerName == `` {
		return true, nil
	}

	container := room.Containers[containerName]

	if len(args) < 1 {
		return true, nil
	}

	var item items.Item
	var itemFound bool
	goldAmt := 0

	if len(args) >= 2 && args[1] == `gold` {

		g, _ := strconv.ParseInt(args[0], 10, 32)
		goldAmt = int(g)
		if goldAmt < 0 {
			goldAmt = -1 * goldAmt
		}

	} else {

		item, itemFound = mob.Character.FindInBackpack(strings.Join(args, ` `))
		if !itemFound && len(args) > 1 {
			item, itemFound = mob.Character.FindInBackpack(args[0])
		}
	}

	if !itemFound && goldAmt == 0 {
		return true, nil
	}

	if goldAmt > 0 {
		container.Gold += goldAmt
		room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> places some <ansi fg="gold">gold</ansi> into the <ansi fg="container">%s</ansi>`, mob.Character.Name, containerName))
	}

	if itemFound {
		container.AddItem(item)
		mob.Character.RemoveItem(item)

		events.AddToQueue(events.ItemOwnership{
			MobInstanceId: mob.InstanceId,
			Item:          item,
			Gained:        false,
		})

		room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> places their <ansi fg="itemname">%s</ansi> into the <ansi fg="container">%s</ansi>`, mob.Character.Name, item.DisplayName(), containerName))

		// Enforce container size limits
		if len(container.Items) > int(configs.GetGamePlayConfig().ContainerSizeMax) {

			randItemToRemove := util.Rand(len(container.Items))
			oopsItem := container.Items[randItemToRemove]

			// get all items that spawn in chests
			for _, spn := range room.SpawnInfo {
				if spn.Container == containerName && oopsItem.ItemId == spn.ItemId {
					// Don't let this one pop out
					oopsItem = item
					break
				}
			}

			container.RemoveItem(oopsItem)
			room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`The <ansi fg="container">%s</ansi> is too full and a <ansi fg="itemname">%s</ansi> falls out and onto the floor.`, containerName, oopsItem.DisplayName()))
			room.AddItem(oopsItem, false)
		}
	}

	room.Containers[containerName] = container

	return true, nil
}
