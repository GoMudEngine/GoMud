# The Confluence — District 7: Craftsmen's Row & Residential

**Date:** 2026-06-27
**Status:** Approved (design phase)
**Umbrella:** `docs/superpowers/specs/2026-06-26-confluence-citywide-design.md` (§3 row 9)
**Predecessor:** The Confluence spine (island + Q74) is complete through 6b. This is the
first of the two outer-quarter closeout districts (the other: East Gate & Brennside).

## 1. Concept

The lived-in working quarter of the Confluence — craft row + ordinary homes, southwest
of the Scholars' Quarter. **No mystery, no quest.** Pure texture: the city as a place
people actually live and work, giving the temple-and-scholars city its everyday human
weight. The contrast (after the undercroft climax) is the point — bread, barrels,
washing-lines, gossip.

## 2. Geography & seam

- **Seam:** room **6231 "The Garden Walk"** (`{-12,-68,0}`, Scholars' Quarter SW
  corner, free WEST exit) → new **6232** at `{-13,-68,0}`. The scholarly quarter's
  outer wall opens to the working quarter (a crisp scholar→artisan transition).
- **Region (collision-free, verified):** x ∈ [−13,−18], y ∈ [−68,−85], z=0. The built
  west-bank city is bounded x [−12,−3], y [−74,−52]; this quarter sits west/south of
  it. Exact per-room coords assigned at build and **`cartcheck`-verified (mode=panic)**.
- Biome `city`.

## 3. Rooms (6232–6245, 14)

A craft row + a small market + residential lanes. Exact layout at build; suggested
shape (a north–south lane off the seam, craft shops and homes branching):
- **6232 The Artisans' Gate** — the seam room (E→6231); the threshold into the working
  quarter; a wall-plaque or the change in the paving.
- **Craft Row (~5 rooms):** a **cooperage** (barrels — ties to NP's cooperage circle
  thematically, but NO mystery here), a **potter's**, a **weaver's**, a **smithy**, a
  **chandler/general craft-supply**. Some are vendor shops.
- **The Daily Market (~2 rooms):** a small produce/goods market square + a **baker's**
  (warm, the best-smelling room in the city).
- **Residential (~4 rooms):** ordinary homes/lanes — a tenement court, a lane of
  craftsfolk's houses, a washing-court, the **riverman's house** (the riverman's wife).
- **A quiet corner (~1 room):** a small shrine-niche or a well-court; the
  **retired functionary's** doorstep.
Each room: `city` biome; a highlighted noun or two of flavor where natural; 2–3
idlemessages (quote colons; nouns/descriptions with colons use `>` blocks).

## 4. NPCs (mobs/dialogue 9472–~9483, ~10–12)

Texture residents; all `non_combatant`. A mix of vendors + ambient + 2 named-character
anchors. Suggested:
- **The Baker** (vendor, `cooking`) — bread/pastry; warm civic heart of the quarter.
- **The Cooper / Potter / Weaver / Smith** — craftsfolk; 1–2 are vendors (a craft
  good each), the rest ambient with idlecommands.
- **The Craft-Supply Seller** (vendor) — general/mixed craft goods.
- **The Riverman's Wife** (named anchor) — a real character: her husband works the
  river (ties to the Quayfolk); daily-life voice, a little lonely, sharp; the human
  texture of a working marriage in a river city. Lore-light (maybe a wry word about
  the temple or the scholars, nothing pre-Founding).
- **The Retired Functionary** (named anchor) — an old civic clerk, pensioned off;
  remembers the city's ordinary history (NOT the mystery); opinions on everything;
  comic-poignant.
- **Ambient residents** (Title Case): a washerwoman, children, a journeyman, an old
  man at the well.
Factions: residents are mostly unfactioned `[humanoid]`; some may carry `quayfolk` or
a civic group if natural. **No new faction.** Title-Case names; vendor `craft_support`
+ `shop:` lists (items carry a real discipline, never `general`).

## 5. Items (40143+)

A few flavor goods: baked goods (loaf/pastry — or reuse 40136 black bread), a craft
good or two (a fired-clay pot, a bolt of cloth — reuse NP/Confluence goods where
possible), traveler/household fare. **Minimal new items (≤4); reuse first.**

## 6. Lore

**None of the pre-Founding mystery.** This is the normal city. At most a *wry, mortal*
nod (a resident's offhand opinion of the temple or the scholars) — never the orbital
symbol, never the fourth, never the undercroft. The contrast with the climax is the
value.

## 7. Schedules

2–4 anchor schedules (the baker up before dawn; the riverman's wife's day; a craftsman
at the bench by day) — findability-preserving, the established pattern. Ambient
residents unscheduled.

## 8. Process & verification

Spec → plan → subagent-driven build → boot/cartcheck (panic) → **world-critic +
feel-tester** (the recurring traps: directions vs the seam/coord layout; mob/room
Title-Case; colon quoting; hyphenated nouns; no `kind:` on exits; vendor category
validation) → merge → fire reviewers. **No quest to harness-verify** — verify the seam
(6231↔6232), the vendor list/buy, the two named NPCs' dialogue, and the quarter's
walkability. Then build the last district (East Gate & Brennside).

## 9. Out of scope

- Any quest (optional Q75 deferred indefinitely).
- The East Gate & Brennside (the next/last district — the Brenn bridge + east bank).
- Any pre-Founding lore.

## 10. World impact

+14 rooms, ~10–12 mobs. The Confluence reaches its lived-in SW edge; only the East
Gate / Brennside remains to finish the city.
