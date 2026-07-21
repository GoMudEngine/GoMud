package weather

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/splash"
	"github.com/GoMudEngine/GoMud/modules/weather/sim"
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

// severeScene maps the three severe weather types to their splash scene ids.
// Only these trigger a splash (per design — not rain/fog/heatwave/etc).
func severeScene(w sim.WeatherType) (string, bool) {
	switch w {
	case "storm":
		return "weather_storm", true
	case "blizzard":
		return "weather_blizzard", true
	case "dust":
		return "weather_dust", true
	}
	return "", false
}

func severeCaption(w sim.WeatherType, zone string) string {
	switch w {
	case "storm":
		return "A storm breaks over " + zone + "."
	case "blizzard":
		return "A blizzard howls across " + zone + "."
	case "dust":
		return "A dust storm scours " + zone + "."
	}
	return "The weather turns violent over " + zone + "."
}

// emitSevereOnsets fires one ZONE-scoped splash when a zone TRANSITIONS INTO a
// severe weather type. It does not re-announce while the weather persists, and
// says nothing when it clears.
func (m *weatherModule) emitSevereOnsets(prev map[sim.ZoneId]sim.WeatherType) {
	for zone, w := range m.state.Weather {
		scene, severe := severeScene(w)
		if !severe {
			continue
		}
		if prev[zone] == w {
			continue // already this severe type last tick — no re-announce
		}
		events.AddToQueue(splash.Splash{
			SceneId: scene,
			Caption: severeCaption(w, zone),
			Target:  splash.TargetZone,
			Zone:    zone, // ZoneId is the zone name
			Data:    map[string]any{"zone": zone},
		})
	}
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
