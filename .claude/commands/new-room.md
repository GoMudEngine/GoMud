# /new-room

Generate a new room YAML file for DOGMud.

## Instructions

You are generating a new room data file for the DOGMud MUD. Follow these steps exactly.

### Step 1 — Load context

Read these files before generating anything:
1. `world.md` — lore, tone, world facts, geography (Gaius, Sanctum Basin layout, etc.)
2. `docs/schemas/room.md` — complete field reference, filename formula, gotchas

Then read 2 existing room files as few-shot examples:
- Glob `_datafiles/world/dogmud/rooms/sanctum_basin/*.yaml`, read the first result
- Glob `_datafiles/world/dogmud/rooms/startland/1.yaml` and read it

### Step 2 — Determine the next available roomid

Glob all room YAML files:
```
_datafiles/world/dogmud/rooms/**/*.yaml
```
Find the highest existing roomid and suggest the next integer. Confirm with the user if there is any ambiguity.

### Step 3 — Determine the zone folder

From the description in `$ARGUMENTS`, identify the zone. Apply `ConvertForFilename` to the zone display name:
- Lowercase everything
- Keep a-z, 0-9
- Drop apostrophes
- All other characters → underscore

Example: `"Sanctum Basin"` → `sanctum_basin`

Check whether `_datafiles/world/dogmud/rooms/{zone_folder}/` exists. If not, warn:
> Zone folder `{folder}/` does not exist. If this is a new zone, the folder will be created when the file is written. Make sure the zone display name in the YAML matches your intent exactly — it determines the folder name.

### Step 4 — Parse exit information

From `$ARGUMENTS`, identify any explicit exit connections (e.g. "connects east to room 102", "north to room 50"). For each exit:
- Use the roomid integer only — **never a room title**
- Note the reverse exit direction (if this room connects north to room 102, room 102 presumably connects south to this new room — remind the user to add that reverse exit manually)

### Step 5 — Generate the room YAML

Using `$ARGUMENTS` as the creative brief:
- Write a title: short (2–5 words), evocative, matches the world tone
- Write a description: 3–5 sentences. Ground it in sensory detail (smell, sound, temperature, texture). Reference the zone's established geography and biome. Do not narrate player emotion.
- Choose `biome` from the valid values in `docs/schemas/room.md`
- Set `mapsymbol` (one character) and `maplegend` (one word) if the room warrants a map presence
- Add `idlemessages` (3–5 entries) that bring the space to life — subtle environmental details, distant sounds, ambient movement
- Add `nouns` if there are interesting features worth examining
- Add `spawninfo` only if the user explicitly requested spawns
- Do **not** add bank/storage/character fields unless explicitly requested

**No hard numbers in descriptions or messages.** Never expose stat values or damage amounts.

### Step 6 — Verify before writing

Check:
1. `{roomid}.yaml` — filename is just the integer ID
2. `roomid` is unique (not already in use)
3. Zone folder name uses underscores, not hyphens
4. `zone` field display name is correct
5. All exit `roomid` values are integers, not text
6. Required fields present: `roomid`, `zone`, `title`, `description`, `exits`

### Step 7 — Write the file

Write to:
```
_datafiles/world/dogmud/rooms/{zone_folder}/{roomid}.yaml
```

### Step 8 — Remind the user

After writing:
> Room written. A few things to check:
> 1. **Reverse exits**: If this room connects to an existing room, add the matching return exit to that room's YAML too.
> 2. **Instance saves**: If editing a room in an existing zone, check `_datafiles/world/dogmud/rooms.instances/{zone_folder}/{roomid}.yaml` for stale instance data.
> 3. **Restart server** to load the new room.

---

## Usage

```
/new-room "narrow basalt ledge overlooking the pool, Sanctum Basin, connects east to room 116"
/new-room "ruined watchtower entrance, A Dark Forest, north to room 77, south to room 78"
/new-room "cramped tunnel junction, Endless Trashheap, all four cardinal directions open"
```

Arguments are a freeform description. Include: room feel/type, zone name, and any explicit exit connections.
