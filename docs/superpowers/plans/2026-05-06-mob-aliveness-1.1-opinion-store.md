# Mob Aliveness 1.1 — Persistent NPC Opinion Store Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the chunk-1.1 substrate from
`docs/superpowers/specs/2026-05-06-mob-aliveness-1.1-opinion-store-design.md`:
a per-`(mobId, userId)` signed-scalar disposition store with
per-NPC YAML persistence, lazy decay toward a per-NPC default, an
admin debug command, and a single combat hookup (player-initiated
attack bumps opinion downward).

**Architecture:** New `internal/opinions/` package mirroring the
`internal/shops/` persistence pattern (in-memory cache + per-NPC
YAML file, synchronous save on mutation). Lazy decay is computed
on read from the anchored `(score, last_updated_round)` pair —
reads do not mutate or persist. The combat hookup lives in the
`attack` user command, firing one `Bump` per fresh-aggro
acquisition. Admin command `opinion` exposes show/set/bump/reset.

**Tech Stack:** Go, `gopkg.in/yaml.v2`, existing GoMud patterns
(`internal/shops/persistence.go` is the closest precedent).

**Spec deviations** (discovered during plan-writing, all simplifications):

1. **Synchronous save on mutation** instead of dirty-flag flush. The
   spec said "mirror shops' periodic flusher," but shops actually
   save synchronously on every mutation (no flusher). Following the
   real pattern is simpler and bounded — first-aggression bumps
   fire at most once per fight, so write volume is tiny.
2. **Reads are pure.** Lazy decay is recomputed on every read from
   the anchored values — `Get` doesn't mutate or save anything.
   `Bump`/`Set` decay-then-mutate-then-stamp atomically, persisting
   the new anchor. This avoids any "Get triggered a write" spookiness
   and makes reads cheap.
3. **No `SaveAllOpinions()` shutdown call.** Shops don't have one
   either, and synchronous-save-on-mutation makes it unnecessary.
   The function is still defined for future use / parity but is
   not wired into the shutdown path in 1.1.

---

## File Structure

**New package: `internal/opinions/`**

| File | Responsibility |
|------|-----------------|
| `types.go` | `Opinion`, `MobOpinions`, `Tier` enum, file schema |
| `decay.go` | Pure decay math (`pull`, `decayedScore`) |
| `opinions.go` | Public API: `Get`, `Set`, `Bump`, `TierOf`, `TierFor` |
| `persistence.go` | Cache, path, `loadFromDisk`, `saveToDisk`, `SaveAllOpinions`, `ClearCache` |
| `opinions_test.go` | Unit tests for API + decay + clamp |
| `persistence_test.go` | Save/load round-trip, corrupt YAML, missing-file fallback |

**Modified files:**

| File | Change |
|------|--------|
| `internal/configs/config.balance.go` | Add `OpinionAttackBump`, `DispositionDecayHalfLifeRounds` |
| `internal/configs/config.balance.go` (defaults) | Set `-15` and `100000` defaults |
| `_datafiles/config.yaml` | Surface the new knobs (optional — defaults cover it) |
| `internal/mobs/mobs.go` | Add `DefaultDisposition int yaml:"default_disposition,omitempty"` field |
| `internal/usercommands/attack.go` | Call `opinions.Bump` after `SetAggro` (PvM branch) |
| `internal/usercommands/attack_test.go` | Integration test for the bump |
| `internal/usercommands/admin.opinion.go` | New admin command file |
| `internal/usercommands/admin.opinion_test.go` | Admin command tests |
| Admin command registry | Register `opinion` |
| Admin help template | Helpfile for `opinion` |
| Admin help index | List `opinion` in admin commands |

---

## Task 1: Package types + Tier enum

**Files:**
- Create: `internal/opinions/types.go`
- Create: `internal/opinions/opinions.go`
- Test: `internal/opinions/opinions_test.go`

- [ ] **Step 1: Write the failing test for `TierOf`**

Create `internal/opinions/opinions_test.go`:

```go
package opinions

import "testing"

func TestTierOf(t *testing.T) {
	cases := []struct {
		score int
		want  Tier
	}{
		{-100, TierHostile},
		{-50, TierHostile},
		{-49, TierCold},
		{-15, TierCold},
		{-14, TierNeutral},
		{0, TierNeutral},
		{14, TierNeutral},
		{15, TierWarm},
		{49, TierWarm},
		{50, TierFriendly},
		{100, TierFriendly},
	}
	for _, c := range cases {
		if got := TierOf(c.score); got != c.want {
			t.Errorf("TierOf(%d) = %v, want %v", c.score, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails (compile error)**

Run: `go test ./internal/opinions/...`
Expected: build failure — `opinions` package does not yet exist.

- [ ] **Step 3: Create `types.go` with the schema**

```go
package opinions

// Opinion is one NPC's stored disposition toward one player.
// Persisted in YAML, keyed by userId inside the parent MobOpinions.
type Opinion struct {
	Score            int    `yaml:"score"`
	LastUpdatedRound uint64 `yaml:"last_updated_round"`
}

// MobOpinions is one mob template's full opinion table — every
// player this NPC has ever held a non-default opinion of, plus the
// snapshotted default disposition the file owns as source of truth.
//
// One MobOpinions persists per mob template at:
//
//	_datafiles/world/dogmud/opinions/{mobId}-{namesimple}.yaml
//
// All instances of a mob template share this table.
type MobOpinions struct {
	MobId              int               `yaml:"mob_id"`
	DefaultDisposition int               `yaml:"default_disposition"`
	Opinions           map[int]*Opinion  `yaml:"opinions"` // userId → opinion
}

// Tier is a banded view of the disposition score, for consumers
// (dialogue gates, combat aggro logic) that prefer thresholds over
// raw numbers.
type Tier int

const (
	TierHostile  Tier = iota // <= -50
	TierCold                 // -49 .. -15
	TierNeutral              // -14 .. +14
	TierWarm                 // +15 .. +49
	TierFriendly             // >= +50
)

// Score range — every Set/Bump clamps to this window.
const (
	ScoreMin = -100
	ScoreMax = +100
)
```

- [ ] **Step 4: Create `opinions.go` with `TierOf`**

```go
package opinions

