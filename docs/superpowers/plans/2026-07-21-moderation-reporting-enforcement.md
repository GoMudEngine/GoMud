# Moderation: Reporting + Enforcement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give players a `petition` command that feeds a durable, admin-reviewable queue (+ online-staff notify), and give staff the missing enforcement primitives — global `boot`, permanent account + optional IP `ban`, and global-by-name `mute`/`deafen`.

**Architecture:** A new `internal/moderation` package owns two durable YAML stores under `_datafiles/moderation/` (petition queue + ban lists), following the `internal/guilds` living-state pattern. Commands live in `internal/usercommands/`; the login completion (`FinalizeLoginOrCreate`) calls the ban-check functions. All new player-facing text uses descriptive, no-hard-number wording.

**Tech Stack:** Go, `gopkg.in/yaml.v2`, testify, the existing `users`/`connections`/`configs`/`events` packages.

**Spec:** `docs/superpowers/specs/2026-07-21-moderation-reporting-enforcement-design.md`

---

## Implementation decisions that refine the approved spec

Both surfaced during code grounding; noted here for transparency:

1. **IP-ban enforcement seam:** implemented at the top of `FinalizeLoginOrCreate`
   (`internal/inputhandlers/login.go`), not `main.go`'s connection-accept path.
   Single-file, lower-risk, and it *also* blocks a banned IP from re-registering
   a new account (ban-evasion win). Cost: a banned IP sees the login banner
   before rejection (negligible at launch scale). Connect-time rejection remains
   a future refinement.
2. **`boot` disconnect mechanism:** uses `connections.Kick(connId, reason)`, the
   engine's standard disconnect. A booted player is a reconnectable zombie for
   `ZombieSeconds` **unless also banned** — the new account-ban check in
   `FinalizeLoginOrCreate` rejects their zombie-reconnect, so `ban`+`boot` is
   effectively permanent. A bespoke "no-linger" path is not cleanly exposed to
   command handlers; deferred.

## Grounded facts (verified — reference while implementing)

- **Command signature:** `func(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error)`.
- **Registration:** the `userCommands map[string]CommandAccess` literal in
  `internal/usercommands/usercommands.go` (starts ~line 53). Struct:
  `CommandAccess{ Func UserCommand; AllowedWhenDowned bool; AllowedInCombat bool; AdminOnly bool }`
  (positional literals: `{Handler, whenDowned, inCombat, adminOnly}`).
- **Output:** `user.SendText(messaging.CategorySystem, "...")`.
- **Staff iteration:** `users.GetAllActiveUsers() []*users.UserRecord`; a staffer
  is any `u.Role != users.RoleUser` (constants `users.RoleUser`, `users.RoleAdmin`
  in `internal/users/userrecord.go`).
- **Global name resolution:** `users.GetByCharacterName(name) *users.UserRecord`
  (online, case-insensitive, prefix-matches). `users.Exists(username) bool`.
- **Round counter (for cooldown):** `util.GetRoundCount() uint64`.
- **Login completion:** `FinalizeLoginOrCreate(results map[string]string, sharedState map[string]any, clientInput *connections.ClientInput) bool`
  in `internal/inputhandlers/login.go`. `username := results["username"]`;
  existing-user password verified at ~line 58-63; `users.LoginUser(...)` at ~line 65.
  Reject a connection with:
  `connections.SendTo([]byte(msg), clientInput.ConnectionId); connections.SendTo(term.CRLF, clientInput.ConnectionId); connections.Remove(clientInput.ConnectionId); return false`.
  Get the connection for IP: `connections.Get(clientInput.ConnectionId)` → `.RemoteAddr() net.Addr`, `.IsLocal() bool`.
- **Disconnect a live user:** `connections.Kick(connId connections.ConnectionId, reason string)`;
  target's connection id via `targetUser.ConnectionId()`.
- **Config pattern:** typed fields in a `configs` struct (`ConfigInt`) + a
  `Validate()` default + a `configs.GetXConfig()` accessor
  (see `internal/configs/config.network.go`; gameplay accessor is
  `configs.GetGamePlayConfig()`).
- **Persistence pattern:** `internal/guilds/persistence.go` — `yaml.Marshal`/
  `Unmarshal`, `os.WriteFile`/`os.MkdirAll`/`os.ReadFile`, a `SetDataDirForTest`
  hook, a `LoadDataFiles()` that logs + skips on error (never panics).
  Data dir root: `configs.GetFilePathsConfig().DataFiles.String()`.

---

## File structure

**Create:**
- `internal/moderation/moderation.go` — shared dir path, test hook, `LoadDataFiles()`.
- `internal/moderation/petitions.go` — `Petition` type + queue store + persistence.
- `internal/moderation/bans.go` — account/IP ban store + checks + persistence.
- `internal/moderation/moderation_test.go` — unit tests for the package.
- `internal/usercommands/petition.go` — `Petition` player command + cooldown.
- `internal/usercommands/petitions.go` — `Petitions` admin review command.
- `internal/usercommands/boot.go` — `Boot` admin command.
- `internal/usercommands/ban.go` — `Ban` admin command (account + `ban ip`).
- `internal/usercommands/unban.go` — `Unban` admin command.

