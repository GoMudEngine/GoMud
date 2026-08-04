# Mob Authoring (3) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A **Mobs** tab in the `/build` page where an admin browses/searches mob templates, edits every field with a sectioned form (full parity incl. the embedded character block), creates and deletes mobs (delete gated by a world-wide reference scan), and test-spawns a template into an admin-chosen zone + room — all persisting to mob YAML and live in-game.

**Architecture:** Reuse 1b's admin `/build` GMCP session. New RoleAdmin-gated `Build.Mob.*` handlers run on MainWorker via the existing `GMCPBuildOp` event, mutating templates through a NEW `mobs` persistence seam (`SaveMobSpec`/`DeleteMobSpec`/`CreateNewMobFile` — no mob-template writer exists today). Validation that the mobs package can reach (schedules, patrols, items, buffs, species, archetype files, numerics) lives in `mobs.ValidateMobSpec`; checks needing packages mobs can't import without a cycle (zone existence, craft supports, recipes) live in the gmcp layer. The delete reference scan and test-spawn also live in gmcp behind injected seams so they're unit-testable.

**Tech Stack:** Go (GoMud fork), GMCP over WebSocket, vanilla JS (matching `builder.js`/`items.js`/`build.html`), `go test`.

**Spec:** `docs/superpowers/specs/2026-07-23-mob-authoring-3-design.md`

**Verified seams:**
- `Mob` struct `internal/mobs/mobs.go:82` (embeds `Character characters.Character`); package cache `mobs map[MobId]*Mob` guarded by `mobsMu`; name cache `mobNameCache map[MobId]string` guarded by `mobNameCacheMu`; `allMobNames []string`.
- `Mob.Filename()` `mobs.go:1163` = `<id>-<ConvertForFilename(name)>.yaml` — **consults `mobNameCache` FIRST**, falls back to `Character.Name`. `Mob.Filepath()` `mobs.go:1175` = `ZoneNameSanitize(Zone)/Filename()`. ⇒ on rename, compute the OLD path BEFORE updating `mobNameCache`, then update the cache, then compute the new path.
- Boot: `LoadDataFiles()` `mobs.go:1210` runs `casing.AssertCanonical(mob.Character.Name, …)` (hard panic on non-canonical names) and cross-checks every `schedule_id` resolves. `mobs.GetSchedule(id)` `schedule.go:32` / `mobs.GetPatrol(id)` `patrol.go:31`; registries `schedules map[string]*Schedule` `schedule.go:27`, `patrols map[string]*Patrol` `patrol.go:26` (no list-all accessor yet — Task 1 adds them).
- Legacy `(*Mob).Save()` `mobs.go:1180` is broken upstream junk (no zone folder, double `.yaml`) — do NOT reuse; `SaveMobSpec` supersedes it.
- `mobs.AllMobTemplates()` `mobs.go:235`; `mobs.MobIdByName(name)` `mobs.go:278`; `mobs.NewMobById(mobId, roomId)` `mobs.go:326` + `room.AddMob(instanceId)` = the spawn path (`admin.mob.go:200-201`).
- `characters.Character` `character.go:107`: `Name`, `Description`, `Adjectives []string`, `SpeciesId int`, `Stats stats.Statistics` (per-stat `.Training` int, yaml `stats.<stat>.training`), `Gold int`, `Shop Shop` (`shop.go:16`: `[]ShopItem{MobId,ItemId,BuffId,PetType,Quantity,QuantityMax,Price…}`), `Items []items.Item`, `Equipment Worn` (`worn.go:8`: slot fields `Weapon/Offhand/Head/Neck/Shoulders/Body/Back/Belt/Wrist1/Wrist2/Gloves/Ring/Ring2/Legs/Feet…`, each `items.Item`).
- `species.GetAllSpecies() []Species` / `species.GetSpecies(id)` `internal/species/species.go:65/73` (`Species{SpeciesId, Name, …}`).
- Behavior archetypes are YAML files at `{DataFiles}/behaviors/archetypes/{name}.yaml` (`behaviortree/helpers.go:82`); no registry accessor — validate/enumerate via the filesystem.
- `rooms.SpawnInfo` `spawninfo.go:3` (`MobId int \`yaml:"mobid"\``); dialogue files at `dialogue/<sanitized-zone>/<mobId>.yaml` (`dialogue/loader.go:34`); mercenary sale = `ShopItem.MobId` on OTHER mobs' shops.
- `shops.ValidCraftSupports` / `shops.IsValidCraftSupport` (`shops/shopinventory.go:26`).
- 1b/2 patterns to MIRROR: `gmcp.Build.go` (`BuildResult`, `buildErr`, `requireAdmin`, `GMCPBuildOp`→`handleBuildOp`, `sendBuildResult`), `gmcp.Item.go` (deps seam, `specToReq`/`reqToSpec`, enum providers, routing), `items.js`/`build.html` (tab toggle, list, form, toasts, dirty tracking).
- **Implementer must verify at task time (codegraph):** `Character.Level` field name for the YAML `level:` key; the `items` package constructor for a fresh item instance (`items.New(itemId)` or equivalent — whatever `give`/spawn paths use); `Quest` struct field names for mob references; `crafting.GetAllRecipes` + `RecipeSpec` field names (already used by the item scan — copy its accessors); conversation pair file layout under `conversations/pairs/`.

---

## File Structure

- `internal/mobs/save.go` *(new)* — `CanonicalizeMobName`, `ValidateMobSpec`, `SaveMobSpec`, `DeleteMobSpec`, `CreateNewMobFile`, `AllScheduleIds`, `AllPatrolIds`. (Task 1)
- `internal/mobs/save_test.go` *(new)* — tests. (Task 1)
- `modules/gmcp/gmcp.Mob.go` *(new)* — `Build.Mob.*` payloads, detail + enums, `mobDeps` seam, core funcs, reference scan, spawn, senders, routing cases. (Tasks 2,3,4)
- `modules/gmcp/gmcp.Mob_test.go` *(new)* — unit tests. (Tasks 2,3,4)
- `modules/gmcp/gmcp.Build.go` — add `MobId`/`MobRefs` to `BuildResult`; route cases in `handleBuildOp`. (Tasks 2,3,4)
- `modules/gmcp/gmcp.go` — add `Build.Mob.*` to the deferred `Build.*` dispatch list. (Task 2)
- `_datafiles/html/public/static/js/mobs.js` *(new)* — mob list + sectioned form + test-spawn control. (Tasks 5,6,7)
- `_datafiles/html/public/build.html` — `Rooms | Items | Mobs` toggle, load `mobs.js`, route `Build.Mob*` GMCP. (Tasks 5,6,7)

---

## Task 1: Mob persistence seams — `ValidateMobSpec` + `SaveMobSpec` + `DeleteMobSpec` + `CreateNewMobFile`

**Files:** Create `internal/mobs/save.go`, `internal/mobs/save_test.go`.

- [ ] **Step 1: Failing tests.** The non-trivial behaviors: (a) **rename relocates the file** (filename embeds the cached name — the old path must be computed from the pre-update cache); (b) **re-zone relocates** (path embeds the zone folder); (c) **delete removes file + caches**; (d) **validation rejects dangling refs** (schedule, buff, loot item); (e) **a fresh stub passes validation** (boot-safe). Use `t.TempDir()`.

