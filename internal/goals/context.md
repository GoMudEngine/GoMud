# Goals Package Context

## Overview

The `internal/goals` package is the strategic-layer substrate for the
mob aliveness roadmap (chunks 4.1, 4.2, 4.3). It owns:

- **Storage**: a persistent, per-mob-template list of typed `Goal`
  objects on disk under `_datafiles/world/dogmud/goals/`.
- **Selection**: a pure scoring pipeline (`Select`) that picks the
  mob's current highest-priority goal each round.
- **Registry**: a type registry (`RegisterGoalType`) where concrete
  goal implementations declare their `Predicate`, `ContextScore`,
  `ParamSchema`, conflict rules, and multi-instance flags.

The subpackage `internal/goals/catalog/` registers the 13 concrete
goal types introduced in chunk 4.3. Behavior-tree integration (chunk
4.4) and reactive seeding from events (chunk 4.5) are out of scope
here.

## File Layout

- **types.go** — all exported structs and error types: `Goal`,
  `MobGoals`, `GoalTypeMeta`, `ParamSchema`, `ErrBadParams`,
  `GoalDefault`, `ConflictError`, `AddResult`, `SelectReason`;
  also the function-type aliases `PredicateFn` and `ContextScoreFn`.
- **store.go** — CRUD API (`Add`, `Remove`, `Clear`, `GoalsOf`,
  `CurrentGoalOf`, `Recompute`) and the lazy-seed path
  (`loadOrLazyInit`, `seedFromArchetype`).
- **select.go** — pure selection logic (`Select`, `effectiveScore`,
  `effectiveContextMod`, `invokeContextScore`).
- **registry.go** — type registry (`RegisterGoalType`, `lookupMeta`,
  `LookupGoalType`, `ValidateSymmetry`).
- **lookup.go** — callbacks that decouple goals from behaviortree:
  `SetWeightsLookup`, `SetArchetypeDefaultsLookup` and their
  internal resolvers.
- **persistence.go** — disk I/O (`loadFromDisk`, `saveToDisk`,
  `goalsBaseDir`, `goalPath`), the in-memory cache, and `ClearCache`.
- **validation.go** — `ValidateParams` called by `Add` when a type
  has a declared `ParamSchema`.

## Key Types

### Goal
```go
type Goal struct {
    Id         string         `yaml:"id"`
    OwnerMobId int            `yaml:"-"`           // stamped at load, not persisted
    Type       string         `yaml:"type"`
    Priority   int            `yaml:"priority"`
    Params     map[string]any `yaml:"params,omitempty"`
    CreatedAt  time.Time      `yaml:"created_at"`
    ExpiresAt  time.Time      `yaml:"expires_at,omitempty"` // zero = never
}
```
Goals are immutable once added — updates go through `Remove` + `Add`.
`Id` is assigned by `Add` (format `"g<N>"`); callers must leave it empty.

### MobGoals
The on-disk shape per mob template:
```go
type MobGoals struct {
    MobId               int     `yaml:"mob_id"`
    NextGoalId          int     `yaml:"next_goal_id"`
    CurrentGoalId       string  `yaml:"current_goal_id,omitempty"`
    CurrentSinceRound   uint64  `yaml:"current_since_round,omitempty"`
    LastSwitchRound     uint64  `yaml:"last_switch_round,omitempty"`
    SeededFromArchetype bool    `yaml:"seeded_from_archetype,omitempty"`
    Goals               []*Goal `yaml:"goals"`
}
```
`SeededFromArchetype` is the lazy-seed sentinel (see below).
`CurrentGoalId` and the round fields are the 4.2 selection state.

### GoalTypeMeta
```go
type GoalTypeMeta struct {
    Predicate     PredicateFn
    ConflictsWith []string
    ContextScore  ContextScoreFn
    Params        []ParamSchema        // nil = no schema validation
    AllowMultiple bool                 // false = only one goal of this type per mob
    DedupKey      func(g *Goal) string // required when AllowMultiple is true
}
```

### ParamSchema
```go
type ParamSchema struct {
    Key      string
    Required bool
    GoType   string // "int" | "string" | "[]string" | "float64" | "bool"
}
```
Registered per-type. `ValidateParams` runs inside `Add` and returns
`*ErrBadParams` on type mismatch or missing required key.

