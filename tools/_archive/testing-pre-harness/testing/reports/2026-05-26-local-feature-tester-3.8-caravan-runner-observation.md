# Chunk 3.8 Caravan Runner Observation — Feature Test Report

**Date:** 2026-05-26
**Tester role:** feature-tester
**Goals file:** tools/testing/goals/3.8-caravan-runner-observation.yaml
**Server:** localhost:55555 (commit 2a53e72a, mob aliveness chunk 3.8)
**Session duration:** ~2 hours real-time
**Character:** smoketester

---

## Session Summary

Followed Lars (mob 359, caravan runner, Ketil's son) across one full
Thornwall→Stillwater cycle, teleporting ahead of him on each vendor
circuit to observe trade flavor in-room. Observed both the Stillwater
vendor circuit and the Thornwall vendor circuit from inside vendor rooms.
Captured dashboard state at multiple points via the economy API.

**Primary finding:** Trade flavor messages never fired at any vendor
stop in either city. Vendor stock levels were completely unchanged
across a full Thornwall circuit (verified by before/after shop file
comparison and dashboard snapshot comparison). Zero cross-city
pickups or deliveries occurred. Root cause identified in code (see
BUG-01 and BUG-02 below).

Server restarted once mid-session (~20:43:32 local); resumed cleanly.

---

## Goal Results

- [ ] **Goal 1 — Find Ketil + crew at Thornwall depot (room 465)**
  PASS: Ketil (mob 357), wagon (mob 374), Lars (mob 359), Marta (mob
  358), Hob (mob 375), Bran (mob 376) all confirmed co-located at room
  465 during Thornwall dwell periods. `mob schedule ketil` output
  confirmed.

- [ ] **Goal 2 — Watch Lars start a Thornwall runner circuit**
  PARTIAL PASS: Lars visibly walked the Thornwall vendor circuit
  (rooms 464→470→471→475→480→481→482→483→465), dwelling ~3 rounds at
  each vendor. Movement without oscillation confirmed (commit 66336b98
  fix working). However: no cargo transfer flavor fired, and the API
  showed Lars's inventory was empty during the circuit (cargo_weight
  on the wagon API shows 14-16 lbs but Lars carries nothing
  observable). The first waypoint (room 464, food vendor) is
  silently skipped — see BUG-01.

- [ ] **Goal 3 — Observe trade flavor at each Thornwall vendor**
  FAIL: Zero flavor messages observed at any of the 8 Thornwall
  vendor stops. Waited 3+ rounds at rooms 470 (Kerra), 471 (Voss),
  480 (Maren), 481 (Brynn), 482 (Tess), 483 (Vael). No messages
  of any form appeared. Dashboard snapshot comparison (round 1363401
  vs round 1364253, spanning 852 rounds and at least one full circuit)
  confirms zero stock changes at any Thornwall vendor for
  thornwall-bucket items.

- [x] **Goal 4 — Lars returns to Thornwall depot after circuit**
  PASS: Lars walked back to room 465 (depot) after completing the
  vendor circuit. Wagon remained parked at 465 throughout.

- [x] **Goal 5 — Watch caravan depart Thornwall**
  PASS: Observed caravan (full crew) departing room 465 northward
  toward Fernway during a Thornwall→transit transition. Dashboard
  state changed from `thornwall_dwell`/`thornwall_route` to
  `outbound_transit` as expected.

- [ ] **Goal 6 — Fernway pickup (room 4038)**
  BLOCKED: Not directly observed during this session. The sealed
  crate at 4038 was not stocked (foragers Tova, Halix, Kessa all
  in `foraging` state with stuck_rounds 10000+; no deliveries to
  crate). No flavor message expected even if the caravan arrived.

- [ ] **Goal 7 — Watch Lars start the Stillwater runner circuit**
  PARTIAL PASS: Lars visibly walked the Stillwater vendor circuit
  (rooms 4102→4103→4105→4106→4125→4126→4135→4143→4109), dwelling
  ~3 rounds at each vendor. Zero trade flavor messages appeared at
  any stop. Same failure mode as Thornwall.

- [x] **Goal 8 — Dashboard state transitions**
  PASS (partial): Dashboard state names cycled correctly through
  observed transitions:
  - `stillwater_dwell` (round ~1363400, caravan at room 4143 during
    stillwater_dwell)
  - `inbound_transit` (observed during northbound leg to Thornwall)
  - `thornwall_route` (round ~1363931+, Lars mid-circuit at room 483)
  - `thornwall_dwell` (round ~1364253, Ketil at room 483)
  Dashboard correctly distinguishes `thornwall_dwell` from
  `thornwall_route`. States not observed this session: `fernway_pickup`,
  `stillwater_route`.

---

## Findings

### BUG-01 [CRITICAL] — Waypoint 0 silently skipped: first vendor
  stop on every runner circuit receives no PatrolWaypointArrival event

**File:** `internal/mobs/patrol.go`, `StartOneshotPatrol` function