```go
package mobs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/characters"
)

// pointMobDataFilesAt sets FilePaths.DataFiles to dir for the test duration.
func pointMobDataFilesAt(t *testing.T, dir string) {
	t.Helper()
	prev := configs.GetFilePathsConfig().DataFiles.String()
	if err := configs.SetVal("FilePaths.DataFiles", dir); err != nil {
		t.Fatalf("set DataFiles: %v", err)
	}
	for _, sub := range []string{"mobs/testzone", "behaviors/archetypes"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() { configs.SetVal("FilePaths.DataFiles", prev) })
}

func seedMob(t *testing.T, id MobId, name string) Mob {
	t.Helper()
	m := Mob{MobId: id, Zone: "Testzone", StatPool: 10, ActivityLevel: 5}
	m.Character = characters.Character{Name: name, Description: "A test mob.", SpeciesId: 1}
	mobsMu.Lock()
	cp := m
	mobs[id] = &cp
	mobsMu.Unlock()
	mobNameCacheMu.Lock()
	mobNameCache[id] = name
	mobNameCacheMu.Unlock()
	t.Cleanup(func() {
		mobsMu.Lock()
		delete(mobs, id)
		mobsMu.Unlock()
		mobNameCacheMu.Lock()
		delete(mobNameCache, id)
		mobNameCacheMu.Unlock()
	})
	return m
}

func TestSaveMobSpec_RelocatesFileOnRename(t *testing.T) {
	dir := t.TempDir()
	pointMobDataFilesAt(t, dir)
	m := seedMob(t, 99901, "Test Grunt")
	if err := SaveMobSpec(m); err != nil {
		t.Fatalf("initial save: %v", err)
	}
	oldPath := filepath.Join(dir, "mobs", "testzone", "99901-test_grunt.yaml")
	if _, err := os.Stat(oldPath); err != nil {
		t.Fatalf("expected %s: %v", oldPath, err)
	}

	renamed := m
	renamed.Character.Name = "Test Bruiser"
	if err := SaveMobSpec(renamed); err != nil {
		t.Fatalf("rename save: %v", err)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old file %s should be gone after rename", oldPath)
	}
	newPath := filepath.Join(dir, "mobs", "testzone", "99901-test_bruiser.yaml")
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("new file %s should exist: %v", newPath, err)
	}
	if mobNameCache[99901] != "Test Bruiser" {
		t.Errorf("name cache not updated: %q", mobNameCache[99901])
	}
	if mobs[99901].Character.Name != "Test Bruiser" {
		t.Errorf("template cache not updated")
	}
}

func TestSaveMobSpec_RejectsDanglingRefs(t *testing.T) {
	dir := t.TempDir()
	pointMobDataFilesAt(t, dir)
	m := seedMob(t, 99902, "Ref Tester")

	bad := m
	bad.ScheduleId = "no_such_schedule"
	if err := SaveMobSpec(bad); err == nil {
		t.Error("expected rejection: dangling schedule_id")
	}
	bad = m
	bad.BuffIds = []int{999999}
	if err := SaveMobSpec(bad); err == nil {
		t.Error("expected rejection: dangling buff id")
	}
	bad = m
	bad.LootPool = []int{999999}
	if err := SaveMobSpec(bad); err == nil {
		t.Error("expected rejection: dangling loot item id")
	}
	if _, err := os.Stat(filepath.Join(dir, "mobs", "testzone", "99902-ref_tester.yaml")); !os.IsNotExist(err) {
		t.Error("no file should have been written for rejected saves")
	}
}

func TestDeleteMobSpec_RemovesFileAndCaches(t *testing.T) {
	dir := t.TempDir()
	pointMobDataFilesAt(t, dir)
	m := seedMob(t, 99903, "Doomed Mob")
	if err := SaveMobSpec(m); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "mobs", "testzone", "99903-doomed_mob.yaml")
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	if err := DeleteMobSpec(99903); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Errorf("file %s should be gone", p)
	}
	if _, ok := mobs[99903]; ok {
		t.Error("template cache entry should be gone")
	}
	if _, ok := mobNameCache[99903]; ok {
		t.Error("name cache entry should be gone")
	}
}

func TestCreateNewMobFile_StubIsBootSafe(t *testing.T) {
	dir := t.TempDir()
	pointMobDataFilesAt(t, dir)
	id, err := CreateNewMobFile("Testzone")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		mobsMu.Lock()
		delete(mobs, id)
		mobsMu.Unlock()
		mobNameCacheMu.Lock()
		delete(mobNameCache, id)
		mobNameCacheMu.Unlock()
	})
	tmpl := GetMobSpec(id)
	if tmpl == nil {
		t.Fatal("stub not in cache")
	}
	if err := ValidateMobSpec(tmpl); err != nil {
		t.Errorf("fresh stub must be boot-safe, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "mobs", "testzone", tmpl.Filename())); err != nil {
		t.Errorf("stub file missing: %v", err)
	}
}
```

> If `GetMobSpec` isn't the template accessor's exact name, check `mobs.go` (the item plan verified `AllMobTemplates`; the single-template getter is nearby — `instance_save.go:111` calls `GetMobSpec(mob.MobId)`, so it exists).

- [ ] **Step 2: Run — FAIL** (undefined SaveMobSpec etc.). `go test ./internal/mobs/ -run 'TestSaveMobSpec|TestDeleteMobSpec|TestCreateNewMobFile'`

- [ ] **Step 3: Implement `internal/mobs/save.go`.**

```go
package mobs

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/casing"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func mobsBasePath() string {
	return configs.GetFilePathsConfig().DataFiles.String() + `/mobs`
}

// CanonicalizeMobName normalizes the mob's player-visible name to the Title
// casing LoadDataFiles' casing.AssertCanonical demands — a single
// non-canonical save bricks the NEXT boot. Every write path runs this first.
func CanonicalizeMobName(m *Mob) {
	m.Character.Name = casing.Title(m.Character.Name)
}

// AllScheduleIds / AllPatrolIds enumerate the loaded registries (sorted) for
// the builder's dropdowns and for validation error messages.
func AllScheduleIds() []string {
	out := make([]string, 0, len(schedules))
	for id := range schedules {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
func AllPatrolIds() []string {
	out := make([]string, 0, len(patrols))
	for id := range patrols {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

var validArchetypes = map[string]bool{"": true, "fighting": true, "casting": true}
var validAIProfiles = map[string]bool{"": true, "default": true, "aggressive": true,
	"defensive": true, "grappler": true, "brawler": true, "tactical": true}
var validSubmissionPolicies = map[string]bool{"": true, "mercy": true, "subdue": true, "cripple": true, "lethal": true}

// ValidateMobSpec replicates every boot-time check the mobs package can reach
// (gmcp adds zone/craft-support/recipe checks). Errors name the field and the
// valid values so the builder can surface them verbatim.
func ValidateMobSpec(m *Mob) error {
	if strings.TrimSpace(m.Character.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if m.Zone == "" {
		return fmt.Errorf("zone is required")
	}
	if m.ScheduleId != "" && GetSchedule(m.ScheduleId) == nil {
		return fmt.Errorf("schedule_id %q does not resolve; valid: %s", m.ScheduleId, strings.Join(AllScheduleIds(), ", "))
	}
	if m.PatrolId != "" && GetPatrol(m.PatrolId) == nil {
		return fmt.Errorf("patrol_id %q does not resolve; valid: %s", m.PatrolId, strings.Join(AllPatrolIds(), ", "))
	}
	if m.BehaviorArchetype != "" {
		p := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `behaviors`, `/`, `archetypes`, `/`, m.BehaviorArchetype+`.yaml`)
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("behavior_archetype %q has no archetype file", m.BehaviorArchetype)
		}
	}
	if species.GetSpecies(m.Character.SpeciesId) == nil {
		return fmt.Errorf("speciesid %d is not a valid species", m.Character.SpeciesId)
	}
	if !validArchetypes[m.Archetype] {
		return fmt.Errorf("archetype %q invalid; valid: fighting, casting, or empty", m.Archetype)
	}
	if !validAIProfiles[m.AIProfile] {
		return fmt.Errorf("aiprofile %q invalid; valid: default, aggressive, defensive, grappler, brawler, tactical", m.AIProfile)
	}
	if !validSubmissionPolicies[m.SubmissionPolicy] {
		return fmt.Errorf("submission_policy %q invalid; valid: mercy, subdue, cripple, lethal", m.SubmissionPolicy)
	}
	// surrender_policy: "", "never", "always", or "auto-tap-below <N>"
	if sp := m.SurrenderPolicy; sp != "" && sp != "never" && sp != "always" && !strings.HasPrefix(sp, "auto-tap-below ") {
		return fmt.Errorf(`surrender_policy %q invalid; valid: never, always, "auto-tap-below <N>"`, sp)
	}
	for _, bid := range m.BuffIds {
		if !buffs.BuffSpecExists(bid) {
			return fmt.Errorf("buff id %d does not exist", bid)
		}
	}
	for _, iid := range m.LootPool {
		if items.GetItemSpec(iid) == nil {
			return fmt.Errorf("loot_pool item %d does not exist", iid)
		}
	}
	for _, si := range m.Character.Shop {
		if si.ItemId > 0 && items.GetItemSpec(si.ItemId) == nil {
			return fmt.Errorf("shop item %d does not exist", si.ItemId)
		}
	}
	for _, it := range m.Character.Items {
		if it.ItemId > 0 && items.GetItemSpec(it.ItemId) == nil {
			return fmt.Errorf("carried item %d does not exist", it.ItemId)
		}
	}
	for _, eq := range equippedItemIds(m.Character.Equipment) {
		if items.GetItemSpec(eq) == nil {
			return fmt.Errorf("equipped item %d does not exist", eq)
		}
	}
	if m.StatPool < 0 {
		return fmt.Errorf("statpool must be >= 0")
	}
	for name, v := range map[string]int{"activitylevel": m.ActivityLevel, "itemdropchance": m.ItemDropChance,
		"mutationchance": m.MutationChance, "specialmovechance": m.SpecialMoveChance} {
		if v < 0 || v > 100 {
			return fmt.Errorf("%s must be 0-100, got %d", name, v)
		}
	}
	return nil
}

// equippedItemIds collects the non-zero item ids across all Worn slots.
func equippedItemIds(w characters.Worn) []int {
	out := []int{}
	for _, it := range []items.Item{w.Weapon, w.Offhand, w.Head, w.Neck, w.Shoulders, w.Body,
		w.Back, w.Belt, w.Wrist1, w.Wrist2, w.Gloves, w.Ring, w.Ring2, w.Legs, w.Feet} {
		if it.ItemId > 0 {
			out = append(out, it.ItemId)
		}
	}
	return out
}

// SaveMobSpec validates and writes a mob template. Mob filenames embed the
// (name-cached) canonical name under the zone folder, so a rename or re-zone
// changes the path — compute the OLD path from the PRE-update caches, remove
// it, update the caches, then write. A stale duplicate must never linger to
// boot as a second copy.
func SaveMobSpec(m Mob) error {
	CanonicalizeMobName(&m)
	if err := ValidateMobSpec(&m); err != nil {
		return err
	}
	base := mobsBasePath()

	// Old path: Filepath() reads mobNameCache, which still holds the old name.
	mobsMu.RLock()
	old, existed := mobs[m.MobId]
	mobsMu.RUnlock()
	var oldRel string
	if existed {
		oldRel = old.Filepath()
	}

	// Update the name cache BEFORE computing the new path (Filename() reads it).
	mobNameCacheMu.Lock()
	mobNameCache[m.MobId] = m.Character.Name
	mobNameCacheMu.Unlock()

	newRel := m.Filepath()
	if existed && oldRel != newRel {
		if err := os.Remove(util.FilePath(base + `/` + oldRel)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	saveModes := []fileloader.SaveOption{}
	if configs.GetFilePathsConfig().CarefulSaveFiles {
		saveModes = append(saveModes, fileloader.SaveCareful)
	}
	if err := fileloader.SaveFlatFile[*Mob](base, &m, saveModes...); err != nil {
		return err
	}

	cp := m
	mobsMu.Lock()
	mobs[m.MobId] = &cp
	mobsMu.Unlock()
	return nil
}

// DeleteMobSpec removes a mob template's file and cache entries. Callers must
// first confirm the mob is unreferenced (the web builder's reference scan).
func DeleteMobSpec(mobId MobId) error {
	mobsMu.RLock()
	tmpl, ok := mobs[mobId]
	mobsMu.RUnlock()
	if !ok {
		return fmt.Errorf(`mob %d not found`, int(mobId))
	}
	if err := os.Remove(util.FilePath(mobsBasePath() + `/` + tmpl.Filepath())); err != nil && !os.IsNotExist(err) {
		return err
	}
	mobsMu.Lock()
	delete(mobs, mobId)
	mobsMu.Unlock()
	mobNameCacheMu.Lock()
	delete(mobNameCache, mobId)
	mobNameCacheMu.Unlock()
	return nil
}

// CreateNewMobFile seeds a boot-safe stub in the given zone at the next free
// mob id (cache max + 1 — the cache mirrors the filesystem at boot and every
// builder save keeps it fresh) and persists it.
func CreateNewMobFile(zone string) (MobId, error) {
	if zone == "" {
		return 0, fmt.Errorf("zone is required")
	}
	mobsMu.RLock()
	nextId := MobId(0)
	for id := range mobs {
		if id > nextId {
			nextId = id
		}
	}
	mobsMu.RUnlock()
	nextId++

	stub := Mob{
		MobId:         nextId,
		Zone:          zone,
		StatPool:      10,
		ActivityLevel: 5,
	}
	stub.Character = characters.Character{
		Name:        fmt.Sprintf("New Mob %d", int(nextId)),
		Description: "An unfinished mob.",
		SpeciesId:   1,
	}
	if err := SaveMobSpec(stub); err != nil {
		return 0, err
	}
	return nextId, nil
}
```

