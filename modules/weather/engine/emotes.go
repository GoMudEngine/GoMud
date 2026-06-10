package engine

import (
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/modules/weather/content"
	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

// EmitAmbient sends one ambient weather line into each occupied room whose
// zone currently has non-calm weather. The room's biome picks the table
// variant; indoor biomes get the intensity-banded indoor section (mild
// weather is silent indoors). roll is the presentation RNG (pass util.Rand)
// — NEVER the sim RNG. Returns lines sent. A nil graph is tolerated: felt
// intensity resolves to 0 (indoor rooms fall to the mild/silent band).
func EmitAmbient(g *sim.Graph, fronts []sim.Front, simCfg sim.Config,
	weather map[sim.ZoneId]sim.WeatherType, tables content.Tables, roll func(int) int) int {

	sent := 0
	felt := map[sim.ZoneId]float64{}

	for _, roomId := range rooms.GetRoomsWithPlayers() {
		room := rooms.LoadRoom(roomId)
		if room == nil {
			continue
		}
		w := weather[room.Zone]
		if w == "" || w == sim.Clear {
			continue
		}

		f, ok := felt[room.Zone]
		if !ok {
			if g != nil {
				if covers := sim.Covering(g, fronts, simCfg, room.Zone); len(covers) > 0 {
					f = covers[0].Effective
				}
			}
			felt[room.Zone] = f
		}

		biomeId, indoor := "", false
		if b := room.GetBiome(); b != nil {
			biomeId, indoor = b.BiomeId, b.Indoor
		}
		line := tables.Pick(w, biomeId, indoor, f, roll)
		if line == "" {
			continue
		}
		room.SendText(messaging.CategoryWeather, line)
		sent++
	}
	return sent
}
