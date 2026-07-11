# Greenford — District 1: River District & Bridge Landing — Design

*Spec date: 2026-06-30. The first of Greenford's 5 districts (city-wide layer:
`docs/superpowers/specs/2026-06-30-greenford-citywide-design.md`). This district
is the zone's ENTRANCE — it opens the East Road's barred Greenford Bridge and
establishes the riverfolk texture at the foot of the hill. No quest, no symbol
beat (the mystery lives in the University + Neighborhood districts).*

## Role

Where the East Road bridge lands on the **east bank** of the Greenford river: a
working riverfront — bridge-gate, landing/quay, watermill, fishing reach — with
a lane that begins **climbing toward the Town Center** (District 2, a described
stub). Riverfolk are warm, practical, unbothered; the contrast that makes the
scholarly uphill town feel like a different world. The bridge-warden is the
narrative key: the East Road's barred gate (6277, "NO PASSAGE WITHOUT THE
WARDEN'S LEAVE") becomes passable because the warden is here and grants leave.

- **Folder:** `_datafiles/world/dogmud/rooms/greenford/` (+ mobs/dialogue/
  schedules under `greenford`). Zone display name **`Greenford`**, region
  **The Tri-Rivers**. **Folder MUST = `ConvertForFilename("Greenford")` =
  `greenford`** (the East Road folder-name boot panic — verify).
- **Rooms:** 6278–6287 (10). **Mobs/dialogue:** 9501–9508. **Items:** 40152+.
- **No quest, no new faction** (riverfolk are `[humanoid]`; Margin appears later).

## Geography & Seam

- **Seam:** the East Road's **6277 "The Greenford Bridge"** `{x:22,y:-70,z:0}`
  (currently exit `north→6276` only; its prose has a barred toll-gate "across the
  river"). Add **`6277 south → 6278`** (the bridge span — a 2-cell jump auto-
  classifies `long` over open water, which is correct) and **lightly revise
  6277's prose** so the gate now reads OPEN/passable (the warden admits
  travelers; Greenford is reached). The far bridgehead 6278 is on the **east
  bank**; the district climbs from the river up toward the town.
- **Suggested coordinate spine** (builder finalizes + re-runs `cartcheck`/boot
  consistency — coords are a recurring boot-panic; all must be collision-free
  against the East Road frame, which occupies x≤22, y≈-66…-70):

| Room | Title | biome | suggested coord | role |
|------|-------|-------|-----------------|------|
| 6278 | The Greenford Gate | city | {22,-72,0} | east bridgehead + the **Bridge-Warden** (opens the seam) |
| 6279 | The River Landing | water | {22,-73,0} | quay/dock, barge trade, dock-hand |
| 6280 | The Watermill | water | {21,-73,0} | the mill + **Miller** (vendor) |
| 6281 | The Fishing Steps | water | {22,-74,0} | riverbank fishing, fisherfolk + fauna |
| 6282 | Riverside Row | city | {23,-73,0} | a short market row + **Fishmonger** (vendor) |
| 6283 | The Boathouse | water | {23,-74,0} | river-trade texture (container nouns, nets, a moored skiff) |
| 6284 | The Climbing Lane | city | {22,-75,0} | the lane starts up toward town (the climb) |
| 6285 | The Riverside Green | city | {21,-75,0} | a small green/shrine, ambient townfolk |
| 6286 | The Upper Bank | city | {22,-76,0} | upstream bank, last riverfolk beat |
| 6287 | The Town Stair | city | {22,-77,0} | **stub** up to the Town Center (District 2) — described, NOT wired |

(Reciprocal exits along the climb; water-biome rooms enable the existing River
Road forageables with no new Go code — see Economy. The builder may use a z-step
for "the climb" if it reads better than planar north, keeping coords clean.)

## NPCs (mobs 9501–9508: ~6 riverfolk + 2 fauna)

All canonical Title-Case names, filenames `ConvertForFilename`, ambient archetype
`noncombat_passive`, unique visible mutations, ≥3 dialogue topics, idle behaviors,
voice rules (NPC text 1st-person; hints 2nd-person; discoverable triggers). No
quest fields. Riverfolk are `groups: [humanoid]` (no Margin here).

| mob | room | role |
|-----|------|------|
| 9501 The Bridge-Warden (named) | 6278 | gate-keeper; grants the "warden's leave"; town/travel/river-trade talk; the welcoming face of Greenford (points uphill to the town/university) |
| 9502 The Miller (named) | 6280 | watermill; **cooking/general vendor** (flour, a loaf, river goods); the dry-year/trade talk |
| 9503 A Fisherman / fisherfolk | 6281 | the river, the catch, the bridge; warm riverfolk voice |
| 9504 A Barge-Hand | 6279 | river trade up/down to the Confluence; cargo, the current |
| 9505 The Fishmonger (named) | 6282 | **cooking vendor** (river fish); market patter |
| 9506 A Riverside Ambient (a washerwoman / child) | 6285 | daily-life color |
| 9507 A Grey Heron | 6281 | river fauna (model on River Road 9416) |
| 9508 A River Otter | 6283 | river fauna (model on River Road 9417) |

The Bridge-Warden (9501) is the one slightly-loaded NPC: friendly, gives the
player their first sense of Greenford as a scholarly town uphill (a soft forward-
gesture toward the University/the Surveyor's Report, NEVER any mystery content).

## Economy & Forageables

- **Vendor goods (40152–40154):** river/mill foods — e.g. a river trout, a sack
  of milled flour, a river loaf — each `vendor_categories: [cooking]` (never
  `general`). Reuse River Road fish goods (40125/40126) where they fit to avoid
  new items.
- **Forageables:** the water-biome rooms (6279/6280/6281/6283) auto-enable the
  EXISTING River Road water forageables (40123 watercress, 40124 mussels — already
  in `water` ForageYields). **No new forage Go code needed** for District 1.

## Mystery / lore

**None in District 1.** No waystone/symbol beat — the riverfront is deliberately
mundane (the scholarly mystery is uphill). The Bridge-Warden may mention the
university/the scholars exist (forward-gesture) but explains nothing.

## Terminus stub

**6287 "The Town Stair"** — the lane reaches a stair/gate up into the Town Center
(District 2). Described in prose (the market, the bookshop, the university tower
higher up), but the onward exit is **NOT wired** (District 2 attaches here later).
Model on the East Road / River Road barred-terminus pattern, but softer (an open
town, just "not built yet" — a stair you'll climb soon, not a barred gate).

## Build conventions & validation

Carry the full Greenford city-wide convention/gotcha list (folder name; Title-
Case; colon/`>`-block; no `kind:`; vendor categories; node-shadowing — gated/
specific nodes first; stage explicit git pathspecs never `-A`). Per-district SOP:
`id_inventory` → author → wipe instances → clean boot (`ValidateZoneConsistency
errors=0 mode=panic`) → `cartcheck greenford` clean → **world-critic + harness
feel-test** (walk the seam from East Road 6277, the bridge opening, all NPCs +
both vendors, forage, the Town Stair stub) → update `docs/ZONE_EXPANSION.md` +
memory → merge `--no-ff`.

## Out of scope

Quest content (spine starts District 2/3), Margin NPCs (District 3/4), the symbol
beat (District 3+), the Town Center itself (District 2).
