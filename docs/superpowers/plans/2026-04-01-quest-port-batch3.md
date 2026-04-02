# Quest Port Batch 3 — Item Fetch + Delivery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port Quests 3, 4, 5, 6, and 16 to the quest engine with `item_give`/`item_gain`/`room_enter` triggers, convert static container items to spawninfo respawn, and disable JS scripts.

**Architecture:** Container items converted to spawninfo first. Then each quest: update YAML with triggers, update dialogue for collapsed steps, disable JS. Quest 3 uses the flag system for any-order dual-item delivery. Quest 16 has a bypass path using `quest_granted` chaining.

**Tech Stack:** YAML data files, existing quest engine triggers/flags infrastructure.

---

### Task 1: Container → Spawninfo Conversions

**Files:**
- Modify: `_datafiles/world/dogmud/rooms/labyrinth_of_low_tunnels/312.yaml`
- Modify: `_datafiles/world/dogmud/rooms/watchers_crossing/421.yaml`
- Modify: `_datafiles/world/dogmud/rooms/ashwick/4033.yaml`

- [ ] **Step 1: Convert room 312 (Carved Niche) — bone totem**

Read `_datafiles/world/dogmud/rooms/labyrinth_of_low_tunnels/312.yaml`. Change the `containers:` section to empty the `shelves` container (remove the static item). Add a spawninfo entry for item 14 with container targeting:

Change the containers section from having item 14 in shelves to `shelves: {}`.

Add to spawninfo:
```yaml
- itemid: 14
  container: shelves
  respawnrate: 5 real minutes
```

- [ ] **Step 2: Convert room 421 (Tollhouse) — crossing ledger**

Read `_datafiles/world/dogmud/rooms/watchers_crossing/421.yaml`. The crossing ledger (item 21) is currently only placed by the room script (421.js), not by container or spawninfo. Add spawninfo for item 21 so it respawns even after the JS script is disabled:

Add to spawninfo (or create the section if missing):
```yaml
- itemid: 21
  respawnrate: 5 real minutes
```

Note: item 21 goes on the room floor (no container), since the room script placed it loosely.

- [ ] **Step 3: Convert room 4033 (Hidden Grove) — forest herbs**

Read `_datafiles/world/dogmud/rooms/ashwick/4033.yaml`. Change `containers:` to empty the `herbs` container. Add spawninfo:

```yaml
- itemid: 40040
  container: herbs
  respawnrate: 5 real minutes
```

- [ ] **Step 4: Delete stale room instance saves**

```bash
rm _datafiles/world/dogmud/rooms.instances/labyrinth_of_low_tunnels/312.yaml 2>/dev/null
rm _datafiles/world/dogmud/rooms.instances/watchers_crossing/421.yaml 2>/dev/null
rm _datafiles/world/dogmud/rooms.instances/ashwick/4033.yaml 2>/dev/null
```

