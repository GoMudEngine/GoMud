# Quest Port Batch 1 — Dialogue-Only Quests

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Port Quests 8, 12, and 13 to the quest engine with working hints, correct rewards, SOP-compliant dialogue, and fix a missing dialogue node in Quest 13.

**Architecture:** These three quests are dialogue-only — all progression happens through NPC `ask` interactions. The dialogue YAML's `grantsQuest` field handles keyword-matched quest progression; the quest engine provides steps (for the `hint` command), rewards (fired by `HandleQuestUpdate` on token grant), and banners. Quest engine `dialogue` triggers use exact topic matching which is too rigid for player-typed input, so `grantsQuest` stays in dialogue YAML as the progression mechanism. Quest engine triggers are added only where they add value (e.g., `quest_granted` chains for reward sequencing).

**Key files:**
- Quest YAMLs: `_datafiles/world/dogmud/quests/{8,12,13}-*.yaml`
- Dialogue YAMLs: `_datafiles/world/dogmud/dialogue/thornwall_city/{94,107}.yaml`, `_datafiles/world/dogmud/dialogue/ironwind_steppe/{241,242}.yaml`
- No Go code changes needed
- No JS script changes needed (241/242 scripts handle Quest 11, not 12/13)

**Design decision — why `grantsQuest` stays in dialogue YAML:**
The quest engine's `dialogue` trigger matches `topic` with strict equality (`t.Topic != d.Topic`). Players type freeform text like `ask velk missing person case`, but the engine would need `topic: "missing person case"` exactly. The dialogue YAML handles this correctly via keyword arrays. Moving `grantsQuest` to quest engine triggers would either (a) require a Go engine change to add keyword-list topic matching, or (b) use empty topics that match ANY ask to the mob regardless of what the player said — causing wrong NPC text to display alongside quest banners. Neither is worth it for Batch 1. If keyword topic matching is added to the engine later, these quests can be migrated in a follow-up pass.

---

### Task 1: Port Quest 8 — The City Watch's Missing Person

**Files:**
- Modify: `_datafiles/world/dogmud/quests/8-the_city_watchs_missing_person.yaml`
- Verify: `_datafiles/world/dogmud/dialogue/thornwall_city/94.yaml` (mob 94 — Guard Captain Velk, room 473)
- Verify: `_datafiles/world/dogmud/dialogue/thornwall_city/107.yaml` (mob 107 — Elara, room 477)

**Current state:** Quest YAML has 3 steps (start, found, end) with basic hints. Dialogue YAML has correct `grantsQuest` fields on all three progression nodes. Rewards defined (15 gold + player/room messages). No JS scripts involved.

**What needs fixing:**
- Hints need explicit cardinal directions (per recent SOP — see commits `2382528c`, `89270e59`)
- Verify `ask <npc> quest/task` triggers exist on all quest-granting nodes
- Verify `questExcluded` prevents double-completion

- [ ] **Step 1: Look up room connections for hint directions**

Run: Read room files to find the path from Thornwall City entrance to the barracks (room 473) and from barracks to residential ward (room 477). Check `_datafiles/world/dogmud/rooms/thornwall_city/` for rooms 473 and 477 exits, and trace the route between them.

- [ ] **Step 2: Update Quest 8 YAML with explicit hints**

Replace the full quest YAML. The `description` fields are forward-looking (tell the player what to do next). Hints give step-by-step cardinal directions. Keep rewards unchanged.

```yaml
questid: 8
name: The City Watch's Missing Person
description: Guard Captain Velk needs someone to find a missing citizen
  named Elara, reported by her family. The family wants her returned.
  Elara wants to stay hidden. There is no clean answer.
secret: false

steps:
  - id: start
    description: "Guard Captain Velk at the barracks has asked you to
      locate a missing person -- a young woman named Elara. Search the
      residential wards north of the market for any sign of her."
    hint: "Search the residential wards for Elara. From the barracks,
      [DIRECTIONS TO ROOM 477 — fill after Step 1 room lookup].
      Look for someone who does not want to be found."
  - id: found
    description: "You found Elara hiding in the residential ward. She
      fled an arranged marriage and does not want to be returned. Return
      to Guard Captain Velk at the barracks and report what you found."
    hint: "Return to the barracks and speak with Captain Velk. From the
      residential ward, [DIRECTIONS TO ROOM 473 — fill after Step 1].
      What you tell him is your choice."
  - id: end
    description: "You reported back to Captain Velk. The case is closed."

rewards:
  playermessage: "Velk listens to your report without interruption, his
    expression unreadable. When you finish, he makes a notation in the
    duty log. 'The case is closed. I will file it accordingly.' He does
    not ask for details beyond what you offered, and you get the sense
    that he has seen enough of these cases to know that the paperwork
    rarely captures the whole truth."
  roommessage: "Velk makes a notation in the duty log and files the
    report."
  gold: 15
```

