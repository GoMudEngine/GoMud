package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Equip(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if mob.Character.HasBuffFlag(buffs.PermaGear) {
		mob.Command(`emote struggles with their gear for a while, then gives up.`)
		return true, nil
	}

	if rest == "all" {
		itemCopies := []items.Item{}
		itemCopies = append(itemCopies, mob.Character.Items...)

		for _, item := range itemCopies {
			iSpec := item.GetSpec()
			if iSpec.Subtype == items.Wearable || iSpec.Type == items.Weapon {
				Equip(item.Name(), mob, room)
			}
		}
		return true, nil
	}

	var matchItem items.Item = items.Item{}
	var found bool = false

	if rest == `random` {
		if len(mob.Character.Items) > 0 {
			matchItem = mob.Character.Items[util.Rand(len(mob.Character.Items))]
			found = true
		}
	}

	if !found {
		matchItem, found = mob.Character.FindInBackpack(rest)
	}

	if !found {
		return true, nil
	}

	iSpec := matchItem.GetSpec()
	if iSpec.Type != items.Weapon && iSpec.Subtype != items.Wearable {
		return true, nil
	}

	// Same-item no-op guard: if the mob already has this exact item on body,
	// skip to avoid noisy "equips / unequips" churn from blind equip commands.
	if existingItem, onBody := mob.Character.FindOnBody(matchItem.Name()); onBody && matchItem.Equals(existingItem) {
		return true, nil
	}

	actor := &actions.MobActor{Mob: mob, Room: room}
	result := actions.EquipItem(actor, matchItem.Name())

	if result.Equipped {
		for _, oldItem := range result.DisplacedItems {
			if oldItem.ItemId != 0 {
				room.SendTextVisual(
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi> removes their <ansi fg="item">%s</ansi> and stores it away.`, mob.Character.Name, oldItem.DisplayName()))
			}
		}

		if iSpec.Subtype == items.Wearable {
			room.SendTextVisual(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> puts on <ansi fg="item">%s</ansi>.`, mob.Character.Name, result.Item.DisplayName()))
		} else {
			room.SendTextVisual(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> wields <ansi fg="item">%s</ansi>.`, mob.Character.Name, result.Item.DisplayName()))
		}
	}

	return true, nil
}
