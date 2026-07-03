# Ferry System — Scheduled Water Travel + Economy Interconnection

**Date:** 2026-07-03 (warehouse-buffer scope expansion approved same day)
**Status:** Approved (Approach A + warehouse buffers, four stages)
**Prior art:** `docs/ZONE_EXPANSION.md` § "Water Routes & Ferries" (routes
reserved 2026-06-19); caravan import circuits (`internal/caravan/`); the
Threshold-Keeper paid-transport pattern (`open_instance_portal`).

## Goal

Real, scheduled ferry vessels on the three reserved water routes:

| Route id | Vessel (working name) | Connects |
|----------|----------------------|----------|
| `stillwater_np_packet` | The Lakewind Packet | Stillwater ↔ NP Docks |
| `stillwater_confluence_barge` | (downriver barge) | Stillwater ↔ The Confluence |
| `confluence_np_barge` | (river barge, novel canon — Davan rode it) | The Confluence ↔ NP Docks |

Both **players and NPC trade-factors ride the same vessels on the same
schedule**. The NPC cargo flow extends the caravan import-circuit
mechanism so regional goods cross cities, stabilizing prices (damping
the 0.25x–5x dynamic-pricing extremes) and opening player cargo-running
as an emergent activity. Fares are a recurring gold sink.

On top of the flows, **warehouse buffer pools** in New Plymouth and The
Confluence soak up surplus stock over time (like caravan wagons, at a
much bigger scale) and release it when shortages hit: a player buying
out Thornwall's crafting mats gets backfilled over the following ferry
and caravan runs from reserves that have been piling up for ages —
each hop following purely local rules.

**Design constraints carried over from ZONE_EXPANSION.md:** paid and
faster than walking; overland corridors stay relevant (low-coin players,
exploration, leveling); a short in-transit experience; never strictly
better than walking for a broke first-timer. No discovery gating — the
fare is the gate.

## Non-Goals

- Coastal/seaboard routes (NP harbormaster stub stays a stub).
- Combat, fishing, or events aboard vessels (flavor only for now).
- Unifying the legacy Thornwall caravan onto ferries (Stage 4 gives it
  a warehouse-aware load step at the NP end, nothing more).
- Player-owned boats; the existing Stillwater Boat Rental Pier prop is
  untouched.
