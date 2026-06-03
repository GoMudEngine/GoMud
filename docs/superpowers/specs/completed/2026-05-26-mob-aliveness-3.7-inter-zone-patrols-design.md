# Mob Aliveness 3.7 — Inter-Zone Patrols + Caravan Unification (Design)

**Status:** Approved (brainstorming) — ready for `writing-plans`
**Roadmap chunk:** 3.7 (Phase 3 — Routine layer)
**Size:** L
**Branch:** `feature/mob-aliveness-3.7-inter-zone-patrols`
**Depends on:** Chunk 3.4 (waypoint patrols — shipped)

## Goal

Two deliverables, weighted equally:

1. **Cross-zone patrol primitive.** Extend chunk 3.4's patrol layer
   to support routes that cross zone boundaries. Lift the
   single-zone restriction.
2. **Caravan unification.** Migrate the existing Thornwall ↔
   Stillwater caravan onto the unified patrol layer. The
   caravan's parallel movement state machine is deleted; only
   non-movement concerns (cargo, vendor trade, throughput,
   deliveries) remain in `internal/caravan/`.

Both ship in one chunk. Caravan is the proof-of-correctness for
the cross-zone primitive; the primitive is the foundation for
future zone-spanning NPCs (traveling merchants, pilgrim NPCs)
that are explicitly deferred to a later content pass.

## In scope

1. Patrol YAML waypoints gain an optional `arrival_event:
   <name>` field. Empty/missing → emit a no-op arrival event.
2. New event `events.PatrolWaypointArrival` carrying
   `{MobInstanceId, PatrolId, WaypointIdx, RoomId,
   ArrivalEvent}`. Emitted by the patrol executor at the
   transition from "moving" → "arrived" — when the mob first
   reaches a waypoint room and enters dwell.
3. Optional per-patrol `max_path_retries: <int>` field — falls
   back to global `ScheduleMaxPathRetries` when unset. Lets
   long-transit patrols (cross-zone caravan) author higher
   retry budgets without bumping the global default.
4. Loader validation: every waypoint room exists in the room
   registry. Zone-spanning patrols supported without any new
   schema (room IDs already globally unique). Loader logs each
   waypoint's resolved zone for human-readable boot-log
   sanity-checking.
5. Caravan migration:
   - Mob 357 (Ketil, caravan leader) gets `patrol_id:
     caravan_thornwall_stillwater`.
   - New patrol YAML at
     `_datafiles/world/dogmud/patrols/thornwall_outskirts/caravan_thornwall_stillwater.yaml`
     enumerating depots + Fernway pickup + vendor stops as
     waypoints with `arrival_event` markers.
   - `internal/caravan/routes.go` deleted (all room constants
     and Route structs move to the patrol YAML).
   - Caravan state machine driver in
     `internal/behaviortree/actions_caravan.go` deleted; the
     mob no longer needs a `caravan_state` BTreeState key
     written by a btree action.
   - `CaravanState` enum + `Name()` + `IsDwellState` /
     `IsTransitState` / `IsRouteState` / `IsFernwayPickupState`
     helpers kept in `internal/caravan/state.go` for dashboard
     reporting. `AdvanceState` + `ParseState` + `nameToState`
     deleted.
   - New arrival listener at
     `internal/caravan/arrival_listener.go` subscribes to
     `events.PatrolWaypointArrival`. Filters on the caravan
     leader mob (mob id 357) + `arrival_event ∈ {caravan_depot,
     caravan_vendor, caravan_fernway_pickup}`. Dispatches:
     - `caravan_vendor` → `VisitVendorsInRoom(roomId, wagon,
       deliveryBuckets, pickupBuckets)`.
     - `caravan_depot` → depot bookkeeping (state-name
       transition stamp; future hook for gold settlement).
     - `caravan_fernway_pickup` → forager handoff bookkeeping
       (extracted from the current Fernway-pickup btree action).
6. Crew model: leader-driven patrol; wagon + 4 guards remain a
   party that follows the leader via existing Stage 1 party
   primitives. No new follower machinery.
7. Leader-respawn crew regroup hook
   (`MobRespawn_CaravanCrewRegroup`): on leader respawn, force-
   move the wagon and any in-world guards to the leader's
   HomeRoomId (Thornwall depot). Caravan effectively "turns
   around and went back to base — try again next cycle" when
   the leader dies mid-route. Cargo on the wagon is preserved.
