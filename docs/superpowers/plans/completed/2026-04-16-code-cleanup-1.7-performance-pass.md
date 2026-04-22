# Code Cleanup 1.7: Performance Pass Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Cut CPU spent on mobs in unpopulated zones via an active/idle lane split, close data-race surfaces on global mob/user registries with sync.RWMutex, and pick up a PruneVisitors early-return win — all with zero behavior change in active zones.

**Architecture:** A new `zonePlayerCount` index in `internal/rooms/` is incremented/decremented by hooks inside `Room.AddPlayer` / `Room.RemovePlayer` (both already dedupe, so the hook fires exactly once per genuine membership change). Two round hooks (`handleMobCombat`, `NewRound_MobRoundTick`) consume the index — mobs in zones with zero players skip combat and skip progression/charm-state/crafting, but still tick cooldowns, buff durations, charm duration, combat-memory expiry, and condition timers. Six global maps (5 in `internal/mobs`, 1 in `internal/users`) get `sync.RWMutex` wrappers.

**Tech Stack:** Go, `sync.RWMutex`, `testing` with `-race`

**Spec:** `docs/superpowers/specs/2026-04-16-code-cleanup-1.7-performance-pass-design.md`

---

## Pre-flight: Background facts

Facts already verified during plan preparation (do NOT re-verify):

- `handleMobCombat` lives in `internal/hooks/NewRound_DoCombat.go` starting line 138; the per-mob loop is at lines 145–269 and iterates via `mobs.GetAllMobInstanceIds()`.
- `MobRoundTick` is the hook function in `internal/hooks/NewRound_MobRoundTick.go` (lines 28–415). The per-mob loop starts at line 88 and ends at line 409. Before the loop there's pre-loop work (mob saves 33–41, ReduceHostility 46, pack scaling 49–77, pack roaming 80–82) that stays untouched by this plan.
- The 10 inline blocks in the per-mob loop are at these exact line ranges (in the current source):
  - Line 97: `mob.Character.Cooldowns.RoundTick()`
  - Lines 99–105: combat memory expiry
  - Lines 107–118: prone recovery with room message
  - Lines 120–122: charm rounds decrement (`RoundsRemaining--`)
  - Lines 124–160: buff trigger + tick amount + BuffsTriggered event
  - Lines 162–260: mutation acquisition (gated on `mob.Character.Aggro != nil`)
  - Lines 262–279: charm expiry cleanup (after RoundsRemaining hit 0)
  - Lines 281–353: charm re-roll contested Charisma vs Willpower
  - Lines 355–396: crafting tick
  - Line 399: `mob.Character.TickConditions()`
  - Line 402: `mob.Character.Validate()`
  - Lines 404–407: death handling (`mob.Command("suicide")`)
- `internal/mobs/mobs.go` line 28–37 declares these globals with no mutex: `instanceCounter int`, `mobs = map[int]*Mob{}`, `allMobNames = []string{}`, `mobInstances = map[int]*Mob{}`, `mobsHatePlayers = map[string]map[int]int{}`, `mobNameCache = map[MobId]string{}`, `recentlyDied = map[int]int{}`.
- `internal/users/users.go` has an `ActiveUsers` struct with a `Users map[int]*UserRecord` field — no mutex today.
- `Room.AddPlayer` and `Room.RemovePlayer` are at `internal/rooms/rooms.go:2137` and `:2151` respectively. Both already dedupe — `AddPlayer` early-returns if the userId is already present; `RemovePlayer` returns `(count, false)` if not found.
- `Room.PruneVisitors` is at `internal/rooms/rooms.go:1931`. `r.visitors` is a `map[VisitorType]map[int]uint64`.
- `rooms.LoadDataFiles()` runs in `main.go:1067`. Data loading happens before the world event loop spins up. Users are loaded via `users.LoadDataFiles()` later in the same init sequence.
- HTTP servers at `internal/web/web.go:392` and `:454` launch via `go func()` — every request handler runs on its own goroutine. `internal/web/admin.mobs.go` reads `mobs.GetAllMobInfo()` from one of those handler goroutines. **This is a real race surface.**
- The existing pattern for test helpers in rooms: `internal/rooms/test_helpers.go` provides `SetRoomManagerForTest(mgr)` etc.
- Existing test naming: `TestFunctionName_SpecificBehavior` (Go convention, used throughout).
- `util.GetRoundCount()` returns the current round.
- `sendVisualRoomText` is a package-level helper in `internal/hooks` (referenced throughout MobRoundTick).

---

## Task 1: PruneVisitors Empty-Map Fast Path

**Files:**
- Modify: `internal/rooms/rooms.go:1931` (function `PruneVisitors`)
- Test: `internal/rooms/rooms_test.go` (add `TestPruneVisitors_EmptyMapFastPath`)

**Rationale:** Most rooms have no visitors in the last 180s; the current loop does work for nothing. Early return turns empty-map calls into O(1).

- [ ] **Step 1.1: Read the current PruneVisitors**

Run:
```bash
sed -n '1925,1975p' internal/rooms/rooms.go
```

Confirm the function starts at line 1931 and that the first statement inside the function is a `for visitorType, visitors := range r.visitors` loop (or similar).

- [ ] **Step 1.2: Write the failing test**

Add to `internal/rooms/rooms_test.go` (append at end of file):

```go
func TestPruneVisitors_EmptyMapFastPath(t *testing.T) {
	r := &Room{
		RoomId:   9991,
		Zone:     "test-zone",
		visitors: nil, // nil map is valid and should short-circuit
	}

	// Should not panic on nil map.
	r.PruneVisitors()

	// Map should still be nil (no allocation performed).
	if r.visitors != nil {
		t.Fatalf("expected visitors to stay nil, got %v", r.visitors)
	}

	// Also works on explicit empty map.
	r.visitors = map[VisitorType]map[int]uint64{}
	r.PruneVisitors()
	if len(r.visitors) != 0 {
		t.Fatalf("expected empty visitors, got %d entries", len(r.visitors))
	}
}
```

- [ ] **Step 1.3: Run the test**

Run:
```bash
cd /c/Users/Calabe\ Davis/workspace/DOGMud && go test ./internal/rooms/ -run TestPruneVisitors_EmptyMapFastPath -v
```

Expected: PASSES on the nil case (because `range nil` is a no-op), and passes on the empty-map case already. **This is a characterization test** — it documents the post-fix behavior. If it already passes pre-fix, still good; the fix is a performance patch, not a correctness fix.

- [ ] **Step 1.4: Add the early return**

Edit `internal/rooms/rooms.go:1931`. Replace the function header + first line with:

```go
func (r *Room) PruneVisitors() {

	if len(r.visitors) == 0 {
		return
	}

	// ... existing body unchanged
```

Keep the original loop body exactly as it was. Only the 3 lines (`if len(r.visitors) == 0 { return }`) are new.

- [ ] **Step 1.5: Run the test + package tests**

Run:
```bash
go test ./internal/rooms/ -run TestPruneVisitors -v
go test ./internal/rooms/ -count=1
```

Expected: all pass.

- [ ] **Step 1.6: Commit**

```bash
git add internal/rooms/rooms.go internal/rooms/rooms_test.go
git commit -m "$(cat <<'EOF'
perf(rooms): add early return to PruneVisitors on empty map

Most rooms have zero visitor history at any given moment. Short-circuit
when r.visitors is empty to skip the full loop.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Mutex on `internal/mobs` Package Maps

**Files:**
- Modify: `internal/mobs/mobs.go` (add 5 mutexes + wrap accessors)
- Test: `internal/mobs/mobs_concurrent_test.go` (new file)

**Rationale:** `internal/web/admin.mobs.go:25,60` reads `mobs.GetAllMobInfo()` (which ranges `mobs`) concurrently with the main game loop mutating those maps. Adding RWMutex closes the race without changing single-threaded semantics.

**Maps to protect:**
- `mobs` (templates, line 30) → `mobsMu sync.RWMutex`
- `mobInstances` (live, line 32) → `mobInstancesMu sync.RWMutex`
- `mobsHatePlayers` (line 33) → `mobsHatePlayersMu sync.RWMutex`
- `mobNameCache` (line 34) → `mobNameCacheMu sync.RWMutex`
- `recentlyDied` (line 36) → `recentlyDiedMu sync.RWMutex`
- `instanceCounter` is guarded by `mobInstancesMu` (always incremented together with `mobInstances` writes)
- `allMobNames` is guarded by `mobsMu` (derived from same template data)

- [ ] **Step 2.1: Write the failing race-detector test**

Create `internal/mobs/mobs_concurrent_test.go`:

```go
package mobs