Note: Replace the `[DIRECTIONS ...]` placeholders with actual cardinal directions found in Step 1. The writing-plans skill requires complete content — these must be filled before committing.

- [ ] **Step 3: Verify dialogue SOP compliance for mob 94 (Velk)**

Check `_datafiles/world/dogmud/dialogue/thornwall_city/94.yaml`:
- Node `missing_case`: Must have `"quest"` and `"task"` in triggers ✅ (already has them)
- Node `case_report`: Must have `"quest"` and `"task"` in triggers ✅ (already has them)
- `questExcluded: ["8-start"]` on `missing_case` prevents re-grant ✅
- `questExcluded: ["8-end"]` on `case_report` prevents re-grant ✅
- Hints use player perspective (no 3rd-person self-references) ✅

No dialogue changes expected unless SOP violations found during verification.

- [ ] **Step 4: Verify dialogue SOP compliance for mob 107 (Elara)**

Check `_datafiles/world/dogmud/dialogue/thornwall_city/107.yaml`:
- Node `reassure`: Must have `"quest"` and `"task"` in triggers ✅ (already has them)
- `questExcluded: ["8-found"]` prevents re-grant ✅
- Hints use player perspective ✅

No dialogue changes expected unless SOP violations found.

- [ ] **Step 5: Commit Quest 8 changes**

```bash
git add _datafiles/world/dogmud/quests/8-the_city_watchs_missing_person.yaml
# Include dialogue files only if changes were made in Steps 3-4
git commit -m "feat: port Quest 8 (Missing Person) hints to quest engine"
```

---

### Task 2: Port Quest 12 — The Warden's Covenant

**Files:**
- Modify: `_datafiles/world/dogmud/quests/12-the_wardens_covenant.yaml`
- Verify: `_datafiles/world/dogmud/dialogue/ironwind_steppe/241.yaml` (mob 241 — Windwarden Sylara, room 3115)

**Current state:** Quest YAML has 3 steps (start, ritual, end). Dialogue YAML has correct `grantsQuest` on all three nodes (`offer_covenant`, `ritual_begin`, `ritual_complete`). Rewards: 15 gold, item 40031 (spirit fetishes), spell summon-steppe-spirit. JS script on mob 241 handles Quest 11 onGive + post-Q12 fetish dispensing — not touched here.

**What needs fixing:**
- Hints need explicit cardinal directions
- Verify SOP compliance

- [ ] **Step 1: Look up room connections for hint directions**

Read room files to find the path to Sylara's location (room 3115) from the Ironwind Steppe entrance. The stone circle area. Check `_datafiles/world/dogmud/rooms/ironwind_steppe/` for room 3115 and adjacent rooms.

- [ ] **Step 2: Update Quest 12 YAML with explicit hints**

```yaml
questid: 12
name: The Warden's Covenant
description: You have chosen to support Windwarden Sylara and protect
  the sacred stone circle. She will teach you to call upon the spirits
  of the steppe. Completing this quest locks The Prospector's Gambit.
secret: false

steps:
  - id: start
    description: "Windwarden Sylara has offered to teach you the ancient
      covenant. Travel to her at the stone circle and tell her you are
      ready for the ritual."
    hint: "Tell Sylara you are ready for the ritual. She is at the stone
      circle. [DIRECTIONS TO ROOM 3115 — fill after Step 1].
      Try: ask sylara ready"
  - id: ritual
    description: "Sylara has begun the binding ritual at the stone
      circle. Open yourself to the spirits of the steppe and complete
      the binding."
    hint: "Complete the ritual with Sylara. Listen and accept the
      spirits' call. Try: ask sylara accept"
  - id: end
    description: "The spirits of the steppe answer your call. You have
      earned the Windwarden's trust."

rewards:
  playermessage: "Sylara's eyes shine with quiet pride. The steppe wind
    swirls around you like a greeting, and you feel a new connection to
    the land -- ancient, vast, and watchful. She presses a bundle of
    spirit fetishes into your hands."
  gold: 15
  itemid: 40031
  spellid: summon-steppe-spirit
```

