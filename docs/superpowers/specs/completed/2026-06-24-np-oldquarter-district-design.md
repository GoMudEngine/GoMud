# New Plymouth — District 7: The Old Quarter (design)

**Status:** approved 2026-06-24 (brainstorming). Build sequence item **#7 — the
FINAL district** of the capital (Docks ✅ → Common ✅ → Crafting ✅ → Merchant ✅ →
Temple ✅ → Noble ✅ → **Old Quarter**). Pulls anchors from the city-wide layer
(`2026-06-20-new-plymouth-citywide-design.md` §6 #43–45). Completing this district
triggers the **whole-capital pre-push SOP + push to prod** (§11).

> **PUSH POLICY (user, 2026-06-20):** Do NOT push New Plymouth to prod until the
> WHOLE capital is built and tests well. The Old Quarter is the last piece — after
> it merges and the capital boot-tests clean, the hold is released (§11).

## 0. Scope (locked with user 2026-06-24)

IN: rooms + mobs + dialogue + **faction MEMBERSHIP only (no new faction)** + the
**Bloom Trail CLIMAX resolved as discoverable lore** + the **pre-Founding lore web
CLOSED as ambient discovery** + **anchor schedules** + the **Gritta↔Coll / Gritta↔Orin**
cross-district relationship edges + the **Deren→Marn** supplier cross-reference.

OUT (deferred): **any quest** — the Bloom Trail pays off as LORE, not a quest
(quests are backfilled later, alongside the deferred **Bloom mechanic** —
[[project-bloom-mechanic]]). **No combat boss** — Deren is **watched/untouchable**
(the Horst model from Merchant); his doorman is a non-combatant menace. No
supply-runner integration (the destitute quarter has no craft vendors — §6). No
new mechanics, no forced respawn, no new faction YAML.

**Three locked user decisions (2026-06-24 brainstorming):**
1. **Bloom Trail payoff = lore-discovery only** ("3 for now; backfill quests later").
2. **Deren = watched/untouchable** (non-combatant, Horst model).
3. **Pre-Founding lore = ambient discovery** (no quest), mirroring how the Noble
   gallery cipher resolved as lore.

## 1. Zone, IDs, coordinates

- **Zone folder:** `new_plymouth_oldquarter` (underscores;
  `ConvertForFilename("New Plymouth Old Quarter")` → `new_plymouth_old_quarter`).
  **VERIFY the exact folder name from the loader's `Filepath()` at build time** —
  a mismatch panics at startup (CLAUDE.md "Data File Naming Convention"). The zone
  display name and folder must agree; pick the display name so the derived folder is
  clean (recommend display name **"New Plymouth Old Quarter"** → folder
  `new_plymouth_old_quarter`).
- **Rooms:** 6020–6039 (20). **Mobs:** 9379–9389 (3 anchors 9379–9381 + ~8 ambient
  9382–9389). **Dialogue:** by mobid. **Items/Quests:** none (see §6 re: an optional
  minimal scavenger stall — no new item ids; reuse only verified ids if built).
- **Coordinates — REAL unified frame, z = −1 (buried).** The whole quarter sits at
  **z−1**; it overlaps surface districts' x/y freely (different z = no collision).
  The **only** existing z−1 rooms in the unified frame are the Docks underdocks
  **5520 (−20,80,z−1)** and **5521 (−19,80,z−1)** — keep the Old Quarter's cells
  clear of those two. Place the quarter **west/south of the underdocks**, rough
  footprint **x[−27..−21], y[76..84], z−1**. Two interior structures descend to
  **z−2 sub-levels** (215 Lintel St's production room; Gritta's deep cellar/pre-
  Founding stonework) — collision-free beneath everything. **Solve exact coords
  cartcheck-clean at build time** (as Noble did); do not trust the citywide §2
  nominal region (x[−28..−8]/y[86..102] was nominal — the real underdock anchor is
  y80, so the quarter runs south/around it, NOT up at y86+).

Role (city-wide §2): *buried canal city; Bloom basement; pre-Founding lore.*

### Connections
- **Primary (only) entry:** Docks Underdocks **5520 → west → 6020** + reciprocal
  `east`. 5520 is already z−1 (−20,80); west → (−21,80,z−1) is free (5521 is EAST of
  5520, so extending WEST avoids it). **Light prose edit to 5520:** add a line that
  the bilge channel / a low passage continues west into the old canal (today 5520
  reads "The passage east continues… The stair is behind you" — append the west
  canal mouth). 5520's existing exits (up→5509, east→5521) and its 9318 spawn stay
  intact.
- **Cooperage cellar-mouth (Crafting 5721) stays BOARDED** — preserve the authored
  sealed dead-end ("sealed from this side when the cellar took water"). Do NOT open
  it. It remains flavor confirming the quarter runs north under the cooperage; no
  spatial exit (avoids a 12-room corridor and a second cross-zone seam).
- **z−2 sub-levels** (215 production room; Gritta's deep cellar) connect via
  vertical (`up`/`down`) exits only — always collision-free and pathto-traversable.
- Interiors compass/vertical only (gotcha #10 — never `enter`/`leave`).

## 2. Layout & connections

A descent from the grimy underdocks into the buried canal quarter — the poorest,
oldest part of the city, where Lintel Street runs beside a dead canal and the
pre-Founding stonework the colony paved over still stands in the dark. Clusters:

- **The canal mouth & Lower Lintel Street** — entry (6020) off the underdocks; the
  canal-side way begins; **the canon stone footbridge** the street ducks under
  (canon §1: "ducks under a stone footbridge, narrows").
- **Lintel Street** — the spine; narrows as it climbs away from the water; canal-side
  hovels and drowned courts (the poorest residents; ambient).
- **215 Lintel Street (the Bloom climax)** — the iron-banded door + a non-combatant
  doorman vestibule → the **seven steps down** → the stone corridor (two oil lamps)
  → the **production room** at z−2 (a discoverable crime scene: drain in the floor,
  a bare pallet, the collection apparatus — glass, copper tubing, wax-sealed clay
  pots; **the captive is gone**) and **Deren's ledger room** (the creaking 4th & 7th
  steps; Deren found here on his furtive schedule).
- **Quill's lamp-walk & hovel** — the canal-side lamps Quill tends; his hovel (the
  Bloom witness breadcrumb; np_commonfolk, poorest).
- **Gritta's flooded cellar & the pre-Founding stonework (the lore climax)** — a
  flooded stair down (z−2) to Gritta's cellar; beyond it the original pre-Founding
  stonework: massive lintels older than the colony, the **buried gray material**,
  and an original lintel carved with the **orbital / eight-pointed symbol** that
  **closes the Noble gallery cipher** (the buried city the whole lore web pointed
  at).

Suggested 20-room map (build solves exact coords/exits cartcheck-clean):

| Room | Title | z | Notes |
|------|-------|---|-------|
| 6020 | The Canal Mouth | −1 | entry off Underdocks 5520 (west); the bilge channel widens into a dead canal |
| 6021 | Lower Lintel Street | −1 | canal-side way begins |
| 6022 | The Stone Footbridge | −1 | canon: the street ducks under it |
| 6023 | Lintel Street | −1 | narrows |
| 6024 | Upper Lintel Street | −1 | toward 215 |
| 6025 | 215 Lintel Street — the Iron Door | −1 | doorman vestibule (9382); pattern-knock lore |
| 6026 | The Seven Steps | −1→−2 | down; creaking steps |
| 6027 | The Stone Corridor | −2 | two oil lamps, low ceiling |
| 6028 | The Production Room | −2 | **the crime scene** — drain, pallet, apparatus; captive gone |
| 6029 | Deren's Ledger Room | −1 | Deren (9379); creaking 4th & 7th steps; up off 6025/6026 |
| 6030 | Quill's Lamp-Walk | −1 | the canal-side lamps |
| 6031 | Quill's Hovel | −1 | Quill (9380); the Bloom witness |
| 6032 | A Drowned Court | −1 | poorest; ambient |
| 6033 | Canal-side Hovels | −1 | poorest; ambient |
| 6034 | The Flooded Stair | −1→−2 | down to Gritta |
| 6035 | Gritta's Flooded Cellar | −2 | Gritta (9381) |
| 6036 | The Pre-Founding Stonework | −2 | massive lintels older than the colony |
| 6037 | The Buried Lintel | −2 | the orbital symbol carved — **closes the gallery cipher** |
| 6038 | The Deep Canal | −2 | flooded dead-end; lore; the gray material at the waterline |
| 6039 | A Silted Gallery | −2 | pre-Founding; Gritta's find (the gray material at source) |

This is a guide, not a contract — the build may adjust titles/adjacency so long as
the clusters, the seam, the z-levels, and the two payoffs survive and cartcheck is
clean.

## 3. Anchors (3 — city-wide §6 #43–45), full life-sheets

Each: home + work room, ≥3 discoverable first-person dialogue topics, the unique
mutation in the description, faction via `groups:`, a 24h `schedule_id` (Stage D).
**No shops** on the anchors (the destitute quarter trades in whispers — see §6).

| # | Mob | Name | Mutation | Role | `groups` | `non_combatant` |
|---|-----|------|----------|------|----------|-----------------|
| 43 | 9379 | **Deren** | veins glowing faint copper (Bloom exposure) | the exposed Bloom supplier; 215 Lintel St ledger room; supplies Marn (Docks); **watched/untouchable** | humanoid, bloom_trade | **true** |
| 44 | 9380 | **Quill the Lamplighter** | night-adapted eyes | Old-Quarter lamplighter; canal-side hovel; **witnessed Deren's traffic** (Bloom breadcrumb); the poorest | humanoid, np_commonfolk | true |
| 45 | 9381 | **Gritta** | senses the buried gray material | pre-Founding relic-scavenger; the flooded cellar; **feeds fragments to Coll (Common) + Orin (Crafting)** | humanoid | true |

**Deren's dialogue (≥3 topics, watched/untouchable):** (1) the operation —
oblique, careful, a man who knows he's exposed and being watched; never confesses
outright but the room around him does (the apparatus, the ledger). (2) Marn — his
Docks contact / successor-vacuum (cross-ref the Bloom Trail: Marn 9305 already
references "his supplier"). (3) the captive — gone now; he speaks of it the way a
ledger speaks of a closed account; canon "surrendered the key." His mutation
(copper veins) visible. He cannot be attacked (`non_combatant`), stolen from, or
harmed — the watch and the bloodline domestic office both have eyes on the
production side (city-wide §6 #43: "Bloodline domestic office runs the captive-
production side"). A player who's followed the trail confronts him verbally and is
told, in effect, that nothing here is theirs to close.

**Quill's dialogue (≥3 topics):** (1) the lamps / his work (sees everything, says
less — "light the lamps, see nothing, say less"). (2) the traffic — what comes and
goes at 215 at odd hours (the Bloom witness breadcrumb; oblique, frightened-poor,
not a confession but a confirmation for a player who already has the address).
(3) the quarter / the poorest — the drowned courts, who lives down here. Night-
adapted eyes in the description.

**Gritta's dialogue (≥3 topics — the lore closer):** (1) the buried gray material —
what she senses in the deep stone (her mutation). (2) the pre-Founding city — the
lintels older than the colony; what the bloodline's Founding story overwrites
(closes the gallery cipher / Dross / Ept / Orin web — see §5). (3) her fragments —
she feeds finds to **Coll** (Common 9320) and **Orin** (Crafting 9332); name both
so the web is discoverable from her end. The orbital symbol (Noble gallery cipher,
Ept's orbital symbol, Orin's pre-survey maps) ties together in her cellar.

### Ambient + transient (~8, mobs 9382–9389, `non_combatant` unless noted)
- **9382 Deren's Doorman** — a heavy at the 215 iron door; immovable, watching;
  `non_combatant` (the watched/untouchable rule extends to his guard — no combat
  climax). He is pure menace/flavor — a presence that watches and disapproves but
  **does NOT block movement**: the player walks past him down the seven steps to the
  crime scene and reaches Deren's ledger room freely (the operation is already
  exposed; there is nothing left to guard). His dialogue/idle lines sell the unease,
  not a gate.
