package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Buy is the player-side wrapper. All purchase logic lives in
// actions.Buy; this entry point handles the empty-request
// fall-through to List(...) and constructs the buyer Actor.
func Buy(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "" {
		return List(rest, user, room, flags)
	}

	actor := &actions.UserActor{User: user, Room: room}
	actions.Buy(actor, actions.BuyOptions{Request: rest})
	return true, nil
}
