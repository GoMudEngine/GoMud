# Eastern Highlands (#21) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build Eastern Highlands — a 30-room endgame zone east of Cascade Pass where the buried ship's hull first appears and the road ends at the locked disc-door, featuring data-driven degraded-defense hazards and a BIS-dropping Sentinel boss.

**Architecture:** A 30-room overworld zone (folder `eastern_highlands`, roomids 6343–6372) authored as YAML, plus data-driven hazards built from EXISTING engine machinery: a new "energy-discharge" DoT buff, a room mutator (`hull_discharge`) that applies it every round via `Room.RoundTick`, and defusable trapped exits (`lock.trapbuffids`). One boss (the Sentinel, ~statpool 1200) drops 2 BIS-competitive items in currently-unserved slots (feet + ranged) plus a trophy and a ~3% ultra-rare craft material. No engine code.

**Tech Stack:** GoMud YAML (rooms/mobs/items/buffs/mutators); `python -c "import yaml"` validation; local boot for load-validation; `/playtest` harness for the geared feel-test.

**Reference spec:** `docs/superpowers/specs/2026-07-01-eastern-highlands-design.md`

---

## Conventions (bake into EVERY subagent prompt)

- Zone folder MUST = `eastern_highlands`. Rooms → `_datafiles/world/dogmud/rooms/eastern_highlands/<id>.yaml`; mobs → `mobs/eastern_highlands/<id>-<convertforfilename(name)>.yaml`; mutators → `mutators/<id>.yaml`; buffs → `buffs/<id>-<convertforfilename(name)>.yaml`.
- Prose: NO colon-space `": "` in a prose value (YAML panic — use em-dash `—`). NO semicolons in any NPC/flavor text. `|` literal blocks for long text. Don't split hyphenated compounds across folded-scalar line breaks. Hard-wrap ~78 cols. NO hard numbers in player-visible text.
- **LORE BOUNDARY (critical):** the hull may be perceived as artificial / made / metallic / seamless / vast / buried / ancient / cursed / WRONG. NEVER: ship / vessel / craft / from the sky / fell / crashed / moons / stars / Earth / another world / machine / technology / builders — and never the revelation. Awe + dread, never the category or the answer. No NPC decodes anything.
- Mob names in `internal/casing.Title()` form; filename = `mobid-ConvertForFilename(name)`. Combat mobs pair `behavior_archetype` + `aiprofile` (valid aiprofiles: skirmisher/predator/ambush_predator/serpent/aggressive/brute/brawler/caster/tactical). `statpool` applied DIRECTLY (overworld, no scaling). Distinct Chrysalis mutation per mob (cross-check the whole roster). Animal speciesids: canine 2, bear 3, boar 6, deer 7, serpent 8, raptor 9, feline 11.
- Biomes: `mountains`/`cliffs`/`land` only.
- Git: explicit pathspecs, never `git add -A`. Branch `feature/eastern-highlands`.
- Emote idlecommands/combatcommands end with a period.

## File structure

```
_datafiles/world/dogmud/
  rooms/eastern_highlands/            # 30 rooms 6343–6372 (T2,T3,T4) + zone-config (T1)
  mobs/eastern_highlands/             # fauna 9543–9549 + adds 9550–9551 + Sentinel 9552 (T7,T8)
  buffs/94-*.yaml, 95-*.yaml          # discharge DoT + seared debuff (T5)
  mutators/hull_discharge.yaml        # hazard mutator (T5)
  items/armor-20000/feet/<id>-*.yaml  # BIS boots (T9)
  items/weapons-10000/<id>-*.yaml     # BIS ranged weapon (T9)
  items/materials-40000/40165-*, 40166-*  # trophy + ultra-rare material (T9)
  rooms/cascade_pass_road/6342.yaml   # MODIFY: open east → 6343 (T6)
```

## Room skeleton (authoritative — exact coords/exits, Cartesian-verified)

All rooms `zone: Eastern Highlands`. east=+x, north=+y, up=+z. **HAZARD** = gets the `hull_discharge` mutator (T5). **TRAP** = the exit noted carries a defusable `lock.trapbuffids` (T5).