- **9383 A Bloom-Addled Wanderer** — reuse the Docks archetype (cf. 9317); the human
  cost of the trade; mutters Bloom-tinged lines.
- **9384 A Canal Beggar** — np_commonfolk poorest; ambient.
- **9385 A Mudlark / Relic-Picker** — works the silt for Gritta; ambient lore color.
- **9386 A Destitute Resident** — the drowned courts; ambient.
- **9387 A Furtive Runner** — `bloom_trade`-adjacent; moves quick, says nothing
  (the felt edge of the trade; non-combatant).
- **9388 A Canal-side Crone / Old Resident** — holds the quarter's memory; ambient.
- **9389 A Lamplighter's Boy** — helps Quill; ambient (pooled with Quill's beat).

Ambient cluster schedules may be pooled (a shared canal-folk pool) rather than
per-mob full life-sheets.

**Authoring gotchas (per-anchor + room):** YAML `": "` in a value → `" — "`;
Title-Case `name:` (casing.AssertCanonical panics); `ConvertForFilename` filenames;
faction via `groups:` (not a `faction:` field); noun ansi `fg="itemname"` always
(post-author grep `fg="[^"]* [^"]*"`); sentence-leading noun tags lowercase; avoid
prefix-shadowing dialogue triggers; interiors compass/vertical only.

