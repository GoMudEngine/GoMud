# Pothole Coulee — Coordinate Budget (Newbie Area Rework, Chunk 0)

**Status:** Reserved 2026-06-12 (Chunk 0 deliverable)
**Parent spec:** `docs/superpowers/specs/2026-05-27-newbie-area-rework-design.md`
(tenet 8 §3, §4.1, §6, §10 chunk 0, §10.1)
**Scanner:** `tools/coord_inventory.py`

---

## 1. How room coordinates actually work (read this first)

There are **two** coordinate notions in DOGMud, and they are not the same thing:

1. **Engine-derived, per-zone (what the game uses at runtime).** The web/ASCII
   mapper (`internal/mapper/mapper.go`) builds each zone's map by BFS-crawling
   exits from the zone's root room, starting that root at `(0,0,0)` and applying
   a fixed per-direction delta for each compass exit
   (`posDeltas`: north `(0,-1,0)`, south `(0,1,0)`, east `(1,0,0)`,
   west `(-1,0,0)`, diagonals ±1 on both axes, up/down `±1` on z, plus
   `-x2`/`-x3` long variants). Collision/reciprocity checking
   (`mapper.consistency.go` → `CheckConsistency`, run at boot by
   `ValidateZoneConsistency` and on demand by the `cartcheck` admin command)
   operates **only within a single zone's crawl**. Each zone is its own
   coordinate space rooted at its own root room — the engine does **not**
   enforce a shared cross-zone plane.

2. **Stored `coord:` field (the world-layout plane authors maintain).** Most
   room YAMLs carry a block like:
   ```yaml
   coord:
     x: -11
     y: 8
     z: 0
   ```
   **The engine's `Room` struct does NOT load this field** (there is no `coord`
   tag in `internal/rooms/rooms.go`; YAML silently ignores it). It is an
   **authoring-discipline / documentation artifact**: authors hand-maintain it
   so the zones tile a single, globally-consistent world map with no two rooms
   anywhere sharing an `(x,y,z)` tuple. This shared plane is what the newbie-area
   spec's tenet 8 ("no newbie-area room may share an `(x,y,z)` tuple with any
   existing room anywhere else in the world") and its "402 unique coordinates
   across 15 zones" sanity pass are about.

**Consequence for this chunk:** the cross-zone uniqueness requirement is a
property of the stored `coord:` field, not of anything the engine validates.
The offline scanner therefore reads the stored `coord:` fields and checks global
uniqueness. (`cartcheck`/`ValidateZoneConsistency` remain useful for the
*intra-zone* geometry check once the new zone is authored, but they cannot see
cross-zone collisions.)

### Zone placement on the shared plane

Zones are not auto-offset by any config — their world position is simply the
range of `coord:` values their authors assigned. The existing zones tile a
shared plane spanning roughly `x[-24..18] y[-22..11] z[-3..1]`. **Ironwind
Steppe is the easternmost zone (max x = 18); nothing in the entire world has
x ≥ 19.** The whole eastern half-plane is empty — exactly the "ample empty
space east" the parent spec promised.

---

## 2. Scanner: `tools/coord_inventory.py`

A dependency-free (no YAML library) filename/line parser that walks
`_datafiles/world/dogmud/rooms/*/*.yaml`, extracts each room's stored
`coord:` block, and reports global + per-zone coordinate facts.

**Choice rationale:** a Python line-scanner of the stored `coord:` field is the
correct tool because the cross-zone world plane *lives in* that field — the
engine's own `cartcheck`/`ValidateZoneConsistency` only check intra-zone geometry
and could not detect a cross-zone collision even in principle. Replicating the
mapper's exit-crawl in Go would reproduce the per-zone view, not the shared
plane, so it would not answer the spec's question. The line-scanner directly
reproduces (and matches) the spec's "unique coordinates across N zones" sanity
pass.

Usage:
```bash
# Full report: totals, global + per-zone bounding boxes, any collisions
python tools/coord_inventory.py

# Verify a reserved box is empty of existing rooms (the acceptance check)
python tools/coord_inventory.py --check-region X0 X1 Y0 Y1 Z0 Z1

# JSON / single-zone variants
python tools/coord_inventory.py --json
python tools/coord_inventory.py --zone stillwater
```
Exit code `0` = clean; `2` = collisions found (or `--check-region` not empty).

---

## 3. Scan results (baseline, 2026-06-12)

```
Total room files          : 546
Rooms with stored coord   : 404
Rooms WITHOUT coord       : 142   (maze/instance/root rooms — not on the plane)
Unique (x,y,z) coordinates: 404
Global collisions         : 0
Global bounding box       : x[-24..18] y[-22..11] z[-3..1]
```

