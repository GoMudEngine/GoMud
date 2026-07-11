# River Road to the Confluence (Zone 5.3) — Design

**Date:** 2026-06-26
**Status:** Approved (design phase)
**Roadmap:** `docs/ZONE_EXPANSION.md` Phase 5, Zone 5.3 — Southern Road leg 2
**Predecessor:** Southern Road leg 1 (South Road + Amber Valley), merged `58a8cf0e`

## Purpose

A short river-valley **connector** linking Amber Valley (north) to the
yet-unbuilt Confluence temple-city (south). Its job is twofold:

1. **Geography:** carry the road from Amber Valley's dry valley lip down to
   the river country, conveying *water shaping land* — the river appearing,
   broadening, and tributaries joining as the land grows lush. The sense of
   **convergence**.
2. **Foreshadowing:** seed the Confluence mystery (pre-Founding architecture,
   the inner-orbit symbol reinterpreted as Chrysalis motif, swelling pilgrim
   traffic) *before* the player arrives at the 70-room payoff zone (5.4).

**No quest.** This is a deliberate lore-and-ambient connector; quest budget is
concentrated in the Confluence proper. Economy texture (a fishmonger vendor +
river forageables) gives it living-world weight without a quest.

## Scope & IDs

- **16 rooms:** `6090`–`6105` (global next-free confirmed `6090`).
- **Mobs/dialogue:** `9410`+ (Amber Valley ended at 9409).
- **Items:** `40123`+ (global next-free confirmed `40123`).
- **Quests:** none.
- **Zone display name:** `River Road` → folder `_datafiles/world/dogmud/rooms/river_road/`
  and `mobs/river_road/` (underscore folder per `ConvertForFilename`).
- **Biome:** `water` for riverside rooms (already has forage yields +
  difficulty 135); `land` for the first 2–3 dry descending road rooms before
  the river appears. Transition dry→lush is conveyed in prose.

## Seams

- **North seam (open the road):** Amber Valley **6071** ("Valley's South Lane")
  currently *bars* the south road in-fiction — its `the south road` noun reads
  "WAY WASHED OUT — NO PASSAGE / SOON". Edit 6071 to **open the passage**: add
  `south: {roomid: 6090, zone: River Road}`, and revise the description + the
  `the south road` / `the open country` nouns so the wash-out is **freshly
  mended** (the work-gang finished; the "SOON" became now). This is the only
  edit outside the new zone.
- **South seam (stub):** room **6105** ("The Confluence Gates") faces the
  unbuilt Confluence. Its south is a **barred stub** framed intentionally —
  the gates/ferry ahead, but "the river ward isn't passing travelers yet" /
  "the last span of road is still being cut." A `nouns` entry describes the
  city beyond. No `south` exit until 5.4 is built. (Same intentional-stub
  technique used at the original 6071 wash-out.)

## Layout — two mini-stages, 16 rooms

The spine runs **south** from the seam, then bends **south-east** for the final
approach (the Confluence lies "south and east, where three waters meet"). Two
short spurs hang off the spine: the 3-room fishing village (east) and a
1-room waystone rise (west of the bluff).

### Stage 5.3a — River road, dock & fishing village (11 rooms)

| Room | Title | Biome | Exits | Notes |
|------|-------|-------|-------|-------|
| 6090 | The Mended Road | land | n→6071 (AV), s→6091 | Where the wash-out was; fresh-cut roadbed, the warden's work-camp |
| 6091 | Descending Track | land | n→6090, s→6092 | Dry road dropping off the valley lip |
| 6092 | Where the River Begins | water | n→6091, s→6093 | First sight of water — a thin stream beside the road |
| 6093 | Riverside Road | water | n→6092, s→6094, e→6097 | Junction to the fishing village; the water broadening |
| 6094 | The Barge Landing | water | n→6093, s→6095 | A dock; river traffic visible; the dock-hand |
| 6095 | Broadwater Road | water | n→6094, s→6096 | The river wide and slow now |
| 6096 | The Confluence Bluff | water | n→6095, s→6101, w→6100 | Overlook: the **second river joining** visible below |
| 6097 | Fishers' Landing | water | w→6093, e→6098 | Village edge; drying nets, beached coracles |
| 6098 | Netmender's Row | water | w→6097, e→6099 | Village center; **fishmonger vendor**; netmender NPC |
| 6099 | The Smokehouse | water | w→6098 | Smoke and river-fish; the old fisher |
| 6100 | Old Waystone Rise | land | e→6096 | A pre-Founding **waystone** with the weathered orbital marker |

