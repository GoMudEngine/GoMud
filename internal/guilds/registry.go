package guilds

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	registryMu sync.RWMutex
	byTag      = map[string]*Guild{} // key: UPPERCASE tag
	byUser     = map[int]string{}    // userId -> UPPERCASE tag
)

func resetRegistry() {
	registryMu.Lock()
	byTag = map[string]*Guild{}
	byUser = map[int]string{}
	registryMu.Unlock()
}

// indexGuild adds a guild to the maps (no disk write). Used by loader + Create.
func indexGuild(g *Guild) {
	registryMu.Lock()
	byTag[strings.ToUpper(g.Tag)] = g
	for _, m := range g.Members {
		byUser[m.UserId] = strings.ToUpper(g.Tag)
	}
	registryMu.Unlock()
}

func Get(tag string) (*Guild, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	g, ok := byTag[strings.ToUpper(tag)]
	return g, ok
}

func GetByUser(userId int) (*Guild, bool) {
	registryMu.RLock()
	tag, ok := byUser[userId]
	registryMu.RUnlock()
	if !ok {
		return nil, false
	}
	return Get(tag)
}

func TagForUser(userId int) string {
	if g, ok := GetByUser(userId); ok {
		return strings.ToUpper(g.Tag)
	}
	return ""
}

func TagExists(tag string) bool { _, ok := Get(tag); return ok }

func NameExists(name string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	for _, g := range byTag {
		if strings.EqualFold(g.Name, name) {
			return true
		}
	}
	return false
}

func All() []*Guild {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Guild, 0, len(byTag))
	for _, g := range byTag {
		out = append(out, g)
	}
	return out
}

// Create validates + builds a new guild with leader as sole member, indexes it,
// and persists it. Does NOT charge the founding fee (the command does that).
func Create(tag, name string, leaderUserId int, leaderName string) (*Guild, error) {
	if err := validGuildTag(tag); err != nil {
		return nil, err
	}
	if err := validGuildName(name); err != nil {
		return nil, err
	}
	if TagExists(tag) {
		return nil, fmt.Errorf("a guild with that tag already exists")
	}
	if NameExists(name) {
		return nil, fmt.Errorf("a guild with that name already exists")
	}
	if _, ok := GetByUser(leaderUserId); ok {
		return nil, fmt.Errorf("you are already in a guild")
	}
	g := &Guild{
		Tag:          strings.ToUpper(tag),
		Name:         name,
		LeaderUserId: leaderUserId,
		Members:      []GuildMember{{UserId: leaderUserId, CharacterName: leaderName, Rank: RankLeader, Joined: time.Now()}},
		Created:      time.Now(),
	}
	indexGuild(g)
	if err := Save(g); err != nil {
		Delete(g.Tag)
		return nil, err
	}
	return g, nil
}

// AddMember adds userId as a member and persists. Errors if already in a guild.
func AddMember(tag string, userId int, charName string) error {
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	if _, in := GetByUser(userId); in {
		return fmt.Errorf("that player is already in a guild")
	}
	registryMu.Lock()
	g.Members = append(g.Members, GuildMember{UserId: userId, CharacterName: charName, Rank: RankMember, Joined: time.Now()})
	g.PendingInvites = removeInt(g.PendingInvites, userId)
	byUser[userId] = strings.ToUpper(g.Tag)
	registryMu.Unlock()
	return Save(g)
}

// RemoveMember removes userId and persists.
func RemoveMember(tag string, userId int) error {
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	registryMu.Lock()
	for i, m := range g.Members {
		if m.UserId == userId {
			g.Members = append(g.Members[:i], g.Members[i+1:]...)
			break
		}
	}
	delete(byUser, userId)
	registryMu.Unlock()
	return Save(g)
}

func SetRank(tag string, userId int, rank GuildRank) error {
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	registryMu.Lock()
	for i := range g.Members {
		if g.Members[i].UserId == userId {
			g.Members[i].Rank = rank
		}
	}
	registryMu.Unlock()
	return Save(g)
}

// TransferLeader makes newLeader the leader and demotes the old leader to officer.
func TransferLeader(tag string, newLeaderId int) error {
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	if !g.IsMember(newLeaderId) {
		return fmt.Errorf("that player is not in the guild")
	}
	registryMu.Lock()
	old := g.LeaderUserId
	for i := range g.Members {
		switch g.Members[i].UserId {
		case newLeaderId:
			g.Members[i].Rank = RankLeader
		case old:
			g.Members[i].Rank = RankOfficer
		}
	}
	g.LeaderUserId = newLeaderId
	registryMu.Unlock()
	return Save(g)
}

// SetMotd sets (or clears, if empty) the guild's message-of-the-day and persists.
func SetMotd(tag, text string) error {
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	registryMu.Lock()
	g.Motd = text
	registryMu.Unlock()
	return Save(g)
}

func AddInvite(tag string, userId int) error {
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	registryMu.Lock()
	if !g.HasInvite(userId) {
		g.PendingInvites = append(g.PendingInvites, userId)
	}
	registryMu.Unlock()
	return Save(g)
}

func RemoveInvite(tag string, userId int) error {
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	registryMu.Lock()
	g.PendingInvites = removeInt(g.PendingInvites, userId)
	registryMu.Unlock()
	return Save(g)
}

// GuildWithInvite finds the guild that has a pending invite for userId, if any.
func GuildWithInvite(userId int) (*Guild, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	for _, g := range byTag {
		if g.HasInvite(userId) {
			return g, true
		}
	}
	return nil, false
}

func removeInt(s []int, v int) []int {
	out := s[:0]
	for _, x := range s {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}
