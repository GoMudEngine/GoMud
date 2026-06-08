# Test Report: Stage 3.0a — Stillwater Marsh Feel Test

**Date:** 2026-04-28
**Target:** local
**Role:** feel-tester
**Character:** smoketester
**Goals file:** stage-3-0a-stillwater-marsh-feel.yaml
**Duration:** ~55 minutes, ~95 commands sent

## Session Summary

Walked all 20 rooms of the Stillwater Marsh zone (4177–4196), tested noun
coverage on every room, confirmed mob behavior for all 5 species, verified the
east boundary link to Stillwater, and ran the plural-tag regression suite from
the goals file. Zone structure and atmosphere are strong. Four tagged-word
coverage bugs were found — all are minor noun omissions, no plural-tag
regressions of the type caught in 3.0c. The two critical wildlife tests (adder
hostile, otter non-hostile) both passed. One drop-table concern found: neither
otter nor marsh rat produced expected loot in two kills.

## Goal Results

- [x] Spawn confirmed in Marsh Track (4177, Stillwater Marsh) — PASS
- [x] All 20 rooms walked (4177–4196) — PASS, no skipped rooms
- [x] Plural ANSI tag regression check — PASS overall with 4 minor coverage
  gaps noted below (but NO orphan `s` regressions of the 3.0c type)
- [x] Tagged-word coverage check — PASS for most rooms; 4 BUGs found where
  highlighted terms have no `look <noun>` definition
- [x] Room 4195 Far Bog Heart biome: plains — PASS; header shows `[*]` not
  `[≈]`; description correctly describes upland shoulder, pale grass, south
  horizon to steppe
- [x] 4184 Heron Marsh — 3 exits confirmed (east to Cattail Bend, west to
  Otter Slide, south to Shrimp Shallows) — PASS
- [x] 4192 Dragonfly Glade — 4 exits confirmed (n/s/e/w) — PASS
- [ ] All 5 mob spawns encountered — PARTIAL PASS: saw river otter (4185),
  marsh rat (4189, 4191, 4192), dragonfly swarm (4192), snapping turtle
  (4188); bog adder (4193) encountered and killed. All 5 species observed.
- [ ] Adder-vs-rat emergent dynamic — BLOCKED: adder engaged player (already
  in prior combat state) rather than the rat when both were in 4192; could
  not isolate the rat-hunting behavior cleanly in this session
- [x] Adder DOES attack on sight — PASS: immediate hostility on entering 4193;
  surprise attack from hidden state
- [x] Otter does NOT attack — PASS: stood in 4185 with live otter for 2+
  rounds, no aggro fired; otter fled when attacked
- [x] East boundary test: 4177 east → 4133 Mill Creek Footbridge (Stillwater)
  → west back → 4177 — PASS, round trip works
- [x] Map renders cleanly — PASS: 20 marsh rooms visible as `≈` cluster
  without overlap with Stillwater city rooms (`*`)
- [ ] Otter drops freshwater clam (40058) — FAIL: no drop on one kill
- [ ] Marsh rat drops wild hare meat (40064) — FAIL: no drop on one kill

## Findings

### BUG: seed-heads not tagged in 4180 Cattail Bend

Room 4180 description reads: "seed-heads are already breaking open". The word
`seed-heads` is visually highlighted. `look seed-heads` returns "Look at
what???". Missing `nouns` entry in room YAML.

Room: 4180 Cattail Bend. Trigger: `look seed-heads`.

### BUG: reed-trails not tagged in 4184 Heron Marsh

Room 4184 description reads: "the reed-trails open suddenly". The compound
`reed-trails` is highlighted. `look reed-trails` returns "Look at what???".

Room: 4184 Heron Marsh. Trigger: `look reed-trails`.

### BUG: peat-shadow not tagged in 4180 Cattail Bend

