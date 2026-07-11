package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Cocoon fires the cocoon mutation for a mob actor. Thin wrapper over
// actions.TriggerCocoon.
func Cocoon(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actor := actions.NewMobActorInRoom(mob, room)
	_ = actions.TriggerCocoon(actor, actions.MutationOpts{})
	return true, nil
}
