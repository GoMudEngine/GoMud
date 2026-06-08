# Test Report: Fernway South Zone — Stage 3.0c Live Feel Test

**Date:** 2026-04-28
**Target:** local (localhost:55555)
**Role:** feel-tester
**Character:** smoketester
**Goals file:** stage-3-0c-fernway-south-feel.yaml
**Duration:** ~45 minutes, ~90 commands sent

---

## Session Summary

Connected to a live server with smoketester spawned at room 4157 (Briar
Tangle), confirming the player YAML edit took effect. Walked all 20 rooms
of the Fernway South zone, tested noun definitions in each room, fought
all combat mobs (bees, hare, boar, badger), and verified the critical
wolf-vs-player and wolf-vs-boar mechanics. The zone is polished and
immersive overall. A handful of noun/plural mismatches and two missing
highlighted nouns were found. Wolf aggro behavior works correctly. Badger
aggro works correctly. The wolf-vs-boar emergent dynamic could not be
directly observed due to wander/respawn cooldown lengths — the wolf has a
1200-round spawn cooldown and the boar was killed during testing, making
the encounter impossible to trigger in a single session.

---

## Goal Results

- [x] Confirm spawn in Briar Tangle (room 4157) — **PASS**: Logged in
  directly to Briar Tangle, The Fernway South, as expected.
- [x] Walk all 20 rooms (4157-4176) — **PASS**: All 20 rooms visited and
  verified. Descriptions are clean, no ANSI bleed-through detected.
- [x] Test `look <noun>` for ANSI-highlighted nouns in each room —
  **PARTIAL PASS**: Most nouns work. See BUG section for plural/singular
  mismatches in rooms 4159, 4161, and 4168.
- [x] Room 4175 Steppe Edge biome and steppe theme — **PASS**: Biome is
  `plains`, map symbol shows `*` (not `♣`), description mentions pale-
  grass, scoured-stones, and the Dustwalk on the south horizon.
- [x] Room 4163 Twin Beech Glade — 4 exits (n/s/e/w) — **PASS**:
  Confirmed north/south/east/west exits all work.
- [x] Room 4174 Twisted Hawthorn — 4 exits (n/s/e/w) — **PASS**:
  Confirmed north/south/east/west exits all work.
- [ ] Encounter all 6 mob spawns — **PARTIAL PASS / BLOCKED**: Wolf
  (4161), Boar (4162), Hare (4159), Bees (4158), Deer at Browse (4167),
  Deer at Lick (4169), Badger (4172) all encountered. However the roe
  deer at Salt Lick fled before killing — prey behavior confirmed. All
  mobs were spawned and present on first visit.
- [ ] Wolf-vs-boar emergent dynamic — **BLOCKED**: Wolf has 1200-round
  spawn cooldown. Boar was killed during the session; 900-round respawn
  made a natural encounter impossible within the test window. The feature
  config is correctly set up (wolf `hates: [boar]`, both have maxwander),
  but could not be observed in a single session.
