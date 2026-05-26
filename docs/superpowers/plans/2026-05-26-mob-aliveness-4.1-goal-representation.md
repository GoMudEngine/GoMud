# Mob Aliveness 4.1 — Goal Representation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an `internal/goals/` substrate package with typed goals, per-mob YAML persistence, a type-metadata registry (predicates + conflicts), priority-resolved conflict policy at Add-time, and an `admin goal …` command. Ships with an empty predicate/conflict registry — 4.3 fills it. No behavior-tree integration yet (4.4's job). No observable behavior change for players.

**Architecture:** New leaf package `internal/goals/` modeled on the existing `internal/opinions/`, `internal/knowledge/`, `internal/facts/`, `internal/bounties/` stores. Per-mob YAML file at `_datafiles/world/dogmud/goals/{mobId}-{namesimple}.yaml` with a monotonic per-mob `next_goal_id` counter producing short string ids (`g1`, `g2`, …). Single package-level RWMutex guards the in-memory cache; a second mutex serializes disk writes (Windows file-lock safety). Atomic writes via `.tmp` + `os.Rename`. Admin command in `internal/usercommands/admin.goal.go` registered through the existing `userCommands` table; uses an opinion-style mob-ident resolver.

**Tech Stack:** Go 1.25 · `gopkg.in/yaml.v3` (matching knowledge/facts/bounties) · existing `mudlog`, `configs`, `util`, `messaging`, `users`, `rooms`, `events`, `mobs`, `templates` packages.

**Spec:** `docs/superpowers/specs/2026-05-26-mob-aliveness-4.1-goal-representation-design.md`

---

## File Structure

**New:**
- `internal/goals/types.go` — `Goal`, `MobGoals`, `GoalTypeMeta`, `PredicateFn`, `AddResult`, `ConflictError`
- `internal/goals/persistence.go` — directory resolution, atomic save, load-from-disk, dir override env var
- `internal/goals/registry.go` — `RegisterGoalType`, `lookupMeta`, symmetric-conflict validation, registry reset for tests
- `internal/goals/store.go` — `Add`, `Remove`, `Clear`, `GoalsOf`, `IsSatisfied`, `IsExpired`, `ClearCache`, lazy-load + conflict resolution
- `internal/goals/test_main_test.go` — temp-dir test seam (mirrors opinions/knowledge)
- `internal/goals/types_test.go`, `persistence_test.go`, `registry_test.go`, `store_test.go`
- `internal/usercommands/admin.goal.go` + `admin.goal_test.go`

**Modified:**
- `.gitignore` — add `_datafiles/world/dogmud/goals/`
- `internal/usercommands/usercommands.go` — register `goal` → `{Goal, true, true, true}` (admin-only)
- `MOB_ALIVENESS_ROADMAP.md` — flip 4.1 status to Done, bump rollup to 23/42
- `PATCH_NOTES.md` — chunk 4.1 entry
- `_datafiles/config.yaml` — confirm `Logging.LogToFile: false` for pre-push SOP

**Not touched in 4.1:** behavior-tree, mob struct, hooks, configs/balance/spawn.

---

## Task 1 — Types & YAML round-trip

The data shapes, with no logic. Sets the contract that every later task references.

**Files:**
- Create: `internal/goals/types.go`
- Create: `internal/goals/types_test.go`

- [ ] **Step 1.1: Create the failing types test**

Write `internal/goals/types_test.go`:

```go
package goals

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestGoal_YAMLRoundTrip(t *testing.T) {
	g := &Goal{
		Id:       "g1",
		Type:     "revenge",
		Priority: 70,
		Params: map[string]any{
			"target_player_name": "smoketester",
			"reason":             "killed brother",
			"observed_round":     12345,
			"intensity":          1.5,
			"public":             true,
		},
		CreatedAt: time.Date(2026, 5, 26, 14, 30, 0, 0, time.UTC),
		ExpiresAt: time.Date(2026, 6, 25, 14, 30, 0, 0, time.UTC),
	}
	out, err := yaml.Marshal(g)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Goal
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Id != g.Id || got.Type != g.Type || got.Priority != g.Priority {
		t.Fatalf("scalar mismatch: %+v vs %+v", got, g)
	}
	if !got.CreatedAt.Equal(g.CreatedAt) || !got.ExpiresAt.Equal(g.ExpiresAt) {
		t.Fatalf("time mismatch: %v / %v vs %v / %v",
			got.CreatedAt, got.ExpiresAt, g.CreatedAt, g.ExpiresAt)
	}
	if got.Params["target_player_name"] != "smoketester" {
		t.Errorf("params string lost: %v", got.Params["target_player_name"])
	}
	if got.Params["public"] != true {
		t.Errorf("params bool lost: %v", got.Params["public"])
	}
}

func TestMobGoals_YAMLRoundTrip(t *testing.T) {
	mg := &MobGoals{
		MobId:      371,
		NextGoalId: 3,
		Goals: []*Goal{
			{Id: "g1", Type: "revenge", Priority: 70, CreatedAt: time.Now().UTC()},
			{Id: "g2", Type: "wealth-target", Priority: 30, CreatedAt: time.Now().UTC()},
		},
	}
	out, err := yaml.Marshal(mg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got MobGoals
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.MobId != 371 || got.NextGoalId != 3 || len(got.Goals) != 2 {
		t.Fatalf("round trip lost data: %+v", got)
	}
	if got.Goals[0].Id != "g1" || got.Goals[1].Id != "g2" {
		t.Errorf("goal order lost: %v / %v", got.Goals[0].Id, got.Goals[1].Id)
	}
}
```

- [ ] **Step 1.2: Run test to verify it fails**

Run: `go test ./internal/goals/...`

Expected: FAIL — `Goal` and `MobGoals` undefined.

- [ ] **Step 1.3: Create `types.go` with the struct definitions**

Write `internal/goals/types.go`:

```go
// Package goals — chunk 4.1 of the mob aliveness roadmap.
//
// Provides a persistent, queryable list of typed goals per mob
// template. Ships with an empty predicate/conflict registry; chunk 4.3
// fills it with concrete goal types. No behavior-tree integration yet
// (chunk 4.4).
package goals

import (
	"errors"
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// Goal is one strategic intent owned by a mob template. Immutable once
// added — "updates" go through Remove + Add.
type Goal struct {
	Id         string         `yaml:"id"`
	OwnerMobId int            `yaml:"-"` // stamped at load time from MobGoals.MobId
	Type       string         `yaml:"type"`
	Priority   int            `yaml:"priority"`
	Params     map[string]any `yaml:"params,omitempty"`
	CreatedAt  time.Time      `yaml:"created_at"`
	ExpiresAt  time.Time      `yaml:"expires_at,omitempty"`
}

// MobGoals is the on-disk shape — one file per mob template.
type MobGoals struct {
	MobId      int     `yaml:"mob_id"`
	NextGoalId int     `yaml:"next_goal_id"`
	Goals      []*Goal `yaml:"goals"`
}

// PredicateFn evaluates whether a goal is currently satisfied. Same
// goal + same mob state → same answer. No side effects — IsSatisfied
// may be called from any context.
type PredicateFn func(g *Goal, mob *mobs.Mob) bool

// GoalTypeMeta is registered once per goal type by chunk 4.3's catalog.
type GoalTypeMeta struct {
	Predicate     PredicateFn
	ConflictsWith []string // type names this goal type conflicts with
}

// AddResult reports what happened on a successful Add.
type AddResult struct {
	Added     *Goal    // the newly-added goal (Id assigned)
	Displaced []string // goal ids removed because new one preempted
}

// ConflictError is returned by Add when an existing goal blocks the
// new one (priority ≤ existing's). Carries the blocker so the admin
// command can render "Blocked by goal <id> (type=<t>, priority=<p>)".
type ConflictError struct {
	BlockerGoalId string
	BlockerType   string
	BlockerPrio   int
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("goals: blocked by goal %s (type=%s, priority=%d)",
		e.BlockerGoalId, e.BlockerType, e.BlockerPrio)
}

// Sentinel returned by Remove when the goal id wasn't found. Tests can
// match it via errors.Is. Production callers usually ignore Remove's
// error since "remove what's not there" is a no-op.
var ErrGoalNotFound = errors.New("goals: goal id not found")
```

- [ ] **Step 1.4: Run test to verify it passes**

Run: `go test ./internal/goals/... -run YAMLRoundTrip -v`

Expected: PASS for both `TestGoal_YAMLRoundTrip` and `TestMobGoals_YAMLRoundTrip`.

- [ ] **Step 1.5: Commit**

