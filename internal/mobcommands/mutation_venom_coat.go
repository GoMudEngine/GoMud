package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// VenomCoat fires the venom-coat mutation for a mob actor. Thin wrapper over
// actions.TriggerVenomCoat.
func VenomCoat(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actor := actions.NewMobActorInRoom(mob, room)
	_ = actions.TriggerVenomCoat(actor, actions.MutationOpts{})
	return true, nil
}
