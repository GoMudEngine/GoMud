package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Whisper(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(rest)

	if len(args) < 1 {
		user.SendText(messaging.CategorySystem, "Whisper to who?")
		return true, nil
	}

	whisperName := args[0]
	if len(rest) < len(whisperName)+1 {
		user.SendText(messaging.CategorySystem, "You need to specify a message.")
		return true, nil
	}

	rest = strings.TrimSpace(rest[len(whisperName)+1:])

	toUser := users.GetByCharacterName(whisperName)
	if toUser == nil {
		user.SendText(messaging.CategorySystem, "You can't find anyone by that name.")
		return true, nil
	}

	sourceIsMod := user.Role != users.RoleUser
	targetIsMod := toUser.Role != users.RoleUser

	if user.Muted && !targetIsMod {
		user.SendText(messaging.CategoryWarning, `You are <ansi fg="alert-5">MUTED</ansi>. You can only send <ansi fg="command">whisper</ansi>'s to Admins and Moderators.`)
		return true, nil
	}

	// Whisper do allow special communication between mods/admins and users
	if toUser.Deafened && !sourceIsMod {
		user.SendText(messaging.CategoryWarning, `That user is <ansi fg="alert-5">DEAFENED</ansi> and cannot receive communications from other players.`)
		return true, nil
	}

	// The body sits inside an open <ansi fg="black-bold">, so an unescaped
	// </ansi> would let the sender repaint the rest of the line.
	rest = util.EscapeAnsiTags(rest)

	whisperMsg := fmt.Sprintf(`<ansi fg="white">***</ansi> <ansi fg="black-bold"><ansi fg="username">%s</ansi> whispers, "%s"</ansi> <ansi fg="white">***</ansi>`, user.Character.Name, rest)
	toUser.SendText(messaging.CategoryWhisper, util.SplitStringNL(whisperMsg, 80))

	toUser.LastWhisperFrom = user.UserId

	user.SendText(messaging.CategoryWhisper, fmt.Sprintf(`You sent a <ansi fg="command">whisper</ansi> to <ansi fg="username">%s</ansi>`, toUser.Character.Name))

	events.AddToQueue(events.Communication{
		SourceUserId: user.UserId,
		TargetUserId: toUser.UserId,
		CommType:     `whisper`,
		Name:         user.Character.Name,
		Message:      rest,
	})

	return true, nil
}
