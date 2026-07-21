package weather

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/splash"
)

// onSeasonChanged turns a per-zone season flip into a GLOBAL season splash
// (a season turn is world-scale flavor, so everyone sees it — per the design).
func (m *weatherModule) onSeasonChanged(e events.Event) events.ListenerReturn {
	evt, ok := e.(WeatherSeasonChanged)
	if !ok {
		return events.Continue
	}
	events.AddToQueue(splash.Splash{
		SceneId: "season_" + evt.Track + "_" + evt.To, // e.g. season_temperate_winter
		Caption: seasonCaption(evt.To, evt.Zone),
		Target:  splash.TargetGlobal,
		Data:    map[string]any{"zone": evt.Zone},
	})
	return events.Continue
}

func seasonCaption(season, zone string) string {
	switch season {
	case "winter":
		return "Winter descends on " + zone + "."
	case "spring":
		return "Spring returns to " + zone + "."
	case "summer":
		return "Summer settles over " + zone + "."
	case "autumn":
		return "Autumn comes to " + zone + "."
	case "wet":
		return "The wet season breaks over " + zone + "."
	case "dry":
		return "The dry season sets in over " + zone + "."
	}
	return capitalize(season) + " comes to " + zone + "."
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	if s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-32) + s[1:]
	}
	return s
}