### Stage 7.2a — the approach (6343–6352, z3)
| id | title | coord | biome | exits | beat |
|----|-------|-------|-------|-------|------|
| 6343 | The Edge of the Reach | −2,61,3 | mountains | W→6342 (zone Cascade Pass Road), E→6344 | Crossing out of the pass into the true highland. Grey, scoured, immense. |
| 6344 | The Broken Plateau | −1,61,3 | mountains | W→6343, E→6345 | Deceptive footing, basalt breaking through. The wrongness sits under everything. |
| 6345 | The Gully Head | 0,61,3 | mountains | W→6344, E→6347, S→6346 | An erosion gully cuts south; the road holds east along the rim. |
| 6346 | The Dry Gully | 0,60,3 | cliffs | N→6345 | Dead-end scree gully — a predator den. Bones. Bad ground. |
| 6347 | Reth's First Marker | 1,61,3 | mountains | W→6345, E→6348 | A weathered survey marker scored with faint nested rings — Reth's territory begins. UNEXPLAINED, threshold-only. Symbol beat. |
| 6348 | The Basalt Colonnade | 2,61,3 | mountains | W→6347, S→6349 | Columns of black rock like a made thing but not — the land teasing the eye. |
| 6349 | The Lightning-Split Cairn | 2,60,3 | mountains | N→6348, E→6350 | Reth's landmark (named in his Q75 testimony) — a cairn cloven top to base by an old strike. Payoff room for Q75 players. |
| 6350 | The Scoured Flat | 3,60,3 | mountains | W→6349, S→6351 | Wind-stripped rock, nothing growing. The silence deepening. |
| 6351 | The Abandoned Survey Camp | 3,59,3 | mountains | N→6350, E→6352 | Reth's old camp — collapsed, weathered, long empty. Environmental lore. |
| 6352 | Where the Green Fails | 4,59,3 | mountains | W→6351, E→6353 | The scrub thinning to nothing on a line that feels deliberate. Transition to the hull. |

### Stage 7.2b — the hull (6353–6362, z3)
| id | title | coord | biome | exits | beat |
|----|-------|-------|-------|-------|------|
| 6353 | The Vegetation Line | 5,59,3 | mountains | W→6352, E→6354 | Where the growth stops — too clean, too precise. Nothing crosses it. |
| 6354 | The First Surface | 6,59,3 | cliffs | W→6353, E→6355 | Exposed surface — smooth, metallic, no grain, no seam. Undeniably made. The first true dread. |
| 6355 | Along the Flank | 7,59,3 | cliffs | W→6354, E→6356 | **HAZARD.** Walking along the buried made-thing — the air over it crawls and stings. |
| 6356 | The Curve | 8,59,3 | cliffs | W→6355, S→6357 | The surface curving away south — the scale beginning to register as impossible. |
| 6357 | The Long Flank | 8,58,3 | cliffs | N→6356, S→6358 | **HAZARD.** Room after room of the same made surface — it does not end. |
| 6358 | The Impossible Length | 8,57,3 | cliffs | N→6357, S→6359 | The mind refusing the size of it. Buried, vast, ancient, wrong. |
| 6359 | Where the Light Catches | 8,56,3 | cliffs | N→6358, E→6360 | The surface takes the light wrong at a certain angle — it is not landscape. |
| 6360 | The First Recess | 9,56,3 | cliffs | W→6359, S→6361 (TRAP) | **HAZARD.** A recess in the surface — the exit south is warded (defusable trap). |
| 6361 | The Sunken Course | 9,55,3 | cliffs | N→6360, E→6362 | Following the hull as it sinks toward the ground. |
| 6362 | The Descending Curve | 10,55,3 | cliffs | W→6361, E→6363 | The made-thing curving down into the earth ahead — the entrance is near. |

