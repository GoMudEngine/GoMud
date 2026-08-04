# Zone Lifecycle & Config Editor — Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Delete and configure zones from the `/build` admin page, with a guarded delete that refuses unless the zone is empty, and zone-config validation that makes it impossible to put a `non_cartesian` zone on the overworld plane.

**Architecture:** New `Build.Zone.List/Get/Update/Delete` GMCP verbs behind a `zoneDeps` seam (mirroring the existing `itemDeps`/`mobDeps` pattern) so handlers unit-test against a fake world with no filesystem. Core delete/scan logic lands in `internal/rooms` as injectable functions. All mutation routes through `GMCPBuildOp` to MainWorker.

**Tech Stack:** Go, `gopkg.in/yaml.v2` via `internal/fileloader`, testify `assert` in `internal/rooms`, plain `testing` in `modules/gmcp`, vanilla JS panel modules in `_datafiles/html/public/static/js/`.

**Spec:** `docs/superpowers/specs/2026-07-25-zone-lifecycle-config-design.md`

**Phase 2 (rename) is NOT in this plan.**

---

## Context the implementer needs

**Zone create already exists and is correct.** `buildZoneCreate`
(`modules/gmcp/gmcp.Build.go:322`) already assigns a fresh plane via
`rooms.NextFreeAuthoredPlane()`. Do not rebuild it. Task 1 adds its one
missing guard.

**A zone owns ten directories**, all named by the *sanitized* zone name
(`rooms.ZoneToFolder`):

```
rooms/<z>/  rooms.instances/<z>/  mobs/<z>/  mobs.instances/<z>/
dialogue/<z>/  behaviors/<z>/  schedules/<z>/  caravans/<z>/
foragers/<z>/  shops/<z>/
```

**`internal/rooms` must NOT import `internal/mapper`** — mapper imports rooms.
Cache invalidation after a delete happens in the GMCP layer (Task 7), not in
`DeleteZone`.

**Existing signatures you will use** (all verified present):

```go
rooms.GetAllZoneNames() []string                      // roommanager.go:240
rooms.GetZoneConfig(zone string) *ZoneConfig          // roommanager.go:602
rooms.SaveZoneConfig(cfg *ZoneConfig) error           // save_and_load.go:411
rooms.GetZoneRoot(zone string) (int, error)           // roommanager.go:593
rooms.ZoneNameSanitize(zone string) string            // roommanager.go:620
rooms.ZoneToFolder(zone string) string                // roommanager.go:630
rooms.ValidateZoneName(zone string) error             // roommanager.go:636
rooms.IsZoneNonCartesian(zone string) bool            // roommanager.go:673
rooms.NextFreeAuthoredPlane() int                     // planes.go:52
rooms.GetAllRoomIds() []int
rooms.LoadRoomTemplate(roomId int) *Room
mapper.ClearCache()                                   // mapper.go:110
```

`ValidateZoneName` returns `nil` for the empty string — callers must check
emptiness separately.

---

## File Structure

| File | Responsibility |
|------|----------------|
| `internal/rooms/zone_lifecycle.go` **(create)** | `ZoneFolderCollision`, `ZoneBlocker`, `ZoneDeletionBlockersWith`/`ZoneDeletionBlockers`, `DeleteZone` |
| `internal/rooms/zone_lifecycle_test.go` **(create)** | Unit tests for the above |
| `internal/rooms/roommanager.go:728` **(modify)** | `CreateZone` calls the collision guard |
| `modules/gmcp/gmcp.Zone.go` **(create)** | `zoneDeps`, payload types, `buildZoneList/Get/Update/Delete`, `sendZoneList`/`sendZoneDetail` |
| `modules/gmcp/gmcp.Zone_test.go` **(create)** | Handler tests against a fake world |
| `modules/gmcp/gmcp.Build.go` **(modify)** | Dispatch the four new verbs |
| `modules/gmcp/gmcp.go:486` **(modify)** | Route + admin-gate the new verbs |
| `_datafiles/html/public/static/js/zones.js` **(create)** | `Builder.ZonesPanel` |
| `_datafiles/html/public/build.html` **(modify)** | Zones mode button + routing |

---

### Task 1: Folder-collision guard on zone creation

Two display names can sanitize onto one folder (`ZoneNameSanitize` only
lowercases and swaps spaces for underscores), so `Amber_Valley` beside
`Amber Valley` hits `os.Mkdir` on a live zone's folder.

**Files:**
- Create: `internal/rooms/zone_lifecycle.go`
- Create: `internal/rooms/zone_lifecycle_test.go`
- Modify: `internal/rooms/roommanager.go` (inside `CreateZone`)

- [ ] **Step 1: Write the failing test**

```go
package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZoneFolderCollision(t *testing.T) {
	existing := []string{"Amber Valley", "Stillwater"}

	// Same folder (amber_valley), different display name -> collision.
	assert.Equal(t, "Amber Valley", ZoneFolderCollision("Amber_Valley", existing))
	assert.Equal(t, "Amber Valley", ZoneFolderCollision("amber valley", existing))

	// Genuinely new zone -> no collision.
	assert.Equal(t, "", ZoneFolderCollision("Thornwall", existing))

	// A name identical to an existing zone still reports it; CreateZone's
	// own duplicate check runs first, but this must not silently pass.
	assert.Equal(t, "Stillwater", ZoneFolderCollision("Stillwater", existing))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/rooms/ -run TestZoneFolderCollision -v`
Expected: FAIL — `undefined: ZoneFolderCollision`

- [ ] **Step 3: Write minimal implementation**

Create `internal/rooms/zone_lifecycle.go`:

