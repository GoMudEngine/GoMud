package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
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

	// Loop through all mobs in the room. If any hate the player, try onPlayerDowned()
	skipMobIds := map[int]struct{}{}
	for _, mobInstanceId := range room.GetMobs() {
		mob := mobs.GetInstance(mobInstanceId)
		if mob == nil {
			continue
		}

		if _, ok := skipMobIds[int(mob.MobId)]; ok {
			continue
		}

		if !mob.HasAttackedPlayer(user.UserId) {
			continue
		}

		if isUnique, _ := scripting.TryPlayerDownedEvent(mobInstanceId, user.UserId); isUnique {
			skipMobIds[int(mob.MobId)] = struct{}{}
		}

	}

	return events.Continue
}
