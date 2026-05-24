# Mob Aliveness 1.2 — Faction System Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the chunk-1.2 substrate from
`docs/superpowers/specs/2026-05-06-mob-aliveness-1.2-faction-system-design.md`:
first-class factions with definitions, NPC membership via the existing
`Groups` field, per-player rep, faction-vs-faction relations, an admin
command, and an end-to-end migration of the `peacefulquest` placeholder
to the new system.

**Architecture:** New `internal/factions/` package mirrors the
`internal/opinions/` patterns (eager-loaded definition registry +
lazy-loaded per-faction rep YAML at `factions.rep/{slug}.yaml`,
`saveMu`-serialized file I/O). Combat hookup on `events.MobDeath`
bumps killer-and-party-room rep when a defined faction member dies.
Quest engine gains a `bump_rep` action so quest rewards can mutate
faction rep directly. The `peacefulquest` field is deleted in the
same chunk after a one-shot live-player migration seeds rep for
existing players.

**Tech Stack:** Go, `gopkg.in/yaml.v2`, existing GoMud patterns —
`internal/opinions/persistence.go` and `internal/migration/0.12.0.go`
are the closest precedents.

---

## File Structure

**New package: `internal/factions/`**

| File | Responsibility |
|------|-----------------|
| `types.go` | `Definition`, `RepEntry`, `FactionRep`, `RepMin/Max` |
| `factions.go` | Public API: `GetRep`, `SetRep`, `BumpRep`, `TierFor`, `FactionsForMob`, `IsPeacefulToward` |
| `persistence.go` | Cache, path, `loadFromDisk`, `saveToDisk`, `SaveAllRep`, `ClearCache`, `saveMu` |
| `registry.go` | Eager `LoadAllDefinitions`, `GetDefinition`, `AllDefinitions`, ally/enemy validation |
| `factions_test.go` | Unit tests for API |
| `persistence_test.go` | Save/load round-trip, corrupt YAML, concurrency |
| `registry_test.go` | Loader, ally/enemy ref validation panics |
| `test_main_test.go` | mudlog init |

**New supporting files:**

| File | Responsibility |
|------|-----------------|
| `internal/hooks/MobDeath_FactionRep.go` | Combat hookup |
| `internal/hooks/MobDeath_FactionRep_test.go` | Combat hookup tests |
| `internal/usercommands/admin.faction.go` | Admin command |
| `internal/usercommands/admin.faction_test.go` | Admin command tests |
| `internal/migration/0.13.0.go` | Live-player rep seed migration |
| `_datafiles/world/dogmud/factions/warren.yaml` | Warren faction definition |
| `_datafiles/world/dogmud/factions/thornwall_guards.yaml` | Thornwall guards definition |
| `_datafiles/world/dogmud/templates/admincommands/help/command.faction.template` | Admin helpfile |

**Modified files:**

| File | Change |
|------|--------|
| `internal/configs/config.balance.go` | Add `FactionMemberKillRep` field |
| `internal/configs/config.balance.misc.go` | Default-set the new field |
| `internal/questengine/types.go` (or actions.go) | Add `BumpRepDef`, `ActionDef.BumpRep`, `ActionContext.BumpRep` |
| `internal/questengine/actions.go` | Dispatch `bump_rep` in `ExecuteAction` |
| `internal/questengine/bridge.go` | Real-bridge `BumpRep` calls `factions.BumpRep` |
| `internal/usercommands/usercommands.go` | Register `faction` admin command |
| `internal/migration/migration.go` | Wire 0.13.0 migration in `doAllMigrations` |
| `internal/mobcommands/lookfortrouble.go` | Replace peacefulquest gate |
| `internal/behaviortree/actions_party.go` | Replace peacefulquest gate |
| `internal/mobs/mobs.go` | Delete `Mob.PeacefulQuest` field |
| `world.go` (or wherever startup runs) | Call `factions.LoadAllDefinitions()` |
| `_datafiles/world/dogmud/quests/2-the_warren_compact.yaml` | Add `bump_rep` action |
| `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/72-warren_scout.yaml` | Drop peacefulquest, add `warren` group |
| `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/73-warren_warrior.yaml` | Drop peacefulquest, add `warren` group |
| `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/74-tunnel_shaman.yaml` | Add `warren` group |
| `_datafiles/world/dogmud/mobs/thornwall_outskirts/92-city_gate_guard.yaml` | Add `thornwall_guards` group |
| `_datafiles/world/dogmud/mobs/thornwall_city/94-guard_captain_velk.yaml` | Add `thornwall_guards` group |
| `_datafiles/world/dogmud/mobs/thornwall_city/106-city_guard.yaml` | Add `thornwall_guards` group |
| `.gitignore` | Add `_datafiles/world/dogmud/factions.rep/` |
| `MOB_ALIVENESS_ROADMAP.md` | Add chunk 6.5a |

---

## Task 1: Package skeleton + types

**Files:**
- Create: `internal/factions/types.go`
- Create: `internal/factions/factions.go` (stub)
- Create: `internal/factions/test_main_test.go`
- Create: `internal/factions/factions_test.go`

- [ ] **Step 1: Write the failing test for `RepMin`/`RepMax` constants**

Create `internal/factions/factions_test.go`:

```go
package factions

import "testing"

func TestRepConstants(t *testing.T) {
	if RepMin != -100 {
		t.Errorf("RepMin = %d, want -100", RepMin)
	}
	if RepMax != 100 {
		t.Errorf("RepMax = %d, want 100", RepMax)
	}
}
```

- [ ] **Step 2: Run test to verify it fails (build error)**

Run: `go test ./internal/factions/...`
Expected: build failure — `factions` package does not exist yet.

- [ ] **Step 3: Create `types.go`**

```go
package factions

// Definition is one faction's authored content. Loaded eagerly at
// startup from _datafiles/world/dogmud/factions/{slug}.yaml.
//
// Definitions are immutable after load. Allies/Enemies are
// declarative — consumers may read the graph, but rep changes do
// NOT auto-propagate through it (per chunk 1.2 design).
type Definition struct {
	FactionId   string   `yaml:"faction_id"`
	DisplayName string   `yaml:"display_name"`
	Description string   `yaml:"description"`
	DefaultRep  int      `yaml:"default_rep"`
	Allies      []string `yaml:"allies"`
	Enemies     []string `yaml:"enemies"`
}

// RepEntry is one player's rep with one faction.
type RepEntry struct {
	Rep              int    `yaml:"rep"`
	LastUpdatedRound uint64 `yaml:"last_updated_round"`
}

// FactionRep is a faction's full per-player rep table. Persisted
// to _datafiles/world/dogmud/factions.rep/{slug}.yaml (gitignored).
type FactionRep struct {
	FactionId string             `yaml:"faction_id"`
	Players   map[int]*RepEntry  `yaml:"players"`
}

// Score range — every Set/Bump clamps to this window.
const (
	RepMin = -100
	RepMax = +100
)
```

- [ ] **Step 4: Create `factions.go` (stub for now)**

```go
package factions

// Public API lives here. Actual implementations land in later tasks.
```

- [ ] **Step 5: Create `test_main_test.go`** to initialize mudlog (mirrors `internal/opinions/test_main_test.go`)

```go
package factions

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}
```

- [ ] **Step 6: Run test to verify pass**

Run: `go test ./internal/factions/... -v`
Expected: PASS — `TestRepConstants`.

- [ ] **Step 7: Run vet/gofmt**

Run: `go vet ./internal/factions/... && gofmt -d internal/factions/`
Expected: clean (no diff).

- [ ] **Step 8: Commit**

```bash
git add internal/factions/types.go internal/factions/factions.go internal/factions/test_main_test.go internal/factions/factions_test.go
git commit -m "$(cat <<'EOF'
feat(factions): package skeleton with definition + rep types

Definition (immutable, authored), RepEntry + FactionRep (per-player
rep state), RepMin/RepMax constants. Public API lives in factions.go
as a stub until later tasks land.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Eager definition loader

**Files:**
- Create: `internal/factions/registry.go`
- Create: `internal/factions/registry_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/factions/registry_test.go`:

```go
package factions

import (
	"os"
	"path/filepath"
	"testing"
)

// helper: write a definition YAML into a temp dir and return the dir path.
func writeFactionDef(t *testing.T, dir, slug, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, slug+".yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadAllDefinitions_LoadsValidYAML(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", dir)

	writeFactionDef(t, dir, "warren", `
faction_id: warren
display_name: "The Warren"
description: "A coalition of warren scouts and warriors."
default_rep: -25
allies: []
enemies: []
`)
	writeFactionDef(t, dir, "thornwall_guards", `
faction_id: thornwall_guards
display_name: "Thornwall Guards"
description: "City watch."
default_rep: 0
allies: []
enemies: [warren]
`)

	clearRegistryForTest()
	if err := LoadAllDefinitions(); err != nil {
		t.Fatalf("LoadAllDefinitions: %v", err)
	}

	if d := GetDefinition("warren"); d == nil || d.DisplayName != "The Warren" {
		t.Errorf("warren definition not loaded: %+v", d)
	}
	if d := GetDefinition("thornwall_guards"); d == nil || d.DefaultRep != 0 {
		t.Errorf("thornwall_guards definition not loaded: %+v", d)
	}
	if all := AllDefinitions(); len(all) != 2 {
		t.Errorf("AllDefinitions returned %d, want 2", len(all))
	}
}

func TestLoadAllDefinitions_PanicsOnUnknownAllyReference(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", dir)

	writeFactionDef(t, dir, "warren", `
faction_id: warren
display_name: "Warren"
description: "x"
default_rep: 0
allies: [nonexistent_faction]
enemies: []
`)

	clearRegistryForTest()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on unknown ally reference, got none")
		}
	}()
	_ = LoadAllDefinitions()
}

func TestLoadAllDefinitions_PanicsOnUnknownEnemyReference(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", dir)

	writeFactionDef(t, dir, "guards", `
faction_id: guards
display_name: "Guards"
description: "x"
default_rep: 0
allies: []
enemies: [missing_faction]
`)

	clearRegistryForTest()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on unknown enemy reference, got none")
		}
	}()
	_ = LoadAllDefinitions()
}