```go
package rooms

// ZoneFolderCollision reports the first zone in existing whose sanitized
// folder name matches newName's, or "" when the folder is free.
//
// ZoneNameSanitize only lowercases and converts spaces to underscores, so
// "Amber Valley", "amber valley" and "Amber_Valley" all map to the folder
// amber_valley. CreateZone's duplicate check compares DISPLAY names and so
// misses this, reaching os.Mkdir on a live zone's folder.
func ZoneFolderCollision(newName string, existing []string) string {
	folder := ZoneNameSanitize(newName)
	if folder == "" {
		return ""
	}
	for _, z := range existing {
		if ZoneNameSanitize(z) == folder {
			return z
		}
	}
	return ""
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/rooms/ -run TestZoneFolderCollision -v`
Expected: PASS

- [ ] **Step 5: Wire it into CreateZone**

In `internal/rooms/roommanager.go`, inside `CreateZone`, directly after the
existing duplicate-name check:

```go
	if zoneInfo, ok := roomManager.zones[zoneName]; ok {
		return zoneInfo.RoomId, errors.New("zone already exists")
	}

	// Two display names can sanitize onto one folder; without this the
	// os.Mkdir below lands on a live zone's directory.
	if clash := ZoneFolderCollision(zoneName, GetAllZoneNames()); clash != "" {
		return 0, fmt.Errorf("zone folder %q is already used by zone %q", ZoneNameSanitize(zoneName), clash)
	}
```

- [ ] **Step 6: Verify the package still builds and tests pass**

Run: `go build ./... && go test ./internal/rooms/`
Expected: build clean, PASS

- [ ] **Step 7: Commit**

```bash
git add internal/rooms/zone_lifecycle.go internal/rooms/zone_lifecycle_test.go internal/rooms/roommanager.go
git commit -m "fix(rooms): reject zone names that collide on the sanitized folder"
```

---

### Task 2: Deletion blocker scan

**Files:**
- Modify: `internal/rooms/zone_lifecycle.go`
- Modify: `internal/rooms/zone_lifecycle_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/rooms/zone_lifecycle_test.go`:

```go
func TestZoneDeletionBlockers_ReportsEachKind(t *testing.T) {
	src := zoneBlockerSources{
		roomIdsInZone:  func(z string) []int { return []int{100, 101, 102} },
		zoneRootRoomId: func(z string) int { return 100 },
		contentFiles:   func(z string) []string { return []string{"mobs/testzone/5-guard.yaml"} },
		inboundExits: func(z string) []string {
			return []string{"room 900 (Other Zone) east"}
		},
		playersInZone: func(z string) []string { return []string{"Meirok"} },
	}

	got := ZoneDeletionBlockersWith("Testzone", src)

	kinds := map[string]int{}
	for _, b := range got {
		kinds[b.Kind]++
	}
	// 101 and 102 are non-root rooms; 100 is the root and is NOT a blocker.
	assert.Equal(t, 2, kinds["room"], "root room must not be reported")
	assert.Equal(t, 1, kinds["content"])
	assert.Equal(t, 1, kinds["inbound-exit"])
	assert.Equal(t, 1, kinds["player"])
}

func TestZoneDeletionBlockers_CleanZoneIsEmpty(t *testing.T) {
	src := zoneBlockerSources{
		roomIdsInZone:  func(z string) []int { return []int{100} }, // root only
		zoneRootRoomId: func(z string) int { return 100 },
		contentFiles:   func(z string) []string { return nil },
		inboundExits:   func(z string) []string { return nil },
		playersInZone:  func(z string) []string { return nil },
	}
	assert.Empty(t, ZoneDeletionBlockersWith("Testzone", src))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/rooms/ -run TestZoneDeletionBlockers -v`
Expected: FAIL — `undefined: zoneBlockerSources`

- [ ] **Step 3: Write minimal implementation**

Append to `internal/rooms/zone_lifecycle.go`:

```go
// ZoneBlocker is one reason a zone cannot be deleted.
type ZoneBlocker struct {
	Kind string `json:"kind"` // room | content | inbound-exit | player
	Id   string `json:"id"`   // human-readable identifier
}

// zoneBlockerSources injects every world lookup the scan needs so the policy
// is testable without a filesystem or a loaded world.
type zoneBlockerSources struct {
	roomIdsInZone  func(zone string) []int
	zoneRootRoomId func(zone string) int
	contentFiles   func(zone string) []string
	inboundExits   func(zone string) []string
	playersInZone  func(zone string) []string
}

// ZoneDeletionBlockersWith applies the deletion policy: a zone may be deleted
// only when it holds nothing but its root room, owns no authored content, has
// no exits pointing into it from other zones, and has no players inside.
//
// Shops and the two .instances/ trees are deliberately NOT blockers — they are
// regenerable living state, not authored work.
func ZoneDeletionBlockersWith(zone string, src zoneBlockerSources) []ZoneBlocker {
	out := []ZoneBlocker{}

	root := src.zoneRootRoomId(zone)
	for _, id := range src.roomIdsInZone(zone) {
		if id == root {
			continue
		}
		out = append(out, ZoneBlocker{Kind: "room", Id: fmt.Sprintf("room %d", id)})
	}
	for _, f := range src.contentFiles(zone) {
		out = append(out, ZoneBlocker{Kind: "content", Id: f})
	}
	for _, e := range src.inboundExits(zone) {
		out = append(out, ZoneBlocker{Kind: "inbound-exit", Id: e})
	}
	for _, p := range src.playersInZone(zone) {
		out = append(out, ZoneBlocker{Kind: "player", Id: p})
	}
	return out
}
```

Add `"fmt"` to the file's imports.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/rooms/ -run TestZoneDeletionBlockers -v`
Expected: PASS (both tests)

- [ ] **Step 5: Commit**

```bash
git add internal/rooms/zone_lifecycle.go internal/rooms/zone_lifecycle_test.go
git commit -m "feat(rooms): zone deletion blocker policy"
```