```bash
git add internal/goals/types.go internal/goals/types_test.go
git commit -m "feat(goals): chunk 4.1 task 1 — Goal/MobGoals types + YAML round-trip

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2 — Persistence layer

Atomic write via `.tmp` + `os.Rename`. Load-from-disk that tolerates missing/corrupt files. `DOGMUD_GOALS_DIR_OVERRIDE` env-var seam so tests don't need a configs fixture. Dual mutex (cache RWMutex, save mutex) matching the established pattern.

**Files:**
- Create: `internal/goals/persistence.go`
- Create: `internal/goals/test_main_test.go`
- Create: `internal/goals/persistence_test.go`

- [ ] **Step 2.1: Create the test-main shim**

Write `internal/goals/test_main_test.go`:

```go
package goals

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain points the goals-dir override at a per-process temp dir
// before any test runs, so every test starts with a clean slate and
// no test needs a configs fixture. Mirrors opinions/test_main_test.go.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "goals-test-*")
	if err != nil {
		panic("goals test: mkdirtemp: " + err.Error())
	}
	os.Setenv("DOGMUD_GOALS_DIR_OVERRIDE", filepath.Join(dir, "goals"))
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
```

- [ ] **Step 2.2: Create the failing persistence test**

Write `internal/goals/persistence_test.go`:

```go
package goals

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func resetCache() {
	cacheMu.Lock()
	cache = map[int]*MobGoals{}
	nameByMobId = map[int]string{}
	cacheMu.Unlock()
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	resetCache()
	mg := &MobGoals{
		MobId:      371,
		NextGoalId: 2,
		Goals: []*Goal{
			{Id: "g1", Type: "revenge", Priority: 70,
				CreatedAt: time.Now().UTC().Truncate(time.Second)},
		},
	}
	cacheStoreForTest("tova", mg)
	if err := saveToDisk(371, "tova"); err != nil {
		t.Fatalf("save: %v", err)
	}
	resetCache()
	got := loadFromDisk(371, "tova")
	if got == nil {
		t.Fatal("load returned nil after save")
	}
	if got.MobId != 371 || got.NextGoalId != 2 || len(got.Goals) != 1 {
		t.Fatalf("round trip lost data: %+v", got)
	}
	if got.Goals[0].Id != "g1" {
		t.Errorf("goal id lost: %q", got.Goals[0].Id)
	}
}

func TestLoadFromDisk_MissingFile(t *testing.T) {
	resetCache()
	if got := loadFromDisk(99999, "nobody"); got != nil {
		t.Errorf("expected nil for missing file, got %+v", got)
	}
}

func TestLoadFromDisk_CorruptYAML(t *testing.T) {
	resetCache()
	path := goalPath(99998, "broken")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not: valid: yaml: ["), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := loadFromDisk(99998, "broken"); got != nil {
		t.Errorf("expected nil for corrupt file, got %+v", got)
	}
}

func TestSaveToDisk_AtomicRename(t *testing.T) {
	resetCache()
	mg := &MobGoals{MobId: 200, NextGoalId: 1, Goals: nil}
	cacheStoreForTest("phantom", mg)
	if err := saveToDisk(200, "phantom"); err != nil {
		t.Fatalf("save: %v", err)
	}
	path := goalPath(200, "phantom")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("final file missing: %v", err)
	}
	tmp := path + ".tmp"
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Errorf("expected .tmp removed after rename, stat err=%v", err)
	}
}

func TestGoalPath_UsesOverride(t *testing.T) {
	override := os.Getenv("DOGMUD_GOALS_DIR_OVERRIDE")
	if override == "" {
		t.Skip("DOGMUD_GOALS_DIR_OVERRIDE unset; TestMain misconfigured?")
	}
	got := goalPath(371, "tova")
	if !strings.HasPrefix(got, override) {
		t.Errorf("goalPath = %q, want prefix %q", got, override)
	}
}
```

- [ ] **Step 2.3: Run test to verify it fails**

Run: `go test ./internal/goals/... -run TestSaveAndLoad -v`

Expected: FAIL — `cacheMu`, `cache`, `nameByMobId`, `goalPath`, `saveToDisk`, `loadFromDisk`, `cacheStoreForTest` undefined.

- [ ] **Step 2.4: Create the persistence file**

Write `internal/goals/persistence.go`:

```go
package goals

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v3"
)

var (
	cacheMu     sync.RWMutex
	cache       = map[int]*MobGoals{}
	nameByMobId = map[int]string{}

	// saveMu serializes disk writes to avoid Windows ERROR_SHARING_VIOLATION
	// when two goroutines write the same path. Held only during marshal +
	// write, never across cache mutations.
	saveMu sync.Mutex
)

func goalsBaseDir() string {
	if override := os.Getenv("DOGMUD_GOALS_DIR_OVERRIDE"); override != "" {
		return override
	}
	return filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "goals")
}

func goalPath(mobId int, namesimple string) string {
	return filepath.Join(goalsBaseDir(),
		fmt.Sprintf("%d-%s.yaml", mobId, util.ConvertForFilename(namesimple)))
}

func loadFromDisk(mobId int, namesimple string) *MobGoals {
	path := goalPath(mobId, namesimple)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	mg := &MobGoals{}
	if err := yaml.Unmarshal(data, mg); err != nil {
		mudlog.Warn("goals.loadFromDisk", "path", path, "error", err)
		return nil
	}
	// Stamp OwnerMobId on every loaded goal — the field is unmarshal-
	// skipped (yaml:"-") so we set it here from the parent.
	for _, g := range mg.Goals {
		g.OwnerMobId = mg.MobId
	}
	return mg
}

// saveToDisk writes the cached MobGoals for mobId. Returns an error if
// the cache is missing the entry or the write fails. Atomic via
// .tmp + os.Rename.
func saveToDisk(mobId int, namesimple string) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	cacheMu.RLock()
	mg, ok := cache[mobId]
	if !ok {
		cacheMu.RUnlock()
		return fmt.Errorf("goals.saveToDisk: no cached entry for mobId=%d", mobId)
	}
	out, err := yaml.Marshal(mg)
	cacheMu.RUnlock()
	if err != nil {
		return fmt.Errorf("goals.saveToDisk: marshal: %w", err)
	}

	path := goalPath(mobId, namesimple)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("goals.saveToDisk: mkdir %s: %w", filepath.Dir(path), err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return fmt.Errorf("goals.saveToDisk: write tmp %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("goals.saveToDisk: rename %s: %w", path, err)
	}
	return nil
}

// cacheStoreForTest seeds the cache directly. Test-only seam used
// before Add() exists.
func cacheStoreForTest(namesimple string, mg *MobGoals) {
	cacheMu.Lock()
	cache[mg.MobId] = mg
	nameByMobId[mg.MobId] = namesimple
	cacheMu.Unlock()
}

// ClearCache drops every cached entry. Tests use this to isolate
// cases; production code should not call it.
func ClearCache() {
	cacheMu.Lock()
	cache = map[int]*MobGoals{}
	nameByMobId = map[int]string{}
	cacheMu.Unlock()
}
```

- [ ] **Step 2.5: Run test to verify it passes**

Run: `go test ./internal/goals/... -run "TestSaveAndLoad|TestLoadFromDisk|TestSaveToDisk|TestGoalPath" -v`

Expected: PASS for all four tests.

- [ ] **Step 2.6: Commit**

```bash
git add internal/goals/persistence.go internal/goals/persistence_test.go internal/goals/test_main_test.go
git commit -m "feat(goals): chunk 4.1 task 2 — persistence layer with atomic writes

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3 — Goal type metadata registry

`RegisterGoalType` + symmetric-conflict validation. Registry starts empty; 4.3 fills it.

**Files:**
- Create: `internal/goals/registry.go`
- Create: `internal/goals/registry_test.go`

- [ ] **Step 3.1: Create the failing registry test**

Write `internal/goals/registry_test.go`:

```go
package goals

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func resetRegistry() {
	registryMu.Lock()
	typeRegistry = map[string]GoalTypeMeta{}
	registryMu.Unlock()
}

func TestRegisterGoalType_LookupRoundTrip(t *testing.T) {
	resetRegistry()
	pred := func(g *Goal, m *mobs.Mob) bool { return true }
	RegisterGoalType("revenge", GoalTypeMeta{
		Predicate:     pred,
		ConflictsWith: []string{"protection"},
	})
	got, ok := lookupMeta("revenge")
	if !ok {
		t.Fatal("expected registered type to be found")
	}
	if got.Predicate == nil {
		t.Error("predicate not preserved")
	}
	if len(got.ConflictsWith) != 1 || got.ConflictsWith[0] != "protection" {
		t.Errorf("ConflictsWith = %v, want [protection]", got.ConflictsWith)
	}
}

func TestLookupMeta_UnknownTypeReturnsFalse(t *testing.T) {
	resetRegistry()
	if _, ok := lookupMeta("noexist"); ok {
		t.Error("expected ok=false for unknown type")
	}
}

func TestValidateSymmetry_NoWarningWhenBothDeclared(t *testing.T) {
	resetRegistry()
	RegisterGoalType("revenge", GoalTypeMeta{ConflictsWith: []string{"protection"}})
	RegisterGoalType("protection", GoalTypeMeta{ConflictsWith: []string{"revenge"}})
	warnings := ValidateSymmetry()
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %v", warnings)
	}
}

func TestValidateSymmetry_WarnsOnOneSidedDeclaration(t *testing.T) {
	resetRegistry()
	RegisterGoalType("revenge", GoalTypeMeta{ConflictsWith: []string{"protection"}})
	RegisterGoalType("protection", GoalTypeMeta{ConflictsWith: nil})
	warnings := ValidateSymmetry()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	// Warning should name both types so an author can locate the gap.
	w := warnings[0]
	if !contains(w, "revenge") || !contains(w, "protection") {
		t.Errorf("warning missing type names: %q", w)
	}
}

func TestRegisterGoalType_ReregisterOverwrites(t *testing.T) {
	resetRegistry()
	RegisterGoalType("revenge", GoalTypeMeta{ConflictsWith: []string{"a"}})
	RegisterGoalType("revenge", GoalTypeMeta{ConflictsWith: []string{"b"}})
	got, _ := lookupMeta("revenge")
	if len(got.ConflictsWith) != 1 || got.ConflictsWith[0] != "b" {
		t.Errorf("reregister did not overwrite: %v", got.ConflictsWith)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
```

