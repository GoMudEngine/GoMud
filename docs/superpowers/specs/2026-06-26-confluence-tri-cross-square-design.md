# The Confluence — District 3: Tri-Cross Square — Design

**Date:** 2026-06-26
**Status:** Approved (design phase)
**Umbrella:** `docs/superpowers/specs/2026-06-26-confluence-citywide-design.md` (district 3 of 10)
**Predecessor:** District 2 The Long Quay, merged — room **6137 "The Quay's South Gate"** is the stub this district opens.

## Purpose

The Confluence's **civic heart** — the main square where the city's roads cross,
inland from the waterfront. The seat of municipal life: the great square, **The
Three Waters inn**, the **Municipal Hall**. This is the district that **seeds the
bulk of the Q73 (Margin Notation) thread** — the notation map, the inn remark,
the Margin scholar — and where the orbital-symbol motif **escalates** from
incidental seeds to *"it's everywhere"* (worked into the old civic stone,
reinterpreted as Chrysalis on the new).

**No quest** (per the umbrella — Q73 grants later in the Scholars' Quarter and
hooks these breadcrumbs; Q74 in the temple). Breadcrumbs are **lore now**
(descriptive nouns + dialogue), not live quest mechanics — no quest token exists
yet.

## Scope & IDs

- **16 rooms, 6138–6153** (the umbrella's reserved block for district 3).
- **Mobs/dialogue 9434–9440** (7).
- **Items 40133–40134** (2 civic goods).
- **Quests:** none.
- **Zone:** `The Confluence` (existing folder; no new zone-config).
- **Biome:** `city` throughout (the waterfront ends at the Long Quay gate; `city`
  is the NP civic biome — valid for rooms, no forage, fine).
- **Factions:** civic NPCs are **unfactioned** (`groups: [humanoid]` — they're
  municipal townsfolk, not Quayfolk). Only the Margin Scholar (9435) carries
  `margin` (the existing faction). No new faction.

## Seam — open the Long Quay stub (6137)

Edit `_datafiles/world/dogmud/rooms/the_confluence/6137.yaml`:
- Add `south: {roomid: 6138}` to `exits:` (keep `north: 6136`).
- Lightly revise the `description` / `the way south` noun so the road now **runs
  on** into Tri-Cross Square (flip "a walk for another day" to the road being
  open south into the square). The noun already describes the square accurately.

## Layout — 16 rooms

A N–S civic spine through the central plaza; **The Three Waters inn east**, **the
Municipal Hall west**. Proposed coordinate grid below — **the build assigns final
coords and `cartcheck`-verifies** against the Long Quay and all prior zones. The
inn's lodging is a **proper vertical** (`up`/z+1, same x,y — apply the foreman
fix lesson). Exits reciprocal as drawn.

| Room | Title | Coord | Mob | Role |
|------|-------|-------|-----|------|
| 6138 | Cross Street | {-5,-65,0} | — | Seam from Long Quay 6137 (north); the road into the civic district |
| 6139 | The North Cross | {-5,-66,0} | 9439 | North edge of the square; a town crier (ambient) |
| 6140 | Tri-Cross Square | {-5,-67,0} | 9440 | **The central plaza** — wide, old paving, a civic monument; a citizen (ambient); the Tri-Cross where the roads meet |
| 6141 | The South Cross | {-5,-68,0} | — | South edge of the square |
| 6153 | Processional Gate | {-5,-69,0} | — | South edge; **stub** toward the Processional/temple approach (district 4); NO south exit yet |
| 6145 | The Inn-Yard | {-4,-66,0} | — | The Three Waters' yard; stables, arriving travelers (east) |
| 6146 | The Three Waters | {-4,-67,0} | 9436 | **The great inn** common room; the Innkeeper; the social hub. **Q73 inn-remark breadcrumb** in dialogue. `up` to lodging |
| 6148 | The Three Waters — Lodging | {-4,-67,1} | — | Inn rooms upstairs (`down` to 6146; proper z+1 vertical) |
| 6147 | Inn Lane | {-4,-68,0} | — | The lane along the inn's south side |
| 6150 | The Civic Stores | {-3,-66,0} | 9437 | A general-goods shop (east); the Shopkeeper **vendor** (`general`) |
| 6152 | The Notary's Office | {-3,-68,0} | 9438 | A notary/scrivener of civic record (east; ambient or minor) |
| 6142 | The Hall Steps | {-6,-66,0} | — | Approach to the Municipal Hall (west) |
| 6143 | The Municipal Hall | {-6,-67,0} | 9434, 9435 | **The civic hall** (west); the **Hall-Clerk** (9434, civic) + **a Margin Scholar** (9435) studying **the margin-notation map** (the central Q73 breadcrumb — `look map` noun) |
| 6144 | Hall Lane | {-6,-68,0} | — | The lane along the hall's south side |
| 6151 | The Bell-Tower | {-7,-66,0} | — | The civic bell-tower (west); **orbital symbol on its pre-Founding base** (mystery; `look symbol`/`look base` noun) |
| 6149 | The Hall of Records | {-7,-67,0} | — | The records room (west of the hall); old charters and maps; more old-stone symbol detail |

Exit skeleton (build finalizes + cartchecks):
```
6138 n->6137(Long Quay) s->6139
6139 n->6138 s->6140 e->6145 w->6142
6140 n->6139 s->6141 e->6146 w->6143
6141 n->6140 s->6153 e->6147 w->6144
6153 n->6141   (south = stub to the Processional, NO exit)
6145 w->6139 s->6146 e->6150
6150 w->6145
6146 w->6140 n->6145 s->6147 up->6148
6148 down->6146
6147 w->6141 n->6146 e->6152
6152 w->6147
6142 e->6139 s->6143 w->6151
6151 e->6142
6143 e->6140 n->6142 s->6144 w->6149
6149 e->6143
6144 e->6141 n->6143
```

