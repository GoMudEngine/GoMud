# Mapping System Context

## Purpose

`internal/mapper` turns the room graph into geometry. It crawls exits outward
from a root room, assigns each reachable room an (x, y, z) coordinate, and
serves three consumers from that one structure: the ASCII `map` command,
A\* pathfinding, and the web client's leather map snapshot.

One `mapper` exists per zone, cached and keyed by room id. Everything below —
consistency checking, fog-of-war snapshots, cross-zone routing — is built on
the same crawl.

## Files

- **mapper.go** — direction/delta tables, the `mapper` type, the crawl, the
  per-zone cache, and the render entry points.
- **mapper.node.go** — `mapNode` and `nodeExit`, the crawl's internal graph.
- **mapper.map.go** — `mapRender` and legend assembly.
- **mapper.config.go** — `Config` (render options) and `SymbolOverride`.
- **mapper.path.go** — A\* pathfinding, the path cache, and `GetPath`/`NextStep`.
- **mapper.consistency.go** — the Cartesian consistency engine (see below).
- **mapper.snapshot.go** — the fog-of-war snapshot for the web map.
- **mapper.crosszone.go** — the coarse zone graph and cross-zone hop routing.

## Coordinates: authored first, crawl second

`(*mapper).discover()` picks one of two placement strategies:

- **`placeByAuthoredCoords()`** when `authoredLayoutClean()` passes — rooms
  carry authored x/y/z/plane, and those are used directly.
- **`startByCrawl()`** otherwise — coordinates are derived by walking exits and
  accumulating deltas from the root room.

This is why the world must stay Cartesian-consistent: under the crawl path, two
rooms whose exit deltas lead to the same cell genuinely collide, and the map
silently loses one of them.

## Public API

### Direction helpers

```go
func GetDelta(exitName string) (x, y, z int)
func GetReciprocalExit(exitDirection string) string
func IsValidExitDirection(exitName string) bool
func IsCompassDirection(exitName string) bool
func IsValidDirection(directionName string) bool
func GetDirectionDeltaNames() []string
func AdjustExitName(exitName string) (newExitName, newExitDirection string, err error)
```

### Mapper lifecycle

```go
func NewMapper(rootRoomId int) *mapper
func GetMapper(roomId int, forceRefresh ...bool) *mapper
func GetMapperIfExists(roomId int) *mapper
func AddMapper(m *mapper, lookupRoomId int)
func PreCacheMaps()
func ClearCache()

func (r *mapper) Start()
func (r *mapper) RootRoomId() int
func (r *mapper) CrawledRoomIds() []int
func (m *mapper) GetCopy() *mapper
func (m *mapper) OverrideRoomIds(replacements map[int]int)
```

`PreCacheMaps` builds every zone's mapper at boot, then warms the cross-zone
graph and runs `ValidateZoneConsistency`. `GetCopy`/`OverrideRoomIds` exist for
ephemeral instance rooms, which reuse a template zone's geometry under
different room ids.

### Lookup

```go
func (r *mapper) HasRoom(roomId int) bool
func (r *mapper) GetRoomId(x, y, z int) (roomId int, err error)
func (r *mapper) GetCoordinates(roomId int) (x, y, z int, err error)
func (r *mapper) FindRoomsInDistance(centerRoomId int, xyRadius int, zRadiusOpt ...int) []int
func (r *mapper) FindAdjacentRoom(centerRoomId int, direction string, limitDistance ...int) (roomId, distance int)
```

### Rendering

```go
func (r *mapper) GetLimitedMap(centerRoomId int, c Config) mapRender
func (r *mapper) GetFullMap(centerRoomId int, c Config) mapRender
func (m *mapRender) GetLegend(overrides map[rune]string) map[rune]string
func (c *Config) OverrideSymbol(roomId int, symbol rune, legend string)
```

`GetLimitedMap` is the player-facing one — the visible radius scales with
Perception. `GetFullMap` is unbounded and used by admin tooling.

### Pathfinding

```go
func GetPath(startRoomId int, endRoomId ...int) ([]pathStep, error)
func NextStep(fromRoomId, toRoomId int) (nextRoomId int, exitName string, found bool)

func (p pathStep) ExitName() string
func (p pathStep) RoomId() int
func (p pathStep) Waypoint() bool
```

A\* over the crawled graph (`heuristic` + a `priorityQueue` of `nodeRecord`),
with results memoised in a path cache keyed by start/end. `GetPath` is
**per-zone by construction** — it resolves its mapper from the cache using the
*start* room, so a target in another zone fails outright. `NextStep` is the
wrapper that falls back to `crossZoneHop` in that case; see Cross-Zone Routing
below.

