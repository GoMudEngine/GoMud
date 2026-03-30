package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Reply(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.LastWhisperFrom == 0 {
		user.SendText("No one has whispered to you recently.")
		return true, nil
	}

	if rest == "" {
		user.SendText("Reply with what?")
		return true, nil
	}

	targetUser := users.GetByUserId(user.LastWhisperFrom)
	if targetUser == nil {
		user.SendText("That person is no longer online.")
		user.LastWhisperFrom = 0
		return true, nil
	}

	wrappedMsg := util.SplitStringNL(rest, 65)

	targetUser.SendText(
		fmt.Sprintf(`<ansi fg="white">***</ansi> <ansi fg="black-bold"><ansi fg="username">%s</ansi> whispers, "%s"</ansi> <ansi fg="white">***</ansi>`, user.Character.Name, wrappedMsg),
	)

	// Track reply chain — target can reply back
	targetUser.LastWhisperFrom = user.UserId

	user.SendText(
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
