package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// fleshGolemSpeciesId is the speciesId for the flesh golem mob type.
const fleshGolemSpeciesId = 35

// Consume lets a mob eat a corpse in the room to gain a regeneration buff.
// This is an idle/out-of-combat command — no aggro requirement.
// Flesh golems (speciesId 35) receive enhanced absorption with stronger regen
// and distinctive flavor text.
func Consume(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if len(room.Corpses) == 0 {
		return true, nil
	}

	// Find the first non-prunable corpse
	corpseIdx := -1
	for i, c := range room.Corpses {
		if !c.Prunable {
			corpseIdx = i
			break
		}
	}
	if corpseIdx < 0 {
		return true, nil
	}

	corpseName := room.Corpses[corpseIdx].Character.Name

	// Remove the corpse
	room.Corpses = append(room.Corpses[:corpseIdx], room.Corpses[corpseIdx+1:]...)

	isGolem := mob.Character.SpeciesId == fleshGolemSpeciesId

	if isGolem {
		// Flesh golems graft fallen flesh onto themselves — stronger and longer regen.
		mob.Character.AddCondition(characters.ConditionRegen, 10, 3.0, "grafted corpse")
		room.SendText(
			fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi> rips a piece from the fallen <ansi fg="mob-corpse">%s</ansi> and grafts it onto itself! Its form grows more massive.`,
				mob.Character.Name, corpseName,
			),
		)
	} else {
		// Standard consume: magnitude 2.0 (2x base regen) for 6 rounds
		mob.Character.AddCondition(characters.ConditionRegen, 6, 2.0, "consumed corpse")
		room.SendText(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tears into a corpse and feeds greedily!`, mob.Character.Name),
		)
	}

	return true, nil
}
