package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

//
// RoomChangeHandler waits for RoomChange events
// Also sends music changes out
//

func CleanupEphemeralRooms(e events.Event) events.ListenerReturn {

	evt := e.(events.RoomChange)

	// If this isn't a user changing rooms, just pass it along.
	if evt.UserId == 0 {
		return events.Continue
	}

	if rooms.IsEphemeralRoomId(evt.FromRoomId) {
		rooms.TryEphemeralCleanup(evt.FromRoomId)
	}

	return events.Continue
}