---

### Task 3: Real blocker wiring + `DeleteZone`

**Files:**
- Modify: `internal/rooms/zone_lifecycle.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/rooms/zone_lifecycle_test.go`:

```go
func TestZoneContentDirs_CoversAllAuthoredTrees(t *testing.T) {
	dirs := zoneContentDirs()
	assert.ElementsMatch(t,
		[]string{"mobs", "dialogue", "behaviors", "schedules", "caravans", "foragers"},
		dirs,
		"authored content trees scanned for delete blockers")
}

func TestZoneAllDirs_CoversAllTenTrees(t *testing.T) {
	assert.Len(t, zoneAllDirs(), 10, "a zone owns ten directories")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/rooms/ -run "TestZoneContentDirs|TestZoneAllDirs" -v`
Expected: FAIL — `undefined: zoneContentDirs`

- [ ] **Step 3: Write minimal implementation**

Append to `internal/rooms/zone_lifecycle.go`:

```go
// zoneContentDirs are the trees holding AUTHORED zone content. Presence of any
// file here blocks deletion.
func zoneContentDirs() []string {
	return []string{"mobs", "dialogue", "behaviors", "schedules", "caravans", "foragers"}
}

// zoneAllDirs is every tree a zone owns, including regenerable living state
// (shops + the two .instances trees). Deletion removes all of them.
func zoneAllDirs() []string {
	return append(zoneContentDirs(), "rooms", "rooms.instances", "mobs.instances", "shops")
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/rooms/ -run "TestZoneContentDirs|TestZoneAllDirs" -v`
Expected: PASS

- [ ] **Step 5: Add the real scan and DeleteZone**

Append to `internal/rooms/zone_lifecycle.go`:

```go
// ZoneDeletionBlockers is the production wiring of the scan.
func ZoneDeletionBlockers(zone string) []ZoneBlocker {
	return ZoneDeletionBlockersWith(zone, zoneBlockerSources{
		roomIdsInZone: func(z string) []int {
			out := []int{}
			for _, id := range GetAllRoomIds() {
				if r := LoadRoomTemplate(id); r != nil && r.Zone == z {
					out = append(out, id)
				}
			}
			return out
		},
		zoneRootRoomId: func(z string) int {
			id, _ := GetZoneRoot(z)
			return id
		},
		contentFiles: func(z string) []string {
			out := []string{}
			base := configs.GetFilePathsConfig().DataFiles.String()
			for _, d := range zoneContentDirs() {
				dir := util.FilePath(base, "/", d, "/", ZoneNameSanitize(z))
				entries, err := os.ReadDir(dir)
				if err != nil {
					continue // tree absent for this zone
				}
				for _, e := range entries {
					if !e.IsDir() {
						out = append(out, d+"/"+ZoneNameSanitize(z)+"/"+e.Name())
					}
				}
			}
			return out
		},
		inboundExits: func(z string) []string {
			out := []string{}
			for _, id := range GetAllRoomIds() {
				src := LoadRoomTemplate(id)
				if src == nil || src.Zone == z {
					continue // same-zone exits die with the zone
				}
				for name, ex := range src.Exits {
					dst := LoadRoomTemplate(ex.RoomId)
					if dst != nil && dst.Zone == z {
						out = append(out, fmt.Sprintf("room %d (%s) %s", id, src.Zone, name))
					}
				}
			}
			return out
		},
		playersInZone: func(z string) []string {
			out := []string{}
			for _, id := range GetAllRoomIds() {
				r := LoadRoomTemplate(id)
				if r == nil || r.Zone != z {
					continue
				}
				if n := len(r.GetPlayers()); n > 0 {
					out = append(out, fmt.Sprintf("%d player(s) in room %d", n, id))
				}
			}
			return out
		},
	})
}

// DeleteZone removes every directory a zone owns and drops it from the room
// manager. It re-checks the blockers itself — never trust the caller.
//
// The caller is responsible for invalidating the mapper cache afterwards;
// internal/rooms cannot import internal/mapper (mapper imports rooms).
func DeleteZone(zone string) error {
	if _, ok := roomManager.zones[zone]; !ok {
		return fmt.Errorf("zone %q does not exist", zone)
	}
	if b := ZoneDeletionBlockers(zone); len(b) > 0 {
		return fmt.Errorf("zone %q is not empty (%d blockers)", zone, len(b))
	}

	base := configs.GetFilePathsConfig().DataFiles.String()
	folder := ZoneNameSanitize(zone)
	for _, d := range zoneAllDirs() {
		dir := util.FilePath(base, "/", d, "/", folder)
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("removing %s: %w", dir, err)
		}
	}

	for _, id := range GetAllRoomIds() {
		if r := LoadRoomTemplate(id); r != nil && r.Zone == zone {
			delete(roomManager.rooms, id)
		}
	}
	delete(roomManager.zones, zone)
	return nil
}
```

Imports needed in this file: `fmt`, `os`, plus
`github.com/GoMudEngine/GoMud/internal/configs` and
`github.com/GoMudEngine/GoMud/internal/util`.

- [ ] **Step 6: Verify build and full package tests**

Run: `go build ./... && go test ./internal/rooms/`
Expected: build clean, PASS

Signatures used above, all verified present:
`(r *Room) GetPlayers(findTypes ...FindFlag) []int` (`rooms.go:1490`) and the
unexported `roomManager.rooms map[int]*Room` (`roommanager.go:51`), which this
file can reach because it is in package `rooms`.

- [ ] **Step 7: Commit**

```bash
git add internal/rooms/zone_lifecycle.go internal/rooms/zone_lifecycle_test.go
git commit -m "feat(rooms): DeleteZone + production blocker scan"
```

