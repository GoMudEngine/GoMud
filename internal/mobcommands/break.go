package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Break(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	if mob.Character.IsInCombat() {
		mob.Character.EndAggro()
		sendRoomText(room, messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> breaks off combat.`, mob.Character.Name))
	}

	return true, nil
}
