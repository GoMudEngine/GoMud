package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Rally is the mob-side shout that applies the rally mitigation buff
// to the casting mob. Mob rally applies the self-buff only; ally
// fan-out is a player-command concern.
func Rally(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	result := actions.ExecuteRally(&actions.MobActor{Mob: mob, Room: room})
	if !result.Executed {
		return true, nil
	}

	sendAudioRoomText(room, mob,
		`<ansi fg="cyan-bold">Something lets out a rallying roar!</ansi>`,
		fmt.Sprintf(`<ansi fg="cyan-bold"><ansi fg="mobname">%s</ansi> lets out a rallying roar!</ansi>`, mob.Character.Name))

	return true, nil
}
