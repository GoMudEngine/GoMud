# The Confluence — District 4: The Scholars' Quarter (+ Quest 73) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Confluence's first quest district — the 14-room Scholars' Quarter (6218–6231) and **Quest 73 "The Margin Notation"**, a linear investigation that gathers three disagreeing map records (the bookseller's chart + Tallis's charts at 6131 + the hall map at 6143 via quest-gated `look` triggers) and synthesizes "there was a fourth," pointing to the temple (the Q74 hook).

**Architecture:** Content build (room/mob/item/dialogue YAML) in the existing `the_confluence` zone, plus a **quest YAML** with `room_interact` triggers and a `grantsQuest` start/end dialogue chain. No new faction (`margin` exists). Cross-district wiring adds quest-gated highlighted nouns to two already-built rooms (6131, 6143). Validated by boot + `cartcheck` + an admin quest run.

**Tech Stack:** GoMud engine; YAML under `_datafiles/world/dogmud/`; quest engine (`quests/*.yaml`); `cartcheck`; `questtoken` admin command.

**Spec:** `docs/superpowers/specs/2026-06-26-confluence-scholars-quarter-design.md`
**Umbrella:** `docs/superpowers/specs/2026-06-26-confluence-citywide-design.md`

---

## Reference patterns (read before authoring)

- **Quest YAML (steps/rewards/room_interact triggers, with the non-quest fallback trigger):** `_datafiles/world/dogmud/quests/69-the_gallery_cipher.yaml` (THE model — room_interact `look <noun>` gated by has/missing, grant + send_text + room_text; plus a `missing:[start]` fallback). Also `70-the_pre_founding_web.yaml`, `14-the_undertow.yaml` (step + reward block shape).
- **Quest-grant dialogue (start + a non-start grant node):** any NP quest giver, e.g. `dialogue/new_plymouth_*/` with `grantsQuest` (grep `grantsQuest` shows "-start" and "-end"/mid tokens both work).
- **`city`-biome rooms + a vendor + a `margin` NPC:** `rooms/the_confluence/6143.yaml` (a hall), `mobs/the_confluence/9429-lenne_the_provisioner.yaml` (general vendor), `mobs/the_confluence/9435-savel_the_margin_scholar.yaml` + `dialogue/.../9435.yaml` (a `margin` scholar). Highlighted noun example: `rooms/the_confluence/6135.yaml` / `6118.yaml`.
- **Item:** `items/materials-40000/40125-smoked_river_fish.yaml`.

### Cross-cutting GOTCHAS

1. Zone folder `the_confluence/` exists; each file's `zone:` = `The Confluence`. **`biome: city`** for all 14 rooms.
2. **`nouns` + multi-sentence `description` use `>` block scalars; `idlemessages` with a colon-space MUST be double-quoted.**
3. **Mob `character.name` MUST be canonical Title Case** (the porter casing panic).
4. **Coords proposed — assign final + `cartcheck the_confluence`** against the Square (6138–6153) and all prior. Reciprocal exits.
5. **room_interact fires on `look <highlighted-hyphenated-noun>`, gated by quest tokens.** Each source needs an **ansi-highlighted** noun (`<ansi fg="itemname">two words</ansi>` → the player types `look two-words`). Author them: bookseller `damaged-chart` (6226), Tallis `old-charts` (added to 6131), hall `marginal-note` (added to 6143 — do NOT reuse the bare "map", which collides with a player's carried "Copy of Edvar's Map" item).
6. **Quest SOP:** the giver's start node needs `grantsQuest: "73-start"`, `questExcluded: ["73-start","73-end"]` (BOTH), and `"quest"`+`"task"` in its triggers. A quest trigger may only **grant a DECLARED step token**. Reward-block keys are **NO-underscore** (`playermessage`/`gold`/`itemid`/`rep_faction`/`rep_amount`).
7. **Place gated grant nodes FIRST in `tree.nodes`** (the substring-shadow lesson) and avoid short lore triggers that substring-match the grant topic.
8. **Instance saves** — wipe before smoke test (NOT `shops/`).

---

## Task 1: Branch + existing-room edits (seam + the two cross-district quest nouns)

**Files:**
- Modify: `rooms/the_confluence/6149.yaml` (seam), `6131.yaml` (Tallis stall — add `old-charts` noun), `6143.yaml` (Municipal Hall — add `marginal-note` noun).

- [ ] **Step 1: Branch**

```bash
git checkout master && git pull --ff-only 2>/dev/null; git checkout -b feature/confluence-scholars-quarter
```

- [ ] **Step 2: Open the 6149 seam**

Edit `rooms/the_confluence/6149.yaml`: add `west: {roomid: 6218}` to `exits:` (keep `east: 6143`). Revise the description so a doorway leads **west** into the Scholars' Quarter (the archive backs onto the scholars' halls).