- [ ] **Step 3.2: Run test to verify it fails**

Run: `go test ./internal/goals/... -run Registry -v`

Expected: FAIL — `registryMu`, `typeRegistry`, `RegisterGoalType`, `lookupMeta`, `ValidateSymmetry` undefined.

- [ ] **Step 3.3: Create the registry file**

Write `internal/goals/registry.go`:

```go
package goals

import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

var (
	registryMu   sync.RWMutex
	typeRegistry = map[string]GoalTypeMeta{}
)

// RegisterGoalType is called from each goal-type package's init().
// 4.1 ships with no callers; 4.3 fills the registry. Re-registration
// of an existing type overwrites and logs a warning (defensive — in
// practice each package registers once).
func RegisterGoalType(goalType string, meta GoalTypeMeta) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := typeRegistry[goalType]; exists {
		mudlog.Warn("goals.RegisterGoalType: overwriting existing registration",
			"type", goalType)
	}
	typeRegistry[goalType] = meta
}

// lookupMeta returns the registered metadata for goalType. Internal —
// store.go and registry validation are the only callers.
func lookupMeta(goalType string) (GoalTypeMeta, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	m, ok := typeRegistry[goalType]
	return m, ok
}

// ValidateSymmetry walks every registered type's ConflictsWith list
// and verifies that each target also declares the source. Returns a
// slice of warning strings (empty if all pairs are symmetric).
//
// Called from cmd/main at boot after all packages have registered;
// soft check (does not panic, does not block startup).
func ValidateSymmetry() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	var warnings []string
	for srcType, srcMeta := range typeRegistry {
		for _, dstType := range srcMeta.ConflictsWith {
			dstMeta, ok := typeRegistry[dstType]
			if !ok {
				// Forward-declared conflict targeting an unregistered
				// type — not a symmetry issue, but worth noting.
				continue
			}
			if !sliceContains(dstMeta.ConflictsWith, srcType) {
				warnings = append(warnings, fmt.Sprintf(
					"goals: conflict %q→%q is one-sided (%q has no reverse declaration)",
					srcType, dstType, dstType))
			}
		}
	}
	return warnings
}

func sliceContains(s []string, needle string) bool {
	for _, x := range s {
		if x == needle {
			return true
		}
	}
	return false
}
```

- [ ] **Step 3.4: Run test to verify it passes**

Run: `go test ./internal/goals/... -run Registry -v`

Expected: PASS for all five Registry tests.

- [ ] **Step 3.5: Commit**

```bash
git add internal/goals/registry.go internal/goals/registry_test.go
git commit -m "feat(goals): chunk 4.1 task 3 — type-metadata registry with symmetry check

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4 — Store API (Add / Remove / Clear / GoalsOf / IsSatisfied / IsExpired)

The main public surface. Conflict resolution lives here. Split into multiple sub-steps because Add has several distinct behaviors.

**Files:**
- Create: `internal/goals/store.go`
- Create: `internal/goals/store_test.go`

### 4a — GoalsOf + lazy-load + IsSatisfied + IsExpired

- [ ] **Step 4.1: Create the failing read-side test**

Write `internal/goals/store_test.go`:

```go
package goals

import (
	"errors"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestGoalsOf_EmptyWhenNothingPersisted(t *testing.T) {
	ClearCache()
	resetRegistry()
	if got := GoalsOf(99001, "ghost"); len(got) != 0 {
		t.Errorf("expected empty, got %d goals", len(got))
	}
}

func TestGoalsOf_LazyLoadFromDisk(t *testing.T) {
	ClearCache()
	resetRegistry()
	mg := &MobGoals{
		MobId:      99002,
		NextGoalId: 2,
		Goals: []*Goal{
			{Id: "g1", Type: "revenge", Priority: 70, CreatedAt: time.Now().UTC()},
		},
	}
	cacheStoreForTest("disk-mob", mg)
	if err := saveToDisk(99002, "disk-mob"); err != nil {
		t.Fatalf("save: %v", err)
	}
	ClearCache()
	got := GoalsOf(99002, "disk-mob")
	if len(got) != 1 {
		t.Fatalf("expected 1 goal after lazy load, got %d", len(got))
	}
	if got[0].OwnerMobId != 99002 {
		t.Errorf("OwnerMobId not stamped on load: got %d", got[0].OwnerMobId)
	}
}

func TestIsSatisfied_NoPredicateReturnsFalse(t *testing.T) {
	resetRegistry()
	g := &Goal{Id: "g1", Type: "noexist", OwnerMobId: 1}
	if IsSatisfied(g, nil) {
		t.Error("expected false when no predicate registered")
	}
}

func TestIsSatisfied_InvokesRegisteredPredicate(t *testing.T) {
	resetRegistry()
	called := false
	RegisterGoalType("pingtest", GoalTypeMeta{
		Predicate: func(g *Goal, m *mobs.Mob) bool {
			called = true
			return true
		},
	})
	g := &Goal{Id: "g1", Type: "pingtest", OwnerMobId: 1}
	if !IsSatisfied(g, nil) {
		t.Error("expected true from predicate")
	}
	if !called {
		t.Error("predicate was not invoked")
	}
}

func TestIsExpired_ZeroNeverExpires(t *testing.T) {
	g := &Goal{Id: "g1"}
	if IsExpired(g, time.Now()) {
		t.Error("zero ExpiresAt should never expire")
	}
}

func TestIsExpired_PastTime(t *testing.T) {
	g := &Goal{Id: "g1", ExpiresAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)}
	if !IsExpired(g, time.Now()) {
		t.Error("expected expired")
	}
}

func TestIsExpired_FutureTime(t *testing.T) {
	g := &Goal{Id: "g1", ExpiresAt: time.Now().Add(time.Hour)}
	if IsExpired(g, time.Now()) {
		t.Error("expected not expired")
	}
}
```

- [ ] **Step 4.2: Run test to verify it fails**

Run: `go test ./internal/goals/... -run "GoalsOf|IsSatisfied|IsExpired" -v`

Expected: FAIL — `GoalsOf`, `IsSatisfied`, `IsExpired` undefined.

- [ ] **Step 4.3: Create initial store.go with read-side functions**

Write `internal/goals/store.go`:

```go
package goals

import (
	"fmt"
	"sort"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// loadOrLazyInit returns the cached MobGoals for mobId, loading from
// disk on first access. Returns a fresh empty MobGoals if neither
// cache nor disk has data. Mirrors the chunk-1.3 double-check pattern.
func loadOrLazyInit(mobId int, namesimple string) *MobGoals {
	cacheMu.RLock()
	if mg, ok := cache[mobId]; ok {
		cacheMu.RUnlock()
		return mg
	}
	cacheMu.RUnlock()

	cacheMu.Lock()
	defer cacheMu.Unlock()
	// Double-check after upgrading to write lock.
	if mg, ok := cache[mobId]; ok {
		return mg
	}
	mg := loadFromDisk(mobId, namesimple)
	if mg == nil {
		mg = &MobGoals{MobId: mobId, NextGoalId: 1, Goals: nil}
	}
	cache[mobId] = mg
	nameByMobId[mobId] = namesimple
	return mg
}

// GoalsOf returns the mob's goals in priority-desc, then id-asc order
// (stable for admin output and any future selection layer). Lazy
// loads from disk on first call.
//
// The returned slice is a copy — callers can sort or slice it freely
// without affecting the cache.
func GoalsOf(mobId int, namesimple string) []*Goal {
	mg := loadOrLazyInit(mobId, namesimple)
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	out := make([]*Goal, len(mg.Goals))
	copy(out, mg.Goals)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority > out[j].Priority
		}
		return out[i].Id < out[j].Id
	})
	return out
}

