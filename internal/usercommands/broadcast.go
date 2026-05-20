package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Global chat room
func Broadcast(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.Muted {
		user.SendText(messaging.CategoryWarning, `You are <ansi fg="alert-5">MUTED</ansi>. You can only send <ansi fg="command">whisper</ansi>'s to Admins and Moderators.`)
		return true, nil
	}

	msg := fmt.Sprintf(`<ansi fg="broadcast-prefix">(broadcast)</ansi> <ansi fg="username">%s</ansi>: <ansi fg="broadcast-body">%s</ansi>`, user.Character.Name, rest)

	events.AddToQueue(events.Broadcast{
		Text:            msg + term.CRLFStr,
		IsCommunication: true,
		SourceIsMod:     user.Role != users.RoleUser,
	})

	events.AddToQueue(events.Communication{
		SourceUserId: user.UserId,
		CommType:     `broadcast`,
		Name:         user.Character.Name,
		Message:      rest,
	})

	return true, nil
}
