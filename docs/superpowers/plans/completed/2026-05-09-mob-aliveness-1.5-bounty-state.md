# Mob Aliveness 1.5 — Bounty State Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a per-bounty registry substrate (issuer × target × reward) with auto-claim on MobDeath, a quest engine `declare_bounty` action, admin + player `bounty` commands, and two physical bounty boards (Thornwall Guard Barracks 473 + Stillwater Constabulary 4110).

**Architecture:** New `internal/bounties/` package mirroring chunks 1.3 / 1.4 / opinions / factions. Single registry YAML at `_datafiles/world/dogmud/bounties.yaml` (gitignored). Lazy-load + cache + sync-save with double-check-lock and marshal-under-RLock (the lessons from chunk 1.4 review baked in v1). Polymorphic target reuses chunk 1.4's `knowledge.Subject`. Auto-claim hook listens for `MobDeath` and pays out gold + faction rep when an open bounty's target dies.

**Tech Stack:** Go 1.21+, YAML persistence (`gopkg.in/yaml.v3`), existing `internal/events` event bus, existing `internal/configs/Balance` config, existing `internal/questengine` for the action.

**Spec:** `docs/superpowers/specs/2026-05-09-mob-aliveness-1.5-bounty-state-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP for the aliveness effort.

---

## File structure

| File | Responsibility |
|------|----------------|
| `internal/bounties/types.go` | `IssuerType`, `Issuer`, `Status`, `Condition`, `Bounty`, `Registry` types + Issuer helpers |
| `internal/bounties/persistence.go` | Base dir, filepath helper, YAML load/save, lazy-load + cache + mutex |
| `internal/bounties/bounties.go` | Public API (Declare, Withdraw, TryClaim, MarkExpired, PruneExpired, queries) + default reward helpers |
| `internal/bounties/test_main_test.go` | TestMain temp-dir harness |
| `internal/bounties/types_test.go` | Type-level tests |
| `internal/bounties/persistence_test.go` | Round-trip + concurrent-load tests |
| `internal/bounties/bounties_test.go` | API tests |
| `internal/configs/config.balance.go`, `config.balance.misc.go` | `BountyGoldDefaultMultiplier`, `BountyGoldFloor` knobs |
| `internal/hooks/MobDeath_BountyClaim.go` | Auto-claim listener |
| `internal/hooks/hooks.go` | Register the new listener |
| `internal/hooks/MobDeath_BountyClaim_test.go` | Hook integration test |
| `internal/questengine/types.go` | Add `DeclareBounty *DeclareBountyDef` to Action struct |
| `internal/questengine/actions.go` | Dispatch `declare_bounty` action |
| `internal/usercommands/admin.bounty.go` | Admin `bounty` command |
| `internal/usercommands/bounty.go` | Player `bounty` command |
| `internal/usercommands/usercommands.go` | Register both `bounty` commands |
| `_datafiles/world/dogmud/templates/admincommands/help/command.bounty.template` | Admin helpfile |
| `_datafiles/world/dogmud/templates/usercommands/help/command.bounty.template` | Player helpfile |
| `_datafiles/world/dogmud/rooms/thornwall_city/473.yaml` | Add `bounty board` noun |
| `_datafiles/world/dogmud/rooms/stillwater/4110.yaml` | Add `bounty board` noun |
| `MOB_ALIVENESS_ROADMAP.md` | Mark 1.5 as Done |

---

## Task 1: Package skeleton + types

**Files:**
- Create: `internal/bounties/types.go`
- Create: `internal/bounties/types_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/bounties/types_test.go
package bounties

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/knowledge"
)

func TestIssuerHelpers(t *testing.T) {
	if FactionIssuer("thornwall_guards") != (Issuer{Type: IssuerFaction, Id: "thornwall_guards"}) {
		t.Errorf("FactionIssuer mismatch")
	}
	if QuestIssuer(14) != (Issuer{Type: IssuerQuest, Id: "14"}) {
		t.Errorf("QuestIssuer mismatch")
	}
	if NPCIssuer(357) != (Issuer{Type: IssuerNPC, Id: "357"}) {
		t.Errorf("NPCIssuer mismatch")
	}
}

func TestBountyDefaultStatus(t *testing.T) {
	b := &Bounty{
		Id:     1,
		Issuer: FactionIssuer("thornwall_guards"),
		Target: knowledge.PlayerSubject(17),
		Status: StatusOpen,
	}
	if b.Status != StatusOpen {
		t.Errorf("Status default mismatch: got %s", b.Status)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/bounties/...`
Expected: build failure — package doesn't exist yet.

- [ ] **Step 3: Implement types.go**

```go
// internal/bounties/types.go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/bounties/...`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/bounties/types.go internal/bounties/types_test.go
git commit -m "feat(bounties): types + issuer helpers (chunk 1.5 T1)"
```

---

## Task 2: TestMain + persistence skeleton

**Files:**
- Create: `internal/bounties/test_main_test.go`
- Create: `internal/bounties/persistence.go`

- [ ] **Step 1: Read the chunk-1.4 reference**

Open `internal/knowledge/test_main_test.go` and `internal/knowledge/persistence.go`. The bounties package mirrors that pattern exactly — temp-dir TestMain with `configs.AddOverlayOverrides`, lazy-load with double-check-lock, marshal-under-RLock for save.

- [ ] **Step 2: Write test_main_test.go**

```go
// internal/bounties/test_main_test.go
package bounties

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "bounties-test-*")
	if err != nil {
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles": tmp,
	})

	if err := os.MkdirAll(filepath.Join(tmp, "world", "dogmud"), 0o755); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// resetCache wipes the in-memory cache between tests.
func resetCache() {
	registryMu.Lock()
	registry = nil
	registryMu.Unlock()
}
```

NOTE: copy the chunk-1.4 pattern exactly. If `configs.AddOverlayOverrides` isn't the right setter for this codebase, mirror what chunk-1.4's `test_main_test.go` actually uses.

- [ ] **Step 3: Implement persistence.go skeleton**

```go
// internal/bounties/persistence.go
package bounties

import (
	"path/filepath"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	_ "gopkg.in/yaml.v3" // Pre-imported for T3 YAML marshaling
)

var (
	registry   *Registry
	registryMu sync.RWMutex
	saveMu     sync.Mutex // serializes disk writes (Windows file-lock safety)
)

func registryFilePath() string {
	return filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "world", "dogmud", "bounties.yaml")
}
```

- [ ] **Step 4: Verify build**

Run: `go test ./internal/bounties/...`
Expected: PASS (T1 tests still pass; new code compiles).

- [ ] **Step 5: Commit**

```bash
git add internal/bounties/test_main_test.go internal/bounties/persistence.go
git commit -m "feat(bounties): test harness + persistence skeleton (T2)"
```

---

## Task 3: YAML save/load round-trip

**Files:**
- Modify: `internal/bounties/persistence.go`
- Create: `internal/bounties/persistence_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/bounties/persistence_test.go
package bounties

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/knowledge"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	resetCache()

	r := &Registry{
		NextId: 2,
		Bounties: []*Bounty{
			{
				Id:             1,
				Issuer:         FactionIssuer("thornwall_guards"),
				Target:         knowledge.PlayerSubject(17),
				GoldReward:     300,
				RepReward:      6,
				Condition:      ConditionKill,
				DeclaredRound:  100,
				ExpiryRound:    1000,
				Status:         StatusOpen,
				DeclaredReason: "Murder of Marek",
			},
		},
	}

	if err := saveRegistry(r); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded := loadRegistryFromDisk()
	if loaded == nil {
		t.Fatalf("expected non-nil load")
	}
	if loaded.NextId != 2 {
		t.Errorf("NextId mismatch: got %d", loaded.NextId)
	}
	if len(loaded.Bounties) != 1 {
		t.Fatalf("expected 1 bounty, got %d", len(loaded.Bounties))
	}
	b := loaded.Bounties[0]
	if b.Issuer != FactionIssuer("thornwall_guards") {
		t.Errorf("issuer mismatch: %+v", b.Issuer)
	}
	if b.Target != knowledge.PlayerSubject(17) {
		t.Errorf("target mismatch: %+v", b.Target)
	}
	if b.GoldReward != 300 || b.RepReward != 6 {
		t.Errorf("rewards mismatch: gold=%d rep=%d", b.GoldReward, b.RepReward)
	}
	if b.DeclaredReason != "Murder of Marek" {
		t.Errorf("reason mismatch: %q", b.DeclaredReason)
	}
}