- [ ] **Step 3: Add the `old-charts` highlighted noun to 6131 (Tallis's stall)**

In `rooms/the_confluence/6131.yaml`: in the description, wrap a phrase as
`<ansi fg="itemname">old charts</ansi>` (the older, disagreeing charts on the
shelves). Add a noun keyed `the old charts: >` describing them — the disagreeing
records, a few showing a fourth channel (the player examines THIS for Q73). Keep
the existing `the charts` noun.

- [ ] **Step 4: Add the `marginal-note` highlighted noun to 6143 (Municipal Hall)**

In `rooms/the_confluence/6143.yaml`: in the description (or the existing `the map`
noun body), add `<ansi fg="itemname">marginal note</ansi>` referring to the
older, struck-through fourth-channel annotation. Add a noun `the marginal note: >`
describing it (the fourth channel pencilled and half-struck, "the surveyors could
not agree"). Leave the praised `the map` noun intact.

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_confluence/6149.yaml _datafiles/world/dogmud/rooms/the_confluence/6131.yaml _datafiles/world/dogmud/rooms/the_confluence/6143.yaml
git commit -m "feat(confluence): Scholars' Quarter seam (6149 west) + Q73 quest nouns (old-charts 6131, marginal-note 6143)"
```

---

## Task 2: The Scholars' Quarter rooms (6218–6231)

**Files:**
- Create: `rooms/the_confluence/6218.yaml` … `6231.yaml`

`zone: The Confluence`, **`biome: city`**, block-scalar nouns, quoted colon
idlemessages, 3 idlemessages each (scholarly flavor: turning pages, a quill's
scratch, dust in lamplight, a muttered argument, the quiet). spawninfo per the table.

| Room | Title | Coord | Mob | Content brief |
|------|-------|-------|-----|---------------|
| 6218 | The Scholars' Gate | {-8,-67,0} | 9447 | Seam (east→6149); the Gate-Porter; the quarter begins — ink-and-vellum quiet after the civic bustle |
| 6219 | Inkwell Court | {-9,-67,0} | 9448 | Central court; a Student; benches, a dry fountain, scholars crossing with armfuls of paper |
| 6220 | The Margin Hall | {-10,-67,0} | 9441 | The Margin's meeting hall; **the Q73 giver** (the old cartographer-historian); maps pinned floor to ceiling |
| 6221 | The Great Archive | {-11,-67,0} | 9442 | The library / source surveys; the Archivist; tall stacks, the smell of old vellum |
| 6230 | The Deep Stacks | {-12,-67,0} | — | The oldest surveys (west end); **a locked case** — `look case` noun (foreshadows the temple's sealed records; no access) |
| 6222 | A Study Hall | {-8,-66,0} | — | Reading desks, a scholar's clutter (north) |
| 6223 | The Cartographers' Room | {-9,-66,0} | 9443 | A map study; the Cartographer-Scholar; instruments, draughting tables (north) |
| 6224 | A Lecture Room | {-10,-66,0} | — | Tiered benches, a slate, chalk-dust (north) |
| 6225 | The Reading Room | {-11,-66,0} | — | Quiet stacks, locked reference cases (north) |
| 6226 | The Bookseller's | {-8,-68,0} | 9444 | The Bookseller (vendor); **the damaged chart** — a `<ansi fg="itemname">damaged chart</ansi>` highlighted noun keyed `the damaged chart: >` (a Q73 source — shows a fourth channel; describe it; the room_interact will gate on it) (south) |
| 6227 | Scholars' Lodgings | {-9,-68,0} | 9445 | Where scholars sleep and argue; a Margin Scholar (south) |
| 6228 | The Copyists' Room | {-10,-68,0} | 9446 | Copy-work (a peer of Tallis); the Copyist (south) |
| 6229 | The Quiet Garden | {-11,-68,0} | — | A contemplative court; a fig tree, a worn bench (south) |
| 6231 | The Garden Walk | {-12,-68,0} | — | A secluded walk; a restrained `look` orbital-mark on an old garden stone (south) |

Exit skeleton (finalize + cartcheck):
```
6218 e->6149(Hall of Records) w->6219 n->6222 s->6226
6219 e->6218 w->6220 n->6223 s->6227
6220 e->6219 w->6221 n->6224 s->6228
6221 e->6220 w->6230 n->6225 s->6229
6230 e->6221
6222 s->6218
6223 s->6219
6224 s->6220
6225 s->6221
6226 n->6218
6227 n->6219
6228 n->6220
6229 n->6221 s->6231
6231 n->6229
```

- [ ] **Step 1: Author 6218–6221, 6230** (the E–W spine: gate, court, Margin Hall, Archive, Deep Stacks + locked case).
- [ ] **Step 2: Author 6222–6225** (the north study rooms).
- [ ] **Step 3: Author 6226–6229, 6231** (the south: bookseller w/ the `damaged chart` noun, lodgings, copyists, garden + walk). The bookseller's `damaged-chart` noun is load-bearing for Q73.
- [ ] **Step 4: Cross-check** exits reciprocal; coords; 6149↔6218 seam.
- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_confluence/621*.yaml _datafiles/world/dogmud/rooms/the_confluence/622*.yaml _datafiles/world/dogmud/rooms/the_confluence/623*.yaml
git commit -m "feat(confluence): Scholars' Quarter rooms 6218-6231 (Margin Hall, Archive, Bookseller, garden)"
```

---

## Task 3: Items (40137 book + 40138 Q73 reward)

**Files:**
- Create: `items/materials-40000/40137-a_confluence_history.yaml`
- Create: `items/materials-40000/40138-fair_copy_of_the_compiled_survey.yaml`

Model on `40125-smoked_river_fish.yaml`. (40134–40136 are taken by the tavern.)

- [ ] **Step 1: Author the two items**

- **40137 A Confluence History** — the Bookseller's vendor good. A bound history
  of the Confluence (flavor reading). `is_component: true`, `component_tag:
  history-book`, `type: object`, `subtype: mundane`, weight ~0.5, value ~6,
  `vendor_categories: [tailoring]` (paper/binding).
- **40138 A Fair Copy of the Compiled Survey** — the **Q73 reward** (lore item).
  A clean copy the Margin made of the three reconciled records, the fourth
  channel drawn back in. **NOT salable** — omit `vendor_categories` (it's a quest
  keepsake), `is_component: false`. `type: object`, `subtype: mundane`, weight
  ~0.2, value ~0.

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/items/materials-40000/4013{7,8}-*.yaml
git commit -m "feat(confluence): Scholars' goods — A Confluence History (40137) + Q73 reward survey (40138)"
```

---

## Task 4: NPCs + dialogue (9441–9448), incl. the Q73 giver chain

**Files:**
- Create: `mobs/the_confluence/9441-...yaml` … `9448-...yaml`
- Create: `dialogue/the_confluence/9441.yaml` … (speakers)

All 8: `non_combatant: true`, `hostile: false`, `charm_immune: true`,
`speciesid: 1`, `level: 1`, `maxwander: 0`, `statpool: 30`, `activitylevel: 10`.
Non-vendors `behavior_archetype: noncombat_passive`. **Unique, Title-Case names.**
`groups: [humanoid, margin]` for the scholars (9441/9442/9443/9445/9446); the
Bookseller 9444 and Student 9448 + Gate-Porter 9447 are `[humanoid]`.

| Mob | Role | Room | Notes |
|-----|------|------|-------|
| 9441 | The Q73 Giver (senior Margin scholar) | 6220 | Old cartographer-historian, failing eyesight. **Q73 grant + synthesis chain** (below). `margin`. |
| 9442 | The Archivist | 6221 | The source surveys; scholarly, helpful. `margin`. |
| 9443 | The Cartographer-Scholar | 6223 | Map-work; river-survey lore (**canon directions: Aldren N, Brenn E, Solt SW**). `margin`. |
| 9444 | The Bookseller | 6226 | **Vendor** (`craft_support: general`, `noncombat_shopkeeper`, `gold: 40`, shop `[itemid: 40137]`); Margin-sympathetic tradesperson; the damaged chart is "not for sale, just old." `[humanoid]`. |
| 9445 | A Margin Scholar | 6227 | The open question; the temple's official account; restrained. `margin`. |
| 9446 | A Copyist | 6228 | Copy-work, the chart trade (a peer of Tallis). `margin`. |
| 9447 | The Gate-Porter | 6218 | Directs visitors (short tree). `[humanoid]`. |
| 9448 | A Student | 6219 | Ambient (idlecommands). `[humanoid]`. |

- [ ] **Step 1: Author 9442/9443/9444/9445/9446/9447/9448** with dialogue
  (vendor + scholars + ambient). The Cartographer (9443) reinforces canon river
  directions; the Bookseller (9444) sells 40137 and points at the damaged chart.

- [ ] **Step 2: Author the Q73 giver (9441) + its dialogue chain.** Put the
  **two grant nodes FIRST** under `tree.nodes` (substring-shadow SOP). Greetings +
  patterns advertise the quest. Two key nodes:
  - **start node:** `grantsQuest: "73-start"`, `questExcluded: ["73-start", "73-end"]`,
    `triggers: ["quest", "task", "notation", "fourth", "waters", "help", "work", ...]`.
    Text: frames the four-waters question; asks the player to examine the three
    disagreeing records — **the bookseller's damaged chart here, then Tallis's old
    charts at the Long Quay, then the marginal note on the hall map** — and return.
  - **synthesis/end node:** `questRequired: "73-map"` (the last examination token —
    implies the prior two in the ordered chain), `grantsQuest: "73-end"`,
    `questExcluded: ["73-end"]`, `triggers: ["return", "found", "done", "report",
    "evidence", "back", ...]`. Text: lays the three side by side; **the synthesis —
    there *was* a fourth, struck from the record over generations; the temple sits
    on what it suppressed** — and points the player to the temple's oldest stonework
    (the Q74 hook). (The reward fires from the quest YAML's `rewards` on `73-end`.)

- [ ] **Step 3: Verify** filenames match `ConvertForFilename`; Title-Case names;
  groups correct; spawninfo placements match Task 2.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/mobs/the_confluence/ _datafiles/world/dogmud/dialogue/the_confluence/
git commit -m "feat(confluence): Scholars' NPCs 9441-9448 + dialogue + the Q73 giver grant/synthesis chain"
```

---

## Task 5: Quest 73 — The Margin Notation (the quest YAML)

**Files:**
- Create: `_datafiles/world/dogmud/quests/73-the_margin_notation.yaml`

Model **exactly** on `quests/69-the_gallery_cipher.yaml`.

- [ ] **Step 1: Author the quest YAML**

Structure:
```yaml
questid: 73
name: The Margin Notation
description: >-
  The Margin's scholars keep finding the same disagreement in the old maps of
  the Confluence -- the count of the waters at the junction. Some of the oldest
  surveys show a fourth channel, struck out and re-drawn over generations. A
  senior scholar of the Margin has asked you to examine the three records that
  disagree, and bring back what you find.
secret: false

steps:
  - id: start
    description: "A senior Margin scholar at the Margin Hall has set you to
      examine the three disagreeing records of the Confluence's waters."
    hint: "Examine the bookseller's damaged chart (here in the Scholars'
      Quarter), then Tallis's old charts at the Scriveners' Stall on the Long
      Quay, then the marginal note on the Municipal Hall map at Tri-Cross
      Square."
  - id: chart
    description: "The bookseller's damaged chart shows a fourth channel at the
      junction, drawn as a distinct waterway."
    hint: "Now examine Tallis's old charts at the Scriveners' Stall on the Long
      Quay, then the marginal note on the Municipal Hall map."
  - id: scrivener
    description: "Tallis's oldest charts agree -- a fourth channel, struck and
      re-marked across sixty years of surveys."
    hint: "Now examine the marginal note on the Municipal Hall map at Tri-Cross
      Square, then return to the Margin Hall."
  - id: map
    description: "The Municipal Hall's own map carries the fourth channel in its
      margin, half-struck-through. Three independent records, the same
      disagreement."
    hint: "Return to the senior scholar at the Margin Hall with what you found."
  - id: end
    description: "The Margin has laid the three records side by side. There was a
      fourth water, struck from the record over generations. The temple at the
      heart of the city sits on what it suppressed -- and the oldest stonework
      there is where the truth is buried."

rewards:
  playermessage: >-
    "Three records, three hands, sixty years apart, and they agree on the one
    thing the official account denies. There was a fourth. Someone unmade it --
    not the water, the memory of it." The old scholar folds the compiled survey
    and presses it on you. "Take the fair copy. And if you mean to follow it
    further, the answer is not in any archive of ours. It is under the temple,
    in the stone the founders built on and would not explain."
  gold: 30
  itemid: 40138
  rep_faction: margin
  rep_amount: 15

triggers:
  # ── SOURCE 1: bookseller's damaged chart (6226) ──
  - event: room_interact
    room: 6226
    noun: damaged-chart
    conditions:
      has: ["73-start"]
      missing: ["73-chart"]
    actions:
      - grant: "73-chart"
      - send_text: >-
          You study the damaged chart. Past the water-stain, at the junction,
          a fourth channel is drawn -- narrow, distinct, no mere smear of
          current. One of three records you were sent to find.
      - room_text: "studies the bookseller's damaged chart with new attention."
  - event: room_interact
    room: 6226
    noun: damaged-chart
    conditions:
      missing: ["73-start"]
    actions:
      - send_text: >-
          A water-damaged old harbor chart, half its detail lost to a stain.
          Interesting, but you have no particular reason to study it.
  # ── SOURCE 2: Tallis's old charts (6131, Long Quay) ──
  - event: room_interact
    room: 6131
    noun: old-charts
    conditions:
      has: ["73-chart"]
      missing: ["73-scrivener"]
    actions:
      - grant: "73-scrivener"
      - send_text: >-
          Tallis's oldest charts show the same fourth channel -- struck out in
          some hands, re-marked in others, across sixty years of copies. The
          disagreement is consistent. Two records now.
      - room_text: "pores over Tallis's oldest charts."
  - event: room_interact
    room: 6131
    noun: old-charts
    conditions:
      missing: ["73-chart"]
    actions:
      - send_text: >-
          Stacks of old hand-copied charts, a few showing the river-junction
          oddly. A curiosity of the trade.
  # ── SOURCE 3: the Municipal Hall marginal note (6143, Tri-Cross Square) ──
  - event: room_interact
    room: 6143
    noun: marginal-note
    conditions:
      has: ["73-scrivener"]
      missing: ["73-map"]
    actions:
      - grant: "73-map"
      - send_text: >-
          The hall's own map carries it too -- a fourth channel pencilled into
          the margin in an older hand, half-struck-through, the note beside it
          reading that the surveyors could not agree. Three independent records.
          Time to report back.
      - room_text: "examines the marginal note on the hall map closely."
  - event: room_interact
    room: 6143
    noun: marginal-note
    conditions:
      missing: ["73-scrivener"]
    actions:
      - send_text: >-
          An old marginal annotation on the civic map -- a fourth channel,
          struck through. The surveyors could not agree, the note says.
```
(The `73-end` step is granted by the giver's synthesis dialogue node — Task 4 Step 2 — which fires the `rewards` block. Declare ALL step ids: start/chart/scrivener/map/end. The giver's grant nodes reference `73-start` and `73-end`.)

- [ ] **Step 2: Commit**

```bash
git add _datafiles/world/dogmud/quests/73-the_margin_notation.yaml
git commit -m "feat(confluence): Quest 73 The Margin Notation — gather 3 disagreeing records, synthesize the fourth"
```

---

## Task 6: Smoke test, quest test, cartcheck, docs, merge

**Files:**
- Modify: `docs/ZONE_EXPANSION.md`

- [ ] **Step 1: Wipe instance saves** (NOT `shops/`)

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

- [ ] **Step 2: Build + boot.** Run: `go build -o C:/tmp/sq.exe . && C:/tmp/sq.exe`.
  Expected: no panics; `rooms.loadAllRoomZones` loadedCount=**1084**; `mobs` +8;
  `quests.LoadDataFiles` +1 (=Q73); **`ValidateZoneConsistency errors=0 mode=panic`**.
  A coord collision, casing/filename mismatch, unquoted colon, an **undeclared
  quest step token** (panic "trigger grants unknown step"), or a missing
  `questExcluded` end token (load warning) shows here — fix and re-boot.

- [ ] **Step 3: `cartcheck the_confluence`** — expect clean.

- [ ] **Step 4: Quest run (admin — reliable per the NP-questline lesson; the
  harness adapter is flaky).** Connect, ASCII charset. `teleport 6220`; `ask
  <giver> quest` → confirm Q73 grants (`questtoken 73-start`). Examine the sources
  in order: `teleport 6226` + `look damaged-chart` (→ `questtoken 73-chart`);
  `teleport 6131` + `look old-charts` (→ `73-scrivener`); `teleport 6143` + `look
  marginal-note` (→ `73-map`). Confirm each `look` WITHOUT the quest shows only
  the lore fallback (test with `questtoken` to see tokens). `teleport 6220`; `ask
  <giver> return` → synthesis fires, `73-end` granted, reward (the survey 40138 +
  `margin` rep via `questtoken flags`/inventory). Walk the quarter; `buy` from the
  Bookseller; check the Deep Stacks locked case + the garden stone.

- [ ] **Step 5: Update `docs/ZONE_EXPANSION.md`** — Confluence row (#17): district
  4 (Scholars' Quarter) built + **Q73 done**, "Building (4/10)", rooms 6218–6231,
  world 40 zones / 1084 rooms; Next = Processional + Temple.

```bash
git add docs/ZONE_EXPANSION.md
git commit -m "docs(zones): The Confluence district 4 (Scholars' Quarter + Q73) built"
```

- [ ] **Step 6: Merge `--no-ff`**

```bash
git checkout master
git merge --no-ff feature/confluence-scholars-quarter -m "Merge: The Confluence district 4 — The Scholars' Quarter + Quest 73"
git branch -d feature/confluence-scholars-quarter
git tag -d master 2>/dev/null || true
```

---

## Self-review checklist (run before merge)

- [ ] 14 rooms 6218–6231, `zone: The Confluence`, `biome: city`, coords cartcheck-clean; 6149→6218 seam.
- [ ] The three Q73 source nouns are ansi-highlighted: `damaged-chart` (6226), `old-charts` (6131), `marginal-note` (6143).
- [ ] Q73: 5 declared steps (start/chart/scrivener/map/end); giver start node has `grantsQuest 73-start` + `questExcluded [73-start,73-end]` + quest/task triggers; synthesis node `questRequired 73-map` + `grantsQuest 73-end`; grant nodes FIRST in tree.nodes.
- [ ] Each room_interact has a `missing:[prev-token]` gated grant AND a `missing:[73-start]` lore fallback.
- [ ] Reward block uses no-underscore keys; `itemid: 40138`, `rep_faction: margin`.
- [ ] Mob names Title Case; the Scholar 9441/9442/9443/9445/9446 carry `margin`; vendor 40137 stocks under `general`.
- [ ] Boot clean, `cartcheck` clean, Q73 grant→3 sources→synthesis→reward verified by `questtoken`.
