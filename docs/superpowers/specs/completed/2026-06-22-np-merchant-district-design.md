# New Plymouth — District 4: The Merchant Quarter (design)

**Status:** approved 2026-06-22 (brainstorming). Build sequence item #4 of the
capital (Docks ✅ → Common ✅ → Crafting ✅ → **Merchant** → Temple → Noble → Old
Quarter). Pulls anchors from the city-wide layer
(`2026-06-20-new-plymouth-citywide-design.md` §6 #22–28).

> **PUSH POLICY (user, 2026-06-20):** Do NOT push New Plymouth to prod until the
> WHOLE capital is built and tests well. Accumulate locally; no mid-build push.

## 0. Scope (locked with user 2026-06-22)

IN: rooms + mobs + dialogue + **the `bloodline_domestic` faction** (+ the
`cooperage_circle` enemy-edge backfill) + **anchor schedules** + **a supply-runner
extension** (Dobb → the Central Square) + a **Bloom Trail link** (Falk,
content-only). Activate the capital's commercial heart.

**Horst = WATCHED / UNTOUCHABLE (user decision).** He is present and ominous — a
**non-combatant** you observe, overhear (via Sephe's inn), and learn to resent —
but you cannot enter his house or confront him. The reckoning is a later
main-quest / Bloom-climax beat. Clerk Vell carries the *felt* bloodline reach
(tribute, checkpoints).

OUT (deferred): **a district quest** (no quest id consumed this build); the
Noble/Temple/Common physical seams (stubbed); the `bloodline_domestic` ↔
`temple_np` institutional **ally** edge (temple_np does not exist yet — wire it in
the Temple build); any Horst confrontation/combat.

## 1. Zone, IDs, coordinates

- **Zone folder:** `new_plymouth_merchant` (underscores;
  `ConvertForFilename("New Plymouth Merchant")`).
- **Rooms:** 5800–5824 (25). **Mobs:** 9344–93xx (Crafting took 9332–9343).
  **Dialogue:** files named by mobid. **Items:** 40106+ only if needed. **Quests:** none.
- **Coordinates — use the REAL unified frame, NOT the city-wide §2 nominal table.**
  (Crafting lesson: the built NP zones are far tighter than nominal.) Live z0
  footprint: Docks x[−20..−17]/y[79..86], Common x[−16..−12]/y[82..88], Crafting
  x[−22..−15]/y[87..92], Outskirts x[−19..−17]/y[72..78]. The Merchant Quarter
  occupies the **open ground EAST of Crafting / north-east of Common** (roughly
  x[−14..+2], y[88..100] — solved collision-free against the live cells at build
  time; `cartcheck` errors=0).

Role (city-wide §2): *the heart; great market; banks; Horst's base.* It is the
crossing of the two arteries — the Long Market (E–W) and the Processional (N–S).

### Connections
- **Primary entry:** Crafting **5704** (The Long Market — East End, at −16/88)
  already prose-stubs "east toward the Central Square / Merchant." Wire **5704 →
  east → Merchant** entry + reciprocal. This is the player's route in.
- **Deferred prose-stubs** (gotcha #5 — never dangle to unbuilt rooms): the
  Processional **north → Noble** and **south → Common** (Common is shipped — defer
  the reciprocal to avoid retrofitting it + unified-frame collision risk; the
  build MAY add one clean reciprocal iff coords abut trivially). East → Temple
  (Long Market continues) prose-stub.
- Interiors use **compass/vertical exits only** (gotcha #10).

## 2. Layout & connections

The **Central Square** is the hub (the great market, the artery crossing, the
surveillance heart); clusters hang off it:
- **The Central Square** (the great market) — Dobb's circuit passes visibly
  through it; a market crier, civic bustle, a watching guard.
- **The financial row** — Goss's Exchange (counting house) + Falk's auction house
  (the Bloom Trail link).
- **The arms row** — Dame Ostry's weapon shop + Brun's armory.
- **The Gilt Threshold** — Madam Sephe's high inn (the overhear-Horst node).
- **The bloodline's apparatus** — Clerk Vell's permit office + Horst's rented
  house (watched; a locked/guarded door you cannot pass — prose, no interior).

## 3. Anchors (7 — city-wide §6 #22–28), full life-sheets

Each: home + work room (live-above-shop `up`/`down` stacks where noted), ≥3
discoverable first-person dialogue topics, the unique mutation woven into the
description, faction via `groups:`, a 24h `schedule_id` (Stage D). Shop mobs need
a valid top-level `craft_support:` or `ValidateShopMobTags` panics.

| # | Mob | Name | Mutation | Trade / shop | `craft_support` | `groups` | Notes |
|---|-----|------|----------|--------------|-----------------|----------|-------|
| 22 | 9344 | **Horst** | none visible (uninfected) | bloodline handler — **NO shop**, **non_combatant** | — | humanoid, bloodline_domestic | watched/untouchable; runs agents; meets Sephe + Vell; the district's menace |
| 23 | 9345 | **Falk the Auctioneer** | never forgets a face or a price | auctioneer / fence | `general` | humanoid | high-end goods; property records → **Bloom Trail link** |
| 24 | 9346 | **Goss the Moneylender** | weighs gold true by touch | the Exchange | `general` | humanoid | holds debts on Common folk (cross-district tension; resented) |
| 25 | 9347 | **Dame Ostry** | an Extra Arm (visible mutation) | weapon dealer | `blacksmithing` | humanoid | blades from Halvard (Crafting) + warehouse |
| 26 | 9348 | **Brun the Armorer** | plated, callused skin | armorer | `blacksmithing` | humanoid | plate from warehouse + Crafting |
| 27 | 9349 | **Clerk Vell** | a seal-shaped birthmark on his palm | bloodline permit-clerk | — (office, may carry a minor `general` shop of permits/seals or none) | humanoid, bloodline_domestic | Horst's contact; collects tribute; the *felt* bloodline reach |
| 28 | 9350 | **Madam Sephe** | a subtle charming glamour | the Gilt Threshold keeper (inn) | `general` (or cooking) | humanoid | hosts merchants + Horst's quiet meetings (the overhear node) |

Plus **~4–5 ambient** (a market crier, a Square guard, a porter, a moneylender's
clerk, a wealthy customer) — `non_combatant` townsfolk (mobs 9351–9355).

**Authoring gotchas:** YAML `": "` → `" — "`; Title-Case names; filenames
`ConvertForFilename`; faction via `groups:`; noun ansi `fg="itemname"` only
(post-grep `fg="[^"]* [^"]*"`); interiors compass/vertical only.

## 4. Factions — create `bloodline_domestic`, backfill `cooperage_circle`

- **Create `_datafiles/world/dogmud/factions/bloodline_domestic.yaml`:**
  `default_rep: -10`; **enemies: [cooperage_circle]** (now exists — legal);
  **allies: []** (the `temple_np` institutional ally is a FORWARD-REF — temple_np
  is not built — so wire it in the Temple build, NOT here; gotcha #2).
  Members (via `groups:`): Horst (9344), Clerk Vell (9349).
- **Backfill `cooperage_circle.yaml`:** add **enemies: [bloodline_domestic]**.
  Both factions are loaded in THIS build, so the mutual reference resolves at the
  post-load validation (gotcha #2 only bites when referencing a faction that is
  not built at all). Boot-verify the faction-load count (16→17) + no panic.
- The merchant anchors (Falk, Goss, Ostry, Brun, Sephe) stay faction-neutral.

## 5. Supply-runner extension (Dobb → the Central Square)

Extend Dobb's existing `np_docks_runner_circuit` east **through the Central
Square** (the runner visibly crosses the great market — the heart of the moving
economy) to **Ostry (9347)** and **Brun (9348)**. Per the Plan-2 lessons
(`2026-06-22-np-supply-runner.md`):
- The runner delivers **only the raw materials Ostry/Brun stock** (e.g. steel
  ingot 40018, iron 40001, leather strip 40002, chain link 40019 — all already in
  the import manifest / base-overlap-thornwall buckets). Their **finished
  weapons/armor stay on normal ticker restock**, so they do NOT go into
  `CaravanServedZones` unless the manifest covers every item they sell (gotcha
  #2: served-zone vendors' every item must be deliverable or it starves). Default:
  **keep Merchant OUT of `CaravanServedZones`** — the runner's value here is the
  visible transit + topping up the materials when depleted (additive).
- **Wiring:** append the Central Square + Ostry/Brun vendor rooms to
  `patrols/new_plymouth/np_docks_runner_circuit.yaml` (depot wp0 stays zero-dwell;
  add `np_runner_vendor` stops). Extend `ImportItems` only if Ostry/Brun stock a
  material not already in the manifest. Boot-verify the patrol still loads + all
  waypoints reachable (the Crafting→Merchant seam via 5704 must be pathto-clean).
- Smoke-test: Dobb walks Docks → Crafting → Central Square; deplete an Ostry
  material and confirm a top-up (parked-player method, per Plan-2 gotcha #3:
  a runner only ticks when its area is player-active).

## 6. Bloom Trail link (content-only)

**Falk's property records** breadcrumb: Falk fences high-end goods and knows who
owns what. A discoverable dialogue topic (+ a `ledger`/`property-roll` noun in the
auction house) where he lets slip an **address discrepancy** — a Noble-quarter
delivery-house that changed hands quietly, or a canal-district property that
shouldn't be occupied — pointing toward the **Noble delivery-house / the Old
Quarter**. Advances the trail from Vesna's Crafting beat (the canal direction).
**Seeds, does not resolve**; the climax (Deren, Old Quarter) is the later build.
Consistent with the city-wide Bloom Trail web (§7: Wenna/Noble knows the delivery
house; Deren/Old Quarter is the source).

## 7. Anchor schedules (Stage D)

One `schedule_id` per anchor at `schedules/new_plymouth_merchant/`, 24h-contiguous
(validators panic on gaps/unreachable targets — boot-test catches these). Routing
compass/vertical only (gotcha #10). Beats: the **market opens** at the Square
(dawn → vendors at their stalls); the **Gilt Threshold fills** in the evening
(Sephe + merchant social, Horst's furtive meeting); **Clerk Vell's tribute rounds**
(office → the Square → back); **Horst** moves furtively between his house and the
inn (watched). Targets must be pathto-reachable within the built rooms.

## 8. Build staging — feeds ONE plan (each stage boot-verified)

> **Pre-smoke ritual** (CLAUDE.md SOP): wipe instance saves before each boot;
> boot-poll for `ERROR:.*PANIC` / `fatal error:`, not bare "panic" (gotcha #8).

- **Stage A — Central Square + arteries (rooms):** the Square + the Long Market
  entry from Crafting 5704 (wire 5704 `east`) + the artery stubs. `cartcheck`-clean.
- **Stage B — cluster rooms:** financial row, arms row, the Gilt Threshold, Vell's
  office, Horst's (watched) house exterior. Boot-verify.
- **Stage C — population:** 7 anchors + ambient + dialogue + shops +
  `bloodline_domestic` faction (+ `cooperage_circle` backfill) + room spawns.
- **Stage D — Bloom link + schedules + supply extension:** §5/§6/§7. Boot-verify
  schedule + patrol validators; supply smoke-test.
- **Stage E — district harness playtest** (`/playtest local feature-tester`) →
  report → merge `--no-ff` (hold push) → update memory.

## 9. Definition of done

- 25 rooms (5800–5824) boot-clean, `cartcheck` errors=0, the Crafting→Merchant
  entry works.
- 7 anchors + ambient live; Horst present + non-combatant + unconfrontale; shops
  priced sanely; dialogue ≥3 topics each.
- `bloodline_domestic` faction active + `cooperage_circle` mutual-enemy edge live.
- Dobb's circuit visibly reaches the Central Square; Ostry/Brun material top-up
  smoke-verified.
- Falk's Bloom Trail breadcrumb in place (points toward Noble/Old Quarter).
- Anchor schedules pass validators.
- Harness-playtested; report committed. Merge to master, hold push.

## 10. Honored gotchas checklist

#2 faction forward-refs (temple_np ally deferred; cooperage_circle↔bloodline_domestic
mutual is legal — both built here) · #5 no exits to unbuilt rooms · #6 shop mobs
need `craft_support:` · #7 `fg="itemname"` ansi · #8 boot-poll `ERROR:.*PANIC` ·
#10 cardinal/vertical interiors + REAL unified-frame coords (not nominal) · Plan-2
supply gotchas (zero-dwell load waypoint; CaravanServedZones-coverage; runner ticks
only when player-active) · YAML `": "` → `" — "` · Title-Case names.