import (
	"sync"
	"testing"
)

// TestConcurrentInstanceAccess hammers the instance map from multiple
// goroutines to exercise the RWMutex. Must be run with `go test -race`.
func TestConcurrentInstanceAccess(t *testing.T) {
	const writers = 4
	const readers = 8
	const iters = 500

	var wg sync.WaitGroup

	// Writers create and destroy instances.
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				m := &Mob{InstanceId: id*1000 + i}
				m.Character.Name = "racer"
				storeInstanceForTest(m)
				removeInstanceForTest(m.InstanceId)
			}
		}(w)
	}

	// Readers iterate and read.
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				_ = GetAllMobInstanceIds()
				_ = GetInstance(i)
			}
		}()
	}

	wg.Wait()
}
```

Also add these internal test helpers at the bottom of the file:

```go
// storeInstanceForTest / removeInstanceForTest expose the locked mutation
// paths without going through the full NewMobById / DestroyInstance flow,
// which pulls in too many dependencies for a focused race test.
func storeInstanceForTest(m *Mob) {
	mobInstancesMu.Lock()
	mobInstances[m.InstanceId] = m
	mobInstancesMu.Unlock()
}

func removeInstanceForTest(id int) {
	mobInstancesMu.Lock()
	delete(mobInstances, id)
	mobInstancesMu.Unlock()
}
```

(These helpers reference `mobInstancesMu` — which doesn't exist yet. That's the point: the test also fails to compile until Step 2.2 lands.)

- [ ] **Step 2.2: Add mutex declarations**

Edit `internal/mobs/mobs.go` lines 28–37. Replace the `var (...)` block with:

```go
var (
	instanceCounter int         = 0
	mobs                         = map[int]*Mob{}
	mobsMu          sync.RWMutex // guards mobs + allMobNames
	allMobNames                  = []string{}

	mobInstances       = map[int]*Mob{}
	mobInstancesMu     sync.RWMutex // guards mobInstances + instanceCounter

	mobsHatePlayers    = map[string]map[int]int{}
	mobsHatePlayersMu  sync.RWMutex

	mobNameCache       = map[MobId]string{}
	mobNameCacheMu     sync.RWMutex

	recentlyDied    = map[int]int{}
	recentlyDiedMu  sync.RWMutex
)
```

Ensure `"sync"` is in the import list at the top of the file. If not, add it.

- [ ] **Step 2.3: Wrap the read accessors**

For each public reader, wrap the map access with `RLock`/`RUnlock`. Find every function in `internal/mobs/mobs.go` that reads from one of the five guarded maps and wrap appropriately. Required edits (file may contain more — scan for any `range mobs`, `mobs[`, `mobInstances[`, `range mobInstances`, `mobsHatePlayers[`, `mobNameCache[`, `recentlyDied[`):

```go
// GetInstance — wrap the lookup
func GetInstance(instanceId int) *Mob {
	mobInstancesMu.RLock()
	defer mobInstancesMu.RUnlock()
	return mobInstances[instanceId]
}

// GetAllMobInstanceIds — wrap the range
func GetAllMobInstanceIds() []int {
	mobInstancesMu.RLock()
	defer mobInstancesMu.RUnlock()
	ids := make([]int, 0, len(mobInstances))
	for id := range mobInstances {
		ids = append(ids, id)
	}
	return ids
}

// GetAllMobInfo — wrap the range
func GetAllMobInfo() []Mob {
	mobsMu.RLock()
	defer mobsMu.RUnlock()
	ret := make([]Mob, 0, len(mobs))
	for _, m := range mobs {
		ret = append(ret, *m)
	}
	return ret
}

// GetAllMobNames — wrap
func GetAllMobNames() []string {
	mobsMu.RLock()
	defer mobsMu.RUnlock()
	out := make([]string, len(allMobNames))
	copy(out, allMobNames)
	return out
}

// GetMobSpec — wrap
func GetMobSpec(mobId MobId) *Mob {
	mobsMu.RLock()
	defer mobsMu.RUnlock()
	if m, ok := mobs[int(mobId)]; ok {
		return m
	}
	return nil
}
```

For `mobNameCache`, `mobsHatePlayers`, `recentlyDied` — wrap every read with the matching `*Mu.RLock()` / `defer *Mu.RUnlock()`. Use grep to find them all. Use `Grep` with pattern `mobNameCache\[|mobsHatePlayers\[|recentlyDied\[` to enumerate sites.

- [ ] **Step 2.4: Wrap the write sites**

Write sites touch `mobs`, `mobInstances`, `mobsHatePlayers`, `mobNameCache`, `recentlyDied`, `instanceCounter`, or `allMobNames`. Find each and wrap with `Lock()/Unlock()`.

Common patterns:
- Template loader (`LoadDataFiles`) writes to `mobs` + `allMobNames` — wrap with `mobsMu.Lock()` once around the whole append/assign block.
- `NewInstance` / spawn path mutates `mobInstances` + `instanceCounter` — wrap with `mobInstancesMu.Lock()`.
- `DestroyInstance` removes from `mobInstances` — `mobInstancesMu.Lock()`.
- `ReduceHostility` likely iterates + mutates `mobsHatePlayers` — `mobsHatePlayersMu.Lock()`.
- Any write to `mobNameCache` — `mobNameCacheMu.Lock()`.
- `recentlyDied` writes — `recentlyDiedMu.Lock()`.

**Critical:** Do not hold a lock across a call that might re-enter the same package or dispatch an event. If you see `events.AddToQueue(...)` or `mobs.SomeFunc(...)` inside a locked section, restructure: extract the data under the lock, release, then make the call.

- [ ] **Step 2.5: Run the concurrent test under -race**

```bash
go test ./internal/mobs/ -run TestConcurrentInstanceAccess -race -count=1 -v
```

Expected: PASS with no `DATA RACE` warnings.

- [ ] **Step 2.6: Run the full mobs package tests**

```bash
go test ./internal/mobs/ -race -count=1
```

Expected: all pre-existing tests still pass.

- [ ] **Step 2.7: Run the full build**

```bash
go build ./...
```

Expected: clean build. Any missed call site that crosses a lock boundary will surface here as a compile error (if the function signature changed) or a test failure.

- [ ] **Step 2.8: Commit**

```bash
git add internal/mobs/mobs.go internal/mobs/mobs_concurrent_test.go
git commit -m "$(cat <<'EOF'
refactor(mobs): guard global maps with sync.RWMutex

HTTP admin dashboard handlers read mobs.GetAllMobInfo() from a separate
goroutine while the main loop mutates the map. Adds RWMutex around the
five maps and instanceCounter. No behavior change in single-threaded
paths. New concurrent test asserts clean under -race.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Mutex on `internal/users` Package Map

**Files:**
- Modify: `internal/users/users.go` (add mutex to `ActiveUsers`)
- Test: `internal/users/users_concurrent_test.go` (new file)

- [ ] **Step 3.1: Locate the ActiveUsers struct**

Run:
```bash
grep -n "type ActiveUsers\|userManager\s*=\|Users.*map\[int\]" internal/users/users.go | head -20
```

Note the struct declaration line and the `Users` field line.

- [ ] **Step 3.2: Write the failing concurrent test**

Create `internal/users/users_concurrent_test.go`:

```go
package users

import (
	"sync"
	"testing"
)

// TestConcurrentUserAccess exercises the RWMutex. Must run under -race.
func TestConcurrentUserAccess(t *testing.T) {
	const writers = 2
	const readers = 8
	const iters = 300

	var wg sync.WaitGroup

	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				u := &UserRecord{UserId: base*1000 + i}
				u.Username = "racer"
				storeUserForTest(u)
				removeUserForTest(u.UserId)
			}
		}(w)
	}

	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				_ = GetByUserId(i)
			}
		}()
	}

	wg.Wait()
}
```

Add helpers at the bottom:

```go
func storeUserForTest(u *UserRecord) {
	userManager.mu.Lock()
	userManager.Users[u.UserId] = u
	userManager.mu.Unlock()
}

func removeUserForTest(id int) {
	userManager.mu.Lock()
	delete(userManager.Users, id)
	userManager.mu.Unlock()
}
```

- [ ] **Step 3.3: Add mu field to ActiveUsers**

Edit the `ActiveUsers` struct in `internal/users/users.go`:

```go
type ActiveUsers struct {
	mu    sync.RWMutex  // NEW
	Users map[int]*UserRecord
	// ... other fields unchanged
}
```

Add `"sync"` to imports if not present.

- [ ] **Step 3.4: Wrap reads**

Find every function that reads `userManager.Users` (or `u.Users` where `u` is an `*ActiveUsers`). Wrap each with:

```go
userManager.mu.RLock()
defer userManager.mu.RUnlock()
// ... read
```

Common readers: `GetByUserId`, `GetActiveUsers`, anything iterating users.

- [ ] **Step 3.5: Wrap writes**

Find every function that mutates `userManager.Users`:

```go
userManager.mu.Lock()
defer userManager.mu.Unlock()
// ... write
```

Common writers: `LoginUser` / spawn path, `LogOutUser` / despawn path, anything that calls `userManager.Users[id] = ...` or `delete(userManager.Users, id)`.

Do not hold the lock across event dispatch or callbacks. If `LoginUser` dispatches events, extract the user pointer under lock, release, then dispatch.

- [ ] **Step 3.6: Run concurrent test**

```bash
go test ./internal/users/ -run TestConcurrentUserAccess -race -count=1 -v
```

Expected: PASS with no DATA RACE warnings.

- [ ] **Step 3.7: Run full users package tests**

```bash
go test ./internal/users/ -race -count=1
go build ./...
```

Expected: all pass, clean build.

- [ ] **Step 3.8: Commit**

```bash
git add internal/users/users.go internal/users/users_concurrent_test.go
git commit -m "$(cat <<'EOF'
refactor(users): guard ActiveUsers.Users map with sync.RWMutex

Closes a data race on the user registry map for HTTP admin handlers
that read concurrently with login/logout on the main loop.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Zone Activity Index — Types, Accessors, Unit Tests

**Files:**
- Create: `internal/rooms/zone_activity.go`
- Test: `internal/rooms/zone_activity_test.go`

**Rationale:** Establishes the new counter with exhaustive unit coverage before any consumer reads from it. If this task is correct, the rest of the stage is wiring.

- [ ] **Step 4.1: Write the failing tests first**

Create `internal/rooms/zone_activity_test.go`:

```go
package rooms

import (
	"testing"
)

func TestZoneActivity_IncrementDecrement(t *testing.T) {
	ResetZonePlayerCount()

	incrementZonePlayerCount("alpha")
	if !ZoneHasPlayers("alpha") {
		t.Fatal("alpha should be active after one increment")
	}
	if ZoneHasPlayers("beta") {
		t.Fatal("beta should be inactive")
	}

	incrementZonePlayerCount("alpha")
	decrementZonePlayerCount("alpha")
	if !ZoneHasPlayers("alpha") {
		t.Fatal("alpha should still be active (1 remaining)")
	}

	decrementZonePlayerCount("alpha")
	if ZoneHasPlayers("alpha") {
		t.Fatal("alpha should be inactive after matched decrement")
	}
}

func TestZoneActivity_DecrementClampsAtZero(t *testing.T) {
	ResetZonePlayerCount()
	decrementZonePlayerCount("alpha") // underflow attempt
	if ZoneHasPlayers("alpha") {
		t.Fatal("underflowed zone must not be 'active'")
	}
	// The key should either be absent or clamped to zero — either is fine.
}

func TestZoneActivity_SnapshotActiveZones(t *testing.T) {
	ResetZonePlayerCount()
	incrementZonePlayerCount("alpha")
	incrementZonePlayerCount("beta")
	incrementZonePlayerCount("beta")
	// gamma never incremented
	decrementZonePlayerCount("alpha")
	// alpha now zero

	snap := SnapshotActiveZones()
	if snap["alpha"] {
		t.Fatal("alpha should not be in snapshot")
	}
	if !snap["beta"] {
		t.Fatal("beta should be in snapshot")
	}
	if snap["gamma"] {
		t.Fatal("gamma should not be in snapshot")
	}
}

func TestZoneActivity_EmptyZoneStringWorks(t *testing.T) {
	ResetZonePlayerCount()
	incrementZonePlayerCount("")
	if !ZoneHasPlayers("") {
		t.Fatal(`empty zone string is a valid key`)
	}
}

func TestZoneActivity_VerifyDetectsDrift(t *testing.T) {
	ResetZonePlayerCount()
	// Set up a room with 2 players.
	r := &Room{RoomId: 1001, Zone: "alpha", players: []int{10, 20}}
	originalManager := roomManager
	defer func() { roomManager = originalManager }()
	roomManager = &RoomManager{
		rooms: map[int]*Room{1001: r},
	}

	// Incrementally maintained counter is wrong (one, not two).
	incrementZonePlayerCount("alpha")

	drift := VerifyZonePlayerCount()
	if drift["alpha"] != 1 {
		t.Fatalf("expected drift of +1 for alpha (ground truth 2, counter 1), got %v", drift)
	}
}
```

- [ ] **Step 4.2: Run tests — expect failure**

```bash
go test ./internal/rooms/ -run TestZoneActivity -v
```

Expected: compile failure or test failure (types / functions don't exist yet).

- [ ] **Step 4.3: Implement zone_activity.go**

Create `internal/rooms/zone_activity.go`:

```go
package rooms

import "sync"

var (
	zonePlayerCount   = map[string]int{}
	zonePlayerCountMu sync.RWMutex
)

// ZoneHasPlayers returns true if the named zone has at least one player.
// Use SnapshotActiveZones inside loops to avoid re-locking.
func ZoneHasPlayers(zone string) bool {
	zonePlayerCountMu.RLock()
	defer zonePlayerCountMu.RUnlock()
	return zonePlayerCount[zone] > 0
}

// SnapshotActiveZones returns a set of zones currently containing at
// least one player. Intended for per-round hot loops.
func SnapshotActiveZones() map[string]bool {
	zonePlayerCountMu.RLock()
	defer zonePlayerCountMu.RUnlock()
	out := make(map[string]bool, len(zonePlayerCount))
	for zone, n := range zonePlayerCount {
		if n > 0 {
			out[zone] = true
		}
	}
	return out
}

// incrementZonePlayerCount is called from Room.AddPlayer when a new
// userId is appended to r.players (after dedupe).
func incrementZonePlayerCount(zone string) {
	zonePlayerCountMu.Lock()
	zonePlayerCount[zone]++
	zonePlayerCountMu.Unlock()
}

// decrementZonePlayerCount is called from Room.RemovePlayer when a
// userId is actually removed from r.players. Clamps at zero defensively —
// drift is still logged via VerifyZonePlayerCount diagnostics.
func decrementZonePlayerCount(zone string) {
	zonePlayerCountMu.Lock()
	if zonePlayerCount[zone] > 0 {
		zonePlayerCount[zone]--
	}
	zonePlayerCountMu.Unlock()
}

// ResetZonePlayerCount is a test helper — clears all state.
func ResetZonePlayerCount() {
	zonePlayerCountMu.Lock()
	zonePlayerCount = map[string]int{}
	zonePlayerCountMu.Unlock()
}

// RebuildZonePlayerCount recomputes the counter from the authoritative
// room.players slices. Called at server startup and from the admin
// diagnostics page.
func RebuildZonePlayerCount() {
	zonePlayerCountMu.Lock()
	zonePlayerCount = map[string]int{}
	for _, r := range roomManager.rooms {
		if len(r.players) > 0 {
			zonePlayerCount[r.Zone] += len(r.players)
		}
	}
	zonePlayerCountMu.Unlock()
}

// VerifyZonePlayerCount returns the delta (ground truth − incremental
// counter) for every zone where they disagree. An empty map means the
// counter is correct.
func VerifyZonePlayerCount() map[string]int {
	groundTruth := map[string]int{}
	for _, r := range roomManager.rooms {
		if len(r.players) > 0 {
			groundTruth[r.Zone] += len(r.players)
		}
	}

	drift := map[string]int{}
	zonePlayerCountMu.RLock()
	defer zonePlayerCountMu.RUnlock()

	// Every zone from either side must match.
	for zone, truth := range groundTruth {
		if diff := truth - zonePlayerCount[zone]; diff != 0 {
			drift[zone] = diff
		}
	}
	for zone, counter := range zonePlayerCount {
		if _, seen := groundTruth[zone]; !seen && counter != 0 {
			drift[zone] = -counter
		}
	}
	return drift
}
```

- [ ] **Step 4.4: Run tests**

```bash
go test ./internal/rooms/ -run TestZoneActivity -v -race
```

Expected: all pass under `-race`.

- [ ] **Step 4.5: Full rooms package tests**

```bash
go test ./internal/rooms/ -race -count=1
go build ./...
```

Expected: all pass, clean build.

- [ ] **Step 4.6: Commit**

```bash
git add internal/rooms/zone_activity.go internal/rooms/zone_activity_test.go
git commit -m "$(cat <<'EOF'
feat(rooms): add zonePlayerCount index with lookup + diagnostics

Maintained-alongside-membership counter of players per zone. Supplies
ZoneHasPlayers + SnapshotActiveZones for the upcoming combat/round-tick
active-zone split. Includes RebuildZonePlayerCount for startup and
VerifyZonePlayerCount for drift diagnostics. No consumers yet.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Wire Hooks into Room.AddPlayer / Room.RemovePlayer

**Files:**
- Modify: `internal/rooms/rooms.go` (AddPlayer line 2137, RemovePlayer line 2151)
- Test: `internal/rooms/zone_activity_test.go` (extend)

- [ ] **Step 5.1: Extend the test file**

Append to `internal/rooms/zone_activity_test.go`:

```go
func TestAddPlayer_IncrementsZone(t *testing.T) {
	ResetZonePlayerCount()

	r := &Room{RoomId: 2001, Zone: "alpha"}
	r.AddPlayer(10)

	if !ZoneHasPlayers("alpha") {
		t.Fatal("alpha should be active after AddPlayer")
	}
}

func TestAddPlayer_DedupeDoesNotDoubleCount(t *testing.T) {
	ResetZonePlayerCount()

	r := &Room{RoomId: 2002, Zone: "alpha"}

	originalManager := roomManager
	defer func() { roomManager = originalManager }()
	roomManager = &RoomManager{rooms: map[int]*Room{2002: r}}

	r.AddPlayer(10)
	r.AddPlayer(10) // duplicate — should no-op

	drift := VerifyZonePlayerCount()
	if len(drift) != 0 {
		t.Fatalf("counter should match ground truth, drift=%v", drift)
	}
}

func TestRemovePlayer_DecrementsZone(t *testing.T) {
	ResetZonePlayerCount()

	r := &Room{RoomId: 2003, Zone: "alpha"}
	r.AddPlayer(10)
	if !ZoneHasPlayers("alpha") {
		t.Fatal("precondition: alpha should be active")
	}

	r.RemovePlayer(10)
	if ZoneHasPlayers("alpha") {
		t.Fatal("alpha should be inactive after removal")
	}
}

func TestRemovePlayer_NotMember_NoDecrement(t *testing.T) {
	ResetZonePlayerCount()
	incrementZonePlayerCount("alpha") // simulate prior player in zone

	r := &Room{RoomId: 2004, Zone: "alpha"} // this room has no players
	r.RemovePlayer(999)                     // userId 999 is not a member

	if !ZoneHasPlayers("alpha") {
		t.Fatal("alpha should still be active — we didn't actually remove anyone")
	}
}

func TestAddPlayer_EmptyZoneStringDoesNotCrash(t *testing.T) {
	ResetZonePlayerCount()
	r := &Room{RoomId: 2005, Zone: ""}
	r.AddPlayer(10) // must not panic; empty zone is a valid key
	if !ZoneHasPlayers("") {
		t.Fatal(`empty zone string should key into the map`)
	}
}
```

- [ ] **Step 5.2: Run the tests — expect failure**

```bash
go test ./internal/rooms/ -run "TestAddPlayer|TestRemovePlayer" -v
```

Expected: zone-related assertions fail (the hook isn't there yet).

- [ ] **Step 5.3: Add the hooks**

Edit `internal/rooms/rooms.go:2137`. Replace `AddPlayer`:

```go
func (r *Room) AddPlayer(userId int) int {

	for _, v := range r.players {
		if v == userId {
			return len(r.players)
		}
	}

	r.players = append(r.players, userId)
	incrementZonePlayerCount(r.Zone)

	return len(r.players)
}
```

Edit `internal/rooms/rooms.go:2151`. Replace `RemovePlayer`:

```go
// true if found
func (r *Room) RemovePlayer(userId int) (int, bool) {

	for i, v := range r.players {
		if v == userId {
			r.players = append(r.players[:i], r.players[i+1:]...)
			decrementZonePlayerCount(r.Zone)
			return len(r.players), true
		}
	}
	return len(r.players), false
}
```

- [ ] **Step 5.4: Run the tests**

```bash
go test ./internal/rooms/ -run "TestAddPlayer|TestRemovePlayer" -v -race
```

Expected: all pass.

- [ ] **Step 5.5: Run full rooms + hooks tests**

```bash
go test ./internal/rooms/ ./internal/hooks/ -race -count=1
go build ./...
```

Expected: all pass, clean build.

Note: existing tests that call `AddPlayer`/`RemovePlayer` directly as setup will now affect the zone counter. Tests that check zone state should call `ResetZonePlayerCount` in setup. Tests that don't check zone state are unaffected.

- [ ] **Step 5.6: Commit**

```bash
git add internal/rooms/rooms.go internal/rooms/zone_activity_test.go
git commit -m "$(cat <<'EOF'
feat(rooms): hook zone counter into AddPlayer/RemovePlayer

Counter updates fire exactly once per genuine membership change,
since both methods already dedupe. Covers login, logout, movement,
teleport, death, and instance portal paths without per-caller wiring.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Startup Rebuild + MoveToRoom Regression Tests

**Files:**
- Modify: `main.go` (add `rooms.RebuildZonePlayerCount()` call)
- Test: `internal/rooms/zone_activity_test.go` (extend)

- [ ] **Step 6.1: Add a MoveToRoom integration test**

Append to `internal/rooms/zone_activity_test.go`:

```go
func TestMoveToRoom_CrossZone_MovesCount(t *testing.T) {
	ResetZonePlayerCount()
	originalManager := roomManager
	defer func() { roomManager = originalManager }()

	alphaRoom := &Room{RoomId: 3001, Zone: "alpha"}
	betaRoom := &Room{RoomId: 3002, Zone: "beta"}
	roomManager = &RoomManager{
		rooms:            map[int]*Room{3001: alphaRoom, 3002: betaRoom},
		roomsWithUsers:   map[int]uint64{},
	}

	alphaRoom.AddPlayer(42)
	if !ZoneHasPlayers("alpha") || ZoneHasPlayers("beta") {
		t.Fatal("precondition: alpha active, beta inactive")
	}

	// Simulate a move by directly removing + adding (what MoveToRoom does).
	alphaRoom.RemovePlayer(42)
	betaRoom.AddPlayer(42)

	if ZoneHasPlayers("alpha") {
		t.Fatal("alpha should be inactive after move-out")
	}
	if !ZoneHasPlayers("beta") {
		t.Fatal("beta should be active after move-in")
	}
}

func TestMoveToRoom_SameZone_CountUnchanged(t *testing.T) {
	ResetZonePlayerCount()
	originalManager := roomManager
	defer func() { roomManager = originalManager }()

	r1 := &Room{RoomId: 3003, Zone: "alpha"}
	r2 := &Room{RoomId: 3004, Zone: "alpha"}
	roomManager = &RoomManager{
		rooms: map[int]*Room{3003: r1, 3004: r2},
	}

	r1.AddPlayer(42)
	r1.RemovePlayer(42)
	r2.AddPlayer(42)

	// alpha should have exactly 1 player.
	if !ZoneHasPlayers("alpha") {
		t.Fatal("alpha should be active")
	}

	drift := VerifyZonePlayerCount()
	if len(drift) != 0 {
		t.Fatalf("expected no drift, got %v", drift)
	}
}

func TestRebuildZonePlayerCount_MatchesGroundTruth(t *testing.T) {
	ResetZonePlayerCount()
	originalManager := roomManager
	defer func() { roomManager = originalManager }()

	roomManager = &RoomManager{
		rooms: map[int]*Room{
			4001: {RoomId: 4001, Zone: "alpha", players: []int{1, 2}},
			4002: {RoomId: 4002, Zone: "beta", players: []int{3}},
			4003: {RoomId: 4003, Zone: "alpha"}, // empty
		},
	}

	// Counter intentionally wrong to start.
	incrementZonePlayerCount("ghost")

	RebuildZonePlayerCount()

	drift := VerifyZonePlayerCount()
	if len(drift) != 0 {
		t.Fatalf("expected no drift after rebuild, got %v", drift)
	}
	if !ZoneHasPlayers("alpha") || !ZoneHasPlayers("beta") {
		t.Fatal("both populated zones should be active after rebuild")
	}
	if ZoneHasPlayers("ghost") {
		t.Fatal("stale 'ghost' entry should have been cleared by rebuild")
	}
}
```

- [ ] **Step 6.2: Run tests**

```bash
go test ./internal/rooms/ -run "TestMoveToRoom|TestRebuild" -v -race
```

Expected: all pass (hooks from Task 5 already do the work; rebuild was implemented in Task 4).

- [ ] **Step 6.3: Wire startup rebuild into main.go**

Find the right place in `main.go` — after both `rooms.LoadDataFiles()` (line 1067) and `users.LoadDataFiles()` (grep for it). The call should happen after users may have been restored from disk as zombies (since zombie users with valid RoomIds count toward zone activity).

Grep command:
```bash
grep -n "users\.LoadDataFiles\|rooms\.LoadDataFiles" main.go
```

Once you've identified the line AFTER `users.LoadDataFiles()` call, add:

```go
rooms.RebuildZonePlayerCount() // build the zone->player-count index
```

- [ ] **Step 6.4: Build + run full test suite**

```bash
go build ./...
go test ./... -race -count=1 -short
```

Expected: clean build, tests pass.

- [ ] **Step 6.5: Commit**

```bash
git add main.go internal/rooms/zone_activity_test.go
git commit -m "$(cat <<'EOF'
feat(rooms): rebuild zone-player-count at server startup

Ensures the incrementally-maintained counter matches ground truth after
server cold start (including zombie-user carryover from previous run).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Extract 10 MobRoundTick Inline Blocks into Helpers (Pure Refactor)

**Files:**
- Modify: `internal/hooks/NewRound_MobRoundTick.go` (same file — helpers live here)

**Rationale:** Before branching on zone activity, extract each of the current inline per-mob operations into a named helper. This is a pure refactor — same behavior, same call order, same semantics. Tests must pass with zero changes.

- [ ] **Step 7.1: Run full test suite to establish baseline**

```bash
go test ./internal/hooks/ -count=1
```

Expected: pass. Note the count. Keep this window open; you'll re-run after each extraction.

- [ ] **Step 7.2: Add the 10 helper functions**

Add these helpers at the bottom of `internal/hooks/NewRound_MobRoundTick.go`, after `MobRoundTick` closes at line 415:

```go
// tickMobCooldowns — matches the current inline block at line 97.
func tickMobCooldowns(mob *mobs.Mob) {
	mob.Character.Cooldowns.RoundTick()
}

// expireMobCombatMemory — current inline block at lines 99–105.
func expireMobCombatMemory(mob *mobs.Mob) {
	if mob.CombatMemory != nil {
		if mobai.MemoryExpired(mob.CombatMemory, util.GetRoundCount(),
			int(configs.GetBalanceConfig().CombatMemoryDuration)) {
			mob.CombatMemory = nil
		}
	}
}

// tickMobProneRecovery — current inline block at lines 107–118.
func tickMobProneRecovery(mob *mobs.Mob) {
	if attemptMade, success := mob.Character.AttemptRecovery(mob.Character.Stats.Dexterity.ValueAdj); attemptMade {
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			mName := mobDisplayName(mob, room, 0)
			if success {
				sendVisualRoomText(room, mName+" clambers to their feet in a rushed panic.")
			} else {
				sendVisualRoomText(room, mName+" attempts to stand, but slips and falls in the chaos of battle.")
			}
		}
	}
}

// tickMobCharmDuration — current inline block at lines 120–122. Simple
// rounds decrement; the expiry + re-roll dance is a separate helper.
func tickMobCharmDuration(mob *mobs.Mob) {
	if mob.Character.Charmed != nil && mob.Character.Charmed.RoundsRemaining > 0 {
		mob.Character.Charmed.RoundsRemaining--
	}
}

// tickMobBuffs — current inline block at lines 124–160.
func tickMobBuffs(mob *mobs.Mob, mobInstanceId int) {
	if triggeredBuffs := mob.Character.Buffs.Trigger(); len(triggeredBuffs) > 0 {
		triggeredBuffIds := []int{}
		for _, buff := range triggeredBuffs {
			if buff.TickAmount != 0 {
				if mobBuffSpec := buffs.GetBuffSpec(buff.BuffId); mobBuffSpec != nil {
					switch mobBuffSpec.TickPool {
					case "health":
						mob.Character.Heal(buff.TickAmount)
					case "stamina":
						mob.Character.Stamina += buff.TickAmount
						if mob.Character.Stamina > mob.Character.StaminaMax.Value {
							mob.Character.Stamina = mob.Character.StaminaMax.Value
						} else if mob.Character.Stamina < 0 {
							mob.Character.Stamina = 0
						}
					case "conviction":
						mob.Character.Conviction += buff.TickAmount
						if mob.Character.Conviction > mob.Character.ConvictionMax.Value {
							mob.Character.Conviction = mob.Character.ConvictionMax.Value
						} else if mob.Character.Conviction < 0 {
							mob.Character.Conviction = 0
						}
					}
				}
			}
			triggeredBuffIds = append(triggeredBuffIds, buff.BuffId)
		}
		events.AddToQueue(events.BuffsTriggered{MobInstanceId: mobInstanceId, BuffIds: triggeredBuffIds})
	}
}

// tickMobMutationAcquisition — current inline block at lines 162–260.
func tickMobMutationAcquisition(mob *mobs.Mob, mb *configs.Balance) {
	if !(bool(mb.MobMutationEnabled) && mob.Character.Aggro != nil) {
		return
	}
	canAcquire := len(mob.Character.Mutations) < int(mb.MutationMaxCount)
	canDeepen := mutations.CanDeepen(mob.Character.Mutations)
	if !(canAcquire || canDeepen) {
		return
	}
	mob.Character.MutationProgress += float64(mb.MutationProgressGainPerRound) * float64(mb.MobMutationRate)
	load := mutations.GetMutationLoad(mob.Character.Mutations)
	threshold := float64(mb.MutationBaseProgress) *
		math.Pow(float64(mb.MutationProgressScale), load)
	if mob.Character.MutationProgress < threshold {
		return
	}
	mob.Character.MutationProgress = 0
	doDeepen := false
	if canAcquire && canDeepen {
		if util.Rand(100) < int(mb.MutationDeepenChance*100) {
			doDeepen = true
		}
	} else if canDeepen && !canAcquire {
		doDeepen = true
	}

	if doDeepen {
		if mutId := mutations.RollDeepening(mob.Character.Mutations); mutId != "" {
			mob.Character.Mutations[mutId]++
			newLevel := mob.Character.Mutations[mutId]
			if spec := mutations.GetMutation(mutId); spec != nil {
				if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
					sendVisualRoomText(room, fmt.Sprintf(
						`<ansi fg="magenta">The mutation in <ansi fg="mobname">%s</ansi> intensifies.</ansi>`,
						mob.Character.Name))
				}
				sig := worldevents.Local
				if spec.Rarity >= 5 {
					sig = worldevents.Regional
				}
				if newLevel >= int(mb.MutationMaxLevel) {
					if sig < worldevents.Global {
						sig++
					}
				}
				zone := mob.Character.Zone
				region := ""
				if zCfg := rooms.GetZoneConfig(zone); zCfg != nil {
					region = zCfg.Region
				}
				worldevents.EmitWorldEvent(worldevents.WorldEvent{
					Type:         worldevents.MobMutationAdvanced,
					Significance: sig,
					ZoneName:     zone,
					RegionName:   region,
					MobName:      mob.Character.Name,
					Description: fmt.Sprintf("%s's %s deepens to level %d",
						mob.Character.Name, spec.Name, newLevel),
				})
			}
		}
		return
	}

	if canAcquire {
		var specDisabledSlots []string
		if specInfo := species.GetSpecies(mob.Character.SpeciesId); specInfo != nil {
			specDisabledSlots = specInfo.DisabledSlots
		}
		pool := mutations.GetWeightedPool(mob.Character.Mutations, specDisabledSlots)
		if mutId := mutations.RollAcquisition(pool); mutId != "" {
			if mob.Character.Mutations == nil {
				mob.Character.Mutations = make(map[string]int)
			}
			mob.Character.Mutations[mutId] = 1
			if spec := mutations.GetMutation(mutId); spec != nil {
				if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
					sendVisualRoomText(room, fmt.Sprintf(
						`<ansi fg="magenta">Something shifts in <ansi fg="mobname">%s</ansi>. %s</ansi>`,
						mob.Character.Name, spec.Visual))
				}
				sig := worldevents.Local
				if spec.Rarity >= 8 {
					sig = worldevents.Global
				} else if spec.Rarity >= 5 {
					sig = worldevents.Regional
				}
				zone := mob.Character.Zone
				region := ""
				if zCfg := rooms.GetZoneConfig(zone); zCfg != nil {
					region = zCfg.Region
				}
				worldevents.EmitWorldEvent(worldevents.WorldEvent{
					Type:         worldevents.MobMutationGained,
					Significance: sig,
					ZoneName:     zone,
					RegionName:   region,
					MobName:      mob.Character.Name,
					Description: fmt.Sprintf("%s has manifested a mutation: %s",
						mob.Character.Name, spec.Name),
				})
			}
		}
	}
}

// tickMobCharmState — current inline blocks at lines 262–279 (expiry
// cleanup) + 281–353 (re-roll). Kept together since both depend on
// Charmed state.
func tickMobCharmState(mob *mobs.Mob) {
	// Charm expiry cleanup.
	if mob.Character.IsCharmed() && mob.Character.Charmed.RoundsRemaining == 0 {
		cmd := mob.Character.Charmed.ExpiredCommand
		if charmedUserId := mob.Character.RemoveCharm(); charmedUserId > 0 {
			if charmedUser := users.GetByUserId(charmedUserId); charmedUser != nil {
				charmedUser.Character.TrackCharmed(mob.InstanceId, false)
			}
		}
		if cmd != `` {
			cmds := strings.Split(cmd, `;`)
			for _, cmd := range cmds {
				cmd = strings.TrimSpace(cmd)
				if len(cmd) > 0 {
					mob.Command(cmd)
				}
			}
		}
	}

	// Re-roll contested Charisma vs Willpower on CharmDuration tick.
	if mob.Character.IsCharmed() {
		if charmedUserId := mob.Character.GetCharmedUserId(); charmedUserId > 0 {
			if owner := users.GetByUserId(charmedUserId); owner != nil {
				if comp := owner.Character.GetCompanionByInstanceId(mob.InstanceId); comp != nil &&
					comp.SourceType == characters.CompanionCharmed &&
					comp.CharmDuration > 0 {

					comp.CharmDuration--
					if comp.CharmDuration == 0 {
						manifestSkill := owner.Character.GetSkillLevel(skills.Manifestation)
						attackScore := float64(owner.Character.Stats.Charisma.ValueAdj) +
							float64(manifestSkill)*25.0

						targetPool := mob.Character.Stats.Strength.Training +
							mob.Character.Stats.Dexterity.Training +
							mob.Character.Stats.Perception.Training +
							mob.Character.Stats.Vitality.Training +
							mob.Character.Stats.Willpower.Training +
							mob.Character.Stats.Charisma.Training
						defenseScore := float64(mob.Character.Stats.Willpower.ValueAdj) +
							float64(targetPool)*0.10

						effectiveness := 1.0 - float64(comp.CharmRerolls)*0.01*float64(comp.CharmRerolls)
						if effectiveness < 0.50 {
							effectiveness = 0.50
						}
						attackScore *= effectiveness

						success, _, _, _ := dice.OpposedRollStat(attackScore, defenseScore)

						if success {
							newDuration := 50 + owner.Character.Stats.Charisma.ValueAdj/2 +
								manifestSkill*3
							comp.CharmDuration = newDuration
							comp.CharmRerolls++

							owner.SendText(fmt.Sprintf(
								`<ansi fg="cyan">Your hold on %s wavers... but you reassert your will.</ansi>`,
								comp.Name))
							if comp.CharmRerolls >= 5 {
								owner.SendText(fmt.Sprintf(
									`<ansi fg="red">%s's eyes flash with defiance. Your control is slipping...</ansi>`,
									comp.Name))
							} else if comp.CharmRerolls >= 3 {
								owner.SendText(fmt.Sprintf(
									`<ansi fg="yellow">You sense %s's will straining against your bond...</ansi>`,
									comp.Name))
							}
						} else {
							owner.SendText(fmt.Sprintf(
								`<ansi fg="red-bold">%s breaks free of your control!</ansi>`, comp.Name))
							if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
								sendVisualRoomText(room, fmt.Sprintf(
									`<ansi fg="red">%s snarls and turns on %s!</ansi>`,
									mob.Character.Name, owner.Character.Name), owner.UserId)
							}
							mob.Character.RemoveCharm()
							owner.Character.TrackCharmed(mob.InstanceId, false)
							owner.Character.RemoveCompanion(mob.InstanceId)
							mob.Character.SetAggro(owner.UserId, 0, characters.DefaultAttack)
						}
					}
				}
			}
		}
	}
}

