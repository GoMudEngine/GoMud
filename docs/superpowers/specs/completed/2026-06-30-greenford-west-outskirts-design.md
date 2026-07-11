# Greenford — District 5: West Outskirts — Design

*Spec date: 2026-06-30. District 5 of 5 — **the LAST Greenford district**
(city-wide layer `2026-06-30-greenford-citywide-design.md`; D1 `6b162857`, D2
`b8b14bcb`, D3 `a3c4fd7c`, D4 in flight). A small, quiet **lore-and-ambient
terminus** — the road leaving Greenford toward New Plymouth: the western edge of
town, a stable, a farewell waypoint shrine, and the **West Road / NP-loop stub**
(Aldric's route, loop-closure deferred). No quest (Q75 completes in D4); no
symbol content (the symbol is the university's). When this merges, **Greenford is
complete (5/5)** and the Eastern Arc has its full approach city. Builds AFTER D4
merges.*

## Role

The mundane outward edge — where Greenford ends and the road to the wider world
begins. After the river (D1), the town (D2), the university + Q75 (D3), and the
neighborhood + Reth's directions (D4), this is the quiet exhale: a traveler
fitting out for a long journey, a stable, a shrine for the road, and the West
Road bending away toward New Plymouth (a described stub — that connecting road
isn't built; NP is reached canonically NW/west, days away). The closing beat for
the whole city. Echoes the East Road / River Road terminus pattern.

- **Folder:** `greenford`. **Rooms:** 6317–6322 (6). **Mobs/dialogue:**
  9530–9534 (5). **Items:** 40162+ (optional travel good).
- **No quest, no faction, no symbol/mystery content.** All NPCs `[humanoid]`.

## Geography & Seam

- **Seam:** the town center's **WEST edge** (z=1). West is the one direction not
  already claimed (the river/bridge is north/down via D1; the university is
  south/up via D3; the East Road is back across the river). Add a **west exit**
  from a clean town-center room → 6317. **Suggested attach: 6295 "The Town Hall
  Steps"** `{x:22,y:-78,z:1}` (west free; a civic building's steps onto a street
  leaving town reads well) **OR** 6293 "The General Store" `{21,-77,1}` (west
  free) — the BUILD picks the cleanest free west exit on the town's west side
  (read the D2 rooms; collision-check). Revise that room's prose lightly so a
  street/lane now leads west out of town.
- **Layout:** the West Road runs west (−x) from the attach, on **z=1**, into the
  outskirts (x ≤ 20, clear of D2 [x 21–24] and D1 [z=0]). Builder finalizes a
  clean reciprocal, collision-free graph + cartcheck.

| Room | Title | role |
|------|-------|------|
| 6317 | The West Gate | the town's west edge (from the attach); a road-warden |
| 6318 | The West Road | leaving town, the country opening; a departing traveler |
| 6319 | The Coaching Stable | the ostler's yard (travel; optional vendor) |
| 6320 | The Wayfarer's Shrine | a farewell waypoint shrine (a shrine-keeper; the road-blessing) — MUNDANE (a traveler's shrine, NOT an orbital marker) |
| 6321 | The Milepost | the open road, a milestone naming New Plymouth (days west/NW) |
| 6322 | The Plymouth Road | **terminus stub** — the road bends on toward NP, NOT passable (the loop isn't built); a described "days west and north, another journey" gate, like the East Road's barred bridge. The closing beat for Greenford. |

(6317 the edge → 6318 the road → 6319/6320 off it → 6321 → 6322 the stub. No
onward exit at 6322; described only.)

## NPCs (mobs 9530–9534: 5)

Canonical Title-Case names, `ConvertForFilename` filenames, ambient
`noncombat_passive`, unique mutations (cross-check vs the WHOLE Greenford roster —
D1+D2+D3+D4 = 27 mutations so far), ≥3 topics, idle behaviors, voice rules
(**every hint word routes**; **`|` literal blocks for ALL long NPC text**), NO
quest fields, all `groups: [humanoid]`.

| mob | room | role |
|-----|------|------|
| 9530 (named) The Road-Warden | 6317 | the town's west edge; directs travelers; the road, the wider world, the long way to NP — a SOFT outward gesture (NO mystery) |
| 9531 (named) The Ostler | 6319 | the coaching stable; travel/coach talk; **optional `cooking`/general vendor** (trail food / a traveler's good) |
| 9532 A Departing Traveler | 6318 | the outward gesture — bound west/NP or beyond; what's out there, the long road (lore-light; NO crash-site/symbol) |
| 9533 (named) The Shrine-Keeper | 6320 | the farewell shrine; the traveler's road-blessing; gentle, mundane faith (NO orbital symbol) |
| 9534 A Resident | 6317 | ambient; the edge-of-town daily life |

The Road-Warden + Departing Traveler are the soft "the world goes on west"
gesture (Aldric's route to NP, the loop the world will close later). Keep
everything mundane — D5 is the quiet close.

## Items

- Optional **40162** — a travel good at the stable (trail rations / feed),
  `vendor_categories: [cooking]` (never `general`) if the ostler vendors; else
  reuse an existing good. D5 may need no new items.

## Mystery / lore boundary

**NONE.** No symbol, no crash-site, no pre-Founding marker. The Wayfarer's Shrine
is a MUNDANE traveler's shrine (a road-blessing post), explicitly NOT an orbital
marker — keep it clean (Greenford's symbol beat is the university's, D3). The
outward gesture toward NP is geography/lore-light only.

## Terminus stub (6322)

The West Road bends on toward New Plymouth — described in prose (the milestone,
the country rolling away west/NW, NP "days off, another journey"), but **NOT
wired** (the loop-closing road to NP isn't built). Model on the East Road's
barred Greenford Bridge / River Road's Confluence Gates — a clean "another
journey, another day" terminus, not a broken bump. This is the **closing beat for
all of Greenford**: the player has walked the whole city, earned Reth's
directions east, and here the road gestures west toward the wider world.

## Build conventions & validation

Full Greenford convention/gotcha list (folder `greenford`; Title-Case; colon/
`>`-block; no `kind:`; vendor categories never `general`; node-shadowing +
hint-routing cross-check; `|` blocks for long text; roster-wide mutation check;
stage explicit git pathspecs never `-A`). Per-district SOP: `id_inventory` →
author → wipe instances → clean boot (`ValidateZoneConsistency errors=0
mode=panic`) → `cartcheck greenford` → **world-critic + harness feel-test** (the
west-edge seam from the town center, the road/stable/shrine, the vendor if any,
the NP-road terminus stub, all NPC hints route, NO symbol leak) → update
`docs/ZONE_EXPANSION.md` row 19 → **mark Greenford ✅ COMPLETE (5/5)** + memory →
merge `--no-ff`.

**On merge: GREENFORD IS DONE** — 45 rooms, 5 districts, Q75 end-to-end, the
Eastern Arc's full approach city. Next Eastern Arc work = Cascade Pass Road
(#20) → Eastern Highlands (#21) → Crash Site (#22), the moon-crash payoff.

## Out of scope

The actual NP-loop road (stub only); Cascade Pass / Eastern Highlands / Crash
Site (#20-22); any crash-site/symbol content.