func TestLoadMissingFileReturnsNil(t *testing.T) {
	resetCache()
	// New temp dir doesn't have a bounties.yaml yet.
	if loadRegistryFromDisk() != nil {
		t.Errorf("expected nil for missing file")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/bounties/...`
Expected: FAIL (saveRegistry/loadRegistryFromDisk undefined).

- [ ] **Step 3: Implement save/load**

Append to `internal/bounties/persistence.go`:

```go
import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// saveRegistry serializes the registry under cache RLock to ensure
// a consistent snapshot, then writes via tmp-rename for atomicity.
// (The chunk-1.4 review caught the missing RLock-around-marshal
// pattern; bake it in v1 here.)
func saveRegistry(r *Registry) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(registryFilePath()), 0o755); err != nil {
		return fmt.Errorf("mkdir bounties dir: %w", err)
	}

	registryMu.RLock()
	out, err := yaml.Marshal(r)
	registryMu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}

	path := registryFilePath()
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return fmt.Errorf("write tmp file %q: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename tmp -> final %q: %w", path, err)
	}
	return nil
}

func loadRegistryFromDisk() *Registry {
	data, err := os.ReadFile(registryFilePath())
	if err != nil {
		return nil
	}
	r := &Registry{}
	if err := yaml.Unmarshal(data, r); err != nil {
		return nil
	}
	return r
}
```

You will need to:
- Reactivate the `os` and `yaml` imports (T2 may have made yaml a blank import; flip it back).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/bounties/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/bounties/persistence.go internal/bounties/persistence_test.go
git commit -m "feat(bounties): YAML round-trip with marshal-under-RLock (T3)"
```

---

## Task 4: Lazy-load + double-check-lock

**Files:**
- Modify: `internal/bounties/persistence.go`
- Modify: `internal/bounties/persistence_test.go`

- [ ] **Step 1: Write the failing concurrent-load test**

Append to `internal/bounties/persistence_test.go`:

```go
import (
	"sync"
)

func TestLoadOrLazyInitConcurrent(t *testing.T) {
	resetCache()
	const N = 50
	var wg sync.WaitGroup
	var seen [N]*Registry
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			seen[i] = loadOrLazyInit()
		}(i)
	}
	wg.Wait()

	first := seen[0]
	if first == nil {
		t.Fatalf("nil result from loadOrLazyInit")
	}
	for i := 1; i < N; i++ {
		if seen[i] != first {
			t.Errorf("goroutine %d got different pointer than goroutine 0", i)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/bounties/...`
Expected: FAIL (`loadOrLazyInit` undefined).

- [ ] **Step 3: Implement loadOrLazyInit**

Append to `internal/bounties/persistence.go`:

```go
// loadOrLazyInit returns the cached *Registry, loading from disk on
// first access. If neither cache nor disk has data, an empty
// Registry is created and cached. Mirrors the chunk 1.3 / 1.4
// double-check-lock pattern.
func loadOrLazyInit() *Registry {
	registryMu.RLock()
	if registry != nil {
		r := registry
		registryMu.RUnlock()
		return r
	}
	registryMu.RUnlock()

	if r := loadRegistryFromDisk(); r != nil {
		registryMu.Lock()
		if registry != nil {
			cached := registry
			registryMu.Unlock()
			return cached
		}
		registry = r
		registryMu.Unlock()
		return r
	}

	r := &Registry{
		NextId:   1,
		Bounties: []*Bounty{},
	}
	registryMu.Lock()
	if registry != nil {
		cached := registry
		registryMu.Unlock()
		return cached
	}
	registry = r
	registryMu.Unlock()
	return r
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/bounties/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/bounties/persistence.go internal/bounties/persistence_test.go
git commit -m "feat(bounties): lazy-load + double-check-lock (T4)"
```

---

## Task 5: Config knobs + default reward helpers

**Files:**
- Modify: `internal/configs/config.balance.go`
- Modify: `internal/configs/config.balance.misc.go`
- Create: `internal/bounties/bounties.go`
- Create: `internal/bounties/bounties_test.go`

- [ ] **Step 1: Add config knobs**

Open `internal/configs/config.balance.go`, locate the existing knobs added in chunks 1.3 / 1.4 (e.g. `KnowledgeObservationLogMax`), and add:

```go
BountyGoldDefaultMultiplier ConfigFloat `yaml:"bounty_gold_default_multiplier"`
BountyGoldFloor             int         `yaml:"bounty_gold_floor"`
```

(Use the same type form as nearby float/int config fields — match the existing convention.)

In `internal/configs/config.balance.misc.go` `validateMisc()`, add defaults:

```go
if b.BountyGoldDefaultMultiplier == 0 {
    b.BountyGoldDefaultMultiplier = 0.5
}
if b.BountyGoldFloor == 0 {
    b.BountyGoldFloor = 50
}
```

- [ ] **Step 2: Write failing test for default reward helpers**

```go
// internal/bounties/bounties_test.go
package bounties

import (
	"testing"
)

func TestComputeDefaultGold(t *testing.T) {
	defer func() { goldMultiplierForTest = nil; goldFloorForTest = nil }()
	goldMultiplierForTest = func() float64 { return 0.5 }
	goldFloorForTest = func() int { return 50 }

	cases := []struct {
		statpool int
		want     int
	}{
		{600, 300}, // 600 * 0.5 = 300
		{1000, 500},
		{50, 50},  // floor wins (25 -> 50)
		{0, 50},   // floor wins
		{200, 100},
	}
	for _, c := range cases {
		if got := computeDefaultGold(c.statpool); got != c.want {
			t.Errorf("statpool=%d: got %d, want %d", c.statpool, got, c.want)
		}
	}
}

func TestComputeDefaultRep(t *testing.T) {
	cases := []struct {
		statpool int
		want     int
	}{
		{600, 6},
		{100, 1},
		{50, 1}, // floor of 1
		{0, 1},
		{1000, 10},
	}
	for _, c := range cases {
		if got := computeDefaultRep(c.statpool); got != c.want {
			t.Errorf("statpool=%d: got %d, want %d", c.statpool, got, c.want)
		}
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/bounties/...`
Expected: FAIL (functions undefined).

- [ ] **Step 4: Implement bounties.go skeleton with default helpers**

```go
// internal/bounties/bounties.go
package bounties

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Test seams.
var (
	roundForTest          func() uint64
	goldMultiplierForTest func() float64
	goldFloorForTest      func() int
)

func currentRound() uint64 {
	if roundForTest != nil {
		return roundForTest()
	}
	return util.GetRoundCount()
}

func goldMultiplier() float64 {
	if goldMultiplierForTest != nil {
		return goldMultiplierForTest()
	}
	return float64(configs.GetBalanceConfig().BountyGoldDefaultMultiplier)
}

func goldFloor() int {
	if goldFloorForTest != nil {
		return goldFloorForTest()
	}
	return configs.GetBalanceConfig().BountyGoldFloor
}

// computeDefaultGold returns floor(statpool * multiplier), with a
// floor of `BountyGoldFloor` so trivial mobs still pay a meaningful
// amount.
func computeDefaultGold(statpool int) int {
	g := int(float64(statpool) * goldMultiplier())
	if floor := goldFloor(); g < floor {
		return floor
	}
	return g
}

// computeDefaultRep returns max(1, floor(statpool / 100)).
func computeDefaultRep(statpool int) int {
	r := statpool / 100
	if r < 1 {
		return 1
	}
	return r
}
```