func TestLoadAllDefinitions_MissingDirReturnsCleanly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "does-not-exist")
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", dir)
	clearRegistryForTest()
	if err := LoadAllDefinitions(); err != nil {
		t.Errorf("missing dir should be clean: %v", err)
	}
	if all := AllDefinitions(); len(all) != 0 {
		t.Errorf("AllDefinitions = %d, want 0", len(all))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail (build errors)**

Run: `go test ./internal/factions/... -run "TestLoadAllDefinitions" -v`
Expected: FAIL — `LoadAllDefinitions`, `GetDefinition`, `AllDefinitions`, `clearRegistryForTest`, `DOGMUD_FACTIONS_DIR_OVERRIDE` undefined.

- [ ] **Step 3: Implement `registry.go`**

```go
package factions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

var (
	definitionsMu sync.RWMutex
	definitions   = map[string]*Definition{}
)

// definitionsBaseDir returns the directory holding faction
// definition YAML files. Honors the DOGMUD_FACTIONS_DIR_OVERRIDE
// env var so tests can redirect to a temp dir.
func definitionsBaseDir() string {
	if override := os.Getenv("DOGMUD_FACTIONS_DIR_OVERRIDE"); override != "" {
		return override
	}
	return util.FilePath(
		configs.GetFilePathsConfig().DataFiles.String(), `/`, `world`, `/`, `dogmud`, `/`, `factions`,
	)
}

// LoadAllDefinitions reads every YAML file under the factions dir
// and populates the in-memory registry. Validates that every
// allies/enemies reference resolves to a loaded faction; panics on
// unknown reference so authoring errors surface at boot, not at
// runtime.
//
// Idempotent — safe to call multiple times. Subsequent calls
// fully replace the registry contents.
func LoadAllDefinitions() error {
	dir := definitionsBaseDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			definitionsMu.Lock()
			definitions = map[string]*Definition{}
			definitionsMu.Unlock()
			return nil
		}
		return fmt.Errorf("factions.LoadAllDefinitions: readdir %s: %w", dir, err)
	}

	loaded := map[string]*Definition{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			mudlog.Warn("factions.LoadAllDefinitions: read", "path", path, "error", readErr)
			continue
		}
		var def Definition
		if unmErr := yaml.Unmarshal(raw, &def); unmErr != nil {
			mudlog.Warn("factions.LoadAllDefinitions: unmarshal", "path", path, "error", unmErr)
			continue
		}
		if def.FactionId == "" {
			mudlog.Warn("factions.LoadAllDefinitions: missing faction_id", "path", path)
			continue
		}
		loaded[def.FactionId] = &def
	}

	// Validate ally/enemy references.
	for _, def := range loaded {
		for _, ally := range def.Allies {
			if _, ok := loaded[ally]; !ok {
				panic(fmt.Sprintf("factions: faction %q lists unknown ally %q",
					def.FactionId, ally))
			}
		}
		for _, enemy := range def.Enemies {
			if _, ok := loaded[enemy]; !ok {
				panic(fmt.Sprintf("factions: faction %q lists unknown enemy %q",
					def.FactionId, enemy))
			}
		}
	}

	definitionsMu.Lock()
	definitions = loaded
	definitionsMu.Unlock()

	mudlog.Info("factions.LoadAllDefinitions", "loadedCount", len(loaded))
	return nil
}

// GetDefinition returns the immutable Definition for factionId, or
// nil if unknown.
func GetDefinition(factionId string) *Definition {
	definitionsMu.RLock()
	defer definitionsMu.RUnlock()
	return definitions[factionId]
}

// AllDefinitions returns a snapshot of every loaded definition.
// Read-only — callers must not mutate.
func AllDefinitions() []*Definition {
	definitionsMu.RLock()
	defer definitionsMu.RUnlock()
	out := make([]*Definition, 0, len(definitions))
	for _, d := range definitions {
		out = append(out, d)
	}
	return out
}

// clearRegistryForTest wipes the registry. Test-only seam.
func clearRegistryForTest() {
	definitionsMu.Lock()
	definitions = map[string]*Definition{}
	definitionsMu.Unlock()
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/factions/... -v`
Expected: all pass.

- [ ] **Step 5: gofmt + vet**

Run: `gofmt -w internal/factions/ && go vet ./internal/factions/...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add internal/factions/registry.go internal/factions/registry_test.go
git commit -m "$(cat <<'EOF'
feat(factions): eager definition loader with ally/enemy validation

LoadAllDefinitions reads every YAML under factions/, validates that
allies/enemies reference loaded factions, and panics on unknown
reference so authoring errors surface at boot. Honors
DOGMUD_FACTIONS_DIR_OVERRIDE env var for tests.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Persistence scaffolding

**Files:**
- Create: `internal/factions/persistence.go`
- Create: `internal/factions/persistence_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/factions/persistence_test.go`:

```go
package factions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepPath(t *testing.T) {
	got := repPath("warren")
	if !strings.HasSuffix(filepath.ToSlash(got), "factions.rep/warren.yaml") {
		t.Errorf("repPath = %q, want suffix factions.rep/warren.yaml", got)
	}
}

func TestRepSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", dir)
	clearRepCacheForTest()

	fr := &FactionRep{
		FactionId: "warren",
		Players: map[int]*RepEntry{
			17: {Rep: 30, LastUpdatedRound: 1843201},
			92: {Rep: -75, LastUpdatedRound: 1846020},
		},
	}
	repCacheStoreForTest(fr)
	if err := saveRepToDisk("warren"); err != nil {
		t.Fatalf("saveRepToDisk: %v", err)
	}

	clearRepCacheForTest()
	got := loadRepFromDisk("warren")
	if got == nil {
		t.Fatal("loadRepFromDisk returned nil")
	}
	if got.FactionId != "warren" {
		t.Errorf("FactionId = %q", got.FactionId)
	}
	if got.Players[17].Rep != 30 || got.Players[17].LastUpdatedRound != 1843201 {
		t.Errorf("user 17 mismatch: %+v", got.Players[17])
	}
	if got.Players[92].Rep != -75 {
		t.Errorf("user 92 mismatch: %+v", got.Players[92])
	}

	expected := filepath.Join(dir, "warren.yaml")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected file missing: %v", err)
	}
}

func TestLoadRepFromDiskMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", dir)
	clearRepCacheForTest()
	if got := loadRepFromDisk("ghost"); got != nil {
		t.Errorf("loadRepFromDisk on missing file = %+v, want nil", got)
	}
}

