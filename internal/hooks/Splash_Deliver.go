package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/splash"
	"github.com/GoMudEngine/GoMud/internal/templates"
)

func splashTemplatePath(sceneId string) string { return "generic/splash/" + sceneId }

// Splash_Deliver renders a splash scene to its recipients: the refined ASCII art
// for normal clients (the web client is xterm.js and renders the same ANSI art),
// or the plain caption for screen-reader users.
func Splash_Deliver(e events.Event) events.ListenerReturn {
	evt, ok := e.(splash.Splash)
	if !ok {
		return events.Continue
	}
	if !bool(configs.GetGamePlayConfig().SplashesEnabled) {
		return events.Continue
	}

	// Render the art once — the template output is identical for every recipient
	// (per-recipient line wrapping happens later in SendText).
	text, terr := templates.Process(splashTemplatePath(evt.SceneId), evt.Data)
	artOK := terr == nil && text != ""

	for _, u := range splash.Recipients(evt) {
		// Screen-reader users get the caption only — never the art.
		if u.ScreenReader || !artOK {
			if evt.Caption != "" {
				u.SendText(messaging.CategorySplash, evt.Caption)
			}
			continue
		}
		u.SendText(messaging.CategorySplash, text)
	}
	return events.Continue
}