### Stage 5.3b — Confluence approach (5 rooms)

| Room | Title | Biome | Exits | Notes |
|------|-------|-------|-------|-------|
| 6101 | The River Road South | water | n→6096, s→6102 | Pilgrim traffic begins on the road |
| 6102 | Pilgrims' Way | water | n→6101, se→6103 | Road bends south-east; way-shrines |
| 6103 | The Pilgrim Camp | water | nw→6102, se→6104 | A camp outside the city; **pilgrim NPCs** |
| 6104 | Sight of the Spires | water | nw→6103, se→6105 | The three rivers + the city's spires emerging in haze |
| 6105 | The Confluence Gates | water | nw→6104, (south STUB) | Barred approach to the unbuilt Confluence |

### Proposed coordinate map (z=0 throughout; **build must `cartcheck`-verify**)

```
6090 {-8,-40}  6091 {-8,-41}  6092 {-8,-42}  6093 {-8,-43}
6094 {-8,-44}  6095 {-8,-45}  6096 {-8,-46}
6097 {-7,-43}  6098 {-6,-43}  6099 {-5,-43}   (fishing village spur, east)
6100 {-9,-46}                                 (waystone rise, west of bluff)
6101 {-8,-47}  6102 {-8,-48}  6103 {-7,-49}  6104 {-6,-50}  6105 {-5,-51}
```

