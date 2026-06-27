# The Confluence — District 5a: The Processional — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Processional district of the Confluence — the ceremonial temple approach (14 rooms, ~8 NPCs, the new Keepers faction, Q74 lore seeding, no quest) — and verify it boots clean and plays well.

**Architecture:** Pure YAML content on the existing GoMud data-file engine, in the existing zone folder `_datafiles/world/dogmud/rooms/the_confluence/` (and parallel `mobs/`, `dialogue/`, `items/`, `factions/`, `schedules/` trees). A west-bank avenue runs south from the existing seam room 6153, opens onto a forecourt, then a causeway crosses east over open water (`long` exits) to the temple steps + portico on the central island. Verification is boot-with-panic-mode + `cartcheck` + a world-critic/feel-tester pass — there is no Go code and no unit test here.

**Tech Stack:** YAML data files; `go run .` boot test; `python tools/id_inventory.py`; `cartcheck` admin command / `ValidateZoneConsistency`; the mudagent playtest harness.

**Spec:** `docs/superpowers/specs/2026-06-27-confluence-processional-design.md`

**Reserved IDs (verified clean 2026-06-27):** rooms **6154–6167**, mobs/dialogue **9449–9456**, items **40139–40141**, no new quest, no new buffs.

---

## Authoring conventions (read once before any task)

These are load-time-fatal or recurring-bug rules from prior districts. Every task below assumes them:

1. **Mob `character.name` MUST be canonical Title Case** ("The Historian", "A Kneeling Pilgrim"). Filename is lowercase via `ConvertForFilename` (`9449-the_historian.yaml`). A non-canonical name panics at boot.
2. **Room `idlemessages` containing a colon-space MUST be quoted** (else YAML parses as a map → panic). `description`/`nouns` with prose colons MUST use `>` block scalars.
3. **Dialogue node match is `strings.Contains(topic, trigger)` (substring), THEN the quest gate.** Place specific/gated nodes FIRST in the `nodes:` list; avoid short triggers that substring-match other topics. `questRequired`/`questExcluded` are **LISTS** (`["73-end"]`), never bare strings (a string is logged-non-fatal but silently kills the NPC's dialogue).
4. **Shops validate item category vs `craft_support`.** A vendor with `craft_support: general` accepts anything; an item itself can **never** be category `general` — each item carries a real discipline (`cooking`, `alchemy`, `tailoring`, …) and the general vendor lists it explicitly in `shop:`.
5. **40xxx items live in `items/materials-40000/`** regardless of type; filenames keep any leading article.
6. **Faction:** create `factions/keepers.yaml`; mobs JOIN via a `groups:` entry. Do **not** create `factions.rep/keepers.yaml` (runtime state, gitignored).
7. **Proper vertical** = `up`/`down` exits + stacked coords (same x,y, z±1).
8. **River/compass canon:** Aldren = the **northern** tributary; Brenn from the **east**; Solt from the **southwest**; combined water spills **southwest**. There is **no Aldren south of the junction.** Double-check every direction word.
9. **Pre-smoke SOP:** `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*` before every boot/smoke (do NOT touch `shops/`).

Prose (descriptions, idlemessages, dialogue text) is authored by the building agent following the schema examples below and the voice of the existing Confluence files (`mobs/the_confluence/9435-savel...`, `dialogue/the_confluence/9441.yaml`). The plan fixes **all** IDs, coords, exits, nouns, flags, faction membership, and quest gating — those are not the agent's to invent.

---

## Task 1: The Keepers faction + the seam exit

**Files:**
- Create: `_datafiles/world/dogmud/factions/keepers.yaml`
- Modify: `_datafiles/world/dogmud/rooms/the_confluence/6153.yaml` (add the south exit to 6154)

- [ ] **Step 1: Create the Keepers faction**

```yaml
faction_id: keepers
display_name: "The Keepers of the Confluence"
description: |
  The temple clergy of the Confluence — priests, wardens, hospitallers, and
  acolytes who keep the great temple on the island and tend the pilgrims who
  come to it. They maintain the official account: that the old orbital marks
  in the temple's oldest stone are early Chrysalis theology, and that the
  three rivers are the whole of the matter. Some of them believe it. The
  inner Keepers, who gate the cloisters and the undercroft, know it is more
  complicated than that, and prefer the question stay settled.
default_rep: 0
allies: []
enemies: []
```

Note: **no `enemies:` edge** to `margin` — the Keeper↔Margin tension is narrative-only in this build; the allegiance mechanic lands on Q74. (Spec §5.)

- [ ] **Step 2: Wire the seam exit on 6153**

In `6153.yaml`, the `exits:` block currently has only `north: {roomid: 6141}`. Add the south exit. The existing `the way south` noun already describes the Processional as "a walk for another day" — soften that one line so it no longer reads as impassable (the gate "stands open"; change "is a walk for another day" → "runs on into the Processional district" in both the description tail and the `the way south` noun, keeping prose ≤80 cols). Final `exits:` block:

```yaml
exits:
  north:
    roomid: 6141
  south:
    roomid: 6154
```

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/factions/keepers.yaml _datafiles/world/dogmud/rooms/the_confluence/6153.yaml
git commit -m "feat(confluence): Keepers faction + Processional seam exit on 6153"
```

---

## Task 2: Items — devotional goods (40139–40141)

**Files:**
- Create: `_datafiles/world/dogmud/items/materials-40000/40139-a_beeswax_votive_candle.yaml`
- Create: `_datafiles/world/dogmud/items/materials-40000/40140-a_river_reed_wreath.yaml`
- Create: `_datafiles/world/dogmud/items/materials-40000/40141-a_twist_of_temple_incense.yaml`

These are the offering-seller's stock. Each carries a **real discipline** (`alchemy` — incense/candle/wreath read as alchemical/devotional supplies; pick `alchemy` for all three for consistency), cheap value, low weight. Model on `items/materials-40000/40130-river_spice.yaml`.

- [ ] **Step 1: Write the three item files**

Example (40139 — repeat the shape for 40140 wreath, 40141 incense with appropriate name/description/component_tag):

```yaml
itemid: 40139
name: A Beeswax Votive Candle
namesimple: candle
description: >
  A short, honey-colored candle of pressed beeswax, sold at the
  votive stalls for pilgrims to light at the temple. It smells
  faintly of the hive and burns clean.
type: object
subtype: mundane
component_tag: votive-candle
weight: 0.1
value: 4
rarity_tier: 40
is_component: true
vendor_categories:
- alchemy
```

40140: name "A River-Reed Wreath", namesimple "wreath", component_tag `reed-wreath`, value 5. 40141: name "A Twist of Temple Incense", namesimple "incense", component_tag `temple-incense`, value 6.

- [ ] **Step 2: Sanity-check filenames** match `ConvertForFilename(name)` (leading article kept, lowercased, spaces→underscores): `40139-a_beeswax_votive_candle.yaml`, etc.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/items/materials-40000/40139-*.yaml _datafiles/world/dogmud/items/materials-40000/40140-*.yaml _datafiles/world/dogmud/items/materials-40000/40141-*.yaml
git commit -m "feat(confluence): Processional devotional goods (40139-40141)"
```

---

## Task 3: The 14 rooms (6154–6167)

**Files (all Create, in `_datafiles/world/dogmud/rooms/the_confluence/`):**
6154, 6155, 6156, 6157, 6158, 6159, 6160, 6161, 6162, 6163, 6164, 6165, 6166, 6167.

**Coordinate + exit table (authoritative — do not deviate; cartcheck verifies):**

| ID | Title | Coord (x,y,z) | Exits | Biome |
|----|-------|---------------|-------|-------|
| 6154 | Processional Avenue, North End | -5,-70,0 | N→6153, S→6155 | city |
| 6155 | Processional Avenue, Votive Stalls | -5,-71,0 | N→6154, S→6156 | city |
| 6156 | Processional Avenue, South End | -5,-72,0 | N→6155, S→6157, E→6162 | city |
| 6162 | The Hall of the Founding | -4,-72,0 | W→6156 | city |
| 6157 | The Temple Forecourt | -5,-73,0 | N→6156, W→6158, S→6160, E→6163 | city |
| 6158 | The Meditation Garden | -6,-73,0 | E→6157, W→6159 | city |
| 6159 | The Still Pool | -7,-73,0 | E→6158 | city |
| 6160 | The Pilgrim Hall | -5,-74,0 | N→6157, U→6161 | city |
| 6161 | The Pilgrim Dormitory | -5,-74,1 | D→6160 | city |
| 6163 | The Causeway, West Span | -4,-73,0 | W→6157, E(long)→6164 | city |
| 6164 | The Causeway, Crown | -1,-73,0 | W(long)→6163, E(long)→6165 | city |
| 6165 | The Causeway, East Span | 1,-73,0 | W(long)→6164, E→6166 | city |
| 6166 | The Temple Steps | 3,-73,0 | W→6165, S→6167 | city |
| 6167 | The Temple Portico | 3,-74,0 | N→6166 | city |

**Exit reciprocity:** every exit above has its mirror in the table — verify each pair.

**Long causeway exits — do NOT author a `kind` field.** Exit `kind` (`normal`/`long`/`vertical`/`wrap`) is **derived by the mapper** (`classifyKind`, `internal/mapper/mapper.consistency.go`) from the coordinate delta — a multi-cell jump auto-classifies as `long`. Authors write only `roomid`. The bridge spans (6163→6164 is −4→−1 = 3 cells; 6164→6165 is −1→1 = 2 cells) become long automatically. The `longcrossing` check is a **warning, not a panic**, and fires *only* if a long span crosses an occupied cell — the water cells between x −4 and +3 at y −73 hold no rooms, so it will not fire. A causeway exit is just:
```yaml
exits:
  east:
    roomid: 6164
```

**Spawninfo (which mob spawns where — mob files come in Task 4; referencing not-yet-created IDs is fine, boot resolves at load):**
- 6155 → 9451 (offering-seller)
- 6157 → 9450 (warden), 9453 (acolyte)
- 6160 → 9452 (hospitaller), 9455 (kneeling pilgrim), 9456 (road-worn pilgrim)
- 6162 → 9449 (the historian)
- 6167 → 9454 (margin scholar)

Use `respawnrate: "20 real minutes"` (matches 6139).

**Key nouns (required, others at the agent's discretion for flavor):**
- 6162 Hall of the Founding: an `<ansi fg="itemname">official plaque</ansi>` noun (key `official plaque`) — states the *official* Keeper line: the door-symbol is "early Chrysalis cocoon-theology, the first Founders' rendering of the sealed soul." Earnest, settled, subtly wrong to a Q73 player. Also a `founding-relief` noun (carved scene of the three rivers + the temple's raising).
- 6164 Causeway, Crown: a `<ansi fg="itemname">old stonework</ansi>` noun (key `old stonework`) — the bridge's pre-Founding base courses, plainly older than the span on top; the orbital ring-mark worn into one block. Environmental beat, no lecture.
- 6167 Temple Portico: a `<ansi fg="itemname">weathered symbol</ansi>` noun (key `weathered symbol`, **no leading "the"** to avoid the article-doubling cosmetic bug) — the four-ring orbital mark cut into the keystone above the great doors, older than everything around it; the official line calls it Chrysalis theology. The great doors themselves: a `great doors` noun describing them shut, "the public temple beyond not open from this approach" (the stub to the Temple-public build).

**Lore discipline:** threshold only — foreshadow older work beneath, the undercroft "not shown to visitors"; never the crash/mutation why.

- [ ] **Step 1: Author 6154–6167** per the table. Worked example (a spawn + noun room):

```yaml
roomid: 6162
zone: The Confluence
title: The Hall of the Founding
description: >
  A modest stone hall off the avenue, given over to the temple's
  account of its own beginning. A long carved
  <ansi fg="itemname">founding-relief</ansi>
  runs the length of one wall — the three rivers, the raising of
  the temple, the first Keepers at the water's edge. A brass
  <ansi fg="itemname">official plaque</ansi>
  beneath it sets out, in confident lettering, what the oldest
  marks in the temple stone are held to mean. A keeper attends
  here through the day to explain it to whoever asks.
biome: city
coord:
  x: -4
  y: -72
  z: 0
exits:
  west:
    roomid: 6156
nouns:
  founding-relief: >
    The relief is old work, well kept: the three rivers shown as
    three braided lines drawing together, the temple rising on the
    island where they meet, robed figures gathered at the banks.
    It is a confident, orderly account of a beginning. Nothing in
    it suggests the beginning was ever in doubt.
  official plaque: >
    THE FIRST MARK, the plaque reads, is the earliest rendering of
    the Chrysalis truth — the sealed soul, the ringed cocoon, cut
    into the temple stone by the first Founders. It is given here
    as settled doctrine. A careful reader who has counted the old
    surveys' waters might find the confidence a little too tidy.
spawninfo:
  - mobid: 9449
    respawnrate: "20 real minutes"
idlemessages:
- A keeper murmurs the founding account to a knot of listeners,
  gesturing at the relief.
- 'Light from the high window crosses the plaque: brass, then
  shadow, then brass again.'
```

(Note the quoted idlemessage with the colon.) The Pilgrim Hall 6160 + Dormitory 6161 use the proper-vertical pattern: 6160 has `up: {roomid: 6161}`, 6161 has `down: {roomid: 6160}`, same x/y, z 0 and 1.

- [ ] **Step 2: Self-check exits + coords.** Walk the table mentally: every exit has a reciprocal; no two rooms share a coord; the four `long` exits are marked. The island portico is at x+3 (sets up Temple-public).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/rooms/the_confluence/615*.yaml _datafiles/world/dogmud/rooms/the_confluence/616*.yaml
git commit -m "feat(confluence): the Processional, 14 rooms (6154-6167)"
```

---

## Task 4: Mobs + dialogue (9449–9456)

**Files (Create, paired mob + dialogue per NPC):**
- `mobs/the_confluence/9449-the_historian.yaml` + `dialogue/the_confluence/9449.yaml`
- `9450-a_processional_warden.yaml` + `dialogue/9450.yaml`
- `9451-the_offering_seller.yaml` + `dialogue/9451.yaml`
- `9452-the_hospitaller.yaml` + `dialogue/9452.yaml`
- `9453-an_acolyte.yaml` + `dialogue/9453.yaml`
- `9454-a_margin_scholar.yaml` + `dialogue/9454.yaml`
- `9455-a_kneeling_pilgrim.yaml` (ambient; dialogue optional)
- `9456-a_road_worn_pilgrim.yaml` (ambient; dialogue optional)

**Common mob shape** (model on `9435-savel...` / `9444-drunn...`): `zone: The Confluence`, `non_combatant: true`, `charm_immune: true`, `hostile: false`, `statpool: 30`, `maxwander: 0`, `activitylevel: 10`, `character.name` Title Case, `character.speciesid: 1`, `level: 1`, small `gold`, a couple of `stats`, and 4–6 `idlecommands` (alternating `emote`/`say`/`''`). Faction via `groups:`.

**Per-NPC `groups:` + archetype:**
| Mob | groups | behavior_archetype | vendor? |
|-----|--------|--------------------|---------|
| 9449 Historian | [humanoid, keepers] | noncombat_passive | no |
| 9450 Warden | [humanoid, keepers] | noncombat_passive | no |
| 9451 Offering-Seller | [humanoid] | noncombat_shopkeeper | yes |
| 9452 Hospitaller | [humanoid, keepers] | noncombat_shopkeeper | yes |
| 9453 Acolyte | [humanoid, keepers] | noncombat_passive | no |
| 9454 Margin Scholar | [humanoid, margin] | noncombat_passive | no |
| 9455 Kneeling Pilgrim | [humanoid] | noncombat_passive | no |
| 9456 Road-Worn Pilgrim | [humanoid] | noncombat_passive | no |

- [ ] **Step 1: Vendors (9451, 9452).** Offering-seller carries `craft_support: general` + a `shop:` list of the devotional goods:

```yaml
  shop:
    - itemid: 40139
    - itemid: 40140
    - itemid: 40141
```

Hospitaller: `craft_support: general` (or `cooking`) + a `shop:` list of pilgrim fare — **reuse existing** cooking goods to avoid new items: `40135` (bowl of river broth) + `40136` (loaf of black bread). (Both already exist from the tavern.)

- [ ] **Step 2: The Historian dialogue (9449.yaml) — the key file.** Three things: (a) the official Keeper account as the default lore; (b) a Q73-completed aside; (c) the Q74 seed (points to the inner Keepers/cloisters + "older work"/"undercroft not shown"), **granting nothing**. Gated node FIRST (conv. #3). Skeleton:

```yaml
mobid: 9449
zone: The Confluence
defaultMood: friendly
greetings:
  - text: "Welcome to the Hall of the Founding. Ask, and I will tell you how the Confluence began."
    moods: ["friendly"]
patterns:
  - keywords: ["hello", "hi", "greet", "hey"]
    responses:
      - "Welcome. The founding account is on the wall, and I am here to explain it."
  - keywords: ["symbol", "mark", "stone", "ring", "cocoon", "chrysalis", "door", "oldest"]
    responses:
      - "The first mark, yes -- the ringed cocoon above the temple doors. The earliest rendering of the Chrysalis truth, the Founders' own hand. Older than anything else we keep."
  - keywords: [""]
    responses:
      - "Ask about the founding, the temple, or the old mark above the doors."
tree:
  root:
    text: "I keep the founding account here -- how the three rivers were
      drawn together, the temple raised on the island, the first Keepers
      set to tend it. What did you wish to know?"
    hints: "You could ask about the temple's founding, about the old mark
      above the doors, or about the inner temple."
  nodes:
    - id: margin_aside
      questRequired: ["73-end"]
      triggers: ["symbol", "mark", "fourth", "water", "waters", "count",
        "margin", "survey", "older", "wrong", "ring", "cocoon", "doubt"]
      text: "You have the look of someone who has been talking to the
        scholars upriver. I will say this once, quietly: I give the account
        as I was given it. Whether the first mark is doctrine or something
        the doctrine was laid over -- that is not a question the Hall is for.
        If you mean to press it, it is the senior Keepers you want, in the
        cloisters across the way. Not me."
      hints: "You could ask the historian about the inner temple, or about
        the old work beneath the foundations."
    - id: founding
      triggers: ["founding", "history", "begin", "temple", "rivers",
        "account", "raised", "first", "keepers", "story"]
      text: "Three rivers -- the Aldren from the north, the Brenn from the
        east, the Solt from the southwest -- join below the island, and on
        that island the first Founders raised the temple, on ground they
        held sacred before a stone was cut. The Keepers have tended it ever
        since. It is a tidy beginning, and we are fond of it."
      hints: "You could ask about the old mark above the doors, or about the
        inner temple."
    - id: inner_temple
      triggers: ["inner", "cloister", "cloisters", "undercroft", "beneath",
        "below", "older", "foundation", "aldric", "senior", "deeper"]
      text: "The public temple is open to all across the causeway. Beyond it
        lie the cloisters and the archive, kept by the senior Keepers, and
        beneath the island older work the Founders consecrated -- the
        undercroft, which is not shown to visitors. If your questions run
        that deep, it is the senior Keepers you must satisfy, not the Hall."
      hints: "You could ask about the founding, or about the old mark above
        the doors."
```

The `margin_aside` node is **questRequired `["73-end"]`** so it only fires for Q73-completers and, being first, takes precedence on the shared triggers; everyone else falls through to `founding`/`inner_temple`. No `grantsQuest` anywhere.

- [ ] **Step 3: The Margin Scholar dialogue (9454.yaml).** A quiet figure copying the door-symbol; wary of being overheard; a Q73 aside that recognizes a fellow traveler. Model voice on Savel (9435). Default lore = guarded ("I copy what's there; I draw no conclusions in the temple's own forecourt"). A `questRequired: ["73-end"]` first node = the knowing aside ("So Quist sent you, or you found it yourself. Then you already see what I'm copying. The keystone is older than the doctrine cut beneath it. I write that down. I do not say it here."). No grants. Keep direction words correct (Scholars' Quarter is upriver / north).

- [ ] **Step 4: Warden / Acolyte / Pilgrims (9450, 9453, 9455, 9456).** Short ambient dialogue or idlecommands-only. The **warden** carries the friction beat: an idlecommand or dialogue line about keeping an eye on "the one copying the stonework — harmless, the prior says, so we leave her be" (shows tolerance + watchfulness without confrontation). Acolyte: tends garden/sweeps, gentle devotional flavor. Pilgrims: road-worn arrival / kneeling in the hall.

- [ ] **Step 5: Self-check** — every mob name Title Case; every dialogue `questRequired`/`questExcluded` a LIST; gated nodes first; no `grantsQuest` anywhere in this district; vendor `shop:` itemids all exist (40139–40141, 40135, 40136).

- [ ] **Step 6: Commit**

```bash
git add _datafiles/world/dogmud/mobs/the_confluence/944*.yaml _datafiles/world/dogmud/mobs/the_confluence/945*.yaml _datafiles/world/dogmud/dialogue/the_confluence/944*.yaml _datafiles/world/dogmud/dialogue/the_confluence/945*.yaml
git commit -m "feat(confluence): Processional NPCs + dialogue (9449-9456)"
```

---

## Task 5: Anchor schedules

**Files (Create, in `_datafiles/world/dogmud/schedules/the_confluence/` — new folder):**
- `cf_historian.yaml` (9449), `cf_offering_seller.yaml` (9451), `cf_hospitaller.yaml` (9452)
- Modify the three mob files to add `schedule_id: cf_historian` (etc.)

Schedules must **preserve findability** (day at post; the historian/seller present through daytime; the hospitaller present day+evening). Model on `schedules/amber_valley/av_hesper.yaml`. Full 24h coverage (validators panic on gaps). Keep the margin scholar, warden, acolyte, and pilgrims unscheduled (always present) so the district never feels empty.

- [ ] **Step 1: Write the three schedule files.** Example:

```yaml
id: cf_historian
description: "The historian keeps the Hall of the Founding through the day,
  explaining the account to whoever asks, then retires at night."
segments:
  - start: 6
    end: 21
    target_room: 6162
    activity: ""
    idlecommands:
      - emote gestures along the founding-relief, naming the rivers for a
        knot of listeners.
      - say The first mark above the doors is the oldest thing we keep. The
        Founders' own hand.
  - start: 21
    end: 6
    target_room: 6162
    activity: sleeping
    idlecommands:
      - emote has banked the Hall's lamps and dozes in the keeper's chair by
        the relief.
```

Offering-seller: day at 6155 (votive stalls), night sleeping. Hospitaller: day+evening at 6160 (pilgrim hall), late-night sleeping. Use `target_room` = the NPC's spawn room.

- [ ] **Step 2: Wire `schedule_id` into the three mob files** (add the field at top level, e.g. `schedule_id: cf_historian`).

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/schedules/the_confluence/ _datafiles/world/dogmud/mobs/the_confluence/9449-*.yaml _datafiles/world/dogmud/mobs/the_confluence/9451-*.yaml _datafiles/world/dogmud/mobs/the_confluence/9452-*.yaml
git commit -m "feat(confluence): Processional anchor schedules"
```

---

## Task 6: Boot test + cartcheck

**No new files.** Verify the whole district loads and is Cartesian-clean.

- [ ] **Step 1: Wipe instance saves (SOP)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

- [ ] **Step 2: Build + boot with panic-mode map enforcement.** Temporarily ensure `GamePlay.MapConsistencyEnforce: panic` in `_datafiles/config.yaml` for this run (revert after). Boot:

```bash
go build ./... && go run . 2>&1 | tee /tmp/cf_proc_boot.log
```

Expected: clean load lines (`rooms.LoadDataFiles ...`, `mobs.LoadDataFiles ...`, `dialogue ...`, `factions ...`, `schedules ...`) with **no panic**, `ValidateZoneConsistency ... errors=0`. Watch specifically for: duplicate-coord panic, non-reciprocal-exit warning, schedule coverage-gap/unreachable-room panic, faction/dialogue load warnings, item-category panic.

- [ ] **Step 3: In-game `cartcheck the_confluence`** (admin) — confirm zero collisions and that the four `long` causeway exits report as long connectors crossing no rooms.

- [ ] **Step 4:** If anything fails, fix the offending file and re-run from Step 1. Revert the temporary `MapConsistencyEnforce` change (back to `warn`) and restore `Logging.LogToFile` to whatever it was. Commit any fixes.

```bash
git add -A && git commit -m "fix(confluence): Processional boot/cartcheck fixes"
```

---

## Task 7: World-critic + feel-tester polish pass (MANDATORY)

The recurring district lesson: subagents botch river/compass directions and re-introduce dialogue node-shadowing. This pass is not optional.

- [ ] **Step 1: World-critic pass (data review).** Dispatch a subagent (or review directly) over all 14 rooms + 8 dialogue files for: (a) **direction/canon errors** — every compass/river word checked against the §8 canon and the coord table (the Aldren is north; nothing flows "Aldren" south of the junction; Scholars' Quarter is upriver/north; the Solt is SW); (b) **dialogue node-shadowing** — any short trigger that substring-matches another node's topic; confirm gated nodes are first; (c) lore-boundary creep (no crash/mutation why); (d) the official-plaque vs portico-symbol echo lands and the Q73 asides read right.

- [ ] **Step 2: Feel-tester harness walk.** Per `tools/playtest/`: walk the seam (6153↔6154), the avenue, the Hall (historian dialogue incl. a Q73-completed character's aside), the garden/still-pool, the pilgrim hall + dormitory (vertical), the causeway long-exits + map rendering, and the portico (margin scholar dialogue, the weathered-symbol noun, the stubbed great doors). Buy from the offering-seller and the hospitaller. Kill ALL `GoMud`/`go` processes before the run (stale instances on 55555 serve old data). Save the report under `tools/playtest/reports/2026-06-27-local-feel-tester-confluence-processional.md`.

- [ ] **Step 3: Fix everything found**, re-boot (Task 6 Step 1–2) to confirm still clean, and commit.

```bash
git add -A && git commit -m "fix(confluence): Processional feel/world-critic polish"
```

---

## Task 8: Finish the branch

- [ ] **Step 1:** Confirm the working tree is clean and the district boots clean one final time (instance wipe + boot, errors=0 mode=panic).
- [ ] **Step 2:** Use `superpowers:finishing-a-development-branch` to merge into `master` with `--no-ff` (the district-build convention), or present options. Do **not** push to prod — the whole Confluence push is held by the user.
- [ ] **Step 3:** Update the memory: append the District 5a build outcome to `project_confluence_build.md` and the MEMORY.md index line; note any new gotchas discovered.

---

## Self-review (completed by plan author)

- **Spec coverage:** §2 seam/coords → T1/T3; §3 rooms → T3; §4 NPCs → T4; §5 faction → T1; §6 items → T2; §7 Q74 seeding → T3 (nouns) + T4 (historian/scholar dialogue); §8 schedules → T5; §9 verification → T6/T7. All covered.
- **Placeholders:** none — every file has a worked example or an exact field-level brief; prose is delegated by design (content build), with all structural values fixed.
- **Type/ID consistency:** room IDs/coords/exits cross-checked between the §3 spec table and T3; spawninfo mob IDs (9449–9456) match T4; vendor `shop:` itemids (40139–40141, 40135–40136) match T2 and existing files; faction id `keepers` consistent T1↔T4 `groups:`; quest tokens (`73-end`) match the shipped Q73.
