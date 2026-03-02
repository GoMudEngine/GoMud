# /sketch-quest

Plan a new quest for DOGMud. Produces a quest design document for human review. **Does not write any game files.** Outputs to a markdown file.

## Instructions

You are planning a new quest for the DOGMud MUD. This is a design document — output only, no game files written.

### Step 1 — Load context

Read these files before generating anything:
1. `world.md` — lore, zones, NPCs, tone
2. `docs/schemas/dialogue.md` — dialogue field reference, quest gating fields

Then read 2 existing quest YAMLs as structural examples. Glob:
```
_datafiles/world/dogmud/quests/*.yaml
```
Read 2 results (not the generic quest).

Then read 2 existing dialogue YAMLs that contain quest nodes. Glob:
```
_datafiles/world/dogmud/dialogue/**/*.yaml
```
Pick 2 files that contain `grantsQuest` fields.

### Step 2 — Parse the quest concept

From `$ARGUMENTS`, identify:
- Quest name and description
- Zone(s) involved
- Quest-giving NPC (existing or new)
- Delivery/completion NPC(s) (existing or new)
- Quest type: fetch, delivery, investigation, combat, escort, multi-zone
- Approximate step count

**If the concept is vague or underspecified, ASK the user for clarification
before proceeding.** Specifically ask about:
- Which existing NPCs should be involved (or if new ones are needed)
- What zone(s) the quest spans
- What the reward should be (gold, item, skill point, narrative)
- Whether the quest should lock/unlock other quests

### Step 3 — Determine next questid

Glob `_datafiles/world/dogmud/quests/*.yaml`, find the highest questid
(ignoring the 1000000 generic quest), and suggest the next integer.

### Step 4 — Generate the quest sketch

Output a structured planning document with the following sections:

---

**QUEST: {Quest Name}**
**Quest ID:** {N}
**Zone(s):** {zone list}
**Type:** {fetch/delivery/investigation/combat/multi-zone}
**Quest giver:** {NPC name} (mob {id}, room {id}, zone)
**Completion NPC:** {NPC name} (mob {id}, room {id}, zone)

---

**STEP CHAIN**

For each step, specify:
- Step ID (e.g. `start`, `investigate`, `evidence`, `end`)
- What triggers advancement to this step
- Trigger mechanism: dialogue node (`grantsQuest`), room script (`onCommand`),
  mob script (`onGive`), or automatic (e.g. entering a room)
- Quest token: `{questid}-{stepid}`
- Player-facing description and hint

Format:
```
Step 1: "start" — granted by dialogue node
  Trigger: ask <npc> quest/task → dialogue node `help_quest` with
           grantsQuest: "{id}-start"
  Token: {id}-start
  Description: "..."
  Hint: "..."

Step 2: "evidence" — granted by room script
  Trigger: `get ledger` in room 421 → room script onCommand gives item,
           grants quest
  Token: {id}-evidence
  Description: "..."
  Hint: "..."
```

---

**ALTERNATIVE PATHS**

For each step where players might reasonably try a different approach, specify
the alternative and how it will be handled:

```
Step 2 alternative: Player uses `give poultice shaman` instead of
                    `ask shaman poultice`
  Mechanism: mob script onGive handler on mob 74
  Grants same token: 2-poultices
  Text: (shaman acceptance dialogue)
```

Always consider:
- `give <item> <npc>` as alternative to `ask <npc> <topic>` for item delivery
- `ask <npc> quest` / `ask <npc> task` as universal discovery path (SOP)
- Players trying to skip steps (return to quest giver early)

---

**QUEST GATING DIAGRAM**

Show the full chain of `questRequired` / `questExcluded` gates:

```
[no quest] → ask npc quest → [5-start]
[5-start] → get ledger (room script) → [5-ledger, 5-evidence]
[5-evidence] + has item 21 → ask tolva ledger → [5-end] (rewards fire)
```