404 stored coordinates, **404 unique, zero collisions** — confirming (and very
slightly updating) the spec's "402 unique across 15 zones" baseline. The 142
coord-less files are the maze/instance/test zones (`a_dark_forest`,
`endless_trashheap`, `labyrinth*` partly, `shadow_realm`, `instance_*`,
`test_arena`) plus one root room per spatial zone — none participate in the
world plane.

Per-zone bounding boxes (15 spatial zones):

| Zone | Rooms | x-range | y-range | z-range |
|---|---:|---|---|---|
| Ashwick | 20 | [-7..-3] | [-15..-8] | [0..0] |
| Dustwalk Road | 10 | [-11..-9] | [-2..5] | [0..0] |
| Ironwind Steppe | 114 | **[8..18]** | [-14..11] | [0..0] |
| Labyrinth of Low Tunnels | 20 | [-13..-11] | [2..7] | [-2..0] |
| Marches Spur Road | 18 | [-11..-8] | [-13..-2] | [0..0] |
| North Road | 24 | [-20..-16] | [-13..0] | [-1..0] |
| Sanctum Basin | 23 | [-12..-7] | [7..11] | [0..0] |
| Stillwater | 49 | [-21..-14] | [1..8] | [-1..1] |
| Stillwater Marsh | 21 | [-24..-21] | [-4..4] | [0..0] |
| The Fernway | 10 | [-15..-9] | [-14..-13] | [0..0] |
| The Fernway South | 21 | [-16..-13] | [-22..-15] | [0..0] |
| Thornwall City | 57 | [0..8] | [-8..2] | [-3..1] |
| Thornwall Outskirts | 8 | [-4..-1] | [0..2] | [0..0] |
| Watchers Crossing | 8 | [-8..-5] | [-1..1] | [0..0] |
| World Road | 1 | [-10..-10] | [6..6] | [0..0] |

---

## 4. Reserved region — Pothole Coulee

```
RESERVED:  x ∈ [30 .. 59]   (30 columns wide)
           y ∈ [-15 .. 14]  (30 rows tall)
           z ∈ [-3 .. 3]    (7 vertical levels)
HUB ANCHOR (zone root, the Awakening plunge-pool room): (x=45, y=0, z=0)
```

- **Capacity:** 30 × 30 × 7 = 6,300 cells for ~170 rooms — roughly **37×
  headroom**. "Generous block + margin" as the spec directs.
- **Placement = due east of the existing world.** Ironwind Steppe's eastern
  edge is x = 18; the reserve starts at x = 30, leaving an **11-column empty
  buffer** (x 19–29) so Pothole Coulee can never become accidentally adjacent
  to Ironwind. Verified: **no existing room anywhere has x ≥ 19**, so the entire
  block is guaranteed empty regardless of margin. This also fits the fiction —
  §4.1 places the coulee in "the channeled scablands east of the Columbia
  River."
- **Why east and not north?** North (smaller y) is more crowded — The Fernway
  South already reaches y = -22 and several road zones sit in the northern
  band — whereas the eastern half-plane (x ≥ 19) is **completely empty**. East
  gives the cleanest, most future-proof buffer.

### Hub-anchored radiation budget

The hub root anchors at the **center** of the block, `(45, 0, 0)`, so all seven
active spokes plus the reserved Spoke H can radiate outward with balanced room.
From the hub center to each block edge:

| Direction | Delta (mapper) | Cells to edge from (45,0) | Spoke (per §6.1 topology) |
|---|---|---|---|
| East | x +1 | x 46→59 = **14** | **B — Forge** (`HUB──B inner──…──B outer`) |
| West | x -1 | x 44→30 = **15** | **E — Folding** (`E outer──…──E inner──HUB`) |
| North | y -1 | y -1→-15 = **15** | **A — Martial** / **F — Lore** (NE/NW arms) |
| South | y +1 | y 1→14 = **14** | **D — Wilderness** / **C — Alchemy** (S/SE arms) |
| Diagonals | ±1,±1 | ≥14 to each corner | lateral outer-ring connectors `══` |
| Up/Down | z ±1 | z 0→±3 = **3 each** | mine shafts (B↓), observatory/bluff terraces (E,G↑), caves |

A spoke is ~20 rooms of concentric rings (inner 4–6, middle 5–8, outer 5–8). A
ring does not run in a straight line — it meanders and the rings stack
perpendicular to the spoke axis — so a 20-room spoke needs roughly 10–15 linear
cells of reach plus a few cells of lateral spread. Every cardinal arm has ≥14
cells and every diagonal corner ≥14, comfortably inside budget. The 7-level z
range covers all vertical content (Spoke B's mine descends, Spoke E's
observatory ruin and Spoke G's bluff-top terraces ascend, cave systems dip
below).

