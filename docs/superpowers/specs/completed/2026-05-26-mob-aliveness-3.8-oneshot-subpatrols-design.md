# Mob Aliveness 3.8 — One-Shot Sub-Patrols (Caravan Runner + Forager Delivery) — Design

**Status:** Approved (brainstorming) — ready for `writing-plans`
**Roadmap chunk:** 3.8 (Phase 3 — Routine layer)
**Size:** M
**Branch:** `feature/mob-aliveness-3.8-oneshot-subpatrols`
**Depends on:** Chunk 3.7 (cross-zone patrols + caravan unification — shipped)

## Goal

Two routine-layer refinements that share a single new engine primitive: a *one-shot sub-patrol* invoked from within a larger NPC routine.

1. **Caravan runner-delivery.** The 3.7 caravan migration preserved the "wagon visits every vendor room" pattern as a faithful refactor. 3.8 closes the flavor loop: when the caravan stops at a depot, the leader's son Lars (mob 359, currently tagged as a guard but canonically Ketil's son) walks the local vendor circuit as a runner. Wagon stays parked at the depot with the horses (Hob 375, Bran 376) and the actual guard (Marta 358); Lars carries cargo to and from each vendor.

2. **Forager delivery-as-patrol.** Foragers currently use a custom state-machine for both wander+forage AND vendor delivery; the delivery loop has been a recurring prod stability issue (stranding, edge-case pathto failures). 3.8 moves the Delivering phase onto the patrol layer's retry-then-home-fallback + standardized interrupt handling. Wander+forage stays in `internal/forager/` unchanged.

Both consumers reuse the same new engine primitive: a oneshot patrol that walks waypoints once, fires `events.PatrolCompleted` on the final dwell expiry, and clears itself.

## In scope

1. **New `loop_shape: oneshot`** on patrol YAML. Walks waypoints once instead of looping; the patrol executor's terminal-waypoint branch emits `events.PatrolCompleted` and calls `ClearOneshotPatrol(mob)`.

2. **New `events.PatrolCompleted` event** in `internal/events/eventtypes.go`:
   ```go
   type PatrolCompleted struct {
       MobInstanceId int
       PatrolId      string
       RoomId        int
   }
   ```

3. **Runtime patrol-assignment helpers** in `internal/mobs/patrol.go`:
   - `StartOneshotPatrol(mob *Mob, patrolId string) bool` — assigns at runtime, resets patrol MiscData. Rejects non-oneshot patrols.
   - `ClearOneshotPatrol(mob *Mob)` — clears `mob.PatrolId` + resets the four patrol MiscData keys.

4. **Patrol executor** in `internal/hooks/NewRound_IdleMobs_patrol.go` gains a `WantsComplete` plan branch for oneshot terminal arrival. Existing strict/yo-yo patrols are unaffected.

5. **Caravan main route truncates** from 22 waypoints to 4:
   - wp0: Thornwall depot (dwell 360, `caravan_depot`) — Lars runs Thornwall circuit
   - wp1: Outbound Fernway pickup (dwell 8, `caravan_fernway_pickup`)
   - wp2: Stillwater depot (dwell 180, `caravan_depot`) — Lars runs Stillwater circuit
   - wp3: Inbound Fernway pickup (dwell 8, `caravan_fernway_pickup`)
   - loops back to wp0

   Stillwater dwell bumps from 20 → 180 to accommodate Lars's circuit + return travel. Thornwall stays at 360.

6. **Two caravan runner-circuit oneshot patrols:**
   - `_datafiles/world/dogmud/patrols/thornwall_outskirts/thornwall_runner_circuit.yaml` — 8 Thornwall vendor rooms (464, 470, 471, 475, 480, 481, 482, 483) + terminal stop at depot 465.
   - `_datafiles/world/dogmud/patrols/thornwall_outskirts/stillwater_runner_circuit.yaml` — 8 Stillwater vendor rooms (4102, 4103, 4105, 4106, 4125, 4126, 4135, 4143) + terminal stop at depot 4109.

   Vendor waypoints carry `arrival_event: caravan_vendor`; terminal carries empty arrival_event.

