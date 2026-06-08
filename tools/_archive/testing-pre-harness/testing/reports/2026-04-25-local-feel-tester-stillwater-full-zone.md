## Test Report: Stillwater Full-Zone Smoke Test

**Date:** 2026-04-25
**Target:** local
**Role:** feel-tester
**Character:** smoketester
**Goals file:** stillwater-full-zone.yaml
**Duration:** ~80 minutes, ~115 commands sent

## Session Summary

Walked the entire Stillwater zone (rooms 4100–4146) in a single session,
exercising every crafting station, every hidden-noun search target, every
narrative breadcrumb, the diagonal exit, the cave-system descent, and the
mapper render check. The zone reads beautifully — atmosphere is dense and
specific, the cross-room narrative threads (Voss family + spiral motif +
forage materials) all land — but a handful of small mechanical issues
deserve fixes before NPCs go in. Three real bugs found (cave darkness,
Bone Shoals cache description, mapper render at 4102), several smaller
concerns, and a fistful of warm passes.

## Goal Results

- [x] Town spine north–south walk — PASS: smooth single-step walk in both
  directions across 4100→4101→4102→4105→4109→4111. The 2-row-jump bug
  between 4102 and 4109 is fully fixed.
- [x] Inn (Pike & Lantern) and Loft — PASS: all four yard nouns (yard,
  hitching rail, net bunting, sign) fire from inside the inn with
  "Visible through the open door" framing. Loft `up` works as the only
  vertical entry. Loft `herbs` noun foreshadows lake-mint, marsh-sage,
  marsh-willow (continuity check below).
