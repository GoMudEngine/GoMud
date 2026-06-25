# South Road + Amber Valley Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first leg of the Southern Road — a 15-room connector south
from the Ashwick crossroads into the 35-room destination town of Amber Valley
(Davan's home), with one branching quest (The Water Dispute) and seeded lore for
a deferred follow-up.

**Architecture:** Data-only content (room/mob/item/quest/dialogue/schedule YAML),
no Go engine changes. Two new zones (`south_road`, `amber_valley`), attaching to
the existing Marches Spur Road crossroads (room 4014). The controller drives all
shell (boot tests, cartcheck, harness, git); subagents Write/Edit YAML only.

**Tech Stack:** GoMud room/mob/quest/dialogue/forager YAML, the quest engine,
the dialogue engine, the schedule system, the playtest harness.

**Spec:** `docs/superpowers/specs/2026-06-25-south-road-amber-valley-design.md`

---

## Engine reference (verified — use these patterns, do not guess)

**Room YAML** (template, modeled on `rooms/kilnreach_works/5450.yaml`):
```yaml
roomid: <id>
zone: <Zone Display Name>          # e.g. "South Road" / "Amber Valley"
title: <Room Title>
description: >
  Three layers, hard-wrapped at ~78 chars: an atmospheric hook (lead with a
  varied sense — sound/smell/heat, not always sight), a paragraph of grounded
  physical detail, and at least one sentence pointing at an interactable. Names
  of examinable/interactable nouns are wrapped <ansi fg="itemname">like so</ansi>.
biome: <land|swamp|city|...>       # Amber Valley dry-valley rooms: land
coord: {x: <int>, y: <int>, z: <int>}
exits:
  <dir>:
    roomid: <id>
    zone: <Other Zone>             # ONLY when the exit crosses a zone boundary
nouns:
  <noun phrase>: <a paragraph worth reading — a direction, warning, clue, or
    flavor. Hard-wrapped. At least 2 nouns per room beyond exits.>
spawninfo:                          # optional; mobs that spawn in this room
  - mobid: <id>
    respawnrate: "<N real minutes>"
idlemessages:                       # optional; ambient room lines
- <line>
```

**Zone-config** (`rooms/<folder>/zone-config.yaml`, modeled on Greywater Flats):
```yaml
name: <Zone Display Name>
roomid: <the zone's lowest/entry roomid>
defaultbiome: <biome>
region: Windward Marches
```

**CRITICAL — folder naming:** the loader derives the expected folder from
`ConvertForFilename(zone)` (lowercase, a-z/0-9 kept, others → `_`). Zone "South
Road" → folder `south_road`; "Amber Valley" → `amber_valley`. A mismatch panics
at load (`filesystem path ... did not end in Filepath()`). Room files are named
`<roomid>.yaml` (just the number). Mob files `<mobid>-<ConvertForFilename(name)>.yaml`.

**Cartesian consistency (HARD):** the web mapper crawls exit deltas, so coords
must be collision-free and exits reciprocal. A startup pass
(`ValidateZoneConsistency`, `GamePlay.MapConsistencyEnforce: panic` in this repo)
PANICS on coord collisions, non-reciprocal exits, or wrap exits in non-wrap
zones. The `cartcheck <zone>` admin command reports the same. Every exit must
have a reciprocal back-exit; every room a unique (x,y,z).

**Interior rooms use cardinal/vertical exits, NOT `enter`/`leave`** (NP lesson):
an inn common-room reached from the road is `east`/`west`/`up` from the yard, at
an offset coord — never an `enter` portal.

**Mob YAML** (non-combatant townsperson — model on an NP anchor; hostile fauna —
model on `mobs/greywater_flats/9195-a_marsh_adder.yaml`):
```yaml
mobid: <id>
zone: <Zone Display Name>
behavior_archetype: noncombat_questgiver   # townsfolk; omit/var for fauna
non_combatant: true                        # townsfolk; false/omit for fauna
hostile: false                             # true for cave fauna
archetype: ""|fighting                     # fighting for fauna
statpool: <int>
maxwander: <0..2>
groups: [<group>, ...]
idlecommands:                              # REQUIRED — alternating emote/'' lines
  - 'emote <does something characterful>.'
  - ''
activitylevel: <int>
character:
  name: <Name>
  description: |
    Appearance incl. a UNIQUE mutation if the NPC has one (no two alike per zone).
  speciesid: <id>
  level: 1
  gold: <int>
  stats: { <stat>: { training: <int> }, ... }
schedule_id: <id>                          # town anchors only
```

**Quest / dialogue / SOPs (carry over from the just-shipped NP questlines —
non-negotiable):**
1. **Grant node FIRST** under the giver's `tree.nodes` (dialogue `TreeAdvance`
   matches in file order by SUBSTRING; a short lore trigger shadows a later gated
   grant node).