Verify:
- Every step has exactly one way to be granted (or explicit alternatives)
- No step is unreachable (dead step check)
- `questExcluded` prevents re-triggering completed steps
- The `end` step fires rewards via the quest YAML

---

**FILES NEEDED**

List every file that must be created or modified:

| Action | File | Purpose |
|--------|------|---------|
| CREATE | `quests/{id}-{name}.yaml` | Quest definition with steps and rewards |
| CREATE | `dialogue/{zone}/{mobid}.yaml` | New NPC dialogue (if NPC has none) |
| MODIFY | `dialogue/{zone}/{mobid}.yaml` | Add quest nodes to existing dialogue |
| CREATE | `rooms/{zone}/{roomid}.js` | Room script for item pickup / command intercept |
| CREATE | `mobs/{zone}/scripts/{id}-{name}.js` | Mob script for onGive handler |
| CREATE | `items/{type}/{id}-{name}.yaml` | New quest item (if needed) |
| MODIFY | `mobs/{zone}/{id}-{name}.yaml` | Mob group/behavior changes |
| MODIFY | `rooms/{zone}/{roomid}.yaml` | Add spawninfo, exits, nouns |

For each file, note:
- Whether it's new or a modification to existing
- What quest-critical content it contains
- Instance saves to check/delete after modification

---

**GOTCHAS CHECKLIST**

The sketch must explicitly verify each of these before being considered
complete:

- [ ] Every `grantsQuest` dialogue node has `"quest"` and `"task"` in triggers
- [ ] Every `grantsQuest` pattern entry has `"quest"` and `"task"` in keywords
- [ ] Item delivery steps have BOTH dialogue path AND `onGive` script path
- [ ] `requiresItem` nodes — confirm item exists, is obtainable, and is in the
      player's inventory at that point (not consumed earlier)
- [ ] Room scripts that give items — confirm the noun appears in room
      description or nouns section so `get <noun>` feels natural
- [ ] Mob groups — quest NPCs are NOT in hostile mob groups (no `warren` on
      the shaman, etc.)
- [ ] No physical item needed? Use narrative delivery (quest token gating
      only) and document why
- [ ] Multi-zone quests — confirm NPCs exist and have spawninfo in their rooms
- [ ] `questExcluded` on completion nodes prevents double-completion
- [ ] Quest YAML `rewards` section is filled out (gold, item, message)
- [ ] Instance saves: list any rooms/mobs that have instance saves to delete
- [ ] Line width: all description text wraps at 80 chars
- [ ] No raw numbers in player-facing text

---

**REWARDS**

```
Gold: {amount}
Item: {item name} (item {id}) — or "none"
Skill: {skill}:{amount} — or "none"
Player message: "{completion text}"
Room message: "{room-visible text}"
```

---

**VERIFICATION PLAN**

Step-by-step in-game test sequence:
1. Restart server
2. Walk to quest giver → `ask <npc> quest` → confirm quest granted
3. For each step: perform the action → confirm quest advances
4. For each alternative path: test that too
5. Try to skip steps (go to completion NPC early) → confirm blocked
6. Complete quest → confirm rewards

---

### Step 5 — Output to file

Write the planning document to:
```
_datafiles/quest_plans/{questid}-{ConvertForFilename(quest_name)}.md
```

Create the `_datafiles/quest_plans/` directory if it does not exist.

Remind the user:

> This is a planning document only — no game files have been written.
>
> Review and annotate the plan, then run:
> `/new-quest {questid}-{ConvertForFilename(quest_name)}.md`
> to generate all files.

---

## Usage

```
/sketch-quest "delivery quest where player carries a report from Harn at Watchers Crossing to a clerk in Thornwall"
/sketch-quest "combat quest in the Labyrinth tunnels, player must defeat the warren chieftain"
/sketch-quest "investigation quest at the Crossing Inn, player finds proof the toll is unauthorized"
```

Arguments are a freeform quest concept. Include: quest type, involved NPCs,
zones, and the general objective. If vague, the command will ask for details.
