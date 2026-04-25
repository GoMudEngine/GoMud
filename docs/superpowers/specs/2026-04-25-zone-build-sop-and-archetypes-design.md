# Zone-Building SOP + Archetype-Aware Tooling

**Date:** 2026-04-25
**Status:** Design (approved)
**Type:** Documentation + slash-command tooling update

## Problem

Two related issues with the current zone-building workflow:

1. **Quests entangle with zone iteration.** Quest design has historically
   happened in parallel with rooms/mobs/items, which means quest state
   gets in the way of zone tuning — fix a balance issue and you have to
   migrate quest playtest data, fix a layout issue and quest paths
   break. The team has had recurring quest-related bugs that trace back
   to zones that weren't fully smoke-tested before quests landed.

2. **Behavior-tree archetypes are widely deployed but not surfaced in
   tooling.** 13 behavior archetypes exist under
   `_datafiles/world/dogmud/behaviors/archetypes/` and are referenced by
   newer-zone mobs (`ashwick`, `dustwalk_road`, etc.) via the
   `behavior_archetype:` field. But:
   - `docs/schemas/mob.md` does not document the field.
   - `.claude/commands/new-mob.md` recommends legacy `aiprofile` /
     `combatcommands` / `tactic_preset` and never mentions archetypes.
   - `.claude/commands/zone-sketch.md` suggests creature types
     generically with no archetype awareness.

   New mobs end up using legacy fields by default. The modern
   archetype system gets bypassed.

## Goals

1. Establish a documented two-phase zone-building workflow with an
   explicit smoke-test gate between phases. Phase 1 = rooms + mobs +
   items + spawns; Phase 2 = quests.
2. Bake the archetype priority order — **reuse existing → author new
   archetype → custom legacy (bosses only)** — into the slash commands
   and the schema doc, so new mobs default to the modern path.
3. Surface tier-1 mob authoring fields (`behavior_archetype`,
   `archetype` stat distribution, `spawnmutations` /
   `mutationchance`) as first-class options in `/new-mob`.

## Non-Goals

- New slash commands (e.g., `/zone-build-checklist`,
  `/zone-populate`). Lightweight edits to existing commands suffice.
- Tooling-enforced gating between phases (e.g., `/sketch-quest`
  refusing to run without a checklist file). Self-discipline checklist
  in the SOP doc is enough.
- Automated cartesian-overlap detection. Manual verification against
  in-game `map` rendering and `docs/coordinate_map.md` is acceptable.
- Migrating existing zones to use `behavior_archetype`. New zones
  follow the new SOP; existing legacy mobs keep their existing fields
  unless touched for other reasons.

## Files Modified

```
┌─────────────────────────────────────────────────────────────────┐
│  docs/CONTENT_GENERATION_GUIDE.md   [centerpiece]               │
│   • Phase 1: zone build (rooms + mobs + items + spawns)         │
│   • Smoke checklist (numbered, copy-pasteable)                  │
│   • Phase 2: quest pass (only after checklist passes)           │
│   • Archetype priority order                                    │
└─────────────────────────────────────────────────────────────────┘
                               │
        ┌──────────────────────┼──────────────────────┐
        ▼                      ▼                      ▼
┌─────────────────┐   ┌─────────────────┐   ┌────────────────────┐
│ /zone-sketch    │   │ /new-mob        │   │ /sketch-quest      │
│ Phase 1 framing │   │ Archetype-first │   │ Precondition note: │
│ Mob suggestions │   │ Priority order  │   │ "smoke must pass"  │
│ surface         │   │ surfaced; tier-1│   │                    │
│ archetypes      │   │ fields listed   │   │                    │
│ Final checklist │   │                 │   │                    │
│ block           │   │                 │   │                    │
└─────────────────┘   └─────────────────┘   └────────────────────┘
                               │
                               ▼
                    ┌──────────────────────────┐
                    │ docs/schemas/mob.md      │
                    │ • Add behavior_archetype │
                    │ • List 13 archetypes     │
                    │ • Cross-reference SOP    │
                    └──────────────────────────┘
```

| File | What changes |
|------|--------------|
| `docs/CONTENT_GENERATION_GUIDE.md` | New "Zone-Building SOP" section: phase model + smoke checklist + archetype priority |
| `docs/schemas/mob.md` | Add `behavior_archetype` field row; new "Behavior Archetypes" section listing all 13; mark legacy fields as legacy; cross-link to SOP |
| `.claude/commands/zone-sketch.md` | Phase 1 framing header; archetype-aware mob suggestions; final smoke checklist block |
| `.claude/commands/new-mob.md` | Reordered Step 4 to surface `behavior_archetype` first with priority order; add `archetype`/`spawnmutations`/`mutationchance` as tier-1 fields; mark legacy fields as exception path |
| `.claude/commands/sketch-quest.md` | One-paragraph preamble linking to the smoke checklist |

