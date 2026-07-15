package guilds

import (
	"fmt"
	"strings"
	"time"
)

type GuildRank string

const (
	RankMember  GuildRank = "member"
	RankOfficer GuildRank = "officer"
	RankLeader  GuildRank = "leader"
)

// rankOrder gives a comparable weight (higher = more authority).
func rankOrder(r GuildRank) int {
	switch r {
	case RankLeader:
		return 3
	case RankOfficer:
		return 2
	case RankMember:
		return 1
	}
	return 0
}

type GuildMember struct {
	UserId        int       `yaml:"userid"`
	CharacterName string    `yaml:"charactername"` // name at join time (display only; may be stale after a rename)
	Rank          GuildRank `yaml:"rank"`
	Joined        time.Time `yaml:"joined"`
}

type Guild struct {
	Tag            string        `yaml:"tag"`
	Name           string        `yaml:"name"`
	LeaderUserId   int           `yaml:"leaderuserid"`
	Members        []GuildMember `yaml:"members"`
	PendingInvites []int         `yaml:"pendinginvites,omitempty"`
	Motd           string        `yaml:"motd,omitempty"`
	Created        time.Time     `yaml:"created"`
}

func validGuildTag(tag string) error {
	if len(tag) < 2 || len(tag) > 4 {
		return fmt.Errorf("tag must be 2-4 characters")
	}
	for _, r := range tag {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return fmt.Errorf("tag must be letters and digits only")
		}
	}
	return nil
}

func validGuildName(name string) error {
	if len(name) < 3 || len(name) > 40 {
		return fmt.Errorf("name must be 3-40 characters")
	}
	if strings.TrimSpace(name) != name {
		return fmt.Errorf("name must not start or end with a space")
	}
	return nil
}

func (g *Guild) MemberRank(userId int) (GuildRank, bool) {
	for _, m := range g.Members {
		if m.UserId == userId {
			return m.Rank, true
		}
	}
	return "", false
}

func (g *Guild) IsMember(userId int) bool { _, ok := g.MemberRank(userId); return ok }
func (g *Guild) IsLeader(userId int) bool { return g.LeaderUserId == userId }

// CanManage reports whether userId is officer or leader.
func (g *Guild) CanManage(userId int) bool {
	r, ok := g.MemberRank(userId)
	return ok && rankOrder(r) >= rankOrder(RankOfficer)
}

// CanKick reports whether actor may kick target: actor is officer+, target is a
// member, and actor outranks target strictly.
func (g *Guild) CanKick(actorId, targetId int) bool {
	ar, aok := g.MemberRank(actorId)
	tr, tok := g.MemberRank(targetId)
	if !aok || !tok || !g.CanManage(actorId) {
		return false
	}
	return rankOrder(ar) > rankOrder(tr)
}

func (g *Guild) HasInvite(userId int) bool {
	for _, id := range g.PendingInvites {
		if id == userId {
			return true
		}
	}
	return false
}