(If `configs.GetBalanceConfig().BountyGoldDefaultMultiplier` is some custom float type instead of plain float, cast appropriately. Look at how chunk 1.4's `KnowledgeObservationLogMax` is read — match that.)

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/bounties/... ./internal/configs/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/bounties/ internal/configs/config.balance.go internal/configs/config.balance.misc.go
git commit -m "feat(bounties): config knobs + default reward helpers (T5)"
```

---

## Task 6: Public API — Declare

**Files:**
- Modify: `internal/bounties/bounties.go`
- Modify: `internal/bounties/bounties_test.go`

- [ ] **Step 1: Write failing tests**

Append to `internal/bounties/bounties_test.go`:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/knowledge"
)

func TestDeclare_DefaultRewards(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil; goldMultiplierForTest = nil; goldFloorForTest = nil; statpoolForTest = nil }()
	roundForTest = func() uint64 { return 100 }
	goldMultiplierForTest = func() float64 { return 0.5 }
	goldFloorForTest = func() int { return 50 }
	// Stub the statpool lookup so the test doesn't need a real mob fixture.
	statpoolForTest = func(target knowledge.Subject) int { return 600 }

	id, err := Declare(
		FactionIssuer("thornwall_guards"),
		knowledge.PlayerSubject(17),
		ConditionKill,
		1000,
		DeclareOpts{},
	)
	if err != nil {
		t.Fatalf("Declare returned error: %v", err)
	}
	if id != 1 {
		t.Errorf("expected first id=1, got %d", id)
	}

	b := Get(id)
	if b == nil {
		t.Fatalf("Get returned nil for id %d", id)
	}
	if b.GoldReward != 300 {
		t.Errorf("default gold should be 300 (600 * 0.5), got %d", b.GoldReward)
	}
	if b.RepReward != 6 {
		t.Errorf("default rep should be 6 (600/100), got %d", b.RepReward)
	}
	if b.Status != StatusOpen {
		t.Errorf("status should be open, got %s", b.Status)
	}
	if b.DeclaredRound != 100 {
		t.Errorf("DeclaredRound mismatch: %d", b.DeclaredRound)
	}
}

func TestDeclare_Overrides(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil; statpoolForTest = nil }()
	roundForTest = func() uint64 { return 100 }
	statpoolForTest = func(target knowledge.Subject) int { return 600 }

	id, _ := Declare(
		FactionIssuer("thornwall_guards"),
		knowledge.PlayerSubject(17),
		ConditionKill,
		1000,
		DeclareOpts{
			GoldOverride:   1500,
			RepOverride:    25,
			DeclaredReason: "High-value target",
		},
	)
	b := Get(id)
	if b.GoldReward != 1500 {
		t.Errorf("override gold not honored: %d", b.GoldReward)
	}
	if b.RepReward != 25 {
		t.Errorf("override rep not honored: %d", b.RepReward)
	}
	if b.DeclaredReason != "High-value target" {
		t.Errorf("reason not stored: %q", b.DeclaredReason)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/bounties/...`
Expected: FAIL (Declare, Get, DeclareOpts, statpoolForTest undefined).

- [ ] **Step 3: Implement Declare + Get + statpool lookup seam**

Append to `internal/bounties/bounties.go`:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type DeclareOpts struct {
	GoldOverride   int
	RepOverride    int
	DeclaredReason string
}

// Test seam: lookup target's statpool. Production resolves via
// mobs.GetMobSpec or users.LoadUser depending on the subject type.
var statpoolForTest func(target knowledge.Subject) int

func statpoolFor(target knowledge.Subject) int {
	if statpoolForTest != nil {
		return statpoolForTest(target)
	}
	switch target.Type {
	case knowledge.SubjectMob:
		spec := mobs.GetMobSpec(mobs.MobId(target.Id))
		if spec == nil {
			return 0
		}
		return spec.Character.GetStatPool()
	case knowledge.SubjectPlayer:
		// Use online lookup first, disk-load fallback if offline.
		u := users.GetByUserId(target.Id)
		if u == nil {
			loaded, err := users.LoadUser(""+strconv.Itoa(target.Id), true)
			if err != nil || loaded == nil {
				return 0
			}
			u = loaded
		}
		return u.Character.GetStatPool()
	}
	return 0
}

func Declare(issuer Issuer, target knowledge.Subject, condition Condition,
	expiryRound uint64, opts DeclareOpts) (int, error) {

	r := loadOrLazyInit()
	now := currentRound()
	statpool := statpoolFor(target)

	gold := opts.GoldOverride
	if gold == 0 {
		gold = computeDefaultGold(statpool)
	}
	rep := opts.RepOverride
	if rep == 0 {
		rep = computeDefaultRep(statpool)
	}

	registryMu.Lock()
	id := r.NextId
	r.NextId++
	b := &Bounty{
		Id:             id,
		Issuer:         issuer,
		Target:         target,
		GoldReward:     gold,
		RepReward:      rep,
		Condition:      condition,
		DeclaredRound:  now,
		ExpiryRound:    expiryRound,
		Status:         StatusOpen,
		DeclaredReason: opts.DeclaredReason,
	}
	r.Bounties = append(r.Bounties, b)
	registryMu.Unlock()

	if err := saveRegistry(r); err != nil {
		mudlog.Warn("bounties.Declare: save failed", "id", id, "error", err)
		return id, err
	}
	return id, nil
}

func Get(bountyId int) *Bounty {
	r := loadOrLazyInit()
	registryMu.RLock()
	defer registryMu.RUnlock()
	for _, b := range r.Bounties {
		if b.Id == bountyId {
			return b
		}
	}
	return nil
}
```

NOTE: the lookup uses `users.LoadUser` for offline-player support (mirrors the helper added at chunk 1.4 review time). If `Character.GetStatPool` doesn't exist with that exact name, find the equivalent in `internal/characters/`. Cross-reference how other code computes "total stats" — the goal is a single integer summing all 6 baseline stats.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/bounties/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/bounties/
git commit -m "feat(bounties): Declare + Get with reward auto-compute (T6)"
```

---

## Task 7: Withdraw + MarkExpired + PruneExpired

**Files:**
- Modify: `internal/bounties/bounties.go`
- Modify: `internal/bounties/bounties_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestWithdraw(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil; statpoolForTest = nil }()
	roundForTest = func() uint64 { return 100 }
	statpoolForTest = func(_ knowledge.Subject) int { return 600 }

	id, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.PlayerSubject(17), ConditionKill, 1000, DeclareOpts{})

	Withdraw(id)
	b := Get(id)
	if b.Status != StatusWithdrawn {
		t.Errorf("expected withdrawn, got %s", b.Status)
	}

	// Idempotent on already-withdrawn.
	Withdraw(id)
	b = Get(id)
	if b.Status != StatusWithdrawn {
		t.Errorf("status should still be withdrawn: %s", b.Status)
	}
}

func TestPruneExpired(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil; statpoolForTest = nil }()
	statpoolForTest = func(_ knowledge.Subject) int { return 600 }

	roundForTest = func() uint64 { return 100 }
	idA, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.PlayerSubject(17), ConditionKill, 200, DeclareOpts{}) // expires at 200
	idB, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.PlayerSubject(18), ConditionKill, 0, DeclareOpts{})   // never expires
	idC, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.PlayerSubject(19), ConditionKill, 5000, DeclareOpts{}) // far future

	roundForTest = func() uint64 { return 300 }
	count := PruneExpired()
	if count != 1 {
		t.Errorf("expected 1 pruned (idA), got %d", count)
	}
	if Get(idA).Status != StatusExpired {
		t.Errorf("idA should be expired")
	}
	if Get(idB).Status != StatusOpen {
		t.Errorf("idB should still be open (never expires)")
	}
	if Get(idC).Status != StatusOpen {
		t.Errorf("idC should still be open (future expiry)")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/bounties/...`
Expected: FAIL (Withdraw, MarkExpired, PruneExpired undefined).

- [ ] **Step 3: Implement**

Append to `internal/bounties/bounties.go`:

```go
func Withdraw(bountyId int) {
	r := loadOrLazyInit()

	registryMu.Lock()
	mutated := false
	for _, b := range r.Bounties {
		if b.Id == bountyId && b.Status == StatusOpen {
			b.Status = StatusWithdrawn
			mutated = true
			break
		}
	}
	registryMu.Unlock()

	if mutated {
		if err := saveRegistry(r); err != nil {
			mudlog.Warn("bounties.Withdraw: save failed", "id", bountyId, "error", err)
		}
	}
}

func MarkExpired(bountyId int) {
	r := loadOrLazyInit()

	registryMu.Lock()
	mutated := false
	for _, b := range r.Bounties {
		if b.Id == bountyId && b.Status == StatusOpen {
			b.Status = StatusExpired
			mutated = true
			break
		}
	}
	registryMu.Unlock()

	if mutated {
		if err := saveRegistry(r); err != nil {
			mudlog.Warn("bounties.MarkExpired: save failed", "id", bountyId, "error", err)
		}
	}
}

// PruneExpired walks open bounties and flips any past their expiry
// round to status=expired. Returns the count flipped.
func PruneExpired() int {
	r := loadOrLazyInit()
	now := currentRound()

	registryMu.Lock()
	count := 0
	for _, b := range r.Bounties {
		if b.Status != StatusOpen {
			continue
		}
		if b.ExpiryRound == 0 {
			continue // never expires
		}
		if b.ExpiryRound <= now {
			b.Status = StatusExpired
			count++
		}
	}
	registryMu.Unlock()

	if count > 0 {
		if err := saveRegistry(r); err != nil {
			mudlog.Warn("bounties.PruneExpired: save failed", "error", err)
		}
	}
	return count
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/bounties/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/bounties/
git commit -m "feat(bounties): Withdraw + MarkExpired + PruneExpired (T7)"
```

---

## Task 8: TryClaim

**Files:**
- Modify: `internal/bounties/bounties.go`
- Modify: `internal/bounties/bounties_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestTryClaim_HappyPath(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil; statpoolForTest = nil }()
	roundForTest = func() uint64 { return 100 }
	statpoolForTest = func(_ knowledge.Subject) int { return 600 }

	id, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.MobSubject(101), ConditionKill, 1000, DeclareOpts{})

	roundForTest = func() uint64 { return 200 }
	b, ok := TryClaim(id, knowledge.PlayerSubject(17))
	if !ok {
		t.Fatalf("TryClaim should have succeeded")
	}
	if b.Status != StatusClaimed {
		t.Errorf("status should be claimed, got %s", b.Status)
	}
	if b.ClaimedBy != knowledge.PlayerSubject(17) {
		t.Errorf("ClaimedBy mismatch: %+v", b.ClaimedBy)
	}
	if b.ClaimedRound != 200 {
		t.Errorf("ClaimedRound mismatch: %d", b.ClaimedRound)
	}
}

func TestTryClaim_AlreadyClaimedReturnsFalse(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil; statpoolForTest = nil }()
	roundForTest = func() uint64 { return 100 }
	statpoolForTest = func(_ knowledge.Subject) int { return 600 }

	id, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.MobSubject(101), ConditionKill, 1000, DeclareOpts{})

	TryClaim(id, knowledge.PlayerSubject(17))
	b, ok := TryClaim(id, knowledge.PlayerSubject(99))
	if ok {
		t.Errorf("second TryClaim should have returned false")
	}
	if b != nil {
		t.Errorf("second TryClaim should have returned nil bounty")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/bounties/...`
Expected: FAIL (TryClaim undefined).

- [ ] **Step 3: Implement**

Append to `internal/bounties/bounties.go`:

```go
// TryClaim records a claim on the given bounty. Returns (bounty, true)
// on success (status was open, now claimed). Returns (nil, false) if
// the bounty was already non-open.
func TryClaim(bountyId int, claimer knowledge.Subject) (*Bounty, bool) {
	r := loadOrLazyInit()
	now := currentRound()

	registryMu.Lock()
	var claimed *Bounty
	for _, b := range r.Bounties {
		if b.Id == bountyId && b.Status == StatusOpen {
			b.Status = StatusClaimed
			b.ClaimedBy = claimer
			b.ClaimedRound = now
			claimed = b
			break
		}
	}
	registryMu.Unlock()

	if claimed == nil {
		return nil, false
	}

	if err := saveRegistry(r); err != nil {
		mudlog.Warn("bounties.TryClaim: save failed", "id", bountyId, "error", err)
	}
	return claimed, true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/bounties/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/bounties/
git commit -m "feat(bounties): TryClaim (T8)"
```

---

## Task 9: Read API (AllOpen, OpenForTarget, OpenForIssuer, OpenAgainstPlayer, AllForTarget)

**Files:**
- Modify: `internal/bounties/bounties.go`
- Modify: `internal/bounties/bounties_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestReadAPI(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil; statpoolForTest = nil }()
	roundForTest = func() uint64 { return 100 }
	statpoolForTest = func(_ knowledge.Subject) int { return 600 }

	idA, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.PlayerSubject(17), ConditionKill, 1000, DeclareOpts{})
	idB, _ := Declare(FactionIssuer("thornwall_citizens"),
		knowledge.PlayerSubject(17), ConditionKill, 1000, DeclareOpts{})
	idC, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.MobSubject(101), ConditionKill, 1000, DeclareOpts{})
	Withdraw(idC)

	// AllOpen returns A and B (C is withdrawn).
	open := AllOpen()
	if len(open) != 2 {
		t.Errorf("AllOpen: got %d, want 2", len(open))
	}

	// OpenForTarget(player 17) returns A and B.
	tgt := OpenForTarget(knowledge.PlayerSubject(17))
	if len(tgt) != 2 {
		t.Errorf("OpenForTarget: got %d, want 2", len(tgt))
	}

	// OpenForIssuer(thornwall_guards) returns only A (C withdrawn).
	iss := OpenForIssuer(FactionIssuer("thornwall_guards"))
	if len(iss) != 1 || iss[0].Id != idA {
		t.Errorf("OpenForIssuer: got %v", iss)
	}

	// OpenAgainstPlayer convenience.
	pl := OpenAgainstPlayer(17)
	if len(pl) != 2 {
		t.Errorf("OpenAgainstPlayer: got %d, want 2", len(pl))
	}

	// AllForTarget with includeNonOpen=true sees the withdrawn one.
	all := AllForTarget(knowledge.MobSubject(101), true)
	if len(all) != 1 || all[0].Id != idC {
		t.Errorf("AllForTarget(include): got %v", all)
	}
	open101 := AllForTarget(knowledge.MobSubject(101), false)
	if len(open101) != 0 {
		t.Errorf("AllForTarget(open only): got %v", open101)
	}

	_ = idB
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/bounties/...`
Expected: FAIL (AllOpen etc. undefined).

- [ ] **Step 3: Implement**

Append to `internal/bounties/bounties.go`:

```go
func AllOpen() []*Bounty {
	r := loadOrLazyInit()
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Bounty, 0)
	for _, b := range r.Bounties {
		if b.Status == StatusOpen {
			out = append(out, b)
		}
	}
	return out
}

func OpenForTarget(target knowledge.Subject) []*Bounty {
	r := loadOrLazyInit()
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Bounty, 0)
	for _, b := range r.Bounties {
		if b.Status == StatusOpen && b.Target == target {
			out = append(out, b)
		}
	}
	return out
}

func OpenForIssuer(issuer Issuer) []*Bounty {
	r := loadOrLazyInit()
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Bounty, 0)
	for _, b := range r.Bounties {
		if b.Status == StatusOpen && b.Issuer == issuer {
			out = append(out, b)
		}
	}
	return out
}

func OpenAgainstPlayer(userId int) []*Bounty {
	return OpenForTarget(knowledge.PlayerSubject(userId))
}

func AllForTarget(target knowledge.Subject, includeNonOpen bool) []*Bounty {
	r := loadOrLazyInit()
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Bounty, 0)
	for _, b := range r.Bounties {
		if b.Target != target {
			continue
		}
		if !includeNonOpen && b.Status != StatusOpen {
			continue
		}
		out = append(out, b)
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/bounties/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/bounties/
git commit -m "feat(bounties): read API — AllOpen + OpenForTarget/Issuer + AllForTarget (T9)"
```

---

## Task 10: Auto-claim hook

**Files:**
- Create: `internal/hooks/MobDeath_BountyClaim.go`
- Modify: `internal/hooks/hooks.go`
- Create: `internal/hooks/MobDeath_BountyClaim_test.go`

- [ ] **Step 1: Read existing patterns**

Open `internal/hooks/MobDeath_FactionRep.go` to see how the existing MobDeath listener is structured. The new bounty listener mirrors that shape.

- [ ] **Step 2: Implement the listener**

```go
// internal/hooks/MobDeath_BountyClaim.go
package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// MobDeathBountyClaim auto-claims open mob-target bounties when the
// mob dies. The killer is the highest entry in evt.PlayerDamage
// (companion damage already rolls up to owners via combat.go's
// GetCharmedUserId path). Faction-issued bounties also bump rep.
func MobDeathBountyClaim(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}
	if len(evt.PlayerDamage) == 0 {
		return events.Continue
	}

	spec := mobs.GetMobSpec(mobs.MobId(evt.MobId))
	if spec == nil {
		return events.Continue
	}
	target := knowledge.MobSubject(int(spec.MobId))

	open := bounties.OpenForTarget(target)
	if len(open) == 0 {
		return events.Continue
	}

	// Highest-damager wins.
	killerUserId, topDamage := 0, 0
	for userId, dmg := range evt.PlayerDamage {
		if dmg > topDamage {
			killerUserId, topDamage = userId, dmg
		}
	}
	if killerUserId == 0 {
		return events.Continue
	}
	killer := knowledge.PlayerSubject(killerUserId)
	user := users.GetByUserId(killerUserId)
	if user == nil {
		return events.Continue
	}

	for _, b := range open {
		claimed, ok := bounties.TryClaim(b.Id, killer)
		if !ok {
			continue
		}
		// Pay gold.
		user.Character.Gold += claimed.GoldReward

		// Faction rep bump (only when issuer is a faction).
		if claimed.Issuer.Type == bounties.IssuerFaction {
			factions.BumpRep(claimed.Issuer.Id, killerUserId, claimed.RepReward)
		}

		user.SendText(fmt.Sprintf(
			"You collect a bounty: %dg.\r\n",
			claimed.GoldReward,
		))
	}

	return events.Continue
}
```

- [ ] **Step 3: Register the listener**

In `internal/hooks/hooks.go`, locate `RegisterListeners()` and add (alongside the existing MobDeath listeners):

```go
events.RegisterListener(events.MobDeath{}, MobDeathBountyClaim)
```

- [ ] **Step 4: Write a hook integration test**

```go
// internal/hooks/MobDeath_BountyClaim_test.go
package hooks

import (
	"testing"
)

func TestMobDeathBountyClaim_Stub(t *testing.T) {
	t.Skip("integration test stub — full coverage in smoke test goal file")
}
```

(This is a deliberate skip because constructing a full mob-fixture + faction-fixture + user-fixture in a unit test is heavy. The smoke test exercises the path end-to-end.)

- [ ] **Step 5: Verify build + tests**

Run: `go build ./... && go test ./internal/hooks/... ./internal/bounties/...`
Expected: clean / PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/MobDeath_BountyClaim.go internal/hooks/MobDeath_BountyClaim_test.go internal/hooks/hooks.go
git commit -m "feat(bounties): auto-claim hook on MobDeath (T10)"
```

---

## Task 11: Quest engine `declare_bounty` action

**Files:**
- Modify: `internal/questengine/types.go`
- Modify: `internal/questengine/actions.go`
- Modify: `internal/questengine/loader.go` (if action allowlist needs an entry)

- [ ] **Step 1: Add the action def to types.go**

Open `internal/questengine/types.go`. Locate the `Action` struct (it has fields like `BumpRep`, `GiveMutation`, etc.) and add:

```go
DeclareBounty *DeclareBountyDef `yaml:"declare_bounty,omitempty"`
```

After the `BumpRepDef` definition, add:

```go
// DeclareBountyDef parameters for the declare_bounty action.
// Either TargetPlayer (auto-fill with quest holder) OR explicit
// Target.Type+Target.Id must be set. Issuer mirrors the bounties
// package's tagged form; type=quest with id="<self>" auto-fills
// with the current quest id.
type DeclareBountyDef struct {
	Issuer struct {
		Type string `yaml:"type"` // "faction" | "quest" | "npc"
		Id   string `yaml:"id"`
	} `yaml:"issuer"`
	TargetPlayer bool `yaml:"target_player,omitempty"`
	Target       *struct {
		Type string `yaml:"type"` // "player" | "mob"
		Id   int    `yaml:"id"`
	} `yaml:"target,omitempty"`
	Condition    string `yaml:"condition"`           // "kill"
	ExpiryRounds uint64 `yaml:"expiry_rounds,omitempty"`
	GoldOverride int    `yaml:"gold_override,omitempty"`
	RepOverride  int    `yaml:"rep_override,omitempty"`
	Reason       string `yaml:"reason,omitempty"`
}
```

- [ ] **Step 2: Implement dispatch in actions.go**

Open `internal/questengine/actions.go` and locate the dispatch chain (the `if a.BumpRep != nil { ... }` block at line ~115). Add a case below it:

```go
import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
)

