package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Say(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Don't bother if no players are present
	if room.PlayerCt() < 1 {
		return true, nil
	}

	actor := &actions.MobActor{Mob: mob, Room: room}
	result := actions.Say(actor, rest)

	if result.IsSneaking {
		anonMsg := fmt.Sprintf(`someone says, "<ansi fg="saytext-mob">%s</ansi>"`, result.Text)
		room.SendText(util.SplitStringNL(anonMsg, 80))
	} else {
		anonMsg := actions.FormatSayText("", result.Text, true, "mobname", "saytext-mob")
		namedMsg := actions.FormatSayText(mob.Character.Name, result.Text, false, "mobname", "saytext-mob")
		sendAudioRoomText(room, mob, anonMsg, namedMsg)
	}

	return true, nil
}
