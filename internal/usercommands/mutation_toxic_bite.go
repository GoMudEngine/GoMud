package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ToxicBite is a thin wrapper over actions.TriggerToxicBite.
// All gates, scoring, single-target damage, poison application, and player
// messages live in the action. Target resolution is delegated to
// actions.ResolveEngagedTarget, which handles both player and mob actors.
func ToxicBite(rest string, user *users.UserRecord, room *rooms.Room,
	flags events.EventFlag) (bool, error) {

	actor := actions.NewUserActorInRoom(user, room)
	target := actions.ResolveEngagedTarget(actor, room)
	_ = actions.TriggerToxicBite(actor, actions.MutationOpts{TargetActor: target})
	return true, nil
}
