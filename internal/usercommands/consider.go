package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Consider(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := util.SplitButRespectQuotes(rest)
	if len(args) == 0 {
		user.SendText(messaging.CategorySystem, "Consider whom? (try consider <target>)")
		return true, nil
	}

	target, err := actions.ResolveTargetActor(room, args[0],
		actions.ResolveTargetOptions{ExcludeUserId: user.UserId})
	if err != nil {
		// Always give feedback for an unresolved target — dead, absent, or no
		// match — instead of silently no-oping (which left the player unsure
		// whether the command even registered).
		user.SendText(messaging.CategorySystem, "You don't see them here.")
		return true, nil
	}

	actor := &actions.UserActor{User: user, Room: room}
	actions.Consider(actor, target)

	// Quest engine: command notification
	bridge := questengine.NewGameBridge(user, room.RoomId)
	questengine.GetEngine().Notify("command", questengine.EventDetails{
		UserId:  user.UserId,
		RoomId:  room.RoomId,
		Command: "consider",
	}, bridge, bridge)

	return true, nil
}
