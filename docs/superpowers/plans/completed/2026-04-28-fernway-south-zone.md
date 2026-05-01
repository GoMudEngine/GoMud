# Fernway South Zone Build Implementation Plan (Stage 3.0c)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Fernway South zone — 20 rooms south of the existing Fernway zone, themed as deep forest tapering to the steppe edge. Adds 1 new species (mustelid), 6 mobs (5 reuse existing species, 1 uses the new species), and a single boundary-edit to Fox Den (4156) opening the south path.

**Architecture:** Pure data zone build — no Go code changes. 20 room YAMLs in a new `the_fernway_south/` folder, 1 species YAML, 6 mob YAMLs, 1 zone-config YAML, 1 boundary edit on the existing Fernway 4156 room, 1 coordinate map update, 1 PATCH_NOTES entry. Subagent-driven build is reliable as long as each task gives the implementer the exact roomid + coord + exit map so cartesian consistency is locked at the spec level (not the implementation level).

**Tech Stack:** YAML data files. Existing engine systems: room loader, mob loader, species registry, behavior_archetype dispatcher, spawn timer, coord/map renderer. No new code.

**Sketch (the durable design):** `docs/superpowers/plans/2026-04-28-fernway-south-zone.md` (this file is both the plan and the sketch — there's no separate spec doc for this stage).

---

## File Structure

| Action | File | Responsibility |
|---|---|---|
| CREATE | `_datafiles/world/dogmud/species/24-mustelid.yaml` | New species: badger, weasel, otter, ferret, marten |
| CREATE | `_datafiles/world/dogmud/rooms/the_fernway_south/zone-config.yaml` | Zone metadata (name, defaultbiome, region) |
| CREATE | `_datafiles/world/dogmud/rooms/the_fernway_south/4157.yaml` … `4176.yaml` (20 files) | Individual rooms |
| MODIFY | `_datafiles/world/dogmud/rooms/the_fernway/4156.yaml` | Add `south: roomid: 4157` exit |
| CREATE | `_datafiles/world/dogmud/mobs/the_fernway_south/360-wild_hare.yaml` | Prey, rodent species |
| CREATE | `_datafiles/world/dogmud/mobs/the_fernway_south/361-roe_deer.yaml` | Prey, deer species |
| CREATE | `_datafiles/world/dogmud/mobs/the_fernway_south/362-honey_bees.yaml` | Combat-passive, insectoid |
| CREATE | `_datafiles/world/dogmud/mobs/the_fernway_south/363-feral_boar.yaml` | Combat-passive, boar species |
| CREATE | `_datafiles/world/dogmud/mobs/the_fernway_south/364-timber_wolf.yaml` | Generic-fighter, canine, hates boar, NOT player-hostile |
| CREATE | `_datafiles/world/dogmud/mobs/the_fernway_south/365-forest_badger.yaml` | Generic-fighter, mustelid (new species), HOSTILE |
| MODIFY | `docs/coordinate_map.md` | Add 10 Fernway rows (catch-up) + 20 Fernway South rows |
| MODIFY | `PATCH_NOTES.md` | Stage 3.0c dev-only entry |

---

## Hard reference: room layout (memorize this — every room task references it)

Coords are unique. Y range -15 to -22 (all unclaimed per coord map). X range -15 to -13.

```
        x=-15    x=-14    x=-13
y=-14            [4156]                ← Fox Den (existing Fernway)
y=-15  [4158]──[4157]──[4159]
                  │
y=-16  [4162]──[4160]──[4161]
                  │
y=-17  [4164]──[4163]──[4167]
         │        │       │
y=-18  [4165]  [4170]  [4168]
         │        │       │
y=-19  [4166]  [4171]  [4169]
                  │
y=-20            [4176]
                  │
y=-21  [4172]──[4174]──[4173]
                  │
y=-22            [4175]                (biome: plains)
```

| ID | Title | Coord | Biome | Exits |
|---|---|---|---|---|
| 4157 | Briar Tangle | (-14,-15,0) | forest | n→4156, s→4160, e→4159, w→4158 |
| 4158 | Beewood Hollow | (-15,-15,0) | forest | e→4157 |
| 4159 | Hare-Run Meadow | (-13,-15,0) | forest | w→4157 |
| 4160 | Old Burn Scar | (-14,-16,0) | forest | n→4157, s→4163, e→4161, w→4162 |
| 4161 | Pine Stand | (-13,-16,0) | forest | w→4160 |
| 4162 | Boar Wallow | (-15,-16,0) | forest | e→4160 |
| 4163 | Twin Beech Glade | (-14,-17,0) | forest | n→4160, s→4170, e→4167, w→4164 |
| 4164 | Brook Rise | (-15,-17,0) | forest | e→4163, s→4165 |
| 4165 | Heron Pool | (-15,-18,0) | forest | n→4164, s→4166 |
| 4166 | Watercress Bend | (-15,-19,0) | forest | n→4165 |
| 4167 | Deer Browse | (-13,-17,0) | forest | w→4163, s→4168 |
| 4168 | Birch Stand | (-13,-18,0) | forest | n→4167, s→4169 |
| 4169 | Salt Lick | (-13,-19,0) | forest | n→4168 |
| 4170 | Tangled Bracken | (-14,-18,0) | forest | n→4163, s→4171 |
| 4171 | Old Stand | (-14,-19,0) | forest | n→4170, s→4176 |
| 4176 | Birdsong Glade | (-14,-20,0) | forest | n→4171, s→4174 |
| 4174 | Twisted Hawthorn | (-14,-21,0) | forest | n→4176, s→4175, e→4173, w→4172 |
| 4172 | Badger Sett | (-15,-21,0) | forest | e→4174 |
| 4173 | Foxglove Clearing | (-13,-21,0) | forest | w→4174 |
| 4175 | Steppe Edge | (-14,-22,0) | **plains** | n→4174 (terminal — no south exit) |

---

## Task 1: Add the mustelid species YAML

**Files:**
- Create: `_datafiles/world/dogmud/species/24-mustelid.yaml`

Real gap in the existing species set (we have rodent and canine but nothing for the mustelid family — badgers, weasels, otters, ferrets). ID 24 is the lowest unused gap (existing species jump from 23 aberration to 30 skeleton).

- [ ] **Step 1: Confirm 24 is unused**

Run: `ls _datafiles/world/dogmud/species/24-*.yaml 2>/dev/null`
Expected: no output.

- [ ] **Step 2: Create the file**

```yaml
speciesid: 24
name: mustelid
description: |
  The mustelid family covers badgers, weasels, otters, ferrets, and
  martens — small to medium carnivores with thick muscular bodies,
  short legs, and a willingness to fight things twice their size.
  Heavy-jawed and stubborn; what they bite they tend to keep.
buffids:
  - 29 # Night Vision (mostly nocturnal hunters)
size: small
unarmedname: jaws
selectable: false
tameable: false
angrycommands:
  - 'emote bares its teeth and chitters a low warning'
  - 'emote hisses and arches its back'
  - 'emote snaps its jaws audibly'
stats:
  strength: {base: 70}
  dexterity: {base: 130}
  perception: {base: 110}
  vitality: {base: 90}
  willpower: {base: 25}
  charisma: {base: 10}
damage:
  basedamage: 6
  variance: 2
damage_multiplier: 0.55
disabledslots: [weapon, offhand, head, neck, belt, gloves, ring, legs, feet]
```

The vitality 90 + damage_multiplier 0.55 makes a small but tough fighter — appropriate for the badger-as-glass-cannon-but-not-soft profile. `tameable: false` because badgers are notoriously not pets.

- [ ] **Step 3: Boot test**

Run: `go run . 2>&1 | grep -E "species.LoadDataFiles|panic" | head -5` (timeout after ~30s).
Expected: `species.LoadDataFiles() loadedCount=N` where N is one greater than the previous baseline. No panic.

If you get a panic about a duplicate species ID or a YAML parse error, fix and retry.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/species/24-mustelid.yaml
git commit -m "$(cat <<'EOF'
feat(species): add mustelid (24) for badger + weasel + otter + ferret

Fills a real gap in the species set — no existing species covered the
mustelid family. Badger added as the first consumer in Stage 3.0c
Fernway South zone build.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Create the zone config + entry trio (rooms 4157–4159)

**Files:**
- Create: `_datafiles/world/dogmud/rooms/the_fernway_south/zone-config.yaml`
- Create: `_datafiles/world/dogmud/rooms/the_fernway_south/4157.yaml`
- Create: `_datafiles/world/dogmud/rooms/the_fernway_south/4158.yaml`
- Create: `_datafiles/world/dogmud/rooms/the_fernway_south/4159.yaml`

The "entry trio" — Briar Tangle is the south-of-Fox-Den entry; Beewood Hollow (W) and Hare-Run Meadow (E) branch off it.

- [ ] **Step 1: Create the zone-config**

`_datafiles/world/dogmud/rooms/the_fernway_south/zone-config.yaml`:

```yaml
name: The Fernway South
roomid: 4157
defaultbiome: forest
region: Windward Marches
```

`roomid` is the zone's spawn / entry room — Briar Tangle.

- [ ] **Step 2: Create 4157 Briar Tangle**

`_datafiles/world/dogmud/rooms/the_fernway_south/4157.yaml`:

```yaml
roomid: 4157
zone: The Fernway South
title: Briar Tangle
description: >
  [WRITE 4-7 sentence forest-prose description matching the existing
  Fernway voice (see the_fernway/4156.yaml for tone reference).
  THEME: just south of Fox Den the path narrows and snags with old
  bramble and thorn-windfalls. Senses: bramble dominant; thorn-cuts
  on the legs; deer-track packed beneath the bramble; smell of
  last season's leaf litter. The trail markers from Foxglade have
  stopped here. ANSI tags REQUIRED on these noun keywords:
  bramble, thorn-windfall, deer-track, leaf-litter — wrap each in
  <ansi fg="itemname">...</ansi>. Hard wrap at ~78 chars.]
biome: forest
coord:
  x: -14
  y: -15
  z: 0
exits:
  north:
    roomid: 4156
    zone: The Fernway
  south:
    roomid: 4160
  east:
    roomid: 4159
  west:
    roomid: 4158
nouns:
  bramble: |
    [1-2 sentence description. Old thorn-bramble densest along the
    eastern shoulder of the path.]
  thorn-windfall: |
    [1-2 sentences. Storm-fallen thorny boughs, half-rotted, snagged
    with last year's leaves.]
  deer-track: |
    [1-2 sentences. The packed earth underneath shows hoof-prints —
    deer use this trail more than humans do.]
  leaf-litter: |
    [1-2 sentences. Last autumn's leaves, brown and curl-edged,
    layered ankle-deep where the bramble shelters them.]
idlemessages:
  - '[idle: a wren flushes from the bramble and is gone]'
  - ''
  - '[idle: somewhere ahead, a fox barks once]'
  - ''
  - '[idle: a snagged twig breaks under settling weight]'
```

The bracketed `[WRITE ...]` and `[1-2 sentences ...]` blocks are directives, not literal placeholders — replace each with actual prose at write time. Match the voice of `_datafiles/world/dogmud/rooms/the_fernway/4156.yaml` (Fox Den) and `4151.yaml` (Bracken Mile). MUD line width 80 chars; no hard numbers in any text.

- [ ] **Step 3: Create 4158 Beewood Hollow**

`_datafiles/world/dogmud/rooms/the_fernway_south/4158.yaml`:

```yaml
roomid: 4158
zone: The Fernway South
title: Beewood Hollow
description: >
  [WRITE 4-7 sentences. THEME: a sunlit clearing west of Briar
  Tangle where wild honeybees nest in the hollows of old oaks.
  Senses: industrial bee-hum; wax-fat air; sun-warmth in a
  break of canopy; honey-smell faint but real. Required ANSI
  nouns: bee-tree, swarm, honeycomb, sun-patch.]
biome: forest
coord:
  x: -15
  y: -15
  z: 0
exits:
  east:
    roomid: 4157
nouns:
  bee-tree: |
    [1-2 sentences. A specific old oak with a wax-darkened hollow
    a body's-length up its trunk; bees come and go in steady traffic.]
  swarm: |
    [1-2 sentences. Hundreds of honeybees, wholly absorbed in the
    work of returning and departing.]
  honeycomb: |
    [1-2 sentences. Visible at the lip of the bee-tree's hollow,
    darkened with old wax.]
  sun-patch: |
    [1-2 sentences. The break in the canopy that opens this hollow
    to the sky and to the bees' need for warmth.]
idlemessages:
  - '[idle: a single bee hovers in front of you, then is gone]'
  - ''
  - '[idle: the hum of the swarm rises and falls]'
  - ''
  - '[idle: a wax-flake drifts down and lands on the leaf litter]'
```

- [ ] **Step 4: Create 4159 Hare-Run Meadow**

`_datafiles/world/dogmud/rooms/the_fernway_south/4159.yaml`:

```yaml
roomid: 4159
zone: The Fernway South
title: Hare-Run Meadow
description: >
  [WRITE 4-7 sentences. THEME: a small grass-edged forest meadow
  east of Briar Tangle, threaded with hare-runs through the cover.
  Senses: long meadow-grass; cropped hare-paths; sky open overhead
  for the first time since Foxglade; cool damp earth under the
  paths. Required ANSI nouns: meadow-grass, hare-path, droppings,
  open-sky.]
biome: forest
coord:
  x: -13
  y: -15
  z: 0
exits:
  west:
    roomid: 4157
nouns:
  meadow-grass: |
    [1-2 sentences. Knee-high meadow grass at the edges; cropped
    short along the hare-paths.]
  hare-path: |
    [1-2 sentences. A network of low cropped paths through the
    grass — clearly the work of small fast feet.]
  droppings: |
    [1-2 sentences. Fresh hare droppings along the path edges.]
  open-sky: |
    [1-2 sentences. The first real opening overhead since the
    forest closed in north of Fox Den.]
idlemessages:
  - '[idle: a hare freezes briefly at the meadow edge, then bolts]'
  - ''
  - '[idle: meadow grass bends in a passing wind]'
  - ''
  - '[idle: a hawk shadow crosses the open patch]'
```

- [ ] **Step 5: Boot test the trio + zone-config**

Run: `go run . 2>&1 | grep -E "rooms.LoadDataFiles|panic|the_fernway_south" | head -10` (timeout 30s).
Expected: `rooms.LoadDataFiles() loadedCount=N` increases by 3. A `New Mapper zone="The Fernway South-4157"` line appears with size=3. No panic.

If the loader complains about a missing exit target (4160/etc.), that's expected at this stage — we only created 4157-4159 so the south exit from 4157 points at a not-yet-existing 4160. The loader logs a warning but doesn't panic. We'll fill in the rest in subsequent tasks.

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_fernway_south/
git commit -m "$(cat <<'EOF'
feat(rooms): Fernway South entry trio (4157-4159) + zone config

Briar Tangle (entry from Fox Den), Beewood Hollow (W, bees), and
Hare-Run Meadow (E, hares). Zone metadata committed alongside.
Stage 3.0c.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Central spine + east/west branches + central hub (rooms 4160–4163)

**Files:**
- Create: `_datafiles/world/dogmud/rooms/the_fernway_south/4160.yaml` (Old Burn Scar — central, 4-way junction)
- Create: `_datafiles/world/dogmud/rooms/the_fernway_south/4161.yaml` (Pine Stand — east branch)
- Create: `_datafiles/world/dogmud/rooms/the_fernway_south/4162.yaml` (Boar Wallow — west branch)
- Create: `_datafiles/world/dogmud/rooms/the_fernway_south/4163.yaml` (Twin Beech Glade — central hub)

- [ ] **Step 1: Create 4160 Old Burn Scar**

```yaml
roomid: 4160
zone: The Fernway South
title: Old Burn Scar
description: >
  [WRITE 4-7 sentences. THEME: a long-ago wildfire cleared the
  canopy here; downed timber lies in patches of new sunlight,
  thick with mushrooms growing along the punky logs. Senses: char
  smell still faint after years; new-growth saplings competing
  for the open sky; mushrooms in damp shade of fallen logs;
  woodpecker drumming somewhere nearby. Required ANSI nouns:
  burn-scar, downed-timber, mushrooms, saplings.]
biome: forest
coord:
  x: -14
  y: -16
  z: 0
exits:
  north:
    roomid: 4157
  south:
    roomid: 4163
  east:
    roomid: 4161
  west:
    roomid: 4162
nouns:
  burn-scar: |
    [1-2 sentences. The blackened stumps and char-streaked rocks
    that mark the old fire's footprint, softened by years.]
  downed-timber: |
    [1-2 sentences. Logs from the burn, half-rotted, lying in
    patches that catch the sun.]
  mushrooms: |
    [1-2 sentences. Pale-capped fungi clustered along the logs'
    damp undersides — shadowcaps and a few less-edible cousins.]
  saplings: |
    [1-2 sentences. Young oaks and birches racing for the canopy
    gap left by the burn; some already shoulder-high.]
idlemessages:
  - '[idle: a woodpecker drums briefly somewhere east]'
  - ''
  - '[idle: a chunk of soft wood crumbles off a downed log]'
  - ''
  - '[idle: warm sunlight shifts across the burn-scar as the canopy moves]'
```

- [ ] **Step 2: Create 4161 Pine Stand**

```yaml
roomid: 4161
zone: The Fernway South
title: Pine Stand
description: >
  [WRITE 4-7 sentences. THEME: a stand of resinous pines east of
  the burn scar; air heavy with sap-smell; bark wept with hardening
  pitch in amber beads. Senses: pine-resin dominant; needle
  carpet underfoot; quiet — pines mute footfall. Required ANSI
  nouns: pine-trunk, pitch-bead, needle-carpet, wolf-sign.]
biome: forest
coord:
  x: -13
  y: -16
  z: 0
exits:
  west:
    roomid: 4160
nouns:
  pine-trunk: |
    [1-2 sentences. Tall straight pines, the bark wept with old
    pitch where a branch was lost or a beetle bored through.]
  pitch-bead: |
    [1-2 sentences. Hardened amber-gold beads of pine pitch on
    the trunks — the kind smiths and alchemists prize.]
  needle-carpet: |
    [1-2 sentences. Years of fallen pine needles, ankle-deep and
    dry, that mute every footstep.]
  wolf-sign: |
    [1-2 sentences. A scratched-up patch on one of the pines and
    a single tuft of grey fur snagged on the bark — something
    large and four-legged uses this stand.]
idlemessages:
  - '[idle: a pine needle drifts down]'
  - ''
  - '[idle: somewhere distant, a deep cough that is not a deer]'
  - ''
  - '[idle: the wind moves through the pines with a low whisper]'
```

- [ ] **Step 3: Create 4162 Boar Wallow**

```yaml
roomid: 4162
zone: The Fernway South
title: Boar Wallow
description: >
  [WRITE 4-7 sentences. THEME: a muddy hollow west of the burn
  scar where wild pigs come to wallow. Senses: ammoniac stink;
  churned mud with hoof-prints; flies; the pigs themselves
  may or may not be present. Required ANSI nouns: wallow-mud,
  hoof-prints, scratching-tree, flies.]
biome: forest
coord:
  x: -15
  y: -16
  z: 0
exits:
  east:
    roomid: 4160
nouns:
  wallow-mud: |
    [1-2 sentences. Knee-deep churned mud at the hollow's center,
    grey-brown and stinking faintly of pig.]
  hoof-prints: |
    [1-2 sentences. Cloven-hoof prints in the softer earth around
    the wallow — recent, some still fresh-rimmed.]
  scratching-tree: |
    [1-2 sentences. An old oak at the wallow's edge with bark
    rubbed smooth at boar-shoulder height.]
  flies: |
    [1-2 sentences. A persistent low cloud of small flies; brushed
    away, they return.]
idlemessages:
  - '[idle: a fly lands and is brushed off]'
  - ''
  - '[idle: a distant grunt from the brush]'
  - ''
  - '[idle: bubbles rise and pop in the wallow mud]'
```

- [ ] **Step 4: Create 4163 Twin Beech Glade**

```yaml
roomid: 4163
zone: The Fernway South
title: Twin Beech Glade
description: >
  [WRITE 4-7 sentences. THEME: a small glade dominated by two
  enormous beech trees grown so close their canopies have fused.
  This is the central hub of the southern forest — paths branch
  east, west, and south from here. Senses: dappled green light;
  beech-mast underfoot; cooler air than the burn scar to the
  north; a sense of arrival. Required ANSI nouns: twin-beech,
  beech-mast, dappled-light, path-junction.]
biome: forest
coord:
  x: -14
  y: -17
  z: 0
exits:
  north:
    roomid: 4160
  south:
    roomid: 4170
  east:
    roomid: 4167
  west:
    roomid: 4164
nouns:
  twin-beech: |
    [1-2 sentences. Two ancient beech trees grown trunk-to-trunk;
    smooth grey bark scarred with a century of weather and
    occasional carved initials nearly weathered out.]
  beech-mast: |
    [1-2 sentences. The thick layer of fallen beech-nuts and
    leaf-litter underfoot — a deer larder.]
  dappled-light: |
    [1-2 sentences. The fused canopy filters sunlight into
    shifting green patches that move with the breeze.]
  path-junction: |
    [1-2 sentences. Worn deer-tracks branch from this clearing
    in three directions — east, west, and south — beaten into
    the leaf litter by years of small feet.]
idlemessages:
  - '[idle: a beech-nut falls and bounces off a root]'
  - ''
  - '[idle: dappled light shifts as the canopy moves]'
  - ''
  - '[idle: a chickadee lands briefly in the twin beech and is gone]'
```

- [ ] **Step 5: Boot test**

Run: `go run . 2>&1 | grep -E "the_fernway_south|panic" | head -5`.
Expected: zone size now reads 7 (3 from Task 2 + 4 from Task 3). No panic. Some "exit target not found" warnings are still expected (4163 → 4170, 4167, 4164 don't exist yet).

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_fernway_south/
git commit -m "$(cat <<'EOF'
feat(rooms): Fernway South central spine 4160-4163

Old Burn Scar (canopy gap with mushrooms), Pine Stand (E branch
with pitch + wolf-sign), Boar Wallow (W branch), Twin Beech Glade
(central hub branching east/west/south). Stage 3.0c.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: West brook valley (rooms 4164–4166)

**Files:**
- Create: `4164.yaml` (Brook Rise), `4165.yaml` (Heron Pool), `4166.yaml` (Watercress Bend)

A 3-room dead-end pocket west of the central hub — water source + herbs.

- [ ] **Step 1: Create 4164 Brook Rise**

```yaml
roomid: 4164
zone: The Fernway South
title: Brook Rise
description: >
  [WRITE 4-7 sentences. THEME: a hidden brook surfaces here in
  a clear pool fed from underground. Senses: water-sound for the
  first time in the zone; cool damp air; mosses on the rocks;
  watercress and other water-loving herbs. Required ANSI nouns:
  brook, spring-pool, moss, watercress.]
biome: forest
coord:
  x: -15
  y: -17
  z: 0
exits:
  east:
    roomid: 4163
  south:
    roomid: 4165
nouns:
  brook: |
    [1-2 sentences. The brook rises clear from a moss-rimmed
    crack in the bedrock, runs a few paces, and slides off
    south through the leaves.]
  spring-pool: |
    [1-2 sentences. A small clear pool at the brook's source,
    no deeper than a forearm, the water cold to the touch.]
  moss: |
    [1-2 sentences. Bright green sphagnum moss carpets every
    rock at the spring; the kind that holds water for hours.]
  watercress: |
    [1-2 sentences. Watercress grows thick at the pool's edge —
    pungent, peppery, and the first clear sign of cooking-herb
    territory.]
idlemessages:
  - '[idle: a small leaf drifts on the brook and is carried south]'
  - ''
  - '[idle: water-sound steady in the background]'
  - ''
  - '[idle: a bubble rises from the spring-pool and breaks]'
```

- [ ] **Step 2: Create 4165 Heron Pool**

```yaml
roomid: 4165
zone: The Fernway South
title: Heron Pool
description: >
  [WRITE 4-7 sentences. THEME: the brook widens here into a
  shallow pool deep enough that a heron stalks the edges some
  mornings. Reedy banks; small fish in the slow water; the
  heron may or may not be present. Senses: still water; reed-
  rustle; fish-shimmer; the heron is atmospheric, not a mob.
  Required ANSI nouns: pool, reeds, minnows, heron-track.]
biome: forest
coord:
  x: -15
  y: -18
  z: 0
exits:
  north:
    roomid: 4164
  south:
    roomid: 4166
nouns:
  pool: |
    [1-2 sentences. The brook slows here into a shallow pool
    perhaps three paces across, the bottom visible through the
    cold clear water.]
  reeds: |
    [1-2 sentences. Marsh-reeds crowd the pool's edges, tall
    enough to hide a small heron from a casual glance.]
  minnows: |
    [1-2 sentences. Schools of small silver fish move in the
    pool's deeper edges — heron breakfast.]
  heron-track: |
    [1-2 sentences. A single line of three-toed heron prints
    in the soft mud at the pool's edge — recent enough that
    they haven't filled with water.]
idlemessages:
  - '[idle: minnows scatter and re-form in the pool]'
  - ''
  - '[idle: a reed bends in a passing breeze]'
  - ''
  - '[idle: somewhere distant, the rough croak of a heron]'
```

- [ ] **Step 3: Create 4166 Watercress Bend**

```yaml
roomid: 4166
zone: The Fernway South
title: Watercress Bend
description: >
  [WRITE 4-7 sentences. THEME: the brook curves sharply here
  into a slow side-eddy thick with watercress and blood-moss.
  Terminal — no further south exit; the brook keeps going but
  the deer-trail doesn't follow. Senses: watercress smell
  intense; soft mud underfoot; the eddy slow and quiet. Required
  ANSI nouns: eddy, watercress-mat, blood-moss, soft-mud.]
biome: forest
coord:
  x: -15
  y: -19
  z: 0
exits:
  north:
    roomid: 4165
nouns:
  eddy: |
    [1-2 sentences. The brook curves and slows here into a small
    side-pool where the current barely moves.]
  watercress-mat: |
    [1-2 sentences. A floating mat of watercress fills most of
    the eddy — pungent, peppery, ready to harvest by the
    handful.]
  blood-moss: |
    [1-2 sentences. Patches of dark red-brown blood-moss carpet
    the soft mud at the eddy's edge — alchemy ingredient, dyer's
    staple.]
  soft-mud: |
    [1-2 sentences. The bank is soft enough that boots leave
    deep prints; the watercress holds the soil together where
    it can.]
idlemessages:
  - '[idle: the eddy turns slowly, watercress drifting]'
  - ''
  - '[idle: a frog plops into the slow water from the bank]'
  - ''
  - '[idle: a watercress leaf detaches and joins the slow circle]'
```

- [ ] **Step 4: Boot test + commit**

Run: `go run . 2>&1 | grep -E "the_fernway_south|panic" | head -5`. Expect zone size 10. No panic.

```bash
git add _datafiles/world/dogmud/rooms/the_fernway_south/
git commit -m "$(cat <<'EOF'
feat(rooms): Fernway South west brook valley 4164-4166

Brook Rise (spring source), Heron Pool (atmospheric heron habitat),
Watercress Bend (terminal — watercress + blood-moss forage).
Stage 3.0c.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: East deer ridge (rooms 4167–4169)

**Files:**
- Create: `4167.yaml` (Deer Browse), `4168.yaml` (Birch Stand), `4169.yaml` (Salt Lick)

A 3-room dead-end pocket east of the central hub — deer territory.

- [ ] **Step 1: Create 4167 Deer Browse**

```yaml
roomid: 4167
zone: The Fernway South
title: Deer Browse
description: >
  [WRITE 4-7 sentences. THEME: an open patch of forest east of
  Twin Beech Glade where the understory has been browsed back by
  generations of deer. Sapling tips bitten flat; ground-cover
  cropped close. Senses: open visibility for once; deer-sign
  everywhere; hoof-prints in the soft earth. Required ANSI
  nouns: browse-line, hoof-prints, scrape, droppings.]
biome: forest
coord:
  x: -13
  y: -17
  z: 0
exits:
  west:
    roomid: 4163
  south:
    roomid: 4168
nouns:
  browse-line: |
    [1-2 sentences. The lowest branches of every sapling and
    young tree are bitten flat at exactly the same height —
    a deer's reach.]
  hoof-prints: |
    [1-2 sentences. Cloven-hoof prints, smaller and narrower
    than the boar prints north of here — roe deer.]
  scrape: |
    [1-2 sentences. A patch of bare earth where a buck has
    pawed away the leaf litter — a rut-sign or a salt-seek.]
  droppings: |
    [1-2 sentences. Dark deer droppings in tidy piles at the
    edges of the browse line.]
idlemessages:
  - '[idle: a doe steps into the open, freezes, and is gone]'
  - ''
  - '[idle: a sapling springs back from being bent]'
  - ''
  - '[idle: a distant snort from the brush — a buck warning]'
```

- [ ] **Step 2: Create 4168 Birch Stand**

```yaml
roomid: 4168
zone: The Fernway South
title: Birch Stand
description: >
  [WRITE 4-7 sentences. THEME: a small grove of paper birches
  south of Deer Browse. Bark peels in pale ribbons; deer come
  here for the bark in lean season. Senses: birch-bark dominant;
  pale silver trunks; quiet — birches don't whisper like oaks.
  Required ANSI nouns: birch-trunk, bark-strip, birch-sap, deer-rub.]
biome: forest
coord:
  x: -13
  y: -18
  z: 0
exits:
  north:
    roomid: 4167
  south:
    roomid: 4169
nouns:
  birch-trunk: |
    [1-2 sentences. Pale silver-grey birch trunks, the bark
    peeling in long ribbons that catch any light.]
  bark-strip: |
    [1-2 sentences. Strips of birch bark hang loose from
    several trunks — flammable, useful, easy to harvest.]
  birch-sap: |
    [1-2 sentences. Some trunks weep clear birch-sap from old
    deer-rub wounds — sweet to taste, prized in spring.]
  deer-rub: |
    [1-2 sentences. Several birches show smooth-rubbed patches
    at antler height where bucks have polished velvet off
    new growth.]
idlemessages:
  - '[idle: a strip of bark detaches and falls slowly]'
  - ''
  - '[idle: birch leaves shimmer briefly in a breath of wind]'
  - ''
  - '[idle: somewhere south, a deer crashes briefly through cover]'
```

- [ ] **Step 3: Create 4169 Salt Lick**

```yaml
roomid: 4169
zone: The Fernway South
title: Salt Lick
description: >
  [WRITE 4-7 sentences. THEME: a natural mineral seep at the
  far end of the deer ridge where the ground tastes of salt and
  iron. Deer congregate here. Terminal — no further south. Senses:
  bare earth licked smooth; iron tang in the air; deer-sign
  thickest in the zone. Required ANSI nouns: lick, mineral-seep,
  deer-trails, lick-marks.]
biome: forest
coord:
  x: -13
  y: -19
  z: 0
exits:
  north:
    roomid: 4168
nouns:
  lick: |
    [1-2 sentences. A bare patch of dark earth a few paces
    across, licked smooth by generations of deer tongues.]
  mineral-seep: |
    [1-2 sentences. The ground here tastes faintly of salt and
    iron — a slow underground seep brings minerals to the
    surface.]
  deer-trails: |
    [1-2 sentences. Deer-trails converge on the lick from
    every direction — a network of small worn paths through
    the surrounding cover.]
  lick-marks: |
    [1-2 sentences. The smooth bare earth shows the long
    grooved marks where tongues have worked at the same spots
    for years.]
idlemessages:
  - '[idle: a doe and her fawn appear at the lick edge, then are gone]'
  - ''
  - '[idle: the iron-salt tang on the air is briefly stronger]'
  - ''
  - '[idle: a beetle scuttles across the bare earth of the lick]'
```

- [ ] **Step 4: Boot test + commit**

Run: `go run . 2>&1 | grep -E "the_fernway_south|panic" | head -5`. Expect zone size 13. No panic.

```bash
git add _datafiles/world/dogmud/rooms/the_fernway_south/
git commit -m "$(cat <<'EOF'
feat(rooms): Fernway South east deer ridge 4167-4169

Deer Browse (browse-line clearing), Birch Stand (bark + deer-rub),
Salt Lick (mineral seep — terminal). Stage 3.0c.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: South spine, branches, and steppe terminus (rooms 4170–4176)

**Files:**
- Create: `4170.yaml` (Tangled Bracken), `4171.yaml` (Old Stand), `4176.yaml` (Birdsong Glade), `4174.yaml` (Twisted Hawthorn — south hub), `4172.yaml` (Badger Sett), `4173.yaml` (Foxglove Clearing), `4175.yaml` (Steppe Edge — terminal, biome: plains)

7 rooms — the southern spine and its branches. **4175 is biome: plains** (not forest) — the only non-forest room in the zone, marking the steppe transition.

- [ ] **Step 1: Create 4170 Tangled Bracken**

```yaml
roomid: 4170
zone: The Fernway South
title: Tangled Bracken
description: >
  [WRITE 4-7 sentences. THEME: bracken-fern higher than a man's
  head, the trail nearly swallowed. Hard to see what's ahead or
  behind. Senses: green dominant; fern-must thick; visibility
  limited; insects in the fronds. Required ANSI nouns: bracken-
  wall, narrow-trail, fern-frond, insect-hum.]
biome: forest
coord:
  x: -14
  y: -18
  z: 0
exits:
  north:
    roomid: 4163
  south:
    roomid: 4171
nouns:
  bracken-wall: |
    [1-2 sentences. Bracken-fern grown taller than head-height
    on both sides of the trail, the green densely woven.]
  narrow-trail: |
    [1-2 sentences. The trail is little more than a beaten dark
    line through the bracken — single-file, claustrophobic.]
  fern-frond: |
    [1-2 sentences. Individual fern fronds arch over the trail,
    catching at sleeves and shoulders.]
  insect-hum: |
    [1-2 sentences. The bracken hides a steady low hum of
    flies, beetles, and unseen things.]
idlemessages:
  - '[idle: a fern frond brushes your arm]'
  - ''
  - '[idle: something small crashes through the bracken east of the trail]'
  - ''
  - '[idle: the insect-hum rises briefly and falls back]'
```

- [ ] **Step 2: Create 4171 Old Stand**

```yaml
roomid: 4171
zone: The Fernway South
title: Old Stand
description: >
  [WRITE 4-7 sentences. THEME: the largest trees in the southern
  wood gather here — generation-old oaks with bark you could
  read like a history. Senses: cathedral quiet; oak-bark thick
  and furrowed; less light than anywhere else in the zone;
  reverence-feel. Required ANSI nouns: old-oak, oak-bark, deep-
  shade, root-throne.]
biome: forest
coord:
  x: -14
  y: -19
  z: 0
exits:
  north:
    roomid: 4170
  south:
    roomid: 4176
nouns:
  old-oak: |
    [1-2 sentences. Generation-old oaks, trunks thicker than
    three people could reach around, branches that close out
    the sky.]
  oak-bark: |
    [1-2 sentences. The bark is deeply furrowed and grey-brown,
    rich in the kind of tannin alchemists work into compounds.]
  deep-shade: |
    [1-2 sentences. Less light reaches this stand than anywhere
    else in the southern wood; the air feels older and cooler.]
  root-throne: |
    [1-2 sentences. The largest oak's exposed roots curl into a
    natural seat at its base — a place generations of foragers
    have rested before pressing south.]
idlemessages:
  - '[idle: an acorn drops and rolls to a stop in the leaf litter]'
  - ''
  - '[idle: deep silence here, even the wind muted]'
  - ''
  - '[idle: a single ray of sun finds a gap in the canopy and moves on]'
```

- [ ] **Step 3: Create 4176 Birdsong Glade**

```yaml
roomid: 4176
zone: The Fernway South
title: Birdsong Glade
description: >
  [WRITE 4-7 sentences. THEME: a small pleasant clearing south
  of the Old Stand where forest birds gather and call. After the
  cathedral hush of the oaks this room is full of sound. Senses:
  birdsong dominant; sun-patch in the middle; long grass; a
  fallen log to rest on. Required ANSI nouns: bird-chorus,
  sun-patch, fallen-log, long-grass.]
biome: forest
coord:
  x: -14
  y: -20
  z: 0
exits:
  north:
    roomid: 4171
  south:
    roomid: 4174
nouns:
  bird-chorus: |
    [1-2 sentences. A constant overlapping song of finches,
    chickadees, and warblers — the most birdsong anywhere
    in the zone.]
  sun-patch: |
    [1-2 sentences. The clearing's center catches direct sun
    most of the day, warming the long grass.]
  fallen-log: |
    [1-2 sentences. An old beech that fell years ago lies
    half-grown-over in the long grass — a natural bench.]
  long-grass: |
    [1-2 sentences. Knee-high grass, dry and rustling, in the
    sun-patch where the canopy doesn't reach.]
idlemessages:
  - '[idle: a chickadee calls dee-dee-dee somewhere very close]'
  - ''
  - '[idle: a warbler answers from across the glade]'
  - ''
  - '[idle: long-grass bends in a passing wind]'
```

- [ ] **Step 4: Create 4174 Twisted Hawthorn**

```yaml
roomid: 4174
zone: The Fernway South
title: Twisted Hawthorn
description: >
  [WRITE 4-7 sentences. THEME: a wind-shaped hawthorn dominates
  this last forest junction. The southern hub before the steppe.
  Branches west to the Badger Sett, east to Foxglove Clearing,
  south to Steppe Edge. Senses: wind for the first time in the
  zone — coming from the south; hawthorn berries dark red; soil
  drier here. Required ANSI nouns: hawthorn, haws, south-wind,
  drying-soil.]
biome: forest
coord:
  x: -14
  y: -21
  z: 0
exits:
  north:
    roomid: 4176
  south:
    roomid: 4175
  east:
    roomid: 4173
  west:
    roomid: 4172
nouns:
  hawthorn: |
    [1-2 sentences. A single old hawthorn, shaped by years of
    south wind into a permanent lean. The branches are dense
    with thorns and clustered red haws.]
  haws: |
    [1-2 sentences. Dark red haws — small thorn-tree berries —
    cluster thickly on the hawthorn's branches; cooked or
    pressed they yield jam, syrup, or medicine.]
  south-wind: |
    [1-2 sentences. The first real wind in the zone, coming from
    the south — drier than the forest air, carrying a hint of
    grass-smell from beyond the trees.]
  drying-soil: |
    [1-2 sentences. The soil underfoot is noticeably drier and
    sandier than the leaf-litter of the deeper forest — a sign
    the woods are giving up to the steppe.]
idlemessages:
  - '[idle: south wind moves the hawthorn briefly]'
  - ''
  - '[idle: a haw drops and rolls a step before stopping]'
  - ''
  - '[idle: dust kicks up in a small swirl from the south]'
```

- [ ] **Step 5: Create 4172 Badger Sett**

```yaml
roomid: 4172
zone: The Fernway South
title: Badger Sett
description: >
  [WRITE 4-7 sentences. THEME: a steep earth-bank west of the
  Twisted Hawthorn, riddled with badger-burrows. The smell warns
  before the sett does. Terminal — no further west. Senses:
  badger-musk dominant and unpleasant; bare-trodden earth;
  multiple burrow entries; bones at the entries from past meals.
  Required ANSI nouns: sett, burrow-entry, badger-musk, bone-pile.]
biome: forest
coord:
  x: -15
  y: -21
  z: 0
exits:
  east:
    roomid: 4174
nouns:
  sett: |
    [1-2 sentences. A steep earth-bank a body's height tall,
    riddled with the dark mouths of badger burrows.]
  burrow-entry: |
    [1-2 sentences. Several burrow entries in the bank, all
    well-trodden, the largest big enough to crawl into if you
    were truly foolish.]
  badger-musk: |
    [1-2 sentences. A heavy musky smell hangs over the sett —
    the kind of warning every other forest animal heeds.]
  bone-pile: |
    [1-2 sentences. Small bones lie scattered at the burrow
    entries — rabbit, hare, vole — the leavings of the
    sett's residents.]
idlemessages:
  - '[idle: a bone is briefly visible at a burrow mouth, then withdrawn]'
  - ''
  - '[idle: the badger-musk gets briefly stronger]'
  - ''
  - '[idle: something heavy moves in the burrow nearest the trail]'
```

- [ ] **Step 6: Create 4173 Foxglove Clearing**

```yaml
roomid: 4173
zone: The Fernway South
title: Foxglove Clearing
description: >
  [WRITE 4-7 sentences. THEME: a small wildflower meadow east of
  the Twisted Hawthorn, dominated by tall foxglove spikes in
  late-summer color. Terminal — no further east. Senses: pleasant
  flower-smell; bees at work in the foxgloves; deep purple and
  white spikes head-high; sense of peace. Required ANSI nouns:
  foxglove, flower-spike, bumblebees, sun-warmth.]
biome: forest
coord:
  x: -13
  y: -21
  z: 0
exits:
  west:
    roomid: 4174
nouns:
  foxglove: |
    [1-2 sentences. Foxgloves stand head-high in the clearing's
    center, tall purple-pink spikes hung with thimble-shaped
    flowers — beautiful and famously poisonous.]
  flower-spike: |
    [1-2 sentences. Each foxglove sends up one or two tall
    flower-spikes laden with the characteristic thimble blooms,
    deepening from white at the throat to dark purple at the
    edges.]
  bumblebees: |
    [1-2 sentences. Heavy fuzzy bumblebees move slowly from
    flower to flower, half-disappearing into each thimble in turn.]
  sun-warmth: |
    [1-2 sentences. The clearing catches direct afternoon sun
    and holds the warmth — a place to stand and rest.]
idlemessages:
  - '[idle: a bumblebee emerges from a foxglove and lands on the next one]'
  - ''
  - '[idle: a foxglove spike sways under a bumblebee landing]'
  - ''
  - '[idle: the sun-warmth holds steady through the afternoon]'
```

- [ ] **Step 7: Create 4175 Steppe Edge** (biome: plains, terminal)

```yaml
roomid: 4175
zone: The Fernway South
title: Steppe Edge
description: >
  [WRITE 4-7 sentences. THEME: the trees give out entirely south
  of the Twisted Hawthorn. Pale grass and wind-scoured stones
  stretch south into open sky. Beyond, the world opens to the
  Dustwalk — but you do not go that way today. Terminal — no
  south exit. Senses: wide open sky for the first time; constant
  south wind; pale grass; loneliness. Required ANSI nouns:
  steppe, pale-grass, scoured-stones, south-horizon.]
biome: plains
coord:
  x: -14
  y: -22
  z: 0
exits:
  north:
    roomid: 4174
nouns:
  steppe: |
    [1-2 sentences. The land south of here is steppe — open,
    treeless, wind-scoured grass and stone stretching to the
    horizon under a wide pale sky.]
  pale-grass: |
    [1-2 sentences. Knee-high pale tan grass bends and rises
    in the constant south wind. It feels older and wilder
    than forest grass.]
  scoured-stones: |
    [1-2 sentences. Round stones, weathered smooth by the
    steppe wind, scatter the ground in all directions —
    nothing to slow them, nothing to catch them.]
  south-horizon: |
    [1-2 sentences. The horizon south is unbroken — a long
    pale line where the steppe meets the sky. The
    Dustwalk lies somewhere out there. You do not go that way
    today.]
idlemessages:
  - '[idle: south wind moves the pale-grass in long even waves]'
  - ''
  - '[idle: a hawk wheels far to the south, very high]'
  - ''
  - '[idle: a single dry tumble-stem rolls past in the wind]'
```

- [ ] **Step 8: Boot test the full zone**

Run: `go run . 2>&1 | grep -E "the_fernway_south|panic" | head -10`.
Expected: `New Mapper zone="The Fernway South-4157" size=20`. No panic. No "exit target not found" warnings (every internal exit now has a corresponding room).

There WILL still be one warning: 4157 → 4156 (Fox Den) is an EXTERNAL exit and the loader may grumble that 4156 doesn't link back. That's intentional — the boundary edit on Fox Den happens in Task 8.

- [ ] **Step 9: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_fernway_south/
git commit -m "$(cat <<'EOF'
feat(rooms): Fernway South southern spine + steppe terminus 4170-4176

Tangled Bracken (head-high fern), Old Stand (cathedral oaks),
Birdsong Glade (post-oak relief), Twisted Hawthorn (south hub),
Badger Sett (W terminal — danger), Foxglove Clearing (E terminal —
peaceful), Steppe Edge (S terminal, biome: plains — the world ends).
Stage 3.0c. All 20 rooms now in place.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Create the 6 mob YAMLs

**Files:**
- Create: `_datafiles/world/dogmud/mobs/the_fernway_south/360-wild_hare.yaml`
- Create: `_datafiles/world/dogmud/mobs/the_fernway_south/361-roe_deer.yaml`
- Create: `_datafiles/world/dogmud/mobs/the_fernway_south/362-honey_bees.yaml`
- Create: `_datafiles/world/dogmud/mobs/the_fernway_south/363-feral_boar.yaml`
- Create: `_datafiles/world/dogmud/mobs/the_fernway_south/364-timber_wolf.yaml`
- Create: `_datafiles/world/dogmud/mobs/the_fernway_south/365-forest_badger.yaml`

**Folder note:** the mobs folder uses the same zone-folder name as the rooms folder — `the_fernway_south/`. Create the folder if it doesn't exist.

**Aggro design summary** (re-stating because it's the most important constraint):

| Mob | hostile | groups | hates | Aggro behavior |
|---|---|---|---|---|
| wild hare | false | [animal, rodent] | (none) | flees on sight (prey archetype) |
| roe deer | false | [animal, deer] | (none) | flees on sight (prey archetype) |
| honey bees | false | [animal, insectoid, swarm] | (none) | passive; sting if attacked |
| feral boar | false | [animal, boar] | (none) | passive; fights HARD if attacked |
| timber wolf | false | [animal, canine, predator] | [boar] | NOT player-hostile; engages boars on sight |
| forest badger | true | [animal, mustelid] | (none) | hostile to ALL on sight |

The wolf's `hostile: false` + `hates: [boar]` is the key emergent dynamic — verified working pattern via `_datafiles/world/dogmud/mobs/ironwind_steppe/205-steppe_wolf.yaml`.

- [ ] **Step 1: Create 360 wild hare**

```yaml
mobid: 360
zone: The Fernway South
behavior_archetype: prey
archetype: fighting
statpool: 25
itemdropchance: 75
hostile: false
charm_immune: false
maxwander: 3
groups:
  - animal
  - rodent
  - prey
items:
  - itemid: 40064
idlecommands:
  - 'emote freezes, ears up, listening'
  - ''
  - 'emote crops a few mouthfuls of meadow-grass'
  - ''
  - 'emote thumps a hind-foot once and bolts a few paces'
  - ''
activitylevel: 30
character:
  name: wild hare
  description: |
    A long-legged forest hare, dun-brown with black-tipped ears,
    sitting motionless in the meadow grass. It is built for one
    thing: to stop being where you can see it. The big dark eyes
    track you steadily.
  speciesid: 10
  level: 1
  gold: 0
  stats:
    strength:
      training: 5
    dexterity:
      training: 18
    perception:
      training: 16
    vitality:
      training: 6
```

`itemid: 40064` is wild hare meat (existing Fernway mat from 3.0b). `itemdropchance: 75` is the overall drop trigger (most kills yield meat, some don't); the loot table is the single `items:` entry above.

- [ ] **Step 2: Create 361 roe deer**

```yaml
mobid: 361
zone: The Fernway South
behavior_archetype: prey
archetype: fighting
statpool: 60
itemdropchance: 60
hostile: false
charm_immune: false
maxwander: 4
groups:
  - animal
  - deer
  - prey
items:
  - itemid: 40064
idlecommands:
  - 'emote lifts its head from browse, ears swivelling'
  - ''
  - 'emote takes two careful steps and stops'
  - ''
  - 'emote crops a low birch-leaf and chews slowly'
  - ''
activitylevel: 25
character:
  name: roe deer
  description: |
    A small deer of the forest interior — neat, reddish-brown in
    summer coat, with a pale rump-patch and large ears. It stops
    moving when you do, and watches you with the absolute
    stillness of an animal that has learned humans bring no good.
  speciesid: 7
  level: 2
  gold: 0
  stats:
    strength:
      training: 10
    dexterity:
      training: 16
    perception:
      training: 14
    vitality:
      training: 12
```

Reuses wild-hare-meat as the meat drop (no separate "venison" item — keeps the mat pool clean per the brief).

- [ ] **Step 3: Create 362 honey bees**

```yaml
mobid: 362
zone: The Fernway South
behavior_archetype: combat_passive
archetype: fighting
statpool: 15
itemdropchance: 50
hostile: false
charm_immune: true
maxwander: 0
groups:
  - animal
  - insectoid
  - swarm
items:
  - itemid: 40065
idlecommands:
  - "emote works the bee-tree's hollow in steady traffic"
  - ''
  - 'emote returns to the hollow heavy with pollen'
  - ''
activitylevel: 50
character:
  name: honey bees
  description: |
    A working swarm of wild honeybees — hundreds of small
    industrious bodies in steady traffic between the bee-tree
    and the meadow. Approached calmly they ignore you. Disturb
    the swarm and you will not enjoy the next minute of your
    life.
  speciesid: 12
  level: 1
  gold: 0
  stats:
    strength:
      training: 4
    dexterity:
      training: 14
    perception:
      training: 12
    vitality:
      training: 10
```

`itemid: 40065` is beeswax. `maxwander: 0` keeps the swarm rooted at the bee-tree. `charm_immune: true` because charming a hive-mind is silly.

- [ ] **Step 4: Create 363 feral boar**

```yaml
mobid: 363
zone: The Fernway South
behavior_archetype: combat_passive
archetype: fighting
statpool: 90
itemdropchance: 80
hostile: false
charm_immune: false
maxwander: 2
groups:
  - animal
  - boar
items:
  - itemid: 40064
idlecommands:
  - 'emote roots in the wallow mud with its snout'
  - ''
  - 'emote scratches its flank against the wallow tree'
  - ''
  - 'emote grunts low and continues rooting'
  - ''
activitylevel: 20
character:
  name: feral boar
  description: |
    A wild forest pig, big-shouldered and grey-bristled, snout
    caked with the wallow mud. It watches you with small mean
    eyes and continues working at the mud. Don't startle it.
  speciesid: 6
  level: 4
  gold: 0
  stats:
    strength:
      training: 18
    dexterity:
      training: 8
    perception:
      training: 10
    vitality:
      training: 20
```

High vitality + strength so it hits hard if engaged. `combat_passive` (not `prey`) because if attacked it fights, doesn't flee.

- [ ] **Step 5: Create 364 timber wolf** *(the key non-hostile-to-player mob)*

```yaml
mobid: 364
zone: The Fernway South
behavior_archetype: generic_fighter
archetype: fighting
statpool: 80
itemdropchance: 0
hostile: false
charm_immune: false
maxwander: 5
groups:
  - animal
  - canine
  - predator
hates:
  - boar
idlecommands:
  - 'emote pads silently between the pine trunks, scenting'
  - ''
  - 'emote lifts its muzzle and tests the wind'
  - ''
  - 'emote stops and listens, head cocked'
  - ''
activitylevel: 30
character:
  name: timber wolf
  description: |
    A lone grey timber wolf, lean and tall-shouldered, moving
    through the pine stand like it belongs to the trees. It
    does not look at you the way it looks at smaller things.
    Whatever it came south for, it isn't you.
  speciesid: 2
  level: 5
  gold: 0
  stats:
    strength:
      training: 14
    dexterity:
      training: 16
    perception:
      training: 18
    vitality:
      training: 14
```

**KEY:** `hostile: false` (will not attack player) + `hates: [boar]` (will engage any boar in range). `maxwander: 5` lets the wolf roam between Pine Stand (4161) and Boar Wallow (4162) via Old Burn Scar (4160), enabling the emergent wolf-vs-boar combat.

No loot — the wolf is wildlife flavor, not a kill target. (If you do kill it, no loot drops; that's intentional design.)

- [ ] **Step 6: Create 365 forest badger** *(the only player-hostile mob)*

```yaml
mobid: 365
zone: The Fernway South
behavior_archetype: generic_fighter
archetype: fighting
statpool: 70
itemdropchance: 0
hostile: true
charm_immune: false
maxwander: 1
groups:
  - animal
  - mustelid
idlecommands:
  - 'emote bares its teeth and chitters from a burrow mouth'
  - ''
  - 'emote stalks a tight circle at the sett edge'
  - ''
  - 'emote drags a small bone deeper into the burrow'
  - ''
activitylevel: 25
character:
  name: forest badger
  description: |
    A sett-defending forest badger — black-and-white striped face,
    short heavy body, claws that mean it. Smaller than you
    expected. Meaner than you expected. The musk-smell tells you
    it has not retreated.
  speciesid: 24
  level: 3
  gold: 0
  stats:
    strength:
      training: 12
    dexterity:
      training: 16
    perception:
      training: 14
    vitality:
      training: 18
```

`speciesid: 24` is the new mustelid species from Task 1. `hostile: true` + no `hates:` means it attacks anything in its room — players, other mobs, everyone. `maxwander: 1` keeps it centered on the sett (4172).

No loot for the same reason as wolf — flavor, not farm bait. (If a future stage wants badger pelts as a quest item, layer in then.)

- [ ] **Step 7: Boot test the mobs**

Run: `go run . 2>&1 | grep -E "mobs.LoadDataFiles|panic" | head -5`.
Expected: `mobs.LoadDataFiles() loadedCount=N` increases by 6. No panic. If panic about unknown speciesid 24, Task 1 didn't land — fix that first.

- [ ] **Step 8: Commit**

```bash
git add _datafiles/world/dogmud/mobs/the_fernway_south/
git commit -m "$(cat <<'EOF'
feat(mobs): Fernway South wildlife (mobs 360-365)

6 mobs for the new zone:
- 360 wild hare (prey, rodent)
- 361 roe deer (prey, deer)
- 362 honey bees (combat_passive, insectoid swarm)
- 363 feral boar (combat_passive, boar; hits hard if engaged)
- 364 timber wolf (generic_fighter, canine; NOT player-hostile,
  hates boars — emergent intra-zone hunt dynamic)
- 365 forest badger (generic_fighter, mustelid 24; the only true
  hostile in the zone)

Stage 3.0c.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Spawninfo wiring + Fox Den boundary edit

**Files:**
- Modify: `_datafiles/world/dogmud/rooms/the_fernway_south/4158.yaml` (bees spawn)
- Modify: `_datafiles/world/dogmud/rooms/the_fernway_south/4159.yaml` (hare spawn)
- Modify: `_datafiles/world/dogmud/rooms/the_fernway_south/4161.yaml` (wolf spawn)
- Modify: `_datafiles/world/dogmud/rooms/the_fernway_south/4162.yaml` (boar spawn)
- Modify: `_datafiles/world/dogmud/rooms/the_fernway_south/4167.yaml` (deer spawn — Browse)
- Modify: `_datafiles/world/dogmud/rooms/the_fernway_south/4169.yaml` (deer spawn — Salt Lick)
- Modify: `_datafiles/world/dogmud/rooms/the_fernway_south/4172.yaml` (badger spawn)
- Modify: `_datafiles/world/dogmud/rooms/the_fernway/4156.yaml` (Fox Den south exit)

**Spawn distribution recap:**

| Room | Mob | Cooldown |
|---|---|---|
| 4158 Beewood Hollow | 362 honey bees | 600 rounds |
| 4159 Hare-Run Meadow | 360 wild hare | 300 rounds |
| 4161 Pine Stand | 364 timber wolf | 1200 rounds |
| 4162 Boar Wallow | 363 feral boar | 900 rounds |
| 4167 Deer Browse | 361 roe deer | 600 rounds |
| 4169 Salt Lick | 361 roe deer | 600 rounds |
| 4172 Badger Sett | 365 forest badger | 1800 rounds |

Higher cooldowns for the rarer / tougher mobs (wolf, badger).

- [ ] **Step 1: Add spawninfo to 4158 (bees)**

In `_datafiles/world/dogmud/rooms/the_fernway_south/4158.yaml`, append at the BOTTOM of the file (after `idlemessages`):

```yaml
spawninfo:
- mobid: 362
  cooldown: 600 rounds
```

- [ ] **Step 2: Add spawninfo to 4159 (hare)**

```yaml
spawninfo:
- mobid: 360
  cooldown: 300 rounds
```

- [ ] **Step 3: Add spawninfo to 4161 (wolf)**

```yaml
spawninfo:
- mobid: 364
  cooldown: 1200 rounds
```

- [ ] **Step 4: Add spawninfo to 4162 (boar)**

```yaml
spawninfo:
- mobid: 363
  cooldown: 900 rounds
```

- [ ] **Step 5: Add spawninfo to 4167 (deer at Browse)**

```yaml
spawninfo:
- mobid: 361
  cooldown: 600 rounds
```

- [ ] **Step 6: Add spawninfo to 4169 (deer at Salt Lick)**

```yaml
spawninfo:
- mobid: 361
  cooldown: 600 rounds
```

- [ ] **Step 7: Add spawninfo to 4172 (badger)**

```yaml
spawninfo:
- mobid: 365
  cooldown: 1800 rounds
```

- [ ] **Step 8: Edit Fox Den 4156 to add the south exit**

In `_datafiles/world/dogmud/rooms/the_fernway/4156.yaml`, locate the `exits:` block (currently has only a north exit). Replace:

```yaml
exits:
  north:
    roomid: 4152
```

with:

```yaml
exits:
  north:
    roomid: 4152
  south:
    roomid: 4157
    zone: The Fernway South
```

YAML indentation is 2-space; the `roomid` and `zone` lines must be indented one level deeper than `south:`.

- [ ] **Step 9: Boot test**

Run: `go run . 2>&1 | grep -E "the_fernway|panic|exit" | head -20`.
Expected:
- No panic
- The Fernway South zone size 20 (unchanged)
- The Fernway zone size 10 (unchanged)
- No "exit target not found" warnings on either zone
- No instance saves committed (run `git status` and confirm no `mobs.instances/`, `rooms.instances/`, `shops/` files appear)

- [ ] **Step 10: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_fernway_south/ _datafiles/world/dogmud/rooms/the_fernway/4156.yaml
git commit -m "$(cat <<'EOF'
feat(spawn): Fernway South spawninfo + Fox Den south exit

Spawn distribution: bees in Beewood Hollow, hare in Hare-Run, wolf
in Pine Stand, boar in Boar Wallow, deer in Browse + Salt Lick,
badger in Sett. Rare-mob cooldowns longer (wolf 1200, badger 1800).

Fox Den (4156) gains south→4157 exit with cross-zone target. Stage 3.0c.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: coordinate_map.md update + PATCH_NOTES + smoke handoff

**Files:**
- Modify: `docs/coordinate_map.md`
- Modify: `PATCH_NOTES.md`

The coord map is currently STALE — it doesn't include the existing Fernway zone (4147-4156). This task brings it current AND adds the new Fernway South.

- [ ] **Step 1: Add 30 rows to coordinate_map.md**

Open `docs/coordinate_map.md`. The table sorts by room ID. The 4xxx range falls AFTER the existing Watchers Crossing (~line 237: row `| 427 | Watchers Crossing | ...`) and BEFORE the next zone block.

Find the current insertion point (the row immediately after `| 427 | Watchers Crossing | Watchers Crossing, East Road | -5 | 0 | 0 |`). Insert the following 30 rows in numeric ID order:

```
| 4147 | The Fernway | Fernway, Eastern Trailhead | -9 | -13 | 0 |
| 4148 | The Fernway | Stoneford Crossing | -10 | -13 | 0 |
| 4149 | The Fernway | Twin Oaks Bend | -11 | -13 | 0 |
| 4150 | The Fernway | Heron Reach | -12 | -13 | 0 |
| 4151 | The Fernway | Bracken Mile | -13 | -13 | 0 |
| 4152 | The Fernway | Foxglade | -14 | -13 | 0 |
| 4153 | The Fernway | Fernway, Western Trailhead | -15 | -13 | 0 |
| 4154 | The Fernway | Old Weddell Farmstead | -16 | -13 | 0 |
| 4155 | The Fernway | Heron Pond | -15 | -12 | 0 |
| 4156 | The Fernway | Fox Den | -14 | -14 | 0 |
| 4157 | The Fernway South | Briar Tangle | -14 | -15 | 0 |
| 4158 | The Fernway South | Beewood Hollow | -15 | -15 | 0 |
| 4159 | The Fernway South | Hare-Run Meadow | -13 | -15 | 0 |
| 4160 | The Fernway South | Old Burn Scar | -14 | -16 | 0 |
| 4161 | The Fernway South | Pine Stand | -13 | -16 | 0 |
| 4162 | The Fernway South | Boar Wallow | -15 | -16 | 0 |
| 4163 | The Fernway South | Twin Beech Glade | -14 | -17 | 0 |
| 4164 | The Fernway South | Brook Rise | -15 | -17 | 0 |
| 4165 | The Fernway South | Heron Pool | -15 | -18 | 0 |
| 4166 | The Fernway South | Watercress Bend | -15 | -19 | 0 |
| 4167 | The Fernway South | Deer Browse | -13 | -17 | 0 |
| 4168 | The Fernway South | Birch Stand | -13 | -18 | 0 |
| 4169 | The Fernway South | Salt Lick | -13 | -19 | 0 |
| 4170 | The Fernway South | Tangled Bracken | -14 | -18 | 0 |
| 4171 | The Fernway South | Old Stand | -14 | -19 | 0 |
| 4172 | The Fernway South | Badger Sett | -15 | -21 | 0 |
| 4173 | The Fernway South | Foxglove Clearing | -13 | -21 | 0 |
| 4174 | The Fernway South | Twisted Hawthorn | -14 | -21 | 0 |
| 4175 | The Fernway South | Steppe Edge | -14 | -22 | 0 |
| 4176 | The Fernway South | Birdsong Glade | -14 | -20 | 0 |
```

Verify Fernway coords against actual room files — values above are read from the YAMLs at plan-write time.

- [ ] **Step 2: Update the "Total reachable" header line**

The current line says `Total reachable: 274 rooms across multiple zones`. After adding 30 rows (10 Fernway catch-up + 20 Fernway South), the number should be `Total reachable: 304 rooms across multiple zones`. (Adjust if the actual current count differs from 274 — re-count or run `go run . 2>&1 | grep "reachable"` if there's a server-side count emitted at boot.)

- [ ] **Step 3: Update the coord range summary**

Find the `## Coordinate Range Summary` block (currently around line 417). Update the Y range:

Old:
```
- Y range: -14 to 11 (height: 26)
```

New:
```
- Y range: -22 to 11 (height: 34)
```

X range and Z range stay the same (the new zone fits within X -15 to -13, Z 0 — both already covered).

- [ ] **Step 4: Add Stage 3.0c entry to PATCH_NOTES.md**

After the title `# DOGMud Patch Notes` and above the existing 2026-04-28 Stage 3.0d entry, insert:

```markdown
## 2026-04-28 — Stage 3.0c: Fernway South Zone (dev only)

**Note:** Dev-only landing. The full economy stack ships to prod (`master`)
as a coherent update once Stage 3.4 lands.

- New 20-room zone south of the existing Fernway, themed as deep
  forest tapering to the steppe edge. Connects from Fox Den (4156)
  via a new south exit; terminates at Steppe Edge (4175, biome:
  plains) with a one-way view of the Dustwalk beyond.
- New mustelid species (24) — fills a real gap in the species set
  (we had rodent and canine but nothing for badgers, weasels,
  otters). First consumer is the forest badger; future zones with
  otters or weasels reuse immediately.
- Six new wildlife mobs (360-365): wild hare, roe deer, honey bees,
  feral boar, timber wolf, forest badger. Only the badger is
  hostile to players — the rest are atmosphere or forage support.
  Wolf is `hostile: false` but `hates: [boar]` — emergent
  intra-zone hunt dynamic where the wolf may engage boars without
  threatening the player.
- The 6 existing Fernway forage mats (oak bark, shadowcap mushroom,
  wild hare meat, beeswax, blood-moss, pine pitch from 3.0b) gain
  fresh territory to spawn in. No new mats added.
- Stage 3.0c is the territory groundwork for Stage 3.1 forager
  NPCs — the forest is now big enough for a Fernway-based forager
  to wander, gather, and recall to depot when injured.

```

- [ ] **Step 5: Final boot + smoke handoff**

Run: `go build ./... && go test ./... 2>&1 | tail -5`.
Expected: clean build, all tests pass.

Run: `go run . 2>&1 | grep -E "panic|the_fernway_south|reachable" | head -10`.
Expected: zone loads at size 20, no panic, no missing-exit warnings.

**Manual smoke checklist** (per `docs/CONTENT_GENERATION_GUIDE.md` Section 2 — defer to user):

```
[ ] Walk every room (4157-4176). Each title and description reads
    cleanly (no missing punctuation, broken ANSI tags, dropped
    sentences, surviving [WRITE ...] directives).
[ ] Verified every exit. Every room reachable; one-way terminal
    south at 4175 is intentional.
[ ] No mapsymbol/maplegend on any non-landmark room. (None should be
    set in this zone — verify with grep.)
[ ] Cartesian consistency: run `map` from a few spread-out rooms,
    confirm no overlap with adjacent existing zones (Fernway is the
    only neighbor; no overlap possible since the new zone occupies
    Y < -14 territory previously empty).
[ ] Fight ≥1 mob of each combat archetype used in the zone:
    - prey (wild hare or roe deer — should flee on sight)
    - combat_passive (honey bees or feral boar — passive until hit)
    - generic_fighter (timber wolf — should NOT attack on sight;
      forest badger — SHOULD attack on sight)
[ ] Spawn the wolf and a boar in the same area (4160-4162 corridor)
    and watch — wolf should engage boar via the hates list.
[ ] Kill at least one mob (boar or badger) and loot the corpse.
    Confirm wild hare meat or beeswax drops where wired.
[ ] Identify a forage mat from the existing Fernway pool (e.g.,
    pick up a beeswax drop, run `identify`). Stats render cleanly,
    no raw numbers leak.
[ ] No instance saves committed: `rooms.instances/the_fernway_south/`,
    `mobs.instances/`, `shops/` should NOT appear in `git status`.
[ ] go build ./... clean. go test ./... clean.
```

When all 10 boxes tick, Stage 3.0c is done. Phase 2 quests (if any) come in a separate pass — likely deferred to align with Stage 3.1 forager wiring.

- [ ] **Step 6: Commit**

```bash
git add docs/coordinate_map.md PATCH_NOTES.md
git commit -m "$(cat <<'EOF'
docs(3.0c): coord map catch-up + Fernway South + patch notes

- coordinate_map.md: added 10 missing Fernway rows (catch-up; the
  doc was stale before this) plus 20 Fernway South rows. Y range
  summary extended from -14 to -22.
- PATCH_NOTES: Stage 3.0c dev-only entry.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Out of scope reminder (push back if scope creep tries to add)

- **Quests in this zone** — Phase 2 work, deferred. Aligns naturally with Stage 3.1 forager wiring.
- **New foragable mats** — the existing 6 Fernway mats (40062-40067) are the supply pool.
- **New zone-specific items** — no curio loot (boar tusk, badger pelt) per user direction.
- **3.0a (west-of-Stillwater zone)** — separate Stage. Sequence after 3.0c.
- **Forager NPCs** — Stage 3.1 work. The territory exists; the NPC will be wired separately.

## Done = ?

All 9 tasks complete, all commits landed on `development` branch, manual smoke (Step 5 of Task 9) all green. Per the multi-stage caravan/economy effort: this lands on `development` only. Nothing ships to `master` until Stage 3.4 lands.
