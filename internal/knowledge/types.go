package knowledge

type SubjectType string

const (
	SubjectPlayer SubjectType = "player"
	SubjectMob    SubjectType = "mob"
)

type Subject struct {
	Type SubjectType `yaml:"type"`
	Id   int         `yaml:"id"`
}

func PlayerSubject(userId int) Subject { return Subject{Type: SubjectPlayer, Id: userId} }
func MobSubject(mobId int) Subject     { return Subject{Type: SubjectMob, Id: mobId} }

type Source string

const (
	SourceWitnessed Source = "witnessed"
	SourceTold      Source = "told"
	SourceDeduced   Source = "deduced"
	SourceUnknown   Source = "unknown"
)

type Confidence string

const (
	ConfidenceHigh Confidence = "high"
	ConfidenceMed  Confidence = "med"
	ConfidenceLow  Confidence = "low"
)

type Observation struct {
	Room  int    `yaml:"room"`
	Round uint64 `yaml:"round"`
}

type Record struct {
	Subject          Subject       `yaml:"subject"`
	HasMet           bool          `yaml:"has_met"`
	NameLearned      string        `yaml:"name_learned,omitempty"`
	Source           Source        `yaml:"source"`
	Confidence       Confidence    `yaml:"confidence"`
	LastSeenRoom     int           `yaml:"last_seen_room"`
	LastSeenRound    uint64        `yaml:"last_seen_round"`
	Observations     []Observation `yaml:"observations,omitempty"`
	CrimesWitnessed  []int         `yaml:"crimes_witnessed,omitempty"`
	LearnedRound     uint64        `yaml:"learned_round"`
	LastUpdatedRound uint64        `yaml:"last_updated_round"`
}

type ObserverFile struct {
	ObserverMobId int       `yaml:"observer_mob_id"`
	ObserverName  string    `yaml:"observer_name"`
	Records       []*Record `yaml:"records"`
}
