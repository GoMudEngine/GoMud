package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// PacifismAura fires the pacifism-aura mutation for a mob actor. Thin
// wrapper over actions.TriggerPacifismAura. The structured result is not
// consumed here; btree dispatch via try_mutation_active reads it.
func PacifismAura(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actor := actions.NewMobActorInRoom(mob, room)
	_ = actions.TriggerPacifismAura(actor, actions.MutationOpts{})
	return true, nil
}