### Stage 7.2c — the entrance (6363–6372, z3)
| id | title | coord | biome | exits | beat |
|----|-------|-------|-------|-------|------|
| 6363 | Where the Hull Enters the Ground | 11,55,3 | cliffs | W→6362, E→6364 | The southeastern curve where the made-thing goes into the earth. |
| 6364 | The Approach Hollow | 12,55,3 | cliffs | W→6363, S→6365 | A worn hollow before the buried face — the ground disturbed by old work. |
| 6365 | The Cleared Section | 12,54,3 | cliffs | N→6364, E→6366 | **Maren's father was here** — his weathered camp, tools, the labor of one man who got this far and stopped. Environmental lore (no NPC, no quest). |
| 6366 | The Warded Recesses | 13,54,3 | cliffs | W→6365, E→6367 | **HAZARD.** Recesses in the buried face, some still watchful. |
| 6367 | The Fork Before the Door | 14,54,3 | cliffs | W→6366, E→6368, S→6369 | The way splits — east toward the disc-face, south toward a sealed side-recess (the Sentinel's vault). |
| 6368 | The Ward-Field | 15,54,3 | cliffs | W→6367, E→6370 | **HAZARD.** The final stretch before the door, the air alive and biting. |
| 6369 | The Sealed Recess | 14,53,3 | cliffs | N→6367, E→6371 | **HAZARD.** The outer chamber of a sealed vault — something still guards it. |
| 6370 | The Disc-Face Approach | 16,54,3 | cliffs | W→6368, E→6372 (TRAP: the door-ward) | The buried face bearing the door. The final exit east is warded (defusable) — the last defense of the threshold. |
| 6371 | The Sentinel's Watch | 15,53,3 | cliffs | W→6369 | **THE SENTINEL** (boss + adds) guards this sealed vault — the BIS loot. Dead-end. Optional; NOT on the path to the door. |
| 6372 | The Disc-Door | 17,54,3 | cliffs | W→6370 | **THE LOCKED THRESHOLD.** The disc-shaped depression, the symbol pressed deep, recesses that do not yield. Examine → learn you need the disc. Does NOT open. Terminus toward #22 (no east exit). |

**Cartesian check:** all deltas verified reciprocal; door path is 6367→6368→6370→6372; the Sentinel vault (6369→6371) is a separate south branch, so reaching the door does NOT require the Sentinel. No coord collisions (all distinct).

---

## Task 1: Zone scaffolding + ID lock + boot baseline

**Files:** Create `rooms/eastern_highlands/zone-config.yaml`

- [ ] **Step 1: Branch + baseline.** `git checkout -b feature/eastern-highlands`. Nuke instance saves (`rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`), `go build -o C:/tmp/dogmud-eh.exe .`, boot briefly, confirm clean (mobs 569, ValidateZoneConsistency errors=0 mode=panic, 0 panics). Kill server.
- [ ] **Step 2: Lock exact IDs.** Run `python tools/id_inventory.py` and record the next-free for: buffs (expect 94), and next-free item ids in the armor-20000 (feet), weapons-10000, and materials-40000 ranges. Fill these into Tasks 5 & 9 before authoring. (Plan assumes buffs 94/95, materials 40165/40166; confirm feet-armor and weapon next-free.)
- [ ] **Step 3: zone-config.yaml**
```yaml
name: Eastern Highlands
roomid: 6343
defaultbiome: mountains
region: The Eastern Reach
```
- [ ] **Step 4: Commit** `git add rooms/eastern_highlands/zone-config.yaml && git commit -m "feat(eastern-highlands): zone scaffolding"`

## Task 2: Approach rooms 6343–6352 (7.2a)

**Files:** Create `rooms/eastern_highlands/6343.yaml … 6352.yaml`

Dispatch two parallel subagents (6343–6347, 6348–6352) with the 7.2a skeleton rows, the Conventions + LORE BOUNDARY block, and exemplars to read: `rooms/cascade_pass_road/6342.yaml` (the room 6343 continues from — match zone voice), `rooms/east_road_to_greenford/6266.yaml` (the Old Waystone — symbol restraint for 6347), and an `ironwind_steppe` cliffs/mountains room. Requirements: 6347 = the survey-marker symbol beat (nested rings, unexplained, threshold-only). 6349 = the lightning-split cairn (Reth's Q75 landmark). 6351 = Reth's abandoned camp. 6343 west exit carries `zone: Cascade Pass Road`. Do NOT reference mobs/items/spawns/hazards yet (added later). Rich description + nouns + idlemessages.

- [ ] **Step 1:** Author 6343–6347 (subagent A).
- [ ] **Step 2:** Author 6348–6352 (subagent B).
- [ ] **Step 3: Validate** `python -c "import yaml,glob; [yaml.safe_load(open(f,encoding='utf-8')) for f in glob.glob('_datafiles/world/dogmud/rooms/eastern_highlands/634*.yaml')+glob.glob('_datafiles/world/dogmud/rooms/eastern_highlands/635[0-2].yaml')]; print('OK')"`
- [ ] **Step 4: Spec-check** (main loop): confirm each room's coord/title/biome/exits match the skeleton and intra-batch reciprocity holds (reuse the Cascade Pass python spec-check pattern).
- [ ] **Step 5: Commit** (explicit pathspecs for 6343–6352).

## Task 3: Hull rooms 6353–6362 (7.2b)

**Files:** Create `6353.yaml … 6362.yaml`

Two subagents (6353–6357, 6358–6362). Exemplar: an `ironwind_steppe` cliffs room + the 7.2a rooms for voice. This is the HULL REVEAL — the most lore-delicate stage. Requirements: 6353 = the vegetation line (too clean). 6354 = first exposed surface (smooth, metallic, no grain, no seam — undeniably made, the first true dread). 6355–6359 = walking ALONG the made-thing, its impossible scale. RESPECT THE LORE BOUNDARY (artificial + wrong, never ship/sky/moons/answer). Mark in each file's prose which rooms are HAZARD (6355, 6357, 6360) so the mutator wiring (T5) is obvious — but do NOT add the `mutators:` field here (T5 does it). 6360 south exit → 6361 will get a trap in T5. Do NOT add mobs/spawns.

- [ ] **Step 1:** Author 6353–6357 (subagent A). **Step 2:** Author 6358–6362 (subagent B). **Step 3:** Validate YAML. **Step 4:** Spec-check coords/exits/reciprocity + a prose read of 6354 (the reveal) confirming lore-boundary restraint. **Step 5:** Commit.

## Task 4: Entrance rooms 6363–6372 (7.2c)

**Files:** Create `6363.yaml … 6372.yaml`

Two subagents (6363–6367, 6368–6372). Requirements: 6365 = Maren's father's cleared section (environmental only — his camp/tools/the work of one man; ties to Ashwick/Maren lore; NO NPC, NO quest). 6371 = The Sentinel's Watch (describe a sealed vault + a great dormant socketed watcher/eye — non-techy; the Sentinel MOB is added T8, do NOT author a mob file). 6372 = the disc-door (the disc-shaped depression + the symbol pressed deep; examinable; the player learns they need the disc; NO east exit; framed as the awe/dread endgame threshold, NOT a "come back later" gate). Mark HAZARD rooms (6366, 6368, 6369) and the TRAP exits (6360→6361 already; 6370→6372 the door-ward) in prose for T5. Do NOT add mutators/traps/mobs here.

- [ ] **Step 1:** Author 6363–6367. **Step 2:** Author 6368–6372. **Step 3:** Validate. **Step 4:** Spec-check + prose read of 6372 (door terminus, no east exit) and 6365 (Maren's father, lore-safe). **Step 5:** Commit.

## Task 5: The hazard system (buffs + mutator + wiring + traps)

**Files:** Create `buffs/94-energy_discharge.yaml`, `buffs/95-seared.yaml`, `mutators/hull_discharge.yaml`; Modify hazard rooms + trap exits.

- [ ] **Step 1: The discharge DoT buff** (id 94; confirm free in T1). Model on buff 39 (Venom):
```yaml
buffid: 94
name: Energy Discharge
description: Something under the ground bites up through you in cold arcs.
triggerrate: 1 round
triggercount: 3
start_user_text: The air over the buried surface crawls, and a cold shock jolts up through your boots.
start_room_text: "{source} jerks as the ground arcs against them."
end_user_text: The stinging in your bones fades — for now.
trigger_user_text: A cold arc bites up through you.
trigger_room_text: "{source} shudders as the ground arcs against them."
tick_pool: health
tick_percent: -0.06
tick_variance: 0.03
tick_min: 4
```
(Non-techy language. `triggercount: 3` = it fades a few rounds after you leave the room; while you stay, the mutator re-applies it each round — persistent presence hazard. Tune `tick_percent`/`tick_min` in the feel-test against the geared char.)

- [ ] **Step 2: The `seared` debuff** (id 95) — a short stat penalty applied by the harsher trap (optional flavor): a brief `perception`/`dexterity` statmod penalty, `triggercount: 5`, no tick. Model statmod-only on buff 26 (Conviction Surge) but with negative statmods. (If YAGNI at build time, skip 95 and use only 94 — note the decision.)

- [ ] **Step 3: The hazard mutator** `mutators/hull_discharge.yaml` (model on `mutators/wildfire.yaml`, but PERMANENT — no `decayrate`):
```yaml
mutatorid: hull_discharge
alertmodifier:
  text: 'The air over the buried surface crawls and stings.'
playerbuffids: [94]
```
(No `decayrate` → does not decay. Verify at boot/feel-test that it re-applies each round and does NOT despawn. If a respawn/enabled flag is needed for persistence, add `enabled: true`.)

- [ ] **Step 4: Wire the mutator into hazard rooms.** Add to each of 6355, 6357, 6360, 6366, 6368, 6369:
```yaml
mutators:
- mutatorid: hull_discharge
```
- [ ] **Step 5: Trapped exits (defusable).** On the exit 6360 `south`→6361, and on 6370 `east`→6372 (the door-ward), add a defusable trap (model on `rooms/thornwall_city/487.yaml`):
```yaml
  south:
    roomid: 6361
    lock:
      difficulty: 40
      trapbuffids: [94]
```
(The door-ward 6370→6372 uses the same shape with a higher `difficulty` — it is the last defense. `defuse` disarms it; failing/forcing fires buff 94. This is the solo skill path to the door. Confirm `defuse` targets the exit — see `internal/actions/defuse.go`.)
- [ ] **Step 6: Validate + boot.** Confirm buffs loadedCount +1/+2, no mutator/buff load errors, `ValidateZoneConsistency errors=0`. **Step 7: Commit.**

## Task 6: Open Cascade Pass 6342 east → 6343

**Files:** Modify `rooms/cascade_pass_road/6342.yaml`

- [ ] **Step 1:** Add the east exit + reword the terminus prose so the east is now the way on (it was "not passable" — now the highland is crossable ahead):
```yaml
  east:
    roomid: 6343
    zone: Eastern Highlands
```
Update 6342's description + `eastern ground` noun so it no longer reads as an impassable wall (the road east now exists). Keep the dread; drop the "not passable" finality.
- [ ] **Step 2: Validate + boot** — 6342↔6343 reciprocity clean, `ValidateZoneConsistency errors=0 mode=panic`. **Step 3: Commit.**

## Task 7: Fauna mobs 9543–9549 + spawn wiring

**Files:** Create `mobs/eastern_highlands/9543-…yaml … 9549-…yaml`; wire `spawninfo` into rooms.

Hostile highland fauna ramping ABOVE Cascade Pass (which tested LOW). Author to this roster (distinct mutations, `archetype: fighting`, `behavior_archetype`+`aiprofile`, loot). Exemplar: the Cascade Pass mobs + `ironwind_steppe` predators.

| id | name | tier | statpool | speciesid | behavior/aiprofile | spawn rooms | loot |
|----|------|------|----------|-----------|--------------------|-------------|------|
| 9543 | A Gully-Denned Wolf | base | 350 | 2 | predator/predator | 6346, 6350 | pelt (Cascade 40163) |
| 9544 | A Highland Ridge-Cat | base | 350 | 11 | ambusher/ambush_predator | 6348, 6359 | 40163 |
| 9545 | A Scoured-Plain Boar | base | 400 | 6 | generic_fighter/brute | 6344, 6352 | 40163 |
| 9546 | A Basalt-Crag Raptor | base | 300 | 9 | generic_fighter/ambush_predator | 6356, 6361 | none |
| 9547 | The Reach-Alpha | tough | 700 | 3 | leader/predator | 6353, 6363 | 40163 + (T9 material low chance) |
| 9548 | A Carrion Highland-Hound | tough | 650 | 2 | predator/predator | 6357, 6366 | 40163 |
| 9549 | A Cliff-Stalker | tough | 650 | 11 | ambusher/ambush_predator | 6360, 6368 | 40163 |

- [ ] **Step 1:** Author the 7 fauna. **Step 2:** Wire `spawninfo` (base ~12 min, tough ~20 min). **Step 3:** Validate + boot (mobs +7). **Step 4:** Commit.

## Task 8: The Sentinel boss + adds (9550–9552)

**Files:** Create `mobs/eastern_highlands/9550-…, 9551-…, 9552-…`; wire spawns into the vault (6369/6371).

The Sentinel is the BIS gate — calibrated head-to-head with a ~300g Oasis royal encounter (boss ~1200 + adds ~300). Described NON-TECHY (a great socketed watcher, a cold unblinking eye, a ward humming awake) — respect the lore boundary.

| id | name | role | statpool | speciesid | notes |
|----|------|------|----------|-----------|-------|
| 9550 | A Defense-Fragment | add | 300 | 20 (orb) | converging add; spawns in 6371 (and 6369); `hostile: true` |
| 9551 | A Warding Node | add | 300 | 20 (orb) | second add; 6371 | 
| 9552 | The Sentinel | BOSS | 1200 | 20 (orb) | `behavior_archetype: leader`, `aiprofile: brute`/`tactical`, `submission_policy: lethal`, `surrender_policy: never`; spawns in 6371; loot = the BIS pieces + trophy + rare material (T9); slow respawn (~45 min) |

(speciesid 20 = orb — a fitting non-techy "socketed watcher" species; confirm it exists, else use a suitable construct/aberration species. Statpools are STARTING values for the calibration pass; the set-piece total threat = Sentinel 1200 + 2 adds 300 = comparable to a royal + ellies.)

- [ ] **Step 1:** Author 9550/9551 (adds), 9552 (Sentinel) — leave `loot_pool` referencing the T9 item ids (author T9 first, or accept forward refs and confirm at T9 boot). **Step 2:** Wire spawninfo (adds + Sentinel in 6371; optionally an add in 6369). **Step 3:** Validate + boot (mobs +3, no casing/statpool errors). **Step 4:** Commit.

## Task 9: BIS loot + trophy + ultra-rare material

**Files:** Create the boots (armor-20000/feet), the ranged weapon (weapons-10000), `materials-40000/40165-*` (trophy), `materials-40000/40166-*` (material); wire into 9552 `loot_pool`.

**Calibration (build task):** read the instance affix-scaling to compute the ~300g-equivalent stat target for Oasis gear. Start here: examine `internal/rooms/instances.go ScaleSpawnStatPools` context + how affix bonuses are rolled onto instance loot (search `affix` in `internal/items/`), and inspect the Oasis templates (Volcanic Plate 20072, Earthshaker Warhammer 10027). Author the fixed items to match a 300g buy-in's power. **Fallback if affix scaling is opaque:** set stats to ~2.5–3× the best existing same-slot item, then tune in the feel-test.

- [ ] **Step 1: BIS boots** (feet). Model on `items/armor-20000/feet/20052-sturdy_leather_boots.yaml`. Starting spec (tune to 300g-Oasis parity): `type: feet`, high `physical_mitigation` + `magical_mitigation`, statmods (e.g. dexterity/vitality), and a small thematic **`conviction_mitigation`/hazard-flavored** edge; `rarity_tier: 82`; distinctive non-techy name (e.g. "The Surveyor's Last Boots" — recovered gear, ambiguous). `value` high.
- [ ] **Step 2: BIS ranged weapon** (`subtype: shooting`). Model on `items/weapons-10000/10042-arbalest.yaml`. The FIRST BIS ranged: high `damage_multiplier` + `basedamage`, appropriate `ammo_tag`, `min_strength`, `rarity_tier: 82`. **Framed as mundane masterwork** (dark horn/steel, ancient make) — NOT an energy weapon.
- [ ] **Step 3: Trophy** `40165` — the Sentinel's "core" (still-warm heartstone / oracle-shard), high `value` sellable + lore hook, non-techy description. **Step 4: Ultra-rare material** `40166` — a new craft material with a `component_tag` (e.g. `hull-relic`), `is_component: true`, high `value`, `rarity_tier: 85`; description marks it as something no craft yet knows how to use (a seed for future pinnacle crafts).
- [ ] **Step 5: Wire loot.** 9552 `loot_pool`: boots + ranged + trophy at normal drop (`itemdropchance: 100` for the set-piece); the material `40166` at **~3%** — implement via a separate low-chance mechanism (if `loot_pool` is all-or-per-item, give the Sentinel `itemdropchance` for guaranteed items and add 40166 to `loot_pool` with the ~3% governed by the loot roll; confirm how `loot_pool` + `itemdropchance` interact and set 40166's rarity/drop so effective ≈3%). Also add 40166 at a tiny chance to 9547 (The Reach-Alpha) per the roster.
- [ ] **Step 6: Validate + boot** (items +4, buffs already in; loot refs resolve). **Step 7: Commit.**

## Task 10: World-critic + boot verify + geared feel-test + merge + docs

- [ ] **Step 1: World-critic pass** — dispatch an adversarial reviewer over all 30 rooms + mobs + buffs/mutator + items + the 6342 edit, checking: LORE BOUNDARY violations (any ship/sky/moons/machine/answer leak — HIGH severity), symbol bleed beyond 6347 + the door, cross-file contradictions, mutation dups across the roster, hazard-room prose vs the `mutators` wiring, the door terminus (6372 no east / not a "come back later"), the Sentinel/vault being OFF the door path (solo can reach the door), biome validity, hyphen-wrap/colon-space/semicolon gotchas, and terminus wording. Fix all findings; re-boot.
- [ ] **Step 2: Full boot verify** — nuke instances, rebuild, boot: `ValidateZoneConsistency errors=0 warnings=0 mode=panic`, `ValidateAllFlags OK`, counts clean (rooms +30, mobs +10, items +4, buffs +1/+2), 0 panics.
- [ ] **Step 3: Geared feel-test (the ARC calibration pass).** This is where Cascade #20's owed re-tune folds in. Point the smoketester at 6343; **buff the smoketester to approximate the user's prod char** (edit 17.yaml stats up to a geared-master level — the user's is ~3800–4000 leaderboard) OR coordinate with the user to drive their own char. Via mudagent: traverse 6343→6372, confirm the hull reveal + lore boundary read right, verify the hazard rooms tick (buff 94 applies each round; HP drains), verify the door-ward `defuse` path works (disarm → reach 6372), examine the door (learn you need the disc), and fight the Sentinel set-piece — **measure it against a 300g Oasis royal encounter** (fight length, lethality, whether the BIS drops). Retune `statpool` (Sentinel + fauna + the Cascade Pass 275/550) and the BIS item stats + buff `tick_percent` to hit parity. Report to `tools/playtest/reports/`.
- [ ] **Step 4: Merge** `git checkout master && git merge --no-ff feature/eastern-highlands`.
- [ ] **Step 5: Docs + memory** — mark #21 built in `docs/ZONE_EXPANSION.md`; update the zone-expansion memory (roomids/mobids/the hazard-mechanic recipe/the calibration results) + MEMORY.md status. Commit.

---

## Self-review notes
- **Spec coverage:** geography/attach (T6) ✓; 3 stages incl. hull reveal + lore boundary (T2–T4) ✓; data-driven hazards — discharge buff + mutator + defusable traps (T5) ✓; difficulty ramp + Sentinel set-piece @1200+adds (T7,T8) ✓; BIS in unserved slots feet+ranged matched to 300g Oasis (T9) ✓; trophy + ~3% ultra-rare material (T9) ✓; locked disc-door terminus (T4) ✓; near-zero NPCs / Maren's father / Reth's markers / symbol peak (T2,T4) ✓; arc calibration pass (T10 step 3) ✓.
- **Deferred to #22 (not in this plan):** disc-acquisition, its frequency, whether the interior is instanced, the door-opening mechanic, the interior, the revelation.
- **Ordering:** T8 mob loot refs T9 items — author T9 before T8's boot, or accept forward refs confirmed at T9. **Calibration risk:** statpools + BIS stats + buff ticks are STARTING values; T10 step 3 is where they're validated against the geared yardstick — do not treat the numbers as final.
- **Type/id consistency:** rooms 6343–6372, mobs 9543–9552, buffs 94/95, mutator `hull_discharge`, materials 40165/40166, folder `eastern_highlands`, region "The Eastern Reach", loot in feet + shooting slots.
```