if a.DeclareBounty != nil {
	def := a.DeclareBounty

	// Resolve issuer.
	var issuer bounties.Issuer
	switch def.Issuer.Type {
	case "faction":
		issuer = bounties.FactionIssuer(def.Issuer.Id)
	case "quest":
		// Special form: id="<self>" -> use current questid.
		qid := def.Issuer.Id
		if qid == "<self>" {
			qid = strconv.Itoa(ctx.GetQuestId())
		}
		intId, err := strconv.Atoi(qid)
		if err != nil {
			return fmt.Errorf("declare_bounty: bad quest issuer id %q: %w", qid, err)
		}
		issuer = bounties.QuestIssuer(intId)
	case "npc":
		intId, err := strconv.Atoi(def.Issuer.Id)
		if err != nil {
			return fmt.Errorf("declare_bounty: bad npc issuer id %q: %w", def.Issuer.Id, err)
		}
		issuer = bounties.NPCIssuer(intId)
	default:
		return fmt.Errorf("declare_bounty: unknown issuer type %q", def.Issuer.Type)
	}

	// Resolve target.
	var target knowledge.Subject
	switch {
	case def.TargetPlayer:
		target = knowledge.PlayerSubject(ctx.GetUserId())
	case def.Target != nil:
		switch def.Target.Type {
		case "player":
			target = knowledge.PlayerSubject(def.Target.Id)
		case "mob":
			target = knowledge.MobSubject(def.Target.Id)
		default:
			return fmt.Errorf("declare_bounty: unknown target type %q", def.Target.Type)
		}
	default:
		return fmt.Errorf("declare_bounty: must set target_player or target")
	}

	// Compute expiry round.
	expiryRound := uint64(0)
	if def.ExpiryRounds > 0 {
		expiryRound = ctx.GetCurrentRound() + def.ExpiryRounds
	}

	condition := bounties.Condition(def.Condition)
	if condition == "" {
		condition = bounties.ConditionKill
	}

	id, err := bounties.Declare(issuer, target, condition, expiryRound,
		bounties.DeclareOpts{
			GoldOverride:   def.GoldOverride,
			RepOverride:    def.RepOverride,
			DeclaredReason: def.Reason,
		})
	if err != nil {
		return fmt.Errorf("declare_bounty: %w", err)
	}
	LogVerboseF(ctx.GetUserId(), "declare_bounty id=%d issuer=%s:%s target=%s:%d",
		id, issuer.Type, issuer.Id, target.Type, target.Id)
	return nil
}
```

NOTE: `ctx.GetCurrentRound()` and `ctx.GetQuestId()` may not exist with those exact names. Look at how `ctx` is used elsewhere in the same file (e.g., `ctx.GetUserId()`, `ctx.BumpRep(...)`). If `GetCurrentRound` doesn't exist, fall back to `util.GetRoundCount()`. If `GetQuestId` doesn't exist, the quest-self-reference can be deferred — surface the constraint as an error in that case.

- [ ] **Step 3: Update loader allowlist if needed**

`internal/questengine/loader.go:42` has an action-name allowlist. If `declare_bounty` needs to be added there, add it. (Read the current allowlist and check.)

- [ ] **Step 4: Write a small unit test**

```go
// internal/questengine/actions_test.go (or extend if exists)
func TestDeclareBountyAction_FactionIssuerPlayerTarget(t *testing.T) {
	t.Skip("integration test stub — full coverage in smoke test")
}
```

(The questengine package may already have action-execution tests; if so, extend the existing pattern. Skipping is acceptable for v1; smoke covers the path.)

- [ ] **Step 5: Verify build + existing tests**

Run: `go build ./... && go test ./internal/questengine/... ./internal/bounties/...`
Expected: clean / PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/questengine/
git commit -m "feat(bounties): quest engine declare_bounty action (T11)"
```