// IsSatisfied looks up the predicate registered for g.Type and
// returns its result. Returns false if no predicate is registered —
// safe default: a goal we don't know how to evaluate stays alive.
func IsSatisfied(g *Goal, mob *mobs.Mob) bool {
	meta, ok := lookupMeta(g.Type)
	if !ok || meta.Predicate == nil {
		return false
	}
	return meta.Predicate(g, mob)
}

// IsExpired is a pure time check. Goals with ExpiresAt.IsZero()
// never expire.
func IsExpired(g *Goal, now time.Time) bool {
	if g.ExpiresAt.IsZero() {
		return false
	}
	return !now.Before(g.ExpiresAt)
}
```

- [ ] **Step 4.4: Run test to verify it passes**

Run: `go test ./internal/goals/... -run "GoalsOf|IsSatisfied|IsExpired" -v`

Expected: PASS for all seven tests.

- [ ] **Step 4.5: Commit**

```bash
git add internal/goals/store.go internal/goals/store_test.go
git commit -m "feat(goals): chunk 4.1 task 4a — GoalsOf/IsSatisfied/IsExpired

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### 4b — Add happy path + sequential id assignment

- [ ] **Step 4.6: Append the Add happy-path test**

Append to `internal/goals/store_test.go`:

```go
func TestAdd_HappyPath(t *testing.T) {
	ClearCache()
	resetRegistry()
	g := &Goal{Type: "revenge", Priority: 70, Params: map[string]any{"k": "v"}}
	res, err := Add(99003, "addmob", g)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if res.Added == nil || res.Added.Id != "g1" {
		t.Errorf("expected Id=g1, got %+v", res.Added)
	}
	if res.Added.OwnerMobId != 99003 {
		t.Errorf("OwnerMobId not stamped: got %d", res.Added.OwnerMobId)
	}
	if res.Added.CreatedAt.IsZero() {
		t.Error("CreatedAt not stamped")
	}
	if len(res.Displaced) != 0 {
		t.Errorf("expected no displacements, got %v", res.Displaced)
	}
	got := GoalsOf(99003, "addmob")
	if len(got) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(got))
	}
}

func TestAdd_AssignsSequentialIds(t *testing.T) {
	ClearCache()
	resetRegistry()
	for i, want := range []string{"g1", "g2", "g3"} {
		res, err := Add(99004, "seqmob", &Goal{
			Type:     fmt.Sprintf("type%d", i),
			Priority: 50 - i, // descending so no conflicts within the test
		})
		if err != nil {
			t.Fatalf("Add %d: %v", i, err)
		}
		if res.Added.Id != want {
			t.Errorf("Add %d: id=%q, want %q", i, res.Added.Id, want)
		}
	}
}

func TestAdd_PersistsAcrossClearCache(t *testing.T) {
	ClearCache()
	resetRegistry()
	if _, err := Add(99005, "persistmob", &Goal{Type: "alpha", Priority: 50}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	ClearCache()
	got := GoalsOf(99005, "persistmob")
	if len(got) != 1 || got[0].Type != "alpha" {
		t.Fatalf("did not persist: %+v", got)
	}
}
```

- [ ] **Step 4.7: Run test to verify it fails**

Run: `go test ./internal/goals/... -run TestAdd -v`

Expected: FAIL — `Add` undefined.

- [ ] **Step 4.8: Append Add and conflict helpers to store.go**

Append to `internal/goals/store.go`:

```go
// Add appends a goal to the mob's list, resolving conflicts by
// priority. Returns *ConflictError if any conflicting existing goal
// has priority >= the new goal's priority. Persists to disk under the
// write mutex.
//
// g.Id is ignored on entry and assigned by Add; g.OwnerMobId is set
// from mobId; g.CreatedAt is stamped to time.Now().UTC() if zero.
func Add(mobId int, namesimple string, g *Goal) (AddResult, error) {
	mg := loadOrLazyInit(mobId, namesimple)

	cacheMu.Lock()

	// Detect conflicting existing goals. "Same type" always conflicts
	// (no AllowMultiple opt-in in 4.1). Cross-type uses the registered
	// ConflictsWith list, with a symmetry safety net checking both
	// directions.
	newMeta, _ := lookupMeta(g.Type)
	var conflicting []*Goal
	for _, e := range mg.Goals {
		if isConflict(g.Type, e.Type, newMeta) {
			conflicting = append(conflicting, e)
		}
	}

	// Priority resolution: every conflicting existing goal must have
	// strictly lower priority for the new goal to win.
	for _, e := range conflicting {
		if g.Priority <= e.Priority {
			cacheMu.Unlock()
			return AddResult{}, &ConflictError{
				BlockerGoalId: e.Id,
				BlockerType:   e.Type,
				BlockerPrio:   e.Priority,
			}
		}
	}

	// Displace lower-priority conflicting goals in place.
	displaced := make([]string, 0, len(conflicting))
	if len(conflicting) > 0 {
		mg.Goals = removeGoals(mg.Goals, conflicting)
		for _, e := range conflicting {
			displaced = append(displaced, e.Id)
		}
	}

	// Assign id, owner, timestamp, and append.
	g.Id = fmt.Sprintf("g%d", mg.NextGoalId)
	g.OwnerMobId = mobId
	if g.CreatedAt.IsZero() {
		g.CreatedAt = time.Now().UTC()
	}
	mg.NextGoalId++
	mg.Goals = append(mg.Goals, g)
	nameByMobId[mobId] = namesimple
	cacheMu.Unlock()

	if err := saveToDisk(mobId, namesimple); err != nil {
		mudlog.Warn("goals.Add: save failed", "mob_id", mobId, "error", err)
		// Cache is still authoritative; caller treats as success.
	}
	return AddResult{Added: g, Displaced: displaced}, nil
}

// isConflict reports whether existingType conflicts with newType per
// the registry. Same-type is always a conflict in 4.1. Cross-type
// checks newMeta.ConflictsWith and also looks up the existing type's
// metadata as a symmetry safety net (so a one-sided declaration still
// catches the conflict).
func isConflict(newType, existingType string, newMeta GoalTypeMeta) bool {
	if newType == existingType {
		return true
	}
	if sliceContains(newMeta.ConflictsWith, existingType) {
		return true
	}
	if existingMeta, ok := lookupMeta(existingType); ok {
		if sliceContains(existingMeta.ConflictsWith, newType) {
			return true
		}
	}
	return false
}

// removeGoals returns goals with the items in drop removed, preserving
// order. O(n*m) but n and m are small (goals-per-mob ≤ ~10 in practice).
func removeGoals(goals []*Goal, drop []*Goal) []*Goal {
	dropIds := make(map[string]bool, len(drop))
	for _, d := range drop {
		dropIds[d.Id] = true
	}
	out := goals[:0:0]
	for _, g := range goals {
		if !dropIds[g.Id] {
			out = append(out, g)
		}
	}
	return out
}
```

- [ ] **Step 4.9: Run test to verify it passes**

Run: `go test ./internal/goals/... -run TestAdd -v`

Expected: PASS for `TestAdd_HappyPath`, `TestAdd_AssignsSequentialIds`, `TestAdd_PersistsAcrossClearCache`.

- [ ] **Step 4.10: Commit**

```bash
git add internal/goals/store.go internal/goals/store_test.go
git commit -m "feat(goals): chunk 4.1 task 4b — Add happy path + sequential ids

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### 4c — Add conflict resolution (same-type + cross-type)

- [ ] **Step 4.11: Append the conflict tests**

Append to `internal/goals/store_test.go`:

```go
func TestAdd_SameTypeBlockedByHigherOrEqualPriority(t *testing.T) {
	ClearCache()
	resetRegistry()
	if _, err := Add(99006, "conflict1", &Goal{Type: "revenge", Priority: 70}); err != nil {
		t.Fatal(err)
	}
	_, err := Add(99006, "conflict1", &Goal{Type: "revenge", Priority: 70})
	var ce *ConflictError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConflictError, got %v", err)
	}
	if ce.BlockerGoalId != "g1" || ce.BlockerType != "revenge" || ce.BlockerPrio != 70 {
		t.Errorf("ConflictError fields wrong: %+v", ce)
	}
	if len(GoalsOf(99006, "conflict1")) != 1 {
		t.Error("blocked Add should not have appended")
	}
}

func TestAdd_SameTypeDisplacesLowerPriority(t *testing.T) {
	ClearCache()
	resetRegistry()
	if _, err := Add(99007, "conflict2", &Goal{Type: "revenge", Priority: 30}); err != nil {
		t.Fatal(err)
	}
	res, err := Add(99007, "conflict2", &Goal{Type: "revenge", Priority: 70})
	if err != nil {
		t.Fatalf("higher-prio Add: %v", err)
	}
	if len(res.Displaced) != 1 || res.Displaced[0] != "g1" {
		t.Errorf("expected displaced=[g1], got %v", res.Displaced)
	}
	got := GoalsOf(99007, "conflict2")
	if len(got) != 1 || got[0].Id != "g2" || got[0].Priority != 70 {
		t.Errorf("after displacement, expected single g2 priority=70; got %+v", got)
	}
}

