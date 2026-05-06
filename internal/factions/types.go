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
