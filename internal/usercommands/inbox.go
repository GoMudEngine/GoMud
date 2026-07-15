package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/language"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Inbox(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == `clear` {
		user.Inbox.Empty()
	}

	if rest == `check` {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(language.T(`Inbox.UnreadMessageWithCheck`), user.Inbox.CountUnread(), user.Inbox.CountRead()))
		return true, nil
	}

	user.SendText(messaging.CategorySystem, fmt.Sprintf(language.T(`Inbox.UnreadMessage`), user.Inbox.CountUnread(), user.Inbox.CountRead()))

	if len(user.Inbox) == 0 {
		return true, nil
	}

	border := `<ansi fg="mail-border">` + strings.Repeat(`_`, user.GetLineWidth()) + `</ansi>`
	user.SendText(messaging.CategorySystem, border)

	for idx, msg := range user.Inbox {

		if rest == `old` {
			if !msg.Read {
				continue
			}
		} else if msg.Read {
			continue
		}

		tplTxt, _ := templates.Process("mail/message", msg, user.UserId)
		user.SendText(messaging.CategorySystem, tplTxt)

		user.SendText(messaging.CategorySystem, border)

		if !msg.Read {
			if !applyMailReceipt(user, &user.Inbox[idx]) {
				// Attached item won't fit — keep the message unread so nothing is
				// lost; the reader frees space and checks mail again. (The message
				// body + border were already printed above this block.)
				user.SendText(messaging.CategorySystem,
					fmt.Sprintf(`Your pack is too full to receive the <ansi fg="item">%s</ansi> — free some space and check your mail again.`, msg.Item.DisplayName()))
				continue
			}
		}

		user.Inbox[idx].Read = true
	}

	user.SendText(messaging.CategorySystem, ``)
	user.SendText(messaging.CategorySystem, language.T(`Inbox.ReadOldMessages`))
	user.SendText(messaging.CategorySystem, language.T(`Inbox.ClearMessages`))
	user.SendText(messaging.CategorySystem, ``)

	return true, nil
}
