package guilds

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// Invariant: all guild mutations run on the single event-processing goroutine
// (command handlers + hooks). registryMu guards the maps; the *Guild pointers
// returned by Get/All are only mutated on that goroutine, so command-layer reads
// of g.Members are safe today. If a concurrent reader is ever added (e.g. a web
// page goroutine like the leaderboards module has), snapshot under the lock.
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
	byUser[userId] = strings.ToUpper(g.Tag)
	registryMu.Unlock()
	// Clear this user's pending invite from EVERY guild (including this one), so a
	// stale cross-guild invitation can't linger and let them later join a second
	// guild off it without a fresh invitation.
	clearInvitesEverywhere(userId)
	return Save(g)
}

// clearInvitesEverywhere removes userId's pending invite from all guilds and
// persists the ones it touched. Used on join and decline.
func clearInvitesEverywhere(userId int) {
	var touched []*Guild
	registryMu.Lock()
	for _, g := range byTag {
		if g.HasInvite(userId) {
			g.PendingInvites = removeInt(g.PendingInvites, userId)
			touched = append(touched, g)
		}
	}
	registryMu.Unlock()
	for _, g := range touched {
		_ = Save(g)
	}
}

// ClearInvites removes every pending invite for userId (decline-all).
func ClearInvites(userId int) { clearInvitesEverywhere(userId) }

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

// DepositGold adds amount to the guild treasury and persists. The caller is
// responsible for debiting the depositor's gold BEFORE calling (deposit is the
// value-in half of the transfer; it never fails on balance).
func DepositGold(tag string, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	registryMu.Lock()
	g.Treasury += amount
	registryMu.Unlock()
	return Save(g)
}

// WithdrawGold subtracts amount from the treasury and persists. Fails (without
// mutating) if the treasury holds less than amount. The caller credits the
// withdrawer only after this returns nil.
func WithdrawGold(tag string, amount int) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	registryMu.Lock()
	if g.Treasury < amount {
		registryMu.Unlock()
		return fmt.Errorf("the treasury does not hold that much gold")
	}
	g.Treasury -= amount
	registryMu.Unlock()
	return Save(g)
}

// DonateItem appends an item to the guild vault and persists. Fails if the vault
// is at capacity. The caller removes the item from the donor BEFORE calling.
func DonateItem(tag string, item items.Item) error {
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	capLimit := int(configs.GetBalanceConfig().GuildVaultCapacity)
	registryMu.Lock()
	if len(g.Vault) >= capLimit {
		registryMu.Unlock()
		return fmt.Errorf("the guild vault is full")
	}
	g.Vault = append(g.Vault, item)
	registryMu.Unlock()
	return Save(g)
}

// TakeItem removes and returns the vault item at index idx, persisting the
// result. Fails if idx is out of range. The caller gives the item to the taker
// only after this returns nil.
func TakeItem(tag string, idx int) (items.Item, error) {
	g, ok := Get(tag)
	if !ok {
		return items.Item{}, fmt.Errorf("no such guild")
	}
	registryMu.Lock()
	if idx < 0 || idx >= len(g.Vault) {
		registryMu.Unlock()
		return items.Item{}, fmt.Errorf("no such item in the vault")
	}
	item := g.Vault[idx]
	g.Vault = append(g.Vault[:idx], g.Vault[idx+1:]...)
	registryMu.Unlock()
	return item, Save(g)
}

// SetTreasuryDelegated toggles whether officers may withdraw gold / take items
// and persists.
func SetTreasuryDelegated(tag string, delegated bool) error {
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	registryMu.Lock()
	g.TreasuryDelegated = delegated
	registryMu.Unlock()
	return Save(g)
}

// SetRankTitle sets (title != "") or clears (title == "") the custom title for
// rank and persists. Caller validates non-empty titles first (ValidRankTitle).
func SetRankTitle(tag string, rank GuildRank, title string) error {
	g, ok := Get(tag)
	if !ok {
		return fmt.Errorf("no such guild")
	}
	registryMu.Lock()
	if title == "" {
		delete(g.RankTitles, rank)
	} else {
		if g.RankTitles == nil {
			g.RankTitles = map[GuildRank]string{}
		}
		g.RankTitles[rank] = title
	}
	registryMu.Unlock()
	return Save(g)
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
