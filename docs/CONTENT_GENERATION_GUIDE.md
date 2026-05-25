# Content Generation Guide

This guide explains how to use Claude Code's slash commands to generate new world content for DOGMud — rooms, mobs, items, and more — with full awareness of world lore, schema rules, and naming conventions.

---

## 1. Which Command to Use

| What you want to create | Command |
|-------------------------|---------|
| A creature, NPC, or enemy | `/new-mob` |
| A single room | `/new-room` |
| A weapon, armor piece, or consumable | `/new-item` |
| A full new zone (planning only) | `/zone-sketch` |
| A new quest (planning)  | `/sketch-quest` |
| A new quest (execution) | `/new-quest <plan-file>` |

For spells and buffs, use the schema docs as reference and write the YAML
manually for now. **Flavor text goes in YAML text fields** (`cast_user_text`,
`cast_room_text`, etc. for spells; `start_user_text`, `end_user_text`, etc.
for buffs) — see `spell.md` Section 2b and `buff.md` Section 4. Only create
a `.js` file if the spell/buff needs custom logic.

**Schema docs** live in `docs/schemas/`:
- `room.md`, `mob.md`, `item.md`, `spell.md`, `buff.md`, `dialogue.md`

---

## 2. Building a Full Zone: Phase 1 → Smoke → Phase 2

DOGMud zones are built in **two phases** with a **smoke gate** between
them. This ordering came out of repeated quest-related issues — quests
built in parallel with rooms/mobs entangle changes and make iteration
painful. Build the zone first; tune it; *then* layer quests on top.

### Phase 1 — Zone Build

Build everything except quests:

- Rooms (descriptions, exits, biome, spawninfo placeholders)
- Mobs (using `behavior_archetype` — see priority order below)
- Items (loot tables, drops, crafting components)
- Spawn placement (`spawninfo` filled in on rooms)

Slash commands: `/zone-sketch` → `/new-room` (loop) → `/new-mob`
(loop) → `/new-item` (loop) → manually edit `spawninfo` blocks.

#### Step 1: Plan with `/zone-sketch`

```
/zone-sketch "flooded salt flats east of Sanctum Basin, 6 rooms, inhospitable terrain"
```

Produces zone metadata, room list with adjacency, mob suggestions
(with proposed `behavior_archetype` for each), item suggestions, and
the smoke checklist. Writes nothing — review and adjust.

#### Step 2: Generate rooms with `/new-room`

Work in ID order:

```
/new-room "cracked salt flat, bleached expanse of fractured earth, Flooded Salt Flats, east to room 202"
/new-room "sunken tidal channel, narrow cut with mineral-stained walls, Flooded Salt Flats, north to room 203, west to room 201"
```

#### Step 3: Update boundary rooms

For each room in an existing zone that should connect to the new
zone, edit that room's YAML to add the exit. Check for instance saves
(see Section 4) if editing an existing zone.

#### Step 4: Generate mobs with `/new-mob`

`/new-mob` will offer the `behavior_archetype` priority order — see
"Mob Behavior Archetype Priority" below.

#### Step 5: Generate items with `/new-item`

Then manually add `spawninfo` entries to room YAMLs to place mobs and
items.

### Smoke Gate — must pass before Phase 2

Run through this checklist for the new zone before opening
`/sketch-quest`:

```
[ ] Walked every room. Each title and description reads cleanly (no
    missing punctuation, broken ANSI tags, dropped sentences).
[ ] Verified every exit. Every room reachable; no one-way dead-ends
    that weren't intentional.
[ ] No `mapsymbol`/`maplegend` set on non-landmark rooms (those break
    the mini-map). Restart server, check the map renders cleanly.
[ ] Cartesian consistency: ran `map` from each room (or from a few
    spread-out rooms) and confirmed no two rooms in the new zone
    overlap. Cross-referenced `docs/coordinate_map.md` to confirm no
    new-zone room shares (X,Y,Z) with an adjacent existing zone's
    rooms. Update `docs/coordinate_map.md` with the new zone's
    coordinates as part of this step.
[ ] Fought ≥1 mob of each combat archetype used in the zone. Confirm
    the archetype actually drives the behavior you expected (e.g., a
    `tank_taunter` actually taunts, an `ambusher` actually ambushes).
[ ] Killed at least one mob and looted the corpse. Spawn loot drops
    fire correctly.
[ ] Identified at least one zone-specific item. Stats render cleanly,
    no raw numbers leak into descriptions.
[ ] Triggered any non-combat archetype interaction worth testing
    (questgiver dialogue, shopkeeper buy/sell, prey flee).
[ ] No instance saves committed: rooms.instances/<zone>/,
    mobs.instances/, shops/<zone>/ are NOT in `git status`.
[ ] No stale instance saves blocking template edits — see CLAUDE.md
    "Room Instance Saves" SOP.
[ ] go build ./... clean. go test ./... clean.
```

### Phase 2 — Quest Pass

Only after the smoke checklist is fully ticked off. See Section 6
("Building a Quest") for the workflow.

This way, if a quest reveals a balance or layout issue, you fix the
zone freely without any quest state to migrate. Quests are the topmost
layer; they should never be load-bearing for zone iteration.

### Mob Behavior Archetype Priority

When generating a new mob, choose `behavior_archetype` in this order:

1. **Reuse an existing archetype.** The 13 in
   `_datafiles/world/dogmud/behaviors/archetypes/` cover the common
   roles. If one fits, use it.
2. **Author a new archetype YAML** if the behavior pattern is reusable
   (i.e., other mobs in this or future zones will share it). Add a new
   file under `behaviors/archetypes/`.
3. **Fall back to legacy `aiprofile` / `combatcommands` /
   `tactic_preset`** *only* for bosses or signature one-off NPCs whose
   behavior is genuinely unique.

`/new-mob` offers these in this order. Picking option 3 should be a
deliberate choice, not the path of least resistance.

See `docs/schemas/mob.md` "Behavior Archetypes" for the list of all 13
archetypes with role descriptions and pairing notes.

---

## 3. Review Checklist

Before a file is finalized, verify:

**Filenames:**
- [ ] Mob: `{id}-{ConvertForFilename(name)}.yaml` — e.g. `12-cave_troll.yaml`
- [ ] Room: `{roomid}.yaml` — e.g. `201.yaml` (no name, just ID)
- [ ] Item: `{id}-{ConvertForFilename(name)}.yaml` — e.g. `10010-obsidian_blade.yaml`
- [ ] Buff: `{id}-{ConvertForFilename(name)}.yaml` — e.g. `8-stone_skin.yaml`
- [ ] Spell: `{spellid}.yaml` — e.g. `fire-bolt.yaml` (no conversion for spells)
- [ ] Zone folder: underscores only, matches `ConvertForFilename(zone display name)`

**IDs:**
- [ ] No two mobs share a mobid
- [ ] No two rooms share a roomid
- [ ] No two items share an itemid within the same type range
- [ ] No two buffs share a buffid

**Required fields:**
- [ ] Room: `roomid`, `zone`, `title`, `description`, `exits`
- [ ] Mob: `mobid`, `zone`, `character.name`, `character.description`, `character.speciesid`
- [ ] Item: `itemid`, `name`, `description`, `type`, `subtype`
- [ ] Buff: `buffid`, `name`, `description`
- [ ] Spell: `spellid`, `name`, `description`, `type`, `schools`

**World tone:**
- [ ] No modern slang, no anachronisms
- [ ] No raw numbers in player-visible text (damage, armor, durations)
- [ ] NPC names and creature names fit the grounded, low-fantasy tone
- [ ] Descriptions reference sensory details, not game mechanics

---

## 4. The Instance Save Gotcha

**What happens:** When the server runs, room state (items on the floor, gold, etc.) is periodically saved to `_datafiles/world/dogmud/rooms.instances/{zone_folder}/{roomid}.yaml`. These **instance saves override template data** on the next server start.

