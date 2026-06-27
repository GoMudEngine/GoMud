# The Confluence — District 5a: The Processional

**Date:** 2026-06-27
**Status:** Approved (design phase)
**Umbrella:** `docs/superpowers/specs/2026-06-26-confluence-citywide-design.md` (§3 row 4, §10 build-order step 4)
**Predecessor:** Scholars' Quarter + Q73, merged 2026-06-26. This is the next district.
**Scope note:** The city-wide build-order step 4 covers two room blocks (Processional
6154–6167 + Temple public 6168–6183). **This spec is the Processional half only**
(6154–6167). The public Temple is a separate follow-up spec→plan→build.

## 1. Concept

The ceremonial temple approach. A broad west-bank avenue runs south from the
built city, opens onto a forecourt at the water's edge, then a **causeway crosses
east over the joining rivers** to the temple's grand entrance on the central
island. The geography itself stages the climax-to-come: the player walks over the
three waters and arrives at *the symbol above the door.*

This is the player's first contact with the **Keepers** (the temple clergy) and
with the **official, settled version of the faith** — the orbital symbol presented
as early Chrysalis cocoon-theology. To a player who completed Q73 (the Margin
Notation), that official line is *knowably* a reinterpretation. The district shows
the Keepers' controlled public face; the contradiction is visible in the stone and
documented, quietly, by a single Margin scholar at the portico.

**Lore boundary (carried from the city-wide layer §5):** threshold only. The
Processional may *foreshadow* that the foundations rest on older work and that the
undercroft is not shown to visitors. It never states the *why* (crash, gray
material, mutation link) — that is reserved for the crash-site zone. No numerology
lectures.

## 2. Geography & coordinate frame

- **Seam:** room **6153 "Processional Gate"** `{-5,-69,0}` (the temple-ward edge
  of Tri-Cross Square; its `the way south` noun is the ready-made stub). The
  Processional attaches via a **new `south` exit on 6153** → 6154.
- **Avenue:** west bank, x −5, running **south** (y −70 → −74).
- **Causeway:** from the forecourt, **east over open water** (x −4 → +1) using
  `long` exit kinds (no rooms span the water — cartcheck-clean).
- **Island:** temple steps + portico at **x +3** — matching the city-wide anchor
  for the temple island, setting up the Temple-public follow-up to extend east/
  south on the island.
- **Vertical:** one proper up/down pair (Pilgrim Hall 6160 ↔ Dormitory 6161 at
  z+1, stacked coords) — the foreman-fix vertical pattern.
- **Exact per-room coords below are the build target and MUST be `cartcheck`-
  verified (mode=panic) against all previously-built districts before commit.**
  No collisions expected: Scholars' Quarter sits far south (~y −82); the open-
  water span at y −73 (x −3…+2) and the island (x +3) hold no existing rooms.

## 3. Rooms (6154–6167, 14)

| ID | Room | Coord | Exits / role |
|----|------|-------|--------------|
| 6154 | Processional Avenue, North End | {-5,-70,0} | N→6153 (seam), S→6155. Ceremonial scale, buildings set back. |
| 6155 | Avenue — Votive Stalls | {-5,-71,0} | N→6154, S→6156. **Offering-seller** (9451). |
| 6156 | Avenue, South End | {-5,-72,0} | N→6155, S→6157, E→6162. |
| 6162 | The Hall of the Founding | {-4,-72,0} | W→6156. **The historian** (9449); founding-relief; official plaque (Q74 seed). |
| 6157 | The Temple Forecourt | {-5,-73,0} | N→6156, W→6158, S→6160, E→6163 (causeway). Hub; **warden** (9450), **acolyte** (9453). |
| 6158 | The Meditation Garden | {-6,-73,0} | E→6157, W→6159. Quiet walled pocket. |
| 6159 | The Still Pool | {-7,-73,0} | E→6158. Contemplative dead-end; faint old-stone beat. |
| 6160 | The Pilgrim Hall | {-5,-74,0} | N→6157, U→6161. **Hospitaller** (9452, light cooking vendor); ambient pilgrims (9455–9456). |
| 6161 | The Pilgrim Dormitory | {-5,-74,1} | D→6160. Bunk room above; window over the avenue. |
| 6163 | Causeway, West Span | {-4,-73,0} | W→6157, E(long)→6164. "Three waters below" begins. |
| 6164 | Causeway, Crown | {-1,-73,0} | W(long)→6163, E(long)→6165. High point, water both sides; old-stone / orbital beat; great-hall-over-joining-waters foreshadow. |
| 6165 | Causeway, East Span | {1,-73,0} | W(long)→6164, E→6166. Reaching the island. |
| 6166 | The Temple Steps | {3,-73,0} | W→6165, S→6167. Island foot; the grand stair. |
| 6167 | The Temple Portico | {3,-74,0} | N→6166. **The symbol above the door**; great doors S = **stub to Temple public** (next build); **Margin scholar** (9454). |

