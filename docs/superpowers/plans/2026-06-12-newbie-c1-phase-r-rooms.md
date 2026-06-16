# Newbie Chunk 1 — Phase R Implementation Plan (Rooms + Nouns)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Author the Pothole Coulee zone scaffold and all 26 chunk-1 rooms (19 hub + 7 spoke stubs) with full prose, nouns, coordinates, and sanctuary flags — ending at the user review gate before any mobs/items exist.

**Architecture:** Pure YAML content in a new zone folder, validated by the offline coordinate scanner, boot-time zone consistency checks, and a scripted walkthrough transcript for the user's review. No Go changes in this phase.

**Tech Stack:** Room YAML (`_datafiles/world/dogmud/rooms/pothole_coulee/`), `tools/coord_inventory.py`, boot smoke, AI-port walkthrough.

**Working directory (ALL tasks):** `C:/Users/Calabe Davis/workspace/DOGMud/.claude/worktrees/feature+newbie-area` — this is an isolated git worktree on branch `worktree-feature+newbie-area`. NEVER work in the main checkout. Never `git add -A` (runtime artifacts).

---

## Verified facts (do not re-derive)

- **Sub-spec (the design of record):** `docs/superpowers/specs/2026-06-12-newbie-chunk1-hub-subspec.md` — §3 holds the authoritative room/stub manifests (ids, names, biomes, coords, exits). The coordinate convention is the ENGINE one: north = y−1, south = y+1, east = x+1, west = x−1, up = z+1.
- **Room YAML shape** (model: `_datafiles/world/dogmud/rooms/sanctum_basin/101.yaml`):
  ```yaml
  roomid: 101
  zone: Sanctum Basin          # zone DISPLAY name, not folder
  title: South Cliffs Overlook
  description: You stand on a narrow basalt ledge... (long prose, YAML-folded)
  biome: desert
  coord:
    x: -11
    y: 8
    z: 0
  mutators:
  - mutatorid: sanctuary
  exits:
    east:
      roomid: 102
  idlemessages:
  - A gust of wind carries <ansi fg="itemname">dust</ansi> off the cliff edge...
  ```
- **Nouns shape** (model: `_datafiles/world/dogmud/rooms/ashwick/4015.yaml`):
  ```yaml
  nouns:
    gate: A wooden gate set into the low stone wall, its iron
      hinges rusted open in a permanent welcome. ...
  ```
  Multi-word nouns must be hyphenated single tokens (`notice-board`), per the
  parser limitation.
- **Zone-config shape** (model: `_datafiles/world/dogmud/rooms/stillwater/zone-config.yaml`):
  ```yaml
  name: Stillwater
  roomid: 4100        # zone root room
  defaultbiome: plains
  region: Windward Marches
  ```
- **Folder naming:** zone display name `Pothole Coulee` → folder `pothole_coulee` (ConvertForFilename: lowercase, non-alnum → underscore). Room files are `{roomid}.yaml`.
- **Biomes available** (all exist in `_datafiles/world/dogmud/biomes/`): water, shore, city, house (indoor), cliffs. `house` carries `indoor: true` (weather-sheltered).
- **Sanctuary mutator** exists (`_datafiles/world/dogmud/mutators/sanctuary.yaml`) — no-combat + 5× regen.
- **Scanner:** `python tools/coord_inventory.py` must report 0 global collisions; all new coords must land inside the reserve x[30..59], y[-15..14], z[-3..3] (`docs/superpowers/specs/newbie-area-coord-budget.md`).
- **Boot baseline:** `rooms.loadAllRoomZones() zoneCount=22 loadedCount=523`. After this phase: zoneCount=23, loadedCount=549 (+26).
- **AI port for the walkthrough:** 55555, admin smoketester/smoke123test, pace ~3s (2 cmds/round), `teleport 5200` works for admins. RoomId teleports verified in prior smokes.

## Global prose requirements (every room, every task)

