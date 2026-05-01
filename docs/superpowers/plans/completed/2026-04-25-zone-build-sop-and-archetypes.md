# Zone-Building SOP + Archetype-Aware Tooling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bake the two-phase zone-build workflow (rooms+mobs+items+spawns → smoke gate → quests) and the `behavior_archetype` priority order into the team's slash commands, schema doc, and content-generation guide.

**Architecture:** Documentation + slash-command-prompt edits only. No new files; no code; no automated tests. Five existing markdown files modified.

**Tech Stack:** Markdown documentation, Claude Code slash command prompts.

**Spec:** `docs/superpowers/specs/2026-04-25-zone-build-sop-and-archetypes-design.md`

---

## File Structure

**Modified files (5):**

| File | Responsibility |
|------|----------------|
| `docs/CONTENT_GENERATION_GUIDE.md` | Centerpiece SOP — Phase 1 / smoke checklist / Phase 2 / archetype priority |
| `docs/schemas/mob.md` | `behavior_archetype` field reference, archetype list, legacy field labeling |
| `.claude/commands/zone-sketch.md` | Phase 1 framing, archetype-aware mob suggestions, embedded smoke checklist |
| `.claude/commands/new-mob.md` | Step 4 restructure: archetype-first, priority order, tier-1 fields surfaced |
| `.claude/commands/sketch-quest.md` | One-paragraph Phase 2 preamble linking to smoke checklist |

**No new files. No deletions.**

---

## Task 1: SOP Centerpiece — `docs/CONTENT_GENERATION_GUIDE.md`

The centerpiece. This task lands the authoritative SOP that the other four files reference. Do this task first so subsequent tasks can link to it.

**Files:**
- Modify: `docs/CONTENT_GENERATION_GUIDE.md`

### Background

The existing guide has these top-level sections:

1. Which Command to Use
2. Building a Full Zone (lines 29–92) — **stale; will be REPLACED**
3. Review Checklist
4. The Instance Save Gotcha
5. Smoke-Test Workflow (lines 145–178) — **stale; will be REPLACED**
6. Building a Quest (lines 181–220) — **augmented (precondition note)**
7. ConvertForFilename Reference

We replace Section 2 and Section 5 with the new SOP, and prepend a one-paragraph precondition note to Section 6.

- [ ] **Step 1: Read the current guide end-to-end**

```bash
sed -n '1,260p' docs/CONTENT_GENERATION_GUIDE.md
```

Confirm sections 2 and 5 still match the line ranges given above (the file may have shifted since the plan was written).

- [ ] **Step 2: Replace Section 2 with the new "Building a Full Zone" SOP**

Section 2 currently runs from `## 2. Building a Full Zone: \`/zone-sketch\` → \`/new-room\` × N` through to the `---` before `## 3. Review Checklist`. Replace the entire block with:

```markdown
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
```

- [ ] **Step 3: Replace Section 5 with a pointer to the smoke checklist**

Section 5 currently runs from `## 5. Smoke-Test Workflow` through to the `---` before `## 6. Building a Quest`. The new smoke checklist is now in Section 2's "Smoke Gate." Replace Section 5 with a short pointer:

```markdown
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
```

- [ ] **Step 4: Augment Section 6 with the Phase 2 precondition note**

Insert a new paragraph immediately after the `## 6. Building a Quest: \`/sketch-quest\` → \`/new-quest\`` heading, BEFORE the existing "The recommended workflow…" line:

```markdown
## 6. Building a Quest: `/sketch-quest` → `/new-quest`

**Phase 2 — only after the smoke checklist passes.** See Section 2's
"Smoke Gate." If the zone for this quest hasn't been smoke-tested,
stop and finish the smoke checklist first. If the zone is older and
the checklist was never formally run, walk through it now anyway —
quest issues we've seen historically trace back to layout/balance
problems that smoke would have caught.

The recommended workflow for adding a new quest:
```

