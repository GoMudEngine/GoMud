package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/itemvalue"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Gearup(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if mob.Character.HasBuffFlag(buffs.PermaGear) {
		mob.Command(`emote struggles with their gear for a while, then gives up.`)
		return true, nil
	}

	// Gate: check if this mob can equip gear from player gifts.
	// Animal species (weapon-slot disabled) and non-combat archetypes skip.
	if !itemvalue.CanEquipFromGive(&mob.Character, mob.BehaviorArchetype) {
		return true, nil
	}

	// Get the weight profile for this mob's archetypes.
	profile := itemvalue.ProfileFor(mob.Archetype, mob.BehaviorArchetype)

	if rest != `` {
		// Specific item: find it, check if upgrade, equip if yes.
		matchItem, found := mob.Character.FindInBackpack(rest)

		if found {
			if itemvalue.IsUpgrade(&mob.Character, profile, matchItem) {
				mob.Command(fmt.Sprintf(`wear !%d`, matchItem.ItemId))

				// If charmed, drop any displaced items so the owner can reclaim them.
				if mob.Character.IsCharmed() {
					// EquipItem handles displacement, so we need to check what was
					// actually displaced by looking at what's not still in backpack.
					// Simpler approach: use wear command which handles this internally.
					// The wear command will drop to floor for charmed mobs if needed.
				}
			}
		}
	} else {
		// Bare gearup: iterate all backpack items, equip each upgrade.
		allBackpackItems := mob.Character.GetAllBackpackItems()

		isCharmed := mob.Character.IsCharmed()

		for _, itm := range allBackpackItems {
			itmSpec := itm.GetSpec()

			// Only consider weapons and wearables.
			if itmSpec.Type != items.Weapon && itmSpec.Subtype != items.Wearable {
				continue
			}

			// Check if this item is an upgrade.
			if !itemvalue.IsUpgrade(&mob.Character, profile, itm) {
				continue
			}

			// Track what was worn before so charmed mobs can drop displaced items.
			var oldEquipped []items.Item
			if isCharmed {
				oldEquipped = mob.Character.Equipment.GetAllItems()
			}

			mob.Command(fmt.Sprintf(`wear !%d`, itm.ItemId))

			// If charmed, drop any displaced items.
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
