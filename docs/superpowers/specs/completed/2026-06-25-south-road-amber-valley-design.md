# South Road + Amber Valley — Design Spec

**Date:** 2026-06-25
**Status:** Design approved (scope + quest scope + seed-don't-wire + cave + latent-Bloom confirmed)
**Source plan:** `docs/ZONE_EXPANSION.md` Phase 5 (zones 5.1 South Road, 5.2 Amber
Valley). This spec narrows that sketch to the approved leg and pins concrete IDs,
the attach point, and the build standards.

## Goal

Open the novel's southern geography by building the first leg of the Southern
Road: a 15-room connector south from the Ashwick crossroads into a 35-room
destination town, Amber Valley (Davan's home — a warm farming community that
celebrates the Chrysalis Rite). ~50 rooms, two new zones, one branching quest,
seeded lore for a deferred follow-up.

## Approved scope decisions (locked)

- **Leg:** South Road (15rm) + Amber Valley (35rm). NOT the River Road or the
  Confluence (those are a later leg).
- **Quests:** build **The Water Dispute** only. **Defer** the Rite Deacon's
  Concern — but seed its hooks (deacon NPC, grove marker, struggling youth) as
  lore so the later quest has somewhere to attach.
- **Cave dungeon:** include the 4–5 room cave in the valley edges as the leg's
  combat pocket.
- **Bloom link:** keep latent — foreshadow only (the deacon's "dramatic
  Bloomings" line, the grove's near-flat orbital marker). No Bloom mechanics
  here; the real connection lands with the Confluence/Crash-Site arc later.

## IDs & attach (verified free 2026-06-25)

- **Rooms:** 6040–6089 (50). South Road 6040–6054; Amber Valley 6055–6089.
- **Mobs:** 9394+.
- **Items:** 40121+ (foraging produce + quest items).
- **Zone folders (lowercase_underscore):** `south_road`, `amber_valley`. Zone
  display names: `South Road`, `Amber Valley`. (Loader derives the folder from
  `ConvertForFilename(zone)` — folder must match or the server panics at load.)
- **Attach point:** Marches Spur Road **room 4014** (the crossroads, zone
  `Marches Spur Road`, at x=-8 y=-13 z=0) has signposts promising "South reads
  'Amber Valley, the Confluence'" but **no south exit**. Add a reciprocal
  `south:` exit 4014 ↔ 6040 (annotate `zone: South Road` on 4014's side).
- **Coordinates:** the southern branch is a fresh region south of the crossroads.
  South Road runs south = **decreasing y** from (x=-8, y=-13): 6040 at y=-14,
  stepping down. Amber Valley sits at the south end. No collision risk (nothing
  else is built south of the crossroads). All exits reciprocal; ValidateZone
  Consistency must read errors=0 mode=panic.
- **South frontier stub:** Amber Valley's south edge gets a gated stub toward the
  unbuilt River Road (a signed dead-end, like the corridor zones used — e.g. a
  river-road signpost + a "the way south is for another day" blocked exit, or
  simply no exit with prose pointing onward). No exit to a non-existent room.

## Zone 1 — South Road (6040–6054, 15 rooms)

Biome: transitioning farmland → dry valley. Theme: the land warming and drying,
orchards giving way to scrub, the valley opening below.

**Stage A (6040–6049, 10 rooms):** the descent from the crossroads. Road winding
through increasingly dry terrain. A **waypoint inn** at the midpoint (The
[name] — an interior room or two with an innkeeper NPC; `sethome`-eligible if it
fits the pattern, optional). Traveling-merchant NPCs heading north (ambient,
dialogue about the valley and the road). A **shepherd NPC** with local knowledge
(dialogue: the valley, the water tension below, the weather). Views of the valley
opening below.

**Stage B (6050–6054, 5 rooms):** valley approach. Orchards and irrigated farms.
Warm air, the smell of sun-baked earth and ripening fruit. A **farmstead** whose
**dried-up irrigation channel** is visible — the first breadcrumb for the Water
Dispute (an examinable noun + a farmer's offhand complaint).

## Zone 2 — Amber Valley (6055–6089, 35 rooms)

Biome: dry valley, irrigated farmland, warm. Theme: growth — agricultural and
personal. The place where the Chrysalis Rite happens and young people discover
what they are becoming. The valley wears mutation with casual pride.

**Stage A — Town center (6055–6064, 10 rooms):** the market square; the **Rite
pavilion** (where Blooming ceremonies happen — atmospheric, lore-bearing); a
general store (vendor); **The Golden Bough inn** (innkeeper NPC — rumor/quest
breadcrumb hub); the **woodworker's shop** (Davan's father's trade — see
residential); fruit stalls (a flavor micro-vendor or forage source); townsfolk
NPCs who mention their changes with pride (unique mutation each).

**Stage B — Residential & farms (6065–6074, 10 rooms):** **Davan's family home**
(his father still works there); irrigated orchards; a vineyard; the irrigation
river that feeds the valley. NPCs: **Davan's father** (a woodworker — dialogue:
the son who left, the carving talent, quiet worry; a warm, grounded anchor);
**the two feuding farmers** (the Water Dispute parties — each with their side);
the **traveling Rite deacon** (SEEDED, no quest — lore dialogue noting this
year's Bloomings are more dramatic than expected; the deferred-quest hook).

**Stage C — Valley edges (6075–6084, 10 rooms):** foothills, dry scrub, the old
paths up toward the ridge. The **cave system** (4–5 rooms, minor dungeon — the
combat pocket; sun-adapted mutated fauna: lizards, a hawk/raptor, a mutated
valley predator as the depth threat; modest loot/forage). The **collapsed
irrigation section** is reachable here (the Water Dispute "fix the source" path).
The road south toward the River Road begins here (the frontier stub).

**Stage D — The Chrysalis grove (6085–6089, 5 rooms):** a sacred site outside
town where notable Bloomings are commemorated. Old stone markers for particular
mutations. Quiet, reverential, deep lore about how the community understands the
Chrysalis. A **hidden marker that predates the theology — the inner-orbit symbol,
weathered almost flat** (SEEDED examinable lore; connects to the orbital-symbol
mystery threaded through New Plymouth and foreshadows the Bloom-mutation link).

## Quest — The Water Dispute (1 quest, branching resolution)

Two neighboring farms fight over irrigation rights from the valley river.

**Breadcrumbs (≥3, per the Breadcrumb Rule):**
1. Both farmers complain at the market square / in their farm rooms.
2. The Golden Bough innkeeper mentions the tension as rumor.
3. The dried-up irrigation channel is a visible, examinable noun in the farmland
   rooms.

**Resolution paths (≥2 — all three sketched; consequence persists via a quest
flag recording which farmer was favored):**
- **(a) Mediate** — dialogue with both farmers, find a compromise (a dialogue
  chain; the "fair" split).
- **(b) Fix the source** — a collapsed section of the old irrigation channel up
  in the foothills/cave needs clearing (exploration + a light fight); restoring
  flow resolves it without picking a side.
- **(c) Find the record** — the original water agreement is filed in the town
  records (the Rite pavilion / a town-hall room); research resolves it by the
  letter of the old agreement, favoring one farm.

**Flag:** declare `<questid>-outcome` with values `[mediated, restored, <farmA>,
<farmB>]` (final values set during build to the actual farmer names/branch keys);
`set_flag` on the resolving action. Rewards: gold + a valley-appropriate item +
modest rep (a valley/farmfolk faction if one is warranted, else gold+item only —
decide at build; do NOT invent a faction unless the leg needs it).

**Quest/dialogue SOPs (all apply):** grant node FIRST in the giver's `tree.nodes`
(substring-shadow rule); `grantsQuest` node includes the end token in
`questExcluded` + `"quest"`/`"task"` in triggers; giver's root `hints` advertise
the hook (the feel-pass lesson — don't hide the quest behind guessing "help");
interaction hints say "examine the &lt;object&gt;" not "make/take/fix" (the
feel-pass verb lesson); a trigger may only `grant` a declared step token (grant
the `end` step directly at completion); reward-block keys are tag-less; NPC text
first person, hints second person; no hard numbers in player-facing text.

## Quality bar (ZONE_EXPANSION standards — non-negotiable)

- **Rooms:** 80-char wrap; 3 layers (atmospheric hook + grounded detail +
  interaction hint); sensory variety (lead with sound/smell/heat, not always
  sight); no generic filler; biome/weather-aware (warm dry valley).
- **Nouns:** ≥2 examinable nouns/room beyond exits; interactable/container nouns
  highlighted with `<ansi fg="itemname">…</ansi>`; ~20% of non-town rooms have a
  container noun (some with items, most flavor).
- **NPCs:** idle behaviors on tick timers; ≥3 dialogue topics beyond quest
  function; unique mutation description per mutated NPC (no two alike in a zone);
  schedules for town NPCs where the engine supports it (the Amber Valley
  anchors — innkeeper, woodworker, farmers — get light day routines).
- **Foraging:** warm-valley produce (orchard fruit, wine-grapes, dry-scrub herbs)
  as forage items 40121+, wired into the valley rooms like the Fernway/corridor
  foraging.
- **Difficulty:** low-mid. The cave is the only real combat; sun-adapted fauna
  scaled for a capital-corridor-level traveller.

## Architecture / file inventory

- `rooms/south_road/6040–6054.yaml` (+ `south_road/zone-config.yaml`).
- `rooms/amber_valley/6055–6089.yaml` (+ `amber_valley/zone-config.yaml`).
- Edit `rooms/marches_spur_road/4014.yaml` — add the reciprocal `south:` exit.
- `mobs/south_road/` + `mobs/amber_valley/` (9394+): shepherd, merchants,
  innkeeper(s), townsfolk, Davan's father, the two farmers, the deacon, the
  struggling youth, cave fauna. Each with `idlecommands`, `groups`, schedules for
  town anchors.
- `dialogue/south_road/` + `dialogue/amber_valley/` per NPC (giver + lore NPCs).
- `quests/<id>-the_water_dispute.yaml` (next free quest id — verify at build,
  likely 72).
- Items (40121+): forage produce + any quest item.
- Schedules under `schedules/amber_valley/` for the town anchors.
- Foraging wiring (the existing forager system — match the Fernway pattern).
- Update `docs/ZONE_EXPANSION.md` status table (South Road, Amber Valley → Built)
  on completion.

## Testing

- Boot test after each stage: ValidateZoneConsistency errors=0 mode=panic; rooms/
  mobs/quests/items load; no filepath/step/flag panics.
- `cartcheck` clean for both new zones (Cartesian-consistent, reciprocal exits).
- Harness playtest: walk Ashwick→South Road→Amber Valley; confirm the attach,
  the inn, the shepherd, the town; run the Water Dispute end-to-end on at least
  two of its three resolution paths; confirm the giver hook surfaces on `talk`
  and the quest grants on natural words; fight the cave; examine the grove marker
  + deacon lore (seeded, no quest).
- Feel/discoverability spot-check per the recent lesson (hooks advertised,
  `examine` verb for interactables).

## Out of scope (deferred)

- The Rite Deacon's Concern quest (seeded only).
- River Road + the Confluence (the next Southern Road leg).
- Any Bloom mechanic in the valley (foreshadow only).
- The prod push (this is normal feature work; merges to master, ships in a later
  bundle).
