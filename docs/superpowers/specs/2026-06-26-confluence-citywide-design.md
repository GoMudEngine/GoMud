# The Confluence — City-Wide Design Layer

**Date:** 2026-06-26
**Status:** Approved (design phase) — the umbrella spec for a multi-district build
**Roadmap:** `docs/ZONE_EXPANSION.md` Phase 5, Zone 5.4 (scaled up from 70→~150 rooms)
**Predecessor seam:** River Road to the Confluence (leg 2), merged `b2b773b8` — room **6105 "The Confluence Gates"** is the barred north stub this city opens.

> **This is a city-wide design layer, not a buildable plan.** Like the New
> Plymouth citywide layer (`2026-06-20-new-plymouth-citywide-design.md`), it
> defines geometry, factions, the mystery/quest spine, ID blocks, and the
> coordinate discipline. **Each district is then its own spec → plan → build.**
> Build order is at the end.

## 1. Purpose & scale

The Confluence is the **keystone of the pre-Founding mystery** — the payoff zone
for everything seeded across New Plymouth and the Southern Road (the orbital
symbol, the three-vs-four motif, "what changed in the sky," the cooperage
circle's question, the buried-lintel/gallery-cipher web). It is Aldric's temple
home: *a city built around a temple built on top of something it doesn't
understand.*

- **Scale: ~150 rooms** (≈ half New Plymouth's planned 300+ — `ZONE_EXPANSION.md:524`).
- **~10 districts**, each a separate spec→plan→build.
- A real second city: faith + scholarship + river trade, smaller than the
  capital but a genuine destination.

## 2. Geography — a true tri-city

`world.md` names it the "Tri-Cities": three rivers meet here.

- The **Aldren** descends from the north (the river the player followed down
  River Road) and enters at the **northwest river-ward gate** (the 6105 seam).
- The **Brenn** comes in from the **east**.
- The **Solt** is the third, from the **southwest**.
- They join in a **Y-junction** and the combined water "spills away southwest"
  toward the coast — exactly what the River Road bluff (6096) showed.
- The **temple sits on the central spit/island** where the three waters meet,
  built over the pre-Founding site, with the great hall where "the three rivers
  join in a single channel below the floor" (Wess the pilgrim's line, 9414).
- Three bank-quarters, bridges/causeways between them, the temple island
  central, the undercroft beneath it.

**Civic flavor — the forgotten fourth.** Everyone calls it the Tri-Cities (three
rivers), but a fourth channel went dry "before living memory" and stopped being
counted (planted in River Road via Sedge 9413). The city's own name encodes the
forgotten fourth — the whole mystery in miniature. Carry this lightly:
flavor, never lecture.

## 3. Districts (~152 rooms, rooms 6106–6257)

Each district gets a **contiguous room block** and a **rough coordinate anchor**.
**Exact per-room coordinates are assigned at district-build time and MUST be
`cartcheck`-verified against all previously-built districts** — the single most
important collision-prevention rule (the core NP lesson; see §7).

| # | District | Rooms | Pos (rough anchor) | Role / key anchors |
|---|----------|-------|--------------------|--------------------|
| 1 | **The Landings** (River District) | 6106–6121 (16) | NW, west bank, just S of 6105 (~x−8,y−56) | Barge docks (Davan's departure), warehouses, fish traders, the **barge master** (Confluence↔NP lore); the River Road seam. |
| 2 | **The Long Quay** | 6122–6137 (16) | West bank waterfront (~x−9,y−64) | Trade waterfront + river market; the prosperous Quayfolk belt. |
| 3 | **Tri-Cross Square** (City Center) | 6138–6153 (16) | West-central lower city (~x−11,y−73) | Main square, **The Three Waters inn**, municipal hall (the margin-notation map), shops; orbital symbol in old stone, Chrysalis motifs in new. |
| 4 | **The Processional** (Temple approach) | 6154–6167 (14) | Central, W→island causeway (~x−3,y−69) | Grand entrance (symbol above the door), forecourt, meditation garden, pilgrim hall, the **historian**. |
| 5 | **Temple of Confluence (public)** | 6168–6183 (16) | Temple island, W half (~x+3,y−69) | Public nave, the great hall over the joining waters, chapels, reliquary. |
| 6 | **Cloisters & Archive** (Temple inner) | 6184–6199 (16) | Temple island, E half (~x+9,y−70) | **Aldric** (or successor), archive/records, **Brother Cael**, **Prioress Crane**, the older east corridor + the stairs down; trust-gated. |
| 7 | **The Undercroft** | 6200–6217 (18) | Beneath the island, z−1…z−3 (~x+4,y−70,z<0) | Multi-level dungeon: pre-Founding construction, wards/guardians, the sealed chamber + the orbital **"face"** (the threshold reveal); endgame, quest-gated. |
| 8 | **The Scholars' Quarter** | 6218–6231 (14) | South-central (~x−4,y−82) | Study halls, library, the **Margin-Notation scholar** + **bookseller**; "where old questions are studied." Q73 home. |
| 9 | **Craftsmen's Row & Residential** | 6232–6245 (14) | South/SW (~x−13,y−84) | Craft row, daily market, baker, riverman's wife, retired functionary; the lived-in city. |
| 10 | **East Gate & the Brennside** | 6246–6257 (12) | East bank, toward Greenford (~x+12,y−67) | East gate (road to Greenford, **stub**), travelers' inn, stable, guards; land drying eastward. |

## 4. Factions

Three, fewer than NP — the story here is tighter.

- **The Keepers of the Confluence** (temple clergy). Hold the temple, gate the
  undercroft, maintain the *official* line that the pre-Founding orbital marks
  are early Chrysalis theology. Anchors: Aldric, Brother Cael, Prioress Crane.
  Not cartoon villains — some genuinely believe; at least one is actively
  keeping the lid on.
- **The Margin** (scholars). The quiet community asking the old questions (the
  margin notations, three-vs-four). They know more than the temple admits. **The
  scholarly node of the pre-Founding web** — linked in spirit to New Plymouth's
  cooperage circle (Orin's "what changed in the sky") and the future Greenford
  university. Q73's faction.
- **The Quayfolk** (river trade). Prosperous, pragmatic, outward-facing; run the
  waterfront and the Confluence↔NP barge. Neutral on the mystery, care about
  cargo. The city's normal-life texture.

**Tension axis:** Keepers ↔ Margin (the mystery). Quayfolk stay neutral, keeping
the city a real working place. Faction reputation hooks live mainly on Q74's
light allegiance coloring (§6).

## 5. The mystery architecture & lore boundary

The orbital symbol = a depiction of the **planet's moons/orbits** (`world.md`'s
moon table). The pre-Founding civilization recorded an older sky; the Confluence
temple is built atop that suppressed record; the Keepers reinterpret the marks
as Chrysalis theology; the Margin seeks the truth. The "fourth" (water/moon)
that stopped being counted is "what changed."

**LORE-BOUNDARY DISCIPLINE (carried into every district spec):**
- The undercroft reveals the **threshold only** — *there was a fourth; the sky
  changed; the temple sits on a suppressed pre-Founding record; the faith is a
  reinterpretation.* The player leaves understanding *something is missing.*
- It **never states the why** — the crash, the gray material, the link to the
  Chrysalis mutations. **That is reserved for the crash-site zone** (roadmap
  #22). The Margin gives the player the thread that points onward.
- Keep it earned and unforced — no numerology lectures (per the River Road
  steer). Environmental storytelling + a few well-placed NPC lines.

## 6. Quest spine

IDs from **73**. (Q75 optional, deferred to a per-district build.)

- **Q73 — The Margin Notation** (entry investigation, lore-heavy). A Margin
  scholar (Scholars' Quarter) studies old maps whose margins disagree on the
  count of waters. Breadcrumbs seeded across the city: a remark at the Three
  Waters inn (Tri-Cross Square), a map in the municipal hall, a bookseller's
  damaged chart (Scholars' Quarter). Gather them, compare three-vs-four, trace
  it to the temple's oldest stonework (the symbol above the door). Resolution
  opens the undercroft question and gates Q74.
- **Q74 — The Undercroft** (endgame descent). Earn the Keepers' trust (or let
  the Margin slip you in), follow the construction-history clues down, reach the
  sealed chamber, witness the orbital **"face"** — the four-ringed old sky. The
  **threshold reveal** (§5). Ends pointing onward (the crash site). **Light
  allegiance coloring** via a quest flag (carry the truth to the Margin vs. keep
  the Keepers' confidence) — a soft arc à la NP's flags, NOT a hard branch, to
  control scope. Declare the flag in the quest YAML.
- **Q75 (optional, deferred)** — a Quayfolk/residential texture quest (the barge
  master, a missing shipment, the riverman's wife) for non-mystery players.
  Specced at its district build if wanted, not in this layer.

## 7. Coordinate frame & collision discipline

- The city sits **south/southwest of River Road 6105 `{-5,-51,0}`**, spanning
  roughly **x ∈ [−25,+15], y ∈ [−52,−92], z=0**, temple island central
  (~`{+3,-70}`), **undercroft z−1…z−3**.
- The §3 anchors are **rough placement guidance, not fixed coords.** Each
  district build assigns exact per-room coords within its neighborhood and
  **MUST run `cartcheck <zone>` / boot `ValidateZoneConsistency` (mode=panic) and
  resolve every collision against all previously-built districts before
  committing.** A duplicate coord panics at boot.
- **Zone name:** `The Confluence` → folder `the_confluence/` (verify via
  `ConvertForFilename`; underscores). One zone, built incrementally district by
  district (all districts share the zone folder + a single `zone-config.yaml`,
  `defaultbiome: water`, region **"The Tri-Rivers"** — a new region; confirm it
  doesn't collide with an existing region name at the first-district build).
- **Every new zone folder needs a `zone-config.yaml`** (the River Road gotcha —
  missing → boot panic "No zone-config.yaml was loaded for roomId"). The
  Confluence's is created with the first district (The Landings).

## 8. ID allocation (reserved up front)

- **Rooms 6106–6257** (152) — per-district blocks in §3.
- **Mobs/dialogue 9419–9489** (~70 reserved; ~45–55 used). Each district build
  pulls its block via `python tools/id_inventory.py --alloc` at build time.
- **Items 40127–40179** (quest items, vendor goods, undercroft relics).
- **Quests 73, 74** (+75 optional).
- **Buffs 94+** (undercroft wards, if any).

## 9. Seams

- **North (river-ward):** River Road **6105** → The Landings (the first district
  build opens the 6105 south stub).
- **East:** East Gate → Greenford (**stub** for the future zone 6.1/6.2).
- **River barge:** The Landings ↔ New Plymouth (the canon Davan barge). The dock
  + barge master are built as **lore now**; the **working transit is deferred**
  (ferries are a flagged later mechanic). Revisit if a simple paid barge
  fast-travel is wanted.

## 10. Build order

Seam-first, climax-aware. Each is its own spec→plan→build.

1. **The Landings** — connects the River Road 6105 seam (must be first).
2. **The Long Quay + Tri-Cross Square** — civic core; plants the Margin-Notation
   breadcrumbs.
3. **The Scholars' Quarter** — the Margin's home; **Q73 grants here**.
4. **The Processional + Temple (public)** — approach + worship.
5. **Cloisters & Archive + The Undercroft** — inner temple + the climax; **Q74 +
   the threshold reveal**.
6. **Craftsmen's/Residential + East Gate** — outer quarters, last.

Quests slot in as their locations come online: Q73 after Scholars' + Square;
Q74 after the Undercroft.

## 11. Out of scope (this layer)

- Per-room prose, exact coords, mob stat blocks, dialogue — all at district builds.
- The working barge transit mechanic (deferred, §9).
- The crash-site reveal (reserved, §5).
- Greenford and the East Road (the east seam is a stub).
- Q75 and any per-district texture quests.

## 12. World impact (at completion)

World grows from **39 zones / 1022 rooms** to **40 zones / ~1174 rooms** when all
ten Confluence districts are built.
