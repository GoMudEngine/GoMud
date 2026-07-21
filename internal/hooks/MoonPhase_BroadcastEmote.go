package hooks

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/splash"
)

//
// Listens for MoonPhase events and fires a global splash scene when a Witness
// crosses a new-moon or full-moon boundary.
//

func BroadcastMoonPhase(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.MoonPhase)
	if !typeOk {
		return events.Cancel
	}

	// Scene id: "moon_<moonslug>_<phase>" e.g. "The Wanderer"+"full" ->
	// "moon_wanderer_full" (the delivery hook prefixes "generic/splash/").
	sceneId := "moon_" + moonNameToSlug(evt.MoonName) + "_" + evt.PhaseName

	events.AddToQueue(splash.Splash{
		SceneId: sceneId,
		Caption: evt.MoonName + " is " + evt.PhaseName + ".",
		Target:  splash.TargetGlobal,
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
