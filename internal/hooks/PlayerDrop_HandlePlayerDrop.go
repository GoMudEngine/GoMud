package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

//
// Some clean up
//

func HandlePlayerDrop(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PlayerDrop)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "PlayerDrop", "Actual Type", e.Type())
		return events.Cancel
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil {
		mudlog.Error("HandlePlayerDrop", "error", fmt.Sprintf(`user %d not found`, evt.UserId))
		return events.Cancel
	}

	user.Character.DownedRounds = 0

	user.SendText(`<ansi fg="red">you drop to the ground!</ansi>`)

	room := rooms.LoadRoom(evt.RoomId)
	if room == nil {
		return events.Continue
	}

	sendVisualRoomText(room, 
		fmt.Sprintf(`<ansi fg="username">%s</ansi> <ansi fg="red">drops to the ground!</ansi>`, user.Character.Name),
		user.UserId)

	return events.Continue
}
