# Stillwater Marsh Zone Build Implementation Plan (Stage 3.0a)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Stillwater Marsh zone — 20 rooms west of Stillwater, themed as wetland tapering to upland steppe at the south terminus. Reuses the mustelid species shipped in 3.0c T1 for the river otter (first non-badger consumer of that species). 5 wildlife mobs, only 1 hostile (bog adder with `hates: [rodent]` for an emergent adder-vs-rat hunt dynamic).

**Architecture:** Pure data zone build — no Go code changes. 20 room YAMLs in a new `stillwater_marsh/` folder, 5 mob YAMLs (4 reuse existing species, 1 reuses mustelid 24 from 3.0c), 1 zone-config YAML, 1 boundary edit on Stillwater 4133 Mill Creek Footbridge (add a west exit), coord_map update (Stillwater catch-up + new zone), 1 PATCH_NOTES entry. Mirrors the 3.0c Fernway South 8-task subagent-driven shape.

**Tech Stack:** YAML data files. Existing engine systems: room loader, mob loader, species registry, behavior_archetype dispatcher, spawn timer, coord/map renderer. No new code.

**Sketch:** `docs/superpowers/specs/2026-04-28-stillwater-marsh-zone-sketch.md` (committed 0740be03; user approved).

---

## File Structure

| Action | File | Responsibility |
|---|---|---|
| CREATE | `_datafiles/world/dogmud/rooms/stillwater_marsh/zone-config.yaml` | Zone metadata |
| CREATE | `_datafiles/world/dogmud/rooms/stillwater_marsh/4177.yaml` … `4196.yaml` (20 files) | Individual rooms |
| MODIFY | `_datafiles/world/dogmud/rooms/stillwater/4133.yaml` | Add `west: roomid: 4177` exit |
| CREATE | `_datafiles/world/dogmud/mobs/stillwater_marsh/366-river_otter.yaml` | Prey, mustelid (24 — reuses 3.0c species) |
| CREATE | `_datafiles/world/dogmud/mobs/stillwater_marsh/367-marsh_rat.yaml` | Prey, rodent |
| CREATE | `_datafiles/world/dogmud/mobs/stillwater_marsh/368-dragonfly_swarm.yaml` | Combat-passive, insectoid |
| CREATE | `_datafiles/world/dogmud/mobs/stillwater_marsh/369-snapping_turtle.yaml` | Combat-passive, reptile |
| CREATE | `_datafiles/world/dogmud/mobs/stillwater_marsh/370-bog_adder.yaml` | Ambusher, serpent, HOSTILE, hates: [rodent] |
| MODIFY | `docs/coordinate_map.md` | Add 48 Stillwater rows (catch-up; doc was stale) + 20 Stillwater Marsh rows |
| MODIFY | `PATCH_NOTES.md` | Stage 3.0a dev-only entry |

---

## Hard reference: room layout (memorize this — every room task references it)

Coords are unique. All unclaimed per `docs/coordinate_map.md` and direct YAML survey of Stillwater rooms (Stillwater occupies x=-21 to -14, y=1 to 8, with Cemetery 4140 at (-21,5,0)). The new zone uses x=-21 to -24, y=-4 to 3, avoiding 4140.

```
        x=-24    x=-23    x=-22    x=-21    x=-20
y=3                                [4180]                       <- Spring Pool (terminal N)
y=2                                [4179]                       <- Mill Creek Source
y=1     [4182]──[4181]──[4178]──[4177]──{4133 Footbridge}      <- entry from Stillwater
y=0     [4186]──[4185]──[4184]──[4183]                          <- Heron Marsh hub at 4184
y=-1    [4190]──[4189]──[4188]──[4187]                          <- forage row
y=-2             [4193]──[4192]──[4191]                         <- Adder Den at 4193 (terminal W)
y=-3                     [4194]   [4196]                        <- 4196 = Hidden Spring (terminal SE)
y=-4                     [4195]                                 <- Far Bog Heart (terminal S, biome plains)
```

| ID | Title | Coord | Biome | Exits |
|---|---|---|---|---|
| 4177 | Marsh Track | (-21,1,0) | water | n→4179, s→4183, e→4133 (Stillwater), w→4178 |
| 4178 | Cattail Verge | (-22,1,0) | water | e→4177, w→4181 |
| 4179 | Mill Creek Source | (-21,2,0) | water | s→4177, n→4180 |
| 4180 | Spring Pool | (-21,3,0) | water | s→4179 (terminal N) |
| 4181 | Reed Beds | (-23,1,0) | water | e→4178, w→4182 |
| 4182 | Willow Grove | (-24,1,0) | water | e→4181 (terminal W) |
| 4183 | Cattail Bend | (-21,0,0) | water | n→4177, s→4187, w→4184 |
| 4184 | Heron Marsh | (-22,0,0) | water | central hub: e→4183, w→4185, s→4188 |
| 4185 | Otter Slide | (-23,0,0) | water | e→4184, w→4186, s→4189 |
| 4186 | Clam Beds | (-24,0,0) | water | e→4185, s→4190 (terminal W) |
| 4187 | Iron Seep | (-21,-1,0) | water | n→4183, s→4191 |
| 4188 | Shrimp Shallows | (-22,-1,0) | water | n→4184, s→4192, e→4187, w→4189 |
| 4189 | Sundew Hollow | (-23,-1,0) | water | n→4185, e→4188, w→4190, s→4193 |
| 4190 | Black Pool | (-24,-1,0) | water | n→4186, e→4189 (terminal W — rare pearl) |
| 4191 | Mossy Hummock | (-21,-2,0) | water | n→4187, s→4196, w→4192 |
| 4192 | Dragonfly Glade | (-22,-2,0) | water | n→4188, e→4191, w→4193, s→4194 |
| 4193 | Adder Den | (-23,-2,0) | water | n→4189, e→4192 (terminal W — **HOSTILE bog adder**) |
| 4194 | Bog Edge | (-22,-3,0) | water | n→4192, s→4195 |
| 4195 | Far Bog Heart | (-22,-4,0) | **plains** | n→4194 (terminal S — biome shift to upland) |
| 4196 | Hidden Spring | (-21,-3,0) | water | n→4191 (terminal SE pocket) |

---

## CRITICAL voice rule — plural ANSI tags (LEARNED FROM 3.0c SMOKE)

When a description prose word is plural AND highlighted, the `s` (or `es`) MUST go INSIDE the `<ansi fg="itemname">...</ansi>` tag, NOT outside. AND the `nouns:` block must include both singular and plural forms as separate keys (with the same definition).

❌ WRONG — `s` outside the tag breaks `look hare-paths`:
```
... a network of <ansi fg="itemname">hare-path</ansi>s threads ...

nouns:
  hare-path: |
    A network of low cropped passages ...
```

✅ RIGHT — `s` inside the tag, plural alias added:
```
... a network of <ansi fg="itemname">hare-paths</ansi> threads ...

nouns:
  hare-path: |
    A network of low cropped passages ...
  hare-paths: |
    A network of low cropped passages ...
```