No new files. No deletions.

## Component: Zone-Building SOP (centerpiece)

New section in `docs/CONTENT_GENERATION_GUIDE.md`. Authoritative text:

### Zone-Building SOP

DOGMud zones are built in **two phases** with a **smoke gate** between
them. This ordering came out of repeated quest-related issues — quests
built in parallel with rooms/mobs entangle changes and make iteration
painful. Build the zone first; tune it; *then* layer quests on top.

#### Phase 1 — Zone Build

Build everything except quests:

- Rooms (descriptions, exits, biome, spawninfo placeholders).
- Mobs (using `behavior_archetype` — see priority order below).
- Items (loot tables, drops, crafting components).
- Spawn placement (`spawninfo` filled in on rooms).

Slash commands: `/zone-sketch` → `/new-room` (loop) → `/new-mob`
(loop) → `/new-item` (loop) → manually edit `spawninfo` blocks.

#### Smoke Gate — must pass before Phase 2

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

#### Phase 2 — Quest Pass

Only after the smoke checklist is fully ticked off. Slash commands:
`/sketch-quest` (plan) → `/new-quest <plan-file>` (generate).

This way, if a quest reveals a balance or layout issue, you fix the
zone freely without any quest state to migrate. Quests are the topmost
layer; they should never be load-bearing for zone iteration.

### Mob Behavior Archetype Priority

When generating a new mob, choose `behavior_archetype` in this order:

1. **Reuse an existing archetype.** The 13 in
   `_datafiles/world/dogmud/behaviors/archetypes/` cover the common
   roles. If one fits, use it.
2. **Author a new archetype YAML** if the behavior pattern is reusable
   (i.e., other mobs in this or future zones will share it). Add a
   new file under `behaviors/archetypes/`.
3. **Fall back to legacy `aiprofile` / `combatcommands` /
   `tactic_preset`** *only* for bosses or signature one-off NPCs whose
   behavior is genuinely unique.

`/new-mob` will offer these in this order. Picking option 3 should be
a deliberate choice, not the path of least resistance.

## Component: `/new-mob` Restructure

Step 4 ("Generate the mob YAML") rewritten in priority order:

**(a) Choose `behavior_archetype` — most important field.** Lists all
13 archetypes with role descriptions. States the priority order
explicitly.

**(b) Choose stat distribution `archetype`.** Table mapping
`"fighting"` / `"casting"` / `""` to stat splits and use cases. Notes
the natural pairing with behavior archetype.

**(c) Decide on `spawnmutations` and `mutationchance`.** Explains
guaranteed mutations vs. random-chance bonus mutations. Provides
guidance: signature traits → `spawnmutations`; variety → `mutationchance`.

**(d) Standard fields:** name, description, `speciesid`, `hostile`,
`maxwander`, `activitylevel`, `groups`, `hates`, `idlecommands`. No
behavior change.

**(e) Skip these unless you've chosen the legacy path:** `aiprofile`,
`combatcommands`, `tactic_preset`, `tactics`. With a
`behavior_archetype` set, these are usually unnecessary.

**Other small edits:**

- Step 1 adds a glob/list of `_datafiles/world/dogmud/behaviors/archetypes/`.
- Step 2 example mob changes to one using `behavior_archetype`
  (e.g., `_datafiles/world/dogmud/mobs/ashwick/264-timber_wolf.yaml`).
- Step 8 reminder mentions the smoke-test gate before
  `/sketch-quest`.

## Component: `/zone-sketch` Restructure

**Header insert** right after `## Instructions`:

> **This is a Phase 1 planning command.** Per the Zone-Building SOP in
> `docs/CONTENT_GENERATION_GUIDE.md`, zones are built in two phases:
> rooms+mobs+items+spawns first, then quests as a separate pass. Do
> NOT plan or sketch quests here. Quest planning happens in
> `/sketch-quest` after the smoke checklist passes.

**Step 1 addition:** list archetypes folder
(`_datafiles/world/dogmud/behaviors/archetypes/*.yaml`).

