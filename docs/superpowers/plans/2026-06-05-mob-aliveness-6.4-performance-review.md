# Mob Aliveness 6.4 — Performance Review (initial) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the 8 aliveness substrate stores and the per-tick aliveness work visible in the existing `server stats` admin command, then capture a reproducible 500-round idle + under-load baseline.

**Architecture:** Reuse the engine's two existing perf primitives — `util.AddMemoryReporter(name, fn)` (per-section memory, shown in `server stats`) and `util.TrackTime(name, seconds)` (named timing accumulators, shown in `server stats`). Add one `memory.go` per substrate package mirroring `internal/{users,mobs,items,rooms}/memory.go`, and wrap five tick seams with `TrackTime`. No new frameworks, no behavior changes.

**Tech Stack:** Go, the GoMud engine's `internal/util` perf helpers.

**Spec:** `docs/superpowers/specs/2026-06-05-mob-aliveness-6.4-performance-review-design.md`

---

## Reference: verified store shapes

The memory reporters access these package-level vars directly (each `memory.go`
lives in the same package as the store, so unexported access is fine):

| Package | Store var(s) | Type | Mutex (RLock) |
|---------|-------------|------|---------------|
| `opinions` | `opinionCache`, `nameByMobId` | `map[int]*MobOpinions`, `map[int]string` | `opinionCacheMu` |
| `factions` | `definitions` | `map[string]*Definition` | `definitionsMu` |
| `factions` | `repCache` | `map[string]*FactionRep` | `repCacheMu` |
| `crimes` | `crimeCache` | `map[string]*FactionCrimes` | `crimeCacheMu` |
| `knowledge` | `knowledgeCache` | `map[int]*ObserverFile` | `knowledgeCacheMu` |
| `bounties` | `registry` (nil-able `*Registry`; rows in `registry.Bounties`) | `*Registry` | `registryMu` |
| `facts` | `registry` (nil-able `*Registry`; rows in `registry.Facts`) | `*Registry` | `registryMu` |
| `facts` | `awarenessCache` | `map[int]*Awareness` | `awarenessCacheMu` |
| `relationships` | `graph` | `map[int][]Relation` | `graphMu` |
| `goals` | `cache`, `nameByMobId` | `map[int]*MobGoals`, `map[int]string` | `cacheMu` |

The reporter pattern (from `internal/mobs/memory.go`):

```go
package mobs

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}
	mobsMu.RLock()
	ret["mobs"] = util.MemoryResult{Memory: util.MemoryUsage(mobs), Count: len(mobs)}
	mobsMu.RUnlock()
	return ret
}

func init() {
	util.AddMemoryReporter(`Mobs`, GetMemoryUsage)
}
```

## Reference: verified tick seams

| Seam timer | File | Site |
|-----------|------|------|
| `IdleMobs::schedule` | `internal/hooks/NewRound_IdleMobs.go` | schedule executor block (currently lines 73-76) |
| `IdleMobs::patrol` | `internal/hooks/NewRound_IdleMobs.go` | patrol executor block (currently lines 83-96) |
| `IdleMobs::conversation` | `internal/hooks/NewRound_IdleMobs.go` | conversation tick + trigger block (currently lines 101-116) |
| `MobIdle::goalplanner` | `internal/hooks/MobIdle_HandleIdleMobs.go` | `behaviortree.RunGoalPlanner(...)` call (currently line 236) |
| `Enforcement` | `internal/hooks/NewRound_MobRoundTick.go` | `justice.RunGuardEnforcement(...)` call (currently line 117) |

**Timer-denominator convention (document in the baseline doc):**
- The three `IdleMobs::*` seams are accumulated into per-tick totals and recorded
  **once per tick** (matching the lumped `IdleMobs()` parent, so the three seams
  should sum to ≤ `IdleMobs()`).
- `MobIdle::goalplanner` and `Enforcement` are recorded **per invocation**
  (per mob), because those calls live in per-mob handlers, not the per-tick loop.

---

## Task 1: Map-based substrate memory reporters

Six packages whose stores are plain maps (cannot nil-panic): `opinions`,
`factions`, `crimes`, `knowledge`, `relationships`, `goals`. No unit tests —
the codebase's existing memory reporters (`users`/`mobs`/`items`/`rooms`) ship
without tests; these are trivial glue exercised end-to-end by Task 7's
`server stats` render. Verification is `go build`.

