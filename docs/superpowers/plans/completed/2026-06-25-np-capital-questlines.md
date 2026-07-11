# New Plymouth Capital Questlines — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the pre-built New Plymouth pre-founding lore web into four new
quests covering the five quest-less capital districts (Crafting, Merchant,
Temple, Noble, Old Quarter).

**Architecture:** Data-only content (quest YAML + dialogue node additions + new
item YAMLs + one new hostile mob), no Go engine changes. Each quest is an
independently completable unit; a soft arc binds them via shared meta-story and a
Q68 allegiance flag that lightly colors the other three's dialogue. All quests
are non-combat except one optional fight (the Q70 Canal Lurker).

**Tech Stack:** GoMud quest engine (`internal/questengine`), YAML data files,
the dialogue engine (`internal/dialogue`), the playtest harness for verification.

**Spec:** `docs/superpowers/specs/2026-06-24-np-capital-questlines-design.md`

---

## Engine reference (verified against the codebase — use these, do not guess)

**Quest YAML** (`internal/questengine/types.go`): top-level `questid`, `name`,
`description`, `secret`, optional `flags:` (list of `{key, values, description}`),
`steps:` (list of `{id, description, hint}`), `rewards:` (tag-less keys:
`gold`, `rep_faction`, `rep_amount`, `playermessage`, `roommessage`, `itemid`,
`skillinfo`), `triggers:` (list).

**Trigger events:** `room_interact` (`room:`, `noun:`), `item_give` (`mob:`,
`item:`), `quest_granted` (`quest_token:`), `mob_death`, `room_enter`,
`command`. Each trigger has `conditions:` and `actions:`.

**Conditions** (`Conditions` struct): `has: []`, `missing: []`, `in_room`,
`has_item`, `missing_item`, `has_flag: {key: value}`, `missing_flag: {key: value}`.

**Actions** (`ActionDef` struct — one field per list entry): `grant:`,
`consume_item:`, `give_item:`, `give_gold:`, `npc_say: {mob, lines:[{delay,text}]}`,
`send_text:`, `room_text:`, `set_flag: {key, value}`, `bump_rep: {faction, delta}`,
`apply_buff`, `teleport`, `give_mutation`, `learn_recipe`, etc.

**Dialogue node fields** (existing examples: `dialogue/new_plymouth_*/93xx.yaml`):
`id`, `triggers: []`, `text`, `hints`, `moodChange`, `questRequired: []`,
`questExcluded: []`, `grantsQuest:`, `givesItem:`, `setsQuestFlag: {key, value}`,
`questFlagRequired: {key: value}`, `questFlagExcluded: {key: value}`.

**Quest item YAML** (`items/materials-40000/40111-case_file.yaml` pattern):
`itemid`, `name`, `namesimple`, `description`, `type` (`object`), `subtype`
(`mundane`), `not_salable: true`, `weight`, `value: 0`.

**CRITICAL SOPs (from prior quest work — non-negotiable):**
1. **Grant nodes FIRST under `tree.nodes`.** Dialogue `TreeAdvance` walks nodes
   in file order and matches by SUBSTRING (`strings.Contains(topic, trigger)`).
   A short lore trigger that is an incidental substring of a topic shadows a
   later gated grant node. Every quest-granting node goes at the TOP of its
   giver's `tree.nodes`, ahead of ungated lore nodes. (This is why `ask ysolde
   source` was broken until reordered — see commit `f72804e2`.)
2. **Re-grant prevention.** Every `grantsQuest: "X-start"` node lists BOTH
   `"X-start"` AND `"X-end"` in `questExcluded`.
3. **Trigger discoverability.** Every grant node includes `"quest"` and `"task"`
   in `triggers`. Every trigger word appears in a hint, NPC line, or quest log.
4. **`grantsQuest` in dialogue does NOT reliably fire the `quest_granted`
   event** (gotcha #9). So: never key a `quest_granted` trigger on a token that a
   DIALOGUE node granted. Branch/auto-complete chaining must happen inside the
   same `room_interact`/`item_give` trigger's `actions:` block (grant + end +
   gold + rep atomically), OR be keyed on a token granted by a quest-YAML
   trigger action.
5. **Reward-block keys are tag-less** (`gold`, not `give_gold`); snake_case
   no-ops there. Trigger actions ARE snake_case (`give_gold`, `bump_rep`).
6. **Voice:** NPC `text`/`npc_say` first person; `hints`/`send_text` narrator
   second person; no hard numbers in player-facing text; wrap prose at ~78 chars.

**Smoke-test SOP:** before every local boot, wipe instance saves:
`rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`
(do NOT touch `shops/`). The server is windowless and safe to run from Bash
(`go run . > /tmp/boot.log 2>&1 &`, watch for `Server Ready`, grep for panics).

**ID allocations (verified free):** quests 68–71; items 40112–40120; mob 9393.

---

## Task 0: Branch + ID sanity

**Files:** none (git + verification only)

- [ ] **Step 1: Create the feature branch off master**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
git checkout master
git checkout -b feature/np-capital-questlines
```

- [ ] **Step 2: Confirm the ID ranges are still free**

```bash
python tools/id_inventory.py --type quests
python tools/id_inventory.py --type items
python tools/id_inventory.py --type mobs
```

Expected: next free quest ≥ 68, next free item ≥ 40112, next free mob ≥ 9393.
If any range is occupied, STOP and report — do not silently shift IDs.

- [ ] **Step 3: Confirm the giver/supporting NPC mob IDs exist**

```bash
for m in 9332 9333 9334 9338 9344 9347 9349 9320 9360 9359 9370 9371 9381; do
  ls _datafiles/world/dogmud/mobs/new_plymouth_*/$m-*.yaml 2>/dev/null
done
```

Expected: all 13 files listed. These are the NPCs the quests hang on (Orin,
Halvard, Vesna, Toby, Horst, Ostry, Vell, Coll, Dross, Ept, Ferrol, Lysha,
Gritta). No commit for this task.

---

## Task 1: Quest items (40112–40120)

**Files (Create):**
- `_datafiles/world/dogmud/items/materials-40000/40112-bench_mark_rubbing.yaml`
- `_datafiles/world/dogmud/items/materials-40000/40113-edvars_map_fragment.yaml`
- `_datafiles/world/dogmud/items/materials-40000/40114-copy_of_edvars_map.yaml`
- `_datafiles/world/dogmud/items/materials-40000/40115-cipher_rubbing.yaml`
- `_datafiles/world/dogmud/items/materials-40000/40116-lyshas_annotated_reading.yaml`
- `_datafiles/world/dogmud/items/materials-40000/40117-sealed_grey_fragment.yaml`
- `_datafiles/world/dogmud/items/materials-40000/40118-lintel_rubbing.yaml`
- `_datafiles/world/dogmud/items/materials-40000/40119-tribute_ledger_page.yaml`
- `_datafiles/world/dogmud/items/weapons-20000_or_existing/40120-ostrys_gratitude.yaml` (see Step 9 for exact path)

> Filename rule: `{itemid}-{ConvertForFilename(name)}.yaml` (lowercase, a-z/0-9
> kept, apostrophes dropped, other chars → underscore). Verify each filename
> matches its `name:` field or the loader panics at startup.

- [ ] **Step 1: Bench-Mark Rubbing (40112)**

```yaml
itemid: 40112
name: Bench-Mark Rubbing
namesimple: rubbing
description: A sheet of thin paper pressed and charcoal-rubbed over a
  worked-bench inscription -- the name "Asha" above the words "Master
  Stave-Hand," and beneath them a second line scratched out before it
  could be finished. The tools it was taken from are clean and freshly
  oiled. Someone is still keeping that cooperage, whatever the shuttered
  door says.
type: object
subtype: mundane
not_salable: true
weight: 0.1
value: 0
```

- [ ] **Step 2: Edvar's Map-Fragment (40113)**

```yaml
itemid: 40113
name: Edvar's Map-Fragment
namesimple: map-fragment
description: A torn quarter of a hand-drawn survey, far older and far more
  exact than any city map sold openly. It charts ground inland of the
  current docks and marks one structure with an eight-pointed figure --
  four concentric rings, eight points at equal intervals -- where the
  official record shows only empty founding ground. The cartographer Edvar
  drew it. Edvar is no longer here.
type: object
subtype: mundane
not_salable: true
weight: 0.1
value: 0
```

- [ ] **Step 3: Copy of Edvar's Map (40114)** — circle-branch reward keepsake

```yaml
itemid: 40114
name: Copy of Edvar's Map
namesimple: map copy
description: A careful copy of Edvar's inland survey, made in Orin's own
  hand and pressed on you for safe-keeping away from the cooperage. The
  eight-pointed mark sits where the founding story insists nothing stood.
  Orin's note in the margin reads: "The original is below, where the canal
  meets the old quarter. Someone should know it is real."
type: object
subtype: mundane
not_salable: true
weight: 0.1
value: 0
```

