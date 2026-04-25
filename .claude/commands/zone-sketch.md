# /zone-sketch

Plan a new zone for DOGMud: produce a room list, adjacency map, and zone metadata for human review. **Does not write any files.**

## Instructions

**This is a Phase 1 planning command.** Per the Zone-Building SOP in
`docs/CONTENT_GENERATION_GUIDE.md` Section 2, zones are built in two
phases: rooms+mobs+items+spawns first, then quests as a separate
pass. Do NOT plan or sketch quests here. Quest planning happens in
`/sketch-quest` after the smoke checklist passes.

You are planning a new zone for the DOGMud MUD. This is a design document — output only, no files written.

### Step 1 — Load context

Read these files before generating anything:
1. `world.md` — lore, geography, tone, existing zones and their relationships
2. `docs/schemas/room.md` — room field reference (biome, exits, etc.)
3. List archetypes available: glob `_datafiles/world/dogmud/behaviors/archetypes/*.yaml`. Note the 13 archetype filenames — these are the values you'll suggest for `behavior_archetype` on each mob in Step 4 below.

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
**IMPORTANT — mapsymbol/maplegend gotcha:** Do NOT suggest `mapsymbol` or `maplegend` for rooms unless they are genuine landmarks (town square, bank, shop). The map renderer embeds these values inside ANSI formatting tags, so special characters (`=`, `|`, `-`, `#`, `~`, `>`, `<`, `!`, `?`, `$`, `*`, `\`, `+`, `.`) break the mini-map — raw tag fragments bleed into the room description. Even plain letters cause issues if `maplegend` is not a recognized map class. When in doubt, omit both fields entirely and let the engine use its default room marker. If mapsymbol/maplegend are set or changed, delete any instance saves in `rooms.instances/<zone>/` — cached old values will silently override template changes.

Example format:
```
Room 201 — "Cracked Salt Flat"
  Bleached expanse of fractured earth, pooled brine collecting in the lowest seams.

Room 202 — "Sunken Tidal Channel"
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

**MOB SUGGESTIONS**

Suggest 3–5 creatures that fit the zone. For each, propose a
`behavior_archetype` from the 13 available — reuse first; flag
"candidate for new archetype" if no existing one fits; flag "boss/
signature, custom legacy" only for unique encounters.

Format:
```
{creature concept} — archetype: {existing_archetype_name}
  {one sentence on what makes them feel zone-appropriate}

{creature concept} — archetype: NEW (proposed: {name})
  {one sentence — and a sentence on why no existing archetype fits}

{boss name} — archetype: CUSTOM (boss/signature)
  {one sentence on why this needs hand-tuned behavior}
```

Aim for ≥80% of suggestions to reuse existing archetypes. If you
find yourself proposing more than one NEW archetype per zone,
reconsider — you may be over-specifying behavior.

---

**ITEM SUGGESTIONS**

List 2–3 zone-flavored items that could be found, looted, or
crafted here. These are suggestions only — generate with `/new-item`
afterward.

---

**TONE NOTES**

2–3 sentences on the zone's emotional register, what players should feel while traveling through it, and how it fits into the world of Gaius. Ground this in world.md lore.

---

### Step 5 — Output and next steps

After the planning document, remind the user:

> This is a Phase 1 planning document — no files have been written.
>
> **Phase 1 build sequence:**
> 1. Review and adjust the room list, adjacency map, and
>    mob/archetype suggestions.
> 2. Run `/new-room "..."` for each room in ID order.
> 3. Run `/new-mob "..."` for each mob — `/new-mob` will surface the
>    archetype priority order (reuse → new archetype → custom legacy
>    for bosses).
> 4. Run `/new-item "..."` for each new item.
> 5. Manually edit room YAMLs to add `spawninfo` blocks placing mobs
>    and items.
> 6. Update existing zone rooms that link into this new zone.
> 7. Restart the server.
>
> **Then run the smoke checklist** (full text from
> `docs/CONTENT_GENERATION_GUIDE.md` Section 2, copied here for
> convenience):
>
> ```
> [ ] Walked every room. Each title and description reads cleanly.
> [ ] Verified every exit. Every room reachable.
> [ ] No `mapsymbol`/`maplegend` set on non-landmark rooms.
> [ ] Cartesian consistency: no overlapping (X,Y,Z) within zone or
>     against `docs/coordinate_map.md`. Update coordinate_map.md
>     with the new zone's coordinates.
> [ ] Fought ≥1 mob of each combat archetype used in the zone.
> [ ] Killed at least one mob and looted the corpse.
> [ ] Identified at least one zone-specific item.
> [ ] Triggered any non-combat archetype interaction worth testing.
> [ ] No instance saves committed.
> [ ] No stale instance saves blocking template edits.
> [ ] go build ./... clean. go test ./... clean.
> ```
>
> Only when this is fully ticked off — run `/sketch-quest` to begin
> Phase 2.

---

## Usage

```
/zone-sketch "flooded salt flats east of Sanctum Basin, 6 rooms, hostile terrain"
/zone-sketch "abandoned Chrysalis monastery in the foothills, 10 rooms, mixture of ruined and intact"
/zone-sketch "the Fold Scar, 4 rooms, a crack in the earth where the ancient cataclysm left a wound"
```

Arguments are a freeform zone concept. Include: environmental character, approximate room count, and geographic context relative to existing zones.