2. **Re-grant prevention:** every `grantsQuest: "X-start"` node lists BOTH
   `"X-start"` and `"X-end"` in `questExcluded`.
3. **Discoverability:** grant node `triggers` include `"quest"` and `"task"`; the
   giver's **root `hints` advertise the hook** (the feel-pass lesson — do not
   hide the quest behind guessing "help").
4. **A trigger may only `grant` a DECLARED STEP token** — granting an undeclared
   intermediate panics at load. Final completion grants the `end` step directly.
5. **room_interact fires on `look <noun>` / `examine <noun>`** (exact noun) — so
   interaction hints say **"examine the &lt;object&gt;"**, not "fix/clear/make".
6. **Reward-block keys are tag-less** (`gold`, `rep_faction`, `rep_amount`,
   `playermessage`, `itemid`); trigger-action keys are snake_case (`give_gold`,
   `set_flag`, `bump_rep`, `give_item`, `send_text`, `npc_say`).
7. NPC `text`/`npc_say` first person; `hints`/`send_text` narrator second person;
   no hard numbers in player-facing text; 80-char wrap.

**Quest trigger schema** (`internal/questengine/types.go`): events `room_interact`
(`room`,`noun`), `item_give` (`mob`,`item`), `quest_granted` (`quest_token`),
`mob_death` (`mob`); conditions `has`/`missing`/`has_item`/`missing_item`/
`has_flag`/`missing_flag`; actions `grant`/`give_item`/`give_gold`/`set_flag`/
`bump_rep`/`send_text`/`npc_say`/`room_text`.

**Smoke SOP:** before each local boot, wipe instance saves:
`rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`
(NOT `shops/`). Server is windowless/safe from Bash:
`go run . > /tmp/boot.log 2>&1 &`, wait for `Server Ready`, grep for `panic` /
`ValidateZoneConsi`. **Kill ALL `GoMud`/`go` processes before each reboot**
(stale instances linger on port 55555 — the feel-pass lesson).

**IDs (verified free 2026-06-25):** rooms 6040–6089; mobs 9394+; items 40121+;
next quest id likely 72 (verify with `python tools/id_inventory.py --type quests`).

---

## Task 0: Branch + coordinate skeleton + zone-configs + the attach

**Files:**
- Create: `rooms/south_road/zone-config.yaml`, `rooms/amber_valley/zone-config.yaml`
- Modify: `rooms/marches_spur_road/4014.yaml` (add south exit)
- Create (reference doc): `docs/superpowers/plans/south-amber-coordmap.md`

- [ ] **Step 1: Branch + ID sanity**
```bash
git checkout master && git checkout -b feature/south-road-amber-valley
python tools/id_inventory.py --type rooms   # expect next free >= 6040
python tools/id_inventory.py --type mobs     # >= 9394
python tools/id_inventory.py --type quests   # note the next free (likely 72)
```
Expect 6040 / 9394 free. If not, STOP and report.

- [ ] **Step 2: Read the attach room's coords**
```bash
sed -n '/^coord:/,/^[a-z]/p' "_datafiles/world/dogmud/rooms/marches_spur_road/4014.yaml"
```
Confirms 4014 is at x=-8, y=-13, z=0. South = decreasing y.

- [ ] **Step 3: Author the coordinate skeleton** — create
`docs/superpowers/plans/south-amber-coordmap.md` assigning every roomid 6040–6089
a unique `(x,y,z)` and its exits, per this geometry (the room-authoring tasks read
this file and place rooms exactly here):
  - **South Road (6040–6054):** a southward road spine down x=-8 from y=-14. The
    spine is the main road; the inn (interior) and the farmstead (interior) branch
    off the spine via a cardinal exit at an offset x (e.g. spine room at (-8,y),
    inn-common at (-7,y) reached by `east`). Lay ~12 spine rooms y=-14..-25 plus
    ~3 side rooms (inn common, farmstead interior, a west-side overlook). 6040
    attaches north to 4014 (reciprocal: 4014 `south`→6040 / 6040 `north`→4014
    with `zone: Marches Spur Road`).
  - **Amber Valley (6055–6089):** a town south of the road. Town center is a small
    grid around a market square (~y=-26..-29, x=-10..-6); residential/farms spread
    south and east; valley edges + cave to the west/up (the cave uses `z: -1`
    descending, or a west spur — keep coords unique); the grove is a short walk
    south-east of town. The cave's 4–5 rooms must have unique coords (descend with
    `down`/`z-1` or spur west). The town's south stub points toward the unbuilt
    River Road (no exit to a nonexistent room — a signed dead-end).
  - Every room gets exactly one (x,y,z); no two share coords; every exit has its
    reciprocal listed. This file is the single source of spatial truth.