8. Dashboard preservation: new
   `caravan.SynthesizeStateForLeader(leader)` helper derives the
   canonical state-name string from the leader's patrol waypoint
   + arrival_event. `internal/economy/health/capture.go` swaps
   the BTreeState read for a call to this helper. JSON payload
   schema (`CaravanSnapshot`) is byte-identical; dashboard UI
   unchanged.
9. `caravan_state_started_round` stamp: written by the arrival
   listener whenever the synthesized state name flips. Lives in
   MiscData (numeric uint64) on the leader's `Character`, not
   in BTreeState.
10. Test updates: `internal/economy/health/capture_test.go`
    fixtures rewritten to use patrol-based state setup instead
    of `bs.Set("caravan_state", ...)`. New tests for
    `SynthesizeStateForLeader` covering each (waypoint,
    arrival_event, in-transit) combination.
11. Admin command `caravan reset` (in
    `internal/usercommands/admin.caravan.go`) and
    `behaviortree.ResetAllCaravanStates` / `ResetByInstanceId`
    helpers — semantics change to "reset the leader's patrol
    state to waypoint 0 and force-move leader + crew to
    Thornwall depot." Implementation surface shrinks (no
    BTreeState manipulation needed).

## Out of scope

- **Multi-stop caravan optimization** (best ordering of vendor
  stops). Author-fixed order only. Roadmap deferred.
- **Seasonal route variation.** Roadmap deferred.
- **Second consumer of the cross-zone primitive** — no pilgrim
  NPC, no second caravan. Roadmap explicitly defers to a
  content-authoring pass after the engine settles.
- **Runner-delivery flavor model** — the migration preserves
  the existing "wagon visits every vendor room" pattern as a
  faithful refactor. Resurrecting Ketil's son as a depot-to-
  vendor runner (the original caravan design intent, lost in
  the first implementation) is a separate post-3.7 flavor pass.
  Tracked in MEMORY.md → `caravan-runner-delivery-flavor`.
- **Multi-tag waypoints.** `arrival_event` is a single optional
  string. Set membership (`tags: [...]`) was considered and
  rejected as overkill with no current use case.
- **Explicit `zone:` field on waypoints.** Room IDs are
  globally unique; adding `zone:` would be redundant
  documentation that can drift. Loader logs resolved zones.
- **Wagon-death recovery.** Wagon is `non_combatant`; no
  current path to wagon death. If one emerges, handle
  separately.
- **Follower-stranding recovery beyond party-primitive
  retries.** Smoke-test risk; pre-emptive force-despawn-and-
  respawn-at-leader machinery deferred unless smoke surfaces
  real stranding.

## Architecture

### Data model — patrol YAML extensions

**Existing schema** (chunk 3.4):

```yaml
id: <kebab-case>
description: <string, optional>
loop_shape: strict | yo-yo
waypoints:
  - room: <int>
    dwell_rounds: <int, default 0>
```

**Extensions in 3.7:**

```yaml
id: <kebab-case>
description: <string, optional>
loop_shape: strict | yo-yo
max_path_retries: <int, optional>          # 3.7 new — falls back to ScheduleMaxPathRetries
waypoints:
  - room: <int>
    dwell_rounds: <int, default 0>
    arrival_event: <string, optional>      # 3.7 new — emitted on arrival; empty → no-op
```

`max_path_retries` is consumed in `patrolTickPlan` in place of
`int(configs.GetBalanceConfig().ScheduleMaxPathRetries)`. Zero
or missing → use the global config knob.

### Event payload

New event type in `internal/events/eventtypes.go`:

```go
// PatrolWaypointArrival fires once when a patrol-running mob
// reaches a waypoint room and enters the dwell phase. Consumers
// filter by ArrivalEvent name. Empty ArrivalEvent fires
// regardless — useful for debug subscribers but skipped by
// name-filtered consumers.
type PatrolWaypointArrival struct {
    MobInstanceId int
    PatrolId      string
    WaypointIdx   int
    RoomId        int
    ArrivalEvent  string
}
```

### Emission point