(Duplicate the prose. YAML anchors are not used elsewhere in room data — risk that the loader doesn't handle them. Plain duplication is safe.)

This was a real bug in 3.0c (commit 0907023e fixed 5 instances across 3 rooms). Stage 3.0a should NOT repeat it.

---

## Task 1: Zone-config + entry hub + north spur (4177, 4179, 4180)

**Files (CREATE all 4):**
- `_datafiles/world/dogmud/rooms/stillwater_marsh/zone-config.yaml`
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4177.yaml` — Marsh Track (entry from Stillwater 4133)
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4179.yaml` — Mill Creek Source (1 north of entry)
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4180.yaml` — Spring Pool (terminal N, where the creek is born)

These are the player's first three rooms — focus voice on "you've crossed from town into wild marsh."

### REFERENCE the established Stillwater + Fernway South voice

BEFORE writing, read for voice:
- `_datafiles/world/dogmud/rooms/stillwater/4141.yaml` (Sluice Pond — closest existing wetland room, biome:water)
- `_datafiles/world/dogmud/rooms/the_fernway_south/4157.yaml` (Briar Tangle — recent precedent for a zone entry room)
- `_datafiles/world/dogmud/rooms/stillwater/4133.yaml` (Mill Creek Footbridge — the room directly east of our entry; references the creek)

Match voice precisely: sensory-led naturalist; ANSI tags wrapping noun keywords with `<ansi fg="itemname">word</ansi>`; 4-7 sentences per description (YAML `description: >` folded scalar wrapped at ~78 chars); each `nouns:` entry is 1-2 sentences; idle messages are short atmospheric one-liners (lowercase).

### Step 1: Create the zone-config

```yaml
name: Stillwater Marsh
roomid: 4177
defaultbiome: water
region: Windward Marches
```

### Step 2: Create 4177 Marsh Track (entry, 4-way junction)

```yaml
roomid: 4177
zone: Stillwater Marsh
title: Marsh Track
description: >
  [4-7 sentences. THEME: you've crossed west from Mill Creek
  Footbridge and the cultivated edge of Stillwater drops away
  within a dozen paces. The footbridge's plank-track gives over
  to a board-trail laid across soft ground; you can hear the
  creek but no longer see its dam. Senses: water everywhere
  underfoot; reed-rustle; a different smell than the pond proper
  — wilder, peatier; the south wind off the bog. Required ANSI
  nouns: board-trail, marsh-water, reeds, creek-sound. Watch
  the plural rule: if you write "reeds" with the ansi tag, the
  s goes INSIDE the tag and you add a `reeds:` key alongside
  `reed:` in the nouns block.]
biome: water
coord:
  x: -21
  y: 1
  z: 0
exits:
  east:
    roomid: 4133
    zone: Stillwater
  west:
    roomid: 4178
  north:
    roomid: 4179
  south:
    roomid: 4183
nouns:
  board-trail: |
    [1-2 sentences. Split-cedar boards laid two-wide across the
    soft marsh ground; some plank-ends sunk and dark with damp.]
  marsh-water: |
    [1-2 sentences. Standing water visible everywhere off the
    board-trail's edges — peaty-brown, slow, knee-deep at most
    but cold and stiller than the creek.]
  reed: |
    [1-2 sentences. Tall marsh-reeds stand in dense stands either
    side of the trail, the seed-heads pale brown and steady in
    the wind.]
  reeds: |
    [1-2 sentences. Tall marsh-reeds stand in dense stands either
    side of the trail, the seed-heads pale brown and steady in
    the wind.]
  creek-sound: |
    [1-2 sentences. Off east the creek-sound continues from the
    sluice and the watermill — a constant water-murmur that
    drops behind as you move west.]
idlemessages:
  - 'a board-end creaks under your weight'
  - ''
  - 'a single reed bends in the southerly wind'
  - ''
  - 'a marsh-bird croaks once somewhere south, then is quiet'
```

The `reed` / `reeds` dual-key pattern is the plural rule in action. Use it everywhere a noun appears in plural form in the prose.

### Step 3: Create 4179 Mill Creek Source

```yaml
roomid: 4179
zone: Stillwater Marsh
title: Mill Creek Source
description: >
  [4-7 sentences. THEME: north of Marsh Track the board-trail
  bends along the creek's actual upstream — narrower and shallower
  than the pond version, running over flat stones and cool dark
  peat. This is where the millstream begins. Senses: water-sound
  rises again; cold pebbled bottom visible in the shallow channel;
  watercress at the edges. Required ANSI nouns: shallow-creek,
  flat-stones, watercress, peat-bank. Apply plural rule.]
biome: water
coord:
  x: -21
  y: 2
  z: 0
exits:
  south:
    roomid: 4177
  north:
    roomid: 4180
nouns:
  shallow-creek: |
    [1-2 sentences. The creek runs perhaps a hand's breadth deep
    over a flat stone bed, slow but clear, the water cold to
    the touch.]
  flat-stones: |
    [1-2 sentences. The creek-bed is paved with flat dark stones,
    visible through the shallow water — worn smooth by long
    slow current.]
  watercress: |
    [1-2 sentences. Watercress grows thick along both banks,
    pungent and peppery — the same kind that flourishes south at
    Watercress Bend in Fernway South, abundant here.]
  peat-bank: |
    [1-2 sentences. The banks are cool dark peat, springy underfoot
    where you step off the boards — the kind of soil that holds
    every print and every bootful of water.]
idlemessages:
  - 'a watercress leaf detaches and rides north on the creek'
  - ''
  - 'a small minnow flickers across the flat-stones and is gone'
  - ''
  - 'water gurgles softly between two of the larger stones'
```

### Step 4: Create 4180 Spring Pool (terminal N)

```yaml
roomid: 4180
zone: Stillwater Marsh
title: Spring Pool
description: >
  [4-7 sentences. THEME: the creek's actual source — a clear deep
  spring-pool rising from a fissure in dark stone. The deepest
  water in the zone. Terminal — the trail ends here at a ring of
  worn pilgrim-stones. Senses: cold cold cold — the water is much
  colder than the rest of the marsh; visibly bottomless even
  though the pool is small; a reverence-feel like the Old Stand
  oaks south in Fernway. Required ANSI nouns: spring-pool,
  pilgrim-stones, cold-water, dark-fissure. Apply plural rule
  for "stones".]
biome: water
coord:
  x: -21
  y: 3
  z: 0
exits:
  south:
    roomid: 4179
nouns:
  spring-pool: |
    [1-2 sentences. The creek's actual birth — a small deep pool
    perhaps four paces across, the water rising clear and cold
    from a fissure visible only as darkness in the stone bed.]
  pilgrim-stone: |
    [1-2 sentences. A ring of low standing-stones around the pool's
    edge, smooth-worn at the tops where generations of hands have
    rested while drinking. Older than the watermill below by an
    order.]
  pilgrim-stones: |
    [1-2 sentences. A ring of low standing-stones around the pool's
    edge, smooth-worn at the tops where generations of hands have
    rested while drinking. Older than the watermill below by an
    order.]
  cold-water: |
    [1-2 sentences. The pool's water is cold enough to numb a
    hand within a count of ten — a deep-aquifer cold that the
    surface marsh never reaches.]
  dark-fissure: |
    [1-2 sentences. The pool's source visible only as a deeper
    dark in the stone bed — a vertical crack from which the
    spring rises continuously and silently.]
idlemessages:
  - 'a single bubble breaks the spring-pool's surface'
  - ''
  - 'cold air drifts up off the water'
  - ''
  - 'the surface stays so still that the sky reflects perfectly'
```

### Step 5: Build verify

`go build ./...` — clean. Boot test optional.

### Step 6: Commit

```bash
git add _datafiles/world/dogmud/rooms/stillwater_marsh/
git commit -m "$(cat <<'EOF'
feat(rooms): Stillwater Marsh zone-config + entry trio (4177, 4179, 4180)

Marsh Track (entry from Mill Creek Footbridge), Mill Creek Source
(creek's upstream), Spring Pool (terminal N — the creek's actual
birth at a cold deep spring). Zone-config + 3 rooms. Stage 3.0a.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: West spur (4178, 4181, 4182)

**Files (CREATE all 3):**
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4178.yaml` — Cattail Verge
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4181.yaml` — Reed Beds
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4182.yaml` — Willow Grove (terminal W)

A 3-room west-extending strip along the y=1 row. Theme: cattails → reeds → willows, getting deeper into wild marsh.

Reference for voice: 4177 (just shipped in T1) and 4141 Sluice Pond.

### Step 1: Create 4178 Cattail Verge

```yaml
roomid: 4178
zone: Stillwater Marsh
title: Cattail Verge
description: >
  [4-7 sentences. THEME: west of Marsh Track the trail runs
  through a stand of cattails head-high on either side, broken
  by lake-mint at the verges. This is forager territory —
  cattail-down (40055) and lake mint (40057) are the visible
  forage. Senses: cattail-down drifting in the air like
  cottonwood; pungent lake-mint underfoot when crushed; trail
  narrows here. Required ANSI nouns: cattail, cattail-down,
  lake-mint, mud-track. Apply plural rule for cattails / cattail-down
  / lake-mints.]
biome: water
coord:
  x: -22
  y: 1
  z: 0
exits:
  east:
    roomid: 4177
  west:
    roomid: 4181
nouns:
  cattail: |
    [1-2 sentences. Tall cattails stand head-high on both sides
    of the trail, the brown sausage-shaped seed-heads ripe and
    splitting open in places — the silky down they release is
    the foraged crop.]
  cattails: |
    [1-2 sentences. Tall cattails stand head-high on both sides
    of the trail, the brown sausage-shaped seed-heads ripe and
    splitting open in places — the silky down they release is
    the foraged crop.]
  cattail-down: |
    [1-2 sentences. Drifts of soft cattail-down hang in the
    air at this time of season — released from split seed-heads,
    drifting slowly in any movement of air. Stillwater spinners
    work it into thread for cold-weather cloak linings.]
  lake-mint: |
    [1-2 sentences. Patches of lake-mint grow at the verges
    where the cattails thin — pungent and sharp underfoot
    when crushed, the small leaves silver-green.]
  mud-track: |
    [1-2 sentences. The board-trail gives over here to a
    packed-mud track, narrower and softer underfoot — the
    marsh swallows the boards if the millers don't replace
    them often.]
idlemessages:
  - 'a tuft of cattail-down drifts past at head-height'
  - ''
  - 'lake-mint releases a sharp green smell where you step'
  - ''
  - 'a cattail-head splits audibly somewhere off the trail'
```

### Step 2: Create 4181 Reed Beds

```yaml
roomid: 4181
zone: Stillwater Marsh
title: Reed Beds
description: >
  [4-7 sentences. THEME: west of Cattail Verge the cattails give
  way to dense pure reed-bed — the grandest stand in the zone,
  reeds rising taller than a person and rustling continuously in
  any breeze. Senses: reed-rustle dominant — almost loud; the
  sense of being inside a moving wall; flat green-yellow color
  in every direction; very few prints in the soft mud. Required
  ANSI nouns: reed-bed, mud-bottom, wind-rustle, lost-feel.
  Apply plural rule for reeds / reed-beds.]
biome: water
coord:
  x: -23
  y: 1
  z: 0
exits:
  east:
    roomid: 4178
  west:
    roomid: 4182
nouns:
  reed-bed: |
    [1-2 sentences. The largest single stand of reeds in the
    zone — perhaps an acre of head-high marsh-reed pressing
    in on the trail from every side, the green stems thick
    enough that ten paces in is invisible from the trail.]
  reed-beds: |
    [1-2 sentences. The largest single stand of reeds in the
    zone — perhaps an acre of head-high marsh-reed pressing
    in on the trail from every side, the green stems thick
    enough that ten paces in is invisible from the trail.]
  mud-bottom: |
    [1-2 sentences. The trail is little more than packed-mud
    here, soft enough that boots leave deep prints. Off the
    trail the mud is bottomless — knee-deep at first then
    suddenly deeper.]
  wind-rustle: |
    [1-2 sentences. The reeds move continuously in any breath
    of air, a steady high rustle that fills the room and
    drowns out subtler sounds.]
  lost-feel: |
    [1-2 sentences. The visual sameness of the reeds in every
    direction makes orientation difficult — leaving the trail
    even briefly is a recipe for not finding it again.]
idlemessages:
  - 'wind moves through the reeds in a long high rustle'
  - ''
  - 'a marsh-bird calls once from somewhere unseen'
  - ''
  - 'the trail surface trembles slightly underfoot — soft mud below'
```

### Step 3: Create 4182 Willow Grove (terminal W)

```yaml
roomid: 4182
zone: Stillwater Marsh
title: Willow Grove
description: >
  [4-7 sentences. THEME: the trail ends in a stand of grey-barked
  marsh-willows, the last solid ground on the west spur. This is
  forager territory for marsh willow bark (40056). Terminal — no
  further west. Senses: willow-leaf shimmer; greyish bark with
  fine furrows; cooler air than the reed-beds; a sense of "end of
  trail." Required ANSI nouns: willow, willow-bark, leaf-shimmer,
  trail-end. Apply plural rule for willows / willow-leaves.]
biome: water
coord:
  x: -24
  y: 1
  z: 0
exits:
  east:
    roomid: 4181
nouns:
  willow: |
    [1-2 sentences. Six or seven grey-barked marsh-willows form
    a small grove at the trail's end, leaning over the standing
    water — the same kind the Stillwater healer harvests for
    her stronger pain-medicines.]
  willows: |
    [1-2 sentences. Six or seven grey-barked marsh-willows form
    a small grove at the trail's end, leaning over the standing
    water — the same kind the Stillwater healer harvests for
    her stronger pain-medicines.]
  willow-bark: |
    [1-2 sentences. The bark is greyish and finely furrowed —
    bitter to the tongue and known to ease aches and reduce
    fevers. The healer in Stillwater pays well for a clean strip.]
  leaf-shimmer: |
    [1-2 sentences. The willow leaves are narrow and silvery on
    their undersides; in any breeze the grove shimmers as the
    leaves turn.]
  trail-end: |
    [1-2 sentences. The trail simply ends here — the last
    plank-foot lost in the mud where the willows take over and
    nothing has bothered to walk further west. There is nothing
    west of here you can reach today.]
idlemessages:
  - 'willow-leaves shimmer briefly silver in a passing breeze'
  - ''
  - 'a leaf detaches and lands flat on the still water'
  - ''
  - 'a frog plops into the marsh somewhere in the grove'
```

### Step 4: Build verify + commit

```bash
go build ./...

git add _datafiles/world/dogmud/rooms/stillwater_marsh/
git commit -m "$(cat <<'EOF'
feat(rooms): Stillwater Marsh west spur (4178, 4181, 4182)

Cattail Verge (cattail + lake-mint forage), Reed Beds (the
zone's biggest reed stand), Willow Grove (terminal W — marsh
willow bark forage). Stage 3.0a.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Heron Marsh row (4183, 4184, 4185, 4186)

**Files (CREATE all 4):**
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4183.yaml` — Cattail Bend (S of entry)
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4184.yaml` — Heron Marsh (central hub, S of Cattail Bend's W neighbor)
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4185.yaml` — Otter Slide (W of Heron Marsh)
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4186.yaml` — Clam Beds (terminal W)

The y=0 row — the marsh's main artery. Includes the central hub (Heron Marsh, 4184) and the otter spawn (4185).

### Step 1: Create 4183 Cattail Bend

```yaml
roomid: 4183
zone: Stillwater Marsh
title: Cattail Bend
description: >
  [4-7 sentences. THEME: south of Marsh Track the board-trail
  bends sharply through another stand of cattails before opening
  into the central marsh. Senses: trail bends visibly; cattails
  again but mixed with frog-spawn pools; the air gets heavier
  going south. Required ANSI nouns: bend, cattail-stand,
  frog-spawn, trail-board. Apply plural rule.]
biome: water
coord:
  x: -21
  y: 0
  z: 0
exits:
  north:
    roomid: 4177
  south:
    roomid: 4187
  west:
    roomid: 4184
nouns:
  bend: |
    [1-2 sentences. The trail makes a sharp bend here, swinging
    west around a denser stand of cattails before continuing
    south.]
  cattail-stand: |
    [1-2 sentences. A second cattail-stand, less dense than the
    Cattail Verge stand to the north — the seed-heads here are
    older and many have already broken open.]
  frog-spawn: |
    [1-2 sentences. Pools of frog-spawn float in the still water
    off the trail's edge — clear jelly with dark spots, dozens
    of egg-masses overlapping. The bullfrogs have been busy.]
  trail-board: |
    [1-2 sentences. A few warped trail-boards lie at the bend,
    half-pulled from the mud — older than the rest of the
    boardwalk, replaced piece by piece over the years.]
idlemessages:
  - 'a frog plops from the trail-edge into the still water'
  - ''
  - 'a cattail-head pops audibly somewhere south'
  - ''
  - 'the boards groan slightly as you cross the bend'
```

### Step 2: Create 4184 Heron Marsh (central hub, 3-way junction)

```yaml
roomid: 4184
zone: Stillwater Marsh
title: Heron Marsh
description: >
  [4-7 sentences. THEME: a wide shallow pond opens here at the
  center of the southern marsh — open water, scattered tussocks,
  reeds at the edges. A grey heron is sometimes here, sometimes
  not (atmospheric only — no mob). This is the hub: branches
  east, west, and south. Senses: open water for the first time;
  sky reflected; long sightlines after the closed reed-trails;
  heron-call may sound somewhere. Required ANSI nouns:
  open-pond, tussocks, heron-track, sky-reflection. Apply plural
  rule for tussocks.]
biome: water
coord:
  x: -22
  y: 0
  z: 0
exits:
  east:
    roomid: 4183
  west:
    roomid: 4185
  south:
    roomid: 4188
nouns:
  open-pond: |
    [1-2 sentences. A wide shallow pond perhaps thirty paces
    across, the water clear enough to see weeds bowing on the
    bottom. The first real open water in the zone.]
  tussock: |
    [1-2 sentences. Round grass-tussocks rise from the pond's
    shallow bottom like small green islands — a heron's
    stalking ground, a frog's hideaway.]
  tussocks: |
    [1-2 sentences. Round grass-tussocks rise from the pond's
    shallow bottom like small green islands — a heron's
    stalking ground, a frog's hideaway.]
  heron-track: |
    [1-2 sentences. Three-toed heron prints mark the soft mud
    at the pond's east edge — fresh enough that they haven't
    filled with water. The grey heron may be here today; may
    not.]
  sky-reflection: |
    [1-2 sentences. The pond's still water mirrors the sky
    perfectly when the wind drops — clouds in the water the
    same as clouds above.]
idlemessages:
  - 'a heron call drifts across the pond from somewhere unseen'
  - ''
  - 'a fish-ring expands silently and fades'
  - ''
  - 'a tussock bends as something small moves across it'
```

### Step 3: Create 4185 Otter Slide

```yaml
roomid: 4185
zone: Stillwater Marsh
title: Otter Slide
description: >
  [4-7 sentences. THEME: west of Heron Marsh the bank steepens
  briefly — a worn otter-slide grooves the mud where the
  river otters have polished a launching path into the water.
  This is the otter spawn. Senses: dark fur-musk; clear smooth
  slide-track in the mud; sometimes splashing; the otter may
  or may not be present. Required ANSI nouns: otter-slide,
  bank, fur-musk, water-tracks. Apply plural rule for tracks.]
biome: water
coord:
  x: -23
  y: 0
  z: 0
exits:
  east:
    roomid: 4184
  west:
    roomid: 4186
  south:
    roomid: 4189
nouns:
  otter-slide: |
    [1-2 sentences. A polished smooth slide-track curves from
    the bank-top down into the water — the otter's launching
    path, used so often the mud is hard and bright underneath.]
  bank: |
    [1-2 sentences. A short steep bank of dark peaty earth —
    the only real elevation in the marsh interior, perhaps a
    body's height above the water.]
  fur-musk: |
    [1-2 sentences. A thin musky animal smell hangs around the
    slide — the otter's scent, distinctive and not unpleasant
    to those who know what they're smelling.]
  water-track: |
    [1-2 sentences. Wet paw-prints lead from the slide's bottom
    a few paces along the bank-edge before disappearing into
    the water — webbed and clearly marked in the mud.]
  water-tracks: |
    [1-2 sentences. Wet paw-prints lead from the slide's bottom
    a few paces along the bank-edge before disappearing into
    the water — webbed and clearly marked in the mud.]
idlemessages:
  - 'a small splash from the water near the slide-bottom'
  - ''
  - 'the fur-musk briefly strengthens'
  - ''
  - 'a single ripple expands silently from where the slide enters'
```

### Step 4: Create 4186 Clam Beds (terminal W)

```yaml
roomid: 4186
zone: Stillwater Marsh
title: Clam Beds
description: >
  [4-7 sentences. THEME: west of Otter Slide the marsh shallows
  out into a wide tract of clam beds — soft sandy bottom under a
  hand's depth of clear water, freshwater clams (40058) thick in
  the mud. Forager territory. Terminal W on this row but exits
  south to Black Pool (4190). Senses: shallow clear water; the
  clams visible as ridges in the sand; egret-tracks; quietest
  room yet. Required ANSI nouns: clam-bed, sand-bottom,
  clam-shells, egret-track. Apply plural rule for shells/beds.]
biome: water
coord:
  x: -24
  y: 0
  z: 0
exits:
  east:
    roomid: 4185
  south:
    roomid: 4190
nouns:
  clam-bed: |
    [1-2 sentences. A wide flat bed of soft sand under a hand's
    depth of clear water, dimpled with the small ridges of
    buried freshwater clams.]
  clam-beds: |
    [1-2 sentences. A wide flat bed of soft sand under a hand's
    depth of clear water, dimpled with the small ridges of
    buried freshwater clams.]
  sand-bottom: |
    [1-2 sentences. The bottom is fine pale sand, an unusual
    colour for the marsh interior — washed in over centuries
    from some upstream source, the foragers say.]
  clam-shell: |
    [1-2 sentences. Empty clam-shells lie at the bed's high-water
    edge — pale grey-blue, the size of a thumbnail to the size
    of a palm. The mussel-shells are usable for buttons and
    inlays, prized by the carver in town.]
  clam-shells: |
    [1-2 sentences. Empty clam-shells lie at the bed's high-water
    edge — pale grey-blue, the size of a thumbnail to the size
    of a palm. The mussel-shells are usable for buttons and
    inlays, prized by the carver in town.]
  egret-track: |
    [1-2 sentences. Three-toed bird-tracks, smaller than the
    Heron Marsh prints — a snowy egret's, working the bed for
    small fish.]
idlemessages:
  - 'a clam ejects a small spurt of water that breaks the surface'
  - ''
  - 'an egret-track in the wet sand fills slowly with water'
  - ''
  - 'a single bubble rises from the sand and breaks'
```

### Step 5: Build verify + commit

```bash
go build ./...

git add _datafiles/world/dogmud/rooms/stillwater_marsh/
git commit -m "$(cat <<'EOF'
feat(rooms): Stillwater Marsh hub row (4183-4186)

Cattail Bend (S of entry, frog-spawn pools), Heron Marsh (central
hub, open water, atmospheric heron), Otter Slide (otter spawn),
Clam Beds (terminal W — freshwater clam forage). Stage 3.0a.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Forage row (4187, 4188, 4189, 4190)

**Files (CREATE all 4):**
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4187.yaml` — Iron Seep
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4188.yaml` — Shrimp Shallows
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4189.yaml` — Sundew Hollow
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4190.yaml` — Black Pool (terminal W — rare pearl)

The y=-1 row. Each room is forager territory for a different existing Stillwater mat. 4188 has the snapping turtle spawn; 4190 is the rare-pearl deep terminal.

### Step 1: Create 4187 Iron Seep

```yaml
roomid: 4187
zone: Stillwater Marsh
title: Iron Seep
description: >
  [4-7 sentences. THEME: south of Cattail Bend the marsh narrows
  to a single broad seep where dark heavy nodules glint under the
  shallow water — lake-iron (40059), the same kind precipitated
  out at Sluice Pond. Forager territory. Senses: faint iron-tang
  in the air; rust-colored stains on the rocks; nodules visible
  through the water like dark beads. Required ANSI nouns:
  iron-seep, lake-iron, rust-stain, mineral-water. Apply plural
  rule for nodules/stains.]
biome: water
coord:
  x: -21
  y: -1
  z: 0
exits:
  north:
    roomid: 4183
  south:
    roomid: 4191
nouns:
  iron-seep: |
    [1-2 sentences. A broad slow seep where mineral-rich water
    rises through the marsh peat — the source of the lake-iron
    nodules concentrated downstream at Sluice Pond. Smells faintly
    of iron and warm stone.]
  lake-iron: |
    [1-2 sentences. Dark heavy nodules cluster on the seep's
    bottom — the size of a knucklebone or smaller, dense in
    the hand. The Stillwater smith pays a premium for a sack.]
  rust-stain: |
    [1-2 sentences. Rust-orange stains mark every stone the
    seep-water has touched for any length of time — distinctive
    enough to identify the seep from a distance.]
  rust-stains: |
    [1-2 sentences. Rust-orange stains mark every stone the
    seep-water has touched for any length of time — distinctive
    enough to identify the seep from a distance.]
  mineral-water: |
    [1-2 sentences. The seep-water tastes faintly of iron and
    salt — drinkable in small amounts, the foragers say,
    though it stains a flask.]
idlemessages:
  - 'a single bubble rises from a deeper part of the seep'
  - ''
  - 'iron-tang strengthens briefly in the air'
  - ''
  - 'a rust-stain near the trail darkens as wet seep-water touches it'
```

### Step 2: Create 4188 Shrimp Shallows (4-way row hub)

```yaml
roomid: 4188
zone: Stillwater Marsh
title: Shrimp Shallows
description: >
  [4-7 sentences. THEME: south of Heron Marsh the pond shallows
  into knee-deep clear water alive with skitter-shrimp — small
  translucent crustaceans whose shells (40051) the foragers
  collect. Snapping turtle territory — the turtle eats the
  shrimp. Senses: clear shallow water; shrimp visible in
  small flickering schools; sometimes the dark shape of a
  turtle on the bottom. Required ANSI nouns: shallows,
  skitter-shrimp, turtle-shape, water-weeds. Apply plural rule
  for shrimp/weeds.]
biome: water
coord:
  x: -22
  y: -1
  z: 0
exits:
  north:
    roomid: 4184
  south:
    roomid: 4192
  east:
    roomid: 4187
  west:
    roomid: 4189
nouns:
  shallows: |
    [1-2 sentences. Clear shallow water perhaps knee-deep,
    extending in every direction — the bottom visible as
    rippled fine sand interspersed with weed-mats.]
  skitter-shrimp: |
    [1-2 sentences. Small translucent shrimp dart in flickering
    schools through the shallows — almost invisible in still
    water, briefly visible as silver flashes when they move.
    Their shed shells are a Stillwater forage staple.]
  turtle-shape: |
    [1-2 sentences. The dark shape of a snapping turtle on the
    sand-bottom — sometimes there, sometimes a tussock instead.
    Hard to tell from above until it moves.]
  water-weed: |
    [1-2 sentences. Strands of dark green water-weed grow in
    mats from the sand — shrimp hide here, the turtle stalks
    among them.]
  water-weeds: |
    [1-2 sentences. Strands of dark green water-weed grow in
    mats from the sand — shrimp hide here, the turtle stalks
    among them.]
idlemessages:
  - 'a school of skitter-shrimp scatters and re-forms'
  - ''
  - 'a shrimp-shell drifts up from the sand and rocks slowly'
  - ''
  - 'something dark moves under the water-weeds and is still again'
```

### Step 3: Create 4189 Sundew Hollow (4-way row hub)

```yaml
roomid: 4189
zone: Stillwater Marsh
title: Sundew Hollow
description: >
  [4-7 sentences. THEME: a small dry-rim hollow west of Shrimp
  Shallows where carnivorous sundew plants colonize the wet
  peat. Marsh-rat territory (the rats nest in the rim). Senses:
  dewdrops sparkling on sundew leaves at any sunlight; small
  insect bodies visible stuck to the leaves; the rats' runs
  lead in and out. Required ANSI nouns: sundew, dewdrops,
  rat-runs, rim. Apply plural rule for sundews/dewdrops/runs.]
biome: water
coord:
  x: -23
  y: -1
  z: 0
exits:
  north:
    roomid: 4185
  east:
    roomid: 4188
  west:
    roomid: 4190
  south:
    roomid: 4193
nouns:
  sundew: |
    [1-2 sentences. Small carnivorous sundew plants colonize the
    hollow's wet peat — flat red-tinged rosettes the size of a
    coin, each leaf studded with sticky dewdrops. Insects landing
    are caught and slowly digested.]
  sundews: |
    [1-2 sentences. Small carnivorous sundew plants colonize the
    hollow's wet peat — flat red-tinged rosettes the size of a
    coin, each leaf studded with sticky dewdrops. Insects landing
    are caught and slowly digested.]
  dewdrops: |
    [1-2 sentences. The sundew-droplets sparkle in any sunlight
    — sticky and surprisingly tough, the trap mechanism for
    small flies.]
  rat-run: |
    [1-2 sentences. Small worn paths lead in and out of the
    hollow's rim — marsh-rat trails, the rats themselves usually
    out of sight in the thicker cover.]
  rat-runs: |
    [1-2 sentences. Small worn paths lead in and out of the
    hollow's rim — marsh-rat trails, the rats themselves usually
    out of sight in the thicker cover.]
  rim: |
    [1-2 sentences. A low rim of slightly-drier peat surrounds
    the hollow — the closest thing to dry ground in this part
    of the marsh, the reason the rats nest here.]
idlemessages:
  - 'a small fly lands on a sundew and is briefly held'
  - ''
  - 'something small rustles through the rim cover'
  - ''
  - 'dewdrops on a sundew catch the light and sparkle briefly'
```

### Step 4: Create 4190 Black Pool (terminal W — rare pearl)

```yaml
roomid: 4190
zone: Stillwater Marsh
title: Black Pool
description: >
  [4-7 sentences. THEME: the west terminal of the y=-1 row — a
  small dark deep pool in a hollow of black peat. The deep
  reward of the zone: the rare Stillwater black pearls (40053)
  occasionally form in the freshwater mussels at the pool's
  bottom. Forager rare-find territory. Senses: water dark and
  bottom unseen; cool air rising; sense of mystery. Required
  ANSI nouns: black-pool, dark-water, mussel-bed, peat-rim.
  Apply plural rule for mussels.]
biome: water
coord:
  x: -24
  y: -1
  z: 0
exits:
  north:
    roomid: 4186
  east:
    roomid: 4189
nouns:
  black-pool: |
    [1-2 sentences. A small deep pool perhaps three paces
    across, the water dark with peat and the bottom invisible
    even in bright light — far deeper than the rest of the
    marsh suggests it could be.]
  dark-water: |
    [1-2 sentences. The water is the color of strong tea —
    stained by the surrounding peat, light-absorbing rather
    than reflecting. Not unsafe to drink but not appealing.]
  mussel-bed: |
    [1-2 sentences. A bed of freshwater mussels somewhere on
    the pool's bottom — visible only as a darker patch in the
    dark water. These particular mussels are the source of
    the famous Stillwater black pearl, when one bothers to form.]
  mussels: |
    [1-2 sentences. A bed of freshwater mussels somewhere on
    the pool's bottom — visible only as a darker patch in the
    dark water. These particular mussels are the source of
    the famous Stillwater black pearl, when one bothers to form.]
  peat-rim: |
    [1-2 sentences. The pool's rim is thick black peat — soft
    enough that you would not want to slip in. The foragers
    work the pool from the rim with long poles.]
idlemessages:
  - 'a single small bubble breaks the dark surface and is gone'
  - ''
  - 'cool air rises off the pool and chills the skin'
  - ''
  - 'something deep in the pool moves; the surface trembles, then stills'
```

### Step 5: Build verify + commit

```bash
go build ./...

git add _datafiles/world/dogmud/rooms/stillwater_marsh/
git commit -m "$(cat <<'EOF'
feat(rooms): Stillwater Marsh forage row (4187-4190)

Iron Seep (lake-iron forage), Shrimp Shallows (skitter-shrimp +
snapping turtle), Sundew Hollow (sundew + marsh-rat territory),
Black Pool (terminal W — rare Stillwater black pearl). Stage 3.0a.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Adder pocket + south spine + steppe terminus (4191, 4192, 4193, 4194, 4195, 4196)

**Files (CREATE all 6):**
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4191.yaml` — Mossy Hummock
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4192.yaml` — Dragonfly Glade
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4193.yaml` — Adder Den (terminal W — HOSTILE)
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4194.yaml` — Bog Edge
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4195.yaml` — Far Bog Heart (terminal S, **biome: plains**)
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4196.yaml` — Hidden Spring (terminal SE pocket)

The largest cluster — finishes the zone. **4195 biome IS plains** (the only non-water room — marks the upland transition).

### Step 1: Create 4191 Mossy Hummock

```yaml
roomid: 4191
zone: Stillwater Marsh
title: Mossy Hummock
description: >
  [4-7 sentences. THEME: south of Iron Seep the marsh lifts
  briefly into a mossy hummock — a few paces of dryish ground
  rising above the standing water, soft moss carpet underfoot.
  The marsh-rats use this as one of their nesting hummocks too.
  Senses: drier underfoot for the first time since the entry;
  bright green moss; rat-droppings scattered. Required ANSI
  nouns: hummock, sphagnum-moss, droppings, rat-trail. Apply
  plural rule for trails.]
biome: water
coord:
  x: -21
  y: -2
  z: 0
exits:
  north:
    roomid: 4187
  south:
    roomid: 4196
  west:
    roomid: 4192
nouns:
  hummock: |
    [1-2 sentences. A small raised mound of peat-and-moss perhaps
    twice a person's stride across, rising a hand's height above
    the surrounding water — the closest thing to dry ground in
    this part of the marsh.]
  sphagnum-moss: |
    [1-2 sentences. Bright green sphagnum moss carpets the
    hummock — soft enough to sit on, holds water like a sponge,
    the kind a healer uses for clean wound-dressing.]
  droppings: |
    [1-2 sentences. Small dark marsh-rat droppings scatter the
    hummock — the rats nest here, away from the standing water.]
  rat-trail: |
    [1-2 sentences. Small worn paths lead off the hummock in
    every direction — rat-trails through the cover, packed
    smooth by small fast feet.]
  rat-trails: |
    [1-2 sentences. Small worn paths lead off the hummock in
    every direction — rat-trails through the cover, packed
    smooth by small fast feet.]
idlemessages:
  - 'something small rustles in the moss at the hummock-edge'
  - ''
  - 'a marsh-rat appears briefly at a trail-mouth, freezes, and is gone'
  - ''
  - 'the moss springs back where you lift your foot'
```

### Step 2: Create 4192 Dragonfly Glade (4-way junction)

```yaml
roomid: 4192
zone: Stillwater Marsh
title: Dragonfly Glade
description: >
  [4-7 sentences. THEME: a small open clearing in the marsh
  where dragonflies hunt over a still pool — iridescent blue
  and green and copper, more here than anywhere else in the
  zone. Dragonfly swarm spawn. Senses: continuous low buzz;
  flashes of color in the air; mosquitoes thinned because the
  dragonflies eat them. Required ANSI nouns: dragonfly,
  iridescence, hunt-pool, mosquito-thinning. Apply plural rule
  for dragonflies/mosquitoes.]
biome: water
coord:
  x: -22
  y: -2
  z: 0
exits:
  north:
    roomid: 4188
  east:
    roomid: 4191
  west:
    roomid: 4193
  south:
    roomid: 4194
nouns:
  dragonfly: |
    [1-2 sentences. Large dragonflies hunt over the pool — body-
    lengths the size of a finger, wings clear and shimmering,
    colors metallic blue and copper and dark green.]
  dragonflies: |
    [1-2 sentences. Large dragonflies hunt over the pool — body-
    lengths the size of a finger, wings clear and shimmering,
    colors metallic blue and copper and dark green.]
  iridescence: |
    [1-2 sentences. The dragonflies catch the light in flashes
    — blue from one angle, green from another, copper at the
    edge of vision. The glade is brighter for them.]
  hunt-pool: |
    [1-2 sentences. A small still pool the dragonflies work as
    territory — they intercept any small flying insect that
    crosses the open water. The pool itself is shallow and
    clear, weeds at the bottom.]
  mosquito-thinning: |
    [1-2 sentences. The mosquitoes are notably scarcer here
    than elsewhere in the marsh — the dragonflies eat them.
    The relief is immediate and noticeable.]
idlemessages:
  - 'a dragonfly intercepts something tiny mid-air with a small click'
  - ''
  - 'iridescence flashes briefly as a dragonfly turns'
  - ''
  - 'two dragonflies tangle briefly over territory and separate'
```

### Step 3: Create 4193 Adder Den (terminal W — HOSTILE)

```yaml
roomid: 4193
zone: Stillwater Marsh
title: Adder Den
description: >
  [4-7 sentences. THEME: a low slate-rimmed hollow west of
  Dragonfly Glade where bog adders shelter and hunt. The
  smell of snake warns before the adder does. Terminal — no
  further west. Senses: dry rocky-warm rim contrasting the
  surrounding wet; slough-skins on the stones; a heavy reptile
  smell; old bones and shed skins evidence of recent activity.
  Required ANSI nouns: slate-rim, slough-skin, bone-pile,
  basking-rock. Apply plural rule for skins/bones/rocks.]
biome: water
coord:
  x: -23
  y: -2
  z: 0
exits:
  east:
    roomid: 4192
nouns:
  slate-rim: |
    [1-2 sentences. A low rim of dark flat slate-stones rises
    around a sunken hollow — the only stones in the marsh,
    weathered to a dull warm-grey, holding heat from any sun.]
  slough-skin: |
    [1-2 sentences. Several papery shed snake-skins lie on the
    stones — translucent and intricate, scale-pattern visible
    in fine relief. Bog adder skins, several seasons' worth.]
  slough-skins: |
    [1-2 sentences. Several papery shed snake-skins lie on the
    stones — translucent and intricate, scale-pattern visible
    in fine relief. Bog adder skins, several seasons' worth.]
  bone-pile: |
    [1-2 sentences. Small bones lie scattered on the slate —
    rat ribs, frog skulls, the remains of the adder's hunting.
    Picked clean.]
  basking-rock: |
    [1-2 sentences. The largest flat slate stone in the rim is
    polished smooth by years of use as a basking-rock — the
    adder's preferred sun-station, marked with an oily sheen
    where the snake's scales have left their traces.]
  basking-rocks: |
    [1-2 sentences. The largest flat slate stone in the rim is
    polished smooth by years of use as a basking-rock — the
    adder's preferred sun-station, marked with an oily sheen
    where the snake's scales have left their traces.]
idlemessages:
  - 'a slough-skin rustles in a passing breeze'
  - ''
  - 'something heavy moves briefly under a slate stone'
  - ''
  - 'the reptile-smell strengthens briefly and is gone'
```

### Step 4: Create 4194 Bog Edge

```yaml
roomid: 4194
zone: Stillwater Marsh
title: Bog Edge
description: >
  [4-7 sentences. THEME: south of Dragonfly Glade the marsh
  shifts character — the standing water gives way to true bog,
  acidic and quaking, the ground unstable underfoot. The
  beginning of the upland transition. Senses: ground softer
  even than peat; tea-color water in any depression; sphagnum
  carpets thicker; a sense of nearing the world's edge.
  Required ANSI nouns: bog, quaking-ground, acid-water,
  sphagnum-carpet. No plurals likely needed but apply rule
  if you write any.]
biome: water
coord:
  x: -22
  y: -3
  z: 0
exits:
  north:
    roomid: 4192
  south:
    roomid: 4195
nouns:
  bog: |
    [1-2 sentences. True bog underfoot now — acidic, slow,
    nothing like the alkaline pond-marsh to the north. The
    plants here are different: cotton-grass, low cranberry,
    sphagnum dominant.]
  quaking-ground: |
    [1-2 sentences. The ground trembles slightly with every
    step — the bog is essentially a thick floating mat of
    living plant matter over deeper water, and your weight
    moves the whole mat.]
  acid-water: |
    [1-2 sentences. Tea-colored water shows in any depression
    — strongly acidic from the sphagnum, undrinkable but
    famously preservative. Things lost in true bog do not
    rot.]
  sphagnum-carpet: |
    [1-2 sentences. Thick sphagnum-moss carpet underfoot, deeper
    here than at the Mossy Hummock — bright green at the
    surface, darker beneath, holding water like a saturated
    cloth.]
idlemessages:
  - 'the ground trembles briefly under your weight and stills'
  - ''
  - 'a small white moth lifts from the sphagnum and lands again'
  - ''
  - 'the wind off the south carries a different smell — drier'
```

### Step 5: Create 4195 Far Bog Heart (terminal S — biome PLAINS!)

```yaml
roomid: 4195
zone: Stillwater Marsh
title: Far Bog Heart
description: >
  [4-7 sentences. THEME: the bog gives out entirely south of
  Bog Edge. The ground rises sharply onto a shoulder of dry
  upland grass — the steppe. The Dustwalk lies somewhere out
  there. Terminal — no south exit. Senses: SUDDEN dryness;
  open sky for the first time since Heron Marsh; pale upland
  grass; constant south wind; the marsh visible behind, the
  steppe ahead. Required ANSI nouns: upland-shoulder,
  pale-grass, south-horizon, marsh-edge. Apply plural rule.]
biome: plains
coord:
  x: -22
  y: -4
  z: 0
exits:
  north:
    roomid: 4194
nouns:
  upland-shoulder: |
    [1-2 sentences. The bog rises sharply here onto a low
    shoulder of dry firm upland — the transition is dramatic,
    a band perhaps two paces wide where wet bog gives over to
    dry grass underfoot.]
  pale-grass: |
    [1-2 sentences. Knee-high pale tan upland grass bends in
    the south wind — the same grass the Dustwalk shows in its
    drier stretches. Distinctly not marsh vegetation.]
  south-horizon: |
    [1-2 sentences. The horizon south is unbroken steppe — pale
    grass and scattered round stones stretching to a long line
    where the land meets the sky. The Dustwalk is somewhere
    out there. You do not go that way today.]
  marsh-edge: |
    [1-2 sentences. Looking back north, the marsh ends sharply
    — green wet bog one moment, dry pale grass the next. The
    foragers say the change is older than the watermill, older
    than Stillwater town.]
idlemessages:
  - 'south wind moves the pale-grass in long even waves'
  - ''
  - 'a hawk wheels far to the south, very high'
  - ''
  - 'the marsh-edge behind you releases a single sphagnum-fragrance whiff'
```

### Step 6: Create 4196 Hidden Spring (terminal SE pocket)

```yaml
roomid: 4196
zone: Stillwater Marsh
title: Hidden Spring
description: >
  [4-7 sentences. THEME: a small clear spring rises in a
  pocket south of Mossy Hummock — different from the deeper
  Spring Pool to the north (which feeds the creek); this one
  is just a pocket-pool the foragers know about. Terminal SE
  pocket. Senses: cold clear water rising silently; smaller
  than Spring Pool; surrounded by waxberry and bog-myrtle.
  Required ANSI nouns: hidden-spring, waxberry, bog-myrtle,
  pocket-pool. Apply plural rule for waxberries.]
biome: water
coord:
  x: -21
  y: -3
  z: 0
exits:
  north:
    roomid: 4191
nouns:
  hidden-spring: |
    [1-2 sentences. A small clear spring rises silently in a
    pocket of bog-rim — perhaps a forearm deep at the source,
    the water visibly rising from a fissure in dark stone
    below. Cold and pure.]
  waxberry: |
    [1-2 sentences. A few low waxberry shrubs grow at the
    spring's edge — small grey-blue waxy berries clustered
    at the branch-tips, edible and faintly sweet, a forager's
    reward for finding the pocket.]
  waxberries: |
    [1-2 sentences. A few low waxberry shrubs grow at the
    spring's edge — small grey-blue waxy berries clustered
    at the branch-tips, edible and faintly sweet, a forager's
    reward for finding the pocket.]
  bog-myrtle: |
    [1-2 sentences. A stand of bog-myrtle around the pocket
    releases a sweet aromatic smell when brushed — used by
    Stillwater brewers to flavor a particular ale, and by the
    healer to ease bug-bites.]
  pocket-pool: |
    [1-2 sentences. The spring forms a small pocket-pool perhaps
    two paces across at most — much smaller than the deeper
    Spring Pool to the north, easier to miss if you don't
    know it's here.]
idlemessages:
  - 'a single bubble rises silently in the pocket-pool'
  - ''
  - 'a waxberry detaches and floats slowly across the surface'
  - ''
  - 'bog-myrtle releases a faint sweet smell when the wind moves'
```

### Step 7: Build verify

`go build ./...`. If port-free: zone size should now be 20.

### Step 8: Commit

```bash
git add _datafiles/world/dogmud/rooms/stillwater_marsh/
git commit -m "$(cat <<'EOF'
feat(rooms): Stillwater Marsh south cluster (4191-4196)

Mossy Hummock (rat-nest dry ground), Dragonfly Glade (4-way
junction, dragonfly spawn), Adder Den (W terminal — HOSTILE),
Bog Edge (true bog transition), Far Bog Heart (S terminal,
biome: plains — upland steppe view), Hidden Spring (SE terminal
pocket). All 20 rooms now in place. Stage 3.0a.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Create the 5 mob YAMLs

**Files (CREATE all 5):**
- `_datafiles/world/dogmud/mobs/stillwater_marsh/366-river_otter.yaml`
- `_datafiles/world/dogmud/mobs/stillwater_marsh/367-marsh_rat.yaml`
- `_datafiles/world/dogmud/mobs/stillwater_marsh/368-dragonfly_swarm.yaml`
- `_datafiles/world/dogmud/mobs/stillwater_marsh/369-snapping_turtle.yaml`
- `_datafiles/world/dogmud/mobs/stillwater_marsh/370-bog_adder.yaml`

The folder `stillwater_marsh/` under `mobs/` won't exist — create it.

**Aggro design:**

| Mob | hostile | groups | hates | Aggro behavior |
|---|---|---|---|---|
| river otter | false | [animal, mustelid, prey] | (none) | flees on sight (prey) |
| marsh rat | false | [animal, rodent, prey] | (none) | flees on sight (prey) |
| dragonfly swarm | false | [animal, insectoid, swarm] | (none) | passive; sting if attacked |
| snapping turtle | false | [animal, reptile] | (none) | passive; fights HARD if attacked |
| bog adder | **true** | [animal, serpent, predator] | **[rodent]** | hostile to player AND hunts marsh-rats |

**The bog adder is the ONLY player-hostile mob** AND has the same emergent dynamic as 3.0c's wolf (predator-with-hate-target hunting prey in the zone). adder + marsh-rat parallels wolf + boar.

### Step 1: Create 366 river otter

```yaml
mobid: 366
zone: Stillwater Marsh
behavior_archetype: prey
archetype: fighting
statpool: 35
itemdropchance: 50
hostile: false
charm_immune: false
maxwander: 4
groups:
  - animal
  - mustelid
  - prey
items:
  - itemid: 40058
idlecommands:
  - 'emote slides down the slick mud slide and into the water'
  - ''
  - 'emote surfaces briefly, watches you, dives again'
  - ''
  - 'emote pops a clam open on its chest and works at it'
  - ''
activitylevel: 35
character:
  name: river otter
  description: |
    A sleek brown river otter, half in the water and half out,
    quick-eyed and quick to the slide if you get too close. It
    holds a clam-shell in its forepaws and watches you steadily
    while it works at it.
  speciesid: 24
  level: 2
  gold: 0
  stats:
    strength:
      training: 8
    dexterity:
      training: 18
    perception:
      training: 16
    vitality:
      training: 8
```

`speciesid: 24` is the mustelid species shipped in 3.0c T1. Otter drops freshwater clam (40058) — otters fish for clams, fits the lore.

### Step 2: Create 367 marsh rat

```yaml
mobid: 367
zone: Stillwater Marsh
behavior_archetype: prey
archetype: fighting
statpool: 20
itemdropchance: 50
hostile: false
charm_immune: false
maxwander: 4
groups:
  - animal
  - rodent
  - prey
items:
  - itemid: 40064
idlecommands:
  - 'emote freezes at a trail-mouth, ears twitching'
  - ''
  - 'emote scuttles a few paces and stops'
  - ''
  - 'emote nibbles at a sphagnum-shoot quickly'
  - ''
activitylevel: 35
character:
  name: marsh rat
  description: |
    A small grey-brown marsh rat, wet-furred and quick, with
    bright black eyes and a pink scaly tail. It freezes at
    your approach and weighs whether you are a problem.
  speciesid: 10
  level: 1
  gold: 0
  stats:
    strength:
      training: 5
    dexterity:
      training: 18
    perception:
      training: 15
    vitality:
      training: 6
```

Reuses wild-hare-meat (40064) for generic small-game meat. Same kind of small forager prey as the wild hare.

### Step 3: Create 368 dragonfly swarm

```yaml
mobid: 368
zone: Stillwater Marsh
behavior_archetype: combat_passive
archetype: fighting
statpool: 18
itemdropchance: 0
hostile: false
charm_immune: true
maxwander: 0
groups:
  - animal
  - insectoid
  - swarm
idlecommands:
  - 'emote intercepts something tiny mid-air with a quiet click'
  - ''
  - 'emote turns in formation over the hunt-pool'
  - ''
activitylevel: 50
character:
  name: dragonfly swarm
  description: |
    A working swarm of dragonflies — perhaps two dozen large
    iridescent bodies in coordinated patrol over the small
    pool. They ignore you unless you make a problem of yourself.
  speciesid: 12
  level: 1
  gold: 0
  stats:
    strength:
      training: 4
    dexterity:
      training: 16
    perception:
      training: 14
    vitality:
      training: 10
```

No items (atmospheric). `maxwander: 0` keeps swarm at the glade. `charm_immune: true` because hive-mind.

### Step 4: Create 369 snapping turtle

```yaml
mobid: 369
zone: Stillwater Marsh
behavior_archetype: combat_passive
archetype: fighting
statpool: 80
itemdropchance: 0
hostile: false
charm_immune: false
maxwander: 1
groups:
  - animal
  - reptile
idlecommands:
  - 'emote drifts slowly across the sand-bottom, half-buried'
  - ''
  - 'emote snaps once at a passing skitter-shrimp'
  - ''
  - 'emote settles deeper into the weed and is still'
  - ''
activitylevel: 15
character:
  name: snapping turtle
  description: |
    A heavy snapping turtle on the sand-bottom — shell the size
    of a cart-wheel, wide-jawed, the head and clawed limbs
    drawn back when you watch it. It is older than you. Don't
    bother it.
  speciesid: 21
  level: 4
  gold: 0
  stats:
    strength:
      training: 18
    dexterity:
      training: 6
    perception:
      training: 12
    vitality:
      training: 22
```

Boar-equivalent — high vit + str, low dex (slow but mean). No drops by design.

### Step 5: Create 370 bog adder (the ONLY player-hostile mob)

```yaml
mobid: 370
zone: Stillwater Marsh
behavior_archetype: ambusher
archetype: fighting
statpool: 60
itemdropchance: 0
hostile: true
charm_immune: false
maxwander: 3
groups:
  - animal
  - serpent
  - predator
hates:
  - rodent
idlecommands:
  - 'emote suns itself on a basking-rock with eyes half-lidded'
  - ''
  - 'emote tongues the air, testing for prey'
  - ''
  - 'emote slips silently into the slate-rim shadows'
  - ''
activitylevel: 25
character:
  name: bog adder
  description: |
    A bog adder coiled on the warm slate, dark zig-zag pattern
    along its grey back, head broad and triangular. Its tongue
    flicks once. Its yellow eyes are fixed on you and the
    coiled body has begun to shift. This is going to bite.
  speciesid: 8
  level: 4
  gold: 0
  stats:
    strength:
      training: 10
    dexterity:
      training: 20
    perception:
      training: 18
    vitality:
      training: 12
```

`hostile: true` AND `hates: [rodent]` — attacks players on sight AND hunts marsh-rats. The "wolf" of this zone (intra-zone predator with hate target) AND "badger" (only true hostile to players) rolled into one. `behavior_archetype: ambusher` matches the "strikes from cover" snake style.

### Step 6: Build verify + commit

```bash
go build ./...

git add _datafiles/world/dogmud/mobs/stillwater_marsh/
git commit -m "$(cat <<'EOF'
feat(mobs): Stillwater Marsh wildlife (mobs 366-370)

5 mobs for the new zone:
- 366 river otter (prey, mustelid 24 — first non-badger consumer
  of the species shipped in 3.0c T1)
- 367 marsh rat (prey, rodent)
- 368 dragonfly swarm (combat_passive, insectoid)
- 369 snapping turtle (combat_passive, reptile; hits hard if engaged)
- 370 bog adder (ambusher, serpent; HOSTILE + hates [rodent] —
  combines the 3.0c wolf-vs-boar dynamic AND the badger-as-only-
  hostile pattern in one mob)

Stage 3.0a.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Spawninfo wiring + Mill Creek Footbridge west exit

**Files (MODIFY all 7):**
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4185.yaml` — otter spawn
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4188.yaml` — turtle spawn
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4189.yaml` — marsh rat spawn (Sundew Hollow)
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4191.yaml` — marsh rat spawn (Mossy Hummock)
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4192.yaml` — dragonfly swarm spawn
- `_datafiles/world/dogmud/rooms/stillwater_marsh/4193.yaml` — bog adder spawn
- `_datafiles/world/dogmud/rooms/stillwater/4133.yaml` — Mill Creek Footbridge west exit

### Spawn distribution

| Room | mobid | cooldown |
|---|---|---|
| 4185 Otter Slide | 366 river otter | 600 rounds |
| 4188 Shrimp Shallows | 369 snapping turtle | 1200 rounds |
| 4189 Sundew Hollow | 367 marsh rat | 300 rounds |
| 4191 Mossy Hummock | 367 marsh rat | 300 rounds |
| 4192 Dragonfly Glade | 368 dragonfly swarm | 600 rounds |
| 4193 Adder Den | 370 bog adder | 1800 rounds |

Cooldowns mirror 3.0c: rare/tough mobs get longer waits (turtle 1200, adder 1800).

### Step 1-6: Append spawninfo blocks

Each room gets the `spawninfo:` block appended at the file END (after `idlemessages:`). Block format:

```yaml
spawninfo:
- mobid: NNN
  cooldown: NNN rounds
```

For 4185 (otter):
```yaml
spawninfo:
- mobid: 366
  cooldown: 600 rounds
```

For 4188 (turtle):
```yaml
spawninfo:
- mobid: 369
  cooldown: 1200 rounds
```

For 4189 (marsh rat — Sundew Hollow):
```yaml
spawninfo:
- mobid: 367
  cooldown: 300 rounds
```

For 4191 (marsh rat — Mossy Hummock):
```yaml
spawninfo:
- mobid: 367
  cooldown: 300 rounds
```

For 4192 (dragonfly swarm):
```yaml
spawninfo:
- mobid: 368
  cooldown: 600 rounds
```

For 4193 (bog adder):
```yaml
spawninfo:
- mobid: 370
  cooldown: 1800 rounds
```

### Step 7: Edit Mill Creek Footbridge 4133 to add the west exit

In `_datafiles/world/dogmud/rooms/stillwater/4133.yaml`, locate the `exits:` block. Today 4133 has east + north exits (or whatever it currently has — verify first by reading the file). Add a west exit pointing to 4177 with the cross-zone target:

```yaml
  west:
    roomid: 4177
    zone: Stillwater Marsh
```

Insert this entry into the existing exits block alongside the others. Don't disturb the rest of the file.

### Step 8: Build verify

`go build ./...`. If port-free: both zones (Stillwater + Stillwater Marsh) should report no missing-exit warnings; Stillwater Marsh size 20.

Verify NO instance saves were created: `git status` should show ONLY the 7 modified files (no `mobs.instances/`, `rooms.instances/`, `shops/` files).

### Step 9: Commit

```bash
git add _datafiles/world/dogmud/rooms/stillwater_marsh/ _datafiles/world/dogmud/rooms/stillwater/4133.yaml
git commit -m "$(cat <<'EOF'
feat(spawn): Stillwater Marsh spawninfo + Footbridge west exit

Spawn distribution: otter at Otter Slide, turtle at Shrimp Shallows,
marsh rats at Sundew Hollow + Mossy Hummock (twin spawns let the
adder always have prey to hunt), dragonfly swarm at Dragonfly Glade,
bog adder at Adder Den. Rare-mob cooldowns longer (turtle 1200,
adder 1800).

Mill Creek Footbridge (4133) gains west→4177 exit with cross-zone
target. Stage 3.0a.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: coord_map + PATCH_NOTES

**Files (MODIFY both):**
- `docs/coordinate_map.md`
- `PATCH_NOTES.md`

The coord map is missing all 48 Stillwater rooms (the doc was stale before 3.0c added Fernway catch-up; this task does Stillwater catch-up + new zone).

### Step 1: Add 48 Stillwater rows to coord map

The coord-map table is sorted by room ID. Stillwater rooms are 4100-4146. They go BETWEEN the Watchers Crossing block (~line 237: `| 427 | Watchers Crossing | Watchers Crossing, East Road | -5 | 0 | 0 |`) and the Fernway block (which 3.0c added at row `| 4147 | The Fernway | ...`).

Find the line just BEFORE `| 4147 | The Fernway | ...` and insert all 48 Stillwater rows above it. Read each Stillwater room file (4100.yaml through 4146.yaml) to get its title, x, y, z. Compose rows:

```
| 4100 | Stillwater | Stillwater Gate, Southern Approach | -18 | 1 | 0 |
| 4101 | Stillwater | South Approach | -18 | 2 | 0 |
... (one row per Stillwater room, sorted by mobid)
```

48 rows. Then below the Fernway South block (which ends at `| 4176 | The Fernway South | Birdsong Glade | -14 | -20 | 0 |`), add the 20 new Stillwater Marsh rows:

```
| 4177 | Stillwater Marsh | Marsh Track | -21 | 1 | 0 |
| 4178 | Stillwater Marsh | Cattail Verge | -22 | 1 | 0 |
| 4179 | Stillwater Marsh | Mill Creek Source | -21 | 2 | 0 |
| 4180 | Stillwater Marsh | Spring Pool | -21 | 3 | 0 |
| 4181 | Stillwater Marsh | Reed Beds | -23 | 1 | 0 |
| 4182 | Stillwater Marsh | Willow Grove | -24 | 1 | 0 |
| 4183 | Stillwater Marsh | Cattail Bend | -21 | 0 | 0 |
| 4184 | Stillwater Marsh | Heron Marsh | -22 | 0 | 0 |
| 4185 | Stillwater Marsh | Otter Slide | -23 | 0 | 0 |
| 4186 | Stillwater Marsh | Clam Beds | -24 | 0 | 0 |
| 4187 | Stillwater Marsh | Iron Seep | -21 | -1 | 0 |
| 4188 | Stillwater Marsh | Shrimp Shallows | -22 | -1 | 0 |
| 4189 | Stillwater Marsh | Sundew Hollow | -23 | -1 | 0 |
| 4190 | Stillwater Marsh | Black Pool | -24 | -1 | 0 |
| 4191 | Stillwater Marsh | Mossy Hummock | -21 | -2 | 0 |
| 4192 | Stillwater Marsh | Dragonfly Glade | -22 | -2 | 0 |
| 4193 | Stillwater Marsh | Adder Den | -23 | -2 | 0 |
| 4194 | Stillwater Marsh | Bog Edge | -22 | -3 | 0 |
| 4195 | Stillwater Marsh | Far Bog Heart | -22 | -4 | 0 |
| 4196 | Stillwater Marsh | Hidden Spring | -21 | -3 | 0 |
```

68 rows total inserted (48 Stillwater catch-up + 20 new).

### Step 2: Update Total reachable header

Currently the doc says `Total reachable: 304 rooms across multiple zones` (after 3.0c). After this task: 304 + 68 = `372 rooms across multiple zones`.

### Step 3: Update coord range summary

Find `## Coordinate Range Summary` (~line 417). Update the X range:

Old:
```
- X range: -13 to 18 (width: 32)
```

New:
```
- X range: -24 to 18 (width: 43)
```

(Stillwater Marsh extends to x=-24. Y range is unchanged — already covered by 3.0c at y=-22.)

### Step 4: Add Stage 3.0a entry to PATCH_NOTES.md

After the title `# DOGMud Patch Notes` and ABOVE the existing 2026-04-28 Stage 3.0c entry, insert:

```markdown
## 2026-04-28 — Stage 3.0a: Stillwater Marsh Zone (dev only)

**Note:** Dev-only landing. The full economy stack ships to prod (`master`)
as a coherent update once Stage 3.4 lands.

- New 20-room wetland zone west of Stillwater, themed as marsh
  giving way to upland steppe at the southern terminus. Connects
  from Mill Creek Footbridge (4133) via a new west exit; terminates
  at Far Bog Heart (4195, biome: plains) with a one-way view of
  the Dustwalk beyond.
- Five new wildlife mobs (366-370): river otter, marsh rat,
  dragonfly swarm, snapping turtle, bog adder. **Only the bog
  adder is hostile to players** AND it `hates: [rodent]` — it
  hunts the marsh-rats in adjacent rooms (mirror of 3.0c's
  wolf-hates-boar dynamic, but combined with the only-hostile-to-
  player role into one mob).
- The river otter is the **first non-badger consumer of the
  mustelid species** (24) added in Stage 3.0c — validates the
  species investment.
- All 6 existing Stillwater forage mats (lake-iron, marsh willow
  bark, lake mint, freshwater clam, skitter-shrimp shell,
  Stillwater black pearl) get fresh territory to spawn in. No
  new mats added.
- Stage 3.0a is the territory groundwork for Stage 3.1 forager
  NPCs — the marsh is now big enough for a Stillwater-anchored
  forager to wander, gather, and recall to depot when injured.
- Coord map gains 48 Stillwater catch-up rows (the doc was
  missing all of Stillwater) plus 20 new Stillwater Marsh rows.

```

### Step 5: Build verify

`go build ./...`.

### Step 6: Commit

```bash
git add docs/coordinate_map.md PATCH_NOTES.md
git commit -m "$(cat <<'EOF'
docs(3.0a): coord map catch-up + Stillwater Marsh + patch notes

- coordinate_map.md: added 48 missing Stillwater rows (catch-up;
  the doc was stale before this) plus 20 Stillwater Marsh rows.
  X range summary extended from -13/18 to -24/18.
- PATCH_NOTES: Stage 3.0a dev-only entry.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Out of scope reminder

- **Quests in this zone** — Phase 2 work, deferred. Aligns with Stage 3.1 forager wiring.
- **New foragable mats** — existing 6 Stillwater mats are the supply pool.
- **New zone-specific items** — no curio loot.
- **Heron / frog mobs** — no avian or amphibian species; both stay as atmospheric nouns.
- **Beaver mob / dam interactivity** — out of scope for v1.
- **Stage 3.0e (corpse salvage) + 3.0a interaction** — corpse salvage applies to all corpses by mob group, so player kills of marsh rat / otter / etc. produce corpse-salvageable carcasses automatically. No 3.0a-specific wiring needed.

## Done = ?

All 8 tasks complete, all commits landed on `development` branch, manual smoke verification green. Per the multi-stage caravan/economy effort: this lands on `development` only. Nothing ships to `master` until Stage 3.4 lands.