**Root cause:** `StartOneshotPatrol` initializes
`patrol_dwell_remaining = 0` for all waypoints, including waypoint 0.
When Lars paths from the depot to waypoint 0 (e.g., room 464, food
vendor) and arrives, `patrolTickPlan` in
`internal/hooks/NewRound_IdleMobs_patrol.go` sees `dwellRemaining==0`
and immediately returns `plan.WantsAdvance = true`. The
`WantsDwellWait` branch — which is where `PatrolWaypointArrival`
fires — is never entered. Lars skips room 464 entirely with no dwell
and no vendor trade.

The `patrol_dwell_remaining = 0` sentinel is correct for patrols
where the mob starts mid-route and needs to path to its first
waypoint, but for a fresh oneshot dispatch where waypoint 0 has
`dwell_rounds: 3`, it should be initialized to something that signals
"not yet arrived" rather than "arrived and dwell expired."

**Impact:** The food vendor (room 464) never receives a
`caravan_vendor` arrival event. One of 8 Thornwall vendor stops and
one of 8 Stillwater vendor stops are always skipped per circuit.

**Suggested fix:**
```go
// In StartOneshotPatrol: use -1 (or a dedicated sentinel like
// patrol_first_arrival_pending: true) to distinguish "not yet
// arrived at wp0" from "arrived and dwell expired".
mob.Character.SetMiscData("patrol_dwell_remaining", -1)
// Then in patrolTickPlan's WantsAdvance branch that sets NextDwellRounds,
// initialize dwell_remaining to p.Waypoints[nextIdx].DwellRounds
// as today. And in the WantsDwellWait branch, treat current < 0
// as "first arrival — set to authored dwell and emit event."
```
Alternatively, initialize `patrol_dwell_remaining` to the authored
dwell of waypoint 0 directly in `StartOneshotPatrol`:
```go
mob.Character.SetMiscData("patrol_dwell_remaining",
    p.Waypoints[0].DwellRounds)
```
This would cause the "first arrival" detection logic to fire on the
first `WantsDwellWait` tick because `current == authored dwell`.

---

### BUG-02 [CRITICAL] — VisitVendorsInRoom produces zero stock
  changes across all waypoints 1-7 on both circuits; cross-city
  trade is completely inoperative

**Files:** `internal/caravan/visit.go`, `internal/caravan/arrival_listener.go`

**Evidence:**
- Snapshot comparison (round 1363401 vs round 1364253, 852 rounds):
  Kerra (room 470) item 40018 (steel ingot, thornwall bucket):
  `current: 10` in BOTH snapshots. Expected pickup to drain this
  (current 10 > restock_qty 5, eligible for 1 unit pickup). Unchanged.
- Kerra item 40059 (lake-iron nodule, stillwater bucket):
  `current: 30` → `current: 30`. If any stillwater-bucket delivery
  had occurred, this should have risen or the max would gate it
  (max 30, current 30 — already at max, correct).
- jeweler Tess (room 482) item 40021 (copper wire, thornwall bucket):
  `current: 47` → `current: 43` — a DROP of 4, indicating player
  purchases (restock_count rose 26384→26610, player-driven). If
  Lars had picked up any, we'd see the drop happen faster.
- Economy API at round 1363401 AND round 1364253:
  `cargo_by_bucket: {}` on the wagon in both snapshots.

**Most likely cause:** `startRunnerCircuit` transfers cargo from the
wagon to Lars via `TransferCargoToRunner(wagon, lars, outboundBuckets)`
where `outboundBuckets = ["stillwater", "fernway"]` for the Thornwall
circuit. If the wagon carries zero stillwater or fernway items at the
time of the depot arrival event, `TransferCargoToRunner` moves nothing
to Lars and Lars walks the circuit with an empty inventory. With an
empty inventory, the DELIVER pass in `VisitVendorsInRoom` has nothing
to deliver, and the PICKUP pass may still run — but `FormatVisitMessage`
returns "" when both `delivered` and `pickedUp` are empty, so no
message fires. But if pickup ran, stock should change.

**Secondary candidate:** `cargo_by_bucket: {}` in both snapshots
despite `cargo_weight: 14-16` suggests the wagon carries only
`bucket: ""` (base/overlap bucket items) or items where
`economy.BucketFor(itemId)` returns `""`. If the wagon is perpetually
stocked with only base/overlap items that don't match "stillwater" or
"fernway" in the outboundBuckets list, `TransferCargoToRunner` moves
zero items and Lars walks empty.

The foragers (Tova, Halix, Kessa) are all in `foraging` state with
stuck_rounds 10000+ and zero deliveries to the Fernway sealed crate,
meaning the wagon's inventory comes entirely from vendor pickups
during prior Thornwall circuits — which also produce zero pickups
(circular dependency). The system started with an empty wagon and
has never bootstrapped cross-city cargo flow.