- **Player access to warehouse stock — backend forever.** The
  warehouse interacts only with caravans, runners, ferry factors, and
  the delivery machinery. Enterable warehouse interiors, theft
  content, and bonded-goods quests remain a future seam (Warehouse
  Row's padlocked doors are the hook).
- Warehouse UI on the admin economy dashboard (logged as a follow-up;
  snapshot *capture* is in scope so history accumulates from day one).

## Staging

| Stage | Scope | Ships alone? |
|-------|-------|--------------|
| 1 | Ferry vessels + schedule + paid passenger travel | Yes |
| 2 | Riding trade factors + cross-region `RestockQty: 0` stock slots | Yes |
| 3 | Warehouses: persistent pools, overflow capture, ambient accrual, snapshot capture | Yes (reserves just build) |
| 4 | Drawdown: warehouse-first loading at depots, shortage-prioritized | Yes (completes the loop) |
| FU | Warehouse panel on the admin economy web dashboard | Follow-up |

Each stage is its own plan → build → playtest cycle. Stages 3–4 don't
touch vessel code, so they can land while Stages 1–2 soak on prod.

If the riding-factor seam proves flaky in Stage 2, the approved
fallback is direct port-call delivery (fire a delivery event at
arrival that restocks destination vendors) — same economics, cargo
just moves invisibly.

## Architecture

### New package: `internal/ferry`

Owns route definitions, the per-round controller, and boarding. No
imports from `internal/mobs` beyond what caravan already uses; reuse
`internal/caravan`'s vendor-visit helpers for deliveries.

### Route data: `_datafiles/world/dogmud/ferries/<routeid>.yaml`

```yaml
routeid: stillwater_np_packet
name: The Lakewind Packet
vessel:
  deck_room: <id>        # boarding/disembark room
  hold_room: <id>        # cabin/hold flavor room (optional)
ports:
  - dock_room: 4118      # Stillwater — Boat Rental Pier (confirm in plan)
    agent_mob: <id>      # dock agent NPC
    gangplank_name: gangplank
  - dock_room: 5502      # NP — The North Quay (confirm in plan)
    agent_mob: <id>
    gangplank_name: gangplank
schedule:
  crossing_hours: 6      # game-hours at sea (per leg)
  layover_hours: 2       # game-hours docked (per port)
  phase_offset_hours: 0  # stagger vessels so departures interleave
fare: 50                 # gold, charged at boarding
```

A startup validator (schedule/patrol rigor) panics on: unknown room
ids, unknown agent mob ids, vessel rooms shared between routes,
non-positive hours, or a routeid/filename mismatch.

### Vessel state machine (core)

Vessel state is a **pure function of the game clock** — no persisted
state, restart-safe, and agents can always quote exact departures.

```
cycle_rounds = 2 × (crossing + layover) × rounds_per_hour
phase(round) = (round + offset) % cycle_rounds
             → DOCKED_A | AT_SEA_A→B | DOCKED_B | AT_SEA_B→A
```

The controller ticks once per round and **reconciles** world state to
computed state (idempotent):

- **Docked:** ensure a temporary exit (`exit.TemporaryRoomExit`) named
  per `gangplank_name` exists **deck → dock** (free disembark). No
  dock → deck exit: boarding is agent-mediated only.
- **Transition to docked:** arrival emotes in dock room + deck.
- **Transition to at-sea:** remove gangplank, cast-off emotes; anyone
  aboard (players, NPCs, sleepers) rides.
- **At sea:** occasional ambiance messages on the vessel rooms (low
  per-round chance, same spirit as room ambient messages).
- Gangplank temp exits carry no expiry (`Expires` unset or long); the
  controller removes them explicitly at cast-off, and reconciliation
  re-adds them if anything else pruned them.

### Boarding & fares (Stage 1)

`ask <agent> passage` (plus `ferry`/`boat`/`passage` dialogue
triggers) at a dock:

- Vessel docked here → charge `fare` gold (refuse politely if the
  player can't pay), `rooms.MoveToRoom` the player onto the deck with
  gangplank-crossing flavor text (Threshold-Keeper gold-charge
  pattern, minus the instance machinery). Departure-time hint
  included ("She casts off within the hour.").
- Vessel not docked → quote the next arrival/departure using the
  clock math ("The packet's mid-crossing; she'll tie up around 4 PM.").
- Fare is charged per boarding; disembarking is free. Riding back
  costs a new fare (board via the far agent).
- Player-visible messaging follows the no-hard-numbers rule except
  gold amounts (prices are the existing exception, as in shops).

Dock rooms and agent dialogue advertise the service (discoverability
SOP: every trigger word appears in a hint or room description).

### Trade-factor circuits (Stage 2)

Per route and direction, a **ferry trade circuit** — an
`ImportCircuit` extension registered in `internal/ferry` (or a
generalized registry in `internal/caravan`):

```
FerryTradeCircuit {
  RouteId          string   // which vessel it rides
  HomePort         int      // dock_room index (0 or 1)
  FactorMobId      int      // the trade-factor NPC
  ExportManifest   []int    // items loaded at home depot
  DeliveryBuckets  []string // buckets delivered at destination
  CityPatrolId     string   // destination-side vendor loop
}
```

Lifecycle per round trip:

1. Factor idles at its home dock (loaded via
   `LoadRunnerFromImport`-style manifest fill at a depot stop).
2. Controller, during layover: if the route's factor is present in
   the dock room → `MoveToRoom` it aboard + emote ("The factor leads
   a laden mule up the gangplank."). The factor never paths through
   the gangplank itself — the controller teleports it, sidestepping
   patrol/temp-exit interplay.
3. On arrival: controller moves it ashore + emote; its city-side
   patrol (existing patrol plumbing) walks vendor waypoints,
   delivering via bucket-gated `VisitVendorsInRoom` with the existing
   delivery emotes.
4. Patrol ends back at the dock; factor waits; rides home; reloads.

Patrol executor suspends while the mob is aboard (no waypoint
progress at sea) and resumes at the next city-side waypoint on
disembark. Failure fallback is the existing
`ScheduleMaxPathRetries` → `pathto home` behavior; the factor's home
is its origin dock, so a lost factor self-recovers to the ferry loop.

Factors are `non_combatant: true` (like other watched/untouchable
NPCs) so cargo can't be farmed.

### Cross-region shop stock (Stage 2)

Destination-city vendors gain `StockEntry` slots for other regions'
bucket goods with **`RestockQty: 0`** — the local restock ticker
never fills them; only ferry deliveries do. Initial coverage:

- NP + Confluence vendors: selected `stillwater` bucket items.
- Stillwater + Confluence vendors: selected NP-side goods (`base` /
  `thornwall` items the import manifest already carries).
- NP ↔ Confluence: river goods per the `confluence_np_barge` leg.

Effects: out-of-stock 5x price spikes at the destination are damped
by the next boat; origin markets gain steady drain; buy-low/sell-high
cargo running by players emerges from existing dynamic pricing with
no new code. Exact item lists chosen in the plan phase from the
bucket map (`internal/economy/buckets.go`) — every delivered item
must already be bucketed, or `BucketFor` returns "" and the delivery
skips it.

Ferry factors appear in the economy dashboard via the existing
caravan snapshot plumbing (`CaravanSnapshot`), so deliveries-by-tier
and lbs-delivered trend like Dobb's circuit does today.

### Warehouse buffer pools (Stage 3)

A **warehouse** is a persistent, backend-only inventory pool per city
— New Plymouth and The Confluence to start. Storage mirrors shop
economic state: `_datafiles/world/dogmud/warehouses/<zone>.yaml`,
**living-economy state** — never wiped by the instance-save SOP,
never deployed over (same policy as `shops/`). Structurally a
`ShopInventory`-like ledger: per-item current + cap, with **very high
per-item caps** (config default, e.g. 50–100× a vendor MaxStock).
Physical presence is the existing Warehouse Row rooms (NP 5505,
Confluence 6113) — prose anchors only; no interiors, no player
interaction, ever.

Two fill feeds:

1. **Overflow capture.** Today, when a forager/caravan/ferry delivery
   finds vendors at `MaxStock`, the excess stays on the carrier. New
   rule: the undeliverable remainder at any delivery stop in a
   warehouse city banks into that city's warehouse (bucket-gated,
   capped). Real surplus is captured instead of circling on carriers.
2. **Ambient accrual.** A slow per-bucket trickle (config-knobbed
   rate, only for items already present in the warehouse's ledger or
   a seeded accrual list) representing off-screen trade, so reserves
   build through quiet stretches. Bounded by the caps.

Snapshot capture: a `WarehouseSnapshot` joins shops/caravans/foragers
in the hourly economy health snapshot from day one (stock by item +
bucket, capture/accrual/drawdown counters), so the follow-up
dashboard panel has history waiting for it.

### Warehouse drawdown (Stage 4)

Purely **local rules — no global demand solver**. When a registered
load event fires at a depot in a warehouse city (Dobb's circuit load
at 5506, a ferry factor loading exports, the legacy Thornwall-bound
caravan loading at the NP end), the loader:

1. Computes what's depleted downstream on *its own circuit* (missing
   stock across its delivery vendors — same visibility
   `VisitVendorsInRoom` already needs).
2. **Pulls from warehouse stock first**, decrementing the pool,
   prioritized by that shortage list, up to its `LoadCap`.
3. Tops up from its normal infinite-trickle import manifest, as
   today.

Multi-hop backfill emerges without coordination: *Confluence
warehouse → ferry factor → NP vendors (+ overflow into NP warehouse)
→ Thornwall-bound caravan load → Thornwall shops*. Backfill is
deliberately slow — one carrier-load per departure, no teleporting
stock. When warehouses are empty, behavior degrades exactly to
today's system.

## Content Inventory (built in plan phase, IDs via `id_inventory.py`)

- **Zone `ferries`** (`zone-config.yaml` required): 2 rooms × 3
  vessels ≈ 6 rooms. Named exits only (gangplanks are temporary
  exits), so the zone is exempt from Cartesian checks; set
  `non_cartesian: true` anyway for clarity.
- **3 dock agents** (Stillwater pier, NP quay, Confluence Barge
  Dock) — `non_combatant`, dialogue trees with `passage`/`ferry`
  triggers + schedule-aware refusals.
- **3 deckhand/bargehand NPCs** (one per vessel) — ambient flavor,
  idle chatter about the route; anchors the deck room.
- **2 trade factors per route** (Stage 2; 6 total) or start with the
  flagship route (Stillwater↔NP) and expand after the pattern proves
  out — plan-phase call.
- Dock-room prose touch-ups where the vessel berths (the berth should
  be visible in the room description).
- Existing prose seams to honor: Severin Pell (Hartcharn ferry agent,
  5409) sells the *idea* of the packet — his dialogue should now
  point at the real service; Tamsin Reed (Greywater ford) unaffected.

## Config Knobs (in `Balance` or a new `Ferries` config block)

Per-route YAML carries schedule + fare. Global knobs: ambiance
message chance, and a master `FerriesEnabled` bool for safe prod
rollout. Defaults (revised at plan time against RoundsPerDay=900, where
1 game-hour = 2.5 real minutes): crossing 2 game-hours (≈5 real min),
layover 1 game-hour (≈2.5 real min) — several round trips per game day;
fares 40–75g by route length. Tuned further in playtest.

Warehouse knobs (Stages 3–4): `WarehousesEnabled`,
`WarehouseItemCapMultiplier` (very high — default order of 50–100×
vendor MaxStock), `WarehouseAccrualPct`/interval (the ambient
trickle), and `WarehouseDrawdownEnabled` (Stage 4 kill switch,
separate from Stage 3 accumulation).

## Testing

- **Unit:** phase-from-clock math (boundaries, offsets, restart
  determinism); fare charge/refusal paths; validator panics; manifest
  load; circuit registry lookups.
- **Harness (Stage 1):** board at Stillwater, ride, arrive NP on
  schedule; agent refusal mid-crossing with a correct time quote;
  broke-player refusal; sleeper rides through.
- **Harness (Stage 2):** watch a factor complete a full circuit;
  verify destination vendor stock rose (admin `questtoken`-style
  direct checks preferred over flaky adapter observation); economy
  dashboard shows the factor.
- **Unit (Stage 3):** overflow banks the exact undeliverable
  remainder; accrual respects caps; persistence round-trips;
  snapshot fields populate.
- **Unit (Stage 4):** shortage prioritization; warehouse decrement +
  manifest top-up ordering; empty-warehouse degrades to today's
  load behavior.
- **Harness (Stages 3–4, the headline scenario):** buy out a
  Thornwall vendor's mats; verify over subsequent ferry + caravan
  runs that stock recovers faster than base restock alone, and that
  warehouse counters drew down in the snapshots.
- **Boot test:** full pre-push SOP; validator runs at startup.

## Risks

- **Patrol executor vs teleported mob (Stage 2, the one new engine
  seam):** mitigated by controller-teleport (never pathing through
  temp exits), suspend/resume semantics, and pathto-home recovery.
  Fallback: direct port-call delivery (approved).
- **Players stranded aboard** (e.g., linkdead through many
  crossings): benign — vessel always returns; disembark is free.
- **Instance-save shadowing:** vessel rooms and factors are ordinary
  templates; the standard instance-wipe SOP applies before smoke
  tests.
- **Economy skew:** cross-region slots start small (`MaxStock` low)
  so ferries season markets rather than flooding them; dashboard
  deltas make skew visible before it hurts.
- **Warehouse inflation:** very high caps + ambient accrual could
  make shortages too painless if the trickle outpaces demand. The
  accrual rate is the tuning lever; snapshot counters (accrued vs
  captured vs drawn) make the ratio observable. Drawdown ships
  behind its own kill switch.
- **Save-file growth:** one YAML per warehouse city, ledger-sized —
  negligible next to `shops/`.

## Open Items for the Plan Phase

- Confirm exact dock rooms (candidates: Stillwater 4118 Boat Rental
  Pier or 4116 Fishing Docks; NP 5502 North Quay or 5503 Long Quay;
  Confluence 6109 The Barge Dock).
- Designate load depots per port: NP has The Caravan Depot (5506);
  Stillwater and the Confluence need a designated dockside depot
  room (existing room double-duty vs a small new room — plan call).
- Reserve room/mob IDs via `python tools/id_inventory.py`.
- Pick exact fares + phase offsets; pick Stage 2 item lists from the
  bucket map; decide 1-route-first vs all-3 for factors.
- Verify `TemporaryRoomExit` visibility in the web mapper snapshot
  (should render as a portal/named exit; confirm no fog-of-war
  weirdness on vessel rooms).
- Stage 3: decide the seeded accrual list per warehouse (which items
  trickle when the ledger is empty) and initial cap values.
- Log the dashboard warehouse panel as a follow-up memory/backlog
  entry when Stage 3 ships.
