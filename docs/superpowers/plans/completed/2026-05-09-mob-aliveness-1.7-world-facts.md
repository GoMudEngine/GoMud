# Mob Aliveness 1.7 — World-Model Facts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `internal/facts/` — a standing-fact registry plus a unified per-NPC awareness store (replaces the in-memory `recentGossipEvents` TempData with a persistent on-disk file that holds both heard-event ids AND known-fact ids). Migrate `buildGossipLine` to consume the new store, extend it to also gossip about known facts, and ship the small additive change to worldevents (stable event IDs).

**Architecture:** Mob YAMLs gain an optional `knows_facts:` field. After `mobs.LoadDataFiles()` completes, a new `facts.LoadFromMobs()` call seeds per-NPC awareness records from those declarations. A new `facts.yaml` registry holds authored standing facts. A new `MobRoomChange_FactsAutoWithdraw` hook flips facts to status=withdrawn when their bound mob template's instance enters a room. Existing emitters of worldevents and existing gossip mob behavior are preserved.

**Tech Stack:** Go 1.21+, YAML via `gopkg.in/yaml.v3`, existing `internal/events` event bus, existing `internal/configs/Balance` config, existing `internal/worldevents` ring buffer.

**Spec:** `docs/superpowers/specs/2026-05-09-mob-aliveness-1.7-world-facts-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP for the aliveness effort.

---

## File structure

| File | Responsibility |
|------|----------------|
| `internal/facts/types.go` | `Status`, `Source`, `Fact`, `FactKnowledge`, `Awareness`, `Registry` types |
| `internal/facts/persistence.go` | Base dirs, filepath helpers, YAML load/save for registry + per-NPC awareness, lazy-load + double-check-lock |
| `internal/facts/facts.go` | Public API: registry lifecycle (Declare/Withdraw/Expire/PruneExpired/WithdrawAllBoundTo) + reads + awareness ops |
| `internal/facts/test_main_test.go` | Test harness with temp dir |
| `internal/facts/types_test.go` | Type-level tests |
| `internal/facts/persistence_test.go` | Round-trip + concurrent-load tests |
| `internal/facts/facts_test.go` | API tests |
| `internal/facts/context.md` | Package documentation |
| `internal/worldevents/worldevents.go` | Add `Id uint64` field on `WorldEvent`, atomic counter |
| `internal/configs/config.balance.go`, `config.balance.misc.go` | Add `FactsHeardEventsMax` knob (default 32) |
| `internal/mobs/mobs.go` | Add `KnowsFacts []string` field on `Mob`; call `facts.LoadFromMobs(...)` after LoadDataFiles |
| `internal/hooks/MobRoomChange_FactsAutoWithdraw.go` | RoomChange listener for auto-withdraw |
| `internal/hooks/hooks.go` | Register the new listener |
| `internal/hooks/MobIdle_HandleIdleMobs.go` | Migrate `buildGossipLine` away from `recentGossipEvents` TempData; extend candidate pool with known facts |
| `internal/usercommands/admin.fact.go` | Admin `fact` command |
| `internal/usercommands/usercommands.go` | Register admin command |
| `_datafiles/world/dogmud/templates/admincommands/help/command.fact.template` | Helpfile |
| `_datafiles/world/dogmud/facts.yaml` | Authored fact registry (committed; ships with chunk 1.7 as an empty seed file) |
| `MOB_ALIVENESS_ROADMAP.md` | Mark 1.7 Done, roll-up to 7/40 |

---

## Task 1: Worldevents `Id` field

**Files:**
- Modify: `internal/worldevents/worldevents.go`
- Modify: `internal/worldevents/*_test.go` (if present, otherwise add a small test)

This is a tiny additive change that everything else depends on. Done first so the new `Id` field is available when the facts package goes to reference events.

- [ ] **Step 1: Read the current state**

```bash
ls internal/worldevents/
wc -l internal/worldevents/worldevents.go
```

Note: `WorldEvent` struct at line ~40, `EmitWorldEvent` at line ~90.

- [ ] **Step 2: Write the failing test**

Create or extend `internal/worldevents/worldevents_test.go`:

```go
// internal/worldevents/worldevents_test.go
package worldevents

import "testing"

func TestEmitAssignsMonotonicIds(t *testing.T) {
	InitWorldEvents()
	EmitWorldEvent(WorldEvent{Description: "first"})
	EmitWorldEvent(WorldEvent{Description: "second"})

	evts := GetRecentWorldEvents(2, nil)
	if len(evts) != 2 {
		t.Fatalf("expected 2 events, got %d", len(evts))
	}
	if evts[0].Id == 0 {
		t.Errorf("first event got id 0; should start >= 1")
	}
	if evts[1].Id != evts[0].Id+1 && evts[0].Id != evts[1].Id+1 {
		t.Errorf("ids not monotonic: %d, %d", evts[0].Id, evts[1].Id)
	}
}
```

(Note: `GetRecentWorldEvents` may return newest-first or oldest-first; the assertion handles either ordering by checking that the two ids are adjacent.)

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/worldevents/...`
Expected: FAIL — `Id` field doesn't exist on `WorldEvent`.

- [ ] **Step 4: Add the field + counter**

In `internal/worldevents/worldevents.go`:

```go
import (
	"sync/atomic"
)

// Add to the WorldEvent struct (place after Round):
type WorldEvent struct {
	// ... existing fields ...
	Id uint64
}

// Add a package-level counter near the other vars:
var nextEventId atomic.Uint64

// Modify EmitWorldEvent — set evt.Id at the top:
func EmitWorldEvent(evt WorldEvent) {
	evt.Id = nextEventId.Add(1)
	// ... existing implementation ...
}
```

Identify the exact `EmitWorldEvent` body, preserve all existing logic, and only add the `evt.Id = nextEventId.Add(1)` assignment as the first line of the function (before any mutex acquisition or ring buffer mutation).

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/worldevents/...`
Expected: PASS.

Run a broader smoke: `go build ./... && go test ./...`
Expected: clean / PASS — existing emitters (combat, suicide, mutation hooks) shouldn't have any behavior change.

- [ ] **Step 6: Commit**

```bash
git add internal/worldevents/worldevents.go internal/worldevents/worldevents_test.go
git commit -m "feat(worldevents): add Id field on WorldEvent (chunk 1.7 T1)"
```

---

## Task 2: Package skeleton + types

**Files:**
- Create: `internal/facts/types.go`
- Create: `internal/facts/types_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/facts/types_test.go
package facts

import (
	"testing"
)

func TestStatusConstants(t *testing.T) {
	if string(StatusActive) != "active" {
		t.Errorf("StatusActive: got %s", StatusActive)
	}
	if string(StatusWithdrawn) != "withdrawn" {
		t.Errorf("StatusWithdrawn: got %s", StatusWithdrawn)
	}
	if string(StatusExpired) != "expired" {
		t.Errorf("StatusExpired: got %s", StatusExpired)
	}
}

func TestSourceConstants(t *testing.T) {
	if string(SourceWitnessed) != "witnessed" {
		t.Errorf("SourceWitnessed: %s", SourceWitnessed)
	}
	if string(SourceTold) != "told" {
		t.Errorf("SourceTold: %s", SourceTold)
	}
	if string(SourceDeduced) != "deduced" {
		t.Errorf("SourceDeduced: %s", SourceDeduced)
	}
	if string(SourceUnknown) != "unknown" {
		t.Errorf("SourceUnknown: %s", SourceUnknown)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/facts/...`
Expected: FAIL (package doesn't exist).

- [ ] **Step 3: Implement types.go**

```go
// internal/facts/types.go
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
	Status              Status                   `yaml:"status,omitempty"` // populated runtime; defaults to active
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/facts/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/facts/types.go internal/facts/types_test.go
git commit -m "feat(facts): types — Status, Source, Fact, Awareness, Registry (T2)"
```

---

## Task 3: TestMain harness + persistence skeleton

**Files:**
- Create: `internal/facts/test_main_test.go`
- Create: `internal/facts/persistence.go`

- [ ] **Step 1: Read the chunk-1.4/1.5 reference**

Open `internal/knowledge/test_main_test.go` and `internal/bounties/persistence.go`. Mirror the temp-dir TestMain pattern (uses `configs.AddOverlayOverrides` for FilePaths.DataFiles).

- [ ] **Step 2: Write test_main_test.go**

```go
// internal/facts/test_main_test.go
package facts

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "facts-test-*")
	if err != nil {
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles": tmp,
	})

	if err := os.MkdirAll(filepath.Join(tmp, "world", "dogmud", "facts.awareness"), 0o755); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func resetCaches() {
	registryMu.Lock()
	registry = nil
	registryMu.Unlock()

	awarenessCacheMu.Lock()
	awarenessCache = make(map[int]*Awareness)
	awarenessCacheMu.Unlock()
}
```

- [ ] **Step 3: Implement persistence.go skeleton**

```go
// internal/facts/persistence.go
package facts

import (
	"fmt"
	"path/filepath"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
	_ "gopkg.in/yaml.v3" // pre-imported for Task 4 YAML calls
)

var (
	registry      *Registry
	registryMu    sync.RWMutex
	registrySaveMu sync.Mutex // serializes registry disk writes

	awarenessCache    = make(map[int]*Awareness)
	awarenessCacheMu  sync.RWMutex
	awarenessSaveMu   sync.Mutex // serializes awareness disk writes
)

func registryFilePath() string {
	return filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "world", "dogmud", "facts.yaml")
}

func awarenessFilePath(mobId int, mobName string) string {
	return filepath.Join(string(configs.GetFilePathsConfig().DataFiles),
		"world", "dogmud", "facts.awareness",
		fmt.Sprintf("%d-%s.yaml", mobId, util.ConvertForFilename(mobName)))
}
```

- [ ] **Step 4: Verify build**

Run: `go test ./internal/facts/...`
Expected: PASS (T2 tests still pass).

- [ ] **Step 5: Commit**

```bash
git add internal/facts/test_main_test.go internal/facts/persistence.go
git commit -m "feat(facts): test harness + persistence skeleton (T3)"
```

---

## Task 4: Registry save/load round-trip

**Files:**
- Modify: `internal/facts/persistence.go`
- Create: `internal/facts/persistence_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/facts/persistence_test.go
package facts

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/worldevents"
)

func TestRegistry_SaveAndLoadRoundTrip(t *testing.T) {
	resetCaches()

	r := &Registry{
		Facts: []*Fact{
			{
				Id:           "king-dead",
				Description:  "The king is dead.",
				Significance: worldevents.Global,
				Tags:         []string{"politics", "death"},
				Status:       StatusActive,
			},
		},
	}

	if err := saveRegistry(r); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded := loadRegistryFromDisk()
	if loaded == nil || len(loaded.Facts) != 1 {
		t.Fatalf("expected 1 fact loaded, got %v", loaded)
	}
	if loaded.Facts[0].Id != "king-dead" {
		t.Errorf("fact id: got %q", loaded.Facts[0].Id)
	}
	if loaded.Facts[0].Significance != worldevents.Global {
		t.Errorf("significance mismatch: %v", loaded.Facts[0].Significance)
	}
}

func TestRegistry_LoadMissingFileReturnsNil(t *testing.T) {
	resetCaches()
	if loadRegistryFromDisk() != nil {
		t.Errorf("expected nil for missing file")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/facts/...`
Expected: FAIL (saveRegistry / loadRegistryFromDisk undefined).

- [ ] **Step 3: Implement save/load**

Append to `internal/facts/persistence.go`:

```go
import (
	"os"

	"gopkg.in/yaml.v3"
)

// saveRegistry serializes under cache RLock for snapshot consistency,
// writes via tmp-rename for atomicity. Mirrors chunk 1.4 review fix.
func saveRegistry(r *Registry) error {
	registrySaveMu.Lock()
	defer registrySaveMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(registryFilePath()), 0o755); err != nil {
		return fmt.Errorf("mkdir registry dir: %w", err)
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
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename: %w", err)
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

Activate the previously-blank yaml import.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/facts/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/facts/persistence.go internal/facts/persistence_test.go
git commit -m "feat(facts): registry round-trip + marshal-under-RLock (T4)"
```

---

## Task 5: Awareness save/load + lazy-load with double-check-lock

**Files:**
- Modify: `internal/facts/persistence.go`
- Modify: `internal/facts/persistence_test.go`

- [ ] **Step 1: Write failing tests**

Append to `internal/facts/persistence_test.go`:

```go
import (
	"sync"
)

func TestAwareness_SaveAndLoadRoundTrip(t *testing.T) {
	resetCaches()

	a := &Awareness{
		ObserverMobId:    114,
		ObserverName:     "old fen",
		HeardEvents:      []uint64{42, 56},
		KnownFacts:       []FactKnowledge{{FactId: "king-dead", Source: SourceWitnessed, LearnedRound: 100}},
		LastUpdatedRound: 100,
	}

	if err := saveAwareness(a); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded := loadAwarenessFromDisk(114, "old fen")
	if loaded == nil {
		t.Fatalf("expected non-nil load")
	}
	if loaded.ObserverMobId != 114 {
		t.Errorf("observer mob id mismatch: %d", loaded.ObserverMobId)
	}
	if len(loaded.HeardEvents) != 2 {
		t.Errorf("heard_events count: %d", len(loaded.HeardEvents))
	}
	if len(loaded.KnownFacts) != 1 || loaded.KnownFacts[0].FactId != "king-dead" {
		t.Errorf("known_facts: %v", loaded.KnownFacts)
	}
}

func TestAwareness_LoadOrLazyInitConcurrent(t *testing.T) {
	resetCaches()
	const N = 50
	var wg sync.WaitGroup
	var seen [N]*Awareness
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			seen[i] = loadOrLazyInitAwareness(123, "city beggar")
		}(i)
	}
	wg.Wait()

	first := seen[0]
	if first == nil {
		t.Fatalf("nil result")
	}
	for i := 1; i < N; i++ {
		if seen[i] != first {
			t.Errorf("goroutine %d got different pointer", i)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/facts/...`
Expected: FAIL (saveAwareness / loadAwarenessFromDisk / loadOrLazyInitAwareness undefined).

- [ ] **Step 3: Implement**

Append to `internal/facts/persistence.go`:

```go
func saveAwareness(a *Awareness) error {
	awarenessSaveMu.Lock()
	defer awarenessSaveMu.Unlock()

	if err := os.MkdirAll(filepath.Dir(awarenessFilePath(a.ObserverMobId, a.ObserverName)), 0o755); err != nil {
		return fmt.Errorf("mkdir awareness dir: %w", err)
	}

	awarenessCacheMu.RLock()
	out, err := yaml.Marshal(a)
	awarenessCacheMu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshal awareness: %w", err)
	}

	path := awarenessFilePath(a.ObserverMobId, a.ObserverName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

func loadAwarenessFromDisk(mobId int, mobName string) *Awareness {
	data, err := os.ReadFile(awarenessFilePath(mobId, mobName))
	if err != nil {
		return nil
	}
	a := &Awareness{}
	if err := yaml.Unmarshal(data, a); err != nil {
		return nil
	}
	return a
}

// loadOrLazyInitAwareness returns the cached *Awareness for the given
// observer mob template id. Mirrors the chunk-1.3/1.4 double-check-
// lock pattern.
func loadOrLazyInitAwareness(mobId int, mobName string) *Awareness {
	awarenessCacheMu.RLock()
	if a, ok := awarenessCache[mobId]; ok {
		awarenessCacheMu.RUnlock()
		return a
	}
	awarenessCacheMu.RUnlock()

	if a := loadAwarenessFromDisk(mobId, mobName); a != nil {
		awarenessCacheMu.Lock()
		if cached, ok := awarenessCache[mobId]; ok {
			awarenessCacheMu.Unlock()
			return cached
		}
		awarenessCache[mobId] = a
		awarenessCacheMu.Unlock()
		return a
	}

	a := &Awareness{ObserverMobId: mobId, ObserverName: mobName}
	awarenessCacheMu.Lock()
	if cached, ok := awarenessCache[mobId]; ok {
		awarenessCacheMu.Unlock()
		return cached
	}
	awarenessCache[mobId] = a
	awarenessCacheMu.Unlock()
	return a
}

// loadOrLazyInitRegistry mirrors the awareness pattern but for the
// single-instance registry.
func loadOrLazyInitRegistry() *Registry {
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

	r := &Registry{Facts: []*Fact{}}
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

Run: `go test ./internal/facts/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/facts/persistence.go internal/facts/persistence_test.go
git commit -m "feat(facts): awareness round-trip + double-check-lock for both stores (T5)"
```

---

## Task 6: Config knob `FactsHeardEventsMax`

**Files:**
- Modify: `internal/configs/config.balance.go`
- Modify: `internal/configs/config.balance.misc.go`

- [ ] **Step 1: Add the knob**

In `internal/configs/config.balance.go`, locate existing knobs added in chunks 1.4-1.6 (e.g. `KnowledgeObservationLogMax`, `BountyGoldDefaultMultiplier`). Add nearby:

```go
FactsHeardEventsMax int `yaml:"facts_heard_events_max"`
```

In `internal/configs/config.balance.misc.go` `validateMisc()`:

```go
if b.FactsHeardEventsMax == 0 {
	b.FactsHeardEventsMax = 32
}
```

- [ ] **Step 2: Verify build + existing config tests**

Run: `go build ./... && go test ./internal/configs/...`
Expected: clean / PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.misc.go
git commit -m "feat(facts): FactsHeardEventsMax config knob (T6)"
```

---

## Task 7: Public API — Declare + GetFact + lifecycle (Withdraw / Expire / PruneExpired / WithdrawAllBoundTo)

**Files:**
- Create: `internal/facts/facts.go`
- Create: `internal/facts/facts_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/facts/facts_test.go
package facts

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/worldevents"
)

func TestDeclare_HappyPath(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	err := Declare("king-dead", DeclareOpts{
		Description:  "The king is dead.",
		Significance: worldevents.Global,
		Tags:         []string{"politics", "death"},
	})
	if err != nil {
		t.Fatalf("Declare returned error: %v", err)
	}

	f := GetFact("king-dead")
	if f == nil {
		t.Fatalf("GetFact returned nil")
	}
	if f.Status != StatusActive {
		t.Errorf("status not active: %s", f.Status)
	}
	if f.DeclaredRound != 100 {
		t.Errorf("DeclaredRound mismatch: %d", f.DeclaredRound)
	}
}

func TestDeclare_CollisionRejected(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	if err := Declare("foo", DeclareOpts{Description: "first"}); err != nil {
		t.Fatalf("first Declare: %v", err)
	}
	if err := Declare("foo", DeclareOpts{Description: "second"}); err == nil {
		t.Errorf("collision should return error")
	}
}

func TestWithdraw(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	Declare("foo", DeclareOpts{Description: "t"})
	Withdraw("foo")
	if f := GetFact("foo"); f.Status != StatusWithdrawn {
		t.Errorf("expected withdrawn, got %s", f.Status)
	}
	// Idempotent.
	Withdraw("foo")
	if f := GetFact("foo"); f.Status != StatusWithdrawn {
		t.Errorf("idempotent expected, got %s", f.Status)
	}
}

func TestPruneExpired(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	Declare("a", DeclareOpts{Description: "a", ExpiryRound: 200})
	Declare("b", DeclareOpts{Description: "b"}) // never expires
	Declare("c", DeclareOpts{Description: "c", ExpiryRound: 5000})

	roundForTest = func() uint64 { return 300 }
	count := PruneExpired()
	if count != 1 {
		t.Errorf("expected 1 pruned, got %d", count)
	}
	if GetFact("a").Status != StatusExpired {
		t.Errorf("a should be expired")
	}
	if GetFact("b").Status != StatusActive {
		t.Errorf("b should still be active (never expires)")
	}
	if GetFact("c").Status != StatusActive {
		t.Errorf("c should still be active (future expiry)")
	}
}

func TestWithdrawAllBoundTo(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	Declare("a", DeclareOpts{Description: "a", WithdrawOnRespawnOf: 500})
	Declare("b", DeclareOpts{Description: "b", WithdrawOnRespawnOf: 500})
	Declare("c", DeclareOpts{Description: "c"})

	count := WithdrawAllBoundTo(500)
	if count != 2 {
		t.Errorf("expected 2 withdrawn, got %d", count)
	}
	if GetFact("a").Status != StatusWithdrawn || GetFact("b").Status != StatusWithdrawn {
		t.Errorf("a/b should be withdrawn")
	}
	if GetFact("c").Status != StatusActive {
		t.Errorf("c should be active")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/facts/...`
Expected: FAIL (functions undefined).

- [ ] **Step 3: Implement**

```go
// internal/facts/facts.go
package facts

import (
	"errors"
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/worldevents"
)

// Test seam.
var roundForTest func() uint64

func currentRound() uint64 {
	if roundForTest != nil {
		return roundForTest()
	}
	return util.GetRoundCount()
}

func heardEventsMax() int {
	return configs.GetBalanceConfig().FactsHeardEventsMax
}

// Suppress unused-import warnings during scaffolding.
var _ = worldevents.Local

type DeclareOpts struct {
	Description         string
	Significance        worldevents.Significance
	Zone                string
	Region              string
	ExpiryRound         uint64
	Tags                []string
	WithdrawOnRespawnOf int
}

// Declare adds a new active fact. Returns ErrFactCollision if a fact
// with the same id already exists.
func Declare(factId string, opts DeclareOpts) error {
	r := loadOrLazyInitRegistry()

	registryMu.Lock()
	for _, f := range r.Facts {
		if f.Id == factId {
			registryMu.Unlock()
			return fmt.Errorf("fact collision: %q already exists", factId)
		}
	}
	now := currentRound()
	f := &Fact{
		Id:                  factId,
		Description:         opts.Description,
		Significance:        opts.Significance,
		Zone:                opts.Zone,
		Region:              opts.Region,
		DeclaredRound:       now,
		ExpiryRound:         opts.ExpiryRound,
		Tags:                opts.Tags,
		WithdrawOnRespawnOf: opts.WithdrawOnRespawnOf,
		Status:              StatusActive,
	}
	r.Facts = append(r.Facts, f)
	registryMu.Unlock()

	if err := saveRegistry(r); err != nil {
		mudlog.Warn("facts.Declare: save failed", "factId", factId, "error", err)
		return err
	}
	return nil
}

// GetFact returns the fact with the given id, regardless of status,
// or nil if not found.
func GetFact(factId string) *Fact {
	r := loadOrLazyInitRegistry()
	registryMu.RLock()
	defer registryMu.RUnlock()
	for _, f := range r.Facts {
		if f.Id == factId {
			return f
		}
	}
	return nil
}

// Withdraw flips a fact to status=withdrawn. Idempotent.
func Withdraw(factId string) {
	setStatus(factId, StatusWithdrawn)
}

// Expire flips a fact to status=expired. Idempotent.
func Expire(factId string) {
	setStatus(factId, StatusExpired)
}

func setStatus(factId string, newStatus Status) {
	r := loadOrLazyInitRegistry()

	registryMu.Lock()
	mutated := false
	for _, f := range r.Facts {
		if f.Id == factId && f.Status == StatusActive {
			f.Status = newStatus
			mutated = true
			break
		}
	}
	registryMu.Unlock()

	if mutated {
		if err := saveRegistry(r); err != nil {
			mudlog.Warn("facts.setStatus: save failed", "factId", factId, "status", newStatus, "error", err)
		}
	}
}

// PruneExpired walks active facts; flips any past expiry_round to
// expired. Returns count.
func PruneExpired() int {
	r := loadOrLazyInitRegistry()
	now := currentRound()

	registryMu.Lock()
	count := 0
	for _, f := range r.Facts {
		if f.Status != StatusActive {
			continue
		}
		if f.ExpiryRound == 0 {
			continue // never
		}
		if f.ExpiryRound <= now {
			f.Status = StatusExpired
			count++
		}
	}
	registryMu.Unlock()

	if count > 0 {
		if err := saveRegistry(r); err != nil {
			mudlog.Warn("facts.PruneExpired: save failed", "error", err)
		}
	}
	return count
}

// WithdrawAllBoundTo flips active facts whose WithdrawOnRespawnOf
// matches the given mob template id to status=withdrawn. Returns
// count flipped. Used by the auto-withdraw RoomChange hook.
func WithdrawAllBoundTo(mobTemplateId int) int {
	if mobTemplateId == 0 {
		return 0
	}
	r := loadOrLazyInitRegistry()

	registryMu.Lock()
	count := 0
	for _, f := range r.Facts {
		if f.Status != StatusActive {
			continue
		}
		if f.WithdrawOnRespawnOf == mobTemplateId {
			f.Status = StatusWithdrawn
			count++
		}
	}
	registryMu.Unlock()

	if count > 0 {
		if err := saveRegistry(r); err != nil {
			mudlog.Warn("facts.WithdrawAllBoundTo: save failed", "mobId", mobTemplateId, "error", err)
		}
	}
	return count
}

var _ = errors.New // keeps errors import if unused above
```

(Drop the `var _ = ...` lines if linter doesn't complain.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/facts/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/facts/
git commit -m "feat(facts): Declare + Withdraw + Expire + PruneExpired + WithdrawAllBoundTo (T7)"
```

---

## Task 8: Registry reads — AllActiveFacts, AllFactsByTag, AllRows

**Files:**
- Modify: `internal/facts/facts.go`
- Modify: `internal/facts/facts_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestRegistryReads(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	Declare("a", DeclareOpts{Description: "a", Tags: []string{"politics"}})
	Declare("b", DeclareOpts{Description: "b", Tags: []string{"politics", "war"}})
	Declare("c", DeclareOpts{Description: "c", Tags: []string{"travel"}})
	Withdraw("c")

	if got := len(AllActiveFacts()); got != 2 {
		t.Errorf("AllActiveFacts: %d want 2", got)
	}
	if got := len(AllRows()); got != 3 {
		t.Errorf("AllRows: %d want 3", got)
	}
	if got := len(AllFactsByTag("politics")); got != 2 {
		t.Errorf("politics tag count: %d", got)
	}
	if got := len(AllFactsByTag("war")); got != 1 {
		t.Errorf("war tag count: %d", got)
	}
	if got := len(AllFactsByTag("travel")); got != 0 {
		t.Errorf("travel tag (withdrawn): %d, want 0 (active only)", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/facts/...`
Expected: FAIL.

- [ ] **Step 3: Implement**

Append to `internal/facts/facts.go`:

```go
// AllActiveFacts returns every fact currently in StatusActive.
func AllActiveFacts() []*Fact {
	r := loadOrLazyInitRegistry()
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Fact, 0)
	for _, f := range r.Facts {
		if f.Status == StatusActive {
			out = append(out, f)
		}
	}
	return out
}

// AllFactsByTag returns active facts that include the given tag.
func AllFactsByTag(tag string) []*Fact {
	r := loadOrLazyInitRegistry()
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Fact, 0)
	for _, f := range r.Facts {
		if f.Status != StatusActive {
			continue
		}
		for _, t := range f.Tags {
			if t == tag {
				out = append(out, f)
				break
			}
		}
	}
	return out
}

// AllRows returns every fact regardless of status. Admin/debug use.
func AllRows() []*Fact {
	r := loadOrLazyInitRegistry()
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Fact, len(r.Facts))
	copy(out, r.Facts)
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/facts/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/facts/
git commit -m "feat(facts): registry reads — AllActiveFacts/ByTag/AllRows (T8)"
```

---

## Task 9: Awareness API — events (RecordHeardEvent + HeardEvent + bounded FIFO)

**Files:**
- Modify: `internal/facts/facts.go`
- Modify: `internal/facts/facts_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestRecordHeardEvent_BoundedFIFO(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil; heardEventsMaxForTest = nil }()
	roundForTest = func() uint64 { return 100 }
	heardEventsMaxForTest = func() int { return 4 }

	for i := uint64(1); i <= 6; i++ {
		RecordHeardEvent(114, i)
	}
	a := loadOrLazyInitAwareness(114, "")
	if len(a.HeardEvents) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(a.HeardEvents))
	}
	want := []uint64{3, 4, 5, 6}
	for i, e := range a.HeardEvents {
		if e != want[i] {
			t.Errorf("entry %d: got %d, want %d", i, e, want[i])
		}
	}
}

func TestHeardEvent(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	if HeardEvent(114, 42) {
		t.Errorf("HeardEvent should be false for unknown")
	}
	RecordHeardEvent(114, 42)
	if !HeardEvent(114, 42) {
		t.Errorf("HeardEvent should be true after Record")
	}
}

func TestRecordHeardEvent_DedupSameId(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	RecordHeardEvent(114, 42)
	RecordHeardEvent(114, 42) // duplicate
	a := loadOrLazyInitAwareness(114, "")
	if len(a.HeardEvents) != 1 {
		t.Errorf("duplicate event id should dedupe, got %d", len(a.HeardEvents))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/facts/...`
Expected: FAIL.

- [ ] **Step 3: Implement**

Append to `internal/facts/facts.go`:

```go
// Test seam for bounded FIFO cap.
var heardEventsMaxForTest func() int

func effectiveHeardEventsMax() int {
	if heardEventsMaxForTest != nil {
		return heardEventsMaxForTest()
	}
	return heardEventsMax()
}

// RecordHeardEvent appends an event id to the observer's heard_events
// FIFO. Bounded; oldest entries fall off when over cap. Dedups: if
// the eventId is already in the list, no-op.
func RecordHeardEvent(observerMobId int, eventId uint64) {
	a := loadOrLazyInitAwareness(observerMobId, observerNameFor(observerMobId))
	now := currentRound()
	cap := effectiveHeardEventsMax()

	awarenessCacheMu.Lock()
	for _, e := range a.HeardEvents {
		if e == eventId {
			awarenessCacheMu.Unlock()
			return
		}
	}
	a.HeardEvents = append(a.HeardEvents, eventId)
	if cap > 0 && len(a.HeardEvents) > cap {
		a.HeardEvents = a.HeardEvents[len(a.HeardEvents)-cap:]
	}
	a.LastUpdatedRound = now
	awarenessCacheMu.Unlock()

	if err := saveAwareness(a); err != nil {
		mudlog.Warn("facts.RecordHeardEvent: save failed", "observer", observerMobId, "error", err)
	}
}

// HeardEvent reports whether the observer has gossiped about this event.
func HeardEvent(observerMobId int, eventId uint64) bool {
	a := loadOrLazyInitAwareness(observerMobId, observerNameFor(observerMobId))
	awarenessCacheMu.RLock()
	defer awarenessCacheMu.RUnlock()
	for _, e := range a.HeardEvents {
		if e == eventId {
			return true
		}
	}
	return false
}

// observerNameFor looks up the mob template name for filename
// purposes. Mirrors chunk 1.4 pattern.
var observerNameFor = func(mobId int) string {
	// Imported via mobs package in production; tests can override
	// directly. For v1 minimal viable, return empty when caller
	// doesn't have a name handy.
	return ""
}
```

NOTE: the `observerNameFor` stub returns empty string by default. T11 (LoadFromMobs) will wire it up to `mobs.GetMobSpec`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/facts/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/facts/
git commit -m "feat(facts): RecordHeardEvent + HeardEvent with bounded FIFO + dedup (T9)"
```

---

## Task 10: Awareness API — facts (Record/Knows/KnownFactsOf with lazy join, ForgetFact, ForgetAll, AllForObserver)

**Files:**
- Modify: `internal/facts/facts.go`
- Modify: `internal/facts/facts_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestRecordKnowsFact(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	Declare("foo", DeclareOpts{Description: "foo"})
	RecordKnowsFact(114, "foo", SourceWitnessed)

	if !KnowsFact(114, "foo") {
		t.Errorf("KnowsFact should be true")
	}

	// Idempotent — re-record doesn't dupe.
	RecordKnowsFact(114, "foo", SourceWitnessed)
	a := loadOrLazyInitAwareness(114, "")
	if len(a.KnownFacts) != 1 {
		t.Errorf("re-record should dedupe: %d entries", len(a.KnownFacts))
	}
}

func TestKnownFactsOf_LazyFilter(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	Declare("a", DeclareOpts{Description: "a"})
	Declare("b", DeclareOpts{Description: "b"})
	RecordKnowsFact(114, "a", SourceWitnessed)
	RecordKnowsFact(114, "b", SourceTold)

	got := KnownFactsOf(114)
	if len(got) != 2 {
		t.Errorf("KnownFactsOf: %d, want 2", len(got))
	}

	// Withdraw 'a'; lazy filter excludes it.
	Withdraw("a")
	got = KnownFactsOf(114)
	if len(got) != 1 {
		t.Errorf("after Withdraw: %d, want 1", len(got))
	}
	if got[0].Fact.Id != "b" {
		t.Errorf("expected only b: %v", got)
	}
}

func TestForgetFact(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	Declare("a", DeclareOpts{Description: "a"})
	Declare("b", DeclareOpts{Description: "b"})
	RecordKnowsFact(114, "a", SourceWitnessed)
	RecordKnowsFact(114, "b", SourceTold)

	ForgetFact(114, "a")
	if KnowsFact(114, "a") {
		t.Errorf("a should be forgotten")
	}
	if !KnowsFact(114, "b") {
		t.Errorf("b should still be known")
	}
}

func TestForgetAll(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	Declare("a", DeclareOpts{Description: "a"})
	RecordKnowsFact(114, "a", SourceWitnessed)
	RecordHeardEvent(114, 42)

	ForgetAll(114)
	if KnowsFact(114, "a") {
		t.Errorf("known facts should be cleared")
	}
	if HeardEvent(114, 42) {
		t.Errorf("heard events should be cleared")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/facts/...`
Expected: FAIL.

- [ ] **Step 3: Implement**

Append to `internal/facts/facts.go`:

```go
// RecordKnowsFact appends a fact-knowledge entry to the observer's
// known_facts. Idempotent — if the same factId is already present,
// updates the source/learnedRound fields rather than adding a dupe.
func RecordKnowsFact(observerMobId int, factId string, source Source) {
	a := loadOrLazyInitAwareness(observerMobId, observerNameFor(observerMobId))
	now := currentRound()

	awarenessCacheMu.Lock()
	for _, fk := range a.KnownFacts {
		if fk.FactId == factId {
			awarenessCacheMu.Unlock()
			return // already known; no change
		}
	}
	a.KnownFacts = append(a.KnownFacts, FactKnowledge{
		FactId:       factId,
		Source:       source,
		LearnedRound: now,
	})
	a.LastUpdatedRound = now
	awarenessCacheMu.Unlock()

	if err := saveAwareness(a); err != nil {
		mudlog.Warn("facts.RecordKnowsFact: save failed", "observer", observerMobId, "factId", factId, "error", err)
	}
}

// KnowsFact reports whether the observer has the given fact in their
// known list, regardless of fact status.
func KnowsFact(observerMobId int, factId string) bool {
	a := loadOrLazyInitAwareness(observerMobId, observerNameFor(observerMobId))
	awarenessCacheMu.RLock()
	defer awarenessCacheMu.RUnlock()
	for _, fk := range a.KnownFacts {
		if fk.FactId == factId {
			return true
		}
	}
	return false
}

type KnownFact struct {
	Fact         *Fact
	Source       Source
	LearnedRound uint64
}

// KnownFactsOf returns the joined view: walks the observer's known
// fact ids against the active fact registry. Lazy filter — withdrawn
// or expired facts are skipped.
func KnownFactsOf(observerMobId int) []KnownFact {
	a := loadOrLazyInitAwareness(observerMobId, observerNameFor(observerMobId))
	awarenessCacheMu.RLock()
	known := make([]FactKnowledge, len(a.KnownFacts))
	copy(known, a.KnownFacts)
	awarenessCacheMu.RUnlock()

	out := make([]KnownFact, 0, len(known))
	for _, fk := range known {
		f := GetFact(fk.FactId)
		if f == nil || f.Status != StatusActive {
			continue
		}
		out = append(out, KnownFact{
			Fact:         f,
			Source:       fk.Source,
			LearnedRound: fk.LearnedRound,
		})
	}
	return out
}

// ForgetFact drops a single fact awareness entry.
func ForgetFact(observerMobId int, factId string) {
	a := loadOrLazyInitAwareness(observerMobId, observerNameFor(observerMobId))
	now := currentRound()

	awarenessCacheMu.Lock()
	out := a.KnownFacts[:0]
	mutated := false
	for _, fk := range a.KnownFacts {
		if fk.FactId == factId {
			mutated = true
			continue
		}
		out = append(out, fk)
	}
	a.KnownFacts = out
	if mutated {
		a.LastUpdatedRound = now
	}
	awarenessCacheMu.Unlock()

	if mutated {
		if err := saveAwareness(a); err != nil {
			mudlog.Warn("facts.ForgetFact: save failed", "observer", observerMobId, "factId", factId, "error", err)
		}
	}
}

// ForgetAll drops every awareness entry (heard events + known facts)
// for an observer.
func ForgetAll(observerMobId int) {
	a := loadOrLazyInitAwareness(observerMobId, observerNameFor(observerMobId))
	now := currentRound()

	awarenessCacheMu.Lock()
	a.HeardEvents = nil
	a.KnownFacts = nil
	a.LastUpdatedRound = now
	awarenessCacheMu.Unlock()

	if err := saveAwareness(a); err != nil {
		mudlog.Warn("facts.ForgetAll: save failed", "observer", observerMobId, "error", err)
	}
}

// AllForObserver returns the raw awareness record for an observer.
// Used by admin commands.
func AllForObserver(observerMobId int) *Awareness {
	return loadOrLazyInitAwareness(observerMobId, observerNameFor(observerMobId))
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/facts/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/facts/
git commit -m "feat(facts): awareness API for facts — Record/Knows/KnownFactsOf/Forget (T10)"
```

---

## Task 11: Mob YAML field + LoadFromMobs hook

**Files:**
- Modify: `internal/mobs/mobs.go`
- Modify: `internal/facts/facts.go`

- [ ] **Step 1: Add the YAML field on the Mob struct**

In `internal/mobs/mobs.go`, locate the `type Mob struct {` block. Add a new field near `Relationships` (chunk 1.6 added that field):

```go
KnowsFacts []string `yaml:"knows_facts,omitempty"`
```

- [ ] **Step 2: Add LoadFromMobs to the facts package**

Append to `internal/facts/facts.go`:

```go
// MobAwarenessSeed is the per-mob authoring shape: which facts a
// given template knows at startup.
type MobAwarenessSeed struct {
	MobId      int
	MobName    string
	KnowsFacts []string
}

// LoadFromMobs seeds per-NPC awareness records from authored
// `knows_facts:` declarations on mob YAMLs. Called by mobs.LoadDataFiles
// after the mob registry is built. Wires the production observerNameFor
// to a real lookup function, then walks the seeds.
func LoadFromMobs(seeds []MobAwarenessSeed, nameLookup func(mobId int) string) {
	if nameLookup != nil {
		observerNameFor = nameLookup
	}

	for _, s := range seeds {
		if len(s.KnowsFacts) == 0 {
			continue
		}
		for _, factId := range s.KnowsFacts {
			RecordKnowsFact(s.MobId, factId, SourceWitnessed)
		}
	}
}
```

- [ ] **Step 3: Wire LoadFromMobs into mobs.LoadDataFiles**

At the end of `LoadDataFiles` in `internal/mobs/mobs.go` (after the `mudlog.Info` line at ~line 1124, alongside the chunk 1.6 `relationships.LoadFromMobs(...)` call):

```go
import (
	"github.com/GoMudEngine/GoMud/internal/facts"
)

// At the end of LoadDataFiles:
mobsMu.RLock()
factSeeds := make([]facts.MobAwarenessSeed, 0, len(mobs))
for id, spec := range mobs {
	if len(spec.KnowsFacts) == 0 {
		continue
	}
	factSeeds = append(factSeeds, facts.MobAwarenessSeed{
		MobId:      id,
		MobName:    spec.Character.Name,
		KnowsFacts: spec.KnowsFacts,
	})
}
mobsMu.RUnlock()

facts.LoadFromMobs(factSeeds, func(mobId int) string {
	mobsMu.RLock()
	defer mobsMu.RUnlock()
	if spec, ok := mobs[mobId]; ok {
		return spec.Character.Name
	}
	return ""
})
```

(Mirror chunk 1.6's pattern — read locks during the snapshot, release before the substrate call.)

- [ ] **Step 4: Verify build + tests**

Run: `go build ./... && go test ./internal/mobs/... ./internal/facts/...`
Expected: clean / PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/mobs/mobs.go internal/facts/facts.go
git commit -m "feat(facts): mob YAML knows_facts field + LoadFromMobs hook (T11)"
```

---

## Task 12: RoomChange auto-withdraw hook

**Files:**
- Create: `internal/hooks/MobRoomChange_FactsAutoWithdraw.go`
- Modify: `internal/hooks/hooks.go`

- [ ] **Step 1: Implement the listener**

```go
// internal/hooks/MobRoomChange_FactsAutoWithdraw.go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/facts"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// MobRoomChangeFactsAutoWithdraw withdraws active facts whose
// WithdrawOnRespawnOf matches the moving mob's template id. Fires
// on every RoomChange where MobInstanceId != 0; the cost is one
// map lookup (most facts won't have the bound field set).
func MobRoomChangeFactsAutoWithdraw(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok || evt.MobInstanceId == 0 {
		return events.Continue
	}
	mob := mobs.GetInstance(evt.MobInstanceId)
	if mob == nil {
		return events.Continue
	}
	facts.WithdrawAllBoundTo(int(mob.MobId))
	return events.Continue
}
```

- [ ] **Step 2: Register the listener**

In `internal/hooks/hooks.go` `RegisterListeners()`, add (alongside the chunk-1.4 `MobRoomChangeKnowledgeObservers`):

```go
events.RegisterListener(events.RoomChange{}, MobRoomChangeFactsAutoWithdraw)
```

- [ ] **Step 3: Verify build + existing tests**

Run: `go build ./... && go test ./internal/hooks/...`
Expected: clean / PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/MobRoomChange_FactsAutoWithdraw.go internal/hooks/hooks.go
git commit -m "feat(facts): MobRoomChange auto-withdraw listener (T12)"
```

---

## Task 13: buildGossipLine migration + fact-candidate extension

**Files:**
- Modify: `internal/hooks/MobIdle_HandleIdleMobs.go`
- Modify: `_datafiles/world/dogmud/templates/gossip_templates.yaml` (or wherever loadGossipTemplates reads from)

- [ ] **Step 1: Read the current state**

Open `internal/hooks/MobIdle_HandleIdleMobs.go`. Locate `buildGossipLine` (~line 252). Identify:
- The `mob.GetTempData("recentGossipEvents")` read.
- The `mob.SetTempData("recentGossipEvents", ...)` write.
- The candidate-selection block.
- The template-resolution call (uses `gossipTemplates` map keyed by event-type).

Also locate where `loadGossipTemplates` reads from — `templatesPath := ...` or similar — and confirm the YAML file path.

- [ ] **Step 2: Migrate the dedup mechanism**

Replace the `recentGossipEvents` TempData read with `facts.HeardEvent` checks. Replace the write with `facts.RecordHeardEvent`.

```go
// Before (paraphrased):
var recentEventKeys []string
if v := mob.GetTempData("recentGossipEvents"); v != nil {
    recentEventKeys, _ = v.([]string)
}
var candidates []worldevents.WorldEvent
for _, e := range evts {
    eKey := fmt.Sprintf("%d-%s", e.Round, e.Description)
    skip := false
    for _, rk := range recentEventKeys {
        if eKey == rk {
            skip = true
            break
        }
    }
    if !skip {
        candidates = append(candidates, e)
    }
}
// ... pick a candidate, then:
recentEventKeys = append(recentEventKeys, eKey)
// (truncate to last N)
mob.SetTempData("recentGossipEvents", recentEventKeys)

// After:
var candidates []worldevents.WorldEvent
for _, e := range evts {
    if facts.HeardEvent(int(mob.MobId), e.Id) {
        continue
    }
    candidates = append(candidates, e)
}
// ... pick a candidate (pickedEvent), then:
facts.RecordHeardEvent(int(mob.MobId), pickedEvent.Id)
```

Add `"github.com/GoMudEngine/GoMud/internal/facts"` to imports.

- [ ] **Step 3: Add fact candidates to the gossip pool**

Right after the event candidate loop, also gather known facts:

```go
factCandidates := facts.KnownFactsOf(int(mob.MobId))
```

Decide candidate-pool merging strategy:
- **Simplest v1:** if `factCandidates` is non-empty AND len(candidates) is zero, pick a random fact instead of falling back. If both pools have entries, prefer the event pool 70% of the time, fact pool 30% of the time (rough heuristic — adjust during smoke).

Pick a fact at random from `factCandidates`, then look up template:

```go
// Render via "fact-{factId}" → "fact-{tag}" → "fact-default" fallback chain.
func renderFactGossip(kf facts.KnownFact) string {
    if tmpls, ok := gossipTemplates["fact-"+kf.Fact.Id]; ok && len(tmpls) > 0 {
        return tmpls[util.Rand(len(tmpls))]
    }
    for _, tag := range kf.Fact.Tags {
        if tmpls, ok := gossipTemplates["fact-"+tag]; ok && len(tmpls) > 0 {
            return tmpls[util.Rand(len(tmpls))]
        }
    }
    if tmpls, ok := gossipTemplates["fact-default"]; ok && len(tmpls) > 0 {
        // Substitute {description} placeholder with fact description.
        line := tmpls[util.Rand(len(tmpls))]
        return strings.ReplaceAll(line, "{description}", kf.Fact.Description)
    }
    return ""
}
```

Wire `renderFactGossip` into the candidate-pool selection. Preserve all existing event-template handling.

- [ ] **Step 4: Add `fact-default` family to gossip templates YAML**

Append to `_datafiles/world/dogmud/templates/gossip_templates.yaml` (or whatever path `loadGossipTemplates` reads):

```yaml
fact-default:
  - "I heard {description}"
  - "Word is, {description}"
  - "They say {description}"
```

(Phrasing: assume `{description}` is a complete sentence; the templates use it as an embedded clause. Authors can refine later.)

- [ ] **Step 5: Verify build + existing tests**

Run: `go build ./... && go test ./internal/hooks/...`
Expected: clean / PASS. Existing buildGossipLine tests should still pass — the migration preserves event-only output behavior.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/MobIdle_HandleIdleMobs.go _datafiles/world/dogmud/templates/gossip_templates.yaml
git commit -m "feat(facts): buildGossipLine migration + fact-candidate extension (T13)"
```

---

## Task 14: Admin command + helpfile

**Files:**
- Create: `internal/usercommands/admin.fact.go`
- Modify: `internal/usercommands/usercommands.go`
- Create: `_datafiles/world/dogmud/templates/admincommands/help/command.fact.template`

- [ ] **Step 1: Read chunk-1.5 / 1.6 admin command references**

Open `internal/usercommands/admin.bounty.go` and `internal/usercommands/admin.relationship.go`. Mirror the dispatcher / help proxy / parsing pattern.

- [ ] **Step 2: Implement the admin command**

```go
// internal/usercommands/admin.fact.go
package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/facts"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/worldevents"
)

/*
 * Role Permissions:
 * fact          (Admin)
 */

