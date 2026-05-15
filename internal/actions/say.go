package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// SayResult contains the results of a Say action for the wrapper to use.
type SayResult struct {
	IsSneaking bool
	Text       string
}

// Say handles shared say logic: checks the Hidden buff, fires the exit echo,
// and fires the Communication event. The caller handles: mute checks, drunk
// text, self-message, room formatting, and darkness-aware display.
func Say(actor Actor, text string) SayResult {
	isSneaking := actor.GetCharacter().IsHidden()

	actor.GetRoom().SendTextToExits(`You hear someone talking.`, true)

	events.AddToQueue(events.Communication{
		SourceUserId:        actor.GetUserId(),
		SourceMobInstanceId: actor.GetMobInstanceId(),
		CommType:            `say`,
		Name:                actor.GetName(),
		Message:             text,
	})

	return SayResult{
		IsSneaking: isSneaking,
		Text:       text,
	}
}

// FormatSayText formats the say message for room display.
// nameColor is "username" for players, "mobname" for mobs.
// textColor is "saytext" for players, "saytext-mob" for mobs.
func FormatSayText(name string, text string, isSneaking bool, nameColor string, textColor string) string {
	var msg string
	if isSneaking {
		msg = fmt.Sprintf(`someone says, "<ansi fg="%s">%s</ansi>"`, textColor, text)
	} else {
		msg = fmt.Sprintf(`<ansi fg="%s">%s</ansi> says, "<ansi fg="%s">%s</ansi>"`, nameColor, name, textColor, text)
	}
	return util.SplitStringNL(msg, 80)
}