**Modify:**
- `internal/usercommands/usercommands.go` — register 5 commands.
- `internal/usercommands/admin.mute.go`, `admin.deafen.go` — global name fallback.
- `internal/inputhandlers/login.go` — IP + account ban checks.
- `internal/configs/config.gameplay.go` + `_datafiles/config.yaml` — 2 knobs.
- the boot loader that calls `guilds.LoadDataFiles()` — add `moderation.LoadDataFiles()`.
- `.gitignore` — `_datafiles/moderation/`.
- `CLAUDE.md` — moderation-persistence note.
- `docs/PATH_TO_1.0.md` — §4 status.

---

## Task 1: moderation package — shared scaffolding + petition queue

**Files:**
- Create: `internal/moderation/moderation.go`
- Create: `internal/moderation/petitions.go`
- Test: `internal/moderation/moderation_test.go`

- [ ] **Step 1: Write the failing test**

```go
package moderation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPetitionQueue(t *testing.T) {
	restore := SetDataDirForTest(t.TempDir())
	defer restore()
	resetForTest()

	p1, err := Add("Alice", 5209, "Sanctum", "Bob is spamming slurs at me")
	assert.NoError(t, err)
	assert.Equal(t, 1, p1.Id)
	assert.Equal(t, StatusOpen, p1.Status)

	p2, _ := Add("Cara", 100, "Town", "stuck in the well")
	assert.Equal(t, 2, p2.Id)

	assert.Len(t, ListOpen(), 2)
	assert.Len(t, ListAll(), 2)

	got, ok := Get(1)
	assert.True(t, ok)
	assert.Equal(t, "Alice", got.Reporter)

	assert.NoError(t, Resolve(1, "AdminZoe", "warned Bob"))
	assert.Len(t, ListOpen(), 1)
	r, _ := Get(1)
	assert.Equal(t, StatusResolved, r.Status)
	assert.Equal(t, "AdminZoe", r.ResolvedBy)

	assert.Error(t, Resolve(999, "x", ""))
}

func TestPetitionPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	restore := SetDataDirForTest(dir)
	resetForTest()
	_, _ = Add("Alice", 5209, "Sanctum", "grief")
	_ = Resolve(1, "Zoe", "done")
	restore()

	// Reload from disk into a fresh in-memory state.
	restore2 := SetDataDirForTest(dir)
	defer restore2()
	resetForTest()
	LoadDataFiles()
	all := ListAll()
	assert.Len(t, all, 1)
	assert.Equal(t, StatusResolved, all[0].Status)
	assert.Equal(t, "Alice", all[0].Reporter)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/moderation/ -run TestPetition -v`
Expected: FAIL — package/functions do not compile/exist.

- [ ] **Step 3: Write `internal/moderation/moderation.go`**

```go
// Package moderation owns durable, admin-facing moderation state: the player
// petition queue and the account/IP ban lists. State persists as YAML under
// _datafiles/moderation/ and is living runtime state (like guilds/shops) — it
// is gitignored, kept on prod, and must NOT be wiped by the instance-save SOP.
package moderation

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

var (
	mu              sync.Mutex
	dataDirOverride string // test hook; empty = use config
)

func moderationDir() string {
	if dataDirOverride != "" {
		return dataDirOverride
	}
	return filepath.FromSlash(configs.GetFilePathsConfig().DataFiles.String() + `/moderation`)
}

// SetDataDirForTest points persistence at dir and returns a restore func.
func SetDataDirForTest(dir string) func() {
	prev := dataDirOverride
	dataDirOverride = dir
	return func() { dataDirOverride = prev }
}

// resetForTest clears all in-memory state (test-only). Task 2 extends this to
// also clear the ban maps.
func resetForTest() {
	mu.Lock()
	defer mu.Unlock()
	petitions = nil
	nextPetitionId = 1
}

// LoadDataFiles loads the moderation stores into memory at boot. Missing or
// malformed files are logged + skipped (runtime state must not crash boot).
// Task 2 extends this to also call loadBans().
func LoadDataFiles() {
	loadPetitions()
	mudlog.Info("moderation.LoadDataFiles()", "petitions", len(petitions))
}

// now is overridable in tests if deterministic timestamps are ever needed.
var now = time.Now
```

- [ ] **Step 4: Write `internal/moderation/petitions.go`**

