# Mob Aliveness 1.4 — NPC Knowledge Model Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a per-NPC × per-subject knowledge substrate (identity, location, observation log, witnessed-crime IDs) with v1 auto-write hooks for forager/caravan observation and 1.3 crime witnessing.

**Architecture:** New `internal/knowledge/` package mirroring the layout of `internal/crimes/` and `internal/opinions/`. Per-observer YAML files at `_datafiles/world/dogmud/knowledge/{mobId}-{namesimple}.yaml` (gitignored). Lazy-load + in-memory cache + sync-save, mutex-guarded with the same double-check-lock pattern chunk 1.3 used. Auto-write triggers via a new hook listener on the existing `RoomChange` event (filters for forager/caravan archetypes) and via direct calls at the three 1.3 crime-record sites.

**Tech Stack:** Go 1.21+, YAML persistence (`gopkg.in/yaml.v3`), existing `internal/events` event bus, existing `internal/configs/Balance` config surface.

**Spec:** `docs/superpowers/specs/2026-05-09-mob-aliveness-1.4-knowledge-model-design.md`

**Branch:** Stay on `feature/mob-aliveness-1.3-crimes` per the single-feature-branch SOP for the aliveness effort. (Branch name predates 1.4; that's fine — it's the long-running aliveness branch.)

---

## File structure

| File | Responsibility |
|------|----------------|
| `internal/knowledge/types.go` | `Subject`, `Source`, `Confidence`, `Observation`, `Record`, `ObserverFile` types |
| `internal/knowledge/persistence.go` | Base dir, filepath helper, YAML load/save, lazy-load + cache + mutex |
| `internal/knowledge/knowledge.go` | Public API (write/read/forget) |
| `internal/knowledge/decay.go` | `FrequentedRooms` tally helper, observation log bounding |
| `internal/knowledge/test_main_test.go` | `TestMain` that sets up temp data dir (mirrors crimes/opinions) |
| `internal/knowledge/types_test.go` | Type-level tests (subject equality, defaults) |
| `internal/knowledge/persistence_test.go` | Round-trip tests |
| `internal/knowledge/knowledge_test.go` | API tests (write paths, reads, forget) |
| `internal/knowledge/decay_test.go` | FrequentedRooms tests |
| `internal/configs/config.balance.misc.go` | Add `KnowledgeObservationLogMax` knob |
| `internal/hooks/MobRoomChange_KnowledgeObservers.go` | Listener: forager/caravan move → write observations |
| `internal/hooks/hooks.go` | Register the new listener |
| `internal/hooks/MobRoomChange_KnowledgeObservers_test.go` | Hook integration test |
| `internal/usercommands/attack.go` | Add knowledge writes after `recordAssaultCrime` |
| `internal/hooks/MobDeath_FactionRep.go` | Add knowledge writes on fresh-murder + upgrade paths |
| `internal/usercommands/skill.skullduggery.steal.go` | Add knowledge writes on failed-steal |
| `internal/usercommands/admin.knowledge.go` | Admin command `knowledge show/forget/frequented` |
| `_datafiles/templates/admincommands/help/command.knowledge.template` | Admin help template |
| `MOB_ALIVENESS_ROADMAP.md` | Mark 1.4 status |

---

## Task 1: Package skeleton + types

**Files:**
- Create: `internal/knowledge/types.go`
- Create: `internal/knowledge/types_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/knowledge/types_test.go
package knowledge

import "testing"

func TestSubjectEquality(t *testing.T) {
	a := Subject{Type: SubjectPlayer, Id: 17}
	b := Subject{Type: SubjectPlayer, Id: 17}
	c := Subject{Type: SubjectMob, Id: 17}
	if a != b {
		t.Errorf("expected equal subjects to compare equal")
	}
	if a == c {
		t.Errorf("player(17) should not equal mob(17)")
	}
}

func TestSubjectHelpers(t *testing.T) {
	if PlayerSubject(17) != (Subject{Type: SubjectPlayer, Id: 17}) {
		t.Errorf("PlayerSubject helper mismatch")
	}
	if MobSubject(99) != (Subject{Type: SubjectMob, Id: 99}) {
		t.Errorf("MobSubject helper mismatch")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/knowledge/...`
Expected: build failure — package doesn't exist yet.

- [ ] **Step 3: Implement types.go**

```go
// internal/knowledge/types.go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/knowledge/...`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/knowledge/types.go internal/knowledge/types_test.go
git commit -m "feat(knowledge): types + subject helpers (chunk 1.4 T1)"
```

---

## Task 2: TestMain + persistence skeleton

**Files:**
- Create: `internal/knowledge/test_main_test.go`
- Create: `internal/knowledge/persistence.go`

- [ ] **Step 1: Read `internal/crimes/test_main_test.go`**

Goal: copy the same temp-dir setup pattern used by crimes/opinions so tests don't write to the real data dir.

- [ ] **Step 2: Write test_main_test.go for knowledge**

```go
// internal/knowledge/test_main_test.go
package knowledge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

func TestMain(m *testing.M) {
	// Use a temp dir so tests don't pollute real data files.
	tmp, err := os.MkdirTemp("", "knowledge-test-*")
	if err != nil {
		os.Exit(1)
	}
	defer os.RemoveAll(tmp)

	cfg := configs.GetFilePathsConfig()
	cfg.DataFiles = configs.ConfigString(tmp)
	configs.SetFilePathsConfig(cfg)

	if err := os.MkdirAll(filepath.Join(tmp, "world", "dogmud", "knowledge"), 0o755); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// resetCache wipes the in-memory cache between tests so each test starts clean.
func resetCache() {
	knowledgeCacheMu.Lock()
	knowledgeCache = make(map[int]*ObserverFile)
	knowledgeCacheMu.Unlock()
}
```

- [ ] **Step 3: Implement persistence.go skeleton**

```go
// internal/knowledge/persistence.go
package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v3"
)

var (
	knowledgeCache   = make(map[int]*ObserverFile)
	knowledgeCacheMu sync.RWMutex
	saveMu           sync.Mutex // serializes disk writes (Windows file-lock safety)
)

func knowledgeBaseDir() string {
	return filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "world", "dogmud", "knowledge")
}

// observerFilePath returns the absolute path for the given observer mob template id.
// The mobName is used in the filename for human readability; mismatch is
// tolerated (filename is not the lookup key).
func observerFilePath(mobId int, mobName string) string {
	return filepath.Join(knowledgeBaseDir(),
		fmt.Sprintf("%d-%s.yaml", mobId, util.ConvertForFilename(mobName)))
}
```

- [ ] **Step 4: Run tests to verify build**

Run: `go test ./internal/knowledge/...`
Expected: PASS (Task 1 tests still pass; new compile clean).

- [ ] **Step 5: Commit**

```bash
git add internal/knowledge/test_main_test.go internal/knowledge/persistence.go
git commit -m "feat(knowledge): test harness + persistence skeleton (T2)"
```

---

## Task 3: Load/save round-trip

**Files:**
- Modify: `internal/knowledge/persistence.go`
- Create: `internal/knowledge/persistence_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/knowledge/persistence_test.go
package knowledge

import (
	"testing"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	resetCache()

	fc := &ObserverFile{
		ObserverMobId: 99,
		ObserverName:  "records_clerk_pell",
		Records: []*Record{
			{
				Subject:       PlayerSubject(17),
				HasMet:        true,
				NameLearned:   "smoketester",
				Source:        SourceWitnessed,
				Confidence:    ConfidenceHigh,
				LastSeenRoom:  462,
				LastSeenRound: 100,
				Observations: []Observation{
					{Room: 462, Round: 100},
				},
				CrimesWitnessed:  []int{1, 2},
				LearnedRound:     50,
				LastUpdatedRound: 100,
			},
		},
	}

	if err := saveObserverFile(fc); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded := loadObserverFileFromDisk(99, "records_clerk_pell")
	if loaded == nil {
		t.Fatalf("expected non-nil load")
	}
	if loaded.ObserverMobId != 99 {
		t.Errorf("ObserverMobId mismatch: got %d", loaded.ObserverMobId)
	}
	if len(loaded.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(loaded.Records))
	}
	r := loaded.Records[0]
	if r.Subject != PlayerSubject(17) {
		t.Errorf("subject mismatch: got %+v", r.Subject)
	}
	if r.NameLearned != "smoketester" {
		t.Errorf("name mismatch: got %q", r.NameLearned)
	}
	if len(r.CrimesWitnessed) != 2 || r.CrimesWitnessed[0] != 1 || r.CrimesWitnessed[1] != 2 {
		t.Errorf("crimes mismatch: got %v", r.CrimesWitnessed)
	}
}