**Suggested per-spoke sub-allotments** (soft guidance for chunk authors — the
only hard rule is global uniqueness; stay inside the reserved box and these keep
spokes from crowding each other):

| Spoke | Primary heading from hub | Suggested sub-box |
|---|---|---|
| Hub town | center | x[40..50] y[-5..5] z[-1..1] |
| A — Martial | north / north-east | x[45..55] y[-15..-3] z[0..2] |
| B — Forge | east (mine ↓) | x[51..59] y[-6..6] z[-3..0] |
| C — Alchemy | south-east | x[48..58] y[3..14] z[-1..0] |
| D — Wilderness | south / south-west | x[33..45] y[3..14] z[0..0] |
| E — Folding | west (observatory ↑) | x[30..40] y[-6..6] z[0..3] |
| F — Lore | north-west | x[33..44] y[-15..-3] z[0..1] |
| G — Ranged | east / north-east bluffs (↑) | x[51..59] y[-15..-7] z[1..3] |
| H — *future slot* | unlocated (no hub exit yet) | leave the SW corner x[30..39] y[8..14] spare |
| Lateral connectors | outer-ring `══` links | wherever adjacent spoke outer rings meet |

These are advisory; the binding constraint is the acceptance rule below.

---

## 5. How future chunks verify against this budget (acceptance rule)

Every newbie-area chunk's coordinate-uniqueness audit (parent spec §10.1) is:

1. **Author each new room with a stored `coord:` block inside the reserved box**
   `x[30..59] y[-15..14] z[-3..3]`, consistent with its exits (an `east` exit
   must lead to the room at `x+1`, etc. — matching the mapper deltas in §1 so
   the intra-zone `cartcheck` also passes).

2. **Run the global scanner — it must report zero collisions:**
   ```bash
   python tools/coord_inventory.py
   # PASS iff "Global collisions : 0"   (exit code 0)
   ```
   This catches any new room colliding with another new room **or** with any
   existing world room (Sanctum Basin still lives until Chunk 10, so both must
   stay collision-free against each other and everything else).

3. **Confirm the new rooms landed inside the reserve** (catches stray coords):
   ```bash
   # Everything OUTSIDE the reserve, in the eastern half-plane, must stay empty
   # except the new zone — i.e. the reserve box must contain all & only Pothole
   # Coulee rooms. Quick emptiness check of a not-yet-built sub-region:
   python tools/coord_inventory.py --check-region <X0> <X1> <Y0> <Y1> <Z0> <Z1>
   # EMPTY  => clear to build there
   ```

4. **Intra-zone geometry** is separately verified by booting the server (boot
   runs `ValidateZoneConsistency`) and/or the `cartcheck pothole_coulee` admin
   command once the zone has a folder — these confirm reciprocal exits and that
   the exit-crawl is self-consistent. Keep `GamePlay.MapConsistencyEnforce` at
   `warn` during authoring; flip to `panic` only when `cartcheck` is clean.

**A chunk fails its coordinate audit if step 2 reports any collision, or step 3
shows a new room outside the reserved box.**

---

## 6. Connector corridor reservation (added 2026-06-12, user note)

The chunk-10 cutover attaches the zone to the wider world via the
"long hike out." That hike must also be collision-free, so its corridor
is reserved NOW, not improvised at cutover:

- **Corridor reserve: x[19..29], y[-6..6], z[0..0]** (11 columns × 13
  rows = 143 cells). Verified EMPTY at reservation time
  (`coord_inventory.py --check-region 19 29 -6 6 0 0` → EMPTY; the entire
  half-plane x>=19 holds nothing but the Pothole reserve).
- **West end:** attaches to Ironwind Steppe's eastern rim (Ironwind spans
  x[8..18], y[-14..11]; the cutover sub-spec picks the exact rim room and
  may extend the corridor's y-band if the chosen room sits outside
  y[-6..6] — extend the reservation FIRST, re-running the emptiness
  check).
- **East end:** enters Pothole Coulee through one badlands-edge room on
  the zone's western boundary (a spoke-B/D outer-ring exterior room, or a
  dedicated "Coulee Rim" room; chunk-10 decision).
- **Shape:** deliberately winding (the spec wants the hike long and
  arduous; a snake path through the corridor cells also leaves room to
  dodge any content that lands east of Ironwind before cutover — though
  this reservation makes that a collision the OTHER content would have to
  avoid: this corridor + the Pothole box are reserved first).
- **Rooms:** ~12–20 trail rooms, authored in chunk 10 (badlands/steppe
  biomes, NOT sanctuary, real-world difficulty — this is the road OUT,
  not part of the tutorial).

Any future non-newbie content that wants coordinates at x>=19 must
consult this document first; the newbie reservations take precedence
until chunk 10 lands and converts reservations into real rooms.
