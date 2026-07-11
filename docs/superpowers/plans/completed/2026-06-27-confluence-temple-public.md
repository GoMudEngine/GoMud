# The Confluence — District 5b: The Temple of Confluence (public) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the public Temple of the Confluence — the worship interior on the island (16 rooms, ~7 NPCs, threshold-only Q74 setup, no quest) — entered through the now-open Processional portico doors, and verify it boots clean and plays well.

**Architecture:** Pure YAML content on the existing GoMud data-file engine, in the existing zone folder `_datafiles/world/dogmud/{rooms,mobs,dialogue,schedules}/the_confluence/`. An axial temple: portico (revised) → narthex → nave → the crossing/great-hall-over-the-waters → sanctuary, with side aisles, three river-chapels + a disused fourth alcove, reliquary, sacristy, almonry, a gallery (vertical), and a Keeper-warded inner threshold stubbing east to the future Cloisters. Verification is boot-with-panic-mode + `cartcheck` + a world-critic/feel-tester pass. No Go code, no unit test.

**Tech Stack:** YAML data files; `go run .` boot test; `cartcheck` / `ValidateZoneConsistency`; the mudagent playtest harness.

**Spec:** `docs/superpowers/specs/2026-06-27-confluence-temple-public-design.md`

**Reserved IDs (verified clean 2026-06-27):** rooms **6168–6183**, mobs/dialogue **9457–9463**, no new items (reuse), no new quest/buffs.

---

## Authoring conventions (read once before any task)

Load-time-fatal or recurring-bug rules, identical to the 5a build:

1. **Mob `character.name` MUST be canonical Title Case** ("The Officiant", "A Kneeling Worshipper"). Filename lowercase via `ConvertForFilename`. Non-canonical name panics at boot.
2. **Room `idlemessages` with a colon-space MUST be single-quoted**; `description`/`nouns` values with prose colons MUST use `>` block scalars.
3. **Dialogue node match is `strings.Contains(topic, trigger)` (substring), THEN the quest gate.** Place gated/specific nodes FIRST in `nodes:`; avoid short triggers that substring-match other topics. `questRequired`/`questExcluded` are **LISTS** (`["73-end"]`), never bare strings (a string silently kills the NPC's dialogue).
4. **NO `grantsQuest`/`givesItem`/`setsQuestFlag` anywhere in this district** (it grants nothing; Q74 is district 6).
5. **Exits are just `roomid`** — do NOT author a `kind:` field. Exit kind is mapper-derived from coord delta.
6. **Highlighted nouns:** wrap the clickable token in `<ansi fg="itemname">token</ansi>` in the description and add a matching key under `nouns:`. **Multi-word noun keys are HYPHENATED** in both the ansi token and the key (`offering-brazier`, `oldest-relic`), for clean `look` + codebase consistency.
7. **Proper vertical** = `up`/`down` exits + stacked coords (same x,y, z±1).
8. **River/compass canon:** Aldren = NORTH tributary, Brenn = EAST, Solt = SOUTHWEST; combined water spills SOUTHWEST; Scholars' Quarter is upriver/NORTH. The three river-chapels must match their rivers' source directions in prose. Double-check every direction word.
9. **Faction:** clergy join the existing `keepers` faction via `groups: [humanoid, keepers]` (the faction file already exists from 5a — do not recreate it; do not create `factions.rep/keepers.yaml`).
10. **Pre-smoke SOP:** `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*` before every boot/smoke (do NOT touch `shops/`).

Prose is authored by the building agent following the schema examples below and the
voice of the existing Confluence files (the 5a Processional set:
`mobs/the_confluence/9449-the_historian.yaml`, `dialogue/the_confluence/9449.yaml`,
`rooms/the_confluence/6162.yaml`, `6172`-area causeway rooms). The plan fixes all
IDs, coords, exits, nouns, gating, and faction membership.

**Branch:** create `feature/confluence-temple-public` off `master` before Task 1
(do NOT build on master).

---

## Task 1: Revise the portico (6167) — open the doors inward

**Files:** Modify `_datafiles/world/dogmud/rooms/the_confluence/6167.yaml`

The 5a portico stubs the great doors shut and redirects pilgrims to "the island's far
side" (a dead-end the feel-review flagged). This task opens them as the temple
entrance.

- [ ] **Step 1: Add the south exit.** Current `exits:` is `north: {roomid: 6166}`. Final:

```yaml
exits:
  north:
    roomid: 6166
  south:
    roomid: 6168
```

- [ ] **Step 2: Open the doors in the description.** The description currently says the
great-doors are "shut against the day." Change that clause so the doors **stand open
through the day onto the temple's nave within** (keep the `weathered-symbol` keystone
text above them unchanged; keep the Margin-scholar-at-the-column line unchanged; keep
"The stair descends to the north."). Keep prose ≤80 cols, `>` block scalar.