```go
package moderation

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

const (
	StatusOpen     = "open"
	StatusResolved = "resolved"
)

type Petition struct {
	Id         int       `yaml:"id"`
	Reporter   string    `yaml:"reporter"`
	Timestamp  time.Time `yaml:"timestamp"`
	RoomId     int       `yaml:"room_id"`
	Zone       string    `yaml:"zone"`
	Message    string    `yaml:"message"`
	Status     string    `yaml:"status"`
	ResolvedBy string    `yaml:"resolved_by,omitempty"`
	ResolvedAt time.Time `yaml:"resolved_at,omitempty"`
	Note       string    `yaml:"note,omitempty"`
}

var (
	petitions      []Petition
	nextPetitionId = 1
)

func petitionsPath() string { return filepath.Join(moderationDir(), "petitions.yaml") }

// Add records a new open petition, persists, and returns it.
func Add(reporter string, roomId int, zone, message string) (Petition, error) {
	mu.Lock()
	defer mu.Unlock()
	p := Petition{
		Id:        nextPetitionId,
		Reporter:  reporter,
		Timestamp: now(),
		RoomId:    roomId,
		Zone:      zone,
		Message:   message,
		Status:    StatusOpen,
	}
	petitions = append(petitions, p)
	nextPetitionId++
	savePetitionsLocked()
	return p, nil
}

func ListOpen() []Petition {
	mu.Lock()
	defer mu.Unlock()
	out := []Petition{}
	for _, p := range petitions {
		if p.Status == StatusOpen {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Id < out[j].Id })
	return out
}

func ListAll() []Petition {
	mu.Lock()
	defer mu.Unlock()
	out := make([]Petition, len(petitions))
	copy(out, petitions)
	sort.Slice(out, func(i, j int) bool { return out[i].Id < out[j].Id })
	return out
}

func Get(id int) (Petition, bool) {
	mu.Lock()
	defer mu.Unlock()
	for _, p := range petitions {
		if p.Id == id {
			return p, true
		}
	}
	return Petition{}, false
}

func Resolve(id int, by, note string) error {
	mu.Lock()
	defer mu.Unlock()
	for i := range petitions {
		if petitions[i].Id == id {
			if petitions[i].Status == StatusResolved {
				return fmt.Errorf("petition %d already resolved", id)
			}
			petitions[i].Status = StatusResolved
			petitions[i].ResolvedBy = by
			petitions[i].ResolvedAt = now()
			petitions[i].Note = note
			savePetitionsLocked()
			return nil
		}
	}
	return fmt.Errorf("no petition with id %d", id)
}

func savePetitionsLocked() {
	if err := os.MkdirAll(moderationDir(), 0755); err != nil {
		mudlog.Error("moderation.savePetitions", "error", err.Error())
		return
	}
	b, err := yaml.Marshal(petitions)
	if err != nil {
		mudlog.Error("moderation.savePetitions", "error", err.Error())
		return
	}
	if err := os.WriteFile(petitionsPath(), b, 0644); err != nil {
		mudlog.Error("moderation.savePetitions", "error", err.Error())
	}
}

func loadPetitions() {
	mu.Lock()
	defer mu.Unlock()
	petitions = nil
	nextPetitionId = 1
	b, err := os.ReadFile(petitionsPath())
	if err != nil {
		return // no file yet — fine
	}
	if err := yaml.Unmarshal(b, &petitions); err != nil {
		mudlog.Error("moderation.loadPetitions", "error", err.Error())
		petitions = nil
		return
	}
	for _, p := range petitions {
		if p.Id >= nextPetitionId {
			nextPetitionId = p.Id + 1
		}
	}
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/moderation/ -run TestPetition -v`
Expected: PASS (both tests).

- [ ] **Step 6: Commit**

```bash
git add internal/moderation/moderation.go internal/moderation/petitions.go internal/moderation/moderation_test.go
git commit -m "feat(moderation): petition queue store + persistence"
```

---

## Task 2: moderation package — ban store

**Files:**
- Create: `internal/moderation/bans.go`
- Test: `internal/moderation/moderation_test.go` (append)

- [ ] **Step 1: Write the failing test (append to moderation_test.go)**

```go
func TestAccountBans(t *testing.T) {
	restore := SetDataDirForTest(t.TempDir())
	defer restore()
	resetForTest()

	assert.NoError(t, BanAccount("Griefer", "spamming slurs", "AdminZoe"))
	reason, banned := IsAccountBanned("griefer") // case-insensitive
	assert.True(t, banned)
	assert.Equal(t, "spamming slurs", reason)

	_, banned = IsAccountBanned("SomeoneElse")
	assert.False(t, banned)

	assert.NoError(t, Unban("GRIEFER"))
	_, banned = IsAccountBanned("Griefer")
	assert.False(t, banned)
}

func TestIPBans(t *testing.T) {
	restore := SetDataDirForTest(t.TempDir())
	defer restore()
	resetForTest()

	assert.NoError(t, BanIP("203.0.113.7", "evasion", "AdminZoe"))
	reason, banned := IsIPBanned("203.0.113.7")
	assert.True(t, banned)
	assert.Equal(t, "evasion", reason)

	_, banned = IsIPBanned("198.51.100.1")
	assert.False(t, banned)

	assert.NoError(t, UnbanIP("203.0.113.7"))
	_, banned = IsIPBanned("203.0.113.7")
	assert.False(t, banned)
}

func TestBanPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	restore := SetDataDirForTest(dir)
	resetForTest()
	_ = BanAccount("Griefer", "grief", "Zoe")
	_ = BanIP("203.0.113.7", "evasion", "Zoe")
	restore()

	restore2 := SetDataDirForTest(dir)
	defer restore2()
	resetForTest()
	LoadDataFiles()
	_, a := IsAccountBanned("griefer")
	_, i := IsIPBanned("203.0.113.7")
	assert.True(t, a)
	assert.True(t, i)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/moderation/ -run 'TestAccountBans|TestIPBans|TestBanPersistence' -v`
Expected: FAIL — ban functions do not exist.

- [ ] **Step 3: Write `internal/moderation/bans.go`**