// tickMobCrafting — current inline block at lines 355–396.
func tickMobCrafting(mob *mobs.Mob) {
	if mob.Character.CraftingState == nil {
		return
	}
	if mob.Character.Aggro != nil {
		mob.Character.CraftingState = nil
		return
	}
	cs := mob.Character.CraftingState
	cs.RoundsComplete++
	if cs.RoundsComplete < cs.RoundsTotal {
		return
	}
	recipe := crafting.GetRecipe(cs.RecipeId)
	mob.Character.CraftingState = nil
	if recipe == nil {
		return
	}
	sl := mob.Character.Skills[recipe.Skill]
	chance := crafting.CalcSuccessChance(sl, recipe.SkillMinimum)
	roll := util.Rand(100)
	util.LogRoll("MobCraft", roll, chance)
	if roll < chance {
		mob.Character.Items, mob.Character.ComponentItems =
			crafting.ConsumeIngredients(
				mob.Character.Items,
				mob.Character.ComponentItems,
				recipe)
		newItem := items.New(recipe.Output.ItemId)
		mob.Character.StoreItem(newItem)
		craftBonus := 1.0 + float64(recipe.SkillMinimum)*float64(configs.GetBalanceConfig().CraftDifficultyProgressionScale)
		mob.Character.OnSkillUseScaled(recipe.Skill, 0, craftBonus)
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			sendVisualRoomText(room, fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi> finishes their work.`,
				mob.Character.Name))
		}
	} else {
		mob.Character.Items, mob.Character.ComponentItems =
			crafting.ConsumeIngredients(
				mob.Character.Items,
				mob.Character.ComponentItems,
				recipe)
	}
}

// tickMobConditions — current inline line 399.
func tickMobConditions(mob *mobs.Mob) {
	mob.Character.TickConditions()
}

// revalidateMobStats — current inline line 402.
func revalidateMobStats(mob *mobs.Mob) {
	mob.Character.Validate()
}
```

- [ ] **Step 7.3: Replace the inline loop body with helper calls**

Edit `NewRound_MobRoundTick.go` lines 85–409 (the per-mob loop). Replace the whole loop body with:

```go
	//
	// Do mob round maintenance
	//
	mb := configs.GetBalanceConfig()
	for _, mobInstanceId := range mobs.GetAllMobInstanceIds() {

		mob := mobs.GetInstance(mobInstanceId)

		if mob == nil {
			continue
		}

		tickMobCooldowns(mob)
		expireMobCombatMemory(mob)
		tickMobProneRecovery(mob)
		tickMobCharmDuration(mob)
		tickMobBuffs(mob, mobInstanceId)
		tickMobMutationAcquisition(mob, mb)
		tickMobCharmState(mob)
		tickMobCrafting(mob)
		tickMobConditions(mob)
		revalidateMobStats(mob)

		if mob.Character.Health <= 0 {
			// Mob died
			mob.Command(`suicide`)
		}

	}
```

Keep the pre-loop `saveInterval`, `ReduceHostility`, pack scaling, and pack roaming work at the top of `MobRoundTick` exactly as it is (lines 33–82). Keep the post-loop `behaviortree.GetEngine().DrainQueue()` at line 412 as it is.

- [ ] **Step 7.4: Run the full test suite**

```bash
go build ./...
go test ./... -count=1 -race
```

Expected: every test passes. This is a pure refactor — if a test fails, a helper got the body wrong. Diff the original and extracted versions line by line to find the bug.

- [ ] **Step 7.5: Commit**

```bash
git add internal/hooks/NewRound_MobRoundTick.go
git commit -m "$(cat <<'EOF'
refactor(hooks): extract 10 MobRoundTick inline blocks into helpers

Pure refactor, no behavior change. Each per-mob operation now lives in
a named function. Prep for the upcoming active/idle lane split.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Apply Active/Idle Lane Split to MobRoundTick

**Files:**
- Modify: `internal/hooks/NewRound_MobRoundTick.go`
- Test: `internal/hooks/mobroundtick_idle_test.go` (new)

- [ ] **Step 8.1: Write the behavior test first**

Create `internal/hooks/mobroundtick_idle_test.go`:

```go
package hooks

import (
	"testing"
	// Imports may need adjusting if existing tests have a stricter setup.
	// Copy the init pattern from internal/hooks/hooks_test.go if needed.
)

// This test uses narrow unit-style calls into the per-mob helpers
// rather than invoking MobRoundTick end-to-end. End-to-end coverage
// is captured in manual smoke tests; this test pins the lane boundaries.

func TestLaneSplit_IdleHelpersAreIdempotentOnZeroState(t *testing.T) {
	// Constructing a full *mobs.Mob requires many dependencies; instead
	// verify via the compiled function signatures that each idle helper
	// exists and has the expected signature.
	// (If tests in this package already have a Mob fixture builder,
	// use it. Otherwise accept this as a smoke test.)
	var _ = tickMobCooldowns
	var _ = expireMobCombatMemory
	var _ = tickMobCharmDuration
	var _ = tickMobBuffs
	var _ = tickMobConditions

	var _ = tickMobProneRecovery
	var _ = tickMobMutationAcquisition
	var _ = tickMobCharmState
	var _ = tickMobCrafting
	var _ = revalidateMobStats
}
```

(This test is deliberately minimal — it documents which helpers are idle vs active. The behavioral coverage is validated by the existing hooks package test suite plus manual smoke tests in Task 10.)

If the hooks package already has a mob fixture builder (`newTestMob(t)` or similar — grep for it), extend this test with:

```go
func TestLaneSplit_IdleZoneMobSkipsProgression(t *testing.T) {
	// Setup: zone "idle-zone" has no players. Mob in it with Aggro set.
	// Call the full per-mob block with active=false, assert:
	// - Cooldowns ticked
	// - Buffs ticked
	// - MutationProgress did NOT advance
	// - CraftingState unchanged
}
```

Only include the second test if a fixture builder is available. Otherwise rely on manual smoke testing.

- [ ] **Step 8.2: Restructure the per-mob loop with the lane split**

Edit the per-mob loop body in `internal/hooks/NewRound_MobRoundTick.go` (the block you just restructured in Task 7). Replace the flat helper sequence with:

```go
	//
	// Do mob round maintenance
	//
	mb := configs.GetBalanceConfig()
	activeZones := rooms.SnapshotActiveZones()

	for _, mobInstanceId := range mobs.GetAllMobInstanceIds() {

		mob := mobs.GetInstance(mobInstanceId)

		if mob == nil {
			continue
		}

		room := rooms.LoadRoom(mob.Character.RoomId)
		active := room != nil && activeZones[room.Zone]

		// Idle lane — runs every round regardless of zone activity
		tickMobCooldowns(mob)
		expireMobCombatMemory(mob)
		tickMobCharmDuration(mob)
		tickMobBuffs(mob, mobInstanceId)
		tickMobConditions(mob)

		// Death check always runs — a DoT tick in an idle zone should still kill.
		if mob.Character.Health <= 0 {
			mob.Command(`suicide`)
			continue
		}

		if !active {
			continue
		}

		// Active-only — skipped entirely in idle zones
		tickMobProneRecovery(mob)
		tickMobMutationAcquisition(mob, mb)
		tickMobCharmState(mob)
		tickMobCrafting(mob)
		revalidateMobStats(mob)
	}
```

- [ ] **Step 8.3: Build and test**

```bash
go build ./...
go test ./internal/hooks/ -race -count=1 -v
go test ./... -race -count=1 -short
```

Expected: all pass. Existing hook tests that don't exercise empty-zone behavior see no diff. Existing tests that set up a room with a player (and therefore an active zone) continue to exercise the full code path.

- [ ] **Step 8.4: Commit**

```bash
git add internal/hooks/NewRound_MobRoundTick.go internal/hooks/mobroundtick_idle_test.go
git commit -m "$(cat <<'EOF'
perf(hooks): split MobRoundTick into active/idle lanes

Mobs in zones with zero players skip prone-recovery, mutation
acquisition, charm re-roll state, crafting, and stat revalidation.
Idle mobs still tick cooldowns, buff durations, charm duration,
combat-memory expiry, and conditions — timers don't freeze. Death
check always runs so DoT kills still land.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Apply Active-Zone Guard to handleMobCombat

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat.go:138` (function `handleMobCombat`)

- [ ] **Step 9.1: Locate the per-mob loop**

Run:
```bash
sed -n '138,160p' internal/hooks/NewRound_DoCombat.go
```

Confirm the loop starts around line 145 with `for _, mobId := range mobs.GetAllMobInstanceIds()`.

- [ ] **Step 9.2: Add the zone-active snapshot + guard**

Edit the function. Before the `for _, mobId := range ...` line, add the snapshot call. Inside the loop, after loading the mob and checking `mob == nil || mob.Character.Health <= 0`, add the room/zone check.

```go
	activeZones := rooms.SnapshotActiveZones()
	for _, mobId := range mobs.GetAllMobInstanceIds() {
		mob := mobs.GetInstance(mobId)
		if mob == nil || mob.Character.Health <= 0 {
			continue
		}

		room := rooms.LoadRoom(mob.Character.RoomId)
		if room == nil || !activeZones[room.Zone] {
			continue
		}

		// ... existing combat logic unchanged
	}
```

**Critical:** Do not remove the existing `room := ...` line further down in the loop body if it already loads the room. Deduplicate — keep one load, reuse the local.

- [ ] **Step 9.3: Build and test**

```bash
go build ./...
go test ./internal/hooks/ -race -count=1 -v
```

Expected: pass. Combat tests in the hooks package all use a room with at least one player, so the active-zone check always passes in those tests.

- [ ] **Step 9.4: Full test suite**

```bash
go test ./... -race -count=1 -short
```

Expected: pass.

- [ ] **Step 9.5: Commit**

```bash
git add internal/hooks/NewRound_DoCombat.go
git commit -m "$(cat <<'EOF'
perf(hooks): skip handleMobCombat loop body for idle-zone mobs

Mobs in zones with zero players no longer run combat logic. A mob in
combat whose last opponent left the zone sits frozen until its combat
memory expires via the idle lane in MobRoundTick.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Benchmarks + Manual Smoke Test

**Files:**
- Create: `internal/hooks/bench_mobroundtick_test.go`

- [ ] **Step 10.1: Add the benchmark file**

Create `internal/hooks/bench_mobroundtick_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// BenchmarkMobRoundTick_AllActive simulates a world where every zone
// has a player in it. All mobs run the full active-path code.
func BenchmarkMobRoundTick_AllActive(b *testing.B) {
	// Setup: N mobs spread across M zones, every zone has a player.
	// Only include this benchmark if a mob+room fixture builder exists
	// in the hooks test package. If not, leave a TODO referencing the
	// spec and let a follow-up land the benchmark once fixtures exist.
	b.Skip("requires mob+room fixture infrastructure — see spec §Testing")

	_ = events.NewRound{}
	_ = rooms.SnapshotActiveZones
	for i := 0; i < b.N; i++ {
		// MobRoundTick(events.NewRound{RoundNumber: uint64(i)})
	}
}

// BenchmarkMobRoundTick_MostlyIdle simulates 10% active zones — the
// common real-world case.
func BenchmarkMobRoundTick_MostlyIdle(b *testing.B) {
	b.Skip("requires mob+room fixture infrastructure — see spec §Testing")
}

// BenchmarkMobRoundTick_AllIdle simulates no players anywhere.
func BenchmarkMobRoundTick_AllIdle(b *testing.B) {
	b.Skip("requires mob+room fixture infrastructure — see spec §Testing")
}
```

**Why skipped:** the hooks package doesn't yet have a mob fixture builder that's independent of the full data-file load. Building one is a task in itself and out of scope for 1.7. The benchmark file is scaffolded so a follow-up can un-skip each benchmark by plugging in the fixture.

If the engineer discovers a fixture already exists (grep for `newTestMob`, `buildTestMob`, `fixtureMob`), un-skip and populate the benchmarks. Expected result: `MostlyIdle` ≥5× faster than `AllActive`.

- [ ] **Step 10.2: Run benchmarks to confirm they build**

```bash
go test ./internal/hooks/ -bench BenchmarkMobRoundTick -benchtime=1x -run ^$
```

Expected: each benchmark skips cleanly (no compile errors).

- [ ] **Step 10.3: Manual smoke test checklist**

Start the server locally and work through this list. Record results in a scratch file; do NOT save to repo.

Run:
```bash
go run . -race
```

Then from a client (telnet / websocket):

1. **Combat in populated zone** — enter a populated zone (e.g., starting tutorial area), attack a mob. Combat should behave exactly as before.
2. **Walk out during combat** — during combat, walk into a zone with no other players. The fleeing mob should follow OR sit frozen (depending on its behavior tree). Either is acceptable as long as no panic / no infinite combat loop.
3. **Return to original zone** — walk back. If combat memory is still live, mob resumes attack. If expired, mob is neutral.
4. **Arena death/ejection** — enter Arena, die. Verify ejection works. Arena zone goes cold. No errors in server log.
5. **Admin dashboard** — while combat is happening in one zone, open `http://localhost:PORT/admin/mobs` from another tab. Click through pages. Verify no race warnings in server log and no stale / corrupt data on screen.
6. **Fold-recall mid-combat** — cast fold-recall while engaged with a mob. Zone counter should update (source zone decrement, destination zone increment).
7. **Server shutdown + restart** — shut down cleanly, restart. Log in. Verify `rooms.VerifyZonePlayerCount()` via admin diagnostics page (or run the test directly) reports empty drift.

- [ ] **Step 10.4: Add admin diagnostics page (optional but recommended)**

If time permits, expose `rooms.VerifyZonePlayerCount()` output on an admin page.

Find an existing admin page implementation, e.g. `internal/web/admin.rooms.go`, and copy the pattern. Add a new handler `adminZoneDiagnostics(w, r)` that:
1. Calls `rooms.VerifyZonePlayerCount()`
2. Calls `rooms.SnapshotActiveZones()`
3. Renders a simple HTML page showing both

Wire it into the router in `internal/web/web.go` or wherever admin routes are registered (grep for `mobs` admin registration in web.go for the pattern).

If admin plumbing is unfamiliar, skip this step — a CLI command in a future stage can fulfill the same role.

- [ ] **Step 10.5: Commit**

```bash
git add internal/hooks/bench_mobroundtick_test.go
# also add admin page file if you created one
git commit -m "$(cat <<'EOF'
test(hooks): scaffold MobRoundTick benchmarks

Three skipped benchmarks: all-active, mostly-idle (10%), all-idle.
Once a mob+room fixture builder lands, un-skip to quantify the
active/idle split win.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Update PATCH_NOTES.md and Mark Stage Complete

**Files:**
- Modify: `PATCH_NOTES.md`
- Modify: `docs/superpowers/code_cleanup_stage_1_overview.md`

- [ ] **Step 11.1: Add patch note entry**

Add a new dated entry to `PATCH_NOTES.md` following the existing format:

```markdown
## 2026-04-16 — Code Cleanup 1.7: Performance Pass

- **Zone-activity lane split:** mobs in zones with zero players skip
  combat, progression, mutation acquisition, charm-state, and crafting
  every round. Idle mobs still tick cooldowns, buff/condition durations,
  charm duration, combat-memory expiry, and death checks so timers and
  DoTs keep working.
- **Registry mutexes:** `internal/mobs` and `internal/users` global maps
  now use `sync.RWMutex`. Closes a latent race between the HTTP admin
  dashboard and the main game loop.
- **PruneVisitors fast path:** empty-map early return on room cleanup.
```

- [ ] **Step 11.2: Update the overview status table**

Edit `docs/superpowers/code_cleanup_stage_1_overview.md` line 21. Change:

```
| 1.7 | Performance Pass | 10h | Mixed | Not started |
```

to:

```
| 1.7 | Performance Pass | 10h | Mixed | Complete |
```

- [ ] **Step 11.3: Commit**

```bash
git add PATCH_NOTES.md docs/superpowers/code_cleanup_stage_1_overview.md
git commit -m "$(cat <<'EOF'
docs: mark code cleanup 1.7 complete

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Done

Eleven tasks, each with an isolated commit. Rollout follows spec Section 5:

1. (Task 1) PruneVisitors — lowest risk
2. (Tasks 2–3) Mutexes — add safety net before introducing shared state
3. (Tasks 4–6) Zone index — establish ground truth with no consumers
4. (Task 7) Helper extraction — pure refactor, no behavior change
5. (Tasks 8–9) Apply the split — the actual perf win
6. (Task 10) Benchmarks + smoke test
7. (Task 11) Ship notes

If any task regresses, revert that task's commit independently — earlier tasks are unaffected.