**When it bites you:** You edit a room template in `rooms/`, restart the server, and your changes don't appear. The instance save from the previous session overwrote your edits.

**How to fix it:**
1. After editing any room template, check:
   ```
   _datafiles/world/dogmud/rooms.instances/{zone_folder}/{roomid}.yaml
   ```
2. If the file exists, delete it.
3. Restart the server. The engine will load the clean template and create a new instance save.

**Note:** `spawninfo` is tagged `instance:"skip"` in the engine, so spawn configuration is always loaded from the template, not the instance save. But room description, title, biome, exits, and other fields CAN be overridden.

---

## 5. Smoke-Test Workflow

The full zone-build smoke checklist lives in Section 2 ("Building a
Full Zone" → "Smoke Gate"). For single-room or single-mob/item smoke
tests outside of a zone build, the same general approach applies:

- Restart the server.
- Walk to the relevant room (use `goto {roomid}` if admin).
- Verify the title, description, and any spawned mobs/items render
  correctly.
- Check the server log for YAML parse errors at startup.
- For mobs: `look {mobname}`, `say hello` (if NPC), or initiate
  combat (if hostile) — verify behavior matches the
  `behavior_archetype` you chose.
- For items: `get`, `look`, `wear`/`use` as appropriate.

---

## 6. Building a Quest: `/sketch-quest` → `/new-quest`

**Phase 2 — only after the smoke checklist passes.** See Section 2's
"Smoke Gate." If the zone for this quest hasn't been smoke-tested,
stop and finish the smoke checklist first. If the zone is older and
the checklist was never formally run, walk through it now anyway —
quest issues we've seen historically trace back to layout/balance
problems that smoke would have caught.

The recommended workflow for adding a new quest:

### Step 1: Plan with `/sketch-quest`

```
/sketch-quest "delivery quest where player carries a report from Harn to a clerk in Thornwall"
```

The command reads `world.md`, existing quests, and dialogue examples, then
produces a structured planning document covering:
- Quest ID, steps, and gating diagram
- Alternative paths (e.g. `give` vs `ask` for item delivery)
- Every file that must be created or modified
- A gotchas checklist (triggers, item consumption, mob groups, etc.)
- A verification plan for in-game testing

**No game files are written.** Review the output in
`_datafiles/quest_plans/`, adjust as needed, then proceed to step 2.

### Step 2: Generate with `/new-quest`

```
/new-quest 14-the_lost_caravan.md
```

The command reads the approved plan and generates all files in order:
quest YAML, items, dialogue, room scripts, mob scripts, and YAML
modifications. It runs the gotchas checklist on each file before writing,
then builds the project to catch errors.

### Step 3: Clean up and test

After generation:
1. Delete any stale instance saves listed in the output
2. Restart the server
3. Follow the verification plan from the sketch document

---

## 7. ConvertForFilename Reference

The engine derives expected filenames using `ConvertForFilename()`. The rules:

1. Lowercase everything
2. Keep: `a–z`, `0–9`
3. Drop: apostrophes (`'`)
4. Replace everything else with: `_` (underscore)

**Examples:**

| Input | Output |
|-------|--------|
| `"Sanctum Basin"` | `sanctum_basin` |
| `"Cave Troll"` | `cave_troll` |
| `"Elder Saris"` | `elder_saris` |
| `"O'Brien's Watch"` | `obriens_watch` |
| `"Iron Longsword"` | `iron_longsword` |
| `"Minor Shield"` | `minor_shield` |
| `"fire-bolt"` | `fire_bolt` *(but spells skip this — use spellid directly)* |

**Exception:** Spell filenames use the `spellid` value verbatim. `spellid: fire-bolt` → file is `fire-bolt.yaml`, not `fire_bolt.yaml`.

---

## 8. Schedules

No `/new-schedule` command yet — author by hand using
`docs/schemas/schedule.md`. Restart required after authoring.
Validators run at boot and panic on coverage / pathing /
reference errors.
