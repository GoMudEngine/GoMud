package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// HealingGel fires the healing-gel mutation for a mob actor. Self-only,
// works out of combat. Thin wrapper over actions.TriggerHealingGel.
func HealingGel(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actor := actions.NewMobActorInRoom(mob, room)
	_ = actions.TriggerHealingGel(actor, actions.MutationOpts{})
	return true, nil
}
