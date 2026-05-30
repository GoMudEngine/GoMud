package factions

// Definition is one faction's authored content. Loaded eagerly at
// startup from _datafiles/world/dogmud/factions/{slug}.yaml.
//
// Definitions are immutable after load. Allies/Enemies are
// declarative — consumers may read the graph, but rep changes do
// NOT auto-propagate through it (per chunk 1.2 design).
type Definition struct {
	FactionId   string   `yaml:"faction_id"`
	DisplayName string   `yaml:"display_name"`
	Description string   `yaml:"description"`
	DefaultRep  int      `yaml:"default_rep"`
	Allies      []string `yaml:"allies"`
	Enemies     []string `yaml:"enemies"`
	// HoldingCellRoom is the room id a guard of this faction hauls arrestees
	// to. 0 / omitted means this faction has no jail (correct for citizen and
	// non-guard factions). Read by internal/justice.
	HoldingCellRoom int `yaml:"holding_cell_room"`
	// ReleaseRoom is the room a freed prisoner of this faction is released to
	// after serving a sentence or paying a fine. 0 / omitted means no
	// faction-specific release room; internal/justice falls back to the
	// hardcoded barracksRoomId default.
	ReleaseRoom int `yaml:"release_room"`
}

// RepEntry is one player's rep with one faction.
type RepEntry struct {
	Rep              int    `yaml:"rep"`
	LastUpdatedRound uint64 `yaml:"last_updated_round"`
}

// FactionRep is a faction's full per-player rep table. Persisted
// to _datafiles/world/dogmud/factions.rep/{slug}.yaml (gitignored).
type FactionRep struct {
	FactionId string            `yaml:"faction_id"`
	Players   map[int]*RepEntry `yaml:"players"`
}

// Score range — every Set/Bump clamps to this window.
const (
	RepMin = -100
	RepMax = +100
)
