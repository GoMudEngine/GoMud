package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Emote(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if len(rest) == 0 {
		user.SendText(messaging.CategoryEmote, "You emote.")
		room.SendTextVisual(messaging.CategoryEmote,
			fmt.Sprintf(`<ansi fg="username">%s</ansi> emotes.`, user.Character.Name),
			user.UserId,
		)
		return true, nil
	}

	// Emote aliases bypass mute/deafen checks (pre-written, not free-form communication).
	result := actions.Emote(rest)
	if result.IsAlias {
		aliasMsg := actions.FormatEmoteText(user.Character.Name, result.AliasText, "username")
		user.SendText(messaging.CategoryEmote, fmt.Sprintf(`You Emote: %s`, aliasMsg))
		room.SendTextVisual(messaging.CategoryEmote, aliasMsg, user.UserId)
		return true, nil
	}

	if user.Muted {
		user.SendText(messaging.CategoryWarning, `You are <ansi fg="alert-5">MUTED</ansi>. You can only send <ansi fg="command">whisper</ansi>'s to Admins and Moderators.`)
		return true, nil
	}

	// Neutralise <ansi> markup before interpolation. Only the free-form path
	// is escaped — result.AliasText above comes from the server-side
	// EmoteAliases table and its markup is legitimate.
	rest = util.EscapeAnsiTags(rest)

	if rest[0] == '@' && len(rest) > 1 {
		rest = rest[1:]
	} else {
		emoteMsg := actions.FormatEmoteText(user.Character.Name, rest, "username")
		user.SendText(messaging.CategoryEmote, fmt.Sprintf(`You Emote: %s`, emoteMsg))
	}

	room.SendTextCommunication(
		actions.FormatEmoteText(user.Character.Name, rest, "username"),
		user.UserId,
	)

	return true, nil
}