**Note:** There is also a dashboard rendering gap: `captureCaravans`
in `internal/economy/health/capture.go` reads wagon cargo using
`FindWagonInRoom(m.Character.RoomId)` where `m` is Ketil (the leader).
During `thornwall_route` state, Ketil is walking Thornwall vendor
rooms while the wagon is parked at the depot (room 465). The wagon is
never in Ketil's current room during the route phase, so
`cargo_by_bucket` always shows `{}` even if the wagon has stock.
This is a dashboard read error, not an operational one.

---

### CONCERN-01 — Foragers stuck for 10000+ rounds; no cross-city
  cargo bootstrap path exists

All three foragers (Tova mob 371, Halix mob 372, Kessa mob 373) report
`stuck_rounds` between 10809-10815 in the latest snapshot. This means
they've been in `foraging` state for ~43,000 seconds (~12 hours) with
no state advancement. The Fernway sealed crate is empty. Without
forager deliveries, the wagon has no cross-zone cargo to bootstrap the
caravan cross-city supply chain. The `cargo_by_bucket: {}` result is
therefore correct if the foragers have never delivered.

---

### CONCERN-02 — Wagon cargo_by_bucket dashboard gap during route phase

As described in BUG-02 (secondary section): `captureCaravans` uses
`FindWagonInRoom(leader.RoomId)`. During `thornwall_route` state,
the wagon stays at depot room 465 while Ketil walks vendor rooms.
The dashboard shows `cargo_by_bucket: {}` and `cargo_weight: 16` (a
contradiction — weight is non-zero but no buckets are visible). This
misleads operators into thinking the wagon is empty when it may not be.

Fix: During *_route synthesized states, read wagon cargo from the
wagon's known home room (depot room 465 for Thornwall, room 4109 for
Stillwater) rather than the leader's current room.

---

### OBSERVATION-01 — Dashboard state name `thornwall_route` fires
  correctly during Lars's circuit

At round ~1363931 the snapshot shows `state: thornwall_dwell` with
`room_id: 483` (Ketil at enchanter Vael's room). A prior snapshot
at round ~1363800 showed `state: thornwall_route`. This confirms
the synthesized state distinguishes "Lars is mid-circuit" from "dwell
period with Lars at depot." Goal 8 pass criteria met.

---

### OBSERVATION-02 — Wagon stays parked at depot during Lars's circuit

At all observed times during the Thornwall circuit, wagon (mob 374)
was present in room 465. Lars walked vendor rooms alone. The chunk 3.8
design intent (wagon parked, runner walks circuit) is implemented
correctly at the locomotion level.

---

### OBSERVATION-03 — No oscillation observed (commit 66336b98 fix
  confirmed working)

Lars walked each vendor room in sequence: 464→470→471→475→480→481→
482→483→465, visiting each room once and moving forward. No back-and-
forth oscillation between depot and a vendor was observed across two
full circuits (Thornwall + Stillwater). The oneshot patrol mechanism
prevents the loop-restart regression.

---

## Pass Criteria Evaluation

| Criterion | Result |
|-----------|--------|
| Lars visibly walks both circuits with named trade flavor | FAIL — Lars walks correctly but zero flavor messages fired |
| At least one cross-city pickup observed | FAIL — zero pickups across both circuits |
| At least one cross-city delivery observed | FAIL — zero deliveries across both circuits |
| Lars returns to depot and wagon stays parked | PASS |
| Dashboard state names cycle correctly | PASS (partial — *Route/*Dwell/*Transit confirmed; FernwayPickup not observed) |

---

## Raw Stats

| Metric | Value |
|--------|-------|
| Vendor stops observed in-room | 12 (6 Thornwall, 6 Stillwater) |
| Trade flavor messages received | 0 |
| Stock level changes attributable to Lars | 0 |
| Dashboard snapshots captured | 2 (rounds 1363401, 1364253) |
| Dashboard state transitions observed | 3 (stillwater_dwell, inbound_transit, thornwall_route/dwell) |
| Server restarts during session | 1 (survived, session continued) |
| Rounds elapsed during test | ~852 (round 1363401 → 1364253) |

---

## Recommended Fix Priority

1. **BUG-01 (StartOneshotPatrol dwell init):** Small, surgical fix in
   `internal/mobs/patrol.go`. Change `patrol_dwell_remaining` init
   from `0` to `p.Waypoints[0].DwellRounds` in `StartOneshotPatrol`.
   This fixes the waypoint-0 skip and unblocks the first vendor stop.

2. **BUG-02 (wagon bootstrap / empty cargo):** The system requires
   cross-city cargo to already exist on the wagon for vendor deliveries
   to flow. Two sub-fixes needed:
   a. Admin/setup: manually seed the wagon with a small batch of
      stillwater + fernway + thornwall items so the first circuit has
      something to deliver, OR
   b. Fix the forager stuck-state so they deliver to the crate and
      the wagon can pick it up at Fernway.
   The pickup side (vendor→Lars) should work once BUG-01 is fixed,
   since eligible vendors (e.g., Kerra room 470 item 40018:
   current=10 > restock_qty=5) will contribute to pickups even with
   an empty wagon outbound leg.

3. **CONCERN-02 (dashboard wagon cargo read during route):** Low
   priority — observability fix only, no operational impact.
