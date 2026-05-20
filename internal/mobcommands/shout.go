package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Shout(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	isSneaking := mob.Character.IsHidden()

	if isSneaking {
		msg := fmt.Sprintf(`someone shouts, "<ansi fg="saytext-mob">%s</ansi>"`, rest)
		room.SendText(messaging.CategoryShout, util.SplitStringNL(msg, 80))
	} else {
		anonMsg := fmt.Sprintf(`someone shouts, "<ansi fg="saytext-mob">%s</ansi>"`, rest)
		namedMsg := fmt.Sprintf(`<ansi fg="mobname">%s</ansi> shouts, "<ansi fg="saytext-mob">%s</ansi>"`, mob.Character.Name, rest)
		sendAudioRoomText(room, mob, messaging.CategoryShout,
			util.SplitStringNL(anonMsg, 80),
			util.SplitStringNL(namedMsg, 80))
	}

	for _, roomInfo := range room.Exits {
		if otherRoom := rooms.LoadRoom(roomInfo.RoomId); otherRoom != nil {
			if sourceExit := otherRoom.FindExitTo(room.RoomId); sourceExit != `` {
				otherRoom.SendText(messaging.CategoryShout, fmt.Sprintf(`Someone is shouting from the <ansi fg="exit">%s</ansi> direction.`, sourceExit))
			}
		}
	}

	return true, nil
}