```go
package moderation

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

type AccountBan struct {
	Username  string    `yaml:"username"`
	Reason    string    `yaml:"reason"`
	BannedBy  string    `yaml:"banned_by"`
	Timestamp time.Time `yaml:"timestamp"`
}

type IPBan struct {
	IP        string    `yaml:"ip"`
	Reason    string    `yaml:"reason"`
	BannedBy  string    `yaml:"banned_by"`
	Timestamp time.Time `yaml:"timestamp"`
}

// banFile is the on-disk shape of bans.yaml.
type banFile struct {
	Accounts []AccountBan `yaml:"accounts"`
	IPs      []IPBan      `yaml:"ips"`
}

var (
	accountBans = map[string]AccountBan{} // key: lowercased username
	ipBans      = map[string]IPBan{}      // key: exact host/ip
)

func bansPath() string { return filepath.Join(moderationDir(), "bans.yaml") }

func BanAccount(username, reason, by string) error {
	mu.Lock()
	defer mu.Unlock()
	key := strings.ToLower(strings.TrimSpace(username))
	accountBans[key] = AccountBan{Username: username, Reason: reason, BannedBy: by, Timestamp: now()}
	saveBansLocked()
	return nil
}

func Unban(username string) error {
	mu.Lock()
	defer mu.Unlock()
	delete(accountBans, strings.ToLower(strings.TrimSpace(username)))
	saveBansLocked()
	return nil
}

func IsAccountBanned(username string) (reason string, banned bool) {
	mu.Lock()
	defer mu.Unlock()
	if b, ok := accountBans[strings.ToLower(strings.TrimSpace(username))]; ok {
		return b.Reason, true
	}
	return "", false
}

func BanIP(ip, reason, by string) error {
	mu.Lock()
	defer mu.Unlock()
	ip = strings.TrimSpace(ip)
	ipBans[ip] = IPBan{IP: ip, Reason: reason, BannedBy: by, Timestamp: now()}
	saveBansLocked()
	return nil
}

func UnbanIP(ip string) error {
	mu.Lock()
	defer mu.Unlock()
	delete(ipBans, strings.TrimSpace(ip))
	saveBansLocked()
	return nil
}

func IsIPBanned(host string) (reason string, banned bool) {
	mu.Lock()
	defer mu.Unlock()
	if b, ok := ipBans[strings.TrimSpace(host)]; ok {
		return b.Reason, true
	}
	return "", false
}

func saveBansLocked() {
	if err := os.MkdirAll(moderationDir(), 0755); err != nil {
		mudlog.Error("moderation.saveBans", "error", err.Error())
		return
	}
	bf := banFile{}
	for _, b := range accountBans {
		bf.Accounts = append(bf.Accounts, b)
	}
	for _, b := range ipBans {
		bf.IPs = append(bf.IPs, b)
	}
	out, err := yaml.Marshal(bf)
	if err != nil {
		mudlog.Error("moderation.saveBans", "error", err.Error())
		return
	}
	if err := os.WriteFile(bansPath(), out, 0644); err != nil {
		mudlog.Error("moderation.saveBans", "error", err.Error())
	}
}

func loadBans() {
	mu.Lock()
	defer mu.Unlock()
	accountBans = map[string]AccountBan{}
	ipBans = map[string]IPBan{}
	b, err := os.ReadFile(bansPath())
	if err != nil {
		return
	}
	var bf banFile
	if err := yaml.Unmarshal(b, &bf); err != nil {
		mudlog.Error("moderation.loadBans", "error", err.Error())
		return
	}
	for _, a := range bf.Accounts {
		accountBans[strings.ToLower(a.Username)] = a
	}
	for _, i := range bf.IPs {
		ipBans[i.IP] = i
	}
}
```

- [ ] **Step 4: Wire bans into `moderation.go`**

In `internal/moderation/moderation.go`, extend the two functions the Task 1 stub
left petition-only:

```go
func resetForTest() {
	mu.Lock()
	defer mu.Unlock()
	petitions = nil
	nextPetitionId = 1
	accountBans = map[string]AccountBan{}
	ipBans = map[string]IPBan{}
}

func LoadDataFiles() {
	loadPetitions()
	loadBans()
	mudlog.Info("moderation.LoadDataFiles()", "petitions", len(petitions), "accountBans", len(accountBans), "ipBans", len(ipBans))
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/moderation/ -v`
Expected: PASS (all moderation tests).

- [ ] **Step 6: Commit**

```bash
git add internal/moderation/bans.go internal/moderation/moderation.go internal/moderation/moderation_test.go
git commit -m "feat(moderation): account + IP ban store + persistence"
```

---

## Task 3: boot-time load wiring + gitignore

**Files:**
- Modify: the boot loader that calls `guilds.LoadDataFiles()`
- Modify: `.gitignore`

- [ ] **Step 1: Find the boot loader**

Run: `grep -rn "guilds.LoadDataFiles" --include=*.go .`
Expected: one call site (the boot data-load sequence, e.g. in `world.go`/`main.go`/a `loadAllDataFiles`).

- [ ] **Step 2: Add the moderation load next to it**

In that file, add the import `"github.com/GoMudEngine/GoMud/internal/moderation"` and, immediately after the `guilds.LoadDataFiles()` line, add:

```go
moderation.LoadDataFiles()
```

- [ ] **Step 3: Add the gitignore entry**

Append to `.gitignore`:

```
# Moderation living state (petition queue + ban lists) — persist on prod, never commit
_datafiles/moderation/
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: builds clean.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore(moderation): load stores at boot; gitignore _datafiles/moderation"
```

---

## Task 4: config knobs (petition cooldown + max length)

**Files:**
- Modify: `internal/configs/config.gameplay.go`
- Modify: `_datafiles/config.yaml`

- [ ] **Step 1: Add the fields**

In `internal/configs/config.gameplay.go`, add to the `GamePlay` struct (follow the existing `ConfigInt` field style):

```go
	PetitionCooldownRounds ConfigInt `yaml:"PetitionCooldownRounds"` // Min rounds between a player's petitions (anti-spam)
	PetitionMaxLen         ConfigInt `yaml:"PetitionMaxLen"`         // Max characters in a petition message
```

In that struct's `Validate()` method, add defaults:

```go
	if g.PetitionCooldownRounds < 0 {
		g.PetitionCooldownRounds = 50
	}
	if g.PetitionMaxLen < 1 {
		g.PetitionMaxLen = 500
	}
```

*(If `config.gameplay.go`'s struct/accessor differs in name, match the file's
actual `GamePlay` struct and its `Validate()`. Accessor is `configs.GetGamePlayConfig()`.)*

- [ ] **Step 2: Add the yaml defaults**

In `_datafiles/config.yaml`, under the `GamePlay:` block, add:

```yaml
  PetitionCooldownRounds: 50
  PetitionMaxLen: 500
```

- [ ] **Step 3: Verify boot reads them**

Run: `go build ./...`
Expected: builds clean. (Values are exercised in Task 5.)

- [ ] **Step 4: Commit**

```bash
git add internal/configs/config.gameplay.go _datafiles/config.yaml
git commit -m "feat(config): PetitionCooldownRounds + PetitionMaxLen knobs"
```

---

## Task 5: player command — `petition`

**Files:**
- Create: `internal/usercommands/petition.go`
- Modify: `internal/usercommands/usercommands.go` (register)

- [ ] **Step 1: Write `internal/usercommands/petition.go`**

```go
package usercommands

import (
	"fmt"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/moderation"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

var (
	petitionCooldownMu sync.Mutex
	lastPetitionRound  = map[int]uint64{} // userId -> round of last petition
)

// Petition lets a player contact staff about anything (harassment, grief,
// stuck, request). Free-text; reporter, time, and room are captured
// automatically. Filed petitions land in a durable admin-reviewable queue and
// ping any online staff.
func Petition(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	rest = strings.TrimSpace(rest)
	if rest == "" {
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Usage: <ansi fg="cyan-bold">petition <message></ansi> — send a message to the staff (harassment, a stuck quest, anything).</ansi>`)
		return true, nil
	}

	gp := configs.GetGamePlayConfig()

	if maxLen := int(gp.PetitionMaxLen); maxLen > 0 && len(rest) > maxLen {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="yellow">That's a bit long — please keep your petition under %d characters.</ansi>`, maxLen))
		return true, nil
	}

	// Anti-spam cooldown.
	cd := uint64(gp.PetitionCooldownRounds)
	nowRound := util.GetRoundCount()
	petitionCooldownMu.Lock()
	last, seen := lastPetitionRound[user.UserId]
	if seen && cd > 0 && nowRound-last < cd {
		petitionCooldownMu.Unlock()
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">You've just sent a petition — give the staff a moment before sending another.</ansi>`)
		return true, nil
	}
	lastPetitionRound[user.UserId] = nowRound
	petitionCooldownMu.Unlock()

	p, err := moderation.Add(user.Username, user.Character.RoomId, user.Character.Zone, rest)
	if err != nil {
		user.SendText(messaging.CategorySystem, `<ansi fg="red">Your petition could not be filed. Please try again or find an admin.</ansi>`)
		return true, nil
	}

	user.SendText(messaging.CategorySystem, `<ansi fg="green">Your petition has been sent to the staff. They'll review it as soon as they can.</ansi>`)

	// Notify online staff.
	alert := fmt.Sprintf(`<ansi fg="alert-5">[PETITION #%d]</ansi> <ansi fg="username">%s</ansi> (%s): %s <ansi fg="black-bold">— type 'petitions' to review.</ansi>`,
		p.Id, user.Username, room.Title, rest)
	for _, staff := range users.GetAllActiveUsers() {
		if staff.Role != users.RoleUser {
			staff.SendText(messaging.CategorySystem, alert)
		}
	}

	return true, nil
}
```

- [ ] **Step 2: Register the command**

In `internal/usercommands/usercommands.go`, add to the `userCommands` map
(alphabetical neighborhood, near `pet`/`picklock`):

```go
		`petition`:        {Petition, true, true, false}, // player: contact staff
```

- [ ] **Step 3: Build + boot-smoke**

Run: `go build ./... && go vet ./internal/usercommands/`
Expected: clean.

- [ ] **Step 4: Manual verify (harness walk)** — deferred to the plan-level
integration check (Task 12); the command's logic (cooldown, maxlen, persist,
notify) is straight-line over the Task 1 store already unit-tested.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/petition.go internal/usercommands/usercommands.go
git commit -m "feat(cmd): petition — player contacts staff into the queue"
```