func Fact(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		factUsage(user)
		return true, nil
	}
	switch strings.ToLower(args[0]) {
	case "list":
		return factList(args[1:], user)
	case "show":
		return factShow(args[1:], user)
	case "declare":
		return factDeclare(rest, user) // pass full rest for `--` description handling
	case "withdraw":
		return factWithdraw(args[1:], user)
	case "expire":
		return factExpire(args[1:], user)
	case "prune-expired":
		return factPrune(user)
	case "awareness":
		return factAwareness(args[1:], user)
	case "teach":
		return factTeach(args[1:], user)
	case "forget":
		return factForget(args[1:], user)
	case "forget-all":
		return factForgetAll(args[1:], user)
	default:
		factUsage(user)
	}
	return true, nil
}

func factUsage(user *users.UserRecord) {
	if out, err := templates.Process("admincommands/help/command.fact", nil, user.UserId); err == nil && strings.TrimSpace(out) != "" {
		user.SendText(out)
		return
	}
	user.SendText(
		"Usage:\r\n" +
			"  fact list [--all]\r\n" +
			"  fact show <factId>\r\n" +
			"  fact declare <factId> [opts...] -- <description>\r\n" +
			"  fact withdraw <factId>\r\n" +
			"  fact expire <factId>\r\n" +
			"  fact prune-expired\r\n" +
			"  fact awareness <mobId>\r\n" +
			"  fact teach <mobId> <factId> [--source S]\r\n" +
			"  fact forget <mobId> <factId>\r\n" +
			"  fact forget-all <mobId>\r\n",
	)
}