> **Implementer notes:** (1) verify `buffs.BuffSpecExists` — if the accessor is `buffs.GetBuffSpec(id) != nil`, use that; (2) verify `Worn` slot list against `worn.go` (Feet + any slots I elided — tail/componentbag; include ALL item-bearing slots); (3) `fileloader.SaveFlatFile[*Mob]` requires `*Mob` to satisfy the same interface `LoadAllFlatFiles[int,*Mob]` uses (`Id()`, `Validate()`, `Filepath()`) — it already does; (4) if `configs.SetVal` in tests races other mob tests, run these serially (no `t.Parallel()`).

- [ ] **Step 4: Run — PASS.** `go test ./internal/mobs/ -run 'TestSaveMobSpec|TestDeleteMobSpec|TestCreateNewMobFile'`

- [ ] **Step 5: Full mobs tests + gofmt.** `go test ./internal/mobs/ && gofmt -l internal/mobs/save.go` (expect empty).

- [ ] **Step 6: Commit.**
```bash
git add internal/mobs/save.go internal/mobs/save_test.go
git commit -m "feat(mobs): SaveMobSpec/DeleteMobSpec/CreateNewMobFile + ValidateMobSpec (builder persistence)

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

## Task 2: `Build.Mob.*` GMCP core — list / get / create / update

**Files:** Create `modules/gmcp/gmcp.Mob.go`, `modules/gmcp/gmcp.Mob_test.go`; Modify `modules/gmcp/gmcp.Build.go`, `modules/gmcp/gmcp.go`.

Mirror `gmcp.Item.go`. Mob mutations touch shared package maps + files ⇒ MainWorker only (`GMCPBuildOp` deferral).

- [ ] **Step 1: Failing tests** for the core behind a `mobDeps` seam (fake world). Cover: update round-trips root + character fields AND **preserves fields the form doesn't carry** (`Relationships`, `KnowsFacts` pass through from base when req omits them — the reqToMob base-copy contract); create returns id; get maps fields; gmcp-layer zone validation rejects an unknown zone.

```go
package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

type fakeMobWorld struct {
	specs   map[int]*mobs.Mob
	saved   []mobs.Mob
	deleted []int
	nextId  int
}

func newFakeMobWorld() *fakeMobWorld {
	return &fakeMobWorld{specs: map[int]*mobs.Mob{}, nextId: 90000}
}

func (w *fakeMobWorld) deps() mobDeps {
	return mobDeps{
		load: func(id int) *mobs.Mob { return w.specs[id] },
		save: func(m mobs.Mob) error {
			w.saved = append(w.saved, m)
			cp := m
			w.specs[int(m.MobId)] = &cp
			return nil
		},
		del: func(id int) error { w.deleted = append(w.deleted, id); delete(w.specs, id); return nil },
		create: func(zone string) (int, error) {
			w.nextId++
			m := mobs.Mob{MobId: mobs.MobId(w.nextId), Zone: zone}
			m.Character.Name = "New Mob"
			w.specs[w.nextId] = &m
			return w.nextId, nil
		},
		zoneExists: func(z string) bool { return z == "Testzone" },
		references: func(id int) []mobRef { return nil },
		spawn:      func(mobId, roomId int) (string, error) { return "", nil },
	}
}

func TestBuildMobUpdate_RoundTripsAndPreserves(t *testing.T) {
	w := newFakeMobWorld()
	base := &mobs.Mob{MobId: 90001, Zone: "Testzone",
		Relationships: []mobs.RelationshipYAMLEntry{{To: 42, Type: "friend"}},
		KnowsFacts:    []string{"fact_a"}}
	base.Character.Name = "Old Name"
	w.specs[90001] = base

	req := mobUpdateReq{MobId: 90001, Zone: "Testzone", Name: "Keen Guard",
		Description: "Sharp-eyed.", SpeciesId: 1, StatPool: 60, Archetype: "fighting",
		AutoAggro: true, AIProfile: "tactical", ActivityLevel: 10,
		IdleCommands: []string{"emote scans the road."}, Gold: 25,
		StatTraining: map[string]int{"strength": 10, "perception": 5},
		LootPool:     []int{40001}, Relationships: nil /* form didn't touch them */}
	res := buildMobUpdate(w.deps(), req)
	if !res.Ok {
		t.Fatalf("update should succeed, got %+v", res)
	}
	got := w.saved[0]
	if got.Character.Name != "Keen Guard" || !got.AutoAggro || got.AIProfile != "tactical" ||
		got.StatPool != 60 || got.Character.Gold != 25 ||
		got.Character.Stats.Strength.Training != 10 {
		t.Errorf("fields not round-tripped: %+v", got)
	}
	if len(got.Relationships) != 1 || len(got.KnowsFacts) != 1 {
		t.Errorf("form-absent fields must be preserved from base, got rel=%v facts=%v",
			got.Relationships, got.KnowsFacts)
	}
}

func TestBuildMobCreate_RejectsUnknownZone(t *testing.T) {
	w := newFakeMobWorld()
	if res := buildMobCreate(w.deps(), "Nowhere"); res.Ok {
		t.Fatal("create must reject an unknown zone")
	}
	res := buildMobCreate(w.deps(), "Testzone")
	if !res.Ok || res.MobId == 0 {
		t.Fatalf("create should return an id, got %+v", res)
	}
}

func TestBuildMobGet_MapsFields(t *testing.T) {
	w := newFakeMobWorld()
	m := &mobs.Mob{MobId: 90005, Zone: "Testzone", StatPool: 40, Archetype: "casting",
		ScheduleId: "tz_sched", MaxWander: 3}
	m.Character.Name = "Hermit"
	m.Character.Description = "Quiet."
	m.Character.SpeciesId = 1
	w.specs[90005] = m
	d, ok := buildMobGet(w.deps(), 90005)
	if !ok {
		t.Fatal("expected found")
	}
	if d.Name != "Hermit" || d.StatPool != 40 || d.Archetype != "casting" || d.ScheduleId != "tz_sched" {
		t.Errorf("detail wrong: %+v", d)
	}
}
```

- [ ] **Step 2: Run — FAIL.** `go test ./modules/gmcp/ -run TestBuildMob`

- [ ] **Step 3a: Extend `BuildResult`** (in `gmcp.Build.go`):
```go
	MobId int `json:"mobId,omitempty"` // the created/updated/spawned mob
```
(`MobRefs` added in Task 3.)

- [ ] **Step 3b: Implement `gmcp.Mob.go`** — payloads, detail, seam, core. The full-parity field set; `reqToMob` starts from the loaded template so anything the form doesn't carry survives a save (LLMProfile round-trips as raw JSON — see Task 6).

```go
package gmcp