(Some may not exist — that's fine.)

- [ ] **Step 5: Verify quest items are not components**

Check items 14, 21, 31, 40008, 40040 for `is_component`. Read each item YAML and confirm it's not set or false.

- [ ] **Step 6: Commit**

```bash
git add -A _datafiles/world/dogmud/rooms/
git commit -m "fix: convert quest container items to spawninfo respawn

Bone totem (room 312), crossing ledger (room 421), forest herbs
(room 4033) now respawn via spawninfo every 5 minutes. Previously
static container items that never came back once taken."
```

---

### Task 2: Port Quest 3 — The Scholar's Collection

**Files:**
- Modify: `_datafiles/world/dogmud/quests/3-the_scholars_collection.yaml`
- Modify: `_datafiles/world/dogmud/dialogue/sanctum_basin/79.yaml`
- Disable: `_datafiles/world/dogmud/mobs/sanctum_basin/scripts/79-basin_scholar.js`

- [ ] **Step 1: Update Quest 3 YAML**

Replace the full content. Steps collapse from [start, totem, sac, end] to [start, partial, end]. Add flag declaration and 4 item_give triggers (first delivery of each item, completing delivery of each item):

```yaml
questid: 3
name: The Scholar's Collection
description: The Basin Scholar needs two specimens from the
  tunnels -- a bone totem from the warren shelves and a spore
  sac from the cave crawlers.
secret: false
flags:
  - key: delivered
    values: [totem, sac]
    description: "Which specimen was delivered first"

steps:
  - id: start
    description: "The Basin Scholar needs a bone totem from the
      Carved Niche and a spore sac from a cave crawler. Bring
      both specimens to complete the collection."
    hint: "Head into the Labyrinth of Low Tunnels. The bone totem
      is on the shelves in the Carved Niche (from the Low Junction
      go south, south, east, down, west, south to the niche). Spore
      sacs drop from spore crawlers deeper in the tunnels."
  - id: partial
    description: "You delivered one specimen. The Scholar still
      needs the other to complete the collection."
    hint: "Check your quest log to see which specimen you still
      need. The bone totem is in the Carved Niche shelves. Spore
      sacs drop from spore crawlers."
  - id: end
    description: "Both specimens delivered. The Scholar is
      delighted."

rewards:
  playermessage: "The Scholar's eyes light up as the final piece
    clicks into place. 'You have given the guild more data than a
    decade of field expeditions. I am deeply grateful. And perhaps
    a little uncomfortable with how it was obtained. But grateful.'"
  roommessage: "The Scholar spreads specimens across the workbench,
    muttering excitedly and scribbling notes."
  gold: 15

triggers:
  # Totem delivered first
  - event: item_give
    mob: 79
    item: 14
    conditions:
      has: [3-start]
      missing: [3-partial]
    actions:
      - grant: 3-partial
      - set_flag: {key: "3-delivered", value: "totem"}
      - npc_say:
          mob: 79
          lines:
            - {delay: 1, text: "Exquisite. The carving technique
                is post-mutation -- look at how the finger grooves
                accommodate clawed digits."}
            - {delay: 3, text: "Now I need the spore sac. The cave
                crawlers carry them deeper in the tunnels."}

  # Sac delivered first
  - event: item_give
    mob: 79
    item: 40008
    conditions:
      has: [3-start]
      missing: [3-partial]
    actions:
      - grant: 3-partial
      - set_flag: {key: "3-delivered", value: "sac"}
      - npc_say:
          mob: 79
          lines:
            - {delay: 1, text: "A spore sac! And intact! The spore
                density is extraordinary."}
            - {delay: 3, text: "Now I need a bone totem from the
                carved niche in the warren."}

  # Totem completes (sac was first)
  - event: item_give
    mob: 79
    item: 14
    conditions:
      has: [3-partial]
      missing: [3-end]
      has_flag: {"3-delivered": "sac"}
    actions:
      - grant: 3-end
      - npc_say:
          mob: 79
          lines:
            - {delay: 1, text: "The totem. Now I have both
                specimens. This will keep me occupied for months."}

  # Sac completes (totem was first)
  - event: item_give
    mob: 79
    item: 40008
    conditions:
      has: [3-partial]
      missing: [3-end]
      has_flag: {"3-delivered": "totem"}
    actions:
      - grant: 3-end
      - npc_say:
          mob: 79
          lines:
            - {delay: 1, text: "The spore sac. Now I have both
                specimens. This will keep me occupied for months."}
```

- [ ] **Step 2: Update Scholar dialogue (mob 79)**

Read `_datafiles/world/dogmud/dialogue/sanctum_basin/79.yaml`. Make these changes:

**Root variants:** Replace all refs to old steps with new ones:
- `questExcluded: ["3-totem", "3-sac"]` → `questExcluded: ["3-partial"]`
- `questRequired: ["3-totem"]` → `questRequired: ["3-partial"]` with `questFlagRequired: {"3-delivered": "totem"}`
- `questRequired: ["3-sac"]` → `questRequired: ["3-partial"]` with `questFlagRequired: {"3-delivered": "sac"}`
- `questRequired: ["3-end"]` stays as-is

**Nodes:** The `totem_delivery`, `sac_delivery`, `totem_completes`, `sac_completes` nodes all have `grantsQuest` and `requiresItem` — update their `questRequired`/`questExcluded` to use the new step names:
- `totem_delivery`: `questExcluded: ["3-totem", "3-sac"]` → `questExcluded: ["3-partial"]`; `grantsQuest: "3-totem"` → `grantsQuest: "3-partial"`
- `sac_delivery`: same pattern
- `totem_completes`: `questRequired: ["3-sac"]` → `questRequired: ["3-partial"]` + `questFlagRequired: {"3-delivered": "sac"}`; `grantsQuest: "3-end"` stays
- `sac_completes`: `questRequired: ["3-totem"]` → `questRequired: ["3-partial"]` + `questFlagRequired: {"3-delivered": "totem"}`; `grantsQuest: "3-end"` stays

NOTE: The dialogue nodes that set `grantsQuest: "3-partial"` don't set the flag — only the quest engine trigger does. But both paths (dialogue `requiresItem` and `give` command) need to work. The dialogue path won't set the flag, so we need `setsQuestFlag` on the dialogue nodes too:
- `totem_delivery`: add `setsQuestFlag: {key: "3-delivered", value: "totem"}`
- `sac_delivery`: add `setsQuestFlag: {key: "3-delivered", value: "sac"}`

**Voice check:** All existing `text` fields are proper NPC speech ✅. All `hints` are narrator voice ✅.

- [ ] **Step 3: Disable JS script**

```bash
mv _datafiles/world/dogmud/mobs/sanctum_basin/scripts/79-basin_scholar.js \
   _datafiles/world/dogmud/mobs/sanctum_basin/scripts/79-basin_scholar.js.bak
```

- [ ] **Step 4: Build and commit**

```bash
go build ./...
git add -A _datafiles/world/dogmud/quests/3-* _datafiles/world/dogmud/dialogue/sanctum_basin/79.yaml _datafiles/world/dogmud/mobs/sanctum_basin/scripts/
git commit -m "feat: port Quest 3 (Scholar's Collection) — dual-item delivery with flags"
```

---

### Task 3: Port Quest 4 — The Warden's Report

**Files:**
- Modify: `_datafiles/world/dogmud/quests/4-the_wardens_report.yaml`
- Verify: `_datafiles/world/dogmud/dialogue/dustwalk_road/83.yaml`
- Disable: `_datafiles/world/dogmud/rooms/dustwalk_road/408.js`
- Disable: Tessara mob script if exists

- [ ] **Step 1: Update Quest 4 YAML — add hints and triggers**

Read the existing quest YAML. Keep the 4 steps [start, investigate, evidence, end]. Add `hint:` fields to each step. Add triggers for room_enter, item_gain, and item_give. Add the full YAML content per the spec.

Quest hint directions:
- start: "Head south from the Warden's Post along the Dustwalk Road to find the bandit camp."
- investigate: "Search the Bandit Hollow for evidence. Look for anything the bandits left behind."
- evidence: "Return the bandit's purse to Road Warden Tessara at the Warden's Post."

- [ ] **Step 2: Verify Tessara dialogue SOP**

Read `_datafiles/world/dogmud/dialogue/dustwalk_road/83.yaml`. Verify quest/task triggers, questExcluded, voice rules. Fix any issues.

- [ ] **Step 3: Disable room script and mob script**

```bash
mv _datafiles/world/dogmud/rooms/dustwalk_road/408.js _datafiles/world/dogmud/rooms/dustwalk_road/408.js.bak
```

Check if Tessara has a mob script: `ls _datafiles/world/dogmud/mobs/dustwalk_road/scripts/83-*`. Disable if exists.

- [ ] **Step 4: Build and commit**

```bash
go build ./...
git add -A _datafiles/world/dogmud/quests/4-* _datafiles/world/dogmud/rooms/dustwalk_road/ _datafiles/world/dogmud/dialogue/dustwalk_road/ _datafiles/world/dogmud/mobs/dustwalk_road/scripts/
git commit -m "feat: port Quest 4 (Warden's Report) — room_enter + item_gain + item_give triggers"
```

---

### Task 4: Port Quest 5 — The Innkeeper's Complaint

**Files:**
- Modify: `_datafiles/world/dogmud/quests/5-the_innkeepers_complaint.yaml`
- Modify: `_datafiles/world/dogmud/dialogue/watchers_crossing/84.yaml`
- Disable: `_datafiles/world/dogmud/rooms/watchers_crossing/421.js`
- Disable: `_datafiles/world/dogmud/mobs/watchers_crossing/scripts/84-innkeeper_tolva.js`

- [ ] **Step 1: Update Quest 5 YAML — collapse to [start, end], add triggers**

Collapse dead steps (ledger, evidence) to [start, end]. Add item_give trigger for ledger delivery (item 21) to Tolva (mob 84). Add hints with directions to the Tollhouse.

- [ ] **Step 2: Update Tolva dialogue (mob 84)**

Read the dialogue. Update any references to dead steps (5-ledger, 5-evidence) to use 5-start/5-end. Verify SOP compliance.

- [ ] **Step 3: Disable scripts**

```bash
mv _datafiles/world/dogmud/rooms/watchers_crossing/421.js _datafiles/world/dogmud/rooms/watchers_crossing/421.js.bak
mv _datafiles/world/dogmud/mobs/watchers_crossing/scripts/84-innkeeper_tolva.js _datafiles/world/dogmud/mobs/watchers_crossing/scripts/84-innkeeper_tolva.js.bak
```

- [ ] **Step 4: Build and commit**

```bash
go build ./...
git add -A _datafiles/world/dogmud/quests/5-* _datafiles/world/dogmud/dialogue/watchers_crossing/84.yaml _datafiles/world/dogmud/rooms/watchers_crossing/ _datafiles/world/dogmud/mobs/watchers_crossing/scripts/
git commit -m "feat: port Quest 5 (Innkeeper's Complaint) — ledger delivery via item_give trigger"
```

---

### Task 5: Port Quest 6 — The Collector's Burden

**Files:**
- Modify: `_datafiles/world/dogmud/quests/6-the_collectors_burden.yaml`
- Verify: `_datafiles/world/dogmud/dialogue/watchers_crossing/86.yaml`
- Verify: `_datafiles/world/dogmud/dialogue/thornwall_city/99.yaml`
- Disable: `_datafiles/world/dogmud/mobs/watchers_crossing/scripts/86-toll_collector_harn.js`

- [ ] **Step 1: Update Quest 6 YAML — add hints and triggers**

Keep [start, report, end]. Add item_give trigger for report delivery (item 31) to Clerk Pell (mob 99). Add hints with directions between Watchers Crossing and Thornwall.

Note: The `6-end` step is granted by returning to Harn AFTER Pell accepts the report. This is a dialogue-only step — Harn's dialogue checks `questRequired: ["6-report"]` and grants `6-end`. No trigger needed for end — dialogue handles it.

- [ ] **Step 2: Verify dialogue SOP for both Harn and Pell**

Check both dialogue files. Harn gives item via `givesItem: 31`. Pell accepts via `requiresItem: 31`. Both should have quest/task triggers. Verify voice rules.

- [ ] **Step 3: Disable Harn's mob script**

```bash
mv _datafiles/world/dogmud/mobs/watchers_crossing/scripts/86-toll_collector_harn.js _datafiles/world/dogmud/mobs/watchers_crossing/scripts/86-toll_collector_harn.js.bak
```

Check if Pell has a script too and disable if so.

- [ ] **Step 4: Build and commit**

```bash
go build ./...
git add -A _datafiles/world/dogmud/quests/6-* _datafiles/world/dogmud/dialogue/watchers_crossing/86.yaml _datafiles/world/dogmud/dialogue/thornwall_city/99.yaml _datafiles/world/dogmud/mobs/watchers_crossing/scripts/
git commit -m "feat: port Quest 6 (Collector's Burden) — cross-zone report delivery"
```

---

### Task 6: Port Quest 16 — The Herbalist's Shortage

**Files:**
- Modify: `_datafiles/world/dogmud/quests/16-the_herbalists_shortage.yaml`
- Verify: `_datafiles/world/dogmud/dialogue/ashwick/259.yaml`
- Verify: `_datafiles/world/dogmud/dialogue/ashwick/262.yaml`
- Disable: `_datafiles/world/dogmud/mobs/ashwick/scripts/259-delia.js`

- [ ] **Step 1: Update Quest 16 YAML — add hints, triggers, and missing rewards**

Keep [start, forager, end]. Add triggers for herb delivery (normal + bypass). Add quest_granted chain for bypass. Add missing rewards (12 gold). Add hints with directions to the forest grove and forager camp.

Per spec: two item_give triggers (normal has 16-start, bypass missing 16-start) plus quest_granted chain for auto-complete on bypass.

- [ ] **Step 2: Verify Delia and Forager dialogue SOP**

Read both dialogues. Verify quest/task triggers, voice rules, hint text. The forager dialogue should grant `16-forager` via `grantsQuest`. Delia's dialogue should have a completion node for the forager path.

- [ ] **Step 3: Disable Delia's mob script**

```bash
mv _datafiles/world/dogmud/mobs/ashwick/scripts/259-delia.js _datafiles/world/dogmud/mobs/ashwick/scripts/259-delia.js.bak
```

- [ ] **Step 4: Build and commit**

```bash
go build ./...
git add -A _datafiles/world/dogmud/quests/16-* _datafiles/world/dogmud/dialogue/ashwick/ _datafiles/world/dogmud/mobs/ashwick/scripts/
git commit -m "feat: port Quest 16 (Herbalist's Shortage) — dual-path herb delivery with bypass"
```

---

### Task 7: Manual Test — All Five Quests

- [ ] **Step 1: Test Quest 3 — totem first, then sac**

New character → ask scholar quest → go to tunnels → get totem from shelves → give scholar totem → verify 3-partial + flag 3-delivered=totem → kill spore crawlers → get sac → give scholar sac → verify 3-end + rewards.

- [ ] **Step 2: Test Quest 3 — sac first, then totem**

Fresh character → reverse order. Verify flag shows 3-delivered=sac after first delivery.

- [ ] **Step 3: Test Quest 4 — full chain**

Ask Tessara → go south to room 408 → verify 4-investigate fires on room entry → pick up purse → verify 4-evidence → give purse to Tessara → verify 4-end.

- [ ] **Step 4: Test Quest 5**

Ask Tolva → go to Tollhouse → get ledger → give ledger to Tolva → verify 5-end. Verify ledger respawns after 5 minutes.

- [ ] **Step 5: Test Quest 6 — cross-zone delivery**

Ask Harn → receive report (item 31) → travel to Thornwall → give report to Pell → verify 6-report → return to Harn → ask about quest → verify 6-end.

- [ ] **Step 6: Test Quest 16 Path A — forager dialogue**

Ask Delia → find forager in forest → negotiate → verify 16-forager → return to Delia → verify 16-end.

- [ ] **Step 7: Test Quest 16 Path B — herb bypass**

Fresh character → find herbs in Hidden Grove → give herbs to Delia without prior conversation → verify both 16-start and 16-end fire.

- [ ] **Step 8: Respawn test**

Take bone totem, crossing ledger, and forest herbs. Wait 5 minutes. Return to each room and verify items reappeared.
