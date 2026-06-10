package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Reload lets a mob chamber/nock a projectile into its equipped ranged
// weapon, consuming the shared special-move cooldown and one use from a
// matching ammo bundle. Early exits are silent (no player connection).
func Reload(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	res := actions.ExecuteReload(&actions.MobActor{Mob: mob, Room: room})

	// Only a successful reload produces a room message; everything else is
	// silent for mobs.
	if !res.Loaded {
		return true, nil
	}

	sendRoomText(room, messaging.CategoryRoomDescription,
		fmt.Sprintf(`<ansi fg="mobname">%s</ansi> readies their <ansi fg="itemname">%s</ansi>.`,
			mob.Character.Name, res.WeaponName))

	return true, nil
}