func factList(args []string, user *users.UserRecord) (bool, error) {
	all := false
	for _, a := range args {
		if a == "--all" {
			all = true
		}
	}
	var rows []*facts.Fact
	if all {
		rows = facts.AllRows()
	} else {
		rows = facts.AllActiveFacts()
	}
	if len(rows) == 0 {
		user.SendText("No facts.\r\n")
		return true, nil
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Id < rows[j].Id })

	var b strings.Builder
	fmt.Fprintf(&b, "fact list (%d):\r\n\r\n", len(rows))
	fmt.Fprintf(&b, "  %-22s  %-10s  %-10s  %-10s  %-10s  %s\r\n",
		"ID", "Status", "Sig", "Zone", "Region", "Tags")
	fmt.Fprintf(&b, "  %-22s  %-10s  %-10s  %-10s  %-10s  %s\r\n",
		"----------------------", "----------", "----------", "----------", "----------", "----")
	for _, f := range rows {
		zone := f.Zone
		if zone == "" {
			zone = "-"
		}
		region := f.Region
		if region == "" {
			region = "-"
		}
		fmt.Fprintf(&b, "  %-22s  %-10s  %-10s  %-10s  %-10s  %s\r\n",
			factTruncate(f.Id, 22),
			f.Status,
			significanceLabel(f.Significance),
			factTruncate(zone, 10),
			factTruncate(region, 10),
			strings.Join(f.Tags, ","))
	}
	user.SendText(b.String())
	return true, nil
}

