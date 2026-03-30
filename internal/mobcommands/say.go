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

	isSneaking := mob.Character.HasBuffFlag(buffs.Hidden)

	if isSneaking {
		msg := fmt.Sprintf(`someone says, "<ansi fg="saytext-mob">%s</ansi>"`, rest)
		room.SendText(util.SplitStringNL(msg, 80))
	} else {
		anonMsg := fmt.Sprintf(`someone says, "<ansi fg="saytext-mob">%s</ansi>"`, rest)
		namedMsg := fmt.Sprintf(`<ansi fg="mobname">%s</ansi> says, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, rest)
		sendAudioRoomText(room, mob,
			util.SplitStringNL(anonMsg, 80),
			util.SplitStringNL(namedMsg, 80))
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
