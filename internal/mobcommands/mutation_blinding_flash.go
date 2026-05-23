package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// BlindingFlash fires the blinding-flash mutation for a mob actor. Thin
// wrapper over actions.TriggerBlindingFlash. The structured result is not
// consumed here; btree dispatch via try_mutation_active reads it.
func BlindingFlash(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actor := actions.NewMobActorInRoom(mob, room)
	_ = actions.TriggerBlindingFlash(actor, actions.MutationOpts{})
	return true, nil
}
