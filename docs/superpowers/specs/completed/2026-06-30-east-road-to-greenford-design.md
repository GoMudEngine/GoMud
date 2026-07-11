# East Road to Greenford — Zone Design (zone 18)

*Spec date: 2026-06-30. Zone-expansion priority #18. The first leg of the
Eastern Arc — the endgame approach toward the crash site.*

## Overview

A **lore-and-ambient connector** (no quest, no faction) linking the Confluence
to the as-yet-unbuilt **Greenford** (zone 19). 15 rooms in 2 mini-stages,
matching the **River Road** pattern (the directly-preceding connector build):
drying terrain, ambient NPCs, one biome vendor, biome forageables, a single
pre-Founding symbol-web breadcrumb, and a described/barred terminus stub
pointing at the next zone.

The road leaves the river country, climbs onto **dry wheat plateau** (the
signature — "the river falling behind"), passes a **waypoint village**, then
descends to the **Greenford river** and a **bridge** where the university
town's profile first appears across the water.

- **Folder:** `_datafiles/world/dogmud/rooms/east_road/` (+ `mobs/east_road/`,
  `dialogue/east_road/`). Zone display name **"East Road to Greenford"**,
  region **"The Tri-Rivers"**.
- **Rooms:** 6263–6277 (15). **Mobs/dialogue:** 9492–9500 (9). **Items:** 40147+.
- **No quest. No new faction. No new buffs.**

## Geography & Coordinate Frame

Attaches at Confluence **6250 "The Greenford Road"** `{x:10, y:-67, z:0}` by
opening that room's **east** exit → 6263. The Confluence's coordinate footprint
stops at x:10 (verified), so the East Road runs east into clear space
(x ≥ 11). Biome arc: river-valley/farmland near the city → dry `land`/scrub
plateau (the bulk) → `farmland`/river at the Greenford approach. Default zone
biome: **`farmland`**.

**Suggested coordinate spine** (collision-checked against the Confluence; the
builder MUST re-run `cartcheck`/boot `ValidateZoneConsistency` after placement —
coord collisions are a recurring boot-panic):

| Room | Coord | Role |
|------|-------|------|
| 6263 | {11,-67,0} | The road climbing out of the river country (river-smell fading west) |
| 6264 | {12,-67,0} | Wheat country begins — rolling dry fields, big sky |
| 6265 | {13,-67,0} | A drovers' stretch (stock pens, a watering trough) |
| 6266 | {14,-67,0} | **The orbital waystone** (symbol-web seed) |
| 6267 | {15,-67,0} | The waypoint-village approach |
| 6268 | {16,-67,0} | **Village center** — the inn/victualler (vendor hub) |
| 6269 | {16,-66,0} | Village edge (north side room: a well-yard / smithy-shed) |
| 6270 | {17,-67,0} | East out of the village, deeper plateau |
| 6271 | {18,-67,0} | The high, lonely milepost (the country at its driest) |
| 6272 | {19,-67,0} | The plateau beginning to fall toward the river |
| 6273 | {20,-67,0} | The descent — **Greenford's profile first visible** (the university on its hill, far off) |
| 6274 | {21,-67,0} | The river valley, suddenly greener |
| 6275 | {21,-68,0} | The riverside road (a watermill; bend south to the water) |
| 6276 | {21,-69,0} | The bridge approach |
| 6277 | {21,-70,0} | **The Greenford Bridge** — terminus stub |

Reciprocal exits along the spine (east/west between consecutive rooms; the
village side-room 6269 hangs north off 6268; the river bend goes south from
6274). All z:0. The `6250 east → 6263` seam is the only edit to existing
Confluence content.

## Mini-Stage A — The Wheat Plateau (6263–6272, 10 rooms)