import (
	"os"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ---- client -> server payloads ----
type mobGetReq struct{ MobId int `json:"mobId"` }
type mobCreateReq struct{ Zone string `json:"zone"` }
type mobDeleteReq struct{ MobId int `json:"mobId"` }
type mobSpawnReq struct {
	MobId  int `json:"mobId"`
	RoomId int `json:"roomId"`
}

type mobShopRow struct {
	ItemId      int `json:"itemId"`
	QuantityMax int `json:"quantityMax"`
	Price       int `json:"price"`
}
type mobRelRow struct {
	To      int    `json:"to"`
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
}

type mobUpdateReq struct {
	MobId int    `json:"mobId"`
	Zone  string `json:"zone"`
	// character block
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Adjectives   []string       `json:"adjectives"`
	SpeciesId    int            `json:"speciesId"`
	Gold         int            `json:"gold"`
	StatTraining map[string]int `json:"statTraining"` // strength/dexterity/perception/vitality/willpower/charisma
	Equipment    map[string]int `json:"equipment"`    // worn-slot -> itemId (0/absent = empty)
	CarriedItems []int          `json:"carriedItems"` // itemIds
	Shop         []mobShopRow   `json:"shop"`
	// stats & combat
	StatPool          int            `json:"statPool"`
	Archetype         string         `json:"archetype"`
	AutoAggro         bool           `json:"autoAggro"`
	AIProfile         string         `json:"aiProfile"`
	SpecialMoveChance int            `json:"specialMoveChance"`
	MovePreferences   map[string]int `json:"movePreferences"`
	ActivityLevel     int            `json:"activityLevel"`
	MaxWander         int            `json:"maxWander"`
	Routine           string         `json:"routine"`
	RoutineLinks      []string       `json:"routineLinks"`
	Hates             []string       `json:"hates"`
	Groups            []string       `json:"groups"`
	SubmissionPolicy  string         `json:"submissionPolicy"`
	SurrenderPolicy   string         `json:"surrenderPolicy"`
	PackFleeImmune    bool           `json:"packFleeImmune"`
	// flavor
	IdleCommands   []string `json:"idleCommands"`
	CombatCommands []string `json:"combatCommands"`
	AngryCommands  []string `json:"angryCommands"`
	// loot
	ItemDropChance int   `json:"itemDropChance"`
	LootPool       []int `json:"lootPool"`
	// commerce
	BuysGeneral             bool     `json:"buysGeneral"`
	StockMultiplier         float64  `json:"stockMultiplier"`
	CraftSupport            string   `json:"craftSupport"`
	Crafter                 bool     `json:"crafter"`
	CrafterSkill            string   `json:"crafterSkill"`
	CrafterRecipeIds        []string `json:"crafterRecipeIds"`
	CrafterRestockMaterials []int    `json:"crafterRestockMaterials"`
	// aliveness
	ScheduleId         string      `json:"scheduleId"`
	PatrolId           string      `json:"patrolId"`
	Relationships      []mobRelRow `json:"relationships"`
	KnowsFacts         []string    `json:"knowsFacts"`
	DefaultDisposition int         `json:"defaultDisposition"`
	FoldAnchorRoom     int         `json:"foldAnchorRoom"`
	StorageChestRoom   int         `json:"storageChestRoom"`
	// hooks
	ScriptTag         string   `json:"scriptTag"`
	BehaviorArchetype string   `json:"behaviorArchetype"`
	BuffIds           []int    `json:"buffIds"`
	QuestFlags        []string `json:"questFlags"`
	SpawnMutations    []string `json:"spawnMutations"`
	MutationChance    int      `json:"mutationChance"`
	LLMProfileJSON    string   `json:"llmProfileJson"` // raw round-trip; empty = clear, "-" sentinel = untouched
	// overrides & flags
	CarryCapacity      float64 `json:"carryCapacity"`
	HealthMax          int     `json:"healthMax"`
	StaminaMax         int     `json:"staminaMax"`
	CorpseName         string  `json:"corpseName"`
	CorpseDescription  string  `json:"corpseDescription"`
	HideEquipmentSlots bool    `json:"hideEquipmentSlots"`
	CharmImmune        bool    `json:"charmImmune"`
	NonCombatant       bool    `json:"nonCombatant"`
	PlayerAttackImmune bool    `json:"playerAttackImmune"`
}

// ---- server -> client detail (Build.Mob) ----
type mobEnums struct {
	Zones              []string          `json:"zones"`
	Species            map[string]string `json:"species"` // id -> name (string keys for JSON)
	Archetypes         []string          `json:"archetypes"`
	AIProfiles         []string          `json:"aiProfiles"`
	BehaviorArchetypes []string          `json:"behaviorArchetypes"`
	ScheduleIds        []string          `json:"scheduleIds"`
	PatrolIds          []string          `json:"patrolIds"`
	CraftSupports      []string          `json:"craftSupports"`
	SubmissionPolicies []string          `json:"submissionPolicies"`
	WornSlots          []string          `json:"wornSlots"`
	Groups             []string          `json:"groups"` // observed values, suggestions
}
type mobDetail struct {
	mobUpdateReq
	Enums mobEnums `json:"enums"`
}

type mobListRow struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	Zone         string `json:"zone"`
	StatPool     int    `json:"statPool"`
	NonCombatant bool   `json:"nonCombatant"`
	HasSchedule  bool   `json:"hasSchedule"`
	HasShop      bool   `json:"hasShop"`
}

// ---- dependency seam ----
type mobRef struct {
	Kind string `json:"kind"` // room-spawn | relationship | quest | dialogue | conversation | merc-shop
	Id   string `json:"id"`
}
type mobDeps struct {
	load       func(id int) *mobs.Mob
	save       func(m mobs.Mob) error
	del        func(id int) error
	create     func(zone string) (int, error)
	zoneExists func(zone string) bool
	references func(id int) []mobRef
	spawn      func(mobId, roomId int) (string, error) // returns spawned mob name
}

func realMobDeps() mobDeps {
	return mobDeps{
		load: func(id int) *mobs.Mob { return mobs.GetMobSpec(mobs.MobId(id)) },
		save: mobs.SaveMobSpec,
		del:  func(id int) error { return mobs.DeleteMobSpec(mobs.MobId(id)) },
		create: func(zone string) (int, error) {
			id, err := mobs.CreateNewMobFile(zone)
			return int(id), err
		},
		zoneExists: func(zone string) bool { return rooms.GetZoneConfig(zone) != nil },
		references: scanMobReferences, // Task 3
		spawn:      spawnMobInRoom,    // Task 4
	}
}

// ---- core ----
func buildMobList(d mobDeps) []mobListRow {
	rows := []mobListRow{}
	for _, m := range mobs.AllMobTemplates() {
		rows = append(rows, mobListRow{
			Id: int(m.MobId), Name: m.Character.Name, Zone: m.Zone, StatPool: m.StatPool,
			NonCombatant: m.IsNonCombatant(), HasSchedule: m.ScheduleId != "", HasShop: len(m.Character.Shop) > 0,
		})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Id < rows[j].Id })
	return rows
}

var statKeys = []string{"strength", "dexterity", "perception", "vitality", "willpower", "charisma"}

func mobToReq(m *mobs.Mob) mobUpdateReq {
	req := mobUpdateReq{
		MobId: int(m.MobId), Zone: m.Zone,
		Name: m.Character.Name, Description: m.Character.Description, Adjectives: m.Character.Adjectives,
		SpeciesId: m.Character.SpeciesId, Gold: m.Character.Gold,
		StatPool: m.StatPool, Archetype: m.Archetype, AutoAggro: m.AutoAggro, AIProfile: m.AIProfile,
		SpecialMoveChance: m.SpecialMoveChance, MovePreferences: m.MovePreferences,
		ActivityLevel: m.ActivityLevel, MaxWander: m.MaxWander,
		Routine: m.Routine, RoutineLinks: m.RoutineLinks, Hates: m.Hates, Groups: m.Groups,
		SubmissionPolicy: m.SubmissionPolicy, SurrenderPolicy: m.SurrenderPolicy, PackFleeImmune: m.PackFleeImmune,
		IdleCommands: m.IdleCommands, CombatCommands: m.CombatCommands, AngryCommands: m.AngryCommands,
		ItemDropChance: m.ItemDropChance, LootPool: m.LootPool,
		BuysGeneral: m.BuysGeneral, StockMultiplier: m.StockMultiplier, CraftSupport: m.ShopCraftSupport,
		Crafter: m.Crafter, CrafterSkill: m.CrafterSkill, CrafterRecipeIds: m.CrafterRecipeIds,
		CrafterRestockMaterials: m.CrafterRestockMaterials,
		ScheduleId:              m.ScheduleId, PatrolId: m.PatrolId,
		KnowsFacts: m.KnowsFacts, DefaultDisposition: m.DefaultDisposition,
		FoldAnchorRoom: m.FoldAnchorRoom, StorageChestRoom: m.StorageChestRoom,
		ScriptTag: m.ScriptTag, BehaviorArchetype: m.BehaviorArchetype,
		BuffIds: m.BuffIds, QuestFlags: m.QuestFlags,
		SpawnMutations: m.SpawnMutations, MutationChance: m.MutationChance,
		CarryCapacity: m.CarryCapacityOverride, HealthMax: m.HealthMaxOverride, StaminaMax: m.StaminaMaxOverride,
		CorpseName: m.CorpseName, CorpseDescription: m.CorpseDescription,
		HideEquipmentSlots: m.HideEquipmentSlots, CharmImmune: m.CharmImmune,
		NonCombatant: m.NonCombatant, PlayerAttackImmune: m.PlayerAttackImmune,
	}
	req.StatTraining = map[string]int{
		"strength": m.Character.Stats.Strength.Training, "dexterity": m.Character.Stats.Dexterity.Training,
		"perception": m.Character.Stats.Perception.Training, "vitality": m.Character.Stats.Vitality.Training,
		"willpower": m.Character.Stats.Willpower.Training, "charisma": m.Character.Stats.Charisma.Training,
	}
	req.Equipment = wornToMap(m.Character.Equipment)
	for _, it := range m.Character.Items {
		req.CarriedItems = append(req.CarriedItems, it.ItemId)
	}
	for _, si := range m.Character.Shop {
		if si.ItemId > 0 {
			req.Shop = append(req.Shop, mobShopRow{ItemId: si.ItemId, QuantityMax: si.QuantityMax, Price: si.Price})
		}
	}
	for _, r := range m.Relationships {
		req.Relationships = append(req.Relationships, mobRelRow{To: r.To, Type: r.Type, Subtype: r.Subtype})
	}
	req.LLMProfileJSON = llmProfileToJSON(m.LLMProfile) // "" when nil
	return req
}

