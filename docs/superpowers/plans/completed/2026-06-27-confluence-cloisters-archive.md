# The Confluence — District 6a: Cloisters & Archive (Q74 part 1) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Cloisters & Archive — the inner temple (16 rooms, ~7 senior/junior Keepers) — and the first half of Q74: grant the quest, gather three construction-history records, and earn the descent (stubbed to district 6b).

**Architecture:** Pure YAML content on the existing GoMud data-file engine, in `_datafiles/world/dogmud/{rooms,mobs,dialogue,quests,schedules}/the_confluence/` (quest file in `quests/`). A monastic cloister east of the public temple, entered via the 5b warded door (6183→6184). Q74 is a multi-step quest modeled exactly on the shipped Q73 (`quests/73-the_margin_notation.yaml`): a dialogue `grantsQuest` start, three quest-gated `room_interact` record-examinations, and a dialogue turn-in granting an in-progress `74-descent` token. The reveal, the `74-end` completion, and the `74-allegiance` flag-set are **district 6b** — this build declares the full step skeleton + the flag but never grants `74-end` (verified: rewards fire only on the step named `end`, `internal/hooks/Quest_HandleQuestUpdate.go:223`).

**Tech Stack:** YAML data files; `go run .` boot test; `cartcheck`/`ValidateZoneConsistency`; the mudagent playtest harness + `questtoken` admin.

**Spec:** `docs/superpowers/specs/2026-06-27-confluence-cloisters-archive-design.md`

**Reserved IDs (verified clean 2026-06-27):** rooms **6184–6199**, mobs/dialogue **9464–9470**, quest **74**, no new items, no new buffs.

**Branch:** create `feature/confluence-cloisters-archive` off `master` before Task 1.

---

## Authoring conventions (read once before any task)

Load-time-fatal / recurring-bug rules, identical to 5a/5b plus the quest SOPs:

1. **`character.name` AND room `title` MUST be canonical Title-Case** (mob casing panics; room-title casing also panics — 5b hit "over"→"Over"; small words `of/the/a/in` stay lowercase, others capitalize). Use the Unicode em-dash `—` (not `--`) in any title.
2. **`idlemessages` with a colon-space MUST be single-quoted**; `description`/`nouns` values with prose colons MUST use `>` block scalars.
3. **Dialogue node match = `strings.Contains(topic, trigger)` then quest gate.** Gated/grant nodes **FIRST**; avoid short triggers that substring-match other topics. `questRequired`/`questExcluded` are **LISTS**.
4. **Quest SOPs (from the Q73 build):**
   - A quest trigger may only **grant a DECLARED step token** (undeclared → "trigger grants unknown step" panic at load).
   - Every `grantsQuest` dialogue node includes the quest **end token in `questExcluded`** (e.g. the `74-start` node excludes `["74-start","74-end"]`), and includes `"quest"`+`"task"` in its `triggers`.
   - `room_interact` nouns are **ansi-highlighted hyphenated tokens** in the room prose, with a matching `nouns:` key; the quest YAML `noun:` matches that exact token.
   - Quest `rewards:` block uses **camelCase-free snake fields per the loader** — actually the loader uses yaml.v2: use `playermessage`, `roommessage`, `gold`, `itemid`, `rep_faction`, `rep_amount` exactly as Q73 does.
   - Declare a quest **flag** in the `flags:` block with a **bare key** (`allegiance`); references elsewhere use `74-allegiance`. Undeclared flag *references* panic; declaring-without-referencing is fine.
5. **Exits are just `roomid`** — no `kind:` field (mapper-derived).
6. **Faction:** clergy join the existing `keepers` faction via `groups: [humanoid, keepers]` (do not recreate the faction file).
7. **Pre-smoke SOP:** wipe `mobs.instances/*` + `rooms.instances/*` before every boot (not `shops/`).

