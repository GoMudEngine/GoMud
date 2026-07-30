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

	// A shop whose every keeper is asleep is shut. 113 of 132 schedules put the
	// NPC to sleep IN their own workplace, so without this a player could trade
	// at 3am with the keeper snoring in the corner.
	if asleep := actions.ShopClosedForSleep(room); asleep != nil {
		actions.RefuseMobIfAsleep(asleep, user)
		return true, nil
	}

	actor := &actions.UserActor{User: user, Room: room}
	actions.Buy(actor, actions.BuyOptions{Request: rest})
	return true, nil
}
