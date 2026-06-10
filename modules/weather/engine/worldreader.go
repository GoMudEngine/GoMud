package engine

import (
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/modules/weather/crawler"
)

// isOutdoorBiome reports whether a biome id is outdoors, per the engine's
// biome registry (BiomeInfo.Indoor, set in biome YAML). Unknown or empty
// biomes default to outdoors.
func isOutdoorBiome(biomeID string) bool {
	if b, ok := rooms.GetBiome(biomeID); ok && b != nil {
		return !b.Indoor
	}
	return true
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