- [ ] **Step 3: Rewrite the `great-doors` noun.** Remove the "not accessible from this
approach / pilgrims are directed to the processional entrance on the island's far
side" text entirely. New noun describes the doors standing open by day for worship
(pilgrims passing through into the nave), barred only after the last rite at night.
Example:

```yaml
  great-doors: >
    The great doors of the Confluence temple stand open through
    the day, folded back against the portico's inner wall so the
    cool of the nave reaches the top of the stair. Their timber is
    old, bound with iron darkened near to black under the overhang.
    By day they are an invitation; the Keepers bar them from within
    only after the last rite, when the temple is given back to the
    water and the dark.
```

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_confluence/6167.yaml
git commit -m "fix(confluence): open the portico great doors into the public temple (5b seam)"
```

---

## Task 2: The 16 temple rooms (6168–6183)

**Files (all Create, in `_datafiles/world/dogmud/rooms/the_confluence/`):**
6168, 6169, 6170, 6171, 6172, 6173, 6174, 6175, 6176, 6177, 6178, 6179, 6180, 6181, 6182, 6183.

**Coordinate + exit table (authoritative — cartcheck verifies; every exit reciprocal):**

| roomid | title | x | y | z | exits |
|--------|-------|---|---|---|-------|
| 6168 | The Narthex | 3 | -75 | 0 | N→6167, S→6169, W→6180 |
| 6169 | The Public Nave | 3 | -76 | 0 | N→6168, S→6172, W→6170, E→6171, U→6181 |
| 6170 | The West Aisle | 2 | -76 | 0 | E→6169, S→6174 |
| 6171 | The East Aisle | 4 | -76 | 0 | W→6169, S→6175 |
| 6172 | The Crossing — Great Hall over the Waters | 3 | -77 | 0 | N→6169, S→6173, W→6174, E→6175 |
| 6173 | The Sanctuary | 3 | -78 | 0 | N→6172, S→6178, W→6176, E→6177 |
| 6174 | Chapel of the Aldren | 2 | -77 | 0 | E→6172, N→6170, S→6176 |
| 6175 | Chapel of the Brenn | 4 | -77 | 0 | W→6172, N→6171, S→6177, E→6183 |
| 6176 | Chapel of the Solt | 2 | -78 | 0 | N→6174, E→6173, S→6179 |
| 6177 | The Disused Alcove | 4 | -78 | 0 | N→6175, W→6173 |
| 6178 | The Reliquary | 3 | -79 | 0 | N→6173, W→6179 |
| 6179 | The Sacristy | 2 | -79 | 0 | E→6178, N→6176 |
| 6180 | The Almonry | 2 | -75 | 0 | E→6168 |
| 6181 | The Gallery | 3 | -76 | 1 | D→6169, S→6182 |
| 6182 | The Gallery Walk | 3 | -77 | 1 | N→6181 |
| 6183 | The Inner Threshold | 5 | -77 | 0 | W→6175 (NO east exit yet — see below) |

**Reciprocity self-check** (walk it): 6168↔6167(done T1)/6169/6180; 6169↔6168/6172/6170/6171/6181(vert); 6172↔6169/6173/6174/6175; 6173↔6172/6178/6176/6177; 6174↔6170/6176/6172; 6175↔6171/6177/6172/6183; 6176↔6174/6173/6179; 6177↔6175/6173; 6178↔6173/6179; 6179↔6178/6176; 6180↔6168; 6181↔6169(vert)/6182; 6182↔6181; 6183↔6175. All pairs present.

**6183 east stub:** 6183 is the warded inner threshold; it has **only `W→6175`** — do NOT add an east exit yet (the Cloisters at x≥+6 are district 6). The room's *prose* describes a heavy warded door/stair leading east-and-down to the cloisters, barred to the public — a described stub, not a wired exit. (Same pattern as 5a's portico south stub.)

**Biome:** `city` for all 16 (island/temple stone), consistent with 5a.

**Required nouns:**
- 6172 Crossing: `<ansi fg="itemname">water-grate</ansi>` (key `water-grate`) — a great grate/oculus in the floor over the channel where Aldren+Brenn+Solt become one water below; the sound and motion beneath; the floor here is the oldest stone in the temple (threshold-only — the temple was set *for* this point, no why). Optional flavor noun `offering-brazier` (pilgrims leave candles/wreaths — flavor only, no item mechanic).
- 6177 Disused Alcove: `<ansi fg="itemname">walled-recess</ansi>` (key `walled-recess`) — a fourth chapel-space bricked up and turned to storage; "no one now remembers what it was consecrated to." One understated forgotten-fourth beat; no numerology.
- 6178 Reliquary: `<ansi fg="itemname">oldest-relic</ansi>` (key `oldest-relic`) — the temple's oldest relic resting on a slab of visibly pre-Founding stone, older than the temple around it (threshold-only).
- 6183 Inner Threshold: `<ansi fg="itemname">warded-door</ansi>` (key `warded-door`) — the heavy iron-bound door/stair east-and-down to the cloisters; barred to the public; the senior Keepers and "what lies beneath" are beyond it. The visible Q74 lead-in.

**Spawninfo** (mob files come in Task 3; referencing not-yet-created IDs is fine):
- 6172 → 9457 (Officiant), 9461 (kneeling worshipper)
- 6178 → 9458 (Sacristan)
- 6183 → 9459 (Threshold Warden)
- 6180 → 9460 (Almoner)
- 6174 → 9462 (pilgrim at prayer)
- 6169 → 9463 (temple visitor)

Use `respawnrate: "20 real minutes"`.

**Per-room content briefs** (the building agent authors prose; structure is fixed):
- 6168 Narthex: vestibule inside the open doors; the hush, holy-water/threshold of sacred space; the nave opening south, the almonry west.
- 6169 Nave: the long worship hall, central axis; benches/standing room; aisles E/W; stairs up to the gallery; the crossing visible south.
- 6170 West Aisle / 6171 East Aisle: side passages flanking the nave; candle-racks, pilgrims; chapels opening south.
- 6172 Crossing — Great Hall over the Waters: THE centerpiece. The `water-grate` over the joining channel; the sound of the three waters becoming one below; the oldest floor-stone; the Officiant leads the public rite here.
- 6173 Sanctuary: the altar + the sacred fire beyond the crossing; the reliquary deeper south; chapels of Solt (W) and the disused alcove (E).
- 6174 Chapel of the Aldren (the NORTH river) / 6175 Chapel of the Brenn (EAST) / 6176 Chapel of the Solt (SOUTHWEST): three river-chapels, each themed to its river's character + source direction (get the directions right).
- 6177 Disused Alcove: the bricked-up fourth space, now storage; the `walled-recess`; understated.
- 6178 Reliquary: relics on display; the `oldest-relic` on pre-Founding stone; the Sacristan attends.
- 6179 Sacristy: the working vestry (vestments, vessels); the Sacristan's domain (off the reliquary/Solt chapel).
- 6180 Almonry: alms and candles for the poor; warm, charitable daily life; the Almoner.
- 6181 Gallery (up from nave) / 6182 Gallery Walk: an upper level; the walk overlooks the crossing and the waters from above (a high vantage on the centerpiece).
- 6183 Inner Threshold: the `warded-door`; the Threshold Warden; the public turned back; the Q74 lead-in.

Give each room 2–3 `idlemessages` (quote any with colons).

- [ ] **Step 1: Author 6168–6183** per the table + briefs. Worked example (the centerpiece):

```yaml
roomid: 6172
zone: The Confluence
title: The Crossing -- Great Hall over the Waters
description: >
  The temple opens out here into its great hall, the
  point the whole island was built to hold. Set into the
  floor at the center is a broad iron
  <ansi fg="itemname">water-grate</ansi>,
  and beneath it the three rivers meet and become one --
  the Aldren from the north, the Brenn from the east, the
  Solt from the southwest, their separate voices closing
  into a single deep sound that fills the hall from below.
  The stone of the floor around the grate is older and
  darker than the temple raised on it. The nave runs back
  to the north; the sanctuary lies south; chapels open to
  either side.
