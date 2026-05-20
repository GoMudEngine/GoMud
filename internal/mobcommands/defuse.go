package mobcommands

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Defuse lets a mob attempt to disarm a trap on a room exit or
// container lock. The mob command CLI accepts the exit or container
// name as the argument. Requires skullduggery rank 3 and a disarm
// kit in inventory (same gates as the player command).
func Defuse(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actor := &actions.MobActor{Mob: mob, Room: room}
	actions.Defuse(actor, actions.DefuseOptions{
		TargetNoun: strings.TrimSpace(strings.ToLower(rest)),
	})
	return true, nil
}
