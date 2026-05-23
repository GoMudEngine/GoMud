package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// BlindingSpit is a thin wrapper over actions.TriggerBlindingSpit.
// All gates, scoring, single-target effects, and player messages live
// in the action. Target resolution is delegated to
// actions.ResolveEngagedTarget, which handles both player and mob actors.
func BlindingSpit(rest string, user *users.UserRecord, room *rooms.Room,
	flags events.EventFlag) (bool, error) {

	actor := actions.NewUserActorInRoom(user, room)
	target := actions.ResolveEngagedTarget(actor, room)
	_ = actions.TriggerBlindingSpit(actor, actions.MutationOpts{TargetActor: target})
	return true, nil
}
