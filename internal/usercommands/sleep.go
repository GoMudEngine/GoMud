package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Sleep is the player-facing sleep verb. Delegates to actions.Sleep
// which applies the Sleeping buff (chunk 3.3, buff id 15).
func Sleep(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	actions.Sleep(&actions.UserActor{User: user, Room: room}, actions.SleepOptions{})
	return true, nil
}