---

## Task 6: admin command — `petitions` (review queue)

**Files:**
- Create: `internal/usercommands/petitions.go`
- Modify: `internal/usercommands/usercommands.go` (register)

- [ ] **Step 1: Write `internal/usercommands/petitions.go`**

```go
package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/moderation"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Petitions is the admin review command for the petition queue.
// Usage: petitions | petitions all | petitions <id> | petitions resolve <id> [note]
func Petitions(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	fields := strings.Fields(rest)

	switch {
	case len(fields) == 0: // list open
		list := moderation.ListOpen()
		if len(list) == 0 {
			user.SendText(messaging.CategorySystem, `<ansi fg="green">No open petitions.</ansi>`)
			return true, nil
		}
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="cyan-bold">%d open petition(s):</ansi>`, len(list)))
		for _, p := range list {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`  <ansi fg="yellow">#%d</ansi> <ansi fg="username">%s</ansi>: %s`, p.Id, p.Reporter, petitionSnippet(p.Message, 60)))
		}
		return true, nil

	case fields[0] == "all":
		list := moderation.ListAll()
		if len(list) == 0 {
			user.SendText(messaging.CategorySystem, `<ansi fg="green">No petitions on record.</ansi>`)
			return true, nil
		}
		for _, p := range list {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`  <ansi fg="yellow">#%d</ansi> [%s] <ansi fg="username">%s</ansi>: %s`, p.Id, p.Status, p.Reporter, petitionSnippet(p.Message, 60)))
		}
		return true, nil

	case fields[0] == "resolve":
		if len(fields) < 2 {
			user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Usage: petitions resolve <id> [note]</ansi>`)
			return true, nil
		}
		id, err := strconv.Atoi(fields[1])
		if err != nil {
			user.SendText(messaging.CategorySystem, `<ansi fg="red">That is not a valid petition id.</ansi>`)
			return true, nil
		}
		note := strings.TrimSpace(strings.TrimPrefix(rest, fields[0]+" "+fields[1]))
		if err := moderation.Resolve(id, user.Username, note); err != nil {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="red">%s</ansi>`, err.Error()))
			return true, nil
		}
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="green">Petition #%d marked resolved.</ansi>`, id))
		return true, nil

	default: // detail by id
		id, err := strconv.Atoi(fields[0])
		if err != nil {
			user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Usage: petitions | petitions all | petitions <id> | petitions resolve <id> [note]</ansi>`)
			return true, nil
		}
		p, ok := moderation.Get(id)
		if !ok {
			user.SendText(messaging.CategorySystem, `<ansi fg="red">No petition with that id.</ansi>`)
			return true, nil
		}
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="cyan-bold">Petition #%d</ansi> [%s]`, p.Id, p.Status))
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`  From: <ansi fg="username">%s</ansi>  When: %s  Room: %d (%s)`, p.Reporter, p.Timestamp.Format("2006-01-02 15:04 MST"), p.RoomId, p.Zone))
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`  Message: %s`, p.Message))
		if p.Status == moderation.StatusResolved {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`  Resolved by <ansi fg="username">%s</ansi>: %s`, p.ResolvedBy, p.Note))
		}
		return true, nil
	}
}

func petitionSnippet(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > max {
		return s[:max-1] + "…"
	}
	return s
}
```

- [ ] **Step 2: Register (AdminOnly)**

In `usercommands.go` `userCommands` map:

```go
		`petitions`:       {Petitions, true, true, true}, // Admin only — review the petition queue
```

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/petitions.go internal/usercommands/usercommands.go
git commit -m "feat(cmd): petitions — admin review/resolve the queue"
```

---

## Task 7: enforcement seams — account + IP ban at login

**Files:**
- Modify: `internal/inputhandlers/login.go`

- [ ] **Step 1: Add the import**

In `login.go`'s import block add:

```go
	"github.com/GoMudEngine/GoMud/internal/moderation"
	"net"
```

- [ ] **Step 2: IP-ban check at the top of `FinalizeLoginOrCreate`**

Immediately after the function opens (before `username := results["username"]`),
add:

```go
	// IP ban: reject (and block re-registration) before any account work.
	if connDetails := connections.Get(clientInput.ConnectionId); connDetails != nil && !connDetails.IsLocal() {
		host, _, err := net.SplitHostPort(connDetails.RemoteAddr().String())
		if err != nil {
			host = connDetails.RemoteAddr().String()
		}
		if reason, banned := moderation.IsIPBanned(host); banned {
			connections.SendTo([]byte("Your connection has been banned. Reason: "+reason), clientInput.ConnectionId)
			connections.SendTo(term.CRLF, clientInput.ConnectionId)
			connections.Remove(clientInput.ConnectionId)
			return false
		}
	}
```

- [ ] **Step 3: Account-ban check after the password matches**

In the existing-user branch, immediately AFTER the password-match block (after
the `if !tmpUser.PasswordMatches(password) { ... }` block, before
`users.LoginUser(...)`), add:

```go
		if reason, banned := moderation.IsAccountBanned(username); banned {
			connections.SendTo([]byte("This account has been banned. Reason: "+reason), clientInput.ConnectionId)
			connections.SendTo(term.CRLF, clientInput.ConnectionId)
			connections.Remove(clientInput.ConnectionId)
			return false
		}
```

- [ ] **Step 4: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add internal/inputhandlers/login.go
git commit -m "feat(moderation): reject banned accounts + IPs at login"
```

---

## Task 8: admin command — `boot`

**Files:**
- Create: `internal/usercommands/boot.go`
- Modify: `internal/usercommands/usercommands.go` (register)

- [ ] **Step 1: Write `internal/usercommands/boot.go`**

```go
package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Boot force-disconnects an online player by name (server-wide, not room-scoped).
// Booted players may reconnect within the zombie window unless also banned.
// Usage: boot <name> [reason]
func Boot(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	fields := strings.Fields(rest)
	if len(fields) == 0 {
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Usage: boot <name> [reason]</ansi>`)
		return true, nil
	}
	name := fields[0]
	reason := strings.TrimSpace(strings.TrimPrefix(rest, name))
	if reason == "" {
		reason = "disconnected by staff"
	}

	target := users.GetByCharacterName(name)
	if target == nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="red">No online player named "%s".</ansi>`, name))
		return true, nil
	}

	target.EventLog.Add(`conn`, fmt.Sprintf("Booted by %s: %s", user.Username, reason))
	target.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="alert-5">You have been disconnected by staff.</ansi> %s`, reason))

	connections.Kick(target.ConnectionId(), fmt.Sprintf("Booted by %s: %s", user.Username, reason))

	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> has been <ansi fg="alert-5">booted</ansi>.`, target.Username))
	return true, nil
}
```

- [ ] **Step 2: Register (AdminOnly)**

```go
		`boot`:            {Boot, true, true, true}, // Admin only — force-disconnect a player
```

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/boot.go internal/usercommands/usercommands.go
git commit -m "feat(cmd): boot — admin force-disconnect a player by name"
```

---

## Task 9: admin commands — `ban` / `unban`

**Files:**
- Create: `internal/usercommands/ban.go`
- Create: `internal/usercommands/unban.go`
- Modify: `internal/usercommands/usercommands.go` (register)

- [ ] **Step 1: Write `internal/usercommands/ban.go`**

```go
package usercommands

import (
	"fmt"
	"net"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/moderation"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Ban permanently bans an account (by name) or an IP.
// Usage: ban <name> [reason]          — account ban (boots if online)
//        ban ip <name|ip> [reason]    — IP ban
func Ban(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	fields := strings.Fields(rest)
	if len(fields) == 0 {
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Usage: ban <name> [reason]  |  ban ip <name|ip> [reason]</ansi>`)
		return true, nil
	}

	// IP ban subcommand.
	if strings.ToLower(fields[0]) == "ip" && len(fields) >= 2 {
		targetArg := fields[1]
		reason := strings.TrimSpace(strings.TrimPrefix(rest, fields[0]+" "+targetArg))
		if reason == "" {
			reason = "banned by staff"
		}
		ip := targetArg
		// If it isn't a literal IP, treat it as an online player's name.
		if net.ParseIP(targetArg) == nil {
			target := users.GetByCharacterName(targetArg)
			if target == nil {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="red">No online player named "%s" (and "%s" is not an IP).</ansi>`, targetArg, targetArg))
				return true, nil
			}
			cd := connections.Get(target.ConnectionId())
			if cd == nil {
				user.SendText(messaging.CategorySystem, `<ansi fg="red">Could not read that player's connection.</ansi>`)
				return true, nil
			}
			host, _, err := net.SplitHostPort(cd.RemoteAddr().String())
			if err != nil {
				host = cd.RemoteAddr().String()
			}
			ip = host
			connections.Kick(target.ConnectionId(), "Banned by staff: "+reason)
		}
		_ = moderation.BanIP(ip, reason, user.Username)
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="alert-5">IP banned:</ansi> %s`, ip))
		return true, nil
	}

	// Account ban.
	name := fields[0]
	reason := strings.TrimSpace(strings.TrimPrefix(rest, name))
	if reason == "" {
		reason = "banned by staff"
	}
	_ = moderation.BanAccount(name, reason, user.Username)

	// Boot if online.
	if target := users.GetByCharacterName(name); target != nil {
		target.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="alert-5">You have been banned.</ansi> %s`, reason))
		connections.Kick(target.ConnectionId(), "Banned by staff: "+reason)
	}

	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> has been <ansi fg="alert-5">banned</ansi>. Reason: %s`, name, reason))
	return true, nil
}
```

- [ ] **Step 2: Write `internal/usercommands/unban.go`**

```go
package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/moderation"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Unban lifts an account ban (by name) or an IP ban.
// Usage: unban <name>  |  unban ip <ip>
func Unban(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	fields := strings.Fields(rest)
	if len(fields) == 0 {
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">Usage: unban <name>  |  unban ip <ip></ansi>`)
		return true, nil
	}

	if strings.ToLower(fields[0]) == "ip" && len(fields) >= 2 {
		_ = moderation.UnbanIP(fields[1])
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="green">IP unbanned:</ansi> %s`, fields[1]))
		return true, nil
	}

	_ = moderation.Unban(fields[0])
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="green"><ansi fg="username">%s</ansi> has been unbanned.</ansi>`, fields[0]))
	return true, nil
}
```

- [ ] **Step 3: Register (both AdminOnly)**

```go
		`ban`:             {Ban, true, true, true},   // Admin only
		`unban`:           {Unban, true, true, true}, // Admin only
