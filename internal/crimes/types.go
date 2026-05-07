package crimes

// Kind classifies a crime. Three kinds are in scope for chunk 1.3:
// assault (attack on a faction member), murder (kill of a faction
// member), theft (failed steal/pickpocket of a faction member).
type Kind string

const (
	KindAssault Kind = "assault"
	KindMurder  Kind = "murder"
	KindTheft   Kind = "theft"
)

// PerpType describes who committed a crime — a player, a mob (for
// future mob-on-mob recording), or unknown when no witness was
// present to identify them.
type PerpType string

const (
	PerpPlayer  PerpType = "player"
	PerpMob     PerpType = "mob"
	PerpUnknown PerpType = "unknown"
)

// Perpetrator identifies who committed a crime. Type=PerpUnknown
// means the act happened but no witness was present to name a
// perpetrator. Id holds the userId or mobId; absent for unknown.
type Perpetrator struct {
	Type PerpType `yaml:"type"`
	Id   int      `yaml:"id,omitempty"`
}

// Crime is one row in a faction's log.
type Crime struct {
	Id               int         `yaml:"id"`
	Kind             Kind        `yaml:"kind"`
	Zone             string      `yaml:"zone"`
	RoomId           int         `yaml:"room_id"`
	Round            uint64      `yaml:"round"`
	VictimMobId      int         `yaml:"victim_mob_id"`
	VictimInstanceId int         `yaml:"victim_instance_id"`
	Perpetrator      Perpetrator `yaml:"perpetrator"`
	ResolvedRound    uint64      `yaml:"resolved_round"`
	ResolvedBy       string      `yaml:"resolved_by"`
}

// FactionCrimes is one faction's full crime log. Persisted to
// _datafiles/world/dogmud/factions.crimes/{slug}.yaml (gitignored).
type FactionCrimes struct {
	FactionId string   `yaml:"faction_id"`
	Crimes    []*Crime `yaml:"crimes"`
	nextId    int      `yaml:"-"` // monotonic; resumed from max(id)+1 on load
}