Prose is authored by the building agent to the schema examples below and the voice of the existing Confluence files (5a/5b set + Q73's `dialogue/the_confluence/9441.yaml` for the quest-grant pattern). The plan fixes all IDs, coords, exits, nouns, quest tokens/gating.

---

## Task 1: Seam — open 6183→6184 + pay off the 5b Officiant/Warden hook

**Files:** Modify `rooms/the_confluence/6183.yaml`, `dialogue/the_confluence/9457.yaml` (Officiant), `dialogue/the_confluence/9459.yaml` (Threshold Warden).

- [ ] **Step 1: Wire the seam.** In `6183.yaml`, add an east exit to 6184. Current `exits:` is `west: {roomid: 6175}`. Final:

```yaml
exits:
  west:
    roomid: 6175
  east:
    roomid: 6184
```
Also lightly extend the `warded-door` noun / description so the door now reads as *passable for those the Keepers have admitted* (it is no longer an absolute dead-end — the inner cloister lies east through it), while staying gated in feel (the Warden still turns back the idle). Keep ≤80 cols.

- [ ] **Step 2: Officiant 73-end admit line (9457.yaml).** Add a node, placed FIRST (before `margin_aside`), gated `questRequired: ["73-end"]` and `questExcluded: ["74-start","74-end"]`, that — for a Q73-completed player who has not yet started Q74 — explicitly admits them inward: the senior Keepers will see them; go through the warded door to the cloisters and speak with Brother Cael in the Archive. No `grantsQuest` here (Q74 grants at Cael). Triggers: `["deeper","cloister","cloisters","senior","keeper","keepers","beyond","further","undercroft","question","records","margin","word"]`. Example:

```yaml
    - id: q74_admit
      questRequired: ["73-end"]
      questExcluded: ["74-start", "74-end"]
      triggers: ["deeper", "cloister", "cloisters", "senior", "keeper",
        "keepers", "beyond", "further", "undercroft", "question",
        "records", "margin", "word"]
      text: "You have followed this further than most, and the senior
        Keepers know it -- I have sent word. They will not turn you back
        at the door now. Go through to the cloisters, beyond the warded
        door off the great hall, and ask for Brother Cael in the Archive.
        He keeps the building records. If anyone will show you what the
        temple was raised on, it is him."
      hints: "You could go through the warded door to the cloisters and
        ask Brother Cael in the Archive about the temple's records."
```

- [ ] **Step 3: Warden 73-end admit line (9459.yaml).** Add a node placed FIRST (before its `margin_aside`), gated `questRequired: ["73-end"]`, that has the Warden step aside for a vouched Q73 player ("The Officiant sent word. Go through — the Archive is across the garth. I'll not stop you."). No grant. This makes the door's narrative gate consistent with the now-open exit.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_confluence/6183.yaml _datafiles/world/dogmud/dialogue/the_confluence/9457.yaml _datafiles/world/dogmud/dialogue/the_confluence/9459.yaml
git commit -m "feat(confluence): open the cloister seam (6183->6184) + 5b Q74-admit dialogue"
```

---

## Task 2: The 16 cloister rooms (6184–6199)

**Files (Create, in `rooms/the_confluence/`):** 6184–6199.

**Coordinate + exit table (authoritative; cartcheck verifies; every exit reciprocal; no `kind:`):**

| roomid | title | x | y | z | exits |
|--------|-------|---|---|---|-------|
| 6184 | The Cloister Gate | 6 | -77 | 0 | west→6183, east→6185 |
| 6185 | The Cloister Garth | 7 | -77 | 0 | west→6184, north→6186, south→6187, east→6192 |
| 6186 | The North Walk | 7 | -76 | 0 | south→6185, north→6190, west→6188, east→6189 |
| 6188 | The Chapter House | 6 | -76 | 0 | east→6186 |
| 6190 | The Scriptorium | 7 | -75 | 0 | south→6186 |
| 6189 | The Archive | 8 | -76 | 0 | west→6186, east→6195 |
| 6195 | Aldric's Study | 9 | -76 | 0 | west→6189 |
| 6192 | The East Walk | 8 | -77 | 0 | west→6185, east→6196, south→6193 |
| 6193 | The Infirmary | 8 | -78 | 0 | north→6192 |
| 6196 | The Prioress's Oratory | 9 | -77 | 0 | west→6192, south→6198 |
| 6198 | The Older East Corridor | 9 | -78 | 0 | north→6196, east→6199 |
| 6199 | The Descent Stairhead | 10 | -78 | 0 | west→6198 (+ described `down` stub — see below) |
| 6187 | The South Walk | 7 | -78 | 0 | north→6185, west→6191, south→6194 |
| 6191 | The Refectory | 6 | -78 | 0 | east→6187, south→6197 |
| 6197 | The Kitchen Court | 6 | -79 | 0 | north→6191 |
| 6194 | The Cells | 7 | -79 | 0 | north→6187 |

**6199 down-stub:** do NOT wire a `down` exit (the Undercroft is 6b). The room's PROSE describes a stair descending into the dark beneath the island, barred/quiet, "not for visitors" — a described stub. Biome `city` for all 16.

**Required quest nouns (the three Q74 records — ansi-highlighted hyphenated token in prose + a `nouns:` key with the BASE/non-quest lore; the quest YAML adds the gated grant + an ungated send_text in Task 4):**
- **6189 The Archive** → `building-ledger` — the temple's construction accounts; base noun lore = an old ledger of the temple's raising, oddly silent on the lowest courses.
- **6188 The Chapter House** → `consecration-record` — the founding consecration record; base lore = the Founders "consecrated" the site, wording that assumes something was already there.
- **6198 The Older East Corridor** → `masons-survey` — a surveyor's note; base lore = the lower courses here are older work, cut by hands the records don't name.

(Other flavor nouns at the agent's discretion.) **Spawninfo:** 6188→9464 (Aldric), 6189→9465 (Cael), 6196→9466 (Crane), 6190→9467 (scribe), 6191→9468 (cellarer), 6193→9469 (infirmarian), 6185→9470 (cloister keeper). `respawnrate: "20 real minutes"`.

- [ ] **Step 1: Author 6184–6199.** Worked example (a record room — note the base noun lore; the quest layers on top in Task 4):

```yaml
roomid: 6189
zone: The Confluence
title: The Archive
description: >
  The temple's archive, a long vaulted room of shelved
  ledgers and map-drawers, kept dim to spare the ink.
  Brother Cael works here among the building records --
  the accounts of the temple's raising, generation by
  generation. One bound
  <ansi fg="itemname">building-ledger</ansi>
  lies open on a reading-stand, turned to the earliest
  pages. The North Walk is back to the west; a low door
  east leads to the head Keeper's study.
biome: city
coord:
  x: 8
  y: -76
  z: 0
exits:
  west:
    roomid: 6186
  east:
    roomid: 6195
nouns:
  building-ledger: >
    The oldest of the temple's construction ledgers, its
    early pages accounting for stone, labor, and coin
    through the years of the raising. It is meticulous --
    and oddly silent on the lowest courses, as though the
    foundation were a thing received rather than built. A
    reader with a reason to look would find the silence
    louder than the figures.
spawninfo:
  - mobid: 9465
    respawnrate: "20 real minutes"
idlemessages:
- Brother Cael moves along the shelves, returning a
  ledger to its place with both hands.
- 'Dust hangs in the one shaft of window-light: the
  archive keeps its own slow weather.'
```

- [ ] **Step 2: Self-check** — every exit reciprocal (walk the table); no coord collisions; no `kind:`; 6199 has no `down` exit; the 3 record nouns present (hyphenated token + `nouns:` key) in 6189/6188/6198; spawninfo on the 7 rooms; titles Title-Case with `—`; colons handled; ≤80 cols.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_confluence/6184.yaml _datafiles/world/dogmud/rooms/the_confluence/6185.yaml _datafiles/world/dogmud/rooms/the_confluence/6186.yaml _datafiles/world/dogmud/rooms/the_confluence/6187.yaml _datafiles/world/dogmud/rooms/the_confluence/6188.yaml _datafiles/world/dogmud/rooms/the_confluence/6189.yaml _datafiles/world/dogmud/rooms/the_confluence/6190.yaml _datafiles/world/dogmud/rooms/the_confluence/6191.yaml _datafiles/world/dogmud/rooms/the_confluence/6192.yaml _datafiles/world/dogmud/rooms/the_confluence/6193.yaml _datafiles/world/dogmud/rooms/the_confluence/6194.yaml _datafiles/world/dogmud/rooms/the_confluence/6195.yaml _datafiles/world/dogmud/rooms/the_confluence/6196.yaml _datafiles/world/dogmud/rooms/the_confluence/6197.yaml _datafiles/world/dogmud/rooms/the_confluence/6198.yaml _datafiles/world/dogmud/rooms/the_confluence/6199.yaml
git commit -m "feat(confluence): Cloisters & Archive, 16 rooms (6184-6199)"
```

---

## Task 3: Mobs + dialogue (9464–9470)

**Files (Create):** mob + dialogue for 9464 (Aldric), 9465 (Cael), 9466 (Crane); mob + light/optional dialogue for 9467 (scribe), 9468 (cellarer), 9469 (infirmarian), 9470 (cloister keeper).

Common mob shape as in 5a/5b: `non_combatant: true`, `charm_immune: true`, `hostile: false`, `statpool: 30`, `maxwander: 0`, `activitylevel: 10`, `groups: [humanoid, keepers]`, Title-Case `character.name`, 4–6 idlecommands, a couple stats. `behavior_archetype: noncombat_passive`.

| mobid | name | room | role |
|-------|------|------|------|
| 9464 | Aldric the Keeper | 6188 | lid-keeper, head; conditions the descent (turn-in) |
| 9465 | Brother Cael | 6189 | sympathizer, archivist; **grants Q74** |
| 9466 | Prioress Crane | 6196 | believer; lore on the faith |
| 9467 | A Scribe | 6190 | ambient |
| 9468 | The Cellarer | 6191 | ambient |
| 9469 | The Infirmarian | 6193 | ambient |
| 9470 | A Cloister Keeper | 6185 | ambient |

- [ ] **Step 1: Brother Cael dialogue (9465.yaml) — the Q74 grantor.** Model on Q73's Quist (`dialogue/the_confluence/9441.yaml`). Nodes, in order:
  1. `q74_grant` (FIRST): `questRequired: ["73-end"]`, `questExcluded: ["74-start","74-end"]`, triggers include `"quest","task","help","records","ledger","building","temple","raised","foundation","deeper","cael","work"`, **`grantsQuest: "74-start"`**. Cael gives the thread: examine the building-ledger here, the consecration-record in the Chapter House, and the mason's survey in the older east corridor; then come back. (Frame Aldric's reluctant permission for a "supervised look.")
  2. `q74_progress`: `questRequired: ["74-start"]`, `questExcluded: ["74-survey"]`, triggers `["progress","remind","which","where","again","records","ledger","survey","record"]` — reminds which three records and where.
  3. `q74_turnin`: `questRequired: ["74-survey"]`, `questExcluded: ["74-descent"]`, triggers `["report","return","found","done","descent","stair","below","ready","back","seen"]`, **`grantsQuest: "74-descent"`** — confirms the three records; Aldric has agreed to a supervised descent; the stair is in the older east corridor (6198→6199); go down when ready. (Q74 now in-progress; the descent itself is 6b.)
  4. Non-quest lore nodes (the archive, the Margin, the building records) for players without 73-end.
  Skeleton for the grant node:

