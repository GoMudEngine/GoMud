# Guild Membership Core Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A persistent social guild: found one (5000g from bank), invite friends, organize by member/officer/leader ranks, roster + MOTD, all persisted per-guild on disk.

**Architecture:** `internal/guilds` owns the durable model, a mutex-guarded registry (`byTag`/`byUser`), and per-guild YAML persistence (the `internal/shops` pattern). A `guild` usercommand drives all mutations through the registry. A login hook shows the MOTD; `who` shows a `[TAG]` prefix. Testable logic (validation, permissions, registry ops) lives in `internal/guilds` and is unit-tested; command handlers are thin + boot/manually verified.

**Tech Stack:** Go, GoMud command/hook/config layers, YAML, testify.

**Spec:** `docs/superpowers/specs/2026-07-15-guild-membership-core-design.md`

---

## File Structure

- `internal/guilds/guilds.go` — `Guild`/`GuildMember`/`GuildRank` + validation + permission helpers.
- `internal/guilds/registry.go` — registry maps, `Create`, membership mutations, lookups.
- `internal/guilds/persistence.go` — `Save`/`LoadDataFiles`/`Delete` + paths.
- `internal/guilds/*_test.go` — validation, permissions, registry, founding.
- `internal/usercommands/guild.go` — the `guild` command (dispatch + subcommands).
- `internal/usercommands/usercommands.go` — register `guild`.
- `internal/usercommands/who.go` — `[TAG]` prefix.
- `internal/hooks/PlayerSpawn_HandleJoin.go` — guild MOTD on login.
- `internal/actions/divergences.go` — allowlist `guild`.
- `internal/configs/config.balance.go` + `config.balance.misc.go` — `GuildFoundingCost`.
- `main.go` — `guilds.LoadDataFiles()`.
- Remove `internal/clans/`.
- `_datafiles/world/dogmud/templates/help/guild.template` (+ default copy).
- `CLAUDE.md`, `PATCH_NOTES.md`, `docs/PATH_TO_1.0.md`.

---

## Task 1: Model + validation + permission helpers

**Files:** Create `internal/guilds/guilds.go`, `internal/guilds/guilds_test.go`

- [ ] **Step 1: Write the failing test**

`internal/guilds/guilds_test.go`:

```go
package guilds

import "testing"

func TestValidGuildTag(t *testing.T) {
	good := []string{"QC", "ABCD", "a1", "X9y"}
	for _, s := range good {
		if err := validGuildTag(s); err != nil {
			t.Errorf("%q should be valid: %v", s, err)
		}
	}
	bad := []string{"", "A", "ABCDE", "Q C", "Q-C", "!!"}
	for _, s := range bad {
		if err := validGuildTag(s); err == nil {
			t.Errorf("%q should be invalid", s)
		}
	}
}

func TestValidGuildName(t *testing.T) {
	if err := validGuildName("Questing Cajuns"); err != nil {
		t.Errorf("normal name should validate: %v", err)
	}
	if err := validGuildName("ab"); err == nil {
		t.Error("2-char name should be too short")
	}
	if err := validGuildName(" Trim Me "); err == nil {
		t.Error("leading/trailing space should be rejected")
	}
}

func TestPermissions(t *testing.T) {
	g := &Guild{Tag: "QC", LeaderUserId: 1, Members: []GuildMember{
		{UserId: 1, Rank: RankLeader}, {UserId: 2, Rank: RankOfficer}, {UserId: 3, Rank: RankMember},
	}}
	if !g.IsLeader(1) || g.IsLeader(2) {
		t.Error("leader detection wrong")
	}
	if !g.CanManage(1) || !g.CanManage(2) || g.CanManage(3) {
		t.Error("CanManage should be officer+ only")
	}
	// kick rule: can kick strictly-lower rank only.
	if !g.CanKick(1, 3) || !g.CanKick(2, 3) {
		t.Error("leader/officer should kick a member")
	}
	if g.CanKick(2, 2) || g.CanKick(2, 1) || g.CanKick(3, 2) {
		t.Error("cannot kick peer/superior; member cannot kick")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/guilds/ -run 'TestValidGuild|TestPermissions' -v`
