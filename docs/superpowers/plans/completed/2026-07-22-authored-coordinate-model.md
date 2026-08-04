# Authored Coordinate Model — Implementation Plan (admin web-building 1a)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make each room the authoritative holder of its `(x, y, z, plane)` coordinate, enforce no-overlap within every plane, migrate today's crawl-derived positions into authored coords losslessly, and switch the mapper to read stored coords — the engine foundation for the web room-builder (1b).

**Architecture:** Data-model first (add coords to `Room`), then a plane registry, then plane-aware enforcement (4-tuple collision + authored-delta check), then a lossless migration that backfills coords from the existing crawl, then the mapper switch, then instance planes. Each task is a committable unit; the migration asserts `authored == crawled` so rendered positions change for zero rooms.

**Tech Stack:** Go (GoMud fork), YAML room templates, `go test`, the `internal/migration` framework.

**Spec:** `docs/superpowers/specs/2026-07-22-authored-coordinate-model-design.md`

**Key anchors (verified):** `Room` struct `internal/rooms/rooms.go:76`; `ZoneConfig` `internal/rooms/zoneconfig.go:8` (`NonCartesian` :21); `mapNode` `internal/mapper/mapper.node.go:4`; `positionDelta`/`Combine` `internal/mapper/mapper.go:163-176`; `RoomGrid`/`addNode` `mapper.go:178-212`; crawl `mapper.Start` `mapper.go:249-339`; `findCollisions` `mapper.consistency.go:50`; `CheckConsistency` `:104`; `longSpanCrossesRoom` `:160`; `ValidateZoneConsistency` `:206`; `roomCrawlable` `:97`; instance creation `internal/rooms/instances.go:297` → `CreateEphemeralZone` `ephemeral.go:141`; migration pattern `internal/migration/0.14.0.go`.

---

## File Structure

- `internal/rooms/rooms.go` — add `X/Y/Z/Plane` to `Room`. (Task 1)
- `internal/rooms/zoneconfig.go` — add `DefaultPlane`. (Task 1)
- `internal/rooms/planes.go` *(new)* — plane registry + non-Euclidean flag. (Task 2)
- `internal/rooms/planes_test.go` *(new)*. (Task 2)
- `internal/rooms/placement.go` *(new)* — `ValidatePlacement` build-time gate. (Task 4)
- `internal/rooms/placement_test.go` *(new)*. (Task 4)
- `internal/mapper/mapper.node.go` — add `Plane` to `mapNode`. (Task 3)
- `internal/mapper/mapper.consistency.go` — 4-tuple collision + authored-delta + non-Euclidean skip. (Task 3)
- `internal/mapper/mapper.consistency_test.go` — update existing tests + add plane cases. (Task 3)
- `internal/migration/0.15.0.go` *(new)* + `_test.go` — coord backfill migration. (Task 5)
- `internal/mapper/mapper.go` — `Start` reads authored coords. (Task 6)
- `internal/rooms/ephemeral.go` / `instances.go` — instance plane assignment. (Task 7)
- Verification only. (Task 8)

---

## Task 1: Add coordinate fields to Room + ZoneConfig

**Files:**
- Modify: `internal/rooms/rooms.go:89` (after `Biome`)
- Modify: `internal/rooms/zoneconfig.go:20`
- Test: `internal/rooms/rooms_coords_test.go` *(new)*

- [ ] **Step 1: Write the failing round-trip test.** Create `internal/rooms/rooms_coords_test.go`:

```go
package rooms

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestRoom_CoordYamlRoundTrip(t *testing.T) {
	in := Room{RoomId: 5, Zone: "Test", Title: "t", Description: "d", X: 3, Y: -1, Z: 0, Plane: 2}
	b, err := yaml.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Room
	if err := yaml.Unmarshal(b, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.X != 3 || out.Y != -1 || out.Z != 0 || out.Plane != 2 {
		t.Errorf("coords lost: got x=%d y=%d z=%d plane=%d", out.X, out.Y, out.Z, out.Plane)
	}
	// Plane 0 / origin should be omitempty (not clutter every YAML).
	zero, _ := yaml.Marshal(Room{RoomId: 1, Zone: "Z", Title: "t", Description: "d"})
	if got := string(zero); containsAny(got, "\nx:", "\nplane:") {
		t.Errorf("zero coords should be omitempty; got:\n%s", got)
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) > 0 && (len(s) >= len(sub)) && (indexOf(s, sub) >= 0) {
			return true
		}
	}
	return false
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
```

- [ ] **Step 2: Run — expect FAIL** (fields undefined)

Run: `go test ./internal/rooms/ -run TestRoom_CoordYamlRoundTrip`
Expected: FAIL — `in.X undefined`.