The bulk of the journey: leaving the river country for dry, open wheat
plateau. Sensory signature = dryness, wind, the smell of cut grain and warm
dust, the river-smell of the Confluence fading behind. Three-layer room
descriptions, ≥2 examinable nouns each, ~20% with container nouns (a hollow
fencepost, a drover's abandoned sack, loose cairn stones).

- **6266 The orbital waystone** — a weathered **pre-Founding waystone**
  standing incongruously at the field's edge among the wheat, far older than
  the farm walls around it. `look waystone` reveals **nested rings / the
  orbital symbol** cut into the stone, softened by weather, with no
  inscription anyone can read. **Unexplained, no NPC tie** — the eastern echo
  of River Road's Old Waystone Rise. Keep it understated; **no numerology
  lecture**, no "fourth" talk. The noun token is the ansi-highlighted
  hyphen-free single word `waystone` (single-word nouns need no hyphenation).
- **6268 The village center** — a small wheat-country hamlet (the road's only
  rest-stop before Greenford). Holds the **victualler/innkeeper** (cooking
  vendor). One side-room (6269) for a second ambient NPC + a container noun.

## Mini-Stage B — The Greenford Approach (6273–6277, 5 rooms)

The plateau falls to the **Greenford river** (a different water from the
Confluence's three). Greenford's profile — the university up on its hill —
first appears at 6273 and grows as the player descends. Greener, wetter,
the road bending south to the bridge.

- **6277 The Greenford Bridge** — the **described/barred terminus stub**.
  Greenford is visible **across the river** (the university, the town roofs),
  the stone bridge runs out over the water, but the way on is **not yet
  passable** (a closed toll-gate at the far end / the bridge described as
  under the town's control, "no passage without the warden's leave" —
  whatever reads as a clean narrative gate, NOT a broken exit). This is the
  seam to unbuilt zone 19, modeled exactly on River Road's barred Confluence
  Gates (6105). The onward direction (east/south toward Greenford) is
  described in prose but carries no wired exit.

## NPC Roster (mobs 9492–9500: 6 ambient + 3 fauna)

All names canonical Title Case; filenames `ConvertForFilename`. Ambient NPCs
use archetype `noncombat_passive`. Each has ≥3 dialogue topics beyond any
function, idle behaviors, and a **unique visible mutation** in its appearance.
Voice rules: NPC `text` first person; `hints` second person (no 3rd-person
self-reference); every trigger discoverable.

| Mob | Room | Role |
|-----|------|------|
| 9492 The Victualler (builder gives a proper name) | 6268 | Innkeeper/victualler — **cooking vendor** (bread, hard cheese, dried fruit). The rest-stop anchor. |
| 9493 A Drover | 6265 | Moving stock east; road/weather/Greenford-market talk. |
| 9494 A Wheat Farmer | 6264 | Harvest, the dry year, the waystone ("older than the walls; nobody minds it"). |
| 9495 A Greenford-Bound Traveler (a scholar) | 6267 | Gentle forward-gesture to the university town — what Greenford is, why they study there. NO crash-site/Reth content (that's the Greenford build). |
| 9496 A Carter | 6271 | A wagoner resting at the lonely milepost; distances, the long haul. |
| 9497 A Village Ambient (a well-woman / child) | 6269 | Hamlet daily life, local color. |
| 9498 A Wheeling Hawk | 6264/6272 | Dry-plateau fauna (ambient, non-combat or low). |
| 9499 A Jackrabbit | 6270 | Dry-plateau fauna. |
| 9500 A Basking Snake | 6271 | Dry-plateau fauna. |

Fauna combat: low-stakes ambient, scaled to a traveling player (this is not a
dungeon). Optional `maxwander` for the hawk/hare so they feel alive.

## Economy & Forageables

- **Vendor goods (40147–40149):** 2–3 wheat-country food items for the
  victualler's cooking shop (e.g. a barley loaf, a wheel of hard cheese,
  dried plums). Each carries a real discipline category (e.g. `cooking`) — an
  item can never be `general`; salable items REQUIRE a vendor category.
- **Forageables (40150–40151):** 1–2 dry-country/wheat forageables wired into
  `forager.ForageYields` (Go code, not data) for the `farmland`/`land` biome —
  e.g. **wild plums** and **grain gleanings** (or a dry-country herb). Add via
  TDD, mirroring the River Road watercress/mussels change
  (`internal/forager/forage_core.go` + a test). `not_salable` only if a forage
  item is genuinely non-vendor; otherwise give it a category.

## Build Conventions & Gotchas (carry from prior district builds)

1. **Every zone folder needs `zone-config.yaml`** (name/roomid/defaultbiome/
   region) — missing → boot panic.
2. **Room `title` gets canonical Title-Case validation** (like mob names) —
   "of/the/a" stay lowercase, other words capitalize ("Over", "Bridge").
3. **Mob `character.name` must be canonical Title Case**; filename lowercase
   via `ConvertForFilename`. A combat mob's filename must match
   `ConvertForFilename(name)` exactly.
4. **Room noun keys with prose colons / multi-word values need `>` block
   scalars** (or quotes); `idlemessages` with a colon-space MUST be quoted
   (can't use `>`).
5. **Highlighted room nouns:** single words need no hyphenation; multi-word
   interactable nouns hyphenate BOTH the ansi token and the noun key.
6. **Items live in `materials-40000/` by id-range** regardless of type;
   filenames keep any leading article.
7. **Dispatch dialogue paths with the FULL `_datafiles/world/dogmud/dialogue/
   east_road/...` prefix** (a relative path made a subagent write to repo root
   in a prior build).
8. **Exit `kind` (long/normal/vertical) is mapper-derived from coord delta** —
   do NOT author a `kind:` field. Keep consecutive spine rooms 1 cell apart so
   nothing auto-classifies `long`.
9. **Coordinate collisions are a boot-panic** — re-run boot
   `ValidateZoneConsistency` (mode=panic) and/or `cartcheck east_road` after
   placement.
10. **Ambient archetype is `noncombat_passive`** (no `noncombat_ambient`).

## Validation & Test Plan

1. **`python tools/id_inventory.py`** before authoring; confirm ranges still free.
2. **Nuke instance saves** (`rooms.instances/*`, `mobs.instances/*`) before
   every smoke test.
3. **Clean boot** — `ValidateZoneConsistency errors=0 mode=panic`, no load
   panics, mob/room/item counts as expected.
4. **World-critic pass** over the zone (data-review mode) — it reliably catches
   (a) river/compass-direction botches and (b) dialogue node-shadowing
   (`strings.Contains(topic,trigger)` substring shadow). RUN IT — it earns its
   keep every district.
5. **Feel-test** — a full west→east walk: the 6250 seam, the dry-plateau
   progression, the orbital waystone, all NPC dialogues + the cooking vendor
   buy, the forageables, Greenford's profile appearing at 6273, the barred
   bridge terminus. Zero-bug bar before merge.
6. **Update `docs/ZONE_EXPANSION.md`** — mark the Confluence row complete
   (it's stale at "Building 4/10"), mark zone 18 ✅ Built with the roomid range,
   and refresh the TOTAL row.

## Build Method

Full **spec → plan (writing-plans) → subagent-driven-development**, the
established district SOP. Merge to master with `--no-ff` after the world-critic
+ feel-test passes. Unpushed (batches with the rest of the Eastern Arc or the
next prod push at the user's discretion).
