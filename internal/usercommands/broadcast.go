package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Broadcast is retained as an alias for the chat channel so existing muscle
// memory (and the web "broadcast" send path) keeps working, while moving player
// chat off the system-announcement stream.
func Broadcast(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	return sendChannel(user, "chat", rest)
}