- [ ] **Step 3: Add the fields.** In `internal/rooms/rooms.go`, immediately after the `Biome` field (line 89), add:

```go
	X                 int                               `yaml:"x,omitempty" instance:"skip"`               // authored grid coordinate within Plane
	Y                 int                               `yaml:"y,omitempty" instance:"skip"`               // authored grid coordinate within Plane (engine frame: north = y-1)
	Z                 int                               `yaml:"z,omitempty" instance:"skip"`               // vertical level (up = z+1, down = z-1)
	Plane             int                               `yaml:"plane,omitempty" instance:"skip"`           // coordinate-space id; 0 = overworld
```

In `internal/rooms/zoneconfig.go` after `NonCartesian` (line 21), add:

```go
	DefaultPlane   int                  `yaml:"default_plane,omitempty"`   // builder convenience: default plane for new rooms in this zone (rooms' own plane is authoritative)
```

- [ ] **Step 4: Run — expect PASS**

Run: `go test ./internal/rooms/ -run TestRoom_CoordYamlRoundTrip`
Expected: PASS.

- [ ] **Step 5: Build + commit**

Run: `go build ./...`
```bash
git add internal/rooms/rooms.go internal/rooms/zoneconfig.go internal/rooms/rooms_coords_test.go
git commit -m "feat(rooms): add authored x/y/z/plane coordinates to Room

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Plane registry + non-Euclidean flag

A plane is an int coordinate-space id with one property: Euclidean (grid-enforced) or non-Euclidean (isolated, not grid-enforced). The registry is built after rooms load: a plane is non-Euclidean if any room on it belongs to a `non_cartesian` zone.

**Files:**
- Create: `internal/rooms/planes.go`, `internal/rooms/planes_test.go`

- [ ] **Step 1: Write the failing test.** Create `internal/rooms/planes_test.go`:

```go
package rooms

import "testing"

func TestPlaneRegistry_NonEuclidean(t *testing.T) {
	reg := NewPlaneRegistry()
	reg.Mark(0, false, "overworld")
	reg.Mark(7, true, "labyrinth")
	if reg.IsNonEuclidean(0) {
		t.Error("plane 0 should be Euclidean")
	}
	if !reg.IsNonEuclidean(7) {
		t.Error("plane 7 should be non-Euclidean")
	}
	// Unknown planes default to Euclidean (safe: enforce by default).
	if reg.IsNonEuclidean(99) {
		t.Error("unknown plane should default Euclidean")
	}
	// Mark wins if ANY contributor is non-Euclidean (idempotent OR).
	reg.Mark(0, true, "overworld")
	if !reg.IsNonEuclidean(0) {
		t.Error("plane 0 should flip non-Euclidean after a non-Euclidean mark")
	}
}
```

- [ ] **Step 2: Run — expect FAIL** (`NewPlaneRegistry` undefined)

Run: `go test ./internal/rooms/ -run TestPlaneRegistry`
Expected: FAIL.

- [ ] **Step 3: Implement the registry.** Create `internal/rooms/planes.go`:

```go
package rooms

import "sync"

// PlaneInfo describes one coordinate-space plane.
type PlaneInfo struct {
	NonEuclidean bool
	Label        string
}

// PlaneRegistry maps plane id -> PlaneInfo. A plane is non-Euclidean if ANY
// contributing zone/room marks it so (OR semantics). Unknown planes default to
// Euclidean, so enforcement is the safe default.
type PlaneRegistry struct {
	mu     sync.RWMutex
	planes map[int]PlaneInfo
}

func NewPlaneRegistry() *PlaneRegistry {
	return &PlaneRegistry{planes: map[int]PlaneInfo{}}
}

func (r *PlaneRegistry) Mark(plane int, nonEuclidean bool, label string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cur := r.planes[plane]
	cur.NonEuclidean = cur.NonEuclidean || nonEuclidean
	if label != "" {
		cur.Label = label
	}
	r.planes[plane] = cur
}

func (r *PlaneRegistry) IsNonEuclidean(plane int) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.planes[plane].NonEuclidean
}

// planeRegistry is the process-wide registry, (re)built after rooms load.
var planeRegistry = NewPlaneRegistry()

// GetPlaneRegistry returns the process registry (consumers: mapper enforcement).
func GetPlaneRegistry() *PlaneRegistry { return planeRegistry }