biome: city
coord:
  x: 3
  y: -77
  z: 0
exits:
  north:
    roomid: 6169
  south:
    roomid: 6173
  west:
    roomid: 6174
  east:
    roomid: 6175
nouns:
  water-grate: >
    A grate of black iron bars set flush into the floor,
    wide as a cart is long. Through it the joined waters
    move close beneath -- you can feel the cool off them
    and the steady pull of the current finding its single
    channel. The slabs the grate is seated in are older
    than anything else underfoot, cut and laid by hands
    the temple's own account does not quite reach. The
    Keepers say the temple was raised to honor this
    meeting of waters. The stone says only that it was
    here first.
spawninfo:
  - mobid: 9457
    respawnrate: "20 real minutes"
  - mobid: 9461
    respawnrate: "20 real minutes"
idlemessages:
- A worshipper kneels at the edge of the grate, listening
  to the water move beneath the floor.
- 'The hall holds a single sound: the three waters closing
  into one, somewhere just below the stone.'
- The Officiant's voice carries across the hall, naming
  the rivers in the old order of the rite.
```

The Gallery 6181 + Walk 6182 use the proper-vertical pattern (6169 `up: {roomid: 6181}`; 6181 `down: {roomid: 6169}` + `south: {roomid: 6182}`; 6182 `north: {roomid: 6181}`; 6181 at z:1, 6182 at z:1, stacked over 6169/6172).

- [ ] **Step 2: Self-check** — every exit reciprocal (per the table); no two rooms share a coord; no `kind:` on any exit; 6183 has no east exit; required nouns present with matching `<ansi>` tokens (all hyphenated); spawninfo on 6172/6178/6183/6180/6174/6169; river/chapel directions correct; colons handled; prose ≤80 cols.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_confluence/6168.yaml _datafiles/world/dogmud/rooms/the_confluence/6169.yaml _datafiles/world/dogmud/rooms/the_confluence/6170.yaml _datafiles/world/dogmud/rooms/the_confluence/6171.yaml _datafiles/world/dogmud/rooms/the_confluence/6172.yaml _datafiles/world/dogmud/rooms/the_confluence/6173.yaml _datafiles/world/dogmud/rooms/the_confluence/6174.yaml _datafiles/world/dogmud/rooms/the_confluence/6175.yaml _datafiles/world/dogmud/rooms/the_confluence/6176.yaml _datafiles/world/dogmud/rooms/the_confluence/6177.yaml _datafiles/world/dogmud/rooms/the_confluence/6178.yaml _datafiles/world/dogmud/rooms/the_confluence/6179.yaml _datafiles/world/dogmud/rooms/the_confluence/6180.yaml _datafiles/world/dogmud/rooms/the_confluence/6181.yaml _datafiles/world/dogmud/rooms/the_confluence/6182.yaml _datafiles/world/dogmud/rooms/the_confluence/6183.yaml
git commit -m "feat(confluence): the public Temple, 16 rooms (6168-6183)"
```

