package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/splash"
)

//
// Fires a global sunrise/sunset splash scene each day-night boundary.
//

func NotifySunriseSunset(e events.Event) events.ListenerReturn {
	evt, typeOk := e.(events.DayNightCycle)
	if !typeOk {
		return events.Cancel
	}

	if evt.IsSunrise {
		events.AddToQueue(splash.Splash{
			SceneId: "sunrise",
			Caption: "The sun rises.",
			Target:  splash.TargetGlobal,
			Data:    gametime.GetDate(),
		})
		return events.Continue
	}

	events.AddToQueue(splash.Splash{
		SceneId: "sunset",
		Caption: "The sun sets. Night draws in.",
		Target:  splash.TargetGlobal,
		Data:    gametime.GetDate(),
	})
	return events.Continue
}