## Gotchas

- **Never compare `positionDelta` values with `==`.** Use `samePos`, which
  compares only x/y/z — the struct also carries an `arrow` display field that
  will produce false mismatches.
- **`crawledRooms` is the authoritative node set, not `RoomGrid`.** `RoomGrid`
  is a 3D slice, so when two rooms land on one cell only the last writer
  survives; a collision is invisible there and visible in `crawledRooms`.
- **`ClearCache` is a package-level name that also exists in several other
  packages.** Confirm which one you are calling.
- **A mapper is per-zone even though the crawl can cross zone boundaries.**
  Findings and renders are filtered back to the owning zone afterwards
  (`FilterFindingsToZone`); the crawl itself does not stop at the border.

## Dependencies

`rooms`, `exit`, `configs`, `mudlog`, `util`, `gametime` (for biome/light
context during render), plus `container/heap` for the A\* queue.

## Consumers

`internal/usercommands` (`map`, `pathto`, `cartcheck`), `internal/mobs`
(schedule and patrol movement), `internal/questengine` (map markers),
`internal/rooms`, and `modules/gmcp` (the `Zone.Map` payload).

## Cartesian Consistency Engine

The mapper enforces geometric coherence between room coordinates and
declared exits. The entry point is `ValidateZoneConsistency()`, called
at the tail of `PreCacheMaps()` and gated by the config knob
`GamePlay.MapConsistencyEnforce` (`off` | `warn` (default) | `panic`).
The same logic is exposed on demand via the `cartcheck [zone]` admin
command (`internal/usercommands/admin.cartcheck.go`).

### Core Method

```go
(*mapper).CheckConsistency(zone string, nonCartesian bool) []Finding
```

Scans `crawledRooms` (the `map[int]*mapNode` populated during BFS
crawl) for the zone and returns a slice of findings. `RoomGrid` is a
separate struct over a `[][][]*mapNode` 3D slice used for rendering, not
for consistency checks. The `nonCartesian` flag is sourced from the
zone's `zone-config.yaml` field `non_cartesian: true`; when set, only
`longcrossing` warnings are emitted (the hard checks are skipped).

### Exit Kind Classification

```go
func classifyKind(nominal, actual positionDelta) ExitKind
```

