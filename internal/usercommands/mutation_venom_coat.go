package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// VenomCoat fires the venom-coat mutation for a player. Thin wrapper over
// actions.TriggerVenomCoat.
func VenomCoat(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	actor := actions.NewUserActorInRoom(user, room)
	_ = actions.TriggerVenomCoat(actor, actions.MutationOpts{})
	return true, nil
}