## 4. Factions — membership only

No new faction YAML. All referenced factions already exist (created in earlier
districts): `bloom_trade` (Docks), `np_commonfolk` (Common). Membership via
`groups:`: Deren 9379 → `bloom_trade`; Quill 9380 → `np_commonfolk`; Gritta 9381 →
faction-neutral (the lore-web figure, not aligned). Ambient: the beggar/resident →
`np_commonfolk`; the furtive runner → `bloom_trade`. (Forward-ref to a nonexistent
faction PANICS — all here exist, so no risk; verify at build.)

## 5. The two payoffs — both discoverable lore

### 5a. The Bloom Trail CLIMAX (no quest, watched/untouchable)
This is the convergence of every Bloom breadcrumb placed across the capital:
- **Falk's Lintel Street pointer** (Merchant — `ask falk about property` → the old
  canal-district address).
- **Wenna's delivery-house** (Noble — the terrified servant's slip about the guarded
  estate where unmarked goods arrive).
- **Marn's supplier** (Docks — Marn the Draper fills Cade's vacuum, "supplied by
  Deren").
- **The Pilings Haunt** olfactory breadcrumb (Docks underdocks 5521 — the copper-
  flower smell, the tally-marks).

A player who has followed the trail arrives at **215 Lintel Street** and finds the
operation **exposed** (canon: "post-novel: exposed, the basement a discoverable
crime scene"). The payoff is **explorable, not a fight and not a quest**:
- The **iron door + doorman** (the canon pattern-knock entry; the doorman gates
  flavor, not the lore).
- The **seven steps → corridor → production room** (z−2): the drain, the pallet, the
  collection apparatus (glass, copper tubing, wax-sealed clay pots), the captive
  **gone** (Junie went south — canon). Room nouns tell the crime scene.
- **Deren** in his ledger room — `non_combatant`, watched/untouchable; the verbal
  confrontation closes the *narrative* trail (he's exposed but the bloodline office
  runs the production side and the watch has eyes on him — nothing here is the
  player's to close). **The Bloom *mechanic* is deferred** — the quest backfill comes
  later ([[project-bloom-mechanic]]); this build lays the placed content the future
  mechanic layers over.

### 5b. The pre-Founding lore web CLOSED (ambient)
Gritta's flooded cellar + the pre-Founding stonework **close** the lore web the
Noble gallery cipher opened:
- The **buried lintels** older than the colony; the **buried gray material**; an
  original lintel carved with the **orbital / eight-pointed symbol**.
- This is the buried city the **Noble gallery cipher** (Lysha vs Ferrol), **Dross**
  (Temple), **Ept** (Temple, the orbital symbol), and **Orin** (Crafting, the pre-
  survey maps) all pointed at. Gritta — who senses the gray material and feeds
  fragments to **Coll** (Common) and **Orin** (Crafting) — is the source. Her
  dialogue + the room nouns let a player who's gathered the threads see the whole
  picture: the bloodline's Founding narrative overwrites an older settlement.
- **No quest** — discoverable like the gallery cipher was in Noble.

## 6. Cross-district edges + supply (membership/relationship only)

- **Gritta↔Coll, Gritta↔Orin (lore-source relationship edges):** add `relationships:`
  to Gritta (9381) and backfill the partners — Coll (`9320-…`, Common) and Orin
  (`9332-…`, Crafting). Schema is a list of `{to: <mobid>, type, subtype}`. Use
  `type: friend` (or the closest valid type — VERIFY allowed relationship types/
  subtypes against the loader at build; do NOT invent a `type` that panics) with a
  descriptive subtype (e.g. `lore_source`). Add dialogue cross-refs both ways
  (Gritta names Coll + Orin; keep the Coll/Orin edits minimal — one line each).
- **Deren→Marn (supplier cross-ref):** dialogue only (and an optional `relationships:`
  edge `Deren → Marn 9305` if a sensible type exists). Marn already references "his
  supplier" — Deren's dialogue names the Docks end. Keep the Marn edit minimal/none.
- **NO supply-runner / Dobb changes.** The Old Quarter is destitute — **no craft
  vendors**, so it is NOT added to `CaravanServedZones` and Dobb's circuit/manifest
  is untouched. **Optional (low priority):** a single self-restocking scavenger
  curio-stall (e.g. Gritta or a fence selling a couple of low-value scavenged items)
  — if built, it uses the mob `shop:` template + a valid `craft_support:` tag
  (`general`), reuses only VERIFIED item ids, self-restocks (template default), and
  stays OUT of CaravanServedZones. Default = **no shop** unless it adds real flavor
  cheaply.

## 7. Anchor schedules (Stage D)

24h-contiguous schedules at `schedules/new_plymouth_old_quarter/` (validators PANIC
on coverage gaps or unreachable `pathto` targets — all targets must be built Old
Quarter rooms). Beats:
- **Quill** — **night-active** lamplighter (lights the canal lamps at dusk, walks
  the lamp-walk through the night, sleeps shallow by day in his hovel) — mirrors the
  Docks/Common night-active tavern rhythm.
- **Deren** — furtive: ledger room by day (the 215 interior), brief night movements;
  `activity: sleeping` overnight in a back room. Stays within the 215 cluster +
  Lower Lintel (his world is small and watched).
- **Gritta** — works the deep cellar / silted gallery (z−2) by day; surfaces to the
  flooded stair; sleeps in the cellar. Mostly stationary in the lore cluster.
- **Ambient pool** — a shared canal-folk pool (beggar/resident/mudlark) drifting the
  drowned courts + Lintel Street; the lamplighter's boy pooled with Quill.

## 8. Build staging — feeds ONE plan (each stage boot-verified)

> **Pre-smoke ritual** (CLAUDE.md SOP): wipe `mobs.instances/*` + `rooms.instances/*`
> before each boot (NOT `shops/`); boot-poll for `ERROR:.*PANIC` / `fatal error:`,
> NOT bare "panic" (gotcha #8 — `MapConsistencyEnforce value=panic` is a normal
> config line).

- **Stage A — the canal mouth + Lintel Street spine (rooms):** 6020–6024 + the
  footbridge; wire Underdocks **5520 → west → 6020** (+ reciprocal) and the light
  5520 prose edit. `cartcheck`-clean (the new z−1 cells must not collide with
  5520/5521).
- **Stage B — the clusters (rooms):** 215 Lintel St (6025–6029, incl. the z−2
  production room + ledger room), Quill's lamp-walk/hovel (6030–6031), the drowned
  courts/hovels (6032–6033), and Gritta's flooded cellar + pre-Founding stonework
  (6034–6039, z−2 — the orbital-symbol lintel). Boot-verify; vertical exits for all
  z transitions.
- **Stage C — population:** 3 anchors (Deren/Quill/Gritta) + ~8 ambient + dialogue +
  faction MEMBERSHIP (`groups:`) + room spawns. Deren + doorman `non_combatant`.
- **Stage D — the two lore payoffs + edges + schedules:** the Bloom crime-scene
  room nouns + Deren's confrontation dialogue (§5a); the pre-Founding lore nouns +
  Gritta's web dialogue (§5b); the Gritta↔Coll / Gritta↔Orin relationship edges +
  cross-ref dialogue (§6); the Deren→Marn cross-ref; anchor + pooled schedules (§7).
  Boot-verify (schedule + relationship loaders; cartcheck; the orbital-symbol lintel
  reads against the Noble gallery cipher).
- **Stage E — district harness playtest** (`/playtest local feature-tester`) →
  report → merge `--no-ff` (still hold push) → update memory.
- **Stage F — WHOLE-CAPITAL pre-push SOP + PUSH PROD** (§11) — the finale.

## 9. Definition of done (district)

- 20 rooms (6020–6039) boot-clean, `cartcheck` errors=0, the Underdocks→Old Quarter
  entry (5520→west→6020) works; all z−1/z−2 transitions via vertical exits.
- 3 anchors + ~8 ambient live; Deren + doorman `non_combatant` (un-attackable,
  verified); dialogue ≥3 topics each anchor.
- Faction membership correct (`bloom_trade` Deren/runner; `np_commonfolk`
  Quill/beggar/resident; Gritta neutral).
- **Bloom Trail climax** pays off as discoverable lore (215 Lintel St crime scene +
  Deren confrontation); the converged breadcrumbs (Falk/Wenna/Marn/Pilings) resolve
  here. The captive is gone (canon).
- **Pre-Founding web closes** (Gritta + the buried orbital-symbol lintel vs the Noble
  gallery cipher / Dross / Ept / Orin).
- Gritta↔Coll + Gritta↔Orin relationship edges + cross-ref dialogue live; Deren→Marn
  cross-ref live.
- Anchor + pooled schedules pass validators (Quill night-active).
- Harness-playtested; report committed. Merge `--no-ff` to master, hold push.

## 10. Honored gotchas checklist

#1 YAML `": "` in a value → `" — "` (esp. noun/desc values — caught a boot panic in
Noble) · #2 faction refs (membership only — all referenced factions exist) · #3
Title-Case `name:` · #4 faction via `groups:` · #5 no exits to unbuilt rooms (the
cooperage stays boarded; no Old-Quarter→elsewhere stubs) · #6 IF an optional shop is
built it needs a valid `craft_support:` tag · #7 `fg="itemname"` ansi (post-grep
`fg="[^"]* [^"]*"`) · #8 boot-poll `ERROR:.*PANIC` not bare "panic" · #10
cardinal/vertical interiors + REAL unified-frame coords (z−1/z−2; clear of
5520/5521) · sentence-leading noun tags lowercase · no prefix-shadowing triggers ·
only verified item ids (if a stall is built) · VERIFY the zone folder name from
`Filepath()` · VERIFY relationship `type`/`subtype` values against the loader before
authoring edges.