Exit-kind classifier used by the map snapshot/render layer (consumed in
later tasks). Compares the nominal compass delta (what the exit direction
implies) to the actual coordinate delta (the difference between the two
rooms' positions). Returns one of four `ExitKind` values:

**Note:** `CheckConsistency` does NOT call `classifyKind`. Detection of
`deltamismatch` and `longcrossing` is performed via inline delta
comparisons (`samePos`) directly inside `CheckConsistency`.

| Kind       | Meaning                                                     |
|------------|-------------------------------------------------------------|
| `normal`   | Nominal == actual (standard single-cell step)               |
| `long`     | Same direction, magnitude > 1 (multi-cell connector)        |
| `vertical` | One axis is Z only (up/down exits)                          |
| `wrap`     | Nominal and actual differ — toroidal or maze-style crossing |

**Important:** `classifyKind` uses the helper `samePos(a, b positionDelta)`,
which compares only the `x`, `y`, `z` fields. Never compare
`positionDelta` values with `==` directly — the struct also carries an
`arrow` display field that will cause false mismatches.

### Finding Kinds

| Kind            | Severity | Description                                              |
|-----------------|----------|----------------------------------------------------------|
| `collision`     | error    | Two distinct rooms share the same (x,y,z) coordinate.   |
|                 |          | Detected by grouping `crawledRooms` nodes by their       |
|                 |          | `(x,y,z)` Pos. Scanning `crawledRooms` (not `RoomGrid`)  |
|                 |          | is required because `RoomGrid` is a 3D slice: when two   |
|                 |          | rooms land on the same cell, the slice assignment keeps  |
|                 |          | only the last writer, making the collision invisible     |
|                 |          | there — every room is present in `crawledRooms`.         |
| `noreciprocal`  | error    | A spatial exit has no matching return exit in the        |
|                 |          | opposite direction, and the exit is not marked           |
|                 |          | `oneway: true`.                                          |
| `deltamismatch` | error    | Exit's compass direction does not match the actual       |
|                 |          | coordinate delta between the two rooms — a wrap exit     |
|                 |          | inside a Cartesian zone.                                 |
| `longcrossing`  | warning  | A long-connector exit's straight span passes through     |
|                 |          | another room's occupied cell. Always emitted regardless  |
|                 |          | of `non_cartesian` setting.                              |

### Exemptions

The engine automatically skips:
- **Non-compass exits** (portals, named exits): filtered via the
  `getMapNode` `mapdirection→name→skip` rule; they are not spatial edges.
- **Ephemeral/instance rooms**: checked via `rooms.IsEphemeralRoomId`.
- **`oneway: true` exits**: exempt from `noreciprocal`; still
  collision-checked.
- **`non_cartesian: true` zones**: exempt from `collision`,
  `noreciprocal`, and `deltamismatch`; their wrap exits render as
  edge stubs in the web mapper.

### Authoring Primitives

- **`oneway: true`** on an exit YAML field — marks an intentional
  one-way passage; suppresses the reciprocity check for that exit.
- **`non_cartesian: true`** in a zone's `zone-config.yaml` — marks the
  entire zone as toroidal/maze geometry; skips the three hard checks
  zone-wide.

### Known Limitation: Cross-Zone Crawl

`CheckConsistency` operates on the BFS-populated `crawledRooms` map,
which follows all exits and can cross zone boundaries. This is mitigated
at the reporting layer by `FilterFindingsToZone`, which drops any finding
whose room's owning `zone:` field does not match the zone being checked.
Both `ValidateZoneConsistency` (startup) and `CartCheck` (admin command)
apply this filter, so findings are correctly scoped to their owning zone.

## Fog-of-War Snapshot (Web Map)

### File

`mapper.snapshot.go`

### Types

```go
type SnapshotRoom struct {
    RoomId int            `json:"num"`
    X      int            `json:"x"`
    Y      int            `json:"y"`
    Z      int            `json:"z"`
    Symbol string         `json:"symbol"`
    Biome  string         `json:"biome"`
    Exits  []SnapshotExit `json:"exits"`
}

type SnapshotExit struct {
    ToRoomId int      `json:"to"`
    DX       int      `json:"dx"`
    DY       int      `json:"dy"`
    DZ       int      `json:"dz"`
    Kind     ExitKind `json:"kind"`
}
```

### Method

```go
func (r *mapper) Snapshot(visited map[int]struct{}) []SnapshotRoom
```

Iterates `crawledRooms` and returns only rooms whose ID is present in
`visited`. For each included room, exits are also filtered: an exit is
included only if its destination room is in `visited` (fog of war — the
client never learns about rooms the player hasn't been to). Each exit's
`Kind` is set by `classifyKind(nominal, actual)` — the same classifier
used by the consistency engine.

This method is the sole output consumed by the `gmcp.Zone` module
(`modules/gmcp/gmcp.Zone.go`), which builds the `Zone.Map` GMCP payload
and sends it to the web client on every room change. The web client
renderer (`RoomGridSVG` in `gmcp.js`) uses the `kind` field to route
each exit to its correct visual treatment: connector line (`normal`/
`long`), teal edge-stub with chevron (`wrap`), or ▲/▼ tick (`vertical`).

### SnapshotExit Extended Fields

`SnapshotExit` carries additional flags that inform per-exit visual
styling on the client:

```go
type SnapshotExit struct {
    ToRoomId int      `json:"to"`
    DX       int      `json:"dx"`
    DY       int      `json:"dy"`
    DZ       int      `json:"dz"`
    Kind     ExitKind `json:"kind"`
    Locked   bool     `json:"locked,omitempty"`
    Secret   bool     `json:"secret,omitempty"`
    OneWay   bool     `json:"oneway,omitempty"`
    Gate     bool     `json:"gate,omitempty"`
    Stub     bool     `json:"stub,omitempty"`
    ToZone   string   `json:"tozone,omitempty"`
}
```

- `Locked` — exit has a key requirement (room has a lock or door key set).
- `Secret` — exit is normally hidden from plain sight.
- `OneWay` — exit is flagged `oneway: true`; no return exit expected.
- `Gate` — set when the exit's `ExitMessage != ""`; indicates a barrier
  or door with flavor text (portcullises, heavy doors, etc.).
- `Stub` — the destination room is **not** in the visited set (unvisited)
  or is in a different zone. Stub exits are now emitted by `Snapshot`
  instead of being dropped, so the client can render a visual hint that
  a passage continues beyond the fog boundary.
- `ToZone` — populated on cross-zone stub exits; contains the destination
  zone name. Allows the client to label or style zone-boundary exits
  distinctly.

**Prior behavior:** `Snapshot` dropped exits whose destination was not in
the visited set. After this change it emits them as `Stub: true` entries,
giving the client enough information to draw a "passage continues"
indicator without revealing the destination room's details.

### nodeExit.Gate

`nodeExit` (the internal per-exit node built during BFS crawl) gained a
`Gate bool` field set from `exit.ExitMessage != ""`. This propagates to
`SnapshotExit.Gate` during snapshot construction.

## Map Consumers

The mapper data is consumed by two independent rendering paths:

### (a) In-Game ASCII `map` Command

`internal/usercommands/skill.map.go` calls `GetLimitedMap` and
`GetLegend` to render a terminal-width ASCII map scaled by the player's
Perception skill. Symbol legend:

| Symbol | Meaning              |
|--------|----------------------|
| `@`    | You (current room)   |
| `☺`    | Player / Party / NPC |
| `☠`    | Hostile mob          |
| `☹`    | Friendly NPC         |
| *(biome/mapsymbol)* | Room terrain glyph |

Detail level (visible radius, secret/locked display) scales with
Perception. This path is text-only and has no awareness of the
`SnapshotExit` extended flags or `Zone.Map.Party`.

### (b) Web Leather Map (GMCP `Zone.Map`)

The `Zone.Map` GMCP snapshot (`Snapshot`) feeds the browser client.
The `Zone.Map` payload now includes a `Party []int` field — a list of
room IDs currently occupied by party members — enabling the client to
render party-member position markers on the map.

The web renderer (`RoomGridSVG` in
`_datafiles/html/public/static/js/gmcp.js`) presents an **antique
tooled-leather** themed map: a fixed leather-textured SVG surface holds
a nested pannable `worldSvg` containing the room grid. Connection
styling is per-exit-type:

- **Biome roads/trails/water** — line color/style derived from room biome.
- **Locked** — distinctive styling for keyed doors.
- **Secret** — rendered as a dimmed or dashed connector.
- **One-way** — directional arrow or asymmetric line weight.
- **Gate** — styled to suggest a barrier (portcullis texture or color).
- **Stairs** — ▲/▼ ticks on the room node for `vertical` exits.
- **Cross-zone stubs** — short labeled stubs at the zone boundary.
- **Fog stubs** — dim stubs for unvisited exits (stub exits in the
  snapshot).

Party markers are small figures drawn on the room node for each room ID
in `Zone.Map.party`. The current player's room is rendered with a raised
(drop-shadow) treatment to distinguish it from adjacent rooms.

Visual source of truth: `docs/superpowers/specs/2026-06-06-mapper-leather-mockups/`.

Connection-type styling and party markers are **web-only** — the ASCII
`map` command does not reflect these.
## Cross-Zone Routing (`mapper.crosszone.go`)

`GetPath` is per-zone by construction — it resolves its mapper from
`mapperZoneCache[zone]` keyed off the **start** room, so it fails outright on a
target in another zone. Before 2026-07-30 that made every cross-zone quest
marker inert: no destination dot (the room isn't in the current zone's
snapshot) and no next-step arrow. Six quests were affected; quest 67 crosses
four New Plymouth districts, so its guidance went dark exactly when a player
was trying to find the right one.

`NextStep` now falls back to `crossZoneHop` when `GetPath` fails and the two
rooms are in different zones. The approach is deliberately **not** a global
room-level pathfinder:

1. A coarse zone graph (`zoneGraph`) records, per zone, one outbound link per
   neighbouring zone: `zoneLink{toZone, exitRoom, exitName, destRoom}`.
2. `nextZoneHop` BFSes zone-to-zone and returns the **first** leg — the border
   out of the player's current zone, even when the target is several zones off.
3. `crossZoneHop` then reuses the in-zone `GetPath` to walk to that border room
   (or returns the crossing exit itself if the player is already standing on
   it).

The hot path therefore stays on the cached per-zone mapper. The graph is built
once and warmed in `PreCacheMaps` (every room is already loaded there);
`InvalidateZoneGraph()` is exported for zone rename/delete/re-zone.

Real-world shape at boot: **39 zones with outbound links, 82 links.**

**Limitation:** one border is kept per zone pair. If that crossing is
unreachable from where the player stands, the arrow is dropped rather than a
second crossing tried — degrading to the old behaviour rather than misleading.

**The web marker still needs the room to be in the current zone's snapshot**, so
cross-zone journeys get the *arrow* but not the destination dot until the player
reaches that zone. The arrow is the part that guides.

### Why there are two zone-adjacency crawls

`modules/weather/crawler/build.go` builds its own zone graph for weather
fronts. Unification was designed and **rejected** on 2026-07-30 — `sim.Edge` is
undirected and discards the border room/exit that routing needs; `internal/`
never imports `modules/`; and the weather packages are vendored from a
standalone repo with arch tests (`TestCrawlerPackageStaysPure`) that forbid
engine imports. The full reasoning, the known behavioural differences, and the
triggers that would reopen the decision are in the header comment of
`mapper.crosszone.go`. Read that before attempting to merge them.
