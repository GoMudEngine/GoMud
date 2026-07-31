# Ferry Context

## Purpose

`internal/ferry` runs passenger vessels between ports and the NPC **factors**
that ride them carrying trade goods. Two subsystems share one route definition:

- **Passenger travel** — a player boards while the vessel is docked and is
  moved when it sails.
- **Trade circuits** — a factor NPC loads cargo from a port warehouse, sails,
  and sells it at the far end, feeding the living economy.

The defining property is that **vessel position is not stored anywhere**.
`StateAt(route, round, roundsPerDay)` is a pure function of the round counter,
so a restart, a copyover, or a rollback cannot desynchronise the schedule.

## Files

- **route.go** — `Port`, `Route`, YAML loading, validation.
- **state.go** — `VesselState` and the pure schedule functions.
- **controller.go** — `Tick()`: gangplank reconciliation, departure/arrival
  emits, ambient messaging.
- **board.go** — `Board()` and the player-facing boarding flow.
- **tradecircuit.go** — `TradeCircuit` definition, lookup, validation.
- **factor.go** — the factor NPC state machine (the largest file).

## Schedule model

```go
type VesselState struct {
    Docked                bool
    PortIdx               int  // berth when docked; DESTINATION when at sea
    RoundsUntilTransition int
}

func StateAt(r Route, round uint64, roundsPerDay int) VesselState
func NextDockedRound(r Route, portIdx int, fromRound uint64, roundsPerDay int) uint64
```

A route is a two-port loop with a fixed cycle of
`2 × (LayoverHours + CrossingHours)`, converted to rounds and offset by
`PhaseOffsetRounds` so different vessels do not all sail in lockstep. `StateAt`
is a modulo over that cycle — four cases: docked at 0, at sea to 1, docked at
1, at sea to 0.

**Note the `PortIdx` overload:** docked means "this is the berth," at sea means
"this is where we are heading." Reading it as the origin is the easy mistake.

`NextDockedRound` linearly scans up to one full cycle. That is deliberate — it
runs only when a player asks about the timetable, and simplicity beat cleverness.

## Trade circuits and factors

```go
type TradeCircuit struct { /* per-route cargo/port config */ }
func TradeCircuitFor(routeId string) (TradeCircuit, bool)
func IsFactorMobId(mobId int) bool
func FactorPhaseName(instanceId int) (string, bool)
```

A factor moves through phases (`FactorPhase`) driven by `factorDecide`, a pure
decision function taking the circuit, the vessel state, the factor's position,
and its stored state, and returning a `factorAction`. `applyFactorAction`
performs it. Keeping the decision pure is what makes factor behaviour testable
without a running world.

Cargo loading pulls from the port's warehouse (`loadFromWarehouse`) and is
ordered by `prioritizeByDemand` against `exportDemand`, so factors carry what
the destination is short of rather than a fixed manifest. Leftovers are banked
at the dock rather than carried indefinitely.

Factors are moved with `moveFactorSilently` when they should not be seen
walking, and `transportFactor` when depart/arrive emotes should fire.

## Public API

```go
func LoadDataFiles()
func RouteFor(routeId string) (Route, bool)
func AllRoutes() []Route
func (r Route) Id() string
func (r Route) Filepath() string
func (r Route) Validate() error

func Tick()
func Board(user *users.UserRecord, mob *mobs.Mob, roomId int, routeId string) BoardResult
```

`Tick()` is the per-round entry point and is where all side effects live —
opening and closing the gangplank exit, emitting departures and arrivals, and
stepping the factors.

## Gotchas

- **Never persist vessel position.** Anything that caches `VesselState` across
  rounds reintroduces the desync this design exists to prevent. Recompute.
- **`hoursToRounds` truncates.** Schedules are accurate to ±1 round by design;
  do not add rounding "fixes" that make the two ports' cycles asymmetric.
- **`NextDockedRound` assumes `portIdx` is 0 or 1.** Its fallback return only
  makes sense for a two-port cycle. Passing anything else silently returns
  `fromRound`.
- **Gangplank exits are reconciled every tick, not event-driven.** If an exit
  looks wrong, the fix belongs in `reconcileGangplank`, not in a one-off emit.
- **Factor state is keyed by mob *instance* id.** A respawned factor is a new
  actor with no memory of its cargo run; `bankLeftoverCargo` exists to stop
  goods vanishing when that happens.

## Dependencies

`mobs`, `rooms`, `users`, `items`, `warehouse`, `events`, `gametime`,
`configs`, `mudlog`.

## Consumers

`internal/hooks` (per-round `Tick`), `internal/usercommands` (`board`, ferry
schedule enquiries), and the economy dashboard.
