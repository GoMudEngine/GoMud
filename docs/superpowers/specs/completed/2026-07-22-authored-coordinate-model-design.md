# Authored Coordinate Model — Design (admin web-building, sub-project 1a)

**Date:** 2026-07-22
**Status:** approved design, pre-plan
**Epic:** Admin web-building. This is **sub-project 1a** — the engine foundation the
room-builder page (**1b**) sits on. No UI in 1a.

## Motivation

The web room-builder (1b) must **enforce Cartesian consistency on `(x, y, z, plane)`
with no overlapping rooms**. Today that is impossible to enforce cleanly because
geometry is **derive-not-author**: rooms store no coordinates; the mapper crawls
exit deltas from a zone root at `(0,0,0)` and positions are an emergent property
of the exit graph, validated only after the fact (`findCollisions` post-pass).
There is also **no plane/layer concept** — instances are excluded from the
coordinate system entirely and multiple live instances crawl to identical coords.

1a inverts this: **the room becomes the authoritative holder of its coordinate**,
overlap becomes impossible by construction within a plane, and instances/interiors
get their own plane. This is independently valuable (the whole world becomes
Cartesian-authoritative) and independently verifiable (via `cartcheck` + boot),
and it is a hard prerequisite for 1b.

Ground truth this builds on (from the 2026-07-22 coordinate scan):
- `Room` has no `X/Y/Z/Coord/plane` field (`internal/rooms/rooms.go:76-115`).
- Mapper crawl + delta table: `internal/mapper/mapper.go:41-98` (deltas: north=y−1,
  south=y+1, west=x−1, east=x+1, up=z+1, down=z−1), crawl at `mapper.go:249-339`.
- Collision detector: `internal/mapper/mapper.consistency.go:50-65` (`[3]int{x,y,z}`
  bucket); all checks `:104-155`; enforce modes `:206-242`; config knob
  `internal/configs/config.gameplay.go:22`.
- Escape hatches: `ZoneConfig.NonCartesian` `internal/rooms/zoneconfig.go:21`;
  `exit.RoomExit.OneWay` `internal/exit/exit.go:21`.
- Instances/ephemeral: `internal/rooms/instances.go:297-434`,
  `internal/rooms/ephemeral.go:56-168,245-274`; ephemeral rooms excluded from the
  checker (`roomCrawlable` false, `consistency.go:97-99`).

## Data model

Add to the `Room` struct + YAML (`internal/rooms/rooms.go`):

```yaml
x: 3        # authored grid coordinates within the room's plane
y: -1
z: 0        # vertical level (up = z+1, down = z-1), matching the engine frame
plane: 0    # coordinate-space id; 0 = overworld (omitempty: 0 not written)
```

- `X, Y, Z int` and `Plane int`, all `yaml:"...,omitempty"`.
- **Plane is per-room** (required: instances assign plane per-room at runtime; also
  lets one zone hold interior sub-planes). `ZoneConfig` gains an optional
  `default_plane int` (`zoneconfig.go`) as a builder convenience only — the room's
  own `plane` is authoritative.
- The **engine internal frame** is authoritative: north = y−1. (Note the doc
  `docs/coordinate_map.md` inverts Y for human readability — the builder UI in 1b
  will present a human frame, but stored coords use the engine frame. Flag this in
  1b.)

### Plane registry + the non-Euclidean flag

A **plane** is an `int` coordinate-space id with one property: whether it is
**Euclidean** (grid-enforced) or **non-Euclidean** (isolated but not grid-enforced).

- Plane `0` = the overworld (the single largest connected crawl component).
- Each other authored coordinate space (isolated zone, interior) = its own plane id.
- The 7 zones currently flagged `non_cartesian: true` (crash_site_interior,
  endless_trashheap, ferries, instance_planar_oasis, labyrinth_of_low_tunnels,
  newcomer_antechamber, the_foldweave) become **non-Euclidean planes**.

Representation: a boot-built registry `map[int]PlaneInfo{ NonEuclidean bool; Label string }`
in `internal/rooms`, populated from zone configs during load (a zone whose
`non_cartesian` flag is set contributes a non-Euclidean plane). `non_cartesian` is
**repurposed**: it no longer means "allow overlap in this zone's shared frame" — it
means "this zone's plane is non-Euclidean." Enforcement consults the plane's
`NonEuclidean` flag, not the per-zone flag directly.

## Plane semantics

- **Within a Euclidean plane:** `(x,y,z)` is unique — no two rooms share a cell.
  Spatial exits must match coordinate deltas.
- **Within a non-Euclidean plane:** rooms are isolated from all other planes (so no
  collision with the overworld), but the grid-layout rules (uniqueness, spatial-delta)
  are **not** enforced — preserving toroidal mazes and "endless" areas as designed.
- **Across planes:** no coordinate relationship; the only links are portal/door exits
  (below).
- **Instances:** at instance creation (`CreateZoneInstanceWithOpts`,
  `instances.go:297`), the instance is assigned a **unique plane id** (a high,
  runtime-allocated range, e.g. `instancePlaneBase + seq`). Ephemeral rooms keep the
  template's `x/y/z` but on the instance's plane → they can never collide with the
  template or other instances. The instance plane inherits the template zone's
  Euclidean/non-Euclidean flag. Ephemeral rooms are **no longer excluded** from the
  coordinate system (`roomCrawlable` exclusion removed for coord purposes; see
  Enforcement).