- [x] Constabulary merged room — PASS: bars, cell, bench, slop bucket
  all fire from the office. No `down` exit; basement is gone.
  Bonus ambient flavor message fired ("Cool stone-air sighs out from
  behind the bars").
- [x] Lakeside loop and Cave Mouth — PASS for navigation; CONCERN for
  Cave Mouth darkness (see findings).
- [x] Cave: Bone Shoals search — PARTIAL: search successfully discovered
  the `cairn cache` hidden noun, but `look cairn cache` returns the
  same description as the visible cairn — the E.V. note text and pearl
  reveal are not present. See bug below.
- [x] Hollow Sump boss room — PASS: bones+net rim, high water mark,
  heavy water displacement text all read clean. Ambient "The water
  displaces in the pool's centre, releasing a slow heavy ring outward"
  fires. Boss-encounter setup is unmistakable.
- [x] Beach + Foreshore search and forage nouns — PASS: 3 of 3 forage
  nouns at Reedy Foreshore (cattails, marsh willow, lake mint).
  Crab-Trap Beach `loose driftwood` hidden noun discovers cleanly via
  search and reveals proper hidden text (oilskin pouch). Freshwater
  clams noun also fires.
- [x] Temple cluster + Garret jeweler bench — PASS for navigation +
  station; CONCERN for Temple Garden marker search behavior.
- [x] West quarter (Cooper, Bakehouse, Mill) — PASS: bakehouse cooking
  fire confirmed, mill yard reads atmospherically.
- [x] Stable, Ulla's, Uncle's Workshop — PASS for navigation; CONCERN
  for bench-vise carving search behavior.
- [x] Healer + Cemetery cache — PASS: alchemy bench confirmed, cemetery
  `look elgar` reveals the kingfisher + "He was your brother. — V."
  note (family lore lands beautifully).
- [x] North chain (Sluice, Tailor, Wardstone, Old Chapel) — PASS:
  lake-iron forage noun reads, loom + enchanting-circle stations
  confirmed, spiral altar at Old Chapel Ruin explicitly cross-references
  all four other spiral locations in its description.
- [x] Outskirts loop + diagonal exit — PASS: SE/NW diagonal between
  4142 and 4111 works both ways. Wagon (Patience the mule) reads.
  Full south spine back to 4062 works clean.
- [x] Boat-Builder's Yard nouns — PASS: hull, shavings, brazier,
  slipway all fire.
- [x] All 7 crafting stations — PASS: cooking_fire (4103, 4134), forge
  (4106), jeweler_bench (4126), alchemy_bench (4136), loom (4143),
  enchanting_circle (4145) all register (verified via "you are missing
  <ingredient>" vs "you need to be at a <station>" response).
- [x] Mapper render check — see findings (only 4102 fails, 4123/4144/
  4145 all render correctly).
- [x] Server log scan — see findings (one critical missing-template
  error, one earlier YAML panic, several non-blocking exit warnings).

## Findings

### BUG: Cave Mouth (4121) is fully dark from outside

`east` from Lake Path Bend (4120) into Cave Mouth (4121) returns
"You can't see anything!" even though the room is described as the
outdoor entrance to a cave at the lake-edge in daylight. Without a
torch or `chrysalis-glow`, the player cannot read the bounty notice
referenced in the goal, cannot navigate the exits, and would assume
the cave is locked. Lit caves typically have at least a daylight
penumbra at the mouth. Recommend marking 4121 as lit (or partially
lit) so the constabulary's bounty notice is reachable without prep.
The deeper rooms (4127 onward) appropriately remain dark.

### BUG: Bone Shoals (4130) cairn cache description not updated after discovery

`search` discovers the `cairn cache` hidden noun ("You discover something:
cairn cache. The cairn's base hides a small natural depression in the
floor — moved-stone shadows betray it."), but `look cairn cache` returns
the SAME description as the visible cairn — no folded note from "E.V.
to Brindle," no Stillwater Black Pearl, no oilskin pouch. Compare to
Crab-Trap Beach (4115) where the discovered `loose driftwood` reveals
proper hidden text. The cairn cache hidden_noun appears to lack its own
description body, or is mis-pointing back to the visible cairn noun.

### BUG: Lakefront Square (4102) `T` mapsymbol not rendered

Confirmed at render-layer specifically:
- From 4102 itself, the legend does not list `T Townsquare` and the
  tile renders as `*` (city default).
- From 4105 (one room north), `T Townsquare` IS in the legend, but
  the tile at 4102's coordinate still renders as `*`.
- From 4111 (further north), legend instead lists `T Tunnel` (a cave
  room takes the T slot); 4102 is off-render.
- Compare: 4123 Temple of Stillwater (`+` Temple) renders correctly
  from 4102's full-map view; 4144 Old Chapel Ruin (`R` Ruins) and 4145
  Wardstone Circle (`W` Wardstone) both render correctly from adjacent
  vantages.

So the bug is narrower than the MEMORY.md hypothesis suggested — only
4102 is broken, not all city-biome rooms with mapsymbols. Both 4123
(also city biome) and the ruins-biome rooms render fine. Suspect a
config quirk on 4102 specifically (biome string, mapsymbol/maplegend
interaction with `T Tunnel`, or per-room override path).

### CONCERN: Search hidden_noun fires inconsistently

At Bone Shoals (4130) and Crab-Trap Beach (4115), `search` discovered
hidden nouns within 3–5 tries. At Temple Garden (4124, "buried marker")
and Uncle's Workshop (4138, "bench-vise carving"), 15+ searches turned
up no discovery message — yet `look marker` and `look bench-vise carving`
both returned the proper hidden text directly. Same pattern at Cemetery
(4140, "beneath Elgar's marker") — 6 searches no discovery, but
`look elgar` shows the cache. Either:
- (a) discovery probability is set drastically lower for these three
  rooms, or
- (b) the nouns are not actually flagged hidden_noun and are merely
  reachable by name without ever needing search.

Either way, players who don't blindly type guess-the-noun will miss
the spiral marker, the bench-vise carving, and the cemetery cache —
which would be a narrative loss given how well the spiral motif and
Voss family threads are woven. Worth standardising hidden_noun
behavior across the four spiral/family caches.

### CONCERN: Missing per-zone map templates produce repeated ERROR-level logs

Every `map` command in Stillwater fires:
`templates\maps\stillwater.md, templates\maps\stillwater.template`
not found (ERROR level, 8+ occurrences in this session alone). The
engine falls back to `default/templates/maps/map.template` — fine —
but the noise is unwarranted for a fallback path. Either create a
`stillwater.template` (lowest-effort fix) or downgrade the missing-
zone-template log to WARN/DEBUG.

### CONCERN: Earlier startup PANIC on 4137.yaml line 51

Server.log shows at 14:26:09:
`PANIC error=filepath: rooms: filepath: ...stillwater\4137.yaml: yaml:
line 51: mapping values are not allowed in this context`
The file appears to have been fixed since (Ulla's Parlor loaded for
us at 4137 with no issue — line 51 currently reads cleanly). Worth
noting as a fragile-YAML datapoint and reinforcing the colon-in-text
rule from MEMORY.md.

### CONCERN: "Room references non-existent" warnings during load

Stillwater rooms 4100, 4111, 4124, 4140, 4142, 4144, 4145 all show
"Room references non-exis[tent]" load-time warnings. Pattern is not
unique to Stillwater — same warning fires for many other zones
(Watchers Crossing, Dustwalk Road, Marches Spur Road, etc.) — likely
a load-order issue (rooms reference exits to rooms not yet loaded).
The walk through these rooms worked perfectly in-game, so this looks
benign at runtime but worth confirming. Also: biome shown in those
warnings (plains/ruins) doesn't match the rendered city `*` glyph,
which raises a separate question about how room biome is determined
at render time vs at load time.

### CONCERN: Minor article grammar in crafting station guard

Crafting an alchemy recipe at the wrong station printed:
"You need to be at a alchemy bench to craft that."
Should be "an alchemy bench." Minor but visible.

### OBSERVATION: Spiral motif breadcrumb is excellent

Five spiral locations:
1. 4122 Temple Approach pillars — "long spiral motif at waist-height
   — neither the Chrysalis spiral the larger temples use, nor anything
   else most travelers would recognize"
2. 4124 Temple Garden marker — "long spiral set inside an outer ring,
   neither Chrysalis nor anything else currently taught in the temple.
   Far older than the present garden"
3. 4138 Uncle's Workshop bench-vise carving — "Old work; older than
   the cottage. Whoever scratched it here knew the symbol meant
   something"
4. 4144 Old Chapel Ruin altar stone — explicitly cross-references
   pillars, garden marker, AND bench-vise carving in its description
5. 4145 Wardstone Circle altar slab — "the same motif that appears
   on the chapel ruin to the west and on the temple pillars in town"

The Old Chapel Ruin altar stone description is the keystone — it
ties everything together so a player who finds it last gets the full
"oh, *that's* what I've been seeing" reveal. Excellent design.

### OBSERVATION: Voss family lore lands

Bone Shoals cairn note (E.V. → Brindle), Cemetery beneath-Elgar's-
marker cache (kingfisher carved in Ulla's husband's hand + V's note
"He was your brother"), and Uncle's Workshop (sawdust untouched in
12 years, wooden bird carvings on Ulla's mantelpiece, husband died
fishing east) all fit together cleanly into a four-person family
unit (Brindle the smith, Elgar the missing fisherman, Ulla the
weaver/widow, Vella the healer). A player who walks the zone in any
reasonable order should be able to assemble the picture.

### OBSERVATION: Forage continuity is good but not perfectly tight

Inn loft `herbs` noun (4104) lists lake-mint, marsh-sage, and
marsh-willow bark. Reedy Foreshore (4114) has cattails, marsh willow,
lake mint. Cattails are NOT foreshadowed in the inn loft (though they
ARE picked up in the Tailor's Cottage description as "cattail-down
lined cloaks"). 2 of 3 forage materials get the inn-loft callback;
the third (cattails) gets its callback at the loom instead. This
is fine, but worth knowing if a future content pass wants to tighten.

### OBSERVATION: All 7 crafting stations register cleanly

Verified by attempting station-specific recipes and observing the
"You are missing: <ingredient>" response (which means the station IS
present) vs the "You need to be at a <station>" response (which means
it's not). Pike & Lantern (4103) cooking_fire, Brindle's Smithy (4106)
forge, Pearl-Carver's Garret (4126) jeweler_bench, Bakehouse (4134)
cooking_fire, Healer's Cottage (4136) alchemy_bench, Tailor's Cottage
(4143) loom, Wardstone Circle (4145) enchanting_circle — all work.
At Healer's Cottage I accidentally completed a craft attempt (I had
healer's-root and a glass vial, just no bottle); it failed naturally
("the mixture separates and goes cloudy") which is the right outcome.

### OBSERVATION: Diagonal SE/NW exit works both ways

4142 → SE → 4111 and 4111 → NW → 4142 both fire cleanly. Mini-map
shows the diagonal connector glyph (`\`) correctly.

### OBSERVATION: UPPERCASE notice at Cave Mouth

The Cave Mouth (4121) room description reads
"a small constabulary notice nailed to a wooden board beside the
entrance reads SOMETHING IN HERE. BOUNTY DOUBLED. ASK CONSTABLE."
The all-caps text fits the in-fiction urgency of a posted notice but
reads slightly mechanical embedded in prose. Minor flavor call —
either replace with sentence case + italic-style phrasing, or wrap in
quotation marks.

### PASS: Spine walk and reverse-spine walk

4100 → ... → 4111 north and 4111 → ... → 4062 south both work as a
single uninterrupted single-step path with no stuck rooms.

### PASS: Constabulary cell nouns

bars, cell, bench, slop bucket — all four reachable from inside the
office with proper "Visible through the bars" framing. Ambient
"Cool stone-air sighs out from behind the bars" fires too.

### PASS: Inn yard nouns

yard, hitching rail, net bunting, sign — all four reachable from
inside Pike & Lantern with "Visible through the open door" framing.
The wedding-bunting backstory and the "before the old constable's
time" dating give it real history.

### PASS: Boat-Builder's Yard nouns

hull, shavings, brazier, slipway — all four read with rich detail
including the "leaping-pike sigil burned in" identifier on the bow.

### PASS: Hollow Sump boss room atmosphere

Bones-and-net rim, high water mark at chest height, slow heavy
displacement, plus ambient "The water displaces in the pool's centre"
fired during the visit. The boss is unspawned but the room sells the
encounter unmistakably.

### PASS: Family-lore caches

Cemetery beneath-Elgar's-marker cache (kingfisher + "He was your
brother. — V.") and Bone Shoals cairn cache (note from E.V. to
Brindle — though see bug above re: description) connect cleanly to
Ulla's parlor (carved birds on mantel) and Brindle's Smithy.

### PASS: Crafting station inventory

All 7 stations across 6 rooms register correctly.

## Overall Feel

(1) **Cohesion** — Stillwater feels like one place. Town spine,
lake quarter, temple cluster, west residential, north outskirts each
have a distinct neighborhood character (working-class flagstone, lake-
brine and rope, sanctified hush, domestic quiet, rural verge) but
share the slate/lake-stone/leaping-pike vocabulary throughout. Smells
in particular do a lot of work: woodsmoke and fish at the gate,
brine and tar at the docks, beeswax and herbs in the temple, sweet
shaved oak in the cooper's lane. A player walking it end-to-end
absolutely knows they're in one town.

(2) **Cross-room narrative legibility** — The Voss family thread
(Brindle → Elgar → Ulla → Vella) is legible if you visit the rooms
in any reasonable order: notice board mention of Elgar Voss → Bone
Shoals cairn note from E.V. to Brindle → Healer's Cottage (Vella)
→ Cemetery cache "He was your brother — V." → Uncle's Workshop
(Ulla's husband's untouched workspace with carved birds matching
the kingfisher in the cemetery cache). The hidden-noun discovery
inconsistency (see Concerns) is the only real risk to legibility.

(3) **Pre-Chrysalis spiral lore** — Discoverable and intriguing.
Five locations, with the Old Chapel Ruin altar stone explicitly
naming the others. A player who spots the spiral once will keep
looking for it elsewhere; the design rewards them.

(4) **Forage seeding payoff** — Good. Inn loft herbs foreshadow
2 of 3 Reedy Foreshore forages (lake-mint, marsh-willow); the
Tailor's Cottage closes the loop with "cattail-down lined cloaks"
and "lake-mint green dye." Lake iron at Sluice Pond connects to
Brindle the smith via the noun description ("the smith in town
pays a premium"). A foraging-focused player would have plenty to
chase.

(5) **Empty/generic/wordy rooms** — Almost none. The Net Yard
(4117), Lake Path Bend (4120), and Lakeshore North are the
shortest descriptions; they earn their length by being transitional
breath-rooms between dense set-pieces. Nothing felt overstuffed.
Everything felt curated.

(6) **Cross-room noun inconsistencies** — None found that are real
inconsistencies. The only continuity gap is cattails being
foreshadowed at the Tailor rather than the inn loft, which is a
design preference, not an error.

(7) **Cardinal-direction or up/down feel** — All directions felt
geometrically right. `up` only appears as a real second-floor entry
(Loft 4104, Workshop 4138, Garret 4126) and a temple stair (4123→
4126). `down` appears only as the cave descent (4121→4127) and
matching `up` reverse. The diagonal `southeast`/`northwest` between
4142 and 4111 makes sense as a worn footpath cutting the corner
between gate and travelers' camp.

The one geographic oddity worth a second look: from 4115 Crab-Trap
Beach, `east` leads directly into 4124 Temple Garden. That makes
sense in coastal-temple terms (the beach is right under the temple's
walled garden) but might feel abrupt to a player who hasn't yet
realized the temple is on the lakefront. Not a problem, just worth
remembering when NPCs go in.

## Raw Stats

- Commands sent: ~115
- Fights: 0 (no NPCs spawned in zone yet)
- Deaths: 0
- Spells cast: 1 (chrysalis-glow, to enter Cave Mouth)
- Items used: 0 (one accidental craft attempt failed cleanly)
- Bugs found: 3 (Cave Mouth darkness, Bone Shoals cache desc, 4102 mapsymbol)
- Concerns: 5 (search inconsistency, missing map template, earlier YAML
  panic, exit warnings, "a alchemy bench" grammar)
- Observations: 7
- Passes: 8