```yaml
    - id: q74_grant
      questRequired: ["73-end"]
      questExcluded: ["74-start", "74-end"]
      triggers: ["quest", "task", "help", "records", "ledger",
        "building", "temple", "raised", "foundation", "deeper",
        "cael", "work", "show"]
      text: "So the Officiant sent you, and you have the Margin's
        question in your teeth. Good -- I am tired of being the only
        one who reads these. Here is the work, and it is reading, not
        digging: three of the temple's own records disagree with its
        founding story. The building-ledger here, open on the stand.
        The consecration-record in the Chapter House. And a mason's
        survey left in the old east corridor, by the stair. See all
        three yourself. Aldric has allowed it -- a supervised look,
        he calls it. Then come back to me."
      hints: "You could examine the building-ledger here in the
        Archive, the consecration-record in the Chapter House, and
        the mason's survey in the older east corridor -- then return
        to Cael."
      grantsQuest: "74-start"
```

- [ ] **Step 2: Aldric dialogue (9464.yaml) — the lid-keeper.** The burdened head. For a Q74 player, a guarded node acknowledging he permits the descent but warns against it (he knows more than he says; threshold-only — never the why). Optional: a `questRequired: ["74-survey"]` node where he gives the conditional permission in his own voice (the turn-in can live on Cael; Aldric reinforces). No grant needed on Aldric if Cael handles the turn-in — but Aldric MUST feel like the gate (his permission is why the descent opens). Keep him sympathetic-to-the-player-but-guarding. Non-quest players get the official line + a sense of a man carrying weight.

