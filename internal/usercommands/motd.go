package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Motd(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	m := configs.GetServerConfig().Motd.String()
	text, err := templates.ProcessText(m, nil)
	if err != nil {
		text = m
	}

	// Wrap lines at ~74 chars to fit inside the box (80 - 4 for borders + padding)
	wrapped := wrapText(text, 74)

	var output string
	output += `<ansi fg="yellow"> ╔══════════════════════════════════════════════════════════════════════════════╗` + "\n"
	output += ` ║  <ansi fg="cyan-bold">.:  M E S S A G E   O F   T H E   D A Y</ansi>` + `                                    ║` + "\n"
	output += ` ╠══════════════════════════════════════════════════════════════════════════════╣` + "\n"
	for _, line := range wrapped {
		// Pad each line to 76 chars inside the box
		padded := line
		for len(padded) < 76 {
			padded += " "
		}
		output += ` ║ ` + padded + `║` + "\n"
	}
	output += ` ╚══════════════════════════════════════════════════════════════════════════════╝</ansi>` + "\n"

	user.SendText(output)

	return true, nil
}

// wrapText splits text into lines of at most maxWidth characters,
// breaking at spaces when possible.
func wrapText(text string, maxWidth int) []string {
	words := []string{}
	current := ""
	for _, ch := range text {
		if ch == ' ' || ch == '\n' {
			if current != "" {
				words = append(words, current)
				current = ""
			}
			if ch == '\n' {
				words = append(words, "\n")
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		words = append(words, current)
	}

	lines := []string{}
	line := ""
	for _, word := range words {
		if word == "\n" {
			lines = append(lines, line)
			line = ""
			continue
		}
		if line == "" {
			line = word
		} else if len(line)+1+len(word) <= maxWidth {
			line += " " + word
		} else {
			lines = append(lines, line)
			line = word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, "")
	}
	return lines
}
