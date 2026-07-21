package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/moderation"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Petitions is the admin review command for the petition queue.
// Usage: petitions | petitions all | petitions <id> | petitions resolve <id> [note]
func Petitions(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	fields := strings.Fields(rest)

	switch {
	case len(fields) == 0: // list open
		list := moderation.ListOpen()
		if len(list) == 0 {
			user.SendText(messaging.CategorySystem, `<ansi fg="green">No open petitions.</ansi>`)
			return true, nil
		}
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="cyan-bold">%d open petition(s):</ansi>`, len(list)))
		for _, p := range list {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`  <ansi fg="yellow">#%d</ansi> <ansi fg="username">%s</ansi>: %s`, p.Id, p.Reporter, petitionSnippet(p.Message, 60)))
		}
		return true, nil

	case fields[0] == "all":
		list := moderation.ListAll()
		if len(list) == 0 {
			user.SendText(messaging.CategorySystem, `<ansi fg="green">No petitions on record.</ansi>`)
			return true, nil
		}
		for _, p := range list {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`  <ansi fg="yellow">#%d</ansi> [%s] <ansi fg="username">%s</ansi>: %s`, p.Id, p.Status, p.Reporter, petitionSnippet(p.Message, 60)))
		}
		return true, nil

	case fields[0] == "resolve":
		if len(fields) < 2 {
			user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Usage: petitions resolve <id> [note]</ansi>`)
			return true, nil
		}
		id, err := strconv.Atoi(fields[1])
		if err != nil {
			user.SendText(messaging.CategorySystem, `<ansi fg="red">That is not a valid petition id.</ansi>`)
			return true, nil
		}
		note := strings.Join(fields[2:], " ") // everything after "resolve <id>"
		if err := moderation.Resolve(id, user.Username, note); err != nil {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="red">%s</ansi>`, err.Error()))
			return true, nil
		}
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="green">Petition #%d marked resolved.</ansi>`, id))
		return true, nil

	default: // detail by id
		id, err := strconv.Atoi(fields[0])
		if err != nil {
			user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Usage: petitions | petitions all | petitions <id> | petitions resolve <id> [note]</ansi>`)
			return true, nil
		}
		p, ok := moderation.Get(id)
		if !ok {
			user.SendText(messaging.CategorySystem, `<ansi fg="red">No petition with that id.</ansi>`)
			return true, nil
		}
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="cyan-bold">Petition #%d</ansi> [%s]`, p.Id, p.Status))
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`  From: <ansi fg="username">%s</ansi>  When: %s  Room: %d (%s)`, p.Reporter, p.Timestamp.Format("2006-01-02 15:04 MST"), p.RoomId, p.Zone))
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`  Message: %s`, p.Message))
		if p.Status == moderation.StatusResolved {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`  Resolved by <ansi fg="username">%s</ansi>: %s`, p.ResolvedBy, p.Note))
		}
		return true, nil
	}
}

func petitionSnippet(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	r := []rune(s) // rune-slice so a cut never splits a multibyte character
	if len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}