- [ ] **Step 4: Cipher Rubbing (40115)**

```yaml
itemid: 40115
name: Cipher Rubbing
namesimple: cipher rubbing
description: A rubbing taken from the processional relief figures in the
  Grand Temple nave -- old work, older than the plaque it is meant to
  illustrate. The hand positions of the figures are not devotional. Read in
  sequence they are a numeric cipher, and Scholar Dross believes the number
  is a date.
type: object
subtype: mundane
not_salable: true
weight: 0.1
value: 0
```

- [ ] **Step 5: Lysha's Annotated Reading (40116)**

```yaml
itemid: 40116
name: Lysha's Annotated Reading
namesimple: annotated reading
description: Keeper Lysha's working notes, matching the cipher rubbing to
  the third panel of the gallery's oldest paintings. Her conclusion, in a
  small even hand: the eight-pointed structure stands at a site inland of
  the present docks, where the canal district meets the Old Quarter -- and
  the panel was painted from observation, by someone who had seen it
  standing.
type: object
subtype: mundane
not_salable: true
weight: 0.1
value: 0
```

- [ ] **Step 6: Sealed Grey Fragment (40117)**

```yaml
itemid: 40117
name: Sealed Grey Fragment
namesimple: grey fragment
description: A shard of the oldest, greyest stone, wrapped and sealed by
  Coll the sweeper to carry down to Gritta in the flooded quarter. It is
  smooth and unscratched and oddly heavy, and it holds a kind of pressure,
  as though the material is keeping a shape it was forced into a very long
  time ago and has not fully accepted.
type: object
subtype: mundane
not_salable: true
weight: 0.4
value: 0
```

- [ ] **Step 7: Lintel Rubbing (40118)**

```yaml
itemid: 40118
name: Lintel Rubbing
namesimple: lintel rubbing
description: A large rubbing Gritta took for you from the underside of the
  Buried Lintel: four concentric rings and eight points at equal intervals,
  carved with a precision that required prior drawing, set at head height
  for people meant to look up and read it as they passed beneath. It is the
  original the gallery's oldest painting only copied.
type: object
subtype: mundane
not_salable: true
weight: 0.2
value: 0
```

- [ ] **Step 8: Tribute Ledger Page (40119)**

```yaml
itemid: 40119
name: Tribute Ledger Page
namesimple: ledger page
description: A single page lifted from a tally left in the Gilt Threshold's
  parlor -- quarterly sums against vendor names, every column routed upward
  to the same unnamed account, with a note in the margin: "hold until the
  permit clears." It does not prove a crime. It proves where the money goes,
  and that is its own kind of leverage.
type: object
subtype: mundane
not_salable: true
weight: 0.1
value: 0
```

- [ ] **Step 9: Ostry's Gratitude (40120)** — reskin of an existing blade

