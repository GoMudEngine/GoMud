# /sketch-quest

Plan a new quest for DOGMud. Produces a quest design document for human review. **Does not write any game files.** Outputs to a markdown file.

## Instructions

**Phase 2 only.** Per the Zone-Building SOP in
`docs/guides/CONTENT_GENERATION_GUIDE.md` Section 2, quests are built AFTER
the zone smoke-test checklist passes. If the zone for this quest
hasn't been smoke-tested:

- Stop and finish the smoke checklist first.
- If the zone is older and the checklist was never formally run,
  walk through it now anyway. Quest issues we've seen historically
  trace back to layout/balance problems that smoke would have
  caught.

If the smoke is genuinely done, proceed.

You are planning a new quest for the DOGMud MUD. This is a design document — output only, no game files written.

### Step 1 — Load context

Read these files before generating anything:
1. `docs/world.md` — lore, zones, NPCs, tone
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
- Is this a branching/opposed quest? If yes, identify:
  - The branch point (which step forces a choice)
  - The branch NPC(s) (one per path)
  - What flag key/values to use (e.g., `branch: [sylara, rhett]`)
  - What followup quests each path unlocks

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

Output a structured planning document with the following sections.

**Before writing the step chain**, work through Step 4a (the player POV
walkthrough) — it is the single most important guard against unguessable
quests. The most common quest failure mode is writing a beautiful lore
puzzle that requires the player to type a magic word they can't possibly
derive. Step 4a forces you to verify each step is actually playable by
someone who only has the game's normal output to go on.

---

**QUEST: {Quest Name}**
**Quest ID:** {N}
**Zone(s):** {zone list}
**Type:** {fetch/delivery/investigation/combat/multi-zone}
**Quest giver:** {NPC name} (mob {id}, room {id}, zone)
**Completion NPC:** {NPC name} (mob {id}, room {id}, zone)

---

**STEP 4A — PLAYER POV WALKTHROUGH** (MANDATORY)

For each step in the chain below, write a 2-line entry from the
player's keyboard:

```
Step N: {step name}
  Player thinks: "{what the player has just been told or shown}"
  Player types:  "{the literal command the player would naturally type}"
  Discovers via: "{where in the game the player learned to type this —
                   NPC dialogue, room description, hint text, item name,
                   or universal MUD intuition}"
```

If you can't fill in "Discovers via:" with a specific in-game source
the player has already encountered by this point in the quest, the
step is broken. Redesign before continuing.

**TRIGGER MECHANICS — RANKED BY PLAYER-DISCOVERABILITY**

Use the highest-discoverability mechanic that fits the design. Default
to the top of this list and only drop down for genuine reasons.

| Tier | Mechanic | Why discoverable |
|------|----------|------------------|
| ★★★★ | `ask <npc> quest` (or `task`) | Universal SOP; players try this on every NPC |
| ★★★★ | `give <item-just-received> <npc-just-talked-to>` | Players naturally hand off items they were just given |
| ★★★★ | `attack <obvious hostile mob>` / mob_death of named target | Players fight what's in the room |
| ★★★ | `<cardinal direction>` to a clearly described room | Players walk through visible exits |
| ★★★ | `search` (verb only, no noun) in a room with hidden_nouns | Players try `search` in suspicious rooms; trigger on `room_interact` with `verb: search` (any noun) gated by room and quest tokens |
| ★★★ | `look <noun in room description>` (regular noun, ANSI-tagged in description) | Players look at things highlighted in room text |
| ★★ | `look <noun discovered via search>` IF hidden_description includes a literal `look <key>` hint AND the key is a single intuitive word | Players follow explicit hints |
| ★ | `look <noun>` where the noun key is multi-word, OR is a verbose phrase | Players have to type the exact magic phrase; brittle. Mitigate with multiple trigger entries (one per plausible phrasing) |
| ☆ | Anything requiring the player to derive a keyword from out-of-band knowledge | DON'T USE — this is the unguessable-magic-word trap |