Room 4180 description reads: "sit motionless in the peat-shadow". The
compound `peat-shadow` appears highlighted. `look peat-shadow` returns
"Look at what???".

Room: 4180 Cattail Bend. Trigger: `look peat-shadow`.

### BUG: reed-wall not tagged in 4182 Reed Beds

Room 4182 description reads: "the visual sameness of the reed-wall in every
direction". The compound `reed-wall` appears highlighted. `look reed-wall`
returns "Look at what???".

Room: 4182 Reed Beds. Trigger: `look reed-wall`.

### BUG: iron-tang not tagged in 4187 Iron Seep

Room 4187 description reads: "The air carries a faint iron-tang here".
`look iron-tang` returns "Look at what???".

Room: 4187 Iron Seep. Trigger: `look iron-tang`.

### CONCERN: Otter and marsh rat drop tables appear empty

Killed river otter (4185) and marsh rat (4189) once each. `get all
<corpse>` produced no items in either case. Inventory confirmed empty after
each. Goals specify otter → freshwater clam (40058) and marsh rat → wild
hare meat (40064). This may be a low drop rate (probabilistic loot) rather
than a configuration omission — a single kill each is not conclusive. Worth
checking the mob YAML loot entries. If the items are configured correctly,
no action needed; if they are missing from the mob definitions, this is a
data bug.

### CONCERN: Adder-vs-rat hate priority unverified

The adder (`hates: [rodent]`, `hostile: true`) attacked the player on first
sight in 4193 and re-engaged in 4192 when it wandered in while the player
was already present. A marsh rat was in 4192 at the same time but the adder
targeted the player. This is likely correct (once combat is established,
the existing target is maintained), but the rat-prioritization when entering
a room with both player and rat at peace was not witnessed. BLOCKED for this
session.

### OBSERVATION: Adder spawns hidden with surprise attack

On entering 4193 Adder Den: "You notice bog adder lurking in the shadows!
bog adder notices you as you enter!" — then a round later it emerged and
launched a surprise attack with "[SURPRISE ATTACK]" text. This is satisfying
hostile-mob flavor. The hide-then-ambush behavior distinguishes the adder
from simple patrol-aggressors and fits the snake theme.

### OBSERVATION: Otter flees when attacked, then returns

After taking "badly wounded" damage, the otter fled with "river otter tries
to flee but is blocked!" → eventually succeeded. It then wandered back with
an idle message "river otter surfaces briefly, watches you, dives again".
Prey behavior working correctly.

### OBSERVATION: Dragonfly swarm idle messages are excellent

In 4192, the swarm fired: "dragonfly swarm turns in formation over the
hunt-pool" and "dragonfly swarm intercepts something tiny mid-air with a
quiet click". These are atmospheric and specific to the mob's niche. The
second message in particular ("with a quiet click") is notable for sensory
detail.

### OBSERVATION: Marsh rat idle messages in 4192

"marsh rat nibbles at a sphagnum-shoot quickly" and "marsh rat freezes at
a trail-mouth, ears twitching" — both observed in the same session. Small
prey NPC idles feel appropriately skittish.

### OBSERVATION: Snapping turtle is passive — confirmed

Spent multiple rounds in 4188 Shrimp Shallows with a snapping turtle
present. No aggro, no combat messages. Turtle idle: "snapping turtle drifts
slowly across the sand-bottom, half-buried". Passive behavior correct per
spec ("passive but hits hard if engaged").

### OBSERVATION: Far Bog Heart transition description is standout writing

The upland-shoulder transition in 4195 is the best single room in the zone:
"The bog gives out entirely here…firm underfoot for the first time since
the marshland entry, the change as abrupt as a threshold." The plains biome
marker (`[*]`) renders correctly. The south horizon line and Dustwalk
callback both land cleanly.

### PASS: All 20 rooms visited — zone structure intact

