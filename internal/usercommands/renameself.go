package usercommands

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// RenameSelf lets a player change their character name once per
// CharacterRenameCooldownHours. Runs a yes/no default-no confirmation
// before applying the change.
func RenameSelf(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	newName := strings.TrimSpace(rest)
	if newName == `` {
		infoOutput, _ := templates.Process("help/rename", nil, user.UserId)
		user.SendText(infoOutput)
		return true, nil
	}

	if strings.EqualFold(newName, user.Character.Name) {
		user.SendText(`That's already your name.`)
		return true, nil
	}

	cooldownHours := int(configs.GetBalanceConfig().CharacterRenameCooldownHours)
	if cooldownHours > 0 && !user.LastRenameAt.IsZero() {
		elapsed := time.Since(user.LastRenameAt)
		cooldown := time.Duration(cooldownHours) * time.Hour
		if elapsed < cooldown {
			nextAt := user.LastRenameAt.Add(cooldown)
			user.SendText(fmt.Sprintf(
				`You renamed yourself recently. You can rename again on %s.`,
				nextAt.Format(`2006-01-02 at 15:04`)))
			return true, nil
		}
	}

	if err := users.ValidateActorName(newName, users.ValidateActorOpts{ExcludeUserId: user.UserId}); err != nil {
		user.SendText(fmt.Sprintf(`That name won't work: %s`, err.Error()))
		return true, nil
	}

	cmdPrompt, _ := user.StartPrompt(`rename`, rest)

	days := cooldownHours / 24
	var confirmText string
	if cooldownHours == 0 {
		confirmText = fmt.Sprintf(
			`Rename yourself from <ansi fg="username">%s</ansi> to <ansi fg="username">%s</ansi>?`,
			user.Character.Name, newName)
	} else {
		confirmText = fmt.Sprintf(
			`Rename yourself from <ansi fg="username">%s</ansi> to <ansi fg="username">%s</ansi>?`+
				` You won't be able to rename again for %d days.`,
			user.Character.Name, newName, days)
	}

	q := cmdPrompt.Ask(confirmText, []string{`yes`, `no`}, `no`)
	if !q.Done {
		return true, nil
	}
	if q.Response != `yes` {
		user.SendText(`Aborted.`)
		user.ClearPrompt()
		return true, nil
	}

	oldName := user.Character.Name
	if err := users.RenameUser(user, newName); err != nil {
		user.SendText(fmt.Sprintf(`Rename failed: %s`, err.Error()))
		user.ClearPrompt()
		return true, err
	}

	user.LastRenameAt = time.Now()
	users.SaveUser(*user)

	user.EventLog.Add(`char`, fmt.Sprintf(
		`Renamed from <ansi fg="username">%s</ansi> to <ansi fg="username">%s</ansi>`,
		oldName, newName))

	user.SendText(`The world ripples briefly — you are now known as <ansi fg="username">` + newName + `</ansi>.`)
	room.SendTextVisual(
		fmt.Sprintf(`<ansi fg="username">%s</ansi> shimmers and is now known as <ansi fg="username">%s</ansi>.`,
			oldName, newName),
		user.UserId)

	user.ClearPrompt()
	return true, nil
}