7. **Lars (mob 359) YAML edits:**
   - Strength training bump (start with ~60, tune at impl time to roughly match wagon carry capacity with headroom).
   - Optional `runner` group tag for future filtering.
   - Idle commands / description nudge toward "Ketil's son / caravan runner" flavor (less explicit "guard").
   - `mobid: 359` and `LeaderMobId`/`WagonMobId` constants unchanged in `internal/caravan/wagon.go`.

8. **`internal/caravan/cargo_handoff.go`** (new file):
   - `TransferCargoToRunner(wagon, runner *mobs.Mob, outboundBuckets []string) int` — moves bucket-matching items wagon → runner at depot arrival.
   - `TransferAllCargoBack(runner, wagon *mobs.Mob) int` — empties the runner's cargo back to wagon on `PatrolCompleted`. No bucket filtering — what didn't sell goes home.
   - Runner mob template id constant `RunnerMobId = 359` (siblings to existing `LeaderMobId`, `WagonMobId`).

9. **Caravan arrival listener changes** (`internal/caravan/arrival_listener.go`):
   - `handleDepotArrival` at wp0/wp2 calls `startRunnerCircuit(leader, arrival)` which finds Lars + wagon, transfers cargo, and calls `StartOneshotPatrol`.
   - `handleVendorArrival` updated: looks up the runner in the arrival room (not the wagon), uses `VisitVendorsInRoom` with the runner as the cargo source. The wagon stays at the depot.
   - **New 5.3 safety:** depot-arrival handler also checks for Lars-with-cargo-and-wagon co-located. If found (path-stranded recovery case), call `TransferAllCargoBack(lars, wagon)` directly.

10. **New `internal/caravan/runner_completion_listener.go`** subscribes to `events.PatrolCompleted`. Filters on `pc.PatrolId ∈ {thornwall_runner_circuit, stillwater_runner_circuit}`. On match, finds the wagon co-located with Lars, transfers residual cargo Lars → wagon.

11. **Forager `DeliveryPatrolId` field** on `ForagerProfile` in `internal/forager/territory.go`:
    - Marsh (Tova, mob 371): `"marsh_forager_delivery"`
    - Steppe (Halix, mob 372): `"steppe_forager_delivery"`
    - Fernway (Kessa, mob 373): `""` (Kessa keeps her single-room meeting-point handoff — no sub-patrol).

12. **Two forager-delivery oneshot patrols:**
    - `_datafiles/world/dogmud/patrols/stillwater_marsh/marsh_forager_delivery.yaml` — Tova's 8 vendor rooms + sanctuary terminal 4123.
    - `_datafiles/world/dogmud/patrols/ironwind_steppe/steppe_forager_delivery.yaml` — Halix's 9 vendor rooms + sanctuary terminal 3040.

    Vendor waypoints carry `arrival_event: forager_vendor`; terminal carries empty.

13. **New `internal/forager/arrival_listener.go`** subscribes to `events.PatrolWaypointArrival`. Filters on `pwa.ArrivalEvent == "forager_vendor"`. On match, calls `SellToVendor(forager, room, profile.Buckets)`.

14. **New `internal/forager/completion_listener.go`** subscribes to `events.PatrolCompleted` for the two forager-delivery patrol ids. Advances the forager state machine from `StateDelivering` to the next state (`StateStoring` if `storage_chest_room` configured, else `StateRecalling`).

15. **`forager_step` btree action** (`internal/behaviortree/actions_forager.go`) — `StateDelivering` branch collapses from "internal vendor-room loop" to "if no active oneshot patrol AND DeliveryPatrolId set, `StartOneshotPatrol`; otherwise wait." Per-vendor pathto-and-sell sequencing is deleted.

16. **5.4 sanctuary-fallback safety** in `forager_step` btree action (`internal/behaviortree/actions_forager.go`): when `StateDelivering` branch runs and finds `mob.Character.RoomId == profile.SanctuaryRoom && mob.PatrolId == ""`, advance state directly (skip the `StartOneshotPatrol` call). Covers the home-fallback stuck case — patrol retry exhaustion brought the forager home, but no `PatrolCompleted` ever fired.

