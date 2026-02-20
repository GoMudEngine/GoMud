package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
Cast — stub for Phase 11 (fold-based magic system).
The old cast system is disconnected pending the full magic rework.
*/
func Cast(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	user.SendText(`<ansi fg="cyan">You reach for the folds of reality, but the technique escapes you.</ansi>`)
	user.SendText(`<ansi fg="cyan-bold">[The fold-based magic system is coming in Phase 11.]</ansi>`)

	return true, nil
}
