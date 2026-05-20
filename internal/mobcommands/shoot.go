package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Shoot(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	res := actions.ExecuteShoot(&actions.MobActor{Mob: mob, Room: room}, rest)

	if res.NoWeapon || res.BadSyntax || res.NoExit || res.ExitLocked ||
		res.NoTarget || res.IsCharmed || !res.Ready {
		return true, nil
	}

	// Room messaging (mob never sends private text to itself).
	if !res.IsSneaking {
		if res.IsTargetMob {
			sendRoomText(room, messaging.CategoryHitRanged,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> prepares to shoot at <ansi fg="mobname">%s</ansi> through the <ansi fg="exit">%s</ansi> exit.`, mob.Character.Name, res.TargetName, res.ExitName))
		} else {
			sendRoomText(room, messaging.CategoryHitRanged,
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> prepares to shoot at <ansi fg="username">%s</ansi> through the <ansi fg="exit">%s</ansi> exit.`, mob.Character.Name, res.TargetName, res.ExitName))
		}
	}

	return true, nil
}
