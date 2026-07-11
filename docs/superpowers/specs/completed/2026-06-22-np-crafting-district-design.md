# New Plymouth — District 3: The Crafting Quarter (design)

**Status:** approved 2026-06-22 (brainstorming). Build sequence item #3 of the
capital (Docks ✅ → Common ✅ → **Crafting** → Merchant → Temple → Noble → Old
Quarter). Pulls anchors from the city-wide design layer
(`2026-06-20-new-plymouth-citywide-design.md` §6 #15–21) and executes the
supply-runner wiring recipe from `2026-06-20-np-supply-pricing-fix-design.md` §2.

> **PUSH POLICY (user, 2026-06-20):** Do NOT push New Plymouth to prod until the
> WHOLE capital is built and tests well. This district accumulates on a branch /
> master locally. No mid-build prod push, no pre-push-SOP nagging.

## 0. Scope (locked with user 2026-06-22)

IN: rooms + mobs + dialogue + **a visible NP supply runner** + **Bloom Trail
middle** + **anchor schedules**. Activate the **`cooperage_circle`** faction.

> **SUPPLY DECISION (user, 2026-06-22) — "generalize engine now."** Investigation
> (§5) found the caravan engine is hardcoded to ONE crew (single consts
> `LeaderMobId=357`/`WagonMobId=374`/`RunnerMobId=359`; `FindRunnerInRoom`/
> `FindWagonInRoom` look up those exact template IDs; the circuit is triggered by
> Ketil's master wagon arriving at depot waypoints). The spec's inherited "~6
> lines" estimate was wrong. The user chose to **generalize the engine** to
> support a second, independent NP runner. This is a distinct subsystem → it gets
> **its own implementation plan** (`…-np-supply-runner.md`), separate from the
> district-content plan (`…-np-crafting-district.md`). See §5 for the architecture.

OUT (deferred): **a district quest** (user: "hold off on quests until we have the
rest in"); the Merchant/Old-Quarter/Common physical seams (stubbed); the
`bloodline_domestic` enmity edge on `cooperage_circle` (that faction does not
exist yet — wire the edge when Merchant/Noble builds create it).

## 1. Zone, IDs, coordinates

- **Zone folder:** `new_plymouth_crafting` (must use underscores;
  `ConvertForFilename("New Plymouth Crafting")` → that folder).
- **Rooms:** 5700–5724 (25). **Coords region:** x −24 … −8, y 100 … 114, z 0.
- **Mobs:** 9332+ (Docks took 9300–9318, Common 9319–9331).
- **Items:** 40102+ (block 40100–40299).
- **Dialogue:** 9326+ (next global free).
- **Buffs:** 90+ only if needed (likely none).
- **Quests:** none this build.

Role in the living city (city-wide §2): *artisan workplaces; the Inkwalk;
cooperage traces.* Sits on the **Long Market** (E–W supply spine) between the
Docks (west) and the future Central Square (Merchant, east).

## 2. Layout & connections

Four working clusters + a lore corner, all hung off the Long Market through-street:

- **The Long Market stretch** — the through-street; vendor stalls the mini-caravan
  runner delivers into. The spine the player walks east from the Docks.
- **The Inkwalk** — Orin's bookstall beside Edvar's **shuttered** cartographer's
  shop (the district's lore corner; pre-Founding / cooperage thread).
- **The hot trades** — Halvard's forge, Vesna's alchemy lab, Edda's glass kiln
  (the loud, smoky end).
- **The soft trades / Common seam** — Nessa's tailor shop, Corwin's tannery
  (south edge, prose-adjacent to Common's tannery streets).
- **The abandoned cooperage** — Toby's domain; a stubbed `down` stair to the
  future Old Quarter.

### Connections

- **Primary entry — the footbridge.** Docks **5523** (Marn's Fabric-Remnants
  Shop, x −18 / y 86) already carries prose of "a low stone footbridge to the
  north" but no `north` exit. Wire **5523 → north (long exit) → Crafting entry
  room** + reciprocal `south`. Thematically the bridge crosses the buried
  Old-Quarter canal. This is the player's route into the Crafting Quarter. The
  Crafting entry room sits at ~x −18 / y 100 (a long exit spans the y-gap; long
  exits are cartcheck-legal and render proportionally longer).
- **Deferred stubs** (gotcha #5 — never dangle an exit to an unbuilt room):
  - **East → Merchant** (Central Square) — Merchant unbuilt; prose-stub the
    eastward continuation of the Long Market, no exit.
  - **Cooperage `down` → Old Quarter** — OQ unbuilt; prose-stub the stair (a
    boarded/locked cellar mouth), no exit.
  - **Common seam (Corwin's tannery)** — Common is already shipped; **defer the
    reciprocal exit** to avoid retrofitting a live district and the unified-coord
    -frame collision risk (gotcha #10 coord-frame lesson). Author Corwin's
    tannery prose to acknowledge the adjacent Common tannery streets. The build
    MAY add one clean reciprocal exit opportunistically **iff** cartcheck shows
    the Crafting south-edge room and a Common north-edge room abut at adjacent
    coords with no collision in the unified frame; otherwise leave it prose-only.

All interior buildings use **compass/vertical exits only** — never `enter`/`leave`
(gotcha #10: bare `enter` is not a movement verb and enter/leave exits are not
`pathto`-traversable, so schedules can't route through them). Above-shop living
quarters = `up`; concealed cellars/back rooms = `down`.

## 3. Anchors (7 — city-wide §6 #15–21), full life-sheets

Each: a **home room** + a **work room** (or a single live-above-the-shop stack via
`up`/`down`), ≥3 dialogue topics, the unique mutation woven into the description,
faction via `groups:`, and (Stage E) a 24h `schedule_id`. Shop-bearing mobs need
a valid top-level `craft_support:` tag or `ValidateShopMobTags` panics (gotcha #6).

| # | Name | Mutation (unique) | Trade / shop | `craft_support` | Faction | Key threads |
|---|------|-------------------|--------------|-----------------|---------|-------------|
| 15 | **Master Halvard** | skin that won't scorch | blacksmith | `blacksmithing` | (artisan, neutral) | **brother of Marda (Common cookshop)** — daily cross-district midday meal; supplies Ostry (Merchant, later); iron from Dunmar's warehouse |
| 16 | **Toby the cooper's lad** | steam-reddened permanent hands | tends the abandoned cooperage (non-vendor) | — | `cooperage_circle` | knows the basement; the cooperage-lore key |
| 17 | **Vesna** | iridescent oil-sheen skin | alchemist | `alchemy` | (artisan, neutral) | supplies Ysolde (Common) **and, unknowingly, `bloom_trade`** ← Bloom Trail middle |
| 18 | **Edda Glass** | heat-scarred forearms, glass-clear fingernails | glassblower | `general` | `cooperage_circle` | knew Asha; the kiln-complex; home in Common → workshop here |
| 19 | **Orin the bookseller** | ink-stained eyes that read in the dark | bookstall by Edvar's shuttered shop | `general` | `cooperage_circle` | sells Edvar's last maps; deals with Temple scholars (later); the Edvar/Gritta lore web |
| 20 | **Corwin the tanner** | leather-tough skin | leatherworker | `tailoring` | (artisan, neutral) | hides from warehouse; Common-seam prose |
| 21 | **Nessa the tailor** | uncannily nimble fingers | tailor | `tailoring` | (artisan, neutral) | cloth from warehouse; supplies Aurel (Noble, later) |

Plus **~4–5 ambient mobs** (a smith's apprentice, a porter/runner, a street-sweep,
a customer or two) for ambient life — `non_combatant: true` townsfolk.

**Authoring gotchas (from Docks/Common stages):** (1) YAML prose with `": "`
inside noun/desc values breaks the parser — use `" — "`. (3) mob `name:` must be
Title-Case (`casing.AssertCanonical` panics). (4) faction membership is via mob
`groups:`, not a `faction:` field. (7) noun-highlight ansi MUST be
`<ansi fg="itemname">noun</ansi>` — never `fg="<noun>"` (breaks multi-word noun
render); post-author `grep -rE 'fg="[^"]* [^"]*"'` to catch leaks.

## 4. Faction — activate `cooperage_circle`

Define `_datafiles/world/dogmud/factions/cooperage_circle.yaml`:
- Default rep (outsiders): **+5**.
- **Allies/enemies: NONE this build** — `bloodline_domestic` does not exist yet,
  and faction forward-refs PANIC (gotcha #2). The `enemy: bloodline_domestic`
  edge is added by the Merchant/Noble build that creates that faction.
- Members (via `groups:`): Toby (16), Edda (18), Orin (19). The hot/soft-trade
  anchors (Halvard, Vesna, Corwin, Nessa) stay faction-neutral artisans.

## 5. The NP supply runner (caravan engine generalization)

**This is its own plan** (`…-np-supply-runner.md`). Architecture below; symbols
verified against the live tree via codegraph 2026-06-22.

### 5.1 Why the engine needs generalizing

The existing supply system (`internal/caravan`) is built around exactly ONE crew:
single constants `LeaderMobId=357` (Ketil), `WagonMobId=374` (cargo wagon),
`RunnerMobId=359` (Lars). `FindRunnerInRoom(roomId)` / `FindWagonInRoom(roomId)`
look up *those exact template IDs only*. The vendor-delivery circuit is started by
**Ketil's master wagon** arriving at a depot waypoint (`handleDepotArrival` →
`startRunnerCircuit` → `TransferCargoToRunner` → `StartOneshotPatrol`); Lars then
walks an `oneshot` patrol whose `caravan_vendor` waypoints fire `handleVendorArrival`
→ `VisitVendorsInRoom`. There is **no per-zone/per-circuit runner registry**.

NP's canon supply is different: imports arrive by **sea** → Dunmar's Docks
warehouse → a local runner (Dobb) distributes. **No traveling inter-zone wagon
arrives to trigger a circuit.** So we need a second, self-perpetuating runner.

### 5.2 Verified existing facts (codegraph 2026-06-22)

- `applyPatrolPlan` (`internal/hooks/NewRound_IdleMobs_patrol.go:100`) emits
  `events.PatrolWaypointArrival{MobInstanceId, PatrolId, WaypointIdx, RoomId,
  ArrivalEvent}` on arrival at **every** waypoint of **any** patrol — including an
  always-on (`patrol_id:`) **looping** patrol. (Confirmed: emits on both the
  zero-dwell `WantsAdvance` branch and the first-dwell-tick `WantsDwellWait` branch.)
- `CaravanArrivalListener` (`internal/caravan/arrival_listener.go:30`) dispatches
  on `ArrivalEvent` name; runner circuits are gated by membership in
  `runnerCircuitPatrols`. New `ArrivalEvent` tags it doesn't recognize are no-ops —
  so an additive NP path cannot regress the shipped Thornwall/Stillwater economy.
- `VisitVendorsInRoom(roomId, carrier, deliveryBuckets, pickupBuckets)`
  (`internal/caravan/visit.go:42`) delivers carrier→vendor for items whose
  `economy.BucketFor` ∈ deliveryBuckets and whose vendor `StockEntry.Current <
  MaxStock`; pickup is independent. Passing `pickupBuckets=nil` makes it
  **delivery-only** (what NP needs).
- Dobb (mob **9304**) and Dunmar (mob **9303**) already exist in the Docks zone.
  Dobb is non-combatant, `maxwander:0`, no patrol yet — purpose-built as the runner
  ("completing the full city circuit without once sitting down"; idle commands
  recite "today's delivery route stop by stop"). Dunmar has a shop + the
  warehouse-master role. **No new mobs needed.**

### 5.3 Design — a looping runner with a self-replenishing import source

Chosen over the wagon-triggered oneshot model because NP has no traveling wagon.

1. **Circuit registry** — new `internal/caravan/import_circuits.go`:
   ```go
   type ImportCircuit struct {
       PatrolId        string   // the runner's looping patrol id
       RunnerMobId     int      // who runs it (Dobb = 9304)
       DepotEvent      string   // ArrivalEvent tag at the load stop ("np_runner_depot")
       VendorEvent     string   // ArrivalEvent tag at delivery stops ("np_runner_vendor")
       DeliveryBuckets []string // {"base","overlap"} — existing crafter feedstock
       ImportItems     []int    // the sea-import manifest (item IDs Dobb loads)
       LoadCap         int      // max items Dobb carries per loop (bounded)
   }
   var importCircuits = map[string]ImportCircuit{ "np_docks_runner_circuit": {…} }
   ```
   Additive — the legacy `runnerCircuitPatrols` oneshot path is left untouched.
2. **Generalize the runner lookup** — add `FindMobByTemplateInRoom(roomId, mobId)`
   (or give `FindRunnerInRoom` a `mobId` param + update its single legacy caller to
   pass `RunnerMobId`). The NP path finds Dobb by `circuit.RunnerMobId`.
3. **Dispatch (additive branch in `CaravanArrivalListener`)** — if
   `arrival.PatrolId` ∈ `importCircuits`:
   - `ArrivalEvent == DepotEvent` → `LoadRunnerFromImport(runner, circuit)`
   - `ArrivalEvent == VendorEvent` → `VisitVendorsInRoom(roomId, runner,
     circuit.DeliveryBuckets, nil)` + room flavor.
4. **`LoadRunnerFromImport`** — new `internal/caravan/import_load.go`: top the
   runner's inventory up to `LoadCap` with fresh `items.New(id)` drawn round-robin
   from `circuit.ImportItems` (the **sea-import abstraction** — bounded, so no
   unbounded growth). On the next loop the leftover is overwritten/added up to cap.
5. **Dobb's looping patrol** — `patrols/new_plymouth/np_docks_runner_circuit.yaml`,
   `loop_shape: strict`. wp0 = the Docks depot room (Dunmar's warehouse;
   `arrival_event: np_runner_depot`, longer dwell = "loading"); wp1..N = each
   **Crafting vendor room** (`arrival_event: np_runner_vendor`, short dwell), then
   the strict loop returns to wp0. Assigned via `patrol_id: np_docks_runner_circuit`
   on Dobb's mob YAML (always-on patrol — auto-runs, no wagon trigger).
6. **Crafting vendors pre-declare deliverables** — each Crafting vendor's shop
   pre-declares the feedstock items Dobb delivers as `StockEntry{Current:0,
   MaxStock:<buffer>, RestockQty:0}`. `VisitVendorsInRoom` silently skips items
   with no existing entry; pricing is sane at `RestockQty:0` (fix already shipped).
7. **Config** — add `New Plymouth Crafting` (display name) to `CaravanServedZones`
   so its tier-30/20/10 slots are runner-served, not ticker-served. (Tier 50/40
   basics still trickle via baseline restock — acceptable.)
8. **Smoke-test** end-to-end: boot, wipe instances, watch Dobb walk the loop and
   the Crafting vendor `Current` counts climb after `np_runner_vendor` arrivals.

### 5.4 TDD seams (engine portion is unit-testable in isolation)

`LoadRunnerFromImport` (caps + round-robin), the dispatch branch (event routing by
tag), and `FindMobByTemplateInRoom` are all table-testable with synthetic
mobs/rooms in `internal/caravan/*_test.go`, mirroring the existing
`visit_test.go` / `cargo_handoff_test.go` patterns. The only manual step is the
in-game delivery smoke (§5.3.8), which depends on the district vendors existing.

*Merchant/Temple/Common vendors join the loop as those districts land (extend the
patrol YAML + their StockEntries); this build proves the Docks→Crafting leg.*

## 6. Bloom Trail middle (content-only)

Tightens the Docks → Old-Quarter thread without mechanics (the Bloom *mechanic*
is deferred — see `project_bloom_mechanic`). The beat: **Vesna unknowingly sells
a reagent up the chain that feeds `bloom_trade`.** Surface it as:
- A Vesna dialogue topic where she frets about a buyer who "pays too well and
  asks no questions" / a standing order she can't place.
- A breadcrumb pointing onward (toward the Old Quarter / Deren) — e.g. an addict
  Ysolde (Common) sent up to Vesna for a salve, or a ledger discrepancy noun in
  the lab. It **seeds, does not resolve**; the climax (Deren, 215 Lintel St) is
  the later Old-Quarter build.

This stays consistent with the Docks breadcrumbs already placed (Marn back room,
the Bloom-addled wanderer) and the city-wide Bloom Trail web (§7).

## 7. Anchor schedules (Stage E)

One `schedule_id` YAML per anchor at `schedules/new_plymouth_crafting/`,
covering all 24h (validators PANIC on coverage gaps or unreachable targets —
caught by the pre-push boot test). Pattern: home → work (`activity: working`
gates `TickMobCraft`) → **midday meal** → evening (tavern/social) → sleep
(`activity: sleeping`). Routing via compass/vertical exits only (gotcha #10).

**Signature beat:** **Halvard walks to the Common cookshop midday to eat with his
sister Marda** — a visible cross-district routine. The path Crafting → (footbridge
south to Docks) → Common cookshop must be fully `pathto`-traversable; validate the
whole route at boot. If the cross-district midday path proves unroutable with the
current seams, fall back to a Crafting-local meal node and keep the sibling bond
as dialogue (don't block the build on it).

Other anchors converge on a Crafting-local social/meal node (or the nearest
existing cookshop) midday/evening.

## 8. Build staging — TWO plans (each stage boot-verified before the next)

> **Pre-smoke ritual every time** (CLAUDE.md SOP): wipe instance saves —
> `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`
> — then restart. Do NOT wipe `shops/`. Boot-poll for `ERROR:.*PANIC` or
> `fatal error:`, NOT bare "panic" (gotcha #8: matches the config line).

This spec produces **two implementation plans** (per the writing-plans scope
check — two independent subsystems):

**Plan 1 — `…-np-crafting-district.md` (content; build first):**
- **Stage A — rooms 5700–5712:** Long Market stretch + the Inkwalk + the
  footbridge entry (wire Docks 5523 `north`). `cartcheck`-clean; update
  `docs/coordinate_map.md`. Boot-verify (`ValidateZoneConsistency errors=0`).
- **Stage B — rooms 5713–5724:** hot trades (forge/lab/kiln), soft trades
  (tailor/tannery), the abandoned cooperage (`down` stub). Boot-verify.
- **Stage C — population:** 7 anchors + ambient + dialogue + shops (vendors
  pre-declare deliverable `StockEntry`s per §5.3.6) + `cooperage_circle` faction
  + room spawns. Boot-verify mob count + faction count.
- **Stage D — Bloom Trail middle + anchor schedules:** §6 dialogue/nouns + §7
  schedule YAMLs. Boot-verify schedule validators. **District harness playtest**
  (`/playtest local feature-tester`) → report in `tools/playtest/reports/`.

**Plan 2 — `…-np-supply-runner.md` (engine generalization; build after Plan 1's
vendors exist):**
- **Stage 1 (engine, TDD, isolation):** circuit registry, `FindMobByTemplateInRoom`,
  `LoadRunnerFromImport`, the additive dispatch branch — unit-tested (§5.4).
- **Stage 2 (wiring, needs Plan 1 rooms):** Dobb's looping patrol YAML, the import
  manifest, `CaravanServedZones` config, Crafting vendor StockEntry confirmation.
- **Stage 3:** in-game delivery smoke-test (§5.3.8).

## 9. Definition of done

- 25 rooms (5700–5724) boot-clean, `cartcheck` errors=0, footbridge entry works.
- 7 anchors + ambient live, shops priced sanely, dialogue ≥3 topics each,
  `cooperage_circle` faction active.
- Supply runner demonstrably delivers Docks→Crafting (smoke evidence).
- Bloom Trail middle breadcrumb in place (Vesna).
- Anchor schedules pass validators; Halvard's midday meal routes (or graceful
  fallback documented).
- District harness-playtested; report committed.
- Merge to master (`--no-ff`), hold push per policy. Update
  `project_new_plymouth_build` memory + the build sequence.

## 10. Honored gotchas checklist (from prior districts)

#2 no faction forward-refs · #5 no exits to unbuilt rooms · #6 shop mobs need
`craft_support:` · #7 `fg="itemname"` ansi · #8 boot-poll `ERROR:.*PANIC` · #10
cardinal/vertical interiors + check the unified coord frame near seams · YAML
`": "` → `" — "` · Title-Case mob names · faction via `groups:`.