---

## Task 3: Mobs + dialogue (9457–9463)

**Files (Create):** mob `mobs/the_confluence/<id>-<name>.yaml` + dialogue `dialogue/the_confluence/<id>.yaml` for the speaking NPCs (9457–9460); ambient pilgrims (9461–9463) are mob-only (idlecommands carry them, no dialogue file).

| mobid | name | filename | groups | archetype |
|-------|------|----------|--------|-----------|
| 9457 | The Officiant | 9457-the_officiant.yaml | humanoid, keepers | noncombat_passive |
| 9458 | The Sacristan | 9458-the_sacristan.yaml | humanoid, keepers | noncombat_passive |
| 9459 | The Threshold Warden | 9459-the_threshold_warden.yaml | humanoid, keepers | noncombat_passive |
| 9460 | The Almoner | 9460-the_almoner.yaml | humanoid, keepers | noncombat_passive |
| 9461 | A Kneeling Worshipper | 9461-a_kneeling_worshipper.yaml | humanoid | noncombat_passive |
| 9462 | A Pilgrim at Prayer | 9462-a_pilgrim_at_prayer.yaml | humanoid | noncombat_passive |
| 9463 | A Temple Visitor | 9463-a_temple_visitor.yaml | humanoid | noncombat_passive |

All `non_combatant: true`, `charm_immune: true`, `hostile: false`, `statpool: 30`,
`maxwander: 0`, `activitylevel: 10`, `speciesid: 1`, `level: 1`, small `gold`, a
couple of `stats`, 4–6 alternating `idlecommands`. No vendors in this district.

