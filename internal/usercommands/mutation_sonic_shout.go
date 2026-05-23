package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// SonicShout is a thin wrapper over actions.TriggerSonicShout.
// All gates, scoring, AoE effects, and player messages live in the action.
func SonicShout(rest string, user *users.UserRecord, room *rooms.Room,
	flags events.EventFlag) (bool, error) {

	actor := actions.NewUserActorInRoom(user, room)
	_ = actions.TriggerSonicShout(actor, actions.MutationOpts{})
	return true, nil
}