All 16 coords are unique and exits are reciprocal as drawn. **GOTCHA (Amber
Valley leg-1 lesson #2):** the build MUST run `cartcheck river_road` / boot
`ValidateZoneConsistency` and resolve any collision against Amber Valley's
southern rooms (6071 sits at `{-8,-39}`; verify no AV room already occupies
`{-8,-40}` etc. before committing the coords).

## NPCs (ambient, no quest) — mobs 9410+

| Mob | Role | Room | Notes |
|-----|------|------|-------|
| 9410 | Road Warden | 6090 | Oversaw the mending; explains the reopening; non_combatant |
| 9411 | Dock-hand / bargeman | 6094 | River traffic, passage to the Confluence/NP (lore, not a transit) |
| 9412 | Netmender | 6098 | Village life; the **fishmonger vendor** merchant |
| 9413 | Old Fisher | 6099 | Carries the **single** three/four-waters breadcrumb (see Lore) |
| 9414 | Pilgrim (one of two) | 6103 | Theology, the temple, the symbol above the door |
| 9415 | Pilgrim (two of two) | 6103/6104 | Foreshadows the Confluence draw |

**River fauna** (combat targets for the leveling path), mobs 9416+:

| Mob | Notes |
|-----|-------|
| 9416 | Grey heron / river-bird — weak, evasive |
| 9417 | River otter — fast, low HP |
| 9418 | Something in the shallows — a mutated eel/gar, the zone's tougher fauna |

Fauna stats follow the existing river/valley fauna power band (compare Amber
Valley cave fauna 9407–9409 and the corridor fauna). `archetype: fighting`
where appropriate.

## Economy texture

### Fishmonger vendor (mob 9412, room 6098)
A small provisioner stocking river goods. **GOTCHA (economy-depth lesson):**
salable items REQUIRE a `vendor_categories` value drawn from `ValidCraftSupports`
**minus `general`** — `general` is invalid on items. River goods therefore carry
a real discipline (e.g. `alchemy` for edible/reagent provisions), and the vendor
stocks them via an explicit `shop:` list as the catch-all, exactly as Mardle the
Sundries-Seller does (9390). Confirm the chosen category against
`ValidCraftSupports` at build.

### Items, 40123+
| ID | Item | Use |
|----|------|-----|
| 40123 | Watercress | River forageable (`water` biome) + fishmonger stock |
| 40124 | River reed / freshwater mussels | River forageable (`water` biome) |
| 40125 | Smoked river-fish | Fishmonger trade good (provision) |
| 40126 | Fresh river catch | Fishmonger trade good (provision) |

### Forage wiring (Go code, not data)
Append the new forageable IDs to `ForageYields["water"]` in
`internal/forager/forage_core.go` (e.g. add `40123, 40124`). Optionally add a
`NightForageYields["water"]` entry if a night-only river forageable is wanted
(not required). Update/extend `internal/forager` tests if they assert the water
yield set. This mirrors the Amber Valley followup (40121/40122 added to `land`/
`farmland`).

## Lore breadcrumbs (the payload)

Keep these **subtle** — the connector seeds, it does not explain.

1. **The fourth water (ONE light touch only).** From the Confluence Bluff
   (6096) a traveler counts the rivers; the **Old Fisher (9413)** offhandedly
   names a *fourth* channel that "went dry before my grandfather's time." Plant
   this in exactly **one** place (the fisher's dialogue, optionally echoed once
   in the bluff room's noun) — do NOT repeat it across markers/NPCs. It mirrors
   the Margin Notation quest's three/four question and the cooperage group's
   "what changed," without tipping into numerology.
2. **The pre-Founding waystone (6100).** An old boundary stone carries the
   weathered **inner-orbit symbol**, which pilgrims now read as a Chrysalis
   motif — the same reinterpretation the Confluence's architecture shows. A
   `look symbol` / `look waystone` noun describes it; no interaction trigger
   (no quest). Ties to the gallery-cipher / orbital-symbol web (Dross / Ept /
   Orin / Gritta / the Buried Lintel).
3. **Pilgrim talk (6103–6104).** The two pilgrims discuss the temple, the
   symbol above its door, and the pull of the Confluence — pure foreshadowing.

## Optional polish (nice-to-have, not blockers)

- **Schedules:** 1–2 simple day-post / night-sleep schedules for the
  fisherfolk (netmender works the row by day, sleeps at the smokehouse), in
  `schedules/river_road/`. Keep them **findability-preserving** (NPCs stay in
  their advertised rooms during play hours), per the Amber Valley followup
  pattern. Skip if it risks the smoke test.
- **Idle/ambient:** river birds, water over stones, wet-earth idle messages per
  the roadmap's ambient note.

## Build approach

Standard content pipeline, on branch `feature/southern-road-river-road`:

1. **Rooms** (`/new-room` or direct YAML) 6090–6105 with the coordmap above;
   edit Amber Valley 6071 to open the seam.
2. **Mobs** (`/new-mob`) 9410–9418 + dialogue for the speaking NPCs.
3. **Items** 40123–40126; wire forage yields in `forage_core.go`; fishmonger
   `shop:` list.
4. **Smoke test:** wipe instance saves (`rooms.instances/*`, `mobs.instances/*`
   — NOT `shops/`), reboot, confirm clean load (no panics, `mobs.LoadDataFiles`
   / `rooms` counts), `cartcheck river_road` clean, walk the seam 6071→6090→…→6105.
5. **Optional** harness playtest (feel/connection pass).
6. **Merge** `--no-ff` to master. Update `docs/ZONE_EXPANSION.md` (mark 5.3
   built) and `PATCH_NOTES.md` at push time.

World total after this leg: **38 zones → 39**, **1006 rooms → 1022**.

## Out of scope (deferred to 5.4 The Confluence)

- The barge **transit** (actual passage north to NP) — the dock here is
  ambient/lore only; the working barge master lives in the Confluence (5.4a).
- The Margin Notation and Undercroft quests, the temple, the sealed chamber.
- Any Bloom/Rite mechanic hooks.