- [ ] **Step 1: The Officiant dialogue (9457.yaml) — the key file.** Three jobs, mirroring the 5a Historian pattern (`dialogue/the_confluence/9449.yaml`):
(a) DEFAULT = the fullest official account (the temple raised to honor the meeting of waters; the symbol = the Chrysalis truth; settled, devout, beautiful);
(b) a Q73-completed aside gated `questRequired: ["73-end"]`, placed FIRST in `nodes:` — loyal-but-aware (acknowledges the player has heard the scholars; gives the account as received; declines to argue it at the altar; points to the senior Keepers in the cloisters), GRANTS NOTHING;
(c) a Q74-seed node (the cloisters / the senior Keepers / "what lies beneath the floor is older than the temple, and is not shown to visitors") — points onward without granting.
Skeleton (extend the prose; keep the gating + node order):

```yaml
mobid: 9457
zone: The Confluence
defaultMood: friendly
greetings:
  - text: "Welcome to the great hall. Stand a moment at the grate -- the waters are best heard, not described."
    moods: ["friendly"]
patterns:
  - keywords: ["hello", "hi", "greet", "hey"]
    responses:
      - "Welcome. The rite is open to all who come. Ask, if you wish to understand it."
  - keywords: ["water", "waters", "grate", "river", "rivers", "below", "sound", "channel"]
    responses:
      - "Three rivers, one water, beneath this floor. The temple was raised to honor the meeting. We keep the rite that marks it."
  - keywords: [""]
    responses:
      - "Ask about the rite, the great hall, or the temple beyond."
tree:
  root:
    text: "I keep the public rite here at the crossing, where the
      three waters become one beneath the floor. The temple was
      raised over this meeting, and the meeting is the heart of
      everything we hold. What did you wish to understand?"
    hints: "You could ask about the rite and the waters below, about
      the symbol the temple keeps, or about the temple beyond the
      sanctuary."
  nodes:
    - id: margin_aside
      questRequired: ["73-end"]
      triggers: ["symbol", "mark", "fourth", "water", "waters", "count",
        "margin", "survey", "older", "wrong", "ring", "cocoon", "doubt",
        "scholar", "disagree", "question", "records", "notation", "stone"]
      text: "You have the look of someone who has stood in the
        Scholars' Quarter and listened. I will say it once, here, where
        the water can hear and no one else: I give the rite as it was
        given to me. Whether the stone beneath the grate was laid for
        our meaning or for one we have forgotten -- I am not the one to
        settle that, and the great hall is not the place to ask it. If
        you must press it, it is the senior Keepers, in the cloisters,
        who hold what there is to hold. Not the rite. Not me."
      hints: "You could ask about the temple beyond the sanctuary, or
        about the old stone beneath the floor."
    - id: rite
      triggers: ["rite", "service", "worship", "ceremony", "pray",
        "prayer", "waters", "water", "grate", "below", "sound",
        "hall", "crossing", "meeting", "how", "do"]
      text: "The rite is simple: we name the three rivers in their old
        order -- Aldren of the north, Brenn of the east, Solt of the
        southwest -- and we mark their meeting beneath the floor. The
        water has done it without us for longer than the temple has
        stood. We only attend, and say so."
      hints: "You could ask about the symbol the temple keeps, or about
        the temple beyond the sanctuary."
    - id: symbol
      triggers: ["symbol", "mark", "ring", "cocoon", "chrysalis",
        "door", "doors", "above", "oldest", "first", "what", "carving",
        "keystone", "relief"]
      text: "The ringed mark above the doors -- the sealed soul in its
        cocoon, the ring of becoming. The earliest rendering of the
        Chrysalis truth, cut by the first Founders. We treat it as the
        oldest and truest thing we keep. The Sacristan can show you
        older still, in the reliquary, if relics interest you."
      hints: "You could ask about the rite, or about the temple beyond
        the sanctuary."
    - id: beyond
      triggers: ["beyond", "cloister", "cloisters", "undercroft",
        "beneath", "below", "older", "senior", "deeper", "inner",
        "sanctuary", "further", "what", "lies", "stone", "foundation"]
      text: "The public temple ends at the sanctuary. Past it lie the
        cloisters and the archive, kept by the senior Keepers, and the
        old stone runs deeper than the cloisters -- beneath the floor,
        beneath the grate, older than the temple raised on it. That
        deep is not shown to visitors. If your questions run that far,
        it is the senior Keepers you must satisfy, not the rite."
      hints: "You could ask about the rite and the waters, or about the
        symbol the temple keeps."
```
The `margin_aside` `hints` must be second-person and not self-reference the Officiant
in third person. No `grantsQuest` anywhere.