func TestAdd_CrossTypeConflict_BothDirectionsDeclared(t *testing.T) {
	ClearCache()
	resetRegistry()
	RegisterGoalType("revenge", GoalTypeMeta{ConflictsWith: []string{"protection"}})
	RegisterGoalType("protection", GoalTypeMeta{ConflictsWith: []string{"revenge"}})

	if _, err := Add(99008, "crossA", &Goal{Type: "revenge", Priority: 50}); err != nil {
		t.Fatal(err)
	}
	_, err := Add(99008, "crossA", &Goal{Type: "protection", Priority: 40})
	var ce *ConflictError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConflictError, got %v", err)
	}
}

func TestAdd_CrossTypeConflict_SymmetrySafetyNet(t *testing.T) {
	// "revenge" declares protection as a conflict, but "protection"
	// does NOT declare revenge. Adding protection while revenge exists
	// should still be blocked, because the existing type's metadata
	// gets checked for the reverse edge by the safety net.
	//
	// Note: in this scenario the safety net path triggers because the
	// EXISTING goal type (revenge) has the declaration, and the NEW
	// goal type (protection) does not. isConflict's third branch (look
	// up the existing type's meta and check its ConflictsWith for the
	// new type) catches this.
	ClearCache()
	resetRegistry()
	RegisterGoalType("revenge", GoalTypeMeta{ConflictsWith: []string{"protection"}})
	RegisterGoalType("protection", GoalTypeMeta{ConflictsWith: nil})

	if _, err := Add(99009, "crossB", &Goal{Type: "revenge", Priority: 50}); err != nil {
		t.Fatal(err)
	}
	_, err := Add(99009, "crossB", &Goal{Type: "protection", Priority: 40})
	var ce *ConflictError
	if !errors.As(err, &ce) {
		t.Fatalf("expected ConflictError via symmetry safety net, got %v", err)
	}
}

func TestAdd_DisplacesMultipleConflicts(t *testing.T) {
	ClearCache()
	resetRegistry()
	RegisterGoalType("a", GoalTypeMeta{ConflictsWith: []string{"b", "c"}})
	RegisterGoalType("b", GoalTypeMeta{ConflictsWith: []string{"a"}})
	RegisterGoalType("c", GoalTypeMeta{ConflictsWith: []string{"a"}})

	if _, err := Add(99010, "multi", &Goal{Type: "b", Priority: 30}); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(99010, "multi", &Goal{Type: "c", Priority: 40}); err != nil {
		t.Fatal(err)
	}
	res, err := Add(99010, "multi", &Goal{Type: "a", Priority: 70})
	if err != nil {
		t.Fatalf("Add a: %v", err)
	}
	if len(res.Displaced) != 2 {
		t.Errorf("expected 2 displaced, got %v", res.Displaced)
	}
	got := GoalsOf(99010, "multi")
	if len(got) != 1 || got[0].Type != "a" {
		t.Errorf("expected only type=a remaining, got %+v", got)
	}
}
```

- [ ] **Step 4.12: Run test to verify it passes**

Run: `go test ./internal/goals/... -run "TestAdd_SameType|TestAdd_CrossType|TestAdd_Displaces" -v`

Expected: PASS for all five conflict tests. (The conflict logic was already implemented in step 4.8; these tests verify it.)

- [ ] **Step 4.13: Commit**

```bash
git add internal/goals/store_test.go
git commit -m "test(goals): chunk 4.1 task 4c — conflict-resolution coverage

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### 4d — Remove + Clear

- [ ] **Step 4.14: Append Remove/Clear tests**

Append to `internal/goals/store_test.go`:

```go
func TestRemove_HappyPath(t *testing.T) {
	ClearCache()
	resetRegistry()
	res, err := Add(99011, "removable", &Goal{Type: "alpha", Priority: 50})
	if err != nil {
		t.Fatal(err)
	}
	if err := Remove(99011, "removable", res.Added.Id); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(GoalsOf(99011, "removable")) != 0 {
		t.Error("expected empty after Remove")
	}
}

func TestRemove_MissingIdReturnsErrGoalNotFound(t *testing.T) {
	ClearCache()
	resetRegistry()
	err := Remove(99012, "empty", "g999")
	if !errors.Is(err, ErrGoalNotFound) {
		t.Errorf("expected ErrGoalNotFound, got %v", err)
	}
}

func TestRemove_DoesNotResetNextGoalId(t *testing.T) {
	// Ids are never reused even after Remove. After Add → Remove → Add,
	// the second Add should produce g2, not g1.
	ClearCache()
	resetRegistry()
	r1, _ := Add(99013, "noreuse", &Goal{Type: "x", Priority: 10})
	if err := Remove(99013, "noreuse", r1.Added.Id); err != nil {
		t.Fatal(err)
	}
	r2, err := Add(99013, "noreuse", &Goal{Type: "y", Priority: 20})
	if err != nil {
		t.Fatal(err)
	}
	if r2.Added.Id != "g2" {
		t.Errorf("expected g2 after Remove+Add, got %q", r2.Added.Id)
	}
}

func TestClear_RemovesAllAndResetsCounter(t *testing.T) {
	// Note: spec says ids never reused for the lifetime of a mob's file.
	// Clear is admin-only and intentionally heavy-hand — it wipes the
	// file AND resets NextGoalId to 1 so the operator gets a clean slate.
	ClearCache()
	resetRegistry()
	if _, err := Add(99014, "clearable", &Goal{Type: "a", Priority: 10}); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(99014, "clearable", &Goal{Type: "b", Priority: 20}); err != nil {
		t.Fatal(err)
	}
	if err := Clear(99014, "clearable"); err != nil {
		t.Fatal(err)
	}
	if len(GoalsOf(99014, "clearable")) != 0 {
		t.Error("expected empty after Clear")
	}
	r, err := Add(99014, "clearable", &Goal{Type: "c", Priority: 30})
	if err != nil {
		t.Fatal(err)
	}
	if r.Added.Id != "g1" {
		t.Errorf("expected counter reset to g1 after Clear, got %q", r.Added.Id)
	}
}
```

- [ ] **Step 4.15: Run test to verify it fails**

Run: `go test ./internal/goals/... -run "TestRemove|TestClear" -v`

Expected: FAIL — `Remove`, `Clear` undefined.

- [ ] **Step 4.16: Append Remove and Clear to store.go**

Append to `internal/goals/store.go`:

```go
// Remove deletes a goal by id. Returns ErrGoalNotFound if the id is
// not present on the mob. NextGoalId is NOT decremented — ids are
// never reused within the lifetime of a mob's file.
func Remove(mobId int, namesimple, goalId string) error {
	mg := loadOrLazyInit(mobId, namesimple)
	cacheMu.Lock()
	found := false
	out := mg.Goals[:0:0]
	for _, g := range mg.Goals {
		if g.Id == goalId {
			found = true
			continue
		}
		out = append(out, g)
	}
	if !found {
		cacheMu.Unlock()
		return ErrGoalNotFound
	}
	mg.Goals = out
	nameByMobId[mobId] = namesimple
	cacheMu.Unlock()

	if err := saveToDisk(mobId, namesimple); err != nil {
		mudlog.Warn("goals.Remove: save failed", "mob_id", mobId, "error", err)
	}
	return nil
}

// Clear removes every goal from the mob and resets NextGoalId to 1.
// Admin-only — intentionally heavy-hand for resetting a mob's goal
// state to defaults.
func Clear(mobId int, namesimple string) error {
	mg := loadOrLazyInit(mobId, namesimple)
	cacheMu.Lock()
	mg.Goals = nil
	mg.NextGoalId = 1
	nameByMobId[mobId] = namesimple
	cacheMu.Unlock()

	if err := saveToDisk(mobId, namesimple); err != nil {
		mudlog.Warn("goals.Clear: save failed", "mob_id", mobId, "error", err)
	}
	return nil
}
```

- [ ] **Step 4.17: Run test to verify it passes**

Run: `go test ./internal/goals/... -run "TestRemove|TestClear" -v`

Expected: PASS for all four Remove/Clear tests.

- [ ] **Step 4.18: Commit**