func factShow(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		factUsage(user)
		return true, nil
	}
	f := facts.GetFact(args[0])
	if f == nil {
		user.SendText(fmt.Sprintf("No fact with id %q\r\n", args[0]))
		return true, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Fact: %s (%s)\r\n", f.Id, f.Status)
	fmt.Fprintf(&b, "  Description: %s\r\n", f.Description)
	fmt.Fprintf(&b, "  Significance: %s   Zone: %s   Region: %s\r\n",
		significanceLabel(f.Significance), defaultDash(f.Zone), defaultDash(f.Region))
	fmt.Fprintf(&b, "  Declared round: %d\r\n", f.DeclaredRound)
	if f.ExpiryRound > 0 {
		fmt.Fprintf(&b, "  Expiry round: %d\r\n", f.ExpiryRound)
	} else {
		fmt.Fprintf(&b, "  Expiry: never\r\n")
	}
	if len(f.Tags) > 0 {
		fmt.Fprintf(&b, "  Tags: %s\r\n", strings.Join(f.Tags, ", "))
	}
	if f.WithdrawOnRespawnOf > 0 {
		fmt.Fprintf(&b, "  Bound mob: %d (auto-withdraws on respawn)\r\n", f.WithdrawOnRespawnOf)
	}
	user.SendText(b.String())
	return true, nil
}