### SelectReason
```go
type SelectReason struct {
    Kind   string // e.g. "switched", "kept_hysteresis_margin"
    Detail string // human-readable, e.g. "g3(80) beat g1(70) by 10pts"
}
```
Emitted in the `goals.switch` structured log line from `Recompute`.
Surfaced by the `goal scores` admin command.

### GoalDefault
```go
type GoalDefault struct {
    Type     string
    Priority int
    Params   map[string]any
}
```
Goals-package mirror of `behaviortree.GoalDefault`. Kept separate to
avoid an import cycle; bridged at boot via `SetArchetypeDefaultsLookup`.

### Error Types
- **`ErrBadParams`** — `Add` rejects goals whose params don't match the
  registered `ParamSchema`. Carries `Key`, `ExpectedType`, `GotType`,
  optional `Reason` ("missing required key").
- **`ConflictError`** — `Add` returns this when a lower-or-equal-priority
  conflicting goal already exists. Carries `BlockerGoalId`, `BlockerType`,
  `BlockerPrio`.
- **`ErrGoalNotFound`** — `Remove` returns this sentinel when the goal id
  is not present. Tests match it via `errors.Is`.

## Core APIs

### CRUD (store.go)
- **`Add(mobId int, namesimple string, g *Goal) (AddResult, error)`**
  Validates params, resolves conflicts, assigns id, persists, triggers
  eager `Recompute`. Returns `*ConflictError` if blocked, `*ErrBadParams`
  if params invalid. Thread-safe.
- **`Remove(mobId int, namesimple, goalId string) error`**
  Removes by id, clears selection state if the goal was current,
  persists, triggers eager `Recompute`.
- **`Clear(mobId int, namesimple string) error`**
  Removes all goals, resets `NextGoalId` to 1, zeros selection state.
  Does NOT clear `SeededFromArchetype` — the sentinel survives `Clear`
  so archetype-seeded defaults are not re-seeded after an admin wipe.
- **`GoalsOf(mobId int, namesimple string) []*Goal`**
  Returns a snapshot copy sorted by priority-desc, id-asc. Lazy-loads
  from disk on first call.
- **`CurrentGoalOf(mobId int, namesimple string) *Goal`**
  Returns the cached current goal or nil if stale/absent.
- **`Recompute(mobId int, namesimple string, mob *mobs.Mob, nowRound uint64)`**
  Snapshots goals, runs `Select`, and on a switch updates `CurrentGoalId`
  + round fields under the write lock, persists, and logs at debug.
  Safe to call with a nil mob.

### Selection (select.go)
- **`Select(goals, weights, mob, prev, sinceRound, switchRound, nowRound) (current, switched, reason)`**
  Pure, lock-free, side-effect-free. Filters by satisfaction/expiry/
  contextMod=0, scores as `priority × archetypeWeight × contextMod`,
  applies hysteresis (config knobs: `GoalSelectSwitchMargin`,
  `GoalSelectMinHoldRounds`). Returns the `SelectReason` explaining the
  outcome.

### Registry (registry.go)
- **`RegisterGoalType(name string, meta GoalTypeMeta)`**
  Called from each catalog `init()`. Overwrites + warns on duplicate.
- **`LookupGoalType(name string) (GoalTypeMeta, bool)`**
  Exported read accessor for catalog tests and admin tooling.
- **`ValidateSymmetry() []string`**
  Boot-time soft check: every `ConflictsWith` pair should be declared
  on both sides. Returns a slice of warning strings; does not panic.

### Callbacks (lookup.go)
- **`SetWeightsLookup(fn WeightsLookupFn)`**
  Registers `func(mob) map[string]float64`. Nil = no archetype weights
  (all goals score at their raw priority). Called once at boot.
- **`SetArchetypeDefaultsLookup(fn ArchetypeDefaultsLookupFn)`**
  Registers `func(mob) []GoalDefault`. Nil = no seeding. Called once
  at boot. Bridges goals → behaviortree without an import cycle.

## Persistence