- [ ] **Step 2: The Threshold Warden dialogue (9459.yaml).** Stationed at the warded
inner door (6183). Default = firmly, politely turns the public back ("the inner
precincts are for the Keepers; the cloisters and what lies beneath are not shown to
visitors"). A node that names the senior Keepers as the only way further (Q74
lead-in, grants nothing). Optional Q73-gated aside (`questRequired: ["73-end"]`,
first) — a shade less dismissive to someone who clearly already knows, still won't
open the door. Keep directions correct (cloisters = east-and-down beyond the door).

- [ ] **Step 3: The Sacristan dialogue (9458.yaml).** At the reliquary (6178). Lore on
the relics, especially the `oldest-relic` resting on pre-Founding stone (threshold-
only: it is older than the temple; the Keepers' account explains it as the Founders'
work; the stone is older than the account). Optional small Q73 aside. No grants.

- [ ] **Step 4: The Almoner dialogue (9460.yaml).** At the almonry (6180). Warm,
practical, charitable; alms and candles for the poor; the daily human life of the
temple. Not mystery-laden — the grounding human texture. Short.

- [ ] **Step 5: Ambient pilgrims (9461, 9462, 9463).** Mob files only, atmospheric
idlecommands (kneeling at the grate; lighting candles at a chapel; taking in the
nave). Minimal or no dialogue file.

- [ ] **Step 6: Self-check** — names Title Case; questRequired/questExcluded are LISTS;
the Officiant `margin_aside` (and any other gated asides) are FIRST in their node
lists; NO grantsQuest/givesItem/setsQuestFlag anywhere; hints second-person; river
directions correct; prose ≤80 cols.

- [ ] **Step 7: Commit**