Inside `applyPatrolPlan` in
`internal/hooks/NewRound_IdleMobs_patrol.go`. The current
`WantsDwellWait` branch:

```go
case plan.WantsDwellWait:
    current := getMiscDataInt(&mob.Character, "patrol_dwell_remaining")
    if current > 0 {
        mob.Character.SetMiscData("patrol_dwell_remaining", current-1)
    }
    mob.Character.SetMiscData("patrol_path_fail_count", 0)
    return
```

is extended to detect the first-tick-of-arrival transition by
comparing `patrol_dwell_remaining` to the waypoint's authored
`dwell_rounds`. When they match (meaning this is the tick we
just arrived, before any decrement), emit the event:

```go
case plan.WantsDwellWait:
    current := getMiscDataInt(&mob.Character, "patrol_dwell_remaining")
    if current == p.Waypoints[idx].DwellRounds {
        // First-tick arrival — emit the per-waypoint event.
        events.AddToQueue(events.PatrolWaypointArrival{
            MobInstanceId: mob.InstanceId,
            PatrolId:      activePatrolId,
            WaypointIdx:   idx,
            RoomId:        mob.Character.RoomId,
            ArrivalEvent:  p.Waypoints[idx].ArrivalEvent,
        })
    }
    if current > 0 {
        mob.Character.SetMiscData("patrol_dwell_remaining", current-1)
    }
    mob.Character.SetMiscData("patrol_path_fail_count", 0)
    return
```

Idempotent: the event fires exactly once per waypoint visit
because the dwell counter monotonically decrements after the
first match.

