package mobcommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Rally is the mob-side shout that applies the rally mitigation buff to
// the casting mob. (Unlike the player version, mobs don't rally allies —
// their "allies" are the summoner's party, which is out of scope here.)
func Rally(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	result := actions.ExecuteRally(&actions.MobActor{Mob: mob, Room: room})
	if !result.Executed {
		return true, nil
	}

	room.SendText(fmt.Sprintf(
		`<ansi fg="cyan-bold"><ansi fg="mobname">%s</ansi> lets out a rallying roar!</ansi>`,
		mob.Character.Name,
	))

	return true, nil
}