## Exit classification (spatial vs portal)

An exit is one of two kinds, decided by its map direction:

- **Spatial exit** — a compass/vertical direction (n/s/e/w, diagonals, up/down, and
  the long `-x2`/`-x3` variants). Constraint: it must connect two **same-plane** rooms
  whose coordinate delta equals the direction's unit (or `x2`/`x3`) delta. The
  `posDeltas` table (`mapper.go:41-98`) is the single source for direction→delta.
- **Portal/door exit** — a named, non-compass exit (`MapDirection` empty / not in
  `posDeltas`). Connects **any** two rooms, possibly **cross-plane**, carries **no**
  coordinate constraint, and renders as a stub (already how "portal/named exits are
  non-spatial and exempt" works). This is the mechanism for overworld → interior →
  instance movement.

`oneway` (`exit.go:21`) is unchanged: it exempts only the reciprocity check.

## Migration (one-time, idempotent)

New migration `migration-authored-coords` (guarded by a global done-flag like the
existing migrations):

1. Run the current mapper crawl per **connected component** across the whole world
   (following all exits by RoomId, cross-zone included).
2. Assign plane ids: the **largest** component = plane `0` (overworld); each other
   component = a fresh plane id; any component belonging to a `non_cartesian` zone is
   marked non-Euclidean. Log the full component → plane assignment for review.
3. Write `x/y/z/plane` onto every non-instance room's YAML template via
   `SaveRoomTemplate` (`save_and_load.go:177`).
4. **Lossless assertion:** for every Euclidean-plane room, assert authored coord ==
   crawled coord. Any mismatch is a hard migration error (abort + report the room) —
   this proves the migration changes rendered positions for **zero** rooms.
   Non-Euclidean planes skip the equality assert (their crawl may legitimately overlap).

After migration, every non-instance room has authored coords; instance planes are
assigned live at clone time.

## Enforcement

- **Collision key → `[4]int{plane,x,y,z}`.** Update `findCollisions`
  (`consistency.go:50`) and `longSpanCrossesRoom` (`consistency.go:166`) to the
  4-tuple. Skip collision + spatial-delta checks for **non-Euclidean** planes.
- **Authoritative, not crawled:** `CheckConsistency` reads authored coords directly
  (no crawl needed for the check). The crawl is retained only as the migration /
  one-off validation tool.
- **Spatial-delta check (new):** for each spatial exit, verify the two rooms are
  same-plane and their authored-coord delta equals the direction delta; else emit a
  `Kind:"deltamismatch"` finding. (Replaces the old wrap-only deltamismatch with the
  authored-coord version.)
- **Build-time gate (the API 1b calls):** a `rooms.ValidatePlacement(room, plane, x, y, z)`
  helper returns an error if the cell is occupied (Euclidean plane) — 1b's `Build.*`
  handlers call this before persisting and reject on error. (1a ships the helper +
  `cartcheck`/boot enforcement; 1b wires it to the UI.)
- `cartcheck` (`admin.cartcheck.go`) and the boot validator
  (`ValidateZoneConsistency`, `consistency.go:206`) use the new 4-tuple/authored path.
  `MapConsistencyEnforce` modes (`off|warn|panic`) unchanged.

## Mapper reads authored coords

- `mapper.Start` (`mapper.go:249`) switches from crawl-assign to **reading
  `Room.X/Y/Z`**, grouping the render grid **per plane** (`RoomGrid` becomes
  per-plane, `mapper.go:180-212`). A room with no authored coords (shouldn't exist
  post-migration) falls back to crawl as a defensive guard + logs a warning.
- `Zone.Map` GMCP snapshot (`modules/gmcp/gmcp.Zone.go`, `mapper.snapshot.go`) sends
  authored coords + `plane`; the web mapper + ASCII `map` render unchanged in
  appearance (coords are identical post-migration by the lossless assertion).

## Out of scope (explicitly)

- **The builder UI, `Build.*` GMCP, the admin page** — that is 1b.
- **Mob/item/dialogue/quest authoring** — later sub-projects.
- **Re-authoring the 7 non-Euclidean zones into grids** — they stay non-Euclidean by
  the per-plane flag; not touched here.
- **Cross-plane coordinate re-basing when the builder connects two planes** — a 1b
  builder operation; 1a only needs planes to exist and be enforced.

## Testing & verification

- **Unit:** `posDeltas` delta table round-trips; `ValidatePlacement` rejects an
  occupied Euclidean cell and allows a free one and any non-Euclidean cell;
  4-tuple `findCollisions` (same x/y/z different plane = no collision; same 4-tuple =
  collision); spatial-delta check flags a mismatched exit.
- **Migration:** on the real world data, the lossless assertion passes for every
  Euclidean-plane room (authored == crawled); the component→plane log is sane
  (one big plane 0 + a handful of isolated/non-Euclidean planes).
- **Boot / `cartcheck`:** with `MapConsistencyEnforce: panic`, a full boot is clean
  world-wide after migration (the authoritative check on authored coords).
- **Mapper parity:** a rendered `Zone.Map`/ASCII map for several zones is byte-identical
  before and after the switch to authored coords (guards the render change).
- **Instances:** spinning up two live instances of one template puts their rooms on
  distinct planes with no collision finding.
- **Content gate:** none (no player-facing prose); this is engine/data-model work.
