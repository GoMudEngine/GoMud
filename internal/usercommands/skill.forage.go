package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Forage is a thin wrapper over actions.Forage. The action handles
// biome check, cooldown, score calculation, rendering, and item
// store. This wrapper handles the quest-engine command notification.
func Forage(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if refuseWhileBusy(user, `forage`) {
		return true, nil
	}
	actor := &actions.UserActor{User: user, Room: room}
	_ = actions.Forage(actor, actions.ForageOptions{})

	// Quest engine: command notification (preserved from prior behavior).
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "forage",
	}, bridge, bridge)

	return true, nil
}
