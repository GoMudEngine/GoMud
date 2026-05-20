package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Consider(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := util.SplitButRespectQuotes(rest)
	if len(args) == 0 {
		return true, nil
	}

	target, err := actions.ResolveTargetActor(room, args[0],
		actions.ResolveTargetOptions{ExcludeUserId: user.UserId})
	if err != nil {
		// Pre-migration silently no-oped on no-match. On
		// ErrTargetVanished (stale mob ID) the original code DID
		// message "You don't see them here." — preserve that.
		if err == actions.ErrTargetVanished {
			user.SendText(messaging.CategorySystem, "You don't see them here.")
		}
		return true, nil
	}

	actor := &actions.UserActor{User: user, Room: room}
	actions.Consider(actor, target)
	return true, nil
}
