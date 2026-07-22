package engine

import (
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/modules/weather/content"
	"github.com/GoMudEngine/GoMud/modules/weather/seasons"
	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

// seasonalEmoteOneIn throttles the seasonal-ambience layer: it fires on
// roughly 1 in N emote passes, only in calm zones — strictly quieter than
// weather (spec S-R1). Promote to config only if play feel demands.
const seasonalEmoteOneIn = 3

// emitChance maps a zone's felt weather intensity (0..1) to the percent chance
// that an ambient weather line fires this pass: mildPct at felt<=0 rising
// linearly to strongPct at felt>=1. Calm weather whispers; severe weather
// speaks steadily. The felt scale is clamp01 (see sim.Coverage.Effective).
func emitChance(mildPct, strongPct int, felt float64) int {
	if felt < 0 {
		felt = 0
	} else if felt > 1 {
		felt = 1
	}
	return mildPct + int(float64(strongPct-mildPct)*felt)
}

// EmitAmbient sends at most ONE ambient line per occupied room per pass. When
// the room's zone has non-calm weather it sends the weather line (season-
// variant aware, indoor intensity-banded by the front's felt intensity); in a
// calm zone it instead — at reduced cadence — sends the zone's seasonal
// ambience. The room's biome picks the table variant; indoor biomes get the
// intensity-banded indoor section (mild weather is silent indoors). zoneSeasons
// and seasonal may be nil/empty when seasons are off or a zone is unbound. roll
// is the presentation RNG (pass util.Rand) — NEVER the sim RNG. Returns lines
// sent. A nil graph is tolerated: felt intensity resolves to 0 (indoor rooms
// fall to the mild/silent band).
func EmitAmbient(g *sim.Graph, fronts []sim.Front, simCfg sim.Config,
	weather map[sim.ZoneId]sim.WeatherType, zoneSeasons map[sim.ZoneId]seasons.ZoneSeason,
	tables content.Tables, seasonal content.SeasonalTables,
	mildChancePct, strongChancePct int, roll func(int) int) int {

	sent := 0
	felt := map[sim.ZoneId]float64{}

	for _, roomId := range rooms.GetRoomsWithPlayers() {
		room := rooms.LoadRoom(roomId)
		if room == nil {
			continue
		}

		biomeId, indoor := "", false
		if b := room.GetBiome(); b != nil {
			biomeId, indoor = b.BiomeId, b.Indoor
		}
		zs, hasSeason := zoneSeasons[room.Zone]

		// Weather wins: one line per room per pass.
		if w := weather[room.Zone]; w != "" && w != sim.Clear {
			f, ok := felt[room.Zone]
			if !ok {
				if g != nil {
					if covers := sim.Covering(g, fronts, simCfg, room.Zone); len(covers) > 0 {
						f = covers[0].Effective
					}
				}
				felt[room.Zone] = f
			}
			// Intensity-scaled cadence: a skipped pass is silence — non-clear
			// weather still owns the room (no seasonal fallback).
			if roll(100) >= emitChance(mildChancePct, strongChancePct, f) {
				continue
			}
			season := ""
			if hasSeason {
				season = zs.Season
			}
			if line := tables.Pick(w, biomeId, indoor, f, season, roll); line != "" {
				room.SendText(messaging.CategoryWeather, line)
				sent++
			}
			continue
		}

		// Calm zone: occasional seasonal ambience. The season is always
		// "felt" (it is the season's persistent voice, not weather through
		// walls), so indoor ambience uses the Strong band.
		if hasSeason && seasonal != nil && roll(seasonalEmoteOneIn) == 0 {
			if line := seasonal.Pick(zs.Track, zs.Season, biomeId, indoor, 1.0, roll); line != "" {
				room.SendText(messaging.CategoryWeather, line)
				sent++
			}
		}
	}
	return sent
}
