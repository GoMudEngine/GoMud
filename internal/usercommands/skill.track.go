package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Track is a thin wrapper over actions.Track. The action handles
// rendering, cooldowns, and buff application; this wrapper handles
// "stop"/"clear" keyword normalization and the quest-engine
// command notification.
func Track(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	opts := actions.TrackOptions{TargetNoun: rest}
	if rest == "stop" || rest == "clear" {
		opts = actions.TrackOptions{CancelTracking: true}
	} else if refuseWhileBusy(user, `track`) {
		// Cancelling a track is always allowed — only STARTING one is a
		// focus-required action.
		return true, nil
	}

	actor := &actions.UserActor{User: user, Room: room}
	_ = actions.Track(actor, opts)

	// Quest engine: command notification (preserved from prior behavior).
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "track",
	}, bridge, bridge)

	return true, nil
}