// RebuildPlaneRegistry walks every loaded room and marks its plane
// non-Euclidean when the room's zone is non_cartesian. Call after all rooms +
// zone configs are loaded (from the same boot pass that runs PreCacheMaps).
func RebuildPlaneRegistry() {
	reg := NewPlaneRegistry()
	for _, roomId := range GetAllRoomIds() {
		room := LoadRoom(roomId)
		if room == nil {
			continue
		}
		reg.Mark(room.Plane, IsZoneNonCartesian(room.Zone), room.Zone)
	}
	planeRegistry = reg
}
```

> **Implementer note:** confirm `GetAllRoomIds()` and `IsZoneNonCartesian(zone)` exist in `internal/rooms` (both are referenced elsewhere — `IsZoneNonCartesian` at `roommanager.go:672`, `GetAllRoomIds` used by `admin.rooms.go`). Wire `RebuildPlaneRegistry()` into the boot sequence right after `PreCacheMaps` (where `ValidateZoneConsistency` is already called) — grep for the `PreCacheMaps` call site and add the rebuild immediately before consistency validation.

- [ ] **Step 4: Run — expect PASS**; **Step 5: build + commit**

Run: `go test ./internal/rooms/ -run TestPlaneRegistry && go build ./...`
```bash
git add internal/rooms/planes.go internal/rooms/planes_test.go
git commit -m "feat(rooms): plane registry with per-plane non-Euclidean flag

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Plane-aware enforcement (4-tuple collision + authored delta)

**Files:**
- Modify: `internal/mapper/mapper.node.go:4` (add `Plane`)
- Modify: `internal/mapper/mapper.consistency.go` (`findCollisions`, `CheckConsistency`, `longSpanCrossesRoom`)
- Modify: `internal/mapper/mapper.consistency_test.go`

- [ ] **Step 1: Add `Plane` to `mapNode`.** In `mapper.node.go`, add to the struct (after `Pos`):

```go
	Plane int // coordinate-space id (0 = overworld); copied from Room.Plane
```

- [ ] **Step 2: Write failing tests** for plane-aware collision. Add to `mapper.consistency_test.go`:

```go
func TestFindCollisions_PlaneAware(t *testing.T) {
	nodes := map[int]*mapNode{
		1: {RoomId: 1, Pos: positionDelta{x: 0, y: 0, z: 0}, Plane: 0},
		2: {RoomId: 2, Pos: positionDelta{x: 0, y: 0, z: 0}, Plane: 1}, // same xyz, different plane — NOT a collision
		3: {RoomId: 3, Pos: positionDelta{x: 0, y: 0, z: 0}, Plane: 0}, // same xyz + plane as 1 — collision
	}
	groups := findCollisions(nodes)
	if len(groups) != 1 || groups[0][0] != 1 || groups[0][1] != 3 {
		t.Errorf("expected one collision {1,3}, got %v", groups)
	}
}
```

- [ ] **Step 3: Run — expect FAIL** (still 3-tuple, flags 1&2&3)

Run: `go test ./internal/mapper/ -run TestFindCollisions_PlaneAware`
Expected: FAIL.

- [ ] **Step 4: Make collision + delta plane-aware.** In `mapper.consistency.go`:

Change `findCollisions` (line 50) key from `[3]int` to `[4]int{plane,x,y,z}`:

```go
func findCollisions(nodes map[int]*mapNode) [][]int {
	byCell := map[[4]int][]int{}
	for id, n := range nodes {
		key := [4]int{n.Plane, n.Pos.x, n.Pos.y, n.Pos.z}
		byCell[key] = append(byCell[key], id)
	}
	groups := [][]int{}
	for _, ids := range byCell {
		if len(ids) >= 2 {
			sort.Ints(ids)
			groups = append(groups, ids)
		}
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i][0] < groups[j][0] })
	return groups
}
```

In `CheckConsistency` (line 104), replace the per-zone `nonCartesian` gate with the **plane registry** so mixed-plane crawls are handled per-room. Change the collision loop (lines 107-114) to skip groups whose plane is non-Euclidean:

```go
	reg := rooms.GetPlaneRegistry()
	for _, group := range findCollisions(r.crawledRooms) {
		if n := r.crawledRooms[group[0]]; n != nil && reg.IsNonEuclidean(n.Plane) {
			continue // non-Euclidean plane: overlap is allowed by design
		}
		findings = append(findings, Finding{
			Severity: "error", Kind: "collision", Zone: zone, RoomId: group[0],
			Detail: fmt.Sprintf("rooms %v occupy the same coordinate on plane %d", group, r.crawledRooms[group[0]].Plane),
		})
	}
```

In the exit loop (lines 127-141), gate the deltamismatch + noreciprocal on the source room's plane instead of the zone flag, and require same-plane for spatial exits:

```go
			srcNonEuclid := reg.IsNonEuclidean(src.Plane)
			if !srcNonEuclid {
				if dst.Plane != src.Plane && isSpatialExit(e.Direction) {
					findings = append(findings, Finding{
						Severity: "error", Kind: "deltamismatch", Zone: zone, RoomId: srcId, ExitName: exitName,
						Detail: fmt.Sprintf("spatial exit crosses planes (%d -> %d); use a portal/door exit for cross-plane links", src.Plane, dst.Plane),
					})
				} else if !samePos(e.Direction, actual) {
					findings = append(findings, Finding{
						Severity: "error", Kind: "deltamismatch", Zone: zone, RoomId: srcId, ExitName: exitName,
						Detail: fmt.Sprintf("nominal delta (%d,%d,%d) != actual (%d,%d,%d) — fix the geometry or move to a non-Euclidean plane",
							e.Direction.x, e.Direction.y, e.Direction.z, actual.x, actual.y, actual.z),
					})
				}
				if !e.OneWay && !hasReturnExit(dst, srcId) {
					findings = append(findings, Finding{
						Severity: "error", Kind: "noreciprocal", Zone: zone, RoomId: srcId, ExitName: exitName,
						Detail: fmt.Sprintf("exit to room %d has no return exit (use oneway: true if intentional)", e.RoomId),
					})
				}
			}
```

Add the `isSpatialExit` helper near `samePos`:

```go
// isSpatialExit reports whether a direction carries a real coordinate delta
// (compass/vertical). A zero delta means a portal/named exit (non-spatial).
func isSpatialExit(d positionDelta) bool {
	return d.x != 0 || d.y != 0 || d.z != 0
}
```

Update `longSpanCrossesRoom` (line 160) to key by plane too and only test cells on the source's plane:

```go
	byCell := map[[4]int]int{}
	for id, n := range r.crawledRooms {
		byCell[[4]int{n.Plane, n.Pos.x, n.Pos.y, n.Pos.z}] = id
	}
	planeOf := r.crawledRooms[srcId].Plane
	for i := 1; i < steps; i++ {
		cell := [4]int{planeOf, start.x + sx*i, start.y + sy*i, start.z}
		if id, ok := byCell[cell]; ok && id != srcId && id != dstId {
			return id
		}
	}
```

Add the `rooms` import if not present, and populate `node.Plane` where nodes are built (in `getMapNode`, copy `room.Plane` onto the node — grep `getMapNode` `mapper.go:915` and set `node.Plane = room.Plane`).

- [ ] **Step 5: Update existing consistency tests.** The `TestCheckConsistency_*` cases construct nodes without `Plane` (defaults to 0, Euclidean) — they keep passing, but the two that exercise `nonCartesian=true` (`WrapAllowedInNonCartesian`) now need the plane marked non-Euclidean instead. In `mapper.consistency_test.go`, for the `WrapAllowedInNonCartesian` test, mark the plane before the call:

```go
	rooms.GetPlaneRegistry().Mark(0, true, "test-non-euclid")
	defer rooms.GetPlaneRegistry().Mark(0, false, "reset") // note: OR semantics — see below
```

> **Implementer note:** because `Mark` is OR-only it can't un-set within a shared registry. For test isolation, give these tests their own plane id (e.g. put the non-Euclidean test's nodes on `Plane: 9` and `Mark(9, true, ...)`), leaving plane 0 Euclidean for the other tests. Adjust the test nodes' `Plane` field accordingly rather than toggling plane 0.

- [ ] **Step 6: Run all mapper tests — expect PASS**

Run: `go test ./internal/mapper/`
Expected: PASS (new plane test + updated existing suite).

- [ ] **Step 7: build + commit**

```bash
go build ./...
git add internal/mapper/mapper.node.go internal/mapper/mapper.consistency.go internal/mapper/mapper.consistency_test.go internal/mapper/mapper.go
git commit -m "feat(mapper): plane-aware collision + authored-delta enforcement

4-tuple collision key {plane,x,y,z}; cross-plane spatial exits rejected;
non-Euclidean planes (per the plane registry) skip grid enforcement.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: `ValidatePlacement` build-time gate (the API 1b calls)

**Files:**
- Create: `internal/rooms/placement.go`, `internal/rooms/placement_test.go`

- [ ] **Step 1: Write the failing test.** Create `internal/rooms/placement_test.go`:

```go
package rooms

import "testing"

