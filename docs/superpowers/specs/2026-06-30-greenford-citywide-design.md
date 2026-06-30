# Greenford — City-Wide Design (zone 19)

*Spec date: 2026-06-30. Zone-expansion priority #19. The Eastern Arc's narrative
crux — the university town where Reth's testimony points to the crash site. This
is the **city-wide design layer**; each district is built from its own
spec → plan → subagent-driven-development, referencing this document (the
Confluence pattern).*

## Identity

A quiet university town on a river, on the **east bank** across the Greenford
bridge from the East Road (zone 18). Where old questions are studied *quietly* —
a scholarly backwater that escaped the attention that drove the Margin
underground at the Confluence. Calm, bookish, autumnal; a deliberate counterweight
to the Confluence's institutional weight. **Its job in the Eastern Arc: deliver
Reth's testimony — the directions to the crash site.** Aldric's route passes
through here (Confluence → east → Greenford → west/NW → New Plymouth).

## Scale, Structure & IDs

- **~45 rooms, one zone folder `greenford`**, built **district-by-district (5
  mini-stages)** — each its own spec → plan → subagent-driven build, world-critic
  + feel-test, merged `--no-ff`, unpushed (the user does the prod push).
- **Folder name MUST be `ConvertForFilename("Greenford")` = `greenford`.** (The
  East Road build cost a boot cycle on exactly this — verify the folder matches
  the zone display name.)
- **Planned ID blocks** (re-run `python tools/id_inventory.py` before EACH
  district build; these are the starting points, not reservations):
  - Rooms **6278+** (East Road ended at 6277; 6258–6262 is the newbie antechamber).
  - Mobs/dialogue **9501+**.
  - Items **40152+**.
  - Quests **75+** (Confluence used 73/74; East Road none).
  - Buffs 94+ only if needed.
- Region: **"The Tri-Rivers"** (extends the Confluence/River Road region).
  `zone-config.yaml`: name `Greenford`, defaultbiome (river/city per district),
  region `The Tri-Rivers`.

## Geography & Coordinate Frame

- **Entrance:** the East Road's barred **Greenford Bridge (6277)** `{x:22,y:-70,z:0}`
  (east-road coordinate frame; the bridge's onward way was prose-only/barred).
  Building Greenford **opens that gate** — District 1 wires `6277 → greenford`
  (the bridge crosses to the **east bank**; the town climbs a hill from the river
  up to the university). District 1's first room is east/south of 6277; the
  builder fixes exact coords and re-checks `cartcheck`/boot consistency.
- **Vertical layout (sense of climbing a hill):** river/bridge-landing at the
  bottom → town center mid-slope → **university up the hill** (its square tower is
  the landmark already visible from the East Road overlook 6273). Use z-steps or
  prose-grade elevation as the builder judges; coordinates stay Cartesian-clean.
- **West Road toward New Plymouth:** a described **stub** (Aldric's loop-closing
  route). That connecting road isn't built; the crash site is reached later from
  NP via Cascade Pass (#20), **not** from Greenford. Greenford gives the
  *directions*, not the route. Model the stub on the East Road's barred-bridge /
  River Road's barred-gate terminus.

## Faction

**Extend the existing `margin` faction into Greenford** — Brennan, Reth, and the
truth-seeking scholars are the Greenford node, operating *openly* (no Keepers
watching, unlike the Confluence). **No new faction file**; mobs join via
`groups: [humanoid, margin]`; quest rep flows through `margin`. The university as
an institution is unfactioned/generic. (The `factions/rep/<id>.yaml` runtime file
is gitignored — do not create it.)

## Key Cast (anchors — detailed per district at build time)

- **Brennan** — scholar of pre-Chrysalis history; knows Reth holds crash-site
  knowledge but can't extract it himself. Narrow house, **blue door**. Margin.
  The hub of the Surveyor's Report. (University District office + Neighborhood home.)
- **Reth** — retired surveyor who mapped the Eastern Highlands, filed an anomalous
  "mineral deposit" survey that says *nothing*, and retired early. Sparse cottage,
  **north end** of the Neighborhood. **He saw the hull.** The payoff NPC —
  reluctant, unsettled; gives directions + "it's not landscape." Margin-adjacent.
- Supporting: a **bookseller** (the "surveyor who retired early" breadcrumb), a
  **librarian/archivist** (the geological records = Reth's filed survey), the
  **Cartographer's Rest** innkeeper, faculty NPCs, townsfolk, river-folk.

## Quest Spine

### The Surveyor's Report (Q75) — anchoring multi-stage line
- **Three breadcrumbs** (the breadcrumb rule): Brennan mentions a "difficult
  source"; the university archive holds Reth's anomalous survey entry (the
  "mineral deposit" language that says nothing); the bookseller mentions "the
  surveyor who retired early."
- **Three resolution paths to Reth's testimony:** (a) approach Reth directly
  (needs an introduction or earned trust — he turns away cold callers); (b) find
  Reth's **original field notes** in the archive (a research path through the
  library/records); (c) earn **Brennan's trust** first and get his introduction.
- **Payoff = Reth's testimony:** verbal directions + landmarks (the
  lightning-split cairn, the route east into the highlands) + the **"it's not
  natural"** beat (exposed metal, no seam, not landscape — but NEVER what it is or
  what's inside). Physical takeaway: Reth's field notes / a marked map pointing
  toward the Eastern Highlands (`not_salable`), + `margin` rep. Sets up zones
  21–22 (the directions are the seam to the unbuilt endgame).
