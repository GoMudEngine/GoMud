package hooks

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/templates"
)

//
// Listens for MoonPhase events and broadcasts atmospheric messages
// to all players when a Witness crosses a new-moon or full-moon boundary.
//

func BroadcastMoonPhase(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.MoonPhase)
	if !typeOk {
		return events.Cancel
	}

	// Build template key: "generic/moon_<moonslug>_<phase>"
	// e.g. "The Wanderer" + "full" -> "generic/moon_wanderer_full"
	moonSlug := moonNameToSlug(evt.MoonName)
	templatePath := "generic/moon_" + moonSlug + "_" + evt.PhaseName

	txt, _ := templates.Process(templatePath, nil)
	srTxt, _ := templates.Process(templatePath, nil, templates.ForceScreenReaderUserId)

	events.AddToQueue(events.Broadcast{
		Text:             txt,
		TextScreenReader: srTxt,
	})

	return events.Continue
}

// moonNameToSlug converts a moon name to the slug used in template filenames.
// "Swiftmoon" -> "swiftmoon"
// "The Wanderer" -> "wanderer"
// "The Eye" -> "eye"
func moonNameToSlug(name string) string {
	lower := strings.ToLower(name)
	lower = strings.TrimPrefix(lower, "the ")
	lower = strings.ReplaceAll(lower, " ", "_")
	return lower
}