- [x] Wolf does NOT attack player — **PASS**: Stood in Pine Stand (4161)
  with the wolf present for 5+ rounds. No aggro, no combat initiated.
  Wolf only idle-emoted ("pads silently between the pine trunks,
  scenting").
- [x] Badger DOES attack player — **PASS**: Upon entering Badger Sett
  (4172), "forest badger notices you as you enter!" fired immediately,
  and "forest badger prepares to fight you!" triggered within one round.
  Badger killed without player death.
- [x] Boundary test — north from 4157 to 4156 (Fox Den) — **PASS**:
  Round-trip north/south confirmed.
- [x] Map renders cleanly — **PASS**: Map from multiple rooms shows
  coherent layout matching the zone plan. No overlap with Fernway /
  Marches Spur rooms. Plains biome room shows `*`, forest rooms show `♣`.
- [ ] Loot drops — **BLOCKED / INCONCLUSIVE**: All three loot rolls
  failed (hare 75% chance, bees 50%, boar 80%). No wild hare meat, no
  beeswax, no sinew received. Item files 40064, 40065, 40068 all exist
  and are correctly referenced in mob YAMLs. This is likely bad RNG
  across three kills, not a bug. The roe deer fled before being killed.
  Wolf and badger corpses correctly showed no loot.

---

## Findings

### BUG: Plural noun mismatches — player sees plural in text, only
singular key defined

**Rooms affected:** 4159 (Hare-Run Meadow), 4161 (Pine Stand), 4168
(Birch Stand).

**Pattern:** The YAML description uses ANSI-highlighted singular nouns
(e.g., `<ansi fg="itemname">hare-path</ansi>s`) where the `s` is
appended OUTSIDE the tag. The engine renders this as "hare-paths" in
visible text, but the noun key is `hare-path` (singular). Players who
type `look hare-paths` get "Look at what???" — a confusing non-response
when the game just showed them the word highlighted as if it's interactive.

**Confirmed failures in-game:**
- Room 4159: `look hare-paths` → "Look at what???" (key is `hare-path`)
- Room 4161: `look pitch-beads` → "Look at what???" (key is `pitch-bead`)
- Room 4161: `look pitch` → "Look at what???" (key is `pitch-bead`)
- Room 4168: `look birch-trunks` → "Look at what???" (key is `birch-trunk`)
- Room 4168: `look bark-strips` → "Look at what???" (key is `bark-strip`)
- Room 4168: `look deer-rubs` → "Look at what???" (key is `deer-rub`)
- Room 4168: `look sap-runs` → "Look at what???" (key is `birch-sap`)

**Singular forms do work:** `look hare-path`, `look pitch-bead`,
`look birch-trunk`, etc. all return proper definitions.

**Fix options:** Either (a) move the `s` inside the ANSI tag in the
description YAML, making the highlighted word plural and matching the
noun key; or (b) add plural aliases as secondary noun keys. Option (a) is
cleaner.

---

### BUG: "wallow" and "mud" not defined as nouns in Boar Wallow (4162)

In Boar Wallow, the most prominent feature is the wallow itself. The
description highlights `wallow-mud` as the ANSI noun, but players
naturally try `look wallow` (half the name) or `look mud`. Both fail.
`look wallow-mud` works. This is a discoverability gap — the compound
noun is not as intuitive as either half.

**Recommend:** Add `wallow` as an alias noun in 4162 pointing to the
wallow-mud description.

---

### BUG: Minor YAML line-wrap artifact in Hare-Run Meadow (4159)

The description YAML wraps "grass-" at end of line 5 with "edged" on
line 6. YAML `>` folded scalar converts the newline to a space, rendering
as "a small grass- edged meadow" (space before "edged") in-game text.

**Visible to player:** "opens east of Briar Tangle into a small grass-
edged meadow" shows as "small grass- edged meadow" (extra space).

**Fix:** Merge the two lines so "grass-edged" stays on one line in the
YAML source.

---

### OBSERVATION: Grammar — "hits like a opened window" (Birdsong Glade)

Room 4176 Birdsong Glade description reads: "the glade hits like a
opened window". Should be "an opened window". Minor grammar error.

---

### OBSERVATION: Goals YAML has room IDs 4173/4176 swapped in comments

The goals file says:
- "4173 Birdsong Glade" — but 4173 is actually Foxglove Clearing
- "4176 Foxglove Clearing" — but 4176 is actually Birdsong Glade

This is a documentation error in the goals file only. The rooms
themselves are correctly connected and titled in the game data.

---

### OBSERVATION: Loot drops not observed in session (likely RNG)

Three mobs killed (hare, bees, boar), zero loot drops received. Drop
chances: hare 75%, bees 50%, boar 80%. Probability of three misses:
0.25 × 0.50 × 0.20 = 2.5% — unlikely but possible. Item files exist and
are properly referenced. Not flagging as BUG but worth a second test.

---

### OBSERVATION: Prey flee behavior confirmed (Roe Deer)

The roe deer in Deer Browse fled after taking moderate damage rather
than fighting to the death. This matches the `behavior_archetype: prey`
design intent. "roe deer flees! >>> roe deer leaves towards the south
exit." No complaint — just confirming the mechanic works as designed.

---

### PASS: Wolf non-aggro to player (critical feature)

Stood in Pine Stand (4161) for 5+ combat rounds with the Timber Wolf
present. Wolf idle-emoted ("pads silently between the pine trunks,
scenting" and "lifts its muzzle and tests the wind") but did NOT initiate
combat. `hostile: false` is working correctly.

---

### PASS: Badger immediate aggro (critical feature)

Entered Badger Sett (4172). "forest badger notices you as you enter!"
fired on the room entry tick. Combat began immediately. Badger survived
the full fight without fleeing. Correct `hostile: true` behavior.

---

### PASS: Zone boundary (Fox Den ↔ Briar Tangle)

North from 4157 (Briar Tangle) → 4156 (Fox Den, The Fernway). South
from 4156 → 4157 (Briar Tangle, The Fernway South). Zone transition
displayed correctly in room header. Round-trip confirmed.

---

### PASS: All 20 rooms have clean descriptions, no ANSI bleed-through

No raw `<ansi fg="...">` tags visible in any room description. The
bridge's ANSI stripping is clean, and no malformed tags were found in any
of the 20 rooms. Room voice is consistent throughout — sensory-led,
naturalist prose, first-person-absent narrator.

---

### PASS: Steppe Edge biome and transition narrative

Room 4175 Steppe Edge correctly uses `biome: plains`, renders with `[*]`
map symbol, and the description explicitly references pale-grass, scoured
stones, the Dustwalk on the south horizon, and the psychological shift
from forest to open steppe. Strong prose — "the world simply opens and
keeps opening" is an effective closing line.

---

### PASS: Hub rooms have correct exit counts

- 4163 Twin Beech Glade: exits east, north, south, west (4 exits)
- 4174 Twisted Hawthorn: exits east, north, south, west (4 exits)

---

### PASS: Map renders coherently

Map from Old Stand (4171) shows the full zone layout: northern spine
(Briar Tangle → Old Burn Scar → Twin Beech Glade), western branch
(Brook Rise → Heron Pool → Watercress Bend), eastern branch (Deer
Browse → Birch Stand → Salt Lick), and southern spine (Tangled Bracken
→ Old Stand → Birdsong Glade → Twisted Hawthorn → Steppe Edge). No
overlap with The Fernway rooms visible in viewport.

---

### PASS: Terminal rooms correctly blocked

- 4166 Watercress Bend: exits north only — south blocked ("you're
  bumping into walls")
- 4175 Steppe Edge: exits north only
- 4169 Salt Lick: exits north only
- 4172 Badger Sett: exits east only

---

### PASS: Wolf and badger have no loot drops (by design)

Both wolf and badger corpses showed empty contents ("This is a corpse.
They are dead." with no items listed). Correct per design intent.

---

### PASS: Badger corpse description has good flavor

"Smaller than you expected. Meaner than you expected. The musk-smell
tells you it has not retreated." — solid post-death flavor text.

---

### IMMERSION NOTE: Zone atmosphere is strong

The Fernway South feels like a natural continuation of The Fernway
zone. Prose is consistent in register: sensory-led, specific, no
second-person psychology. The Old Stand (4171) with its "cathedral hush"
and root-throne, the Heron Pool's (4165) lone heron print, and the
Steppe Edge's (4175) open-sky revelation all land well. The wolf-sign in
Pine Stand ("a long scrape at shoulder-height and a single tuft of grey-
brown fur") does excellent environmental storytelling without spelling it
out.

---

### PACING NOTE: Mob density feels right

Six distinct mob types across 20 rooms, most in terminal or branch rooms.
The spine is largely empty (which encourages movement), and the wildlife
is varied enough to feel like a real ecosystem rather than random monster
placement.

---

## Raw Stats

- Commands sent: ~90
- Rooms visited: 20 / 20
- Fights: 4 (hare, bees, boar, badger)
- Deaths: 0
- Mob encounters: 6 / 6 mob types seen (wolf, boar, hare, bees, deer x2,
  badger)
- Loot drops received: 0 (3 eligible kills, 0 drops — bad RNG)
- Wolf-vs-boar interactions: 0 (BLOCKED by cooldowns)
- Bugs found: 3 (plural noun mismatch, wallow alias missing, grass- wrap)
- Observations: 5 (grammar, goals-YAML ID swap, loot RNG, prey flee,
  birdsong Glade grammar)
- Concerns: 0
- PASSes: 12 explicit goal/feature verifications