---

## Task 12: Admin command + helpfile

**Files:**
- Modify: `internal/bounties/bounties.go` (add `AllRows()` helper)
- Modify: `internal/bounties/bounties_test.go` (test `AllRows`)
- Create: `internal/usercommands/admin.bounty.go`
- Modify: `internal/usercommands/usercommands.go` (register the admin command)
- Create: `_datafiles/world/dogmud/templates/admincommands/help/command.bounty.template`

- [ ] **Step 0a: Add `AllRows()` to the bounties package**

Append to `internal/bounties/bounties.go`:

```go
// AllRows returns a snapshot of every row in the registry,
// regardless of status. Used by the admin --all listing.
func AllRows() []*Bounty {
	r := loadOrLazyInit()
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Bounty, len(r.Bounties))
	copy(out, r.Bounties)
	return out
}
```

Append to `internal/bounties/bounties_test.go`:

```go
func TestAllRows_IncludesNonOpen(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil; statpoolForTest = nil }()
	roundForTest = func() uint64 { return 100 }
	statpoolForTest = func(_ knowledge.Subject) int { return 600 }

	idA, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.MobSubject(101), ConditionKill, 1000, DeclareOpts{})
	idB, _ := Declare(FactionIssuer("thornwall_guards"),
		knowledge.MobSubject(102), ConditionKill, 1000, DeclareOpts{})
	Withdraw(idB)

	rows := AllRows()
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
	_ = idA
}
```

