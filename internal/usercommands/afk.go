package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/presence"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func AFK(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.Character == nil || user.Character.Presence == nil {
		return true, nil // safety
	}

	currentState := user.Character.Presence.State()

	// Toggle off if already AFK (manual) and no message argument.
	if currentState == presence.AFK && rest == "" {
		if d, ok := user.Character.Presence.AFKData(); ok && d.Manual {
			_ = user.Character.Presence.TransitionTo(presence.Active,
				state.TransitionReason{Trigger: presence.TriggerInputReceived})
			user.SendText(`You are no longer AFK.`)
			room.SendTextVisual(fmt.Sprintf(
				`<ansi fg="username">%s</ansi> is back.`,
				user.Character.Name), user.UserId)
			return true, nil
		}
	}

	// Set AFK with optional message.
	msg := strings.TrimSpace(rest)
	_ = user.Character.Presence.TransitionToAFK(
		presence.AFKData{Message: msg, Manual: true},
		state.TransitionReason{Trigger: presence.TriggerManualAFK})

	if msg != "" {
		user.SendText(fmt.Sprintf(`You are now AFK: %s`, msg))
		room.SendTextVisual(fmt.Sprintf(
			`<ansi fg="username">%s</ansi> goes AFK: %s`,
			user.Character.Name, msg), user.UserId)
	} else {
		user.SendText(`You are now AFK. Type <ansi fg="command">afk</ansi> again to return.`)
		room.SendTextVisual(fmt.Sprintf(
			`<ansi fg="username">%s</ansi> goes AFK.`,
			user.Character.Name), user.UserId)
	}

	return true, nil
}