Expected: FAIL — package/symbols undefined.

- [ ] **Step 3: Implement the model + helpers**

`internal/guilds/guilds.go`:

```go
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
	CharacterName string    `yaml:"charactername"`
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
```

- [ ] **Step 4: Run test + commit**

Run: `go test ./internal/guilds/ -run 'TestValidGuild|TestPermissions' -v` → PASS
```bash
git add internal/guilds/guilds.go internal/guilds/guilds_test.go
git commit -m "feat(guilds): guild model + validation + permission helpers"
```

---

## Task 2: Persistence + registry

**Files:** Create `internal/guilds/persistence.go`, `internal/guilds/registry.go`, `internal/guilds/registry_test.go`

- [ ] **Step 1: Write the failing test**

`internal/guilds/registry_test.go` — drive the registry through an in-memory-ish path. Since
`Save` writes to disk, override the data dir to a temp dir for the test (add a small
`SetDataDirForTest(dir string) func()` hook in persistence.go that swaps the base path and
returns a restore func). Test:

```go
package guilds

import "testing"

func TestRegistry_CreateAndLookup(t *testing.T) {
	defer SetDataDirForTest(t.TempDir())()
	resetRegistry() // clears maps (test helper)

	g, err := Create("QC", "Questing Cajuns", 1, "Founder")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if g.Tag != "QC" || !g.IsLeader(1) || len(g.Members) != 1 {
		t.Fatalf("bad new guild: %+v", g)
	}
	if TagForUser(1) != "QC" {
		t.Errorf("TagForUser(1) = %q, want QC", TagForUser(1))
	}
	if _, ok := Get("qc"); !ok { // case-insensitive
		t.Error("Get should be case-insensitive")
	}

	// Uniqueness.
	if _, err := Create("QC", "Other", 2, "B"); err == nil {
		t.Error("duplicate tag should fail")
	}
	if _, err := Create("ZZ", "Questing Cajuns", 2, "B"); err == nil {
		t.Error("duplicate name should fail")
	}

	// Add / rank / remove.
	if err := AddMember("QC", 2, "Second"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if TagForUser(2) != "QC" {
		t.Error("member 2 not indexed")
	}
	if err := SetRank("QC", 2, RankOfficer); err != nil {
		t.Fatalf("setrank: %v", err)
	}
	if r, _ := g.MemberRank(2); r != RankOfficer {
		t.Errorf("rank = %q, want officer", r)
	}
	RemoveMember("QC", 2)
	if TagForUser(2) != "" {
		t.Error("member 2 should be de-indexed after removal")
	}

	// Transfer + delete.
	AddMember("QC", 3, "Third")
	if err := TransferLeader("QC", 3); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if !g.IsLeader(3) {
		t.Error("3 should be leader after transfer")
	}
	Delete("QC")
	if _, ok := Get("QC"); ok {
		t.Error("guild should be gone after delete")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/guilds/ -run TestRegistry -v`
Expected: FAIL — registry symbols undefined.

- [ ] **Step 3: Implement persistence**

`internal/guilds/persistence.go` — model on `internal/shops/persistence.go`:

```go
package guilds

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

var dataDirOverride string // test hook; empty = use config

func guildsDir() string {
	if dataDirOverride != "" {
		return dataDirOverride
	}
	return filepath.FromSlash(configs.GetFilePathsConfig().DataFiles.String() + `/guilds`)
}

// SetDataDirForTest points persistence at dir and returns a restore func.
func SetDataDirForTest(dir string) func() {
	prev := dataDirOverride
	dataDirOverride = dir
	return func() { dataDirOverride = prev }
}

func guildPath(tag string) string {
	return filepath.Join(guildsDir(), strings.ToLower(tag)+".yaml")
}

// Save writes a guild to disk.
func Save(g *Guild) error {
	if err := os.MkdirAll(guildsDir(), 0755); err != nil {
		return fmt.Errorf("guilds.Save: mkdir: %w", err)
	}
	b, err := yaml.Marshal(g)
	if err != nil {
		return fmt.Errorf("guilds.Save: marshal: %w", err)
	}
	return os.WriteFile(guildPath(g.Tag), b, 0644)
}

// Delete removes a guild's file.
func Delete(tag string) {
	registryMu.Lock()
	if g, ok := byTag[strings.ToUpper(tag)]; ok {
		for _, m := range g.Members {
			delete(byUser, m.UserId)
		}
		delete(byTag, strings.ToUpper(tag))
	}
	registryMu.Unlock()
	_ = os.Remove(guildPath(tag))
}

// LoadDataFiles loads all guild files into the registry at boot. A malformed
// file is logged + skipped (runtime-generated state must not crash the server).
func LoadDataFiles() {
	start := time.Now()
	resetRegistry()
	dir := guildsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		mudlog.Info("guilds.LoadDataFiles()", "loadedCount", 0, "note", "no guilds dir")
		return
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		b, rerr := os.ReadFile(filepath.Join(dir, e.Name()))
		if rerr != nil {
			mudlog.Error("guilds.LoadDataFiles", "file", e.Name(), "error", rerr)
			continue
		}
		var g Guild
		if uerr := yaml.Unmarshal(b, &g); uerr != nil {
			mudlog.Error("guilds.LoadDataFiles", "file", e.Name(), "error", uerr)
			continue
		}
		indexGuild(&g)
		n++
	}
	mudlog.Info("guilds.LoadDataFiles()", "loadedCount", n, "Time Taken", time.Since(start))
}
```

- [ ] **Step 4: Implement the registry**

`internal/guilds/registry.go`:

```go
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

func TagExists(tag string) bool  { _, ok := Get(tag); return ok }
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
	// remove any pending invite
	g.PendingInvites = removeInt(g.PendingInvites, userId)
	byUser[userId] = strings.ToUpper(g.Tag)
	registryMu.Unlock()
	return Save(g)
}

// RemoveMember removes userId and persists. If they were the leader, caller must
// have transferred/disbanded first (command enforces).
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
```

- [ ] **Step 5: Run test + commit**

Run: `go test ./internal/guilds/ -v` → PASS
```bash
git add internal/guilds/persistence.go internal/guilds/registry.go internal/guilds/registry_test.go
git commit -m "feat(guilds): durable registry + per-guild YAML persistence"
```

---

## Task 3: Config `GuildFoundingCost` + boot wiring + remove clans stub

**Files:** `internal/configs/config.balance.go`, `config.balance.misc.go`, `main.go`, remove `internal/clans/`

- [ ] **Step 1: Config field + default**

`config.balance.go` (near `GuildFoundingCost` sibling knobs):
```go
	GuildFoundingCost           ConfigInt   `yaml:"GuildFoundingCost"`                  // One-time gold cost (from bank) to found a guild (default 5000).
```
`config.balance.misc.go` validator:
```go
	if b.GuildFoundingCost <= 0 {
		b.GuildFoundingCost = 5000
	}
```

- [ ] **Step 2: Boot wiring**

In `main.go`, after `achievements.LoadDataFiles()`:
```go
	guilds.LoadDataFiles()
```
Add import `"github.com/GoMudEngine/GoMud/internal/guilds"`.

- [ ] **Step 3: Remove the superseded clans stub**

Delete `internal/clans/clans.go` (verified referenced nowhere). If the directory is now empty,
remove it.

- [ ] **Step 4: Build + commit**

Run: `go build ./...` → success (aside from the pre-existing copyover darwin-tag notes).
```bash
git add internal/configs/config.balance.go internal/configs/config.balance.misc.go main.go
git rm internal/clans/clans.go
git commit -m "feat(guilds): founding-cost config, boot wiring; remove clans stub"
```

---

## Task 4: The `guild` command — founding, info, list, leave, disband

**Files:** Create `internal/usercommands/guild.go`; modify `usercommands.go`, `divergences.go`; create help templates.

