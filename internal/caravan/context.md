# Caravan Context

## Purpose

`internal/caravan` moves goods between towns with visible NPCs. A **leader**
walks an authored patrol route; a **wagon** carries the cargo; **runners** peel
off at a destination, visit vendor shops to deliver and collect stock, and
return. The result is that shop inventory is physically hauled rather than
conjured.

Since chunk 3.7 the caravan has **no state machine of its own**. State is
*synthesised* from where the leader currently is on its patrol
(`SynthesizeStateForLeader`), and the old behaviour-tree `caravan_step` driver
is gone. The patrol layer is the source of truth; this package reacts to
waypoint-arrival events.

## Files

- **state.go** — the eight `CaravanState` phases and their predicates.
- **synthesize_state.go** — derives the current state from patrol position.
- **arrival_listener.go** — the main event handler: depot, vendor, and Fernway
  pickup arrivals.
- **runner_completion_listener.go** — handles a runner finishing its circuit.
- **cargo_handoff.go** — `TransferCargoToRunner`, `TransferAllCargoBack`.
- **visit.go** — vendor delivery/pickup and the player-visible visit message.
- **wagon.go** — locating caravan mobs in a room by template id.
- **import_circuits.go / import_arrival.go / import_load.go** — the import
  variant, which loads from a warehouse rather than a depot.
- **regroup.go** — `ForceRegroupCrew`.
- **reset.go** — `ResetLeaderToDepot`.
- **throughput.go** — persisted per-caravan delivery statistics.

## The cycle

```
ThornwallDwell → OutboundTransit → OutboundFernwayPickup → StillwaterRoute →
StillwaterDwell → InboundTransit → InboundFernwayPickup → ThornwallRoute → ↺
```

```go
func IsDwellState(s CaravanState) bool          // waiting out a depot timer
func IsTransitState(s CaravanState) bool        // long haul, incl. Fernway substates
func IsRouteState(s CaravanState) bool          // visiting vendors in town
func IsFernwayPickupState(s CaravanState) bool  // waiting for the forager handoff
func (s CaravanState) Name() string             // canonical string for the dashboard
```

`IsTransitState` deliberately includes the two Fernway pickup substates — they
are brief stops *inside* a transit leg, not separate phases, and anything
reporting "is the caravan travelling?" must treat them as travel.

## Vendor visits

```go
type ItemMove struct { /* item + direction */ }
type VisitOpts struct { /* delivery/pickup buckets, filters */ }

func VisitVendorsInRoom(...) 
func VisitVendorsInRoomOpts(...)
func FormatVisitMessage(runnerName string, delivered, pickedUp []ItemMove) string
```

Deliveries and pickups are filtered by **bucket** — named groups of item
categories carried on the patrol definition. `pickupQualifies` decides what a
runner will accept back; `isFinishedGood` distinguishes crafted output from raw
material so a runner does not haul a vendor's own production away again.

## Throughput

```go
func GetThroughput(zone string, mobId int) *Throughput
func IncrementDelivery(zone string, mobId int, rarityTier int)
func AddLbsDelivered(zone string, mobId int, lbs uint64)
func SaveThroughput(zone string, mobId int) error
func SaveAllThroughputs()
func PrewarmThroughputFromPersistedFiles() (int, error)
func ClearThroughputCache()
func SetThroughputBaseDirForTest(dir string)
```

Persisted living-economy state, cached in memory and flushed to disk. Like
`shops/` and `guilds/`, this is **not** instance-save data and must not be
wiped by the smoke-test cleanup ritual.

## Gotchas

- **Do not add a stored caravan state field.** State is synthesised on demand;
  a cached copy will drift from the patrol the first time combat interrupts a
  leg. `SynthesizeStateForLeader` returns `(state, ok)` — respect the `ok`.
- **Caravan mobs are identified by template id** (`IsCaravanMob`,
  `FindMobByTemplateInRoom`), not by name or instance. Renaming a caravan mob
  is safe; changing its template id is not.
- **Crates are drained, not moved.** `drainCrateIntoWagonCaravan` returns
  `(stored, rejected)` — a full wagon silently rejects the remainder, so check
  both counts before reporting success.
- **`ForceRegroupCrew` only covers leader respawn.** A crew member that dies or
  is separated mid-route is not regrouped; this is a known limitation the
  project has accepted and is watching.
- **Throughput files are living state.** Deleting one resets that caravan's
  statistics; it does not reset the caravan.
- **Runner circuits are guarded by `isRunnerCircuitActive`.** Two runner
  circuits must not overlap; the guard is what prevents cargo being handed to a
  runner that is already out.

## Dependencies

`mobs`, `rooms`, `items`, `characters`, `events`, `sealedcrate`, `warehouse`,
`shops`, `mudlog`, `util`.

## Consumers

`internal/hooks` (event listener registration), the economy dashboard, and
admin caravan inspection commands.