Note: Replace `[DIRECTIONS ...]` with actual cardinal directions after Step 1 lookup.

- [ ] **Step 3: Verify dialogue SOP compliance for mob 241 (Sylara)**

Check `_datafiles/world/dogmud/dialogue/ironwind_steppe/241.yaml`:
- Node `offer_covenant`: triggers include `"quest"`, `"task"` ✅
- Node `ritual_begin`: triggers include `"ritual"`, `"ready"` — check if `"quest"` and `"task"` should be added. Since this is a mid-quest step (not quest-granting), `quest`/`task` is not required by SOP. But the player might try `ask sylara quest` to progress — verify root variant handles this case.
- Node `ritual_complete`: triggers include `"accept"`, `"yes"` — same consideration.
- `questExcluded` prevents double-completion on all nodes ✅
- Root variants correctly gate by quest state ✅
- Hints use player perspective ✅

- [ ] **Step 4: Commit Quest 12 changes**

```bash
git add _datafiles/world/dogmud/quests/12-the_wardens_covenant.yaml
# Include dialogue file only if changes were made
git commit -m "feat: port Quest 12 (Warden's Covenant) hints to quest engine"
```

---

### Task 3: Port Quest 13 — The Prospector's Gambit (+ fix missing dialogue node)

**Files:**
- Modify: `_datafiles/world/dogmud/quests/13-the_prospectors_gambit.yaml`
- Modify: `_datafiles/world/dogmud/dialogue/ironwind_steppe/242.yaml` (mob 242 — Geomancer Rhett, room 3030)

**Current state:** Quest YAML has 3 steps (start, extraction, end). Dialogue YAML has `grantsQuest` on `extraction_begin` (grants 13-extraction) and `extraction_complete` (grants 13-end). Rewards: 15 gold, item 20053 (Windstone Aegis).

**BUG: Quest 13 has no dialogue node that grants `13-start`.** After Quest 11 ends (Rhett path), the root variant says "Ask me about the work if you are interested" but there is no node with triggers matching "work"/"extract"/"gambit" that grants `13-start`. The `extraction_begin` node requires `13-start` already be present. This means Quest 13 is currently **impossible to start**.

- [ ] **Step 1: Look up room connections for hint directions**

Read room files for Rhett's location (room 3030) and the extraction site. Check `_datafiles/world/dogmud/rooms/ironwind_steppe/` for room 3030 and the rocky shelf.

- [ ] **Step 2: Add missing `offer_gambit` dialogue node to mob 242**

Add this node to `_datafiles/world/dogmud/dialogue/ironwind_steppe/242.yaml` in the `nodes:` list, between `sample_return` and `extraction_begin`:

```yaml
    - id: offer_gambit
      triggers: ["work", "extract", "gambit", "help", "quest", "task"]
      questRequired: ["11-end"]
      questExcluded: ["13-start", "12-start"]
      grantsQuest: "13-start"
      text: "'The extraction site is ready. I have mapped the crystal
        grain, prepared the tools, set the fracture points. All I need
        is someone with steady hands to hold the brace while I work
        the seam.' Rhett's eyes are bright with barely contained
        excitement. 'One clean slab of windstone -- enough to forge a
        proper aegis for Thornwall's failing wards. What do you say?
        Will you help?'"
      hints: "Tell Rhett you are ready to help with the extraction."
      moodChange: friendly
```