Run: `go test ./internal/bounties/...`
Expected: PASS.

- [ ] **Step 1: Read the chunk-1.4 reference**

Open `internal/usercommands/admin.knowledge.go` for the established admin-command pattern (subcommand dispatcher, help proxy, table rendering). Mirror that structure.

- [ ] **Step 2: Implement the admin command**

Create `internal/usercommands/admin.bounty.go`:

```go
package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
 * Role Permissions:
 * bounty         (Admin)
 */

func Bounty(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		bountyAdminUsage(user)
		return true, nil
	}
	switch strings.ToLower(args[0]) {
	case "list":
		return bountyAdminList(args[1:], user)
	case "show":
		return bountyAdminShow(args[1:], user)
	case "declare":
		return bountyAdminDeclare(args[1:], user)
	case "withdraw":
		return bountyAdminWithdraw(args[1:], user)
	case "prune-expired":
		return bountyAdminPrune(user)
	default:
		bountyAdminUsage(user)
		return true, nil
	}
}

func bountyAdminUsage(user *users.UserRecord) {
	if out, err := templates.Process("admincommands/help/command.bounty", nil, user.UserId); err == nil && strings.TrimSpace(out) != "" {
		user.SendText(out)
		return
	}
	user.SendText(
		"Usage:\r\n" +
			"  bounty list [--all]\r\n" +
			"  bounty show <id>\r\n" +
			"  bounty declare <issuer-spec> <target-spec> [--gold N] [--rep N] [--expiry-rounds N] [--reason \"...\"]\r\n" +
			"  bounty withdraw <id>\r\n" +
			"  bounty prune-expired\r\n" +
			"\r\n" +
			"Issuer/target spec: <type>:<id>  e.g.  faction:thornwall_guards  player:17  mob:101  quest:14\r\n",
	)
}

func bountyAdminList(args []string, user *users.UserRecord) (bool, error) {
	includeNonOpen := false
	for _, a := range args {
		if a == "--all" {
			includeNonOpen = true
		}
	}
	var rows []*bounties.Bounty
	if includeNonOpen {
		rows = bountyAllRows()
	} else {
		rows = bounties.AllOpen()
	}
	if len(rows) == 0 {
		user.SendText("No bounties.\r\n")
		return true, nil
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].GoldReward > rows[j].GoldReward })

	var b strings.Builder
	fmt.Fprintf(&b, "bounty list (%d):\r\n\r\n", len(rows))
	fmt.Fprintf(&b, "  %4s  %-10s  %-22s  %-15s  %5s  %4s  %s\r\n",
		"ID", "Status", "Issuer", "Target", "Gold", "Rep", "Reason")
	fmt.Fprintf(&b, "  %4s  %-10s  %-22s  %-15s  %5s  %4s  %s\r\n",
		"----", "----------", "----------------------", "---------------", "-----", "----", "----------")
	for _, r := range rows {
		fmt.Fprintf(&b, "  %4d  %-10s  %-22s  %-15s  %5d  %4d  %s\r\n",
			r.Id, r.Status,
			truncate(fmt.Sprintf("%s:%s", r.Issuer.Type, r.Issuer.Id), 22),
			fmt.Sprintf("%s:%d", r.Target.Type, r.Target.Id),
			r.GoldReward, r.RepReward, r.DeclaredReason)
	}
	user.SendText(b.String())
	return true, nil
}

// bountyAllRows returns every row in the registry regardless of
// status. Implementation requires adding bounties.AllRows() — see
// the additional step below. v1 ships this helper; smoke testing
// of the --all flag depends on it.
func bountyAllRows() []*bounties.Bounty {
	return bounties.AllRows()
}

func bountyAdminShow(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		bountyAdminUsage(user)
		return true, nil
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad bounty id %q\r\n", args[0]))
		return true, nil
	}
	b := bounties.Get(id)
	if b == nil {
		user.SendText(fmt.Sprintf("No bounty with id %d\r\n", id))
		return true, nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Bounty #%d (%s)\r\n", b.Id, b.Status)
	fmt.Fprintf(&sb, "  Issuer:    %s:%s\r\n", b.Issuer.Type, b.Issuer.Id)
	fmt.Fprintf(&sb, "  Target:    %s:%d\r\n", b.Target.Type, b.Target.Id)
	fmt.Fprintf(&sb, "  Reward:    %dg + %d rep\r\n", b.GoldReward, b.RepReward)
	fmt.Fprintf(&sb, "  Condition: %s\r\n", b.Condition)
	fmt.Fprintf(&sb, "  Declared:  round %d (reason: %q)\r\n", b.DeclaredRound, b.DeclaredReason)
	if b.ExpiryRound > 0 {
		fmt.Fprintf(&sb, "  Expires:   round %d\r\n", b.ExpiryRound)
	} else {
		fmt.Fprintf(&sb, "  Expires:   never\r\n")
	}
	if b.Status == bounties.StatusClaimed {
		fmt.Fprintf(&sb, "  Claimed:   round %d by %s:%d\r\n",
			b.ClaimedRound, b.ClaimedBy.Type, b.ClaimedBy.Id)
	}
	user.SendText(sb.String())
	return true, nil
}

func bountyAdminDeclare(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 2 {
		bountyAdminUsage(user)
		return true, nil
	}
	issuer, ok := parseIssuerSpec(args[0])
	if !ok {
		user.SendText(fmt.Sprintf("Bad issuer spec %q (use type:id)\r\n", args[0]))
		return true, nil
	}
	target, ok := parseTargetSpec(args[1])
	if !ok {
		user.SendText(fmt.Sprintf("Bad target spec %q (use type:id)\r\n", args[1]))
		return true, nil
	}
	opts := bounties.DeclareOpts{}
	expiryRounds := uint64(0)
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--gold":
			i++
			if i < len(args) {
				if v, err := strconv.Atoi(args[i]); err == nil {
					opts.GoldOverride = v
				}
			}
		case "--rep":
			i++
			if i < len(args) {
				if v, err := strconv.Atoi(args[i]); err == nil {
					opts.RepOverride = v
				}
			}
		case "--expiry-rounds":
			i++
			if i < len(args) {
				if v, err := strconv.ParseUint(args[i], 10, 64); err == nil {
					expiryRounds = v
				}
			}
		case "--reason":
			// Consume the rest as the reason (handles spaces).
			opts.DeclaredReason = strings.Join(args[i+1:], " ")
			i = len(args)
		}
	}
	expiryRound := uint64(0)
	if expiryRounds > 0 {
		expiryRound = uint64(util.GetRoundCount()) + expiryRounds
	}
	id, err := bounties.Declare(issuer, target, bounties.ConditionKill, expiryRound, opts)
	if err != nil {
		user.SendText(fmt.Sprintf("Declare failed: %v\r\n", err))
		return true, nil
	}
	user.SendText(fmt.Sprintf("Declared bounty #%d.\r\n", id))
	return true, nil
}

func bountyAdminWithdraw(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		bountyAdminUsage(user)
		return true, nil
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad bounty id %q\r\n", args[0]))
		return true, nil
	}
	bounties.Withdraw(id)
	user.SendText(fmt.Sprintf("Withdrew bounty #%d.\r\n", id))
	return true, nil
}

func bountyAdminPrune(user *users.UserRecord) (bool, error) {
	count := bounties.PruneExpired()
	user.SendText(fmt.Sprintf("Pruned %d expired bounty rows.\r\n", count))
	return true, nil
}

// parseIssuerSpec parses "<type>:<id>" → Issuer.
func parseIssuerSpec(s string) (bounties.Issuer, bool) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return bounties.Issuer{}, false
	}
	switch strings.ToLower(parts[0]) {
	case "faction":
		return bounties.FactionIssuer(parts[1]), true
	case "quest":
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			return bounties.Issuer{}, false
		}
		return bounties.QuestIssuer(id), true
	case "npc":
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			return bounties.Issuer{}, false
		}
		return bounties.NPCIssuer(id), true
	}
	return bounties.Issuer{}, false
}

// parseTargetSpec parses "<type>:<id>" → knowledge.Subject.
func parseTargetSpec(s string) (knowledge.Subject, bool) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return knowledge.Subject{}, false
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil {
		return knowledge.Subject{}, false
	}
	switch strings.ToLower(parts[0]) {
	case "player":
		return knowledge.PlayerSubject(id), true
	case "mob":
		return knowledge.MobSubject(id), true
	}
	return knowledge.Subject{}, false
}

// truncate is the same shape as the chunk-1.4 admin command's helper.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
```

