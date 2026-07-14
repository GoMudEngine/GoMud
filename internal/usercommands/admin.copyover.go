package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* copyover			(All)
 */
func Copyover(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	user.SendText(messaging.CategorySystem, `Initiating copyover...`)
	if err := triggerCopyoverFunc(); err != nil {
		mudlog.Error("copyover command", "error", err)
		user.SendText(messaging.CategorySystem, `Copyover failed: `+err.Error())
	}
	return true, nil
}

// triggerCopyoverFunc is set by main at startup to point to the platform-specific
// triggerCopyover helper.
var triggerCopyoverFunc func() error = func() error { return nil }

// SetCopyoverFunc registers the platform-specific copyover trigger with the usercommands package.
func SetCopyoverFunc(f func() error) {
	triggerCopyoverFunc = f
}
