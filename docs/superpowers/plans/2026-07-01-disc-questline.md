# The Disc Questline (Quest 76) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build Quest 76 — the disc questline that turns the Pothole Coulee disc into the usable Attuned Disc key for the crash-site door, playable by any character regardless of creation date.

**Architecture:** A data-only quest (no engine code): two items (Inert Disc, Attuned Disc), a quest YAML with `room_interact` triggers (modeled exactly on Q75's filed-survey/field-notes triggers), a Brennan dialogue grant node, and noun additions to two existing rooms. Universality comes from a `missing_item`-gated per-player disc grant at Pothole Coulee 5343 (reachable overland by anyone) + Brennan's directions.

**Tech Stack:** GoMud YAML (items/quests/dialogue/rooms); `python -c "import yaml"`; local boot (`ValidateAllFlags`); the mudagent harness for the end-to-end quest test.

**Reference spec:** `docs/superpowers/specs/2026-07-01-disc-questline-design.md`

**Verified engine facts (do not re-derive):**
- Quest `room_interact` trigger shape (from `quests/75-the_surveyors_report.yaml`): `- event: room_interact` / `room: <id>` / `noun: <noun>` / `conditions: {has: [...], missing: [...], has_item: N, missing_item: N}` / `actions: [...]`.
- Conditions support `has_item`/`missing_item` (`internal/questengine/conditions.go:28,31`).
- Actions (`internal/questengine/types.go:74` ActionDef): `grant: "<token>"`, `consume_item: N`, `give_item: N`, `send_text: "..."`, `room_text: "..."`, `bump_rep: {faction, delta}`.
- `room_interact` fires on `examine`/`look` of the noun (write hints as "examine the X"); the noun must exist in the room.
- Quest tokens are validated at boot (`ValidateAllFlags`) — an undeclared token panics; declare all four steps.
- IDs: Inert Disc **40167**, Attuned Disc **40168**, Quest **76**. Brennan = `dialogue/greenford/9516.yaml`; Q75 end token = `75-end`.

**Conventions:** no colon-space `": "` in prose values; no semicolons in NPC/quest text; `|` block scalars for long text; hard-wrap ~78 cols; no hard numbers in player text; stage explicit git pathspecs.

---

## Task 1: The two disc items

**Files:** Create `items/materials-40000/40167-inert_disc.yaml`, `40168-attuned_disc.yaml`

- [ ] **Step 1: Inert Disc (40167).** Model on an existing simple `object` item (e.g. `40165-sentinels_heartstone.yaml`). A key-story object, `not_salable`, no mechanical effect:
```yaml
itemid: 40167
name: an inert disc
namesimple: disc
description: >
  A disc of pale grey metal the size of a spread hand, too smooth and
  too regular to be anything the folk made — fine raised lines threading
  inward to a single mark at its centre, a ring struck through with a
  line. It stays cool in the hand and holds the light a half-instant
  longer than it should. It does nothing. Whatever it is for, it is not
  awake.
type: object
subtype: mundane
not_salable: true
weight: 0.3
value: 0
rarity_tier: 70
```
- [ ] **Step 2: Attuned Disc (40168).** Same shape; the woken key:
```yaml
itemid: 40168
name: an attuned disc
namesimple: disc
description: >
  The same pale grey disc — but the raised lines hold a faint steady
  warmth now, and the mark at its centre catches the light and keeps it,
  a ring struck through with a line, patient and awake. It answers
  something. It is a key, and it knows the shape of its lock.
type: object
subtype: mundane
not_salable: true
weight: 0.3
value: 0
rarity_tier: 85
```
- [ ] **Step 3: Validate + commit.** `python -c "import yaml,glob;[yaml.safe_load(open(f,encoding='utf-8')) for f in glob.glob('_datafiles/world/dogmud/items/materials-40000/4016[78]-*.yaml')];print('OK')"`; `git add _datafiles/world/dogmud/items/materials-40000/40167-inert_disc.yaml _datafiles/world/dogmud/items/materials-40000/40168-attuned_disc.yaml && git commit -m "feat(disc-questline): inert + attuned disc items"`

## Task 2: Room nouns — 5343 (disc take) and 4023 (journal + ritual)

**Files:** Modify `rooms/pothole_coulee/5343.yaml`, `rooms/ashwick/4023.yaml`

The quest triggers (Task 3) reference nouns that must exist in the rooms. 5343 already has the `disc` hidden_noun (keep it). 4023 needs a `journal` noun + a ritual-focus noun.

- [ ] **Step 1: 5343 — soften the disc noun's finality.** The existing `disc` hidden_noun ends "...waiting under the floor for a hand to lift it. Now yours has." Since taking it is now the quest grant (Task 3), reword the ending so the look-text doesn't pre-empt the take: change the final sentence to "...waiting under the floor for a hand to lift it." (drop "Now yours has."). Keep everything else. (The room_interact send_text in Task 3 delivers the actual "you work it free" moment.)
- [ ] **Step 2: 4023 (Maren's Cottage) — add the `journal` + ritual nouns.** Read `rooms/ashwick/4023.yaml` first. Add two `nouns` entries (rich, lore-appropriate, threshold-only re the buried thing):
  - `journal`: the father's field journal on the cottage table — years of obsession with a buried thing in the east, sketches of the ring-struck-through mark, and (the useful part) his worked-out method for *waking* the disc. Mentions his daughter by name once, gently. Do NOT reveal what the buried thing IS.
  - a ritual focus, key `hearth-marks` (or similar): marks he cut into the hearthstone — the pattern the attunement follows. Examinable; the place the ritual happens.
  Also add a line to the room `description` so both nouns are discoverable (mention the journal on the table + the marked hearth), and add the nouns to any `<ansi fg="itemname">` tags you introduce.
- [ ] **Step 3: Validate + commit** (both rooms). `python -c "import yaml; [yaml.safe_load(open(p,encoding='utf-8')) for p in ['_datafiles/world/dogmud/rooms/pothole_coulee/5343.yaml','_datafiles/world/dogmud/rooms/ashwick/4023.yaml']]; print('OK')"`; commit.

## Task 3: The quest YAML (76) — steps + room_interact triggers

**Files:** Create `quests/76-the_disc.yaml`

Model exactly on `quests/75-the_surveyors_report.yaml`. Four steps; three room_interact triggers (disc grant, journal, attunement) + ungated fallbacks; `76-start` is granted by Brennan (Task 4), NOT a trigger here.

- [ ] **Step 1: Write the quest skeleton + steps.**
```yaml
questid: 76
name: The Disc
description: >-
  The inert disc is a key that has never been woken. Brennan named the
  man who studied how — a scholar who went east from Ashwick years ago
  and did not come back. His research waits in the cottage he left
  behind.
secret: false
steps:
  - id: start
    description: "Brennan believes the disc is a key that must be woken, and
      that the man who knew how left his work in an Ashwick cottage."
    hint: "You will need the disc itself — it lies in the old Reliquary at
      Pothole Coulee, worked into the floor. Then seek the abandoned
      cottage in Ashwick."
  - id: journal
    description: "You read the father's journal — a life spent chasing a
      buried thing in the east, and the method he worked out for waking
      the disc."
    hint: "Follow his method at the marked hearth in the cottage, with the
      disc in hand."
  - id: attune
    description: "You followed his method at the hearth, and the disc woke
      — the lines hold warmth now, and the mark keeps the light."
    hint: "The disc is a key now. It knows the shape of its lock, east."
  - id: end
    description: "You carry the woken disc — a key for the door in the east,
      and the last of a lost man's work, done at last."
rewards:
  playermessage: >-
    The disc is awake in your hand, warm along its lines, and it answers
    a shape you have not yet reached. A father's work, finished by a
    stranger, in the cottage he left to chase it. What it opens waits
    east, in the broken country, behind a door in the buried face.
  roommessage: ""
  gold: 0
```
- [ ] **Step 2: Add the disc-grant trigger (5343) — universal, per-player-once, no quest token needed.**
```yaml
triggers:
  # ── Take the disc from the Reliquary (5343) — anyone, once. ──
  - event: room_interact
    room: 5343
    noun: disc
    conditions:
      missing_item: 40167
    actions:
      - give_item: 40167
      - send_text: >-
          You work your fingers into the crack and lift the disc free of
          the floor — cool, too smooth, heavier than its size. Fine raised
          lines thread across its face to a ring struck through with a
          line. You do not understand it. You are fairly certain no one
          here does. You put it away.
      - room_text: "works something free of the cracked floor and pockets it."
  # ── Already have it (or already woken it): flavor only. ──
  - event: room_interact
    room: 5343
    noun: disc
    conditions:
      has_item: 40167
    actions:
      - send_text: "The crack in the floor is empty now. You already carry what lay beneath it."
```
(Note: a player holding the Attuned Disc 40168 but not the Inert 40167 would still match `missing_item: 40167` and get a second inert disc — acceptable, and it's the recovery path for a lost key: re-fetch, re-attune. The `has_item: 40167` fallback covers the common case.)
- [ ] **Step 3: Add the journal trigger (4023).**
```yaml
  # ── The father's journal (4023, Maren's Cottage) — has 76-start, missing 76-journal. ──
  - event: room_interact
    room: 4023
    noun: journal
    conditions:
      has: ["76-start"]
      missing: ["76-journal"]
    actions:
      - grant: "76-journal"
      - send_text: >-
          [the father's research — his obsession with the buried thing,
          the ring-struck symbol, and his worked-out method for waking the
          disc at the marked hearth; his daughter's name once, gently.
          Threshold-only: never name what the buried thing IS. ~120-180
          words, | block, first-person-of-the-journal or close narration.]
      - room_text: "reads a long while at the cottage table."
  # ── Ungated fallback (no 76-start): flavor. ──
  - event: room_interact
    room: 4023
    noun: journal
    conditions:
      missing: ["76-start"]
    actions:
      - send_text: >-
          A field journal lies open on the table, its pages dense with a
          careful obsessive hand and sketches of a mark you have half-seen
          elsewhere. Without knowing what you are looking for, it is only
          a lonely man's long notes.
```
(Author the bracketed `send_text` as real prose in the build — no placeholders in the shipped file.)
- [ ] **Step 4: Add the attunement trigger (4023) — the payoff.**
```yaml
  # ── The attunement at the hearth (4023) — has 76-journal + the inert disc, missing 76-end. ──
  - event: room_interact
    room: 4023
    noun: hearth-marks
    conditions:
      has: ["76-journal"]
      has_item: 40167
      missing: ["76-end"]
    actions:
      - grant: "76-attune"
      - grant: "76-end"
      - consume_item: 40167
      - give_item: 40168
      - send_text: >-
          [follow his method at the marked hearth — the disc warms, the
          lines take the light and keep it, the mark wakes. The quiet
          close: his daughter's name, the man who kept her in mind while
          chasing this to its end, the work finished by a stranger's hand.
          ~140-200 words, | block. Threshold-only.]
      - room_text: "does something careful at the marked hearth, and the air goes briefly strange."
  # ── Fallbacks: has 76-journal but no disc / no 76-journal. ──
  - event: room_interact
    room: 4023
    noun: hearth-marks
    conditions:
      has: ["76-journal"]
      missing_item: 40167
    actions:
      - send_text: "The marks are cut for a thing you do not have in hand. Without the disc, they are only scratches in old stone. (The disc lies in the old Reliquary at Pothole Coulee.)"
  - event: room_interact
    room: 4023
    noun: hearth-marks
    conditions:
      missing: ["76-journal"]
    actions:
      - send_text: "Marks are cut into the hearthstone in a deliberate pattern — a method for something, though you cannot read it yet."
```
- [ ] **Step 5: Boot-verify** — nuke instances, rebuild, boot; confirm `quests.LoadDataFiles loadedCount` rose by 1 (→66), `ValidateAllFlags OK` (76 tokens declared via the steps), 0 panics. **Step 6: Commit.**

## Task 4: Brennan's grant dialogue (post-Q75)

**Files:** Modify `dialogue/greenford/9516.yaml`

- [ ] **Step 1: Read `dialogue/greenford/9516.yaml`** (Brennan) — note the `tree.nodes` order and existing triggers.
- [ ] **Step 2: Add the Q76 grant node FIRST under `tree.nodes`** (gated grant nodes go first — short-trigger substring-shadowing). Follow the Quest NPC Dialogue SOP (`"quest"`+`"task"` in triggers) and the re-grant SOP (`questExcluded` includes the end token):
```yaml
    - id: disc-quest
      triggers: ["disc", "symbol", "ring", "mark", "key", "quest", "task"]
      questRequired: ["75-end"]
      questExcluded: ["76-start", "76-end"]
      grantsQuest: "76-start"
      text: |
        [Brennan recognizes the disc/symbol — it is a key, but inert.
        Names the scholar who went east from Ashwick to wake it and never
        returned (Maren's father). Directs the player: you will need the
        disc itself — it lies in the old Reliquary at Pothole Coulee — and
        his research waits in the cottage he left in Ashwick. First person,
        Brennan's voice, | block, no semicolons. ~120-160 words.]
      hints: "You could ask Brennan where the disc is, or about the man who
        went east to wake it."
```
(Author the bracketed `text` as real prose in the build.)
- [ ] **Step 3: Ensure discoverability + no shadowing** — the trigger words (`disc`/`symbol`/`ring`/`key`) must be discoverable (Brennan's root/greeting or a hint mentions the disc/symbol — add a line if needed). Confirm no earlier node's short trigger substring-matches these.
- [ ] **Step 4: Boot-verify + no dialogue warnings** (re-grant warning absent since `questExcluded` has `76-end`). **Step 5: Commit.**

## Task 5: End-to-end test + world-critic + finish

- [ ] **Step 1: World-critic pass** — dispatch a reviewer over the quest YAML + Brennan node + the two rooms + the two items: check the quest-dialogue SOPs (quest/task triggers, gated nodes first, questExcluded has 76-end, no semicolons, `|` blocks), lore boundary (the journal/attunement/Brennan NEVER name the buried thing — threshold-only), the universality logic (disc grant gated `missing_item: 40167`, reachable, Brennan directs), the recovery path (lost Attuned Disc → re-fetch inert → re-attune works), dead hint keywords, and that every referenced noun (`disc` 5343, `journal`+`hearth-marks` 4023) exists. Fix findings.
- [ ] **Step 2: Full boot verify** — `ValidateAllFlags OK`, quests +1, items +2, `ValidateZoneConsistency errors=0 mode=panic`, 0 panics.
- [ ] **Step 3: End-to-end harness test** — with a test char that has completed Q75: `ask brennan disc` (grant 76-start + directions) → travel to Pothole Coulee 5343, search + examine the disc (get Inert Disc 40167) → travel to Ashwick 4023, examine `journal` (76-journal) → examine `hearth-marks` with the disc (consume 40167, receive Attuned Disc 40168, 76-end, completion). Verify: the grant is gated on Q75; the disc grant is per-player-once (`missing_item`); the attunement requires the disc + journal; re-grant prevented; and the recovery path (drop the Attuned Disc, re-fetch inert, re-attune) works. Also confirm a player WITHOUT Q75 can't get the grant, and a fresh char can take the disc (universality). Capture the report under `tools/playtest/reports/`.
- [ ] **Step 4: Commit any fixes; update docs + memory** — mark the disc questline done; note it produces the Attuned Disc (40168) that #22's Threshold-Keeper consumes as the entry key. Commit.

---

## Self-review notes
- **Spec coverage:** Maren's-father link (journal/attunement lore) ✓; universal per-player disc grant (T3 disc trigger, `missing_item: 40167`) ✓; Brennan post-Q75 guide + directions (T4) ✓; father's journal + attunement room_interact at Ashwick 4023 (T2/T3) ✓; Inert→Attuned disc (T1, consume+give) ✓; reckoning-bone NOT a gate (absent from all conditions) — optional enrichment can be added as extra journal prose (noted, not required) ✓; recovery path (T3 note + T5 test) ✓; Q75 hard gate (T4 questRequired) ✓.
- **Placeholder note:** three `send_text`/`text` bodies are bracketed prose-specs — the BUILD must author them as real prose (they're content, not code); every other field is concrete. Flagged in each task.
- **Type/id consistency:** items 40167/40168, quest 76, tokens 76-start/journal/attune/end, rooms 5343/4023, Brennan 9516, Q75 gate 75-end — consistent across tasks.
- **Deferred:** the reckoning-bone optional lore beat (add to the journal prose if desired); #22 wiring (the Threshold-Keeper consuming 40168) belongs to the #22 zone plan.