**Restructured "MOB AND ITEM SUGGESTIONS" section** — splits into
"MOB SUGGESTIONS" and "ITEM SUGGESTIONS". Mob entries now propose a
`behavior_archetype` from the 13, flag NEW-archetype candidates, and
flag boss/CUSTOM cases. Format:

```
{creature concept} — archetype: {existing_archetype_name}
  {one sentence on what makes them feel zone-appropriate}

{creature concept} — archetype: NEW (proposed: {name})
  {one sentence — and a sentence on why no existing archetype fits}

{boss name} — archetype: CUSTOM (boss/signature)
  {one sentence on why this needs hand-tuned behavior}
```

Guideline: ≥80% of suggestions should reuse existing archetypes.

**Step 5 next-steps block** rewritten as a Phase 1 build sequence
followed by the full smoke checklist (copy of the centerpiece
checklist for convenience), followed by "Only when this is fully
ticked off — run `/sketch-quest` to begin Phase 2."

## Component: `/sketch-quest` Preamble

One block inserted right after `## Instructions`:

> **Phase 2 only.** Per the Zone-Building SOP in
> `docs/CONTENT_GENERATION_GUIDE.md`, quests are built AFTER the zone
> smoke-test checklist passes. If the zone for this quest hasn't been
> smoke-tested:
>
> - Stop and finish the smoke checklist first.
> - If the zone is older and the checklist was never formally run,
>   walk through it now anyway. Quest issues we've seen historically
>   trace back to layout/balance problems that smoke would have
>   caught.
>
> If the smoke is genuinely done, proceed.

No other changes to `/sketch-quest`. The precondition is a reminder,
not enforcement.

## Component: `docs/schemas/mob.md` Updates

**1. Add `behavior_archetype` to the top-level fields table** (near
the existing `aiprofile` row):

```markdown
| `behavior_archetype` | string | no | Filename (no extension) of an
archetype YAML in
`_datafiles/world/dogmud/behaviors/archetypes/`. Drives the mob's
behavior tree. **Strongly preferred over legacy
`aiprofile`/`combatcommands`/`tactic_preset` for new mobs.** See
archetype list below. |
```

Mark legacy fields (`aiprofile`, `tactic_preset`, `tactics`,
`reaction_delay`, `tactical_discipline`) with "(legacy — prefer
`behavior_archetype`)" in their description columns.

**2. New section "Behavior Archetypes"** after the existing `archetype`
(stat distribution) discussion. Includes:

- Available archetypes table (all 13 with one-line role descriptions).
- Priority order for new mobs (1. reuse, 2. new YAML, 3. legacy bosses
  only).
- Cross-link to `docs/CONTENT_GENERATION_GUIDE.md` SOP.
- Pairing guidance with stat-distribution `archetype`:
  - `pure_caster` / `support_caster` → `archetype: "casting"`
  - `generic_fighter` / `tank_taunter` / `ambusher` /
    `melee_self_buff` → `archetype: "fighting"`
  - `prey` / `noncombat_*` → `archetype: ""` (uniform)

**3. Cross-link** added to "Filename & Location" pointing readers to
the SOP for the broader workflow.

## Testing

This is documentation + tooling-prompt work. No automated tests.

**Manual verification on landing:**

1. Open each modified file and read end-to-end. Each should be
   internally consistent and not contradict the others.
2. Spot-check that the smoke checklist appears identically in
   `CONTENT_GENERATION_GUIDE.md` (centerpiece) and `/zone-sketch.md`
   (final block).
3. Read `/new-mob.md` Step 4 fresh — does it lead an author naturally
   to picking an existing archetype?
4. Read `docs/schemas/mob.md` "Behavior Archetypes" section — does the
   archetype list match the actual file list under
   `_datafiles/world/dogmud/behaviors/archetypes/`?

**Validation by use:** the next zone built after this lands should
naturally follow the new SOP. Any friction points get logged for
follow-up.

## Open Questions Resolved During Brainstorm

- **Scope** → lightweight edits, no new slash commands.
- **Archetype as required vs. preferred** → strongly preferred default
  with explicit priority order (reuse → new archetype → legacy
  bosses).
- **Phase 1 deliverable** → rooms + mobs + items + spawns. Quests
  alone are Phase 2.
- **Tier-1 mob fields surfaced** → `behavior_archetype` + stat-distribution
  `archetype` + `spawnmutations` + `mutationchance`.
- **Smoke gate definition** → concrete numbered checklist (not
  self-defined, not tooling-enforced).
- **Cartesian consistency** → manual verification via in-game `map` +
  `docs/coordinate_map.md`; doc-update is part of the smoke step.
