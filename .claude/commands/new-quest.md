# /new-quest

Generate all files for a planned quest from an approved sketch. Reads a quest
plan markdown file and creates/modifies all listed files.

## Instructions

You are generating all game files for a DOGMud quest from an approved plan.
Follow these steps exactly.

### Step 1 — Validate argument

`$ARGUMENTS` must be a filename (with or without path). Look for it in:
1. `_datafiles/quest_plans/$ARGUMENTS`
2. `_datafiles/quest_plans/$ARGUMENTS.md` (append `.md` if missing)
3. The literal path if it looks like a full path

**If no argument is provided or the file is not found:**

> No quest plan file specified. Create a plan first with:
> `/sketch-quest "your quest concept here"`
>
> Then run `/new-quest <plan-filename>` to generate the files.

### Step 2 — Load and parse the plan

Read the plan markdown file. Extract:
- Quest ID, name, steps, rewards
- File list (creates and modifications)
- Step chain with trigger mechanisms
- Alternative paths
- Gotchas checklist items

### Step 3 — Load context

Read:
1. `world.md` — lore, tone, world context
2. `docs/schemas/dialogue.md` — dialogue field reference
3. All existing files that will be modified (read each one before editing)
4. Relevant schema docs for any items/mobs being created:
   - `docs/schemas/item.md` if creating items
   - `docs/schemas/mob.md` if creating mobs
   - `docs/schemas/room.md` if modifying rooms

### Step 4 — Generate files in order

Work through the file list from the plan. For each file:

**4a. Quest YAML** — create `_datafiles/world/dogmud/quests/{id}-{name}.yaml`
with steps and rewards from the plan.

**4b. Items** — create any new quest items under the appropriate
`_datafiles/world/dogmud/items/` subdirectory.

**4c. Dialogue files** — create new or modify existing dialogue YAMLs, adding
quest nodes with proper gating. For each dialogue file:
- Every `grantsQuest` tree node MUST include `"quest"` and `"task"` in
  `triggers`
- Every `grantsQuest` pattern entry MUST include `"quest"` and `"task"` in
  `keywords`
- Use `questRequired` and `questExcluded` for proper step gating
- Item delivery steps need BOTH a dialogue path AND an `onGive` script
- **Narrative voice:** NPC `text` must use first person ("I", "my"). Hints
  must describe player options from the player's perspective. Never write
  3rd-person self-references like "Ask about why she left" — write "You
  could ask why she left" or "You could ask about the marriage."
- **Trigger discoverability:** Every trigger word must appear in a hint,
  NPC dialogue text, room description, or quest log. If a hint says "calm
  her down", the triggers MUST include "calm". Undiscoverable = broken.
- **Prefer `questRequired` over `requires`** for quest-gated dialogue
  nodes. `requires` depends on per-player memory that can expire and brick
  quests. Only use `requires` for non-quest conversational branching.

**4d. Room scripts** — create `.js` files for item pickups / command
intercepts. Confirm the noun appears in the room's `description` or `nouns`
section so `get <noun>` feels natural.

**4e. Mob scripts** — create `.js` files for `onGive` handlers.
**CRITICAL: give.go transfers the item to the mob BEFORE onGive fires.** This
means:
- Quest-accepting NPCs: `onGive` just grants the quest token (item is already
  consumed by the transfer)
- "Wrong NPC" handlers (quest giver who should NOT keep the item): `onGive`
  must call `user.GiveItem(itemId)` to return a copy to the player
- Every NPC involved in item-delivery quests needs an `onGive` script, or the
  player loses the item to the default "considers the item" behavior
- Quest givers who hand out items via `givesItem` need a recovery dialogue node
  (fires when player has the quest but not the item) that gives a replacement

**4f. Mob YAML modifications** — group changes, spawninfo additions. Verify
quest NPCs are NOT in hostile mob groups.

**4g. Room YAML modifications** — spawninfo, nouns, exits. Check for matching
instance saves in `rooms.instances/` before modifying.

Before writing each file, run the relevant gotchas checklist items from the
plan to verify correctness.

### Step 5 — Post-generation checklist

After all files are written:

1. Run `go build ./...` to verify no compilation errors
2. List all instance saves that need deletion:
   ```
   _datafiles/world/dogmud/rooms.instances/{zone}/
   _datafiles/world/dogmud/mobs.instances/{zone}/
   ```
3. Delete stale instance saves (after confirmation from the user)

Remind the user:

> Quest files generated. Restart the server and test with the verification
> plan from the sketch document.

### Step 6 — Print verification plan

Reproduce the verification plan from the sketch so the user can test
immediately:

1. Restart server
2. Walk to quest giver → `ask <npc> quest` → confirm quest granted
3. For each step: perform the action → confirm quest advances
4. For each alternative path: test that too
5. Try to skip steps (go to completion NPC early) → confirm blocked
6. Complete quest → confirm rewards

---

## Usage

```
/new-quest 5-the_innkeepers_complaint.md
/new-quest 14-the_lost_caravan.md
```

`$ARGUMENTS` is the filename of a quest plan in `_datafiles/quest_plans/`.
Create a plan first with `/sketch-quest` if you don't have one.
