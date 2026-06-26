# The Confluence — District 2: The Long Quay — Design

**Date:** 2026-06-26
**Status:** Approved (design phase)
**Umbrella:** `docs/superpowers/specs/2026-06-26-confluence-citywide-design.md` (district 2 of 10)
**Predecessor:** District 1 The Landings, merged `16b6eb7c` — room **6121 "South Quay Gate"** is the stub this district opens.

## Purpose

The Confluence's **commercial waterfront** — the prosperous three-rivers trade
belt, distinct from the Landings' gritty working dock. A long river **market**,
**guild halls + counting houses**, bigger trade wharves, the Quayfolk at their
wealthier tier. It bridges the working dock (Landings, north) to the civic center
(Tri-Cross Square, south — a barred stub for district 3).

**No quest** (per the umbrella — Q73 grants in the Scholars' Quarter, Q74 in the
temple). This district extends the Quayfolk, plants **the first whisper of The
Margin** (the scholar faction, seeding the Q73 thread), and continues the
orbital-symbol environmental storytelling with one mystery seed.

## Scope & IDs

- **16 rooms, 6122–6137** (the umbrella's reserved block for district 2).
- **Mobs/dialogue 9427–9433** (7).
- **Items 40130–40132** (3 river-import/spice goods).
- **Quests:** none.
- **Zone:** `The Confluence` (existing folder `the_confluence/`; no new zone-config).
- **Biome:** `water` throughout.
- **New faction:** `margin` (The Margin — the Confluence scholars). Debuts here
  with **one** member (the Scrivener 9431); grows in the Scholars' Quarter
  (district ~8). Create `factions/margin.yaml`; the member joins via a `groups:`
  entry `margin`. (Mirror `quayfolk` from the Landings.)

## Seam — open the Landings stub (6121)

Edit `_datafiles/world/dogmud/rooms/the_confluence/6121.yaml`:
- Add `south: {roomid: 6122}` to `exits:` (keep `north: 6120`).
- Lightly revise the `description` / `the way south` noun so the road now **runs
  on** into the Long Quay (today's walk no longer "ends at the Landings"). Keep
  the voice; the noun already describes the Long Quay accurately — just flip the
  "a walk for another day" framing to "the road runs on south."

## Layout — 16 rooms

A N–S **quay-road spine** (x −5); riverside **trade wharves** east (x −4/−3); the
**market + guild row** west (x −6/−7). Proposed coordinate grid below — **the
build assigns final coords and `cartcheck`-verifies** against the Landings and all
prior zones (umbrella §7). Exits reciprocal as drawn.

| Room | Title | Coord | Mob | Role |
|------|-------|-------|-----|------|
| 6122 | South Quay Road | {-5,-59,0} | — | Seam from Landings 6121 (north); the trade belt begins |
| 6123 | The Market Cross | {-5,-60,0} | 9433 | Central junction; the market bell; a porter/crier (ambient) |
| 6124 | The Long Quay | {-4,-60,0} | 9427 | The great trade wharf (riverside); the **River-Trade Factor** |
| 6125 | The Import Wharf | {-3,-60,0} | — | Cargo/import wharf, cranes, bonded goods (riverside) |
| 6126 | Quay Road South | {-5,-61,0} | — | Spine continues |
| 6128 | The Spice Quay | {-4,-61,0} | 9428 | River-import market (riverside); the **Importer** vendor |
| 6127 | The River Market | {-6,-60,0} | 9429 | Market stalls (west); the **Provisioner** vendor |
| 6130 | The Market Lane | {-6,-61,0} | — | Market row continues |
| 6133 | The Rivermen's Guildhall | {-7,-60,0} | 9432 | The river-trade guild; the **Guild Steward** |
| 6134 | The Counting House | {-7,-61,0} | — | A trade counting-house (ambient ledgers; quiet) |
| 6129 | The Weighhouse Yard | {-5,-62,0} | — | The customs yard |
| 6132 | The Weighhouse | {-4,-62,0} | 9430 | Customs scales; the **Weighmaster** |
| 6131 | The Scriveners' Stall | {-6,-62,0} | 9431 | **THE MARGIN WHISPER** — the Scrivener (old charts; the four-waters thread); `margin` faction |
| 6135 | The Old Customs House | {-7,-62,0} | — | **THE MYSTERY SEED** — pre-Founding foundation-stone with the orbital mark |
| 6136 | The South Market | {-5,-63,0} | — | Toward the south gate |
| 6137 | The Quay's South Gate | {-5,-64,0} | — | **Stub** toward Tri-Cross Square (NO south exit yet) |

Exit skeleton (build finalizes + cartchecks):
```
6122 n->6121(Landings) s->6123
6123 n->6122 s->6126 e->6124 w->6127
6124 w->6123 e->6125 s->6128
6125 w->6124
6126 n->6123 s->6129 e->6128 w->6130
6128 w->6126 n->6124
6127 e->6123 s->6130 w->6133
6133 e->6127
6130 n->6127 s->6131 e->6126 w->6134
6134 e->6130
6129 n->6126 s->6136 e->6132 w->6131
6132 w->6129
6131 n->6130 e->6129 w->6135
6135 e->6131
6136 n->6129 s->6137
6137 n->6136   (south = stub to Tri-Cross Square, NO exit)
```

## NPCs (9427–9433)

All `non_combatant: true`, `hostile: false`, `charm_immune: true`,
`speciesid: 1`, `level: 1`, `maxwander: 0`, `statpool ~30`. Non-vendors:
`behavior_archetype: noncombat_passive`. **Unique names** (check roster + novel
cast). `groups: [humanoid, quayfolk]` EXCEPT the Scrivener.

| Mob | Role | Room | Notes |
|-----|------|------|-------|
| 9427 | River-Trade Factor | 6124 | The district's prosperous Quayfolk face; the three-rivers trade, the guild, the wealth of the quay |
| 9428 | The Importer | 6128 | **Vendor** (`craft_support: cooking`); river-import/spice goods (40130–40132) + reuse cooking goods |
| 9429 | The Provisioner | 6127 | **Vendor** (`craft_support: general`, ship's-chandler style); reuse NP trade goods + a new trade item |
| 9430 | The Weighmaster | 6132 | Customs/weighhouse; the pragmatic regulatory voice of the Quayfolk |
| 9431 | The Scrivener | 6131 | **THE MARGIN WHISPER** — `groups: [humanoid, margin]`. Keeps old harbor charts; notes offhand that they **disagree on the count of waters** (some show a fourth channel), and that "the scholars up in their quarter" have argued it for years. Points curious players toward the Scholars' Quarter. **Subtle, no quest.** |
| 9432 | The Guild Steward | 6133 | The Rivermen's Guildhall official; guild matters, river-trade prosperity |
| 9433 | A Quay Porter | 6123 | Ambient (idlecommands; minimal dialogue) |

## Economy — items 40130–40132

- **The Importer (9428):** a `cooking` vendor stocking exotic river-import food.
  New items + reuse: stock 40130, 40131, plus River Road/Landings goods (40123
  watercress, 40125 smoked river-fish) for breadth.
- **The Provisioner (9429):** a `general` vendor (ship's-chandler style — accepts
  any category); stock the new trade good 40132 + reuse NP goods (40102 rope,
  40103 waterskin, 40105 tinderbox).
- **Items (model on `items/materials-40000/40125-smoked_river_fish.yaml`):**
  | ID | Item | Vendor cat |
  |----|------|-----------|
  | 40130 | River-spice / preserved southern fruit (an Import good) | `cooking` |
  | 40131 | A second import delicacy (e.g. salt-cured eel-roe) | `cooking` |
  | 40132 | A trade good (e.g. a bolt of dyed river-cloth) | `tailoring` |
  All `is_component: true`, a `component_tag`, never `general` on the item.

## Lore touches (keep light)

- **The Margin whisper (Scrivener 9431):** ONE NPC, understated. The four-waters
  discrepancy lives in their dialogue as a scholarly aside (the old charts
  disagree; some count four; the scholars argue it). This **seeds Q73** and
  points to the Scholars' Quarter — without a quest token. It echoes Sedge's
  River Road line but from the scholarly side (charts, not a fisherman's memory).
- **The mystery seed (6135 Old Customs House):** one weathered orbital mark on a
  pre-Founding foundation-stone built into the customs-house wall — `look stone` /
  `look mark` noun, incidental, no trigger. Continues the Old Mole motif; the
  city's older stone keeps showing the same sign.
- **Tri-Cities flavor:** the Factor/Steward may reference the three rivers' trade
  (Aldren grain, Brenn wool, Solt ore — consistent with the corrected canon:
  Brenn east, Solt southwest). Never a fourth (that's the Scrivener's thread only).

## Build approach

Standard content pipeline, branch `feature/confluence-long-quay`:
1. Create `factions/margin.yaml`; open the 6121 seam.
2. Rooms 6122–6137 (final coords + exits, cartchecked). **GOTCHA: quote any
   `idlemessages` containing a colon-space** (the Landings 6106 YAML-panic lesson).
3. Items 40130–40132.
4. Mobs 9427–9433 + dialogue; vendor `shop:` lists; the Scrivener's `margin` group.
5. Smoke test: wipe instances, boot clean (no panics, `ValidateZoneConsistency
   errors=0 mode=panic`, `margin` faction loads), `cartcheck the_confluence`
   clean, walk 6121→6122→…, buy from both vendors, hear the Scrivener's four-waters
   aside, `look mark` at 6135, confirm 6137 south is barred.
6. Merge `--no-ff`; update `ZONE_EXPANSION.md` (Confluence 2/10).

World after this district: **40 zones / 1038 → 1054 rooms** (+16).

## Out of scope

- Districts 3–10 (south stub left unbuilt).
- The Margin's full debut + Q73 (Scholars' Quarter).
- Any mystery exposition beyond the Scrivener's aside + the Old Customs House seed.