```

- [ ] **Step 4: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/ban.go internal/usercommands/unban.go internal/usercommands/usercommands.go
git commit -m "feat(cmd): ban/unban — account + optional IP bans"
```

---

## Task 10: global-by-name mute/deafen

**Files:**
- Modify: `internal/usercommands/admin.mute.go`
- Modify: `internal/usercommands/admin.deafen.go`

- [ ] **Step 1: Add a shared global-resolve helper**

Create `internal/usercommands/moderation_target.go`:

```go
package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// resolveModTarget finds a player for a moderation action: first in the current
// room (fuzzy), then globally by character name. Returns nil if not found.
func resolveModTarget(room *rooms.Room, rest string) *users.UserRecord {
	if target, err := actions.ResolveTargetActor(room, rest); err == nil && target.IsPlayer() {
		return target.(*actions.UserActor).User
	}
	return users.GetByCharacterName(rest)
}
```

- [ ] **Step 2: Use it in `admin.mute.go`**

In `Mute` and `UnMute`, replace the `actions.ResolveTargetActor(...)` block +
`u := target.(*actions.UserActor).User` with:

```go
	u := resolveModTarget(room, rest)
	if u == nil {
		user.SendText(messaging.CategorySystem, "Could not find user.")
		return true, nil
	}
```

Remove the now-unused `actions` import if it is no longer referenced in the file
(the helper owns it). Keep the `u.Muted = true/false` and the confirmation lines.

- [ ] **Step 3: Same change in `admin.deafen.go`**

Apply the identical replacement in `Deafen`/`UnDeafen` (set `u.Deafened`).

- [ ] **Step 4: Build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/moderation_target.go internal/usercommands/admin.mute.go internal/usercommands/admin.deafen.go
git commit -m "feat(moderation): mute/deafen resolve targets globally by name"
```

---

## Task 11: docs

**Files:**
- Modify: `CLAUDE.md`
- Modify: `docs/PATH_TO_1.0.md`

- [ ] **Step 1: CLAUDE.md persistence note**

Add a short section near the "Shop Persistence" note:

```markdown
## Moderation Persistence
Petition queue + ban lists live in `_datafiles/moderation/` (`petitions.yaml`,
`bans.yaml`). Like `shops/` and `guilds/`, this is persistent living state — it
is gitignored, kept on prod, and must NOT be wiped by the instance-save
smoke-test SOP. A malformed file logs + skips at boot (does not panic).
```

- [ ] **Step 2: PATH_TO_1.0 §4 update**

Update the §4 "Player-vs-player report" bullet to record that the `petition`
command + queue + `boot`/`ban`/global-mute shipped, and that the `report`
command was found to be a vital-bar utility (not moderation).

- [ ] **Step 3: Commit**

```bash
git add CLAUDE.md docs/PATH_TO_1.0.md
git commit -m "docs(moderation): persistence note + PATH_TO_1.0 §4 status"
```

---

## Task 12: integration verification (full boot + harness walk)

**Files:** none (verification only)

- [ ] **Step 1: Nuke instances + boot**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/dogmud_mod.log 2>&1 &
```
Watch for `moderation.LoadDataFiles()` in the log and a clean boot (no panics,
`ValidateZoneConsistency errors=0`).

- [ ] **Step 2: Drive the harness (two aspects)**

Using the `playtest` harness bridge (see the playtest skill):
1. As a normal player, `petition Bob is harassing me` → confirm the green
   confirmation, and that `_datafiles/moderation/petitions.yaml` gains an entry.
2. As an admin character (edit a test user's `role: admin` in their save, or use
   an existing admin), verify: the `[PETITION #1]` staff alert arrived,
   `petitions` lists it, `petitions 1` shows detail, `petitions resolve 1 done`
   flips status; `boot <player>` disconnects a second test client; `ban <name>
   grief` then a reconnect attempt is rejected with the reason at login.

- [ ] **Step 3: Record results + stop the server**

Note outcomes in the branch. Kill the server (by port 55555 owner).

- [ ] **Step 4: Final full test run**

Run: `go test ./...`
Expected: EXIT 0.

---

## Self-review checklist (run after implementing)

- Every spec section maps to a task: petition queue (T1), bans (T2), boot-load
  (T3), config (T4), petition cmd (T5), petitions review (T6), ban seams (T7),
  boot (T8), ban/unban (T9), global mute/deafen (T10), docs (T11), verify (T12). ✅
- Type names consistent across tasks: `moderation.Add/ListOpen/ListAll/Get/
  Resolve`, `BanAccount/BanIP/Unban/UnbanIP/IsAccountBanned/IsIPBanned`,
  `StatusOpen/StatusResolved`, command handlers `Petition/Petitions/Boot/Ban/
  Unban`, helper `resolveModTarget`. ✅
- No placeholders — every code step shows real code. ✅
