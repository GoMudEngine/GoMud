# Mob Aliveness 4.1 — Goal Representation Design

**Date:** 2026-05-26
**Status:** Approved (design)
**Roadmap position:** Phase 4 (strategic layer) foundation. After 4.1: 23/42.
**Next chunks:** 4.2 Goal selection → 4.3 Concrete goal catalog → 4.4 Goal→tactical wiring.

---

## Summary

Add a substrate package `internal/goals/` that gives every mob a persistent,
queryable list of typed goals. Each goal is a typed struct with a free-form
`Params` map, owned by a mob template id, scored by `Priority`, optionally
time-limited via `ExpiresAt`, and evaluated lazily via a per-type predicate
that consumers register at boot.

4.1 ships the data model, persistence layer, type-metadata + conflict
registry, public API, and an `admin goal …` command. It ships with an
**empty** predicate / conflict registry — 4.3 fills it. No behavior-tree
integration yet (4.4's job).

The chunk is "no observable behavior change" by design. Goals exist, can be
added, can be queried, persist across restarts; nothing yet reads them to
drive tactical decisions.

---

## Goals

- Provide a single source of truth for "what does this NPC want?"
- Persist goals across mob death / unload / server restart, keyed by mob
  template id (so a respawned mob inherits the goals of its predecessor —
  matches opinion / knowledge / fact-store semantics).
- Support multiple goals per mob; resolve conflicts at Add-time by priority.
- Expose admin tooling for inspection and authored seeding.
- Stay decoupled: 4.1 owns shape + persistence + API; downstream chunks
  fill in semantics (predicates, selection, generation, wiring).

## Non-goals

- Goal selection (which goal a mob acts on right now) — 4.2.
- Concrete goal types and their predicates — 4.3.
- Behavior-tree integration / tactical execution — 4.4.
- Reactive goal generation from events — 4.5.
- Automatic satisfaction scanning — 4.6.
- Cross-mob shared goals (faction objectives, etc.) — out of Phase 4 scope.

---

## 1. Storage shape & schema

### 1.1 In-memory struct

```go
package goals

// Goal is one strategic intent owned by a mob template.
//
// Immutable once added — "updates" are Remove + Add new.
type Goal struct {
    Id          string         // sequential per-mob string ("g1", "g2", ...); never reused
    OwnerMobId  int            // mob TEMPLATE id (not instance id)
    Type        string         // registry key, e.g. "revenge", "wealth-target"
    Priority    int            // 0..100; ties broken by Id (older = lower number wins)
    Params      map[string]any // type-specific payload; YAML-friendly types
    CreatedAt   time.Time      // for audit / future TTL grace
    ExpiresAt   time.Time      // zero value = never expires
}
```

`Params` allowed value types are restricted to YAML-round-trippable Go
kinds: `string`, `int`, `int64`, `float64`, `bool`, `[]any`, `map[string]any`.
The persistence layer rejects others with a logged warning on save (drops
the goal) and on load (skips the entry). 4.3's catalog will document the
expected param schema per goal type, but 4.1 does not enforce it — that's
a 4.3 concern bound to the predicate registration.

### 1.2 On-disk layout

```
_datafiles/world/dogmud/goals/<mobid>-<namesimple>.yaml
```

Flat directory, matching the existing opinion / knowledge / facts-awareness
stores. `namesimple` runs through `util.ConvertForFilename`. The file
itself carries the `mob_id` and the per-mob `next_goal_id` counter
(monotonic; never reused across the lifetime of this mob's file).

Empty `goals:` list = "this mob has no goals" (file still written so disk
presence indicates the mob is goal-tracked).

File contents:

```yaml
mob_id: 371
next_goal_id: 3
goals:
  - id: g1
    type: revenge
    priority: 70
    params:
      target_player_name: smoketester
      reason: "killed brother"
      observed_round: 12345
    created_at: 2026-05-26T14:30:00Z
    expires_at: 2026-06-25T14:30:00Z
  - id: g2
    type: wealth-target
    priority: 30
    params:
      amount: 500
    created_at: 2026-05-26T14:31:00Z
```

`OwnerMobId` is not stored in each `Goal` entry — the top-level `mob_id`
covers it; the load path stamps each Goal's `OwnerMobId` after read.

### 1.3 Gitignore

`_datafiles/world/dogmud/goals/` is gitignored, matching the existing
runtime-data stores (opinions, knowledge, facts, shops). Authored seed
goals (if any future chunk wants them) live elsewhere and are loaded into
the runtime store at boot.

---

## 2. Public API surface

```go
package goals

// ConflictError is returned by Add when an existing goal blocks the new
// one (i.e. new.Priority <= existing.Priority on a conflicting type).
// Carries the first blocking goal's id so the admin command can surface
// it in the "Blocked by ..." output.
type ConflictError struct {
    BlockerGoalId string
    BlockerType   string
    BlockerPrio   int
}

func (e *ConflictError) Error() string { return fmt.Sprintf(...) }

// PredicateFn evaluates whether a goal is currently satisfied.
// Predicates are pure(ish): same goal + same mob state → same answer.
// Side effects forbidden — IsSatisfied may be called from any context.
type PredicateFn func(g *Goal, mob *mobs.Mob) bool

// GoalTypeMeta is registered once per goal type at boot.
type GoalTypeMeta struct {
    Predicate     PredicateFn
    ConflictsWith []string // type names this goal type conflicts with
}

// Registry — called from each goal-type package's init() (chunk 4.3).
func RegisterGoalType(goalType string, meta GoalTypeMeta)

// AddResult reports what happened on Add — useful for the admin command's
// "displaced X, Y" output.
type AddResult struct {
    Added     *Goal    // the newly-added goal (its assigned Id is set)
    Displaced []string // goal ids removed because new one preempted
}

// Add appends a goal to the mob's list, resolving conflicts by priority.
// Returns a *ConflictError if any conflicting existing goal has priority
// >= the new goal's priority. Persists to disk under the write mutex.
func Add(g *Goal) (AddResult, error)

// Remove deletes a goal by id. Idempotent — no-op + nil if id missing.
func Remove(ownerMobId int, goalId string) error

// Clear removes all goals from a mob. Used by the admin command.
func Clear(ownerMobId int) error

// GoalsOf returns the mob's goals in priority-desc, ULID-asc order
// (stable for admin output and any future selection layer).
// Lazy-loaded on first access per mob; cached thereafter.
func GoalsOf(ownerMobId int) []*Goal

// IsSatisfied looks up the registered predicate for g.Type and invokes it.
// Returns false (treated as still-pending) if no predicate registered.
func IsSatisfied(g *Goal, mob *mobs.Mob) bool

// IsExpired is a pure time check. Goals with ExpiresAt.IsZero() never expire.
func IsExpired(g *Goal, now time.Time) bool
```

Concurrency: the package holds a single `sync.RWMutex` guarding the in-memory
cache. Reads use RLock; mutations + persistence use Lock and hold it for the
duration of the disk write. Matches the opinions / knowledge stores.

---

## 3. Admin command + predicate registry

### 3.1 Command surface

`goal` admin command (`internal/usercommands/admin.goal.go`), registered
under the `goal` keyword in `usercommands.go`'s command table with the
admin-only flag triple (matching `opinion`):

```
goal list <mob-ident>
goal show <mob-ident> <goal-id>
goal add <mob-ident> <type> <priority> [key=value ...]
goal remove <mob-ident> <goal-id>
goal clear <mob-ident>
```

- `<mob-ident>` is a numeric mob template id OR a name (matched via
  `util.ConvertForFilename` against `mobs.AllMobTemplates()`). The
  resolution helper lives next to the command and mirrors
  `opinionResolveMobIdent` shape (returns `(id int, displayName string,
  ok bool)`).
- `<goal-id>` is the exact short id (e.g. `g3`). No prefix matching —
  ids are short by design.
- `add`'s `key=value` pairs build `Params`. Values parsed as scalars:
  `42` → int, `3.14` → float, `true`/`false` → bool, otherwise string.
  Quoting not required.
- `add` output:
  - On success: `Added goal <id> (type=<t>, priority=<p>)`.
  - With displacement: `... — displaced goals: <id1>, <id2>`.
  - On conflict: `Blocked by goal <id> (type=<t>, priority=<p>)`.

### 3.2 Predicate registry mechanics

- `RegisterGoalType` is called from each goal-type package's `init()`.
  4.1 ships with no registrations. 4.3 fills the registry.
- `IsSatisfied(g, mob)` looks up `g.Type`. Missing type → returns false
  (i.e. "not satisfied / still pending"). This is the safe default: a goal
  the engine doesn't know how to evaluate stays alive rather than getting
  treated as done.
- Symmetric-conflict validation: at boot, after all packages have called
  `RegisterGoalType`, the package walks every registration and checks that
  each `ConflictsWith` entry's target also lists the source. Mismatches log
  a single-line warning per direction. Soft check — doesn't panic, doesn't
  block startup.
- Re-registration of an existing type logs a warning and overwrites. This
  is mostly defensive; in practice packages register once.

---

## 4. Contingencies & edge cases

### 4.1 Concurrent goal mutation
Per-package `sync.RWMutex` guarding cache and persistence. Goal mutations
serialize per-package, not per-mob; mob counts are low enough that a
single mutex is fine.

### 4.2 Mob template id vs instance id
Goals owned by template id, persisted under the template id's filename.
The admin command takes `<mob-ident>` (numeric template id OR namesimple)
and resolves to the template id via an opinion-style helper. Matches the
chunk 1.4 knowledge-store / opinions-store pattern.

### 4.3 Predicate registered after goals already exist
Order of registration vs goal creation is independent. Predicates are looked
up by `g.Type` at `IsSatisfied()` call time, not stamped on the Goal. If
no predicate is registered at call time, `IsSatisfied` returns false.

### 4.4 Goal file corruption
Unparseable YAML → log a warning, return an empty goal list for that mob,
continue boot. Acceptable loss (4.5 reactive generation will re-populate
over time once it exists). Always write to `.tmp` then `os.Rename` to
prevent partial-write corruption from crashes mid-write.

### 4.5 Orphaned owner mob id
If a goal file references a mob template id that no longer exists in the
mob registry (deleted template), `GoalsOf` still returns the goals, but the
admin command surfaces a warning when listing. Cleanup of orphans is a 4.6
concern.

### 4.6 Disk-full / write errors
Log a warning, keep the in-memory cache live, return success to the caller
of `Add`/`Remove`/`Clear`. The next mutation retries the write. Graceful
shutdown attempts a final flush. No silent data loss path.

---

## 5. Goal type metadata + conflict resolution

### 5.1 Declaration

Each goal-type registration carries both the satisfaction predicate AND its
conflict-list:

```go
goals.RegisterGoalType("revenge", goals.GoalTypeMeta{
    Predicate:     revengePredicate,
    ConflictsWith: []string{"protection"},
})
```

Symmetric pairs are authored explicitly (`revenge` ↔ `protection`,
`wealth-hoard` ↔ `wealth-spend`, etc.). The boot validation in §3.2 catches
missing-direction authoring drift.

### 5.2 Conflict scan in Add()

On `Add(newGoal)`:

1. Load `meta := lookupMeta(newGoal.Type)`.
2. Iterate existing goals on the same mob. Mark `existing` as conflicting if:
   - `existing.Type == newGoal.Type` (default same-type self-conflict; future
     `AllowMultiple bool` field per type can opt out), OR
   - `meta.ConflictsWith` lists `existing.Type`, OR
   - `lookupMeta(existing.Type).ConflictsWith` lists `newGoal.Type` (symmetry
     safety net so a one-sided declaration still works).
3. **Priority-resolved policy:**
   - If `newGoal.Priority > existing.Priority` for **every** conflicting
     existing goal → call `Remove` on each, then append `newGoal`. Return
     `AddResult{Added: newGoal, Displaced: [...]}`.
   - If `newGoal.Priority <= existing.Priority` for **any** conflicting
     existing goal → return `(AddResult{}, &ConflictError{...})` carrying
     the first blocker's id / type / priority so the admin command can
     render the "Blocked by goal <id> (type=<t>, priority=<p>)" line.

### 5.3 Goal immutability

Goals are immutable once added. To "update" — bump priority, change params —
the caller does `Remove(old)` then `Add(new)`. Simpler persistence, no
in-place-mutation races, mirrors the existing crime/fact stores.

---

## 6. Testing strategy & rollout

### 6.1 Unit tests (`internal/goals/*_test.go`)

- `Goal` YAML round-trip: marshal → unmarshal → DeepEqual. Cover all
  permitted `Params` value types (string, int, float64, bool, []any,
  nested map).
- `Add` happy path: appends, persists, returns assigned id.
- `Add` priority-resolved displacement: higher priority displaces lower
  priority same-type, returns displaced ids in `AddResult`.
- `Add` blocked: lower-or-equal priority returns `*ConflictError` with
  blocker id / type / priority populated; existing goal untouched on disk.
- `Add` same-type self-conflict: second `revenge` rejected without opt-in.
- `Add` symmetric-conflict safety net: works even when only one direction
  is registered (the other direction's `lookupMeta` returns zero `meta`
  and we still detect the conflict via the registered side).
- `Remove` happy path + idempotent missing-id.
- `GoalsOf` returns priority-desc, ULID-asc order.
- `IsSatisfied` returns false when no predicate registered.
- `IsSatisfied` calls the registered predicate with the right args and
  returns its result verbatim.
- `IsExpired` pure time check, including `ExpiresAt.IsZero()` → never.
- Symmetric-conflict warning: register A→B with no B→A, observe the
  warning logged (test hook on the logger).
- Lazy load: first `GoalsOf` reads disk; second call serves from cache
  without disk hit (FS-spy fake).
- Atomic write failure path: simulate `os.Rename` failure, assert existing
  on-disk file intact and in-memory cache reflects attempted change with a
  logged warning.
- Per-package mutex under `-race`: two goroutines mutating the same mob
  serialize without data race.

### 6.2 Admin command tests
(`internal/usercommands/admin_goal_test.go`) — table-driven over `list`,
`show`, `add`, `remove`, `clear`, including the displaced-goal output line,
the blocked-by line, and `<goal-id-prefix>` ambiguity handling.

### 6.3 Boot validation
Registry-validation pass runs at server boot (already exercised by
`go test ./...` via package init). Pre-push SOP boot-test catches it
naturally if a future authoring change breaks symmetry.

### 6.4 Out of scope for 4.1 tests
- Predicate behavior — that's 4.3.
- Selection — 4.2.
- Reactive generation — 4.5.
- Behavior-tree integration — 4.4.
- Full-cycle smoke (admin sets a goal → btree acts on it) — needs 4.4
  to even be observable. 4.1's smoke is the admin command exercise.

### 6.5 Rollout — feature branch decomposition

Single chunk on `feature/aliveness-4.1-goal-representation`, dispatched via
subagent-driven-development. Task ordering:

1. **Goal struct + ULID id generator + YAML round-trip.** Pure data type.
2. **Per-mob YAML persistence** (`internal/goals/persistence.go`) — atomic
   write, lazy load, RWMutex; gitignore entry for
   `_datafiles/world/dogmud/goals/`.
3. **Goal type metadata registry** — `RegisterGoalType`, lookup,
   symmetric-conflict boot warning. Empty registry at 4.1 ship.
4. **Add / Remove / Clear / GoalsOf / IsSatisfied / IsExpired** —
   wraps 1–3, applies conflict resolution at Add().
5. **Admin command `goal …`** (`internal/usercommands/admin_goal.go`) —
   list/show/add/remove/clear; instance-id → template-id resolution;
   output formatting per §3.1; key=value param parsing.
6. **Smoke checklist** — pre-push boot test passes; in-game admin command
   exercise on a test mob (add → list → show → remove → clear), verifying
   YAML appears under `_datafiles/world/dogmud/goals/<zone>/` after add
   and persists across server restart.

No data migrations, no config knobs (the predicate registry is the only
"config"), no behavior-tree integration. Push to prod is safe — substrate
goals do nothing observable.

### 6.6 Roadmap rollup after 4.1 ships

23/42. Next: 4.2 Goal selection (pick the active goal when multiple exist)
→ 4.3 Concrete goal catalog (where the predicate + conflict registry gets
populated with revenge, debt, wealth-target, status, etc.).

---

## File touch list

**New:**
- `internal/goals/types.go` — `Goal`, `MobGoals` (per-mob file struct),
  `GoalTypeMeta`, `PredicateFn`, `AddResult`, `ConflictError`.
- `internal/goals/registry.go` — `RegisterGoalType`, lookup,
  symmetric-conflict validation.
- `internal/goals/store.go` — `Add`, `Remove`, `Clear`, `GoalsOf`,
  `IsSatisfied`, `IsExpired`. Cache + mutex.
- `internal/goals/persistence.go` — load / save / atomic-rename plumbing,
  `DOGMUD_GOALS_DIR_OVERRIDE` test seam.
- `internal/goals/test_main_test.go` — temp-dir setup mirroring
  opinions/knowledge.
- `internal/goals/types_test.go`, `registry_test.go`, `store_test.go`,
  `persistence_test.go`.
- `internal/usercommands/admin.goal.go` + `_test.go`.

**Modified:**
- `.gitignore` — add `_datafiles/world/dogmud/goals/`.
- `internal/usercommands/usercommands.go` (or wherever the admin command
  table lives) — register `goal` admin command.
- `MOB_ALIVENESS_ROADMAP.md` — mark 4.1 done at ship time, rollup 23/42.

**Not touched in 4.1:** behavior-tree, mob struct, hooks, configs.
