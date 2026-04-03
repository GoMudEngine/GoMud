package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Emote(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if len(rest) == 0 {
		user.SendText("You emote.")
		room.SendText(
			fmt.Sprintf(`<ansi fg="username">%s</ansi> emotes.`, user.Character.Name),
			user.UserId,
		)
		return true, nil
	}

	// Emote aliases bypass mute/deafen checks (pre-written, not free-form communication).
	result := actions.Emote(rest)
	if result.IsAlias {
		aliasMsg := actions.FormatEmoteText(user.Character.Name, result.AliasText, "username")
		user.SendText(fmt.Sprintf(`You Emote: %s`, aliasMsg))
		room.SendText(aliasMsg, user.UserId)
		return true, nil
	}

	if user.Muted {
		user.SendText(`You are <ansi fg="alert-5">MUTED</ansi>. You can only send <ansi fg="command">whisper</ansi>'s to Admins and Moderators.`)
		return true, nil
	}

	if rest[0] == '@' && len(rest) > 1 {
		rest = rest[1:]
	} else {
		emoteMsg := actions.FormatEmoteText(user.Character.Name, rest, "username")
		user.SendText(fmt.Sprintf(`You Emote: %s`, emoteMsg))
	}

	room.SendTextCommunication(
		actions.FormatEmoteText(user.Character.Name, rest, "username"),
		user.UserId,
	)

	return true, nil
}
