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

// Suggest records a player-submitted suggestion to a persistent file.
// Usage: suggest <description>
func Suggest(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	rest = strings.TrimSpace(rest)
	if rest == `` {
		user.SendText(`<ansi fg="yellow">Usage: <ansi fg="cyan-bold">suggest <your idea></ansi>`)
		user.SendText(`<ansi fg="yellow">Use this command to share ideas and suggestions for improving the game.</ansi>`)
		return true, nil
	}

	feedbackDir := util.FilePath(`_datafiles`, `feedback`)
	if err := os.MkdirAll(feedbackDir, 0755); err != nil {
		user.SendText(`<ansi fg="red">Could not save suggestion. Please notify an admin.</ansi>`)
		return true, nil
	}

	filePath := util.FilePath(feedbackDir, `suggestions.txt`)
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		user.SendText(`<ansi fg="red">Could not save suggestion. Please notify an admin.</ansi>`)
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
		user.SendText(`<ansi fg="red">Could not save suggestion. Please notify an admin.</ansi>`)
		return true, nil
	}

	user.SendText(`<ansi fg="green">Suggestion submitted. Thank you for your feedback!</ansi>`)

	return true, nil
}
