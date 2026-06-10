package engine

import (
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/modules/weather/crawler"
)

// indoorBiomes are biome ids treated as indoors/underground, so the crawler
// records their zones as having no outdoor rooms. GoMud has no explicit
// indoor/outdoor room flag, so this is a heuristic; a later milestone can make
// the set configurable when weather presentation needs finer control.
var indoorBiomes = map[string]bool{
	"cave":        true,
	"underground": true,
	"dungeon":     true,
	"indoor":      true,
	"tunnel":      true,
	"sewer":       true,
}

// isOutdoorBiome reports whether a biome id is considered outdoors. An unknown
// or empty biome defaults to outdoors.
func isOutdoorBiome(biomeID string) bool {
	return !indoorBiomes[biomeID]
}

// WorldReader implements crawler.WorldReader over the live GoMud engine.
type WorldReader struct{}

// NewWorldReader returns a crawler.WorldReader backed by internal/rooms.
func NewWorldReader() crawler.WorldReader { return WorldReader{} }

func (WorldReader) ZoneNames() []string { return rooms.GetAllZoneNames() }

func (WorldReader) ZoneBiome(zone string) string { return rooms.GetZoneBiome(zone) }

func (WorldReader) RoomIDs(zone string) []int { return rooms.GetAllZoneRoomsIds(zone) }

func (WorldReader) Room(id int) (crawler.RoomView, bool) {
	r := rooms.LoadRoom(id)
	if r == nil {
		return crawler.RoomView{}, false
	}
	exits := make([]crawler.ExitView, 0, len(r.Exits))
	for _, ex := range r.Exits {
		exits = append(exits, crawler.ExitView{ToRoom: ex.RoomId, Secret: ex.Secret})
	}
	biomeID := ""
	if b := r.GetBiome(); b != nil {
		biomeID = b.BiomeId
	}
	return crawler.RoomView{
		ID:      r.RoomId,
		Zone:    r.Zone,
		Outdoor: isOutdoorBiome(biomeID),
		Exits:   exits,
	}, true
}