- [ ] **Step 1: Implement the dispatcher + these subcommands**

Create `internal/usercommands/guild.go` with `Guild(rest, user, room, flags)` that splits
`rest` into `sub` + args and dispatches. Implement in this task: **(no arg / info), create,
list, leave, disband.** Key behaviors:

- **info** (`guild` or `guild info`): `g, ok := guilds.GetByUser(user.UserId)`. Not in one →
  hint. Else print Name `[TAG]`, MOTD if set, then roster grouped leader→officers→members with
  names + joined dates; total count.
- **create `<TAG> <Name…>`**: reject if `guilds.GetByUser` already; validate args present.
  Confirm via `StartPrompt`: `Found <Name> [TAG] for <cost> gold?` [Yes/No]. On Yes:
  `cost := int(configs.GetBalanceConfig().GuildFoundingCost)`; require `user.Character.Bank >= cost`
  (else refuse); `g, err := guilds.Create(TAG, Name, user.UserId, user.Character.Name)`; on err
  show it (don't charge); on success `user.Character.Bank -= cost` +
  `events.AddToQueue(events.EquipmentChange{UserId, BankChange: -cost})`, announce.
- **list**: `guilds.All()` → lines `[TAG] Name — N members, led by <leaderName>` (resolve leader
  name from the guild's members). Sort by name for stability.
- **leave**: not in a guild → hint. If leader AND `len(Members) > 1` → refuse ("transfer or
  disband first"). If leader AND sole member → confirm → `guilds.Delete(tag)`. Else
  `guilds.RemoveMember(tag, user.UserId)` + announce to the guild (message online members).
- **disband** (leader only, confirm): notify all online members, `guilds.Delete(tag)`.

Use a helper `announceGuild(g, msg, exceptUserId)` that sends `msg` to each online member via
`users.GetByUserId`. Guild-tag/name in ANSI, no emoji.

- [ ] **Step 2: Register + help + allowlist**

- `usercommands.go`: `` `guild`: {Guild, true, true, false}, ``.
- `internal/actions/divergences.go`: add `"guild": "player-mechanic",` (alphabetical).
- Help template `_datafiles/world/dogmud/templates/help/guild.template` (list all subcommands),
  copy to `default/`.

- [ ] **Step 3: Build + full usercommands test + commit**

Run: `go build ./... && go test ./internal/usercommands/` → PASS (help completeness satisfied).
```bash
git add internal/usercommands/guild.go internal/usercommands/usercommands.go internal/actions/divergences.go _datafiles/world/dogmud/templates/help/guild.template _datafiles/world/default/templates/help/guild.template
git commit -m "feat(guilds): guild command — create/info/list/leave/disband"
```

---

## Task 5: The `guild` command — invite/accept/decline, kick/promote/demote/transfer, motd

**Files:** modify `internal/usercommands/guild.go`

- [ ] **Step 1: Add these subcommands**

- **invite `<player>`** (officer+): `g` = caller's guild; `g.CanManage(user.UserId)` else refuse.
  Resolve target ONLINE via `users.GetByCharacterName(name)`; must exist, not already in a guild
  (`guilds.GetByUser`), not already invited. `guilds.AddInvite(tag, target.UserId)`; notify target
  ("<caller> invites you to <Name>; `guild accept`/`guild decline`") + confirm to caller.
- **accept**: `g, ok := guilds.GuildWithInvite(user.UserId)`; already in a guild → refuse; else
  `guilds.AddMember(g.Tag, user.UserId, user.Character.Name)` + announce to guild + welcome.
- **decline**: `guilds.GuildWithInvite` → `guilds.RemoveInvite(g.Tag, user.UserId)`.
- **kick `<player>`** (officer+): resolve target as a MEMBER of the caller's guild (by name;
  search `g.Members` for a matching current character name via `users.GetByUserId(m.UserId)` or
  the stored `CharacterName`); `g.CanKick(user.UserId, target.UserId)` else refuse;
  `guilds.RemoveMember` + notify.
- **promote/demote `<player>`** (leader only): target is a member; promote member→officer,
  demote officer→member; never touches leader (use transfer). `guilds.SetRank`.
- **transfer `<player>`** (leader only, confirm): target is a member; `guilds.TransferLeader`.
- **motd `<text>`** (officer+): set `g.Motd = text` via a small `guilds.SetMotd(tag, text)`
  registry fn (persist); empty text clears it. Confirm to caller.

Add `guilds.SetMotd(tag, text string) error` to `registry.go` (mirror `SetRank`'s persist).

Member-name resolution helper: prefer matching an online player's current character name; fall
back to the stored `CharacterName` on the member record (a member may be offline).

- [ ] **Step 2: Build + commit**

Run: `go build ./...` → success.
```bash
git add internal/usercommands/guild.go internal/guilds/registry.go
git commit -m "feat(guilds): guild invite/accept/kick/promote/transfer/motd"
```

---

## Task 6: MOTD on login + `who` guild tag

**Files:** `internal/hooks/PlayerSpawn_HandleJoin.go`, `internal/usercommands/who.go`

- [ ] **Step 1: MOTD on login**

In `PlayerSpawn_HandleJoin.go`'s `HandleJoin`, after the existing join handling, if
`g, ok := guilds.GetByUser(userId); ok && g.Motd != ""`, send it to the spawning user:
`user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="cyan">[%s] %s</ansi>`, g.Tag, g.Motd))`.
(Confirm how `HandleJoin` gets the userId/user — match the existing code; import `guilds`.)

- [ ] **Step 2: `who` guild tag prefix**

In `who.go`, where each online user's name is rendered, prefix `[TAG] ` when
`guilds.TagForUser(u.UserId) != ""`. Keep formatting 80-col friendly. Import `guilds`.

- [ ] **Step 3: Build + commit**

Run: `go build ./...` → success.
```bash
git add internal/hooks/PlayerSpawn_HandleJoin.go internal/usercommands/who.go
git commit -m "feat(guilds): MOTD on login + [TAG] in who"
```

---

## Task 7: Boot smoke test + docs

- [ ] **Step 1: Full build + touched tests**

Run: `go build ./... && go test ./internal/guilds/ ./internal/usercommands/ ./internal/configs/`
Expected: all `ok`.

- [ ] **Step 2: Boot smoke test**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```
Run `go run .`; confirm `guilds.LoadDataFiles() loadedCount=0` (no guilds yet) and `Server Ready`
with no panic + no CommandParity warning for `guild`; stop.

- [ ] **Step 3: Docs**

- `CLAUDE.md`: add `guilds/` to the durable-state "do NOT wipe" note alongside `shops/`.
- `PATCH_NOTES.md`: player-facing entry (found/join guilds, ranks, MOTD, `guild` command).
- `docs/PATH_TO_1.0.md` §3: mark guilds membership-core ✅ (note remaining sub-projects: chat,
  treasury, perks).

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md PATCH_NOTES.md docs/PATH_TO_1.0.md
git commit -m "docs(guilds): CLAUDE do-not-wipe note, patch notes, roadmap"
```

---

## Notes for the implementer

- **Verify before coding** (codegraph/Read): `Who`'s exact user-rendering loop (`who.go`),
  `HandleJoin`'s signature/user access (`PlayerSpawn_HandleJoin.go`),
  `configs.GetFilePathsConfig().DataFiles.String()`, `users.GetByCharacterName`,
  `events.EquipmentChange{BankChange}`, `StartPrompt`/`Ask` (see `admin.mudmail.go`/`mail.go`).
- **Registry is the single source of truth** — command handlers only call registry funcs; never
  mutate `Members`/maps directly from the command layer.
- **No emoji** in player-facing text; ANSI color only.
- **Guild files are durable state** — never add `guilds/` to instance-cleanup; excluded like `shops/`.
- **Non-panic loading** — a corrupt guild file logs + skips (unlike authored content), so one bad
  file can't take down the server.
```