First locate the blade to reskin (Halvard/Ostry's steel blade, itemid 40001):

```bash
find _datafiles/world/dogmud/items -name "40001-*.yaml"
```

Copy that file to `40120-ostrys_gratitude.yaml` in the SAME directory, then
change ONLY `itemid` → `40120`, `name` → `Ostry's Gratitude`, `namesimple`,
and `description`. **Keep every combat/stat/type/subtype/slot field identical**
to 40001 (this is a reskin, not a rebalance). Set `not_salable: false` (it's a
real blade the player can keep or sell). New description:

```yaml
description: A well-made working blade, pressed on you by Dame Ostry with the
  flat practicality she brings to everything -- no ceremony, just "you'll get
  more use from it than my thanks." The balance is honest and the edge is
  true. It is the gratitude of a woman who does not waste words or steel.
```

- [ ] **Step 10: Boot-test the items load, then commit**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/boot_items.log 2>&1 &
sleep 30; grep -iE "Server Ready|panic|did not end in" /tmp/boot_items.log | head
# kill the server after confirming clean boot
```

Expected: `Server Ready`, no panic, no `did not end in Filepath()` error.

```bash
git add _datafiles/world/dogmud/items/
git commit -m "feat(np-quests): 9 quest items (40112-40120) for the capital questlines"
```

---

## Task 2: Canal Lurker mob (9393) + spawn

**Files:**
- Create: `_datafiles/world/dogmud/mobs/new_plymouth_old_quarter/9393-a_canal_lurker.yaml`
- Modify: the Old Quarter descent room where it spawns (see Step 3)

- [ ] **Step 1: Confirm the zone display name + folder + a descent room id**

```bash
ls "_datafiles/world/dogmud/mobs/" | grep -i "Old Quarter"   # exact display-name folder
grep -rl "roomid: 603" "_datafiles/world/dogmud/rooms/New Plymouth Old Quarter/" 2>/dev/null | head
```

The zone display name is **New Plymouth Old Quarter** (folder
`new_plymouth_old_quarter` for mobs; `New Plymouth Old Quarter` for rooms). Pick
a descent room on the path to 6037 — **6038 The Deep Canal** is the intended
spawn (it contests the most direct approach to the Buried Lintel 6037). Confirm
6038 exists and read it to learn its current `spawninfo`/exits.

- [ ] **Step 2: Write the Canal Lurker mob YAML**

Model on `mobs/greywater_flats/9195-a_marsh_adder.yaml` (a verified hostile
fighting mob). Capital-newcomer power level (level 1, modest statpool):

```yaml
mobid: 9393
zone: New Plymouth Old Quarter
behavior_archetype: ambusher
aiprofile: serpent
archetype: fighting
hostile: true
statpool: 30
itemdropchance: 20
maxwander: 1
groups:
  - animal
  - aquatic
activitylevel: 10
idlecommands:
  - 'emote breaks the black water with a slow ripple and is gone again before
    the eye can fix it.'
  - ''
  - 'emote drags a pale, segmented length across the canal stone and tastes
    the air with something that is not quite a tongue.'
  - ''
  - 'emote settles below the flood line, only the wet gleam of its back
    showing in the lamplight.'
character:
  name: A Canal Lurker
  description: |
    A pale, eyeless thing the length of a man's arm, bred by generations of
    lightless water into something between an eel and a crayfish -- a
    segmented body, too many short legs folded along its underside, and a
    blunt head that hunts by pressure and smell. It has never needed sight
    and has lost it. It lies along the flood line where the canal narrows,
    and it does not flee warm movement; it closes on it. The bite is not
    venomous. It is simply strong, and the thing is not afraid of you.
  speciesid: 8
  level: 1
  gold: 0
  stats:
    strength:
      training: 6
    dexterity:
      training: 6
    vitality:
      training: 5
    perception:
      training: 4
```

> If `speciesid: 8` (used by the adder) is wrong for an aquatic invertebrate,
> use the same speciesid as another non-humanoid NP creature; verify against an
> existing NP animal mob rather than guessing. The fight must be survivable by a
> capital newcomer — keep statpool ≤ ~32.

- [ ] **Step 3: Add 9393 to room 6038's spawn list**

Read `rooms/New Plymouth Old Quarter/6038-*.yaml`, find its spawn block
(`spawninfo:` list with `mobid:` / `cooldown:` entries — match the existing
format in that file exactly), and add a 9393 entry with a reasonable respawn
cooldown (mirror the cooldown style already used in that room/zone).

- [ ] **Step 4: Boot-test, confirm the Lurker spawns, commit**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/boot_lurker.log 2>&1 &
sleep 30; grep -iE "Server Ready|panic|mobs.LoadDataFiles" /tmp/boot_lurker.log | head
```

Expected: clean boot, mob loadedCount incremented. (Live spawn verification
happens in the Task 7 playtest.)

```bash
git add _datafiles/world/dogmud/mobs/new_plymouth_old_quarter/9393-a_canal_lurker.yaml \
        "_datafiles/world/dogmud/rooms/New Plymouth Old Quarter/"
git commit -m "feat(np-quests): Canal Lurker (9393) hostile mob + spawn for Q70"
```

---

## Task 3: Q68 — The Cooperage Circle (Crafting; branching)

**Files:**
- Create: `_datafiles/world/dogmud/quests/68-the_cooperage_circle.yaml`
- Modify: `_datafiles/world/dogmud/dialogue/new_plymouth_crafting/9332.yaml` (Orin — grant + report-back/branch)
- Modify: `_datafiles/world/dogmud/dialogue/new_plymouth_crafting/9338.yaml` (Toby — hand-off; create if absent)
- Modify: `_datafiles/world/dogmud/dialogue/new_plymouth_merchant/9349.yaml` (Vell — report branch + dismissal; create if absent)
- Modify: a Crafting room (cooperage 5719) for the circle-branch `room_interact`

**Branch architecture (respects gotcha #9):**
- Orin's dialogue grants `68-start` (start) and, at report-back, explains the
  choice but grants NO branch token.
- **Circle branch** = a `room_interact` at the cooperage (quest-YAML trigger,
  reliable): grants `68-end`, `set_flag {68-allegiance: circle}`, `give_gold`,
  `bump_rep` ×2, `give_item 40114`, messages — all atomically.
- **Bloodline branch** = an `item_give` of 40113 to Vell (quest-YAML trigger,
  reliable): grants `68-end`, `set_flag {68-allegiance: bloodline}`,
  `give_gold` (larger), `bump_rep` ×2, messages — atomically.
- Toby's hand-off grants `68-evidence` + gives 40112 and 40113 via dialogue.

- [ ] **Step 1: Write the quest YAML**

```yaml
questid: 68
name: The Cooperage Circle
description: >-
  Orin the bookseller asks the wrong questions about the city's founding,
  and he is not the only one. A shuttered cooperage in the quarter still
  has someone tending its tools. The bloodline means to seize it. What you
  do about that is yours to decide.
secret: false

flags:
  - key: allegiance
    values: [circle, bloodline]
    description: "Whether the player aided the cooperage circle or reported
      it to the bloodline domestic"

steps:
  - id: start
    description: >-
      Orin asked you to visit the shuttered cooperage and see whether anyone
      still tends it -- and to bring back proof. The cooper's lad, Toby, is
      the one to find.
    hint: >-
      Go to the abandoned cooperage in the Crafting Quarter and speak with
      Toby. Bring back something that shows the place is still kept.
  - id: evidence
    description: >-
      Toby keeps the cooperage's tools oiled and its bench-mark clean. He
      gave you a rubbing of it -- and Edvar's hidden map-fragment, the
      pre-founding survey the bloodline would seize the place to bury. Clerk
      Vell has filed to take the cooperage as derelict, and the inspection
      is close.
    hint: >-
      Take what Toby gave you back to Orin and decide what to do.
  - id: end
    description: >-
      The matter of the cooperage is settled -- one way, or the other.

rewards:
  playermessage: ""
  roommessage: ""

triggers:
  # ── CIRCLE BRANCH: re-register the cooperage "in use" (room_interact 5719) ──
  - event: room_interact
    room: 5719
    noun: guild-mark
    conditions:
      has: ["68-evidence"]
      missing: ["68-end"]
    actions:
      - set_flag: {key: "68-allegiance", value: "circle"}
      - give_gold: 60
      - bump_rep: {faction: cooperage_circle, delta: 20}
      - bump_rep: {faction: bloodline_domestic, delta: -15}
      - give_item: 40114
      - send_text: >-
          You re-cut the old guild-mark over the door so it reads as a
          working shop again, and you carry Edvar's survey out of the
          cooperage to where the network can keep it. When the inspection
          comes, the place is in use, with a cooper's mark a clerk cannot
          easily dispute. Orin presses a copy of the map on you before you
          go. "The original is below," he says. "Someone outside this should
          know it is real."
      - room_text: "re-cuts a faded guild-mark over a shuttered door until it reads as a working shop again."
      - grant: "68-end"

  # ── guild-mark, before the choice ──
  - event: room_interact
    room: 5719
    noun: guild-mark
    conditions:
      missing: ["68-evidence"]
    actions:
      - send_text: >-
          A cooper's guild-mark is set above the door in faded paint -- a
          barrel and a hammer. Re-cutting it would be a real act with real
          consequences. You do not yet have a reason to.

  # ── BLOODLINE BRANCH: hand Edvar's map-fragment to Clerk Vell (9349) ──
  - event: item_give
    mob: 9349
    item: 40113
    conditions:
      has: ["68-evidence"]
      missing: ["68-end"]
    actions:
      - set_flag: {key: "68-allegiance", value: "bloodline"}
      - give_gold: 100
      - bump_rep: {faction: bloodline_domestic, delta: 20}
      - bump_rep: {faction: cooperage_circle, delta: -15}
      - npc_say:
          mob: 9349
          lines:
            - {delay: 1, text: "Let me see what you've brought me."}
            - {delay: 4, text: "A pre-founding survey, and the address that
                keeps it. This resolves the question of whether the cooperage
                is genuinely derelict. It is not. It is occupied by people
                without standing to occupy it."}
            - {delay: 7, text: "The office is grateful. Discretion is part of
                the arrangement, and the office rewards it. The matter is
                closed."}
      - send_text: >-
          The cooperage will be seized within the quarter. Toby will be put
          out of the only place he keeps. You have what the office pays for
          discretion, and the weight of having earned it.
      - grant: "68-end"
```

- [ ] **Step 2: Add Orin's grant node FIRST in his `tree.nodes`**

Read `dialogue/new_plymouth_crafting/9332.yaml`. Insert this as the FIRST node
under `tree.nodes` (ahead of his existing `old_questions` lore node):

```yaml
    - id: q68_start
      triggers: ["quest", "task", "cooperage", "circle", "help", "old",
                 "questions", "truth", "edvar", "join", "do"]
      questExcluded: ["68-start", "68-end"]
      grantsQuest: "68-start"
      text: "You keep asking the kind of question that closed the cooperage.
        I will not pretend I do not have an answer. But I do not hand answers
        to strangers. There is a shuttered cooper's shop two streets from
        here -- shut by order, years ago. Someone still tends the tools
        inside. Go and see for yourself, and find the lad, Toby. Bring me
        proof the place is still kept, and we will talk about what it is you
        are really asking."
      hints: "Orin will not say more until you have proven you can be trusted.
        He wants you to visit the shuttered cooperage, find Toby, and bring
        back proof someone still keeps the place. You could ask about the
        quest or the task to be sure of what he wants."
```

Add a second Orin node, AFTER the grant node but still ahead of lore, for the
report-back / choice (gated to the in-progress quest):

```yaml
    - id: q68_choice
      triggers: ["report", "back", "proof", "toby", "rubbing", "map",
                 "edvar", "decide", "choice", "seize", "vell", "circle"]
      questRequired: ["68-evidence"]
      questExcluded: ["68-end"]
      text: "So it is true -- the tools are kept, the mark is clean, and the
        circle did not all leave. Then you know what we are. We keep what the
        founding story leaves out, and we pay for it quietly. Clerk Vell has
        filed to take the cooperage as derelict; the inspection is days away.
        You can help me hold it -- re-cut the guild-mark over the door so it
        reads as a working shop, and I will move Edvar's survey somewhere
        safe. Or you can take what you found to Vell, and be paid for it.
        That is a real choice, and it is yours."
      hints: "Orin lays out the choice plainly. To aid the circle, return to
        the cooperage and re-cut the guild-mark over its door. To side with
        the bloodline instead, take Edvar's map-fragment to Clerk Vell in the
        Merchant Quarter's permit office. Either way, decide for yourself."
      moodChange: cautious
```

- [ ] **Step 3: Add Toby's hand-off node (9338)**

If `dialogue/new_plymouth_crafting/9338.yaml` does not exist, create it with the
standard `mobid`/`zone`/`greetings`/`patterns`/`tree.root` skeleton (model on
9332.yaml), then add this as the FIRST `tree.nodes` entry. Toby gives BOTH items
and grants `68-evidence`:

```yaml
    - id: q68_toby
      triggers: ["quest", "task", "cooperage", "tools", "circle", "orin",
                 "kept", "tend", "proof", "rubbing", "edvar", "map", "who"]
      questRequired: ["68-start"]
      questExcluded: ["68-evidence", "68-end"]
      grantsQuest: "68-evidence"
      givesItem: 40112
      text: "You came from Orin, then. Yes -- I keep them. Somebody has to.
        The day they shut this place they did not take the tools, and a tool
        left to rust is a thing I cannot watch happen. So I oil them, and I
        keep the mark clean, and I sleep where the master slept. There is a
        rubbing of the bench-mark -- take it, it is proof enough. And take
        this too: Edvar's last survey, the one they would pull this whole
        shop down to bury. Vell has filed to call us derelict. There is not
        much time. Orin will know what to do with it."
      hints: "Toby tends the cooperage alone and gave you a rubbing of its
        bench-mark as proof. He also pressed Edvar's hidden map-fragment on
        you -- the bloodline means to seize the place to bury it. Take both
        back to Orin."
      moodChange: cautious
```

> A dialogue node reliably gives only its single `givesItem` (here 40112). The
> second item (40113, Edvar's map-fragment) is delivered by a `room_interact` in
> the cooperage, gated on the `68-evidence` token Toby's node grants — reliable
> and gotcha-#9-safe (it does not depend on a `quest_granted` event from a
> dialogue grant). Add this trigger to the quest YAML (the Task 3 Step 1 file):

```yaml
  # ── Toby's other gift: Edvar's map-fragment, taken in the cooperage ──
  - event: room_interact
    room: 5719
    noun: floorboard
    conditions:
      has: ["68-evidence"]
      missing_item: 40113
      missing: ["68-end"]
    actions:
      - give_item: 40113
      - send_text: >-
          Toby lifts a floorboard near the cot and works free a flat oilcloth
          bundle -- Edvar's survey, the pre-founding map he has been keeping
          under the master's old bed. He presses it into your hands. "Get it
          to Orin," he says. "Or to whoever you decide. Just get it out of
          here before the inspection."
```

> This makes 40113 acquisition explicit and reliable: the player gets 40112 from
> Toby's dialogue (granting `68-evidence`), then `interact floorboard` in 5719
> yields 40113. Update Toby's `hints` to mention checking under the cot/floor.
> Confirm `floorboard` (or the actual noun in 5719's room description) — read
> 5719's description and use a noun that appears in it, or add a sentence
> mentioning a loose floorboard near the cot.

- [ ] **Step 4: Add Clerk Vell's report-branch + dismissal nodes (9349)**

If `dialogue/new_plymouth_merchant/9349.yaml` is absent, create the skeleton.
Vell does NOT grant a quest (the bloodline branch resolves via the `item_give
40113` quest trigger from Step 1). Add, as the FIRST tree node, a **dismissal
node** so keyword pokes don't imply a hidden quest, and a quest-aware node:

```yaml
    - id: q68_vell_dismiss
      triggers: ["cooperage", "circle", "orin", "edvar", "survey", "derelict",
                 "report", "founding", "map"]
      questExcluded: ["68-start", "68-evidence", "68-end"]
      text: "The cooperage on the craft streets? It is a derelict-property
        matter, filed and pending. I do not discuss filings with the public,
        and I would not know what a survey or a founding has to do with a
        question of occupancy. If you have business at this office, state the
        form you need."
      hints: "Clerk Vell will not discuss the cooperage filing with someone
        who has no standing in it. He processes permits and tribute, nothing
        more."

    - id: q68_vell_report
      triggers: ["cooperage", "edvar", "survey", "report", "derelict",
                 "occupied", "proof", "map", "deliver"]
      questRequired: ["68-evidence"]
      questExcluded: ["68-end"]
      text: "You have something pertaining to the cooperage filing? Then it is
        no longer idle talk -- it is a document. Hand it across the desk and I
        will examine it. The office is attentive to citizens who assist it."
      hints: "Clerk Vell will take Edvar's map-fragment if you choose to side
        with the bloodline. Give the map-fragment to Vell to report the
        circle. (Use: give map-fragment to vell.)"
```

- [ ] **Step 5: Boot-test, then commit**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/boot_q68.log 2>&1 &
sleep 30; grep -iE "Server Ready|panic|undeclared flag|quests.LoadDataFiles|grantsQuest" /tmp/boot_q68.log | head
```

Expected: clean boot, quests loadedCount incremented, NO "undeclared flag" panic
(the `68-allegiance` flag is declared), no grantsQuest exclusion warnings.

```bash
git add _datafiles/world/dogmud/quests/68-the_cooperage_circle.yaml \
        _datafiles/world/dogmud/dialogue/new_plymouth_crafting/9332.yaml \
        _datafiles/world/dogmud/dialogue/new_plymouth_crafting/9338.yaml \
        _datafiles/world/dogmud/dialogue/new_plymouth_merchant/9349.yaml \
        "_datafiles/world/dogmud/rooms/New Plymouth Crafting/"
git commit -m "feat(np-quests): Q68 The Cooperage Circle (branching: circle vs bloodline)"
```

---

## Task 4: Q69 — The Gallery Cipher (Temple → Noble; linear)

**Files:**
- Create: `_datafiles/world/dogmud/quests/69-the_gallery_cipher.yaml`
- Modify: `dialogue/new_plymouth_temple/9360.yaml` (Dross — grant + complete)
- Modify: `dialogue/new_plymouth_noble/9370.yaml` (Ferrol — steer/warn node)
- Modify: `dialogue/new_plymouth_noble/9371.yaml` (Lysha — reading + allegiance flavor)
- Modify: Grand Temple nave room 5901 (`room_interact` reliefs)

**Flow:** Dross grants `69-start` → player `interact reliefs` in 5901 grants
`69-rubbing` + gives 40115 → back to Dross (dialogue node, `questRequired
69-rubbing`, grants `69-gallery`) → Ferrol steers (no grant) → Lysha gives 40116
+ grants `69-reading` (dialogue givesItem + grantsQuest) → give 40116 to Dross
(`item_give` trigger) grants `69-complete` → auto `quest_granted` → `69-end`.

- [ ] **Step 1: Write the quest YAML**

```yaml
questid: 69
name: The Gallery Cipher
description: >-
  Scholar Dross can prove the Grand Temple's eastern inscription predates
  the city's official founding -- if he can read the cipher hidden in the
  processional reliefs. The key is in the Noble Quarter gallery, behind a
  guide who steers visitors away from it.
secret: false

steps:
  - id: start
    description: >-
      Dross needs a rubbing of the hand-position cipher worked into the
      processional relief figures in the Grand Temple nave.
    hint: >-
      In the Grand Temple sanctuary, make a rubbing of the processional
      relief figures, then bring it to Dross.
  - id: rubbing
    description: >-
      You took the cipher rubbing from the reliefs. Dross can read part of
      it, but the key to the full sequence is in the Noble Quarter gallery.
    hint: >-
      Take the rubbing to Dross, then go to the Noble Quarter art gallery
      and find the key to the cipher.
  - id: gallery
    description: >-
      Dross sent you to the gallery. The guide, Ferrol, steers visitors to
      the bloodline portraits and away from the older paintings -- and away
      from the keeper upstairs.
    hint: >-
      In the gallery, get past Ferrol's tour to Keeper Lysha upstairs. Ask
      her about the third panel from the left.
  - id: reading
    description: >-
      Keeper Lysha read the third panel against your rubbing: the
      eight-pointed structure stands at a site inland of the present docks.
      She gave you her annotated reading.
    hint: >-
      Bring Lysha's annotated reading back to Scholar Dross at the Temple.
  - id: end
    description: >-
      Dross has his proof: the founding the city celebrates is not the
      first, and the date is wrong by something close to a century. The
      truth is provable. Whether anyone with power will hear it is another
      matter.

rewards:
  gold: 70
  rep_faction: temple_np
  rep_amount: 10
  playermessage: >-
    Dross lays the rubbing and the reading side by side and goes very
    quiet, which for him is remarkable. "It holds," he says. "Every line of
    it holds." He has a proof now that no honest examiner could dismiss --
    and he is, you both understand, a man no one with power has ever been
    obliged to honestly examine. Still. It is known now, and you are one of
    the few who know it.
  roommessage: >-
    The traveller has the look of someone who has read something that cannot
    be un-read.

triggers:
  # ── RUBBING: make a rubbing of the processional reliefs (5901) ──
  - event: room_interact
    room: 5901
    noun: reliefs
    conditions:
      has: ["69-start"]
      missing: ["69-rubbing"]
    actions:
      - grant: "69-rubbing"
      - give_item: 40115
      - send_text: >-
          You press paper to the processional relief and work charcoal
          across it. The figures come up under your hand -- and the longer
          you look, the less devotional they seem. The hands are positioned.
          Each one holds a posture too specific to be art. Read along the
          row, they are a sequence: a number, set into stone by someone who
          wanted it kept where it could not be edited out.
      - room_text: "takes a careful charcoal rubbing of the old processional reliefs."

  # ── reliefs, no quest ──
  - event: room_interact
    room: 5901
    noun: reliefs
    conditions:
      missing: ["69-start"]
    actions:
      - send_text: >-
          The processional figures are very old -- older, somehow, than the
          plaque that names them. Their hands are oddly placed. Without a
          reason to study them you move on.

  # ── COMPLETE: give Lysha's annotated reading to Dross (9360) ──
  - event: item_give
    mob: 9360
    item: 40116
    conditions:
      has: ["69-reading"]
      missing: ["69-complete"]
    actions:
      - grant: "69-complete"
      - npc_say:
          mob: 9360
          lines:
            - {delay: 1, text: "You reached her. You actually reached her."}
            - {delay: 4, text: "The third panel. Painted from observation, of
                a structure that stands inland of the docks -- exactly where
                the cipher's geometry points. The rubbing and the panel agree.
                They were made by the same hand, or the same workshop, and
                they agree."}
            - {delay: 8, text: "Then it is settled, as far as evidence settles
                anything. The inscription predates the founding. The founding
                date is wrong. And everything the bloodline holds in law
                derives from a date that is wrong. ... You should know what you
                have helped prove. And you should be careful who you prove it
                to."}

  # ── AUTO-COMPLETE: complete → end ──
  - event: quest_granted
    quest_token: "69-complete"
    conditions:
      missing: ["69-end"]
    actions:
      - bump_rep: {faction: cooperage_circle, delta: 5}
      - grant: "69-end"
```

- [ ] **Step 2: Add Dross's grant + receive nodes FIRST (9360)**

Read `dialogue/new_plymouth_temple/9360.yaml`. Insert as the FIRST two
`tree.nodes`:

```yaml
    - id: q69_start
      triggers: ["quest", "task", "cipher", "inscription", "founding",
                 "reliefs", "help", "date", "proof", "gallery", "do"]
      questExcluded: ["69-start", "69-end"]
      grantsQuest: "69-start"
      text: "You will listen? Good -- most do not. The eastern inscription is
        older than the founding it supposedly records; I have the oxidation
        data and the stone comparison. And the processional reliefs in the
        nave encode the real date in the hand positions of the figures. I
        cannot prove the full sequence without the key, and the key is a set
        of relief fragments in the Noble Quarter gallery -- where my formal
        inquiry has gone unanswered for a year. But you are not a formal
        inquiry. Make me a rubbing of those reliefs, here in the nave, and
        bring it to me. That is where it starts."
      hints: "Scholar Dross wants a rubbing of the processional reliefs in
        the Grand Temple nave. Make the rubbing in the sanctuary, then bring
        it to him. You could ask about the quest, the cipher, or the
        inscription."

    - id: q69_gallery
      triggers: ["rubbing", "cipher", "read", "gallery", "key", "next",
                 "noble", "lysha", "panel", "fragments"]
      questRequired: ["69-rubbing"]
      questExcluded: ["69-gallery", "69-end"]
      grantsQuest: "69-gallery"
      text: "Yes -- yes, this is it, this is the sequence, but it runs past
        what I can anchor. I need the gallery's fragments to fix the zero
        point. Go to the Noble Quarter gallery. There is a guide there,
        Ferrol, who will walk you past the portraits and tell you the older
        panels are decorative and of uncertain provenance. They are not. Get
        past him to the keeper upstairs -- her name is Lysha -- and ask her,
        in those words, about the third panel from the left. She will know
        what you mean."
      hints: "Dross read part of the cipher but needs the gallery's key. Go to
        the Noble Quarter art gallery, get past the guide Ferrol to Keeper
        Lysha upstairs, and ask her about the third panel from the left."
      moodChange: friendly
```

- [ ] **Step 3: Add Ferrol's steer/warn node FIRST (9370)** — no grant

Read `dialogue/new_plymouth_noble/9370.yaml`. Insert as the FIRST tree node.
Ferrol enforces the official line but, when pressed, names Lysha (the
discoverable hint) by warning the player off:

```yaml
    - id: q69_ferrol_steer
      triggers: ["panel", "third", "older", "paintings", "lysha", "keeper",
                 "cipher", "gallery", "founding", "provenance", "settlement",
                 "upstairs"]
      text: "The older panels? Decorative pieces, uncertain provenance, not
        really part of the curated collection -- I usually steer visitors
        toward the bloodline portraits, which are the genuine artistry here.
        The founding history is thoroughly documented; there is no shortage
        of evidence for it. If you insist on the older work, that is not my
        department. There is a keeper upstairs -- Lysha. She will talk to you
        about what the gallery actually contains. I would not necessarily
        recommend asking her about the third panel from the left."
      hints: "Ferrol steers you firmly toward the bloodline portraits and the
        official founding story -- but in warning you off, he names a keeper
        upstairs, Lysha, and the third panel from the left. That is plainly
        where to look. Go up and ask Lysha about the third panel."
      moodChange: cautious
```

- [ ] **Step 4: Add Lysha's reading node FIRST (9371)** — gives 40116, grants reading

Read `dialogue/new_plymouth_noble/9371.yaml`. Insert as the FIRST tree node.
Include allegiance-flavor variants (warmer if circle) as SEPARATE optional nodes
placed AFTER the grant node so they don't shadow it:

```yaml
    - id: q69_lysha
      triggers: ["quest", "task", "third", "panel", "cipher", "rubbing",
                 "dross", "founding", "settlement", "symbol", "read", "left"]
      questRequired: ["69-gallery"]
      questExcluded: ["69-reading", "69-end"]
      grantsQuest: "69-reading"
      givesItem: 40116
      text: "The third panel from the left. So someone finally came up the
        stairs to ask. Give me the rubbing. ... Yes. This is the same hand as
        the panel, or the same workshop -- the eight-pointed device, the
        ringed geometry, identical. The painter worked from observation; the
        landscape is accurate, and so is the structure. It stands inland of
        the present docks, where the canal district meets the edge of the Old
        Quarter. The gallery's lighting was not built to make this panel easy
        to examine. Someone decided what this room was for. Here -- my reading,
        written out. Take it to Dross. He will know what it completes."
      hints: "Keeper Lysha matched your rubbing to the third panel: the
        eight-pointed structure stood inland of the docks, before the
        founding. She gave you her annotated reading. Take it back to Scholar
        Dross at the Temple."
      moodChange: cautious

    - id: q69_lysha_circle_aside
      triggers: ["circle", "cooperage", "orin", "trust", "side"]
      questRequired: ["69-gallery"]
      questFlagRequired: {"68-allegiance": "circle"}
      text: "Orin vouches for you -- quietly, the way we do anything. Then I
        will say plainly what I would not say to a stranger: I am of the
        circle, here, inside the bloodline's own gallery, reading the thing
        they hung the lights to hide. Be careful with what you now carry."
      hints: "Lysha, knowing you stood with the circle, trusts you with her
        own part in it. The work is the same -- take her reading to Dross."
```

- [ ] **Step 5: Boot-test, then commit**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/boot_q69.log 2>&1 &
sleep 30; grep -iE "Server Ready|panic|quests.LoadDataFiles|grantsQuest" /tmp/boot_q69.log | head
```

Expected: clean boot, quests loadedCount incremented.

```bash
git add _datafiles/world/dogmud/quests/69-the_gallery_cipher.yaml \
        _datafiles/world/dogmud/dialogue/new_plymouth_temple/9360.yaml \
        _datafiles/world/dogmud/dialogue/new_plymouth_noble/9370.yaml \
        _datafiles/world/dogmud/dialogue/new_plymouth_noble/9371.yaml \
        "_datafiles/world/dogmud/rooms/New Plymouth Temple/"
git commit -m "feat(np-quests): Q69 The Gallery Cipher (Temple->Noble investigation)"
```

---

## Task 5: Q70 — The Pre-Founding Web (Old Quarter; linear + fight)

**Files:**
- Create: `_datafiles/world/dogmud/quests/70-the_pre_founding_web.yaml`
- Modify: `dialogue/new_plymouth_common/9320.yaml` (Coll — grant + return)
- Modify: `dialogue/new_plymouth_old_quarter/9381.yaml` (Gritta — lintel hand-off; create if absent)
- Modify: Old Quarter rooms 6037 (lintel) and 6038 (grey seam) for `room_interact`

**Flow:** Coll grants `70-start` + gives 40117 → descend (Canal Lurker contests
the path; not a quest trigger, just terrain) → give 40117 to Gritta (`item_give`
trigger) grants `70-lintel` + gives 40118 + Gritta's lines → `interact lintel`
in 6037 is flavor → give 40118 to Coll (`item_give` trigger) grants `70-return`
→ auto `quest_granted` → `70-end`.

- [ ] **Step 1: Write the quest YAML**

```yaml
questid: 70
name: The Pre-Founding Web
description: >-
  Coll the sweeper has spent years routing fragments of the oldest stone
  up out of the flooded quarter to people who can read them. He wants
  someone from outside the network to go down, carry a fragment to its
  source, and see with their own eyes the thing the founding story leaves
  out.
secret: false

steps:
  - id: start
    description: >-
      Coll gave you a sealed grey fragment to carry down to Gritta in the
      flooded Old Quarter, and asked you to see the Buried Lintel yourself.
    hint: >-
      Take the sealed grey fragment down into the Old Quarter canals and
      find Gritta. Mind the water -- it is not empty.
  - id: lintel
    description: >-
      You found Gritta at the Buried Lintel: gray stone worked and carved
      with the eight-pointed symbol, set before the colony ever arrived.
      She gave you a rubbing of it -- the original the gallery's painting
      only copied.
    hint: >-
      Examine the Buried Lintel, then carry Gritta's rubbing back up to
      Coll in the Common Quarter.
  - id: end
    description: >-
      The web is closed: the discovery below, the cipher at the Temple, the
      panel in the gallery -- all of it agrees. The colony arrived, found
      this, and wrote itself into the record as first. The stone simply goes
      on being older.

rewards:
  gold: 70
  rep_faction: np_commonfolk
  rep_amount: 10
  playermessage: >-
    Coll turns the rubbing toward the light from the well and studies it for
    a long moment, the way a man reads a thing he already knew but needed to
    see anyway. "So you stood under it," he says. "Good. Now there is one
    more who knows, who is not one of us, and was not told -- you saw it.
    That matters more than you think, for the day it stops being only ours
    to carry." He goes back to his broom. You are, without anything being
    said, part of the quiet now.
  roommessage: >-
    The old sweeper gives the traveller a slow nod, one that seems to settle
    something between them.

triggers:
  # ── LINTEL: give the sealed grey fragment to Gritta (9381) ──
  - event: item_give
    mob: 9381
    item: 40117
    conditions:
      has: ["70-start"]
      missing: ["70-lintel"]
    actions:
      - grant: "70-lintel"
      - give_item: 40118
      - npc_say:
          mob: 9381
          lines:
            - {delay: 1, text: "From Coll. Yes. He still sends them up, and I
                still send them down to him to read. Come -- look up."}
            - {delay: 4, text: "Gray stone. The oldest layer. Four rings, eight
                points, cut by someone who drew it first and then carved it at
                height, for people to read as they passed beneath. The founding
                story has us arriving to empty ground. This is worked stone,
                and it was here first."}
            - {delay: 8, text: "I cannot read the script. That is Coll's part,
                and Orin's. My part is the finding. Here -- a rubbing. Carry it
                up to Coll. Someone outside should know it is real."}

  # ── LINTEL flavor: examine the Buried Lintel (6037) ──
  - event: room_interact
    room: 6037
    noun: lintel
    conditions:
      has: ["70-start"]
    actions:
      - send_text: >-
          You look up at the underside of the lintel. Four concentric rings,
          eight points at equal intervals, the incision clean and deliberate.
          It was cut to be read by people passing beneath, by lamplight, a
          long time before anyone here admits there were people. The partial
          device in the gallery's oldest painting was a copy of something. You
          are standing beneath the something.

  # ── grey seam flavor: examine the Deep Canal seam (6038) ──
  - event: room_interact
    room: 6038
    noun: seam
    conditions:
      has: ["70-start"]
    actions:
      - send_text: >-
          A seam of the gray material runs along the flood line, smooth and
          unscratched where the canal stone above it is pitted and worn. Water
          has not marked it. Time has not marked it. It simply continues to be
          older than everything around it.

  # ── RETURN: give the lintel rubbing to Coll (9320) ──
  - event: item_give
    mob: 9320
    item: 40118
    conditions:
      has: ["70-lintel"]
      missing: ["70-return"]
    actions:
      - grant: "70-return"

  # ── AUTO-COMPLETE: return → end ──
  - event: quest_granted
    quest_token: "70-return"
    conditions:
      missing: ["70-end"]
    actions:
      - bump_rep: {faction: cooperage_circle, delta: 5}
      - grant: "70-end"
```

- [ ] **Step 2: Add Coll's grant node FIRST (9320)**

Read `dialogue/new_plymouth_common/9320.yaml`. Insert as the FIRST tree node:

```yaml
    - id: q70_start
      triggers: ["quest", "task", "fragment", "stone", "grey", "gray",
                 "gritta", "lintel", "old", "quarter", "founding", "help",
                 "collect", "do"]
      questExcluded: ["70-start", "70-end"]
      grantsQuest: "70-start"
      givesItem: 40117
      text: "You have a careful way of asking, so I will answer carefully. I
        sweep these stones, and I find things in the gutters that are older
        than the city says anything here can be -- gray stone, worked stone.
        I send the best of them down to Gritta in the flooded quarter, and
        she sends up what she finds below. Take this fragment down to her.
        And do more than carry it -- look at what she is standing under, with
        your own eyes. It is time someone outside our little chain saw it.
        Mind the water on the way down. It is not empty."
      hints: "Coll routes fragments of the oldest stone between the streets and
        the flooded Old Quarter. He gave you a sealed grey fragment to carry
        down to Gritta, and asked you to see the Buried Lintel yourself. Watch
        the canal water on the way down."
      moodChange: cautious
```

- [ ] **Step 3: Add Gritta's lintel node (9381)** — the hand-off is the quest trigger

Gritta's giving of 40118 and the lore lines happen in the `item_give 40117`
quest trigger (Step 1), which is reliable. Her DIALOGUE only needs a
quest-aware greeting node so talking to her before/after the hand-off reads
right. Read/create `dialogue/new_plymouth_old_quarter/9381.yaml` and add FIRST:

```yaml
    - id: q70_gritta_hint
      triggers: ["quest", "task", "coll", "fragment", "lintel", "stone",
                 "rubbing", "give", "deliver", "founding"]
      questRequired: ["70-start"]
      questExcluded: ["70-end"]
      text: "You came down from Coll? Then you have something for me, and I
        have something for you in return. Give me the fragment -- put it in
        my hand -- and I will show you what is carved above us."
      hints: "Gritta is waiting for Coll's fragment. Give the sealed grey
        fragment to Gritta. (Use: give fragment to gritta.)"
      moodChange: neutral
```

> Verify the `room_interact` nouns: read 6037 and 6038's descriptions and
> confirm `lintel` and `seam` appear in them (they should, per the build). If a
> noun is absent, add a short sentence to that room's description naming it, or
> change the trigger `noun:` to a word that is present.

- [ ] **Step 4: Boot-test, then commit**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/boot_q70.log 2>&1 &
sleep 30; grep -iE "Server Ready|panic|quests.LoadDataFiles" /tmp/boot_q70.log | head
```

Expected: clean boot, quests loadedCount incremented.

```bash
git add _datafiles/world/dogmud/quests/70-the_pre_founding_web.yaml \
        _datafiles/world/dogmud/dialogue/new_plymouth_common/9320.yaml \
        _datafiles/world/dogmud/dialogue/new_plymouth_old_quarter/9381.yaml \
        "_datafiles/world/dogmud/rooms/New Plymouth Old Quarter/"
git commit -m "feat(np-quests): Q70 The Pre-Founding Web (Old Quarter relic descent)"
```

---

## Task 6: Q71 — Horst / The Tribute (Merchant → Noble; linear)

**Files:**
- Create: `_datafiles/world/dogmud/quests/71-horst_and_the_tribute.yaml`
- Modify: `dialogue/new_plymouth_merchant/9347.yaml` (Ostry — grant + report)
- Modify: `dialogue/new_plymouth_merchant/9349.yaml` (Vell — charter node; file already touched in Task 3)
- Modify: `dialogue/new_plymouth_merchant/9344.yaml` (Horst — apparatus node)
- Modify: Gilt Threshold room 5815 (`room_interact` ledger scrap)

**Flow:** Ostry grants `71-start` → talk Vell (dialogue, `questRequired
71-start`, grants `71-vell`) → reach Horst, talk (dialogue grants `71-horst`) →
`interact scrap` in 5815 (`room_interact`, requires `71-horst`) grants `71-ledger`
+ gives 40119 → give 40119 to Ostry (`item_give` trigger) grants `71-report` →
auto `quest_granted` → `71-end` + reward (gold + 40120 blade + rep).

- [ ] **Step 1: Write the quest YAML**

```yaml
questid: 71
name: The Tribute
description: >-
  Dame Ostry pays the bloodline's quarterly tribute like every vendor in
  the city, but an audit and a held deposit are bleeding her. She wants to
  know what the tribute is for, and whether there is any floor under it at
  all. The answer runs from a permit clerk's desk to a private gate no one
  argues with.
secret: false

steps:
  - id: start
    description: >-
      Ostry is squeezed by Clerk Vell's audit and a held surety deposit.
      She asked you to find out what the tribute pays for and whether it can
      be appealed.
    hint: >-
      Ask Clerk Vell at the Civic Permit Office about the tribute and the
      audit on Ostry's shop.
  - id: vell
    description: >-
      Vell says the rate is set by charter and is not negotiable. The
      charter derives from the founding -- and you may know more about the
      founding than Vell does.
    hint: >-
      The tribute runs upward, past the clerk. Get to Horst at the Gilt
      Threshold and learn where it goes.
  - id: horst
    description: >-
      Horst gatekeeps the bloodline's private business and told you, in so
      many words, that the tribute funnels up and discretion is the whole
      arrangement. A tally was left in the parlor.
    hint: >-
      Look for evidence in the Gilt Threshold parlor of where the tribute
      goes.
  - id: ledger
    description: >-
      You lifted a page of the tribute ledger -- vendor sums all routed to
      one unnamed account, with a note to hold filings until permits clear.
      It does not prove a crime. It proves leverage.
    hint: >-
      Take the ledger page back to Dame Ostry.
  - id: end
    description: >-
      Ostry cannot break the bloodline, and neither can you. But with the
      ledger page and a quiet word, her deposit is released and her permit
      clears. One merchant breathes. The arrangement goes on.

rewards:
  gold: 110
  rep_faction: np_dockfolk
  rep_amount: 10
  itemid: 40120
  playermessage: >-
    Ostry reads the page once, folds it, and puts it somewhere you do not
    see. "I cannot fight them with this," she says, "and I am not fool
    enough to try. But I can let the right person know I have it, and
    tomorrow my deposit comes back and my permit clears, and nobody says why.
    That is how it works down here." She presses a blade into your hands
    before you can decline it. "You will get more use from it than from my
    thanks."
  roommessage: >-
    Dame Ostry clasps the traveller's forearm once, hard and brief, the
    nearest thing she has to ceremony.

triggers:
  # ── LEDGER: lift the tribute ledger page in the Gilt Threshold parlor (5815) ──
  - event: room_interact
    room: 5815
    noun: tally
    conditions:
      has: ["71-horst"]
      missing: ["71-ledger"]
    actions:
      - grant: "71-ledger"
      - give_item: 40119
      - send_text: >-
          A tally has been left on the parlor table among the cup-rings -- a
          page of quarterly sums against vendor names, every column routed up
          to the same unnamed account, with a note in the margin: hold until
          the permit clears. You take the page. It does not prove a crime. It
          proves exactly where the money goes, and who waits on whom.
      - room_text: "palms a page from a tally left on the parlor table."

  # ── tally, before reaching Horst ──
  - event: room_interact
    room: 5815
    noun: tally
    conditions:
      missing: ["71-horst"]
    actions:
      - send_text: >-
          There are papers on the parlor table, but taking anything from this
          room uninvited, without knowing whose it is or what it means, is the
          kind of mistake the Gilt Threshold is built to punish. Not yet.

  # ── REPORT: give the ledger page to Ostry (9347) ──
  - event: item_give
    mob: 9347
    item: 40119
    conditions:
      has: ["71-ledger"]
      missing: ["71-report"]
    actions:
      - grant: "71-report"

  # ── AUTO-COMPLETE: report → end (+ bonus circle rep if founding proven) ──
  - event: quest_granted
    quest_token: "71-report"
    conditions:
      missing: ["71-end"]
    actions:
      - grant: "71-end"

  # ── bonus: cooperage rep if the player has proven the false founding ──
  - event: quest_granted
    quest_token: "71-end"
    conditions:
      has: ["69-end"]
    actions:
      - bump_rep: {faction: cooperage_circle, delta: 5}
```

> Note: the reward block grants gold 110, np_dockfolk +10, and item 40120
> (Ostry's Gratitude blade). The bonus cooperage +5 only fires if the player has
> completed Q69 (proven the false founding) — leverage made real.

- [ ] **Step 2: Add Ostry's grant node FIRST (9347)**

Read `dialogue/new_plymouth_merchant/9347.yaml`. Insert as the FIRST tree node:

```yaml
    - id: q71_start
      triggers: ["quest", "task", "tribute", "audit", "permit", "deposit",
                 "vell", "squeeze", "help", "charter", "tax", "do"]
      questExcluded: ["71-start", "71-end"]
      grantsQuest: "71-start"
      text: "You want honest work or honest talk? Take talk, it is cheaper.
        I pay the tribute like everyone -- a cut of gross, every quarter, to
        the bloodline through Vell's office. Fine. But now there is an audit
        on my books and my surety deposit is held, and between the two I am
        bleeding for no reason anyone will put in writing. I am too practical
        to rage at it. What I want is to understand it. Go and ask Vell what
        the tribute actually pays for, and whether there is any floor under
        it at all. Start there. See how far it goes."
      hints: "Dame Ostry is squeezed by an audit and a held deposit on top of
        the quarterly tribute. She wants you to ask Clerk Vell, at the Civic
        Permit Office, what the tribute pays for and whether it can be
        appealed. You could ask about the quest or the tribute."
      moodChange: neutral
```

- [ ] **Step 3: Add Vell's charter node (9349)** — grants `71-vell`, no item

In `dialogue/new_plymouth_merchant/9349.yaml` (already created/edited in Task 3),
add these nodes AFTER the Q68 nodes but still ahead of generic lore. The
arc-aware variant (player has proven the founding) is a SEPARATE node placed
after the base grant node:

```yaml
    - id: q71_vell
      triggers: ["quest", "task", "tribute", "charter", "audit", "ostry",
                 "deposit", "rate", "appeal", "founding", "pays"]
      questRequired: ["71-start"]
      questExcluded: ["71-vell", "71-end"]
      grantsQuest: "71-vell"
      text: "The tribute rate is set by charter and is not a subject I
        negotiate. It is a legal obligation, not a preference, and I apply it
        uniformly. What it pays for is not a question this office answers; the
        charter was granted by the governing authority and the governing
        authority is its own justification. As for Dame Ostry's audit -- a
        filing is pending and will resolve when it resolves. If you want to
        know where the obligation leads above this desk, that is not a door I
        open. The Gilt Threshold keeps its own gate."
      hints: "Vell will say only that the rate is set by charter -- and the
        charter rests on the founding. The tribute runs upward, past the
        clerk, to the Gilt Threshold and a gatekeeper named Horst. Go there."
      moodChange: cautious

    - id: q71_vell_charter_aside
      triggers: ["charter", "founding", "date", "false", "wrong", "cipher",
                 "lintel"]
      questRequired: ["71-vell", "69-end"]
      text: "You say the founding date is wrong as though it were a small
        thing. If it were true it would not be small at all -- the charter,
        the courts, this office, all of it derives from the founding and its
        date. But I process filings; I do not adjudicate the foundations of
        the state, and I would advise you not to say that sentence aloud in
        the wrong room. ... The obligation still runs upward. The gate is
        still Horst's."
      hints: "Knowing what you know about the founding, Vell's certainty reads
        differently -- and he half-admits it. Either way the trail leads up to
        Horst at the Gilt Threshold."
      moodChange: cautious
```

> This optional arc-aware variant is gated on the all-of `questRequired:
> ["71-vell", "69-end"]` — it only appears once the player has both started the
> tribute quest AND proven the false founding via Q69. It uses only the
> confirmed all-of `questRequired` gate (no uncertain any-of field). The base
> `q71_vell` node is NOT gated on 69, so the quest always works regardless of
> arc order. If you'd also like it to trigger off Q70 alone, add a second
> identical node gated `questRequired: ["71-vell", "70-end"]`.

- [ ] **Step 4: Add Horst's apparatus node FIRST (9344)** — grants `71-horst`

Read `dialogue/new_plymouth_merchant/9344.yaml`. Insert as the FIRST tree node.
Horst stays untouchable and reveals only the shape, pointing at the parlor:

```yaml
    - id: q71_horst
      triggers: ["quest", "task", "tribute", "gate", "bloodline", "ostry",
                 "vell", "money", "where", "arrangement", "discretion",
                 "charter", "parlor"]
      questRequired: ["71-vell"]
      questExcluded: ["71-horst", "71-end"]
      grantsQuest: "71-horst"
      text: "You came up the chain politely, so I will be polite in return,
        and brief. I facilitate meetings. Money, like people, comes to the
        gate first, and I assess whether passing it along serves everyone's
        interests, and then it passes along -- upward, to parties who value
        discretion above almost everything. That is not incidental. It is the
        arrangement, and the arrangement is why the city runs. I will not name
        the account and I will not show you the books. What is left lying in
        the parlor when men talk over a cup is their own carelessness, not my
        disclosure. Mind how you leave."
      hints: "Horst confirms the tribute funnels upward to unnamed parties and
        will show you nothing himself -- but he as good as points at the
        parlor, where a tally has been left out. Look there for the evidence
        Ostry needs."
      moodChange: cautious
```

- [ ] **Step 5: Boot-test, then commit**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/boot_q71.log 2>&1 &
sleep 30; grep -iE "Server Ready|panic|quests.LoadDataFiles" /tmp/boot_q71.log | head
```

Expected: clean boot, quests loadedCount = +4 from baseline (68–71 all load).

```bash
git add _datafiles/world/dogmud/quests/71-horst_and_the_tribute.yaml \
        _datafiles/world/dogmud/dialogue/new_plymouth_merchant/9347.yaml \
        _datafiles/world/dogmud/dialogue/new_plymouth_merchant/9349.yaml \
        _datafiles/world/dogmud/dialogue/new_plymouth_merchant/9344.yaml \
        "_datafiles/world/dogmud/rooms/New Plymouth Merchant/"
git commit -m "feat(np-quests): Q71 The Tribute (Merchant->Noble; Ostry vs the bloodline)"
```

---

## Task 7: Full harness playtest + discoverability + fixes

**Files:** fixes as needed across Tasks 3–6 files; a report under
`tools/playtest/reports/`.

The four quests touch a lot of dialogue; the recurring failure mode is the
substring/node-order shadow (gotcha #1). Playtest each quest end-to-end via the
harness, driving the giver with the NATURAL words a cold player would type, and
fix any grant that does not fire.

- [ ] **Step 1: Boot the server and connect the harness adapter**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/boot_qtest.log 2>&1 &
sleep 30; grep -iE "Server Ready|panic" /tmp/boot_qtest.log | head
mkdir -p tools/playtest/.run && : > tools/playtest/.run/commands.txt && : > tools/playtest/.run/events.jsonl
( tail -n +1 -f tools/playtest/.run/commands.txt | ../gomud-playtest-harness/mudagent.exe \
    --target localhost:55555 --user smoketester --password smoke123test \
    > tools/playtest/.run/events.jsonl 2>&1 ) & disown
sleep 12
```

Use `teleport <roomid>` to reach givers, drive one command per round (the AI
input cap is 2/round — never batch teleport+ask), and read results from
`events.jsonl` (read with `python -c` and `encoding='utf-8', errors='replace'`
to avoid Windows cp1252 crashes on the prompt's emoji).

- [ ] **Step 2: Q69 (linear, simplest) end-to-end**

Drive: `teleport <Dross room 5904>` → `ask dross cipher` (expect `69-start`
grant) → `teleport 5901` → `interact reliefs` (expect Cipher Rubbing + rubbing
step) → back to Dross `ask dross gallery` (expect `69-gallery`) → `teleport
<Ferrol 6004>` → `ask ferrol panel` (expect the steer/warn, NO grant) →
`teleport 6008` → `ask lysha third panel` (expect Lysha's reading + 40116) →
back to Dross `give reading to dross` (expect completion + reward gold/rep).
Verify each grant fires on the NATURAL word; if a node is shadowed, reorder it
ahead of the shadowing lore node and note it.

- [ ] **Step 3: Q70 end-to-end including the Canal Lurker**

Drive: `teleport 5602` → `ask coll fragment` (expect `70-start` + Sealed Grey
Fragment) → navigate down the Old Quarter descent → confirm the **Canal Lurker
(9393) spawns and is attackable** in 6038 and the fight is survivable at
newcomer power (`attack lurker`, watch the round beacons) → reach Gritta 6037 →
`give fragment to gritta` (expect `70-lintel` + Lintel Rubbing + her lines) →
`interact lintel` (flavor) → `teleport 5602` → `give rubbing to coll` (expect
completion + reward). If the Lurker is too hard/soft, tune its `statpool`.

- [ ] **Step 4: Q71 end-to-end**

Drive: `teleport 5804` → `ask ostry tribute` (`71-start`) → `teleport 5814` →
`ask vell tribute` (`71-vell`) → `teleport 5815` → `ask horst tribute`
(`71-horst`) → `interact tally` (expect Ledger Page + `71-ledger`) → `teleport
5804` → `give ledger to ostry` (expect completion + gold + the Ostry's Gratitude
blade in inventory + rep). Confirm the blade (40120) is received and wieldable.

- [ ] **Step 5: Q68 BOTH branches (reset between)**

Q68 is the marquee; test both branches. To re-run, edit the smoketester save to
strip `68-*` tokens/flags between runs (player save at
`_datafiles/world/dogmud/users/<id>.yaml`, `questprogress:` block) and reboot, or
use a second test character.

- Circle branch: `teleport 5711` → `ask orin cooperage` (`68-start`) → `teleport
  5719` → `ask toby tools` (`68-evidence` + Bench-Mark Rubbing) → `interact
  floorboard` (Edvar's Map-Fragment) → back to Orin `ask orin report` (the
  choice node) → `interact guild-mark` in 5719 (expect: circle flag set, +60
  gold, cooperage_circle up / bloodline down, Copy of Edvar's Map, `68-end`).
- Bloodline branch: fresh token state → through `68-evidence` → `teleport 5814`
  → `give map-fragment to vell` (expect: bloodline flag set, +100 gold,
  bloodline up / cooperage down, `68-end`).
- After a circle run, verify the soft-arc coloring: start Q69, reach Lysha, and
  confirm `ask lysha circle` returns the `q69_lysha_circle_aside` warmth line
  (flag `68-allegiance: circle` gate works).

- [ ] **Step 6: Re-grant + discoverability spot-check**

For each quest, after completing it, `ask <giver> quest` again and confirm it is
NOT re-offered (end token in `questExcluded`). Confirm `ask <giver> quest` and
`ask <giver> task` both reach each grant node (SOP 3).

- [ ] **Step 7: Clean up the harness, write the report**

```bash
printf '%s\n' '{"control":"quit"}' >> tools/playtest/.run/commands.txt
sleep 1
powershell -NoProfile -Command "Get-Process | Where-Object {\$_.ProcessName -match 'mudagent|DOGMud'} | Stop-Process -Force -ErrorAction SilentlyContinue"
```

Write `tools/playtest/reports/2026-06-25-local-capital-questlines.md`: per-quest
outcome (grants fired on natural words, items handed over, rewards/rep applied,
both Q68 branches, the Canal Lurker fight, soft-arc coloring), any bugs found +
fixed, and any tuning done. Commit any fixes made during this task with clear
messages, then commit the report.

```bash
git add tools/playtest/reports/2026-06-25-local-capital-questlines.md
git commit -m "test(np-quests): harness playtest report for the four capital questlines"
```

---

## Task 8: Final review, merge to master (HOLD push)

**Files:** none new (merge + memory)

- [ ] **Step 1: Dispatch a final code/content reviewer**

Dispatch a reviewer subagent over the whole branch diff (`git diff
master...feature/np-capital-questlines`) checking: every grant node first in its
tree; every `grantsQuest` has the end token in `questExcluded`; every grant node
has `quest`/`task` triggers; no hard numbers in player-facing text; voice (NPC
first person, hints second person); 80-char wrap; flag `68-allegiance` declared;
no `quest_granted` trigger keyed on a dialogue-granted token; reward blocks use
tag-less keys. Fix anything it flags.

- [ ] **Step 2: Final clean boot test (panic mode)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go run . > /tmp/boot_final.log 2>&1 &
sleep 35
grep -iE "Server Ready|panic|ValidateZoneConsistency|quests.LoadDataFiles|mobs.LoadDataFiles" /tmp/boot_final.log | head -20
```

Expected: `Server Ready`; quests loadedCount +4, mobs +1, items +9 vs the
pre-branch baseline; ValidateZoneConsistency errors=0; no panics.

- [ ] **Step 3: Merge to master with --no-ff (do NOT push)**

```bash
git checkout master
git merge --no-ff feature/np-capital-questlines -m "Merge: New Plymouth capital questlines (Q68-71)

Four quests covering the five quest-less districts, wiring the pre-founding
lore web into gameplay:
- 68 The Cooperage Circle (Crafting; branching: aid the circle vs report it)
- 69 The Gallery Cipher (Temple->Noble; cipher investigation)
- 70 The Pre-Founding Web (Old Quarter; relic descent + Canal Lurker fight)
- 71 The Tribute (Merchant->Noble; Ostry vs the bloodline)

Soft-arc cohesion via the 68-allegiance flag; 9 quest items (40112-40120),
one new hostile mob (9393), no new factions. Harness-playtested end-to-end.

Push HELD per user policy."
```

> **DO NOT push.** The push to origin/master is held by user policy (the user
> does the droplet deploy). Stop after the local merge.

- [ ] **Step 4: Update project memory**

Update `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\MEMORY.md`
and `project_new_plymouth_build.md`: capital questlines phase DONE+MERGED (4
quests, quest-less districts now all have a hook), note the soft-arc allegiance
flag, the Canal Lurker, and the still-HELD push. Keep the index line ≤200 chars.

---

## Self-review notes (plan author)

- **Spec coverage:** Q68 (Task 3, branching + flag + dismissal nodes ✓), Q69
  (Task 4 ✓), Q70 (Task 5 + Canal Lurker Task 2 ✓), Q71 (Task 6, blade reward
  reskin Task 1 Step 9 ✓), soft-arc flag coloring (Q69 Lysha aside + Q71 Vell
  aside + Q71 bonus rep ✓), all items 40112–40120 (Task 1 ✓), recommended-order
  hints (each giver's grant hints point onward ✓), testing (Task 7, both
  branches + Lurker + discoverability ✓).
- **Open verification items flagged inline for the implementer (not
  placeholders — explicit "confirm X against the codebase" instructions):**
  `room_interact` nouns exist in 5719/5901/6037/6038/5815 (else add a sentence);
  `speciesid` for an aquatic mob; the any-of quest-gate field name
  (`questRequiredAny`) — with a concrete fallback if unsupported.
- **Consistency:** token names (`68-start/evidence/end`, `69-start/rubbing/
  gallery/reading/complete/end`, `70-start/lintel/return/end`, `71-start/vell/
  horst/ledger/report/end`), item IDs, and mob/room IDs are used identically
  across quest YAML and dialogue. Branch resolution never keys a `quest_granted`
  trigger on a dialogue-granted token (gotcha #9 honored): Q68 branches resolve
  via `room_interact`/`item_give` triggers; Q69/Q70/Q71 auto-completes key on
  tokens granted by `item_give` quest triggers, not dialogue.
