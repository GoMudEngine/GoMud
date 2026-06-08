# Player Web-Client Mapper — Hybrid Style + Cartesian Consistency — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild the player web-client minimap into a clean, zone-scoped *hybrid* map (line-art + subtle biome tint + faint glyph, with edge-stub wrap rendering and up/down ticks), and add a phased Cartesian-consistency engine that protects the coordinate data the map depends on.

**Architecture:** Two shippable milestones. **Part 1 (server-only)** adds an exit-`kind` classifier plus collision / reciprocity / delta-mismatch checks built on the existing mapper crawl, exposed through a `cartcheck` admin command and a startup validation pass gated by a config knob (warn→panic). It also adds the `oneway` exit flag and `non_cartesian` zone flag. **Part 2** adds a GMCP "explored zone" snapshot (reusing the Part 1 classifier) plus per-character visited-room memory, and restyles the client `RoomGridSVG` renderer into the hybrid look with long/wrap/vertical handling. Part 1 ships and provides value (the consistency report) before Part 2 starts.

**Tech Stack:** Go (server, `internal/mapper`, `internal/rooms`, `internal/exit`, `internal/configs`, `internal/usercommands`, `modules/gmcp`), vanilla JS + SVG (client, `_datafiles/html/public`), YAML data files, `mudlog` (slog) logging, Go `testing` for unit tests.

**Spec:** `docs/superpowers/specs/2026-06-06-player-mapper-hybrid-design.md`

---

## Conventions for the implementing engineer (read first)

- **You know nothing about this codebase.** Follow file paths exactly.
- **Handler signature** for every usercommand: `func Name(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error)`.
- **Send text to a player:** `user.SendText(messaging.CategorySystem, "text")`. Never print raw numbers in *player-facing* combat/spell text (not relevant here — `cartcheck` is an admin tool, raw numbers are fine).
- **Logging:** `mudlog.Info/Warn/Error(msg string, kvPairs ...any)` — structured key/value. Hard failures at data-load use bare `panic(fmt.Sprintf("pkg.Func() context: %v", err))`.
- **`positionDelta` gotcha:** the struct is `{x,y,z int; arrow rune}`. NEVER compare two `positionDelta` with `==` — the `arrow` differs. Always compare `x,y,z` componentwise (a `samePos` helper is provided in Task 4).
- **Tests:** `go test ./internal/<pkg>/... -run TestName -v`. White-box tests (same package) may touch unexported fields like `positionDelta.x`.
- **Commit** after each task's tests pass. Conventional commits (`feat:`, `fix:`, `docs:`, `test:`). Co-author trailer: `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`.
- **Build check:** `go build ./...` must pass before any commit.
- **Smoke SOP:** before in-game smoke testing, wipe stale instance saves: `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*` (do NOT touch `shops/`).

---

## File Structure

**Part 1 — Consistency engine (server only)**

| File | Create/Modify | Responsibility |
|---|---|---|
| `internal/exit/exit.go` | Modify | Add `OneWay bool` to `RoomExit`. |
| `internal/rooms/zoneconfig.go` | Modify | Add `NonCartesian bool` to `ZoneConfig`. |
| `internal/configs/config.gameplay.go` | Modify | Add `MapConsistencyEnforce` knob + default. |
| `_datafiles/config.yaml` | Modify | Document the new knob under `GamePlay:`. |
| `internal/mapper/mapper.node.go` | Modify | Add `OneWay bool` to `nodeExit`. |
| `internal/mapper/mapper.go` | Modify | Populate `nodeExit.OneWay` in `getMapNode`; call new validator in `PreCacheMaps`. |
| `internal/mapper/mapper.consistency.go` | Create | Exit-`kind` classifier + collision/reciprocity/delta/long-crossing checks + `CheckConsistency` + startup `ValidateZoneConsistency`. |
| `internal/mapper/mapper.consistency_test.go` | Create | White-box unit tests for the above. |
| `internal/usercommands/admin.cartcheck.go` | Create | `cartcheck [zone]` admin command. |
| `internal/usercommands/usercommands.go` | Modify | Register `cartcheck`. |
| `_datafiles/world/default/templates/admincommands/help/command.cartcheck.template` | Create | Helpfile. |

**Part 2 — Hybrid rendering (GMCP + client)**

| File | Create/Modify | Responsibility |
|---|---|---|
| `internal/characters/character.go` | Modify | `VisitedRooms map[string][]int` + `MarkRoomVisited` / `HasVisitedRoom`. |
| `internal/characters/character_visited_test.go` | Create | Tests for visited-room memory. |
| `internal/usercommands/go.go` | Modify | Mark current room visited after a successful move. |
| `internal/mapper/mapper.snapshot.go` | Create | `Snapshot(visited)` → classified rooms/exits for the client. |
| `internal/mapper/mapper.snapshot_test.go` | Create | Tests for the snapshot builder. |
| `modules/gmcp/gmcp.Room.go` | Modify | Add `symbol` field to room payload. |
| `modules/gmcp/gmcp.Zone.go` | Create | `gmcp.Zone` module → emits `Zone.Map` snapshot on zone entry/connect. |
| `_datafiles/html/public/static/js/gmcp.js` | Modify | Hybrid restyle of `RoomGridSVG`; biome→tint; glyphs; long/wrap/vertical rendering; fit/center. |
| `_datafiles/html/public/webclient-pure.html` | Modify | Subscribe to `Zone.Map`; pass new options to `RoomGridSVG`. |

**Docs (woven into tasks; never deferred)**

| File | Responsibility |
|---|---|
| `docs/schemas/room.md` | Document `oneway` exit field + `non_cartesian` zone-config field. |
| `docs/coordinate_map.md` | Document consistency engine, exit kinds, wrap rendering, the snapshot, the knob. |
| `internal/exit/context.md` | Note `OneWay`. |
| `internal/mapper/context.md` | Note consistency engine + snapshot + classifier. |
| `CLAUDE.md` | New "Map Consistency & Web Mapper" section. |
| `PATCH_NOTES.md` | Dated entries (one per part) at the end of each part. |

---

# PART 1 — Cartesian Consistency Engine (server, shippable)

### Task 1: Add the `oneway` exit flag

**Files:**
- Modify: `internal/exit/exit.go`
- Modify: `internal/mapper/mapper.node.go`
- Modify: `internal/mapper/mapper.go` (`getMapNode`)
- Modify: `docs/schemas/room.md`, `internal/exit/context.md`

- [ ] **Step 1: Add `OneWay` to `RoomExit`.** In `internal/exit/exit.go`, inside the `RoomExit` struct, add the field after `Lock`:

```go
type RoomExit struct {
	RoomId       int
	Secret       bool          `yaml:"secret,omitempty"`
	MapDirection string        `yaml:"mapdirection,omitempty"` // Optionally indicate the direction of this exit for mapping purposes
	ExitMessage  string        `yaml:"exitmessage,omitempty"`  // If set, this message is sent to the user, followed by a delay, before they actually go through the exit.
	Lock         gamelock.Lock `yaml:"lock,omitempty"`         // 0 - no lock. greater than zero = difficulty to unlock.
	OneWay       bool          `yaml:"oneway,omitempty"`       // true = intentional one-way spatial exit; skips the reciprocity consistency check.
}
```

`RoomExit` has no custom `UnmarshalYAML`; the field parses automatically from `oneway: true` under an exit block. No loader change needed.

- [ ] **Step 2: Add `OneWay` to `nodeExit`.** In `internal/mapper/mapper.node.go`:

```go
type nodeExit struct {
	RoomId         int    // where it leads to
	Secret         bool   // is it secret?
	LockDifficulty int    // If > 0, the lock difficulty.
	LockId         string // What's the lock id?
	OneWay         bool   // intentional one-way spatial exit (skips reciprocity check)
	Direction      positionDelta
}
```

- [ ] **Step 3: Populate `OneWay` in `getMapNode`.** In `internal/mapper/mapper.go`, in `getMapNode`, set it where the `nodeExit` is constructed:

```go
		exitNode := nodeExit{
			RoomId:         exitInfo.RoomId,
			Secret:         exitInfo.Secret,
			LockDifficulty: int(exitInfo.Lock.Difficulty),
			OneWay:         exitInfo.OneWay,
		}
```

- [ ] **Step 4: Build.** Run: `go build ./...` — Expected: success, no errors.