func buildMobGet(d mobDeps, mobId int) (mobDetail, bool) {
	m := d.load(mobId)
	if m == nil {
		return mobDetail{}, false
	}
	return mobDetail{mobUpdateReq: mobToReq(m), Enums: collectMobEnums()}, true
}

func reqToMob(base *mobs.Mob, req mobUpdateReq) mobs.Mob {
	m := *base // preserve anything the form doesn't carry (BTree state, VisitedZones, etc.)
	m.Zone = req.Zone
	m.Character.Name, m.Character.Description = req.Name, req.Description
	m.Character.Adjectives, m.Character.SpeciesId, m.Character.Gold = req.Adjectives, req.SpeciesId, req.Gold
	m.Character.Stats.Strength.Training = req.StatTraining["strength"]
	m.Character.Stats.Dexterity.Training = req.StatTraining["dexterity"]
	m.Character.Stats.Perception.Training = req.StatTraining["perception"]
	m.Character.Stats.Vitality.Training = req.StatTraining["vitality"]
	m.Character.Stats.Willpower.Training = req.StatTraining["willpower"]
	m.Character.Stats.Charisma.Training = req.StatTraining["charisma"]
	m.Character.Equipment = mapToWorn(req.Equipment)
	m.Character.Items = nil
	for _, iid := range req.CarriedItems {
		if iid > 0 {
			m.Character.Items = append(m.Character.Items, items.New(iid))
		}
	}
	m.Character.Shop = nil
	for _, row := range req.Shop {
		if row.ItemId > 0 {
			m.Character.Shop = append(m.Character.Shop, characters.ShopItem{ItemId: row.ItemId, QuantityMax: row.QuantityMax, Price: row.Price})
		}
	}
	m.StatPool, m.Archetype, m.AutoAggro, m.AIProfile = req.StatPool, req.Archetype, req.AutoAggro, req.AIProfile
	m.LegacyHostile = false // canonical field only; a save migrates legacy files
	m.SpecialMoveChance, m.MovePreferences = req.SpecialMoveChance, req.MovePreferences
	m.ActivityLevel, m.MaxWander = req.ActivityLevel, req.MaxWander
	m.Routine, m.RoutineLinks, m.Hates, m.Groups = req.Routine, req.RoutineLinks, req.Hates, req.Groups
	m.SubmissionPolicy, m.SurrenderPolicy, m.PackFleeImmune = req.SubmissionPolicy, req.SurrenderPolicy, req.PackFleeImmune
	m.IdleCommands, m.CombatCommands, m.AngryCommands = req.IdleCommands, req.CombatCommands, req.AngryCommands
	m.ItemDropChance, m.LootPool = req.ItemDropChance, req.LootPool
	m.BuysGeneral, m.StockMultiplier, m.ShopCraftSupport = req.BuysGeneral, req.StockMultiplier, req.CraftSupport
	m.Crafter, m.CrafterSkill, m.CrafterRecipeIds, m.CrafterRestockMaterials = req.Crafter, req.CrafterSkill, req.CrafterRecipeIds, req.CrafterRestockMaterials
	m.ScheduleId, m.PatrolId = req.ScheduleId, req.PatrolId
	if req.Relationships != nil {
		m.Relationships = nil
		for _, r := range req.Relationships {
			m.Relationships = append(m.Relationships, mobs.RelationshipYAMLEntry{To: r.To, Type: r.Type, Subtype: r.Subtype})
		}
	}
	if req.KnowsFacts != nil {
		m.KnowsFacts = req.KnowsFacts
	}
	m.DefaultDisposition, m.FoldAnchorRoom, m.StorageChestRoom = req.DefaultDisposition, req.FoldAnchorRoom, req.StorageChestRoom
	m.ScriptTag, m.BehaviorArchetype = req.ScriptTag, req.BehaviorArchetype
	m.BuffIds, m.QuestFlags = req.BuffIds, req.QuestFlags
	m.SpawnMutations, m.MutationChance = req.SpawnMutations, req.MutationChance
	if req.LLMProfileJSON != "-" {
		m.LLMProfile = llmProfileFromJSON(req.LLMProfileJSON) // nil on ""
	}
	m.CarryCapacityOverride, m.HealthMaxOverride, m.StaminaMaxOverride = req.CarryCapacity, req.HealthMax, req.StaminaMax
	m.CorpseName, m.CorpseDescription = req.CorpseName, req.CorpseDescription
	m.HideEquipmentSlots, m.CharmImmune = req.HideEquipmentSlots, req.CharmImmune
	m.NonCombatant, m.PlayerAttackImmune = req.NonCombatant, req.PlayerAttackImmune
	return m
}

// buildMobUpdate: gmcp-layer checks (zone/craft-support/recipes — packages the
// mobs pkg can't import), then save (which runs ValidateMobSpec).
func buildMobUpdate(d mobDeps, req mobUpdateReq) BuildResult {
	base := d.load(req.MobId)
	if base == nil {
		return buildErr("mob %d not found", req.MobId)
	}
	if req.Name == "" {
		return buildErr("name is required")
	}
	if !d.zoneExists(req.Zone) {
		return buildErr("zone %q does not exist", req.Zone)
	}
	if req.CraftSupport != "" && !shops.IsValidCraftSupport(req.CraftSupport) {
		return buildErr("craft_support %q invalid; valid: %s", req.CraftSupport, strings.Join(shops.ValidCraftSupports, ", "))
	}
	// crafterRecipeIds validated here via crafting.GetAllRecipes (same accessor
	// the item reference scan uses) — reject unknown recipe ids by name.
	if err := d.save(reqToMob(base, req)); err != nil {
		return buildErr("could not save mob %d: %s", req.MobId, err.Error())
	}
	return BuildResult{Ok: true, MobId: req.MobId}
}

func buildMobCreate(d mobDeps, zone string) BuildResult {
	if !d.zoneExists(zone) {
		return buildErr("zone %q does not exist", zone)
	}
	id, err := d.create(zone)
	if err != nil {
		return buildErr("%s", err.Error())
	}
	return BuildResult{Ok: true, MobId: id}
}

// ---- enum providers ----
func collectMobEnums() mobEnums {
	e := mobEnums{
		Archetypes:         []string{"", "fighting", "casting"},
		AIProfiles:         []string{"", "default", "aggressive", "defensive", "grappler", "brawler", "tactical"},
		ScheduleIds:        mobs.AllScheduleIds(),
		PatrolIds:          mobs.AllPatrolIds(),
		CraftSupports:      shops.ValidCraftSupports,
		SubmissionPolicies: []string{"", "mercy", "subdue", "cripple", "lethal"},
		WornSlots:          wornSlotNames(),
		Species:            map[string]string{},
	}
	for _, s := range species.GetAllSpecies() {
		e.Species[fmt.Sprintf("%d", s.SpeciesId)] = s.Name
	}
	e.Zones = allZoneNames() // reuse/extract the zone list the room builder already sends
	e.BehaviorArchetypes = listBehaviorArchetypeFiles()
	seen := map[string]bool{}
	for _, m := range mobs.AllMobTemplates() {
		for _, g := range m.Groups {
			if !seen[g] {
				seen[g] = true
				e.Groups = append(e.Groups, g)
			}
		}
	}
	sort.Strings(e.Groups)
	return e
}