```bash
git add internal/goals/store.go internal/goals/store_test.go
git commit -m "feat(goals): chunk 4.1 task 4d — Remove + Clear

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

### 4e — Stable ordering + concurrent safety

- [ ] **Step 4.19: Append ordering + race tests**

Append to `internal/goals/store_test.go`:

```go
func TestGoalsOf_PriorityDescThenIdAsc(t *testing.T) {
	ClearCache()
	resetRegistry()
	// Different types to avoid same-type conflict.
	if _, err := Add(99015, "ordering", &Goal{Type: "a", Priority: 30}); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(99015, "ordering", &Goal{Type: "b", Priority: 70}); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(99015, "ordering", &Goal{Type: "c", Priority: 70}); err != nil {
		t.Fatal(err)
	}
	got := GoalsOf(99015, "ordering")
	if len(got) != 3 {
		t.Fatalf("expected 3 goals, got %d", len(got))
	}
	// Priority 70 g2 first (older id wins ties), then g3, then g1 prio 30.
	wantIds := []string{"g2", "g3", "g1"}
	for i, want := range wantIds {
		if got[i].Id != want {
			t.Errorf("position %d: got %q, want %q", i, got[i].Id, want)
		}
	}
}

func TestGoalsOf_ReturnedSliceIsCopy(t *testing.T) {
	// Mutating the returned slice (sort, append) must not affect the cache.
	ClearCache()
	resetRegistry()
	if _, err := Add(99016, "copytest", &Goal{Type: "a", Priority: 10}); err != nil {
		t.Fatal(err)
	}
	got := GoalsOf(99016, "copytest")
	sort.Slice(got, func(i, j int) bool { return false }) // reverse meaningless; just touch
	got = append(got, &Goal{Id: "fake"})
	if len(GoalsOf(99016, "copytest")) != 1 {
		t.Error("mutating returned slice affected cache")
	}
}

func TestConcurrentAdd_DifferentMobsNoRace(t *testing.T) {
	ClearCache()
	resetRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(mobId int) {
			defer wg.Done()
			if _, err := Add(mobId, fmt.Sprintf("racer%d", mobId), &Goal{
				Type: "x", Priority: 10,
			}); err != nil {
				t.Errorf("Add %d: %v", mobId, err)
			}
		}(99100 + i)
	}
	wg.Wait()
}

func TestConcurrentAdd_SameMobSerializes(t *testing.T) {
	ClearCache()
	resetRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// Unique type per goroutine so no same-type conflict.
			if _, err := Add(99200, "samemob", &Goal{
				Type:     fmt.Sprintf("t%d", idx),
				Priority: 10 + idx,
			}); err != nil {
				t.Errorf("Add %d: %v", idx, err)
			}
		}(i)
	}
	wg.Wait()
	got := GoalsOf(99200, "samemob")
	if len(got) != 5 {
		t.Errorf("expected 5 goals after concurrent Adds, got %d", len(got))
	}
}
```

- [ ] **Step 4.20: Run test (including race detector)**

Run: `go test ./internal/goals/... -race -run "TestGoalsOf_|TestConcurrent" -v`

Expected: PASS for all four; no race detector output.

- [ ] **Step 4.21: Commit**

```bash
git add internal/goals/store_test.go
git commit -m "test(goals): chunk 4.1 task 4e — ordering + concurrent-safety coverage

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5 — Admin command `goal`

In-game admin command for inspecting and seeding goals. Mirrors the shape of `admin.opinion.go`.

**Files:**
- Create: `internal/usercommands/admin.goal.go`
- Create: `internal/usercommands/admin.goal_test.go`
- Modify: `internal/usercommands/usercommands.go` (one-line registration)

- [ ] **Step 5.1: Register the command in the table**

Edit `internal/usercommands/usercommands.go` — locate the `opinion` entry on line ~138 and add a `goal` entry immediately after, alphabetically positioned between `gearset`-area commands. The exact insertion point: find the alphabetical neighborhood and add this line:

```go
		`goal`:            {Goal, true, true, true}, // Admin only
```

(The flag triple `true, true, true` matches `opinion`: admin-required, available out-of-combat, not stat-altering.)

- [ ] **Step 5.2: Create the failing admin-command test**

Pattern note: `user.SendText` enqueues to the event queue rather than a
capturable buffer, so `admin.opinion_test.go` deliberately skips
output-content assertions and relies on STATE assertions (calling
`opinions.Get` to verify mutations took effect). The goal command tests
follow the same pattern — every test verifies state via `goals.GoalsOf`
rather than inspecting sent text. The "no goal added after blocked Add"
state IS the assertion that the block path was taken.

Existing helpers reused from `usercommands_test.go`:
- `getTestUserAndRoom(t) (*users.UserRecord, *rooms.Room)` — returns the
  fixture admin user (id 1) and a loaded room.
- `seedAllRegistries() func()` — seeds keyword/buff/mob/etc registries
  needed by usercommands tests, returns a cleanup.

Mob id 1 is the standard "Skeleton" fixture; we need its namesimple for
`goals.GoalsOf` lookups. Resolve once via `mobs.GetMobSpec(1).Character.Name`
and pass through `util.ConvertForFilename`.

Write `internal/usercommands/admin.goal_test.go`:

```go
package usercommands

import (
	"errors"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// goalFixtureMobIdent returns the (mobId, namesimple) of the test fixture
// mob seeded by usercommands_test.go (Skeleton, id 1).
func goalFixtureMobIdent(t *testing.T) (int, string) {
	t.Helper()
	spec := mobs.GetMobSpec(mobs.MobId(1))
	if spec == nil {
		t.Fatal("test fixture mob id=1 missing — usercommands_test.go did not seed registry")
	}
	return 1, util.ConvertForFilename(spec.Character.Name)
}

// goalTestSetup arranges per-test isolation: temp goals dir, clean cache,
// clean registry, seeded usercommands registries. Returns the cleanup
// to defer.
func goalTestSetup(t *testing.T) func() {
	t.Helper()
	cleanup := seedAllRegistries()
	t.Setenv("DOGMUD_GOALS_DIR_OVERRIDE", t.TempDir())
	goals.ClearCache()
	return cleanup
}

func TestGoalCmd_NoArgsRunsWithoutError(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	if _, err := Goal("", admin, room, 0); err != nil {
		t.Fatalf("Goal usage path returned error: %v", err)
	}
	// State assertion: usage print should not have mutated any goals.
	mobId, ns := goalFixtureMobIdent(t)
	if len(goals.GoalsOf(mobId, ns)) != 0 {
		t.Error("usage print should not have created goals")
	}
}

func TestGoalCmd_AddCreatesGoal(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	mobId, ns := goalFixtureMobIdent(t)
	if _, err := Goal("add 1 alpha 50 reason=because", admin, room, 0); err != nil {
		t.Fatalf("Goal add: %v", err)
	}
	got := goals.GoalsOf(mobId, ns)
	if len(got) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(got))
	}
	if got[0].Type != "alpha" || got[0].Priority != 50 || got[0].Id != "g1" {
		t.Errorf("goal fields wrong: %+v", got[0])
	}
	if got[0].Params["reason"] != "because" {
		t.Errorf("param lost: %v", got[0].Params["reason"])
	}
}

func TestGoalCmd_AddBlockedDoesNotAppend(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	mobId, ns := goalFixtureMobIdent(t)
	if _, err := Goal("add 1 alpha 50", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	// Lower priority same-type Add should be blocked.
	if _, err := Goal("add 1 alpha 30", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	got := goals.GoalsOf(mobId, ns)
	if len(got) != 1 {
		t.Fatalf("expected 1 goal (block path), got %d", len(got))
	}
	if got[0].Priority != 50 {
		t.Errorf("existing goal should be untouched, prio=%d", got[0].Priority)
	}
}

func TestGoalCmd_AddDisplacesLowerPriority(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	mobId, ns := goalFixtureMobIdent(t)
	if _, err := Goal("add 1 alpha 30", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := Goal("add 1 alpha 70", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	got := goals.GoalsOf(mobId, ns)
	if len(got) != 1 {
		t.Fatalf("expected 1 goal after displacement, got %d", len(got))
	}
	if got[0].Id != "g2" || got[0].Priority != 70 {
		t.Errorf("expected only g2 prio=70, got %+v", got[0])
	}
}

func TestGoalCmd_KeyValueParamTypes(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	mobId, ns := goalFixtureMobIdent(t)
	cmd := "add 1 alpha 50 name=tova count=42 ratio=1.5 active=true"
	if _, err := Goal(cmd, admin, room, 0); err != nil {
		t.Fatal(err)
	}
	got := goals.GoalsOf(mobId, ns)
	if len(got) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(got))
	}
	p := got[0].Params
	if p["name"] != "tova" {
		t.Errorf("string param: %v (%T)", p["name"], p["name"])
	}
	if p["count"] != 42 {
		t.Errorf("int param: %v (%T)", p["count"], p["count"])
	}
	if p["ratio"] != 1.5 {
		t.Errorf("float param: %v (%T)", p["ratio"], p["ratio"])
	}
	if p["active"] != true {
		t.Errorf("bool param: %v (%T)", p["active"], p["active"])
	}
}

func TestGoalCmd_RemoveDeletesGoal(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	mobId, ns := goalFixtureMobIdent(t)
	if _, err := Goal("add 1 alpha 50", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := Goal("remove 1 g1", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if len(goals.GoalsOf(mobId, ns)) != 0 {
		t.Error("expected empty after remove")
	}
}

func TestGoalCmd_ClearWipesGoals(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	mobId, ns := goalFixtureMobIdent(t)
	if _, err := Goal("add 1 alpha 50", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := Goal("add 1 beta 30", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := Goal("clear 1", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if len(goals.GoalsOf(mobId, ns)) != 0 {
		t.Error("expected empty after clear")
	}
}

func TestGoalCmd_BadMobIdentRunsWithoutError(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	// The command should print "Unknown mob" and return (true, nil) —
	// no Go-level error. State should be untouched.
	if _, err := Goal("list nosuchmob_abcxyz", admin, room, 0); err != nil {
		t.Fatalf("expected nil error for bad ident, got: %v", err)
	}
}

func TestGoalCmd_RemoveMissingGoalNoError(t *testing.T) {
	defer goalTestSetup(t)()
	admin, room := getTestUserAndRoom(t)
	// Underlying goals.Remove returns ErrGoalNotFound, but the admin
	// command swallows that into a user-facing message and returns nil.
	if _, err := Goal("remove 1 g99", admin, room, 0); err != nil {
		// Sanity: make sure the wrapped error class isn't being leaked.
		if errors.Is(err, goals.ErrGoalNotFound) {
			t.Fatal("admin command leaked ErrGoalNotFound to caller")
		}
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 5.3: Run test to verify it fails**

Run: `go test ./internal/usercommands/... -run TestGoalCmd -v`

Expected: FAIL — `Goal` undefined (the function in `admin.goal.go`).

- [ ] **Step 5.4: Create `admin.goal.go`**

Write `internal/usercommands/admin.goal.go`:

```go
package usercommands

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
 * Role Permissions:
 * goal           (Admin)
 */

