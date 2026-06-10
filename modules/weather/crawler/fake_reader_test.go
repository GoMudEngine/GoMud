package crawler

import (
	"sort"
	"testing"
)

// fakeReader is an in-memory WorldReader for tests.
type fakeReader struct {
	biomes    map[string]string // zone -> biome
	rooms     map[int]RoomView  // id -> room
	zoneRooms map[string][]int  // zone -> room ids (insertion order)
}

func newFakeReader() *fakeReader {
	return &fakeReader{
		biomes:    map[string]string{},
		rooms:     map[int]RoomView{},
		zoneRooms: map[string][]int{},
	}
}

// addRoom registers a room in a zone with the given biome, outdoor flag, and
// exits. The first biome seen for a zone wins.
func (f *fakeReader) addRoom(zone, biome string, id int, outdoor bool, exits ...ExitView) {
	if _, ok := f.biomes[zone]; !ok {
		f.biomes[zone] = biome
	}
	f.rooms[id] = RoomView{ID: id, Zone: zone, Outdoor: outdoor, Exits: exits}
	f.zoneRooms[zone] = append(f.zoneRooms[zone], id)
}

func (f *fakeReader) ZoneNames() []string {
	out := make([]string, 0, len(f.biomes))
	for z := range f.biomes {
		out = append(out, z)
	}
	sort.Strings(out)
	return out
}

func (f *fakeReader) ZoneBiome(zone string) string { return f.biomes[zone] }
func (f *fakeReader) RoomIDs(zone string) []int    { return f.zoneRooms[zone] }
func (f *fakeReader) Room(id int) (RoomView, bool) { r, ok := f.rooms[id]; return r, ok }

func TestFakeReaderSatisfiesInterface(t *testing.T) {
	var _ WorldReader = newFakeReader()
}