func listBehaviorArchetypeFiles() []string {
	dir := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `behaviors`, `/`, `archetypes`)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := []string{}
	for _, en := range entries {
		if n, ok := strings.CutSuffix(en.Name(), ".yaml"); ok {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// ---- senders ----
func sendMobList(uid int) {
	events.AddToQueue(GMCPOut{UserId: uid, Module: "Build.Mobs", Payload: buildMobList(realMobDeps())})
}
func sendMobDetail(uid int, d mobDetail) {
	events.AddToQueue(GMCPOut{UserId: uid, Module: "Build.Mob", Payload: d})
}
```

Also implement the small helpers referenced above **in the same file**:

```go
// wornToMap / mapToWorn / wornSlotNames — one authoritative slot table.
type wornSlotAccessor struct {
	name string
	get  func(w *characters.Worn) *items.Item
}

func wornSlotAccessors() []wornSlotAccessor {
	return []wornSlotAccessor{
		{"weapon", func(w *characters.Worn) *items.Item { return &w.Weapon }},
		{"offhand", func(w *characters.Worn) *items.Item { return &w.Offhand }},
		{"head", func(w *characters.Worn) *items.Item { return &w.Head }},
		{"neck", func(w *characters.Worn) *items.Item { return &w.Neck }},
		{"shoulders", func(w *characters.Worn) *items.Item { return &w.Shoulders }},
		{"body", func(w *characters.Worn) *items.Item { return &w.Body }},
		{"back", func(w *characters.Worn) *items.Item { return &w.Back }},
		{"belt", func(w *characters.Worn) *items.Item { return &w.Belt }},
		{"wrist1", func(w *characters.Worn) *items.Item { return &w.Wrist1 }},
		{"wrist2", func(w *characters.Worn) *items.Item { return &w.Wrist2 }},
		{"gloves", func(w *characters.Worn) *items.Item { return &w.Gloves }},
		{"ring", func(w *characters.Worn) *items.Item { return &w.Ring }},
		{"ring2", func(w *characters.Worn) *items.Item { return &w.Ring2 }},
		{"legs", func(w *characters.Worn) *items.Item { return &w.Legs }},
		{"feet", func(w *characters.Worn) *items.Item { return &w.Feet }},
		// implementer: append any further item-bearing slots present in worn.go
		// (tail, componentbag) — keep this the ONE slot table.
	}
}
func wornSlotNames() []string {
	out := []string{}
	for _, a := range wornSlotAccessors() {
		out = append(out, a.name)
	}
	return out
}
func wornToMap(w characters.Worn) map[string]int {
	out := map[string]int{}
	for _, a := range wornSlotAccessors() {
		if it := a.get(&w); it.ItemId > 0 {
			out[a.name] = it.ItemId
		}
	}
	return out
}
func mapToWorn(m map[string]int) characters.Worn {
	w := characters.Worn{}
	for _, a := range wornSlotAccessors() {
		if id := m[a.name]; id > 0 {
			*a.get(&w) = items.New(id)
		}
	}
	return w
}

// llmProfileToJSON / llmProfileFromJSON — raw JSON round-trip for the nested
// *llm.LLMProfile (form shows it as an editable JSON textarea; Task 6).
func llmProfileToJSON(p *llm.LLMProfile) string {
	if p == nil {
		return ""
	}
	b, err := json.Marshal(p)
	if err != nil {
		return ""
	}
	return string(b)
}
func llmProfileFromJSON(s string) *llm.LLMProfile {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var p llm.LLMProfile
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		return nil
	}
	return &p
}
```

> **Implementer notes:** (1) `items.New(itemId)` — verify the constructor name/signature (whatever `give`/spawn uses) and that a zero-arg-options call is right; (2) `rooms.GetZoneConfig` — verify; the room builder (`gmcp.Build.go`) already resolves zones — reuse its exact accessor + extract its zone list into `allZoneNames()` if not already shared; (3) `llm` import path is `github.com/GoMudEngine/GoMud/internal/llm` (Mob.LLMProfile field type); (4) if `buildMobUpdate`'s recipe validation needs `crafting`, import it — gmcp already imports it for the item scan; (5) add missing imports (`fmt`, `encoding/json`) as the compiler demands; (6) a malformed `LLMProfileJSON` silently nils the profile in `llmProfileFromJSON` — return an error instead and surface it via `buildErr` (adjust signature to `(profile, error)`).

- [ ] **Step 4: Run — PASS.** `go test ./modules/gmcp/ -run TestBuildMob`

- [ ] **Step 5: Route.** In `gmcp.Build.go` `handleBuildOp` add:

```go
	case `Build.Mob.List`:
		sendMobList(uid)
	case `Build.Mob.Get`:
		var req mobGetReq
		if json.Unmarshal(evt.Payload, &req) != nil { sendBuildResult(uid, buildErr("bad Build.Mob.Get payload")); break }
		if d, ok := buildMobGet(realMobDeps(), req.MobId); ok { sendMobDetail(uid, d) } else { sendBuildResult(uid, buildErr("mob %d not found", req.MobId)) }
	case `Build.Mob.Create`:
		var req mobCreateReq
		if json.Unmarshal(evt.Payload, &req) != nil { sendBuildResult(uid, buildErr("bad Build.Mob.Create payload")); break }
		res := buildMobCreate(realMobDeps(), req.Zone)
		sendBuildResult(uid, res)
		sendMobList(uid)
	case `Build.Mob.Update`:
		var req mobUpdateReq
		req.LLMProfileJSON = "-" // sentinel: absent field = leave profile untouched
		if json.Unmarshal(evt.Payload, &req) != nil { sendBuildResult(uid, buildErr("bad Build.Mob.Update payload")); break }
		sendBuildResult(uid, buildMobUpdate(realMobDeps(), req))
		sendMobList(uid)
	// Build.Mob.Delete — Task 3; Build.Mob.Spawn — Task 4
```

Then in `gmcp.go` add to the deferred `Build.*` dispatch list: `Build.Mob.List`, `Build.Mob.Get`, `Build.Mob.Create`, `Build.Mob.Update`, `Build.Mob.Delete`, `Build.Mob.Spawn`.

- [ ] **Step 6: Build + tests + gofmt.** `go build ./... && go test ./modules/gmcp/ && gofmt -l modules/gmcp/gmcp.Mob.go`

- [ ] **Step 7: Commit.**
```bash
git add modules/gmcp/gmcp.Mob.go modules/gmcp/gmcp.Mob_test.go modules/gmcp/gmcp.Build.go modules/gmcp/gmcp.go
git commit -m "feat(gmcp): Build.Mob.* list/get/create/update (admin-gated)

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

## Task 3: Reference scan + `Build.Mob.Delete`

**Files:** Modify `modules/gmcp/gmcp.Mob.go`, `modules/gmcp/gmcp.Mob_test.go`, `gmcp.Build.go` (route + `MobRefs`).

- [ ] **Step 1: Failing tests.** (a) `buildMobDelete` blocks when refs exist / deletes when clean; (b) `scanMobReferencesWith` finds a room-spawn ref and a relationship ref via injected iterators.

```go
func TestBuildMobDelete_BlocksWhenReferenced(t *testing.T) {
	w := newFakeMobWorld()
	m := &mobs.Mob{MobId: 90010, Zone: "Testzone"}
	m.Character.Name = "Referenced Mob"
	w.specs[90010] = m
	d := w.deps()
	d.references = func(id int) []mobRef { return []mobRef{{Kind: "room-spawn", Id: "room 101"}} }
	res := buildMobDelete(d, 90010)
	if res.Ok {
		t.Fatal("delete should be blocked when referenced")
	}
	if len(w.deleted) != 0 {
		t.Error("nothing should be deleted when blocked")
	}
	if len(res.MobRefs) != 1 || res.MobRefs[0].Id != "room 101" {
		t.Errorf("blocked result must carry references, got %+v", res.MobRefs)
	}
}

func TestBuildMobDelete_DeletesWhenClean(t *testing.T) {
	w := newFakeMobWorld()
	m := &mobs.Mob{MobId: 90011, Zone: "Testzone"}
	m.Character.Name = "Unused Mob"
	w.specs[90011] = m
	res := buildMobDelete(w.deps(), 90011)
	if !res.Ok {
		t.Fatalf("clean delete should succeed, got %+v", res)
	}
	if len(w.deleted) != 1 || w.deleted[0] != 90011 {
		t.Errorf("expected delete of 90011, got %+v", w.deleted)
	}
}

func TestScanMobReferences_FindsSpawnAndRelationship(t *testing.T) {
	refs := scanMobReferencesWith(9538, mobRefIterators{
		roomSpawns: func(yield func(roomId int, mobIds []int)) {
			yield(101, []int{9538})
			yield(102, []int{1})
		},
		mobRelationships: func(yield func(fromMobId int, name string, toIds []int)) {
			yield(9600, "Gossip Gert", []int{9538})
		},
	})
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %+v", refs)
	}
	if refs[0].Kind != "room-spawn" || refs[1].Kind != "relationship" {
		t.Errorf("wrong kinds: %+v", refs)
	}
}
```

- [ ] **Step 2: Run — FAIL.**

- [ ] **Step 3: Implement.** Add to `BuildResult` (in `gmcp.Build.go`): `MobRefs []mobRef \`json:"mobRefs,omitempty"\``. In `gmcp.Mob.go`:

```go
func buildMobDelete(d mobDeps, mobId int) BuildResult {
	if d.load(mobId) == nil {
		return buildErr("mob %d not found", mobId)
	}
	if refs := d.references(mobId); len(refs) > 0 {
		return BuildResult{Error: "mob is still referenced — remove these first", MobRefs: refs}
	}
	if err := d.del(mobId); err != nil {
		return buildErr("could not delete mob %d: %s", mobId, err.Error())
	}
	return BuildResult{Ok: true, MobId: mobId}
}

type mobRefIterators struct {
	roomSpawns       func(yield func(roomId int, mobIds []int))
	mobRelationships func(yield func(fromMobId int, name string, toIds []int))
	quests           func(yield func(questToken string, mobIds []int))
	dialogueExists   func(zone string, mobId int) bool
	convPairs        func(yield func(pairFile string, mobIds []int))
	mercShops        func(yield func(sellerMobId int, name string, mobIds []int))
}

func scanMobReferencesWith(mobId int, it mobRefIterators) []mobRef {
	out := []mobRef{}
	if it.roomSpawns != nil {
		it.roomSpawns(func(roomId int, ids []int) {
			if containsInt(ids, mobId) {
				out = append(out, mobRef{Kind: "room-spawn", Id: fmt.Sprintf("room %d", roomId)})
			}
		})
	}
	if it.mobRelationships != nil {
		it.mobRelationships(func(fromId int, name string, ids []int) {
			if containsInt(ids, mobId) {
				out = append(out, mobRef{Kind: "relationship", Id: fmt.Sprintf("mob %d (%s)", fromId, name)})
			}
		})
	}
	if it.quests != nil {
		it.quests(func(token string, ids []int) {
			if containsInt(ids, mobId) {
				out = append(out, mobRef{Kind: "quest", Id: "quest " + token})
			}
		})
	}
	if it.convPairs != nil {
		it.convPairs(func(file string, ids []int) {
			if containsInt(ids, mobId) {
				out = append(out, mobRef{Kind: "conversation", Id: file})
			}
		})
	}
	if it.mercShops != nil {
		it.mercShops(func(sellerId int, name string, ids []int) {
			if containsInt(ids, mobId) {
				out = append(out, mobRef{Kind: "merc-shop", Id: fmt.Sprintf("mob %d (%s) sells it", sellerId, name)})
			}
		})
	}
	return out
}

// scanMobReferences wires the real world. The dialogue check is separate
// (needs the mob's own zone), appended after the generic iterators.
func scanMobReferences(mobId int) []mobRef {
	m := mobs.GetMobSpec(mobs.MobId(mobId))
	refs := scanMobReferencesWith(mobId, mobRefIterators{
		roomSpawns: func(yield func(int, []int)) {
			for _, roomId := range rooms.GetAllRoomIds() {
				r := rooms.LoadRoom(roomId)
				if r == nil {
					continue
				}
				ids := []int{}
				for _, si := range r.SpawnInfo {
					if si.MobId > 0 {
						ids = append(ids, si.MobId)
					}
				}
				if len(ids) > 0 {
					yield(roomId, ids)
				}
			}
		},
		mobRelationships: func(yield func(int, string, []int)) {
			for _, other := range mobs.AllMobTemplates() {
				if int(other.MobId) == mobId {
					continue
				}
				ids := []int{}
				for _, rel := range other.Relationships {
					ids = append(ids, rel.To)
				}
				if len(ids) > 0 {
					yield(int(other.MobId), other.Character.Name, ids)
				}
			}
		},
		quests: func(yield func(string, []int)) {
			// Implementer: iterate quests.GetAllQuests(); collect mob ids from
			// step triggers (mob_death targets etc.) — verify the Quest struct
			// field names via codegraph (the item scan's quest iterator is the
			// template). If quest mob refs are stored as strings, parse ints.
		},
		mercShops: func(yield func(int, string, []int)) {
			for _, other := range mobs.AllMobTemplates() {
				ids := []int{}
				for _, si := range other.Character.Shop {
					if si.MobId > 0 {
						ids = append(ids, si.MobId)
					}
				}
				if len(ids) > 0 {
					yield(int(other.MobId), other.Character.Name, ids)
				}
			}
		},
		convPairs: func(yield func(string, []int)) {
			// Implementer: conversations/pairs/<lower>_<higher>.yaml — parse the
			// two ids from each filename in the pairs dir; skip if dir absent.
		},
	})
	// dialogue file keyed by this mob's id in its zone
	if m != nil {
		zone := mobs.ZoneNameSanitize(m.Zone)
		p := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `dialogue`, `/`, zone, `/`, fmt.Sprintf("%d.yaml", mobId))
		if _, err := os.Stat(p); err == nil {
			refs = append(refs, mobRef{Kind: "dialogue", Id: fmt.Sprintf("dialogue/%s/%d.yaml", zone, mobId)})
		}
	}
	return refs
}
```

(`containsInt` already exists from the item scan.)

- [ ] **Step 4: Run — PASS.** `go test ./modules/gmcp/ -run 'TestBuildMobDelete|TestScanMobReferences'`

- [ ] **Step 5: Route delete** in `handleBuildOp`:
```go
	case `Build.Mob.Delete`:
		var req mobDeleteReq
		if json.Unmarshal(evt.Payload, &req) != nil { sendBuildResult(uid, buildErr("bad Build.Mob.Delete payload")); break }
		res := buildMobDelete(realMobDeps(), req.MobId)
		sendBuildResult(uid, res)
		if res.Ok { sendMobList(uid) }
```

- [ ] **Step 6: Build + tests + gofmt; commit.**
```bash
git add modules/gmcp/
git commit -m "feat(gmcp): Build.Mob.Delete with world-wide reference scan

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

## Task 4: `Build.Mob.Spawn` — test-spawn into a chosen room

**Files:** Modify `modules/gmcp/gmcp.Mob.go`, `modules/gmcp/gmcp.Mob_test.go`, `gmcp.Build.go` (route).

- [ ] **Step 1: Failing test.**

```go
func TestBuildMobSpawn_ValidatesAndSpawns(t *testing.T) {
	w := newFakeMobWorld()
	m := &mobs.Mob{MobId: 90020, Zone: "Testzone"}
	m.Character.Name = "Spawnling"
	w.specs[90020] = m
	d := w.deps()
	spawned := []int{}
	d.spawn = func(mobId, roomId int) (string, error) {
		spawned = append(spawned, roomId)
		return "Spawnling", nil
	}
	res := buildMobSpawn(d, mobSpawnReq{MobId: 90020, RoomId: 101})
	if !res.Ok || res.MobId != 90020 {
		t.Fatalf("spawn should succeed, got %+v", res)
	}
	if len(spawned) != 1 || spawned[0] != 101 {
		t.Errorf("expected spawn in room 101, got %+v", spawned)
	}
	if res := buildMobSpawn(d, mobSpawnReq{MobId: 424242, RoomId: 101}); res.Ok {
		t.Error("unknown mob must be rejected")
	}
}
```

- [ ] **Step 2: Run — FAIL.**

- [ ] **Step 3: Implement.**

```go
func buildMobSpawn(d mobDeps, req mobSpawnReq) BuildResult {
	if d.load(req.MobId) == nil {
		return buildErr("mob %d not found", req.MobId)
	}
	name, err := d.spawn(req.MobId, req.RoomId)
	if err != nil {
		return buildErr("could not spawn: %s", err.Error())
	}
	return BuildResult{Ok: true, MobId: req.MobId, Message: fmt.Sprintf("%s spawned in room %d", name, req.RoomId)}
}

// spawnMobInRoom mirrors the in-game `mob spawn` admin command
// (admin.mob.go:200): NewMobById + room.AddMob + a room announce. Runs on
// MainWorker via handleBuildOp, so world access is safe.
func spawnMobInRoom(mobId, roomId int) (string, error) {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return "", fmt.Errorf("room %d not found", roomId)
	}
	mob := mobs.NewMobById(mobs.MobId(mobId), roomId)
	if mob == nil {
		return "", fmt.Errorf("mob %d could not be instantiated", mobId)
	}
	room.AddMob(mob.InstanceId)
	room.SendTextVisual(messaging.CategoryMobEmote,
		fmt.Sprintf(`<ansi fg="mobname">%s</ansi> appears in the air and falls to the ground.`, mob.Character.Name), 0)
	return mob.Character.Name, nil
}
```

If `BuildResult` has no `Message` field, add it: `Message string \`json:"message,omitempty"\``. Verify `SendTextVisual`'s exclude-userId parameter accepts 0 (no exclusion) — `admin.mob.go:206` passes a real user id; 0 is the "nobody to exclude" convention, confirm against its signature.

