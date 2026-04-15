package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Remove(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if mob.Character.HasBuffFlag(buffs.PermaGear) {
		mob.Command(`emote struggles with their gear for a while, then gives up.`)
		return true, nil
	}

	actor := &actions.MobActor{Mob: mob, Room: room}

	if rest == "all" {
		removedItems := []items.Item{}
		for _, item := range mob.Character.Equipment.GetAllItems() {
			result := actions.RemoveEquipment(actor, item.Name())
			if result.Removed {
				removedItems = append(removedItems, result.Item)
			}
		}

		events.AddToQueue(events.EquipmentChange{
			MobInstanceId: mob.InstanceId,
			ItemsRemoved:  removedItems,
		})

		return true, nil
	}

	// Single-item remove path
	result := actions.RemoveEquipment(actor, rest)

	if result.Removed {
		room.SendTextVisual(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> removes their <ansi fg="item">%s</ansi> and stores it away.`, mob.Character.Name, result.Item.DisplayName()))
	}

	return true, nil
}