- [ ] **Step 5: Document.** In `docs/schemas/room.md`, find the exits section and add a row/paragraph: `oneway: <bool>` — "Marks an intentional one-way spatial exit (e.g. a slippery slope). The mapper still checks the destination doesn't collide, but skips the requirement for a reciprocal return exit. Only needed for compass-direction exits; portal/named exits are non-spatial and already exempt." In `internal/exit/context.md`, add `OneWay` to the `RoomExit` field list with the same one-line description.

- [ ] **Step 6: Commit.**

```bash
git add internal/exit/exit.go internal/mapper/mapper.node.go internal/mapper/mapper.go docs/schemas/room.md internal/exit/context.md
git commit -m "feat(exit): add oneway flag for intentional one-way spatial exits"
```

---

### Task 2: Add the `non_cartesian` zone flag

**Files:**
- Modify: `internal/rooms/zoneconfig.go`
- Modify: `docs/schemas/room.md`

- [ ] **Step 1: Add the field.** In `internal/rooms/zoneconfig.go`, add `NonCartesian` immediately before `RoomIds`:

```go
	EntryRoom      int                  `yaml:"entry_room,omitempty"`       // room ID where portal drops players
	AllowRecall    bool                 `yaml:"allow_recall,omitempty"`     // whether recall works inside (default true)
	NonCartesian   bool                 `yaml:"non_cartesian,omitempty"`    // true = zone is intentionally non-Euclidean (toroidal/maze); skip Cartesian consistency checks and render wrap exits as edge stubs
	RoomIds        map[int]struct{}     `yaml:"-"`                          // Does not get written. Built dynamically when rooms are loaded.
```

`Validate()` and `NewZoneConfig()` need no change — `false` is the correct zero value.

- [ ] **Step 2: Build.** Run: `go build ./...` — Expected: success.

- [ ] **Step 3: Document.** In `docs/schemas/room.md`, in the zone-config section, add: `non_cartesian: <bool>` — "Marks a zone whose exit graph is intentionally non-Euclidean (e.g. a toroidal zone where edge exits wrap to the opposite edge, or a deliberate maze). Exempts the zone from the collision + reciprocity consistency checks, and switches the web mapper to render wrap exits as edge stubs instead of long connectors."

- [ ] **Step 4: Commit.**

```bash
git add internal/rooms/zoneconfig.go docs/schemas/room.md
git commit -m "feat(rooms): add non_cartesian zone flag"
```

---

### Task 3: Add the `MapConsistencyEnforce` config knob

**Files:**
- Modify: `internal/configs/config.gameplay.go`
- Modify: `_datafiles/config.yaml`

- [ ] **Step 1: Add the field.** In `internal/configs/config.gameplay.go`, add to the `GamePlay` struct:

```go
type GamePlay struct {
	AllowItemBuffRemoval  ConfigBool    `yaml:"AllowItemBuffRemoval"`
	Death                 GameplayDeath `yaml:"Death"`
	ShopRestockRate       ConfigString  `yaml:"ShopRestockRate"`
	ContainerSizeMax      ConfigInt     `yaml:"ContainerSizeMax"`
	MaxAltCharacters      ConfigInt     `yaml:"MaxAltCharacters"`
	PVP                   ConfigString  `yaml:"PVP"`
	PVPMinimumSkillRanks  ConfigInt     `yaml:"PVPMinimumSkillRanks"`
	UseSkillProgression   ConfigBool    `yaml:"UseSkillProgression"`
	DualProgressionMode   ConfigBool    `yaml:"DualProgressionMode"`
	MapConsistencyEnforce ConfigString  `yaml:"MapConsistencyEnforce"` // "off" | "warn" (default) | "panic" — startup Cartesian-consistency enforcement level
}
```

- [ ] **Step 2: Default it.** In the same file, inside `func (g *GamePlay) Validate()`, add (near the other defaults):

```go
	switch string(g.MapConsistencyEnforce) {
	case "off", "warn", "panic":
		// valid
	default:
		g.MapConsistencyEnforce = "warn"
	}
```

- [ ] **Step 3: Build.** Run: `go build ./...` — Expected: success.

- [ ] **Step 4: Document in config.yaml.** In `_datafiles/config.yaml`, under the `GamePlay:` section, add:

```yaml
  # Cartesian map consistency enforcement at startup:
  #   off   = no checking
  #   warn  = log collisions / non-reciprocal exits, server still boots (default)
  #   panic = block server boot on any inconsistency
  # Toroidal/maze zones must set `non_cartesian: true` in their zone-config.yaml
  # to be exempt; one-way exits use `oneway: true` on the exit.
  MapConsistencyEnforce: warn
```

- [ ] **Step 5: Commit.**

```bash
git add internal/configs/config.gameplay.go _datafiles/config.yaml
git commit -m "feat(configs): add MapConsistencyEnforce knob (off|warn|panic)"
```

---

### Task 4: Exit-kind classifier + pure consistency checks

**Files:**
- Create: `internal/mapper/mapper.consistency.go`
- Create: `internal/mapper/mapper.consistency_test.go`

Context: after `mapper.Start()`, `r.crawledRooms map[int]*mapNode` holds every room with its normalized `Pos`. `RoomGrid` silently overwrites on a coordinate collision, so collisions are detected by scanning `crawledRooms`, not the grid.

- [ ] **Step 1: Write the failing test.** Create `internal/mapper/mapper.consistency_test.go`:

```go
package mapper

import "testing"

func d(x, y, z int) positionDelta { return positionDelta{x: x, y: y, z: z} }

func TestSamePos(t *testing.T) {
	if !samePos(positionDelta{1, 2, 3, '│'}, positionDelta{1, 2, 3, ' '}) {
		t.Fatal("samePos must ignore the arrow field")
	}
	if samePos(d(1, 0, 0), d(0, 1, 0)) {
		t.Fatal("different coords must not be samePos")
	}
}

func TestClassifyKind(t *testing.T) {
	cases := []struct {
		name             string
		nominal, actual  positionDelta
		want             ExitKind
	}{
		{"normal", d(0, -1, 0), d(0, -1, 0), ExitNormal},
		{"long-x3", d(0, -3, 0), d(0, -3, 0), ExitLong},
		{"vertical-up", d(0, 0, 1), d(0, 0, 1), ExitVertical},
		{"wrap", d(0, 1, 0), d(0, -4, 0), ExitWrap},
	}
	for _, c := range cases {
		if got := classifyKind(c.nominal, c.actual); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

func TestFindCollisions(t *testing.T) {
	nodes := map[int]*mapNode{
		1: {RoomId: 1, Pos: d(0, 0, 0)},
		2: {RoomId: 2, Pos: d(1, 0, 0)},
		3: {RoomId: 3, Pos: d(1, 0, 0)}, // collides with 2
	}
	groups := findCollisions(nodes)
	if len(groups) != 1 || len(groups[0]) != 2 {
		t.Fatalf("expected one collision group of 2, got %v", groups)
	}
}
```

- [ ] **Step 2: Run it — verify it fails.** Run: `go test ./internal/mapper/ -run 'TestSamePos|TestClassifyKind|TestFindCollisions' -v` — Expected: FAIL (undefined: `samePos`, `classifyKind`, `ExitKind`, `findCollisions`).

- [ ] **Step 3: Implement the classifier + collision finder.** Create `internal/mapper/mapper.consistency.go`:

```go
package mapper

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// ExitKind classifies a spatial exit by comparing its nominal compass delta to
// the actual placed coordinate delta between the two crawled rooms.
type ExitKind string

const (
	ExitNormal   ExitKind = "normal"   // unit cardinal/diagonal, placed exactly as nominal
	ExitLong     ExitKind = "long"     // multi-cell (-x2/-x3), placed exactly as nominal
	ExitVertical ExitKind = "vertical" // up/down (dz != 0)
	ExitWrap     ExitKind = "wrap"     // actual delta != nominal (toroidal/torn edge)
)

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// samePos compares only x/y/z (NOT arrow — positionDelta.arrow differs by direction).
func samePos(a, b positionDelta) bool {
	return a.x == b.x && a.y == b.y && a.z == b.z
}

// classifyKind decides the render/consistency kind from nominal vs actual delta.
func classifyKind(nominal, actual positionDelta) ExitKind {
	if !samePos(nominal, actual) {
		return ExitWrap
	}
	if nominal.z != 0 {
		return ExitVertical
	}
	if absInt(nominal.x) > 1 || absInt(nominal.y) > 1 {
		return ExitLong
	}
	return ExitNormal
}

// findCollisions returns groups of >=2 roomIds that share the same (x,y,z).
func findCollisions(nodes map[int]*mapNode) [][]int {
	byCell := map[[3]int][]int{}
	for id, n := range nodes {
		key := [3]int{n.Pos.x, n.Pos.y, n.Pos.z}
		byCell[key] = append(byCell[key], id)
	}
	groups := [][]int{}
	for _, ids := range byCell {
		if len(ids) >= 2 {
			sort.Ints(ids)
			groups = append(groups, ids)
		}
	}
	return groups
}

// Finding is one consistency problem.
type Finding struct {
	Severity string // "error" (collision/reciprocity/delta) | "warn" (long-crossing)
	Kind     string // "collision" | "noreciprocal" | "deltamismatch" | "longcrossing"
	Zone     string
	RoomId   int
	ExitName string
	Detail   string
}

func (f Finding) String() string {
	loc := fmt.Sprintf("zone=%s room=%d", f.Zone, f.RoomId)
	if f.ExitName != "" {
		loc += " exit=" + f.ExitName
	}
	return fmt.Sprintf("[%s] %s: %s (%s)", f.Severity, f.Kind, f.Detail, loc)
}

// hasReturnExit reports whether dst has any spatial exit leading back to srcId.
func hasReturnExit(dst *mapNode, srcId int) bool {
	for _, e := range dst.Exits {
		if e.RoomId == srcId {
			return true
		}
	}
	return false
}

// roomVisibleToCrawl skips ephemeral/instance rooms (they don't exist at boot;
// guards the live cartcheck command).
func roomCrawlable(roomId int) bool {
	return !rooms.IsEphemeralRoomId(roomId)
}
```

- [ ] **Step 4: Run the tests — verify pass.** Run: `go test ./internal/mapper/ -run 'TestSamePos|TestClassifyKind|TestFindCollisions' -v` — Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/mapper/mapper.consistency.go internal/mapper/mapper.consistency_test.go
git commit -m "feat(mapper): exit-kind classifier + collision finder (pure)"
```

---

### Task 5: `CheckConsistency` aggregation

**Files:**
- Modify: `internal/mapper/mapper.consistency.go`
- Modify: `internal/mapper/mapper.consistency_test.go`

- [ ] **Step 1: Write the failing test.** Append to `internal/mapper/mapper.consistency_test.go`:

```go
// helper: build a minimal *mapper with hand-placed nodes (no crawl).
func mkMapper(nodes map[int]*mapNode) *mapper {
	return &mapper{crawledRooms: nodes}
}

func node(id, x, y, z int, exits map[string]nodeExit) *mapNode {
	return &mapNode{RoomId: id, Pos: d(x, y, z), Exits: exits}
}

func TestCheckConsistency_CleanGrid(t *testing.T) {
	// 1 --north--> 2, 2 --south--> 1 (reciprocal, unit)
	nodes := map[int]*mapNode{
		1: node(1, 0, 0, 0, map[string]nodeExit{"north": {RoomId: 2, Direction: d(0, -1, 0)}}),
		2: node(2, 0, -1, 0, map[string]nodeExit{"south": {RoomId: 1, Direction: d(0, 1, 0)}}),
	}
	if f := mkMapper(nodes).CheckConsistency("test", false); len(f) != 0 {
		t.Fatalf("clean grid should yield no findings, got %v", f)
	}
}

func TestCheckConsistency_MissingReciprocal(t *testing.T) {
	nodes := map[int]*mapNode{
		1: node(1, 0, 0, 0, map[string]nodeExit{"north": {RoomId: 2, Direction: d(0, -1, 0)}}),
		2: node(2, 0, -1, 0, map[string]nodeExit{}), // no way back
	}
	f := mkMapper(nodes).CheckConsistency("test", false)
	if len(f) != 1 || f[0].Kind != "noreciprocal" {
		t.Fatalf("expected one noreciprocal finding, got %v", f)
	}
}

func TestCheckConsistency_OnewaySuppressesReciprocal(t *testing.T) {
	nodes := map[int]*mapNode{
		1: node(1, 0, 0, 0, map[string]nodeExit{"north": {RoomId: 2, Direction: d(0, -1, 0), OneWay: true}}),
		2: node(2, 0, -1, 0, map[string]nodeExit{}),
	}
	if f := mkMapper(nodes).CheckConsistency("test", false); len(f) != 0 {
		t.Fatalf("oneway should suppress reciprocity finding, got %v", f)
	}
}

func TestCheckConsistency_WrapFlaggedInCartesian(t *testing.T) {
	// south (nominal +1y) actually lands at +4y (opposite edge): wrap.
	nodes := map[int]*mapNode{
		1: node(1, 0, -2, 0, map[string]nodeExit{"south": {RoomId: 2, Direction: d(0, 1, 0)}}),
		2: node(2, 0, 2, 0, map[string]nodeExit{"north": {RoomId: 1, Direction: d(0, -1, 0)}}),
	}
	f := mkMapper(nodes).CheckConsistency("test", false)
	if len(f) == 0 {
		t.Fatal("wrap in a Cartesian zone must be flagged")
	}
	saw := false
	for _, x := range f {
		if x.Kind == "deltamismatch" {
			saw = true
		}
	}
	if !saw {
		t.Fatalf("expected a deltamismatch finding, got %v", f)
	}
}

func TestCheckConsistency_WrapAllowedInNonCartesian(t *testing.T) {
	nodes := map[int]*mapNode{
		1: node(1, 0, -2, 0, map[string]nodeExit{"south": {RoomId: 2, Direction: d(0, 1, 0)}}),
		2: node(2, 0, 2, 0, map[string]nodeExit{"north": {RoomId: 1, Direction: d(0, -1, 0)}}),
	}
	if f := mkMapper(nodes).CheckConsistency("test", true); len(f) != 0 {
		t.Fatalf("non_cartesian zone should suppress wrap/reciprocity findings, got %v", f)
	}
}