- [ ] **Step 4: Run — PASS.** `go test ./modules/gmcp/ -run TestBuildMobSpawn`

- [ ] **Step 5: Route** in `handleBuildOp`:
```go
	case `Build.Mob.Spawn`:
		var req mobSpawnReq
		if json.Unmarshal(evt.Payload, &req) != nil { sendBuildResult(uid, buildErr("bad Build.Mob.Spawn payload")); break }
		sendBuildResult(uid, buildMobSpawn(realMobDeps(), req))
```

- [ ] **Step 6: Build + tests + gofmt; commit.**
```bash
git add modules/gmcp/
git commit -m "feat(gmcp): Build.Mob.Spawn — web test-spawn into a chosen room

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

## Task 5: Mobs-mode UI — tab + mob list

**Files:** Create `_datafiles/html/public/static/js/mobs.js`; Modify `build.html`.

- [ ] **Step 1: Extend the mode toggle** in `build.html` to `Rooms | Items | Mobs` (`<button id="tb-mode-mobs">Mobs</button>`), add `<div id="moblist" style="display:none;flex:0 0 320px;overflow-y:auto;background:var(--leather2);border-right:2px solid var(--tooled);"></div>`, and load `mobs.js` after `items.js`:
```html
<script src="{{ .CONFIG.FilePaths.WebCDNLocation }}/static/js/mobs.js"></script>
```

- [ ] **Step 2: Mode switching.** Mobs mode hides `#canvas`/`#itemlist` (+ room toolbar controls), shows `#moblist`, sends `Build.Mob.List`, clears the inspector. Keep the existing Rooms/Items behavior symmetrical (each mode hides the other two lists). Track via `window.Builder.mode = "mobs"`.