**Files:**
- Create: `internal/opinions/memory.go`
- Create: `internal/factions/memory.go`
- Create: `internal/crimes/memory.go`
- Create: `internal/knowledge/memory.go`
- Create: `internal/relationships/memory.go`
- Create: `internal/goals/memory.go`

- [ ] **Step 1: Write `internal/opinions/memory.go`**

```go
package opinions

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}
	opinionCacheMu.RLock()
	ret["opinionCache"] = util.MemoryResult{Memory: util.MemoryUsage(opinionCache), Count: len(opinionCache)}
	ret["nameByMobId"] = util.MemoryResult{Memory: util.MemoryUsage(nameByMobId), Count: len(nameByMobId)}
	opinionCacheMu.RUnlock()
	return ret
}

func init() {
	util.AddMemoryReporter(`Opinions`, GetMemoryUsage)
}
```

- [ ] **Step 2: Write `internal/factions/memory.go`**

```go
package factions

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}
	definitionsMu.RLock()
	ret["definitions"] = util.MemoryResult{Memory: util.MemoryUsage(definitions), Count: len(definitions)}
	definitionsMu.RUnlock()
	repCacheMu.RLock()
	ret["repCache"] = util.MemoryResult{Memory: util.MemoryUsage(repCache), Count: len(repCache)}
	repCacheMu.RUnlock()
	return ret
}

func init() {
	util.AddMemoryReporter(`Factions`, GetMemoryUsage)
}
```

- [ ] **Step 3: Write `internal/crimes/memory.go`**

```go
package crimes

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}
	crimeCacheMu.RLock()
	ret["crimeCache"] = util.MemoryResult{Memory: util.MemoryUsage(crimeCache), Count: len(crimeCache)}
	crimeCacheMu.RUnlock()
	return ret
}

func init() {
	util.AddMemoryReporter(`Crimes`, GetMemoryUsage)
}
```

- [ ] **Step 4: Write `internal/knowledge/memory.go`**

```go
package knowledge

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}
	knowledgeCacheMu.RLock()
	ret["knowledgeCache"] = util.MemoryResult{Memory: util.MemoryUsage(knowledgeCache), Count: len(knowledgeCache)}
	knowledgeCacheMu.RUnlock()
	return ret
}

func init() {
	util.AddMemoryReporter(`Knowledge`, GetMemoryUsage)
}
```

- [ ] **Step 5: Write `internal/relationships/memory.go`**

```go
package relationships

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}
	graphMu.RLock()
	ret["graph"] = util.MemoryResult{Memory: util.MemoryUsage(graph), Count: len(graph)}
	graphMu.RUnlock()
	return ret
}

func init() {
	util.AddMemoryReporter(`Relationships`, GetMemoryUsage)
}
```

- [ ] **Step 6: Write `internal/goals/memory.go`**

```go
package goals

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}
	cacheMu.RLock()
	ret["cache"] = util.MemoryResult{Memory: util.MemoryUsage(cache), Count: len(cache)}
	ret["nameByMobId"] = util.MemoryResult{Memory: util.MemoryUsage(nameByMobId), Count: len(nameByMobId)}
	cacheMu.RUnlock()
	return ret
}

func init() {
	util.AddMemoryReporter(`Goals`, GetMemoryUsage)
}
```

- [ ] **Step 7: Build**

Run: `go build ./...`
Expected: clean, no errors.

- [ ] **Step 8: Commit**

```bash
git add internal/opinions/memory.go internal/factions/memory.go internal/crimes/memory.go internal/knowledge/memory.go internal/relationships/memory.go internal/goals/memory.go
git commit -m "feat(perf): memory reporters for map-based aliveness substrate stores"
```

---

## Task 2: Pointer-registry memory reporters (nil-safe) — bounties + facts

`bounties.registry` and `facts.registry` are `*Registry` pointers that are
**nil before first load**. The reporter must not panic on nil. This edge case
gets a unit test (TDD) since it's a real failure mode the map-based reporters
don't have.

**Files:**
- Create: `internal/bounties/memory.go`
- Create: `internal/bounties/memory_test.go`
- Create: `internal/facts/memory.go`
- Create: `internal/facts/memory_test.go`

- [ ] **Step 1: Write the failing test `internal/bounties/memory_test.go`**