func factDeclare(rest string, user *users.UserRecord) (bool, error) {
	// Custom parser: split on `--` first to extract description.
	idx := strings.Index(rest, " -- ")
	if idx < 0 {
		user.SendText("Missing -- separator before description.\r\nUsage: fact declare <factId> [opts] -- <description>\r\n")
		return true, nil
	}
	beforeDash := strings.TrimSpace(rest[:idx])
	description := strings.TrimSpace(rest[idx+4:])

	args := strings.Fields(beforeDash)
	if len(args) < 2 || strings.ToLower(args[0]) != "declare" {
		factUsage(user)
		return true, nil
	}
	factId := args[1]
	opts := facts.DeclareOpts{Description: description, Significance: worldevents.Local}
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--zone":
			i++
			if i < len(args) {
				opts.Zone = args[i]
			}
		case "--region":
			i++
			if i < len(args) {
				opts.Region = args[i]
			}
		case "--significance":
			i++
			if i < len(args) {
				opts.Significance = parseSignificance(args[i])
			}
		case "--expiry-rounds":
			i++
			if i < len(args) {
				if v, err := strconv.ParseUint(args[i], 10, 64); err == nil {
					// Convert relative rounds to absolute expiry.
					opts.ExpiryRound = util.GetRoundCount() + v
				}
			}
		case "--tags":
			i++
			if i < len(args) {
				opts.Tags = strings.Split(args[i], ",")
			}
		case "--withdraw-on-respawn-of":
			i++
			if i < len(args) {
				if v, err := strconv.Atoi(args[i]); err == nil {
					opts.WithdrawOnRespawnOf = v
				}
			}
		}
	}

	if err := facts.Declare(factId, opts); err != nil {
		user.SendText(fmt.Sprintf("Declare failed: %v\r\n", err))
		return true, nil
	}
	user.SendText(fmt.Sprintf("Declared fact %q.\r\n", factId))
	return true, nil
}

