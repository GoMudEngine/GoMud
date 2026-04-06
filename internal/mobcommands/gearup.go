package mobcommands

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Gearup(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if mob.Character.HasBuffFlag(buffs.PermaGear) {
		mob.Command(`emote struggles with their gear for a while, then gives up.`)
		return true, nil
	}

	if rest != `` {
		// Check whether the mob has an item in their inventory that matches
		matchItem, found := mob.Character.FindInBackpack(rest)

		if found {

			matchSpec := matchItem.GetSpec()
			for _, itm := range mob.Character.Equipment.GetAllItems() {
				itmSpec := itm.GetSpec()
				if itmSpec.Type == matchSpec.Type && matchSpec.Value > itmSpec.Value {

					mob.Command(fmt.Sprintf(`wear !%d`, matchItem.ItemId))
					mob.Command(fmt.Sprintf(`drop !%d`, itm.ItemId))

				}
			}

		}
	} else {
		allBackpackItems := mob.Character.GetAllBackpackItems()

		// Sort by value descending — best items first
		sort.Slice(allBackpackItems, func(i, j int) bool {
			return allBackpackItems[i].GetSpec().Value > allBackpackItems[j].GetSpec().Value
		})

		isCharmed := mob.Character.IsCharmed()

		for _, itm := range allBackpackItems {
			itmSpec := itm.GetSpec()

			if itmSpec.Type != items.Weapon && itmSpec.Subtype != items.Wearable {
				continue
			}

			// Track what was worn before so charmed mobs can drop displaced items
			var oldEquipped []items.Item
			if isCharmed {
				oldEquipped = mob.Character.Equipment.GetAllItems()
			}

			mob.Command(fmt.Sprintf(`wear !%d`, itm.ItemId))

			// If charmed, drop any displaced items
			if isCharmed {
				newEquipped := mob.Character.Equipment.GetAllItems()
				for _, oldItm := range oldEquipped {
					if oldItm.ItemId < 1 {
						continue
					}
					found := false
					for _, newItm := range newEquipped {
						if oldItm.ItemId == newItm.ItemId {
							found = true
							break
						}
					}
					if !found {
						mob.Command(fmt.Sprintf(`drop !%d`, oldItm.ItemId))
					}
				}
			}
		}
	}

	return true, nil
}