---

### Task 4: `zoneDeps` seam and `Build.Zone.Delete`

**Files:**
- Create: `modules/gmcp/gmcp.Zone.go`
- Create: `modules/gmcp/gmcp.Zone_test.go`

- [ ] **Step 1: Write the failing test**

Create `modules/gmcp/gmcp.Zone_test.go`:

```go
package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/rooms"
)

type fakeZoneWorld struct {
	cfgs     map[string]*rooms.ZoneConfig
	blockers map[string][]rooms.ZoneBlocker
	deleted  []string
	saved    []rooms.ZoneConfig
}

func newFakeZoneWorld() *fakeZoneWorld {
	return &fakeZoneWorld{
		cfgs:     map[string]*rooms.ZoneConfig{"Testzone": {Name: "Testzone", RoomId: 100}},
		blockers: map[string][]rooms.ZoneBlocker{},
	}
}

func (w *fakeZoneWorld) deps() zoneDeps {
	return zoneDeps{
		load: func(z string) *rooms.ZoneConfig { return w.cfgs[z] },
		save: func(c rooms.ZoneConfig) error {
			w.saved = append(w.saved, c)
			cp := c
			w.cfgs[c.Name] = &cp
			return nil
		},
		del: func(z string) error {
			w.deleted = append(w.deleted, z)
			delete(w.cfgs, z)
			return nil
		},
		blockers:  func(z string) []rooms.ZoneBlocker { return w.blockers[z] },
		zoneNames: func() []string { return []string{"Testzone"} },
		roomIds:   func(z string) []int { return []int{100} },
	}
}

func TestBuildZoneDelete_BlocksWhenNotEmpty(t *testing.T) {
	w := newFakeZoneWorld()
	w.blockers["Testzone"] = []rooms.ZoneBlocker{
		{Kind: "room", Id: "room 101"},
		{Kind: "inbound-exit", Id: "room 900 (Other) east"},
	}

	res := buildZoneDelete(w.deps(), "Testzone")

	if res.Ok {
		t.Fatal("delete must be refused when the zone is not empty")
	}
	if len(res.ZoneRefs) != 2 {
		t.Errorf("expected 2 blockers surfaced, got %d", len(res.ZoneRefs))
	}
	if len(w.deleted) != 0 {
		t.Error("d.del must not be called when blocked")
	}
}

func TestBuildZoneDelete_DeletesCleanZone(t *testing.T) {
	w := newFakeZoneWorld()
	res := buildZoneDelete(w.deps(), "Testzone")
	if !res.Ok {
		t.Fatalf("clean zone should delete, got %+v", res)
	}
	if len(w.deleted) != 1 || w.deleted[0] != "Testzone" {
		t.Errorf("expected Testzone deleted, got %v", w.deleted)
	}
}

func TestBuildZoneDelete_UnknownZone(t *testing.T) {
	w := newFakeZoneWorld()
	if res := buildZoneDelete(w.deps(), "Nowhere"); res.Ok {
		t.Error("unknown zone must not report success")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./modules/gmcp/ -run TestBuildZoneDelete -v`
Expected: FAIL — `undefined: zoneDeps`

- [ ] **Step 3: Write minimal implementation**

Create `modules/gmcp/gmcp.Zone.go`:

```go
package gmcp

// Build.Zone.* GMCP packages — the server side of the admin web zone editor.
// Handlers take a zoneDeps so they unit-test against a fake world; realZoneDeps
// wires the live engine.

import (
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

type zoneDeps struct {
	load      func(zone string) *rooms.ZoneConfig
	save      func(cfg rooms.ZoneConfig) error
	del       func(zone string) error
	blockers  func(zone string) []rooms.ZoneBlocker
	zoneNames func() []string
	roomIds   func(zone string) []int
}

func realZoneDeps() zoneDeps {
	return zoneDeps{
		load: rooms.GetZoneConfig,
		save: func(cfg rooms.ZoneConfig) error { return rooms.SaveZoneConfig(&cfg) },
		del:  rooms.DeleteZone,
		blockers: rooms.ZoneDeletionBlockers,
		zoneNames: rooms.GetAllZoneNames,
		roomIds: func(zone string) []int {
			out := []int{}
			for _, id := range rooms.GetAllRoomIds() {
				if r := rooms.LoadRoomTemplate(id); r != nil && r.Zone == zone {
					out = append(out, id)
				}
			}
			return out
		},
	}
}

func buildZoneDelete(d zoneDeps, zone string) BuildResult {
	if d.load(zone) == nil {
		return buildErr("zone %q not found", zone)
	}
	if b := d.blockers(zone); len(b) > 0 {
		return BuildResult{
			Error:    "zone is not empty — remove these first",
			ZoneRefs: b,
		}
	}
	if err := d.del(zone); err != nil {
		return buildErr("could not delete zone %q: %s", zone, err.Error())
	}
	return BuildResult{Ok: true, Message: "zone " + zone + " deleted"}
}
```

Add the `ZoneRefs` field to `BuildResult` in `modules/gmcp/gmcp.Build.go`:

```go
	ZoneRefs []rooms.ZoneBlocker `json:"zoneRefs,omitempty"` // Build.Zone.Delete: what blocks a delete
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./modules/gmcp/ -run TestBuildZoneDelete -v`
Expected: PASS (all three)

- [ ] **Step 5: Commit**

```bash
git add modules/gmcp/gmcp.Zone.go modules/gmcp/gmcp.Zone_test.go modules/gmcp/gmcp.Build.go
git commit -m "feat(build): Build.Zone.Delete behind a zoneDeps seam"
```

---

### Task 5: `Build.Zone.List` and `Build.Zone.Get`

**Files:**
- Modify: `modules/gmcp/gmcp.Zone.go`
- Modify: `modules/gmcp/gmcp.Zone_test.go`

