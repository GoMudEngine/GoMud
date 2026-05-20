package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// DeleteCharacter permanently deletes the player's account. Two-stage
// confirmation: yes/no (default no) followed by case-sensitive
// type-the-name match.
func DeleteCharacter(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	cmdPrompt, _ := user.StartPrompt(`deletecharacter`, rest)

	q1 := cmdPrompt.Ask(
		`<ansi fg="red">This will permanently delete your account and free your name`+
			` for someone else to claim. This cannot be undone.</ansi> Continue?`,
		[]string{`yes`, `no`}, `no`)
	if !q1.Done {
		return true, nil
	}
	if q1.Response != `yes` {
		user.SendText(messaging.CategorySystem, `Aborted.`)
		user.ClearPrompt()
		return true, nil
	}

	q2 := cmdPrompt.Ask(
		fmt.Sprintf(`To confirm, type your character's name exactly: <ansi fg="username">%s</ansi>`,
			user.Character.Name),
		[]string{})
	if !q2.Done {
		return true, nil
	}
	if q2.Response != user.Character.Name { // case-sensitive
		user.SendText(messaging.CategorySystem, `That doesn't match. Aborted.`)
		user.ClearPrompt()
		return true, nil
	}

	oldName := user.Character.Name
	user.EventLog.Add(`char`, `Account deleted by user.`)

	room.SendTextVisual(messaging.CategoryMobEmote, 
		fmt.Sprintf(`<ansi fg="username">%s</ansi>'s form dissolves into shimmering dust.`, oldName),
		user.UserId)
	user.SendText(messaging.CategorySystem, fmt.Sprintf(
		`Your form dissolves into shimmering dust. Farewell, <ansi fg="username">%s</ansi>.`,
		oldName))

	if err := users.RemoveUserAndDisconnect(user.UserId); err != nil {
		mudlog.Error(`deletecharacter`, `error`, err, `userId`, user.UserId)
	}

	return true, nil
}
