package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Consume lets a mob eat a corpse in the room to gain a regeneration buff.
// This is an idle/out-of-combat command — no aggro requirement.
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

	// Remove the corpse
	room.Corpses = append(room.Corpses[:corpseIdx], room.Corpses[corpseIdx+1:]...)

	// Apply ConditionRegen: magnitude 2.0 (2x base regen) for 6 rounds
	mob.Character.AddCondition(characters.ConditionRegen, 6, 2.0, "consumed corpse")

	// Room message
	room.SendText(
		fmt.Sprintf(`<ansi fg="mobname">%s</ansi> tears into a corpse and feeds greedily!`, mob.Character.Name),
	)

	return true, nil
}