17. **Dashboard synthesizer evolution** in `internal/caravan/synthesize_state.go`:
    - When leader at wp0 AND Lars has an active oneshot patrol → return `StateThornwallRoute`.
    - When leader at wp0 AND Lars idle → return `StateThornwallDwell`.
    - Same pattern for wp2 / Stillwater.
    - In-transit (RoomId != waypoint.Room): unchanged from 3.7 (`OutboundTransit` / `InboundTransit`).
    - JSON enum names unchanged. Pre-existing `*Route` / `*Dwell` strings keep working in the UI.

18. **Hooks registration** (`internal/hooks/hooks.go`) adds three new listeners:
    - `events.PatrolCompleted{}` → caravan runner-completion handler
    - `events.PatrolWaypointArrival{}` → forager vendor-stop handler (filters by arrival_event)
    - `events.PatrolCompleted{}` → forager state-machine completion handler

## Out of scope

- **Multi-runner per caravan** — Lars is enough for the proof of concept. If we ever scale up to a larger trade convoy, a second runner can reuse the same primitive without engine changes.
- **Generalized "delegate vendor visits to a crew member" abstraction beyond caravan + forager** — the two consumers each have ~one short listener and one cargo helper. We'll generalize when a third consumer materializes, not before.
- **Town justice consequences for attacking the caravan** — see the forward-looking note in §Architecture below. 3.8 doesn't wire any 1.3 crime hooks; chunk 5.1 owns that.
- **Foragers entirely on patrols** — 3.8 only moves the Delivering phase onto a patrol. Wander+forage stays in `internal/forager/`. A future chunk could explore a `loop_shape: random` "wander territory" patrol; deferred.
- **Cargo-on-corpse "return-to-wagon" hook on Lars's death** — accept the cargo loss as a real-world freight failure mode. See §Contingencies 5.1.
- **Per-runner inventory limit beyond Strength × CarryCapacityMultiplier** — Lars's stat bump is the only carry-capacity knob. If a single circuit's outbound stock exceeds his limit, leftover items stay on the wagon until next cycle; no special handling needed.

## Architecture

### The shared primitive

```
                        OUTER STATE MACHINE
                ┌────────────────────────────────┐
                │ ...                            │
                │ delivery phase reached         │
                │     ↓                          │
                │ TransferCargo(wagon → runner)  │
                │ StartOneshotPatrol(...)        │
                │     ↓                          │
                │ wait for events.PatrolCompleted│
                │     ↓                          │
                │ advance to next state          │
                └────────────────────────────────┘
                              ▲
                              │ events.PatrolCompleted{mobInst, patrolId}
                              │
                       PATROL EXECUTOR (runtime)
                ┌────────────────────────────────┐
                │ loop_shape: oneshot            │
                │ walks waypoints once           │
                │ at terminal, dwell-done:       │
                │   emit PatrolCompleted         │
                │   ClearOneshotPatrol(mob)      │
                └────────────────────────────────┘
```

The outer state machine — whether the caravan listener's depot handler or the forager `forager_step` btree action's `StateDelivering` branch — is responsible for *when* to start the sub-patrol and *how to react* on its completion. The patrol layer owns *how* the sub-patrol walks, including retry-and-home-fallback, combat interrupt, and the standard `IsInCombat` guard.

### Forward-looking note (Phase 5 town justice)

Attacking the caravan crew or wagon will eventually carry severe consequences once the chunk 5.1 Town Justice work lands. Crimes against caravan members (assault, murder, theft from wagon) trigger Thornwall guard faction reputation damage — probably massive, since caravans are the city's economic lifeline — and unprovoked killings file murder records keyed to the Thornwall jurisdiction.

The 3.8 caravan listener doesn't wire any 1.3 crime hooks. The crime substrate already records witnessed assault/murder via the existing combat pipeline (chunk 1.3); Phase 5 will tune per-faction rep deltas for caravan-specific crimes. Spec-level note only — no engine surface in 3.8.

### Caravan flow under 3.8