```go
package bounties

import "testing"

func TestGetMemoryUsage_NilRegistry_NoPanic(t *testing.T) {
	registryMu.Lock()
	registry = nil
	registryMu.Unlock()

	got := GetMemoryUsage()

	row, ok := got["registry"]
	if !ok {
		t.Fatalf("expected a 'registry' row even when registry is nil")
	}
	if row.Count != 0 {
		t.Fatalf("expected Count 0 for nil registry, got %d", row.Count)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/bounties/ -run TestGetMemoryUsage_NilRegistry_NoPanic -v`
Expected: FAIL — `GetMemoryUsage` undefined.

- [ ] **Step 3: Write `internal/bounties/memory.go`**

```go
package bounties

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}
	registryMu.RLock()
	count := 0
	if registry != nil {
		count = len(registry.Bounties)
	}
	ret["registry"] = util.MemoryResult{Memory: util.MemoryUsage(registry), Count: count}
	registryMu.RUnlock()
	return ret
}

func init() {
	util.AddMemoryReporter(`Bounties`, GetMemoryUsage)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/bounties/ -run TestGetMemoryUsage_NilRegistry_NoPanic -v`
Expected: PASS.

- [ ] **Step 5: Write the failing test `internal/facts/memory_test.go`**

```go
package facts

import "testing"

func TestGetMemoryUsage_NilRegistry_NoPanic(t *testing.T) {
	registryMu.Lock()
	registry = nil
	registryMu.Unlock()

	got := GetMemoryUsage()

	row, ok := got["registry"]
	if !ok {
		t.Fatalf("expected a 'registry' row even when registry is nil")
	}
	if row.Count != 0 {
		t.Fatalf("expected Count 0 for nil registry, got %d", row.Count)
	}
	if _, ok := got["awarenessCache"]; !ok {
		t.Fatalf("expected an 'awarenessCache' row")
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./internal/facts/ -run TestGetMemoryUsage_NilRegistry_NoPanic -v`
Expected: FAIL — `GetMemoryUsage` undefined.

- [ ] **Step 7: Write `internal/facts/memory.go`**

```go
package facts

import "github.com/GoMudEngine/GoMud/internal/util"

func GetMemoryUsage() map[string]util.MemoryResult {
	ret := map[string]util.MemoryResult{}

	registryMu.RLock()
	count := 0
	if registry != nil {
		count = len(registry.Facts)
	}
	ret["registry"] = util.MemoryResult{Memory: util.MemoryUsage(registry), Count: count}
	registryMu.RUnlock()

	awarenessCacheMu.RLock()
	ret["awarenessCache"] = util.MemoryResult{Memory: util.MemoryUsage(awarenessCache), Count: len(awarenessCache)}
	awarenessCacheMu.RUnlock()

	return ret
}

func init() {
	util.AddMemoryReporter(`Facts`, GetMemoryUsage)
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `go test ./internal/facts/ -run TestGetMemoryUsage_NilRegistry_NoPanic -v`
Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/bounties/memory.go internal/bounties/memory_test.go internal/facts/memory.go internal/facts/memory_test.go
git commit -m "feat(perf): nil-safe memory reporters for bounties + facts registries"
```

---

## Task 3: `IdleMobs::*` per-tick sub-timers

Break the schedule / patrol / conversation work out of the lumped `IdleMobs()`
timer. Accumulate per-tick totals across all mobs, record once after the loop.

**Files:**
- Modify: `internal/hooks/NewRound_IdleMobs.go`

- [ ] **Step 1: Add per-tick duration accumulators before the mob loop**

In `IdleMobs`, immediately after `tStart := time.Now()` (currently line 37) and
before `for _, mobId := range allMobInstances {`, add:

```go
	// Chunk 6.4: per-tick sub-timers, broken out of the lumped IdleMobs()
	// total so 6.6 can attribute growth. Accumulated across all mobs, recorded
	// once after the loop (same denominator as the IdleMobs() parent).
	var schedDur, patrolDur, convDur time.Duration
```

- [ ] **Step 2: Wrap the schedule executor block**

Replace the current schedule block:

```go
		// Chunk 3.2: schedule executor. ...
		if mob.ScheduleId != "" {
			plan := scheduleTickPlan(mob, gametime.GetDate().Hour24)
			applySchedulePlan(mob, plan)
		}
```

with:

```go
		// Chunk 3.2: schedule executor. ...
		if mob.ScheduleId != "" {
			tSched := time.Now()
			plan := scheduleTickPlan(mob, gametime.GetDate().Hour24)
			applySchedulePlan(mob, plan)
			schedDur += time.Since(tSched)
		}
```