Room count: 14 (6154–6167). The avenue/forecourt are `city` biome (matching
Tri-Cross Square); the garden/still-pool lean garden/contemplative flavor in prose;
the causeway/island read over water. Use `city` biome throughout for map
consistency unless a `water`-adjacent tint reads better on the causeway (builder's
call, kept consistent).

## 4. NPCs (mobs/dialogue 9449–9456)

All clergy are **`non_combatant: true`**. Faction membership via `groups:`.

| ID | Name (Title Case) | Faction groups | Where | Role |
|----|-------------------|----------------|-------|------|
| 9449 | The Historian | [humanoid, keepers] | 6162 | **Anchor.** Guided-tour founding lore; states the *official* symbol line (early Chrysalis cocoon-theology). A Q73-completed player can tell it's a reinterpretation (an aside gated on the Q73 end token). **Q74 seed:** foundations rest on "older work the Founders consecrated"; points onward to the senior Keepers / inner precincts (the Cloisters) and notes the undercroft "is not shown to visitors" — **grants nothing.** |
| 9450 | A Processional Warden | [humanoid, keepers] | 6157 | Keeps order at the forecourt; the one who mildly eyes the Margin scholar (the friction, made legible without confrontation). |
| 9451 | The Offering-Seller | [humanoid] | 6155 | Vendor: votive candle, river-reed wreath, temple incense (40139–40141). The merchant's `craft_support` may be `general` (mixed devotional goods); the **items themselves** carry a real discipline (never `general`) and the vendor stocks them via its explicit `shop:` list (the chandler-as-catch-all pattern). |
| 9452 | The Hospitaller | [humanoid, keepers] | 6160 | Pilgrim-hall keeper; light **cooking** vendor (pilgrim broth/bread — reuse existing where possible, ≤1–2 new). |
| 9453 | An Acolyte | [humanoid, keepers] | 6157/6158 | Ambient Keeper; tends the garden, sweeps the forecourt; small flavor lines. |
| 9454 | A Margin Scholar | [humanoid, margin] | 6167 | **The one quiet Margin presence.** Quietly copying the symbol above the door; the Keepers tolerate but watch. Q73-completed player gets a knowing aside (the Margin's whole project, in miniature, at the climax-approach). |
| 9455 | A Kneeling Pilgrim | [humanoid] | 6160/6167 | Ambient, `noncombat_passive`. |
| 9456 | A Road-Worn Pilgrim | [humanoid] | 6160 | Ambient, `noncombat_passive`. |

Roster: 8 mobs. (Use `noncombat_passive` archetype for pure-ambient pilgrims; the
named clergy/vendors get appropriate archetypes; `non_combatant: true` on all
clergy so they cannot be attacked/stolen-from.)

## 5. Factions

- **Create `factions/keepers.yaml`** — `faction_id: keepers`, display name
  "The Keepers of the Confluence", description (temple clergy; maintain the
  official line that the pre-Founding orbital marks are early Chrysalis theology;
  gate the inner temple and undercroft), `default_rep` neutral.
- **No hard combat enmity edge** keepers↔margin in this build. The tension stays
  narrative/dialogue-only here; the real allegiance mechanic lands on **Q74's
  light flag** (city-wide §4, §6). Do **not** set an `enemies:` edge that would
  make clergy aggress a Margin-aligned player. (`margin` faction already exists
  from the Long Quay; leave it unchanged.)
- Do **not** create `factions.rep/keepers.yaml` — that's runtime state, gitignored.

## 6. Items (40139+)

- **40139** a beeswax votive candle, **40140** a woven river-reed wreath,
  **40141** a pinch of temple incense — offering-seller goods (flavor; small
  value; a real discipline category, NOT `general` on the item itself).
- Pilgrim fare: **reuse existing** cooking goods where possible; ≤1–2 new only if
  nothing fits.
- **No quest item** (no quest in this district).
- 40xxx item files live under `items/materials-40000/` regardless of type
  (Filepath() routes by id range); filenames keep any leading article.

## 7. Q74 seeding (no grant)

The Processional plants threads the Cloisters/Undercroft build will pay off — it
**grants no quest token**:
1. **The door symbol** (6167 portico) — the oldest stonework, a four-ring orbital
   mark, plainly older than the temple; the official plaque (in the Hall, 6162)
   calls it Chrysalis cocoon-theology; the Margin scholar documents the
   discrepancy. Environmental + dialogue.
2. **The historian** (6162) — points onward to the senior Keepers / the inner
   precincts (the Cloisters) and "older work" beneath the foundations; one line
   that the undercroft is not shown to visitors.
3. **The Q73 echo** — Q73-completed players get knowing asides from the historian
   and the Margin scholar (gated on the Q73 end token via dialogue `questRequired`
   lists).

Threshold discipline: no crash, no gray material, no mutation link.

## 8. Schedules (anchor NPCs)

Follow the established district pattern: simple day-at-post + night routines that
**preserve findability**. Candidates: the historian (gives tours by day, retires
at night), the offering-seller (day at the stalls), the hospitaller (the pilgrim
hall, day and evening). Keep the Margin scholar and wardens findable. May be built
with the rooms or in the polish pass — author them so the world-critic/feel-tester
check passes.

## 9. Process & verification

1. Build via subagent-driven dispatch (rooms, then mobs/dialogue, then items,
   faction, schedules), pre-allocating ID blocks if parallelized.
2. **Boot test** with `ValidateZoneConsistency` mode=panic + `cartcheck
   the_confluence` clean (zero collisions, reciprocal exits, long causeway exits
   render correctly). Wipe instance saves before smoke per the SOP.
3. **Polish pass (mandatory):** run the **world-critic + feel-tester** over
   6154–6167. The recurring district lessons to check explicitly:
   - **River/compass directions** — subagents reliably botch these. The Aldren is
     the *northern* tributary; no Aldren south of the junction; the combined water
     spills SW. Double-check every direction word against room canon.
   - **Dialogue node-shadowing** — `strings.Contains(topic, trigger)` substring
     match; place specific/gated nodes FIRST or drop colliding short triggers.
   - Mob names canonical Title Case; idlemessages with colon-space quoted; nouns
     using `>` block scalars where prose has colons.
4. No quest to harness-verify here; verify the seam zone-transition (6153↔6154),
   the historian/Margin-scholar dialogue (incl. the Q73-gated asides), the vendor
   list/buy, the causeway long-exits and map rendering, and the portico stub.

## 10. Out of scope

- The public Temple interior (6168–6183) — next spec.
- Q74 itself and the Cloisters/Undercroft — the district-5b/6 build.
- The working barge transit (deferred city-wide).
- Any keepers↔margin combat enmity / rep aggression (Q74's flag, later).

## 11. World impact

World grows by 14 rooms and ~8 mobs. The Confluence reaches 4 districts + the
Processional approach (~58 of its ~152 rooms), with the temple island's western
foot established for the Temple-public follow-up.