1. Caravan main patrol takes Ketil from Thornwall depot (wp0) through Outbound Fernway (wp1) to Stillwater depot (wp2), with Lars + Marta + wagon + 2 horses party-following.
2. At wp2 (Stillwater depot, 180 round dwell), caravan listener fires `caravan_depot` arrival event. The listener:
   - Calls `TransferCargoToRunner(wagon, lars, []string{"thornwall", "fernway"})` (outbound goods from the Thornwall→Stillwater leg).
   - Calls `mobs.StartOneshotPatrol(lars, "stillwater_runner_circuit")`.
3. Lars walks the 8 Stillwater vendor rooms. Each `caravan_vendor` arrival event triggers `VisitVendorsInRoom(roomId, lars, deliveryBuckets, pickupBuckets)` (existing trade logic, source-mob swapped from wagon to Lars).
4. Lars reaches terminal waypoint (room 4109 — back at Stillwater depot). Final dwell of 1 round expires, executor emits `events.PatrolCompleted{MobInstanceId: lars.InstanceId, PatrolId: "stillwater_runner_circuit", RoomId: 4109}` and clears `lars.PatrolId`.
5. The caravan's `PatrolCompleted` listener fires: finds wagon co-located with Lars at 4109, calls `TransferAllCargoBack(lars, wagon)`. Lars's inventory is empty; wagon now carries Stillwater pickup goods + any unsold Thornwall remainder.
6. Stillwater dwell continues until the 180-round counter expires. Lars party-follows the wagon for the rest of the dwell (uses existing party plumbing, same as during transit).
7. Caravan main patrol advances wp2 → wp3 (Inbound Fernway) → wp0 (Thornwall depot). Cycle restarts.

The same pattern happens at wp0 with the Thornwall circuit. Thornwall's longer dwell (360) gives the caravan its full pre-departure rest after Lars completes his circuit.

### Forager flow under 3.8

