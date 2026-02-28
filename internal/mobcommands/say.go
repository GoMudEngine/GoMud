package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Say(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Don't bother if no players are present
	if room.PlayerCt() < 1 {

		return true, nil
	}

	// Wrap long dialogue text to 55 chars to account for "Name says, ..."
	// prefix keeping total line width under 80 for most mob names
	rest = util.NormalizeAndWrap(rest, 55)

	isSneaking := mob.Character.HasBuffFlag(buffs.Hidden)

	if isSneaking {
		room.SendText(fmt.Sprintf(`someone says, "<ansi fg="saytext-mob">%s</ansi>"`, rest))
	} else {
		room.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> says, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, rest))
	}

	events.AddToQueue(events.Communication{
		SourceMobInstanceId: mob.InstanceId,
		CommType:            `say`,
		Name:                mob.Character.Name,
		Message:             rest,
	})

	room.SendTextToExits(`You hear someone talking.`, true)

	return true, nil
}
