# Test Report: Stage 3.0c — The Fernway South Feel Test

**Date:** 2026-04-28
**Target:** local
**Role:** feel-tester
**Character:** smoketester
**Goals file:** stage-3-0c-fernway-south-feel.yaml
**Duration:** ~25 minutes, 0 live commands sent (server down — static analysis only)

---

## Session Summary

The MUD server was not running on localhost:55555 at the time of the test.
The bridge exited immediately with `ConnectionRefusedError: [WinError 10061]`
and `Test-NetConnection` confirmed the port was closed. No live session was
possible.

To compensate, a full static analysis was performed: all 20 room YAML files
(4157–4176), all 6 mob YAML files (mobs 360–365), both forage-drop item
files (40064–40065), the zone-config, the player YAML (users/17.yaml), and
the rooms.instances directory were read and audited. Goals that can be
answered from file contents are graded; goals that require a live server are
marked BLOCKED.

---

## Goal Results

- [ ] **Confirm spawn room** — BLOCKED: Server down. Player YAML `users/17.yaml`
  correctly shows `roomid: 4157 / zone: The Fernway South` — the edit was
  saved. Whether the server has reloaded it is unknown.

- [ ] **Walk all 20 rooms; prose quality check** — BLOCKED (live): Cannot walk.
  Static prose review conducted instead — see Voice Review section below.
  All 20 room files present and loadable. Prose quality is high across the
  zone; detailed notes in Voice Review.

- [ ] **`look <noun>` for all named ANSI nouns** — BLOCKED (live): Cannot
  issue in-game commands. Static verification done: every ANSI-tagged noun
  in every description has a matching key in the room's `nouns:` block.
  No orphaned tags found. (See Noun Audit section.)