- **Mystery boundary (LOCKED):** directions + "it's not natural" only. The
  revelation (the fourth ship, the truth) stays for the crash-site interior
  (#22). The orbital symbol may recur in Brennan's old maps / the archive as
  continuity, **unexplained**. No numerology, no "fourth"-counting lecture.

### 1–2 supporting district quests (texture; may seed/gate the spine)
- Candidates (finalized per district at build): an **archive/library task** that
  unlocks or eases the records-search path to Reth's notes; a **town-or-scholar
  problem** (e.g. a scholar's heretical question, a lost/sought book, a
  river-district livelihood problem). Keep them small and self-contained; at
  most two. They should reinforce the scholarly feel and can plant breadcrumbs
  for the Surveyor's Report.

## District Build Order (each = own spec → plan → build)

1. **River District & Bridge Landing** — where the East Road bridge lands (opens
   6277→greenford); the river, a dock, a watermill, fishing; the climb up into
   town. The zone's entrance + the `margin`-free riverfolk texture. *(Build first.)*
2. **Town Center** — market square (small, quiet — contrast the Confluence's
   plaza), a **bookshop** (the "retired early" breadcrumb), **The Cartographer's
   Rest** inn, a general store; the university visible uphill. The civic hub.
3. **University District** — lecture halls, the **library** (public stacks /
   reading room / **restricted section** — a stub or lightly gated), **Brennan's
   office**, faculty NPCs, the **archive's geological records** (= Reth's filed
   survey, the research path's target). The scholarly + symbol-web heart; the
   Surveyor's Report's investigation rooms live here.
4. **Brennan's & Reth's Neighborhood** — residential; **Brennan's blue-door
   house**, **Reth's north-end cottage**, a tea house, gardens. **The quest
   payoff (Reth's testimony) lives here.** The most narratively loaded district.
5. **West Outskirts** — the **West Road toward NP** (loop-closing stub), a stable,
   a **farewell waypoint shrine**. The quiet outward edge (echoes the East Road /
   River Road terminus pattern).

## Build Conventions (carry into every Greenford district)

Same load-fatal rules proven across the Confluence + East Road builds:
1. Zone folder = `ConvertForFilename(zone name)` = `greenford`; every folder
   (rooms/mobs/dialogue/schedules) under that name. **Stage explicit pathspecs in
   git, never `git add -A`** (the repo carries dirty economy-snapshot noise that
   `-A` will sweep in).
2. `zone-config.yaml` required (name/roomid/defaultbiome/region) or boot panic.
3. Mob `name` + room `title` canonical Title-Case; mob filename =
   `ConvertForFilename(name)`; ambient archetype `noncombat_passive`.
4. `idlemessages`/idle lines with colon-space single-quoted; description/noun
   prose-colons in `>` block scalars.
5. Exits are `{roomid}` only — no `kind:` field (mapper-derives it).
6. Vendors: `craft_support` + `shop:`; items carry a real discipline, never
   `general`; non-vendor reward items need `not_salable: true`.
7. **Dialogue node-shadowing:** topic→node match is `strings.Contains(topic,
   trigger)` in file order — put gated/specific nodes first, avoid short triggers
   that substring-match other nodes' topics. (Bit us repeatedly; run the
   world-critic + a trigger-shadow check every district.)
8. **Quest dialogue SOP:** grant nodes need `grantsQuest` + `questExcluded`
   (incl. the end token); `questRequired`/`questExcluded` are LISTS; quest-giving
   nodes include `"quest"`/`"task"` triggers; `room_interact` nouns are
   ansi-highlighted HYPHENATED tokens with matching hyphenated noun keys.
9. Items 40xxx live in `items/materials-40000/` by id-range; filenames keep
   leading articles.
10. **Per district:** `id_inventory` first; nuke instance saves before smoke;
    clean boot (`ValidateZoneConsistency errors=0 mode=panic`); **world-critic +
    feel-test before merge** (the world-critic reliably catches river/compass
    botches + node-shadowing — run it every time).

## Validation per district

`id_inventory` → author → wipe instances → clean boot (errors=0 mode=panic, no
load panics) → `cartcheck greenford` clean → world-critic pass → harness
feel-test (walk + dialogue + any quest end-to-end) → update `docs/ZONE_EXPANSION.md`
row 19 + memory → merge `--no-ff`.

## Out of Scope (this zone)

- The actual crash-site route/zones (Cascade Pass #20, Eastern Highlands #21,
  Crash Site #22) — Greenford only points at them.
- Closing the NP loop (the full Greenford→NP west road) — stubbed only.
- The ferry mechanic.