```bash
git add _datafiles/world/dogmud/mobs/the_confluence/945*.yaml _datafiles/world/dogmud/mobs/the_confluence/946*.yaml _datafiles/world/dogmud/dialogue/the_confluence/945*.yaml _datafiles/world/dogmud/dialogue/the_confluence/946*.yaml
git commit -m "feat(confluence): public Temple NPCs + dialogue (9457-9463)"
```

---

## Task 4: Anchor schedules

**Files (Create, in `_datafiles/world/dogmud/schedules/the_confluence/`):**
- `cf_officiant.yaml` (9457), `cf_sacristan.yaml` (9458), `cf_warden_inner.yaml` (9459), `cf_almoner.yaml` (9460)
- Modify those four mob files to add `schedule_id: <id>`.

Day-at-post + night, findability-preserving (model on `schedules/the_confluence/cf_historian.yaml`). Full 24h coverage (validators panic on gaps). `target_room` = the NPC's spawn room (Officiant 6172, Sacristan 6178, Warden 6183, Almoner 6180). Keep the ambient worshippers (9461–9463) unscheduled (always present).

- [ ] **Step 1: Write the four schedule files.** Example:

```yaml
id: cf_officiant
description: "The Officiant keeps the great hall through the day, leading the
  public rite at the crossing, then withdraws and the hall is given to the
  water overnight."
segments:
  - start: 6
    end: 21
    target_room: 6172
    activity: ""
    idlecommands:
      - emote names the three rivers in the old order of the rite, voice
        carrying across the hall.
      - say Stand at the grate a while. The waters are best heard, not
        described.
      - emote bows toward the water-grate at the close of a rite and turns
        to the next who have come.
  - start: 21
    end: 6
    target_room: 6172
    activity: sleeping
    idlecommands:
      - emote has given the hall back to the water for the night and rests
        near the cold altar.
```
Sacristan: day at 6178 (reliquary), night sleeping. Warden: present at 6183 across a
long day-and-evening (the door is watched), night sleeping. Almoner: day at 6180,
night sleeping.

