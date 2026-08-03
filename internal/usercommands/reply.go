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

func Reply(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.LastWhisperFrom == 0 {
		user.SendText(messaging.CategorySystem, "No one has whispered to you recently.")
		return true, nil
	}

	if rest == "" {
		user.SendText(messaging.CategorySystem, "Reply with what?")
		return true, nil
	}

	targetUser := users.GetByUserId(user.LastWhisperFrom)
	if targetUser == nil {
		user.SendText(messaging.CategorySystem, "That person is no longer online.")
		user.LastWhisperFrom = 0
		return true, nil
	}

	// Same shape as whisper: the body is inside an open <ansi fg="black-bold">.
	rest = util.EscapeAnsiTags(rest)

	replyMsg := fmt.Sprintf(`<ansi fg="white">***</ansi> <ansi fg="black-bold"><ansi fg="username">%s</ansi> whispers, "%s"</ansi> <ansi fg="white">***</ansi>`, user.Character.Name, rest)
	targetUser.SendText(messaging.CategoryWhisper, util.SplitStringNL(replyMsg, 80))

	// Track reply chain — target can reply back
	targetUser.LastWhisperFrom = user.UserId

	user.SendText(messaging.CategoryWhisper,
		fmt.Sprintf(`You sent a <ansi fg="command">whisper</ansi> to <ansi fg="username">%s</ansi>`, targetUser.Character.Name),
	)

	events.AddToQueue(events.Communication{
		SourceUserId: user.UserId,
		TargetUserId: targetUser.UserId,
		CommType:     `whisper`,
		Name:         user.Character.Name,
		Message:      strings.TrimSpace(rest),
	})

	return true, nil
}
