package facts

import (
	"github.com/GoMudEngine/GoMud/internal/worldevents"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusWithdrawn Status = "withdrawn"
	StatusExpired   Status = "expired"
)

type Source string

const (
	SourceWitnessed Source = "witnessed"
	SourceTold      Source = "told"
	SourceDeduced   Source = "deduced"
	SourceUnknown   Source = "unknown"
)

type Fact struct {
	Id                  string                   `yaml:"id"`
	Description         string                   `yaml:"description"`
	Significance        worldevents.Significance `yaml:"significance"`
	Zone                string                   `yaml:"zone,omitempty"`
	Region              string                   `yaml:"region,omitempty"`
	DeclaredRound       uint64                   `yaml:"declared_round"`
	ExpiryRound         uint64                   `yaml:"expiry_round,omitempty"`
	Tags                []string                 `yaml:"tags,omitempty"`
	WithdrawOnRespawnOf int                      `yaml:"withdraw_on_respawn_of,omitempty"`
	Status              Status                   `yaml:"status,omitempty"`
}

type FactKnowledge struct {
	FactId       string `yaml:"fact_id"`
	Source       Source `yaml:"source"`
	LearnedRound uint64 `yaml:"learned_round"`
}

type Awareness struct {
	ObserverMobId    int             `yaml:"observer_mob_id"`
	ObserverName     string          `yaml:"observer_name"`
	HeardEvents      []uint64        `yaml:"heard_events,omitempty"`
	KnownFacts       []FactKnowledge `yaml:"known_facts,omitempty"`
	LastUpdatedRound uint64          `yaml:"last_updated_round"`
}

type Registry struct {
	Facts []*Fact `yaml:"facts"`
}