func TestLoadMissingFileReturnsNil(t *testing.T) {
	resetCache()
	if loadObserverFileFromDisk(404, "ghost") != nil {
		t.Errorf("expected nil for missing file")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/knowledge/...`
Expected: FAIL (saveObserverFile/loadObserverFileFromDisk undefined).

- [ ] **Step 3: Implement save/load**

Append to `internal/knowledge/persistence.go`:

```go
func saveObserverFile(fc *ObserverFile) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	if err := os.MkdirAll(knowledgeBaseDir(), 0o755); err != nil {
		return fmt.Errorf("mkdir knowledge dir: %w", err)
	}
	path := observerFilePath(fc.ObserverMobId, fc.ObserverName)

	out, err := yaml.Marshal(fc)
	if err != nil {
		return fmt.Errorf("marshal observer file %d: %w", fc.ObserverMobId, err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return fmt.Errorf("write tmp file %q: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename tmp -> final %q: %w", path, err)
	}
	return nil
}

func loadObserverFileFromDisk(mobId int, mobName string) *ObserverFile {
	path := observerFilePath(mobId, mobName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	fc := &ObserverFile{}
	if err := yaml.Unmarshal(data, fc); err != nil {
		return nil
	}
	return fc
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/knowledge/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/knowledge/persistence.go internal/knowledge/persistence_test.go
git commit -m "feat(knowledge): YAML round-trip persistence (T3)"
```

---

## Task 4: Lazy-load + cache with double-check-lock

**Files:**
- Modify: `internal/knowledge/persistence.go`
- Modify: `internal/knowledge/persistence_test.go`

This is the chunk 1.3 T7 race pattern. Two concurrent readers on a cold cache must not both create separate `ObserverFile` instances.

- [ ] **Step 1: Write the failing concurrent-load test**

Append to `internal/knowledge/persistence_test.go`:

```go
import (
	"sync"
	"testing"
)

func TestLoadOrLazyInitConcurrent(t *testing.T) {
	resetCache()
	const N = 50
	var wg sync.WaitGroup
	var seen [N]*ObserverFile
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			seen[i] = loadOrLazyInit(123, "city_beggar")
		}(i)
	}
	wg.Wait()

	// All goroutines must see the same pointer (single shared cached object).
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

Run: `go test ./internal/knowledge/...`
Expected: FAIL (`loadOrLazyInit` undefined).

- [ ] **Step 3: Implement loadOrLazyInit**

Append to `internal/knowledge/persistence.go`:

```go
// loadOrLazyInit returns the cached *ObserverFile for the given observer
// mob id, loading from disk on first access. If neither cache nor disk
// has data, an empty ObserverFile is created and cached. Mirrors the
// chunk 1.3 double-check-lock pattern.
func loadOrLazyInit(mobId int, mobName string) *ObserverFile {
	knowledgeCacheMu.RLock()
	if fc, ok := knowledgeCache[mobId]; ok {
		knowledgeCacheMu.RUnlock()
		return fc
	}
	knowledgeCacheMu.RUnlock()

	if fc := loadObserverFileFromDisk(mobId, mobName); fc != nil {
		knowledgeCacheMu.Lock()
		if cached, ok := knowledgeCache[mobId]; ok {
			knowledgeCacheMu.Unlock()
			return cached
		}
		knowledgeCache[mobId] = fc
		knowledgeCacheMu.Unlock()
		return fc
	}

	fc := &ObserverFile{
		ObserverMobId: mobId,
		ObserverName:  mobName,
		Records:       []*Record{},
	}
	knowledgeCacheMu.Lock()
	if cached, ok := knowledgeCache[mobId]; ok {
		knowledgeCacheMu.Unlock()
		return cached
	}
	knowledgeCache[mobId] = fc
	knowledgeCacheMu.Unlock()
	return fc
}
```

- [ ] **Step 4: Run tests to verify they pass (race-free)**

Run: `go test -race ./internal/knowledge/...`
Expected: PASS, no race detected.

- [ ] **Step 5: Commit**

```bash
git add internal/knowledge/persistence.go internal/knowledge/persistence_test.go
git commit -m "feat(knowledge): lazy-load + cache with double-check-lock (T4)"
```

---

## Task 5: Public API — RecordMet

**Files:**
- Create: `internal/knowledge/knowledge.go`
- Create: `internal/knowledge/knowledge_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/knowledge/knowledge_test.go
package knowledge

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/util"
)

func TestRecordMet_FreshThenIdempotent(t *testing.T) {
	resetCache()
	originalRound := util.GetRoundCount()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return originalRound + 100 }

	RecordMet(99, PlayerSubject(17), 462, SourceWitnessed)

	r := Get(99, PlayerSubject(17))
	if r == nil {
		t.Fatalf("expected record after RecordMet")
	}
	if !r.HasMet {
		t.Errorf("HasMet should be true")
	}
	if r.LastSeenRoom != 462 {
		t.Errorf("LastSeenRoom mismatch: got %d", r.LastSeenRoom)
	}
	if r.LearnedRound != originalRound+100 {
		t.Errorf("LearnedRound not set: got %d", r.LearnedRound)
	}
	if r.Source != SourceWitnessed || r.Confidence != ConfidenceHigh {
		t.Errorf("source/confidence wrong: %s/%s", r.Source, r.Confidence)
	}

	// Second call — LearnedRound stays, LastSeenRound updates.
	roundForTest = func() uint64 { return originalRound + 200 }
	RecordMet(99, PlayerSubject(17), 463, SourceWitnessed)
	r = Get(99, PlayerSubject(17))
	if r.LearnedRound != originalRound+100 {
		t.Errorf("LearnedRound should not change on second call: got %d", r.LearnedRound)
	}
	if r.LastSeenRoom != 463 {
		t.Errorf("LastSeenRoom should update: got %d", r.LastSeenRoom)
	}
	if r.LastSeenRound != originalRound+200 {
		t.Errorf("LastSeenRound should update: got %d", r.LastSeenRound)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/knowledge/...`
Expected: FAIL (`RecordMet`, `Get`, `roundForTest` undefined).

- [ ] **Step 3: Implement RecordMet + Get + roundForTest seam**

```go
// internal/knowledge/knowledge.go
package knowledge

import (
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Test-only seam — overrides util.GetRoundCount(). Production never sets this.
var roundForTest func() uint64

func currentRound() uint64 {
	if roundForTest != nil {
		return roundForTest()
	}
	return util.GetRoundCount()
}

// findRecord returns the record for the given subject, or nil. Caller must
// hold knowledgeCacheMu (read or write).
func findRecord(fc *ObserverFile, subject Subject) *Record {
	for _, r := range fc.Records {
		if r.Subject == subject {
			return r
		}
	}
	return nil
}

// observerNameFor looks up the mob template name for filename purposes. If
// the lookup fails, returns "" — saveObserverFile tolerates blank names.
// Implementations may use mobs.GetMobSpec(mobs.MobId(id)).Name; this helper
// keeps the persistence layer decoupled from mobs imports.
var observerNameFor = func(mobId int) string {
	// In production this will be wired in init() or via a setter; for tests
	// we can override directly.
	return ""
}

func RecordMet(observerMobId int, subject Subject, room int, source Source) {
	fc := loadOrLazyInit(observerMobId, observerNameFor(observerMobId))
	now := currentRound()

	knowledgeCacheMu.Lock()
	r := findRecord(fc, subject)
	if r == nil {
		r = &Record{
			Subject:      subject,
			HasMet:       true,
			Source:       source,
			Confidence:   ConfidenceHigh,
			LearnedRound: now,
		}
		fc.Records = append(fc.Records, r)
	} else {
		r.HasMet = true
	}
	r.LastSeenRoom = room
	r.LastSeenRound = now
	r.LastUpdatedRound = now
	knowledgeCacheMu.Unlock()

	if err := saveObserverFile(fc); err != nil {
		mudlog.Warn("knowledge.RecordMet: save failed", "observer", observerMobId, "error", err)
	}
}

func Get(observerMobId int, subject Subject) *Record {
	fc := loadOrLazyInit(observerMobId, observerNameFor(observerMobId))
	knowledgeCacheMu.RLock()
	defer knowledgeCacheMu.RUnlock()
	return findRecord(fc, subject)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/knowledge/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/knowledge/knowledge.go internal/knowledge/knowledge_test.go
git commit -m "feat(knowledge): RecordMet + Get (T5)"
```

---

## Task 6: RecordObservation with bounded log

**Files:**
- Modify: `internal/knowledge/knowledge.go`
- Modify: `internal/knowledge/knowledge_test.go`
- Modify: `internal/configs/config.balance.misc.go`

- [ ] **Step 1: Add config knob**

Open `internal/configs/config.balance.misc.go` and locate the existing crime-related knobs (e.g. `CrimeStaleAfterRounds`). Add immediately after them:

```go
// KnowledgeObservationLogMax caps the per-record observation log
// at the most-recent N entries (FIFO). Sized so the frequented-rooms
// top-K query has signal without files growing unbounded.
KnowledgeObservationLogMax: 32,
```

(Add the field declaration to the `Balance` struct in `config.balance.go` if not already present:)

```go
KnowledgeObservationLogMax int `yaml:"knowledge_observation_log_max"`
```

Also add a default in `validateMisc()`:

```go
if b.KnowledgeObservationLogMax == 0 {
    b.KnowledgeObservationLogMax = 32
}
```

- [ ] **Step 2: Write the failing test**

Append to `internal/knowledge/knowledge_test.go`:

```go
func TestRecordObservation_BoundedFIFO(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil; observationLogMaxForTest = nil }()
	r := uint64(1000)
	roundForTest = func() uint64 { r++; return r }

	// Force a small bound for the test.
	observationLogMaxForTest = func() int { return 4 }

	for i := 1; i <= 6; i++ {
		RecordObservation(99, PlayerSubject(17), 100+i)
	}

	rec := Get(99, PlayerSubject(17))
	if rec == nil {
		t.Fatalf("expected record")
	}
	if len(rec.Observations) != 4 {
		t.Fatalf("expected bounded log of 4, got %d", len(rec.Observations))
	}
	// Should hold the LAST 4 entries (rooms 103..106).
	wantRooms := []int{103, 104, 105, 106}
	for i, o := range rec.Observations {
		if o.Room != wantRooms[i] {
			t.Errorf("entry %d: room %d, want %d", i, o.Room, wantRooms[i])
		}
	}
}

func TestRecordObservation_SameRoundDedup(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 500 }

	RecordObservation(99, PlayerSubject(17), 462)
	RecordObservation(99, PlayerSubject(17), 462) // same room+round

	rec := Get(99, PlayerSubject(17))
	if len(rec.Observations) != 1 {
		t.Errorf("expected dedup at same round, got %d entries", len(rec.Observations))
	}
}
```

- [ ] **Step 3: Implement RecordObservation + observationLogMax helper**

Append to `internal/knowledge/knowledge.go`:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/configs"
)

// Test-only override for the log cap.
var observationLogMaxForTest func() int

func observationLogMax() int {
	if observationLogMaxForTest != nil {
		return observationLogMaxForTest()
	}
	return configs.GetBalanceConfig().KnowledgeObservationLogMax
}

func RecordObservation(observerMobId int, subject Subject, room int) {
	fc := loadOrLazyInit(observerMobId, observerNameFor(observerMobId))
	now := currentRound()
	maxLog := observationLogMax()

	knowledgeCacheMu.Lock()
	r := findRecord(fc, subject)
	if r == nil {
		r = &Record{
			Subject:      subject,
			HasMet:       true,
			Source:       SourceWitnessed,
			Confidence:   ConfidenceHigh,
			LearnedRound: now,
		}
		fc.Records = append(fc.Records, r)
	}

	// Same-round dedup at tail.
	if n := len(r.Observations); n > 0 {
		tail := r.Observations[n-1]
		if tail.Room == room && tail.Round == now {
			r.LastSeenRoom = room
			r.LastSeenRound = now
			r.LastUpdatedRound = now
			knowledgeCacheMu.Unlock()
			return
		}
	}

	r.Observations = append(r.Observations, Observation{Room: room, Round: now})
	if maxLog > 0 && len(r.Observations) > maxLog {
		r.Observations = r.Observations[len(r.Observations)-maxLog:]
	}
	r.LastSeenRoom = room
	r.LastSeenRound = now
	r.LastUpdatedRound = now
	knowledgeCacheMu.Unlock()

	if err := saveObserverFile(fc); err != nil {
		mudlog.Warn("knowledge.RecordObservation: save failed", "observer", observerMobId, "error", err)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/knowledge/... ./internal/configs/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/knowledge/ internal/configs/config.balance.misc.go internal/configs/config.balance.go
git commit -m "feat(knowledge): RecordObservation + KnowledgeObservationLogMax (T6)"
```

---

## Task 7: RecordName + RecordCrimeWitnessed

**Files:**
- Modify: `internal/knowledge/knowledge.go`
- Modify: `internal/knowledge/knowledge_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestRecordName(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	RecordName(99, PlayerSubject(17), "Bob", SourceWitnessed)
	r := Get(99, PlayerSubject(17))
	if r.NameLearned != "Bob" {
		t.Errorf("name not set: got %q", r.NameLearned)
	}

	// Idempotent on same value.
	RecordName(99, PlayerSubject(17), "Bob", SourceWitnessed)
	if r.NameLearned != "Bob" {
		t.Errorf("name corrupted on re-write: %q", r.NameLearned)
	}
}

func TestRecordCrimeWitnessed_Dedup(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	RecordCrimeWitnessed(99, PlayerSubject(17), 5)
	RecordCrimeWitnessed(99, PlayerSubject(17), 7)
	RecordCrimeWitnessed(99, PlayerSubject(17), 5) // duplicate

	r := Get(99, PlayerSubject(17))
	if len(r.CrimesWitnessed) != 2 {
		t.Errorf("expected dedup, got %v", r.CrimesWitnessed)
	}
	want := map[int]bool{5: true, 7: true}
	for _, id := range r.CrimesWitnessed {
		if !want[id] {
			t.Errorf("unexpected crime id %d", id)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/knowledge/...`
Expected: FAIL (functions undefined).

- [ ] **Step 3: Implement**

Append to `internal/knowledge/knowledge.go`:

```go
func RecordName(observerMobId int, subject Subject, name string, source Source) {
	fc := loadOrLazyInit(observerMobId, observerNameFor(observerMobId))
	now := currentRound()

	knowledgeCacheMu.Lock()
	r := findRecord(fc, subject)
	if r == nil {
		r = &Record{
			Subject:      subject,
			HasMet:       true,
			Source:       source,
			Confidence:   ConfidenceHigh,
			LearnedRound: now,
		}
		fc.Records = append(fc.Records, r)
	}
	if r.NameLearned == name {
		knowledgeCacheMu.Unlock()
		return
	}
	r.NameLearned = name
	r.LastUpdatedRound = now
	knowledgeCacheMu.Unlock()

	if err := saveObserverFile(fc); err != nil {
		mudlog.Warn("knowledge.RecordName: save failed", "observer", observerMobId, "error", err)
	}
}

func RecordCrimeWitnessed(observerMobId int, subject Subject, crimeId int) {
	fc := loadOrLazyInit(observerMobId, observerNameFor(observerMobId))
	now := currentRound()

	knowledgeCacheMu.Lock()
	r := findRecord(fc, subject)
	if r == nil {
		r = &Record{
			Subject:      subject,
			HasMet:       true,
			Source:       SourceWitnessed,
			Confidence:   ConfidenceHigh,
			LearnedRound: now,
		}
		fc.Records = append(fc.Records, r)
	}
	for _, existing := range r.CrimesWitnessed {
		if existing == crimeId {
			knowledgeCacheMu.Unlock()
			return
		}
	}
	r.CrimesWitnessed = append(r.CrimesWitnessed, crimeId)
	r.LastUpdatedRound = now
	knowledgeCacheMu.Unlock()

	if err := saveObserverFile(fc); err != nil {
		mudlog.Warn("knowledge.RecordCrimeWitnessed: save failed", "observer", observerMobId, "error", err)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/knowledge/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/knowledge/
git commit -m "feat(knowledge): RecordName + RecordCrimeWitnessed (T7)"
```

---

## Task 8: Forget + ForgetFact

**Files:**
- Modify: `internal/knowledge/knowledge.go`
- Modify: `internal/knowledge/knowledge_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestForget_DropsRecord(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	RecordMet(99, PlayerSubject(17), 462, SourceWitnessed)
	RecordMet(99, PlayerSubject(18), 463, SourceWitnessed)

	Forget(99, PlayerSubject(17))

	if Get(99, PlayerSubject(17)) != nil {
		t.Errorf("Forget did not drop record for subject 17")
	}
	if Get(99, PlayerSubject(18)) == nil {
		t.Errorf("Forget should have left subject 18 alone")
	}
}

func TestForgetFact(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	RecordMet(99, PlayerSubject(17), 462, SourceWitnessed)
	RecordName(99, PlayerSubject(17), "Bob", SourceWitnessed)
	RecordObservation(99, PlayerSubject(17), 463)
	RecordCrimeWitnessed(99, PlayerSubject(17), 5)

	ForgetFact(99, PlayerSubject(17), "name")
	r := Get(99, PlayerSubject(17))
	if r.NameLearned != "" {
		t.Errorf("expected name cleared, got %q", r.NameLearned)
	}
	if len(r.Observations) == 0 || len(r.CrimesWitnessed) == 0 {
		t.Errorf("ForgetFact name should not touch observations/crimes")
	}

	ForgetFact(99, PlayerSubject(17), "observations")
	r = Get(99, PlayerSubject(17))
	if len(r.Observations) != 0 {
		t.Errorf("expected observations cleared, got %d", len(r.Observations))
	}

	ForgetFact(99, PlayerSubject(17), "crimes")
	r = Get(99, PlayerSubject(17))
	if len(r.CrimesWitnessed) != 0 {
		t.Errorf("expected crimes cleared, got %d", len(r.CrimesWitnessed))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/knowledge/...`
Expected: FAIL.

- [ ] **Step 3: Implement**

Append to `internal/knowledge/knowledge.go`:

```go
func Forget(observerMobId int, subject Subject) {
	fc := loadOrLazyInit(observerMobId, observerNameFor(observerMobId))

	knowledgeCacheMu.Lock()
	mutated := false
	for i, r := range fc.Records {
		if r.Subject == subject {
			fc.Records = append(fc.Records[:i], fc.Records[i+1:]...)
			mutated = true
			break
		}
	}
	knowledgeCacheMu.Unlock()

	if mutated {
		if err := saveObserverFile(fc); err != nil {
			mudlog.Warn("knowledge.Forget: save failed", "observer", observerMobId, "error", err)
		}
	}
}

func ForgetFact(observerMobId int, subject Subject, fact string) {
	fc := loadOrLazyInit(observerMobId, observerNameFor(observerMobId))
	now := currentRound()

	knowledgeCacheMu.Lock()
	r := findRecord(fc, subject)
	if r == nil {
		knowledgeCacheMu.Unlock()
		return
	}
	switch fact {
	case "name":
		r.NameLearned = ""
	case "observations":
		r.Observations = nil
	case "crimes":
		r.CrimesWitnessed = nil
	default:
		// Unknown fact: no-op.
		knowledgeCacheMu.Unlock()
		return
	}
	r.LastUpdatedRound = now
	knowledgeCacheMu.Unlock()

	if err := saveObserverFile(fc); err != nil {
		mudlog.Warn("knowledge.ForgetFact: save failed", "observer", observerMobId, "error", err)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/knowledge/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/knowledge/
git commit -m "feat(knowledge): Forget + ForgetFact (T8)"
```

---

## Task 9: Read API — HasMet, NameOf, LastSeen

**Files:**
- Modify: `internal/knowledge/knowledge.go`
- Modify: `internal/knowledge/knowledge_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestReadAPIs(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	if HasMet(99, PlayerSubject(17)) {
		t.Errorf("HasMet should be false for unknown subject")
	}
	if _, ok := NameOf(99, PlayerSubject(17)); ok {
		t.Errorf("NameOf should return ok=false for unknown")
	}
	if _, _, ok := LastSeen(99, PlayerSubject(17)); ok {
		t.Errorf("LastSeen should return ok=false for unknown")
	}

	RecordMet(99, PlayerSubject(17), 462, SourceWitnessed)
	RecordName(99, PlayerSubject(17), "Bob", SourceWitnessed)

	if !HasMet(99, PlayerSubject(17)) {
		t.Errorf("HasMet should be true after RecordMet")
	}
	if name, ok := NameOf(99, PlayerSubject(17)); !ok || name != "Bob" {
		t.Errorf("NameOf: got %q ok=%v", name, ok)
	}
	if room, round, ok := LastSeen(99, PlayerSubject(17)); !ok || room != 462 || round != 100 {
		t.Errorf("LastSeen: got room=%d round=%d ok=%v", room, round, ok)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/knowledge/...`
Expected: FAIL (functions undefined).

- [ ] **Step 3: Implement**

Append to `internal/knowledge/knowledge.go`:

```go
func HasMet(observerMobId int, subject Subject) bool {
	r := Get(observerMobId, subject)
	return r != nil && r.HasMet
}

func NameOf(observerMobId int, subject Subject) (string, bool) {
	r := Get(observerMobId, subject)
	if r == nil || r.NameLearned == "" {
		return "", false
	}
	return r.NameLearned, true
}

func LastSeen(observerMobId int, subject Subject) (int, uint64, bool) {
	r := Get(observerMobId, subject)
	if r == nil || r.LastSeenRound == 0 {
		return 0, 0, false
	}
	return r.LastSeenRoom, r.LastSeenRound, true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/knowledge/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/knowledge/
git commit -m "feat(knowledge): HasMet/NameOf/LastSeen read API (T9)"
```

---

## Task 10: FrequentedRooms top-K

**Files:**
- Create: `internal/knowledge/decay.go`
- Create: `internal/knowledge/decay_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/knowledge/decay_test.go
package knowledge

import (
	"testing"
)

func TestFrequentedRooms_Tally(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	r := uint64(0)
	roundForTest = func() uint64 { r++; return r }

	for _, room := range []int{462, 462, 463, 462, 464, 463, 462} {
		RecordObservation(99, PlayerSubject(17), room)
	}

	got := FrequentedRooms(99, PlayerSubject(17), 3)
	if len(got) != 3 {
		t.Fatalf("expected 3 rooms, got %d", len(got))
	}
	// Counts: 462=4, 463=2, 464=1
	if got[0].Room != 462 || got[0].Count != 4 {
		t.Errorf("top: got room=%d count=%d", got[0].Room, got[0].Count)
	}
	if got[1].Room != 463 || got[1].Count != 2 {
		t.Errorf("second: got room=%d count=%d", got[1].Room, got[1].Count)
	}
	if got[2].Room != 464 || got[2].Count != 1 {
		t.Errorf("third: got room=%d count=%d", got[2].Room, got[2].Count)
	}
}

func TestFrequentedRooms_TopKLargerThanUnique(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	r := uint64(0)
	roundForTest = func() uint64 { r++; return r }

	RecordObservation(99, PlayerSubject(17), 462)
	RecordObservation(99, PlayerSubject(17), 463)

	got := FrequentedRooms(99, PlayerSubject(17), 10)
	if len(got) != 2 {
		t.Errorf("expected 2 unique rooms, got %d", len(got))
	}
}

func TestFrequentedRooms_NoRecord(t *testing.T) {
	resetCache()
	got := FrequentedRooms(99, PlayerSubject(17), 5)
	if len(got) != 0 {
		t.Errorf("expected empty result for unknown subject, got %v", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/knowledge/...`
Expected: FAIL (`FrequentedRooms` undefined).

- [ ] **Step 3: Implement**

```go
// internal/knowledge/decay.go
package knowledge

import "sort"

type RoomCount struct {
	Room  int
	Count int
}

// FrequentedRooms returns the top-K rooms in this observer's bounded
// observation log of the subject, sorted by count descending. Stable
// secondary sort by room id (for deterministic output across ties).
func FrequentedRooms(observerMobId int, subject Subject, topK int) []RoomCount {
	r := Get(observerMobId, subject)
	if r == nil || len(r.Observations) == 0 {
		return nil
	}

	tally := make(map[int]int, 8)
	for _, o := range r.Observations {
		tally[o.Room]++
	}

	rows := make([]RoomCount, 0, len(tally))
	for room, count := range tally {
		rows = append(rows, RoomCount{Room: room, Count: count})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Count != rows[j].Count {
			return rows[i].Count > rows[j].Count
		}
		return rows[i].Room < rows[j].Room
	})

	if topK > 0 && len(rows) > topK {
		rows = rows[:topK]
	}
	return rows
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/knowledge/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/knowledge/decay.go internal/knowledge/decay_test.go
git commit -m "feat(knowledge): FrequentedRooms top-K query (T10)"
```

---

## Task 11: WitnessedCrimes lazy join with 1.3

**Files:**
- Modify: `internal/knowledge/knowledge.go`
- Modify: `internal/knowledge/knowledge_test.go`

- [ ] **Step 1: Write failing test**

```go
import (
	"github.com/GoMudEngine/GoMud/internal/crimes"
)

func TestWitnessedCrimes_LazyJoin(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	// Seed 1.3 with two crimes: one unresolved, one resolved.
	mockMob := &mobs.Mob{} // minimal — Record signature requires *mobs.Mob
	mockMob.MobId = 100
	ids := crimes.Record([]string{"thornwall_citizens"},
		crimes.KindAssault,
		crimes.Perpetrator{Type: crimes.PerpPlayer, Id: 17},
		mockMob, 1, 462, "Thornwall City", false)
	if len(ids) != 1 {
		t.Fatalf("expected 1 crime id, got %d", len(ids))
	}
	crimeIdA := ids[0]

	ids = crimes.Record([]string{"thornwall_citizens"},
		crimes.KindMurder,
		crimes.Perpetrator{Type: crimes.PerpPlayer, Id: 17},
		mockMob, 2, 463, "Thornwall City", false)
	crimeIdB := ids[0]
	crimes.Resolve("thornwall_citizens", crimeIdB, "fine_paid")

	// Knowledge layer references both.
	RecordCrimeWitnessed(99, PlayerSubject(17), crimeIdA)
	RecordCrimeWitnessed(99, PlayerSubject(17), crimeIdB)

	got := WitnessedCrimes(99, PlayerSubject(17))
	if len(got) != 2 {
		t.Fatalf("expected 2 crimes returned, got %d", len(got))
	}
	for _, w := range got {
		switch w.CrimeId {
		case crimeIdA:
			if w.ResolvedRound != 0 {
				t.Errorf("crime %d should be unresolved", w.CrimeId)
			}
		case crimeIdB:
			if w.ResolvedRound == 0 {
				t.Errorf("crime %d should be resolved", w.CrimeId)
			}
		default:
			t.Errorf("unexpected crime id %d", w.CrimeId)
		}
	}
}
```

Note: this test depends on `crimes.Record` taking the post-T1.3 signature (8 params). The implementer should verify the imports and adjust if `crimes` test setup needs a faction definition seeded; see `internal/crimes/test_main_test.go` for the pattern.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/knowledge/...`
Expected: FAIL (`WitnessedCrimes` undefined).

- [ ] **Step 3: Implement**

Append to `internal/knowledge/knowledge.go`:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/crimes"
)

type WitnessedCrime struct {
	CrimeId       int
	Kind          crimes.Kind
	ResolvedRound uint64
}

// WitnessedCrimes returns the joined view of this observer's witnessed
// crime IDs against the 1.3 substrate. Lazy-filter-on-read: callers
// see all referenced crimes plus their current resolved status, and
// decide whether to surface stale entries.
//
// The 1.3 lookup walks all known factions because the knowledge layer
// doesn't store which faction the crime belongs to. For v1 this is
// fine — the underlying crimes.AllForFaction call walks an in-memory
// cache. If this becomes hot, a follow-on can store the faction id
// alongside the crime id on the knowledge record.
func WitnessedCrimes(observerMobId int, subject Subject) []WitnessedCrime {
	r := Get(observerMobId, subject)
	if r == nil || len(r.CrimesWitnessed) == 0 {
		return nil
	}
	want := make(map[int]bool, len(r.CrimesWitnessed))
	for _, id := range r.CrimesWitnessed {
		want[id] = true
	}

	out := make([]WitnessedCrime, 0, len(r.CrimesWitnessed))
	seen := make(map[int]bool, len(r.CrimesWitnessed))

	// Walk every loaded faction's crimes, keeping rows whose IDs we know.
	// We can't list all factions cheaply from the crimes pkg today, so
	// expose a public helper there or iterate via factions.AllDefinitions().
	for _, def := range factions.AllDefinitions() {
		for _, c := range crimes.AllForFaction(def.FactionId, true /*includeResolved*/) {
			if !want[c.Id] || seen[c.Id] {
				continue
			}
			seen[c.Id] = true
			out = append(out, WitnessedCrime{
				CrimeId:       c.Id,
				Kind:          c.Kind,
				ResolvedRound: c.ResolvedRound,
			})
		}
	}
	return out
}
```

(Add `"github.com/GoMudEngine/GoMud/internal/factions"` to imports.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/knowledge/... ./internal/crimes/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/knowledge/
git commit -m "feat(knowledge): WitnessedCrimes lazy join with 1.3 (T11)"
```

---

## Task 12: Bulk read helpers + observerNameFor wiring

**Files:**
- Modify: `internal/knowledge/knowledge.go`

- [ ] **Step 1: Wire observerNameFor to mobs package**

The default stub in T5 returned `""`. Replace with a proper lookup so production code uses the right filename. Open `internal/knowledge/knowledge.go` and replace the `observerNameFor` block:

```go
// observerNameFor returns the mob template's display name for filename
// purposes. Tests can override via `observerNameFor = func(int) string {...}`.
var observerNameFor = func(mobId int) string {
	if spec := mobs.GetMobSpec(mobs.MobId(mobId)); spec != nil {
		return spec.Character.Name
	}
	return ""
}
```

(Add `"github.com/GoMudEngine/GoMud/internal/mobs"` to imports.)

- [ ] **Step 2: Implement bulk readers**

Append to `internal/knowledge/knowledge.go`:

```go
// AllForObserver returns a snapshot copy of every record this observer
// holds. Returns nil if the observer has no records.
func AllForObserver(observerMobId int) []*Record {
	fc := loadOrLazyInit(observerMobId, observerNameFor(observerMobId))
	knowledgeCacheMu.RLock()
	defer knowledgeCacheMu.RUnlock()
	if len(fc.Records) == 0 {
		return nil
	}
	out := make([]*Record, len(fc.Records))
	copy(out, fc.Records)
	return out
}

// AllObserversOfPlayer returns the in-memory observer mob IDs that hold
// any record about this player. Best-effort: walks only loaded files.
func AllObserversOfPlayer(userId int) []int {
	subj := PlayerSubject(userId)
	knowledgeCacheMu.RLock()
	defer knowledgeCacheMu.RUnlock()
	out := make([]int, 0)
	for mobId, fc := range knowledgeCache {
		for _, r := range fc.Records {
			if r.Subject == subj {
				out = append(out, mobId)
				break
			}
		}
	}
	return out
}
```

- [ ] **Step 3: Verify build clean**

Run: `go build ./...`
Expected: clean.

Run: `go test ./internal/knowledge/...`
Expected: existing tests still pass.

- [ ] **Step 4: Commit**

```bash
git add internal/knowledge/
git commit -m "feat(knowledge): bulk readers + observerNameFor wired to mobs (T12)"
```

---

## Task 13: RoutineObservers helper

**Files:**
- Modify: `internal/knowledge/knowledge.go`
- Modify: `internal/knowledge/knowledge_test.go`

- [ ] **Step 1: Write failing test**

```go
import (
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestRoutineObservers_WritesForOthers(t *testing.T) {
	resetCache()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	// Set up a room with three mobs: a moving forager (template id 50),
	// a citizen observer (template id 99), a guard observer (template id 92).
	// Test fixture details depend on internal/rooms test helpers; copy
	// from internal/hooks tests if needed.
	room := rooms.LoadRoom(1)
	if room == nil {
		t.Fatalf("test fixture room missing")
	}
	movingInst := 1001
	observer1Inst := 1002
	observer2Inst := 1003

	// Seed instances. Use `mobs.NewMobByIdFresh(...)` per project SOP.
	moving := mobs.NewMobByIdFresh(50)
	observer1 := mobs.NewMobByIdFresh(99)
	observer2 := mobs.NewMobByIdFresh(92)
	if moving == nil || observer1 == nil || observer2 == nil {
		t.Skip("test fixture mob templates 50/99/92 not registered")
	}
	moving.InstanceId = movingInst
	observer1.InstanceId = observer1Inst
	observer2.InstanceId = observer2Inst

	room.AddMob(movingInst)
	room.AddMob(observer1Inst)
	room.AddMob(observer2Inst)
	defer func() {
		room.RemoveMob(movingInst)
		room.RemoveMob(observer1Inst)
		room.RemoveMob(observer2Inst)
	}()

	RecordRoutineObservers(movingInst, 50, room)

	// observer1 and observer2 should each have a record about mob 50.
	if Get(99, MobSubject(50)) == nil {
		t.Errorf("observer 99 should have a record about moving mob 50")
	}
	if Get(92, MobSubject(50)) == nil {
		t.Errorf("observer 92 should have a record about moving mob 50")
	}
	// The moving mob itself should NOT have written a record about itself.
	if Get(50, MobSubject(50)) != nil {
		t.Errorf("moving mob should not record itself")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/knowledge/...`
Expected: FAIL (`RecordRoutineObservers` undefined). May skip if fixture mobs aren't available — that's acceptable; mark as known limitation and verify with an integration test in T15.

- [ ] **Step 3: Implement**

Append to `internal/knowledge/knowledge.go`:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// RecordRoutineObservers writes a knowledge observation for every other
// NPC mob currently in the room, treating the moving entity as the
// subject. Used by the forager/caravan room-change hook.
func RecordRoutineObservers(movingInstanceId int, movingTemplateId int, room *rooms.Room) {
	if room == nil {
		return
	}
	subject := MobSubject(movingTemplateId)
	for _, otherInstId := range room.GetMobs() {
		if otherInstId == movingInstanceId {
			continue
		}
		other := mobs.GetInstance(otherInstId)
		if other == nil {
			continue
		}
		RecordObservation(int(other.MobId), subject, room.RoomId)
		RecordMet(int(other.MobId), subject, room.RoomId, SourceWitnessed)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass (or are skipped cleanly)**

Run: `go test ./internal/knowledge/...`
Expected: PASS or SKIP for fixture-dependent test; build clean.

- [ ] **Step 5: Commit**

```bash
git add internal/knowledge/
git commit -m "feat(knowledge): RecordRoutineObservers helper (T13)"
```

---

## Task 14: Hook listener for forager/caravan room change

**Files:**
- Create: `internal/hooks/MobRoomChange_KnowledgeObservers.go`
- Modify: `internal/hooks/hooks.go`
- Create: `internal/hooks/MobRoomChange_KnowledgeObservers_test.go`

- [ ] **Step 1: Inspect mob archetype detection**

Find how foragers and caravans are detected:
- Foragers: search `internal/forager/` for `IsForager` or similar; if absent, check for a flag like `Spec.IsForager`. The `forager_state` MiscData key may be the most reliable signal.
- Caravans: search `internal/caravan/` for an analogous discriminator. Caravan mobs may have a tag like `caravan-wagon` in their `Groups` slice or a behavior tree archetype.

If no clean discriminator exists, add a helper to each package: `forager.IsForagerMob(mob *mobs.Mob) bool` and `caravan.IsCaravanMob(mob *mobs.Mob) bool`. ~5 lines each.

- [ ] **Step 2: Write the failing integration test**

```go
// internal/hooks/MobRoomChange_KnowledgeObservers_test.go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestMobRoomChange_KnowledgeObservers_ForagerTriggers(t *testing.T) {
	// Set up a room with a forager mob and an observer mob.
	// The forager moves into the room (RoomChange event fires).
	// The observer's knowledge of the forager should be updated.

	room := rooms.LoadRoom(1)
	if room == nil {
		t.Fatal("fixture room missing")
	}

	// Stand up a forager (template id assumed to map to a forager template
	// in your test fixtures). If this test cannot find a real forager
	// template, mark as a known limitation and rely on the smoke test.
	foragerTemplateId := 50 // adjust to a real forager template id
	observerTemplateId := 99

	forager := mobs.NewMobByIdFresh(mobs.MobId(foragerTemplateId))
	observer := mobs.NewMobByIdFresh(mobs.MobId(observerTemplateId))
	if forager == nil || observer == nil {
		t.Skip("forager or observer template not registered in test fixtures")
	}
	forager.InstanceId = 5001
	observer.InstanceId = 5002

	room.AddMob(observer.InstanceId)
	defer room.RemoveMob(observer.InstanceId)

	// Fire the event the listener consumes.
	evt := events.RoomChange{
		MobInstanceId: forager.InstanceId,
		FromRoomId:    0,
		ToRoomId:      room.RoomId,
	}
	MobRoomChangeKnowledgeObservers(evt)

	if knowledge.Get(observerTemplateId, knowledge.MobSubject(foragerTemplateId)) == nil {
		t.Errorf("observer should have a knowledge record of the forager")
	}
}

func TestMobRoomChange_KnowledgeObservers_NonForagerSilent(t *testing.T) {
	room := rooms.LoadRoom(1)
	if room == nil {
		t.Fatal("fixture room missing")
	}

	// A non-forager non-caravan mob moves. No knowledge writes.
	regularTemplateId := 100
	observerTemplateId := 92
	regular := mobs.NewMobByIdFresh(mobs.MobId(regularTemplateId))
	observer := mobs.NewMobByIdFresh(mobs.MobId(observerTemplateId))
	if regular == nil || observer == nil {
		t.Skip("template not registered")
	}
	regular.InstanceId = 5101
	observer.InstanceId = 5102

	room.AddMob(observer.InstanceId)
	defer room.RemoveMob(observer.InstanceId)

	evt := events.RoomChange{
		MobInstanceId: regular.InstanceId,
		ToRoomId:      room.RoomId,
	}
	MobRoomChangeKnowledgeObservers(evt)

	if knowledge.Get(observerTemplateId, knowledge.MobSubject(regularTemplateId)) != nil {
		t.Errorf("non-forager mob should not trigger knowledge writes")
	}
}
```

- [ ] **Step 3: Implement the listener**

```go
// internal/hooks/MobRoomChange_KnowledgeObservers.go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// MobRoomChangeKnowledgeObservers fires when a mob enters a new room.
// If the mob is a forager or caravan, every other NPC mob in the room
// records an observation. Hot-path filter excludes player room changes
// and non-forager/non-caravan mobs cheaply.
func MobRoomChangeKnowledgeObservers(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok {
		return events.Continue
	}
	if evt.MobInstanceId == 0 {
		// Player movement — not relevant to v1 knowledge auto-write.
		return events.Continue
	}
	mob := mobs.GetInstance(evt.MobInstanceId)
	if mob == nil {
		return events.Continue
	}
	if !forager.IsForagerMob(mob) && !caravan.IsCaravanMob(mob) {
		return events.Continue
	}
	room := rooms.LoadRoom(evt.ToRoomId)
	if room == nil {
		return events.Continue
	}
	knowledge.RecordRoutineObservers(evt.MobInstanceId, int(mob.MobId), room)
	return events.Continue
}
```

(If `IsForagerMob` / `IsCaravanMob` don't exist, add them in their respective packages first — ~5 lines each.)

- [ ] **Step 4: Register the listener**

In `internal/hooks/hooks.go`, locate `RegisterListeners()` and add:

```go
events.RegisterListener(events.RoomChange{}, MobRoomChangeKnowledgeObservers)
```

- [ ] **Step 5: Run tests + verify build**

Run: `go test ./internal/hooks/... ./internal/knowledge/...`
Expected: PASS (or SKIP cleanly for fixture-dependent tests).

Run: `go build ./...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/ internal/forager/ internal/caravan/
git commit -m "feat(knowledge): forager/caravan room-change listener (T14)"
```

---

## Task 15: Crime witnessing — attack.go integration

**Files:**
- Modify: `internal/usercommands/attack.go`
- Create: `internal/usercommands/attack_knowledge_test.go`

- [ ] **Step 1: Write the failing integration test**

```go
// internal/usercommands/attack_knowledge_test.go
package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestRecordAssaultCrime_WritesWitnessKnowledge(t *testing.T) {
	// Stand up a player + victim mob (faction-aligned) + witness mob.
	// Call recordAssaultCrime. Witness's knowledge of the player should
	// include the new assault crime ID.
	// Heavy fixture; copy setup from existing internal/usercommands tests
	// that exercise the crime path.

	t.Skip("integration test stub — fill in once attack_test.go fixture pattern is in scope")
}
```

(This task is **integration-light** because the actual assertion is in the smoke test. The unit-level test stub captures intent; full coverage lands in the smoke test goal file.)

- [ ] **Step 2: Read the existing recordAssaultCrime function**

Open `internal/usercommands/attack.go` around line 319-338 (the function defined in T1.3). Confirm signature:

```go
func recordAssaultCrime(user *users.UserRecord, mob *mobs.Mob, room *rooms.Room) {
    factionIds := factions.FactionsForMob(mob)
    if len(factionIds) == 0 {
        return
    }
    witnesses := crimes.WitnessesInRoom(factionIds, room, 0)
    perp := crimes.IdentifiedPerp(user.UserId, witnesses)
    delta := int(configs.GetBalanceConfig().CrimeRepDeltaAssault)
    for _, fid := range factionIds {
        crimes.Record([]string{fid}, crimes.KindAssault, perp,
            mob, mob.InstanceId, room.RoomId, mob.Character.Zone, /*hadExternal*/ ...)
        ...
    }
}
```

- [ ] **Step 3: Add knowledge writes**

Modify `recordAssaultCrime` to capture returned crime IDs and write knowledge per witness when the perp is a player. Insert after the `crimes.Record` call inside the loop:

```go
import (
    "github.com/GoMudEngine/GoMud/internal/knowledge"
)

// ... inside recordAssaultCrime, replacing the existing Record call:
crimeIds := crimes.Record([]string{fid}, crimes.KindAssault, perp,
    mob, mob.InstanceId, room.RoomId, mob.Character.Zone, hadExternal)
if perp.Type == crimes.PerpPlayer {
    factions.BumpRep(fid, user.UserId, delta)
    // Knowledge: each witness records the player as the perp of these crimes.
    subject := knowledge.PlayerSubject(user.UserId)
    for _, witnessInstId := range witnesses {
        w := mobs.GetInstance(witnessInstId)
        if w == nil {
            continue
        }
        for _, crimeId := range crimeIds {
            knowledge.RecordCrimeWitnessed(int(w.MobId), subject, crimeId)
        }
        knowledge.RecordMet(int(w.MobId), subject, room.RoomId, knowledge.SourceWitnessed)
    }
}
```

(If the existing T1.3 implementation differs in detail, preserve its structure and only add the knowledge-write block.)

- [ ] **Step 4: Build + existing test suite**

Run: `go build ./...`
Expected: clean.

Run: `go test ./internal/usercommands/...`
Expected: existing tests still pass.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/attack.go internal/usercommands/attack_knowledge_test.go
git commit -m "feat(knowledge): write witness knowledge on assault crime (T15)"
```

---

## Task 16: Crime witnessing — MobDeath_FactionRep integration

**Files:**
- Modify: `internal/hooks/MobDeath_FactionRep.go`

- [ ] **Step 1: Read the existing handler**

Open `internal/hooks/MobDeath_FactionRep.go`. Identify the four cases (A, B, C, fresh-murder) from chunk 1.3 — they're the four branches inside the witness-aware loop.

- [ ] **Step 2: Add knowledge writes at all four branches**

Each branch eventually identifies the perpetrator (player or unknown) and the resulting crime ID. After the substrate writes, add knowledge writes for each witness when perp is a player.

Insert after each `UpgradeAssaultToMurder` and after the fresh `crimes.Record` call. Use a shared helper inside the same file to keep the code DRY:

```go
import (
    "github.com/GoMudEngine/GoMud/internal/knowledge"
)

// writeKnowledgeForWitnesses writes a knowledge record for each witness
// linking them to the given crime IDs and the player perp. Skips when
// perp is not a player.
func writeKnowledgeForWitnesses(witnesses []int, perp crimes.Perpetrator,
    crimeIds []int, roomId int) {
    if perp.Type != crimes.PerpPlayer {
        return
    }
    subject := knowledge.PlayerSubject(perp.Id)
    for _, witnessInstId := range witnesses {
        w := mobs.GetInstance(witnessInstId)
        if w == nil {
            continue
        }
        for _, crimeId := range crimeIds {
            knowledge.RecordCrimeWitnessed(int(w.MobId), subject, crimeId)
        }
        knowledge.RecordMet(int(w.MobId), subject, roomId, knowledge.SourceWitnessed)
    }
}
```

In the upgrade branches (A/B/C):
- The crime ID is `assault.Id` (the row that was upgraded).
- Pass `[]int{assault.Id}` to the helper.

In the fresh-murder branch:
- `crimes.Record(...)` returns the new crime IDs. Capture them and pass to the helper.

Example for the fresh-murder branch:

```go
crimeIds := crimes.Record([]string{fid}, crimes.KindMurder, perp,
    spec, evt.InstanceId, evt.RoomId, spec.Zone, /*hadExternal*/ false)
if perp.Type == crimes.PerpPlayer {
    factions.BumpRep(fid, userId, deltaMurder)
}
writeKnowledgeForWitnesses(witnesses, perp, crimeIds, evt.RoomId)
```

For the upgrade branches, the crime ID is already known:

```go
crimes.UpgradeAssaultToMurder(fid, assault.Id, perp, ...)
// rep delta logic ...
writeKnowledgeForWitnesses(witnesses, perp, []int{assault.Id}, evt.RoomId)
```

(Case C with perp=unknown will fall through `writeKnowledgeForWitnesses`'s early-return guard.)

- [ ] **Step 3: Verify build + existing tests**

Run: `go build ./... && go test ./internal/hooks/...`
Expected: clean / PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/MobDeath_FactionRep.go
git commit -m "feat(knowledge): write witness knowledge on murder crime (T16)"
```

---

## Task 17: Crime witnessing — failed-steal integration

**Files:**
- Modify: `internal/usercommands/skill.skullduggery.steal.go`

- [ ] **Step 1: Read the existing failed-steal path**

Open `internal/usercommands/skill.skullduggery.steal.go`. Locate the failure branch where a theft crime is recorded (the call site added in chunk 1.3 T14).

- [ ] **Step 2: Add knowledge writes for witnesses**

Right after the `crimes.Record(...)` call in the failure branch, mirror the pattern from T15:

```go
import (
    "github.com/GoMudEngine/GoMud/internal/knowledge"
)

// ... inside the failure branch, after the existing crimes.Record:
crimeIds := crimes.Record([]string{fid}, crimes.KindTheft, perp,
    mob, mob.InstanceId, room.RoomId, mob.Character.Zone, /*hadExternal*/ ...)
if perp.Type == crimes.PerpPlayer {
    factions.BumpRep(fid, user.UserId, deltaTheft)
    subject := knowledge.PlayerSubject(user.UserId)
    for _, witnessInstId := range witnesses {
        w := mobs.GetInstance(witnessInstId)
        if w == nil {
            continue
        }
        for _, crimeId := range crimeIds {
            knowledge.RecordCrimeWitnessed(int(w.MobId), subject, crimeId)
        }
        knowledge.RecordMet(int(w.MobId), subject, room.RoomId, knowledge.SourceWitnessed)
    }
}
```

- [ ] **Step 3: Verify build + existing tests**

Run: `go build ./... && go test ./internal/usercommands/...`
Expected: clean / PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/skill.skullduggery.steal.go
git commit -m "feat(knowledge): write witness knowledge on failed-steal theft (T17)"
```

---

## Task 18: Admin command

**Files:**
- Create: `internal/usercommands/admin.knowledge.go`
- Create: `_datafiles/templates/admincommands/help/command.knowledge.template`
- Modify: appropriate command-registration file (search for where `crime` and `faction` are registered).

- [ ] **Step 1: Write the admin command**

```go
// internal/usercommands/admin.knowledge.go
package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
 * Role Permissions:
 * knowledge      (Admin)
 */

func Knowledge(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		knowledgeUsage(user)
		return true, nil
	}
	switch strings.ToLower(args[0]) {
	case "show":
		return knowledgeShow(args[1:], user)
	case "frequented":
		return knowledgeFrequented(args[1:], user)
	case "forget":
		return knowledgeForget(args[1:], user)
	default:
		knowledgeUsage(user)
		return true, nil
	}
}

func knowledgeUsage(user *users.UserRecord) {
	if out, err := templates.Process("admincommands/help/command.knowledge", nil, user.UserId); err == nil && strings.TrimSpace(out) != "" {
		user.SendText(out)
		return
	}
	user.SendText(
		"Usage:\r\n" +
			"  knowledge show <mobId>\r\n" +
			"  knowledge show <mobId> <playerName>\r\n" +
			"  knowledge show <mobId> mob <targetId>\r\n" +
			"  knowledge frequented <mobId> <playerName> [topK]\r\n" +
			"  knowledge forget <mobId> <playerName> [fact]\r\n",
	)
}

func resolveMobIdFromArg(arg string) (int, bool) {
	if id, err := strconv.Atoi(arg); err == nil {
		return id, true
	}
	if id, ok := mobs.GetMobIdByName(arg); ok {
		return int(id), true
	}
	return 0, false
}

func resolveSubjectFromArgs(args []string) (knowledge.Subject, bool) {
	switch len(args) {
	case 1:
		// player name shorthand
		target := users.GetByCharacterNameOrLoad(args[0])
		if target == nil {
			return knowledge.Subject{}, false
		}
		return knowledge.PlayerSubject(target.UserId), true
	case 2:
		if strings.ToLower(args[0]) == "mob" {
			id, err := strconv.Atoi(args[1])
			if err != nil {
				return knowledge.Subject{}, false
			}
			return knowledge.MobSubject(id), true
		}
	}
	return knowledge.Subject{}, false
}

func knowledgeShow(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		knowledgeUsage(user)
		return true, nil
	}
	mobId, ok := resolveMobIdFromArg(args[0])
	if !ok {
		user.SendText(fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}

	if len(args) == 1 {
		// List all records for this observer.
		records := knowledge.AllForObserver(mobId)
		if len(records) == 0 {
			user.SendText(fmt.Sprintf("No knowledge records for mob %d.\r\n", mobId))
			return true, nil
		}
		var b strings.Builder
		fmt.Fprintf(&b, "knowledge show %d (%d records):\r\n\r\n", mobId, len(records))
		fmt.Fprintf(&b, "  %-12s  %4s  %-20s  %s\r\n", "Subject", "Met", "Name", "LastSeen")
		fmt.Fprintf(&b, "  %-12s  %4s  %-20s  %s\r\n",
			"------------", "----", "--------------------", "----------")
		sort.Slice(records, func(i, j int) bool {
			return records[i].LastSeenRound > records[j].LastSeenRound
		})
		for _, r := range records {
			subject := fmt.Sprintf("%s:%d", r.Subject.Type, r.Subject.Id)
			met := "no"
			if r.HasMet {
				met = "yes"
			}
			lastSeen := fmt.Sprintf("room %d @ round %d", r.LastSeenRoom, r.LastSeenRound)
			fmt.Fprintf(&b, "  %-12s  %4s  %-20s  %s\r\n", subject, met, r.NameLearned, lastSeen)
		}
		user.SendText(b.String())
		return true, nil
	}

	// Drill into a specific subject.
	subj, ok := resolveSubjectFromArgs(args[1:])
	if !ok {
		user.SendText(fmt.Sprintf("Unknown subject\r\n"))
		return true, nil
	}
	r := knowledge.Get(mobId, subj)
	if r == nil {
		user.SendText(fmt.Sprintf("No record: mob %d ↔ %s:%d\r\n", mobId, subj.Type, subj.Id))
		return true, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Subject: %s:%d\r\n", subj.Type, subj.Id)
	fmt.Fprintf(&b, "  HasMet: %v   Name: %q   Source: %s   Confidence: %s\r\n",
		r.HasMet, r.NameLearned, r.Source, r.Confidence)
	fmt.Fprintf(&b, "  LastSeen: room %d @ round %d\r\n", r.LastSeenRoom, r.LastSeenRound)
	fmt.Fprintf(&b, "  Observations: %d entries\r\n", len(r.Observations))
	wcs := knowledge.WitnessedCrimes(mobId, subj)
	if len(wcs) > 0 {
		fmt.Fprintf(&b, "  Witnessed crimes:\r\n")
		for _, w := range wcs {
			resolved := "no"
			if w.ResolvedRound > 0 {
				resolved = fmt.Sprintf("round %d", w.ResolvedRound)
			}
			fmt.Fprintf(&b, "    crime %d (%s) resolved: %s\r\n", w.CrimeId, w.Kind, resolved)
		}
	}
	user.SendText(b.String())
	return true, nil
}

func knowledgeFrequented(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 2 {
		knowledgeUsage(user)
		return true, nil
	}
	mobId, ok := resolveMobIdFromArg(args[0])
	if !ok {
		user.SendText(fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}
	target := users.GetByCharacterNameOrLoad(args[1])
	if target == nil {
		user.SendText(fmt.Sprintf("No such player: %s\r\n", args[1]))
		return true, nil
	}
	topK := 5
	if len(args) >= 3 {
		if v, err := strconv.Atoi(args[2]); err == nil && v > 0 {
			topK = v
		}
	}
	rows := knowledge.FrequentedRooms(mobId, knowledge.PlayerSubject(target.UserId), topK)
	if len(rows) == 0 {
		user.SendText(fmt.Sprintf("No observations recorded.\r\n"))
		return true, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Top %d rooms for mob %d watching %s:\r\n\r\n", topK, mobId, args[1])
	for _, r := range rows {
		fmt.Fprintf(&b, "  room %d  (%d sightings)\r\n", r.Room, r.Count)
	}
	user.SendText(b.String())
	return true, nil
}

func knowledgeForget(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 2 {
		knowledgeUsage(user)
		return true, nil
	}
	mobId, ok := resolveMobIdFromArg(args[0])
	if !ok {
		user.SendText(fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}
	target := users.GetByCharacterNameOrLoad(args[1])
	if target == nil {
		user.SendText(fmt.Sprintf("No such player: %s\r\n", args[1]))
		return true, nil
	}
	subj := knowledge.PlayerSubject(target.UserId)
	if len(args) == 2 {
		knowledge.Forget(mobId, subj)
		user.SendText(fmt.Sprintf("Forgot record: mob %d ↔ %s\r\n", mobId, args[1]))
		return true, nil
	}
	knowledge.ForgetFact(mobId, subj, strings.ToLower(args[2]))
	user.SendText(fmt.Sprintf("Forgot fact %q: mob %d ↔ %s\r\n", args[2], mobId, args[1]))
	return true, nil
}
```

- [ ] **Step 2: Write the help template**

```
# _datafiles/templates/admincommands/help/command.knowledge.template

<ansi fg="white-bold">knowledge</ansi> — inspect and manipulate NPC knowledge

  <ansi fg="cyan">knowledge show</ansi> <ansi fg="yellow"><mobId></ansi>
    List all knowledge records held by this observer NPC.

  <ansi fg="cyan">knowledge show</ansi> <ansi fg="yellow"><mobId> <playerName></ansi>
    Show this observer's knowledge of one player.

  <ansi fg="cyan">knowledge show</ansi> <ansi fg="yellow"><mobId> mob <targetId></ansi>
    Show this observer's knowledge of one NPC.

  <ansi fg="cyan">knowledge frequented</ansi> <ansi fg="yellow"><mobId> <playerName> [topK]</ansi>
    Top-K rooms where this observer has seen the player.

  <ansi fg="cyan">knowledge forget</ansi> <ansi fg="yellow"><mobId> <playerName> [fact]</ansi>
    Drop the entire record, or a single fact: name | observations | crimes.

<mobId> accepts a numeric template id or a mob name.
```

- [ ] **Step 3: Register the admin command**

Search for where `Crime` and `Faction` admin commands are registered (likely in `internal/usercommands/registration.go` or similar). Add:

```go
// In the admin command registration block:
RegisterAdminCommand("knowledge", Knowledge)
```

(Or follow whatever exact pattern the existing crime/faction admin commands use.)

- [ ] **Step 4: Verify build**

Run: `go build ./... && go test ./internal/usercommands/...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/admin.knowledge.go _datafiles/templates/admincommands/help/command.knowledge.template
git commit -m "feat(knowledge): admin show/forget/frequented command (T18)"
```

---

## Task 19: Roadmap update

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Mark chunk 1.4 as Done in the progress tracker**

Locate the `| 1.4 | Substrate | NPC knowledge model | M | 1.1 | Not started |` row and change `Not started` to `Done`. Update the roll-up line below the table from `3 / 40 done` to `4 / 40 done`.

- [ ] **Step 2: Update the chunk's mini-brief**

Locate the `### 1.4 NPC knowledge model` section. Change `**Status:** Not started` to `**Status:** Done (2026-05-09)`. Append a `Shipped:` paragraph mirroring the format used by 1.1, 1.2, 1.3 — list the package paths, key API entry points, integration sites, and spec/plan filenames.

Suggested template:

```markdown
- **Shipped:** `internal/knowledge/` package storing per-observer-NPC YAML at `_datafiles/world/dogmud/knowledge/{mobId}-{namesimple}.yaml` (gitignored). Polymorphic subject (`{type, id}` for player or mob template), source/confidence tier on every record, per-fact decay rules, NPC-on-NPC supported. v1 fact types: identity (HasMet + NameLearned), location (LastSeen + bounded observation log capped at `KnowledgeObservationLogMax = 32`), routine (FrequentedRooms top-K query), deeds-witnessed (crime row IDs, lazy-filtered against 1.3 on read via WitnessedCrimes). Auto-write triggers v1: forager/caravan room change (new hook listener `MobRoomChange_KnowledgeObservers` wraps `knowledge.RecordRoutineObservers`) and 1.3 crime witnessing (three call-site additions in attack.go, MobDeath_FactionRep.go, skullduggery steal). Explicit Forget / ForgetFact API for amnesia consumers. New admin command `knowledge show/forget/frequented` + helpfile. No cross-substrate cascade in v1 (documented as a deferred decision pending amnesia spell consumer). Spec at `docs/superpowers/specs/2026-05-09-mob-aliveness-1.4-knowledge-model-design.md`, plan at `docs/superpowers/plans/2026-05-09-mob-aliveness-1.4-knowledge-model.md`.
```

- [ ] **Step 3: Commit**

```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(roadmap): mark chunk 1.4 (NPC knowledge model) as Done"
```

---

## Final review

After all tasks complete, dispatch the `superpowers:code-reviewer` agent for a holistic pass. Then run a smoke test (manual or scripted via `/test-mud local feature-tester knowledge-thornwall-smoke.yaml`) before merging the feature branch.

Suggested smoke goal file: `tools/testing/goals/knowledge-thornwall-smoke.yaml` with the scenarios from the spec's Testing section (forager/caravan observation, witnessed assault knowledge write, lone-perp skip, forget op).

Branch state after completion: `feature/mob-aliveness-1.3-crimes` carries chunks 1.1, 1.2, 1.3, AND 1.4. When confidence is high, merge to development with `--no-ff`.