func TestLoadRepFromDiskCorruptYAMLReturnsNil(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", dir)
	clearRepCacheForTest()

	bad := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(bad, []byte("\t\nplayers:\n  bad: {structure"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := loadRepFromDisk("bad"); got != nil {
		t.Errorf("loadRepFromDisk on corrupt YAML = %+v, want nil", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail (build errors)**

Run: `go test ./internal/factions/... -run "TestRepPath|TestRepSaveAndLoad|TestLoadRepFromDiskMissing|TestLoadRepFromDiskCorrupt" -v`
Expected: FAIL — `repPath`, `loadRepFromDisk`, `saveRepToDisk`, `clearRepCacheForTest`, `repCacheStoreForTest` undefined.

- [ ] **Step 3: Implement `persistence.go`**

```go
package factions

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

var (
	repCacheMu sync.RWMutex
	repCache   = map[string]*FactionRep{}

	// saveMu serializes file I/O so concurrent BumpRep on the same
	// faction don't trigger Windows ERROR_SHARING_VIOLATION on
	// overlapping os.WriteFile calls. Mirrors internal/opinions.
	saveMu sync.Mutex
)

func repBaseDir() string {
	if override := os.Getenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE"); override != "" {
		return override
	}
	return util.FilePath(
		configs.GetFilePathsConfig().DataFiles.String(), `/`, `world`, `/`, `dogmud`, `/`, `factions.rep`,
	)
}

func repPath(factionId string) string {
	return filepath.Join(repBaseDir(), factionId+".yaml")
}

func loadRepFromDisk(factionId string) *FactionRep {
	path := repPath(factionId)
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var fr FactionRep
	if err := yaml.Unmarshal(bytes, &fr); err != nil {
		mudlog.Error("factions.loadRepFromDisk", "path", path, "error", err)
		return nil
	}
	if fr.Players == nil {
		fr.Players = map[int]*RepEntry{}
	}
	return &fr
}

// saveRepToDisk persists the cached FactionRep for factionId. File
// I/O is serialized via saveMu so concurrent callers don't race
// (Windows ERROR_SHARING_VIOLATION). Re-acquires the cache RLock
// inside the saveMu critical section so the marshal sees a
// consistent snapshot of any Bumps that completed after the
// caller released the cache lock.
func saveRepToDisk(factionId string) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	repCacheMu.RLock()
	fr, ok := repCache[factionId]
	if !ok {
		repCacheMu.RUnlock()
		return fmt.Errorf("factions.saveRepToDisk: no cached entry for %q", factionId)
	}
	bytes, err := yaml.Marshal(fr)
	repCacheMu.RUnlock()
	if err != nil {
		return fmt.Errorf("factions.saveRepToDisk: marshal: %w", err)
	}

	path := repPath(factionId)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("factions.saveRepToDisk: mkdir %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, bytes, 0644); err != nil {
		return fmt.Errorf("factions.saveRepToDisk: write %s: %w", path, err)
	}
	return nil
}

// SaveAllRep flushes every cached FactionRep to disk. Defined for
// parity with opinions.SaveAllOpinions; not currently wired into
// shutdown (synchronous-on-mutation save covers it).
func SaveAllRep() {
	repCacheMu.RLock()
	ids := make([]string, 0, len(repCache))
	for id := range repCache {
		ids = append(ids, id)
	}
	repCacheMu.RUnlock()
	for _, id := range ids {
		if err := saveRepToDisk(id); err != nil {
			mudlog.Error("factions.SaveAllRep", "error", err)
		}
	}
}

// ClearCache wipes the rep cache. Tests use this for isolation.
func ClearCache() {
	repCacheMu.Lock()
	repCache = map[string]*FactionRep{}
	repCacheMu.Unlock()
}

// Test-only seams.

func clearRepCacheForTest()                                   { ClearCache() }
func repCacheStoreForTest(fr *FactionRep) {
	repCacheMu.Lock()
	repCache[fr.FactionId] = fr
	repCacheMu.Unlock()
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/factions/... -v`
Expected: all pass.

- [ ] **Step 5: gofmt + vet**

Run: `gofmt -w internal/factions/ && go vet ./internal/factions/...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add internal/factions/persistence.go internal/factions/persistence_test.go
git commit -m "$(cat <<'EOF'
feat(factions): YAML persistence and rep cache

repPath, loadRepFromDisk, saveRepToDisk, ClearCache, SaveAllRep +
saveMu (serializes file I/O on Windows) and the
DOGMUD_FACTIONS_REP_DIR_OVERRIDE env hook for tests. Mirrors the
internal/opinions persistence pattern verbatim.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Public API — `GetRep`, `SetRep`, `BumpRep`, `TierFor`

**Files:**
- Modify: `internal/factions/factions.go`
- Modify: `internal/factions/factions_test.go`

- [ ] **Step 1: Append failing tests**

Append to `internal/factions/factions_test.go`:

```go
import "github.com/GoMudEngine/GoMud/internal/opinions"

func setupTestFaction(t *testing.T, slug string, defaultRep int) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", dir)
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", t.TempDir())
	body := "faction_id: " + slug + "\n" +
		"display_name: \"" + slug + "\"\n" +
		"description: \"x\"\n" +
		"default_rep: " + intToStr(defaultRep) + "\n" +
		"allies: []\n" +
		"enemies: []\n"
	if err := os.WriteFile(filepath.Join(dir, slug+".yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	clearRegistryForTest()
	clearRepCacheForTest()
	if err := LoadAllDefinitions(); err != nil {
		t.Fatal(err)
	}
	roundForTest = func() uint64 { return 1000 }
	t.Cleanup(func() { roundForTest = nil })
}

func intToStr(n int) string {
	return fmt.Sprintf("%d", n)
}

func TestGetRep_ReturnsDefaultWhenNoRow(t *testing.T) {
	setupTestFaction(t, "warren", -25)
	if got := GetRep("warren", 17); got != -25 {
		t.Errorf("GetRep with no row = %d, want -25 (default_rep)", got)
	}
}

func TestGetRep_UnknownFactionReturnsZero(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	if got := GetRep("nonexistent", 17); got != 0 {
		t.Errorf("GetRep on unknown faction = %d, want 0", got)
	}
}

func TestSetRep_PersistsAndClamps(t *testing.T) {
	setupTestFaction(t, "warren", 0)

	SetRep("warren", 17, 50)
	if got := GetRep("warren", 17); got != 50 {
		t.Errorf("after Set 50, GetRep = %d, want 50", got)
	}

	SetRep("warren", 17, 500)
	if got := GetRep("warren", 17); got != 100 {
		t.Errorf("after Set 500 (clamped), GetRep = %d, want 100", got)
	}

	SetRep("warren", 17, -500)
	if got := GetRep("warren", 17); got != -100 {
		t.Errorf("after Set -500 (clamped), GetRep = %d, want -100", got)
	}

	// Drop cache, reload from disk.
	ClearCache()
	if got := GetRep("warren", 17); got != -100 {
		t.Errorf("after restart-equivalent, GetRep = %d, want -100", got)
	}
}

func TestBumpRep_StartsFromDefaultWhenNoRow(t *testing.T) {
	setupTestFaction(t, "warren", -25)
	BumpRep("warren", 17, 30)
	// -25 + 30 = +5
	if got := GetRep("warren", 17); got != 5 {
		t.Errorf("after first Bump from default, GetRep = %d, want 5", got)
	}
}

func TestBumpRep_AccumulatesAndClamps(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	for i := 0; i < 12; i++ {
		BumpRep("warren", 17, 10)
	}
	// 12 * 10 = 120, clamped to 100
	if got := GetRep("warren", 17); got != 100 {
		t.Errorf("after 12 bumps of 10, GetRep = %d, want 100 (clamped)", got)
	}
}

func TestBumpRep_UnknownFactionIsNoop(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	BumpRep("nonexistent", 17, 50)
	// No panic; nothing else to assert (no rep file should be written).
}

func TestTierFor(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	SetRep("warren", 17, -60)
	if got := TierFor("warren", 17); got != opinions.TierHostile {
		t.Errorf("after Set -60, TierFor = %v, want Hostile", got)
	}
	SetRep("warren", 17, 25)
	if got := TierFor("warren", 17); got != opinions.TierWarm {
		t.Errorf("after Set 25, TierFor = %v, want Warm", got)
	}
}
```

Add `"fmt"`, `"os"`, `"path/filepath"` to the test file's imports if not already present.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/factions/... -v`
Expected: FAIL — `GetRep`, `SetRep`, `BumpRep`, `TierFor`, `roundForTest` undefined.

- [ ] **Step 3: Replace `factions.go` stub with the API**

```go
package factions

import (
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Test-only seam — overrides util.GetRoundCount(). Production code
// never sets this.
var roundForTest func() uint64

func currentRound() uint64 {
	if roundForTest != nil {
		return roundForTest()
	}
	return util.GetRoundCount()
}

// loadOrLazyInit returns the cached *FactionRep for factionId,
// loading from disk on first access. If neither cache nor disk
// has data, an empty FactionRep is created and cached. Returns
// nil if the faction is unknown (no definition loaded).
func loadOrLazyInit(factionId string) *FactionRep {
	repCacheMu.RLock()
	if fr, ok := repCache[factionId]; ok {
		repCacheMu.RUnlock()
		return fr
	}
	repCacheMu.RUnlock()

	def := GetDefinition(factionId)
	if def == nil {
		return nil
	}

	if fr := loadRepFromDisk(factionId); fr != nil {
		repCacheMu.Lock()
		repCache[factionId] = fr
		repCacheMu.Unlock()
		return fr
	}

	fr := &FactionRep{
		FactionId: factionId,
		Players:   map[int]*RepEntry{},
	}
	repCacheMu.Lock()
	repCache[factionId] = fr
	repCacheMu.Unlock()
	return fr
}

// GetRep returns the player's reputation with the given faction,
// or the faction's DefaultRep if no row exists. Pure read — does
// not write to disk. Lazy cache priming may add an empty
// FactionRep to the cache on first access. Returns 0 if the
// faction is unknown.
func GetRep(factionId string, userId int) int {
	def := GetDefinition(factionId)
	if def == nil {
		return 0
	}
	fr := loadOrLazyInit(factionId)
	if fr == nil {
		return 0
	}
	repCacheMu.RLock()
	entry, has := fr.Players[userId]
	repCacheMu.RUnlock()
	if !has {
		return def.DefaultRep
	}
	return entry.Rep
}

// SetRep assigns an absolute rep, clamped to [-100, +100], stamps
// last_updated_round, and persists synchronously.
func SetRep(factionId string, userId int, rep int) {
	fr := loadOrLazyInit(factionId)
	if fr == nil {
		return
	}
	clamped := clampRep(rep)
	now := currentRound()

	repCacheMu.Lock()
	if fr.Players == nil {
		fr.Players = map[int]*RepEntry{}
	}
	fr.Players[userId] = &RepEntry{Rep: clamped, LastUpdatedRound: now}
	repCacheMu.Unlock()

	if err := saveRepToDisk(factionId); err != nil {
		mudlog.Warn("factions.SetRep: saveRepToDisk", "factionId", factionId, "userId", userId, "error", err)
	}
}

// BumpRep adds delta to current rep, clamps, re-stamps, persists.
// No-op if the faction is unknown.
func BumpRep(factionId string, userId int, delta int) {
	fr := loadOrLazyInit(factionId)
	if fr == nil {
		mudlog.Warn("factions.BumpRep: unknown faction", "factionId", factionId)
		return
	}
	def := GetDefinition(factionId)
	now := currentRound()

	repCacheMu.Lock()
	if fr.Players == nil {
		fr.Players = map[int]*RepEntry{}
	}
	entry, has := fr.Players[userId]
	var base int
	if has {
		base = entry.Rep
	} else {
		base = def.DefaultRep
	}
	newRep := clampRep(base + delta)
	fr.Players[userId] = &RepEntry{Rep: newRep, LastUpdatedRound: now}
	repCacheMu.Unlock()

	if err := saveRepToDisk(factionId); err != nil {
		mudlog.Warn("factions.BumpRep: saveRepToDisk", "factionId", factionId, "userId", userId, "error", err)
	}
}

// TierFor returns opinions.TierOf(GetRep(factionId, userId)).
// Reuses the opinion package's banding logic — no duplicate enum.
func TierFor(factionId string, userId int) opinions.Tier {
	return opinions.TierOf(GetRep(factionId, userId))
}

func clampRep(r int) int {
	if r < RepMin {
		return RepMin
	}
	if r > RepMax {
		return RepMax
	}
	return r
}
```

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/factions/... -v`
Expected: all pass (existing + new).

- [ ] **Step 5: gofmt + vet + build**

Run: `gofmt -w internal/factions/ && go vet ./internal/factions/... && go build ./...`
Expected: all clean.

- [ ] **Step 6: Commit**

```bash
git add internal/factions/factions.go internal/factions/factions_test.go
git commit -m "$(cat <<'EOF'
feat(factions): public API — GetRep, SetRep, BumpRep, TierFor

GetRep is a pure read returning the faction's DefaultRep when no
row exists. SetRep clamps and persists synchronously. BumpRep
reads-then-adds-then-clamps-then-stamps under the cache lock.
TierFor delegates to opinions.TierOf — single source of truth for
tier banding. Test seam roundForTest mirrors opinions package.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: `FactionsForMob` + `IsPeacefulToward`

**Files:**
- Modify: `internal/factions/factions.go`
- Modify: `internal/factions/factions_test.go`

- [ ] **Step 1: Append failing tests**

Append to `internal/factions/factions_test.go`:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestFactionsForMob_OnlyReturnsDefinedFactions(t *testing.T) {
	setupTestFaction(t, "warren", 0)

	mob := &mobs.Mob{
		Groups: []string{"humanoid", "warren", "undocumented_group"},
	}
	got := FactionsForMob(mob)
	if len(got) != 1 || got[0] != "warren" {
		t.Errorf("FactionsForMob = %v, want [warren]", got)
	}
}

func TestFactionsForMob_EmptyForNoMatches(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	mob := &mobs.Mob{Groups: []string{"humanoid", "undead"}}
	if got := FactionsForMob(mob); len(got) != 0 {
		t.Errorf("FactionsForMob = %v, want []", got)
	}
}

func TestIsPeacefulToward_FalseAtNeutral(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	mob := &mobs.Mob{Groups: []string{"warren"}}
	if IsPeacefulToward(mob, 17) {
		t.Errorf("IsPeacefulToward at default 0 should be false (Neutral tier)")
	}
}

func TestIsPeacefulToward_TrueAtWarmTier(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	SetRep("warren", 17, 30)
	mob := &mobs.Mob{Groups: []string{"warren"}}
	if !IsPeacefulToward(mob, 17) {
		t.Errorf("IsPeacefulToward at rep 30 (Warm) should be true")
	}
}

func TestIsPeacefulToward_AnyMatchingFactionTriggers(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	// Set up a second faction inline.
	dir := os.Getenv("DOGMUD_FACTIONS_DIR_OVERRIDE")
	if err := os.WriteFile(filepath.Join(dir, "thornwall_guards.yaml"),
		[]byte(`faction_id: thornwall_guards
display_name: "Thornwall Guards"
description: "x"
default_rep: 0
allies: []
enemies: []
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := LoadAllDefinitions(); err != nil {
		t.Fatal(err)
	}
	SetRep("warren", 17, 30) // Warm with warren only
	mob := &mobs.Mob{Groups: []string{"warren", "thornwall_guards"}}
	if !IsPeacefulToward(mob, 17) {
		t.Errorf("IsPeacefulToward should be true if any defined faction is Warm+")
	}
}

func TestIsPeacefulToward_FalseWhenMobHasNoDefinedFactions(t *testing.T) {
	setupTestFaction(t, "warren", 0)
	mob := &mobs.Mob{Groups: []string{"humanoid"}}
	if IsPeacefulToward(mob, 17) {
		t.Errorf("IsPeacefulToward with no defined factions should be false")
	}
	_ = characters.Character{} // keep import alive
}
```

Update imports at top of test file: add `"github.com/GoMudEngine/GoMud/internal/characters"` and `"github.com/GoMudEngine/GoMud/internal/mobs"`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/factions/... -run "TestFactionsForMob|TestIsPeaceful" -v`
Expected: FAIL — undefined functions.

- [ ] **Step 3: Append helpers to `factions.go`**

```go
import "github.com/GoMudEngine/GoMud/internal/mobs"

// FactionsForMob returns the subset of mob.Groups that have
// definition files. Used by combat hookup (which factions get
// bumped on death) and by IsPeacefulToward.
func FactionsForMob(mob *mobs.Mob) []string {
	if mob == nil {
		return nil
	}
	out := make([]string, 0)
	for _, g := range mob.Groups {
		if GetDefinition(g) != nil {
			out = append(out, g)
		}
	}
	return out
}

// IsPeacefulToward returns true when the player has TierWarm or
// higher with at least one of the mob's defined factions. Used by
// lookfortrouble.go and behaviortree/actions_party.go to gate
// whether the mob initiates combat.
func IsPeacefulToward(mob *mobs.Mob, userId int) bool {
	for _, fid := range FactionsForMob(mob) {
		if TierFor(fid, userId) >= opinions.TierWarm {
			return true
		}
	}
	return false
}
```

Note: the imports should consolidate at the top of `factions.go` — merge `mobs` into the existing import block.

- [ ] **Step 4: Run tests to verify pass**

Run: `go test ./internal/factions/... -v`
Expected: all pass.

- [ ] **Step 5: gofmt + vet + build**

Run: `gofmt -w internal/factions/ && go vet ./internal/factions/... && go build ./...`
Expected: clean.

- [ ] **Step 6: Commit**

```bash
git add internal/factions/factions.go internal/factions/factions_test.go
git commit -m "$(cat <<'EOF'
feat(factions): FactionsForMob + IsPeacefulToward

FactionsForMob filters mob.Groups to those with definition files
(only some groups are factions). IsPeacefulToward gates hostility:
true when the player is Warm tier or higher with any of the mob's
defined factions. Replaces the lookfortrouble peacefulquest gate
in the next task.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Concurrency safety test

**Files:**
- Modify: `internal/factions/factions_test.go`

- [ ] **Step 1: Append the parallel-bump test**

```go
import "sync"

func TestParallelBumpsConverge(t *testing.T) {
	setupTestFaction(t, "warren", 0)

	const goroutines = 10
	const bumpsPer = 100
	const delta = 1

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < bumpsPer; j++ {
				BumpRep("warren", 17, delta)
			}
		}()
	}
	wg.Wait()

	want := goroutines * bumpsPer
	if want > RepMax {
		want = RepMax
	}
	if got := GetRep("warren", 17); got != want {
		t.Errorf("after %d parallel bumps: got %d, want %d", goroutines*bumpsPer, got, want)
	}
}
```

Add `"sync"` import if not already present.

- [ ] **Step 2: Run with race detector (if available) or without**

Run: `go test ./internal/factions/... -run TestParallelBumpsConverge -v`
Expected: PASS. If `-race` is available (Linux/macOS): `go test ./internal/factions/... -race -run TestParallelBumpsConverge -v` — should produce no race warnings.

- [ ] **Step 3: Run the full opinions+factions suites a few times to confirm stability** (the same Windows file-lock concurrency issue from chunk 1.1 applies):

```bash
for i in 1 2 3 4 5; do go test ./internal/factions/... -count=1 2>&1 | tail -1; done
```

Expected: every run passes.

- [ ] **Step 4: Commit**

```bash
git add internal/factions/factions_test.go
git commit -m "$(cat <<'EOF'
test(factions): parallel-bump convergence under concurrency

Confirms saveMu correctly serializes file I/O when many goroutines
hit BumpRep on the same (factionId, userId) pair. Mirrors the
opinions package's TestParallelBumpsConverge.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Balance config knob — `FactionMemberKillRep`

**Files:**
- Modify: `internal/configs/config.balance.go`
- Modify: `internal/configs/config.balance.misc.go`

- [ ] **Step 1: Add the field to the `Balance` struct**

Open `internal/configs/config.balance.go`. Find the existing `// ── OPINIONS / DISPOSITION ─` section (added during chunk 1.1 — search for `OpinionAttackBump`). Append a new section after it:

```go
	// ── FACTIONS ─────────────────────────────────────────────────────────────
	FactionMemberKillRep ConfigInt `yaml:"FactionMemberKillRep"` // Rep delta when a player kills a member of a defined faction (default -10)
```

- [ ] **Step 2: Add the default-setter**

Open `internal/configs/config.balance.misc.go`. Find `validateMisc()`. Find the existing `// ── OPINIONS / DISPOSITION ─` block at the bottom. Append a new block after it:

```go
	// ── FACTIONS ─────────────────────────────────────────────────────────────
	if b.FactionMemberKillRep == 0 {
		b.FactionMemberKillRep = -10
	}
```

- [ ] **Step 3: Verify build**

Run: `go build ./... && go test ./internal/configs/... -count=1`
Expected: build succeeds, config tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.misc.go
git commit -m "$(cat <<'EOF'
feat(config): FactionMemberKillRep balance knob

Default -10 — the rep delta applied when a player (or party-room
member) kills a mob with a defined faction. Used by the
MobDeath_FactionRep hook in the next task.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Quest engine `bump_rep` action

**Files:**
- Modify: `internal/questengine/types.go` (or wherever `ActionDef` lives)
- Modify: `internal/questengine/actions.go`
- Modify: `internal/questengine/bridge.go`
- Modify: `internal/questengine/actions_test.go`

- [ ] **Step 1: Locate the existing ActionDef + ActionContext types**

Run: `grep -n "type ActionDef\|type ActionContext\|GrantQuest" internal/questengine/*.go`
Read the file(s) that surface to understand the existing pattern. The `GrantQuest` method on `ActionContext` and the corresponding `Grant string` field on `ActionDef` are the closest precedent.

- [ ] **Step 2: Add the `BumpRepDef` type**

In whichever file holds `ActionDef` (search results from Step 1 will show — likely `internal/questengine/types.go`), add:

```go
// BumpRepDef parameters for the bump_rep action: which faction
// gets the bump and by how much.
type BumpRepDef struct {
	Faction string `yaml:"faction"`
	Delta   int    `yaml:"delta"`
}
```

And add a field to `ActionDef`:

```go
	BumpRep *BumpRepDef `yaml:"bump_rep,omitempty"`
```

- [ ] **Step 3: Add the `BumpRep` method to `ActionContext` interface**

Find `type ActionContext interface { ... }` (likely in `internal/questengine/actions.go`). Add a method:

```go
	BumpRep(factionId string, delta int)
```

- [ ] **Step 4: Add the dispatch case to `ExecuteAction`**

Open `internal/questengine/actions.go`. Find `ExecuteAction`. Add a case (place it near `GrantQuest`/`SetQuestFlag` for proximity):

```go
	if a.BumpRep != nil {
		LogVerboseF(ctx.GetUserId(), "bump_rep %s %+d", a.BumpRep.Faction, a.BumpRep.Delta)
		ctx.BumpRep(a.BumpRep.Faction, a.BumpRep.Delta)
		return nil
	}
```

- [ ] **Step 5: Implement `BumpRep` on the real bridge**

Open `internal/questengine/bridge.go`. Find the existing `GrantQuest` method on the bridge type. Add:

```go
import "github.com/GoMudEngine/GoMud/internal/factions"

func (b *GameBridge) BumpRep(factionId string, delta int) {
	factions.BumpRep(factionId, b.userId, delta)
}
```

(The bridge method receiver and field name `userId` may differ — match what the existing methods use.)

- [ ] **Step 6: Update any test mocks of `ActionContext`**

Run: `grep -n "GrantQuest\|GiveItem" internal/questengine/*_test.go`
Find existing mock implementations of `ActionContext`. Each will need a `BumpRep` method added. Add a no-op or capturing implementation:

```go
// In whatever mock struct (e.g., fakeActionContext in actions_test.go):
func (f *fakeActionContext) BumpRep(factionId string, delta int) {
	f.bumpedFactions = append(f.bumpedFactions, struct{ Faction string; Delta int }{factionId, delta})
}
```

(The exact struct field name depends on the existing mock. If the mock currently has no fields tracking calls, add a slice field on the struct first.)

- [ ] **Step 7: Add a unit test for the dispatch**

Append to `internal/questengine/actions_test.go`:

```go
func TestExecuteAction_BumpRep(t *testing.T) {
	ctx := &fakeActionContext{userId: 17}
	a := ActionDef{BumpRep: &BumpRepDef{Faction: "warren", Delta: 30}}
	if err := ExecuteAction(a, ctx); err != nil {
		t.Fatalf("ExecuteAction: %v", err)
	}
	if len(ctx.bumpedFactions) != 1 ||
		ctx.bumpedFactions[0].Faction != "warren" ||
		ctx.bumpedFactions[0].Delta != 30 {
		t.Errorf("bump_rep dispatch failed: %+v", ctx.bumpedFactions)
	}
}
```

(Adapt the fake-context field name to whatever's actually used in the test file.)

- [ ] **Step 8: Run tests**

Run: `go test ./internal/questengine/... -count=1 -v`
Expected: all pass.

- [ ] **Step 9: gofmt + vet + build**

Run: `gofmt -w internal/questengine/ && go vet ./... && go build ./...`
Expected: clean.

- [ ] **Step 10: Commit**

```bash
git add internal/questengine/types.go internal/questengine/actions.go internal/questengine/bridge.go internal/questengine/actions_test.go
git commit -m "$(cat <<'EOF'
feat(questengine): bump_rep action

Quest YAML can now include actions like
    - bump_rep: {faction: warren, delta: 30}
which calls factions.BumpRep through the game bridge. Real bridge
forwards to internal/factions; test bridges capture for assertion.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Faction definitions + Thornwall guard tagging + `.gitignore` + roadmap + startup wiring

**Files (all NEW or modify):**
- Create: `_datafiles/world/dogmud/factions/warren.yaml`
- Create: `_datafiles/world/dogmud/factions/thornwall_guards.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_outskirts/92-city_gate_guard.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/94-guard_captain_velk.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/106-city_guard.yaml`
- Modify: `.gitignore`
- Modify: `MOB_ALIVENESS_ROADMAP.md`
- Modify: server startup path (locate during the task)

- [ ] **Step 1: Create `_datafiles/world/dogmud/factions/warren.yaml`**

```yaml
faction_id: warren
display_name: "The Warren"
description: |
  A coalition of warren scouts, warriors, and the tunnel shaman that
  have carved out territory in the Labyrinth of Low Tunnels.
  Surface-dwellers are mistrusted on sight.
default_rep: -25
allies: []
enemies: [thornwall_guards]
```

- [ ] **Step 2: Create `_datafiles/world/dogmud/factions/thornwall_guards.yaml`**

```yaml
faction_id: thornwall_guards
display_name: "Thornwall Guards"
description: |
  The city watch of Thornwall. They keep the streets safe and the
  gates manned. Indifferent to strangers; hostile to enemies of
  the city.
default_rep: 0
allies: []
enemies: [warren]
```

- [ ] **Step 3: Tag the three thornwall guard mobs with the `thornwall_guards` group**

For each of `_datafiles/world/dogmud/mobs/thornwall_outskirts/92-city_gate_guard.yaml`, `_datafiles/world/dogmud/mobs/thornwall_city/94-guard_captain_velk.yaml`, `_datafiles/world/dogmud/mobs/thornwall_city/106-city_guard.yaml`:

Find the existing `groups:` block. It currently looks like:
```yaml
groups:
  - humanoid
```
Update to:
```yaml
groups:
  - humanoid
  - thornwall_guards
```

- [ ] **Step 4: Update `.gitignore`**

Open `.gitignore`. Find the existing entries for `_datafiles/world/dogmud/shops/` (or similar runtime dirs — search for `shops/` and `mobs.instances/`). After the existing block, add:

```
# Runtime faction rep state — not committed
_datafiles/world/dogmud/factions.rep/
```

- [ ] **Step 5: Update `MOB_ALIVENESS_ROADMAP.md`**

Open `MOB_ALIVENESS_ROADMAP.md`. Find the progress tracker table. Mark 1.2 as `In progress`:

```markdown
| 1.2 | Substrate | Faction system | L | 1.1 | In progress |
```

Re-tally the rollup line (one chunk moved from "Not started" to "In progress").

In the **Phase 6** mini-briefs section, AFTER the existing `### 6.5 Content pass` block, insert:

```markdown
### 6.5a Faction definitions content pass
**Status:** Not started • **Size:** M

- **Goal:** Author the rest of the world's factions on top of the
  1.2/1.3 substrate.
- **In:** YAML faction definitions for bandits, warden, ironwind
  shaman, Sanctum Basin guards, Dustwalk caravans, Stillwater
  militia & citizens, etc. Tag remaining faction-relevant mobs
  with their `groups: [<faction_id>]`. Define ally/enemy graphs
  across the full set. Surface any schema gaps the substrate
  didn't anticipate.
- **Out:** Per-faction quests (own content chunk).
- **Depends on:** 1.2, 1.3
- **Why:** 1.2 ships substrate + warren + thornwall_guards. 1.3
  adds thornwall_citizens + alliance-aware guard logic. Bulk
  authoring the rest now would risk schema churn — better to
  validate the substrate against two reference factions, then
  bulk-author once the pattern is settled.
```

Also add a row to the progress tracker table (just above the rollup line):

```markdown
| 6.5a | Polish | Faction definitions content pass | M | 1.2, 1.3 | Not started |
```

Re-tally the rollup (one new chunk added, total goes from 39 to 40).

- [ ] **Step 6: Wire `factions.LoadAllDefinitions()` into server startup**

Find the place where other "load all data files" calls happen. Run:
```bash
grep -rn "mobs.LoadDataFiles\|opinions.\|rooms.LoadDataFiles" world.go main.go internal/ 2>&1 | head -10
```

The most likely location is `world.go` in the project root (which contains `WorldInput` per main.go). Read it to find where data-file loaders run. Add a call:

```go
if err := factions.LoadAllDefinitions(); err != nil {
    mudlog.Error("factions.LoadAllDefinitions", "error", err)
}
```

Place it in the same block that loads other game data, AFTER mobs are loaded if possible (so faction membership errors surface only against loaded mobs).

Add the import to that file:
```go
"github.com/GoMudEngine/GoMud/internal/factions"
```

- [ ] **Step 7: Verify build**

Run: `go build ./...`
Expected: builds cleanly.

- [ ] **Step 8: Commit**

```bash
git add _datafiles/world/dogmud/factions/warren.yaml \
        _datafiles/world/dogmud/factions/thornwall_guards.yaml \
        _datafiles/world/dogmud/mobs/thornwall_outskirts/92-city_gate_guard.yaml \
        _datafiles/world/dogmud/mobs/thornwall_city/94-guard_captain_velk.yaml \
        _datafiles/world/dogmud/mobs/thornwall_city/106-city_guard.yaml \
        .gitignore \
        MOB_ALIVENESS_ROADMAP.md \
        world.go
git commit -m "$(cat <<'EOF'
feat(factions): warren + thornwall_guards definitions, startup load

Two faction definitions ship with chunk 1.2: warren (default -25,
warren↔thornwall_guards enemies) and thornwall_guards (default 0,
narrow membership of just the three guard mobs — thornwall_citizens
deferred to chunk 1.3).

Server startup now calls factions.LoadAllDefinitions(); ally/enemy
references are validated at boot so authoring errors panic before
any player can hit them. .gitignore picks up factions.rep/ as the
runtime-state dir (mirrors shops/, mobs.instances/, etc).

Roadmap gains chunk 6.5a (broader faction content pass) and 1.2 is
flipped to In progress.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Warren mob tagging (additive only — keeps `peacefulquest` for migration safety)

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/72-warren_scout.yaml`
- Modify: `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/73-warren_warrior.yaml`
- Modify: `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/74-tunnel_shaman.yaml`

- [ ] **Step 1: Add `warren` group to mob 72 (warren scout)**

Find the existing `groups:` block in `72-warren_scout.yaml`. Append `warren`:

Before:
```yaml
groups:
  - humanoid
```
After:
```yaml
groups:
  - humanoid
  - warren
```

**Do NOT remove the existing `peacefulquest: "2-end"` line** — it stays in place during this task. Migration cleanup happens in Task 13.

- [ ] **Step 2: Add `warren` group to mob 73 (warren warrior)** — same edit shape.

- [ ] **Step 3: Add `warren` group to mob 74 (tunnel shaman)** — same edit shape. The shaman doesn't currently have `peacefulquest:` set, but does need the warren group.

- [ ] **Step 4: Boot the server and confirm clean load**

Run: `go build -o /tmp/dogmud-test.exe . && /tmp/dogmud-test.exe > /tmp/boot.log 2>&1 &`
Wait until you see "Server Ready" in `/tmp/boot.log` (or until startup panics). If panic, read the panic message — most likely cause is a YAML parse error in the edited mob files. Fix and re-run.

Once Server Ready: kill the process per the SOP (`tasklist | grep dogmud`, kill PID).

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/72-warren_scout.yaml \
        _datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/73-warren_warrior.yaml \
        _datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/74-tunnel_shaman.yaml
git commit -m "$(cat <<'EOF'
feat(mobs): tag warren scouts/warriors/shaman with warren faction

Adds the warren group to mobs 72/73/74 alongside their existing
humanoid group. The peacefulquest: "2-end" line is intentionally
LEFT IN PLACE during this task — Task 13 removes it after the
gate replacement and Migration 0.13.0 land. This split keeps
warren mobs functional throughout the migration.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Modify quest 2 + Migration 0.13.0

**Files:**
- Modify: `_datafiles/world/dogmud/quests/2-the_warren_compact.yaml`
- Create: `internal/migration/0.13.0.go`
- Modify: `internal/migration/migration.go`

- [ ] **Step 1: Add `bump_rep` action to quest 2 end-step**

Open `_datafiles/world/dogmud/quests/2-the_warren_compact.yaml`. Find the `triggers:` block with `event: item_give, mob: 74, item: 30036`. Inside its `actions:` list, after `- grant: 2-end` and before the existing `- npc_say:` block, add:

```yaml
      - bump_rep: {faction: warren, delta: 30}
```

The full surrounding context after the edit looks like:

```yaml
triggers:
  - event: item_give
    mob: 74
    item: 30036
    conditions:
      has: [2-start]
      missing: [2-end]
    actions:
      - grant: 2-end
      - bump_rep: {faction: warren, delta: 30}
      - npc_say:
          mob: 74
          lines:
            - {delay: 1, text: "These are good. Strong. Our sick will benefit."}
            - {delay: 3, text: "..."}
```

- [ ] **Step 2: Create `internal/migration/0.13.0.go`**

```go
package migration

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// Description:
// Seeds warren faction rep for any user holding the legacy
// peacefulquest token "2-end" (The Warren Compact). Without this
// migration, players who completed quest 2 pre-deploy would lose
// their hostility immunity from warren mobs after the
// peacefulquest field is removed in chunk 1.2.
func migrate_SeedWarrenRepFromQuestToken() error {
	c := configs.GetConfig()

	usersGlob := filepath.Join(string(c.FilePaths.DataFiles), "users", "*.yaml")
	matches, err := filepath.Glob(usersGlob)
	if err != nil {
		return err
	}

	mudlog.Info("Migration 0.13.0", "message", "Seeding warren faction rep from legacy 2-end quest token")

	for _, userFilePath := range matches {
		if filepath.Base(userFilePath) == "users.idx" {
			continue
		}

		raw, err := os.ReadFile(userFilePath)
		if err != nil {
			mudlog.Warn("Migration 0.13.0", "file", userFilePath, "error", err)
			continue
		}

		// Parse only the fields we need.
		var u struct {
			UserId    int `yaml:"userid"`
			Character struct {
				QuestProgress map[string]int `yaml:"questprogress"`
			} `yaml:"character"`
		}
		if err := yaml.Unmarshal(raw, &u); err != nil {
			mudlog.Warn("Migration 0.13.0", "file", userFilePath, "error", err)
			continue
		}

		mudlog.Info("Migration 0.13.0", "file", userFilePath, "message", "checking user file")

		if u.UserId <= 0 {
			mudlog.Info("Migration 0.13.0", "file", userFilePath, "message", "no userid, skipping")
			continue
		}
		if _, has := u.Character.QuestProgress["2-end"]; !has {
			mudlog.Info("Migration 0.13.0", "file", userFilePath, "message", "no 2-end token, skipping")
			continue
		}

		// Only seed if no warren rep entry exists yet (idempotency).
		if existing := factions.GetRep("warren", u.UserId); existing != factions.GetDefinition("warren").DefaultRep {
			mudlog.Info("Migration 0.13.0", "file", userFilePath,
				"message", fmt.Sprintf("warren rep already at %d, skipping seed", existing))
			continue
		}

		factions.BumpRep("warren", u.UserId, 30)
		mudlog.Info("Migration 0.13.0", "file", userFilePath,
			"message", fmt.Sprintf("seeded warren rep +30 for userId %d", u.UserId))
	}

	mudlog.Info("Migration 0.13.0", "message", "Warren rep seeding complete!")
	return nil
}
```

(NOTE: the user-file struct may use a different YAML key for the quest-token storage — `questprogress` is the most likely name based on common conventions. Verify by reading one user file at `_datafiles/world/dogmud/users/17.yaml` and adjusting the struct field/yaml tag accordingly. The user yaml from chunk 1.1 work showed fields like `userid`, `character`, `name`, `roomid`. Find the quest-token field in a user yaml that has any quest progress, and match the struct accordingly. If quest tokens live under `character.quests` or similar, adjust.)

- [ ] **Step 3: Wire 0.13.0 into the migration registry**

Open `internal/migration/migration.go`. Find `doAllMigrations`. After the existing `// 0.11.0 -> 0.12.0` block, add:

```go
	// 0.12.0 -> 0.13.0
	if lastConfigVersion.IsOlderThan(version.New(0, 13, 0)) {

		// Faction rep system needs definitions loaded before the
		// migration can call factions.BumpRep. Load now.
		if err := factions.LoadAllDefinitions(); err != nil {
			return err
		}

		if err := migrate_SeedWarrenRepFromQuestToken(); err != nil {
			return err
		}

	}
```

Add the `factions` import to the top of `migration.go`:

```go
"github.com/GoMudEngine/GoMud/internal/factions"
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/quests/2-the_warren_compact.yaml \
        internal/migration/0.13.0.go \
        internal/migration/migration.go
git commit -m "$(cat <<'EOF'
feat(quest+migration): warren compact awards faction rep + migrate live players

Quest 2 (The Warren Compact) end-step now fires
    bump_rep: {faction: warren, delta: 30}
alongside the existing 2-end token grant. Token is preserved for
any other dialogue/quest gates that may reference it.

Migration 0.13.0 seeds warren rep for any user holding the legacy
2-end token at deploy time, so existing peace-makers keep their
hostility immunity. Idempotent — re-running doesn't double-seed.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Replace the `peacefulquest` gate in `lookfortrouble` and `actions_party`

**Files:**
- Modify: `internal/mobcommands/lookfortrouble.go`
- Modify: `internal/behaviortree/actions_party.go`

- [ ] **Step 1: Replace the gate in `lookfortrouble.go`**

Open `internal/mobcommands/lookfortrouble.go`. Find the lines:

```go
// peacefulquest: mob won't attack players who have this token
if mob.PeacefulQuest != "" && user.Character.HasQuest(mob.PeacefulQuest) {
    continue
}
```

Replace with:

```go
// Faction-rep peace gate: skip players who are at TierWarm or
// higher with at least one of the mob's defined factions.
if factions.IsPeacefulToward(mob, user.UserId) {
    continue
}
```

Add the import:
```go
"github.com/GoMudEngine/GoMud/internal/factions"
```

- [ ] **Step 2: Replace the gate in `actions_party.go`**

Open `internal/behaviortree/actions_party.go`. Find the lines (around line 237):

```go
// Honor peacefulquest immunity — same gate as mobcommands.LookForTrouble.
if mob.PeacefulQuest != "" && u.Character.HasQuest(mob.PeacefulQuest) {
    // ... whatever the body is — likely a continue or skip
}
```

Replace with:

```go
// Honor faction-rep peace gate — same as mobcommands.LookForTrouble.
if factions.IsPeacefulToward(mob, u.UserId) {
    // ... preserve the existing body verbatim
}
```

Add the `factions` import to this file too.

- [ ] **Step 3: Verify build + tests**

Run: `go build ./... && go test ./internal/mobcommands/... ./internal/behaviortree/... -count=1`
Expected: clean build, tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/mobcommands/lookfortrouble.go internal/behaviortree/actions_party.go
git commit -m "$(cat <<'EOF'
refactor(mobs): replace peacefulquest gate with factions.IsPeacefulToward

lookfortrouble and behaviortree.actions_party now consult the new
faction-rep system instead of the old peacefulquest token check.
The peacefulquest field on Mob is unused after this commit but
not yet deleted (Task 13 handles the YAML cleanup + struct
deletion).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 13: Drop `peacefulquest` from warren mob YAMLs + delete `Mob.PeacefulQuest` field

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/72-warren_scout.yaml`
- Modify: `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/73-warren_warrior.yaml`
- Modify: `internal/mobs/mobs.go`

- [ ] **Step 1: Delete the `peacefulquest:` line from mob 72**

Open `_datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/72-warren_scout.yaml`. Find:

```yaml
peacefulquest: "2-end"
```

Delete that entire line (and any blank line that becomes redundant).

- [ ] **Step 2: Delete the `peacefulquest:` line from mob 73** — same edit.

- [ ] **Step 3: Delete the `PeacefulQuest` field from `Mob`**

Open `internal/mobs/mobs.go`. Find:

```go
PeacefulQuest   string   `yaml:"peacefulquest,omitempty"`    // if set, mob won't attack players who have this quest token
```

Delete that line.

- [ ] **Step 4: Verify build (catches any stragglers)**

Run: `go build ./...`
Expected: clean. If anything errors with `mob.PeacefulQuest undefined`, that's a missed call site — grep for it: `grep -rn "PeacefulQuest" internal/`. Fix any remaining references.

- [ ] **Step 5: Run full test suite**

Run: `go test ./... -count=1 2>&1 | grep -E "FAIL|^---"`
Expected: no failures.

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/72-warren_scout.yaml \
        _datafiles/world/dogmud/mobs/labyrinth_of_low_tunnels/73-warren_warrior.yaml \
        internal/mobs/mobs.go
git commit -m "$(cat <<'EOF'
refactor(mobs): delete legacy peacefulquest field

The faction-rep gate (Task 12) replaces all uses of peacefulquest.
Existing player immunity is preserved via Migration 0.13.0 which
seeds warren rep at +30 for anyone holding the legacy 2-end token.

YAML files for warren scouts/warriors no longer carry the
peacefulquest line; the Mob struct field is gone too.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 14: Combat hookup — `MobDeath_FactionRep`

**Files:**
- Create: `internal/hooks/MobDeath_FactionRep.go`
- Create: `internal/hooks/MobDeath_FactionRep_test.go`
- Modify: server startup or hook registration (locate during task)

- [ ] **Step 1: Find where existing MobDeath hooks are registered**

Run: `grep -rn "MobDeath_QuestNotify\|MobDeath_PackFlee\|MobDeath_CompanionCleanup" --include="*.go" | head -10`

The output should show the registration site (likely `world.go` or a setup function in `internal/hooks/`). Read the registration block to understand the pattern (e.g., `events.AddListener(events.MobDeath{}.Type(), MobDeathQuestNotify)`).

- [ ] **Step 2: Write the failing test**

Create `internal/hooks/MobDeath_FactionRep_test.go`. Match the pattern of an existing MobDeath test if any exists (`grep -l "MobDeath" internal/hooks/*_test.go` first). If no existing test, scaffold from scratch:

```go
package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// helper: write a faction definition into the test override dir.
func writeFactionDef(t *testing.T, slug, body string) {
	t.Helper()
	dir := os.Getenv("DOGMUD_FACTIONS_DIR_OVERRIDE")
	if err := os.WriteFile(filepath.Join(dir, slug+".yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func setupFactionsForHookTest(t *testing.T) {
	t.Helper()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", t.TempDir())
	writeFactionDef(t, "warren", `
faction_id: warren
display_name: "The Warren"
description: "x"
default_rep: 0
allies: []
enemies: []
`)
	if err := factions.LoadAllDefinitions(); err != nil {
		t.Fatal(err)
	}
	factions.ClearCache()
}

func TestMobDeathFactionRep_BumpsForDamager(t *testing.T) {
	setupFactionsForHookTest(t)

	mob := &mobs.Mob{
		MobId:      72,
		InstanceId: 200,
		Groups:     []string{"warren"},
		Character: characters.Character{
			Name:   "warren scout",
			RoomId: 301,
			Buffs:  buffs.New(),
		},
	}
	mobs.SetInstanceForTest(200, mob)
	defer mobs.SetInstanceForTest(200, nil)

	evt := events.MobDeath{
		MobId:        72,
		InstanceId:   200,
		RoomId:       301,
		PlayerDamage: map[int]int{17: 50},
	}
	MobDeathFactionRep(evt)

	want := int(configs.GetBalanceConfig().FactionMemberKillRep)
	if got := factions.GetRep("warren", 17); got != want {
		t.Errorf("after kill, warren rep for user 17 = %d, want %d", got, want)
	}
}

func TestMobDeathFactionRep_NoFactionsNoChange(t *testing.T) {
	setupFactionsForHookTest(t)

	mob := &mobs.Mob{
		MobId:      999,
		InstanceId: 201,
		Groups:     []string{"humanoid"}, // no defined faction
		Character:  characters.Character{Name: "x", RoomId: 301, Buffs: buffs.New()},
	}
	mobs.SetInstanceForTest(201, mob)
	defer mobs.SetInstanceForTest(201, nil)

	evt := events.MobDeath{
		MobId:        999,
		InstanceId:   201,
		RoomId:       301,
		PlayerDamage: map[int]int{17: 50},
	}
	MobDeathFactionRep(evt)

	if got := factions.GetRep("warren", 17); got != 0 {
		t.Errorf("warren rep changed for non-faction kill: %d", got)
	}
}

func TestMobDeathFactionRep_NoPlayersNoChange(t *testing.T) {
	setupFactionsForHookTest(t)

	mob := &mobs.Mob{
		MobId:      72,
		InstanceId: 202,
		Groups:     []string{"warren"},
		Character:  characters.Character{Name: "scout", RoomId: 301, Buffs: buffs.New()},
	}
	mobs.SetInstanceForTest(202, mob)
	defer mobs.SetInstanceForTest(202, nil)

	evt := events.MobDeath{
		MobId:        72,
		InstanceId:   202,
		RoomId:       301,
		PlayerDamage: map[int]int{}, // env damage / no contributors
	}
	MobDeathFactionRep(evt)

	if got := factions.GetRep("warren", 17); got != 0 {
		t.Errorf("warren rep changed despite no damager: %d", got)
	}
}
```

The test file uses helpers (`mobs.SetInstanceForTest` already exists from chunk 1.1 work; `buffs.New()` etc. are existing). Verify all imports compile before proceeding.

- [ ] **Step 3: Run tests to verify failure**

Run: `go test ./internal/hooks/... -run "TestMobDeathFactionRep" -v`
Expected: FAIL — `MobDeathFactionRep` undefined.

- [ ] **Step 4: Implement `MobDeath_FactionRep.go`**

Create `internal/hooks/MobDeath_FactionRep.go`:

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// MobDeathFactionRep bumps faction rep downward for every player
// who participated in killing a mob with at least one defined
// faction. Participation = direct damage (PlayerDamage map) OR
// being in the same room as the death AND in the killer's party.
//
// Magnitude: Balance.FactionMemberKillRep (default -10) per
// affected (player, faction) pair. Saturates Hostile after ~10
// kills of the same faction.
func MobDeathFactionRep(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}

	if len(evt.PlayerDamage) == 0 {
		return events.Continue
	}

	mob := mobs.GetInstance(evt.InstanceId)
	if mob == nil {
		return events.Continue
	}

	factionIds := factions.FactionsForMob(mob)
	if len(factionIds) == 0 {
		return events.Continue
	}

	delta := int(configs.GetBalanceConfig().FactionMemberKillRep)

	// Build set of users to bump: direct damagers + party members in
	// the death room.
	toBump := make(map[int]struct{}, len(evt.PlayerDamage))
	for userId := range evt.PlayerDamage {
		toBump[userId] = struct{}{}
	}
	for damagerUserId := range evt.PlayerDamage {
		party := parties.Get(damagerUserId)
		if party == nil {
			continue
		}
		for _, memberId := range party.UserIds {
			if _, already := toBump[memberId]; already {
				continue
			}
			u := users.GetByUserId(memberId)
			if u == nil {
				continue
			}
			if u.Character.RoomId == evt.RoomId {
				toBump[memberId] = struct{}{}
			}
		}
	}

	for userId := range toBump {
		for _, fid := range factionIds {
			factions.BumpRep(fid, userId, delta)
		}
	}

	return events.Continue
}
```

- [ ] **Step 5: Register the hook on server startup**

Find the existing MobDeath registration (from Step 1). Add a registration line in the same block:

```go
events.AddListener(events.MobDeath{}.Type(), MobDeathFactionRep)
```

- [ ] **Step 6: Run tests to verify pass**

Run: `go test ./internal/hooks/... -run "TestMobDeathFactionRep" -v`
Expected: all three new tests pass.

- [ ] **Step 7: Run the full test suite**

Run: `go test ./... -count=1 2>&1 | grep -E "FAIL|^---"`
Expected: no failures.

- [ ] **Step 8: Commit**

```bash
git add internal/hooks/MobDeath_FactionRep.go internal/hooks/MobDeath_FactionRep_test.go
# Plus whichever file got the registration line added
git commit -m "$(cat <<'EOF'
feat(combat): MobDeath_FactionRep — kill-faction-member rep bump

When a mob with one or more defined factions dies, every player
who damaged it (per evt.PlayerDamage) plus every party member in
the same room takes the FactionMemberKillRep delta (-10 by
default) on the dead mob's faction(s). Pure-support healers in a
party still get the hit if they were in the room; offline or
distant party members do not.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 15: Admin command + helpfile

**Files:**
- Create: `internal/usercommands/admin.faction.go`
- Create: `internal/usercommands/admin.faction_test.go`
- Create: `_datafiles/world/dogmud/templates/admincommands/help/command.faction.template`
- Modify: `internal/usercommands/usercommands.go`

- [ ] **Step 1: Skim a peer admin command for shape**

Read `internal/usercommands/admin.opinion.go` end-to-end — that's the closest analog (chunk 1.1). The faction command is structurally similar with `list` added.

- [ ] **Step 2: Write the failing tests**

Create `internal/usercommands/admin.faction_test.go`:

```go
package usercommands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func setupFactionsForAdminTest(t *testing.T) {
	t.Helper()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", t.TempDir())
	dir := os.Getenv("DOGMUD_FACTIONS_DIR_OVERRIDE")
	body := `faction_id: warren
display_name: "The Warren"
description: "x"
default_rep: -25
allies: []
enemies: []
`
	if err := os.WriteFile(filepath.Join(dir, "warren.yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	if err := factions.LoadAllDefinitions(); err != nil {
		t.Fatal(err)
	}
	factions.ClearCache()
}

func TestAdminFaction_SetAndShow(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	setupFactionsForAdminTest(t)

	admin, room := getTestUserAndRoom(t)
	target := users.GetByUserId(1)
	if target == nil {
		t.Fatal("test user 1 missing")
	}

	cmd := "set warren " + target.Character.Name + " 50"
	if _, err := Faction(cmd, admin, room, 0); err != nil {
		t.Fatalf("faction set: %v", err)
	}
	if got := factions.GetRep("warren", target.UserId); got != 50 {
		t.Errorf("after set, GetRep = %d, want 50", got)
	}
}

func TestAdminFaction_Bump(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	setupFactionsForAdminTest(t)

	admin, room := getTestUserAndRoom(t)
	target := users.GetByUserId(1)

	if _, err := Faction("bump warren "+target.Character.Name+" 20", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	// default_rep -25, +20 → -5
	if got := factions.GetRep("warren", target.UserId); got != -5 {
		t.Errorf("after bump 20 from default -25, GetRep = %d, want -5", got)
	}
}

func TestAdminFaction_Reset(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	setupFactionsForAdminTest(t)

	admin, room := getTestUserAndRoom(t)
	target := users.GetByUserId(1)

	factions.SetRep("warren", target.UserId, 80)
	if _, err := Faction("reset warren "+target.Character.Name, admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if got := factions.GetRep("warren", target.UserId); got != -25 {
		t.Errorf("after reset, GetRep = %d, want -25 (default)", got)
	}
}

func TestAdminFaction_List(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	setupFactionsForAdminTest(t)

	admin, room := getTestUserAndRoom(t)
	if _, err := Faction("list", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	// We can't easily capture user.SendText output in tests; the
	// behavioral assertion is "no panic, returns true". Future
	// improvement: capture and assert content.
}

func TestAdminFaction_UnknownFaction(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	setupFactionsForAdminTest(t)

	admin, room := getTestUserAndRoom(t)
	target := users.GetByUserId(1)

	// Should produce a friendly error, not a panic.
	if _, err := Faction("set nonexistent "+target.Character.Name+" 50", admin, room, 0); err != nil {
		t.Fatalf("faction set on unknown faction returned error: %v", err)
	}
	if got := factions.GetRep("nonexistent", target.UserId); got != 0 {
		t.Errorf("unknown faction should return 0, got %d", got)
	}
}
```

- [ ] **Step 3: Run tests to verify failure**

Run: `go test ./internal/usercommands/... -run "TestAdminFaction" -v`
Expected: FAIL — `Faction` undefined.

- [ ] **Step 4: Create `admin.faction.go`**

```go
package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
 * Role Permissions:
 * faction          (Admin)
 */

func Faction(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		factionShowUsage(user)
		return true, nil
	}

	switch strings.ToLower(args[0]) {
	case "list":
		return factionListAll(user)
	case "show":
		return factionShow(args[1:], user)
	case "set":
		return factionMutate(args[1:], user, factionMutateSet)
	case "bump":
		return factionMutate(args[1:], user, factionMutateBump)
	case "reset":
		return factionMutate(args[1:], user, factionMutateReset)
	default:
		factionShowUsage(user)
		return true, nil
	}
}

func factionShowUsage(user *users.UserRecord) {
	if out, err := templates.Process("admincommands/help/command.faction", nil, user.UserId); err == nil && strings.TrimSpace(out) != "" {
		user.SendText(out)
		return
	}
	user.SendText(
		"Usage:\r\n" +
			"  faction list\r\n" +
			"  faction show <playerName>\r\n" +
			"  faction show <factionId> <playerName>\r\n" +
			"  faction set <factionId> <playerName> <rep>\r\n" +
			"  faction bump <factionId> <playerName> <delta>\r\n" +
			"  faction reset <factionId> <playerName>\r\n",
	)
}

func factionListAll(user *users.UserRecord) (bool, error) {
	defs := factions.AllDefinitions()
	if len(defs) == 0 {
		user.SendText("No factions loaded.\r\n")
		return true, nil
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].FactionId < defs[j].FactionId })

	var b strings.Builder
	fmt.Fprintf(&b, "Loaded factions:\r\n\r\n")
	fmt.Fprintf(&b, "  %-25s %-25s %-7s %s\r\n", "ID", "Name", "Default", "Allies / Enemies")
	fmt.Fprintf(&b, "  %-25s %-25s %-7s %s\r\n",
		strings.Repeat("-", 25), strings.Repeat("-", 25), "-------", strings.Repeat("-", 30))
	for _, d := range defs {
		fmt.Fprintf(&b, "  %-25s %-25s %-7d allies=%v enemies=%v\r\n",
			factionTruncate(d.FactionId, 25),
			factionTruncate(d.DisplayName, 25),
			d.DefaultRep,
			d.Allies, d.Enemies)
	}
	user.SendText(b.String())
	return true, nil
}

func factionShow(args []string, user *users.UserRecord) (bool, error) {
	switch len(args) {
	case 1:
		return factionShowAllForUser(args[0], user)
	case 2:
		return factionShowOne(args[0], args[1], user)
	default:
		factionShowUsage(user)
		return true, nil
	}
}

func factionShowAllForUser(playerName string, user *users.UserRecord) (bool, error) {
	target := users.GetByCharacterName(playerName)
	if target == nil {
		user.SendText(fmt.Sprintf("No such player: %s\r\n", playerName))
		return true, nil
	}

	defs := factions.AllDefinitions()
	type row struct {
		factionId string
		rep       int
		round     uint64
	}
	rows := make([]row, 0)
	for _, d := range defs {
		rep := factions.GetRep(d.FactionId, target.UserId)
		if rep == d.DefaultRep {
			continue
		}
		rows = append(rows, row{d.FactionId, rep, 0})
	}
	if len(rows) == 0 {
		user.SendText(fmt.Sprintf("No factions hold a non-default rep for %s.\r\n", playerName))
		return true, nil
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].rep < rows[j].rep })

	var b strings.Builder
	fmt.Fprintf(&b, "faction show %s:\r\n\r\n", playerName)
	fmt.Fprintf(&b, "  %-25s %5s  %s\r\n", "Faction", "Rep", "Tier")
	fmt.Fprintf(&b, "  %-25s %5s  %s\r\n", strings.Repeat("-", 25), "-----", "--------")
	for _, r := range rows {
		fmt.Fprintf(&b, "  %-25s %5d  %s\r\n",
			factionTruncate(r.factionId, 25),
			r.rep, factionTierName(opinions.TierOf(r.rep)))
	}
	user.SendText(b.String())
	return true, nil
}

func factionShowOne(factionId, playerName string, user *users.UserRecord) (bool, error) {
	if factions.GetDefinition(factionId) == nil {
		user.SendText(fmt.Sprintf("Unknown faction: %s\r\n", factionId))
		return true, nil
	}
	target := users.GetByCharacterName(playerName)
	if target == nil {
		user.SendText(fmt.Sprintf("No such player: %s\r\n", playerName))
		return true, nil
	}
	rep := factions.GetRep(factionId, target.UserId)
	user.SendText(fmt.Sprintf("%s -> %s: rep=%d, tier=%s\r\n",
		factionId, playerName, rep, factionTierName(opinions.TierOf(rep))))
	return true, nil
}

type factionMutateMode int

const (
	factionMutateSet factionMutateMode = iota
	factionMutateBump
	factionMutateReset
)

func factionMutate(args []string, user *users.UserRecord, mode factionMutateMode) (bool, error) {
	expected := 3
	if mode == factionMutateReset {
		expected = 2
	}
	if len(args) != expected {
		factionShowUsage(user)
		return true, nil
	}
	factionId := args[0]
	if factions.GetDefinition(factionId) == nil {
		user.SendText(fmt.Sprintf("Unknown faction: %s\r\n", factionId))
		return true, nil
	}
	target := users.GetByCharacterName(args[1])
	if target == nil {
		user.SendText(fmt.Sprintf("No such player: %s\r\n", args[1]))
		return true, nil
	}

	switch mode {
	case factionMutateSet:
		v, err := strconv.Atoi(args[2])
		if err != nil {
			user.SendText(fmt.Sprintf("Bad rep %q: %v\r\n", args[2], err))
			return true, nil
		}
		factions.SetRep(factionId, target.UserId, v)
		user.SendText(fmt.Sprintf("Set %s -> %s = %d\r\n", factionId, args[1], v))
	case factionMutateBump:
		v, err := strconv.Atoi(args[2])
		if err != nil {
			user.SendText(fmt.Sprintf("Bad delta %q: %v\r\n", args[2], err))
			return true, nil
		}
		factions.BumpRep(factionId, target.UserId, v)
		user.SendText(fmt.Sprintf("Bumped %s -> %s by %d (now %d)\r\n",
			factionId, args[1], v, factions.GetRep(factionId, target.UserId)))
	case factionMutateReset:
		def := factions.GetDefinition(factionId)
		factions.SetRep(factionId, target.UserId, def.DefaultRep)
		user.SendText(fmt.Sprintf("Reset %s -> %s to default %d\r\n",
			factionId, args[1], def.DefaultRep))
	}
	return true, nil
}

func factionTierName(t opinions.Tier) string {
	switch t {
	case opinions.TierHostile:
		return "Hostile"
	case opinions.TierCold:
		return "Cold"
	case opinions.TierNeutral:
		return "Neutral"
	case opinions.TierWarm:
		return "Warm"
	case opinions.TierFriendly:
		return "Friendly"
	default:
		return "?"
	}
}

func factionTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n < 1 {
		return ""
	}
	return s[:n-1] + "…"
}

// Reference util to keep the import; remove if unused.
var _ = util.GetRoundCount
```

(NOTE: if `factionTierName` collides with an existing `tierName` in the package — there's already one for `opinion` — keeping the prefix avoids collision. The test file doesn't reference it directly, just tests behavioral asserts.)

- [ ] **Step 5: Register the command**

Open `internal/usercommands/usercommands.go`. Find the alphabetical insertion point — between `f`-prefixed entries. Add:

```go
	`faction`: {Faction, true, true, true}, // Admin only
```

- [ ] **Step 6: Create the helpfile**

Create `_datafiles/world/dogmud/templates/admincommands/help/command.faction.template`:

```
The <ansi fg="command">faction</ansi> command inspects and adjusts
per-player reputation with each loaded faction.

<ansi fg="yellow-bold">Usage:</ansi>

<ansi fg="command">faction list</ansi>
  List every loaded faction with its display name, default rep,
  and ally/enemy lists.

<ansi fg="command">faction show <playerName></ansi>
  List every faction that holds a non-default rep for the named
  player, sorted worst to best.

<ansi fg="command">faction show <factionId> <playerName></ansi>
  Show one specific (faction × player) row.

<ansi fg="command">faction set <factionId> <playerName> <rep></ansi>
  Assign an absolute rep (-100 to +100).

<ansi fg="command">faction bump <factionId> <playerName> <delta></ansi>
  Add delta to the current rep. Positive = warmer.

<ansi fg="command">faction reset <factionId> <playerName></ansi>
  Reset rep to the faction's default.

<ansi fg="yellow-bold">Tier thresholds:</ansi>
  Hostile (<= -50)  Cold (-49..-15)  Neutral (-14..+14)
  Warm (+15..+49)   Friendly (>= +50)

<ansi fg="yellow-bold">Notes:</ansi>
  - Faction rep does NOT decay (unlike per-NPC opinion). It is
    institutional memory.
  - Changes persist immediately to disk under
    _datafiles/world/dogmud/factions.rep/<faction>.yaml.
  - Players never see numeric rep — relationships surface
    through dialogue, NPC behavior, and fiction.
```

- [ ] **Step 7: Run tests**

Run: `go test ./internal/usercommands/... -run "TestAdminFaction" -v`
Expected: all pass.

- [ ] **Step 8: Run full suite**

Run: `go test ./... -count=1 2>&1 | grep -E "FAIL|^---"`
Expected: no failures.

- [ ] **Step 9: Commit**

```bash
git add internal/usercommands/admin.faction.go \
        internal/usercommands/admin.faction_test.go \
        internal/usercommands/usercommands.go \
        _datafiles/world/dogmud/templates/admincommands/help/command.faction.template
git commit -m "$(cat <<'EOF'
feat(admin): faction list/show/set/bump/reset

Admin command for inspecting and adjusting per-faction rep.
Mirrors the chunk 1.1 opinion command structure with `list` added
(definitions are a closed authored set, worth listing whole).
Helpfile and registry entry both ship.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 16: End-to-end smoke test

This is a manual verification — no code changes. Follow the spec's "Live smoke test" section.

- [ ] **Step 1: Build the binary**

Run: `go build -o /tmp/dogmud-test.exe .`
Expected: clean build.

- [ ] **Step 2: Set up smoketester for the warren quest**

Edit `_datafiles/world/dogmud/users/17.yaml`:

1. Locate the quest-progress field on the character (likely under `character.quests` or `character.questprogress`). Remove any tokens for quest 2 (`2-start`, `2-end`).
2. Set `character.roomid: 300` (Narrow Descent — one room above Low Junction).
3. Add item id 30036 (healing salve) to the character's inventory list.
4. Grant the chrysalis-glow spell to the character (locate the character's known-spells field).

Delete `_datafiles/world/dogmud/factions.rep/warren.yaml` if it exists (so the file lazy-recreates with no smoketester rep entry).

- [ ] **Step 3: Boot the server**

Run: `/tmp/dogmud-test.exe > /tmp/boot.log 2>&1 &`
Wait for "Server Ready" or panic.

- [ ] **Step 4: Run the smoketester quest path**

In a separate terminal, telnet to the server (default port from config). Log in as smoketester / smoketester (or whatever the password is — read from `users/17.yaml` if needed).

Execute:
```
cast chrysalis-glow
down
look shaman
give healing salve to shaman
```

Verify:
- The chieftain's reward message appears (per quest 2's `playermessage`).
- Quest 2 completes (look at quest log if there's a command for it, or just trust the dialogue).

- [ ] **Step 5: Verify rep with admin command**

Log in as an admin (e.g., Megalomania at userId 1) in a third terminal. Execute:

```
faction show smoketester
```

Expected output: shows `warren` at +30 (Warm tier).

- [ ] **Step 6: Verify the peace gate**

As the smoketester:
- Walk into a room with a warren scout (or wait for one to wander in).
- Confirm the scout does NOT aggro (the new IsPeacefulToward gate fires).

- [ ] **Step 7: Verify the kill-rep flow**

As the smoketester:
- `attack scout` (or `attack warrior`)
- Kill it.

As admin again: `faction show smoketester` — confirm warren rep dropped by `FactionMemberKillRep` (-10), now at +20.

- [ ] **Step 8: Saturate to Hostile**

Repeat killing warren mobs until rep drops below TierWarm (-15..-49 territory). Confirm warren mobs now aggro on sight.

- [ ] **Step 9: Cleanup**

1. Kill the running server process per the SOP:
   `tasklist | grep dogmud` to find the PID, then kill it.
2. Reset `_datafiles/world/dogmud/users/17.yaml` to the original state if the smoketester will be reused for other tests.
3. Delete `/tmp/dogmud-test.exe` and `/tmp/boot.log`.

- [ ] **Step 10: Mark roadmap status as Done**

Open `MOB_ALIVENESS_ROADMAP.md`. Flip 1.2's Status from `In progress` to `Done (2026-MM-DD)` (use today's date). Re-tally the rollup line. Add a `Shipped:` bullet to the 1.2 mini-brief listing the spec/plan paths and key artifacts (mirror chunk 1.1's pattern).

Commit:
```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "$(cat <<'EOF'
docs(roadmap): mark chunk 1.2 (faction system) as Done

Roll-up updated. Mini-brief gains a Shipped: line linking the
spec, plan, and key artifacts.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-review

**Spec coverage check:**

| Spec section | Task(s) |
|---|---|
| Identity: factions = Groups + def file | T1 (types), T5 (FactionsForMob) |
| Storage split (definitions vs runtime rep) | T2 (definition load), T3 (rep persistence) |
| Score scale, no decay, default rep | T1 (constants), T4 (Get returns default), T2 (Definition.DefaultRep) |
| Faction-vs-faction relations declarative | T1 (Definition fields), T2 (validation only — no propagation) |
| Integration scope: API + ONE consumer | T4-T5 (API), T14 (combat hookup as the consumer) |
| Quest engine bump_rep | T8 |
| peacefulquest migration | T11 (quest YAML + migration), T12 (gate replacement), T13 (field deletion) |
| Combat hookup with party prop | T14 |
| Admin command faction | T15 |
| Two faction defs (warren, thornwall_guards) | T9 |
| Out-of-scope guards | All tasks; no extra features |
| Roadmap chunk 6.5a | T9 |
| Tests | T1-T15 each contain unit tests; T16 is the live smoke |
| Non-functional reqs | Implicit in task design; saveMu in T3 |

No gaps.

**Placeholder scan:** No "TBD"/"TODO"/"fill in later" present. The "verify yaml field name" notes in T11 step 2 are flagged inline as discovery work, not placeholders — the task tells you exactly how to discover the right field name (read a user yaml).

**Type consistency:**
- `factionId` (string) used consistently across all tasks
- `userId` (int) consistent
- `Definition.FactionId`, `RepEntry.Rep`, `FactionRep.Players` consistent across T1, T2, T3, T4
- `factions.BumpRep`, `factions.SetRep`, `factions.GetRep`, `factions.TierFor` signatures match across T4, T8, T11, T14, T15
- `events.MobDeath` field names (`InstanceId`, `RoomId`, `PlayerDamage`) verified against actual struct in T14

No drift.