// Goal is the admin command for inspecting and seeding mob goals.
// Subcommands:
//
//	goal list <mob-ident>
//	goal show <mob-ident> <goal-id>
//	goal add <mob-ident> <type> <priority> [key=value ...]
//	goal remove <mob-ident> <goal-id>
//	goal clear <mob-ident>
func Goal(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		goalShowUsage(user)
		return true, nil
	}
	switch strings.ToLower(args[0]) {
	case "list":
		return goalList(args[1:], user)
	case "show":
		return goalShow(args[1:], user)
	case "add":
		return goalAdd(args[1:], user)
	case "remove", "rm":
		return goalRemove(args[1:], user)
	case "clear":
		return goalClear(args[1:], user)
	default:
		goalShowUsage(user)
		return true, nil
	}
}

func goalShowUsage(user *users.UserRecord) {
	user.SendText(messaging.CategorySystem,
		"Usage:\r\n"+
			"  goal list <mob-ident>\r\n"+
			"  goal show <mob-ident> <goal-id>\r\n"+
			"  goal add <mob-ident> <type> <priority> [key=value ...]\r\n"+
			"  goal remove <mob-ident> <goal-id>\r\n"+
			"  goal clear <mob-ident>\r\n"+
			"\r\n"+
			"mob-ident: numeric template id (e.g. 371) or namesimple (e.g. tova).\r\n")
}

func goalResolveMobIdent(s string) (mobId int, name string, ok bool) {
	if id, err := strconv.Atoi(s); err == nil {
		spec := mobs.GetMobSpec(mobs.MobId(id))
		if spec == nil {
			return 0, "", false
		}
		return id, spec.Character.Name, true
	}
	wanted := strings.ToLower(s)
	for _, spec := range mobs.AllMobTemplates() {
		if strings.EqualFold(util.ConvertForFilename(spec.Character.Name), wanted) {
			return int(spec.MobId), spec.Character.Name, true
		}
	}
	return 0, "", false
}

func goalList(args []string, user *users.UserRecord) (bool, error) {
	if len(args) != 1 {
		goalShowUsage(user)
		return true, nil
	}
	mobId, name, ok := goalResolveMobIdent(args[0])
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}
	all := goals.GoalsOf(mobId, util.ConvertForFilename(name))
	if len(all) == 0 {
		user.SendText(messaging.CategorySystem,
			fmt.Sprintf("%s (%d) has no goals.\r\n", name, mobId))
		return true, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Goals for %s (%d):\r\n", name, mobId)
	fmt.Fprintf(&b, "  %-4s  %-20s  %-4s  %s\r\n", "ID", "Type", "Prio", "Params")
	fmt.Fprintf(&b, "  %-4s  %-20s  %-4s  %s\r\n", "----", strings.Repeat("-", 20), "----", "------")
	for _, g := range all {
		fmt.Fprintf(&b, "  %-4s  %-20s  %-4d  %s\r\n",
			g.Id, g.Type, g.Priority, formatParamsInline(g.Params))
	}
	user.SendText(messaging.CategorySystem, b.String())
	return true, nil
}

func goalShow(args []string, user *users.UserRecord) (bool, error) {
	if len(args) != 2 {
		goalShowUsage(user)
		return true, nil
	}
	mobId, name, ok := goalResolveMobIdent(args[0])
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}
	for _, g := range goals.GoalsOf(mobId, util.ConvertForFilename(name)) {
		if g.Id == args[1] {
			var b strings.Builder
			fmt.Fprintf(&b, "Goal %s on %s (%d):\r\n", g.Id, name, mobId)
			fmt.Fprintf(&b, "  type:        %s\r\n", g.Type)
			fmt.Fprintf(&b, "  priority:    %d\r\n", g.Priority)
			fmt.Fprintf(&b, "  created_at:  %s\r\n", g.CreatedAt.Format("2006-01-02 15:04:05Z"))
			if !g.ExpiresAt.IsZero() {
				fmt.Fprintf(&b, "  expires_at:  %s\r\n", g.ExpiresAt.Format("2006-01-02 15:04:05Z"))
			}
			if len(g.Params) > 0 {
				fmt.Fprintf(&b, "  params:\r\n")
				keys := make([]string, 0, len(g.Params))
				for k := range g.Params {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Fprintf(&b, "    %s: %v\r\n", k, g.Params[k])
				}
			}
			user.SendText(messaging.CategorySystem, b.String())
			return true, nil
		}
	}
	user.SendText(messaging.CategorySystem,
		fmt.Sprintf("No goal %s on %s (%d).\r\n", args[1], name, mobId))
	return true, nil
}

func goalAdd(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 3 {
		goalShowUsage(user)
		return true, nil
	}
	mobId, name, ok := goalResolveMobIdent(args[0])
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}
	goalType := args[1]
	prio, err := strconv.Atoi(args[2])
	if err != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Bad priority %q: %v\r\n", args[2], err))
		return true, nil
	}
	params := map[string]any{}
	for _, kv := range args[3:] {
		k, v, found := strings.Cut(kv, "=")
		if !found {
			user.SendText(messaging.CategorySystem,
				fmt.Sprintf("Bad param %q (expected key=value)\r\n", kv))
			return true, nil
		}
		params[k] = parseScalar(v)
	}
	g := &goals.Goal{
		Type:     goalType,
		Priority: prio,
		Params:   params,
	}
	res, err := goals.Add(mobId, util.ConvertForFilename(name), g)
	var ce *goals.ConflictError
	if errors.As(err, &ce) {
		user.SendText(messaging.CategorySystem,
			fmt.Sprintf("Blocked by goal %s (type=%s, priority=%d).\r\n",
				ce.BlockerGoalId, ce.BlockerType, ce.BlockerPrio))
		return true, nil
	}
	if err != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Add failed: %v\r\n", err))
		return true, nil
	}
	msg := fmt.Sprintf("Added goal %s (type=%s, priority=%d)",
		res.Added.Id, res.Added.Type, res.Added.Priority)
	if len(res.Displaced) > 0 {
		msg += fmt.Sprintf(" — displaced goals: %s", strings.Join(res.Displaced, ", "))
	}
	user.SendText(messaging.CategorySystem, msg+".\r\n")
	return true, nil
}

func goalRemove(args []string, user *users.UserRecord) (bool, error) {
	if len(args) != 2 {
		goalShowUsage(user)
		return true, nil
	}
	mobId, name, ok := goalResolveMobIdent(args[0])
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}
	err := goals.Remove(mobId, util.ConvertForFilename(name), args[1])
	if errors.Is(err, goals.ErrGoalNotFound) {
		user.SendText(messaging.CategorySystem,
			fmt.Sprintf("No goal %s on %s (%d).\r\n", args[1], name, mobId))
		return true, nil
	}
	if err != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Remove failed: %v\r\n", err))
		return true, nil
	}
	user.SendText(messaging.CategorySystem,
		fmt.Sprintf("Removed goal %s from %s (%d).\r\n", args[1], name, mobId))
	return true, nil
}