## 11. THE FINALE — whole-capital pre-push SOP + push prod

After the Old Quarter merges to master, the capital is **complete (7/7 districts)**
and the **push hold is released**. Execute the pre-push SOP (CLAUDE.md) against the
WHOLE accumulated bundle (~95+ commits ahead of prod):

1. **PATCH_NOTES.md** — add a dated entry for New Plymouth, the capital (all 7
   districts + the supply runner + the engine prereqs), describing the player-facing
   shape of the city. (A New Plymouth patch-notes entry already exists for the
   earlier districts — extend/complete it for the whole capital.)
2. **`_datafiles/config.yaml`** — confirm `Logging.LogToFile: false` (prod droplet
   disk).
3. **Full local boot test** — wipe instance saves, rebuild, boot, and confirm clean
   load past data-file loading: room count across **all 7 NP zones** + every other
   zone, mobs/quests loadedCount, **`ValidateZoneConsistency` errors=0** (note the
   mode), schedule/patrol/relationship validators, **no panics**. This is the only
   reliable check before prod (YAML load-time issues don't show in `go build`).
4. **Push `origin/master`** (the user does the droplet deploy). Append a datapoint to
   [[reference_prod_perf_baseline]] after the droplet restart (pull/restart time +
   idle CPU) — this is the largest content bundle yet (the whole capital), watch the
   restart time.
5. **Update memory** — mark the capital DONE; note the prod state; close the New
   Plymouth build project.

This is the end of the New Plymouth capital build.
