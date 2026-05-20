package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Drink(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Chunk 4e: can't drink while grappled — both hands committed.
	if mob.Character.Position != nil && mob.Character.Position.IsGrappling() {
		return true, nil
	}

	// Check whether the user has an item in their inventory that matches
	if matchItem, found := mob.Character.FindInBackpack(rest); found {

		itemSpec := matchItem.GetSpec()

		if itemSpec.Subtype != items.Drinkable {
			return true, nil
		}

		mob.Character.CancelBuffsWithFlag(buffs.Hidden)

		mob.Character.UseItem(matchItem)

		room.SendTextVisual(messaging.CategoryMobEmote, fmt.Sprintf(`<ansi fg="mobname">%s</ansi> drinks <ansi fg="itemname">%s</ansi>.`, mob.Character.Name, matchItem.DisplayName()))

		for _, buffId := range itemSpec.BuffIds {
			mob.AddBuff(buffId, `drink`)
		}
	}

	return true, nil
}