Add the missing `util` import for the GetRoundCount call.

If `bounties.AllOpen` covers the v1 admin needs, the `--all` flag can punt for now (or add a small `bounties.AllRows()` API if desired). Pick the simpler path during implementation.

- [ ] **Step 3: Register the admin command**

In `internal/usercommands/usercommands.go`, find where `Knowledge` was registered (chunk 1.4 added it). Mirror the pattern:

```go
"bounty": {Bounty, true, true, true},  // adjust the bool flags to match the role-permission pattern used elsewhere
```

Read the surrounding entries to verify the right shape (especially the role / admin flag).

- [ ] **Step 4: Write the help template**

Create `_datafiles/world/dogmud/templates/admincommands/help/command.bounty.template`:

```
<ansi fg="white-bold">bounty</ansi> — declare, view, withdraw, and resolve bounties

  <ansi fg="cyan">bounty list</ansi> [<ansi fg="yellow">--all</ansi>]
    List open bounties (default) or every row including claimed/expired/withdrawn.

  <ansi fg="cyan">bounty show</ansi> <ansi fg="yellow"><id></ansi>
    Show full detail of one bounty.

  <ansi fg="cyan">bounty declare</ansi> <ansi fg="yellow"><issuer-spec> <target-spec> [opts]</ansi>
    Manually declare a bounty.
      Specs: <type>:<id>
        faction:thornwall_guards  quest:14  npc:357
        player:17                 mob:101
      Options:
        --gold N                  override default gold reward
        --rep N                   override default rep reward
        --expiry-rounds N         expire after N rounds (0 = never)
        --reason "..."            flavor text shown on board

  <ansi fg="cyan">bounty withdraw</ansi> <ansi fg="yellow"><id></ansi>
    Cancel an open bounty.

  <ansi fg="cyan">bounty prune-expired</ansi>
    Sweep open bounties past their expiry round.
```

- [ ] **Step 5: Verify build + tests**

Run: `go build ./... && go test ./internal/usercommands/... ./internal/bounties/...`
Expected: clean / PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/admin.bounty.go internal/usercommands/usercommands.go _datafiles/world/dogmud/templates/admincommands/help/command.bounty.template
git commit -m "feat(bounties): admin bounty list/show/declare/withdraw/prune (T12)"
```

---

## Task 13: Player command + helpfile

**Files:**
- Create: `internal/usercommands/bounty.go`
- Modify: `internal/usercommands/usercommands.go` (register the user command)
- Create: `_datafiles/world/dogmud/templates/usercommands/help/command.bounty.template`

- [ ] **Step 1: Find an existing user-command helpfile path**

Look at where other user commands' help templates live (e.g., search `_datafiles/world/dogmud/templates/` for a recent user-command helpfile). Mirror that exact path. The path I've assumed is `templates/usercommands/help/`; verify by listing.

- [ ] **Step 2: Implement the user command**

Create `internal/usercommands/bounty.go`:

```go
package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func BountyUser(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 || strings.ToLower(args[0]) == "list" {
		return bountyUserList(args, user)
	}
	switch strings.ToLower(args[0]) {
	case "show":
		return bountyUserShow(args[1:], user)
	default:
		bountyUserUsage(user)
		return true, nil
	}
}

func bountyUserUsage(user *users.UserRecord) {
	if out, err := templates.Process("usercommands/help/command.bounty", nil, user.UserId); err == nil && strings.TrimSpace(out) != "" {
		user.SendText(out)
		return
	}
	user.SendText(
		"Usage:\r\n" +
			"  bounty list [filter]      filter = mob | player | <faction-slug>\r\n" +
			"  bounty show <id>\r\n",
	)
}