**Zero-dwell waypoint case.** When `dwell_rounds: 0`, the
patrol executor short-circuits `WantsDwellWait` and proceeds
directly to `WantsAdvance` on the same tick the mob arrives
in the waypoint room. To preserve emission semantics ("fires
exactly once per waypoint visit, before any state advance"),
the same event-emission block is mirrored in the
`WantsAdvance` branch with a guard: emit only when
`mob.Character.RoomId == p.Waypoints[idx].Room` (i.e. the
mob is currently at the waypoint, not transitioning toward
it from a path-walker step). The caravan patrol authored in
this chunk has no zero-dwell waypoints, so this path is
covered for correctness/future-use; smoke testing focuses
on the first-dwell-tick path.

### Caravan code surface — what stays, what goes

`internal/caravan/`:

| File | Disposition |
|---|---|
| `routes.go` | **Deleted.** Depot rooms, vendor room lists, Route structs all move to the patrol YAML. |
| `state.go` | Shrinks. Keep `CaravanState` enum, `Name()`, four `Is*State` predicates. Delete `AdvanceState`, `ParseState`, `nameToState`, `RouteForState`. |
| `visit.go` | Unchanged (`VisitVendorsInRoom`, `FormatVisitMessage`). |
| `wagon.go` | Unchanged (`IsCaravanMob`, `FindWagonInRoom`, `WagonMobId`, `LeaderMobId`). |
| `throughput.go` | Unchanged (cargo throughput accounting). |
| **new** `arrival_listener.go` | Subscribes to `events.PatrolWaypointArrival`, dispatches based on `ArrivalEvent`. |
| **new** `synthesize_state.go` | `SynthesizeStateForLeader(leader *mobs.Mob) (CaravanState, bool)` — derives dashboard state name from patrol waypoint index + arrival_event. |
| **new** `respawn_regroup.go` | `MobRespawn_CaravanCrewRegroup` hook — force-moves wagon + guards to leader's HomeRoomId when leader respawns. |

`internal/behaviortree/`:

| File | Disposition |
|---|---|
| `actions_caravan.go` | **Deleted** (state-machine driver). |
| `actions_caravan_test.go` | **Deleted**. |
| `actions_wagon.go` | **Deleted** if all of its actions are wagon-as-follower movement helpers (likely the case — wagon is `non_combatant` and has no unique non-movement behavior). Implementation plan re-checks each action's call sites; anything not movement-related (e.g. cargo accounting helpers) gets moved to `internal/caravan/`. |
| `actions_wagon_test.go` | Matches `actions_wagon.go` disposition. |
| `caravan_reset.go` | Rewritten — instead of mutating `BTreeState["caravan_state"]`, resets the leader's `patrol_waypoint_idx` to 0 and force-moves the crew to Thornwall depot. |

`internal/economy/health/capture.go:163-185` (the
`captureCaravans` helper):

```go
// Before — reads BTreeState
bs, ok := mob.BTreeState.(*behaviortree.BehaviorState)
if !ok || bs == nil { continue }
stateName := bs.GetString("caravan_state")
if stateName == "" { continue }
startedRound, _ := strconv.ParseUint(
    bs.GetString("caravan_state_started_round"), 10, 64)

// After — uses synthesized state
state, ok := caravan.SynthesizeStateForLeader(mob)
if !ok { continue }
stateName := state.Name()
startedRound, _ := mob.Character.GetMiscData(
    "caravan_state_started_round").(uint64)
```

### Caravan patrol YAML

`_datafiles/world/dogmud/patrols/thornwall_outskirts/caravan_thornwall_stillwater.yaml`:

```yaml
id: caravan_thornwall_stillwater
description: "Ketil's caravan crew: Thornwall depot → Stillwater vendors → back."
loop_shape: strict
max_path_retries: 40   # cross-zone transits need more headroom than the 20 default
waypoints:
  # ── Thornwall depot: long dwell before next departure ──────
  - room: 465
    dwell_rounds: 360
    arrival_event: caravan_depot

  # ── Outbound: Fernway pickup (forager handoff) ─────────────
  - room: 4038
    dwell_rounds: 8
    arrival_event: caravan_fernway_pickup

  # ── Stillwater arrival depot ───────────────────────────────
  - room: 4109
    dwell_rounds: 20
    arrival_event: caravan_depot

  # ── Stillwater vendor circuit ──────────────────────────────
  - { room: 4102, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 4103, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 4105, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 4106, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 4125, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 4126, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 4135, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 4143, dwell_rounds: 5, arrival_event: caravan_vendor }

  # ── Stillwater departure depot ─────────────────────────────
  - room: 4109
    dwell_rounds: 20
    arrival_event: caravan_depot

  # ── Inbound: Fernway pickup ────────────────────────────────
  - room: 4038
    dwell_rounds: 8
    arrival_event: caravan_fernway_pickup

  # ── Thornwall arrival depot ────────────────────────────────
  - room: 465
    dwell_rounds: 20
    arrival_event: caravan_depot

  # ── Thornwall vendor circuit ───────────────────────────────
  - { room: 464, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 470, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 471, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 475, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 480, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 481, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 482, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 483, dwell_rounds: 5, arrival_event: caravan_vendor }
  # loops back to wp0 (Thornwall depot, 360-round dwell)
```

Strict loop. The depot rooms appear twice (arrival + departure)
on each side — intentional, mirrors the current state-machine
phasing. Fernway pickup (room 4038) appears twice (outbound +
inbound).

Total round-trip cycle ≈ 360 + 8 + 20 + 8×5 + 20 + 8 + 20 + 8×5
+ transit-rounds ≈ ~940 + walk time. Matches current cadence.

### Contingency design

**1. Leader enters combat mid-transit.**
Existing 3.4 plumbing. `IsInCombat()` check at the top of the
idle/patrol/schedule branch short-circuits. Combat ends →
patrol resumes the next tick, re-pathing to the current
waypoint room if combat moved the leader.

**2. Leader dies mid-transit.**
- Patrol MiscData on the leader instance is destroyed with the
  instance. On respawn at HomeRoomId (Thornwall depot 465),
  patrol restarts at waypoint 0 (`patrol_waypoint_idx` MiscData
  unset).
- **New hook `MobRespawn_CaravanCrewRegroup`** force-moves the
  wagon + any in-world surviving guards to the leader's
  HomeRoomId on leader respawn. The caravan re-anchors at
  Thornwall regardless of where it died. Cargo on the wagon is
  preserved.
- Net behavior: leader death = caravan turns around, returns
  to base, restarts next cycle.

**3. Wagon dies.**
Wagon is `non_combatant`. No current path to wagon death.
Tracked as a known risk; no pre-emptive code.

**4. Guard dies.**
Crew continues with surviving guards. Dead guard respawns at
its HomeRoomId via standard mob respawn machinery; party-follow
re-attaches on next idle tick. No new code.

**5. Crew member left behind in transit.**
Existing party-primitive plumbing handles re-anchoring. If a
follower's pathto-leader exceeds party-follow's max-recovery
distance, the follower may be stranded. Documented as a
smoke-test risk; force-despawn-and-respawn-at-leader recovery
deferred unless smoke surfaces real stranding.

**6. Path-retry exhaustion on long transits.**
`max_path_retries: 40` on the caravan patrol gives ~2x the
default budget. If exhausted, the patrol executor's
home-fallback fires (leader walks back to Thornwall depot —
already correct for this caravan).

**7. Server restart mid-transit.**
Patrol MiscData on the leader instance + wagon cargo on the
wagon instance both persist in instance YAMLs. On restart, the
leader resumes at the saved `patrol_waypoint_idx`; crew
re-attaches via party-follow on first tick.

**8. Fernway pickup (forager handoff).**
Room 4038 becomes a waypoint with
`arrival_event: caravan_fernway_pickup` and 8-round dwell.
Caravan listener handles forager-handoff bookkeeping on the
event. Minor behavior change vs. the current state machine:
dwell is fixed at 8 rounds rather than "leave early if no
forager present." Acceptable simplification.

### Boot-time validation

The patrol loader (chunk 3.4) already validates `room` exists
in the room registry. 3.7 additions:

- Each waypoint's room is resolved to a zone for boot-log
  visibility (info log: `patrol=<id> waypoint=<idx>
  room=<id> zone=<name>`).
- `arrival_event` strings are not validated against any enum —
  consumers do their own filtering. Free-form by design.
- `max_path_retries` must be ≥ 0 when set. Zero or unset →
  global default.
- Mob-side cross-check (chunk 3.4 carried over): every
  `patrol_id` reference on a mob template must resolve to a
  loaded patrol; boot panic on miss.

### Testing strategy

**Unit:**
- `caravan.SynthesizeStateForLeader` — table-driven, one case
  per (waypoint range, arrival_event, in-transit-flag)
  combination. Verify each maps to the right `CaravanState`.
- `economy/health/capture_test.go` fixtures rewritten:
  `bs.Set("caravan_state", ...)` → `mob.PatrolId =
  "caravan_thornwall_stillwater"; mob.Character.SetMiscData(
  "patrol_waypoint_idx", N)` + register a stub patrol via
  `mobs.RegisterPatrolForTest`.
- New `TestPatrolWaypointArrival_EmittedOnceOnArrival` in
  hooks-package: register a stub patrol, run multiple ticks,
  assert the event fires on the first dwell tick and not
  subsequent ones.
- New `TestApplyPatrolPlan_RespectsPerPatrolMaxRetries`:
  patrol with `max_path_retries: 40`; assert home-fallback
  fires at 40, not 20.
- `TestSynthesizeStateForLeader_NonCaravanMobReturnsFalse`:
  mob without `patrol_id` returns `(_, false)`.

**Integration:**
- Existing `internal/caravan/visit_test.go` and
  `state_test.go` — most pass unchanged (visit logic and
  remaining state helpers don't depend on the state-machine
  driver). Tests for `AdvanceState` / `ParseState` /
  `RouteForState` are deleted (those functions are deleted).
- `internal/behaviortree/actions_caravan_test.go` —
  **deleted entirely** (the actions it tests are deleted).
- `internal/behaviortree/actions_wagon_test.go` — audit;
  keep whatever still applies.

**Smoke test (manual, in-game):**

1. Boot server. Verify `mobs.LoadPatrols()` reports
   `loadedCount=2` (existing market beat + new caravan).
2. `mob schedule <ketil-instId>` shows patrol state with
   `caravan_thornwall_stillwater` waypoints listed.
3. Watch a full Thornwall → Stillwater leg:
   - Leader queues a `pathto 4109` (or first transit step)
     when wp0 dwell expires.
   - Wagon + guards follow via party plumbing.
   - At Fernway pickup (room 4038), caravan dwells 8 rounds.
4. Watch a Stillwater vendor cycle:
   - At each vendor waypoint, watch for
     `<ansi fg="yellow">The caravan crew unloads supplies for
     the local merchants.</ansi>` (or trade variant) in the
     room.
   - Verify vendor stock counts change via the economy
     dashboard.
5. `/admin/economy/` dashboard:
   - Caravan card shows the right state name throughout the
     cycle (`thornwall_dwell`, `outbound_transit`,
     `outbound_fernway_pickup`, `stillwater_dwell`,
     `stillwater_route`, etc.).
   - `state_entered_round` advances when state name flips.
   - `cargo_weight` / `cargo_by_bucket` / `deliveries_by_tier`
     all update.
6. Kill the leader mid-Stillwater-vendor-circuit:
   - Verify wagon and surviving guards force-move to
     Thornwall depot on leader respawn.
   - Cargo preserved (check `cargo_weight` doesn't reset).
   - Caravan restarts cycle from waypoint 0.
7. `caravan reset` admin command:
   - Resets leader to wp0 and force-moves crew to Thornwall
     depot.
8. Restart server mid-Stillwater-cycle:
   - Verify leader resumes at saved `patrol_waypoint_idx`,
     crew re-attaches.

### Rollout order

Two atomic phases. The caravan flip is **one shot**: there is
no intermediate dual-driver state, because both the legacy
state machine and the new arrival listener would call
`VisitVendorsInRoom` at the same waypoints, double-restocking
every vendor on every visit.

1. **Phase 1 — generic patrol primitive (no caravan effect).**
   - Patrol YAML schema: optional `arrival_event` +
     `max_path_retries` fields, parser support, loader log of
     resolved zones.
   - Event type `events.PatrolWaypointArrival` defined and
     emitted by the patrol executor (both first-dwell-tick
     and zero-dwell `WantsAdvance` paths).
   - `caravan.SynthesizeStateForLeader` helper added but not
     yet wired in.
   - Unit tests for the new helper + emission idempotency
     pass.
   - The existing Thornwall market-beat patrol still works
     unchanged (no `arrival_event` → no-op emission).
   - The legacy caravan state machine still drives the
     caravan, unchanged. Dashboard reads still hit
     `BTreeState["caravan_state"]`.
2. **Phase 2 — caravan flip (single atomic change).**
   - Author the new caravan patrol YAML.
   - Add `patrol_id: caravan_thornwall_stillwater` to mob
     357.
   - Add `internal/caravan/arrival_listener.go` (subscribes
     to `PatrolWaypointArrival`, dispatches restock /
     forager-handoff / depot bookkeeping).
   - Add `MobRespawn_CaravanCrewRegroup`.
   - Swap `economy/health/capture.go` to call
     `SynthesizeStateForLeader`.
   - Rewrite `caravan_reset.go` for the patrol-based reset.
   - Delete `internal/caravan/routes.go`.
   - Delete `internal/behaviortree/actions_caravan.go` +
     `actions_caravan_test.go`.
   - Delete obsolete bits of `internal/caravan/state.go`
     (`AdvanceState`, `ParseState`, `nameToState`,
     `RouteForState`).
   - Update `economy/health/capture_test.go` fixtures.
3. **Boot test + smoke.**
   - Local boot per project SOP. Verify
     `mobs.LoadPatrols()` reports `loadedCount=2` (market
     beat + caravan). No panics.
   - Watch caravan run at least one full Thornwall →
     Stillwater leg + vendor circuit + return leg, verifying
     each contingency case in the smoke checklist below.
   - Hand off to user for in-game smoke validation (per
     project SOP — manual playtest precedes prod push).

## Open questions

None as of brainstorming close. Open items deferred to
follow-ups:

- Caravan runner-delivery flavor pass (Ketil's son).
- Follower-stranding recovery threshold tuning (smoke-test
  driven).
- Wagon-death recovery design (no current trigger, so no rush).

## Related work

- Chunk 3.4 — Waypoint patrols (foundation).
- Chunk 3.2 — NPC schedules (composes with patrols via
  `activity: patrol`).
- Stage 2 caravan system spec (original design):
  `docs/superpowers/specs/completed/2026-04-27-caravan-system-design.md`.
- Stage 3.1 foragers spec (Fernway handoff partner):
  `docs/superpowers/specs/completed/2026-04-29-stage-3-1-foragers-design.md`.
- Economy dashboard spec:
  `docs/superpowers/specs/completed/2026-05-01-economy-health-dashboard-design.md`.

## Memory items

- `project_caravan_runner_delivery_flavor.md` — post-3.7
  flavor pass.