- [ ] **Step 3: `mobs.js` list rendering.** `MobsPanel` object mirroring `ItemsPanel`: search input + **zone** `<select>` (distinct zones from rows) + scrollable rows (`#<id> · <name> · <zone> · pool <statPool>` with small badges for `non-combatant` / `schedule` / `shop`), filtered live. Row click → `sendGMCP("Build.Mob.Get",{mobId})`. `+ New Mob` button → zone picker (from the rows' zones) → `sendGMCP("Build.Mob.Create",{zone})`.

- [ ] **Step 4: Route GMCP** in `build.html` `onGMCP`: `if (ns==="Build.Mobs") window.Builder.MobsPanel.render(obj);` (detail in Task 6).

- [ ] **Step 5: Verify in browser.** Mobs tab loads the list (641 templates), filters by search/zone, `+ New Mob` creates a stub that appears in the list and on disk under the zone folder.

- [ ] **Step 6: Commit.**
```bash
git add _datafiles/html/public/static/js/mobs.js _datafiles/html/public/build.html
git commit -m "feat(build): Mobs tab + searchable mob list

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

## Task 6: Sectioned mob form + Save / Create-load / Delete

**Files:** Modify `mobs.js`, `build.html`.

- [ ] **Step 1: Render the form on `Build.Mob`.** Route `if (ns==="Build.Mob") window.Builder.MobsPanel.renderForm(obj);`. `renderForm(detail)` builds into `#inspector` (reuse `ce()` + the items form's field/hint/dirty helpers) with sections per the spec:
  - **Identity:** name, description (textarea), zone (dropdown `detail.enums.zones`), species (dropdown `detail.enums.species`), adjectives (chips), groups (chips + datalist `detail.enums.groups`).
  - **Stats & Combat:** statPool, per-stat training row (6 numeric inputs from `statTraining`), archetype dropdown, autoAggro checkbox, aiProfile dropdown, specialMoveChance, activityLevel, maxWander, routine, routineLinks (chips), hates (chips), submissionPolicy dropdown, surrenderPolicy text (hint: `never | always | "auto-tap-below N"`), packFleeImmune.
  - **Equipment:** one row per `detail.enums.wornSlots` — item picker (searchable input resolving against the item list already fetched for the Items tab; fetch `Build.Item.List` on first open if absent) + carried-items picker list.
  - **Flavor:** idle/combat/angry command list editors (textarea-rows with add/remove/reorder; preserve blank rows — they're real "do nothing" entries).
  - **Loot:** itemDropChance, lootPool item picker rows, gold.
  - **Advanced (collapsible):**
    - *Commerce:* shop rows (item picker / quantityMax / price), buysGeneral, stockMultiplier, craftSupport dropdown, crafter + crafterSkill + crafterRecipeIds (chips) + crafterRestockMaterials (item picker rows).
    - *Aliveness:* scheduleId dropdown (`detail.enums.scheduleIds`), patrolId dropdown, relationships rows (to-mob numeric + type + subtype), knowsFacts (chips), defaultDisposition, foldAnchorRoom, storageChestRoom.
    - *Hooks:* scriptTag, behaviorArchetype dropdown, buffIds (numeric chips), questFlags (chips), spawnMutations (chips), mutationChance, **LLM profile** as a JSON textarea (value `detail.llmProfileJson`; hint: "advanced — raw JSON; leave empty for none").
    - *Overrides & flags:* carryCapacity, healthMax, staminaMax, corpseName, corpseDescription, and the checkboxes (hideEquipmentSlots, charmImmune, nonCombatant, playerAttackImmune).

- [ ] **Step 2: Save.** **Save** button gathers everything into the `mobUpdateReq` shape → `Build.Mob.Update`. On `Build.Result{ok}` clear dirty + toast; on error show the message inline (it names the field + valid values). On a result with `mobRefs`, ignore here (delete-only).

- [ ] **Step 3: Create → auto-load.** After `Build.Mob.Create` → `Build.Result{ok,mobId}`, auto-send `Build.Mob.Get{mobId}` so the stub opens ready to edit.

- [ ] **Step 4: Delete + reference block.** **Delete mob** button (confirm) → `Build.Mob.Delete{mobId}`. On `{ok}` clear form + toast. On blocked (`obj.mobRefs`), render "Still referenced: room 101 spawn, mob 9600 (Gossip Gert) relationship, dialogue/…" and do NOT clear.

- [ ] **Step 5: Verify in browser.** Edit an existing mob (e.g. change idle emotes + statpool) → YAML updates under the zone folder. Rename → file relocates. Create a stub → edit into a small combat mob (species, statpool, autoAggro, loot) → Save. Delete blocked on a room-spawned mob (refs listed); clean delete on the stub.

- [ ] **Step 6: Commit.**
```bash
git add _datafiles/html/public/static/js/mobs.js _datafiles/html/public/build.html
git commit -m "feat(build): sectioned mob form + save/create/delete

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

## Task 7: Test-spawn control

**Files:** Modify `mobs.js`, `build.html` (if any shared toast/route bits are missing).

- [ ] **Step 1: Spawn UI.** At the top of the mob form add a "Test spawn" row: zone dropdown (`detail.enums.zones`) → room picker. Room picker = numeric room-id input + a "check" that sends the existing `Build.Room.Get {roomId}` and shows the room's title next to the input (confirming the target before enabling **Spawn**). Spawn button → `Build.Mob.Spawn {mobId, roomId}`.

- [ ] **Step 2: Result handling.** On `Build.Result{ok, message}` toast the message ("Spawnling spawned in room 101"); on error toast the error. Disable the button for 2s after a click (no accidental double-spawn).

- [ ] **Step 3: Verify in browser + in-game.** Spawn a mob into a room you can reach; confirm the toast. Then (or via telnet) walk to the room and confirm the mob is present, idles, and is killable. Confirm spawning into a bogus room id errors cleanly.

- [ ] **Step 4: Commit.**
```bash
git add _datafiles/html/public/static/js/mobs.js _datafiles/html/public/build.html
git commit -m "feat(build): mob test-spawn into a chosen zone/room

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>"
```

---

## Task 8: End-to-end + gate (REQUIRED)

**Files:** none (verification).

- [ ] **Step 1: Full build + tests.** `go build ./... && go test ./internal/mobs/ ./modules/gmcp/`
- [ ] **Step 2: gofmt + vet.** `gofmt -l internal/mobs/save.go modules/gmcp/gmcp.Mob.go` (empty) and `go vet ./internal/mobs/ ./modules/gmcp/`.
- [ ] **Step 3: Boot-clean.** Wipe instance saves (`mobs.instances/` + `rooms.instances/` — NOT `shops/`/`guilds/`); boot; confirm `mobs.LoadDataFiles() loadedCount=641+` (plus any stubs), schedule cross-check passes, no panics. **Then**: create a stub via the web editor, leave it UNEDITED, reboot — the stub must load clean (the boot-safe-stub guarantee).
- [ ] **Step 4: Headless end-to-end.** Drive the WS→GMCP flow exactly as 1b/2 were verified: login bridge → `Build.Mob.List` → `Create` → `Get` → `Update` (rename + schedule + shop rows) → `Spawn` into a known room → `Delete` (blocked case + clean case). Assert each response payload.
- [ ] **Step 5: In-game half.** Via telnet/harness: confirm the test-spawned mob exists in the target room, runs its idle commands, and is killable; confirm an edited template's changes appear on a FRESH spawn (remember: instance saves shadow templates — wipe first).
- [ ] **Step 6: REQUIRED adversarial pass + report.** Read every builder interaction as a confused human (bad inputs: dangling schedule id typed via… it can't be typed — verify the dropdowns actually prevent it; negative statpool; empty name; unknown zone). Fix what it surfaces, re-run. Summarize findings. Do not claim done on boot-clean alone. **User owes the browser visual/UX eyeball before prod** (same as 1b/2).

---

## Self-Review notes
- **Spec coverage:** §1 writer seam (Canonicalize/Validate/Save/Delete/Create + churn note)→T1; §2 GMCP protocol + enums + ranges→T2 (ranges: statpool/gold/activity hints can reuse the items ranges mechanism — if `itemRangesForType` isn't reusable as-is, ship without numeric ranges in v1 and note it; the enums payload is the load-bearing guardrail); §3 reference scan→T3; test-spawn (spec §2 table + §4 control)→T4+T7; §4 UI sections incl. equipment/stat-training/level→T5+T6; §5 guardrails (dropdown-fed refs, legacy hostile never written — `reqToMob` clears `LegacyHostile`)→T2/T6; §6 testing→per-task tests + T8.
- **Known deviations from spec:** (1) numeric **range hints** are best-effort v1 (see above) — the boot-brick prevention never depended on them; (2) `Character.Level` — the YAML `level:` key's Go field wasn't located during planning; implementer: find it via codegraph (`level` tag in characters pkg) and add it to `mobUpdateReq`/`mobToReq`/`reqToMob`/form Identity section — if it turns out unused for mobs, drop it from the form and note that instead; (3) LLM profile is a raw-JSON textarea rather than a field-by-field sub-form — it round-trips safely and stays editable; revisit if it sees real use.
- **Type consistency:** `mobUpdateReq` defined once (T2), reused as Save payload + embedded in `mobDetail`; `mobDeps`/`mobRef`/`mobRefIterators` defined T2/T3 and used by routing T2-T4; `BuildResult.MobId` (T2), `.MobRefs` (T3), `.Message` (T4) consumed by the T6/T7 UI; `wornSlotAccessors` is the single slot table shared by `wornToMap`/`mapToWorn`/`wornSlotNames` and mirrored by T1's `equippedItemIds` (implementer: keep both lists in sync with `worn.go` — or export the gmcp table into a shared helper if that's cleaner at implementation time).
- **Risk notes:** (1) `Filename()` reads `mobNameCache` — the rename-relocation ordering in `SaveMobSpec` (old path BEFORE cache update) is the key correctness point; T1's rename test pins it. (2) Import-cycle split: mobs-pkg validation vs gmcp-layer validation (zone/craft-support/recipes) is deliberate — don't "clean it up" into one place. (3) `reqToMob` starts from the loaded template so unexposed runtime fields survive; slices the form DOES carry are replaced wholesale (idle commands etc.), and `Relationships`/`KnowsFacts` use nil-means-untouched. (4) The quest + conversation iterators in `scanMobReferences` need field-name verification at implementation (stubs marked). (5) Test-spawned mobs are ordinary transient instances — no cleanup path needed beyond killing them.
