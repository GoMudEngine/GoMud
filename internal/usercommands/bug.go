package usercommands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Bug records a player-submitted bug report to a persistent file.
// Usage: bug <description>
// Also accepts typo reports — see help text.
func Bug(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	rest = strings.TrimSpace(rest)
	if rest == `` {
		user.SendText(`<ansi fg="yellow">Usage: <ansi fg="cyan-bold">bug <description></ansi>`)
		user.SendText(`<ansi fg="yellow">Use this command to report bugs or typos you encounter in the game.</ansi>`)
		user.SendText(`<ansi fg="yellow">Your character name, location, and timestamp will be recorded automatically.</ansi>`)
		return true, nil
	}

	feedbackDir := util.FilePath(`_datafiles/feedback`)
	if err := os.MkdirAll(feedbackDir, 0755); err != nil {
		user.SendText(`<ansi fg="red">Could not save bug report. Please notify an admin.</ansi>`)
		return true, nil
	}

	filePath := util.FilePath(`_datafiles/feedback/bugs.txt`)
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		user.SendText(`<ansi fg="red">Could not save bug report. Please notify an admin.</ansi>`)
		return true, nil
	}
	defer f.Close()

	entry := fmt.Sprintf("[%s] %s (zone: %s, room: %d %q): %s\n",
		time.Now().UTC().Format("2006-01-02 15:04:05"),
		user.Character.Name,
		user.Character.Zone,
		user.Character.RoomId,
		room.Title,
		rest,
	)

	if _, err := f.WriteString(entry); err != nil {
		user.SendText(`<ansi fg="red">Could not save bug report. Please notify an admin.</ansi>`)
		return true, nil
	}

	user.SendText(`<ansi fg="green">Bug report submitted. Thank you for helping improve the game!</ansi>`)
	user.SendText(`<ansi fg="yellow">(You can also use <ansi fg="cyan-bold">bug</ansi> to report typos.)</ansi>`)

	return true, nil
}
