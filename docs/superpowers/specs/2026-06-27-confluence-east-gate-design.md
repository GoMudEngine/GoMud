# The Confluence — District 8: East Gate & the Brennside (final district)

**Date:** 2026-06-27
**Status:** Approved (design phase)
**Umbrella:** `docs/superpowers/specs/2026-06-26-confluence-citywide-design.md` (§3 row 10, §9 east seam)
**Predecessor:** District 7 (Craftsmen's Row) merged. **This is the LAST district — it
completes the Confluence.**

## 1. Concept

The east bank of the Brenn, reached by a new public bridge from the civic core — the
city's outward edge toward Greenford. A travelers' gate-quarter: the east gate (the
**road to Greenford, a stub** for a future zone), a travelers' inn, a stable, gate
guards; the land drying out and thinning eastward. **No quest, no mystery** — the
city's threshold to the wider world, and a quiet outward gesture (the road continues;
the world is bigger than the Confluence).

## 2. Geography & seam

- **Seam:** room **6146 "The Three Waters"** (the Tri-Cross Square inn, `{-4,-67,0}`,
  free EAST exit) → **6246**, a bridge-head at `{-3,-67,0}`, then a **public bridge over
  the Brenn** (the eastern river) running EAST across open water (long exits; no rooms
  in the span — `cartcheck`-clean) to the **east bank** at x ≈ +9..+12. This is the
  public crossing, distinct from the temple causeway (which is at y−73).
- **Region (collision-free, verified):** the bridge spans y=−67, x −3 → +8 (open
  river); the east-bank quarter sits x [+9,+12], y [−66,−70]. No existing rooms there
  (the island/cloisters are at y≤−73 / z<0). Exact coords below; `cartcheck`-verified
  (mode=panic).
- Biome `city` (the gate/inn/quarter); the bridge reads over water.

## 3. Rooms (6246–6257, 12)

| roomid | title | coord | exits |
|--------|-------|-------|-------|
| 6246 | The Brenn Bridge, West End | -3,-67,0 | W→6146, E(long)→6247 |
| 6247 | The Brenn Bridge, Crown | 2,-67,0 | W(long)→6246, E(long)→6248 |
| 6248 | The Brenn Bridge, East End | 8,-67,0 | W(long)→6247, E→6249 |
| 6249 | The East Gate | 9,-67,0 | W→6248, E→6250, N→6251, S→6252 |
| 6250 | The Greenford Road | 10,-67,0 | W→6249 (+ described EAST stub → Greenford, NOT wired) |
| 6251 | The Brennside Wharf | 9,-66,0 | S→6249, E→6253 |
| 6253 | The Eastern Landing | 10,-66,0 | W→6251 |
| 6252 | The Travelers' Inn | 9,-68,0 | N→6249, S→6254, E→6255 |
| 6255 | The Stable | 10,-68,0 | W→6252 |
| 6254 | The Inn Common Room | 9,-69,0 | N→6252, E→6256 |
| 6256 | The Drying Flats | 10,-69,0 | W→6254, S→6257 |
| 6257 | The Eastern Edge | 10,-70,0 | N→6256 |

12 rooms, all exits reciprocal; the 3 bridge exits (6246E, 6247W+E, 6248W) are
auto-`long` over open water (no rooms in the span). **6250's east is a DESCRIBED stub**
(the road out to Greenford — a future zone, not wired), like the River Road 6105 stub.
**6257 The Eastern Edge** is the city's literal outward terminus — land drying to
scrub, the river-country opening east — a quiet closing beat for the whole city.

Required noun(s): light — a `gate-arch` or `milestone` at 6249/6250 (a road-marker
noting the distance to Greenford — the outward gesture). No quest nouns.

## 4. NPCs (mobs/dialogue 9484–~9490, ~6–7)

Texture; all `non_combatant`. Suggested:
- **The Gate Captain** (9484) — runs the east gate; civic/road-warden voice; the
  Confluence's outward face; a word about the Greenford road (the world beyond).
- **A Gate Guard** (9485) — ambient.
- **The Innkeeper** (9486) — the travelers' inn; **cooking vendor** (traveler's fare);
  warm, sees everyone passing through.
- **The Stablehand** (9487) — ambient; horses, mules, the carters' trade.
- **A Wayfarer** (9488) — a traveler bound to/from Greenford (the outward nod: there's
  a world out there; lore-light, NO mystery).
- **A Brennside Fisher / Dockhand** (9489) — Quayfolk; the east-bank waterfront.
- (optional **9490** a drover/child ambient.)
Factions: guards may carry `road_wardens` or a civic group if natural; fisher
`quayfolk`; rest `[humanoid]`. No new faction. Title-Case names; vendor `craft_support`
+ `shop:` (items a real discipline).

## 5. Items (40146+)

Minimal: a traveler's meal (reuse 40135 broth / 40136 bread, or 1 new "traveler's
stew" 40146) for the innkeeper; optionally a road-keepsake. **≤2 new; reuse first.**

## 6. Lore

No pre-Founding mystery. The only outward thread: the Greenford road / a wayfarer —
"the world continues east," never naming the answer to anything. The Eastern Edge
(6257) closes the city on an open, outward note.

## 7. Schedules

2–3 anchors (the innkeeper day+evening; the gate captain by day; the stablehand) —
findability-preserving.

## 8. Process & verification

Spec → plan → subagent-driven build → boot/cartcheck (panic; confirm the bridge long
exits + the 6250 Greenford stub) → world-critic + feel (directions; the bridge reads;
no mystery leak; vendor categories; casing) → merge → fire reviewers. **No quest.**
On merge, **the Confluence is COMPLETE** (40 zones; the full ~150-room tri-city) and
the whole multi-district bundle is ready for the user's prod push.

## 9. Out of scope

- Greenford & the East Road (the future zone the 6250 stub points to).
- Any quest (optional Q75 never built — deferred indefinitely).
- A working bridge/ferry mechanic beyond ordinary room exits (the bridge is just rooms).

## 10. World impact

+12 rooms, ~6–7 mobs. **Completes the Confluence** — the tri-city is done end to end:
river districts, civic core, scholars, the temple island + Q74 climax, the lived-in
working quarter, and the east gate to the wider world.