func factWithdraw(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		factUsage(user)
		return true, nil
	}
	facts.Withdraw(args[0])
	user.SendText(fmt.Sprintf("Withdrew fact %q.\r\n", args[0]))
	return true, nil
}

func factExpire(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		factUsage(user)
		return true, nil
	}
	facts.Expire(args[0])
	user.SendText(fmt.Sprintf("Expired fact %q.\r\n", args[0]))
	return true, nil
}

func factPrune(user *users.UserRecord) (bool, error) {
	count := facts.PruneExpired()
	user.SendText(fmt.Sprintf("Pruned %d expired fact(s).\r\n", count))
	return true, nil
}

func factAwareness(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		factUsage(user)
		return true, nil
	}
	mobId, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[0]))
		return true, nil
	}
	a := facts.AllForObserver(mobId)
	known := facts.KnownFactsOf(mobId)
	var b strings.Builder
	fmt.Fprintf(&b, "Awareness for mob %d (%s):\r\n", mobId, defaultDash(a.ObserverName))
	fmt.Fprintf(&b, "  Heard events (%d): %v\r\n", len(a.HeardEvents), a.HeardEvents)
	fmt.Fprintf(&b, "  Known facts (%d):\r\n", len(known))
	for _, k := range known {
		fmt.Fprintf(&b, "    %-22s source: %-10s round: %d\r\n",
			factTruncate(k.Fact.Id, 22), k.Source, k.LearnedRound)
	}
	user.SendText(b.String())
	return true, nil
}