- [ ] **Step 1: Write the failing test**

Append to `modules/gmcp/gmcp.Zone_test.go`:

```go
func TestBuildZoneList_ReportsRoomCounts(t *testing.T) {
	w := newFakeZoneWorld()
	rowsAny := buildZoneList(w.deps())
	if len(rowsAny.Zones) != 1 {
		t.Fatalf("expected 1 zone, got %d", len(rowsAny.Zones))
	}
	row := rowsAny.Zones[0]
	if row.Zone != "Testzone" || row.RoomCount != 1 {
		t.Errorf("unexpected row: %+v", row)
	}
}

func TestBuildZoneGet_MapsFieldsAndEnums(t *testing.T) {
	w := newFakeZoneWorld()
	w.cfgs["Testzone"] = &rooms.ZoneConfig{
		Name: "Testzone", RoomId: 100, DefaultBiome: "land",
		Region: "Windward Marches", Instanced: true, DeathPolicy: "ejected",
		NonCartesian: true, DefaultPlane: 7,
	}
	d, ok := buildZoneGet(w.deps(), "Testzone")
	if !ok {
		t.Fatal("expected zone detail")
	}
	if d.Name != "Testzone" || d.DefaultBiome != "land" || !d.Instanced ||
		d.DeathPolicy != "ejected" || !d.NonCartesian || d.DefaultPlane != 7 {
		t.Errorf("fields not mapped: %+v", d)
	}
	if len(d.Enums.DeathPolicies) == 0 {
		t.Error("death policy enum must be server-supplied")
	}
}

func TestBuildZoneGet_UnknownZone(t *testing.T) {
	w := newFakeZoneWorld()
	if _, ok := buildZoneGet(w.deps(), "Nowhere"); ok {
		t.Error("unknown zone must not return detail")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./modules/gmcp/ -run "TestBuildZoneList|TestBuildZoneGet" -v`
Expected: FAIL — `undefined: buildZoneList`

- [ ] **Step 3: Write minimal implementation**

Append to `modules/gmcp/gmcp.Zone.go`:

```go
// ---- server -> client payloads ----

type zoneListRow struct {
	Zone      string `json:"zone"`
	RoomCount int    `json:"roomCount"`
	Instanced bool   `json:"instanced"`
}

type zoneListPayload struct {
	Zones []zoneListRow `json:"zones"`
}

type zoneEnums struct {
	Biomes        []string `json:"biomes"`
	DeathPolicies []string `json:"deathPolicies"`
	Regions       []string `json:"regions"` // observed values, as suggestions
}

type zoneDetail struct {
	zoneUpdateReq
	Enums zoneEnums `json:"enums"`
}

// zoneUpdateReq is both the client->server update payload and the editable
// half of zoneDetail. Name is NOT editable here — renaming is Phase 2.
type zoneUpdateReq struct {
	Name           string   `json:"name"`
	RoomId         int      `json:"roomId"`
	DefaultBiome   string   `json:"defaultBiome"`
	Region         string   `json:"region"`
	MusicFile      string   `json:"musicFile"`
	IdleMessages   []string `json:"idleMessages"`
	Instanced      bool     `json:"instanced"`
	DeathPolicy    string   `json:"deathPolicy"`
	PortalDuration string   `json:"portalDuration"`
	EntryRoom      int      `json:"entryRoom"`
	AllowRecall    bool     `json:"allowRecall"`
	NonCartesian   bool     `json:"nonCartesian"`
	DefaultPlane   int      `json:"defaultPlane"`
}

type zoneGetReq struct {
	Zone string `json:"zone"`
}

type zoneDeleteReq struct {
	Zone string `json:"zone"`
}

func buildZoneList(d zoneDeps) zoneListPayload {
	out := zoneListPayload{Zones: []zoneListRow{}}
	for _, z := range d.zoneNames() {
		cfg := d.load(z)
		instanced := cfg != nil && cfg.Instanced
		out.Zones = append(out.Zones, zoneListRow{
			Zone:      z,
			RoomCount: len(d.roomIds(z)),
			Instanced: instanced,
		})
	}
	return out
}

func buildZoneGet(d zoneDeps, zone string) (zoneDetail, bool) {
	cfg := d.load(zone)
	if cfg == nil {
		return zoneDetail{}, false
	}
	return zoneDetail{
		zoneUpdateReq: zoneUpdateReq{
			Name: cfg.Name, RoomId: cfg.RoomId,
			DefaultBiome: cfg.DefaultBiome, Region: cfg.Region,
			MusicFile: cfg.MusicFile, IdleMessages: cfg.IdleMessages,
			Instanced: cfg.Instanced, DeathPolicy: cfg.DeathPolicy,
			PortalDuration: cfg.PortalDuration, EntryRoom: cfg.EntryRoom,
			AllowRecall: cfg.AllowRecall, NonCartesian: cfg.NonCartesian,
			DefaultPlane: cfg.DefaultPlane,
		},
		Enums: collectZoneEnums(d),
	}, true
}

func collectZoneEnums(d zoneDeps) zoneEnums {
	regions := []string{}
	seen := map[string]bool{}
	for _, z := range d.zoneNames() {
		if cfg := d.load(z); cfg != nil && cfg.Region != "" && !seen[cfg.Region] {
			seen[cfg.Region] = true
			regions = append(regions, cfg.Region)
		}
	}
	return zoneEnums{
		Biomes:        zoneBiomeNames(),
		DeathPolicies: []string{"rejoin", "ejected"},
		Regions:       regions,
	}
}
```

- [ ] **Step 4: Add the biome list helper**

Append to `gmcp.Zone.go`. Note the id accessor is `Id()`, not `BiomeId()`, and
`GetAllBiomes` returns values (not pointers), so take the loop variable's
address to call the pointer-receiver method:

```go
// zoneBiomeNames returns every biome id the engine knows, for the editor's
// dropdown. Server-supplied so a typo cannot reach zone-config.
// rooms.GetAllBiomes() []BiomeInfo — biomes.go:143; (*BiomeInfo).Id() — :64.
func zoneBiomeNames() []string {
	out := []string{}
	for _, b := range rooms.GetAllBiomes() {
		bi := b
		out = append(out, bi.Id())
	}
	sort.Strings(out)
	return out
}
```

Add `"sort"` to imports.

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./modules/gmcp/ -run "TestBuildZoneList|TestBuildZoneGet" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add modules/gmcp/gmcp.Zone.go modules/gmcp/gmcp.Zone_test.go
git commit -m "feat(build): Build.Zone.List and Build.Zone.Get"
```

---

### Task 6: `Build.Zone.Update` with the non-Cartesian plane rule

This is the task that enforces spec §5.1. A `non_cartesian` zone on plane 0
marks the whole overworld non-Euclidean via the sticky OR in
`PlaneRegistry.Mark`, silently disabling collision and reciprocity enforcement
world-wide. That exact bug was live until `af565b1f0`.

**Files:**
- Modify: `modules/gmcp/gmcp.Zone.go`
- Modify: `modules/gmcp/gmcp.Zone_test.go`

- [ ] **Step 1: Write the failing test**

Append to `modules/gmcp/gmcp.Zone_test.go`:

```go
func baseZoneReq() zoneUpdateReq {
	return zoneUpdateReq{
		Name: "Testzone", RoomId: 100, DefaultBiome: "land",
		DeathPolicy: "rejoin", DefaultPlane: 7,
	}
}

// A non_cartesian zone on plane 0 marks plane 0 (the overworld) non-Euclidean
// via the sticky OR in PlaneRegistry.Mark, disabling collision and reciprocity
// enforcement for the ENTIRE world. Refuse to save that combination.
func TestBuildZoneUpdate_RejectsNonCartesianOnOverworldPlane(t *testing.T) {
	w := newFakeZoneWorld()
	req := baseZoneReq()
	req.NonCartesian = true
	req.DefaultPlane = 0

	if res := buildZoneUpdate(w.deps(), req); res.Ok {
		t.Error("non_cartesian with plane 0 must be rejected")
	}
	if len(w.saved) != 0 {
		t.Errorf("nothing may be saved on validation failure, got %d", len(w.saved))
	}
}

func TestBuildZoneUpdate_AllowsNonCartesianOnOwnPlane(t *testing.T) {
	w := newFakeZoneWorld()
	req := baseZoneReq()
	req.NonCartesian = true
	req.DefaultPlane = 7

	if res := buildZoneUpdate(w.deps(), req); !res.Ok {
		t.Errorf("non_cartesian on its own plane should save: %+v", res)
	}
}

func TestBuildZoneUpdate_RejectsUnknownDeathPolicy(t *testing.T) {
	w := newFakeZoneWorld()
	req := baseZoneReq()
	req.DeathPolicy = "vaporize"
	if res := buildZoneUpdate(w.deps(), req); res.Ok {
		t.Error("unknown death policy must be rejected")
	}
}

func TestBuildZoneUpdate_RoundTripsFields(t *testing.T) {
	w := newFakeZoneWorld()
	req := baseZoneReq()
	req.Region = "Windward Marches"
	req.MusicFile = "theme.mp3"
	req.Instanced = true
	req.EntryRoom = 100

	if res := buildZoneUpdate(w.deps(), req); !res.Ok {
		t.Fatalf("update should succeed: %+v", res)
	}
	got := w.saved[0]
	if got.Region != "Windward Marches" || got.MusicFile != "theme.mp3" ||
		!got.Instanced || got.EntryRoom != 100 {
		t.Errorf("fields not round-tripped: %+v", got)
	}
}

