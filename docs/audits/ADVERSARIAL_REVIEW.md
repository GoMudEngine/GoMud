# DOGMud Adversarial Review: Technical Teardown & Risk Assessment
**Date**: August 2026
**Auditor**: Jules
**Target Audience**: Principal Developer

---

## Executive Summary

DOGMud is a highly ambitious, custom-built MUD that successfully implements
many sophisticated gameplay mechanics—such as use-based progression, a three-
channel combat system, and a dynamic living economy. However, from a rigorous
software engineering and red-team architectural perspective, the codebase is
built on several fragile foundations that severely limit its scalability,
reliability, and long-term maintainability.

This document exposes the critical architectural flaws, concurrency bottlenecks,
performance hazards, and mathematical inconsistencies currently present in the
repository, ordered by severity.

---

## 1. Concurrency & Global Locking Bottlenecks
**Severity: Critical**

### The Core Problem: The Single Global Mutex
DOGMud relies on a single, coarse-grained read-write mutex to synchronize
access across the entire application: `mudLock` in `internal/util/util.go`,
surfaced as `util.LockMud()`, `util.UnlockMud()`, `util.RLockMud()`, and
`util.RUnlockMud()`.

Instead of localized thread-safe data structures, channels, or actor-based
concurrency, the game engine uses this global lock as a blunt hammer.

```go
// internal/util/util.go
var mudLock = sync.RWMutex{}
```

### The Architectural Lockup: `RunWithMUDLocked`
The most egregious symptom of this design is the HTTP middleware used for the
web administration dashboard and APIs:

```go
// internal/web/web.go
func RunWithMUDLocked(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		util.LockMud()
		defer util.UnlockMud()
		next.ServeHTTP(w, r)
	})
}
```

Practically every route under `/admin/` (including `/admin/items/`, `/admin/mobs/`,
`/admin/api/combat-stats/`, and `/admin/api/economy/`) is wrapped in this
middleware.

**The consequence is catastrophic**: When an administrator requests a slow web page,
the web server acquires a **global write lock** on the MUD. The main game loop
(`MainWorker` in `world.go`) entirely freezes and ceases ticking for the duration
of the HTTP request's parsing, template execution, and synchronous socket write.
A slow client reading admin data will freeze the game server for all active
players.

### Tab-Completion Lock Contention
Tab-completion suggestions (`GetAutoComplete` in `world.go`) run asynchronously
on the per-connection goroutines rather than the main tick thread. To prevent
concurrency panics, `GetAutoComplete` takes a read lock:

```go
// world.go
func (w *World) GetAutoComplete(userId int, inputText string) []string {
	util.RLockMud()
	defer util.RUnlockMud()
	...
}
```

While a read lock is less disruptive than a write lock, `GetAutoComplete` is an
extremely heavy function. It iterates over exits, room containers, active mobs,
all backpack and equipped items, and the user's spellbook.

If multiple players spam the `Tab` key, or if automated clients script tab-
completion queries, the accumulation of active read locks will block the main
`MainWorker` loop from acquiring the write lock (`util.LockMud()`) at its turn or
event boundaries, leading to severe input-processing lag and frame-time spikes.

---

## 2. Synchronous State Persistence & I/O Overhead
**Severity: Critical**

### The Autosave Freeze
Every few minutes, a `NewTurn` event triggers the autosave sequence:

```go
// internal/hooks/NewTurn_AutoSave.go
func AutoSave(e events.Event) events.ListenerReturn {
	...
	users.SaveAllUsers(true)
	rooms.SaveAllRooms()
	plugins.Save()
	...
}
```

Because `AutoSave` runs as part of the event-processing loop inside `MainWorker`,
it executes **synchronously** while holding the global `util.LockMud()` write lock.
During this period, the entire game state is completely frozen.

### Reflection-Based Instance Diffing
The save mechanism for rooms is remarkably inefficient. To minimize the disk
footprint of saved room instances, the engine diffs the live state against the
static template:

```go
// internal/rooms/save_and_load.go
func SaveRoomInstance(r Room) error {
	...
	rTpl := LoadRoomTemplate(r.RoomId)
	...
	rVal := reflect.ValueOf(r)
	tplVal := reflect.ValueOf(*rTpl)
	t := reflect.TypeOf(r)
	...
	for i := 0; i < t.NumField(); i++ {
		...
		if reflect.DeepEqual(rVal2.Interface(), tplVal2.Interface()) {
			continue
		}
		...
	}
	...
}
```

For **every single non-ephemeral room** loaded in memory (which can scale to
thousands of rooms), `SaveAllRooms`:
1. Reloads or fetches the original room template from disk/cache.
2. Uses runtime reflection (`reflect.ValueOf`, `reflect.TypeOf`) to inspect
   every field in the `Room` struct.
3. Invokes `reflect.DeepEqual` on each field to check for mutations.
4. Marshals modified values to YAML.
5. Synchronously writes or deletes files on disk using `util.Save`.

Doing hundreds or thousands of reflection scans and synchronous disk writes on
the main gameplay loop thread guarantees major periodic performance lag spikes.
As the player count and modified room instance count grow, this approach will
rapidly degrade the server into an unplayable state.

---

## 3. Tight Architectural Coupling & "Star-Topology" Dependency Gaping
**Severity: Medium-High**

### Sidestepping the Compiler via Global Function Pointers
Because Go strictly forbids circular package imports at compile time, and
DOGMud's subsystems are deeply interdependent, the engine uses package-level
mutable function pointers as a dynamic dependency injection mechanism.

These callbacks are registered globally in `main.go` at startup:

```go
// main.go
rooms.SetBTreeStateEvictor(behaviortree.EvictRoomBTreeState)
rooms.SetCompanionTransport(hooks.CompanionTransportCallback)
behaviortree.SetCompanionSweep(hooks.CompanionSweepCallback)
characters.SetUserUntargetableCheck(...)
users.SetCanSeeInRoomCheck(...)
goals.SetWeightsLookup(...)
...
```

### The Technical Debt & Fragility
While this pattern allows the project to compile, it introduces several major
architectural smells:

1. **Implicit Mutable Global State**: These are mutable global variables. There
   is nothing preventing another goroutine or plugin from overwriting these
   function pointers at runtime, leading to hard-to-debug race conditions.
2. **Initialization & Bootstrapping Hazards**: If a subsystem, tool, or unit
   test initializes and attempts to use a package (like `rooms`) before `main.go`
   executes and wires up the pointers, the application will crash with a silent,
   untraceable `nil` pointer dereference.
3. **Brittle Test Harnesses**: Unit tests are forced to perform complex setup
   and cleanup routines, manually backing up global function pointers and
   restoring them in `defer` statements to avoid cross-test pollution:

```go
// internal/rooms/instances_test.go
func TestBTreeStateEviction(t *testing.T) {
	origEvictor := bTreeStateEvictor
	defer SetBTreeStateEvictor(origEvictor)
	SetBTreeStateEvictor(func(roomId int) { ... })
	...
}
```

This "star-topology" where `main.go` acts as a central wiring board hiding
behind dynamic pointers is a major anti-pattern. It masks tight coupling rather
than resolving it through clean interface boundaries or event-driven pipelines.

---

## 4. Game Design Mathematics & Mechanical Contradictions
**Severity: Medium**

### The "No Cap" Progression Illusion
DOGMud’s marketing and documentation place heavy emphasis on use-based
progression, boasting that there are no levels, no classes, and no hard
ceilings. However, the unified damage pipeline contains a hard cap on skill
effectiveness:

```go
// internal/combat/damage_pipeline.go
func SkillMultiplier(rank int) float64 {
	...
	softCap := float64(bal.SkillSoftCap) // Default: 50.0

	if rank <= 0 {
		return base
	}
	r := float64(rank)
	if r > softCap {
		r = softCap
	}
	return base + (max-base)*math.Sqrt(r/softCap)
}
```