Key design notes:
- `questExcluded: ["13-start", "12-start"]` — prevents re-grant AND prevents starting Q13 if player already chose Q12 (Sylara's path). This mirrors how `offer_covenant` excludes `12-start`.
- Triggers include `"quest"` and `"task"` per SOP.
- `"work"` matches the root variant hint ("ask me about the work").

- [ ] **Step 3: Verify the `extraction_begin` node's `questExcluded` is correct**

The existing `extraction_begin` node has `questRequired: ["13-start"]` and `questExcluded: ["13-extraction"]`. This is correct — it gates on `13-start` (now grantable) and prevents re-trigger. No change needed.

- [ ] **Step 4: Update Quest 13 YAML with explicit hints**

```yaml
questid: 13
name: The Prospector's Gambit
description: You have chosen to help Geomancer Rhett extract windstone
  from the ridge. The crystal could advance healing magic and protect
  Thornwall. Completing this quest locks The Warden's Covenant.
secret: false

steps:
  - id: start
    description: "Geomancer Rhett has asked for your help extracting a
      slab of windstone from the ridge. Travel to his camp and tell him
      you are ready to begin."
    hint: "Tell Rhett you are ready for the extraction. He is at his
      camp on the rocky shelf. [DIRECTIONS TO ROOM 3030 — fill after
      Step 1]. Try: ask rhett ready"
  - id: extraction
    description: "Rhett is working the crystal seam. Hold the brace
      steady while he extracts the windstone slab."
    hint: "Help Rhett complete the extraction. Hold steady and let him
      work. Try: ask rhett hold"
  - id: end
    description: "The extraction is complete. Rhett has the windstone
      he needs."

rewards:
  playermessage: "Rhett grins broadly, cradling the extracted crystal
    like a newborn. He disappears into his work for hours, and when
    he emerges, he holds a shield of pale blue crystal -- the
    Windstone Aegis, forged in gratitude."
  roommessage: "Rhett emerges from his camp, grinning, and presents
    a shield of pale blue crystal to his companion."
  gold: 15
  itemid: 20053
```

Note: Replace `[DIRECTIONS ...]` with actual cardinal directions after Step 1 lookup.

- [ ] **Step 5: Commit Quest 13 changes**

```bash
git add _datafiles/world/dogmud/quests/13-the_prospectors_gambit.yaml
git add _datafiles/world/dogmud/dialogue/ironwind_steppe/242.yaml
git commit -m "fix: Quest 13 missing offer_gambit node + port hints"
```

---

### Task 4: SOP Compliance Audit — All Three Quests

Run through the full checklist from the Quest Fix & Creation SOP for all three quests.

- [ ] **Step 1: Elephant path audit — Quest 8**

Verify these player paths are handled:
1. `ask velk quest` / `ask velk task` → must reach `missing_case` node. Check triggers list includes both. ✅
2. `ask elara quest` / `ask elara task` → must reach `reassure` node. Check triggers list. ✅
3. `give <item> velk` when player has no quest items → Velk has no onGive script. This is fine since Q8 has no item delivery. No action needed.
4. Player returns to Velk before finding Elara → Root variant shows "Any news on the missing girl?" with hints. ✅
5. Player asks Elara about quest after already finding her → Root variant for `questRequired: ["8-found"]` shows appropriate text. ✅

- [ ] **Step 2: Elephant path audit — Quest 12**

1. `ask sylara quest` / `ask sylara task` → reaches `offer_covenant` node (has "quest", "task" in triggers). But only if `11-end` is present and `12-start` is not. ✅
2. Player tries `ask sylara ritual` before being offered covenant → no node matches (all require quest tokens). Root variant handles this. ✅
3. Player asks for fetishes before completing Q12 → `give_fetish_noqust` node handles this. Note: this node has no `questRequired` but gives a fetish anyway. This is an existing design choice (Sylara gives fetishes to anyone the wind trusts). Leave as-is.

- [ ] **Step 3: Elephant path audit — Quest 13**

1. `ask rhett quest` / `ask rhett task` → reaches `offer_gambit` node (new, has "quest", "task"). ✅
2. `ask rhett work` → reaches `offer_gambit` (has "work" trigger). ✅
3. Player tries extraction before being offered gambit → `extraction_begin` requires `13-start`. Falls through to root variant or pattern. ✅
4. Player who chose Sylara's path (has `12-start`) asks Rhett about work → `offer_gambit` has `questExcluded: ["12-start"]` so it won't fire. Root variant for `questRequired: ["12-end"]` shows disappointment text. ✅

- [ ] **Step 4: Double-completion guard audit**

Verify every `grantsQuest` node has matching `questExcluded`:
- Q8: `missing_case` → `questExcluded: ["8-start"]` ✅, `reassure` → `questExcluded: ["8-found"]` ✅, `case_report` → `questExcluded: ["8-end"]` ✅
- Q12: `offer_covenant` → `questExcluded: ["12-start"]` ✅, `ritual_begin` → `questExcluded: ["12-ritual"]` ✅, `ritual_complete` → `questExcluded: ["12-end"]` ✅
- Q13: `offer_gambit` → `questExcluded: ["13-start", "12-start"]` ✅, `extraction_begin` → `questExcluded: ["13-extraction"]` ✅, `extraction_complete` → `questExcluded: ["13-end"]` ✅

- [ ] **Step 5: Hint discoverability audit**

For each quest step's hint, verify every suggested action appears in NPC text or a previous hint:
- Q8 "Search the residential wards" — Velk says "the residential wards need someone to knock on doors" ✅
- Q8 "Return to the barracks and speak with Captain Velk" — Elara's `the_choice` node says "go to the guard captain" ✅
- Q12 "Tell Sylara you are ready" — root variant says "Are you ready for the ritual?" ✅
- Q12 "accept the spirits' call" — `ritual_begin` text says "Open yourself" and the node triggers include "accept" ✅
- Q13 "Tell Rhett you are ready" — `offer_gambit` text ends with "Will you help?" ✅
- Q13 "Hold steady" — `extraction_begin` text says "Hold this brace steady" ✅

- [ ] **Step 6: Commit any fixes found during audit**

If any SOP violations were found and fixed in Steps 1-5:
```bash
git add -A _datafiles/world/dogmud/dialogue/ _datafiles/world/dogmud/quests/
git commit -m "fix: SOP compliance fixes for Batch 1 quests"
```

If no fixes needed, skip this step.

---

### Task 5: Manual Test Walkthrough

Start the server and walk through each quest as a player.

- [ ] **Step 1: Test Quest 8 — start**

1. Travel to Velk (room 473)
2. `ask velk quest` → should see missing person text + quest start banner
3. `hint` → should show Quest 8 step hint with directions to residential ward
4. `ask velk quest` again → should NOT re-grant (questExcluded gates it)

- [ ] **Step 2: Test Quest 8 — found**

1. Travel to Elara (room 477)
2. `ask elara quest` → should see reassure text + quest progress banner
3. `hint` → should show hint about returning to Velk
4. `ask elara quest` again → should NOT re-grant

- [ ] **Step 3: Test Quest 8 — end**

1. Travel back to Velk (room 473)
2. `ask velk report` → should see case closed text + quest complete banner + 15 gold reward
3. `hint` → should show no active hint for Q8 (quest complete)

- [ ] **Step 4: Test Quest 12 — full chain**

Requires Quest 11 (Sylara path) to be completed first.
1. Ensure player has `11-end` token (via `givequest 11-end` admin command if needed)
2. `ask sylara covenant` → grants 12-start + quest start banner
3. `ask sylara ready` → grants 12-ritual
4. `ask sylara accept` → grants 12-end + quest complete banner + rewards (15 gold, spirit fetishes, spell)
5. Verify `cast summon-steppe-spirit` works after quest completion

- [ ] **Step 5: Test Quest 13 — full chain (NEW: includes offer_gambit fix)**

Requires Quest 11 (Rhett path) completed, Quest 12 NOT started.
1. Ensure player has `11-end` token, does NOT have `12-start`
2. `ask rhett quest` → should fire new `offer_gambit` node, grants 13-start + quest start banner
3. `ask rhett ready` → grants 13-extraction
4. `ask rhett hold` → grants 13-end + quest complete banner + rewards (15 gold, Windstone Aegis)
5. Verify shield item is in inventory

- [ ] **Step 6: Test mutual exclusion**

1. Start fresh character (or clear quest tokens)
2. Complete Q11 via Sylara path → get `11-end`
3. Start Q12 → get `12-start`
4. `ask rhett quest` → should NOT offer gambit (questExcluded: ["12-start"])
5. Verify Rhett's root variant shows disappointment text

- [ ] **Step 7: Commit test notes**

No code commit — just verify all tests pass. If any test reveals a bug, fix it and commit the fix before proceeding.