func TestValidatePlacement(t *testing.T) {
	// occupied(plane,x,y,z) is injected so the test doesn't need loaded rooms.
	occ := map[[4]int]int{{0, 1, 1, 0}: 42}
	lookup := func(p, x, y, z int) (int, bool) { id, ok := occ[[4]int{p, x, y, z}]; return id, ok }

	// Free cell on a Euclidean plane: ok.
	if err := validatePlacement(0, 2, 2, 0, 0, false, lookup); err != nil {
		t.Errorf("free cell should validate: %v", err)
	}
	// Occupied cell by another room on a Euclidean plane: rejected.
	if err := validatePlacement(0, 1, 1, 0, 0, false, lookup); err == nil {
		t.Error("occupied Euclidean cell must be rejected")
	}
	// Occupied cell but it's the SAME room being moved (excludeRoomId): ok.
	if err := validatePlacement(0, 1, 1, 0, 42, false, lookup); err != nil {
		t.Errorf("self-occupied cell should validate: %v", err)
	}
	// Occupied cell on a non-Euclidean plane: ok (no grid enforcement).
	if err := validatePlacement(0, 1, 1, 0, 0, true, lookup); err != nil {
		t.Errorf("non-Euclidean overlap should be allowed: %v", err)
	}
}
```

- [ ] **Step 2: Run — expect FAIL**; **Step 3: implement.** Create `internal/rooms/placement.go`:

```go
package rooms

import "fmt"

// ValidatePlacement reports an error if placing a room at (plane,x,y,z) would
// overlap another room on a Euclidean plane. excludeRoomId is the room being
// created/moved (so it doesn't collide with itself). This is the build-time
// gate the web builder (1b) calls before persisting.
func ValidatePlacement(plane, x, y, z, excludeRoomId int) error {
	nonEuclid := GetPlaneRegistry().IsNonEuclidean(plane)
	return validatePlacement(plane, x, y, z, excludeRoomId, nonEuclid, occupantAt)
}

// validatePlacement is the testable core (dependency-injected occupancy lookup).
func validatePlacement(plane, x, y, z, excludeRoomId int, nonEuclidean bool,
	occupant func(p, x, y, z int) (int, bool)) error {
	if nonEuclidean {
		return nil
	}
	if id, ok := occupant(plane, x, y, z); ok && id != excludeRoomId {
		return fmt.Errorf("cell (plane %d, %d,%d,%d) is already occupied by room %d", plane, x, y, z, id)
	}
	return nil
}

// occupantAt scans loaded rooms for one at the given authored coordinate.
func occupantAt(plane, x, y, z int) (int, bool) {
	for _, roomId := range GetAllRoomIds() {
		r := LoadRoom(roomId)
		if r != nil && r.Plane == plane && r.X == x && r.Y == y && r.Z == z {
			return roomId, true
		}
	}
	return 0, false
}
```

- [ ] **Step 4: Run — expect PASS**; **Step 5: build + commit**

Run: `go test ./internal/rooms/ -run TestValidatePlacement && go build ./...`
```bash
git add internal/rooms/placement.go internal/rooms/placement_test.go
git commit -m "feat(rooms): ValidatePlacement build-time no-overlap gate

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Migration — backfill authored coords + planes (lossless)

Follows the `internal/migration/0.14.0.go` pattern (dry-run, idempotent, framework backs up datafiles + restores on error). It crawls per connected component, assigns planes (largest component = 0), writes `x/y/z/plane` to each non-instance room YAML, and asserts `authored == crawled` for Euclidean planes.

**Files:**
- Create: `internal/migration/0.15.0.go`, `internal/migration/0.15.0_test.go`