func factTeach(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 2 {
		factUsage(user)
		return true, nil
	}
	mobId, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[0]))
		return true, nil
	}
	factId := args[1]
	source := facts.SourceWitnessed
	for i := 2; i < len(args); i++ {
		if args[i] == "--source" && i+1 < len(args) {
			source = facts.Source(strings.ToLower(args[i+1]))
			i++
		}
	}
	facts.RecordKnowsFact(mobId, factId, source)
	user.SendText(fmt.Sprintf("Taught mob %d the fact %q (source: %s).\r\n", mobId, factId, source))
	return true, nil
}

func factForget(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 2 {
		factUsage(user)
		return true, nil
	}
	mobId, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[0]))
		return true, nil
	}
	facts.ForgetFact(mobId, args[1])
	user.SendText(fmt.Sprintf("Mob %d forgot fact %q.\r\n", mobId, args[1]))
	return true, nil
}

func factForgetAll(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		factUsage(user)
		return true, nil
	}
	mobId, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[0]))
		return true, nil
	}
	facts.ForgetAll(mobId)
	user.SendText(fmt.Sprintf("Mob %d forgot all facts and events.\r\n", mobId))
	return true, nil
}

// Helpers
func factTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func defaultDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func significanceLabel(s worldevents.Significance) string {
	switch s {
	case worldevents.Global:
		return "global"
	case worldevents.Regional:
		return "regional"
	default:
		return "local"
	}
}