- [ ] **Step 3: Wrap the patrol executor block**

In the patrol block (the `{ ... }` scope currently lines 83-96), wrap only the
work that runs when a patrol is active. Replace:

```go
			if activePatrolId != "" {
				plan := patrolTickPlan(mob, activePatrolId)
				applyPatrolPlan(mob, plan, activePatrolId)
			}
```

with:

```go
			if activePatrolId != "" {
				tPatrol := time.Now()
				plan := patrolTickPlan(mob, activePatrolId)
				applyPatrolPlan(mob, plan, activePatrolId)
				patrolDur += time.Since(tPatrol)
			}
```

- [ ] **Step 4: Wrap the conversation tick + trigger block**

Wrap both conversation phases as a single seam. Put `tConv := time.Now()`
immediately before the Phase-1 `if partnerId, ok := ...` block (currently line
101) and accumulate immediately after the Phase-2 trigger block closes
(currently after line 116). Concretely, surround the two conversation `if`
blocks:

```go
		tConv := time.Now()
		// Phase 1: advance an in-progress conversation.
		if partnerId, ok := mob.Character.GetMiscData(conversations.MiscDataPartnerId).(int); ok && partnerId > 0 {
			conversations.TickConversation(conversationadapter.AdaptMob(mob), partnerId)
		}
		// Phase 2: roll for a new conversation.
		if conversationsTriggerEligible(mob) {
			cfg := configs.GetBalanceConfig()
			if util.Rand(10000) < int(float64(cfg.ConversationBaseChancePct)*100) {
				if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
					conversations.TryStart(conversationadapter.AdaptMob(mob), room.GetMobs())
				}
			}
		}
		convDur += time.Since(tConv)
```

- [ ] **Step 5: Record the sub-timers after the loop**

Immediately before the existing `util.TrackTime(`IdleMobs()`, time.Since(tStart).Seconds())`
(currently line 182), add:

```go
	util.TrackTime(`IdleMobs::schedule`, schedDur.Seconds())
	util.TrackTime(`IdleMobs::patrol`, patrolDur.Seconds())
	util.TrackTime(`IdleMobs::conversation`, convDur.Seconds())
```

- [ ] **Step 6: Build**

Run: `go build ./internal/hooks/`
Expected: clean.

- [ ] **Step 7: Commit**

```bash
git add internal/hooks/NewRound_IdleMobs.go
git commit -m "feat(perf): break IdleMobs() into schedule/patrol/conversation sub-timers"
```

---

## Task 4: `MobIdle::goalplanner` sub-timer

**Files:**
- Modify: `internal/hooks/MobIdle_HandleIdleMobs.go`

- [ ] **Step 1: Wrap the goal-planner call**

Find the goal-planner dispatch (currently line 236):

```go
		behaviortree.RunGoalPlanner(mob, util.GetRoundCount())
```

Replace with:

```go
		tGoal := time.Now()
		behaviortree.RunGoalPlanner(mob, util.GetRoundCount())
		util.TrackTime(`MobIdle::goalplanner`, time.Since(tGoal).Seconds())
```

- [ ] **Step 2: Ensure `time` and `util` are imported**

Check the import block of `internal/hooks/MobIdle_HandleIdleMobs.go`. `util` is
already imported (the file uses `util.GetRoundCount()`). If `time` is not in the
import list, add `"time"` to it.

Run: `go build ./internal/hooks/`
Expected: clean. If it fails with `"time" imported and not used` or
`undefined: time`, fix the import block accordingly.

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/MobIdle_HandleIdleMobs.go
git commit -m "feat(perf): add MobIdle::goalplanner sub-timer"
```

---

## Task 5: `Enforcement` sub-timer

**Files:**
- Modify: `internal/hooks/NewRound_MobRoundTick.go`

- [ ] **Step 1: Wrap the guard-enforcement call**

Find the enforcement dispatch (currently line 117):

```go
			justice.RunGuardEnforcement(mob, room, roundCount)
```

Replace with:

```go
			tEnf := time.Now()
			justice.RunGuardEnforcement(mob, room, roundCount)
			util.TrackTime(`Enforcement`, time.Since(tEnf).Seconds())