func bountyUserList(args []string, user *users.UserRecord) (bool, error) {
	rows := bounties.AllOpen()
	// Filter
	filter := ""
	if len(args) >= 2 {
		filter = strings.ToLower(args[1])
	}
	if filter != "" {
		filtered := make([]*bounties.Bounty, 0, len(rows))
		for _, r := range rows {
			switch filter {
			case "mob", "player":
				if string(r.Target.Type) == filter {
					filtered = append(filtered, r)
				}
			default:
				// Treat as a faction slug.
				if r.Issuer.Type == bounties.IssuerFaction && r.Issuer.Id == filter {
					filtered = append(filtered, r)
				}
			}
		}
		rows = filtered
	}
	if len(rows) == 0 {
		user.SendText("No open bounties match your filter.\r\n")
		return true, nil
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].GoldReward > rows[j].GoldReward })

	var b strings.Builder
	fmt.Fprintf(&b, "Open bounties (%d):\r\n\r\n", len(rows))
	fmt.Fprintf(&b, "  %4s  %-22s  %-15s  %5s  %s\r\n", "ID", "Issuer", "Target", "Gold", "Reason")
	fmt.Fprintf(&b, "  %4s  %-22s  %-15s  %5s  %s\r\n", "----", "----------------------", "---------------", "-----", "----------")
	for _, r := range rows {
		fmt.Fprintf(&b, "  %4d  %-22s  %-15s  %5d  %s\r\n",
			r.Id,
			truncate(fmt.Sprintf("%s:%s", r.Issuer.Type, r.Issuer.Id), 22),
			fmt.Sprintf("%s:%d", r.Target.Type, r.Target.Id),
			r.GoldReward, r.DeclaredReason)
	}
	user.SendText(b.String())
	return true, nil
}

func bountyUserShow(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		bountyUserUsage(user)
		return true, nil
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad bounty id %q\r\n", args[0]))
		return true, nil
	}
	b := bounties.Get(id)
	if b == nil || b.Status != bounties.StatusOpen {
		user.SendText("No open bounty with that id.\r\n")
		return true, nil
	}
	user.SendText(fmt.Sprintf(
		"Bounty #%d\r\n  Issuer: %s:%s\r\n  Target: %s:%d\r\n  Reward: %dg + %d rep\r\n  Reason: %s\r\n",
		b.Id, b.Issuer.Type, b.Issuer.Id, b.Target.Type, b.Target.Id,
		b.GoldReward, b.RepReward, b.DeclaredReason,
	))
	return true, nil
}
```

- [ ] **Step 3: Register the user command**

In `internal/usercommands/usercommands.go`, add a row similar to the admin one but with the right flags for a public-user command. Look at how non-admin user commands are registered nearby to verify the flag pattern. Use a different exported name (`BountyUser`) so it doesn't collide with the admin `Bounty`.

If the dispatcher doesn't allow two functions for the same command name, route via the admin command's handler with role-gated branching, OR use a single `bounty` entry that checks role internally. The chunk-1.4 reference uses one handler per command — adapt as needed.

- [ ] **Step 4: Write the help template**

Create `_datafiles/world/dogmud/templates/usercommands/help/command.bounty.template`:

```
<ansi fg="white-bold">bounty</ansi> — open bounty board

  <ansi fg="cyan">bounty list</ansi> [<ansi fg="yellow">filter</ansi>]
    List open bounties. Filter is one of:
      <ansi fg="yellow">mob</ansi>     — only bounties on creatures
      <ansi fg="yellow">player</ansi>  — only bounties on players
      <ansi fg="yellow"><faction-slug></ansi>  — only bounties from the named faction
                  (e.g. <ansi fg="yellow">thornwall_guards</ansi>)

  <ansi fg="cyan">bounty show</ansi> <ansi fg="yellow"><id></ansi>
    Show full detail of one open bounty.

Visit a notice board (Thornwall Guard Barracks, Stillwater Constabulary)
for the in-fiction posting.
```

- [ ] **Step 5: Verify build + tests**

Run: `go build ./... && go test ./internal/usercommands/... ./internal/bounties/...`
Expected: clean / PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/bounty.go internal/usercommands/usercommands.go _datafiles/world/dogmud/templates/usercommands/help/command.bounty.template
git commit -m "feat(bounties): user bounty list/show command (T13)"
```

---

## Task 14: Bounty board nouns (Thornwall + Stillwater)

**Files:**
- Modify: `_datafiles/world/dogmud/rooms/thornwall_city/473.yaml`
- Modify: `_datafiles/world/dogmud/rooms/stillwater/4110.yaml`

- [ ] **Step 1: Read the current YAMLs**

Open both files. Locate the `nouns:` section (or check if it exists). For Thornwall 473 (Guard Barracks) and Stillwater 4110 (Constabulary), the noun additions must not collide with existing nouns.

- [ ] **Step 2: Add the bounty-board noun**

In each file, add (or extend the existing `nouns:` map):

```yaml
nouns:
  bounty board: |
    A weather-worn corkboard stands beside the wall, papered with
    wanted notices and contract slips, recent ones still clean,
    older ones curling at the edges. Anyone with the will to read
    them can run "bounty list" anywhere in town to see what's open.
```

(Adjust the room descriptions to mention the bounty board if they don't already — e.g., add to the room's `description:` field a phrase like "A bounty board hangs by the door, papered with notices." This makes the noun discoverable.)

- [ ] **Step 3: Verify with the room loader**

Run: `go build ./...`
Expected: clean.

(Boot the server briefly if you want to confirm the rooms load past the data-file pass; per the project SOP, YAML mistakes panic at startup, not at compile.)

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/rooms/thornwall_city/473.yaml _datafiles/world/dogmud/rooms/stillwater/4110.yaml
git commit -m "feat(bounties): bounty-board nouns in Thornwall + Stillwater (T14)"
```

NOTE: Per the project SOP, check `rooms.instances/` for cached instance saves of these two rooms. If present, delete them so the engine picks up the noun changes on next boot.

---

## Task 15: Roadmap update

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Mark chunk 1.5 as Done in the progress tracker**

Locate the `| 1.5 | Substrate | Bounty state | S | 1.2 | Not started |` row and change `Not started` to `Done`. Update the roll-up: `4 / 40 done` → `5 / 40 done`.

- [ ] **Step 2: Update the chunk's mini-brief**

Locate the `### 1.5 Bounty state` section. Change `**Status:** Not started` to `**Status:** Done (2026-05-09)`. Append a `Shipped:` paragraph mirroring the format used by 1.1 / 1.2 / 1.3 / 1.4.

Suggested template:

```markdown
- **Shipped:** `internal/bounties/` package storing per-bounty registry at `_datafiles/world/dogmud/bounties.yaml` (gitignored). Polymorphic target via `knowledge.Subject` (player or mob template); three issuer types (faction, quest, npc). Reward auto-computes from target statpool — `gold = floor(statpool × BountyGoldDefaultMultiplier)` (default 0.5, floor 50) and `rep = max(1, floor(statpool / 100))`, both stored on the row at declaration with declarer override available via `DeclareOpts`. Auto-claim hook `MobDeath_BountyClaim` fires on mob death — highest-damager wins (companion damage already rolls up via `combat.go`'s charmed-userId path), gold transferred to character, faction rep bumped when issuer is a faction. Quest engine `declare_bounty` action wires the substrate into quest content. Admin command `bounty list/show/declare/withdraw/prune-expired` + helpfile. Player command `bounty list/show` + helpfile. Two physical bounty boards as flavor nouns (Thornwall Guard Barracks 473, Stillwater Constabulary 4110) — discovery via `look bounty board`; data flow via the universal command. Withdraw + expiry semantics; non-open rows preserved for audit. Spec at `docs/superpowers/specs/2026-05-09-mob-aliveness-1.5-bounty-state-design.md`, plan at `docs/superpowers/plans/2026-05-09-mob-aliveness-1.5-bounty-state.md`.
```

- [ ] **Step 3: Commit**

```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(roadmap): mark chunk 1.5 (bounty state) as Done"
```

---

## Final review

After all tasks complete, dispatch the `superpowers:code-reviewer` agent for a holistic pass before smoke testing.

Suggested smoke goal file: `tools/testing/goals/bounty-thornwall-smoke.yaml` covering the scenarios from the spec's Testing section (admin declare → player bounty list → player kill target → reward paid + bounty closed → bounty list shows it gone → admin --all confirms claimed status; plus a board check at Thornwall 473 and Stillwater 4110).

Branch state after completion: `feature/mob-aliveness-1.3-crimes` carries chunks 1.1, 1.2, 1.3, 1.4, AND 1.5. When confidence is high, merge to development with `--no-ff`.
