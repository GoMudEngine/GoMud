# Test Report: Stillwater (Zone 2.2) — Stage A Partial Build Smoke Test

**Date:** 2026-04-25
**Target:** local
**Role:** feel-tester
**Character:** smoketester
**Goals file:** stillwater-stage-a-partial.yaml
**Duration:** ~30 minutes, ~50 commands sent

## Session Summary

Walked the six-room Stage A spine of Stillwater from the southern arch
(4100) up through South Approach (4101), Lakefront Square (4102), into
the Pike & Lantern yard (4103), the common room (4104), and up to the
lodging loft (4105), then reversed the entire journey back through the
zone boundary to north_road's North Road End (4062). Examined every
declared noun in every room, exercised the missing-direction handling
for unbuilt Stage B/C exits, watched for idle messages in two rooms,
and tested the 4104 cooking_fire station via `craft`. The prose is
strong throughout — Stillwater reads as a real, lived-in lakefront town
distinct from the road behind it. Two issues worth flagging: the inn
uses non-cardinal `enter`/`leave` exit names (CLAUDE.md requires
cardinals only), and 4102's declared `T` Townsquare mapsymbol shows up
in the legend but not on the rendered map tile.

## Goal Results

- [x] **4100 entry & nouns** — PASS. Room reads as evocative arrival
  prose. All four nouns reward examination richly. The notice-board
  successfully seeds three quest hooks: lake-caves bounty (silver from
  the constabulary), missing fisherman Elgar Voss, and the lost dog
  Sumac near the docks. Flagstone arrival cue ("the road firms beneath
  your feet here, the gravel laid fresh") is excellent.
- [x] **4101 spine walk** — PASS. The road-to-town transition feels
  right — gravel gives way to old paving, cottages line both sides,
  woodsmoke and lake-coolness change the sensory palette from the gate.
  Milestone, flagstone, and cottages nouns all distinct and grounded.
  The milestone naming three southern towns (Ashwick / Watchers
  Crossing / Thornwall) and going blank northward is a great
  end-of-the-road touch.
- [x] **4102 Lakefront Square + map** — PARTIAL. Square reads as a
  proper hub: well, slate map-board chalked with road labels, lake
  vista over the retaining wall, worn paths radiating from the well.
  All five nouns rich. **However:** `map` shows the room rendered as
  `*` (default city glyph), not `T` (Townsquare). The legend DOES list
  `T Townsquare` — so the symbol is registered but not rendering at
  the tile. See CONCERN below.
- [x] **Missing-direction handling** — PASS. From 4102 `east` and
  `north`, and from 4103 `north` and `west`, all return
  `"You're bumping into walls."` Graceful, no crashes, no
  broken-exit warnings.
- [x] **4103 yard & nouns** — PASS. The smell-before-sight opener
  ("pan-fried lake-pike, hot butter, and the dark malty undertone of
  beer") is a great inn arrival. Hitching-rail (with iron rings sized
  for pony and two horses), net-bunting (hung "before the old
  constable's time"), sign, and inn nouns all evocative.
- [x] **4104 common room & nouns** — PASS. The interior reads
  distinctly from the yard — yellow lamplight, low beams, hearth with
  iron arm and fish-pan, slate of the day's fare in copper not silver.
  All six nouns examined cleanly. Atmosphere lands.
- [x] **Cooking station / craft** — PASS. `craft` (no args) lists
  recipes and the cooking entries show `[cooking fire]` (station
  available), with `missing raw-meat` flagged correctly. `craft grilled
  meat` — graceful failure: `"You are missing: raw-meat."` No crash,
  no station-registration warning. (Note: the goal said try
  `recipes` and `craft grilled-meat` — `recipes` is not a recognized
  command, and the hyphenated form returns "No recipe found"; you must
  use `craft grilled meat` with a space. Probably worth updating the
  goal text.)
- [x] **4105 loft & nouns** — PASS. Distinct atmosphere: clean linen,
  beeswax, lake-air through the gable window. The bedsteads description
  ("uncomfortable for a single night, surprisingly forgiving on the
  second, and after a week of road-sleep nearly luxurious") is one of
  the best lines in the whole zone. The herbs noun successfully seeds
  curiosity about local materials — lake-mint, marsh-sage, and
  marsh-willow bark are concrete, plausible alchemy/cooking inputs.
- [x] **Loft idle messages** — PARTIAL. One idle message fired during
  a ~70-second wait: *"Below, in the common room, a low murmur of
  voices rises briefly, then settles."* Room-appropriate; the cadence
  felt sparse but not annoying. An earlier 64-second wait produced no
  ambient text at all, suggesting the timer can run quite long between
  messages. May want to verify the loft's idle pool isn't unexpectedly
  small.
- [x] **4102 idle messages** — PASS. Two ambient lines fired in roughly
  70 seconds: *"A wagon turns in from the south road, its driver
  calling a horse to a stop near the western lane,"* and *"The
  lake-wind shifts and the smell of drying fish carries through the
  square for a moment."* Both excellent — concrete, sensory, true to
  the room. Cadence felt right for a town hub.
- [x] **Reverse journey** — PASS. `down` (loft → common), `go leave`
  (common → yard), `east` (yard → square), `south`×3 (square → approach
  → gate → North Road End). All six steps worked. Zone boundary at
  4100 ↔ 4062 traversed cleanly in both directions, no one-way issues.
- [x] **Map from 4062** — PASS with caveat. The Stillwater spine
  renders north of the player marker, the connection is visually
  contiguous, and the legend correctly shows `, Farmland` for
  north_road plus `* Default` for the Stillwater rooms (note: from
  4062's vantage the legend label is `* Default`, while from inside
  Stillwater the SAME glyph is labeled `* City` — a minor cross-zone
  legend inconsistency but not a coordinate clash). No overlaps, no
  weird coordinate jumps.
- [x] **Overall feel** — see narrative section below.
- [ ] **Server log check** — BLOCKED. `Logging.LogToFile: false` in
  the local config, so nothing is being written to `_datafiles/logs/`
  by the running server. The on-disk `server.log` and `startup.log`
  are stale (Feb 15 / Apr 15). No way to grep the live server's
  warnings without restarting with logging enabled. In-game session
  produced no visible YAML, missing-exit, or station-registration
  errors during play.

## Findings

### BUG: 4103 ↔ 4104 use non-cardinal `enter` / `leave` exits

The yard (4103) shows `Exits: east, enter` and the common room (4104)
shows `Exits: leave, up`. CLAUDE.md memory rule
`feedback_cardinal_exits_only.md` requires
`north/south/east/west, not enter/leave`. Beyond the convention
violation, the practical UX issue is severe: `enter` is not a
recognized command on its own — typing `enter` returns
`"enter not recognized. Type help for commands."` You must type
`go enter` (or `go leave`) to traverse, which a new player has no
reason to discover. The exit list itself is what tipped me off, and I
only got in by trying `go enter` after exhausting cardinals, `in`, and
the bare word. Recommend renaming both directions to cardinals
(probably `north`/`south` since the inn sign and the yard sit on the
square's western flank with the common room behind it — pick whichever
matches the intended layout).

### CONCERN: 4102 `T` Townsquare mapsymbol registered but not rendered

`map` legend lists `T Townsquare` once you've seen 4102, confirming the
mapsymbol field is parsed and registered. But on the rendered map the
4102 tile shows as `*` (the default city glyph), not `T`. Verified by
viewing the map from both 4101 (south of 4102, T should appear in the
row above @) and 4103 (yard, T should appear in the row east of @) —
in both views the 4102 cell renders as `*`. The mapsymbol declaration
is being read but not applied at draw time, OR the city default is
overriding the per-room symbol. Worth a quick check of the map render
code to confirm precedence.

### CONCERN: Phantom `*` tiles east of 4101 and east of 4102 on the map

When viewing `map` from 4101, the row at the player's level shows
`@   *` — but no Stage A room exists east of 4101. Similarly from
4103, the row above @ shows `*---*---*` (3 rooms in a horizontal line)
where only 2 rooms (yard, square) exist on that axis. Could indicate
that one of the omitted Stage B/C exits in 4102's YAML is still
present as an exit reference (just pointing to a not-yet-built room
ID), or the map renderer is showing placeholder tiles for stubbed
neighbors. Not a crash, no error in-game, but worth verifying the
4101/4102/4103 YAML doesn't have stale `4106`/`4107`/etc. exit entries.

### CONCERN: Loft idle-message cadence may be too slow

In 64 seconds of idle in 4105 the loft produced zero ambient text. A
second 70-second wait produced one message. By contrast 4102 produced
two messages in roughly the same window. Not necessarily a bug — a
quiet inn loft is thematically right — but if the design target is
"periodically display ambient flavor," one message per 70+ seconds is
on the edge of feeling dead, especially in a room a player might wait
in for sleep/rest mechanics. Suggest reviewing the loft's idle pool
size and weight relative to 4102's.

### OBSERVATION: Time-of-day prompt change midway through the session

The prompt glyph shifted from `☀️` to `☾` while I was in the inn
(somewhere around `look hitching-rail`). No bug — just a nice
world-tick detail confirming the day/night cycle ticks even during
flavor exploration. Worth noting because some of the room descriptions
reference "this hour" or "at midday" (4101 cottages, 4104 hearth) and
on a long visit those static lines start to clash with the prompt.
Long-term not a Stage A concern.

### OBSERVATION: Notice-board does heavy lifting for the zone

The 4100 notice-board single-handedly seeds three quest hooks (lake
caves bounty, Elgar Voss missing-person, Sumac the lost dog) plus a
piece of economic flavor (bait-fish farmer). For a player who walks
in cold this is a dense but well-organized hook surface — the layered,
rain-bled postings framing tells the player at a glance that this is
a town with ongoing concerns. When Stage B/C drop, make sure the
NPCs and quest givers reference the same names and locations from the
board (Elgar Voss, Sumac, "eastern lake caves") for continuity.

### PASS: Map-board in 4102 is a great UX prop

The chalked slate on the well's eastern face naming "constabulary,
north square" and arrowing east to "lake" is exactly the kind of
lightweight diegetic signposting that helps new players orient in a
hub. When 4109+ rooms (north square, constabulary, lake) come online,
the map-board text already tells the player they exist.

### PASS: Sensory progression from gate → approach → square is excellent

The gate room leads with sound (chimes, distant smithy clang), the
approach with smell (woodsmoke, kitchen chimneys), and the square
opens with sight (the lake spreading enormous to the horizon). Three
rooms, three different lead senses, none repeating. Reads as
deliberately authored.

### PASS: Cooking station detection works correctly

`craft` recognizes the cooking_fire in 4104 — recipes show
`[cooking fire]` (available) rather than the
`need cooking fire [cooking fire]` you'd see in a non-station room.
Recipe gating on missing ingredients fails gracefully. Stage A's only
station works.

### PASS: Zone boundary 4100 ↔ 4062 is invisible to the player

Crossing south from Stillwater's gate room to north_road's "North Road
End" produces no zone-change banner, no jarring style shift. The 4062
description picks up Stillwater's name on the horizon ("woodsmoke and
fish, tallow and wet rope. Stillwater.") which feels like a real
narrative handoff. Map renders both zones contiguously.

## Overall Feel — Summary

**(1) Different place than the road?** Yes, decisively. The road in
north_road 4062 is wagon-rutted, smoke-on-the-horizon, "wilds reach
their end" prose. Within two rooms of crossing into Stillwater you
have flagstone underfoot, drying-fish smell, the lake's brackish
coolness, and the iron-clang of a smithy. The brief — "first finished
town after a long road" — is delivered.

**(2) Enough flavor without bloat?** Yes. Each room sits at roughly
the right density — descriptions run 5-7 lines, every room has 4-6
examinable nouns, no room reads as word-soup. The longest descriptions
(4102, 4104) earn it because they're hubs.

**(3) Empty / generic / underwhelming rooms?** None of the six. Even
4101 South Approach — which exists primarily as connective tissue
between the gate and the square — pulls its weight with the milestone
gag (south faces inscribed, north face blank).

**(4) Overstuffed / word-soup rooms?** None. Closest call is 4102
(four sentences in the room body plus five distinct nouns), but the
content is well-organized — well-as-landmark in the middle, retaining
wall and lake to the east, map-board on the well, smells anchoring
the whole. It reads coherently.

**(5) Quest hooks pull the player in?** Yes. After the notice-board
read I genuinely wanted to look for the eastern lake caves (the
constabulary bounty has stakes — fishing nets being savaged is a
specific harm), and the missing fisherman has narrative pull
specifically because the notice is "half torn away, the ink long
faded" — the framing implies the town has already given up on Elgar
Voss, which is more interesting than a fresh missing-person poster.
Sumac is light comic relief but works as a side-hook. The herbs in
the loft (lake-mint, marsh-sage, marsh-willow bark) are an additional
crafting hook — the player now knows which materials to look for in
nearby biomes.

**(6) Inconsistencies between rooms?** Two minor:
- The 4103 yard description mentions "the inn's main door stands open
  under a carved sign of a leaping pike" — but to actually go in you
  must type `go enter`, not `north` or `in`. This is the cardinal-exit
  bug above. Diegetically the door is open and inviting; mechanically
  the path is hidden behind a non-obvious command.
- The 4102 map-board text mentions a north square and a constabulary
  to the north; the room's own exit list correctly omits north (those
  rooms aren't built). The map-board is forward-looking and that's
  fine — but make sure when 4109/4110 ship, those references match
  exactly.

## Raw Stats

- Commands sent: ~50
- Fights: 0
- Deaths: 0
- Spells cast: 0
- Items used: 0
- Bugs found: 1 (`enter`/`leave` non-cardinal exits)
- Concerns: 3 (T mapsymbol not rendering, phantom map tiles, loft idle cadence)
- Observations: 2 (time-of-day glyph change, notice-board hook density)
- Passes: 4 (map-board signposting, sensory progression, cooking station, zone boundary)