- [x] **Room 4175 Steppe Edge — plains biome, steppe theme** — PASS (static):
  `biome: plains` confirmed. Description is fully steppe-themed (pale-grass,
  scoured-stones, south-horizon, constant south wind, Dustwalk reference).
  Nouns: steppe, pale-grass, scoured-stones, south-horizon. All present and
  themed correctly. The prose transition ("One step the canopy is overhead;
  the next it is not") is the best single sentence in the zone.

- [x] **Twin Beech Glade (4163) — 4 exits** — PASS (static): exits confirmed
  as north→4160, south→4170, east→4167, west→4164. All four cardinal
  directions are wired.

- [x] **Twisted Hawthorn (4174) — 4 exits** — PASS (static): exits confirmed
  as north→4176 (Birdsong Glade), south→4175 (Steppe Edge), east→4173
  (Foxglove Clearing), west→4172 (Badger Sett). Matches goals spec exactly.

- [ ] **Encounter all 6 mob spawns** — BLOCKED (live): Cannot trigger spawns.
  Static verification: all 6 mob YAML files exist (360–365), mob IDs match
  spawninfo entries in the correct rooms. Spawn cooldowns noted (see Wildlife
  section). Item drops verified.

- [ ] **Wolf-vs-boar emergent dynamic** — BLOCKED (live): Cannot observe.
  Static verification: wolf (364) has `hates: [boar]` and `maxwander: 5`.
  Boar (363) is in 4162, wolf spawns in 4161. They are 2 rooms apart
  (4161 → 4160 → 4162). Wolf can reach boar in 2 wander steps. The
  condition for combat is structurally sound. See Wildlife section.

- [ ] **Verify wolf does NOT attack player** — BLOCKED (live): Cannot test.
  Static: wolf has `hostile: false`. No `hates:` entry for player groups.
  Should be safe.

- [ ] **Verify badger DOES attack player** — BLOCKED (live): Cannot test.
  Static: badger (365) has `hostile: true`. Should aggro on room entry.

- [x] **Boundary edit (4157 ↔ 4156 Fox Den)** — PASS (static): Room 4157
  has `north: roomid 4156 zone: The Fernway`. Room 4156 (Fox Den) has
  `south: roomid 4157 zone: The Fernway South`. Bidirectional cross-zone
  link is correctly set up on both sides.

- [ ] **Map renders cleanly** — BLOCKED (live): Cannot run `map` command.
  Coordinate audit shows no duplicates within the zone. Coord range is
  x [-15,-13], y [-22,-15]. Verified no overlap with Fox Den (4156 at
  [-14,-14]) — 4157 is at [-14,-15], one step south. The Fernway South
  rooms do not overlap any known neighboring zone coordinates.

- [x] **Forage drops present** — PASS (static): Item 40064 (wild hare meat)
  exists in `materials-40000/`. Item 40065 (beeswax) exists. Mobs 360
  (hare), 361 (deer) correctly reference 40064. Mob 362 (honey bees)
  correctly references 40065. Drop chances: hare 75%, deer 60%, bees 50%.
  See CONCERN below about boar drops.

---

## Voice Review

Overall voice is **excellent** and consistent with Fox Den (4156). The writing
is sensory-led, naturalist, present-tense, no second-person psychology, no
mechanical intrusion. Matches the brief perfectly.

### Standout prose

- **4175 Steppe Edge**: "One step the canopy is overhead; the next it is not,
  and the sky opens in all its width for the first time since entering the
  forest." Strong transition moment.
- **4160 Old Burn Scar**: "A faint char smell persists on still days, old and
  dry, less a burning than a memory of one." Excellent sensory restraint.
- **4172 Badger Sett**: "The badger-musk reaches you before the sett does."
  Good atmospheric arrival beat.
- **4171 Old Stand**: "The bracken gives way here to a cathedral hush." Strong
  tonal shift coming out of 4170's claustrophobic bracken.
- **364 timber wolf description**: "Whatever it came south for, it isn't you."
  Perfect behavioral signal to the player that this mob is not hostile.

### Minor prose notes (OBSERVATIONs, not bugs)

1. **4170 Tangled Bracken** — the idle message "something small crashes through
   the bracken east of the trail" implies east of the trail has something
   noteworthy. There is no east exit from 4170. Minor immersion note; not a
   bug (idle messages are flavor, not hints).

2. **4163 Twin Beech Glade** — description says "The air beneath is cooler than
   the burn scar to the north." The burn scar is indeed north (4160). Then:
   "worn deer-tracks branch from the glade at the path-junction, one east,
   one west, one south." This mentions three branch tracks but there are four
   exits (the fourth being north). The north exit is implied by "burn scar to
   the north" in the description text, but it's not explicitly named in the
   track branching line. Not a bug — players will see the north exit listed —
   but it's a minor descriptive gap.

3. **4174 Twisted Hawthorn** — "For the first time in the zone the south-wind
   is felt rather than merely guessed at." The zone's first proper wind mention
   is a good escalation. The description works.

4. **4159 Hare-Run Meadow** — "the first break in the canopy since Foxglade."
   The room description references "Foxglade" by name, which is the zone north
   of The Fernway (not a room in The Fernway South). Players who have not come
   from the north may not know what Foxglade is. Minor clarity note.

### ANSI tag hygiene

No malformed `<ansi ...>` tags found in any room. All tags use the standard
`<ansi fg="itemname">...</ansi>` format. No tag bleeding expected.

---

## Noun Audit (Static)

All 20 rooms audited. Every `<ansi fg="itemname">` tagged noun in descriptions
has a matching key in the `nouns:` block. Confirmed:

| Room | Nouns tagged | Nouns defined | Match |
|------|-------------|---------------|-------|
| 4157 Briar Tangle | bramble, thorn-windfall, deer-track, leaf-litter | 4 | OK |
| 4158 Beewood Hollow | bee-tree, swarm, honeycomb, sun-patch | 4 | OK |
| 4159 Hare-Run Meadow | open-sky, meadow-grass, hare-path, droppings | 4 | OK |
| 4160 Old Burn Scar | burn-scar, downed-timber, mushrooms, saplings | 4 | OK |
| 4161 Pine Stand | needle-carpet, pine-trunk, pitch-bead, wolf-sign | 4 | OK |
| 4162 Boar Wallow | wallow-mud, hoof-prints, scratching-tree, flies | 4 | OK |
| 4163 Twin Beech Glade | dappled-light, beech-mast, twin-beech, path-junction | 4 | OK |
| 4164 Brook Rise | brook, spring-pool, moss, watercress | 4 | OK |
| 4165 Heron Pool | pool, reeds, minnows, heron-track | 4 | OK |
| 4166 Watercress Bend | eddy, watercress-mat, blood-moss, soft-mud | 4 | OK |
| 4167 Deer Browse | browse-line, hoof-prints, scrape, droppings | 4 | OK |
| 4168 Birch Stand | birch-trunk, bark-strip, birch-sap, deer-rub | 4 | OK |
| 4169 Salt Lick | lick, mineral-seep, deer-trails, lick-marks | 4 | OK |
| 4170 Tangled Bracken | bracken-wall, fern-frond, narrow-trail, insect-hum | 4 | OK |
| 4171 Old Stand | old-oak, deep-shade, oak-bark, root-throne | 4 | OK |
| 4172 Badger Sett | badger-musk, sett, burrow-entry, bone-pile | 4 | OK |
| 4173 Foxglove Clearing | foxglove, flower-spike, bumblebees, sun-warmth | 4 | OK |
| 4174 Twisted Hawthorn | hawthorn, haws, south-wind, drying-soil | 4 | OK |
| 4175 Steppe Edge | steppe, pale-grass, scoured-stones, south-horizon | 4 | OK |
| 4176 Birdsong Glade | bird-chorus, sun-patch, fallen-log, long-grass | 4 | OK |

**Total: 80 noun definitions across 20 rooms. All matched. Zero orphans.**

Note: Room 4161 Pine Stand uses `<ansi fg="itemname">pitch-bead</ansi>s` (noun
tag wraps just the stem, not the trailing 's'). The `look pitch-bead` noun key
is correctly defined. This is standard MUD convention — the noun key is the
stem. No bug.

---

## Wildlife Behavior (Static Analysis)

### Mob roster summary

| Mob | ID | Hostile | Behavior | Hates | Wander | Spawn Room | Cooldown |
|-----|-----|---------|----------|-------|--------|-----------|---------|
| wild hare | 360 | false | prey | — | 3 | 4159 | 300 |
| roe deer | 361 | false | prey | — | 4 | 4167, 4169 | 600 |
| honey bees | 362 | false | combat_passive | — | 0 | 4158 | 600 |
| feral boar | 363 | false | combat_passive | — | 2 | 4162 | 900 |
| timber wolf | 364 | false | generic_fighter | boar | 5 | 4161 | 1200 |
| forest badger | 365 | true | generic_fighter | — | 1 | 4172 | 1800 |

### Wolf-vs-boar dynamic (structural analysis)

- Wolf spawns at 4161 (Pine Stand), maxwander 5
- Boar spawns at 4162 (Boar Wallow), maxwander 2
- Path between them: 4161 → 4160 → 4162 (2 steps)
- Wolf has `hates: [boar]` and boar is in group `boar`
- The wolf can reach the boar in 2 wander steps; the boar can reach the wolf
  in 2 wander steps
- Structural condition for emergent combat is correct. Actual trigger
  depends on the engine's hates-group matching implementation.

### Honey bees: charm_immune
Bees have `charm_immune: true`. This is a sound design choice — you shouldn't
be able to charm a swarm of bees. The other mobs do not have this flag, which
means players can attempt to charm wolves, deer, boars, badgers. Whether that
creates exploits depends on the charm spell's level scaling, but it's not a
data issue.

### Badger cooldown
The badger has a 1800-round cooldown (30 minutes at 1 round/second). This is
the longest cooldown in the zone. A player who kills the badger won't see
another for 30 minutes. Given `hostile: true`, this feels correct — the
badger is meant to be a rare dangerous encounter, not a grind mob.

---

## Map / Cartesian Analysis

Zone coordinate range: x [-15, -13], y [-22, -15], z 0

```
y=-15: 4158[-15] 4157[-14] 4159[-13]   (entry row: Beewood/Briar/Hare-Run)
y=-16: 4162[-15] 4160[-14] 4161[-13]   (spine row: Boar/OldBurn/Pine)
y=-17: 4164[-15] 4163[-14] 4167[-13]   (hub row: Brook/TwinBeech/Deer)
y=-18: 4165[-15] 4170[-14] 4168[-13]   (mid row: Heron/Bracken/Birch)
y=-19: 4166[-15] 4171[-14] 4169[-13]   (lower row: Watercress/OldStand/SaltLick)
y=-20:           4176[-14]              (Birdsong Glade — spine continues)
y=-21: 4172[-15] 4174[-14] 4173[-13]   (south hub: Badger/TwistedHawthorn/Foxglove)
y=-22:           4175[-14]             (Steppe Edge — terminal)
```

The layout is a clean 3-column spine with spur terminals. No coordinate
overlaps. The connection to Fox Den (4156 at [-14,-14]) is correct — 4157
is at [-14,-15], one step south.

### CONCERN: Coord gap at y=-20 (Birdsong Glade)

The path 4171 Old Stand (y=-19) → 4176 Birdsong Glade (y=-20) → 4174
Twisted Hawthorn (y=-21) is a proper south chain, but rooms 4172 (Badger
Sett) and 4173 (Foxglove Clearing) are both at y=-21 with exits only to
4174 — they are "spur" terminals hanging off the south hub at y=-21. This
is correct per the plan.

However, the description of Birdsong Glade says "The forest resumes south
of the glade" and has exits north (to 4171) and south (to 4174). The prose
implies continuity — Birdsong Glade feels like a waypoint rather than a
distinctive room. Minor concern, not a bug.

### No conflicts with neighboring zones

Fernway South's x range is [-15,-13], y range is [-22,-15]. The Fernway
proper rooms (checked via 4156 at [-14,-14]) are in y >= -14. The
Marches Spur Road rooms are further east. No coordinate collision expected.

---

## Findings

### OBSERVATION: Boar drops wild hare meat

The feral boar (363) has `items: [itemid: 40064]` — which is "wild hare meat".
The goals file itself notes this ("killing the boar should drop wild hare meat
(item 40064)") so this is apparently intentional, but it's worth flagging for
player-immersion reasons. A boar killed in its wallow would plausibly yield
pork, tusks, or sinew — not hare meat. Item 40068 (sinew) exists in the
materials folder and would be a more naturalistic boar drop. The goals file
description may have been a documentation shorthand ("all prey mobs drop 40064")
rather than a design decision to have a boar yield hare meat.

**Recommendation:** Evaluate whether a boar-specific drop (sinew 40068, or a
new boar-meat item) would be more immersive. Not blocking.

### OBSERVATION: Wolf has no loot drop (intentional)

Wolf (364) has `itemdropchance: 0` — confirmed intentional per goals file.
Badger (365) also has no drop. These are "ecology mobs" rather than loot mobs.
Makes sense: the zone is about wildlife feel, not farming.

### OBSERVATION: Honey bees — combat_passive vs prey archetype

Bees use `behavior_archetype: combat_passive` while hare and deer use `prey`.
This means bees will fight back if attacked but won't flee. This is exactly
right for a swarm — you can't outrun angry bees. Good design call.

### OBSERVATION: No instance saves exist for the zone

`rooms.instances/the_fernway_south/` is empty. Zone is clean — no stale
overrides will mask template changes.

### OBSERVATION: Pine Stand is single-exit (west only)

Room 4161 Pine Stand has only `west: 4160`. There are no east/south exits.
The room description does not hint at blocked paths — the pines simply close
in. From a player navigation standpoint this is fine (the wolf-sign noun gives
reason to visit), but a player might try to push east expecting a forest path.
The "you can't go that way" response is the only feedback. Consistent with
other terminal arms (4158, 4159, 4162 are all single-exit).

### CONCERN: Boar's wallow description says "Whether the pigs themselves are
here or gone, the hollow is unmistakably theirs."

This is good atmospheric writing for when the boar is absent (between spawns).
However, if the boar IS present, the description still says "whether here or
gone" — a slight immersion break. This is a known limitation of static room
descriptions vs. dynamic mob presence and is not unique to this zone. Not a
bug, just a design note.

### PASS: Zone-config correct

`zone-config.yaml` has `name: The Fernway South`, `roomid: 4157`,
`defaultbiome: forest`, `region: Windward Marches`. All correct.

### PASS: Folder naming convention correct

Zone folder is `the_fernway_south` which matches
`ConvertForFilename("The Fernway South")` → lowercase, spaces to underscores.
Mob folder is `the_fernway_south`. No filename/name-field mismatch risk.

### PASS: Cross-zone boundary wiring

Room 4157 north→4156 (zone: The Fernway) and room 4156 south→4157
(zone: The Fernway South) are both present and consistent.

### PASS: Forage item 40066 blood-moss present in game world

Room 4166 Watercress Bend describes `blood-moss` as "a dyer's ingredient and
an alchemist's staple" — item 40066 exists in materials-40000. However, no mob
in the zone drops blood-moss as loot. Blood-moss appears to be a foraging item
(if the forage command supports it) rather than a drop. This is consistent with
the zone design focus on mob ecology over resource farming.

---

## Blocked Goals Summary

The following goals require a live server and could not be tested:

1. Spawn room confirmation (users/17.yaml edit pickup)
2. Walking all 20 rooms and reporting prose experience
3. `look <noun>` in-game response verification (all 80 nouns)
4. Wildlife mob spawn encounters
5. Wolf-vs-boar combat observation
6. Wolf non-aggro confirmation (in-game)
7. Badger aggro confirmation (in-game)
8. `map` render verification

---

## Raw Stats

- Commands sent: 0 (server down)
- Live sessions: 0
- Fights: 0
- Deaths: 0
- Rooms statically audited: 20/20
- Mob files audited: 6/6
- Item files audited: 2/2
- Noun definitions verified: 80/80
- Bugs found: 0
- Concerns: 2 (boar drop immersion; Birdsong Glade description feel)
- Observations: 6
- Goals PASS (static): 4
- Goals BLOCKED (need live server): 8
- Goals FAIL: 0

---

## Recommendation

**Restart the MUD server and re-run this test live.** All static checks pass.
The zone is structurally sound, noun coverage is complete, cross-zone wiring
is correct, and the prose quality is high. The four live-only goals (wolf
behavior, badger aggro, room navigation feel, map render) are the primary
remaining unknowns. The boar-drops-hare-meat observation is worth discussing
before the live test so the tester knows whether to flag it as a bug.
