package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Scan is a thin wrapper over actions.Scan. The action handles all
// rendering for player actors via SendText; this wrapper just
// constructs the actor + opts.
func Scan(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	actor := &actions.UserActor{User: user, Room: room}
	_ = actions.Scan(actor, actions.ScanOptions{HostileOnly: false})
	return true, nil
}