- [ ] **Step 2: Wire `schedule_id`** into the four mob files (top-level field, e.g.
`schedule_id: cf_officiant`, mirroring how 5a's 9449 carries it).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/schedules/the_confluence/cf_officiant.yaml _datafiles/world/dogmud/schedules/the_confluence/cf_sacristan.yaml _datafiles/world/dogmud/schedules/the_confluence/cf_warden_inner.yaml _datafiles/world/dogmud/schedules/the_confluence/cf_almoner.yaml _datafiles/world/dogmud/mobs/the_confluence/9457-the_officiant.yaml _datafiles/world/dogmud/mobs/the_confluence/9458-the_sacristan.yaml _datafiles/world/dogmud/mobs/the_confluence/9459-the_threshold_warden.yaml _datafiles/world/dogmud/mobs/the_confluence/9460-the_almoner.yaml
git commit -m "feat(confluence): public Temple anchor schedules"
```

---

## Task 5: Boot test + cartcheck

**No new files.** Verify the whole district loads and is Cartesian-clean.

- [ ] **Step 1: Wipe instance saves (SOP)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

- [ ] **Step 2: Build + boot.** `GamePlay.MapConsistencyEnforce` is already `panic` in
`_datafiles/config.yaml`; `LogToFile` already false. Boot:

```bash
go build ./... && go run . > /tmp/cf_temple_boot.log 2>&1 &
```
Wait for load, then scan the log. Expected: clean load lines (rooms zoneCount,
mobs, dialogue, factions, schedules) with **no panic**, and
`mapper.ValidateZoneConsistency errors=0 warnings=0 mode="panic"`. Watch for:
duplicate-coord panic, non-reciprocal-exit error, schedule coverage-gap/unreachable
panic, dialogue questRequired/list warning, mob non-canonical-name panic. Confirm
**no WARN/ERROR line references any 5b id (6168–6183 / 9457–9463)**.

- [ ] **Step 3: `cartcheck the_confluence`** (in-game admin, or rely on the boot
ValidateZoneConsistency which scopes the whole world) — confirm zero collisions and
the gallery vertical + the 6183 stub are clean. Kill the server when done.

- [ ] **Step 4:** Fix any failure, re-run from Step 1, commit fixes.

```bash
git add -A && git commit -m "fix(confluence): public Temple boot/cartcheck fixes"
```

---

## Task 6: World-critic + feel-tester polish pass (MANDATORY)

The recurring lesson: subagents botch river/compass directions and re-introduce
dialogue node-shadowing. Not optional.

- [ ] **Step 1: World-critic (data review).** Review all 16 rooms + 4 dialogue files
for: (a) **direction/canon errors** — every river/compass word vs §8 canon and the
coord table; the three river-chapels must match their rivers' source directions
(Aldren N, Brenn E, Solt SW); the cloisters are east-and-down beyond 6183; (b)
**dialogue node-shadowing** — short triggers substring-matching other topics; gated
nodes first; (c) lore-boundary creep (no crash/material/mutation why; the disused
alcove + oldest-relic + water-grate stay threshold-only); (d) the Officiant/Warden
Q74 seeds and any Q73-gated asides read right; (e) names Title Case, colon quoting,
hyphenated nouns, no `kind:` on exits.

- [ ] **Step 2: Feel-tester harness walk.** Per `tools/playtest/` (local feel-tester),
walk: the 6167↔6168 entry (doors now open), the nave → crossing (the `water-grate`
centerpiece) → sanctuary spine, the three chapels + disused alcove, the reliquary
(`oldest-relic`), the gallery vertical (6169→up→6181→6182, the view down), the inner
threshold (6183 `warded-door`, the Warden turning you back), and the
Officiant/Sacristan/Warden dialogue including the Q73-gated asides (grant the Q73
token chain `73-start`→`73-map`→`73-end` to test — `questtoken <end>` alone won't
take on a char who never started). **Kill ALL GoMud/go processes before the run**
(stale instances on 55555 serve old data). Save the report to
`tools/playtest/reports/2026-06-27-local-feel-tester-confluence-temple-public.md`.

- [ ] **Step 3: Fix everything found**, re-boot (Task 5 Steps 1–2) to confirm clean,
commit.

```bash
git add -A && git commit -m "fix(confluence): public Temple feel/world-critic polish"
```

---

## Task 7: Finish the branch

- [ ] **Step 1:** Confirm working tree clean and one final clean boot (instance wipe +
boot, errors=0 mode=panic).
- [ ] **Step 2:** Use `superpowers:finishing-a-development-branch` — merge
`feature/confluence-temple-public` into `master` with `--no-ff` (district convention),
delete the feature branch. Do **not** push to prod (the whole Confluence push is held
by the user).
- [ ] **Step 3:** Update memory: append the District 5b outcome to
`project_confluence_build.md` and the MEMORY.md index line; note any new gotchas.

---

## Self-review (completed by plan author)

- **Spec coverage:** §2 entry-revision → T1; §3/§4 rooms+coords → T2; §5 NPCs → T3;
  §7 Q74 seeding → T2 (nouns) + T3 (Officiant/Warden/Sacristan dialogue); §8 schedules
  → T4; §9 verification → T5/T6; §6 items → none (reuse, per spec — the offering-brazier
  is a flavor noun in T2, no item). All covered.
- **Placeholders:** none — every file has a worked example or a field-level brief;
  prose is delegated by design with all structural values fixed.
- **ID/type consistency:** room ids/coords/exits cross-checked between the §4 spec
  table and T2; spawninfo mob ids (9457–9463) match T3; faction `keepers` (existing)
  consistent T3 `groups:`; quest token `73-end` matches the shipped Q73; the 6167
  south exit (T1) matches the 6168 north exit (T2); the gallery vertical (6169↔6181,
  6181↔6182) consistent across T2; 6183 has no east exit (stub) in both the table and
  the note.
