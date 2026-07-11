# Cascade Pass Road (#20) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build Cascade Pass Road — a hidden, solo-endgame road that branches east off Kingsbarrow Vale into forest and a mountain pass, foreshadowing the trapped lethal ship and stubbing toward the Eastern Highlands (#21).

**Architecture:** A 20-room overworld connector zone (folder `cascade_pass_road`, roomids 6323–6342), authored as YAML data files (rooms/mobs/dialogue/schedules) — no engine code changes. The zone is reached only through a `secret: true` east exit added to existing room Kingsbarrow Vale 5441 (revealed by `search`, or followed via Reth's map from Quest 75). Two mini-stages: forest road (6323–6332, z0) climbing to a mountain pass (6333–6342, z1–z3 via vertical exits). Combat mobs use `statpool` tuned directly (275 base / 550 tough) since overworld spawns skip instance gold-scaling.

**Tech Stack:** GoMud YAML data files; `python -c "import yaml"` for validation; local server boot for load-validation (`ValidateZoneConsistency` panic mode); the `/playtest` mudagent harness for feel-testing.

**Reference spec:** `docs/superpowers/specs/2026-07-01-cascade-pass-road-design.md`

---

## Conventions (bake into EVERY subagent prompt)

- **Zone folder** MUST equal `ConvertForFilename("Cascade Pass Road")` = `cascade_pass_road`. Rooms → `_datafiles/world/dogmud/rooms/cascade_pass_road/<roomid>.yaml`; mobs → `mobs/cascade_pass_road/<mobid>-<convertforfilename(name)>.yaml`; dialogue → `dialogue/cascade_pass_road/<mobid>.yaml`; schedules → `schedules/cascade_pass_road/<id>.yaml`.
- **Prose rules:** NO colon-space `": "` in any prose value (YAML reads it as a key → load panic; use em-dash `—`). NO semicolons in NPC dialogue `text:`/`hints:` (parsed as a command separator → drops the rest; use ` -- ` or a period). Use `|` literal block scalars for any NPC `text:`/`description:` ≥ ~120 chars (double-quoted flow scalars truncate in-game). Do NOT wrap a hyphenated compound across folded-scalar line breaks (renders "rough- dressed" with a stray space).
- **Line width:** hard-wrap all player-visible prose at ~78–80 cols.
- **No hard numbers** in any player-facing text (describe, don't display mechanics).
- **Mob names** must equal `internal/casing.Title(name)` — interior minor words lowercased (a, an, and, as, at, but, by, for, from, in, nor, of, on, or, the, to, with), first AND last words always capitalized ("A Lone Hunter-Wolf", "The One Who Came Back"). Filename = `mobid-ConvertForFilename(name).yaml` (lowercase, keep the article: "The Pass-Apex" → `9541-the_pass_apex.yaml`). Use the correct `speciesid` for animals (canine 2, bird 9, lagomorph 10; look up others in `ironwind_steppe` mobs).
- **Each mob gets a DISTINCT Chrysalis mutation** — cross-check against the WHOLE zone roster (Task 4 table below assigns them; do not duplicate).
- **Biomes:** only `farmland`, `forest`, `mountains`, `cliffs`, `land` are used here (all confirmed valid). `wilderness` is INVALID.
- **Terminus/onward stubs must NOT invite `go <onward>`** — frame the far edge as country-not-yet-crossable, not a walkable exit.
- **Git:** stage EXPLICIT pathspecs, NEVER `git add -A` (dirty repo). Work on branch `feature/cascade-pass-road`.
- **Dialogue node order:** place gated / lore grant nodes FIRST under `tree.nodes` (short triggers substring-match topics in file order).

## File structure

```
_datafiles/world/dogmud/
  rooms/cascade_pass_road/           # 20 rooms 6323–6342 (Tasks 2,3)
    zone-config.yaml                 # zone metadata (Task 1)
    6323.yaml … 6342.yaml
  mobs/cascade_pass_road/            # 8 mobs 9535–9542 (Task 4)
  dialogue/cascade_pass_road/        # 9535 survivor, 9536 foreman (Task 5)
  schedules/cascade_pass_road/       # 1 foreman schedule (Task 8)
  items/                             # 40163 pelt, 40164 apex trophy (Task 7)
  rooms/kingsbarrow_vale/5441.yaml   # MODIFY: add secret east exit (Task 6)
```

## Room skeleton (authoritative — subagents build to these exact specs)

All rooms `zone: Cascade Pass Road`. Exits are reciprocal; coords are Cartesian-consistent (east = +x, north = +y, up = +z). Frontier/onward exits are omitted or framed as not-yet per notes.

### Stage 7.1a — the forest road (z0)

| roomid | title | coord (x,y,z) | biome | exits | beat |
|--------|-------|---------------|-------|-------|------|
| 6323 | The Hedge-Gap Trail | −17,61,0 | farmland | W→5441 (secret, zone Kingsbarrow Vale), E→6324 | Through the farm-gate — but the trail runs the wrong way for farm traffic, worn by something else. Fields falling away behind. First "this isn't a farm track" note. |
| 6324 | The Forest Edge | −16,61,0 | forest | W→6323, E→6325 | The timber begins — old trees, the light dropping. The vale's noise cuts off behind. |
| 6325 | The Climbing Road | −15,61,0 | forest | W→6324, E→6326 | Road narrowing and lifting. Trees larger the further east. Woodcutter sign (stumps) — but only on the WEST side of the path. |
| 6326 | The Lumber-Camp Track | −14,61,0 | forest | W→6325, E→6328, S→6327 | A junction — a churned track south into a working camp; the road continues east, fainter. |
| 6327 | The Lumber Camp | −14,60,0 | forest | N→6326 | Dead-end clearing. Sawpits, stacked timber, a lean-to. The crews cut west and haul west; nobody takes the eastern timber. Foreman + woodcutter NPCs. First oblique "east is wrong" in working-man's terms. |
| 6328 | The Deep Timber | −13,61,0 | forest | W→6326, E→6329 | Past the last cut. Trees enormous and old, no axe-marks. Predator sign. The quiet getting wrong. Base predator spawns here. |
| 6329 | The Ruined Waypoint | −12,61,0 | forest | W→6328, E→6330 | A travellers' rest abandoned years ago — collapsed roof, a cold hearth, a board with names scratched out. The road east of it barely a trace. |
| 6330 | The Reclaimed Road | −11,61,0 | forest | W→6329, E→6331 | The forest taking the road back — roots buckling the old stones. A game-trail crosses but avoids the eastern slope. Base predator spawns. |
| 6331 | Under the Old Trees | −10,61,0 | forest | W→6330, E→6332 | The oldest timber, a cathedral hush. Nothing sings here. The sense of being unwelcome at its strongest. Base predator spawns. |
| 6332 | The Foot of the Climb | −9,61,0 | forest | W→6331, U→6333 | The trees thinning ahead, rock breaking through. The road gives out; only a climb remains. End of the forest. |

### Stage 7.1b — the pass (z1–z3, climbing via vertical exits)

| roomid | title | coord (x,y,z) | biome | exits | beat |
|--------|-------|---------------|-------|-------|------|
| 6333 | The Tree Line | −9,61,1 | mountains | D→6332, U→6334 | The last stunted trees, then none. Wind. The air thinner and colder. Looking back — the whole settled country laid out small and warm below. |
| 6334 | The Exposed Switchback | −9,61,2 | cliffs | D→6333, U→6335 | Bare rock, a narrow ledge-path, a long drop. Handholds worn by old use. Highland fauna (a wheeling raptor that never lands). |
| 6335 | The High Shoulder | −9,61,3 | mountains | D→6334, E→6336 | The pass tops out. Views BOTH ways — the green country behind and below; ahead, broken grey highland running to a hard horizon. Something about the east reads as "old." |
| 6336 | The Crumbling Watchtower | −8,61,3 | mountains | W→6335, E→6337 | An old drystone tower, pre-anyone's-memory, half-fallen. Its one intact window-slot faces EAST, fixed on the highlands — never back toward the road. Who watched, and for what. |
| 6337 | The Wind-Gap | −7,61,3 | mountains | W→6336, E→6338, N→6339 | The saddle of the pass, wind funnelling through. A faint side-path north to a shelter. The main way runs east onto the plateau. Base highland predator spawns. |
| 6339 | The Survivor's Shelter | −7,62,3 | cliffs | S→6337 | Dead-end pocket — a rock overhang, a banked fire, a one-handed hermit who came back from the east and does not go again. THE warning centerpiece (dialogue). Tucked off the main line — found by exploring. |
| 6338 | The First Survey Marker | −6,61,3 | mountains | W→6337, E→6340 | Where Reth's old survey territory begins — a weathered stone survey marker, and beside it a small cairn scored with faint NESTED RINGS. Unexplained. Threshold-only. THE symbol beat (lookable noun). |
| 6340 | The Broken Ground | −5,61,3 | mountains | W→6338, E→6341 | Highland scrub, erosion gullies starting. Rough, deceptive footing. The pass APEX (tough mob) hunts this ground — the hardest fight on the road. |
| 6341 | The Plateau Edge | −4,61,3 | mountains | W→6340, E→6342 | The pass giving way to open highland — grey, scoured, immense. The land ahead is Reth's country. Cold coming off it. |
| 6342 | The Threshold of the Reach | −3,61,3 | mountains | W→6341 | The far edge. The broken highland runs east beyond crossing — no road, no shelter, and something underneath the silence the body reads before the mind does. Framed as not-yet-passable (NO east exit; the onward country is #21 Eastern Highlands). |

**Cartesian check:** verified — every listed exit has a reciprocal with the correct delta; the z-stack 6332→6333→6334→6335 uses vertical exits (no x/y change); the z3 shelf runs east x−9→x−3 at y61; 6339 sits at y62 (north of 6337). No collisions (all x ≥ −17 east of Kingsbarrow's x−18 spine is unclaimed).

## Mob roster (Task 4 — authoritative)

Mobs `zone: Cascade Pass Road`. Combat mobs `archetype: fighting`, `hostile: true`, carry loot with `itemdropchance` ≥ 60. Distinct mutation each.

| mobid | name | role | statpool | speciesid | mutation | spawn room(s) | notes |
|-------|------|------|----------|-----------|----------|---------------|-------|
| 9535 | The One Who Came Back | returned-survivor hermit (WARNING NPC) | — | 1 (human) | (thematic — e.g. `withered`/scar-flavored; pick one not reused) | 6339 | `non_combatant: true`, no loot, dialogue-only. Oblique dread. |
| 9536 | Camp-Foreman Bertt | lumber foreman (ambient) | — | 1 | distinct | 6327 | `non_combatant: true`, dialogue, day/night schedule (Task 8). "We don't cut east." |
| 9537 | A Weary Woodcutter | ambient flavor | — | 1 | distinct | 6327 (or 6325) | non-combatant, idle emotes, no dialogue. |
| 9538 | A Lone Hunter-Wolf | base predator | **275** | canine 2 | distinct | 6328, 6330 | loot 40163. |
| 9539 | A Tusked Forest-Boar | base predator | **275** | (boar — verify speciesid) | distinct | 6330, 6331 | loot 40163. |
| 9540 | A Highland Stalker-Cat | base predator | **275** | (feline — verify speciesid) | distinct | 6337, 6340 | loot 40163. |
| 9541 | The Pass-Apex | tough apex (hardest road fight) | **550** | (great predator) | distinct | 6340 | loot 40163 + 40164 (trophy); slower respawn. |
| 9542 | A Wheeling Highland-Raptor | highland fauna (flavor / light) | 275 (or lower) | bird 9 | distinct | 6334, 6341 | mostly flavor; light combat, minimal loot. |

**Spawn wiring:** after authoring mob YAMLs, add `spawninfo: [{mobid: N, respawnrate: "X real minutes"}]` blocks to the listed rooms (base predators ~10–15 min; apex ~25–30 min; NPCs on spawn at their room). Verify at boot (`mobs.LoadDataFiles() loadedCount`).

## Items (Task 7)

| itemid | name | type | notes |
|--------|------|------|-------|
| 40163 | a thick predator pelt | crafting material / sellable | loot from base predators + apex. Reuse an existing hide/pelt itemspec as the template; set `component_tag` consistent with existing pelts if any. |
| 40164 | a splintered pass-apex claw | trophy / sellable | apex-only trophy. Eerie flavor ("a claw that does not feel quite like bone") — but keep it a normal item mechanically; no mystery reveal. |

No quest items. The survivor and the marker are dialogue / lookable nouns.

---

## Task 1: Zone scaffolding + boot baseline

**Files:**
- Create: `_datafiles/world/dogmud/rooms/cascade_pass_road/zone-config.yaml`

- [ ] **Step 1: Create the feature branch**

Run: `git checkout -b feature/cascade-pass-road`
Expected: `Switched to a new branch 'feature/cascade-pass-road'`

- [ ] **Step 2: Establish a clean boot baseline BEFORE adding content**

Nuke instance saves (SOP) and boot the current tree to confirm it's clean before we add anything:

Run:
```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go build -o C:/tmp/dogmud-cpr.exe ./...
```
Then boot `C:/tmp/dogmud-cpr.exe` briefly and confirm `ValidateZoneConsistency errors=0` (mode=panic), room/mob/quest `loadedCount` lines, and 0 panics. Record the baseline mob count (expect ~561). Kill the server.

- [ ] **Step 3: Write `zone-config.yaml`**

```yaml
name: Cascade Pass Road
roomid: 6323
defaultbiome: forest
region: The Eastern Reach
```
(`roomid` = the zone's entry room. `region` groups it with the endgame arc — reuse "The Eastern Reach" for #21/#22 later.)

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/rooms/cascade_pass_road/zone-config.yaml
git commit -m "feat(cascade-pass): zone scaffolding (zone-config)"
```

## Task 2: Forest-road rooms 6323–6332 (stage 7.1a)

**Files:** Create `_datafiles/world/dogmud/rooms/cascade_pass_road/6323.yaml` … `6332.yaml`

Dispatch subagent(s) to author these 10 rooms to the skeleton table above. Two parallel batches (6323–6327, 6328–6332) are safe — exact roomids are given, so no ID scan/collision. Each subagent receives: the Conventions block, the skeleton rows for its rooms, the biome, exact exits (roomid + direction + any `secret`/`zone` annotation), the beat, and an existing forest/road room as a style exemplar (e.g. `east_road_to_greenford/6268.yaml` or a Kingsbarrow room). At this stage do NOT reference mobs/items not yet created; write rich `description`, `nouns`, and `idlemessages` only. Leave 6323's west exit `secret: true` with `zone: Kingsbarrow Vale` (the reciprocal side is added in Task 6).

- [ ] **Step 1: Author 6323–6327** (subagent batch A) — build to the skeleton; 6327 is the lumber camp (name its foreman/woodcutter as *described-but-mob-added-Task-4*; do not invent mob YAML here).
- [ ] **Step 2: Author 6328–6332** (subagent batch B).
- [ ] **Step 3: Validate YAML**

Run: `python -c "import yaml,glob; [yaml.safe_load(open(f,encoding='utf-8')) for f in glob.glob('_datafiles/world/dogmud/rooms/cascade_pass_road/*.yaml')]; print('OK')"`
Expected: `OK` (no exception).

- [ ] **Step 4: Boot-test the partial zone**

Point the smoketester at the zone entry: set `_datafiles/users/17.yaml` `roomid: 6323`, `zone: Cascade Pass Road`. Nuke instances, rebuild `C:/tmp/dogmud-cpr.exe`, boot, confirm `ValidateZoneConsistency errors=0` (mode=panic) and no panics. (6323's west secret exit will warn about a non-reciprocal until Task 6 — acceptable mid-build, but note it; if `MapConsistencyEnforce` is `panic`, temporarily leave 6323 west omitted and add it in Task 6, OR add the 5441 side now. Prefer: add the 5441 reciprocal in Task 6 and keep 6323 west present — secret exits are still collision/reciprocity-checked, so if boot panics, do Task 6 before this boot.)

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/rooms/cascade_pass_road/6323.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6324.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6325.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6326.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6327.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6328.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6329.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6330.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6331.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6332.yaml
git commit -m "feat(cascade-pass): forest-road rooms 6323-6332 (stage 7.1a)"
```

## Task 3: Pass rooms 6333–6342 (stage 7.1b)

**Files:** Create `6333.yaml` … `6342.yaml`

Same method. Batches: 6333–6337 and 6338–6342. Exemplar: a `cliffs`/`mountains` room from `ironwind_steppe` (biome cliffs/mountains) for tone. Key per-room requirements:
- 6335 must describe views BOTH directions (settled country behind, broken highland ahead).
- 6336 (watchtower): the window-slot faces EAST only.
- 6338: include the survey-marker + nested-rings cairn nouns (the symbol beat) — UNEXPLAINED, threshold-only, no numerology, no decode. Exactly one symbol surface in the whole zone (guard against bleed elsewhere).
- 6339: the survivor's shelter (describe the overhang/fire; the NPC is added Task 4/5).
- 6342: NO east exit; frame the onward highland as not-yet-crossable (the dread beat), NOT a walkable stub.

- [ ] **Step 1: Author 6333–6337** (subagent batch A).
- [ ] **Step 2: Author 6338–6342** (subagent batch B).
- [ ] **Step 3: Validate YAML** — same command as Task 2 Step 3.
- [ ] **Step 4: Boot-test** — nuke instances, rebuild, boot, `ValidateZoneConsistency errors=0` (mode=panic), 0 panics. Walk 6323→…→6342 (or verify via consistency pass).
- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/rooms/cascade_pass_road/6333.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6334.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6335.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6336.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6337.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6338.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6339.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6340.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6341.yaml _datafiles/world/dogmud/rooms/cascade_pass_road/6342.yaml
git commit -m "feat(cascade-pass): mountain-pass rooms 6333-6342 (stage 7.1b)"
```

## Task 4: Mobs 9535–9542 + spawn wiring

**Files:** Create `_datafiles/world/dogmud/mobs/cascade_pass_road/<mobid>-<name>.yaml` (8 files); Modify the room files listed in the roster for `spawninfo`.

Author to the roster table. Use `instance_planar_oasis/318-sand_elemental.yaml` (base, statpool multiplier form) and a corridor predator (e.g. `new_plymouth_outskirts` combat mob) as exemplars — but note Cascade Pass mobs set `statpool` to the DIRECT final value (275 / 550), NOT a multiplier, because this is an overworld zone (no instance gold-scaling). Verify each animal `speciesid` against `ironwind_steppe` mobs before writing (boar/feline/raptor). Confirm each mutation is unique across all 8.

- [ ] **Step 1: Author the two NPCs** (9535 survivor `non_combatant: true`; 9536 foreman `non_combatant: true`; 9537 woodcutter non-combatant flavor). Names in `Title()` form; filenames match `ConvertForFilename`.
- [ ] **Step 2: Author the five fauna** (9538–9542) with `statpool` per the table, `archetype: fighting`, `hostile: true` (9542 may be lower/lighter), `loot_pool` + `itemdropchance` referencing items 40163/40164 (created Task 7 — if authoring order matters, create items first or accept a load-time note; loot_pool referencing a not-yet-existing item is tolerated at load but verify after Task 7).
- [ ] **Step 3: Wire `spawninfo`** into the roster rooms (add the block to each listed room YAML).
- [ ] **Step 4: Validate YAML** (mobs + touched rooms) and boot: nuke instances, rebuild, boot, confirm `mobs.LoadDataFiles() loadedCount` rose by 8 (~569), no `casing`/`Filepath()` panics, `ValidateZoneConsistency errors=0`.
- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/mobs/cascade_pass_road/ _datafiles/world/dogmud/rooms/cascade_pass_road/
git commit -m "feat(cascade-pass): 8 mobs (275/550 statpool) + spawn wiring"
```

## Task 5: Dialogue — the survivor (9535) + foreman (9536)

**Files:** Create `_datafiles/world/dogmud/dialogue/cascade_pass_road/9535.yaml`, `9536.yaml`

- **9535 The One Who Came Back** — the warning centerpiece. Oblique dread, NEVER names the ship or "trap." Triggers a player would try (`east`, `warning`, `danger`, `hand`, `back`, `what happened`) route to lore nodes describing: the ground opening, walls that move, a place that "doesn't want to be opened," the cost he paid. He does not advise, does not quest, does not reward — a person, not a mechanism. Long `text:` → `|` literal blocks. `hints:` in player-perspective ("You could ask what happened to his hand"). No semicolons.
- **9536 Camp-Foreman Bertt** — working-man's "we don't cut east" — the timber's good east but the crews won't take it, and he won't be drawn on why beyond unease. Triggers: `east`, `timber`, `work`, `camp`. Lighter than 9535.

Both: gated/lore nodes ordered FIRST; every trigger word discoverable in a hint or the mob/room text.

- [ ] **Step 1: Author 9535 dialogue** (subagent).
- [ ] **Step 2: Author 9536 dialogue** (subagent).
- [ ] **Step 3: Validate YAML** + boot; in-game (or via harness) `talk`/`ask` a few triggers on each; confirm no node-shadowing (short trigger substring-matching a topic) and no truncation.
- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/dialogue/cascade_pass_road/9535.yaml _datafiles/world/dogmud/dialogue/cascade_pass_road/9536.yaml
git commit -m "feat(cascade-pass): survivor + foreman dialogue (oblique ship foreshadowing)"
```

## Task 6: The hidden branch — modify Kingsbarrow Vale 5441

**Files:** Modify `_datafiles/world/dogmud/rooms/kingsbarrow_vale/5441.yaml`

Add the secret east exit + a searchable hint. The existing `roadside hedge` noun already mentions the open farm-gate; extend the reveal path.

- [ ] **Step 1: Add the secret east exit** — under `exits:` add:

```yaml
  east:
    roomid: 6323
    secret: true
    zone: Cascade Pass Road
```
(Keep existing `south: 5440` and `north: 5442`.)

- [ ] **Step 2: Add a `hidden_nouns` entry** that a `search` reveals, hinting the eastward trail without a giveaway — e.g. key `cart-ruts` or `game-trail`: a set of ruts/prints running EAST past the farm-gate, worn deeper than farm traffic explains, `hidden_description` appending a "there is a way east here" line to the room. Mark `instance: "skip"` per the hidden-noun schema. Keep the mundane surface reading intact on plain `look`.

- [ ] **Step 3: Validate YAML** + boot. Confirm: on `look` at 5441 the east exit is NOT listed; `search` (with sufficient Perception) reveals "You found a secret exit: east"; `east` then traverses to 6323; `ValidateZoneConsistency errors=0` (the 6323↔5441 reciprocity now resolves cleanly, both annotated cross-zone).

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/rooms/kingsbarrow_vale/5441.yaml
git commit -m "feat(cascade-pass): hidden secret east branch off Kingsbarrow Vale 5441"
```

## Task 7: Items 40163–40164 (predator loot)

**Files:** Create the two item YAMLs under `_datafiles/world/dogmud/items/` (match the loader's expected path/filename via `Filepath()`/`ConvertForFilename` — mirror an existing pelt/trophy item's location).

- [ ] **Step 1: Author 40163 (predator pelt)** — reuse an existing hide/pelt itemspec as template; sellable; `component_tag` consistent with existing pelts if the game has a hide tag.
- [ ] **Step 2: Author 40164 (pass-apex claw trophy)** — sellable trophy; eerie flavor text, mechanically normal; no mystery reveal.
- [ ] **Step 3: Validate YAML** + boot; confirm `items` loadedCount rose by 2 and the mob `loot_pool` references now resolve (no "unknown item" warnings). Optionally kill a base predator in-game and confirm a pelt drops.
- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/items/
git commit -m "feat(cascade-pass): predator pelt + apex trophy loot items"
```

## Task 8: Foreman schedule

**Files:** Create `_datafiles/world/dogmud/schedules/cascade_pass_road/<id>.yaml`; add `schedule_id:` to 9536.

- [ ] **Step 1: Author a day/night schedule** for Bertt (9536) — daytime `activity` at the camp working, night `activity: sleeping` in the lean-to; full 24h coverage (validators panic on gaps). Reference `docs/schemas/schedule.md` and an existing anchor schedule (e.g. a Greenford or corridor schedule) as exemplar. Target rooms limited to 6327 (single-room camp) — keep it simple; no cross-room path needed.
- [ ] **Step 2: Add `schedule_id:` to 9536's mob YAML.**
- [ ] **Step 3: Validate + boot** — schedule validators pass (no coverage gap / unreachable target / unresolved id), `ValidateZoneConsistency errors=0`, 0 panics.
- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/schedules/cascade_pass_road/ _datafiles/world/dogmud/mobs/cascade_pass_road/9536-camp_foreman_bertt.yaml
git commit -m "feat(cascade-pass): foreman day/night schedule"
```

## Task 9: World-critic integration pass + feel-test + merge

- [ ] **Step 1: World-critic pass** — dispatch an adversarial reviewer subagent over the whole zone (all 20 rooms + mobs + dialogue + the 5441 edit) checking for: cross-file contradictions (distances/directions, a referenced thing that doesn't exist), dialogue node-shadowing (short triggers substring-matching topics), long double-quoted NPC text that should be `|` blocks, semicolons in dialogue, dead hint keywords (every hinted word must route to its node or noun), mutation duplicates across the roster, symbol bleed beyond 6338, `biome` validity, hyphen-wrap render artifacts, and terminus-stub wording at 6342. Fix all findings; re-boot.

- [ ] **Step 2: Full boot verification**

Run: nuke instances, rebuild, boot `C:/tmp/dogmud-cpr.exe`. Confirm ALL: `ValidateZoneConsistency errors=0 warnings=0 mode=panic`; `ValidateAllFlags OK`; room/mob/item/quest `loadedCount` all clean (rooms +20 ≈ 1285, mobs +8 ≈ 569); 0 panics.

- [ ] **Step 3: Feel-test via mudagent harness** — point smoketester (17.yaml) at 5441; run `/playtest local feel-tester` (or a bug-finder) to: find the hidden branch via `search`, traverse 6323→6342, fight a base predator and gauge the solo-endgame difficulty (report fight length/lethality for tuning), talk to the survivor + foreman (verify the oblique foreshadowing lands and no triggers misroute), `look` the survey marker (symbol reads threshold-only), and hit the 6342 threshold (reads as dread, not a broken wall). Capture the report under `tools/playtest/reports/`. Fix any findings (esp. difficulty tuning — adjust `statpool`/`itemdropchance` if fights are HP-sponges or trivial).

- [ ] **Step 4: Merge to master**

```bash
git checkout master
git merge --no-ff feature/cascade-pass-road -m "Merge: Cascade Pass Road (#20) — hidden solo-endgame road east of NP (Eastern Arc endgame leg 1)"
```

- [ ] **Step 5: Update ZONE_EXPANSION.md status + memory** — mark #20 built in the `docs/ZONE_EXPANSION.md` build-priority table; append a build note to the zone-expansion memory (`project_zone_expansion_redesign.md`) with roomids/mobids/gotchas; update MEMORY.md top status. Commit.

```bash
git add docs/ZONE_EXPANSION.md
git commit -m "docs(zone-expansion): Cascade Pass Road (#20) BUILT — Eastern Arc endgame leg 1"
```

---

## Self-review notes

- **Spec coverage:** hidden branch (T6) ✓; forest→pass 20-room layout (T2,T3) ✓; solo-endgame 275/550 combat (T4) ✓; returned-survivor + environmental foreshadowing (T3 rooms, T5 dialogue) ✓; one understated symbol beat @6338 (T3) ✓; light loot, no new forageables (T7) ✓; terminus stub not-yet framing (T3 6342) ✓; no quest ✓; folder/naming/dialogue gotchas (Conventions) ✓.
- **Deferred by decision:** forest/mountain forageables (to the pinnacle-craft-items work).
- **Ordering note:** Task 4 mob `loot_pool` references items created in Task 7; load tolerates forward refs but the T7 boot is where loot resolution is confirmed. If a strict loader complains, do Task 7 before Task 4.
- **Type consistency:** roomids 6323–6342, mobids 9535–9542, itemids 40163–40164, folder `cascade_pass_road`, zone display "Cascade Pass Road", region "The Eastern Reach" — used consistently across all tasks.
