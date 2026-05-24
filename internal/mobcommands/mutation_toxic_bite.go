package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// ToxicBite fires the toxic-bite mutation for a mob actor. Thin wrapper
// over actions.TriggerToxicBite. Target resolution is delegated to
// actions.ResolveEngagedTarget, which consults CurrentCombatTarget
// (CombatPhase FSM + Aggro fallback) for a unified polymorphic lookup.
func ToxicBite(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actor := actions.NewMobActorInRoom(mob, room)
	target := actions.ResolveEngagedTarget(actor, room)
	_ = actions.TriggerToxicBite(actor, actions.MutationOpts{TargetActor: target})
	return true, nil
}