If a player progresses a skill beyond the soft cap of 50, the rank `r` is
strictly hard-capped at `softCap`. The multiplier remains permanently locked at
`SkillMultiplierMax` (typically 3.0).

While the use counter and the character sheet can progress infinitely, **the
actual combat damage scaling is hard-capped at rank 50**. This creates a major
disconnect between player expectations of infinite growth and the underlying
mathematical implementation.

### Asymmetric Standard Deviation Roll Coupling
The core rolling system utilizes normal distribution curves. When resolving
opposed checks, the engine couples both participants to the attacker's power:

```go
// internal/dice/dice.go
func OpposedRollStat(atk, def float64) (bool, float64, RollResult, RollResult) {
	return OpposedRoll(atk, def, StdDevFor(atk))
}
```

In `OpposedRoll`, both the attacker's and defender's rolls are evaluated using
the standard deviation calculated purely from the **attacker's** stat:

1. **Weak Attacker vs. Strong Defender**: If an attacker with a stat of 10
   swings at a defender with a stat of 200, the standard deviation is
   `10 * 0.15 = 1.5`. The defender's roll has incredibly low variance, making
   their defensive capability highly deterministic and tightly bound to 200.
2. **Strong Attacker vs. Weak Defender**: If a boss with a stat of 200 attacks
   a weak character with a stat of 10, the standard deviation is
   `200 * 0.15 = 30.0`. The weak defender's roll now has an enormous variance of
   `30.0`. Their defense roll value can easily swing from `-50` to `+70` purely
   because a powerful boss is attacking them.

This asymmetric math means a character's internal consistency (their variance)
is controlled entirely by who is targeting them. This is a highly unintuitive
design choice that can lead to bizarre fumbles and critical defense swings.

---

## 5. Testing & QA Strategy: The Playtest LLM Mirage
**Severity: Low-Medium**

DOGMud relies heavily on autonomous LLM agents (running under the `/playtest`
harness) to find bugs and provide experience feedback. While this is an
innovative and flashy tool, it is an ineffective substitute for rigorous,
deterministic QA.

1. **Nondeterminism & Flakiness**: LLM-based testers are probabilistic. They
   cannot provide consistent regression checks because they may navigate a path
   successfully in one run and fail in the next.
2. **Inefficient Path Coverage**: Checking a branching dialogue tree or a quest
   chain with an LLM agent takes minutes and incurs substantial API token costs.
   A simple depth-first search (DFS) parser could validate all dialogue options,
   quest exclusions, and room transitions in less than 5 milliseconds with 100%
   coverage.
3. **Loop Traps & Pacing Gaps**: LLMs are notoriously easy to trap in local
   optima (e.g., getting stuck trying to open a locked container, repeating
   NPC greetings, or walking back and forth between two rooms). This results in
   wasted execution cycles and leaves critical mid-to-late game systems untested.

---

## Recommendations & Mitigation Path

To transition DOGMud into a production-ready, highly scalable architecture, the
following refactorings are recommended:

1. **Decouple the Web Tier**: Remove `RunWithMUDLocked`. Convert HTTP handlers
   to query a read-only, thread-safe memory snapshot of the world, or fetch
   metrics that are updated on a decouple schedule (e.g., once a second) to
   avoid halting the game loop.
2. **Asynchronous File Writes**: Offload user and room serialization to a
   background worker queue. The main thread should flag a room or user as
   "dirty" and push a cloned state copy to a channel. The disk I/O should
   happen entirely in a background goroutine, eliminating autosave lag spikes.
3. **Event-Driven Architecture**: Replace dynamic global function pointers with
   an asynchronous or decoupled Event Bus. Subsystems should publish events
   (e.g., `RoomEvict`) that interested packages listen to, removing compiling
   circularity without resorting to mutable global variables.
4. **Fix the Progression Math**: Allow skill ranks beyond the soft cap to
   provide diminishing but real damage multiplier returns (e.g., using a logarithmic
   scaling curve above 50) rather than a hard ceiling.
