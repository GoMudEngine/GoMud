package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/channels"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// sendChannel formats and dispatches a global chat-channel message. It emits a
// ChannelMessage (terminal fan-out) and a Communication (web/GMCP tab).
func sendChannel(user *users.UserRecord, channelName, rest string) (bool, error) {
	ch, ok := channels.Get(channelName)
	if !ok {
		return true, nil
	}

	if user.Muted {
		user.SendText(messaging.CategoryWarning, `You are <ansi fg="alert-5">MUTED</ansi>. You can only send <ansi fg="command">whisper</ansi>'s to Admins and Moderators.`)
		return true, nil
	}

	rest = strings.TrimSpace(rest)
	if rest == `` {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`Usage: <ansi fg="command">%s</ansi> &lt;message&gt;`, ch.Name))
		return true, nil
	}

	text := fmt.Sprintf(`<ansi fg="%s">%s</ansi> <ansi fg="username">%s</ansi>: %s`,
		ch.Color, ch.Prefix, user.Character.Name, rest)

	events.AddToQueue(events.ChannelMessage{
		Channel:      ch.Name,
		SourceUserId: user.UserId,
		Name:         user.Character.Name,
		Text:         text + term.CRLFStr,
	})

	events.AddToQueue(events.Communication{
		SourceUserId: user.UserId,
		CommType:     ch.Name,
		Name:         user.Character.Name,
		Message:      rest,
	})

	return true, nil
}

func Chat(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	return sendChannel(user, "chat", rest)
}

func Newbie(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	return sendChannel(user, "newbie", rest)
}

func Trade(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	return sendChannel(user, "trade", rest)
}

// Channels lists the channels and their on/off state, or toggles one.
//
//	channels             -> list all with state
//	channels <name>      -> toggle
//	channels <name> on   -> enable
//	channels <name> off  -> disable
func Channels(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)

	if len(args) == 0 {
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Channels</ansi> <ansi fg="8">(channels &lt;name&gt; to toggle):</ansi>`)
		for _, ch := range channels.All() {
			state := `<ansi fg="red">off</ansi>`
			if channels.Enabled(user.GetConfigOption(ch.ConfigKey)) {
				state = `<ansi fg="green">on</ansi>`
			}
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`  <ansi fg="%s">%s</ansi> — %s`, ch.Color, ch.Name, state))
		}
		return true, nil
	}

	ch, ok := channels.Get(strings.ToLower(args[0]))
	if !ok {
		names := make([]string, 0, len(channels.All()))
		for _, c := range channels.All() {
			names = append(names, c.Name)
		}
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`No such channel. Valid channels: %s`, strings.Join(names, ", ")))
		return true, nil
	}

	newState := !channels.Enabled(user.GetConfigOption(ch.ConfigKey))
	if len(args) >= 2 {
		switch strings.ToLower(args[1]) {
		case "on":
			newState = true
		case "off":
			newState = false
		}
	}
	user.SetConfigOption(ch.ConfigKey, newState)

	word, color := "off", "red"
	if newState {
		word, color = "on", "green"
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`Channel <ansi fg="%s">%s</ansi> is now <ansi fg="%s">%s</ansi>.`, ch.Color, ch.Name, color, word))
	return true, nil
}