```

- [ ] **Step 2: Ensure `time` and `util` are imported**

`NewRound_MobRoundTick.go` already uses `time.Now()` and `time.Since` elsewhere
(per the spec's grep), so `time` is imported. Confirm `util` is imported; if not,
add `"github.com/GoMudEngine/GoMud/internal/util"`.

Run: `go build ./internal/hooks/`
Expected: clean.

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/NewRound_MobRoundTick.go
git commit -m "feat(perf): add Enforcement sub-timer around guard enforcement"
```

---

## Task 6: Update package `context.md` files

Per the roadmap's per-chunk `context.md` rule, each package that gained a
`memory.go` gets a one-line note. Add to the relevant section (Key Components or
a short "Memory Reporting" note) of each `context.md`:

> **Memory Reporting** (`memory.go`): registers a `util.AddMemoryReporter` under
> the section name `<Name>`, surfacing the in-memory store size + entry count in
> the `server stats` admin command (mob-aliveness chunk 6.4).

**Files:**
- Modify: `internal/opinions/context.md`
- Modify: `internal/factions/context.md`
- Modify: `internal/crimes/context.md`
- Modify: `internal/knowledge/context.md`
- Modify: `internal/bounties/context.md`
- Modify: `internal/facts/context.md`
- Modify: `internal/relationships/context.md`
- Modify: `internal/goals/context.md`

- [ ] **Step 1: Add the Memory Reporting note to each of the 8 context.md files**

Use the section name actually registered (`Opinions`, `Factions`, `Crimes`,
`Knowledge`, `Bounties`, `Facts`, `Relationships`, `Goals` respectively).

- [ ] **Step 2: Commit**

```bash
git add internal/*/context.md
git commit -m "docs(context): note 6.4 memory reporters in substrate package context.md"
```

---

## Task 7: Build, test, boot smoke

Verify the instrumentation compiles, all tests pass, and the server boots and
renders the new rows/seams.

- [ ] **Step 1: Full build + test**

Run: `go build ./... && go test ./...`
Expected: build clean; all packages pass (including the two new nil-safety tests).

- [ ] **Step 2: Wipe instance saves (smoke SOP)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

- [ ] **Step 3: Boot the server and confirm clean data-file load**

Run the server (background) and watch startup log for `mobs.LoadDataFiles()`,
`quests.LoadDataFiles()`, etc. with NO panics. Confirm it reaches the listening
state.

- [ ] **Step 4: Confirm `server stats` renders the new rows and seams**

Connect as an admin character and run `server stats`. Confirm the memory report
now lists sections: `Opinions`, `Factions`, `Crimes`, `Knowledge`, `Bounties`,
`Facts`, `Relationships`, `Goals` (counts may be small/zero at boot — that's
fine). Confirm the timer table will accrue `IdleMobs::schedule`,
`IdleMobs::patrol`, `IdleMobs::conversation`, `MobIdle::goalplanner`, and
`Enforcement` once ticks have run (they appear lazily on first sample).

- [ ] **Step 5: Sanity-check the parent/child timer relationship**

After the server has ticked for a few minutes, run `server stats` again. Confirm
`IdleMobs::schedule` + `IdleMobs::patrol` + `IdleMobs::conversation` averages sum
to ≤ the `IdleMobs()` average (they are sub-portions of the same per-tick loop).
If a child exceeds the parent, a wrap is mis-placed — fix before proceeding.

There is no commit for this task (verification only).

---

## Task 8: Capture the baseline + write the deliverable

This task produces the actual numbers and the living doc. It runs the two capture
procedures from the spec.

**Files:**
- Create: `docs/perf/aliveness-perf-baseline.md`

- [ ] **Step 1: Capture the idle floor (500 rounds)**

With instance saves wiped and the server freshly booted (no players), let it idle
until the `IdleMobs()` accumulator `Ct` column in `server stats` reaches **500**
(the count equals ticks/rounds elapsed). Then snapshot the full `server stats`
output — both the Timer Stats table and the memory report.

- [ ] **Step 2: Capture under load (feel-tester)**

Restart fresh (wipe instance saves, reboot). Start one AI tester:
`/test-mud local feel-tester` (see CLAUDE.md "AI Testing"). Let the server reach
`IdleMobs()` `Ct` = 500 again, then snapshot `server stats`.

- [ ] **Step 3: Write `docs/perf/aliveness-perf-baseline.md`**

Use this structure, filling the tables from the two snapshots:

```markdown
# Aliveness Performance Baseline

Fine-grained baseline for the mob-aliveness substrate + per-tick work, captured
after chunk 6.4 instrumentation landed. Re-run the procedure below verbatim for
chunk 6.6 (post-content-pass re-review) and append a new dated section.

See also: the coarse prod pull/restart + idle-CPU log in the
`reference_prod_perf_baseline` memory.

## Capture procedure (re-run exactly for 6.6)

**Idle floor:**
1. `rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*`
2. Boot locally, no players.
3. Wait until `server stats` shows `IdleMobs()` Ct = 500.
4. Snapshot `server stats` (timer table + memory report).

**Under load:** same, but with one `test-mud local feel-tester` running; snapshot
at IdleMobs() Ct = 500.

**Timer denominators:** `IdleMobs::{schedule,patrol,conversation}` are per-tick
totals (sum ≤ `IdleMobs()`); `MobIdle::goalplanner` and `Enforcement` are
per-invocation (per mob).

## YYYY-MM-DD — 6.4 baseline (master @ <sha>, <N mobs / N items / N quests loaded>)

### Substrate memory footprint

| Section | Store | Count | Size |
|---------|-------|-------|------|
| Opinions | opinionCache | | |
| Opinions | nameByMobId | | |
| Factions | definitions | | |
| Factions | repCache | | |
| Crimes | crimeCache | | |
| Knowledge | knowledgeCache | | |
| Bounties | registry | | |
| Facts | registry | | |
| Facts | awarenessCache | | |
| Relationships | graph | | |
| Goals | cache | | |
| Goals | nameByMobId | | |

### Tick budget (avg ms / low / high / count / per-sec)

| Seam | Idle avg | Under-load avg | Notes |
|------|----------|----------------|-------|
| IdleMobs() (roll-up) | | | |
| IdleMobs::schedule | | | |
| IdleMobs::patrol | | | |
| IdleMobs::conversation | | | |
| MobIdle::goalplanner | | | per-invocation |
| Enforcement | | | per-invocation |
| events.ProcessEvents() (roll-up) | | | |

### Reading

<2-4 sentences: what's hot, what's negligible, any surprise. State whether the
substrate footprint and per-tick seams leave comfortable headroom before the 6.5
content pass scales mob/zone counts up.>
```

- [ ] **Step 4: Append a pointer to the prod-perf memory**

Add a one-line cross-reference to the `reference_prod_perf_baseline` memory file
(top, under the intro) pointing at `docs/perf/aliveness-perf-baseline.md` as the
fine-grained companion to the coarse pull/restart + idle-CPU log.

- [ ] **Step 5: Commit**

```bash
git add docs/perf/aliveness-perf-baseline.md
git commit -m "docs(perf): capture mob-aliveness 6.4 fine-grained baseline"
```

---

## Task 9: Roadmap status update

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`

- [ ] **Step 1: Mark 6.4 Done in both places**

In the Progress tracker table, change the 6.4 row Status to `Done (2026-06-05)`.
In the 6.4 mini-brief, change `**Status:** Not started` to
`**Status:** Done (2026-06-05)` and add a `- **Shipped:**` bullet summarizing:
8 substrate memory reporters, 5 tick sub-timers, baseline doc at
`docs/perf/aliveness-perf-baseline.md`. Re-tally the roll-up line
(39 / 42 done • 0 in progress • 3 not started).

- [ ] **Step 2: Commit**

```bash
git add MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(roadmap): mark 6.4 performance review (initial) Done"
```

---

## Self-review notes

**Spec coverage:**
- Spec §1 (memory instrumentation, 8 packages) → Tasks 1-2 (6 map-based + 2 nil-safe).
- Spec §2 (tick sub-timers: schedule/patrol/conversation/goalplanner/enforcement) → Tasks 3-5.
- Spec §3 (capture: idle 500 + under-load) → Task 8 Steps 1-2.
- Spec §4 (deliverable doc + memory pointer) → Task 8 Steps 3-4.
- Spec "Testing/validation" (build, boot, render, sub ≤ parent) → Task 7.
- Spec "Files touched" includes context.md updates → Task 6.
- Roadmap maintenance (status in both places, roll-up) → Task 9.

**Out-of-scope guards honored:** no on-disk byte accounting, no persistence-write
timing, no synthetic harness, no optimization.

**Type/name consistency:** store var + mutex names verified against source (see
Reference table). Reporter section names (`Opinions`/`Factions`/.../`Goals`) used
consistently in Tasks 1-2, 6, 7, and 8's doc template.
