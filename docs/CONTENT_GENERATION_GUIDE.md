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

For spells and buffs, use the schema docs as reference and write the YAML manually for now — the `/new-mob` and `/new-item` commands will reference buff IDs as needed.

**Schema docs** live in `docs/schemas/`:
- `room.md`, `mob.md`, `item.md`, `spell.md`, `buff.md`, `dialogue.md`

---

## 2. Building a Full Zone: `/zone-sketch` → `/new-room` × N

The recommended workflow for adding a new area:

### Step 1: Plan with `/zone-sketch`

```
/zone-sketch "flooded salt flats east of Sanctum Basin, 6 rooms, inhospitable terrain"
```

The command reads `world.md`, studies an existing zone for structural reference, and produces:
- Zone display name and folder name
- Suggested roomid range
- A room list with titles and one-line descriptions
- An adjacency map showing exit connections
- Boundary connections to existing zones
- Mob and item suggestions
- Tone notes

**No files are written.** Review the output, adjust room titles and connections as needed, then proceed to step 2.

### Step 2: Generate rooms with `/new-room`

Work in ID order (start from the suggested base ID):

```
/new-room "cracked salt flat, bleached expanse of fractured earth, Flooded Salt Flats, east to room 202"
/new-room "sunken tidal channel, narrow cut with mineral-stained walls, Flooded Salt Flats, north to room 203, west to room 201"
```

Each call:
1. Reads `world.md` and `docs/schemas/room.md`
2. Samples existing rooms as style examples
3. Generates the YAML
4. Writes it to the correct path
5. Reminds you about reverse exits

### Step 3: Update boundary rooms

For each room in an existing zone that should connect to the new zone, edit that room's YAML to add the exit:

```yaml
exits:
  east:
    roomid: 201   # new zone entry room
```

Check for instance saves (see Section 4) if editing an existing zone.

### Step 4: Add mobs and items

Run `/new-mob` for creatures. Then add `spawninfo` entries to room YAMLs to place them.

Run `/new-item` for loot/props. Add items to mob `character.items` lists or room `spawninfo`.

### Step 5: Restart and walk the zone

Restart the server and walk through the zone. Verify:
- All exits connect correctly (test both directions)
- Mob descriptions display as intended
- Items work when picked up / used
- No startup errors in the server log

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

After generating content, verify it in-game:

**For a new room:**
1. Restart the server
2. Teleport or walk to the room (use `goto {roomid}` if you have admin access)
3. Verify the title and description display correctly
4. Test each exit direction — confirm the reverse exit exists
5. Check the map if `mapsymbol` was set

**For a new mob:**
1. Restart the server
2. Walk to a room with that mob's `spawninfo` entry
3. Wait for the mob to spawn (or force-spawn if admin tools are available)
4. `look {mobname}` — verify description
5. If the mob is an NPC, `say hello` — verify dialogue responds
6. If hostile, initiate combat — verify it behaves as intended

**For a new item:**
1. Restart the server
2. Find the item in the world (carried by a mob, in a room, etc.)
3. `get {item}` — verify it can be picked up
4. `look {item}` or `examine {item}` — verify description
5. If wearable: `wear {item}` — verify it equips to the right slot
6. If consumable: `use {item}` — verify the buff applies

**For a new zone:**
1. Walk the full path from the boundary connection into the zone
2. Test all exit connections (go in, come back)
3. Verify all rooms render correctly
4. Check that mobs spawn and behave
5. Check the server log for any YAML parse errors at startup

---

## 6. ConvertForFilename Reference

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