- [ ] **Step 4: Write the two zone-config files**
```yaml
# rooms/south_road/zone-config.yaml
name: South Road
roomid: 6040
defaultbiome: land
region: Windward Marches
```
```yaml
# rooms/amber_valley/zone-config.yaml
name: Amber Valley
roomid: 6055
defaultbiome: land
region: Windward Marches
```

- [ ] **Step 5: Add the south exit on 4014** — in
`rooms/marches_spur_road/4014.yaml`, add under `exits:`:
```yaml
  south:
    roomid: 6040
    zone: South Road
```
(6040's reciprocal `north`→4014 is authored in Task 1.)

- [ ] **Step 6: Commit**
```bash
git add docs/superpowers/plans/south-amber-coordmap.md \
        _datafiles/world/dogmud/rooms/south_road/zone-config.yaml \
        _datafiles/world/dogmud/rooms/amber_valley/zone-config.yaml \
        _datafiles/world/dogmud/rooms/marches_spur_road/4014.yaml
git commit -m "feat(south-road): groundwork — coordmap, zone-configs, 4014 attach"
```
(Boot test deferred to Task 1, after 6040 exists — 4014's south→6040 dangles until then; that's fine pre-boot, but do not boot between Step 5 and Task 1.)

---

## Task 1: South Road — all 15 rooms (6040–6054)

**Files:** Create `rooms/south_road/6040.yaml` … `6054.yaml`

Dispatch ONE subagent (a linear road; no intra-task ID collision). Give it: the
coordmap rows for 6040–6054, the room template above, the ZONE_EXPANSION quality
bar, and these per-room roles (from the spec). Biome `land`.

- [ ] **Step 1: Author the rooms.** Roles:
  - **6040–6049 (Stage A — the descent):** the crossroads descent; road winding
    through drying terrain (orchards thinning to scrub); the **waypoint inn** at
    the midpoint (a yard room + an interior common room with the innkeeper — see
    Task 6 for the mob; the room just needs the space + an `<ansi>` inn-sign / hearth
    noun); **traveling merchants** pass northbound (ambient — spawn placed in Task
    6); a **shepherd**'s pasture/vantage (the shepherd NPC placed Task 6); a room
    with the **valley opening below** (a vista noun). Sensory variety across the
    run (heat rising, the smell of dust then fruit, the sound of bells/sheep).
  - **6050–6054 (Stage B — valley approach):** orchards and irrigated farms; warm
    air, sun-baked earth, ripening fruit; a **farmstead** whose **dried-up
    irrigation channel** is a prominent `<ansi fg="itemname">dried channel</ansi>`
    noun (the Water Dispute breadcrumb — its `look` text notes the water's been
    cut off and a neighbour is blamed). 6054 connects south to Amber Valley 6055
    (`zone: Amber Valley`; reciprocal authored in Task 2).
  Each room: 3-layer description, ≥2 nouns, 80-char wrap, ~20% with a container
  noun. Assign coords EXACTLY per the coordmap. 6040 `north`→4014
  (`zone: Marches Spur Road`).

> **Worked example (one room — match this quality, vary every room):**
> ```yaml
> roomid: 6041
> zone: South Road
> title: The Drovers' Descent
> description: >
>   The heat finds you before the view does — it climbs off the pale road
>   in slow waves, and the cool of the crossroads behind is gone within a
>   dozen steps. The way bends south and downward here, the verge-grass
>   yellowing where the orchard country gives out, and the first
>   <ansi fg="itemname">milestone</ansi> of the southern road leans at the
>   bend, its cut face softened by weather. Below and ahead the land falls
>   open: a long warm valley hazed gold with afternoon, the green threads
>   of irrigation channels stitched across it. A
>   <ansi fg="itemname">drystone wall</ansi> runs along the western verge,
>   tumbled in one place where something heavy went over it.
> biome: land
> coord: {x: -8, y: -15, z: 0}
> exits:
>   north: {roomid: 6040}
>   south: {roomid: 6042}
> nouns:
>   milestone: A squat stone, its cut face reading "AMBER VALLEY" above a
>     worn figure that was once a number of miles and is now a smooth dish.
>     Someone has scratched a newer mark beneath it — a tally of days, four
>     strokes and a fifth struck through, the count of a walk done often.
>   drystone wall: Fieldstone laid without mortar, the patient work of
>     hands that had more time than timber. Where it has tumbled, the gap
>     is wide enough to step through, and the grass beyond is cropped low —
>     sheep have been using the breach as a gate.
> ```

- [ ] **Step 2: Boot test (controller)**
```bash
# kill all GoMud/go first, wipe instances, then:
go run . > /tmp/boot_t1.log 2>&1 &   # wait Server Ready
grep -iE "panic|did not end in|ValidateZoneConsi" /tmp/boot_t1.log | tail -3
```
Expect: Server Ready, ValidateZoneConsistency errors=0 mode=panic, no panic. If a
coord-collision/reciprocity error fires, fix the offending room's coord/exit per
the coordmap and re-boot.

- [ ] **Step 3: Commit**
```bash
git add _datafiles/world/dogmud/rooms/south_road/
git commit -m "feat(south-road): South Road — 15 rooms (6040-6054) + Ashwick attach"
```

---

## Task 2: Amber Valley — Town Center (6055–6064)

**Files:** Create `rooms/amber_valley/6055.yaml` … `6064.yaml`

One subagent. Coordmap rows 6055–6064; biome `land`; town-center grid.

- [ ] **Step 1: Author the rooms.** Roles (spec Stage A): the **market square**
(hub; fruit stalls; the Water-Dispute farmers grumble here — Task 6/8); the
**Rite pavilion** (where Blooming ceremonies happen — atmospheric, lore-bearing;
holds the town **records** the quest's research path needs, as an
`<ansi>records</ansi>` noun); a **general store** (vendor mob in Task 6); **The
Golden Bough inn** (yard + common room; innkeeper in Task 6 — the rumor/breadcrumb
hub); the **woodworker's shop** (Davan's father's trade — his home is Task 3);
townsfolk-thronged lanes. 6055 connects north to South Road 6054 (reciprocal +
`zone: South Road`). Town NPCs wear mutation with pride — leave room in prose for
that (the NPCs carry it; rooms can mention "folk whose changes catch the light").
≥2 nouns/room, 3-layer, 80-char.

- [ ] **Step 2: Boot test** (as Task 1 Step 2, `/tmp/boot_t2.log`). errors=0.
- [ ] **Step 3: Commit**
```bash
git add _datafiles/world/dogmud/rooms/amber_valley/
git commit -m "feat(amber-valley): Town Center — 10 rooms (6055-6064)"
```

---

## Task 3: Amber Valley — Residential & Farms (6065–6074)

**Files:** Create `rooms/amber_valley/6065.yaml` … `6074.yaml`

One subagent. Coordmap rows 6065–6074.

- [ ] **Step 1: Author the rooms.** Roles (spec Stage B): **Davan's family home**
(the woodworker's house — carvings, an empty place at the bench where the son
worked; Davan's father lives here, Task 6); **irrigated orchards**; a **vineyard**;
the **irrigation river/main channel** that feeds the valley (the source of the
dispute — the channel runs from here toward the dry farmstead; an examinable
`<ansi>head-gate</ansi>` / sluice noun that the dispute turns on); the **two
feuding farms** (each a room with its farmer — Task 6 — and its side of the water
story in the prose/nouns). Container nouns ~20%. 3-layer, ≥2 nouns, 80-char.

- [ ] **Step 2: Boot test** (`/tmp/boot_t3.log`). errors=0.
- [ ] **Step 3: Commit**
```bash
git add _datafiles/world/dogmud/rooms/amber_valley/
git commit -m "feat(amber-valley): Residential & Farms — 10 rooms (6065-6074)"
```

---

## Task 4: Amber Valley — Valley Edges + Cave (6075–6084)

**Files:** Create `rooms/amber_valley/6075.yaml` … `6084.yaml`

One subagent. Coordmap rows 6075–6084.

- [ ] **Step 1: Author the rooms.** Roles (spec Stage C): foothills, dry scrub,
old ridge paths; the **collapsed irrigation section** up here (a
`<ansi fg="itemname">collapsed channel</ansi>` noun — the Water-Dispute "fix the
source" path; `look`/examine on it with the quest active clears it, Task 8); the
**cave system** (4–5 contiguous rooms descending via `down`/unique coords — the
combat pocket; sun-adapted fauna spawn here, Task 6; a modest loot/forage noun
at the deepest room); the **south stub** toward the River Road (a signed
dead-end — prose points onward, NO exit to a nonexistent room). 3-layer, ≥2
nouns, 80-char. Cave rooms lean on sound/dark/cool/echo for sensory variety.

- [ ] **Step 2: Boot test** (`/tmp/boot_t4.log`). errors=0 (watch cave coords —
descending rooms need unique (x,y,z); use `z:-1` for cave depth or a west spur).
- [ ] **Step 3: Commit**
```bash
git add _datafiles/world/dogmud/rooms/amber_valley/
git commit -m "feat(amber-valley): Valley Edges + Cave — 10 rooms (6075-6084)"
```

---

## Task 5: Amber Valley — The Chrysalis Grove (6085–6089)

**Files:** Create `rooms/amber_valley/6085.yaml` … `6089.yaml`

One subagent. Coordmap rows 6085–6089.

- [ ] **Step 1: Author the rooms.** Roles (spec Stage D): a quiet, reverential
sacred site outside town; **stone markers** commemorating notable Bloomings (deep
lore on how the community understands the Chrysalis — mutation as gift, not
curse); and, in the innermost room, a **hidden marker that predates the
theology** — `<ansi fg="itemname">weathered marker</ansi>` whose `look` text
reveals an eight-pointed inner-orbit symbol worn almost flat, older than the
Rite, set here before anyone in the valley can account for (SEEDED lore — ties to
the orbital-symbol mystery; do NOT wire any quest or Bloom mechanic to it). The
grove connects to town (reciprocal). 3-layer, ≥2 nouns, 80-char.

- [ ] **Step 2: Boot test** (`/tmp/boot_t5.log`). errors=0. **Run cartcheck**
once both zones are fully roomed:
```bash
# in-game as admin, or confirm via the boot-time ValidateZoneConsistency:
grep -iE "ValidateZoneConsi" /tmp/boot_t5.log | tail -1   # errors=0 warnings=0
```
- [ ] **Step 3: Commit**
```bash
git add _datafiles/world/dogmud/rooms/amber_valley/
git commit -m "feat(amber-valley): Chrysalis Grove — 5 rooms (6085-6089) + seeded orbital marker"
```

---

## Task 6: Mobs + spawns (9394+)

**Files:** Create `mobs/south_road/*.yaml`, `mobs/amber_valley/*.yaml`; Modify the
room files to add `spawninfo` for each placed mob.

One subagent (single ID block 9394+, no collision). Roster (assign sequential
ids from 9394; each NPC with a UNIQUE mutation where mutated, `idlecommands`,
`groups`, and ≥ the stats its role needs):

**South Road:**
- **The shepherd** — `non_combatant`, local-knowledge dialogue anchor (Task 7).
- **Traveling merchants (×1–2)** — `non_combatant`, ambient, northbound flavor.
- **The waypoint innkeeper** — `non_combatant`, anchor (schedule in Task 9).

**Amber Valley:**
- **Innkeeper of The Golden Bough** — anchor, the rumor/breadcrumb hub.
- **General-store keeper** — vendor (give it a `shop:`/vendor setup matching an
  existing valley/general vendor; goods = valley produce + basics).
- **Davan's father (the woodworker)** — anchor; warm, grounded; the emotional
  centre (dialogue Task 7). Unique mutation (e.g. a craftsman's change — hands).
- **Farmer A & Farmer B** — the two Water-Dispute parties (quest NPCs, Task 8);
  each a `non_combatant` with their side.
- **The traveling Rite deacon** — `non_combatant`; SEEDED lore only (Task 7),
  no quest.
- **A struggling youth** — ambient `non_combatant`; visibly mid-accelerating-
  change (SEEDED foreshadow; no quest, no Bloom mechanic).
- **2–4 townsfolk** — ambient, each a unique mutation worn with pride.
- **Cave fauna** — hostile `fighting` mobs (model on the marsh adder 9195):
  sun-adapted lizards (low), a hawk/raptor, and one tougher **valley predator** as
  the cave's depth threat. statpool scaled for a corridor-level traveller (≤ ~36).

- [ ] **Step 1: Author the mob YAMLs** (9394+), each with `idlecommands`,
`groups`, `zone`, unique-mutation descriptions. Town anchors get `schedule_id`
referencing schedules authored in Task 9 (use the ids you'll create there).
- [ ] **Step 2: Add `spawninfo` to the room files** for each placed mob (match
the `spawninfo: [{mobid, respawnrate}]` format; townsfolk in their rooms, fauna
in the cave rooms, merchants/shepherd along the road).
- [ ] **Step 3: Boot test** (`/tmp/boot_t6.log`): `mobs.LoadDataFiles` count up,
no `did not end in Filepath()` (check mob filenames), errors=0.
- [ ] **Step 4: Commit**
```bash
git add _datafiles/world/dogmud/mobs/south_road/ _datafiles/world/dogmud/mobs/amber_valley/ \
        _datafiles/world/dogmud/rooms/south_road/ _datafiles/world/dogmud/rooms/amber_valley/
git commit -m "feat(south-amber): NPCs + cave fauna (9394+) + spawns"
```

---

## Task 7: Dialogue (NPC personalities + seeded lore)

**Files:** Create `dialogue/south_road/<mobid>.yaml`, `dialogue/amber_valley/<mobid>.yaml`

One subagent. Every named NPC gets a dialogue file with `greetings`, `patterns`,
`tree.root` (+ hints), and ≥3 topic nodes beyond any quest function (the quality
bar). Voice: NPC `text` first person; `hints` second person.

- [ ] **Step 1: Author dialogue.** Key NPCs:
  - **Shepherd** — the valley, the water tension below (a Water-Dispute
    breadcrumb), the weather/road; root hints advertise local knowledge.
  - **Golden Bough innkeeper** — the town, rumor (the Water-Dispute tension as a
    breadcrumb), the Rite; root hints mention the farmers' feud as gossip.
  - **Davan's father (woodworker)** — his trade/carving, **the son who left**
    (warm, worried, proud), the valley's changes; 3+ topics, no quest. A
    memorable anchor.
  - **Rite deacon** — the Chrysalis Rite, this year's **unusually dramatic
    Bloomings** (SEEDED hook — lore only, phrased to invite a future quest), the
    grove. NO `grantsQuest`. Root hints hint he's troubled by something.
  - **Farmer A & Farmer B** — each gives their side of the dispute (Task 8 wires
    the quest grant onto ONE of them or a neutral; see Task 8).
  - **General-store keeper, townsfolk** — light flavor, 3 topics where it fits.
- [ ] **Step 2: Boot test** (`/tmp/boot_t7.log`): dialogue loads, no
`validateQuestExclusions` warnings (none expected — no grants yet), errors=0.
- [ ] **Step 3: Commit**
```bash
git add _datafiles/world/dogmud/dialogue/
git commit -m "feat(south-amber): NPC dialogue + seeded Rite-deacon/Davan's-father lore"
```

---

## Task 8: The Water Dispute quest (branching, 3 paths)

**Files:** Create `quests/<id>-the_water_dispute.yaml` (id from Task 0 Step 1,
likely 72 — use the verified value); Modify the giver's dialogue (grant node) +
the two farmers' dialogue; Modify the relevant rooms for `room_interact` nouns
(the collapsed channel; the town records).

**Design (from spec):** two farms fight over irrigation. Giver = a neutral hook
(the innkeeper or the market — recommend the **Golden Bough innkeeper** grants the
quest, since a neutral giver fits "mediate between two parties"; the farmers each
state their side but don't grant). 3 breadcrumbs: both farmers grumble (their
dialogue), the innkeeper's rumor, the dried-channel noun (Task 1) + the head-gate
noun (Task 3). Flag `<id>-outcome` values `[mediated, restored, farmerA, farmerB]`
(use the farmers' actual names as the branch keys).

- [ ] **Step 1: Write the quest YAML.** Steps: `start` → `end` (single resolving
step; each path is a different trigger that grants `end` + sets the flag + reward
atomically — all three converge on `end`). Structure:
```yaml
questid: <id>
name: The Water Dispute
description: >-
  Two farms at the valley's dry edge are at each other's throats over the
  water that used to feed them both. The channel between them has been cut,
  and each blames the other. Someone from outside might see it clearer.
secret: false
flags:
  - key: outcome
    values: [mediated, restored, <farmerA>, <farmerB>]
    description: "How the water dispute was settled"
steps:
  - id: start
    description: >-
      The valley's two dryside farms are feuding over a cut irrigation
      channel. Hear both sides, and find a way to settle it.
    hint: >-
      Ask both farmers their side. You might broker a compromise, restore
      the water at its source up in the foothills, or settle it by the old
      written agreement in the town records.
  - id: end
    description: >-
      The water dispute is settled -- one way or another, the valley's edge
      has its answer.
rewards:
  gold: 60
  playermessage: >-
    <neutral close — the valley settles, the channel decided>
  roommessage: >-
    <short observed-resolution line>
triggers:
  # PATH A — mediate: give_item/none; resolved via the giver's dialogue node
  #   granting end + set_flag mediated (dialogue grants the end token directly;
  #   reward block fires on end). See Step 2.
  # PATH B — restore the source: examine the collapsed channel in the foothills
  - event: room_interact
    room: <collapsed-channel room 607x>
    noun: collapsed channel
    conditions: { has: ["<id>-start"], missing: ["<id>-end"] }
    actions:
      - set_flag: {key: "<id>-outcome", value: "restored"}
      - send_text: >-
          You haul the fallen stone clear and re-set the broken lip of the
          channel until the water finds its old path and runs again --
          downhill, to both farms at once. Neither of them gets to be right.
          The valley does.
      - grant: "<id>-end"
  # PATH C — the record: examine the town records in the Rite pavilion
  - event: room_interact
    room: <records room 605x>
    noun: records
    conditions: { has: ["<id>-start"], missing: ["<id>-end"] }
    actions:
      - set_flag: {key: "<id>-outcome", value: "<farmerA or farmerB per the record>"}
      - send_text: >-
          The old water-right is here in a clerk's careful hand, dated and
          witnessed: the head-gate share belongs to <farmer>, plainly, and
          has for two generations. It is not a kind answer. It is the true one.
      - grant: "<id>-end"
```
> Note: each path grants `<id>-end` directly (declared step — SOP #4). The reward
> block fires once, on `end`. The flag records which path. PATH A is granted in
> the giver's dialogue (Step 2). Replace `<...>` with the real room ids, farmer
> names, and quest id before shipping.

- [ ] **Step 2: Wire the giver + farmers' dialogue.** On the **innkeeper**,
prepend a grant node FIRST:
```yaml
    - id: water_start
      triggers: ["quest", "task", "water", "dispute", "feud", "farmers",
                 "channel", "help"]
      questExcluded: ["<id>-start", "<id>-end"]
      grantsQuest: "<id>-start"
      text: "<1st person — the two dryside farms, the cut channel, each
        blaming the other; she's tired of the grumbling in her common room>"
      hints: "<2nd person — hear both farmers, then mediate, restore the
        water at its source in the foothills, or settle it by the old record
        in the town hall. Ask about the quest or the task.>"
```
And a **PATH A mediate** node (gated `questRequired: ["<id>-start"]`,
`questExcluded: ["<id>-end"]`) on the giver OR a farmer, reached after the player
has heard both sides — its action set is in dialogue via `grantsQuest: "<id>-end"`
+ `setsQuestFlag: {key: "<id>-outcome", value: "mediated"}`. (Dialogue grants the
`end` token directly — reliable for the reward block, which keys on holding `end`,
not on a `quest_granted` event.) Update the innkeeper's **root hints** to mention
the feud (the feel-pass discoverability lesson). Each farmer's dialogue states
their side and points the player at the three paths.

- [ ] **Step 3: Confirm the `room_interact` nouns exist** — the `collapsed
channel` (Task 4 room) and `records` (Task 2 Rite pavilion) nouns must be present
in those rooms' `nouns:`; if absent, add them (with `<ansi fg="itemname">`).
- [ ] **Step 4: Boot test** (`/tmp/boot_t8.log`): quest loads (count +1), NO
"undeclared flag" / "unknown step" panic, no `validateQuestExclusions` warning.
- [ ] **Step 5: Commit**
```bash
git add _datafiles/world/dogmud/quests/ _datafiles/world/dogmud/dialogue/ \
        _datafiles/world/dogmud/rooms/amber_valley/
git commit -m "feat(amber-valley): The Water Dispute quest (mediate/restore/record)"
```

---

## Task 9: Schedules for the town anchors

**Files:** Create `schedules/amber_valley/<id>.yaml` (+ `schedules/south_road/` if
the inn/shepherd warrant one); the `schedule_id` fields were set in Task 6.

One subagent. Light daily routines for the town anchors (innkeeper, woodworker/
Davan's father, general-store keeper; the shepherd/waypoint innkeeper on South
Road). Follow the schedule schema (`docs/schemas/schedule.md`): cover all 24
hours, valid target rooms, no coverage gaps (the validator panics on gaps).

- [ ] **Step 1: Read the schema + an existing NP anchor schedule** as the model
(`schedules/new_plymouth_*/`), then author each anchor's schedule (work hours at
their shop/post, meal, sleep). Keep targets to rooms that exist and are reachable.
- [ ] **Step 2: Boot test** (`/tmp/boot_t9.log`): schedule validators pass (they
panic on coverage gaps / unreachable targets / unresolved `schedule_id`), errors=0.
- [ ] **Step 3: Commit**
```bash
git add _datafiles/world/dogmud/schedules/ _datafiles/world/dogmud/mobs/
git commit -m "feat(amber-valley): daily schedules for the town anchors"
```

---

## Task 10: Foraging (optional enhancement — discovery-first)

**Files:** Create valley-produce items (40121+); wire per the discovered mechanism.

**This is the lowest-priority task — trim it (YAGNI) if the forage system proves
heavier than a few items.** Foraging is biome-driven (`actions.Forage`).

- [ ] **Step 1: Discover the mechanism** — read `internal/actions/forage*.go`
(or wherever `actions.Forage` lives) and ONE existing forageable item to learn how
an item is declared eligible for a biome's forage pool (a biome/forage field on
the ItemSpec, or a biome→item table). Report the exact declaration.
- [ ] **Step 2: Create 3–4 warm-valley produce items** (40121+) declared
forageable for the valley biome (orchard fruit, wine-grapes, a dry-scrub herb),
following the discovered pattern. If the mechanism is non-trivial or unclear,
STOP and report — do not invent a system; the leg ships fine without forage.
- [ ] **Step 3: Boot test** + a quick `forage` check in-game (Task 11 covers the
live check). Commit:
```bash
git add _datafiles/world/dogmud/items/
git commit -m "feat(amber-valley): warm-valley forageable produce (40121+)"
```

---

## Task 11: Harness playtest + fixes

**Files:** fixes across prior tasks as needed; report under
`tools/playtest/reports/`.

Controller drives the harness (subagents are shell-denied). Kill all GoMud/go,
wipe instances, boot, connect `mudagent` (see the NP playtest pattern). Drive one
command per round; read events with `python ... encoding='utf-8', errors='replace'`.

- [ ] **Step 1: Walk the leg** — from Ashwick crossroads 4014 `south` into South
Road, down to Amber Valley. Confirm the attach, the inn, the shepherd, the town
center, that NPCs are present and have idle behaviour + 3 topics, and that the
seam reciprocates (cartcheck-clean already, but walk it).
- [ ] **Step 2: The Water Dispute, ≥2 of 3 paths** — confirm the innkeeper's root
hints surface the feud (discoverability), the quest grants on a natural word, the
two farmers give their sides, and at least the **restore** path (`look collapsed
channel`) and the **record** path (`look records`) each complete + set the flag +
pay out. Verify `questtoken flags` shows `<id>-outcome`. (Use `questtoken` admin
to set up state quickly; NPCs wander on schedules — teleport to current locations.)
- [ ] **Step 3: The cave** — confirm fauna spawn and are fightable + survivable at
corridor-traveller power; tune statpool if needed.
- [ ] **Step 4: Seeded lore** — `look` the grove's weathered marker and the
deacon's dramatic-Bloomings lore reachable but un-wired (no quest fires).
- [ ] **Step 5: Fix anything found; commit fixes; write the report** to
`tools/playtest/reports/2026-06-25-local-south-road-amber-valley.md`.

---

## Task 12: Final review, status update, merge

- [ ] **Step 1: Final content review** — dispatch a reviewer over the branch diff
(`git diff master...feature/south-road-amber-valley`) checking: 80-char wrap;
3-layer rooms; ≥2 nouns/room; unique mutations (no dupes per zone); grant node
first + end-token in questExcluded + quest/task triggers; giver root hints
advertise the hook; "examine" verb phrasing on interaction hints; no hard numbers
in player-facing text; reward-block tag-less keys; reciprocal exits. Fix flags.
- [ ] **Step 2: Final clean boot (panic mode)**
```bash
# kill all GoMud/go, wipe instances
go run . > /tmp/boot_final.log 2>&1 &   # wait Server Ready
grep -iE "panic|did not end|unknown step|undeclared|ValidateZoneConsi|LoadDataFiles" /tmp/boot_final.log | tail -8
```
Expect: Server Ready; rooms +50, mobs +N, quests +1, items +forage; ValidateZone
Consistency errors=0 mode=panic; no panics.
- [ ] **Step 3: Update `docs/ZONE_EXPANSION.md`** — set South Road and Amber
Valley to ✅ Built in the status table (with roomid ranges + the attach note), and
update the TOTAL / built count.
- [ ] **Step 4: Merge to master `--no-ff`** (prod is current after the 2026-06-25
push, so this is normal feature work — it ships in a later bundle, no special hold
needed, but do NOT push unless the user asks):
```bash
git add docs/ZONE_EXPANSION.md && git commit -m "docs(zones): mark South Road + Amber Valley built"
git checkout master
git merge --no-ff feature/south-road-amber-valley -m "Merge: South Road + Amber Valley (Phase 5 leg 1)

15-room connector from the Ashwick crossroads + the 35-room Amber Valley
(Davan's home, Chrysalis Rite culture). The Water Dispute quest (3 paths);
Rite Deacon's Concern seeded-not-wired; cave dungeon; Bloom link latent.
Rooms 6040-6089, mobs 9394+, items 40121+. Harness-playtested; boot-clean."
```
- [ ] **Step 5: Update project memory** — note the leg built + the next Southern
Road leg (River Road → Confluence) as the remaining frontier.

---

## Self-review notes (plan author)

- **Spec coverage:** South Road (Task 1 ✓), Amber Valley 4 stages (Tasks 2–5 ✓),
  the Water Dispute 3 paths (Task 8 ✓), seeded deacon/grove-marker/youth (Tasks
  5–7 ✓, explicitly un-wired), cave combat pocket (Task 4 rooms + Task 6 fauna ✓),
  foraging (Task 10, optional ✓), the 4014 attach (Task 0/1 ✓), quality bar +
  feel-pass lessons (baked into every authoring task + Task 12 review ✓), latent
  Bloom link (no mechanics anywhere — only lore prose ✓).
- **Coordinate safety:** Task 0 produces a single coordmap as the spatial source
  of truth; every room task places rooms per it; boot-time ValidateZoneConsistency
  (mode=panic) + cartcheck are the hard verifiers after each task.
- **Build-time fills flagged (not placeholders — explicit verify-then-fill):** the
  exact quest id (Task 0 Step 1), the farmers' names as flag branch keys (Task 8),
  the `room_interact` room ids for the collapsed channel + records (Task 8 Step 1,
  filled from Tasks 2/4), and the forage declaration mechanism (Task 10 Step 1).
- **Consistency:** roomid ranges partition cleanly (SR 6040–6054; AV town
  6055–6064 / residential 6065–6074 / edges+cave 6075–6084 / grove 6085–6089);
  ids, zone names, and folder names used identically across tasks; no quest
  trigger grants a non-step token (all paths grant `end` directly).