Files live at `_datafiles/world/dogmud/goals/{mobId}-{namesimple}.yaml`.
`namesimple` is passed in by the caller (usually the mob's
`NameSimple` field); the path uses `util.ConvertForFilename(namesimple)`.

Write path: atomic `.tmp` + `os.Rename` under `saveMu sync.Mutex`
(prevents `ERROR_SHARING_VIOLATION` on Windows). The in-memory cache
remains authoritative even if a write fails — `mudlog.Warn` is emitted
but the mutation is not rolled back.

These files are not deployed to production. On prod the directory is
absent, so every mob starts fresh and is seeded from archetype defaults
on first access.

## Concurrency

```go
var (
    cacheMu     sync.RWMutex   // guards cache + nameByMobId
    cache       = map[int]*MobGoals{}
    nameByMobId = map[int]string{}
    saveMu      sync.Mutex     // serialises disk writes
    registryMu  sync.RWMutex  // guards typeRegistry
    lookupMu    sync.RWMutex  // guards weightsLookup + archetypeDefaultsLookup
)
```

`Select` is lock-free: `Recompute` snapshots goals under the read lock,
then calls `Select` on the snapshot outside the lock. The write lock is
only held for the `CurrentGoalId` / round update that follows.
`ContextScoreFn` and `PredicateFn` implementations must be safe to call
from multiple goroutines simultaneously.

## Lazy Seeding (chunk 4.3)

On `loadOrLazyInit` for a mob with no disk file:

1. A blank `MobGoals` is created and stored in `cache[mobId]`.
2. The write lock is released.
3. `seedFromArchetype(mobId, namesimple, mg)` runs outside the lock.
4. `seedFromArchetype` calls `resolveArchetypeDefaults(mob)` (via the
   registered `ArchetypeDefaultsLookupFn`) and calls `Add` for each
   default goal. `Add` re-entrancy is safe because the cache entry
   already exists.
5. `SeededFromArchetype` is flipped to `true` and the file is persisted.
   The sentinel is set regardless of whether any defaults exist — it
   records "seeding was attempted."

`Clear` preserves `SeededFromArchetype`. After a clear, the mob has no
goals but the sentinel prevents re-seeding — the admin wipe is permanent
until an explicit `goal add` or server restart with a wiped file.

## Integration Points

### Archetype YAML
Archetype YAML files under
`_datafiles/world/dogmud/behaviors/<archetype>.yaml` may declare a
`default_goals:` block (parsed by `behaviortree.LoadArchetypeYAMLFromFile`
in chunk 4.3). This feeds the lazy-seed path via the
`SetArchetypeDefaultsLookup` callback.

### Per-Round Tick Hook
The goals tick hook (`internal/hooks/NewRound_GoalsRecompute.go`,
wired in chunk 4.3) calls `Recompute` for each loaded mob instance.
It passes the live `*mobs.Mob` so `ContextScoreFn` implementations
can read current mob state.

### Admin Commands
`internal/usercommands/admin.goal.go` exposes:
```
goal show <mobId>       — all goals on file
goal current <mobId>    — current goal + selection state
goal scores <mobId>     — scoring table (priority × weight × contextMod)
goal add <mobId> ...    — manual add
goal remove <mobId> <id>
goal clear <mobId>
```

## Memory Reporting

`memory.go` registers a `util.AddMemoryReporter` under the section name
`Goals`, surfacing the in-memory store size in the `server stats` admin
command (mob-aliveness chunk 6.4). Two rows are reported: `cache`
(number of mob templates with a loaded goal file) and `nameByMobId`
(mob-name lookup map size).

## Out of Scope

- **Behavior-tree integration (4.4)**: btree actions reading
  `CurrentGoalOf` to steer mob behavior are not wired yet.
- **Reactive seeding from events (4.5)**: hooks that call `Add` when
  a mob's ally is killed, a theft is witnessed, etc.
- **Satisfaction sweep (4.6)**: periodic removal of satisfied/expired
  goals from disk to keep files compact.
- **Cross-type conflict mechanism**: `ConflictsWith` is declared on
  `GoalTypeMeta` and checked symmetrically by `isConflict`, but the
  4.3 spec (§7) defers the full design; no cross-type conflicts are
  declared in the shipped 4.3 catalog.
