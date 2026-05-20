package bounties

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/knowledge"
)

type IssuerType string

const (
	IssuerFaction IssuerType = "faction"
	IssuerQuest   IssuerType = "quest"
	IssuerNPC     IssuerType = "npc"
)

type Issuer struct {
	Type IssuerType `yaml:"type"`
	Id   string     `yaml:"id"`
}

func FactionIssuer(slug string) Issuer { return Issuer{Type: IssuerFaction, Id: slug} }
func QuestIssuer(questId int) Issuer   { return Issuer{Type: IssuerQuest, Id: strconv.Itoa(questId)} }
func NPCIssuer(mobId int) Issuer       { return Issuer{Type: IssuerNPC, Id: strconv.Itoa(mobId)} }

type Condition string

const (
	ConditionKill Condition = "kill"
)

type Status string

const (
	StatusOpen      Status = "open"
	StatusClaimed   Status = "claimed"
	StatusExpired   Status = "expired"
	StatusWithdrawn Status = "withdrawn"
)

type Bounty struct {
	Id             int               `yaml:"id"`
	Issuer         Issuer            `yaml:"issuer"`
	Target         knowledge.Subject `yaml:"target"`
	GoldReward     int               `yaml:"gold_reward"`
	RepReward      int               `yaml:"rep_reward"`
	Condition      Condition         `yaml:"condition"`
	DeclaredRound  uint64            `yaml:"declared_round"`
	ExpiryRound    uint64            `yaml:"expiry_round"`
	Status         Status            `yaml:"status"`
	ClaimedBy      knowledge.Subject `yaml:"claimed_by,omitempty"`
	ClaimedRound   uint64            `yaml:"claimed_round,omitempty"`
	DeclaredReason string            `yaml:"declared_reason,omitempty"`
}

type Registry struct {
	NextId   int       `yaml:"next_id"`
	Bounties []*Bounty `yaml:"bounties"`
}