1. Forager (e.g., Tova) wanders her territory in `StateForaging` — unchanged from chunk 2.9.
2. When forage carry-cap threshold or fatigue trigger fires, state advances to `StateTravelingToDropoff`, then `StateDelivering` — unchanged.
3. `forager_step` btree action on entry to `StateDelivering`: if Tova has no active oneshot patrol and `profile.DeliveryPatrolId == "marsh_forager_delivery"`, call `mobs.StartOneshotPatrol(tova, "marsh_forager_delivery")`. (Kessa's empty `DeliveryPatrolId` means this branch skips and her existing single-room handoff logic continues to run.)
4. Tova walks her 8 vendor rooms. Each `forager_vendor` arrival event fires `SellToVendor(tova, roomId, []string{"stillwater", "base", "overlap"})` via `internal/forager/arrival_listener.go`.
5. Tova reaches terminal waypoint (sanctuary 4123). `PatrolCompleted` fires.
6. `internal/forager/completion_listener.go` advances Tova's state machine: `StateDelivering → StateStoring` (Tova has a storage chest), or `→ StateRecalling` (for foragers without one).

### Code surface

| Package | New files | Modified files | Notes |
|---|---|---|---|
| `internal/events/` | — | `eventtypes.go` | +1 event struct |
| `internal/mobs/` | — | `patrol.go` | +2 helpers, +`loop_shape: oneshot` parse |
| `internal/hooks/` | — | `NewRound_IdleMobs_patrol.go`, `hooks.go` | +`WantsComplete` branch, +3 listener registrations |
| `internal/caravan/` | `cargo_handoff.go`, `runner_completion_listener.go` | `arrival_listener.go`, `synthesize_state.go` | Listener swap from wagon → runner; synthesizer reads Lars's patrol state |
| `internal/forager/` | `arrival_listener.go`, `completion_listener.go` | `territory.go`, `actions_forager.go` (deletes), `state.go` (5.4 safety) | StateDelivering vendor loop deleted |
| `_datafiles/world/dogmud/patrols/` | 4 new YAMLs | `caravan_thornwall_stillwater.yaml` | 22 → 4 waypoints |
| `_datafiles/world/dogmud/mobs/thornwall_city/` | — | `359-lars.yaml` | Strength bump + flavor |

### Contingency design

| # | Case | Disposition |
|---|---|---|
| 5.1 | Lars dies mid-circuit | Cargo drops as corpse loot (existing behavior). Acceptable. |
| 5.2 | Forager dies mid-delivery | Same shape as 5.1. Acceptable. |
| 5.3 | Lars path-stranded (home-fallback never completes circuit) | **New code:** caravan `caravan_depot` handler also transfers cargo back when Lars-with-cargo-and-wagon are co-located at depot arrival. |
| 5.4 | Forager path-stranded at sanctuary | **New code:** forager idle tick advances `StateDelivering` → next state when forager is at `SanctuaryRoom` with no active oneshot patrol. |
| 5.5 | Caravan departs while Lars mid-circuit | Tune Stillwater dwell (180 → 240 if smoke shows tightness); Lars re-engages party-follow on `PatrolCompleted` clear of his PatrolId. Cargo-return safety from 5.3 fires on his next depot arrival. |
| 5.6 | Server restart mid-circuit | Patrol MiscData persists on instance; existing behavior. |
| 5.7 | Lars dies in transit (caravan crossing zones) | Lars respawns at HomeRoomId Thornwall depot; party-follow walks him cross-zone to catch up. Cargo loss already accepted (5.1). |
| 5.8 | Player attacks Lars or another crew member | Combat interrupts patrol via existing `IsInCombat` guard; resumes after combat. See forward-looking town-justice note. |

Two new small safeties total (5.3 in caravan, 5.4 in forager), both in existing listener files or short idle-tick checks.

## Testing strategy

### Unit tests

- `internal/mobs/patrol_test.go` — `loop_shape: oneshot` parses; `StartOneshotPatrol` rejects non-oneshot patrols, resets MiscData; `ClearOneshotPatrol` clears all four patrol MiscData keys.
- `internal/hooks/NewRound_IdleMobs_patrol_test.go` — emission of `PatrolCompleted` at terminal waypoint; `mob.PatrolId` cleared post-emission; idempotent (subsequent ticks don't re-emit).
- `internal/caravan/cargo_handoff_test.go` (new) — `TransferCargoToRunner` filters by bucket and moves matching items; `TransferAllCargoBack` empties the runner unconditionally.
- `internal/caravan/arrival_listener_test.go` — extend: wp0/wp2 `caravan_depot` arrival starts the right runner-circuit patrol; 5.3 safety transfers stranded cargo when Lars-and-wagon co-located on depot arrival.
- `internal/caravan/runner_completion_listener_test.go` (new) — Lars's residual inventory moves back to wagon on `PatrolCompleted`; no-op when wagon absent.
- `internal/forager/arrival_listener_test.go` (new) — `forager_vendor` arrival triggers `SellToVendor`; non-`forager_vendor` events ignored.
- `internal/forager/state_test.go` — `StateDelivering` advances on `PatrolCompleted`; sanctuary-fallback safety advances state when forager at sanctuary with no active patrol.
- `internal/caravan/synthesize_state_test.go` — extend: with Lars's runner patrol active, synthesizer returns `*Route`; without, `*Dwell`.

### Integration / boot validation

- `mobs.LoadPatrols() loadedCount=6` (existing market beat + caravan main + 2 caravan runner + 2 forager delivery).
- Truncated caravan main loads as 4 waypoints (boot log shows 4 zone-resolutions).
- Lars's YAML loads cleanly; mob template id 359 still resolves to the same mob.

### Manual smoke (12-step checklist)

Per CLAUDE.md SOP: nuke instance saves before booting. Then:

1. Boot. `LoadPatrols loadedCount` matches expected total (6).
2. Caravan at Thornwall depot during wp0 dwell. Lars co-located.
3. Lars's Thornwall circuit starts: cargo transfers wagon → Lars, Lars walks 8 vendor rooms. Trade flavor fires at each.
4. Lars returns to depot, `PatrolCompleted` fires, residual cargo transfers Lars → wagon.
5. Caravan departs Thornwall after 360-round dwell. Lars rejoins party-follow.
6. Caravan crosses Fernway, arrives Stillwater. Stillwater circuit fires.
7. Caravan returns to Thornwall. Cycle restarts.
8. Forager Tova: wander+forage unchanged; on `StateDelivering`, sub-patrol starts; vendor stops fire `SellToVendor`; on `PatrolCompleted`, state advances to `StateStoring`.
9. Force stranding: admin command stops Lars mid-circuit. Watch home-fallback. Verify cargo-return safety on next depot arrival (5.3).
10. Kill Lars mid-circuit. Verify cargo drops, Lars respawns at depot, party-follow re-engages.
11. `/admin/economy/` dashboard: caravan state cycles through `ThornwallDwell` → `ThornwallRoute` → `ThornwallDwell` → `OutboundTransit` → … etc.
12. Sanity-check chunk 3.6 conversation pilot — Dal's `thornwall_barmaid` schedule (fixed in `cafe482d`) is firing back-room banter.

## Rollout order

Three independently shippable phases plus docs.

### Phase 1 — Engine primitive (no behavior change)

1. `loop_shape: oneshot` parsing in `internal/mobs/patrol.go`.
2. `events.PatrolCompleted` event type.
3. `StartOneshotPatrol` / `ClearOneshotPatrol` helpers.
4. Patrol executor `WantsComplete` branch + tests.

After Phase 1: existing patrols (market beat, caravan main) unchanged. Engine primitive callable but dormant.

### Phase 2 — Forager migration (lower-risk consumer first)

Foragers go first because the prod stranding issues are already a pain point, the state-machine has clean phase boundaries, and isolating from caravan means a failure here doesn't break caravans.

1. Author 2 forager-delivery oneshot YAMLs (Marsh, Steppe).
2. Add `DeliveryPatrolId` to `ForagerProfile`; populate for Marsh/Steppe.
3. New `internal/forager/arrival_listener.go` + `completion_listener.go`.
4. Register listeners in `hooks.RegisterListeners`.
5. Rewrite `actions_forager.go` `StateDelivering` branch.
6. 5.4 sanctuary-fallback safety.
7. Nuke instance saves + boot test + smoke.

### Phase 3 — Caravan runner-delivery flip

Higher risk because it changes caravan main route shape *and* introduces cargo handoff.

1. Author 2 runner-circuit oneshot YAMLs (Thornwall, Stillwater).
2. Lars (mob 359) YAML edits.
3. New `internal/caravan/cargo_handoff.go`.
4. New `internal/caravan/runner_completion_listener.go`.
5. Caravan listener changes (wp0/wp2 dispatch, vendor handler source swap, 5.3 safety).
6. Truncate caravan main YAML 22 → 4 waypoints.
7. Synthesizer evolution.
8. Register listener in `hooks.RegisterListeners`.
9. Nuke instance saves + boot test + smoke.

Phase 3 commits in 2-3 sub-steps where intermediate states still work (e.g., YAMLs + Lars edits + helpers first as "loaded but unused"; listener flip last).

### Phase 4 — Documentation + cleanup

1. PATCH_NOTES.md dated entry.
2. MOB_ALIVENESS_ROADMAP.md — mark 3.8 Done.
3. Resolve / archive related memory items (`project_caravan_runner_delivery_flavor.md`, applicable forager-stranding entries).

Each phase merges to master with `--no-ff` per project convention. Phases 1–3 each warrant their own smoke session.

## Open questions

None as of brainstorming close. Open items deferred to follow-ups:

- Lars strength bump exact value (tune at impl time, target wagon-cap-ish).
- Stillwater dwell tuning (180 → 240 if smoke shows tightness).
- "wander territory" patrol shape for the rest of the forager state machine — deferred to a future chunk if it surfaces value.

## Related work

- Chunk 3.4 — Waypoint patrols (foundation).
- Chunk 3.7 — Inter-zone patrols + caravan unification (the chunk this builds on).
- Chunk 2.9 — Forager command lift (the chunk whose state machine 3.8 partially simplifies).
- Chunk 3.6 — NPC↔NPC conversations (the post-3.7 Dal back-room fix at `cafe482d` is unrelated but adjacent).
- Chunk 5.1 — Town Justice (future; the forward-looking caravan-attack-consequences note).

## Memory items

- `project_caravan_runner_delivery_flavor.md` — the originating memory; resolved by Phase 3.
- (Plus any forager-stranding entries that 3.8 closes; mark at Phase 4.)