## NPCs (9434–9440)

All `non_combatant: true`, `hostile: false`, `charm_immune: true`,
`speciesid: 1`, `level: 1`, `maxwander: 0`, `statpool ~30`. Non-vendors:
`behavior_archetype: noncombat_passive`. **Unique names** (check roster + novel
cast). `groups: [humanoid]` (civic) EXCEPT the Scholar → `[humanoid, margin]`.
**Mob `character.name` MUST be Title Case** (the Long Quay porter casing panic) —
e.g. an ambient citizen is `A Confluence Citizen`, not `a confluence citizen`.

| Mob | Role | Room | Notes |
|-----|------|------|-------|
| 9434 | The Hall-Clerk | 6143 | Civic official of the Municipal Hall; the map, civic record, the city's business |
| 9435 | A Margin Scholar | 6143 | `groups: [humanoid, margin]`. Studying the notation map; the four-waters thread (the maps disagree on the count); **points the player to the Scholars' Quarter** (the Q73 giver's home). Continues Tallis's thread. **No quest token.** |
| 9436 | The Three Waters Innkeeper | 6146 | The great inn; gossip, lodging. **Q73 inn-remark breadcrumb**: an old traveler's tale / a remark about the maps and a fourth water (offhand, the inn's lore) |
| 9437 | The Shopkeeper | 6150 | **Vendor** (`craft_support: general`); civic everyday goods |
| 9438 | The Notary | 6152 | Civic scrivener/record-keeper; deeds, seals (ambient or short tree) |
| 9439 | A Town Crier | 6139 | Ambient (idlecommands crying the day's notices) |
| 9440 | A Confluence Citizen | 6140 | Ambient (idlecommands; the life of the square) |

## Economy — items 40133–40134

- **The Shopkeeper (9437):** a `general` civic store (accepts any category).
  Stock 1–2 new civic goods + reuse NP/Confluence goods (40102 rope, 40103
  waterskin, 40105 tinderbox, 40123 watercress).
- **Items (model on `40125-smoked_river_fish.yaml`):**
  | ID | Item | Vendor cat |
  |----|------|-----------|
  | 40133 | A tallow candle (civic everyday good) | `general`-acceptable real discipline, e.g. `alchemy` (or reuse an existing candle if one exists — check first) |
  | 40134 | A sheaf of writing-paper / a civic broadsheet | `tailoring` (paper/fiber) or `alchemy` — pick a valid `ValidCraftSupports` value |
  All `is_component: true`, a `component_tag`, **never `general` on the item**.
  (If a suitable item already exists in `materials-40000/`, reuse it and cut the
  new one — DRY.)

## Lore touches — the mystery escalates (restrained)

This is the **"it's everywhere" beat**, but kept environmental, not lectured
(the numerology steer):
- **The notation map (6143 Municipal Hall):** the central Q73 breadcrumb — a
  `look map` noun: a large civic chart of the rivers whose **margin carries an
  older annotation disagreeing on the count of waters** (a fourth channel noted
  and crossed out, or noted in a different hand). Clearly the same thread Tallis
  and the Scrivener's charts hinted at — here it's official and on the wall.
- **The orbital symbol on old civic stone:** the **Bell-Tower's pre-Founding
  base (6151)** and the **Hall of Records (6149)** carry the weathered nested-
  rings mark, plainly visible; the **newer civic facades** around the square show
  it **reinterpreted as Chrysalis motifs** (the cocoon/transformation reading).
  Give 2–3 `look` nouns across these rooms — observed plainly, no NPC explaining
  it. The Square is where a player realizes the mark is *in the city's bones*.
- **The Margin Scholar (9435)** carries the only explicit verbal thread, and
  points onward to the Scholars' Quarter. The Innkeeper's remark is folk-tale
  flavor, not analysis.
- **Tri-Cities flavor:** civic pride in "the Tri-Cities, three rivers and one
  city"; rivers consistent with canon (Aldren north, Brenn east, Solt southwest).
  The fourth is only the map's margin + the scholar's thread.

## Build approach

Standard content pipeline, branch `feature/confluence-tri-cross-square`:
1. Open the 6137 seam.
2. Rooms 6138–6153 (`city` biome; final coords + exits, cartchecked; the inn's
   `up`/z+1 lodging is a proper vertical). **Quote colon-space idlemessages.**
3. Items 40133–40134 (or reuse).
4. Mobs 9434–9440 + dialogue; the Shopkeeper vendor; the Scholar's `margin`
   group. **Title-Case all mob names.**
5. Smoke test: wipe instances, boot clean (no panics, `ValidateZoneConsistency
   errors=0 mode=panic`), `cartcheck the_confluence` clean, walk 6137→6138→…, buy
   from the Shopkeeper, hear the Innkeeper's remark + the Scholar's thread, `look
   map` at the Hall, `look symbol` at the Bell-Tower, go `up` to the inn lodging
   and back `down`, confirm 6153 south is barred.
6. Merge `--no-ff`; update `ZONE_EXPANSION.md` (Confluence 3/10).

World after this district: **40 zones / 1054 → 1070 rooms** (+16).

## Out of scope

- Districts 4–10 (south stub left unbuilt).
- Q73 itself (the Scholars' Quarter wires these breadcrumbs into the quest).
- Any mystery exposition beyond the map, the scholar's thread, and the architectural symbol.
