package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Search is a thin wrapper over actions.Search. The action handles
// all tier rolls, template rendering, cooldown gating, and skill
// progression.
func Search(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if refuseWhileBusy(user, `search`) {
		return true, nil
	}
	actor := &actions.UserActor{User: user, Room: room}
	_ = actions.Search(actor, actions.SearchOptions{})

	// Quest engine: command notification — lets a quest gate a step on the
	// `search` command (the newbie Lore spoke's shrine beat: searching out the
	// hidden Orbital Stone in the Reliquary). Mirrors forage/drink/throw/cast.
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "search",
	}, bridge, bridge)

	return true, nil
}