func TestCheckConsistency_Collision(t *testing.T) {
	nodes := map[int]*mapNode{
		1: node(1, 0, 0, 0, map[string]nodeExit{}),
		2: node(2, 0, 0, 0, map[string]nodeExit{}), // same cell
	}
	f := mkMapper(nodes).CheckConsistency("test", false)
	if len(f) != 1 || f[0].Kind != "collision" {
		t.Fatalf("expected one collision finding, got %v", f)
	}
}
```

- [ ] **Step 2: Run it — verify it fails.** Run: `go test ./internal/mapper/ -run TestCheckConsistency -v` — Expected: FAIL (undefined: `CheckConsistency`).

- [ ] **Step 3: Implement.** Append to `internal/mapper/mapper.consistency.go`:

```go
// CheckConsistency walks the crawled rooms of this mapper and returns findings.
// nonCartesian=true (zone flag) suppresses collision/reciprocity/deltamismatch
// (the zone is intentionally non-Euclidean); the long-crossing warning still runs.
func (r *mapper) CheckConsistency(zone string, nonCartesian bool) []Finding {
	findings := []Finding{}

	if !nonCartesian {
		for _, group := range findCollisions(r.crawledRooms) {
			// group is sorted; report against the lowest roomId.
			findings = append(findings, Finding{
				Severity: "error", Kind: "collision", Zone: zone, RoomId: group[0],
				Detail:   fmt.Sprintf("rooms %v occupy the same coordinate", group),
			})
		}
	}

	for srcId, src := range r.crawledRooms {
		if !roomCrawlable(srcId) {
			continue
		}
		for exitName, e := range src.Exits {
			dst, ok := r.crawledRooms[e.RoomId]
			if !ok {
				continue // cross-zone or uncrawled — not part of this coordinate space
			}
			actual := positionDelta{x: dst.Pos.x - src.Pos.x, y: dst.Pos.y - src.Pos.y, z: dst.Pos.z - src.Pos.z}

			if !nonCartesian {
				if !samePos(e.Direction, actual) {
					findings = append(findings, Finding{
						Severity: "error", Kind: "deltamismatch", Zone: zone, RoomId: srcId, ExitName: exitName,
						Detail: fmt.Sprintf("nominal delta (%d,%d,%d) != actual (%d,%d,%d) — wrap exit in a Cartesian zone (set non_cartesian or fix geometry)",
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

			// Soft, always-on: long exit whose straight span crosses an occupied cell.
			if samePos(e.Direction, actual) && (absInt(actual.x) > 1 || absInt(actual.y) > 1) {
				if crossed := r.longSpanCrossesRoom(src.Pos, actual, srcId, e.RoomId); crossed != 0 {
					findings = append(findings, Finding{
						Severity: "warn", Kind: "longcrossing", Zone: zone, RoomId: srcId, ExitName: exitName,
						Detail: fmt.Sprintf("long exit connector passes over room %d", crossed),
					})
				}
			}
		}
	}
	return findings
}

// longSpanCrossesRoom returns the roomId of an intervening occupied cell on the
// straight line from start by delta (exclusive of endpoints), or 0 if none.
// Only handles axis-aligned and pure-diagonal spans (the only shapes posDeltas produce).
func (r *mapper) longSpanCrossesRoom(start, delta positionDelta, srcId, dstId int) int {
	steps := absInt(delta.x)
	if absInt(delta.y) > steps {
		steps = absInt(delta.y)
	}
	if steps <= 1 {
		return 0
	}
	sx, sy := sign(delta.x), sign(delta.y)
	byCell := map[[3]int]int{}
	for id, n := range r.crawledRooms {
		byCell[[3]int{n.Pos.x, n.Pos.y, n.Pos.z}] = id
	}
	for i := 1; i < steps; i++ {
		cell := [3]int{start.x + sx*i, start.y + sy*i, start.z}
		if id, ok := byCell[cell]; ok && id != srcId && id != dstId {
			return id
		}
	}
	return 0
}

func sign(v int) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}
```

- [ ] **Step 4: Run the tests — verify pass.** Run: `go test ./internal/mapper/ -run TestCheckConsistency -v` — Expected: PASS (all 6 subtests).

- [ ] **Step 5: Run the whole mapper package.** Run: `go test ./internal/mapper/... -v` — Expected: PASS (existing tests + new).

- [ ] **Step 6: Commit.**

```bash
git add internal/mapper/mapper.consistency.go internal/mapper/mapper.consistency_test.go
git commit -m "feat(mapper): CheckConsistency — collision/reciprocity/wrap/long-crossing"
```

---

### Task 6: Startup validation wiring (warn|panic)

**Files:**
- Modify: `internal/mapper/mapper.consistency.go`
- Modify: `internal/mapper/mapper.go` (`PreCacheMaps`)

`PreCacheMaps()` (mapper.go ~994) already calls `validateRoomBiomes()` and then builds every zone mapper. We add a zone-consistency pass after the mappers are built.

- [ ] **Step 1: Implement the startup pass.** Append to `internal/mapper/mapper.consistency.go`:

```go
import (
	// add to the existing import block:
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// ValidateZoneConsistency runs CheckConsistency for every loaded zone and either
// warns or panics per the MapConsistencyEnforce config knob. Called from PreCacheMaps
// AFTER all zone mappers are built. non_cartesian zones are passed their flag.
func ValidateZoneConsistency() {
	mode := string(configs.GetGamePlayConfig().MapConsistencyEnforce)
	if mode == "off" {
		return
	}

	errorCount, warnCount := 0, 0
	var firstError string

	for _, zoneName := range rooms.GetAllZoneNames() {
		rootRoomId, _ := rooms.GetZoneRoot(zoneName)
		m := GetMapperIfExists(rootRoomId)
		if m == nil {
			continue
		}
		nonCartesian := rooms.IsZoneNonCartesian(zoneName)
		for _, f := range m.CheckConsistency(zoneName, nonCartesian) {
			if f.Severity == "error" {
				errorCount++
				if firstError == "" {
					firstError = f.String()
				}
				mudlog.Warn("mapper.ValidateZoneConsistency", "finding", f.String())
			} else {
				warnCount++
				mudlog.Warn("mapper.ValidateZoneConsistency", "finding", f.String())
			}
		}
	}

	mudlog.Info("mapper.ValidateZoneConsistency", "errors", errorCount, "warnings", warnCount, "mode", mode)

	if mode == "panic" && errorCount > 0 {
		panic(fmt.Sprintf("mapper.ValidateZoneConsistency: %d consistency error(s); first: %s", errorCount, firstError))
	}
}
```

- [ ] **Step 2: Add the `IsZoneNonCartesian` helper.** In `internal/rooms/zoneconfig.go` (or wherever zone configs are looked up — mirror `GetZoneBiome`), add a lookup that returns the `NonCartesian` flag for a zone name. Search for the existing `GetZoneBiome` function and add alongside it:

```go
// IsZoneNonCartesian reports the non_cartesian flag for a zone (false if unknown).
func IsZoneNonCartesian(zoneName string) bool {
	if z := getZoneConfig(zoneName); z != nil {
		return z.NonCartesian
	}
	return false
}
```

If the internal accessor is named differently than `getZoneConfig`, use whatever `GetZoneBiome` uses to resolve a `*ZoneConfig` by name. (Confirm by reading the `GetZoneBiome` implementation.)

- [ ] **Step 3: Wire into `PreCacheMaps`.** In `internal/mapper/mapper.go`, at the END of `PreCacheMaps()` (after both `GetMapper` loops finish so every zone is built), add:

```go
	ValidateZoneConsistency()
```

- [ ] **Step 4: Build + boot test.** Run: `go build ./...` — Expected: success. Then boot the server locally and watch the log for the `mapper.ValidateZoneConsistency` summary line (errors/warnings counts) with no panic (default mode is `warn`):

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . 2>&1 | grep -i "ValidateZoneConsistency"
```

Expected: an Info summary line; any `error`-severity findings are real inconsistencies in the live world to triage (record them — they must be fixed or flagged `oneway`/`non_cartesian` before Phase 2 flips the knob to `panic`).

- [ ] **Step 5: Commit.**

```bash
git add internal/mapper/mapper.consistency.go internal/mapper/mapper.go internal/rooms/zoneconfig.go
git commit -m "feat(mapper): startup zone-consistency pass gated by MapConsistencyEnforce"
```

---

### Task 7: `cartcheck` admin command

**Files:**
- Create: `internal/usercommands/admin.cartcheck.go`
- Modify: `internal/usercommands/usercommands.go`
- Create: `_datafiles/world/default/templates/admincommands/help/command.cartcheck.template`

- [ ] **Step 1: Create the handler.** Create `internal/usercommands/admin.cartcheck.go`:

```go
package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* cartcheck 				(Admin)
 */
func CartCheck(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Determine which zones to check: argument zone name, or all zones.
	var zoneNames []string
	rest = strings.TrimSpace(rest)
	if rest != "" {
		zoneNames = []string{rest}
	} else {
		zoneNames = rooms.GetAllZoneNames()
	}

	totalErr, totalWarn := 0, 0
	var lines []string

	for _, zoneName := range zoneNames {
		rootRoomId, err := rooms.GetZoneRoot(zoneName)
		if err != nil {
			continue
		}
		m := mapper.GetMapper(rootRoomId)
		if m == nil {
			continue
		}
		findings := m.CheckConsistency(zoneName, rooms.IsZoneNonCartesian(zoneName))
		if len(findings) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf(`<ansi fg="yellow">%s</ansi>`, zoneName))
		for _, f := range findings {
			if f.Severity == "error" {
				totalErr++
			} else {
				totalWarn++
			}
			lines = append(lines, "  "+f.String())
		}
	}

	if len(lines) == 0 {
		user.SendText(messaging.CategorySystem, "Cartesian consistency: no findings. All checked zones are clean.")
		return true, nil
	}

	out := strings.Join(lines, "\n")
	out += fmt.Sprintf("\n\n%d error(s), %d warning(s).", totalErr, totalWarn)
	user.SendText(messaging.CategorySystem, out)
	return true, nil
}
```

- [ ] **Step 2: Register it.** In `internal/usercommands/usercommands.go`, add to the `userCommands` map (keep alphabetical placement near other `c` commands; the four bools are `{Func, AllowedWhenDowned, AllowedInCombat, AdminOnly}`):

```go
	`cartcheck`:       {CartCheck, true, true, true}, // Admin only
```

- [ ] **Step 3: Create the helpfile.** Create `_datafiles/world/default/templates/admincommands/help/command.cartcheck.template` (hard-wrap at 80; no trailing blank line):

```
The <ansi fg="command">cartcheck</ansi> command reports Cartesian map
consistency problems (coordinate collisions, non-reciprocal exits, wrap
exits in non-wrap zones, and long connectors crossing rooms).

<ansi fg="command">cartcheck</ansi> - Check every zone.
<ansi fg="command">cartcheck [zone name]</ansi> - Check a single zone.

Fixes: add <ansi fg="yellow">oneway: true</ansi> to an intentional one-way
exit, or <ansi fg="yellow">non_cartesian: true</ansi> to a zone-config.yaml
for an intentionally toroidal/maze zone.
```

- [ ] **Step 4: Build.** Run: `go build ./...` — Expected: success.

- [ ] **Step 5: Smoke.** Boot the server, log in as an admin, run `cartcheck` and `cartcheck Instance Planar Oasis`. Expected: a readable report (or "no findings"); the help renders via `help cartcheck`.

- [ ] **Step 6: Commit.**

```bash
git add internal/usercommands/admin.cartcheck.go internal/usercommands/usercommands.go _datafiles/world/default/templates/admincommands/help/command.cartcheck.template
git commit -m "feat(usercommands): cartcheck admin command for map consistency report"
```

---

### Task 8: Part 1 documentation

**Files:**
- Modify: `internal/mapper/context.md`, `docs/coordinate_map.md`, `CLAUDE.md`, `PATCH_NOTES.md`

- [ ] **Step 1: Update `internal/mapper/context.md`.** Add a section documenting: the exit-`kind` classifier (`classifyKind`), `CheckConsistency`, the four finding kinds, `ValidateZoneConsistency` (called from `PreCacheMaps`), and the `MapConsistencyEnforce` knob. Note the `positionDelta.arrow` comparison gotcha and that collisions are found by scanning `crawledRooms` (not the silently-overwriting `RoomGrid`).

- [ ] **Step 2: Update `docs/coordinate_map.md`.** Add a "Cartesian Consistency" section: the two hard checks (collision, reciprocity) + delta-mismatch/wrap detection + the long-crossing soft warning; the exemptions (non-spatial exits via the mapdirection→name→skip rule, ephemeral rooms, cross-zone edges, `oneway` exits, `non_cartesian` zones); the `cartcheck` command; and the phased rollout (`warn`→`panic`).

- [ ] **Step 3: Add a `CLAUDE.md` section.** After an existing subsystem section, add:

```markdown
## Map Consistency & the `non_cartesian` / `oneway` Flags
The web mapper places rooms by crawling exit deltas (`internal/mapper`),
so the world must stay Cartesian-consistent. A startup pass
(`ValidateZoneConsistency`, gated by `GamePlay.MapConsistencyEnforce`:
`off|warn|panic`, default `warn`) and the `cartcheck [zone]` admin command
report coordinate collisions, non-reciprocal exits, wrap exits in
non-wrap zones, and long connectors crossing rooms.

Escape hatches: `oneway: true` on an exit (intentional one-way; skips the
reciprocity check, still collision-checked) and `non_cartesian: true` in a
zone's `zone-config.yaml` (intentionally toroidal/maze; skips the hard
checks and renders wrap exits as edge stubs). Portal/named (non-compass)
exits are automatically non-spatial and exempt. Flip the knob to `panic`
only after `cartcheck` is clean world-wide.
```

- [ ] **Step 4: Update `PATCH_NOTES.md`.** Add a dated entry for the consistency engine (cartcheck, the two flags, the knob, warn-mode default).

- [ ] **Step 5: Commit.**

```bash
git add internal/mapper/context.md docs/coordinate_map.md CLAUDE.md PATCH_NOTES.md
git commit -m "docs: Cartesian map-consistency engine, flags, and cartcheck"
```

> **Part 1 is now independently shippable.** It can be merged to `master` (`--no-ff`) and run in `warn` mode in production while Part 2 is built. Triage any real findings from Task 6/7 before flipping to `panic`.

---

# PART 2 — Hybrid Mapper Rendering (GMCP snapshot + client)

### Task 9: Per-character visited-room memory

**Files:**
- Modify: `internal/characters/character.go`
- Create: `internal/characters/character_visited_test.go`
- Modify: `internal/usercommands/go.go`

No visited-room storage exists today. We add a persisted per-zone set on `Character` (serialized via `UserRecord`).

- [ ] **Step 1: Write the failing test.** Create `internal/characters/character_visited_test.go`:

```go
package characters

import "testing"

func TestMarkRoomVisited(t *testing.T) {
	c := &Character{}
	c.MarkRoomVisited("Stillwater Marsh", 4101)
	c.MarkRoomVisited("Stillwater Marsh", 4101) // dedup
	c.MarkRoomVisited("Stillwater Marsh", 4102)

	if !c.HasVisitedRoom("Stillwater Marsh", 4101) {
		t.Fatal("4101 should be visited")
	}
	if c.HasVisitedRoom("Stillwater Marsh", 9999) {
		t.Fatal("9999 should not be visited")
	}
	if got := c.GetVisitedRooms("Stillwater Marsh"); len(got) != 2 {
		t.Fatalf("expected 2 visited rooms, got %d (%v)", len(got), got)
	}
}
```

- [ ] **Step 2: Run it — verify it fails.** Run: `go test ./internal/characters/ -run TestMarkRoomVisited -v` — Expected: FAIL (undefined methods).

- [ ] **Step 3: Implement.** In `internal/characters/character.go`, add the field to the `Character` struct (near `Discoveries`):

```go
	VisitedRooms map[string][]int `yaml:"visitedrooms,omitempty"` // zone name -> visited roomIds (fog-of-war for the web map)
```

And add the methods (anywhere in the file, package level on `*Character`):

```go
// MarkRoomVisited records that this character has seen roomId in zone (dedup'd).
func (c *Character) MarkRoomVisited(zone string, roomId int) {
	if c.VisitedRooms == nil {
		c.VisitedRooms = map[string][]int{}
	}
	for _, id := range c.VisitedRooms[zone] {
		if id == roomId {
			return
		}
	}
	c.VisitedRooms[zone] = append(c.VisitedRooms[zone], roomId)
}

// HasVisitedRoom reports whether roomId in zone has been seen.
func (c *Character) HasVisitedRoom(zone string, roomId int) bool {
	for _, id := range c.VisitedRooms[zone] {
		if id == roomId {
			return true
		}
	}
	return false
}

// GetVisitedRooms returns the visited roomIds for a zone (nil if none).
func (c *Character) GetVisitedRooms(zone string) []int {
	return c.VisitedRooms[zone]
}
```

- [ ] **Step 4: Run the test — verify pass.** Run: `go test ./internal/characters/ -run TestMarkRoomVisited -v` — Expected: PASS.

- [ ] **Step 5: Mark visited on move.** In `internal/usercommands/go.go`, after the player has successfully entered the destination room (after the `rooms.MoveToRoom(user.UserId, destRoom.RoomId)` success path), add:

```go
		user.Character.MarkRoomVisited(destRoom.Zone, destRoom.RoomId)
```

(Place it where `destRoom` is in scope and the move has succeeded — near the existing post-move GMCP/quest handling.)

- [ ] **Step 6: Build.** Run: `go build ./...` — Expected: success.

- [ ] **Step 7: Commit.**

```bash
git add internal/characters/character.go internal/characters/character_visited_test.go internal/usercommands/go.go
git commit -m "feat(characters): per-zone visited-room memory; mark on move"
```

---

### Task 10: Mapper `Snapshot` builder

**Files:**
- Create: `internal/mapper/mapper.snapshot.go`
- Create: `internal/mapper/mapper.snapshot_test.go`

- [ ] **Step 1: Write the failing test.** Create `internal/mapper/mapper.snapshot_test.go`:

```go
package mapper

import "testing"

func TestSnapshotFiltersVisitedAndClassifies(t *testing.T) {
	nodes := map[int]*mapNode{
		1: node(1, 0, 0, 0, map[string]nodeExit{
			"north":   {RoomId: 2, Direction: d(0, -1, 0)},
			"east-x3": {RoomId: 3, Direction: d(3, 0, 0)},
		}),
		2: node(2, 0, -1, 0, map[string]nodeExit{"south": {RoomId: 1, Direction: d(0, 1, 0)}}),
		3: node(3, 3, 0, 0, map[string]nodeExit{}),
	}
	nodes[1].Symbol = 'T'
	m := mkMapper(nodes)

	visited := map[int]struct{}{1: {}, 2: {}} // room 3 not visited

	snap := m.Snapshot(visited)
	if len(snap) != 2 {
		t.Fatalf("expected 2 visited rooms in snapshot, got %d", len(snap))
	}

	var r1 *SnapshotRoom
	for i := range snap {
		if snap[i].RoomId == 1 {
			r1 = &snap[i]
		}
	}
	if r1 == nil {
		t.Fatal("room 1 missing from snapshot")
	}
	if r1.Symbol != "T" {
		t.Errorf("room 1 symbol: got %q want T", r1.Symbol)
	}
	// Only exits to other *visited* rooms are included; east-x3 -> room 3 (unvisited) is dropped.
	if len(r1.Exits) != 1 || r1.Exits[0].ToRoomId != 2 || r1.Exits[0].Kind != ExitNormal {
		t.Fatalf("room 1 exits wrong: %+v", r1.Exits)
	}
}
```

- [ ] **Step 2: Run it — verify it fails.** Run: `go test ./internal/mapper/ -run TestSnapshot -v` — Expected: FAIL (undefined `Snapshot`, `SnapshotRoom`).

- [ ] **Step 3: Implement.** Create `internal/mapper/mapper.snapshot.go`:

```go
package mapper

import "github.com/GoMudEngine/GoMud/internal/rooms"

// SnapshotExit is one classified spatial edge for the web map renderer.
type SnapshotExit struct {
	ToRoomId int      `json:"to"`
	DX       int      `json:"dx"`
	DY       int      `json:"dy"`
	DZ       int      `json:"dz"`
	Kind     ExitKind `json:"kind"`
}

// SnapshotRoom is one room placed in the zone coordinate space.
type SnapshotRoom struct {
	RoomId int            `json:"num"`
	X      int            `json:"x"`
	Y      int            `json:"y"`
	Z      int            `json:"z"`
	Symbol string         `json:"symbol"`
	Biome  string         `json:"biome"`
	Exits  []SnapshotExit `json:"exits"`
}

// Snapshot returns the visited rooms of this zone with classified exits to
// other visited rooms. Exits to unvisited or uncrawled rooms are omitted (fog
// of war). The exit Kind drives client rendering (normal/long/wrap/vertical).
func (r *mapper) Snapshot(visited map[int]struct{}) []SnapshotRoom {
	out := make([]SnapshotRoom, 0, len(visited))

	for id, n := range r.crawledRooms {
		if _, ok := visited[id]; !ok {
			continue
		}

		sr := SnapshotRoom{
			RoomId: id,
			X:      n.Pos.x,
			Y:      n.Pos.y,
			Z:      n.Pos.z,
			Symbol: string(n.Symbol),
		}
		if room := rooms.LoadRoom(id); room != nil {
			if b := room.GetBiome(); b != nil {
				sr.Biome = b.Name
			}
		}

		for _, e := range n.Exits {
			dst, ok := r.crawledRooms[e.RoomId]
			if !ok {
				continue
			}
			if _, ok := visited[e.RoomId]; !ok {
				continue
			}
			actual := positionDelta{x: dst.Pos.x - n.Pos.x, y: dst.Pos.y - n.Pos.y, z: dst.Pos.z - n.Pos.z}
			sr.Exits = append(sr.Exits, SnapshotExit{
				ToRoomId: e.RoomId,
				DX:       e.Direction.x,
				DY:       e.Direction.y,
				DZ:       e.Direction.z,
				Kind:     classifyKind(e.Direction, actual),
			})
		}
		out = append(out, sr)
	}
	return out
}
```

- [ ] **Step 4: Run the tests — verify pass.** Run: `go test ./internal/mapper/ -run TestSnapshot -v` — Expected: PASS. Then `go test ./internal/mapper/... ` — Expected: PASS.

- [ ] **Step 5: Commit.**

```bash
git add internal/mapper/mapper.snapshot.go internal/mapper/mapper.snapshot_test.go
git commit -m "feat(mapper): zone Snapshot builder with classified exits (fog of war)"
```

---

### Task 11: GMCP `Zone.Map` snapshot module + `symbol` on Room.Info

**Files:**
- Modify: `modules/gmcp/gmcp.Room.go`
- Create: `modules/gmcp/gmcp.Zone.go`

- [ ] **Step 1: Add `symbol` to the room payload.** In `modules/gmcp/gmcp.Room.go`, add a field to `GMCPRoomModule_Payload`:

```go
	Environment string                                              `json:"environment"`
	Symbol      string                                              `json:"symbol"`
```

And set it in `GetRoomNode` where `Environment` is set:

```go
		payload.Environment = room.GetBiome().Name
		if room.MapSymbol != `` {
			payload.Symbol = room.MapSymbol
		} else if b := room.GetBiome(); b != nil {
			payload.Symbol = b.SymbolString()
		}
```

- [ ] **Step 2: Create the Zone module.** Create `modules/gmcp/gmcp.Zone.go`:

```go
package gmcp

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type GMCPZoneModule struct {
	plug *plugins.Plugin
}

// GMCPZoneUpdate asks the system to (re)send the explored-zone map snapshot.
type GMCPZoneUpdate struct {
	UserId int
}

func (g GMCPZoneUpdate) Type() string { return `GMCPZoneUpdate` }

type GMCPZoneModule_Payload struct {
	Zone  string                `json:"zone"`
	Rooms []mapper.SnapshotRoom `json:"rooms"`
}

func init() {
	g := GMCPZoneModule{
		plug: plugins.New(`gmcp.Zone`, `1.0`),
	}
	// Re-send the snapshot whenever the player changes rooms (cheap; grows as
	// they explore, and reframes when they cross into a new zone).
	events.RegisterListener(events.RoomChange{}, g.roomChangeHandler)
	events.RegisterListener(GMCPZoneUpdate{}, g.buildAndSend)
}

func (g *GMCPZoneModule) roomChangeHandler(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok || evt.UserId == 0 {
		return events.Continue
	}
	events.AddToQueue(GMCPZoneUpdate{UserId: evt.UserId})
	return events.Continue
}

func (g *GMCPZoneModule) buildAndSend(e events.Event) events.ListenerReturn {
	evt, ok := e.(GMCPZoneUpdate)
	if !ok || evt.UserId < 1 {
		return events.Continue
	}
	user := users.GetByUserId(evt.UserId)
	if user == nil {
		return events.Continue
	}
	if !isGMCPEnabled(user.ConnectionId()) {
		return events.Cancel
	}

	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {
		return events.Continue
	}

	m := mapper.GetMapper(room.RoomId)
	if m == nil {
		return events.Continue
	}

	// Build the visited set for this zone.
	visited := map[int]struct{}{}
	for _, id := range user.Character.GetVisitedRooms(room.Zone) {
		visited[id] = struct{}{}
	}
	// Ensure the current room is always present even before the move-handler marks it.
	visited[room.RoomId] = struct{}{}

	payload := GMCPZoneModule_Payload{
		Zone:  room.Zone,
		Rooms: m.Snapshot(visited),
	}

	events.AddToQueue(GMCPOut{
		UserId:  evt.UserId,
		Module:  `Zone.Map`,
		Payload: payload,
	})

	mudlog.Debug("gmcp.Zone", "userId", evt.UserId, "zone", room.Zone, "rooms", len(payload.Rooms))
	return events.Continue
}
```

> Note: `GMCPOut` is already registered for dispatch in `gmcp.go`'s init; this module only enqueues it. The dispatcher JSON-marshals the payload and prepends `Zone.Map `.

- [ ] **Step 3: Build.** Run: `go build ./...` — Expected: success.

- [ ] **Step 4: Boot + inspect.** Boot the server, connect with the **web client**, move between a couple of rooms, and confirm (browser devtools console / network GMCP frames) that `Zone.Map` messages arrive with a growing `rooms` array. Expected: JSON like `Zone.Map {"zone":"...","rooms":[{"num":...,"x":..,"y":..,"z":..,"symbol":"~","biome":"Marsh","exits":[{"to":..,"dx":..,"dy":..,"dz":..,"kind":"normal"}]}]}`.

- [ ] **Step 5: Commit.**

```bash
git add modules/gmcp/gmcp.Room.go modules/gmcp/gmcp.Zone.go
git commit -m "feat(gmcp): Zone.Map explored-zone snapshot + symbol on Room.Info"
```

---

### Task 12: Client — hybrid room styling

**Files:**
- Modify: `_datafiles/html/public/static/js/gmcp.js`

Restyle `RoomGridSVG` toward the hybrid look. This task changes node appearance + adds a biome→tint table; wrap/long/vertical come in Tasks 13–14.

- [ ] **Step 1: Add the biome→tint table + options.** At the top of the `RoomGridSVG` constructor options block in `gmcp.js`, add defaults:

```js
      this.roomSize = options.roomSize || 16;          // hybrid: small nodes
      this.connectionColor = options.connectionColor || "#b8893f"; // amber
      this.connectionWidth = options.connectionWidth || 1.6;
      this.glyphColor = options.glyphColor || "#c9b48f";
      this.biomeTints = options.biomeTints || RoomGridSVG.DEFAULT_BIOME_TINTS;
```

And define the table as a static (after the class, or as a const above it):

```js
RoomGridSVG.DEFAULT_BIOME_TINTS = {
  // biome name (lowercased) -> subtle desaturated fill
  "city":   "#3a342c", "town": "#3a342c",
  "forest": "#25382a", "swamp": "#243226", "marsh": "#243226",
  "water":  "#243246", "lake":  "#243246", "river": "#243246",
  "hills":  "#3e3422", "mountain": "#3e3422",
  "cave":   "#2c2530", "dungeon": "#2c2530",
  "desert": "#3e3622", "road": "#3a3226",
  "_default": "#2a2018"
};
RoomGridSVG.prototype.tintFor = function (biome) {
  if (!biome) return this.biomeTints._default;
  var key = String(biome).toLowerCase();
  return this.biomeTints[key] || this.biomeTints._default;
};
```

- [ ] **Step 2: Restyle the room rect + add glyph.** In the NEW-room drawing branch of `addRoom`, change the rect to the hybrid style (small, tinted, thin amber stroke, corner radius) and add a faint glyph using `room.symbol` / biome:

```js
      const s = this.roomSize;
      rect.setAttribute('width', s);
      rect.setAttribute('height', s);
      rect.setAttribute('x', room.x * this.spacing - s / 2);
      rect.setAttribute('y', room.y * this.spacing - s / 2);
      rect.setAttribute('rx', 4);
      rect.setAttribute('ry', 4);
      rect.setAttribute('stroke', this.roomEdgeColor); // existing amber-ish edge
      rect.setAttribute('stroke-width', '1');
      rect.setAttribute('fill', room.Color || this.tintFor(room.biome));
```

Add the glyph element after the rect (faint, low-opacity), using `room.symbol`:

```js
      if (room.symbol) {
        const gly = document.createElementNS(this.svg.namespaceURI, 'text');
        gly.setAttribute('x', room.x * this.spacing);
        gly.setAttribute('y', room.y * this.spacing + s * 0.22);
        gly.setAttribute('text-anchor', 'middle');
        gly.setAttribute('font-size', s * 0.5);
        gly.setAttribute('font-family', 'monospace');
        gly.setAttribute('fill', this.glyphColor);
        gly.setAttribute('opacity', '0.75');
        gly.textContent = room.symbol;
        g.appendChild(gly);
      }
```

Update the current-room highlight to use the small-node geometry (red fill, light stroke) where `visitingColor` is applied.

- [ ] **Step 3: Thin the connectors.** In `_drawEdge`, set `stroke` to `this.connectionColor` and `stroke-width` to `this.connectionWidth` (replacing the thick width 20).

- [ ] **Step 4: Lint + visual smoke.** Run the JS lint if present (`npm run lint` or the project's lint task — check `package.json`; if no JS lint exists, skip). Boot the server, open the web client, and confirm the map panel renders small tinted nodes with thin amber connectors and a faint glyph, current room red. (No JS unit-test harness exists in this repo; validation is lint + manual smoke.)

- [ ] **Step 5: Commit.**

```bash
git add _datafiles/html/public/static/js/gmcp.js
git commit -m "feat(webclient): hybrid map styling — small tinted nodes, glyphs, thin connectors"
```

---

### Task 13: Client — consume `Zone.Map` snapshot

**Files:**
- Modify: `_datafiles/html/public/webclient-pure.html`
- Modify: `_datafiles/html/public/static/js/gmcp.js`

- [ ] **Step 1: Add a bulk snapshot loader to `RoomGridSVG`.** In `gmcp.js`, add a method that ingests the `Zone.Map` payload, replacing the map when the zone changes and merging otherwise:

```js
  /** Ingest a Zone.Map snapshot: [{num,x,y,z,symbol,biome,exits:[{to,dx,dy,dz,kind}]}]. */
  setZoneSnapshot(zone, snapshotRooms) {
    if (this._zone !== zone) {
      this.reset();
      this._zone = zone;
    }
    snapshotRooms.forEach(r => {
      this.addRoom({
        RoomId: r.num,
        x: r.x,
        y: r.y,
        z: r.z,
        symbol: r.symbol,
        biome: r.biome,
        Exits: (r.exits || []).map(e => ({ RoomId: e.to, kind: e.kind, dx: e.dx, dy: e.dy, dz: e.dz })),
        ExitsMeta: r.exits || []
      });
    });
    this._applyZoom();
  }
```

(Keep the existing per-`Room.Info` `addRoom` path for the current room; the snapshot is the bulk source.)

- [ ] **Step 2: Subscribe to `Zone.Map`.** In `webclient-pure.html`, find the GMCP message dispatch (where `Room.Info` / map messages are handled, near the `RoomGridSVG` usage around line 400–445) and add a branch:

```js
                else if (gmcpModule === 'Zone.Map') {
                    if (GMCPWindows['Map'] && GMCPWindows['Map'] !== false && gr) {
                        gr.setZoneSnapshot(obj.zone, obj.rooms || []);
                        GMCPWindows['Map'].setTitle('Map (' + obj.zone + ')');
                    }
                }
```

(Match the existing variable names for the parsed module string and payload object — read the surrounding handler to confirm `gmcpModule` / `obj`.)

- [ ] **Step 3: Build + smoke.** Boot, open the web client, walk through several rooms in a normal zone (e.g. Stillwater). Expected: the map fills in with the explored rooms (fog of war — only visited rooms appear), persists across the session, and the title shows the zone name.

- [ ] **Step 4: Commit.**

```bash
git add _datafiles/html/public/webclient-pure.html _datafiles/html/public/static/js/gmcp.js
git commit -m "feat(webclient): consume Zone.Map snapshot for explored-zone fog-of-war map"
```

---

### Task 14: Client — wrap edge-stubs + up/down ticks

**Files:**
- Modify: `_datafiles/html/public/static/js/gmcp.js`

Use the per-exit `kind` from the snapshot. `normal`/`long` draw connectors (existing `_drawEdge`, with `long` naturally longer because the destination sits multiple cells away). `wrap` draws an edge stub; `vertical` adds a ▲/▼ tick.

- [ ] **Step 1: Route edges by kind.** In `_drawEdgesForRoom` (or wherever edges are drawn from a room's exits), branch on the exit meta. Replace the unconditional `_drawEdge` with:

```js
    (me.ExitsMeta || []).forEach(e => {
      if (e.kind === 'vertical') {
        this._drawVerticalTick(id, e.dz);
        return;
      }
      if (e.kind === 'wrap') {
        this._drawWrapStub(id, e.dx, e.dy);
        return;
      }
      // normal / long: connect to the destination room if known
      if (this.rooms.has(e.to)) this._drawEdge(id, e.to);
    });
```

- [ ] **Step 2: Implement the wrap stub.** Add to `RoomGridSVG`:

```js
  _drawWrapStub(id, dx, dy) {
    const me = this.rooms.get(id); if (!me) return;
    const cx = me.room.x * this.spacing, cy = me.room.y * this.spacing;
    // unit direction toward the edge the exit nominally leaves by
    const ux = dx === 0 ? 0 : (dx > 0 ? 1 : -1);
    const uy = dy === 0 ? 0 : (dy > 0 ? 1 : -1);
    const len = this.roomSize * 1.4, start = this.roomSize * 0.55;
    const ex = cx + ux * (start + len), ey = cy + uy * (start + len);
    const WC = '#3fb0a0';
    const line = document.createElementNS(this.svg.namespaceURI, 'line');
    line.setAttribute('x1', cx + ux * start); line.setAttribute('y1', cy + uy * start);
    line.setAttribute('x2', ex); line.setAttribute('y2', ey);
    line.setAttribute('stroke', WC); line.setAttribute('stroke-width', this.connectionWidth);
    line.setAttribute('stroke-linecap', 'round');
    this.connectionsGroup.appendChild(line);
    // chevron
    const px = -uy, py = ux, c = this.roomSize * 0.28, b = this.roomSize * 0.34;
    [[+1], [-1]].forEach(sgn => {
      const ch = document.createElementNS(this.svg.namespaceURI, 'line');
      ch.setAttribute('x1', ex); ch.setAttribute('y1', ey);
      ch.setAttribute('x2', ex - ux * b + px * c * sgn[0]);
      ch.setAttribute('y2', ey - uy * b + py * c * sgn[0]);
      ch.setAttribute('stroke', WC); ch.setAttribute('stroke-width', this.connectionWidth);
      ch.setAttribute('stroke-linecap', 'round');
      this.connectionsGroup.appendChild(ch);
    });
  }
```

- [ ] **Step 3: Implement the vertical tick.** Add:

```js
  _drawVerticalTick(id, dz) {
    const me = this.rooms.get(id); if (!me) return;
    const s = this.roomSize;
    const cx = me.room.x * this.spacing, cy = me.room.y * this.spacing;
    const t = document.createElementNS(this.svg.namespaceURI, 'text');
    t.setAttribute('x', cx + s * 0.42);
    t.setAttribute('y', cy + (dz > 0 ? -s * 0.30 : s * 0.46));
    t.setAttribute('text-anchor', 'middle');
    t.setAttribute('font-size', s * 0.32);
    t.setAttribute('fill', '#8a6a3a');
    t.setAttribute('opacity', '0.6');
    t.textContent = dz > 0 ? '▲' : '▼'; // ▲ / ▼
    this.roomsGroup.appendChild(t);
  }
```

- [ ] **Step 4: Smoke.** Boot, open the web client, visit **Instance Planar Oasis** (mark `non_cartesian: true` on its `zone-config.yaml` first — see Task 15 note) and a multi-level/up-down zone. Expected: wrap exits render as teal edge stubs (not long lines), up/down show faint ▲/▼ ticks.

- [ ] **Step 5: Commit.**

```bash
git add _datafiles/html/public/static/js/gmcp.js
git commit -m "feat(webclient): wrap edge-stubs + up/down ticks from exit kind"
```

---

### Task 15: Client — fit & center controls; flag the oasis zone

**Files:**
- Modify: `_datafiles/html/public/static/js/gmcp.js`
- Modify: `_datafiles/world/dogmud/rooms/instance_planar_oasis/zone-config.yaml`

- [ ] **Step 1: Add fit + center buttons.** In `_createHTMLControls`, add two buttons next to the existing zoom `+`/`−`:

```js
      div.append(
        mk('⤢', () => this.fit()),
        mk('◎', () => this.centerOnRoom(this.currentCenterId)),
        mk('−', () => this.zoomOut()),
        mk('+', () => this.zoomIn())
      );
```

- [ ] **Step 2: Implement `fit()`.** Add:

```js
  /** Reset zoom so the whole explored map fits in view. */
  fit() {
    this._updateBounds();
    this.zoomLevel = 1;
    this.center = {
      x: this.bounds.minX * this.spacing + this.worldWidth / 2,
      y: this.bounds.minY * this.spacing + this.worldHeight / 2
    };
    this._applyZoom();
  }
```

- [ ] **Step 3: Flag the oasis as non-Cartesian.** In `_datafiles/world/dogmud/rooms/instance_planar_oasis/zone-config.yaml`, add:

```yaml
non_cartesian: true
```

(This both exempts it from the consistency checks and makes its wrap exits render as edge stubs. Confirm whether the oasis actually authors edge wraps; if the current 5003–5005 strip has none yet, the flag is harmless and future-proofs the zone.)

- [ ] **Step 4: Smoke + boot test.** Wipe instance saves, boot, run `cartcheck Instance Planar Oasis` (expect no findings now), open the web client and confirm fit/center work. Run `go build ./...` and `go test ./internal/...` — Expected: pass.

- [ ] **Step 5: Commit.**

```bash
git add _datafiles/html/public/static/js/gmcp.js _datafiles/world/dogmud/rooms/instance_planar_oasis/zone-config.yaml
git commit -m "feat(webclient): fit/center controls; flag planar oasis non_cartesian"
```

---

### Task 16: Part 2 documentation + smoke checklist

**Files:**
- Modify: `docs/coordinate_map.md`, `internal/mapper/context.md`, `CLAUDE.md`, `PATCH_NOTES.md`

- [ ] **Step 1: Document the snapshot + client.** In `docs/coordinate_map.md`, add a "Web Mapper (hybrid)" section: the `Zone.Map` GMCP message shape, per-character visited-room fog of war, the exit-`kind` → render mapping (normal/long = connector, wrap = teal stub, vertical = ▲/▼ tick), the biome→tint table, and fit/center/zoom controls. In `internal/mapper/context.md`, note `Snapshot` and `SnapshotRoom`/`SnapshotExit`.

- [ ] **Step 2: Extend the `CLAUDE.md` section.** Append to the "Map Consistency & the `non_cartesian` / `oneway` Flags" section a paragraph on the web mapper: hybrid style, `Zone.Map` snapshot driven by `Character.VisitedRooms`, edge-stub wrap rendering for `non_cartesian` zones, client renderer in `gmcp.js` (`RoomGridSVG`).

- [ ] **Step 3: PATCH_NOTES entry.** Add a dated entry for the hybrid web mapper (explored-zone map, biome tints, wrap stubs, up/down ticks, fit/center).

- [ ] **Step 4: Full regression + boot test.** Run:

```bash
go build ./...
go test ./internal/mapper/... ./internal/characters/... -v
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run .   # watch for clean load + ValidateZoneConsistency summary, no panic
```

Manual smoke matrix (web client): (a) a normal zone fills in as fog-of-war; (b) a zone with a `-x3` long exit shows a long connector; (c) Instance Planar Oasis wraps render as stubs; (d) an up/down zone shows ▲/▼ ticks; (e) fit/center/zoom work.

- [ ] **Step 5: Commit.**

```bash
git add docs/coordinate_map.md internal/mapper/context.md CLAUDE.md PATCH_NOTES.md
git commit -m "docs: hybrid web mapper, Zone.Map snapshot, fog-of-war"
```

---

## Self-Review (completed by plan author)

**Spec coverage** — every spec section maps to a task:
- 4.A Hybrid style → Task 12. 4.B Evolve RoomGridSVG → Tasks 12–15. 4.C Data pipeline (symbol + zone snapshot) → Tasks 10–11. 4.D Long exits → Task 13 (connector via coordinate placement) + Task 5 long-crossing warning. 4.E Wrap edge-stubs + detection → Tasks 5 (classify/flag), 10 (kind in snapshot), 14 (stub render). 4.F Up/down ticks → Task 14. 4.G Consistency engine → Tasks 4–7. New primitives (oneway/non_cartesian/knob/cartcheck/symbol/snapshot) → Tasks 1,2,3,7,11,10. Exemptions → Task 5 logic. Rollout phasing → Tasks 6 (warn) + the Part 1 ship note. Testing → tests in Tasks 4,5,9,10 + smoke in 6,7,15,16. Docs → Tasks 1,2,8,16.

**Placeholder scan** — no "TBD/handle edge cases/similar to" left; all code blocks are concrete. Two explicit "confirm by reading the surrounding handler" notes remain in Task 6 (`getZoneConfig` accessor name) and Task 13 (`gmcpModule`/`obj` variable names) — these are accurate "match the existing local symbol" instructions, not deferred design.

**Type consistency** — `positionDelta` always compared via `samePos`; `ExitKind` constants (`ExitNormal/Long/Wrap/Vertical`) consistent across Tasks 4/5/10; `SnapshotRoom`/`SnapshotExit` JSON tags (`num/x/y/z/symbol/biome/exits`, `to/dx/dy/dz/kind`) match the client's `setZoneSnapshot` field reads in Task 13; `MarkRoomVisited/HasVisitedRoom/GetVisitedRooms` signatures consistent between Tasks 9 and 11; `CheckConsistency(zone, nonCartesian)` signature consistent across Tasks 5/6/7.

**Open items from spec §10 resolved:** visited-room storage did not exist → added in Task 9. Snapshot namespace chosen → `Zone.Map`. `-gap` exits: out of scope (treated as normal unit exits by `posDeltas`; no special handling needed).