1. **No hard numbers** in any player-facing string. Ever.
2. **≤80 chars per rendered line** (YAML folded scalars re-wrap; keep source lines ≤80 too).
3. **Voice:** channeled-scablands frontier — basalt, glacial-flood geology, plunge-pool water, dry coulee washes, scrubby steppe above; folksy and earthy, not grand. The Opened (visible mutation) is NORMAL here — rooms never treat mutation as alarming. Second person, present tense, like the existing world prose.
4. **Descriptions 3–6 sentences.** Each room mentions at least one feature that is also a `nouns:` entry, so `look <noun>` is discoverable from the prose.
5. **Nouns:** 2–4 per room (the sub-spec's per-room intents below name the REQUIRED ones; add 0–2 more if natural). Each noun's text is 2–5 sentences, same voice. Hyphenate multi-word keys.
6. **Idlemessages:** 1–2 per OUTDOOR room (water/shore/city/cliffs), optional for indoor (`house`) rooms. Use the `<ansi fg="itemname">`/`<ansi fg="mobname">` accent convention sparingly, as sanctum_basin/101 does.
7. **Every room: `mutators: [{mutatorid: sanctuary}]`** (list form as in the model file) — hub AND stubs are safe ground in chunk 1.
8. **Exits exactly per the sub-spec manifest** — reciprocal pairs, no extras. Stubs have ONLY the return exit.
9. **`zone: Pothole Coulee`** in every room file.

## Room-by-room content intents

(The authoritative ids/biomes/coords/exits live in sub-spec §3 — copy them
exactly. This table adds the prose intent + REQUIRED nouns.)

| Id | Title | Prose intent | Required nouns |
|---|---|---|---|
| 5200 | The Awakening Pool | The centerpiece: a deep basalt plunge pool, unnaturally still, faint light in the depths; stone bowl walls; the rite happens here. Reverent but not churchy. | pool, basalt, light |
| 5201 | West Shore Path | Packed-gravel path hugging the west shore; stilt-houses visible south; reeds north. | path, reeds, stilts |
| 5202 | East Shore Path | East shore gravel; the Mending Hut's herb smell drifts; overlook crag above. | path, herbs, crag |
| 5203 | Hub Square | Small town heart: a well-worn gathering space of flat basalt slabs, notice post, the crier's pitch. | square, notice-post, slabs |
| 5204 | Market Row | Short row of awnings and stalls between inn and store; barter sounds. | stalls, awnings |
| 5205 | The Drowned Lantern (inn) | Common room: lantern made from a salvaged diving-bell, long tables, peat fire. | lantern, tables, fire |
| 5206 | Lantern Sleeping Loft | Low-beamed loft, straw ticks and wool blankets; THE sleep-lesson room — prose invites resting. | beds, blankets, beams |
| 5207 | Coulee Provisions (store) | Cramped, well-stocked frontier store; shelves to the ceiling; smells of rope and tallow. | shelves, counter, goods |
| 5208 | Strongbox House (bank) | Squat basalt blockhouse, iron-banded strongboxes, a ledger desk. | strongboxes, ledger, door |
| 5209 | The Mending Hut | Warm healer's hut: drying herbs, cots, clean bandages; the room players will call home (sethome teaching lands here in Phase D — prose should feel like waking somewhere safe). | cots, herbs, bandages |
| 5210 | Wickerwork Cottage | The folk counterpoint: woven willow walls, older charms hung among Church tokens, hearth smoke. | charms, wickerwork, hearth |
| 5211 | Basalt Stair | Switchback stair cut into the bowl wall, pool below, shelf above. | stair, basalt |
| 5212 | School Shelf | Open basalt shelf over the pool; benches, wind, the whole basin visible. | benches, shelf, view |
| 5213 | Chrysalis School Hall | Single-hall school: slate boards, mismatched desks, the notice board that carries the help-system teaching text (Phase D fills the board's deeper text; the NOUN exists now with orientation flavor incl. an in-character mention that `help <topic>` answers most questions and that a player can adjust their text width — the §7 `set linewidth` surface). | notice-board, desks, slates |
| 5214 | Cleric's Study | Book-crowded study; rubbings of concentric-arc iconography (quiet orbital-truth foreshadow — NOT explained); a kettle. | books, rubbings, kettle |
| 5215 | The Threshold House | The portal room: a free-standing basalt arch, older than the town, faint hum; warden's stool beside it. | arch, hum, stool |
| 5216 | Stilt-House Walk | Boardwalk between stilted homes over the shallows; nets and floats. | boardwalk, nets, stilts |
| 5217 | North Shore Overlook | Crag with the best view of the pool's depths; the light below is clearest from here. | crag, depths |
| 5218 | Reed Jetty | Short fishing jetty in the reeds; tied coracle; dragonflies. | jetty, coracle, reeds |

Stubs (5220–5226): ONE room each, 2–3 sentences + per-spoke teaser line
(user-approved option: per-spoke one-liners), 1–2 nouns, `mutators:
sanctuary`, single return exit. Teaser intents:

| Id | Teaser intent |
|---|---|
| 5220 | Dry training canyon beyond; sound of someone striking a post; "the way is being cleared." (Martial) |
| 5221 | Talus slope and a cold forge-smell; timbers stacked for shoring. (Forge) |
| 5222 | Reedy wash, sweet rot and herb smell; flagged stakes mark a path being cut. (Alchemy) |
| 5223 | Scrub steppe rising; game trails; a hunter's cairn. (Wilderness) |
| 5224 | A cut toward a ruin's silhouette; the air feels folded, expectant. (Folding) |
| 5225 | Old field track; standing stone just visible; cart ruts. (Lore) |
| 5226 | Bluff steps up toward shooting terraces; spent practice bolts in the dirt. (Ranged) |

---

### Task 0: Zone scaffold + root room

**Files:**
- Create: `_datafiles/world/dogmud/rooms/pothole_coulee/zone-config.yaml`
- Create: `_datafiles/world/dogmud/rooms/pothole_coulee/5200.yaml`

- [ ] **Step 1: zone-config**

```yaml
name: Pothole Coulee
roomid: 5200
defaultbiome: shore
region: Channeled Scablands
```

(Read 2 other zone-configs first to confirm no additional required fields;
`region` is cosmetic grouping — a new region name is fine, mirror the field.)

- [ ] **Step 2: Room 5200** — full room per the model shape: id/zone/title/
description/biome `water`/coord (45,0,0)/sanctuary mutator/nouns (pool,
basalt, light)/idlemessages (1–2)/exits per sub-spec §3 (W→5201, E→5202,
S→5203, N→5211). The exit target rooms don't exist yet — **verify the
loader tolerates exits to not-yet-existing rooms at boot** (other zones
under construction historically did this; if boot panics on dangling
exits, author 5200 with NO exits in this task and add them in Task 1 —
report which).

- [ ] **Step 3: Boot check** — wipe instance saves
(`rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`),
boot ~40s, expect `zoneCount=23` and loadedCount=524, zero panics. Check
whether the loader needs `_datafiles/world/dogmud/rooms.instances/pothole_coulee/`
to exist (CreateZone makes one at runtime; hand-authored zones may need
`mkdir`) — create it if boot complains; note that instance dirs are
gitignored. Kill the server; re-wipe.

- [ ] **Step 4: Commit**
```bash
git add _datafiles/world/dogmud/rooms/pothole_coulee/
git commit -m "content(newbie-c1): Pothole Coulee zone scaffold + Awakening Pool root room"
```

---

### Task 1: Pool ring (6 rooms: 5201, 5202, 5211, 5216, 5217, 5218)

**Files:** Create `5201.yaml`, `5202.yaml`, `5211.yaml`, `5216.yaml`, `5217.yaml`, `5218.yaml` in the zone folder.

- [ ] **Step 1:** Author all six per the global requirements + per-room
intents. Ids/biomes/coords/exits EXACTLY per sub-spec §3 (e.g. 5201:
shore, (44,0,0), E→5200 W→5210 S→5216 N→5218). Exits referencing rooms
from later tasks (5210, 5203, 5208, 5209) are authored NOW on these rooms
— reciprocals land with their rooms (confirm dangling-exit tolerance from
Task 0; if intolerable, defer those exit lines to the task that creates
the target and note it).
- [ ] **Step 2:** `python tools/coord_inventory.py` → 0 collisions; boot
check (+6 rooms, loadedCount=530), kill, re-wipe instances.
- [ ] **Step 3:** Commit: `git commit -m "content(newbie-c1): pool ring rooms (shores, stair base, overlook, jetty, stilt walk)"`

---

### Task 2: Town south (7 rooms: 5203, 5204, 5205, 5206, 5207, 5208, 5215)

**Files:** Create the seven YAMLs.

- [ ] **Step 1:** Author per intents. Note 5205→5206 is U/D (z+1 loft).
5206's prose is the sleep-lesson room — make resting *inviting*. 5213's
notice-board is NOT here (school task) — 5203's notice-post is the
outdoor crier's post, distinct.
- [ ] **Step 2:** Scanner + boot (+7 → loadedCount=537), kill, re-wipe.
- [ ] **Step 3:** Commit: `git commit -m "content(newbie-c1): town south rooms (square, market, inn+loft, store, bank, threshold house)"`

---

### Task 3: East/west edges + school (5 rooms: 5209, 5210, 5212, 5213, 5214)

**Files:** Create the five YAMLs.

- [ ] **Step 1:** Author per intents. 5209 (Mending Hut) is the future
sethome anchor — homey, safe, convalescent. 5213's `notice-board` noun
carries the help/linewidth orientation flavor (in-character; no numbers —
"the board explains how to ask the world for help on any topic"). 5214's
`rubbings` noun is the orbital foreshadow — concentric arcs, no
explanation.
- [ ] **Step 2:** Scanner + boot (+5 → loadedCount=542), kill, re-wipe.
- [ ] **Step 3:** Commit: `git commit -m "content(newbie-c1): mending hut, wickerwork cottage, school shelf/hall/study"`

---

### Task 4: Spoke stubs (7 rooms: 5220–5226) + host exits

**Files:** Create the seven stub YAMLs; modify host rooms `5209.yaml`, `5210.yaml`, `5207.yaml`, `5216.yaml`, `5218.yaml`, `5217.yaml` (exit additions per sub-spec §3 stub table).

- [ ] **Step 1:** Author the stubs: 2–3 sentence descriptions + the
per-spoke teaser intents, 1–2 nouns each, sanctuary, ONLY the return
exit. Add the outbound exits on the six host rooms (5209 gets two: E→5220
and N→5226).
- [ ] **Step 2:** Scanner (all 26 in-reserve, 0 collisions) + boot (+7 →
loadedCount=549, zoneCount=23), kill, re-wipe.
- [ ] **Step 3:** Commit: `git commit -m "content(newbie-c1): seven spoke-mouth stubs with per-spoke teasers"`

---

### Task 5: Phase audits + review-gate walkthrough artifact

**Files:** Create `docs/superpowers/specs/newbie-c1-phase-r-walkthrough.txt` (review artifact; committed).

- [ ] **Step 1: Audit sweep**
```bash
python tools/coord_inventory.py            # 0 collisions, all inside reserve
grep -rn "[0-9]" _datafiles/world/dogmud/rooms/pothole_coulee/*.yaml | grep -vE "roomid|coord|x:|y:|z:|5[0-9]{3}"   # no-numbers audit: hits = review each (ansi codes ok? there should be none in prose)
```
Manual checks: every room has ≥2 nouns; every noun key is a single token;
every description mentions ≥1 noun; reciprocal exits match the manifest
exactly (script it: a small python assert over the YAMLs comparing to the
sub-spec table is 20 lines and worth it — write it inline, run it, show
output; don't commit it unless it's clean enough to keep under tools/).

- [ ] **Step 2: Boot + consistency** — full boot, confirm
`ValidateZoneConsistency` reports 0 errors/0 warnings for the new zone
(mode is `panic` in config — a clean boot IS the proof). `cartcheck
Pothole Coulee` via the AI port as admin for the explicit per-zone report.

- [ ] **Step 3: Walkthrough transcript (the user's review artifact)** —
scripted AI-port session: `teleport 5200`, then walk EVERY room in a
deterministic order; in each room run `look` plus `look <noun>` for every
authored noun. Strip ANSI; save the full transcript to
`docs/superpowers/specs/newbie-c1-phase-r-walkthrough.txt` with a room
header per section. Kill the server, re-wipe instances. This file is what
the user reads at the review gate — completeness matters more than
brevity.

- [ ] **Step 4: Commit**
```bash
git add docs/superpowers/specs/newbie-c1-phase-r-walkthrough.txt <tools script if kept>
git commit -m "content(newbie-c1): phase-R audits + review walkthrough transcript"
```

- [ ] **Step 5: STOP — review gate.** Phase R ends here. Report to the
controller; the user reviews the walkthrough + files before Phase M
(mobs + items) begins. Do NOT start mobs.

---

## Self-review notes

- Sub-spec §3 coverage: all 19 hub rooms (T0–T3) + 7 stubs (T4) + nouns
  requirement + sanctuary on all + exact exits/coords; §8 acceptance items
  1, 2, 6, and the §3 noun rules are exercised in T5; §8 items 3–5, 7
  (rite/portal/sethome/quests) belong to later phases by design.
- The dangling-exit question (T0 step 2) is the one sequencing unknown;
  both outcomes are handled in-plan (author-with-exits vs defer-exit-lines).
- Walkthrough artifact gives the user a single readable review document
  rather than 26 YAML files.
