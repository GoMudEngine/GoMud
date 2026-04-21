package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Warcry is the mob-side shout that applies the warcry damage buff to
// the casting mob. Mob warcry applies the self-buff only; ally fan-out
// is a player-command concern.
func Warcry(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	result := actions.ExecuteWarcry(&actions.MobActor{Mob: mob, Room: room})
	if !result.Executed {
		return true, nil
	}

	sendAudioRoomText(room, mob,
		`<ansi fg="red-bold">Something lets out a bone-shaking warcry!</ansi>`,
		fmt.Sprintf(`<ansi fg="red-bold"><ansi fg="mobname">%s</ansi> lets out a bone-shaking warcry!</ansi>`, mob.Character.Name))

	return true, nil
}
