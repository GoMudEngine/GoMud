package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Sleep is the mob-side sleep verb. Used by the chunk 3.3 schedule
// executor via mob.Command("sleep") when entering an activity: sleeping
// segment. Delegates to actions.Sleep which applies the Sleeping buff
// (buff id 15).
func Sleep(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actions.Sleep(&actions.MobActor{Mob: mob, Room: room}, actions.SleepOptions{})
	return true, nil
}