// TierOf bucket-maps a raw disposition score to its Tier.
func TierOf(score int) Tier {
	switch {
	case score <= -50:
		return TierHostile
	case score <= -15:
		return TierCold
	case score <= 14:
		return TierNeutral
	case score <= 49:
		return TierWarm
	default:
		return TierFriendly
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/opinions/... -run TestTierOf -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/opinions/types.go internal/opinions/opinions.go internal/opinions/opinions_test.go
git commit -m "$(cat <<'EOF'
feat(opinions): package skeleton with Tier banding

Types, Tier enum, and TierOf for chunk 1.1 of the mob aliveness
roadmap. No persistence or mutation yet — purely the data shape and
the banding helper.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Decay math

**Files:**
- Create: `internal/opinions/decay.go`
- Test: `internal/opinions/opinions_test.go` (append cases)

- [ ] **Step 1: Append failing tests for decay**

Append to `internal/opinions/opinions_test.go`:

```go
func TestPullTowardDefault(t *testing.T) {
	cases := []struct {
		name     string
		score    int
		def      int
		steps    int
		want     int
	}{
		{"no steps", -50, 0, 0, -50},
		{"one step toward 0 from negative", -50, 0, 1, -49},
		{"five steps toward 0 from negative", -50, 0, 5, -45},
		{"steps cannot overshoot zero default", -3, 0, 10, 0},
		{"toward positive default", -10, 5, 3, -7},
		{"steps cannot overshoot positive default", -3, 5, 100, 5},
		{"already at default", 5, 5, 100, 5},
		{"positive score decays toward 0", 50, 0, 5, 45},
		{"positive score decays toward negative default", 50, -10, 100, -10},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := pull(c.score, c.def, c.steps); got != c.want {
				t.Errorf("pull(%d, %d, %d) = %d, want %d", c.score, c.def, c.steps, got, c.want)
			}
		})
	}
}

func TestDecayedScore(t *testing.T) {
	const halfLife uint64 = 100
	cases := []struct {
		name     string
		score    int
		def      int
		anchor   uint64
		now      uint64
		halfLife uint64
		want     int
	}{
		{"no time elapsed", -50, 0, 1000, 1000, halfLife, -50},
		{"half a half-life elapsed (no integer step)", -50, 0, 1000, 1049, halfLife, -50},
		{"one half-life elapsed", -50, 0, 1000, 1100, halfLife, -49},
		{"ten half-lives elapsed", -50, 0, 1000, 2000, halfLife, -40},
		{"halfLife=0 means no decay", -50, 0, 1000, 99999, 0, -50},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := decayedScore(c.score, c.def, c.anchor, c.now, c.halfLife); got != c.want {
				t.Errorf("decayedScore(%d, %d, anchor=%d, now=%d, hl=%d) = %d, want %d",
					c.score, c.def, c.anchor, c.now, c.halfLife, got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail (undefined funcs)**

Run: `go test ./internal/opinions/... -run "TestPull|TestDecayedScore" -v`
Expected: FAIL — `pull` / `decayedScore` undefined.

- [ ] **Step 3: Create `decay.go`**

```go
package opinions

// pull moves score toward def by exactly N integer steps without
// overshooting. n=0 is identity. n<0 is treated as 0.
func pull(score, def, n int) int {
	if n <= 0 || score == def {
		return score
	}
	if score < def {
		next := score + n
		if next > def {
			return def
		}
		return next
	}
	// score > def
	next := score - n
	if next < def {
		return def
	}
	return next
}

// decayedScore returns the score after applying integer-step decay
// from anchor to now, using the given half-life in rounds. A
// half-life of 0 disables decay (returns score unchanged).
func decayedScore(score, def int, anchorRound, nowRound, halfLifeRounds uint64) int {
	if halfLifeRounds == 0 || nowRound <= anchorRound {
		return score
	}
	steps := int((nowRound - anchorRound) / halfLifeRounds)
	return pull(score, def, steps)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/opinions/... -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/opinions/decay.go internal/opinions/opinions_test.go
git commit -m "$(cat <<'EOF'
feat(opinions): pure decay math

`pull(score, def, n)` moves score toward def by n integer steps
with no overshoot. `decayedScore` builds the per-round step count
from the anchored `(score, last_updated_round)` pair. Both are
pure functions so reads stay cheap and persistence stays anchored.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Score clamp helper

**Files:**
- Modify: `internal/opinions/opinions.go`
- Test: `internal/opinions/opinions_test.go`

- [ ] **Step 1: Append failing test for `clampScore`**

Append to `opinions_test.go`:

```go
func TestClampScore(t *testing.T) {
	cases := []struct{ in, want int }{
		{-200, -100},
		{-100, -100},
		{-50, -50},
		{0, 0},
		{50, 50},
		{100, 100},
		{200, 100},
	}
	for _, c := range cases {
		if got := clampScore(c.in); got != c.want {
			t.Errorf("clampScore(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test ./internal/opinions/... -run TestClampScore -v`
Expected: FAIL — `clampScore` undefined.

- [ ] **Step 3: Add `clampScore` to `opinions.go`**

Append to `internal/opinions/opinions.go`:

```go
// clampScore restricts a score to the [ScoreMin, ScoreMax] window.
func clampScore(s int) int {
	if s < ScoreMin {
		return ScoreMin
	}
	if s > ScoreMax {
		return ScoreMax
	}
	return s
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/opinions/... -run TestClampScore -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/opinions/opinions.go internal/opinions/opinions_test.go
git commit -m "$(cat <<'EOF'
feat(opinions): score clamp helper

clampScore restricts to [-100, +100] — every Set/Bump that lands a
new score routes through it, so no caller can produce an
out-of-range value.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Persistence — path, cache, load/save scaffolding

**Files:**
- Create: `internal/opinions/persistence.go`
- Test: `internal/opinions/persistence_test.go`

- [ ] **Step 1: Write the failing path test**

Create `internal/opinions/persistence_test.go`:

```go
package opinions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpinionPath(t *testing.T) {
	got := opinionPath(41, "lars")
	if !strings.HasSuffix(filepath.ToSlash(got), "opinions/41-lars.yaml") {
		t.Errorf("opinionPath = %q, want suffix opinions/41-lars.yaml", got)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir) // honored by opinionPath in tests

	ClearCache()

	mo := &MobOpinions{
		MobId:              41,
		DefaultDisposition: 5,
		Opinions: map[int]*Opinion{
			17: {Score: -42, LastUpdatedRound: 1843201},
			92: {Score: 28, LastUpdatedRound: 1846020},
		},
	}
	cacheStoreForTest("lars", mo)
	if err := saveToDisk(41, "lars"); err != nil {
		t.Fatalf("saveToDisk: %v", err)
	}

	ClearCache()
	got := loadFromDisk(41, "lars")
	if got == nil {
		t.Fatal("loadFromDisk returned nil for round-tripped file")
	}
	if got.MobId != 41 || got.DefaultDisposition != 5 {
		t.Errorf("round-trip header mismatch: %+v", got)
	}
	if got.Opinions[17].Score != -42 || got.Opinions[17].LastUpdatedRound != 1843201 {
		t.Errorf("round-trip user 17 mismatch: %+v", got.Opinions[17])
	}
	if got.Opinions[92].Score != 28 {
		t.Errorf("round-trip user 92 mismatch: %+v", got.Opinions[92])
	}

	// Sanity: file exists where we expected.
	expected := filepath.Join(dir, "41-lars.yaml")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected file missing: %v", err)
	}
}

func TestLoadFromDiskMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	ClearCache()

	if got := loadFromDisk(999, "ghost"); got != nil {
		t.Errorf("loadFromDisk on missing file = %+v, want nil", got)
	}
}

func TestLoadFromDiskCorruptYAMLReturnsNil(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	ClearCache()

	bad := filepath.Join(dir, "1-bad.yaml")
	if err := os.WriteFile(bad, []byte("this is: not: valid: yaml: at all"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := loadFromDisk(1, "bad"); got != nil {
		t.Errorf("loadFromDisk on corrupt YAML = %+v, want nil", got)
	}
}
```

The test references `cacheStoreForTest` (a test-only seam to put a
`*MobOpinions` into the cache without requiring `Bump` to function
yet). Define it in `persistence.go` along with everything else.

- [ ] **Step 2: Run tests to verify they fail (build errors)**

Run: `go test ./internal/opinions/... -run "TestOpinionPath|TestSaveAndLoadRoundTrip|TestLoadFromDiskMissing|TestLoadFromDiskCorrupt" -v`
Expected: FAIL — `opinionPath`, `saveToDisk`, `loadFromDisk`,
`ClearCache`, `cacheStoreForTest` undefined.

- [ ] **Step 3: Implement `persistence.go`**

```go
package opinions

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

// opinionCache is the in-memory store of all loaded MobOpinions
// keyed by mobId.
var (
	opinionCacheMu sync.RWMutex
	opinionCache   = map[int]*MobOpinions{}
	// nameByMobId remembers the namesimple used at file write time,
	// so we can reconstruct the path when persisting again without
	// re-reading the mob template.
	nameByMobId = map[int]string{}
)

// opinionsBaseDir returns the directory that holds opinion files.
// Honors the DOGMUD_OPINIONS_DIR_OVERRIDE env var so tests can
// redirect to a temp dir without a real DataFiles config.
func opinionsBaseDir() string {
	if override := os.Getenv("DOGMUD_OPINIONS_DIR_OVERRIDE"); override != "" {
		return override
	}
	return util.FilePath(
		configs.GetFilePathsConfig().DataFiles.String(), `/`, `world`, `/`, `dogmud`, `/`, `opinions`,
	)
}

// opinionPath returns the absolute path to a mob's opinion file.
func opinionPath(mobId int, namesimple string) string {
	filename := fmt.Sprintf("%d-%s.yaml", mobId, namesimple)
	return filepath.Join(opinionsBaseDir(), filename)
}

// loadFromDisk reads the YAML file for mobId and returns the parsed
// MobOpinions. Returns nil if the file is missing or malformed.
func loadFromDisk(mobId int, namesimple string) *MobOpinions {
	path := opinionPath(mobId, namesimple)
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil // missing file — fresh slate
	}
	var mo MobOpinions
	if err := yaml.Unmarshal(bytes, &mo); err != nil {
		mudlog.Error("opinions.loadFromDisk", "path", path, "error", err)
		return nil
	}
	if mo.Opinions == nil {
		mo.Opinions = map[int]*Opinion{}
	}
	return &mo
}

// saveToDisk persists the cached MobOpinions for mobId to the
// configured opinions directory. Returns an error if the cache is
// missing the entry or the write fails.
func saveToDisk(mobId int, namesimple string) error {
	opinionCacheMu.RLock()
	mo, ok := opinionCache[mobId]
	opinionCacheMu.RUnlock()
	if !ok {
		return fmt.Errorf("opinions.saveToDisk: no cached entry for mobId=%d", mobId)
	}

	path := opinionPath(mobId, namesimple)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("opinions.saveToDisk: mkdir %s: %w", filepath.Dir(path), err)
	}
	bytes, err := yaml.Marshal(mo)
	if err != nil {
		return fmt.Errorf("opinions.saveToDisk: marshal: %w", err)
	}
	if err := os.WriteFile(path, bytes, 0644); err != nil {
		return fmt.Errorf("opinions.saveToDisk: write %s: %w", path, err)
	}
	return nil
}

// SaveAllOpinions writes every cached MobOpinions to disk. Defined
// for parity with shops.SaveAllShops; not currently wired into
// shutdown (synchronous-on-mutation save covers it). Useful for
// admin commands and future graceful-shutdown work.
func SaveAllOpinions() {
	opinionCacheMu.RLock()
	type entry struct {
		mobId int
		name  string
	}
	entries := make([]entry, 0, len(opinionCache))
	for mobId := range opinionCache {
		name := nameByMobId[mobId]
		entries = append(entries, entry{mobId, name})
	}
	opinionCacheMu.RUnlock()
	for _, e := range entries {
		if err := saveToDisk(e.mobId, e.name); err != nil {
			mudlog.Error("opinions.SaveAllOpinions", "error", err)
		}
	}
}

// ClearCache drops every cached MobOpinions. Tests use this to
// isolate cases; production code should not call it.
func ClearCache() {
	opinionCacheMu.Lock()
	opinionCache = map[int]*MobOpinions{}
	nameByMobId = map[int]string{}
	opinionCacheMu.Unlock()
}

// cacheStoreForTest seeds the cache directly. Test-only seam used
// by persistence_test.go before Get/Bump/Set exist.
func cacheStoreForTest(namesimple string, mo *MobOpinions) {
	opinionCacheMu.Lock()
	opinionCache[mo.MobId] = mo
	nameByMobId[mo.MobId] = namesimple
	opinionCacheMu.Unlock()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/opinions/... -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/opinions/persistence.go internal/opinions/persistence_test.go
git commit -m "$(cat <<'EOF'
feat(opinions): YAML persistence and cache scaffolding

opinionPath, loadFromDisk, saveToDisk, ClearCache, SaveAllOpinions
+ a DOGMUD_OPINIONS_DIR_OVERRIDE env hook so tests don't need a
real DataFiles config. Round-trip and corrupt-yaml tests included.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: `Get`, `Set`, `Bump` API

**Files:**
- Modify: `internal/opinions/opinions.go`
- Modify: `internal/opinions/opinions_test.go`

- [ ] **Step 1: Append failing tests for the API**

Append to `opinions_test.go`:

```go
func TestGetReturnsDefaultWhenNoFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	ClearCache()
	defaultProviderForTest = func(mobId int) (string, int, bool) {
		if mobId == 41 {
			return "lars", 7, true
		}
		return "", 0, false
	}
	t.Cleanup(func() { defaultProviderForTest = nil })

	if got := Get(41, 17); got != 7 {
		t.Errorf("Get with no file = %d, want 7 (template default)", got)
	}
}

func TestSetWritesAndPersists(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	ClearCache()
	defaultProviderForTest = func(mobId int) (string, int, bool) { return "lars", 0, true }
	t.Cleanup(func() { defaultProviderForTest = nil })
	roundForTest = func() uint64 { return 1000 }
	t.Cleanup(func() { roundForTest = nil })

	Set(41, 17, -75)
	if got := Get(41, 17); got != -75 {
		t.Errorf("Get after Set = %d, want -75", got)
	}

	// Drop cache, reload from disk.
	ClearCache()
	if got := Get(41, 17); got != -75 {
		t.Errorf("Get after restart-equivalent = %d, want -75 (persistence)", got)
	}
}

func TestSetClampsToRange(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	ClearCache()
	defaultProviderForTest = func(mobId int) (string, int, bool) { return "lars", 0, true }
	t.Cleanup(func() { defaultProviderForTest = nil })
	roundForTest = func() uint64 { return 1000 }
	t.Cleanup(func() { roundForTest = nil })

	Set(41, 17, -500)
	if got := Get(41, 17); got != -100 {
		t.Errorf("Set(-500) → Get = %d, want -100", got)
	}
	Set(41, 17, 500)
	if got := Get(41, 17); got != 100 {
		t.Errorf("Set(500) → Get = %d, want 100", got)
	}
}

func TestBumpStartsFromDefaultWhenNoRow(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	ClearCache()
	defaultProviderForTest = func(mobId int) (string, int, bool) { return "lars", 5, true }
	t.Cleanup(func() { defaultProviderForTest = nil })
	roundForTest = func() uint64 { return 1000 }
	t.Cleanup(func() { roundForTest = nil })

	Bump(41, 17, -20)
	// Starting from default (5), bumped by -20, expected -15.
	if got := Get(41, 17); got != -15 {
		t.Errorf("Get after first Bump = %d, want -15", got)
	}
}

func TestBumpDecaysBeforeApplying(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	ClearCache()
	defaultProviderForTest = func(mobId int) (string, int, bool) { return "lars", 0, true }
	t.Cleanup(func() { defaultProviderForTest = nil })

	roundForTest = func() uint64 { return 1000 }
	Set(41, 17, -50)

	// Override decay half-life via test-only seam, then advance round.
	halfLifeForTest = func() uint64 { return 100 }
	t.Cleanup(func() { halfLifeForTest = nil })
	roundForTest = func() uint64 { return 1500 } // five half-lives
	t.Cleanup(func() { roundForTest = nil })

	Bump(41, 17, -10)
	// -50 decayed by 5 toward 0 = -45, then bumped by -10 = -55.
	if got := Get(41, 17); got != -55 {
		t.Errorf("Bump after decay = %d, want -55", got)
	}
}

func TestGetWithoutRowDoesNotCreateRow(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	ClearCache()
	defaultProviderForTest = func(mobId int) (string, int, bool) { return "lars", 5, true }
	t.Cleanup(func() { defaultProviderForTest = nil })

	_ = Get(41, 17)

	// File should not exist — Get is read-only when there's no data.
	path := opinionPath(41, "lars")
	if _, err := os.Stat(path); err == nil {
		t.Errorf("Get created file %q — should not have", path)
	}
}

func TestTierFor(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	ClearCache()
	defaultProviderForTest = func(mobId int) (string, int, bool) { return "lars", 0, true }
	t.Cleanup(func() { defaultProviderForTest = nil })
	roundForTest = func() uint64 { return 1000 }
	t.Cleanup(func() { roundForTest = nil })

	Set(41, 17, -60)
	if got := TierFor(41, 17); got != TierHostile {
		t.Errorf("TierFor after Set(-60) = %v, want Hostile", got)
	}
}
```

Add `import "os"` at the top of the test file if not already present.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/opinions/... -v`
Expected: FAIL — `Get`, `Set`, `Bump`, `TierFor`,
`defaultProviderForTest`, `roundForTest`, `halfLifeForTest`
undefined.

- [ ] **Step 3: Add the API to `opinions.go`**

Append to `internal/opinions/opinions.go`:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Test-only seams. Production code never sets these — they let the
// unit tests inject a mob template lookup, force the round count,
// and override the half-life without standing up the full mob/config
// stack. Each is consulted via a tiny helper that prefers the seam
// when non-nil.
var (
	defaultProviderForTest func(mobId int) (namesimple string, def int, ok bool)
	roundForTest           func() uint64
	halfLifeForTest        func() uint64
)

func currentRound() uint64 {
	if roundForTest != nil {
		return roundForTest()
	}
	return util.GetRoundCount()
}

func currentHalfLife() uint64 {
	if halfLifeForTest != nil {
		return halfLifeForTest()
	}
	return uint64(configs.GetBalanceConfig().DispositionDecayHalfLifeRounds.Int())
}

// resolveTemplate returns (namesimple, default_disposition, ok) for
// a mob template. In production it consults internal/mobs; tests
// override via defaultProviderForTest.
func resolveTemplate(mobId int) (string, int, bool) {
	if defaultProviderForTest != nil {
		return defaultProviderForTest(mobId)
	}
	spec := mobs.GetMobSpec(mobs.MobId(mobId))
	if spec == nil {
		return "", 0, false
	}
	return util.ConvertForFilename(spec.Character.Name), spec.DefaultDisposition, true
}

// loadOrLazyInit returns the cached *MobOpinions for mobId,
// loading from disk on first access. If neither cache nor disk
// has data, an empty MobOpinions seeded from the mob template's
// default disposition is created and cached, but no row is added
// to the inner Opinions map. Returns (nil, "", false) if the mob
// template is unknown.
func loadOrLazyInit(mobId int) (*MobOpinions, string, bool) {
	opinionCacheMu.RLock()
	if mo, ok := opinionCache[mobId]; ok {
		name := nameByMobId[mobId]
		opinionCacheMu.RUnlock()
		return mo, name, true
	}
	opinionCacheMu.RUnlock()

	name, def, ok := resolveTemplate(mobId)
	if !ok {
		return nil, "", false
	}

	if mo := loadFromDisk(mobId, name); mo != nil {
		opinionCacheMu.Lock()
		opinionCache[mobId] = mo
		nameByMobId[mobId] = name
		opinionCacheMu.Unlock()
		return mo, name, true
	}

	mo := &MobOpinions{
		MobId:              mobId,
		DefaultDisposition: def,
		Opinions:           map[int]*Opinion{},
	}
	opinionCacheMu.Lock()
	opinionCache[mobId] = mo
	nameByMobId[mobId] = name
	opinionCacheMu.Unlock()
	return mo, name, true
}

// Get returns the (decay-adjusted) score this NPC has of the given
// user. Returns the NPC's default disposition if no row exists.
// Pure read — does not mutate cache, does not write to disk.
func Get(mobId int, userId int) int {
	mo, _, ok := loadOrLazyInit(mobId)
	if !ok {
		return 0
	}
	op, has := mo.Opinions[userId]
	if !has {
		return mo.DefaultDisposition
	}
	return decayedScore(op.Score, mo.DefaultDisposition,
		op.LastUpdatedRound, currentRound(), currentHalfLife())
}

// Set assigns an absolute score, clamped to [-100, +100], stamps
// last_updated_round, and persists synchronously.
func Set(mobId int, userId int, score int) {
	mo, name, ok := loadOrLazyInit(mobId)
	if !ok {
		return
	}
	clamped := clampScore(score)
	now := currentRound()

	opinionCacheMu.Lock()
	if mo.Opinions == nil {
		mo.Opinions = map[int]*Opinion{}
	}
	mo.Opinions[userId] = &Opinion{Score: clamped, LastUpdatedRound: now}
	opinionCacheMu.Unlock()

	if err := saveToDisk(mobId, name); err != nil {
		mudlog.Warn("opinions.Set: saveToDisk", "mobId", mobId, "userId", userId, "error", err)
	}
}

// Bump adds delta to the current decay-adjusted score, clamps,
// re-stamps the round, and persists synchronously. The everyday
// mutator — combat, dialogue, quest hooks all call this.
func Bump(mobId int, userId int, delta int) {
	mo, name, ok := loadOrLazyInit(mobId)
	if !ok {
		return
	}
	now := currentRound()
	hl := currentHalfLife()

	opinionCacheMu.Lock()
	if mo.Opinions == nil {
		mo.Opinions = map[int]*Opinion{}
	}
	op, has := mo.Opinions[userId]
	var base int
	if has {
		base = decayedScore(op.Score, mo.DefaultDisposition, op.LastUpdatedRound, now, hl)
	} else {
		base = mo.DefaultDisposition
	}
	newScore := clampScore(base + delta)
	mo.Opinions[userId] = &Opinion{Score: newScore, LastUpdatedRound: now}
	opinionCacheMu.Unlock()

	if err := saveToDisk(mobId, name); err != nil {
		mudlog.Warn("opinions.Bump: saveToDisk", "mobId", mobId, "userId", userId, "error", err)
	}
}

// TierFor is sugar for TierOf(Get(mobId, userId)).
func TierFor(mobId int, userId int) Tier {
	return TierOf(Get(mobId, userId))
}
```

Note: this introduces an import on `internal/configs` and
`internal/mobs`. Open `internal/opinions/opinions.go` and place the
imports inside the existing import block. Add a `mudlog` import
too (used by Set/Bump). Tests will catch any missing import.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/opinions/... -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/opinions/opinions.go internal/opinions/opinions_test.go
git commit -m "$(cat <<'EOF'
feat(opinions): public API — Get, Set, Bump, TierFor

Get is a pure read (decay applied on-the-fly, no mutation). Set and
Bump decay-then-apply-then-clamp-then-stamp atomically and persist
synchronously. Test-only seams (defaultProviderForTest,
roundForTest, halfLifeForTest) keep unit tests independent of the
mob/config/util stack.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Concurrency safety test

**Files:**
- Modify: `internal/opinions/opinions_test.go`

- [ ] **Step 1: Append the parallel-bump test**

```go
func TestParallelBumpsConverge(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	ClearCache()
	defaultProviderForTest = func(mobId int) (string, int, bool) { return "lars", 0, true }
	t.Cleanup(func() { defaultProviderForTest = nil })
	roundForTest = func() uint64 { return 1000 }
	t.Cleanup(func() { roundForTest = nil })

	const goroutines = 10
	const bumpsPer = 100
	const delta = 1

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < bumpsPer; j++ {
				Bump(41, 17, delta)
			}
		}()
	}
	wg.Wait()

	// Every Bump adds 1; total expected is min(clamp, total).
	want := goroutines * bumpsPer
	if want > ScoreMax {
		want = ScoreMax
	}
	if got := Get(41, 17); got != want {
		t.Errorf("after %d parallel bumps: got %d, want %d", goroutines*bumpsPer, got, want)
	}
}
```

Add `"sync"` to imports if not already present.

- [ ] **Step 2: Run with race detector**

Run: `go test ./internal/opinions/... -race -run TestParallelBumpsConverge -v`
Expected: PASS, no race warnings.

- [ ] **Step 3: Commit**

```bash
git add internal/opinions/opinions_test.go
git commit -m "$(cat <<'EOF'
test(opinions): parallel-bump convergence under -race

Confirms the package-level mutex covers all read/write seams
between Bump calls — no lost writes, no data races.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Balance config knobs

**Files:**
- Modify: `internal/configs/config.balance.go` (add fields + defaults)
- Modify: `internal/configs/config.balance.go` or its
  `applyDefaults`/`SetDefaults` equivalent

- [ ] **Step 1: Identify the existing defaults pattern**

Read `internal/configs/config.balance.go` and locate where
existing `Balance` fields get their defaults set (e.g., look for a
function named `SetDefaults`, `applyDefaults`, or pattern matching
on struct-tag values). Note the convention used (`ConfigInt`,
`ConfigFloat`, `ConfigBool` wrappers).

Run: `grep -n "ConfigInt\|SetDefault\|applyDefault" internal/configs/config.balance.go | head -30`

Read the surrounding 30 lines of the first hit to understand the
defaults-setting pattern.

- [ ] **Step 2: Add the two fields to the `Balance` struct**

Find the `type Balance struct` block in
`internal/configs/config.balance.go` and append (in a section that
fits — combat/aliveness near the bottom, or a new "OPINIONS"
section header):

```go
	// ── OPINIONS / DISPOSITION ───────────────────────────────────────────────
	OpinionAttackBump              ConfigInt `yaml:"OpinionAttackBump"`              // Disposition delta when a player initiates aggression on a mob (default -15)
	DispositionDecayHalfLifeRounds ConfigInt `yaml:"DispositionDecayHalfLifeRounds"` // Rounds for one half-life of disposition decay toward default (default 100000; 0 disables decay)
```

- [ ] **Step 3: Add the defaults**

Locate the function that sets defaults for Balance (search for
`SetDefaults` or similar in `config.balance.go` — most likely
`func (b *Balance) SetDefaults()` or similar). Add (matching the
existing default-setting style for `ConfigInt`):

```go
	if b.OpinionAttackBump == 0 {
		b.OpinionAttackBump = -15
	}
	if b.DispositionDecayHalfLifeRounds == 0 {
		b.DispositionDecayHalfLifeRounds = 100000
	}
```

If the codebase uses a different default-setting idiom (e.g., a
table of default values in a separate file), match that idiom
instead. The two values are `-15` and `100000`.

- [ ] **Step 4: Verify `go build` passes**

Run: `go build ./...`
Expected: build succeeds.

- [ ] **Step 5: Commit**

```bash
git add internal/configs/config.balance.go
git commit -m "$(cat <<'EOF'
feat(config): OpinionAttackBump and DispositionDecayHalfLifeRounds

Knobs for chunk 1.1 of the mob aliveness substrate. Default
-15 per first-aggression bump and 100000-round half-life
(~4-5 real days at typical round cadence).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: `default_disposition` field on Mob struct

**Files:**
- Modify: `internal/mobs/mobs.go`

- [ ] **Step 1: Add the field**

Open `internal/mobs/mobs.go`, find the `Mob` struct (around line 67,
look for `type Mob struct`). Add this field in the same area as
`Archetype` (around line 108):

```go
	DefaultDisposition int `yaml:"default_disposition,omitempty"` // Per-NPC starting disposition score on the [-100, +100] scale; 0 means neutral. Used by internal/opinions to seed first-time interactions and as the asymptote for decay.
```

- [ ] **Step 2: Confirm build is clean**

Run: `go build ./...`
Expected: builds successfully.

- [ ] **Step 3: Confirm existing tests still pass**

Run: `go test ./internal/mobs/... -count=1`
Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add internal/mobs/mobs.go
git commit -m "$(cat <<'EOF'
feat(mobs): default_disposition YAML field

Per-NPC starting disposition on the [-100, +100] scale, optional in
mob YAML (defaults to 0 / neutral). Read by internal/opinions to
seed first-time interactions and as the decay asymptote.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Combat hookup — first-aggression bump in `attack.go`

**Files:**
- Modify: `internal/usercommands/attack.go`
- Modify: `internal/usercommands/attack_test.go`

- [ ] **Step 1: Read the relevant section of attack.go**

Read `internal/usercommands/attack.go` around line 199 to confirm
the structure. The PvM aggression call is:

```go
user.Character.SetAggro(0, attackMobInstanceId, characters.DefaultAttack)
```

The mob instance was resolved earlier via `mobs.GetInstance` — find
the local variable holding the `*mobs.Mob` (commonly `mob`).

- [ ] **Step 2: Add the failing integration test**

Open `internal/usercommands/attack_test.go`. Find an existing
PvM-style test as a structural reference (it'll set up a user, a
mob instance, and call `Attack` directly). If no PvM `Attack` test
exists yet, create one. The new test asserts that a fresh attack
on a mob bumps the mob's opinion of the user.

```go
func TestAttackBumpsOpinion(t *testing.T) {
	// Test scaffolding: real opinion store backed by a temp dir,
	// real mob instance, real user. The exact setup pattern follows
	// the existing PvM tests in this file (target resolution, room
	// init, etc.) — adapt accordingly.

	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	opinions.ClearCache()

	// Build a mob template/instance with default_disposition=0.
	// Build a user with userId=42 in the same room.
	// (Use the same helpers other PvM attack tests in this file use;
	// the exact helper names depend on the test infrastructure
	// already present.)

	user := newTestUser(t, 42)            // helper from existing tests
	mob := newTestMobInstance(t, 41, 0)   // mobId=41, default_disposition=0
	placeUserAndMobInRoom(t, user, mob)   // helper from existing tests

	// Sanity: opinion starts at default (0).
	if got := opinions.Get(41, user.UserId); got != 0 {
		t.Fatalf("pre-attack opinion = %d, want 0", got)
	}

	rest := fmt.Sprintf("#%d", mob.InstanceId)
	if _, err := Attack(rest, user, /* room */ getRoomFor(user), 0); err != nil {
		t.Fatalf("Attack: %v", err)
	}

	want := configs.GetBalanceConfig().OpinionAttackBump.Int()
	if got := opinions.Get(41, user.UserId); got != want {
		t.Errorf("post-attack opinion = %d, want %d", got, want)
	}
}

func TestAttackOnSameTargetDoesNotDoubleBump(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	opinions.ClearCache()

	user := newTestUser(t, 42)
	mob := newTestMobInstance(t, 41, 0)
	placeUserAndMobInRoom(t, user, mob)

	rest := fmt.Sprintf("#%d", mob.InstanceId)
	if _, err := Attack(rest, user, getRoomFor(user), 0); err != nil {
		t.Fatalf("first Attack: %v", err)
	}
	first := opinions.Get(41, user.UserId)

	// Second invocation while still aggroed on same target — should be a no-op.
	if _, err := Attack(rest, user, getRoomFor(user), 0); err != nil {
		t.Fatalf("second Attack: %v", err)
	}
	second := opinions.Get(41, user.UserId)
	if second != first {
		t.Errorf("re-attack opinion changed: was %d, now %d (want stable)", first, second)
	}
}
```

If the helpers (`newTestUser`, `newTestMobInstance`, `placeUserAndMobInRoom`,
`getRoomFor`) don't exist, examine an existing PvM test in
`attack_test.go` and reuse its setup verbatim. The conceptual asserts
above are what matter; the scaffolding adapts to what's already
present.

Add `"github.com/GoMudEngine/GoMud/internal/opinions"` and
`"github.com/GoMudEngine/GoMud/internal/configs"` to the test file's
imports.

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/usercommands/... -run TestAttackBumpsOpinion -v`
Expected: FAIL — opinion is still 0 after attack (no bump wired in
yet).

- [ ] **Step 4: Wire the bump in attack.go**

Open `internal/usercommands/attack.go`. Find the line:

```go
user.Character.SetAggro(0, attackMobInstanceId, characters.DefaultAttack)
```

Modify the surrounding block to:

```go
// Detect "fresh aggression" before SetAggro overwrites prior state:
// either no prior aggro, or aggro on a different target.
isFreshAggro := user.Character.Aggro == nil ||
	user.Character.Aggro.MobInstanceId != attackMobInstanceId

user.Character.SetAggro(0, attackMobInstanceId, characters.DefaultAttack)

if isFreshAggro {
	if mob := mobs.GetInstance(attackMobInstanceId); mob != nil {
		opinions.Bump(int(mob.MobId), user.UserId,
			configs.GetBalanceConfig().OpinionAttackBump.Int())
	}
}
```

Add `"github.com/GoMudEngine/GoMud/internal/opinions"` to the file's
imports.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/usercommands/... -run "TestAttackBumpsOpinion|TestAttackOnSameTargetDoesNotDoubleBump" -v`
Expected: both PASS.

- [ ] **Step 6: Run the full usercommands test suite to catch regressions**

Run: `go test ./internal/usercommands/... -count=1`
Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add internal/usercommands/attack.go internal/usercommands/attack_test.go
git commit -m "$(cat <<'EOF'
feat(combat): attack initiation bumps mob opinion of attacker

When a player invokes attack and the aggro target is fresh (no
prior aggro, or aggro on a different target), the mob's
template-keyed opinion of that user is bumped by
Balance.OpinionAttackBump (default -15). Re-issuing attack on the
same target while already aggroed is a no-op.

The first concrete consumer of internal/opinions, validating the
substrate end-to-end. Other initiation paths (bash, kick, taunt,
trip, grapple) remain to be wired in later chunks once the
parity-audit chunk surfaces them.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Admin command — `opinion show / set / bump / reset`

**Files:**
- Create: `internal/usercommands/admin.opinion.go`
- Create: `internal/usercommands/admin.opinion_test.go`
- Modify: admin command registry (location verified during task)

- [ ] **Step 1: Find the admin command registry pattern**

Run: `grep -rn "admin\." internal/usercommands/ | grep -v _test | grep "func .*= " | head -10`

This surfaces how existing admin commands register. Read one
neighboring admin command (e.g., `admin.server.go`) end-to-end to
mirror its structure: function signature, argument parsing,
registration call, output formatting.

- [ ] **Step 2: Write the failing test**

Create `internal/usercommands/admin.opinion_test.go`:

```go
package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/opinions"
)

func TestAdminOpinionSetAndShow(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	opinions.ClearCache()

	user := newTestAdminUser(t, 1)             // existing admin-user helper
	target := newTestUser(t, 42)               // existing user helper
	mob := newTestMobTemplate(t, 41, "lars")   // existing mob-template helper

	// Set opinion to -60 via admin command.
	if _, err := Opinion("set lars Aliceia -60", user, getRoomFor(user), 0); err != nil {
		t.Fatalf("opinion set: %v", err)
	}
	if got := opinions.Get(41, target.UserId); got != -60 {
		t.Errorf("after set, Get = %d, want -60", got)
	}

	// Show — output should mention the score and tier.
	out := captureSendText(t, user, func() {
		_, _ = Opinion("show Aliceia", user, getRoomFor(user), 0)
	})
	if !strings.Contains(out, "-60") {
		t.Errorf("opinion show output missing score: %q", out)
	}
	if !strings.Contains(strings.ToLower(out), "hostile") {
		t.Errorf("opinion show output missing tier: %q", out)
	}
	_ = mob // referenced to keep helper output stable
}

func TestAdminOpinionBump(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	opinions.ClearCache()

	user := newTestAdminUser(t, 1)
	target := newTestUser(t, 42)
	_ = newTestMobTemplate(t, 41, "lars")

	if _, err := Opinion("bump lars Aliceia -10", user, getRoomFor(user), 0); err != nil {
		t.Fatal(err)
	}
	if got := opinions.Get(41, target.UserId); got != -10 {
		t.Errorf("after bump -10, Get = %d, want -10", got)
	}
	if _, err := Opinion("bump lars Aliceia -10", user, getRoomFor(user), 0); err != nil {
		t.Fatal(err)
	}
	if got := opinions.Get(41, target.UserId); got != -20 {
		t.Errorf("after second bump, Get = %d, want -20", got)
	}
}

func TestAdminOpinionReset(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	opinions.ClearCache()

	user := newTestAdminUser(t, 1)
	target := newTestUser(t, 42)
	_ = newTestMobTemplate(t, 41, "lars")

	if _, err := Opinion("set lars Aliceia -90", user, getRoomFor(user), 0); err != nil {
		t.Fatal(err)
	}
	if _, err := Opinion("reset lars Aliceia", user, getRoomFor(user), 0); err != nil {
		t.Fatal(err)
	}
	// Default for mob 41 is whatever the test helper set; assume 0 unless
	// newTestMobTemplate is given a default.
	if got := opinions.Get(41, target.UserId); got != 0 {
		t.Errorf("after reset, Get = %d, want 0 (default)", got)
	}
}
```

Adapt `newTestAdminUser`, `newTestUser`, `newTestMobTemplate`,
`getRoomFor`, `captureSendText` to the helpers actually present in
the test file. If a helper is missing, write it minimally. The
behavioral asserts are what we're testing.

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/usercommands/... -run TestAdminOpinion -v`
Expected: FAIL — `Opinion` undefined.

- [ ] **Step 4: Create `admin.opinion.go`**

```go
package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Opinion is the admin command for inspecting and adjusting per-NPC
// disposition scores stored by internal/opinions. Subcommands:
//
//	opinion show <playerName>
//	opinion show <mobName|mobId> <playerName>
//	opinion set <mobName|mobId> <playerName> <score>
//	opinion bump <mobName|mobId> <playerName> <delta>
//	opinion reset <mobName|mobId> <playerName>
func Opinion(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		user.SendText(opinionUsage())
		return true, nil
	}

	switch strings.ToLower(args[0]) {
	case "show":
		return opinionShow(args[1:], user)
	case "set":
		return opinionMutate(args[1:], user, mutateSet)
	case "bump":
		return opinionMutate(args[1:], user, mutateBump)
	case "reset":
		return opinionMutate(args[1:], user, mutateReset)
	default:
		user.SendText(opinionUsage())
		return true, nil
	}
}

func opinionUsage() string {
	return "Usage:\r\n" +
		"  opinion show <playerName>\r\n" +
		"  opinion show <mobName|mobId> <playerName>\r\n" +
		"  opinion set <mobName|mobId> <playerName> <score>\r\n" +
		"  opinion bump <mobName|mobId> <playerName> <delta>\r\n" +
		"  opinion reset <mobName|mobId> <playerName>\r\n"
}

func opinionShow(args []string, user *users.UserRecord) (bool, error) {
	switch len(args) {
	case 1:
		return opinionShowAll(args[0], user)
	case 2:
		return opinionShowOne(args[0], args[1], user)
	default:
		user.SendText(opinionUsage())
		return true, nil
	}
}

// opinionShowAll lists every cached MobOpinions row for the named
// player. Walks the cache (does not lazy-load every mob template
// from disk — that would be too expensive).
func opinionShowAll(playerName string, user *users.UserRecord) (bool, error) {
	target := users.GetByCharacterName(playerName)
	if target == nil {
		user.SendText(fmt.Sprintf("No such player: %s\r\n", playerName))
		return true, nil
	}

	rows := opinions.AllRowsForUser(target.UserId)
	if len(rows) == 0 {
		user.SendText(fmt.Sprintf("No NPCs hold an opinion of %s.\r\n", playerName))
		return true, nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "opinion show %s:\r\n\r\n", playerName)
	fmt.Fprintf(&b, "  %-25s %5s  %-9s  %s\r\n", "Mob", "Score", "Tier", "Last bump")
	fmt.Fprintf(&b, "  %-25s %5s  %-9s  %s\r\n",
		strings.Repeat("-", 25), "-----", "---------", "---------")
	now := util.GetRoundCount()
	for _, r := range rows {
		fmt.Fprintf(&b, "  %-25s %5d  %-9s  %s\r\n",
			truncate(fmt.Sprintf("%s (%d)", r.MobName, r.MobId), 25),
			r.Score, tierName(opinions.TierOf(r.Score)),
			roundsAgo(now, r.LastUpdatedRound))
	}
	user.SendText(b.String())
	return true, nil
}

func opinionShowOne(mobIdent, playerName string, user *users.UserRecord) (bool, error) {
	mobId, mobName, ok := resolveMobIdent(mobIdent)
	if !ok {
		user.SendText(fmt.Sprintf("Unknown mob: %s\r\n", mobIdent))
		return true, nil
	}
	target := users.GetByCharacterName(playerName)
	if target == nil {
		user.SendText(fmt.Sprintf("No such player: %s\r\n", playerName))
		return true, nil
	}
	score := opinions.Get(mobId, target.UserId)
	user.SendText(fmt.Sprintf("%s (%d) → %s: score=%d, tier=%s\r\n",
		mobName, mobId, playerName, score, tierName(opinions.TierOf(score))))
	return true, nil
}

type mutateMode int

const (
	mutateSet mutateMode = iota
	mutateBump
	mutateReset
)

func opinionMutate(args []string, user *users.UserRecord, mode mutateMode) (bool, error) {
	expected := 3
	if mode == mutateReset {
		expected = 2
	}
	if len(args) != expected {
		user.SendText(opinionUsage())
		return true, nil
	}
	mobId, mobName, ok := resolveMobIdent(args[0])
	if !ok {
		user.SendText(fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}
	target := users.GetByCharacterName(args[1])
	if target == nil {
		user.SendText(fmt.Sprintf("No such player: %s\r\n", args[1]))
		return true, nil
	}

	switch mode {
	case mutateSet:
		v, err := strconv.Atoi(args[2])
		if err != nil {
			user.SendText(fmt.Sprintf("Bad score %q: %v\r\n", args[2], err))
			return true, nil
		}
		opinions.Set(mobId, target.UserId, v)
		user.SendText(fmt.Sprintf("Set %s (%d) → %s = %d\r\n", mobName, mobId, args[1], v))
	case mutateBump:
		v, err := strconv.Atoi(args[2])
		if err != nil {
			user.SendText(fmt.Sprintf("Bad delta %q: %v\r\n", args[2], err))
			return true, nil
		}
		opinions.Bump(mobId, target.UserId, v)
		user.SendText(fmt.Sprintf("Bumped %s (%d) → %s by %d (now %d)\r\n",
			mobName, mobId, args[1], v, opinions.Get(mobId, target.UserId)))
	case mutateReset:
		def := mobDefaultDisposition(mobId)
		opinions.Set(mobId, target.UserId, def)
		user.SendText(fmt.Sprintf("Reset %s (%d) → %s to default %d\r\n",
			mobName, mobId, args[1], def))
	}
	return true, nil
}

// resolveMobIdent accepts either a numeric mobId or a mob namesimple
// (lowercase canonical name); returns (mobId, displayName, ok).
func resolveMobIdent(s string) (int, string, bool) {
	if id, err := strconv.Atoi(s); err == nil {
		spec := mobs.GetMobSpec(mobs.MobId(id))
		if spec == nil {
			return 0, "", false
		}
		return id, spec.Character.Name, true
	}
	// Linear scan over loaded mob templates by namesimple.
	wanted := strings.ToLower(s)
	for _, spec := range mobs.AllMobTemplates() {
		if strings.EqualFold(util.ConvertForFilename(spec.Character.Name), wanted) {
			return int(spec.MobId), spec.Character.Name, true
		}
	}
	return 0, "", false
}

func mobDefaultDisposition(mobId int) int {
	spec := mobs.GetMobSpec(mobs.MobId(mobId))
	if spec == nil {
		return 0
	}
	return spec.DefaultDisposition
}

func tierName(t opinions.Tier) string {
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

func roundsAgo(now, then uint64) string {
	if then == 0 {
		return "default"
	}
	delta := now - then
	switch {
	case delta < 60:
		return fmt.Sprintf("%d rounds ago", delta)
	case delta < 3600:
		return fmt.Sprintf("~%d game-min ago", delta/60)
	default:
		return fmt.Sprintf("~%d game-hr ago", delta/3600)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
```

Note the call to `opinions.AllRowsForUser` and `mobs.AllMobTemplates`
— add those if they don't exist:

- `opinions.AllRowsForUser` returns a flat list of `(MobId, MobName,
  Score, LastUpdatedRound)` for every cached MobOpinions row whose
  `Opinions` map contains `userId`. Implementation in
  `internal/opinions/opinions.go`:

```go
type Row struct {
	MobId            int
	MobName          string
	Score            int
	LastUpdatedRound uint64
}

// AllRowsForUser returns every cached row for the given userId.
// Walks only the in-memory cache — does not load from disk.
func AllRowsForUser(userId int) []Row {
	opinionCacheMu.RLock()
	defer opinionCacheMu.RUnlock()
	out := make([]Row, 0)
	for mobId, mo := range opinionCache {
		op, has := mo.Opinions[userId]
		if !has {
			continue
		}
		name, _, ok := resolveTemplate(mobId)
		if !ok {
			name = "?"
		}
		// Decay-adjust before reporting so the admin sees the live view.
		score := decayedScore(op.Score, mo.DefaultDisposition,
			op.LastUpdatedRound, currentRound(), currentHalfLife())
		out = append(out, Row{MobId: mobId, MobName: name,
			Score: score, LastUpdatedRound: op.LastUpdatedRound})
	}
	return out
}
```

- `mobs.AllMobTemplates` — check if it already exists. If not, find
  the existing mob-template registry (`mobsMu`, `mobs` map at the
  top of `internal/mobs/mobs.go`) and add a snapshot accessor:

```go
// AllMobTemplates returns a snapshot of every loaded mob template.
// Read-only — callers must not mutate.
func AllMobTemplates() []*Mob {
	mobsMu.RLock()
	defer mobsMu.RUnlock()
	out := make([]*Mob, 0, len(mobs))
	for _, m := range mobs {
		out = append(out, m)
	}
	return out
}
```

- [ ] **Step 5: Register `Opinion` in the admin command registry**

Find where existing admin commands are registered (look for the
existing `Server` admin command in the registry — same pattern).
Run: `grep -n "admin.*Server\|Server.*admin" internal/usercommands/*.go | head -10`

Follow the registration idiom and add:

```go
"opinion": Opinion,
```

(or however the registry maps names to functions).

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/usercommands/... -run TestAdminOpinion -v`
Expected: all three tests PASS.

- [ ] **Step 7: Run the full suite to catch regressions**

Run: `go test ./... -count=1`
Expected: all pass.

- [ ] **Step 8: Commit**

```bash
git add internal/usercommands/admin.opinion.go internal/usercommands/admin.opinion_test.go internal/opinions/opinions.go internal/mobs/mobs.go
# (also add any registry file modified above)
git commit -m "$(cat <<'EOF'
feat(admin): opinion show/set/bump/reset

Admin command for inspecting and adjusting per-NPC disposition
scores. Walks the opinion cache for show-all (no disk fan-out),
delegates to opinions.Set / Bump for mutations, snaps to the mob
template's default_disposition on reset.

Adds opinions.AllRowsForUser and mobs.AllMobTemplates as the
supporting accessors.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Helpfile + admin command-list listing

**Files:**
- Create: `_datafiles/world/dogmud/templates/help/admin/opinion.template`
  (path verified by listing the existing admin help dir)
- Modify: admin help index (location verified by inspecting an
  existing admin command's help registration)

- [ ] **Step 1: Find the helpfile path convention**

Run: `find _datafiles/world/dogmud/templates/help -type d 2>/dev/null` and pick the directory that admin command helpfiles live in.

Then: `ls _datafiles/world/dogmud/templates/help/admin/ 2>/dev/null | head -10` (or wherever they live).

Pick a peer admin command (e.g., `server.template`) and read it to mirror voice/format.

- [ ] **Step 2: Write the helpfile**

Create the helpfile (path adjusted to whatever Step 1 surfaced):

```
ADMIN COMMAND: OPINION

Inspect and adjust per-NPC disposition scores stored by the
opinion substrate. Scores are signed integers in the range
[-100, +100] and decay slowly toward each NPC's default over
game-time.

USAGE:
  opinion show <playerName>
      List every NPC currently holding an opinion of the player.

  opinion show <mobName|mobId> <playerName>
      Show one specific (NPC × player) row.

  opinion set <mobName|mobId> <playerName> <score>
      Set an absolute score (clamped to [-100, +100]).

  opinion bump <mobName|mobId> <playerName> <delta>
      Add delta to the current decay-adjusted score.

  opinion reset <mobName|mobId> <playerName>
      Snap the row back to the NPC's default_disposition.

TIERS (banded view):
  <= -50      Hostile
  -49 .. -15  Cold
  -14 .. +14  Neutral
  +15 .. +49  Warm
  >= +50      Friendly

NOTES:
  * Scores are template-keyed: every instance of a mob template
    shares one row per player.
  * Decay is computed on read; set values stay anchored at the
    last set/bump until something nudges them again.
  * Players never see these numbers — relationships surface
    through dialogue, NPC behavior, and fiction.
```

Wrap at 80 columns — current draft is fine.

- [ ] **Step 3: Add `opinion` to the admin command index**

Find the admin command index template/file (the listing admins see
when they run `help admin` or similar). Pattern-match to the index
format and add a one-line entry such as:

```
opinion         Inspect/adjust per-NPC disposition scores
```

The exact filename for the index will surface from Step 1.

- [ ] **Step 4: Smoke test the help system**

Boot the local server (`go run main.go` from the repo root) and
log in as an admin. Type `help opinion` and verify the helpfile
renders. Type `help admin` (or whatever surfaces the admin index)
and verify `opinion` appears.

If the server doesn't have `help <admin command>` plumbing for
admin-specific helps, mirror exactly what an existing admin
command does.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/templates/help/admin/opinion.template
# (also add the admin index file modified above)
git commit -m "$(cat <<'EOF'
docs(admin): help text and command-list entry for opinion

Admins can now run `help opinion` to learn the subcommands and see
the tier banding. The admin command index lists `opinion`
alongside other admin tooling.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: End-to-end smoke test — server restart preserves bumps

**Files:**
- Modify: `internal/opinions/persistence_test.go`

- [ ] **Step 1: Append the smoke test**

```go
func TestRestartPreservesOpinionWithDecay(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	ClearCache()

	defaultProviderForTest = func(mobId int) (string, int, bool) { return "lars", 0, true }
	t.Cleanup(func() { defaultProviderForTest = nil })

	halfLifeForTest = func() uint64 { return 100 }
	t.Cleanup(func() { halfLifeForTest = nil })

	roundForTest = func() uint64 { return 1000 }
	Bump(41, 17, -50)

	// Simulate restart: drop cache, leave file on disk.
	ClearCache()

	// "Restart" — round count advances by five half-lives.
	roundForTest = func() uint64 { return 1500 }
	t.Cleanup(func() { roundForTest = nil })

	got := Get(41, 17)
	// -50 decayed by 5 toward 0 → -45.
	if got != -45 {
		t.Errorf("post-restart Get = %d, want -45 (anchored read with decay)", got)
	}
}
```

- [ ] **Step 2: Run the smoke test**

Run: `go test ./internal/opinions/... -run TestRestartPreservesOpinionWithDecay -v`
Expected: PASS.

- [ ] **Step 3: Run the full suite one last time**

Run: `go test ./... -count=1 -race`
Expected: all pass, no races.

- [ ] **Step 4: Boot the server locally and verify**

Per CLAUDE.md's pre-push SOP:
1. Confirm `Logging.LogToFile` flag state matches your intent
   (false for prod-bound, true is fine while developing).
2. Run `go build ./...` — must compile cleanly.
3. Boot the server locally and watch the loader output for any
   panics around `mobs.LoadDataFiles()`. With the new
   `default_disposition` field, mob YAML loading must still pass.
4. Smoke-attack a mob in-game, then `opinion show <yourname>` as
   admin and confirm the bump landed.

- [ ] **Step 5: Commit**

```bash
git add internal/opinions/persistence_test.go
git commit -m "$(cat <<'EOF'
test(opinions): server-restart smoke covers anchored decay

Bumps an opinion, drops the cache, advances round count by five
half-lives, and confirms the on-disk anchor + decay-on-read
combine to give the correct projected score (-45 from -50).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-review

Verified each spec section against tasks:

| Spec section | Task(s) |
|---|---|
| Identity granularity (mobId-keyed) | T1, T4 (file path uses mobId) |
| Score shape (signed scalar + tier) | T1 (Tier enum), T1 (TierOf), T3 (clamp) |
| Player identity (userId) | T5 (API signatures) |
| Default disposition + decay | T2 (decay math), T5 (Bump/Get use default), T8 (mob YAML field) |
| Storage layout (per-NPC YAML) | T4 (opinionPath, save/load) |
| Persistence model (synchronous save) | T4 (saveToDisk), T5 (Set/Bump call saveToDisk) |
| Lifecycle (lazy create, no prune) | T5 (loadOrLazyInit, Get does not create row) |
| Concurrency | T4 (mutex), T6 (parallel test) |
| Public API | T1, T3, T5 |
| Integration scope (combat hookup only) | T9 (attack.go) |
| Admin debug command | T10 |
| Helpfile + index entry | T11 |
| Test plan (unit/persistence/concurrency/integration/admin) | T1-T12 |
| Non-functional: single source of truth | API design (T5) |
| Non-functional: write amp budget | Synchronous + first-aggression-only (T9) |
| Non-functional: startup cost lazy | T5 (loadOrLazyInit), no prewarm |
| Non-functional: decay determinism | T2 (pure functions) |
| Non-functional: no player-facing numbers | Admin-only (T10), helpfile reminds (T11) |

No placeholders, no contradictions, type signatures consistent
across tasks (`func Get(mobId int, userId int) int` everywhere,
`pull(score, def, n int) int`, `decayedScore(...)`, etc.).