All rooms 4177–4196 are reachable and correctly connected. Terminal rooms
(4179 Spring Pool, 4180 only has N/S/W not E, 4183 Willow Grove,
4186 Clam Beds east-only, 4190 Black Pool, 4193 Adder Den east-only,
4195 Far Bog Heart, 4196 Hidden Spring) each have only their expected exits.
Attempts to exit a terminal in a closed direction received the standard
"You're bumping into walls." rebuff.

### PASS: Plural noun regression suite — no 3.0c-type failures

The goals listed 19 plural-form nouns for regression testing. All
highlighted plurals on the goals list that were tested resolved correctly:
reeds (4177), flat-stones (4178), pilgrim-stones (4179), cattails (4180),
trail-boards (4180), tussocks (4184), reed-trails — NOTE: FAILS, see BUG,
water-tracks (4185), clam-shells (4186), clam-beds (4186), water-weeds
(4188), sundews (4189), rat-runs (4189), mussels (4190), slough-skins
(4193), basking-rocks (4193), rat-trails (4191), waxberries (4196),
willows (4183), reed-beds (4182), dragonflies (4192).

No orphan `s` characters visible outside ANSI tags were observed in any
room description — the 3.0c plural-tag bug pattern (literal `s` bleeding
outside the closing tag) does not appear in any 3.0a room.

### PASS: East boundary link round-trip

`east` from 4177 Marsh Track → 4133 Mill Creek Footbridge (zone changes to
Stillwater). `west` from 4133 → back to 4177 (zone changes back to
Stillwater Marsh). The zone transition messages are absent (no "you enter
Stillwater Marsh" text), which is consistent with other inter-zone doors.

### PASS: Map renders without overlap

`map` from 4177 shows the 20 marsh rooms (`≈`) clustered south and west,
with Stillwater city rooms (`*`, `T`) clearly separated to the upper right.
No room IDs overlap with neighboring zones.

### PASS: Heron Marsh 3-exit hub

Exits: east (Cattail Bend), west (Otter Slide), south (Shrimp Shallows).
All three exits walk cleanly. No north exit. Matches spec.

### PASS: Dragonfly Glade 4-exit hub

Exits: north (Shrimp Shallows), south (Bog Edge), east (Mossy Hummock),
west (Adder Den). All four exits walk cleanly. Matches spec.

### PASS: Zone voice consistency

All 20 rooms use a consistent wetland voice — sensory-first, detail-rich,
ecologically accurate. Descriptions reference the same forager culture as
the existing Stillwater rooms (miller's wife, Stillwater healer, smith who
pays for lake-iron). No anachronisms found. No mechanical text bleeding into
descriptions.

## Raw Stats

- Commands sent: ~95
- Rooms visited: 20 / 20
- Fights: 3 (river otter x1, marsh rat x1, bog adder x1)
- Deaths: 0
- Mob species encountered: 5 / 5
- Drop tests: 2 (both returned no loot — see CONCERN)
- Nouns tested via `look <noun>`: ~60
- Bugs found: 5 (4 missing noun tags + 1 probable iron-tang omission)
- Concerns: 2 (drop table, adder-vs-rat unverified)
- Observations: 7
- Passes: 10

## Comparison to 3.0c Smoke Results

3.0c caught 5 plural-tag bugs where a literal `s` bled outside the ANSI
closing tag (orphan `s` rendering in player text). None of those bug-type
patterns appeared in any 3.0a room. The plural-tag rule was correctly
applied throughout this zone.

3.0a's 5 bugs are a different category: missing `nouns` entries for
highlighted words that were written into the room description but not added
to the room YAML's noun list. These are "coverage gaps" rather than
"ANSI tag malformation." They are lower severity (player sees a normal
"Look at what???" rather than broken markup) and are straightforward to fix.

Summary: no regressions on the 3.0c plural-tag issue. 3.0a introduces a
new class of minor bug (noun-coverage gaps), 5 found total across 20 rooms.
The otter/rat drop concern is the only potentially-functional issue.