func parseSignificance(s string) worldevents.Significance {
	switch strings.ToLower(s) {
	case "global":
		return worldevents.Global
	case "regional":
		return worldevents.Regional
	default:
		return worldevents.Local
	}
}
```

NOTE: this command uses a `util` package import for `util.GetRoundCount()`; add that import.

- [ ] **Step 3: Register the command**

In `internal/usercommands/usercommands.go`, find where `Bounty` and `Relationship` are registered (chunks 1.5 and 1.6). Add an entry for `fact`:

```go
"fact": {Fact, true, true, true},  // admin-only, mirror Knowledge / Relationship
```

Read surrounding entries to confirm the right shape.

- [ ] **Step 4: Write the help template**

Create `_datafiles/world/dogmud/templates/admincommands/help/command.fact.template`:

```
<ansi fg="white-bold">fact</ansi> — manage standing facts and per-NPC awareness

  <ansi fg="cyan">fact list</ansi> [<ansi fg="yellow">--all</ansi>]
    List active facts (default) or every status.

  <ansi fg="cyan">fact show</ansi> <ansi fg="yellow"><factId></ansi>
    Full detail for a fact.

  <ansi fg="cyan">fact declare</ansi> <ansi fg="yellow"><factId> [opts] -- <description></ansi>
    Declare a new fact.
      Options:
        --zone Z
        --region R
        --significance S       (local | regional | global)
        --expiry-rounds N      (relative; 0 = never)
        --tags a,b,c
        --withdraw-on-respawn-of MOBID

  <ansi fg="cyan">fact withdraw</ansi> <ansi fg="yellow"><factId></ansi>
  <ansi fg="cyan">fact expire</ansi> <ansi fg="yellow"><factId></ansi>
  <ansi fg="cyan">fact prune-expired</ansi>

  <ansi fg="cyan">fact awareness</ansi> <ansi fg="yellow"><mobId></ansi>
    Show heard events + known facts for an NPC.

  <ansi fg="cyan">fact teach</ansi> <ansi fg="yellow"><mobId> <factId></ansi> [<ansi fg="yellow">--source S</ansi>]
    Make an NPC aware of a fact. Source ∈ witnessed | told | deduced | unknown.

  <ansi fg="cyan">fact forget</ansi> <ansi fg="yellow"><mobId> <factId></ansi>
    Drop a single fact awareness from an NPC.

  <ansi fg="cyan">fact forget-all</ansi> <ansi fg="yellow"><mobId></ansi>
    Drop all awareness (heard events + known facts) for an NPC.
```

- [ ] **Step 5: Verify build + tests**

Run: `go build ./... && go test ./internal/usercommands/... ./internal/facts/...`
Expected: clean / PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/admin.fact.go internal/usercommands/usercommands.go _datafiles/world/dogmud/templates/admincommands/help/command.fact.template
git commit -m "feat(facts): admin fact command + helpfile (T14)"
```

---

## Task 15: `context.md` for `internal/facts/`

**Files:**
- Create: `internal/facts/context.md`

Per the aliveness-roadmap rule: every chunk that creates a new package SHIPS a `context.md`. Mirror the style of chunks 1.4 / 1.5 / 1.6 backfills + chunk 1.6's fresh one.

- [ ] **Step 1: Read the package**

```bash
ls internal/facts/
wc -l internal/facts/*.go
```

Read each `.go` file.

- [ ] **Step 2: Write the file**

Required sections:
- **Overview** — standing-fact registry + per-NPC awareness store; replaces `recentGossipEvents` TempData.
- **Key Components** — file map (types.go, persistence.go, facts.go).
- **Key Functions** — public API: Declare, Withdraw, Expire, PruneExpired, WithdrawAllBoundTo, GetFact, AllActiveFacts, AllFactsByTag, AllRows, RecordHeardEvent, HeardEvent, RecordKnowsFact, KnowsFact, KnownFactsOf, ForgetFact, ForgetAll, AllForObserver, LoadFromMobs, MobAwarenessSeed.
- **Global State** — registry + cache + mutexes + test seams (roundForTest, heardEventsMaxForTest, observerNameFor).
- **Data Structure Design** — Fact, FactKnowledge, Awareness, Registry. Sample YAMLs for both registry and awareness.
- **Lifecycle** — three withdraw signals (manual, time-based, auto-withdraw on respawn). Lazy filter on read for awareness/registry join.
- **Integration Notes** — worldevents Id field; mob YAML knows_facts seed; MobRoomChange auto-withdraw hook; buildGossipLine consumer.
- **Testing Notes** — *_test.go files; TestMain temp-dir; key test scenarios.

Length target ~250–350 lines.

- [ ] **Step 3: Verify renders cleanly**

- [ ] **Step 4: Commit**

```bash
git add internal/facts/context.md
git commit -m "docs(facts): context.md for chunk 1.7"
```

---

## Task 16: Roadmap update + empty registry seed

**Files:**
- Create: `_datafiles/world/dogmud/facts.yaml`
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Create empty registry seed file**

```yaml
# _datafiles/world/dogmud/facts.yaml
facts: []
```

This ships as a committed empty seed so the registry file exists from day one. Content authors can append authored facts here.

- [ ] **Step 2: Mark chunk 1.7 as Done**

Locate the `| 1.7 | Substrate | World-model facts | M | 1.4 | Not started |` row and change `Not started` to `Done`. Update the roll-up: `6 / 40 done` → `7 / 40 done`.

- [ ] **Step 3: Update the chunk's mini-brief**

Locate the `### 1.7 World-model facts` section. Change `**Status:** Not started` to `**Status:** Done (2026-05-09)`. Append a `Shipped:` paragraph mirroring the format of 1.1–1.6.

Suggested template:

```markdown
- **Shipped:** `internal/facts/` package storing the standing-fact registry at `_datafiles/world/dogmud/facts.yaml` (committed; empty seed) and per-NPC awareness at `_datafiles/world/dogmud/facts.awareness/{mobId}-{namesimple}.yaml` (gitignored). Awareness store is unified — it holds BOTH heard-event ids (bounded FIFO via `FactsHeardEventsMax`, default 32; replaces the in-memory `recentGossipEvents` TempData) AND known-fact ids (persistent). Three withdraw signals: manual `Withdraw`, time-based `expiry_round` + `PruneExpired` sweep, auto via `withdraw_on_respawn_of` field with new `MobRoomChange_FactsAutoWithdraw` listener. Lazy-filter on read for awareness × registry join. Public API: Declare, Withdraw, Expire, PruneExpired, WithdrawAllBoundTo, GetFact, AllActiveFacts, AllFactsByTag, AllRows, RecordHeardEvent, HeardEvent, RecordKnowsFact, KnowsFact, KnownFactsOf, ForgetFact, ForgetAll, AllForObserver, LoadFromMobs. Mob YAML extension: `knows_facts: [factId, ...]` for inline authoring (chunk 1.6 pattern); seeded into awareness at `mobs.LoadDataFiles` post-load. Worldevents gained `Id uint64` field, atomic-counter assigned at `EmitWorldEvent` time. `buildGossipLine` migrated from TempData to facts substrate; gossip candidate pool extended with known facts (new `fact-default` template family). Admin command `fact list/show/declare/withdraw/expire/prune-expired/awareness/teach/forget/forget-all` + helpfile. New package context.md authored. Spec at `docs/superpowers/specs/2026-05-09-mob-aliveness-1.7-world-facts-design.md`, plan at `docs/superpowers/plans/2026-05-09-mob-aliveness-1.7-world-facts.md`.
```

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/facts.yaml MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(roadmap): mark chunk 1.7 (world-model facts) as Done"
```

---

## Final review

After all tasks complete, dispatch the `superpowers:code-reviewer` agent for a holistic pass before smoke testing.

Suggested smoke goal file: `tools/testing/goals/facts-thornwall-smoke.yaml` covering the steps from the spec's Testing section (admin declares fact, teaches Fen, idle in Back Corner room 484, observe gossip line, withdraw fact, observe absence; declare fact bound to mob 500 + force respawn + observe auto-withdraw).

Branch state after completion: `feature/mob-aliveness-1.3-crimes` carries chunks 1.1, 1.2, 1.3, 1.4, 1.5, 1.6, AND 1.7 — the complete Phase 1 substrate. Next phase (Phase 2 tactical, Phase 3 routine, or pivoting to a Phase 5 cross-cutting feature now that all substrate is in place) is open. When confidence is high on chunk 1.7, merge to development with `--no-ff`.
