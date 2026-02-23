# /zone-sketch

Plan a new zone for DOGMud: produce a room list, adjacency map, and zone metadata for human review. **Does not write any files.**

## Instructions

You are planning a new zone for the DOGMud MUD. This is a design document — output only, no files written.

### Step 1 — Load context

Read these files before generating anything:
1. `world.md` — lore, geography, tone, existing zones and their relationships
2. `docs/schemas/room.md` — room field reference (biome, exits, etc.)

Then read one existing zone as a structural reference. Glob:
```
_datafiles/world/dogmud/rooms/sanctum_basin/*.yaml
```
Read 3–4 files to understand room density, description style, and exit patterns.

### Step 2 — Parse the zone concept

From `$ARGUMENTS`, identify:
- Zone display name (exactly as it should appear in YAML `zone:` fields)
- Zone folder name: apply `ConvertForFilename` to the display name
- Approximate room count
- Biome and environmental character
- Geographic relationship to existing zones (near what? connected how?)
- Any named features, landmarks, or significant locations

### Step 3 — Determine roomid range

Glob all room YAMLs:
```
_datafiles/world/dogmud/rooms/**/*.yaml
```
Find the highest existing roomid. Suggest a starting roomid for the new zone (highest + 1 or a convenient round number for readability).

### Step 4 — Generate the zone sketch

Produce a planning document with the following sections:

---

**ZONE: {Display Name}**
**Folder:** `_datafiles/world/dogmud/rooms/{folder_name}/`
**Zone field value:** `"{Display Name}"`
**Biome:** {biome}
**Suggested roomid range:** {start}–{start + count - 1}
**Room count:** {N}

---

**ROOM LIST**

For each room, provide:
- Roomid
- Proposed title (2–5 words)
- One-sentence description of character/feel
- Biome (if different from zone default)
- Suggested mapsymbol

Example format:
```
Room 201 — "Cracked Salt Flat"       [mapsymbol: ~]
  Bleached expanse of fractured earth, pooled brine collecting in the lowest seams.

Room 202 — "Sunken Tidal Channel"    [mapsymbol: =]
  A narrow cut where the ancient sea once flowed; the walls are streaked with mineral stains.
```

---

**ADJACENCY MAP**

Show the exit connections as a simple text diagram or list. Format:
```
201 (east) → 202
202 (north) → 203, (west) → 201
203 (south) → 202, (east) → 204
```

Optionally include an ASCII grid if the layout is simple enough:

```
      [203]
        |
[201]--[202]--[205]
        |
      [204]
```

---

**ZONE BOUNDARY CONNECTIONS**

List any rooms in existing zones that should link into this new zone (room ID, direction, target room in this zone). Remind the user these existing rooms will need their YAML files updated to add the exit.

---

**MOB AND ITEM SUGGESTIONS** (brief, no YAML)

List 3–5 creature types or NPCs that would feel appropriate in this zone. For each: name, one sentence on role/behavior. These are suggestions only — generate with `/new-mob` afterward.

List 2–3 items that could be found or crafted here. These are suggestions only — generate with `/new-item` afterward.

---

**TONE NOTES**

2–3 sentences on the zone's emotional register, what players should feel while traveling through it, and how it fits into the world of Gaius. Ground this in world.md lore.

---

### Step 5 — Output and next steps

After the planning document, remind the user:

> This is a planning document only — no files have been written.
>
> To build this zone:
> 1. Review and adjust the room list and adjacency map.
> 2. Run `/new-room "{room title and description}, {Zone Name}, {exit connections}"` for each room in ID order.
> 3. Run `/new-mob` for any creatures you want to populate the zone with.
> 4. Add `spawninfo` to room YAMLs to place mobs and items.
> 5. Update any existing zone rooms that should link into this new zone.
> 6. Restart the server and walk through the zone to verify.

---

## Usage

```
/zone-sketch "flooded salt flats east of Sanctum Basin, 6 rooms, hostile terrain"
/zone-sketch "abandoned Chrysalis monastery in the foothills, 10 rooms, mixture of ruined and intact"
/zone-sketch "the Fold Scar, 4 rooms, a crack in the earth where the ancient cataclysm left a wound"
```

Arguments are a freeform zone concept. Include: environmental character, approximate room count, and geographic context relative to existing zones.
