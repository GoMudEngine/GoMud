package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// BlindingSpit fires the blinding-spit mutation for a mob actor. Thin
// wrapper over actions.TriggerBlindingSpit. Resolves the mob's current
// Aggro target into an Actor. If Aggro is nil or the target cannot be
// found, TriggerBlindingSpit returns BlockReason="no-target" silently
// (MobActor.SendText is a no-op).
func BlindingSpit(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	actor := actions.NewMobActorInRoom(mob, room)

	// Resolve the mob's current Aggro target into an Actor.
	var target actions.Actor
	if mob.Character.Aggro != nil {
		if mob.Character.Aggro.MobInstanceId > 0 {
			if victim := mobs.GetInstance(mob.Character.Aggro.MobInstanceId); victim != nil {
				target = actions.NewMobActorInRoom(victim, room)
			}
		} else if mob.Character.Aggro.UserId > 0 {
			if u := users.GetByUserId(mob.Character.Aggro.UserId); u != nil {
				target = actions.NewUserActorInRoom(u, room)
			}
		}
	}

	_ = actions.TriggerBlindingSpit(actor, actions.MutationOpts{TargetActor: target})
	return true, nil
}