func goalClear(args []string, user *users.UserRecord) (bool, error) {
	if len(args) != 1 {
		goalShowUsage(user)
		return true, nil
	}
	mobId, name, ok := goalResolveMobIdent(args[0])
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}
	if err := goals.Clear(mobId, util.ConvertForFilename(name)); err != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Clear failed: %v\r\n", err))
		return true, nil
	}
	user.SendText(messaging.CategorySystem,
		fmt.Sprintf("Cleared all goals from %s (%d).\r\n", name, mobId))
	return true, nil
}

// parseScalar converts an unquoted token to int / float / bool /
// string, in that priority order.
func parseScalar(s string) any {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	switch strings.ToLower(s) {
	case "true":
		return true
	case "false":
		return false
	}
	return s
}

func formatParamsInline(p map[string]any) string {
	if len(p) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, p[k]))
	}
	return strings.Join(parts, " ")
}
```

Note: don't add a `type Goal = goals.Goal` alias — it would clash with
the `Goal` function name (the admin command entry point) in this same
package. Always use `goals.Goal` explicitly when constructing or
referring to the goal type.

- [ ] **Step 5.5: Run test to verify it passes**

Run: `go test ./internal/usercommands/... -run TestGoalCmd -v`

Expected: PASS for all seven `TestGoalCmd_*` tests.

The `getTestUserAndRoom` and `seedAllRegistries` helpers come from
`internal/usercommands/usercommands_test.go`. The skeleton-mob fixture
(id 1) is seeded by `seedAllRegistries`; its `Character.Name` flows
through `util.ConvertForFilename` to produce the namesimple key the
goals store uses. If the namesimple resolves to something other than
what `goalFixtureMobIdent` returns, the existing `opinion` admin tests
would already be broken — they share the fixture.

- [ ] **Step 5.6: Run the full package test suite**

Run: `go test ./internal/goals/... ./internal/usercommands/...`

Expected: all green.

- [ ] **Step 5.7: Commit**

```bash
git add internal/usercommands/admin.goal.go internal/usercommands/admin.goal_test.go internal/usercommands/usercommands.go
git commit -m "feat(goals): chunk 4.1 task 5 — admin goal command

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6 — Push-prep + smoke + roadmap rollup

The final task: gitignore the runtime store, update the roadmap, write a PATCH_NOTES entry, run the pre-push SOP boot test, and exercise the admin command in-game.

**Files:**
- Modify: `.gitignore`
- Modify: `MOB_ALIVENESS_ROADMAP.md`
- Modify: `PATCH_NOTES.md`
- Modify: `_datafiles/config.yaml` (verify `Logging.LogToFile: false`)

- [ ] **Step 6.1: Add goals/ to gitignore**

Edit `.gitignore`. Find the section that ignores other runtime stores (likely a block containing `_datafiles/world/dogmud/opinions/` or `_datafiles/world/dogmud/knowledge/`). Add this line in the same section:

```
_datafiles/world/dogmud/goals/
```

- [ ] **Step 6.2: Flip the roadmap status**

Edit `MOB_ALIVENESS_ROADMAP.md`. Find line 104:

```markdown
| 4.1 | Strategic | Goal representation | M | 1.1, 1.4 | Not started |
```

Change to:

```markdown
| 4.1 | Strategic | Goal representation | M | 1.1, 1.4 | Done |
```

Also find the rollup counter elsewhere in the document (typically near the top — something like `**Completed:** 22/42`) and bump from 22 to 23.

Then update the §4.1 section status near line 652:

```markdown
### 4.1 Goal representation
**Status:** Done • **Size:** M
```

- [ ] **Step 6.3: Append PATCH_NOTES entry**

Edit `PATCH_NOTES.md`. Find the most recent dated section header (e.g. `## 2026-05-26`) and append a chunk-4.1 entry under it. If no entry exists for today, add a new section. Example entry:

```markdown
### Mob aliveness chunk 4.1 — Goal representation (substrate)

Foundation for Phase 4 strategic-layer work. New `internal/goals/`
package: typed `Goal` struct (id, owner mob template, type,
priority, params, expiry), per-mob YAML persistence at
`_datafiles/world/dogmud/goals/{mobId}-{namesimple}.yaml`,
goal-type metadata registry with predicate + `conflicts_with`
declarations, priority-resolved conflict policy at Add-time. Empty
registry at ship — chunk 4.3 fills it with concrete goal types.
No behavior-tree integration yet (4.4). No observable behavior
change for players. New admin command `goal` for inspection and
seeding (admin-only).
```

- [ ] **Step 6.4: Verify pre-push config**

Read `_datafiles/config.yaml` and confirm `Logging.LogToFile` is `false`. If `true`, set it to `false`. (This is the standing pre-push SOP — prod droplet has limited disk space.)

- [ ] **Step 6.5: Local boot smoke test (manual)**

Per CLAUDE.md SOP, nuke instance saves and boot the server locally:

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* \
       _datafiles/world/dogmud/rooms.instances/*
go build ./... && ./GoMud
```

Watch for: `mobs.LoadDataFiles() loadedCount=...`, `quests.LoadDataFiles() loadedCount=...` etc. without panics. The goals registry's ValidateSymmetry will log no warnings because the registry is empty at 4.1 ship.

Expected: clean startup, server reaches "ready" state without panic. New goals/ files do not appear on disk until an admin issues `goal add`.

- [ ] **Step 6.6: Exercise the admin command in-game (manual)**

Connect as the admin character. Find a test mob (any in-room NPC will do — use `look` to identify, then `goal list <id>` to inspect). Then:

```
goal add 1 testgoal 50 reason=smoketest count=42
goal list 1
goal show 1 g1
goal add 1 testgoal 30        # expect: Blocked by goal g1
goal add 1 testgoal 80        # expect: ... displaced goals: g1
goal remove 1 g2
goal clear 1
```

Verify each subcommand prints the expected output per §3.1 of the spec.

After the exercise, confirm a file appeared at
`_datafiles/world/dogmud/goals/{mobId}-{namesimple}.yaml`. Restart
the server (`quit`, re-launch). After reconnect, `goal list 1` should
return the file's contents (or "no goals" if the final exercise step
was `clear`).

- [ ] **Step 6.7: Commit the push-prep changes**

```bash
git add .gitignore MOB_ALIVENESS_ROADMAP.md PATCH_NOTES.md _datafiles/config.yaml
git commit -m "chore(aliveness 4.1): gitignore, roadmap, patch notes, prepush config

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

- [ ] **Step 6.8: Run the full test suite one final time**

Run: `go test ./... -race`

Expected: all green, no race output.

- [ ] **Step 6.9: Hand back to user for merge**

Per the existing `finishing-a-development-branch` workflow, present the
"merge locally / push PR / keep as-is / discard" options to the user.
Default recommendation: merge locally to master with `--no-ff` (per
DOGMud git workflow), then push origin/master.

---

## Notes for the implementer

- **Patterns to mirror exactly:** `internal/opinions/` is the closest analog
  for the package layout (types.go + persistence.go + ops file +
  test_main_test.go + per-file tests). `internal/bounties/` is the closest
  analog for the sequential-id-on-the-registry pattern (we just put the
  counter in each per-mob file instead of one global registry).

- **Mutex discipline:** every cache mutation takes `cacheMu.Lock()`, every
  read takes `cacheMu.RLock()`. Disk writes take `saveMu.Lock()` separately
  so they don't block reads. Never hold both at once (saveMu briefly
  releases cacheMu to marshal under RLock then writes outside that
  RLock — see `opinions/persistence.go:saveToDisk` for the canonical
  pattern).

- **Why `Goal.OwnerMobId` is `yaml:"-"`:** the on-disk MobGoals struct
  already carries the mob id at the top level. Storing it inside every
  Goal entry would be redundant and fragile (if the two disagreed, which
  wins?). Stamping it from `MobGoals.MobId` at load time is the single
  source of truth.

- **Why ids never reuse:** future chunks (4.5 reactive generation, 4.6
  satisfaction tracking, plus admin tooling) may want stable references
  to historical goal ids. Reusing g1 after a delete creates a footgun.
  `Clear` is the one exception — it explicitly wipes everything including
  the counter, and it's admin-only.

- **Registry validation is opt-in:** `ValidateSymmetry()` is a function
  the caller invokes (4.3's catalog-init code, or `cmd/main`). 4.1 doesn't
  wire it into boot because the registry is empty at 4.1 ship — calling
  it would always return an empty slice. 4.3's plan should wire it in.

- **No new dependencies:** every package used here is already in
  `go.mod`. Confirm with `go mod tidy` after Task 5.

- **Test isolation:** every test that touches `cache` or `typeRegistry`
  calls `ClearCache()` / `resetRegistry()` at the top. Tests in the
  goals package share `TestMain` which redirects the goals dir to a
  per-process temp; tests in the usercommands package need to call
  `goals.ClearCache()` between cases.