- [ ] **Step 3: Prioress Crane dialogue (9466.yaml) — the believer.** Genuine, devout, troubled. Lore on the faith and the rite; a Q73-gated aside where her certainty shows a hairline crack ("I have prayed on the questions your sort bring. I do not have an answer that satisfies me. I have faith, which is not the same thing, and I have made my peace with the difference."). No grant. Threshold-only.

- [ ] **Step 4: Ambient (9467–9470).** Mob files + short/optional dialogue; atmospheric idlecommands (copying, provisioning, tending the sick, sweeping the garth).

- [ ] **Step 5: Self-check** — names Title-Case; questRequired/questExcluded LISTS; `q74_grant` and `q74_turnin` placed correctly (gated, grant nodes); the only `grantsQuest` values are `74-start` (Cael grant) and `74-descent` (Cael turn-in) — **NO `74-end`** anywhere (that's 6b); both grant nodes carry `"quest"`+`"task"` triggers and exclude `74-end`; hints second-person; ≤80 cols.

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/mobs/the_confluence/9464-aldric_the_keeper.yaml _datafiles/world/dogmud/mobs/the_confluence/9465-brother_cael.yaml _datafiles/world/dogmud/mobs/the_confluence/9466-prioress_crane.yaml _datafiles/world/dogmud/mobs/the_confluence/9467-a_scribe.yaml _datafiles/world/dogmud/mobs/the_confluence/9468-the_cellarer.yaml _datafiles/world/dogmud/mobs/the_confluence/9469-the_infirmarian.yaml _datafiles/world/dogmud/mobs/the_confluence/9470-a_cloister_keeper.yaml _datafiles/world/dogmud/dialogue/the_confluence/9464.yaml _datafiles/world/dogmud/dialogue/the_confluence/9465.yaml _datafiles/world/dogmud/dialogue/the_confluence/9466.yaml
git commit -m "feat(confluence): Cloisters NPCs + dialogue incl. Q74 grant/turn-in (9464-9470)"
```

---

## Task 4: The Q74 quest YAML

**Files:** Create `_datafiles/world/dogmud/quests/74-the_undercroft.yaml`.

Modeled exactly on `quests/73-the_margin_notation.yaml`. Declares the **full step skeleton** (start → ledger → record → survey → descent → reveal → end) so `descent` is NOT terminal; 6a wires only the three `room_interact` triggers (the dialogue grants start/descent live in Task 3). `reveal`/`end` are forward-declared for 6b (no 6a trigger grants them, so Q74 stays in-progress). The `allegiance` flag is declared (set in 6b). The `rewards:` block is forward-declared (fires only on `74-end` = 6b); 6b finalizes flag-conditional rep + any reward item.

- [ ] **Step 1: Write the quest file** (sequential records, Q73 pattern — each gated on the prior token; ungated fallback gives lore):

```yaml
questid: 74
name: The Undercroft
description: >-
  The Margin's question has led you under the temple's own roof. Brother Cael,
  the Confluence's archivist, has set you to examine three of the temple's own
  building records -- the ones that do not match its founding story -- and, if
  what you find warrants it, to earn the senior Keepers' leave to descend into
  the undercroft beneath the island.
secret: false

steps:
  - id: start
    description: "Brother Cael has set you to examine three of the temple's
      construction records that disagree with its founding story."
    hint: "Examine the building-ledger in the Archive, the consecration-record
      in the Chapter House, and the mason's survey in the older east corridor.
      Then return to Cael."
  - id: ledger
    description: "The building-ledger accounts for the temple's raising in
      meticulous detail -- and is oddly silent on the oldest, lowest courses,
      as if the foundation were received, not built."
    hint: "Now examine the consecration-record in the Chapter House, and the
      mason's survey in the older east corridor. Then return to Cael."
  - id: record
    description: "The consecration-record's own wording gives it away: the
      Founders consecrated the site -- language that assumes something was
      already standing there to consecrate."
    hint: "Now examine the mason's survey in the older east corridor, by the
      descent stair. Then return to Cael in the Archive."
  - id: survey
    description: "The mason's survey is plainest of all: the lower courses are
      older work, cut by hands the temple's records do not name. Three records,
      one conclusion -- the temple was raised on something older."
    hint: "Return to Brother Cael in the Archive with what you found."
  - id: descent
    description: "Cael has heard you out, and Aldric -- reluctantly -- has
      granted a supervised descent. The stair down lies in the older east
      corridor. Go down when you are ready."
    hint: "Descend the stair in the older east corridor when you are ready."
  # ── steps below are granted in district 6b (the Undercroft) ──
  - id: reveal
    description: "(The descent into the undercroft -- district 6b.)"
  - id: end
    description: "(The threshold reveal and its aftermath -- district 6b.)"

flags:
  - key: allegiance
    values: [margin, keepers]
    description: "Whether the player carries the truth to the Margin or keeps
      the Keepers' confidence. Set at the climax in the undercroft (6b)."

# rewards fire only on the `end` step (district 6b finalizes flag-conditional
# reputation and any reward item); declared here so the quest is well-formed.
rewards:
  playermessage: >-
    (Q74 completes in the undercroft -- district 6b.)
  gold: 0

triggers:
  # ── RECORD 1: the building-ledger (6189, the Archive) ──
  - event: room_interact
    room: 6189
    noun: building-ledger
    conditions:
      has: ["74-start"]
      missing: ["74-ledger"]
    actions:
      - grant: "74-ledger"
      - send_text: >-
          You read the building-ledger past its careful columns. It accounts
          for every stone of the temple's raising -- except the lowest courses,
          which it simply assumes, as though the foundation were a thing
          received rather than built. The first of the three records.
      - room_text: "reads the open building-ledger with sudden attention."
  - event: room_interact
    room: 6189
    noun: building-ledger
    conditions:
      missing: ["74-start"]
    actions:
      - send_text: >-
          An old construction ledger open on a reading-stand, meticulous and
          dry. Without a reason to read it closely, it is only an account book.
  # ── RECORD 2: the consecration-record (6188, the Chapter House) ──
  - event: room_interact
    room: 6188
    noun: consecration-record
    conditions:
      has: ["74-ledger"]
      missing: ["74-record"]
    actions:
      - grant: "74-record"
      - send_text: >-
          The consecration-record sets out the Founders' dedication of the
          temple -- and its own wording betrays it: they consecrated the site,
          a word that assumes something already stood here to be made holy. The
          second record agrees with the first.
      - room_text: "studies the framed consecration-record closely."
  - event: room_interact
    room: 6188
    noun: consecration-record
    conditions:
      missing: ["74-ledger"]
    actions:
      - send_text: >-
          The temple's consecration-record, framed in the Chapter House. A
          formal dedication in old script -- nothing you have reason to parse
          word by word.
  # ── RECORD 3: the mason's survey (6198, the Older East Corridor) ──
  - event: room_interact
    room: 6198
    noun: masons-survey
    conditions:
      has: ["74-record"]
      missing: ["74-survey"]
    actions:
      - grant: "74-survey"
      - send_text: >-
          The mason's survey is the plainest of the three: the lower courses of
          this corridor are older work, it notes, cut and laid by hands the
          temple's own records do not name. Three records, three different
          rooms, one conclusion. Time to report to Cael.
      - room_text: "examines the old mason's survey pinned in the corridor."
  - event: room_interact
    room: 6198
    noun: masons-survey
    conditions:
      missing: ["74-record"]
    actions:
      - send_text: >-
          A mason's survey-note pinned to the corridor wall, marking the
          stonework's courses. Technical, and old. You have no particular
          reason to study it.
```

- [ ] **Step 2: Self-check** — `questid: 74`, name matches the dialogue; every token a trigger grants (`74-ledger/record/survey`) is a **declared step**; the dialogue `grantsQuest` values (`74-start`, `74-descent`, Task 3) are declared steps; **no trigger or dialogue grants `74-end`**; the flag key is bare `allegiance`; rewards uses Q73's snake field names; the three `room:`/`noun:` pairs match the room nouns from Task 2 (6189/`building-ledger`, 6188/`consecration-record`, 6198/`masons-survey`).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/quests/74-the_undercroft.yaml
git commit -m "feat(confluence): Q74 The Undercroft quest (part 1 - grant, 3 records, descent)"
```

---

## Task 5: Anchor schedules

**Files (Create, in `schedules/the_confluence/`):** `cf_aldric.yaml` (9464), `cf_cael.yaml` (9465), `cf_crane.yaml` (9466); modify those three mob files to add `schedule_id`.

Day-at-post + night (model on `cf_historian.yaml`), full 24h coverage, findability-preserving. Keep Cael findable in/near the Archive (6189) through the day (he's the Q74 giver); Aldric in the Chapter House (6188) by day (study 6195 evenings is fine); Crane in the Oratory (6196). Ambient keepers (9467–9470) unscheduled.

- [ ] **Step 1: Write the three schedule files** (example — Cael, who must stay findable for Q74):

```yaml
id: cf_cael
description: "Brother Cael keeps the Archive through the day among the building
  records, then sleeps in a cell off the scriptorium overnight."
segments:
  - start: 6
    end: 22
    target_room: 6189
    activity: ""
    idlecommands:
      - emote returns a ledger to its shelf with both hands and squares it
        against the edge.
      - say The records are honest, if you read all of them. Most people read
        only the ones that agree with them.
      - emote turns a brittle page of the open building-ledger, careful at the
        corner.
  - start: 22
    end: 6
    target_room: 6189
    activity: sleeping
    idlecommands:
      - emote has banked the archive lamp and dozes in the chair by the
        reading-stand, a ledger still open before him.
```
Aldric: Chapter House 6188 by day, sleeping there at night (or his study). Crane: Oratory 6196 by day, sleeping at night.

- [ ] **Step 2: Wire `schedule_id`** into 9464/9465/9466 (top-level field, after `zone:`).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/schedules/the_confluence/cf_aldric.yaml _datafiles/world/dogmud/schedules/the_confluence/cf_cael.yaml _datafiles/world/dogmud/schedules/the_confluence/cf_crane.yaml _datafiles/world/dogmud/mobs/the_confluence/9464-aldric_the_keeper.yaml _datafiles/world/dogmud/mobs/the_confluence/9465-brother_cael.yaml _datafiles/world/dogmud/mobs/the_confluence/9466-prioress_crane.yaml
git commit -m "feat(confluence): Cloisters anchor schedules"
```

---

## Task 6: Boot test + cartcheck

- [ ] **Step 1: Wipe instances**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

- [ ] **Step 2: Build + boot** (`MapConsistencyEnforce: panic` already set):

```bash
go build ./... && go run . > /tmp/cf_cloister_boot.log 2>&1 &
```
Wait for load; scan the log. Expected: clean load lines + `mapper.ValidateZoneConsistency errors=0 warnings=0 mode="panic"`, and **`quests.LoadDataFiles ... loadedCount`** without panics. Watch specifically for: duplicate-coord panic; non-reciprocal-exit error; room-title/mob-name casing panic; **quest panics** — "trigger grants unknown step" (a trigger granting an undeclared token) or an undeclared-flag-reference panic; dialogue questRequired/list warning. Confirm **no WARN/ERROR references any 6a id (6184–6199 / 9464–9470 / quest 74)**.

- [ ] **Step 3: `cartcheck the_confluence`** (or rely on the boot validation) — zero collisions; the 6199 down-stub is fine (no wired down). Kill the server.

- [ ] **Step 4:** Fix any failure; re-run from Step 1; commit fixes.

```bash
git add -A && git commit -m "fix(confluence): Cloisters boot/cartcheck/quest-load fixes"
```

---

## Task 7: World-critic + feel-tester polish (MANDATORY) + Q74 harness-verify

- [ ] **Step 1: World-critic (data review)** over 6184–6199 + dialogue 9464–9466 + the Q74 YAML: direction/canon (the cloister is east of the public temple; the descent is down-and-beneath; Scholars' Quarter still north); dialogue node-shadowing (gated/grant nodes FIRST; `q74_grant`/`q74_turnin` not shadowed); lore-boundary (threshold-only — Aldric/Crane never state the why); the three record nouns hyphenated + matching the quest triggers; Title-Case (rooms+mobs); colons; no `kind:`; the Q74 step/trigger/flag wiring (no `74-end` grant in 6a).

- [ ] **Step 2: Feel-tester + Q74 mechanics harness walk** (per `tools/playtest/`, local). Kill ALL GoMud/go procs first. Walk: the 6183→6184 entry (the warded door now admits a vouched player), the cloister garth + branches, Cael/Aldric/Crane dialogue. **Verify Q74 end-to-end (6a half):** grant the Q73 token chain (`questtoken 73-start`→`73-map`→`73-end`) on the test char, then `ask cael quest`/`records` → **74-start grants**; `look building-ledger` (6189) → 74-ledger; `look consecration-record` (6188) → 74-record; `look masons-survey` (6198) → 74-survey; return to Cael → **74-descent grants**; confirm via `questtoken` admin that Q74 sits **in-progress at descent (NOT completed, no `74-end`)** and the 6199 stairhead describes the stub. (`questtoken` grants don't survive a force-killed server — grant the chain in the same session.) Save the report to `tools/playtest/reports/2026-06-27-local-feel-tester-confluence-cloisters.md`.

- [ ] **Step 3: Fix everything found**, re-boot (Task 6 Steps 1–2), commit.

```bash
git add -A && git commit -m "fix(confluence): Cloisters feel/world-critic polish"
```

---

## Task 8: Finish the branch

- [ ] **Step 1:** Confirm working tree clean + one final clean boot (instance wipe + boot, errors=0 mode=panic).
- [ ] **Step 2:** Use `superpowers:finishing-a-development-branch` — merge `feature/confluence-cloisters-archive` into `master` with `--no-ff`; delete the branch. Do NOT push to prod.
- [ ] **Step 3:** Update memory: append the 6a outcome to `project_confluence_build.md` + the MEMORY.md index; note any new gotchas (esp. anything learned wiring a split quest). Flag for 6b: the descent (6199 down), the undercroft rooms (6200–6217), the `reveal`/`end` steps + the `74-allegiance` flag-set + the reward finalization.

---

## Self-review (completed by plan author)

- **Spec coverage:** §2 entry+seam+Officiant payoff → T1; §3/§4 rooms → T2; §5 NPCs → T3; §6 Q74 (grant/records/turn-in/descent, flag declared) → T3 (dialogue grants) + T4 (quest YAML); §8 schedules → T5; §9 verification → T6/T7; §7 items=none. All covered.
- **Placeholder scan:** none — the quest YAML is given in full; rooms/dialogue have worked examples + field-level briefs; `reveal`/`end`/rewards are deliberately forward-declared for 6b with explicit markers (not TODOs).
- **ID/type consistency:** room ids/coords/exits cross-checked (T2 table reciprocal); the three quest `room`/`noun` pairs (6189/`building-ledger`, 6188/`consecration-record`, 6198/`masons-survey`) match between T2 nouns and T4 triggers; quest tokens consistent (start/ledger/record/survey/descent granted in 6a; reveal/end only declared); dialogue `grantsQuest` values (`74-start`,`74-descent`) are declared steps; `73-end` gate on the grant matches shipped Q73; `keepers` faction reused; mob ids (9464–9470) match T2 spawninfo and T3.