func TestBuildZoneUpdate_RejectsEntryRoomOutsideZone(t *testing.T) {
	w := newFakeZoneWorld() // roomIds returns only [100]
	req := baseZoneReq()
	req.Instanced = true
	req.EntryRoom = 999
	if res := buildZoneUpdate(w.deps(), req); res.Ok {
		t.Error("entry room outside the zone must be rejected")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./modules/gmcp/ -run TestBuildZoneUpdate -v`
Expected: FAIL — `undefined: buildZoneUpdate`

- [ ] **Step 3: Write minimal implementation**

Append to `modules/gmcp/gmcp.Zone.go`:

```go
func buildZoneUpdate(d zoneDeps, req zoneUpdateReq) BuildResult {
	cfg := d.load(req.Name)
	if cfg == nil {
		return buildErr("zone %q not found", req.Name)
	}

	// A non_cartesian zone marks its plane non-Euclidean, and PlaneRegistry.Mark
	// accumulates with a sticky OR — so on plane 0 it marks the ENTIRE overworld
	// non-Euclidean and silently disables collision + reciprocity enforcement
	// world-wide. Never allow that combination to be saved.
	if req.NonCartesian && req.DefaultPlane == 0 {
		return buildErr("a non-Cartesian zone needs its own plane — plane 0 is the overworld, " +
			"and flagging it non-Euclidean disables map consistency checks for the whole world")
	}

	if req.DeathPolicy != "" && req.DeathPolicy != "rejoin" && req.DeathPolicy != "ejected" {
		return buildErr("death_policy %q invalid; valid: rejoin, ejected", req.DeathPolicy)
	}

	inZone := map[int]bool{}
	for _, id := range d.roomIds(req.Name) {
		inZone[id] = true
	}
	if req.EntryRoom != 0 && !inZone[req.EntryRoom] {
		return buildErr("entry room %d is not in zone %q", req.EntryRoom, req.Name)
	}
	if req.RoomId != 0 && !inZone[req.RoomId] {
		return buildErr("root room %d is not in zone %q", req.RoomId, req.Name)
	}

	// Copy the editable fields onto the loaded config so anything the form does
	// not carry (RoomIds, Mutators) survives the save.
	out := *cfg
	out.RoomId = req.RoomId
	out.DefaultBiome = req.DefaultBiome
	out.Region = req.Region
	out.MusicFile = req.MusicFile
	out.IdleMessages = req.IdleMessages
	out.Instanced = req.Instanced
	out.DeathPolicy = req.DeathPolicy
	out.PortalDuration = req.PortalDuration
	out.EntryRoom = req.EntryRoom
	out.AllowRecall = req.AllowRecall
	out.NonCartesian = req.NonCartesian
	out.DefaultPlane = req.DefaultPlane

	if err := d.save(out); err != nil {
		return buildErr("could not save zone %q: %s", req.Name, err.Error())
	}
	return BuildResult{Ok: true, Message: "zone " + req.Name + " saved"}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./modules/gmcp/ -run TestBuildZoneUpdate -v`
Expected: PASS (all five)

- [ ] **Step 5: Commit**

```bash
git add modules/gmcp/gmcp.Zone.go modules/gmcp/gmcp.Zone_test.go
git commit -m "feat(build): Build.Zone.Update with the non-Cartesian plane guard"
```

---

### Task 7: Dispatch and admin gate

**Files:**
- Modify: `modules/gmcp/gmcp.go` (line ~486, the routed-verb case list)
- Modify: `modules/gmcp/gmcp.Build.go` (the `handleBuildOp` switch, near the existing `Build.Zone.Create` case at :322)

- [ ] **Step 1: Add the verbs to the routed list**

In `modules/gmcp/gmcp.go`, extend the existing case list:

```go
		case `Build.Room.Create`, `Build.Room.Update`, `Build.Room.Delete`,
			`Build.Exit.Add`, `Build.Exit.Remove`, `Build.Room.Get`, `Build.Room.List`, `Build.Map.Request`,
			`Build.Zone.Create`, `Build.Zone.List`, `Build.Zone.Get`, `Build.Zone.Update`, `Build.Zone.Delete`,
			`Build.Item.List`, `Build.Item.Get`, `Build.Item.Create`, `Build.Item.Update`, `Build.Item.Delete`,
			`Build.Mob.List`, `Build.Mob.Get`, `Build.Mob.Create`, `Build.Mob.Update`, `Build.Mob.Delete`, `Build.Mob.Spawn`:
```

- [ ] **Step 2: Add the dispatch cases**

In `modules/gmcp/gmcp.Build.go`, immediately after the existing
`case Build.Zone.Create:` block:

```go
	case `Build.Zone.List`:
		sendZoneList(uid)
	case `Build.Zone.Get`:
		var req zoneGetReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Zone.Get payload"))
			break
		}
		if d, ok := buildZoneGet(realZoneDeps(), req.Zone); ok {
			sendZoneDetail(uid, d)
		} else {
			sendBuildResult(uid, buildErr("zone %q not found", req.Zone))
		}
	case `Build.Zone.Update`:
		var req zoneUpdateReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Zone.Update payload"))
			break
		}
		sendBuildResult(uid, buildZoneUpdate(realZoneDeps(), req))
		sendZoneList(uid)
	case `Build.Zone.Delete`:
		var req zoneDeleteReq
		if json.Unmarshal(evt.Payload, &req) != nil {
			sendBuildResult(uid, buildErr("bad Build.Zone.Delete payload"))
			break
		}
		res := buildZoneDelete(realZoneDeps(), req.Zone)
		if res.Ok {
			// rooms cannot import mapper, so cache invalidation happens here.
			mapper.ClearCache()
		}
		sendBuildResult(uid, res)
		sendZoneList(uid)
```

- [ ] **Step 3: Add the send helpers**

Append to `modules/gmcp/gmcp.Zone.go`:

```go
func sendZoneList(uid int) {
	sendGMCP(uid, `Build.Zones`, buildZoneList(realZoneDeps()))
}

func sendZoneDetail(uid int, d zoneDetail) {
	sendGMCP(uid, `Build.Zone`, d)
}
```

`sendGMCP(userId int, module string, payload interface{})` is at
`gmcp.Mudlet.go:180` — verified; the calls above match it. `sendMobList`
(`gmcp.Mob.go:882`) is the pattern being mirrored.

- [ ] **Step 4: Verify build and the whole gmcp package**

Run: `go build ./... && go test ./modules/gmcp/`
Expected: build clean, PASS

- [ ] **Step 5: Commit**

```bash
git add modules/gmcp/gmcp.go modules/gmcp/gmcp.Build.go modules/gmcp/gmcp.Zone.go
git commit -m "feat(build): route and admin-gate the Build.Zone.* verbs"
```

---

### Task 8: Zones panel in the web builder

**Files:**
- Create: `_datafiles/html/public/static/js/zones.js`
- Modify: `_datafiles/html/public/build.html`

Follow `_datafiles/html/public/static/js/mobs.js` for module shape, field
helpers (`H.selectField`, `H.textField`, `H.checkField`, `H.numField`,
`H.listField`), and list rendering. **Read `mobs.js` before writing this.**

- [ ] **Step 1: Add the mode button**

In `build.html`, beside the existing mode buttons (~line 181):

```html
      <button id="tb-mode-zones" class="mode-btn">Zones</button>
```

- [ ] **Step 2: Route the payloads**

In `build.html`'s GMCP dispatch (~line 291, where `Build.Items` and
`Build.Mobs` are routed):

```javascript
    var ZP = window.Builder.ZonesPanel;
    if (ns === "Build.Zones") { if (ZP) ZP.renderList(obj); return; }
    if (ns === "Build.Zone")  { if (ZP) ZP.renderDetail(obj); return; }
```

- [ ] **Step 3: Write the panel**

Create `zones.js` exposing `window.Builder.ZonesPanel` with `renderList`,
`renderDetail`, and a `mode` hook matching `MobsPanel`. Requirements:

- Zone list rows show name, room count, and an instanced badge.
- Form fields per spec §5: root room, biome (`enums.biomes` dropdown), region
  (text + `enums.regions` datalist), music file, idle messages (list),
  non-Cartesian (checkbox), default plane (number), and an instanced block —
  entry room, portal duration, death policy (`enums.deathPolicies` dropdown),
  allow recall — revealed only when Instanced is ticked.
- **Non-Cartesian checkbox behaviour:** when ticked while default plane is 0,
  show inline text `"A non-Cartesian zone needs its own plane."` and disable
  Save until the plane is non-zero. This mirrors the server guard from Task 6;
  the server remains authoritative.
- Delete button renders `res.zoneRefs` as a list of `kind — id` lines rather
  than a bare failure message.

- [ ] **Step 4: Manual smoke**

Run: `go run .`, open `http://localhost:8090/build`, log in as an admin,
click **Zones**. Verify: list populates with room counts; selecting a zone
fills the form; ticking Instanced reveals the block; ticking Non-Cartesian
with plane 0 blocks Save; deleting a populated zone lists blockers.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/html/public/static/js/zones.js _datafiles/html/public/build.html
git commit -m "feat(build): Zones panel in the web builder"
```

---

### Task 9: Filesystem integration test

Unit tests use fakes; this proves the real delete removes real directories.

**Files:**
- Modify: `internal/rooms/zone_lifecycle_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestDeleteZone_RemovesEveryTree(t *testing.T) {
	if os.Getenv(`DOGMUD_BOOT_SMOKE`) == `` {
		t.Skip("set DOGMUD_BOOT_SMOKE=1 to run the filesystem zone test")
	}
	const zone = "Ziggurat Test Zone"

	roomId, err := CreateZone(zone)
	assert.NoError(t, err)
	assert.NotZero(t, roomId)

	folder := ZoneNameSanitize(zone)
	base := configs.GetFilePathsConfig().DataFiles.String()
	roomsDir := util.FilePath(base, "/", "rooms", "/", folder)
	_, statErr := os.Stat(roomsDir)
	assert.NoError(t, statErr, "zone folder should exist after CreateZone")

	// Root room only, no content -> deletable.
	assert.Empty(t, ZoneDeletionBlockers(zone))
	assert.NoError(t, DeleteZone(zone))

	for _, d := range zoneAllDirs() {
		dir := util.FilePath(base, "/", d, "/", folder)
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("%s still exists after DeleteZone", dir)
		}
	}
	assert.NotContains(t, GetAllZoneNames(), zone)
}
```

- [ ] **Step 2: Run it**

Run: `DOGMUD_BOOT_SMOKE=1 go test ./internal/rooms/ -run TestDeleteZone_RemovesEveryTree -v`
Expected: PASS. If it fails leaving the zone behind, remove
`_datafiles/world/dogmud/*/ziggurat_test_zone/` by hand before re-running.

- [ ] **Step 3: Commit**

```bash
git add internal/rooms/zone_lifecycle_test.go
git commit -m "test(rooms): filesystem integration test for DeleteZone"
```

---

### Task 10: Full verification gate

- [ ] **Step 1: Full suite**

Run: `go test ./...`
Expected: no failures.

- [ ] **Step 2: Format and vet**

Run: `gofmt -l internal modules && go vet ./...`
Expected: no output from either.

- [ ] **Step 3: Boot + consistency under the prod setting**

Confirm `_datafiles/config.yaml` has `MapConsistencyEnforce: panic` (note:
this file carries git **skip-worktree**, so local edits never show in
`git status`).

Run: `go run .` and watch for
`mapper.ValidateZoneConsistency errors=0 warnings=0 mode="panic"` with no
`ERROR: PANIC` lines. Ctrl-C after the HTTP server starts.

- [ ] **Step 4: Unknown-key drift gate**

Run: `DOGMUD_BOOT_SMOKE=1 go test . -run TestSmoke_NoNewSilentlyIgnoredYAMLKeys -v`
Expected: PASS with `new: 0`.

- [ ] **Step 5: Plane-0 regression check**

Confirm no `non_cartesian` zone sits on the overworld plane:

Run:
```bash
grep -rln "non_cartesian: true" _datafiles/world/dogmud/rooms/*/zone-config.yaml \
  | sed 's|.*/rooms/||;s|/zone-config.yaml||' \
  | while read z; do echo "$z: $(grep -L '^plane:' _datafiles/world/dogmud/rooms/$z/[0-9]*.yaml | wc -l) room(s) without a plane"; done
```
Expected: every zone reports `0 room(s) without a plane`.

- [ ] **Step 6: Update PATCH_NOTES.md**

Add a dated staff-facing entry in the house voice (prose, no identifiers, no
raw numbers) describing zone deletion and the zone settings editor.

- [ ] **Step 7: Commit**

```bash
git add PATCH_NOTES.md
git commit -m "docs(patch-notes): zone lifecycle + config editor"
```

- [ ] **Step 8: Browser gate — hand to the user**

Headless checks cannot judge form usability or blocker messaging. Ask the user
to drive `/build` → Zones and confirm before this is called done.

---

## Out of scope for Phase 1

Zone **rename** (Phase 2), moving rooms between zones, zone-level spawn lists
(sub-project 4), and any migration of player fog-of-war.