- [ ] **Step 1: Confirm the framework gating.** Read `internal/migration/migration.go` (the `Run` entrypoint referenced by 0.14.0.go's comment) to see how version migrations are registered + gated as run-once. Register `0.15.0` the same way 0.14.0 is. (If registration is by adding a func to a slice/switch, mirror it exactly.)

- [ ] **Step 2: Write the failing test** against a disposable fixture world. Create `internal/migration/0.15.0_test.go`:

```go
package migration

import (
	"os"
	"path/filepath"
	"testing"
)

// A minimal 3-room line: 1 -e-> 2 -e-> 3, all zone "T". After migration the
// authored coords must equal the crawl (0,0,0),(1,0,0),(2,0,0) on plane 0.
func TestBackfillCoords_LosslessLine(t *testing.T) {
	dir := t.TempDir()
	writeRoom(t, dir, "T", 1, "exits:\n  east:\n    roomid: 2\n")
	writeRoom(t, dir, "T", 2, "exits:\n  west:\n    roomid: 1\n  east:\n    roomid: 3\n")
	writeRoom(t, dir, "T", 3, "exits:\n  west:\n    roomid: 2\n")

	if err := backfillCoordsInDir(dir, false); err != nil {
		t.Fatalf("migration: %v", err)
	}
	assertCoord(t, dir, "T", 1, 0, 0, 0, 0)
	assertCoord(t, dir, "T", 2, 1, 0, 0, 0)
	assertCoord(t, dir, "T", 3, 2, 0, 0, 0)
}
```

(Provide `writeRoom` and `assertCoord` helpers in the same test file: `writeRoom` writes `rooms/<zoneFolder>/<id>.yaml` with `roomid/zone/title/description` + the exits fragment; `assertCoord` re-reads the YAML and checks `x/y/z/plane`. Zone folder = `ZoneNameSanitize` — reuse `rooms.ZoneNameSanitize` or hardcode "t" for the fixture.)

- [ ] **Step 3: Run — expect FAIL**; **Step 4: implement** `internal/migration/0.15.0.go`:

```go
package migration

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// migrate_BackfillCoords writes authored x/y/z/plane onto every non-instance
// room by crawling exit deltas (the pre-authored source of truth) per connected
// component, assigning the largest component plane 0 and each other component a
// fresh plane. It asserts authored == crawled for Euclidean planes so rendered
// positions change for zero rooms. Idempotent: rooms already carrying a nonzero
// coord or an explicit plane are treated as migrated and left untouched.
func migrate_BackfillCoords(dryRun bool) error {
	c := configs.GetConfig()
	return backfillCoordsInDir(filepath.Join(string(c.FilePaths.DataFiles), "rooms"), dryRun)
}

func backfillCoordsInDir(roomsDir string, dryRun bool) error {
	// 1. Load every room's (id, zone, exits) from YAML into an in-memory graph.
	// 2. Compute connected components (BFS over exit RoomIds, cross-zone included).
	// 3. Crawl each component from its lowest room id at (0,0,0), combining the
	//    posDeltas for each exit direction (north y-1, east x+1, up z+1, ...);
	//    first-visit-wins (mirror mapper.Start). Re-base to the component's lowest
	//    room so coords match the mapper's final frame.
	// 4. Assign plane ids: component with the most rooms => 0; others => 1,2,...
	//    (stable order by lowest room id). Log the component->plane table.
	// 5. For each room, write x/y/z/plane back to its YAML (skip if dryRun).
	// 6. Euclidean-plane lossless assert: a room's written coord must equal its
	//    crawled coord (they are the same by construction here — the assert guards
	//    against a bug in the write-back, e.g. a wrong zone-folder path).
	//
	// The direction->delta table MUST match internal/mapper posDeltas exactly.
	// Import it via a small exported helper rooms/mapper accessor, or inline the
	// cardinal+vertical+diagonal+x2/x3 deltas verbatim from mapper.go:41-98.
	mode := "APPLY"
	if dryRun {
		mode = "DRY-RUN"
	}
	mudlog.Info("Migration 0.15.0", "message", "backfilling authored room coordinates", "mode", mode, "dir", roomsDir)

	graph, err := loadRoomGraph(roomsDir) // helper: returns map[id]{zone, path, exits}
	if err != nil {
		return err
	}
	comps := connectedComponents(graph)                 // [][]int of room ids
	coords, planeOf := crawlAndAssignPlanes(graph, comps) // map[id]coord, map[id]plane
	for id, co := range coords {
		if dryRun {
			continue
		}
		if err := writeCoordToYAML(graph[id].path, co, planeOf[id]); err != nil {
			return fmt.Errorf("room %d: %w", id, err)
		}
	}
	mudlog.Info("Migration 0.15.0", "rooms", len(coords), "components", len(comps))
	return nil
}
```

> **Implementer note (real code, not placeholder):** flesh out the six lettered steps as concrete helpers in this file — `loadRoomGraph`, `connectedComponents`, `crawlAndAssignPlanes` (BFS using the verbatim `posDeltas` deltas from `internal/mapper/mapper.go:41-98`; re-base to the component's lowest room like `mapper.Start` lines 322-338), `writeCoordToYAML` (read YAML into `map[string]interface{}`, set `x/y/z/plane`, marshal, `os.WriteFile`, matching the 0.14.0.go read-mutate-write style). Non-instance only: skip files under `instance_*` / ephemeral. Mark a component non-Euclidean (skip the equality assert) when its zone's `zone-config.yaml` has `non_cartesian: true`. Idempotency: skip a room that already has any nonzero `x/y/z` or an explicit `plane`. The migration framework's version gate (Step 1) prevents re-runs across boots; this per-room guard makes a partial re-run safe.

- [ ] **Step 5: Run the fixture test — expect PASS**

Run: `go test ./internal/migration/ -run TestBackfillCoords_LosslessLine`
Expected: PASS (coords match the crawl).

- [ ] **Step 6: Dry-run against the REAL world data** (manual verification, no writes):

Run: `go test ./internal/migration/ -run TestBackfillCoords_LosslessLine -v` then wire a temporary `main`-less dry-run (or a `-run` gated test that points at `_datafiles/world/dogmud/rooms` with `dryRun=true`) and inspect the log: exactly one large plane-0 component + a handful of small/non-Euclidean components; zero lossless-assert failures.
Expected: clean dry-run log; no assert failures.

- [ ] **Step 7: Register + commit** (do NOT run APPLY on the real tree yet — that happens once, reviewed, in Task 8's boot pass or a deliberate one-shot).

```bash
go build ./...
git add internal/migration/0.15.0.go internal/migration/0.15.0_test.go internal/migration/migration.go
git commit -m "feat(migration): 0.15.0 backfill authored room coordinates

Crawl-derived x/y/z + per-component plane assignment, written to room
YAML templates; lossless (authored == crawled) for Euclidean planes.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Mapper reads authored coords

**Files:**
- Modify: `internal/mapper/mapper.go:249` (`Start`); `getMapNode` (`mapper.go:915`) to copy `room.Plane`; `modules/gmcp/gmcp.Zone.go` + `internal/mapper/mapper.snapshot.go` to carry `plane`.

- [ ] **Step 1: Write a mapper-parity test.** In `internal/mapper/mapper_authored_test.go` *(new)*: build a small room set with authored coords that also form a valid crawl, render via `Start`, and assert every node's `Pos` equals its authored coord (i.e. `Start` used stored coords, not a re-crawl). (Use the existing test helpers for constructing rooms; if the mapper reads from `rooms.LoadRoom`, back it with a fixture or the in-memory test rooms the existing consistency tests use.)

- [ ] **Step 2: Run — expect FAIL** (Start still crawls).

- [ ] **Step 3: Rewrite `Start` to prefer authored coords.** Replace the crawl body (`mapper.go:249-339`) so that, for each room reachable from the root, it reads `room.X/Y/Z/Plane` when present (nonzero coord or explicit plane, or a per-room `HasAuthoredCoords` guard) and falls back to the delta-crawl only for un-migrated rooms (defensive; logs a warning). Concretely: keep the BFS traversal (to discover the room set + exits), but set `node.Pos = positionDelta{x: room.X, y: room.Y, z: room.Z}` and `node.Plane = room.Plane` from stored coords instead of `roomNow.Pos.Combine(...)`. Keep the min/max bound tracking off the authored coords to size `RoomGrid` (grids are per plane — if a mapper spans planes, size/index per plane; in practice a zone mapper is one plane). Show the full rewritten `Start` in the commit.

> **Implementer note:** the render must be byte-identical to today for migrated rooms — that is guaranteed because the migration wrote the crawl's own final (re-based) coords. The parity is verified in Step 5. Retain the delta-crawl code path (extract to `startByCrawl()`) as the fallback + as the migration/validation tool.

- [ ] **Step 4: Carry `plane` in the snapshot.** Add `Plane int` to `SnapshotRoom` (`mapper.snapshot.go`) and the `Zone.Map` payload (`gmcp.Zone.go`); populate from `node.Plane`. (Web/ASCII rendering is unaffected in appearance.)

- [ ] **Step 5: Run mapper tests + parity — expect PASS**

Run: `go test ./internal/mapper/`
Expected: PASS.

- [ ] **Step 6: build + commit**

```bash
go build ./...
git add internal/mapper/mapper.go internal/mapper/mapper.snapshot.go modules/gmcp/gmcp.Zone.go internal/mapper/mapper_authored_test.go
git commit -m "feat(mapper): read authored coords (crawl becomes fallback)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: Instance plane assignment

**Files:**
- Modify: `internal/rooms/ephemeral.go` (`CreateEphemeralZone`/`CreateEphemeralRoomIds`), `internal/rooms/instances.go:297`

- [ ] **Step 1: Write the failing test.** In `internal/rooms/instances_test.go`, add a case that creates two instances of the same template and asserts their ephemeral rooms carry **distinct, non-zero plane ids** (so they can't collide with each other or the template). Use the existing instance-test scaffolding (`TestCreateZoneInstanceWithOpts_SuppressReturnPortal` at `instances_test.go:539` shows setup).

- [ ] **Step 2: Run — expect FAIL** (ephemeral rooms inherit template plane).

- [ ] **Step 3: Assign a unique plane per instance.** Add a monotonic allocator:

```go
// instancePlaneBase keeps instance planes well clear of authored world planes
// (which are small ints assigned by the 0.15.0 migration). Each live instance
// gets its own plane so its rooms never collide with the template or siblings.
const instancePlaneBase = 1_000_000

var instancePlaneSeq int // guarded by the same lock as instance creation

func nextInstancePlane() int {
	instancePlaneSeq++
	return instancePlaneBase + instancePlaneSeq
}
```

In `CreateEphemeralZone` (`ephemeral.go:141`), when cloning each template room into an ephemeral room, set `ephemeralRoom.Plane = instancePlane` (the value from `nextInstancePlane()`, allocated once per instance in `CreateZoneInstanceWithOpts` and threaded in). Keep the ephemeral rooms' `X/Y/Z` inherited from the template. Register the instance plane's Euclidean-ness from the template zone: `GetPlaneRegistry().Mark(instancePlane, IsZoneNonCartesian(templateZone), templateZone)`.

- [ ] **Step 4: Un-exclude ephemeral rooms from placement (they now have planes).** `roomCrawlable` (`mapper.consistency.go:97`) can stay as-is for the boot validator (instances don't exist at boot), but `ValidatePlacement`/`occupantAt` naturally handle instance rooms via their unique plane — no change needed since each instance plane is isolated. Add a comment at `roomCrawlable` noting instances are plane-isolated at runtime.

- [ ] **Step 5: Run — expect PASS**; **Step 6: build + commit**

Run: `go test ./internal/rooms/ && go build ./...`
```bash
git add internal/rooms/ephemeral.go internal/rooms/instances.go internal/rooms/instances_test.go internal/mapper/mapper.consistency.go
git commit -m "feat(rooms): assign a unique coordinate plane per live instance

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: Apply migration + full boot verification (REQUIRED)

**Files:** none (runs the migration once + verifies).

- [ ] **Step 1: Full suite + build**

Run: `go build ./... && go test ./internal/rooms/ ./internal/mapper/ ./internal/migration/`
Expected: all green.

- [ ] **Step 2: Apply the 0.15.0 migration to the real world tree** (one-shot, reviewed). Ensure the migration framework's backup ran; then boot once so the version migration executes (or a deliberate one-shot APPLY). Inspect the git diff of `_datafiles/world/dogmud/rooms/` — every non-instance room gains `x/y/z/plane` (plane omitted when 0); spot-check a handful against `docs/coordinate_map.md`.

- [ ] **Step 3: Boot with strict consistency.** Set `GamePlay.MapConsistencyEnforce: panic` in `_datafiles/config.yaml`, wipe instance saves (SOP), and boot:

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run .
```
Expected: clean boot past `ValidateZoneConsistency` with **zero** coordinate errors world-wide (the authoritative check now runs on authored coords + planes). Watch the plane-registry + validator log lines. Kill after "Server Ready".

- [ ] **Step 4: `cartcheck` + map parity spot-check.** Via telnet (admin), run `cartcheck` on a few zones (clean), and eyeball the `map` command for a migrated zone — identical to pre-migration.

- [ ] **Step 5: Instances live check.** Spin up two instances of one template (e.g. enter the crash-site interior or planar oasis twice) — no consistency error; each instance's rooms are plane-isolated.

- [ ] **Step 6: Restore config + commit the migrated data.** Set `MapConsistencyEnforce` back to its prior value (`warn` unless we're promoting to `panic` intentionally). Commit the migrated room YAMLs.

```bash
git add _datafiles/world/dogmud/rooms/ _datafiles/config.yaml
git commit -m "chore(world): apply 0.15.0 authored-coordinate migration

Backfilled x/y/z/plane onto all non-instance rooms; boot-clean under
MapConsistencyEnforce checks. Foundation ready for the builder (1b).

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Self-Review notes

- **Spec coverage:** data model → T1; plane registry + non-Euclidean flag → T2; 4-tuple collision + authored-delta + spatial/portal split → T3; build-time gate (`ValidatePlacement`) → T4; lossless migration → T5; mapper reads authored coords → T6; instance planes → T7; migration apply + boot/cartcheck/parity/instance verification → T8. All spec sections covered.
- **Type consistency:** `mapNode.Plane` (T3) is read by `findCollisions`/`longSpanCrossesRoom` (T3) and populated by `getMapNode` (T3) + `Start` (T6). `GetPlaneRegistry()`/`IsNonEuclidean` (T2) are consumed by T3, T4, T7. `ValidatePlacement` signature `(plane,x,y,z,excludeRoomId int) error` (T4) is the seam 1b calls. `nextInstancePlane()` (T7) feeds `mapNode.Plane` via `Room.Plane`.
- **Ordering:** T1→T2→T3→T4→T5→T6→T7 build linearly; T8 (apply + boot) runs last. T6 (mapper reads coords) depends on T5 having written them — but T6's fallback-to-crawl keeps the build green even before the real-tree APPLY (which is T8). Tests use fixtures, so no task requires the real migration to have run.
- **Escape hatches:** `oneway` unchanged (reciprocity-only); `non_cartesian` repurposed via the plane registry (T2/T3); the 7 special zones become non-Euclidean planes and skip grid enforcement.
- **Risk note:** T6 (mapper) + T8 (apply) are the highest-risk. The lossless `authored == crawled` design + the map-parity spot-check are the guardrails; if parity fails for any zone, stop and reconcile before committing migrated data.
```
