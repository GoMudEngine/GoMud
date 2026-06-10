package engine

import (
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/modules/weather/content"
	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

// EmitAmbient sends one ambient weather line into each occupied room whose
// zone currently has non-calm weather (spec §9.4, EmoteMode "module"). The
// room's biome picks the table variant; indoor biomes get the indoor section.
// roll is the presentation RNG (pass util.Rand) — NEVER the sim RNG, which
// must stay isolated from presentation randomness. Returns lines sent.
func EmitAmbient(weather map[sim.ZoneId]sim.WeatherType, tables content.Tables, roll func(int) int) int {
	sent := 0
	for _, roomId := range rooms.GetRoomsWithPlayers() {
		room := rooms.LoadRoom(roomId)
		if room == nil {
			continue
		}
		w := weather[room.Zone]
		if w == "" || w == sim.Clear {
			continue
		}
		biomeId := ""
		if b := room.GetBiome(); b != nil {
			biomeId = b.BiomeId
		}
		line := tables.Pick(w, biomeId, !isOutdoorBiome(biomeId), roll)
		if line == "" {
			continue
		}
		room.SendText(line)
		sent++
	}
	return sent
}