**Concrete preferences for hidden_noun discovery:**
- The cleanest pattern is `verb: search` + `room: <id>` + `missing: <token>`
  conditions on the trigger — fires when the player searches the room
  successfully (no specific noun required). The send_text can describe
  what they find in narrative prose.
- If the trigger MUST be specific to a noun (e.g., the room has multiple
  hidden_nouns and only one advances the quest), make the hidden_noun key
  a single intuitive word (e.g., `carving`, `marker`, `disturbance` —
  NOT `bench-vise carving` or `beneath Elgar's marker`) AND embed
  ``Try `look <key>`.`` in the hidden_description so the player knows
  exactly what to type after `search` succeeds.
- For ANY `room_interact` trigger keyed on a multi-word noun, write
  multiple trigger entries — one per plausible noun phrasing the player
  might type (`altar stone`, `altar`, `stone`). All entries share the
  same conditions and actions; the `missing: <token>` condition prevents
  re-fire. The quest engine matches the noun field exactly against
  `strings.ToLower(rest)`; it does NOT use the room's noun aliasing.

**The thousand-mudder test:**
For each step, ask: "Out of 1000 random mudders playing through this
quest, how many would advance past this step without external help?" If
the honest answer is less than ~700, the step is broken. Hard puzzles
are fine; unguessable keywords are not.

---

**STEP CHAIN**

For each step, specify:
- Step ID (e.g. `start`, `investigate`, `evidence`, `end`)
- What triggers advancement to this step
- Trigger mechanism: dialogue node (`grantsQuest`), room behavior tree
  (`room_command`), quest engine `item_give` trigger, behavior tree
  `player_give` handler, or automatic (e.g. entering a room)
- Quest token: `{questid}-{stepid}`
- Player-facing description and hint
- **Map destination** — see below. Every step gets one, even if the answer
  is "deliberately none".

---

**MAP DESTINATION (do not skip this)**

If a step sends the player somewhere, the minimap must point the way. A step
with no destination decision is a step where a lost player has nothing to
follow. Pick exactly one:

| Situation | What to write |
|---|---|
| Go see a **named NPC** (deliver, report, ask) | `map_target_mob: <mobid>` **plus** `map_target: <their usual room>` |
| Go to a **place or fixture** (a room, a chest, a carving, an item's spawn room) | `map_target: <roomid>` |
| Done wherever the player stands (type a command, cast during any fight) | `map_target: -1` |
| Player chooses between two+ destinations | `map_target: -1`, and say why in a comment |

**Why `map_target_mob` for NPCs:** most named NPCs carry a `schedule_id` and
move during the day. A fixed room is therefore *wrong for part of every day* —
a tavern keeper who serves until 22:00 and then sleeps upstairs will have the
marker pointing at an empty common room all night. `map_target_mob` resolves to
wherever that NPC currently is. Always pair it with a `map_target` fallback for
when the NPC is dead or not yet spawned; the validator warns if you don't.

**⚠ Do NOT build quest objectives around generic mobs.**
`map_target_mob` needs to name *one* creature. If the template has several live
instances — "a marsh adder", "a bandit scout", "a dock rat" — the engine cannot
know which one you meant, so it declines and the marker silently falls back or
disappears. That is a symptom of a deeper authoring problem: a quest step whose
target is an interchangeable generic mob is also a step the player cannot be
guided to, cannot reliably find, and cannot tell they have finished.

So when a quest needs a specific creature or contact:
- Give it a **unique name** ("Torvan Cresk", not "a smuggler") and a **single
  spawn room**, and target it with `map_target_mob`.
- If it genuinely must be a generic mob (clear N of a roaming type), do **not**
  use `map_target_mob`. Point `map_target` at the sub-area's entrance or anchor
  room so the player is at least sent to the right region, and say so in a
  comment.
- Never point `map_target_mob` at a **hunt target**. Live-tracking a monster
  turns a hunt into a guided kill — that is a gameplay change, not guidance.

**Cross-zone targets currently render NOTHING.** The marker needs the room to be
on the player's current-zone map, and the next-step arrow is computed from the
start room's zone. If a step's destination is in a different zone from where the
player will be standing, expect no marker and no arrow until cross-zone stitching
lands. Prefer keeping a step's destination inside one zone; if you cannot, put
walking directions in the `hint` text as the fallback.

The engine's resolution order is: `map_target: -1` (hard off) → `map_target_mob`
→ `map_target` → inference from a `room_enter` trigger gated on this step's
token → no marker. See `internal/questengine/map_target.go`.

Format:
```
Step 1: "start" — granted by dialogue node
  Trigger: ask <npc> quest/task → dialogue node `help_quest` with
           grantsQuest: "{id}-start"
  Token: {id}-start
  Description: "..."
  Hint: "..."

Step 2: "evidence" — granted by room behavior tree
  Trigger: `get ledger` in room 421 → room behavior tree room_command
           gives item, grants quest
  Token: {id}-evidence
  Description: "..."
  Hint: "..."
```

---

**BRANCHING / OPPOSED QUEST** (include only if applicable)

If this quest has mutually exclusive paths:

Flag declaration:
```yaml
flags:
  - key: {flagname}
    values: [{path1}, {path2}]
    description: "{what the flag tracks}"
```

Branch NPCs:
- Path A: {NPC name} (mob {id}) — sets `{questid}-{flagname}: {path1}`
- Path B: {NPC name} (mob {id}) — sets `{questid}-{flagname}: {path2}`

Followup quest gating:
- Path A quest: `questFlagRequired: {"{questid}-{flagname}": "{path1}"}`
- Path B quest: `questFlagRequired: {"{questid}-{flagname}": "{path2}"}`

Dismissal nodes needed:
- Path A NPC must dismiss Path B players (and vice versa)
- Place at TOP of nodes list in dialogue YAML

---

**ALTERNATIVE PATHS**

For each step where players might reasonably try a different approach, specify
the alternative and how it will be handled:

```
Step 2 alternative: Player uses `give poultice shaman` instead of
                    `ask shaman poultice`
  Mechanism: quest engine item_give trigger on quest YAML (mob 74)
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
| MODIFY | `quests/{id}-{name}.yaml` | Add item_give triggers for item delivery steps |
| MODIFY | `mobs/{zone}/{id}-{name}.yaml` | Add behavior tree player_give handler for item rejection |
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
- [ ] **Narrative voice:** NPC `text` uses first person ("I", "my"). Hints
      describe player options from the player's perspective. Never write
      3rd-person self-references like "Ask about why she left" — write
      "You could ask why she left" or "You could ask about the marriage."
- [ ] **Trigger discoverability:** every trigger word appears in a hint,
      NPC dialogue, room description, or quest log entry. If the player
      has to guess a keyword, the quest is broken.
- [ ] **Player POV walkthrough complete (Step 4a):** every step has
      `Player thinks / Player types / Discovers via` filled in with a
      concrete in-game source the player has reached by that point. No
      "the player would intuit it" hand-waves.
- [ ] **Trigger mechanic at the right tier:** every step uses the
      highest-discoverability mechanic that fits, per the ranking table
      in Step 4a. Quest-engine `room_interact` triggers prefer
      `verb: search` (any noun) over noun-specific keys. Noun-specific
      `room_interact` triggers ONLY use single-word hidden_noun keys
      with `look <key>` hints in the hidden_description, OR provide
      multiple trigger entries covering plausible phrasings.
- [ ] **Thousand-mudder test:** would 700+ out of 1000 random mudders
      advance past each step without help? If not, redesign — hard
      puzzles are fine, unguessable magic words are not.
- [ ] **Narrator never overreaches:** quest engine `send_text`,
      room descriptions, and noun descriptions stick to what the
      player can directly observe — physical details, things the
      player just did, contents of notes/journals/dialogue the
      player is actually reading. NEVER attribute internal motives
      or thoughts to absent/dead characters ("Elgar knew this symbol
      and felt the need to scratch it"), and NEVER invent details
      not present in the room being described ("a second mark on
      the slab points east" when the slab's noun text mentions no
      such mark). Forward-step hints are fine ("the temple priest
      may know what this means") because they're narrator guidance,
      not character mind-reading. When in doubt, ask: could the
      player observe this with their own eyes right now? If no,
      redesign.
- [ ] **Prefer `questRequired` over `requires`** for quest-gated nodes.
      `requires` depends on memory that can expire. Quest tokens are permanent.
- [ ] **`expiryPeriod` should almost never be set.** Memory expiry bricks
      quests when `requires`-gated nodes become unreachable. Only use
      `expiryPeriod` for quests where urgency is the explicit design
      intent (e.g., "deliver this before the trolls attack"). Default:
      leave empty or omit entirely.
- [ ] Item delivery steps have BOTH dialogue path AND quest YAML `item_give`
      trigger for the quest-accepting NPC
- [ ] **give.go flow & `consume_item` requirement:** `give.go` calls
      the quest engine FIRST, then (if not consumed) transfers the
      item to the mob, then runs the behavior tree `player_give`
      handler. **Every quest engine `item_give` trigger that
      represents a successful handover MUST include `consume_item:
      <itemId>` as one of its actions** — without it, give.go falls
      through to the behavior tree, and the NPC's archetype default
      (`noncombat_questgiver` and `noncombat_shopkeeper` both have a
      "declines politely and hands it back" `player_give` branch)
      fires AFTER the quest already accepted the handover, so the
      player sees a confusing decline AND gets the item bounced
      back. With `consume_item` set, give.go marks the result
      Handled and skips both the transfer and the behavior tree.
- [ ] **`return_item` on the rejection paths:** NPCs that should NOT
      keep a wrong item (e.g., quest giver receiving the wrong item
      type) need a behavior tree `player_give` handler with
      `return_item` action. This is separate from the `consume_item`
      requirement above — they cover the wrong-item-given case where
      no quest engine trigger matches.
- [ ] **Lost item recovery:** Every quest giver who hands out a physical item
      must have a recovery dialogue node (e.g., `lost_report`) that gives a
      replacement copy if the player has the quest but lost the item.
- [ ] **NPC item handoff via dialogue:** Use `givesItem: <itemId>` on dialogue
      tree nodes to hand quest items to the player. This is preferred over JS
      scripts for simple handoffs. The player sees "You receive a <itemname>."
- [ ] `requiresItem` nodes — confirm item exists, is obtainable, and is in the
      player's inventory at that point (not consumed earlier)
- [ ] Room behavior trees that give items — confirm the noun appears in room
      description or nouns section so `get <noun>` feels natural
- [ ] Mob groups — quest NPCs are NOT in hostile mob groups (no `warren` on
      the shaman, etc.)
- [ ] No physical item needed? Use narrative delivery (quest token gating
      only) and document why
- [ ] Multi-zone quests — confirm NPCs exist and have spawninfo in their rooms
- [ ] `questExcluded` on completion nodes prevents double-completion
- [ ] **End-token exclusion:** every `grantsQuest` node excludes BOTH the
      granted token AND `{questid}-end` in `questExcluded` — prevents
      re-offering completed quests
- [ ] Quest YAML `rewards` section is filled out (gold, item, message)
- [ ] Instance saves: list any rooms/mobs that have instance saves to delete
- [ ] Line width: all description text wraps at 80 chars
- [ ] No raw numbers in player-facing text
- [ ] **Branching quests:** If quest has flags, declare them in quest YAML
      with allowed values. Undeclared flags cause startup panic.
- [ ] **Flag-gated nodes:** Followup quest offers use `questFlagRequired`
      to gate on the player's branch choice
- [ ] **Dismissal nodes:** Every branch NPC has a dismissal node at the
      TOP of the nodes list for wrong-path players. Without this, keyword
      patterns fire and players think there's a quest to discover.
- [ ] **Mid-quest variants:** Root variants exist for all cross-NPC visit
      states (wrong-path player visits during Q, after Q, etc.)
- [ ] **Quest items not components:** Delivery items must NOT have
      `is_component: true` — component pouch is not checked by give/requiresItem

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