(The rest of Section 6 stays unchanged.)

- [ ] **Step 5: Verify the file reads cleanly end-to-end**

```bash
sed -n '1,300p' docs/CONTENT_GENERATION_GUIDE.md | head -300
```

Spot-check:
- Section 2's table of contents reference (in Section 1) still makes sense.
- The smoke checklist is exactly one place (Section 2). Section 5 points back to it.
- Section 6 starts with the Phase 2 precondition note.
- No dangling "Step N" references that no longer exist after the rewrite.
- Markdown heading levels (## vs ###) are consistent.

- [ ] **Step 6: Commit**

```bash
git add docs/CONTENT_GENERATION_GUIDE.md
git commit -m "$(cat <<'EOF'
docs(guide): zone-building SOP — Phase 1 / smoke gate / Phase 2

Replaces Section 2 with explicit two-phase workflow + smoke checklist.
Replaces Section 5 with a pointer to the unified checklist. Adds
Phase 2 precondition note to Section 6. Adds the mob behavior
archetype priority order (existing → new archetype → legacy).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Schema Update — `docs/schemas/mob.md`

Add `behavior_archetype` to the field reference, mark legacy fields, and add the "Behavior Archetypes" section listing all 13 archetypes.

**Files:**
- Modify: `docs/schemas/mob.md`

- [ ] **Step 1: Read the current schema**

```bash
sed -n '1,200p' docs/schemas/mob.md
```

Locate the top-level fields table (around lines 36–62) and the section after the existing `statpool` archetype-distribution discussion (around line 188).

- [ ] **Step 2: Add `behavior_archetype` row to the top-level fields table**

In the table that begins around line 36 with `| Field | Type | Required | Notes |`, find the row for `aiprofile` (around line 51). Insert a new row immediately ABOVE it:

```markdown
| `behavior_archetype` | string | no | Filename (without `.yaml`) of an archetype in `_datafiles/world/dogmud/behaviors/archetypes/`. Drives the mob's behavior tree. **Strongly preferred over legacy `aiprofile`/`combatcommands`/`tactic_preset` for new mobs.** See "Behavior Archetypes" below. |
```

- [ ] **Step 3: Mark legacy fields with "(legacy)" annotations**

Edit these table rows (in the same top-level fields table) by appending `(legacy — prefer \`behavior_archetype\`)` to their Notes column:

| Field | Add to Notes |
|-------|--------------|
| `aiprofile` | ` (legacy — prefer \`behavior_archetype\`)` |
| `tactic_preset` | ` (legacy — prefer \`behavior_archetype\`)` |
| `tactics` | ` (legacy — prefer \`behavior_archetype\`)` |
| `reaction_delay` | ` (legacy — prefer \`behavior_archetype\`)` |
| `tactical_discipline` | ` (legacy — prefer \`behavior_archetype\`)` |

For each, find the existing row and append the parenthetical to the end of the existing description, before the closing `|`. Don't disturb the rest of the row content.

- [ ] **Step 4: Insert "Behavior Archetypes" section**

Find the existing paragraph that begins ``**`statpool` distributes by archetype.**`` (around line 188 in Section 4 "Gotchas"). Immediately AFTER the paragraph that ends with "...identical mob templates will still vary. Use explicit `stats:` overrides when a specific stat spread matters." but BEFORE the next bold-italic gotcha, insert a new top-level section:

```markdown
---

## 5. Behavior Archetypes

`behavior_archetype` selects a behavior-tree YAML from
`_datafiles/world/dogmud/behaviors/archetypes/`. The archetype drives
combat decision-making, packmate awareness, and reactive AI without
needing to author per-mob `combatcommands`/`tactics`.

### Available archetypes

| Archetype | Role |
|-----------|------|
| `generic_fighter` | Melee with bash/trip/grapple toolkit. Default for non-tank fighters. |
| `tank_taunter` | Melee with signature taunt + self-buffs. For high-priority threats. |
| `melee_self_buff` | Melee fighter who pre-buffs before engaging. |
| `ambusher` | Hidden until engagement; high opening burst. |
| `pure_caster` | Spell-focused; flees from melee, kites with damage. |
| `support_caster` | Buffs/heals packmates; rarely the front-line target. |
| `leader` | Commands packmates, calls for help, coordinates. |
| `prey` | Flees on engagement; non-aggressive. |
| `lookout` | Stationary observer; calls for help when triggered. |
| `combat_passive` | In combat but doesn't attack — atmospheric or quest fodder. |
| `noncombat_passive` | Walks idles, no combat behavior. |
| `noncombat_questgiver` | Stationary, dialogue-only NPC. |
| `noncombat_shopkeeper` | Stationary shop NPC. |

### Priority for new mobs

When authoring a new mob:

1. **Reuse** an existing archetype if one fits.
2. **Author a new archetype YAML** under `behaviors/archetypes/` if
   the behavior pattern is reusable across multiple mobs.
3. **Custom legacy** (`aiprofile` + `combatcommands` + `tactic_preset`)
   ONLY for bosses or signature one-off NPCs.

See `docs/CONTENT_GENERATION_GUIDE.md` "Building a Full Zone" for the
full zone-build workflow including the smoke-test checklist.

### Pairing with stat distribution

`behavior_archetype` and `archetype` (stat distribution) usually pair
naturally:

- `pure_caster` / `support_caster` → `archetype: "casting"`
- `generic_fighter` / `tank_taunter` / `ambusher` /
  `melee_self_buff` → `archetype: "fighting"`
- `prey` / `noncombat_*` → `archetype: ""` (uniform)

---
```

Note: this inserts a new section. Renumber any subsequent section headings that exist below it (look for `## 5.`, `## 6.`, etc., and bump them by 1). If the existing schema has no numbered sections after this insertion point, the renumber is unnecessary.

- [ ] **Step 5: Add cross-link to Section 1 (Filename & Location)**

Section 1 of `mob.md` is "Filename & Location" (starts at line 3). Add a one-line pointer at the END of that section, before the `---` separator:

```markdown
**Workflow:** new mobs are usually built as part of a zone — see
`docs/CONTENT_GENERATION_GUIDE.md` Section 2 for the full zone-build
SOP including the `behavior_archetype` priority order.
```

- [ ] **Step 6: Verify**

```bash
sed -n '30,75p' docs/schemas/mob.md   # field table area
sed -n '180,260p' docs/schemas/mob.md  # statpool gotcha + new section
```

Spot-check:
- `behavior_archetype` row appears in the field table.
- The five legacy fields each have `(legacy — prefer behavior_archetype)`.
- The new "Behavior Archetypes" section is present, with all 13 archetypes.
- The cross-link to CONTENT_GENERATION_GUIDE.md is in Section 1.

- [ ] **Step 7: Commit**

```bash
git add docs/schemas/mob.md
git commit -m "$(cat <<'EOF'
docs(schema): document behavior_archetype field + 13 archetypes

Adds behavior_archetype to the field table; marks aiprofile,
tactic_preset, tactics, reaction_delay, and tactical_discipline as
legacy. New "Behavior Archetypes" section lists all 13 with role
descriptions, priority order for new mobs, and pairing notes with
stat-distribution archetype.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Restructure `/zone-sketch` slash command

Add Phase 1 framing header, surface archetypes in mob suggestions, embed the smoke checklist.

**Files:**
- Modify: `.claude/commands/zone-sketch.md`

- [ ] **Step 1: Read the current command**

```bash
sed -n '1,140p' .claude/commands/zone-sketch.md
```

- [ ] **Step 2: Insert Phase 1 framing header**

After the line `## Instructions` (currently line 5) and before `You are planning a new zone for the DOGMud MUD.`, insert:

```markdown
**This is a Phase 1 planning command.** Per the Zone-Building SOP in
`docs/CONTENT_GENERATION_GUIDE.md` Section 2, zones are built in two
phases: rooms+mobs+items+spawns first, then quests as a separate
pass. Do NOT plan or sketch quests here. Quest planning happens in
`/sketch-quest` after the smoke checklist passes.

```

- [ ] **Step 3: Add archetype folder to Step 1 context**

In Step 1 ("Load context"), the current numbered list of files to read has 2 entries. Add a third:

```markdown
3. List archetypes available: glob `_datafiles/world/dogmud/behaviors/archetypes/*.yaml`. Note the 13 archetype filenames — these are the values you'll suggest for `behavior_archetype` on each mob in Step 4 below.
```

- [ ] **Step 4: Restructure "MOB AND ITEM SUGGESTIONS" section**

Find the existing section that begins `**MOB AND ITEM SUGGESTIONS** (brief, no YAML)`. Replace the entire section (from that header through to the `---` separator before `**TONE NOTES**`) with:

```markdown
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
```

- [ ] **Step 5: Replace the Step 5 "next steps" block**

Find the section that begins `### Step 5 — Output and next steps`. Replace its body (from after the `### Step 5` heading through to the `---` separator before `## Usage`) with:

```markdown
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
```

- [ ] **Step 6: Verify**

```bash
sed -n '1,180p' .claude/commands/zone-sketch.md
```

Spot-check:
- Phase 1 framing header is right after `## Instructions`.
- Step 1 lists 3 context items including archetypes folder.
- Mob suggestions section uses the new archetype-aware format.
- Step 5 ends with the smoke checklist + the "run /sketch-quest only after" line.
- The `## Usage` section at the bottom is unchanged.

- [ ] **Step 7: Commit**

```bash
git add .claude/commands/zone-sketch.md
git commit -m "$(cat <<'EOF'
docs(zone-sketch): Phase 1 framing + archetype-aware mob suggestions

Adds Phase 1 header pointing to the SOP. Mob suggestions now propose
behavior_archetype per creature (reuse / NEW / CUSTOM). Final
next-steps block ends with the smoke checklist and the gate to
/sketch-quest.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Restructure `/new-mob` slash command

The biggest single edit. Step 4 ("Generate the mob YAML") becomes archetype-first with explicit priority order and tier-1 fields surfaced.

**Files:**
- Modify: `.claude/commands/new-mob.md`

- [ ] **Step 1: Read the current command**

```bash
sed -n '1,100p' .claude/commands/new-mob.md
```

- [ ] **Step 2: Add archetypes folder to Step 1 context**

Step 1 currently lists two files to read and then "glob 2 existing mob files." Append a third numbered context item BEFORE the "Then glob..." line:

```markdown
3. List archetypes: glob `_datafiles/world/dogmud/behaviors/archetypes/*.yaml`. The 13 filenames here are the valid values for `behavior_archetype` in Step 4 below.
```

- [ ] **Step 3: Update the Step 1 example mob to one using `behavior_archetype`**

The current Step 1 example mobs are:
- `_datafiles/world/dogmud/mobs/sanctum_basin/55-elder_saris.yaml` (complex NPC with LLMProfile)
- `_datafiles/world/dogmud/mobs/startland/1-rat.yaml` (simple hostile creature)

Replace `1-rat.yaml` with a mob that uses `behavior_archetype` so the few-shot demonstrates the modern pattern:

```markdown
- `_datafiles/world/dogmud/mobs/sanctum_basin/55-elder_saris.yaml` (complex NPC with LLMProfile)
- `_datafiles/world/dogmud/mobs/ashwick/264-timber_wolf.yaml` (simple hostile creature with `behavior_archetype: generic_fighter`)
```

- [ ] **Step 4: Replace Step 4 "Generate the mob YAML" entirely**

Find `### Step 4 — Generate the mob YAML` and replace its full body (everything from that heading through to the `### Step 5 — Verify before writing` heading) with:

```markdown
### Step 4 — Generate the mob YAML

Using `$ARGUMENTS` as the creative brief, fill in the YAML. The
fields below are listed in priority order — start with behavior, then
stats, then flavor.

#### (a) Choose `behavior_archetype` — most important field

The 13 available archetypes (loaded in Step 1):

| Archetype | Role |
|-----------|------|
| `generic_fighter` | Melee with bash/trip/grapple toolkit. Default for non-tank fighters. |
| `tank_taunter` | Melee with signature taunt + self-buffs. For high-priority threats. |
| `melee_self_buff` | Melee fighter who pre-buffs before engaging. |
| `ambusher` | Hidden until engagement; high opening burst. |
| `pure_caster` | Spell-focused; flees from melee, kites with damage. |
| `support_caster` | Buffs/heals packmates; rarely the front-line target. |
| `leader` | Commands packmates, calls for help, coordinates. |
| `prey` | Flees on engagement; non-aggressive. |
| `lookout` | Stationary observer; calls for help when triggered. |
| `combat_passive` | In combat but doesn't attack — atmospheric or quest fodder. |
| `noncombat_passive` | Walks idles, no combat behavior. |
| `noncombat_questgiver` | Stationary, dialogue-only NPC. |
| `noncombat_shopkeeper` | Stationary shop NPC. |

**Priority order** (from `docs/CONTENT_GENERATION_GUIDE.md` Section
2):

1. **Reuse** an existing archetype if one fits the brief.
2. **Author a new archetype YAML** under `behaviors/archetypes/` if
   the behavior is reusable across multiple mobs. Tell the user this
   is what you're doing — they will need to author the new archetype
   YAML separately.
3. **Custom legacy** (`aiprofile` + `combatcommands` +
   `tactic_preset`) ONLY for bosses or signature one-off NPCs.

Picking option 3 should be rare and deliberate. If `$ARGUMENTS`
describes a generic creature, option 1 is the answer.

#### (b) Choose stat distribution `archetype`

| Value | Stat split | Use when |
|-------|------------|----------|
| `"fighting"` | 80% physical (Str/Dex/Vit) | Brawlers, beasts, melee-focused humanoids |
| `"casting"` | 80% mental (Per/Wil/Cha) | Spellcasters, scholars, charisma-driven NPCs |
| `""` (default) | uniform random | Mixed roles, generic NPCs |

Pair with `behavior_archetype` sensibly — a `pure_caster` should
almost always have stat archetype `"casting"`.

#### (c) Decide on `spawnmutations` and `mutationchance`

Mutations differentiate mobs of the same archetype. A
`generic_fighter` goblin and a `generic_fighter` ogre share AI but
feel different because one has thick-skin and the other has rage.

- `spawnmutations: [42, 18]` — guaranteed mutation IDs on every
  spawn. IDs from `_datafiles/world/dogmud/mutations/`.
- `mutationchance: 25` — % chance (0–100) to gain ONE extra random
  mutation on top.

Use `spawnmutations` for signature traits (a "stone goblin" always
has Stone Skin); use `mutationchance` for variety (most spawned
wolves are normal, occasional pack-leader has bonus mutations).

#### (d) Standard fields

`character.name`, `character.description` (2–4 sentences, physical
only, no personality narration), `character.speciesid`, `hostile`,
`maxwander`, `activitylevel`, `groups`, `hates`, `idlecommands`. See
`docs/schemas/mob.md` for the full reference.

**Naming:** fits the world's tone (no modern slang, no fantasy
clichés).

**Description:** physical details, no behavior/personality narration.
No raw numbers (damage, armor, etc.).

#### (e) Skip these unless you've chosen the legacy path (option 3)

`aiprofile`, `combatcommands`, `tactic_preset`, `tactics`,
`reaction_delay`, `tactical_discipline`. With a `behavior_archetype`
set, these are usually unnecessary — the archetype YAML drives
behavior. Including them alongside `behavior_archetype` is
redundant and potentially confusing.

**Do not add an LLMProfile unless the user explicitly requests it.**
LLMProfile requires Ollama to be running.

```

- [ ] **Step 5: Update Step 8 reminder to mention the smoke gate**

Find `### Step 8 — Remind the user`. The current reminder mentions restarting the server and adding spawninfo. APPEND a new paragraph at the end of the existing reminder:

```markdown
>
> Once your zone has all its mobs and items in place, run the smoke
> checklist (in `docs/CONTENT_GENERATION_GUIDE.md` Section 2) before
> starting `/sketch-quest` for any quests in this zone.
```

- [ ] **Step 6: Verify**

```bash
sed -n '1,140p' .claude/commands/new-mob.md
```

Spot-check:
- Step 1 lists 3 numbered context items including archetypes folder.
- Step 1 example mobs include the timber_wolf with `behavior_archetype`.
- Step 4 leads with `behavior_archetype` (sub-section a), then stat archetype (b), then mutations (c), then standard fields (d), then "skip these" legacy fields (e).
- Step 8 reminder ends with the smoke-checklist note.

- [ ] **Step 7: Commit**

```bash
git add .claude/commands/new-mob.md
git commit -m "$(cat <<'EOF'
docs(new-mob): archetype-first Step 4 + tier-1 field surfacing

Step 4 now leads with behavior_archetype (priority: reuse → new
archetype → legacy bosses). Stat-distribution archetype, spawn
mutations, and mutationchance are tier-1 fields with usage tables.
Legacy aiprofile/combatcommands/tactics fields documented as
exception path. Step 8 reminder mentions the zone smoke gate.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: `/sketch-quest` Phase 2 preamble

One-paragraph preamble. Smallest task.

**Files:**
- Modify: `.claude/commands/sketch-quest.md`

- [ ] **Step 1: Read the current command**

```bash
sed -n '1,30p' .claude/commands/sketch-quest.md
```

The first lines should be the title (`# /sketch-quest`), description, `## Instructions` heading, and the body starting with "You are planning a new quest for the DOGMud MUD."

- [ ] **Step 2: Insert the preamble**

After the line `## Instructions` (currently line 5) and BEFORE the line beginning "You are planning a new quest for the DOGMud MUD.", insert:

```markdown
**Phase 2 only.** Per the Zone-Building SOP in
`docs/CONTENT_GENERATION_GUIDE.md` Section 2, quests are built AFTER
the zone smoke-test checklist passes. If the zone for this quest
hasn't been smoke-tested:

- Stop and finish the smoke checklist first.
- If the zone is older and the checklist was never formally run,
  walk through it now anyway. Quest issues we've seen historically
  trace back to layout/balance problems that smoke would have
  caught.

If the smoke is genuinely done, proceed.

```

- [ ] **Step 3: Verify**

```bash
sed -n '1,30p' .claude/commands/sketch-quest.md
```

Confirm the preamble is right after `## Instructions` and before the existing body. The rest of the file is unchanged.

- [ ] **Step 4: Commit**

```bash
git add .claude/commands/sketch-quest.md
git commit -m "$(cat <<'EOF'
docs(sketch-quest): Phase 2 preamble linking to smoke checklist

One-paragraph reminder that quest planning is Phase 2 and the zone
smoke checklist must pass first. No body changes.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Read-through verification

Cross-check the five files for internal consistency. No code changes; this is the manual verification step the spec calls for.

**Files:** none (read-through only)

- [ ] **Step 1: Read all five files end-to-end**

```bash
sed -n '1,260p' docs/CONTENT_GENERATION_GUIDE.md
sed -n '1,260p' docs/schemas/mob.md
sed -n '1,180p' .claude/commands/zone-sketch.md
sed -n '1,140p' .claude/commands/new-mob.md
sed -n '1,30p' .claude/commands/sketch-quest.md
```

- [ ] **Step 2: Run the consistency checklist**

For each item below, confirm by visual inspection:

```
[ ] The smoke checklist appears in exactly TWO places, identically:
    docs/CONTENT_GENERATION_GUIDE.md Section 2 (centerpiece) and
    .claude/commands/zone-sketch.md Step 5 (copy for convenience).
[ ] The 13 archetype names appear identically in:
    docs/schemas/mob.md Section "Behavior Archetypes" AND
    .claude/commands/new-mob.md Step 4 (a).
    No typos; no archetypes added or omitted in one but not the other.
[ ] The priority order text (1. Reuse, 2. New archetype, 3. Custom
    legacy bosses) appears identically in:
    docs/CONTENT_GENERATION_GUIDE.md Section 2,
    docs/schemas/mob.md "Behavior Archetypes" section,
    .claude/commands/new-mob.md Step 4 (a).
[ ] Cross-references resolve. Files reference each other by section:
    - schemas/mob.md → CONTENT_GENERATION_GUIDE.md Section 2 ✓
    - sketch-quest.md → CONTENT_GENERATION_GUIDE.md Section 2 ✓
    - new-mob.md → CONTENT_GENERATION_GUIDE.md Section 2 ✓
    - zone-sketch.md → /sketch-quest (next step) ✓
[ ] /new-mob.md no longer recommends legacy aiprofile/combatcommands
    as default — they're listed as Step 4 (e) "skip these unless
    legacy path."
[ ] /zone-sketch.md no longer mentions "MOB AND ITEM SUGGESTIONS"
    as one combined section — it's "MOB SUGGESTIONS" and "ITEM
    SUGGESTIONS" separately, and mob suggestions propose archetypes.
[ ] /sketch-quest.md preamble references the smoke checklist by
    location (Section 2 of CONTENT_GENERATION_GUIDE.md).
```

- [ ] **Step 3: Re-read against the actual archetype filesystem**

```bash
ls _datafiles/world/dogmud/behaviors/archetypes/
```

Compare the listed archetypes (13 expected) to the tables in
`docs/schemas/mob.md` and `.claude/commands/new-mob.md`. Confirm exact
correspondence. If a mismatch exists (filesystem has more or fewer than
the tables), fix the tables to match the filesystem and recommit the
fix.

- [ ] **Step 4: Compare against the spec**

Open the spec one more time:

```bash
sed -n '1,260p' docs/superpowers/specs/2026-04-25-zone-build-sop-and-archetypes-design.md
```

For each spec section ("Component: ..."), confirm the corresponding
file change matches. List any drift.

- [ ] **Step 5: If everything is consistent, no commit needed**

This task is verification only. If you found drift in Step 2 or Step
3, the fix commit covers it; otherwise, this task ends with no new
commit.

If you fixed something in Step 3:

```bash
git add docs/schemas/mob.md .claude/commands/new-mob.md
git commit -m "$(cat <<'EOF'
docs: sync archetype tables with filesystem

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-Review Notes

**Spec coverage:** every spec component has a task —
- Centerpiece SOP (CONTENT_GENERATION_GUIDE.md) → T1
- Schema doc updates → T2
- /zone-sketch restructure → T3
- /new-mob restructure → T4
- /sketch-quest preamble → T5
- Manual verification → T6

**Type/identifier consistency:** the 13 archetype names appear in three
files (schema, /new-mob, /zone-sketch via the user's mob suggestions).
T6 explicitly verifies they match each other and the filesystem. The
priority-order text is duplicated (intentional — three audiences) and
T6 verifies it.

**No placeholders:** all "fill in" content blocks contain actual text.
The task instructions tell the engineer exactly what to insert and
where. Test code is N/A (docs only).

**Scope:** five files, sequential tasks, no code, no automated tests.
Single-PR-sized.
