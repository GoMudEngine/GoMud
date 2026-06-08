# Mob Aliveness 6.4 — Performance Review (initial)

**Status:** Design approved — pending implementation plan
**Roadmap chunk:** 6.4 (Phase 6 / Polish), Size S, depends on 6.3 (Done)
**Date:** 2026-06-05

## Goal

Establish an **attributable** performance baseline for the aliveness substrate
and its per-tick work, so that 6.6 (the post-content-pass re-review) can localize
any regression to a specific store or tick seam rather than a single
undifferentiated number.

This chunk **only measures**. It adds instrumentation and captures a baseline.
It does NOT optimize anything — performance fixes, if the numbers warrant them,
belong to 6.6.

## Why now

6.4's whole purpose is to capture the baseline *before* the XL content pass (6.5)
scales the world up. Measuring after 6.5 would mean comparing against nothing.

## Current state (what already exists)

The engine already has the two measurement primitives we need, both surfaced by
the `server stats` admin command:

- **Tick timing** — `util.TrackTime(name, seconds)` accumulates into a
  `util.Accumulator` (avg / low / high / count / per-sec). Currently tracks
  `IdleMobs()`, `DoCombat::handlePlayerCombat()`, `World::handleMobCombat()`,
  `AutoSave`, `RoomMaintenance()`, `EphemeralRoomMaintenance()`,
  `events.ProcessEvents()`, `mapper.GetPath()`, and per-command timers
  (`usr-cmd[...]`, `mob-cmd[...]`).
- **Memory** — `util.AddMemoryReporter(name, fn)` registers a per-section
  footprint reporter, shown in `server stats`. Currently registered: `Go`
  (runtime MemStats), `Users`, `Items`, `Mobs`, `Rooms`.
- **pprof** — heap profile viewable via `make view_pprof_mem`.
- **Coarse prod baseline** — the `reference_prod_perf_baseline` memory tracks
  pull/restart time + idle CPU per prod push.

### The gap

- **No substrate package registers a memory reporter.** opinions, factions,
  crimes, knowledge, bounties, facts, relationships, and goals are all invisible
  in `server stats` — they report 0.
- **All aliveness per-tick work is lumped.** `IdleMobs()` runs the schedule
  executor, patrol executor, and conversation tick/trigger synchronously under a
  single timer. The goal planner does NOT run in `IdleMobs()` — line 178 of
  `internal/hooks/NewRound_IdleMobs.go` only *queues* a `MobIdle` event;
  `RunGoalPlanner` fires later under the already-lumped `events.ProcessEvents()`
  timer. So no individual aliveness cost is attributable today.

That gap is exactly what would leave 6.6 unable to say *what* grew.

## Design

### 1. Memory instrumentation (substrate footprint)

Add a `memory.go` to each of the 8 substrate packages, mirroring the established
`internal/{users,items,mobs,rooms}/memory.go` pattern: a
`GetMemoryUsage() map[string]util.MemoryResult` function plus an `init()` that
calls `util.AddMemoryReporter`.

Packages: `opinions`, `factions` (rep store), `crimes`, `knowledge`, `bounties`,
`facts` (+ awareness store), `relationships`, `goals`.

Each reporter returns the in-memory size (via `util.MemoryUsage` reflection
sizing, as the existing reporters do) and an entry count for its primary
store(s). Where a package holds more than one store (e.g. facts registry +
awareness; factions rep + definitions), report them as separate named rows so
6.6 can see which half grew.

Result: `server stats` shows each substrate store's in-memory size + entry count.

### 2. Tick-budget instrumentation (sub-timers)

Break the lumped timers into named seams with `util.TrackTime`, keeping the
existing roll-up timers intact:

- Inside `IdleMobs()` (`internal/hooks/NewRound_IdleMobs.go`):
  - `IdleMobs::schedule` — wraps the schedule executor block
    (`scheduleTickPlan` + `applySchedulePlan`).
  - `IdleMobs::patrol` — wraps the patrol executor block
    (`patrolTickPlan` + `applyPatrolPlan`).
  - `IdleMobs::conversation` — wraps the conversation tick + trigger block.
- The `MobIdle` event handler:
  - `MobIdle::goalplanner` — wraps `RunGoalPlanner`.
- Justice/enforcement per-round tick (`internal/hooks/NewRound_MobRoundTick.go`),
  if not already split out:
  - `Enforcement` — wraps the per-round justice/enforcement work.

The existing lumped `IdleMobs()` and `events.ProcessEvents()` timers remain as
roll-ups so we keep both the total and the breakdown.

Sub-timers measure only the wrapped block, accumulated across all mobs in the
tick (i.e. wrap the per-mob block and let the accumulator sum), matching how the
existing per-command timers work.

### 3. Capture procedure (two conditions)

Both procedures are documented verbatim in the deliverable so 6.6 re-runs them
identically.

**Idle floor (reproducible regression baseline):**
1. Wipe instance saves per the SOP
   (`rm -rf _datafiles/world/dogmud/mobs.instances/* rooms.instances/*`).
2. Boot locally with the real world (Stillwater + Thornwall aliveness live).
3. Idle-tick **500 rounds** (~33 min game time) with no players connected —
   enough for schedules to cross segment boundaries, conversations to fire and
   cool down, and goal planners to cycle, so accumulator averages stabilize.
4. Snapshot `server stats` (timer table + memory report).

**Under load:**
1. Same boot.
2. Run one `test-mud local feel-tester` AI walking the world.
3. Snapshot `server stats` at the same 500-round mark.

### 4. Deliverable

A living doc at `docs/perf/aliveness-perf-baseline.md` containing:
- The capture procedure (both conditions), verbatim and re-runnable.
- The two snapshots (idle + under-load) as tables: substrate memory footprint
  and tick-seam timings.
- A short narrative reading of the numbers (what's hot, what's negligible).

Plus a one-line cross-reference appended to the `reference_prod_perf_baseline`
memory pointing at the new doc, so the coarse prod log and the fine-grained
baseline find each other.

## Out of scope

- On-disk YAML byte accounting per store.
- Persistence-write timing (substrate save-call latency).
- A synthetic-load harness.

(These were the "Instrument broadly" option, declined to keep the chunk Size S.)

- Any optimization. 6.4 measures only.

## Testing / validation

- `go build` clean and full test suite green after instrumentation.
- Server boots cleanly past data-file loading (pre-push SOP boot test) — the new
  `memory.go` files and timer wraps must not change load behavior.
- `server stats` renders the new memory rows and timer seams without error.
- The new sub-timers' summed averages are sane relative to their roll-up parents
  (e.g. `IdleMobs::schedule` + `IdleMobs::patrol` + `IdleMobs::conversation`
  should not exceed the `IdleMobs()` total).

## Files touched (anticipated)

- New: `internal/opinions/memory.go`, `internal/factions/memory.go`,
  `internal/crimes/memory.go`, `internal/knowledge/memory.go`,
  `internal/bounties/memory.go`, `internal/facts/memory.go`,
  `internal/relationships/memory.go`, `internal/goals/memory.go`.
- Edit: `internal/hooks/NewRound_IdleMobs.go` (3 sub-timers),
  the `MobIdle` event handler (1 sub-timer),
  `internal/hooks/NewRound_MobRoundTick.go` (enforcement sub-timer, if needed).
- New: `docs/perf/aliveness-perf-baseline.md`.
- Memory: append pointer to `reference_prod_perf_baseline`.
- Context: update each touched package's `context.md` if it gains a `memory.go`
  (per the roadmap's per-chunk context.md rule — these are small additions, so a
  one-line "Memory Reporting" note per package).
