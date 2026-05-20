package usercommands

import (
	"regexp"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// motdAnsiTagRE strips <ansi …> / </ansi> tags for visible-width math
// when padding MOTD lines. Mirrors the pattern in messaging/wrap.go.
var motdAnsiTagRE = regexp.MustCompile(`<ansi[^>]*>|</ansi>`)

func motdVisibleWidth(s string) int {
	return len(motdAnsiTagRE.ReplaceAllString(s, ""))
}

func Motd(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	m := configs.GetServerConfig().Motd.String()
	text, err := templates.ProcessText(m, nil)
	if err != nil {
		text = m
	}

	// Box outer width scales with the player's configured line width.
	// Layout: " " + "║" + " " + <content padded to padWidth> + "║"
	//          1     1     1          ...                       1   = lw
	// So padWidth = lw - 4 and wrap width = padWidth - 1 (leave a
	// trailing space gutter inside the right border).
	lw := user.GetLineWidth()
	if lw < 20 {
		lw = 20 // sanity floor; below this the box can't render
	}
	innerWidth := lw - 2  // chars between the ║ borders
	padWidth := lw - 4    // ` ║ ` (3) + `║` (1) = 4 frame chars per line
	wrapWidth := padWidth - 1

	wrapped := strings.Split(messaging.WrapAnsi(text, wrapWidth), "\n")

	horiz := strings.Repeat("═", innerWidth)

	// Title row: pad the visible title to padWidth (ANSI tags don't
	// count toward width — we add them after measuring).
	title := `.:  M E S S A G E   O F   T H E   D A Y`
	// Legacy layout was ` ║  TITLE...║` — content cell already opens with
	// one space (the ` ║ ` frame), so we add 1 more space inside to match
	// the legacy 2-space leading indent.
	titleInner := " " + title
	if len(titleInner) > padWidth {
		titleInner = titleInner[:padWidth]
	}
	titlePadded := titleInner + strings.Repeat(" ", padWidth-len(titleInner))
	// Re-insert the ansi styling around the visible title only.
	titleStyled := strings.Replace(titlePadded, title, `<ansi fg="cyan-bold">`+title+`</ansi>`, 1)

	var output string
	output += `<ansi fg="yellow"> ╔` + horiz + `╗` + "\n"
	output += ` ║ ` + titleStyled + `║` + "\n"
	output += ` ╠` + horiz + `╣` + "\n"
	for _, line := range wrapped {
		// Pad each line to padWidth visible chars inside the box.
		// ANSI tag bytes don't count toward width.
		padLen := padWidth - motdVisibleWidth(line)
		if padLen < 0 {
			padLen = 0
		}
		output += ` ║ ` + line + strings.Repeat(" ", padLen) + `║` + "\n"
	}
	output += ` ╚` + horiz + `╝</ansi>` + "\n"

	user.SendText(messaging.CategoryBroadcast, output)

	return true, nil
}

