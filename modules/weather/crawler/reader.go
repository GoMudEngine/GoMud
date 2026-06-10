package crawler

// WorldReader is the minimal, engine-agnostic view of the world the crawler
// needs to build the geography graph. The live implementation (in a separate,
// checkout-only package) wraps internal/rooms; tests use an in-memory fake.
type WorldReader interface {
	// ZoneNames returns every zone in the world.
	ZoneNames() []string
	// ZoneBiome returns the default biome for a zone, or "" if unknown.
	ZoneBiome(zone string) string
	// RoomIDs returns the ids of the rooms belonging to a zone.
	RoomIDs(zone string) []int
	// Room returns a read-only snapshot of a room, or ok=false if it can't be
	// loaded (e.g. a dangling exit target).
	Room(id int) (RoomView, bool)
}

// RoomView is a read-only snapshot of the room facts the crawler uses.
type RoomView struct {
	ID int
	// Zone is the room's home zone. The crawler attributes rooms to zones via
	// WorldReader.RoomIDs, not this field; Zone is provided for callers that
	// consume a RoomView directly.
	Zone    string
	Outdoor bool
	Exits   []ExitView
}

// ExitView is a single exit from a room to a destination room id. Secret
// records whether the exit is hidden; the crawler decides whether to count it
// via Options.IncludeSecretExits.
type ExitView struct {
	ToRoom int
	Secret bool
}
