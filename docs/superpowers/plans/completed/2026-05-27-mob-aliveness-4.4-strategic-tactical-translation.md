# Mob Aliveness 4.4 — Strategic → Tactical Translation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship 13 deep per-goal-type planners + framework so NPCs visibly pursue 4.3's current goals — combat-capable mobs flee when low on HP, thieves wander to vendors and sell loot when seeded with `wealth-gold`, named NPCs pursue revenge/protection/befriend targets, foragers visit unvisited zones, crafters seek stations to produce known recipes.

**Architecture:** A new `internal/planners/` package registers one `PlanFn` per goal type via init() (mirrors 4.3's catalog pattern). Planners are pure Go functions called every mob round tick when a goal of their type is current; intermediate progress lives in `mob.Character.MiscData` under a `plan:<goal_type>:` key prefix, wiped on goal switch via a registered `SetPlanStateClear` callback. One new btree action `try_goal_planner` dispatches per `goals.CurrentGoalOf` lookup; authors insert it explicitly into each archetype's tree.

**Tech Stack:** Go 1.25 · existing `mudlog`, `configs`, `util`, `mobs`, `goals`, `behaviortree`, `items`, `rooms`, `factions`, `shops`, `crafting`, `users`, `opinions` packages.

**Spec:** `docs/superpowers/specs/2026-05-27-mob-aliveness-4.4-strategic-tactical-translation-design.md`

---

## Task 1 — Planner framework: types, registry, lookup

The skeleton. Pure types + a registration map. No planners registered yet; no behavior change yet.

**Files:**
- Create: `internal/planners/planners.go`
- Create: `internal/planners/planners_test.go`

- [ ] **Step 1.1: Create the package file with types + registry**

Create `internal/planners/planners.go`:

```go
// Package planners contains chunk-4.4 per-goal-type planners that turn
// the current strategic goal (selected by chunk 4.2 from the chunk 4.3
// catalog) into concrete tactical actions per mob round tick.
//
// Each goal type's planner lives in its own <type>.go file. Each file's
// init() calls RegisterPlanner. main.go pulls these registrations via
// a blank import.
//
// Planners are stateless from the framework's perspective. For multi-
// step plans, write intermediate progress to mob.Character.MiscData
// under the convention "plan:<goal_type>:<key>". State is wiped
// automatically on goal switch via ClearPlanState (registered into
// goals.Recompute via SetPlanStateClear at boot).
//
// See docs/superpowers/specs/2026-05-27-mob-aliveness-4.4-strategic-tactical-translation-design.md
package planners

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// BTreeStatus mirrors the behavior-tree status enum (re-exported to
// avoid forcing planners to import internal/behaviortree, which would
// create a cycle).
type BTreeStatus int

const (
	StatusFailure BTreeStatus = iota
	StatusSuccess
	StatusRunning
)

// PlanResult is what a planner returns each tick.
type PlanResult struct {
	// Command to execute this tick (empty string = no action; btree falls
	// through to the next node). Executed via mob.Command(cmd) by the
	// try_goal_planner btree action.
	Command string

	// Status propagated as the try_goal_planner btree action's result.
	Status BTreeStatus
}

// PlanFn is the per-tick planner. Stateless from the framework's
// perspective — for multi-step plans, write to mob.Character.MiscData
// under "plan:<goal_type>:" prefix.
type PlanFn func(mob *mobs.Mob, goal *goals.Goal) PlanResult

var (
	registryMu sync.RWMutex
	registry   = map[string]PlanFn{}
)

// RegisterPlanner registers a planner for a goal type. Called from each
// per-type planner file's init() function. Late registrations overwrite
// earlier ones (last-write-wins; useful for test override).
func RegisterPlanner(goalType string, fn PlanFn) {
	registryMu.Lock()
	registry[goalType] = fn
	registryMu.Unlock()
}

// LookupPlanner returns the registered planner for a goal type, or nil
// if none. Called by the try_goal_planner btree action.
func LookupPlanner(goalType string) PlanFn {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[goalType]
}
```

- [ ] **Step 1.2: Write tests**

Create `internal/planners/planners_test.go`:

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestLookupPlanner_Unregistered_ReturnsNil(t *testing.T) {
	if fn := LookupPlanner("nonexistent-type"); fn != nil {
		t.Errorf("got non-nil for unregistered type")
	}
}

func TestRegisterPlanner_RoundTrip(t *testing.T) {
	called := false
	RegisterPlanner("test-roundtrip", func(mob *mobs.Mob, goal *goals.Goal) PlanResult {
		called = true
		return PlanResult{Command: "rest", Status: StatusRunning}
	})
	defer RegisterPlanner("test-roundtrip", nil)

	fn := LookupPlanner("test-roundtrip")
	if fn == nil {
		t.Fatalf("LookupPlanner returned nil after RegisterPlanner")
	}
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "test-roundtrip"})
	if !called {
		t.Errorf("registered fn not invoked")
	}
	if res.Command != "rest" {
		t.Errorf("Command: got %q, want rest", res.Command)
	}
	if res.Status != StatusRunning {
		t.Errorf("Status: got %v, want StatusRunning", res.Status)
	}
}

func TestRegisterPlanner_OverwritesPrevious(t *testing.T) {
	RegisterPlanner("test-overwrite", func(*mobs.Mob, *goals.Goal) PlanResult {
		return PlanResult{Command: "first"}
	})
	RegisterPlanner("test-overwrite", func(*mobs.Mob, *goals.Goal) PlanResult {
		return PlanResult{Command: "second"}
	})
	defer RegisterPlanner("test-overwrite", nil)

	fn := LookupPlanner("test-overwrite")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "test-overwrite"})
	if res.Command != "second" {
		t.Errorf("Command: got %q, want second (last-write-wins)", res.Command)
	}
}

func TestRegisterPlanner_NilUnregisters(t *testing.T) {
	RegisterPlanner("test-nil-unreg", func(*mobs.Mob, *goals.Goal) PlanResult {
		return PlanResult{Command: "x"}
	})
	RegisterPlanner("test-nil-unreg", nil)
	// Map still has the key but value is nil — LookupPlanner returns nil.
	if fn := LookupPlanner("test-nil-unreg"); fn != nil {
		t.Errorf("expected nil after Register(nil), got non-nil")
	}
}
```

- [ ] **Step 1.3: Run tests + build**

Run: `go test ./internal/planners/ -v`
Expected: PASS for 4 tests.

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 1.4: Commit**

```bash
git add internal/planners/planners.go internal/planners/planners_test.go
git commit -m "feat(planners): framework — PlanFn + PlanResult + registry (4.4)" -m "New internal/planners/ subpackage. Defines PlanFn type, PlanResult
(Command + BTreeStatus), and a sync.RWMutex-guarded registry with
RegisterPlanner/LookupPlanner. Per-type planner files (Tasks 8-20)
will call RegisterPlanner in init(); main.go (Task 4) blank-imports
the package to fire those inits.

BTreeStatus is re-exported here to avoid forcing planners to import
internal/behaviortree (which would form a cycle).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2 — ClearPlanState + plan-key prefix

Wipes `plan:`-prefixed MiscData on goal switch. Called via callback (Task 3) from `goals.Recompute`.

**Files:**
- Create: `internal/planners/state.go`
- Create: `internal/planners/state_test.go`

- [ ] **Step 2.1: Create state.go**

```go
package planners

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// PlanKeyPrefix is the MiscData key prefix every planner uses for its
// intermediate state. ClearPlanState wipes all keys with this prefix
// on goal switch.
const PlanKeyPrefix = "plan:"

// ClearPlanState wipes all "plan:" prefixed keys from
// mob.Character.MiscData. Wired into goals.Recompute via a
// SetPlanStateClear callback registered in main.go (Task 4).
//
// Nil-safe on mob == nil and MiscData == nil.
func ClearPlanState(mob *mobs.Mob) {
	if mob == nil || mob.Character.MiscData == nil {
		return
	}
	for k := range mob.Character.MiscData {
		if strings.HasPrefix(k, PlanKeyPrefix) {
			delete(mob.Character.MiscData, k)
		}
	}
}
```

- [ ] **Step 2.2: Write tests**

Create `internal/planners/state_test.go`:

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestClearPlanState_RemovesPrefixedKeys(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.MiscData = map[string]any{
		"plan:wealth-gold:target_shop_room": 12,
		"plan:befriend:cooldown_round":      uint64(5000),
		"plan:visit-zone:next_hop_zone":     "stillwater",
	}
	ClearPlanState(mob)
	if len(mob.Character.MiscData) != 0 {
		t.Errorf("expected 0 keys after Clear, got %d: %v", len(mob.Character.MiscData), mob.Character.MiscData)
	}
}

func TestClearPlanState_LeavesUnprefixedKeysUntouched(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.MiscData = map[string]any{
		"plan:wealth-gold:target_shop_room":   12,
		"faction_kills_inflicted:bandits":     3,
		"conversation_line_idx":               2,
		"some_other_key":                      "value",
	}
	ClearPlanState(mob)
	if _, has := mob.Character.MiscData["plan:wealth-gold:target_shop_room"]; has {
		t.Errorf("plan: key not wiped")
	}
	if mob.Character.MiscData["faction_kills_inflicted:bandits"] != 3 {
		t.Errorf("non-prefixed key wiped (faction_kills): %v", mob.Character.MiscData)
	}
	if mob.Character.MiscData["conversation_line_idx"] != 2 {
		t.Errorf("non-prefixed key wiped (conversation_line_idx): %v", mob.Character.MiscData)
	}
	if mob.Character.MiscData["some_other_key"] != "value" {
		t.Errorf("non-prefixed key wiped (some_other_key): %v", mob.Character.MiscData)
	}
}

func TestClearPlanState_NilMob_NoOp(t *testing.T) {
	ClearPlanState(nil) // must not panic
}

func TestClearPlanState_NilMiscData_NoOp(t *testing.T) {
	mob := &mobs.Mob{} // MiscData is nil
	ClearPlanState(mob) // must not panic
}

func TestClearPlanState_EmptyMiscData_NoOp(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.MiscData = map[string]any{}
	ClearPlanState(mob)
	if len(mob.Character.MiscData) != 0 {
		t.Errorf("len=%d, want 0", len(mob.Character.MiscData))
	}
}
```

- [ ] **Step 2.3: Run tests + build**

Run: `go test ./internal/planners/ -v`
Expected: PASS for all 5 ClearPlanState tests + the 4 framework tests from Task 1.

- [ ] **Step 2.4: Commit**

```bash
git add internal/planners/state.go internal/planners/state_test.go
git commit -m "feat(planners): ClearPlanState + plan-key prefix (4.4)" -m "Walks mob.Character.MiscData and deletes any keys with the
'plan:' prefix. Nil-safe. Used by goals.Recompute via the
SetPlanStateClear callback wired in Task 3 + main.go (Task 4).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3 — Goals-side callback registration

Mirrors chunks 4.2 (SetWeightsLookup) and 4.3 (SetArchetypeDefaultsLookup) callback patterns. `Recompute` invokes the registered callback on every goal switch.

**Files:**
- Modify: `internal/goals/lookup.go` (extend with `PlanStateClearFn` + `SetPlanStateClear` + `planStateClear` var + internal invocation helper)
- Modify: `internal/goals/store.go` (`Recompute` invokes `planStateClear` after the existing switch-log emission, under panic recovery)
- Modify: `internal/goals/store_test.go` (callback-invocation tests)

- [ ] **Step 3.1: Extend lookup.go**

Append to `internal/goals/lookup.go`:

```go
// PlanStateClearFn wipes any planner-owned intermediate state on a mob.
// Registered once at boot from main.go; invoked by Recompute on every
// goal switch. nil = no cleanup (safe default for tests).
//
// Chunk 4.4.
type PlanStateClearFn func(mob *mobs.Mob)

var planStateClear PlanStateClearFn // guarded by lookupMu

// SetPlanStateClear registers the plan-state cleanup callback. Called
// once at boot from main.go to bridge goals → planners without an
// import cycle. Pass nil to unregister (tests use this for isolation).
//
// Chunk 4.4.
func SetPlanStateClear(fn PlanStateClearFn) {
	lookupMu.Lock()
	planStateClear = fn
	lookupMu.Unlock()
}

// invokePlanStateClear calls the registered callback under panic
// recovery — mirrors invokeContextScore (4.2). A panic in the callback
// must not compromise the goal switch itself.
func invokePlanStateClear(mob *mobs.Mob) {
	lookupMu.RLock()
	fn := planStateClear
	lookupMu.RUnlock()
	if fn == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			mudlog.Warn("goals.planStateClear panic",
				"mob_id", mob.MobId,
				"panic", fmt.Sprintf("%v", r))
		}
	}()
	fn(mob)
}
```

⚠️ The `mudlog` + `fmt` imports — check whether `lookup.go` already imports them. If not, add them.

- [ ] **Step 3.2: Wire into Recompute**

In `internal/goals/store.go`, find `Recompute` (added in chunk 4.2). After the existing `mudlog.Debug("goals.switch", ...)` line at the end of the switched path, insert:

```go
	// Chunk 4.4: invoke registered plan-state cleanup so the new goal's
	// planner starts with fresh MiscData. Best-effort; nil callback is
	// fine (tests, unboot).
	invokePlanStateClear(mob)
```

(The call goes inside the `if !switched { return }` post-check — i.e., only when an actual switch happened.)

- [ ] **Step 3.3: Write integration tests**

Append to `internal/goals/store_test.go`:

```go
func TestRecompute_SwitchInvokesPlanStateClear(t *testing.T) {
	ClearCache()
	called := false
	SetPlanStateClear(func(mob *mobs.Mob) {
		called = true
	})
	defer SetPlanStateClear(nil)

	mobId := 99601
	name := "switch_invokes_clear"
	g := &Goal{Type: "wealth", Priority: 50}
	if _, err := Add(mobId, name, g); err != nil {
		t.Fatalf("Add: %v", err)
	}
	// Add's eager Recompute will fire a switch (nil → new goal) and
	// invoke the callback.
	if !called {
		t.Errorf("PlanStateClear callback not invoked on switch")
	}
}

func TestRecompute_NoSwitch_DoesNotInvokeCallback(t *testing.T) {
	ClearCache()
	mobId := 99602
	name := "noswitch_callback"
	g := &Goal{Type: "wealth", Priority: 50}
	if _, err := Add(mobId, name, g); err != nil {
		t.Fatalf("Add: %v", err)
	}
	// Switch happened during Add. Now register the callback and force
	// a no-switch Recompute (same goal still wins).
	calls := 0
	SetPlanStateClear(func(mob *mobs.Mob) {
		calls++
	})
	defer SetPlanStateClear(nil)
	Recompute(mobId, name, &mobs.Mob{}, 1000)
	if calls != 0 {
		t.Errorf("callback fired on no-switch tick: %d call(s)", calls)
	}
}

func TestRecompute_NoCallbackRegistered_NoError(t *testing.T) {
	ClearCache()
	SetPlanStateClear(nil)
	mobId := 99603
	name := "no_callback"
	g := &Goal{Type: "wealth", Priority: 50}
	// Must not panic / error even without a callback.
	if _, err := Add(mobId, name, g); err != nil {
		t.Fatalf("Add: %v", err)
	}
}
```

- [ ] **Step 3.4: Run tests + build**

Run: `go test ./internal/goals/ -run "TestRecompute_SwitchInvokesPlanStateClear|TestRecompute_NoSwitch_DoesNotInvokeCallback|TestRecompute_NoCallbackRegistered_NoError" -v`
Expected: PASS for all 3.

Run: `go test ./internal/goals/...`
Expected: PASS (existing tests still pass).

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 3.5: Commit**

```bash
git add internal/goals/lookup.go internal/goals/store.go internal/goals/store_test.go
git commit -m "feat(goals): PlanStateClear callback in Recompute (4.4)" -m "Adds PlanStateClearFn type + SetPlanStateClear registration + internal
invokePlanStateClear helper (panic-recovered, mirrors invokeContextScore).
Recompute invokes the callback on every goal switch, after persistence
+ log emission. Bridges goals → planners without an import cycle —
main.go (Task 4) registers planners.ClearPlanState as the adapter.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4 — main.go boot wiring

Registers `planners.ClearPlanState` as the goals-side callback + blank-imports the planners package so per-planner init()s fire. At Task 4 ship, the planners package has zero registered planners (Tasks 8-20 add them), but the boot path works end-to-end.

**Files:**
- Modify: `main.go`

- [ ] **Step 4.1: Add the boot wiring**

In `main.go`, after the existing `goals.SetArchetypeDefaultsLookup(...)` block from chunk 4.3, add:

```go
	// Wire the goals → planners plan-state cleanup callback. Mirrors the
	// 4.2 SetWeightsLookup + 4.3 SetArchetypeDefaultsLookup patterns —
	// avoids the goals → planners import cycle. Chunk 4.4.
	goals.SetPlanStateClear(planners.ClearPlanState)
```

Also add to the imports block (near the existing `goals` blank import for the catalog):

```go
	_ "github.com/GoMudEngine/GoMud/internal/planners" // chunk 4.4 — fire planner init() registrations
```

⚠️ The `planners` package needs both a regular import (for `planners.ClearPlanState`) AND blank import? Actually no — a single regular import is enough; the regular import also fires the inits. Use a regular import:

```go
	"github.com/GoMudEngine/GoMud/internal/planners"
```

Remove the blank import alias if you added one in the prior step. One regular import does both jobs.

- [ ] **Step 4.2: Build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 4.3: Sanity check — boot doesn't panic**

Run: `timeout 30 go run . 2>&1 | grep -iE "panic|started|fatal" | head -10`
Expected: `MainWorker state="Started"` appears; no panic.

Stop with Ctrl+C.

- [ ] **Step 4.4: Commit**

```bash
git add main.go
git commit -m "feat(boot): register goals.SetPlanStateClear adapter (4.4)" -m "Bridges goals → planners via the SetPlanStateClear callback from Task 3.
On every goal switch, goals.Recompute now calls planners.ClearPlanState
which wipes any plan:-prefixed MiscData keys so the new goal's planner
starts fresh.

Regular import of internal/planners pulls per-planner init()
registrations as they're added in Tasks 8-20.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5 — `try_goal_planner` btree action

The dispatch entry point. Looks up the mob's current goal, finds the registered planner, invokes it under panic recovery, executes the returned command, propagates the status to the btree.

**Files:**
- Create: `internal/behaviortree/actions_goal.go`
- Create: `internal/behaviortree/actions_goal_test.go`

- [ ] **Step 5.1: Determine the action-registration mechanism**

Use codegraph to find how existing actions register themselves:

Run: `codegraph_search RegisterAction` (or whatever the registration helper is called — e.g. `registerActionHandler`, `actionsByName`, etc.)

Look at how `try_combat`, `try_forage`, or `try_patrol` register — `actions_combat.go`, `actions_forager.go`, etc. **Confirm the registration API** before writing the new file. The pattern likely involves a package-level `init()` adding to a map.

- [ ] **Step 5.2: Write the failing tests**

Create `internal/behaviortree/actions_goal_test.go`:

```go
package behaviortree

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/planners"
)

func TestActGoalPlanner_NoCurrentGoal_Failure(t *testing.T) {
	goals.ClearCache()
	mob := &mobs.Mob{MobId: mobs.MobId(99701)}
	mob.Character.Name = "no_goal_test"
	// No goals added — CurrentGoalOf returns nil.
	status := actGoalPlanner(mob, nil)
	if status != StatusFailure {
		t.Errorf("status=%v, want StatusFailure", status)
	}
}

func TestActGoalPlanner_NoRegisteredPlanner_Failure(t *testing.T) {
	goals.ClearCache()
	mob := &mobs.Mob{MobId: mobs.MobId(99702)}
	mob.Character.Name = "no_planner_test"
	if _, err := goals.Add(int(mob.MobId), "no_planner_test", &goals.Goal{Type: "unregistered-type", Priority: 50}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	status := actGoalPlanner(mob, nil)
	if status != StatusFailure {
		t.Errorf("status=%v, want StatusFailure (no planner registered)", status)
	}
}

func TestActGoalPlanner_PlannerSuccess_PropagatesStatus(t *testing.T) {
	goals.ClearCache()
	planners.RegisterPlanner("test-success", func(*mobs.Mob, *goals.Goal) planners.PlanResult {
		return planners.PlanResult{Command: "", Status: planners.StatusSuccess}
	})
	defer planners.RegisterPlanner("test-success", nil)

	mob := &mobs.Mob{MobId: mobs.MobId(99703)}
	mob.Character.Name = "planner_success_test"
	if _, err := goals.Add(int(mob.MobId), "planner_success_test", &goals.Goal{Type: "test-success", Priority: 50}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	status := actGoalPlanner(mob, nil)
	if status != StatusSuccess {
		t.Errorf("status=%v, want StatusSuccess", status)
	}
}

func TestActGoalPlanner_PlannerPanic_RecoveredFailure(t *testing.T) {
	goals.ClearCache()
	planners.RegisterPlanner("test-panicky", func(*mobs.Mob, *goals.Goal) planners.PlanResult {
		panic("planner boom")
	})
	defer planners.RegisterPlanner("test-panicky", nil)

	mob := &mobs.Mob{MobId: mobs.MobId(99704)}
	mob.Character.Name = "planner_panic_test"
	if _, err := goals.Add(int(mob.MobId), "planner_panic_test", &goals.Goal{Type: "test-panicky", Priority: 50}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	// Must not panic.
	status := actGoalPlanner(mob, nil)
	if status != StatusFailure {
		t.Errorf("status=%v, want StatusFailure (panic should map to Failure)", status)
	}
}
```

⚠️ The test calls `actGoalPlanner(mob, nil)` directly — `nil` is the args map (the btree action signature typically has `(mob, args)`). Verify the exact signature pattern by looking at any existing action like `actCombat` or `actForage`. Adapt the test signature if the real one differs.

⚠️ `goals.Add` from the test fires `Recompute` eagerly (chunk 4.2 behavior). That selects the new goal as current — exactly what we want for the test. No further setup needed.

- [ ] **Step 5.3: Run tests to verify they fail**

Run: `go test ./internal/behaviortree/ -run "TestActGoalPlanner_" -v`
Expected: FAIL — `actGoalPlanner` undefined.

- [ ] **Step 5.4: Create the action file**

Create `internal/behaviortree/actions_goal.go`:

```go
package behaviortree

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/planners"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func init() {
	// Register try_goal_planner with the action registry.
	// TODO-ADAPT: replace this comment + call with the actual registration
	// API from this package (verify via codegraph — see e.g. actions_combat.go
	// or actions_forager.go for the pattern). The signature is something like
	//   registerAction("try_goal_planner", actGoalPlanner)
	// or possibly a map literal append. Match the existing pattern exactly.
	registerAction("try_goal_planner", actGoalPlanner)
}

// actGoalPlanner is the chunk-4.4 btree action that dispatches per the
// mob's current goal (selected by chunk 4.2 from chunk 4.3's catalog)
// to the registered planner (chunk 4.4's per-type planner files).
//
// Returns:
//   - StatusFailure if no current goal OR no registered planner for the
//     goal's type OR the planner returns Failure.
//   - StatusSuccess if the planner returns Success.
//   - StatusRunning if the planner returns Running (multi-tick plan in
//     progress).
//
// If the planner returns a non-empty Command, executes it via
// mob.Command(cmd) — same path as schedule idle commands.
//
// Panics inside the planner are recovered, logged, and mapped to
// StatusFailure (the btree falls through to its next action).
func actGoalPlanner(mob *mobs.Mob, args map[string]any) Status {
	if mob == nil {
		return StatusFailure
	}
	templateId := int(mob.MobId)
	name := util.ConvertForFilename(mob.Character.Name)
	currentGoal := goals.CurrentGoalOf(templateId, name)
	if currentGoal == nil {
		return StatusFailure
	}
	fn := planners.LookupPlanner(currentGoal.Type)
	if fn == nil {
		return StatusFailure
	}

	result := invokePlannerSafely(fn, mob, currentGoal)

	if result.Command != "" {
		mob.Command(result.Command)
	}
	return translatePlannerStatus(result.Status)
}

// invokePlannerSafely wraps the planner call in panic recovery. A panic
// logs a warn line with goal type + mob id and returns Failure.
// Mirrors invokeContextScore (4.2) and invokeDedupKey (4.3).
func invokePlannerSafely(fn planners.PlanFn, mob *mobs.Mob, goal *goals.Goal) (result planners.PlanResult) {
	defer func() {
		if r := recover(); r != nil {
			mudlog.Warn("planners.plan panic",
				"goal_type", goal.Type,
				"goal_id", goal.Id,
				"mob_id", mob.MobId,
				"panic", fmt.Sprintf("%v", r))
			result = planners.PlanResult{Status: planners.StatusFailure}
		}
	}()
	return fn(mob, goal)
}

// translatePlannerStatus maps planners.BTreeStatus -> behaviortree.Status.
// Two separate enums by design (planners can't import behaviortree
// without forming a cycle). This one-line switch is the only place the
// translation happens.
func translatePlannerStatus(ps planners.BTreeStatus) Status {
	switch ps {
	case planners.StatusSuccess:
		return StatusSuccess
	case planners.StatusRunning:
		return StatusRunning
	}
	return StatusFailure
}
```

⚠️ Several names here are placeholders that must be verified against actual behaviortree package code:

1. `registerAction("try_goal_planner", actGoalPlanner)` — the exact registration helper name. Likely `registerActionHandler` or similar. Find via codegraph (`codegraph_search registerAction`) before commit.
2. `Status`, `StatusFailure`, `StatusSuccess`, `StatusRunning` (behaviortree package's own enum) — these are the btree-side status type. Verify they exist with these names. If they're called `NodeStatus` / `Failure` / `Success` / `Running` (no `Status` prefix), adapt the file accordingly.
3. The action signature `func(mob *mobs.Mob, args map[string]any) Status` — verify by looking at existing actions like `actCombat`. If the real signature is different (e.g., `(ctx *ExecutionContext) Status`), adapt.

**Implementer MUST verify these via codegraph and adapt the file before commit.** Use `codegraph_node actCombat` or `codegraph_node actForageStep` for the actual reference shape.

- [ ] **Step 5.5: Run tests + build**

Run: `go test ./internal/behaviortree/ -run "TestActGoalPlanner_" -v`
Expected: PASS for all 4.

Run: `go test ./internal/behaviortree/...`
Expected: PASS (existing 4.2/4.3 behaviortree tests still pass).

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 5.6: Commit**

```bash
git add internal/behaviortree/actions_goal.go internal/behaviortree/actions_goal_test.go
git commit -m "feat(behaviortree): try_goal_planner action — dispatch to 4.4 planners" -m "actGoalPlanner reads CurrentGoalOf, looks up the registered planner
for the goal type, invokes under panic recovery, executes returned
command via mob.Command, propagates BTreeStatus.

No current goal OR no registered planner OR planner returns Failure
or panics → StatusFailure (btree falls through). Otherwise propagates
Running / Success as returned.

Two enums (planners.BTreeStatus + behaviortree.Status) bridged by a
one-line translatePlannerStatus switch — keeps the planner package
free of behaviortree imports.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6 — Supporting helpers (`helpers.go`)

Shared subsystem-adapter functions used across multiple planners. Per-helper unit test where the adapter shape is verifiable in isolation.

**Files:**
- Create: `internal/planners/helpers.go`
- Create: `internal/planners/helpers_test.go`

This task is "skeleton + TODO-ADAPT" pattern — many helpers wrap subsystems whose exact API the implementer must verify via codegraph at implementation time. Write the file with explicit TODO-ADAPT shims; per-planner tasks 8-20 will exercise the helpers, exposing any shim that returns wrong-shape data.

- [ ] **Step 6.1: Create helpers.go with all 10 functions (some real, some TODO-ADAPT)**

```go
package planners

import (
	"sort"

	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ─── Shop helpers ─────────────────────────────────────────────────

// findShopInZoneSelling returns the room id of a shop in mob's zone
// that stocks an item matching item_tag or item_id. ok=false if none.
//
// TODO-ADAPT: the shops package's accessor surface needs codegraph
// verification. Likely candidates:
//   - shops.InZone(zoneName) []Shop
//   - shops.GetShopsByZone(zoneName) []*ShopInstance
// Once known, walk results, inspect each shop's stock list for a
// match on ComponentTag (string) or ItemId (int), and return the
// shop's RoomId.
func findShopInZoneSelling(mob *mobs.Mob, tag string, itemId int) (roomId int, ok bool) {
	// TODO-ADAPT: implement against shops package once API verified.
	// Stub: never finds a shop. Planners using this will fall through to
	// their no-shop branch — that's the correct behavior for unverified
	// API.
	return 0, false
}

// findShopInZoneBuying returns a shop in mob's zone with gold reserves
// (i.e., one that can buy items from the mob). ok=false if none.
//
// TODO-ADAPT: same shops API as above, plus filter on shop's Gold > 0.
// Most shops buy most things in DOGMud; the constraint that matters
// is gold availability.
func findShopInZoneBuying(mob *mobs.Mob) (roomId int, ok bool) {
	// TODO-ADAPT: implement against shops package.
	return 0, false
}

// ─── Faction helpers ─────────────────────────────────────────────

// findFactionMemberInZone returns the first mob instance in the same
// zone as mob that belongs to factionId. If mustBeInCombat is true,
// only members with Aggro != nil match.
func findFactionMemberInZone(mob *mobs.Mob, factionId string, mustBeInCombat bool) (target *mobs.Mob, ok bool) {
	zone := mob.Character.Zone
	for _, instId := range mobs.GetAllMobInstanceIds() {
		inst := mobs.GetInstance(instId)
		if inst == nil || inst.Character.Zone != zone {
			continue
		}
		if inst.InstanceId == mob.InstanceId {
			continue
		}
		if mustBeInCombat && inst.Character.Aggro == nil {
			continue
		}
		for _, fid := range factions.FactionsForMob(inst) {
			if fid == factionId {
				return inst, true
			}
		}
	}
	return nil, false
}

// findHostileInZone returns the first auto-aggro mob in mob's zone
// (excluding self).
func findHostileInZone(mob *mobs.Mob) (target *mobs.Mob, ok bool) {
	zone := mob.Character.Zone
	for _, instId := range mobs.GetAllMobInstanceIds() {
		inst := mobs.GetInstance(instId)
		if inst == nil || inst.Character.Zone != zone {
			continue
		}
		if inst.InstanceId == mob.InstanceId {
			continue
		}
		if inst.AutoAggro {
			return inst, true
		}
	}
	return nil, false
}

// ─── Crafting station helpers ─────────────────────────────────────

// findCraftingStationInZone returns the room id of a room in mob's
// zone tagged with the named station (e.g., "forge", "loom"). ok=false
// if none.
//
// TODO-ADAPT: rooms package's API for station tags needs codegraph
// verification. Likely candidates:
//   - rooms.LoadRoom(id).Station string field
//   - rooms.RoomsByZone(zone) walk + inspect each room's Station tag
func findCraftingStationInZone(mob *mobs.Mob, stationName string) (roomId int, ok bool) {
	// TODO-ADAPT.
	return 0, false
}

// ─── Zone graph helpers ───────────────────────────────────────────

var zoneAdjacencyCache map[string][]string

// zoneAdjacentTo returns the list of zones adjacent to zoneA via inter-
// zone room exits. Lazy-computed on first call per zone; cached
// (room graph is static after boot).
//
// TODO-ADAPT: rooms.GetZoneConfig + room.Exits iteration. The exact
// shape of Exits (map[string]Exit, where Exit has RoomId int) needs
// codegraph verification. Once known, for each room in zoneA, look at
// each exit's destination room, grab its Zone, deduplicate, return.
func zoneAdjacentTo(zoneA string) []string {
	if zoneAdjacencyCache == nil {
		zoneAdjacencyCache = map[string][]string{}
	}
	if cached, ok := zoneAdjacencyCache[zoneA]; ok {
		return cached
	}
	// TODO-ADAPT: real BFS. Stub returns empty list.
	zoneAdjacencyCache[zoneA] = []string{}
	return zoneAdjacencyCache[zoneA]
}

// ─── Inventory helpers ────────────────────────────────────────────

// pickGiftItemFromInventory returns the highest-value non-quest item
// in mob's backpack suitable for gifting. nil if none.
//
// TODO-ADAPT: ItemSpec.Value (int) for value scoring + ItemSpec.QuestToken
// (string, "" = not a quest item) for the filter — verify field names
// via codegraph.
func pickGiftItemFromInventory(mob *mobs.Mob) *items.Item {
	if mob == nil || len(mob.Character.Items) == 0 {
		return nil
	}
	var best *items.Item
	bestValue := 0
	for i := range mob.Character.Items {
		it := &mob.Character.Items[i]
		spec := it.GetSpec()
		// TODO-ADAPT: replace the field accesses below with verified names.
		// Likely spec.QuestToken / spec.Value but verify.
		if spec.QuestToken != "" {
			continue
		}
		if spec.Value > bestValue {
			best = it
			bestValue = spec.Value
		}
	}
	return best
}

// ─── Exit + emote helpers ────────────────────────────────────────

// pickRandomExit returns a random exit direction name from mob's
// current room. Empty string if no exits.
//
// TODO-ADAPT: rooms.LoadRoom(mob.Character.RoomId).Exits is the map;
// pick a random key. Use util.Rand for the random pick.
func pickRandomExit(mob *mobs.Mob) string {
	if mob == nil {
		return ""
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return ""
	}
	// TODO-ADAPT: iterate room.Exits keys; pick a random one.
	// Stub returns empty (planners using this fall through gracefully).
	return ""
}

// pickSocialEmote returns a random social emote command string.
// Used by befriend's interaction rotation and mastery-skill's social
// branch.
func pickSocialEmote() string {
	emotes := []string{
		"emote nods",
		"emote bows",
		"emote smiles warmly",
		"emote waves",
		"emote grins",
	}
	return emotes[util.Rand(len(emotes))]
}

// ─── Recipe helpers ────────────────────────────────────────────

// pickKnownRecipeForSkill returns the lowest-skill_minimum recipe id
// the mob knows that trains the given skill. Empty string if none.
//
// TODO-ADAPT: walk mob.Character.KnownRecipes (map[string]int),
// filter by crafting.GetRecipe(rid).Skill == skillName, sort by
// SkillMinimum ascending, return first.
func pickKnownRecipeForSkill(mob *mobs.Mob, skillName string) string {
	if mob == nil || len(mob.Character.KnownRecipes) == 0 {
		return ""
	}
	type candidate struct {
		id  string
		min int
	}
	var cands []candidate
	for rid := range mob.Character.KnownRecipes {
		// TODO-ADAPT: lookup recipe via crafting.GetRecipe(rid); skip if
		// nil or recipe.Skill != skillName; collect (rid, recipe.SkillMinimum).
		_ = rid
	}
	if len(cands) == 0 {
		return ""
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].min < cands[j].min })
	return cands[0].id
}
```

⚠️ **Critical for implementer:** every TODO-ADAPT comment marks a shim that returns the conservative-failure default. Per-planner tasks (8-20) will exercise these and surface any wrong assumptions. The implementer SHOULD wire each helper to the real subsystem API during this task (it's the bulk of the work) — use codegraph (`codegraph_search`, `codegraph_node`) to discover the actual APIs:

- `codegraph_node Shop` and `codegraph_search shops` for shop accessors
- `codegraph_search station` and `codegraph_node Room` for station-room lookup
- `codegraph_node Exit` for room exit iteration
- `codegraph_node ItemSpec` to verify QuestToken + Value fields
- `codegraph_node Recipe` to verify Skill + SkillMinimum fields

Replace each TODO-ADAPT with real code. **If a subsystem doesn't expose the needed API cleanly, document the gap in the function's comment and leave the stub** — the per-planner tasks will then fall through to a no-op branch, which is the correct safe behavior.

- [ ] **Step 6.2: Write helper tests**

Create `internal/planners/helpers_test.go`:

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestPickSocialEmote_ReturnsNonEmpty(t *testing.T) {
	got := pickSocialEmote()
	if got == "" {
		t.Errorf("pickSocialEmote returned empty string")
	}
}

func TestZoneAdjacentTo_Cached(t *testing.T) {
	// First call computes; second call returns cached.
	zoneAdjacencyCache = nil
	first := zoneAdjacentTo("stillwater")
	if zoneAdjacencyCache == nil {
		t.Fatalf("cache not populated after first call")
	}
	second := zoneAdjacentTo("stillwater")
	// Pointer equality on the slice header isn't guaranteed (Go map
	// returns the same backing array but copy is fine) — semantic
	// equality is what matters.
	if len(first) != len(second) {
		t.Errorf("first=%v second=%v differ", first, second)
	}
}

func TestPickGiftItemFromInventory_EmptyMob_ReturnsNil(t *testing.T) {
	mob := &mobs.Mob{}
	if got := pickGiftItemFromInventory(mob); got != nil {
		t.Errorf("got non-nil for empty inventory")
	}
}

// Tests for findShopInZoneSelling, findShopInZoneBuying,
// findFactionMemberInZone, findHostileInZone,
// findCraftingStationInZone require live mob/zone/shop state — defer
// to integration coverage in per-planner tasks (8-20) and Task 23 smoke.
```

- [ ] **Step 6.3: Run tests + build**

Run: `go test ./internal/planners/ -v`
Expected: PASS for all tests added so far (Tasks 1, 2, 6).

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 6.4: Commit**

```bash
git add internal/planners/helpers.go internal/planners/helpers_test.go
git commit -m "feat(planners): supporting helpers — shops/factions/zones/inventory (4.4)" -m "10 helper functions used across multiple per-type planners. Several
are TODO-ADAPT shims pending subsystem-API verification at planner-
implementation time (shops, station-room lookup, room exits, recipe
filter). Conservative-failure defaults so planners using them fall
through to no-op branches if shims aren't fully wired.

findFactionMemberInZone + findHostileInZone are fully wired (factions
+ mobs APIs verified during 4.3 catalog work).

pickSocialEmote is a hand-authored emote rotation for befriend +
mastery-skill social branches.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 7 — Skill training table

Static data file mapping skill names to training-context kinds. Consumed by `mastery-skill` planner.

**Files:**
- Create: `internal/planners/skill_training_table.go`
- Create: `internal/planners/skill_training_table_test.go`

- [ ] **Step 7.1: Create the table**

```go
package planners

// SkillTrainingContext describes the kind of opportunity a mob needs
// to train a particular skill. Used by mastery-skill planner to choose
// what to do.
type SkillTrainingContext string

const (
	TrainingCombat       SkillTrainingContext = "combat"       // weapon/unarmed/ranged/spellcasting
	TrainingCrafting     SkillTrainingContext = "crafting"     // smithing/alchemy/etc.
	TrainingForaging     SkillTrainingContext = "foraging"
	TrainingSocial       SkillTrainingContext = "social"       // rhetoric
	TrainingSkullduggery SkillTrainingContext = "skullduggery"
	TrainingUnknown      SkillTrainingContext = "unknown"
)

// skillTrainingTable maps a skill name (matching skills.SkillTag string
// values) to its training context. Used by mastery-skill planner.
//
// Verified against the existing skills package via codegraph at 4.4
// authoring time. If a new skill is added later, append a row here.
var skillTrainingTable = map[string]SkillTrainingContext{
	"weapon-combat":  TrainingCombat,
	"unarmed-combat": TrainingCombat,
	"ranged-combat":  TrainingCombat,
	"spellcasting":   TrainingCombat,
	"rhetoric":       TrainingSocial,
	"skullduggery":   TrainingSkullduggery,
	"smithing":       TrainingCrafting,
	"tanning":        TrainingCrafting,
	"tailoring":      TrainingCrafting,
	"alchemy":        TrainingCrafting,
	"fletching":      TrainingCrafting,
	"cooking":        TrainingCrafting,
	"salvage":        TrainingCrafting,
	"foraging":       TrainingForaging,
}

// SkillTrainingContextOf returns the training context for a skill name,
// or TrainingUnknown if the skill isn't in the table.
func SkillTrainingContextOf(skillName string) SkillTrainingContext {
	if ctx, ok := skillTrainingTable[skillName]; ok {
		return ctx
	}
	return TrainingUnknown
}
```

⚠️ Skill name strings must match the actual `skills.SkillTag` string values. Verify by running `codegraph_search SkillTag` or grepping `_datafiles/skills/` for the canonical names. Adjust the table if any are off.

- [ ] **Step 7.2: Write tests**

Create `internal/planners/skill_training_table_test.go`:

```go
package planners

import "testing"

func TestSkillTrainingContextOf_CombatSkills(t *testing.T) {
	for _, name := range []string{"weapon-combat", "unarmed-combat", "ranged-combat", "spellcasting"} {
		if got := SkillTrainingContextOf(name); got != TrainingCombat {
			t.Errorf("%s: got %v, want TrainingCombat", name, got)
		}
	}
}

func TestSkillTrainingContextOf_CraftingSkills(t *testing.T) {
	for _, name := range []string{"smithing", "alchemy", "cooking", "salvage"} {
		if got := SkillTrainingContextOf(name); got != TrainingCrafting {
			t.Errorf("%s: got %v, want TrainingCrafting", name, got)
		}
	}
}

func TestSkillTrainingContextOf_Foraging(t *testing.T) {
	if got := SkillTrainingContextOf("foraging"); got != TrainingForaging {
		t.Errorf("got %v, want TrainingForaging", got)
	}
}

func TestSkillTrainingContextOf_Unknown(t *testing.T) {
	if got := SkillTrainingContextOf("nonexistent-skill"); got != TrainingUnknown {
		t.Errorf("got %v, want TrainingUnknown", got)
	}
}
```

- [ ] **Step 7.3: Run tests + build**

Run: `go test ./internal/planners/ -v`
Expected: PASS.

- [ ] **Step 7.4: Commit**

```bash
git add internal/planners/skill_training_table.go internal/planners/skill_training_table_test.go
git commit -m "feat(planners): skill training context table (4.4)" -m "Static map from skill names (matching skills.SkillTag values) to
TrainingContext kinds (Combat / Crafting / Foraging / Social /
Skullduggery / Unknown). Used by mastery-skill planner to dispatch
to the appropriate training action.

If a new skill is added to the skills package later, append a row
to skillTrainingTable.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Per-planner tasks (8–20): conventions

All 13 per-planner tasks share these conventions:

- Each creates `internal/planners/<type>.go` (snake_case filename matching goal type with hyphens → underscores).
- Each file's `init()` calls `RegisterPlanner("<goal-type-name>", <plannerFn>)`.
- Each task adds a `<type>_test.go` alongside with: registration check + per-branch unit tests.
- Test mob fixtures use the pattern from 4.3 catalog tests: `mob := &mobs.Mob{}; mob.Character.Health = 50; mob.Character.HealthMax.Value = 100; ...`. The implementer should verify the exact field path for HealthMax via codegraph if unsure (4.3 catalog Tasks 7+ found it's `HealthMax.Value` because `HealthMax` is a `stats.StatInfo` struct).
- Task 8 (survival, the first planner task) additionally adds `goalParamIntOr` to `helpers.go` for shared int-param reads from `Goal.Params`. Subsequent tasks reuse it.

---

## Task 8 — `survival` planner + `goalParamIntOr` helper

**Files:**
- Modify: `internal/planners/helpers.go` (add `goalParamIntOr` helper)
- Create: `internal/planners/survival.go`
- Create: `internal/planners/survival_test.go`

- [ ] **Step 8.1: Add `goalParamIntOr` to helpers.go**

Append to `internal/planners/helpers.go`:

```go
// goalParamIntOr reads an int param from goal.Params with a fallback
// default. Tolerates int64 (YAML's default int type) and int.
// Used by every per-type planner that reads numeric Params.
func goalParamIntOr(g *goals.Goal, key string, def int) int {
	if g == nil || g.Params == nil {
		return def
	}
	raw, ok := g.Params[key]
	if !ok {
		return def
	}
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	}
	return def
}

// goalParamStringOr reads a string param from goal.Params with a fallback default.
func goalParamStringOr(g *goals.Goal, key string, def string) string {
	if g == nil || g.Params == nil {
		return def
	}
	if s, ok := g.Params[key].(string); ok {
		return s
	}
	return def
}
```

Add `goals` import to `helpers.go` (likely already imported via existing helpers — verify).

- [ ] **Step 8.2: Create survival.go**

```go
package planners

import (
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const (
	survivalDefaultSafePct = 60
	survivalDefaultFleePct = 25
)

func init() {
	RegisterPlanner("survival", survivalPlanner)
}

// survivalPlanner: flee combat, drink heal potion, or rest until HP is
// at safe threshold. Reactive — no MiscData state.
func survivalPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil || mob.Character.HealthMax.Value <= 0 {
		return PlanResult{Status: StatusFailure}
	}
	safePct := goalParamIntOr(goal, "safe_threshold_pct", survivalDefaultSafePct)
	fleePct := goalParamIntOr(goal, "flee_threshold_pct", survivalDefaultFleePct)
	hpPct := (mob.Character.Health * 100) / mob.Character.HealthMax.Value

	// Recovered → predicate satisfies next tick.
	if hpPct >= safePct && mob.Character.Aggro == nil {
		return PlanResult{Status: StatusSuccess}
	}

	// In combat → flee.
	if mob.Character.Aggro != nil {
		exit := pickRandomExit(mob)
		if exit == "" {
			return PlanResult{Status: StatusFailure}
		}
		return PlanResult{Command: "flee " + exit, Status: StatusRunning}
	}

	// Low HP, not in combat → drink healing potion if available, else rest.
	if hpPct < fleePct {
		if potion := pickHealingPotion(mob); potion != "" {
			return PlanResult{Command: "drink " + potion, Status: StatusRunning}
		}
	}
	return PlanResult{Command: "rest", Status: StatusRunning}
}

// pickHealingPotion returns the name of a healing potion in the mob's
// inventory, or empty string if none. Survival's local helper.
//
// TODO-ADAPT: heuristic candidates for "is this item a healing potion":
//   - item.GetSpec().Type == "potion" AND name contains "heal" / "regen"
//   - explicit potion-effect inspection via items.Item.GetSpec().Effects
// Verify via codegraph (`codegraph_node ItemSpec` for the effect field
// shape). Conservative stub: return empty string (planner falls through
// to rest).
func pickHealingPotion(mob *mobs.Mob) string {
	if mob == nil || len(mob.Character.Items) == 0 {
		return ""
	}
	// TODO-ADAPT
	return ""
}
```

- [ ] **Step 8.3: Write tests**

Create `internal/planners/survival_test.go`:

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestSurvival_Registered(t *testing.T) {
	if LookupPlanner("survival") == nil {
		t.Fatalf("survival not registered")
	}
}

func TestSurvival_HighHP_NotInCombat_Success(t *testing.T) {
	fn := LookupPlanner("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 100
	mob.Character.HealthMax.Value = 100
	res := fn(mob, &goals.Goal{Type: "survival"})
	if res.Status != StatusSuccess {
		t.Errorf("status=%v, want StatusSuccess", res.Status)
	}
}

func TestSurvival_InCombat_FleesRandomExit(t *testing.T) {
	fn := LookupPlanner("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 30
	mob.Character.HealthMax.Value = 100
	mob.Character.Aggro = &someAggro{}
	res := fn(mob, &goals.Goal{Type: "survival"})
	// pickRandomExit returns "" (TODO-ADAPT stub from Task 6) → StatusFailure.
	// When the helper is wired, this test should be updated to expect
	// StatusRunning + a "flee <exit>" command.
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want StatusFailure (pickRandomExit stubbed)", res.Status)
	}
	_ = res.Command
}

// someAggro is a minimal Aggro fixture for tests.
// Note: characters.Aggro is a struct, mob.Character.Aggro is *Aggro.
// Tests just need a non-nil pointer.
type someAggro = mobs.Mob // placeholder — adapt to real Aggro type

func TestSurvival_LowHP_NotInCombat_RestsByDefault(t *testing.T) {
	fn := LookupPlanner("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 10
	mob.Character.HealthMax.Value = 100
	// No potion (Items empty + stub returns "")
	res := fn(mob, &goals.Goal{Type: "survival"})
	if res.Command != "rest" {
		t.Errorf("command=%q, want rest", res.Command)
	}
	if res.Status != StatusRunning {
		t.Errorf("status=%v, want StatusRunning", res.Status)
	}
}
```

⚠️ The `someAggro = mobs.Mob` alias is a test-only placeholder. The real `characters.Aggro` type lives at `internal/characters/combat_state_compat.go:36`. Implementer: replace the placeholder with the actual import + `&characters.Aggro{}` construction. (Per 4.3 catalog Task 7 findings: `mob.Character.Aggro` is `*characters.Aggro`.)

- [ ] **Step 8.4: Run tests + build**

Run: `go test ./internal/planners/ -run "TestSurvival_" -v`
Expected: PASS for all 4.

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 8.5: Commit**

```bash
git add internal/planners/helpers.go internal/planners/survival.go internal/planners/survival_test.go
git commit -m "feat(planners): survival planner + goalParamIntOr helper (4.4)" -m "Reactive planner: success when HP >= safe AND not in combat; flee
when in combat; drink healing potion when low HP + potion held; else
rest. No MiscData state.

Adds shared goalParamIntOr + goalParamStringOr helpers to helpers.go
(used by every per-type planner that reads numeric/string Params).

pickHealingPotion is a TODO-ADAPT stub — verify ItemSpec.Type +
effect-shape via codegraph and wire to real potion detection.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 9 — `wealth-gold` planner

**Files:**
- Create: `internal/planners/wealth_gold.go`
- Create: `internal/planners/wealth_gold_test.go`

- [ ] **Step 9.1: Create wealth_gold.go**

```go
package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const wealthGoldShopRoomKey = "plan:wealth-gold:target_shop_room"

func init() {
	RegisterPlanner("wealth-gold", wealthGoldPlanner)
}

// wealthGoldPlanner: sell-loot loop. Goes to a vendor in zone, sells
// inventory, accumulates gold until target.
func wealthGoldPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	target := goalParamIntOr(goal, "target", 0)
	if target <= 0 {
		return PlanResult{Status: StatusFailure}
	}

	// Target met → predicate satisfies next tick.
	if mob.Character.Gold >= target {
		return PlanResult{Status: StatusSuccess}
	}

	hasSellable := mobHasSellableItems(mob)

	// Has sellable + at a vendor → sell all.
	if hasSellable && mobInVendorRoom(mob) {
		return PlanResult{Command: "sell all", Status: StatusRunning}
	}

	// Has sellable + not at vendor → resolve target_shop_room (sticky)
	// and pathto.
	if hasSellable {
		shopRoom := mobMiscIntOr(mob, wealthGoldShopRoomKey, 0)
		if shopRoom == 0 {
			room, ok := findShopInZoneBuying(mob)
			if !ok {
				return PlanResult{Command: "wander", Status: StatusRunning}
			}
			shopRoom = room
			mobSetMisc(mob, wealthGoldShopRoomKey, shopRoom)
		}
		return PlanResult{Command: "pathto " + strconv.Itoa(shopRoom), Status: StatusRunning}
	}

	// No sellable items → wander to find loot.
	return PlanResult{Command: "wander", Status: StatusRunning}
}

// mobHasSellableItems heuristic: any inventory item with non-zero Value
// and no quest-token flag is sellable.
//
// TODO-ADAPT: verify ItemSpec.Value + .QuestToken via codegraph (likely
// matches what pickGiftItemFromInventory uses).
func mobHasSellableItems(mob *mobs.Mob) bool {
	for i := range mob.Character.Items {
		spec := mob.Character.Items[i].GetSpec()
		if spec.QuestToken != "" {
			continue
		}
		if spec.Value > 0 {
			return true
		}
	}
	return false
}

// mobInVendorRoom returns true if mob's current room hosts a vendor.
//
// TODO-ADAPT: walk shops in zone, check if any has RoomId == mob's room.
// (Cheaper: filter on Shop.RoomId == mob.Character.RoomId directly.)
func mobInVendorRoom(mob *mobs.Mob) bool {
	// TODO-ADAPT: implement once shops API is verified.
	return false
}
```

⚠️ Add MiscData helpers (`mobMiscIntOr`, `mobSetMisc`) — they don't exist yet in the planners package. Implementer should add them to `helpers.go` here OR define inline. Reasonable shape:

```go
func mobMiscIntOr(mob *mobs.Mob, key string, def int) int {
    raw := mob.Character.GetMiscData(key)
    if raw == nil { return def }
    switch v := raw.(type) {
    case int: return v
    case int64: return int(v)
    }
    return def
}
func mobSetMisc(mob *mobs.Mob, key string, val any) {
    if mob.Character.MiscData == nil {
        mob.Character.MiscData = map[string]any{}
    }
    mob.Character.SetMiscData(key, val)
}
```

Verify the actual `Character.SetMiscData` signature via codegraph (`codegraph_node SetMiscData` — chunk 4.3 used `GetMiscData`; verify the setter symmetry). Put both helpers in `helpers.go` — every stateful planner uses them.

- [ ] **Step 9.2: Write tests**

Create `internal/planners/wealth_gold_test.go`:

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestWealthGold_Registered(t *testing.T) {
	if LookupPlanner("wealth-gold") == nil {
		t.Fatalf("wealth-gold not registered")
	}
}

func TestWealthGold_NoTargetParam_Failure(t *testing.T) {
	fn := LookupPlanner("wealth-gold")
	mob := &mobs.Mob{}
	res := fn(mob, &goals.Goal{Type: "wealth-gold"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want StatusFailure (missing target param)", res.Status)
	}
}

func TestWealthGold_TargetMet_Success(t *testing.T) {
	fn := LookupPlanner("wealth-gold")
	mob := &mobs.Mob{}
	mob.Character.Gold = 500
	g := &goals.Goal{Type: "wealth-gold", Params: map[string]any{"target": 500}}
	res := fn(mob, g)
	if res.Status != StatusSuccess {
		t.Errorf("status=%v, want StatusSuccess", res.Status)
	}
}

func TestWealthGold_NoSellable_Wanders(t *testing.T) {
	fn := LookupPlanner("wealth-gold")
	mob := &mobs.Mob{}
	mob.Character.Gold = 100
	g := &goals.Goal{Type: "wealth-gold", Params: map[string]any{"target": 500}}
	res := fn(mob, g)
	if res.Command != "wander" {
		t.Errorf("command=%q, want wander", res.Command)
	}
	if res.Status != StatusRunning {
		t.Errorf("status=%v, want StatusRunning", res.Status)
	}
}

// Integration coverage for the sell + pathto branches requires a wired
// findShopInZoneBuying + mobInVendorRoom (TODO-ADAPT in this task and
// Task 6). Defer to Task 23 smoke.
```

- [ ] **Step 9.3: Run tests + build**

Run: `go test ./internal/planners/ -run "TestWealthGold_" -v`
Expected: PASS.

- [ ] **Step 9.4: Commit**

```bash
git add internal/planners/helpers.go internal/planners/wealth_gold.go internal/planners/wealth_gold_test.go
git commit -m "feat(planners): wealth-gold sell-loop planner (4.4)" -m "Sell-loot loop: at vendor + has sellable → sell all; has sellable
+ not at vendor → resolve sticky target_shop_room MiscData + pathto;
no sellable items → wander.

Adds mobMiscIntOr / mobSetMisc shared helpers (every stateful planner
needs them). Adds mobHasSellableItems + mobInVendorRoom heuristic
helpers (TODO-ADAPT for vendor-room check).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 10 — `wealth-item` planner

**Files:**
- Create: `internal/planners/wealth_item.go`
- Create: `internal/planners/wealth_item_test.go`

- [ ] **Step 10.1: Create wealth_item.go**

```go
package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const wealthItemShopRoomKey = "plan:wealth-item:target_shop_room"

func init() {
	RegisterPlanner("wealth-item", wealthItemPlanner)
}

// wealthItemPlanner: acquire specific item by buying at a vendor that
// stocks it, or wandering to find loot if no vendor sells it.
func wealthItemPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	tag := goalParamStringOr(goal, "item_tag", "")
	itemId := goalParamIntOr(goal, "item_id", 0)
	if tag == "" && itemId == 0 {
		return PlanResult{Status: StatusFailure}
	}

	// Item present → success.
	if mobHasItem(mob, tag, itemId) {
		return PlanResult{Status: StatusSuccess}
	}

	// Find a shop in zone selling it.
	shopRoom := mobMiscIntOr(mob, wealthItemShopRoomKey, 0)
	if shopRoom == 0 {
		room, ok := findShopInZoneSelling(mob, tag, itemId)
		if !ok {
			// No vendor → wander (loot chance) or forage if mob is forager.
			return PlanResult{Command: "wander", Status: StatusRunning}
		}
		shopRoom = room
		mobSetMisc(mob, wealthItemShopRoomKey, shopRoom)
	}

	// At shop → buy.
	if mob.Character.RoomId == shopRoom {
		desc := tag
		if desc == "" {
			desc = strconv.Itoa(itemId)
		}
		return PlanResult{Command: "buy " + desc, Status: StatusRunning}
	}

	// Else pathto.
	return PlanResult{Command: "pathto " + strconv.Itoa(shopRoom), Status: StatusRunning}
}

// mobHasItem checks if mob has an item matching tag or id in backpack
// or equipment. Duplicates the catalog package's helper of the same
// name (separate packages — cleanest is to duplicate; if churn grows,
// extract to a shared sub-package).
func mobHasItem(mob *mobs.Mob, tag string, itemId int) bool {
	for i := range mob.Character.Items {
		it := &mob.Character.Items[i]
		if itemId > 0 && it.ItemId == itemId {
			return true
		}
		if tag != "" && it.GetSpec().ComponentTag == tag {
			return true
		}
	}
	// TODO-ADAPT: also walk equipment via mob.Character.Equipment.GetAllItems()
	// — verify the method name matches 4.3 catalog's wealth-item check.
	return false
}
```

- [ ] **Step 10.2: Write tests**

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestWealthItem_Registered(t *testing.T) {
	if LookupPlanner("wealth-item") == nil {
		t.Fatalf("wealth-item not registered")
	}
}

func TestWealthItem_NoParams_Failure(t *testing.T) {
	fn := LookupPlanner("wealth-item")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "wealth-item"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure (no params)", res.Status)
	}
}

func TestWealthItem_ItemPresent_Success(t *testing.T) {
	fn := LookupPlanner("wealth-item")
	mob := &mobs.Mob{}
	mob.Character.Items = []items.Item{{ItemId: 42}}
	g := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_id": 42}}
	res := fn(mob, g)
	if res.Status != StatusSuccess {
		t.Errorf("status=%v, want Success", res.Status)
	}
}

func TestWealthItem_NoShopInZone_Wanders(t *testing.T) {
	fn := LookupPlanner("wealth-item")
	mob := &mobs.Mob{}
	g := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_id": 42}}
	res := fn(mob, g)
	// findShopInZoneSelling stub returns false → wander branch.
	if res.Command != "wander" {
		t.Errorf("command=%q, want wander", res.Command)
	}
}
```

- [ ] **Step 10.3: Run + commit**

Run: `go test ./internal/planners/ -run "TestWealthItem_" -v` → PASS.

```bash
git add internal/planners/wealth_item.go internal/planners/wealth_item_test.go
git commit -m "feat(planners): wealth-item planner (4.4)" -m "Item present → success. Shop in zone sells it + at shop → buy.
Shop in zone but not at it → resolve sticky target_shop_room + pathto.
No shop sells it → wander.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 11 — `craft-item` planner

**Files:**
- Create: `internal/planners/craft_item.go`
- Create: `internal/planners/craft_item_test.go`

- [ ] **Step 11.1: Create craft_item.go**

```go
package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const craftItemStationRoomKey = "plan:craft-item:target_station_room"

func init() {
	RegisterPlanner("craft-item", craftItemPlanner)
}

// craftItemPlanner: produce a specific recipe. Branches:
//   - recipe unknown / skill too low / materials missing → Failure (4.5
//     will reactively seed a wealth-item or mastery-skill goal).
//   - materials on hand + at station (or station empty) → craft.
//   - materials on hand + need station + not at one → resolve sticky
//     target_station_room MiscData + pathto.
func craftItemPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	rid := goalParamStringOr(goal, "recipe_id", "")
	if rid == "" {
		return PlanResult{Status: StatusFailure}
	}
	r := crafting.GetRecipe(rid)
	if r == nil {
		return PlanResult{Status: StatusFailure}
	}

	// Output present? success.
	if r.Output.ItemId > 0 && mobHasItem(mob, "", r.Output.ItemId) {
		return PlanResult{Status: StatusSuccess}
	}

	// Recipe unknown → Failure.
	if !mobKnowsRecipe(mob, rid) {
		return PlanResult{Status: StatusFailure}
	}

	// Skill too low → Failure (let coexisting mastery-skill goal win).
	// TODO-ADAPT: use the same skill-level call as 4.3 catalog's craft-item.
	if !mobMeetsRecipeSkill(mob, r) {
		return PlanResult{Status: StatusFailure}
	}

	// Materials missing → Failure (4.5 will seed wealth-item).
	if !mobHasRecipeMaterials(mob, r) {
		return PlanResult{Status: StatusFailure}
	}

	// At station OR no station required → craft.
	if r.Station == "" || mobInStationRoom(mob, r.Station) {
		return PlanResult{Command: "craft " + rid, Status: StatusRunning}
	}

	// Need station + not at one → pathto.
	stationRoom := mobMiscIntOr(mob, craftItemStationRoomKey, 0)
	if stationRoom == 0 {
		room, ok := findCraftingStationInZone(mob, r.Station)
		if !ok {
			return PlanResult{Status: StatusFailure}
		}
		stationRoom = room
		mobSetMisc(mob, craftItemStationRoomKey, stationRoom)
	}
	return PlanResult{Command: "pathto " + strconv.Itoa(stationRoom), Status: StatusRunning}
}

// mobKnowsRecipe / mobMeetsRecipeSkill / mobHasRecipeMaterials mirror
// the helpers wired in 4.3 catalog's craft_item.go. Duplicate here OR
// move both copies to a shared sub-package — decide based on whether
// the catalog version is still in use after 4.4 ships (it is — predicate
// is independent of planner).
//
// Cleanest: keep both. Catalog's lives in goals/catalog; planners' lives
// here. Both call the same underlying APIs.
func mobKnowsRecipe(mob *mobs.Mob, recipeId string) bool {
	// TODO-ADAPT: replace with the verified API call from 4.3's catalog.
	// Likely: mob.Character.HasRecipe(recipeId) — verify.
	return false
}

func mobMeetsRecipeSkill(mob *mobs.Mob, r *crafting.RecipeSpec) bool {
	// TODO-ADAPT: replace with the verified API call from 4.3's catalog.
	// Likely: mob.Character.GetSkillLevel(skills.SkillTag(r.Skill)) >= r.SkillMinimum
	return false
}

func mobHasRecipeMaterials(mob *mobs.Mob, r *crafting.RecipeSpec) bool {
	// TODO-ADAPT: crafting.HasIngredients(mob.Character.Items, mob.Character.ComponentItems, r)
	// — signature returns (bool, string); use first return.
	return false
}

func mobInStationRoom(mob *mobs.Mob, stationName string) bool {
	// TODO-ADAPT: check rooms.LoadRoom(mob.Character.RoomId).Station == stationName
	return false
}
```

⚠️ Four TODO-ADAPT helpers — implementer wires against the same APIs verified during 4.3 catalog Task 10 (`crafting.GetRecipe`, `mob.Character.HasRecipe`, `mob.Character.GetSkillLevel`, `crafting.HasIngredients`). Plus `mobInStationRoom` for room-station tag (4.3's `findCraftingStationInZone` Task 6 stub).

- [ ] **Step 11.2: Write tests**

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestCraftItem_Registered(t *testing.T) {
	if LookupPlanner("craft-item") == nil {
		t.Fatalf("craft-item not registered")
	}
}

func TestCraftItem_NoRecipeIdParam_Failure(t *testing.T) {
	fn := LookupPlanner("craft-item")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "craft-item"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestCraftItem_UnknownRecipe_Failure(t *testing.T) {
	fn := LookupPlanner("craft-item")
	g := &goals.Goal{Type: "craft-item", Params: map[string]any{"recipe_id": "does-not-exist-recipe"}}
	res := fn(&mobs.Mob{}, g)
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure (unknown recipe)", res.Status)
	}
}
```

- [ ] **Step 11.3: Run + commit**

Run: `go test ./internal/planners/ -run "TestCraftItem_" -v` → PASS.

```bash
git add internal/planners/craft_item.go internal/planners/craft_item_test.go
git commit -m "feat(planners): craft-item planner (4.4)" -m "Output present → Success. Recipe unknown/skill too low/materials
missing → Failure (4.5 will reactively seed coexisting goals).
Materials on hand + station ready → craft. Need station + not at
one → resolve sticky target_station_room MiscData + pathto.

TODO-ADAPT helpers mirror the 4.3 catalog's craft-item APIs.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 12 — `revenge-mob` planner

**Files:**
- Create: `internal/planners/revenge_mob.go`
- Create: `internal/planners/revenge_mob_test.go`

- [ ] **Step 12.1: Create revenge_mob.go**

```go
package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func init() {
	RegisterPlanner("revenge-mob", revengeMobPlanner)
}

// revengeMobPlanner: pursue target across rooms in zone; attack when
// adjacent or same room. No cross-zone pursuit in 4.4.
func revengeMobPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	kind := goalParamStringOr(goal, "target_kind", "")
	id := goalParamIntOr(goal, "target_id", 0)
	if kind == "" || id == 0 {
		return PlanResult{Status: StatusFailure}
	}

	// Target dead → success (catalog predicate will satisfy next tick).
	if !targetExists(kind, id) {
		return PlanResult{Status: StatusSuccess}
	}

	targetRoom := resolveTargetRoomId(kind, id)
	if targetRoom == 0 {
		return PlanResult{Status: StatusFailure}
	}

	// Target out of zone → Failure (4.4 does not pursue cross-zone).
	if r := rooms.LoadRoom(targetRoom); r == nil || r.Zone != mob.Character.Zone {
		return PlanResult{Status: StatusFailure}
	}

	// Same room → attack. (Use bare 'attack' since the existing target
	// resolution in mob commands will find the right target. If the
	// command requires an explicit name, use targetName lookup below.)
	if targetRoom == mob.Character.RoomId {
		name := targetCommandName(kind, id)
		if name == "" {
			return PlanResult{Status: StatusFailure}
		}
		return PlanResult{Command: "attack " + name, Status: StatusRunning}
	}

	// Different room same zone → pathto.
	return PlanResult{Command: "pathto " + strconv.Itoa(targetRoom), Status: StatusRunning}
}

// targetExists returns true iff a live mob/player matching kind+id exists.
func targetExists(kind string, id int) bool {
	switch kind {
	case "mob":
		for _, instId := range mobs.GetAllMobInstanceIds() {
			if inst := mobs.GetInstance(instId); inst != nil && int(inst.MobId) == id {
				return true
			}
		}
		return false
	case "player":
		u := users.GetByUserId(id)
		return u != nil && u.Character.Health > 0
	}
	return false
}

// resolveTargetRoomId returns the current room of a target, or 0 if
// unresolvable. (Same pattern as 4.3 catalog's helpers.go — separate
// package, duplicated.)
func resolveTargetRoomId(kind string, id int) int {
	switch kind {
	case "mob":
		for _, instId := range mobs.GetAllMobInstanceIds() {
			if inst := mobs.GetInstance(instId); inst != nil && int(inst.MobId) == id {
				return inst.Character.RoomId
			}
		}
	case "player":
		if u := users.GetByUserId(id); u != nil {
			return u.Character.RoomId
		}
	}
	return 0
}

// targetCommandName returns the noun string for command targeting.
// For mobs: shorthand or name; for players: username.
func targetCommandName(kind string, id int) string {
	switch kind {
	case "mob":
		for _, instId := range mobs.GetAllMobInstanceIds() {
			if inst := mobs.GetInstance(instId); inst != nil && int(inst.MobId) == id {
				// TODO-ADAPT: prefer inst.Character.Name or ShorthandId for command parsing.
				return inst.Character.Name
			}
		}
	case "player":
		if u := users.GetByUserId(id); u != nil {
			return u.Character.Name
		}
	}
	return ""
}
```

- [ ] **Step 12.2: Write tests**

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestRevengeMob_Registered(t *testing.T) {
	if LookupPlanner("revenge-mob") == nil {
		t.Fatalf("revenge-mob not registered")
	}
}

func TestRevengeMob_NoParams_Failure(t *testing.T) {
	fn := LookupPlanner("revenge-mob")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "revenge-mob"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestRevengeMob_TargetGone_Success(t *testing.T) {
	fn := LookupPlanner("revenge-mob")
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	g := &goals.Goal{Type: "revenge-mob", Params: map[string]any{
		"target_kind": "mob", "target_id": 99999, // doesn't exist
	}}
	res := fn(mob, g)
	if res.Status != StatusSuccess {
		t.Errorf("status=%v, want Success (target nonexistent)", res.Status)
	}
}
```

- [ ] **Step 12.3: Run + commit**

Run: `go test ./internal/planners/ -run "TestRevengeMob_" -v` → PASS.

```bash
git add internal/planners/revenge_mob.go internal/planners/revenge_mob_test.go
git commit -m "feat(planners): revenge-mob planner (4.4)" -m "Target dead → Success. Same room → attack. Same zone different room
→ pathto target's room. Different zone → Failure (4.4 doesn't cross
zones). No MiscData — target room re-resolved each tick, so the
planner adapts immediately if target moves.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 13 — `revenge-faction` planner

**Files:**
- Create: `internal/planners/revenge_faction.go`
- Create: `internal/planners/revenge_faction_test.go`

- [ ] **Step 13.1: Create revenge_faction.go**

```go
package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	RegisterPlanner("revenge-faction", revengeFactionPlanner)
}

// revengeFactionPlanner: search for and attack hostile faction members
// in zone. (Counter is incremented by 4.5's reactive kill hook; predicate
// in 4.3 catalog checks it. This planner just drives the search+attack.)
func revengeFactionPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	factionId := goalParamStringOr(goal, "faction_id", "")
	if factionId == "" {
		return PlanResult{Status: StatusFailure}
	}

	// Find a faction member in zone (any combat state).
	target, ok := findFactionMemberInZone(mob, factionId, false)
	if !ok {
		return PlanResult{Command: "wander", Status: StatusRunning}
	}

	// Same room → attack.
	if target.Character.RoomId == mob.Character.RoomId {
		return PlanResult{Command: "attack " + target.Character.Name, Status: StatusRunning}
	}

	// Different room same zone → pathto.
	return PlanResult{Command: "pathto " + strconv.Itoa(target.Character.RoomId), Status: StatusRunning}
}
```

- [ ] **Step 13.2: Write tests**

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestRevengeFaction_Registered(t *testing.T) {
	if LookupPlanner("revenge-faction") == nil {
		t.Fatalf("revenge-faction not registered")
	}
}

func TestRevengeFaction_NoFactionParam_Failure(t *testing.T) {
	fn := LookupPlanner("revenge-faction")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "revenge-faction"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestRevengeFaction_NoMembersInZone_Wanders(t *testing.T) {
	fn := LookupPlanner("revenge-faction")
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	g := &goals.Goal{Type: "revenge-faction", Params: map[string]any{
		"faction_id": "nonexistent-faction", "target_kill_count": 5,
	}}
	res := fn(mob, g)
	if res.Command != "wander" {
		t.Errorf("command=%q, want wander", res.Command)
	}
	if res.Status != StatusRunning {
		t.Errorf("status=%v, want Running", res.Status)
	}
}
```

- [ ] **Step 13.3: Run + commit**

Run: `go test ./internal/planners/ -run "TestRevengeFaction_" -v` → PASS.

```bash
git add internal/planners/revenge_faction.go internal/planners/revenge_faction_test.go
git commit -m "feat(planners): revenge-faction planner (4.4)" -m "No member in zone → wander to search. Member in same room → attack.
Member in zone different room → pathto. Counter writes for the
predicate live in 4.5's reactive hook.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 14 — `protection-mob` planner

**Files:**
- Create: `internal/planners/protection_mob.go`
- Create: `internal/planners/protection_mob_test.go`

- [ ] **Step 14.1: Create protection_mob.go**

```go
package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func init() {
	RegisterPlanner("protection-mob", protectionMobPlanner)
}

// protectionMobPlanner: defend named ally. Attack their aggressor if
// in combat; else close distance; if target safe in same room, hold.
func protectionMobPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	kind := goalParamStringOr(goal, "target_kind", "")
	id := goalParamIntOr(goal, "target_id", 0)
	if kind == "" || id == 0 {
		return PlanResult{Status: StatusFailure}
	}

	if !targetExists(kind, id) {
		return PlanResult{Status: StatusFailure} // target dead — 4.6 prunes
	}
	targetRoom := resolveTargetRoomId(kind, id)
	if targetRoom == 0 {
		return PlanResult{Status: StatusFailure}
	}
	// Cross-zone protection not supported in 4.4.
	if r := rooms.LoadRoom(targetRoom); r == nil || r.Zone != mob.Character.Zone {
		return PlanResult{Status: StatusFailure}
	}

	aggressor := targetAggressorName(kind, id)

	// Target in combat in same room → attack the aggressor.
	if targetRoom == mob.Character.RoomId && aggressor != "" {
		return PlanResult{Command: "attack " + aggressor, Status: StatusRunning}
	}

	// Target in combat in another room → close the distance.
	if aggressor != "" && targetRoom != mob.Character.RoomId {
		return PlanResult{Command: "pathto " + strconv.Itoa(targetRoom), Status: StatusRunning}
	}

	// Target safe in same room → hold (success — no action this tick).
	if targetRoom == mob.Character.RoomId {
		return PlanResult{Status: StatusSuccess}
	}

	// Target safe in another room same zone → close distance (ready
	// posture).
	return PlanResult{Command: "pathto " + strconv.Itoa(targetRoom), Status: StatusRunning}
}

// targetAggressorName returns the name of whoever is currently attacking
// the target, or empty if target isn't in combat. Inspects the target's
// Aggro pointer's UserId/MobInstanceId.
func targetAggressorName(kind string, id int) string {
	switch kind {
	case "mob":
		for _, instId := range mobs.GetAllMobInstanceIds() {
			inst := mobs.GetInstance(instId)
			if inst == nil || int(inst.MobId) != id {
				continue
			}
			if inst.Character.Aggro == nil {
				return ""
			}
			return aggroToName(inst.Character.Aggro)
		}
	case "player":
		u := users.GetByUserId(id)
		if u == nil || u.Character.Aggro == nil {
			return ""
		}
		return aggroToName(u.Character.Aggro)
	}
	return ""
}

// aggroToName resolves an Aggro pointer to a command-targetable name.
// TODO-ADAPT: the actual Aggro struct fields (UserId int + MobInstanceId
// int) are documented at internal/characters/combat_state_compat.go:36.
// This helper resolves whichever is non-zero to a player username or
// mob name. Implementer wires once and reuses.
func aggroToName(a interface{}) string {
	// Strongly-typed unwrap — see characters.Aggro for the real type.
	// Implementer: replace interface{} with *characters.Aggro and dereference
	// directly. (Left as interface to avoid an import shim in this scaffold.)
	return ""
}
```

⚠️ The `aggroToName` helper takes `interface{}` to avoid importing `characters` in the plan scaffold. Implementer must replace with the real `*characters.Aggro` type and dereference `a.UserId` / `a.MobInstanceId`, looking up the appropriate name via `users.GetByUserId` or `mobs.GetInstance`. Use codegraph to verify the Aggro struct fields if needed.

- [ ] **Step 14.2: Write tests**

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestProtectionMob_Registered(t *testing.T) {
	if LookupPlanner("protection-mob") == nil {
		t.Fatalf("protection-mob not registered")
	}
}

func TestProtectionMob_NoParams_Failure(t *testing.T) {
	fn := LookupPlanner("protection-mob")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "protection-mob"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestProtectionMob_TargetGone_Failure(t *testing.T) {
	fn := LookupPlanner("protection-mob")
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	g := &goals.Goal{Type: "protection-mob", Params: map[string]any{
		"target_kind": "mob", "target_id": 99999,
	}}
	res := fn(mob, g)
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure (target nonexistent → 4.6 prunes)", res.Status)
	}
}
```

- [ ] **Step 14.3: Run + commit**

Run: `go test ./internal/planners/ -run "TestProtectionMob_" -v` → PASS.

```bash
git add internal/planners/protection_mob.go internal/planners/protection_mob_test.go
git commit -m "feat(planners): protection-mob planner (4.4)" -m "Target dead → Failure (4.6 prunes). Target in combat same room → attack
aggressor. Target in combat elsewhere → pathto target room. Target safe
same room → hold (Success). Target safe other room same zone → close
distance (ready posture). Out of zone → Failure.

aggroToName helper needs real type wiring at implementation time
(interface{} placeholder in scaffold).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 15 — `protection-faction` planner

**Files:**
- Create: `internal/planners/protection_faction.go`
- Create: `internal/planners/protection_faction_test.go`

- [ ] **Step 15.1: Create protection_faction.go**

```go
package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	RegisterPlanner("protection-faction", protectionFactionPlanner)
}

// protectionFactionPlanner: defend faction members in zone.
//   - Member in combat in same room → attack their aggressor.
//   - Member in combat in zone (different room) → pathto member's room.
//   - Hostile mob in zone (no member-in-combat) → pathto hostile + attack.
//   - Zone calm → Success (no action; goal stays current — predicate never satisfies).
//   - No members in zone → Failure.
func protectionFactionPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	factionId := goalParamStringOr(goal, "faction_id", "")
	if factionId == "" {
		return PlanResult{Status: StatusFailure}
	}

	// Find a faction member currently in combat in zone.
	if member, ok := findFactionMemberInZone(mob, factionId, true); ok {
		aggressor := aggroToName(member.Character.Aggro)
		if member.Character.RoomId == mob.Character.RoomId && aggressor != "" {
			return PlanResult{Command: "attack " + aggressor, Status: StatusRunning}
		}
		return PlanResult{Command: "pathto " + strconv.Itoa(member.Character.RoomId), Status: StatusRunning}
	}

	// Any members in zone at all?
	if _, ok := findFactionMemberInZone(mob, factionId, false); !ok {
		return PlanResult{Status: StatusFailure}
	}

	// Hostile mob in zone but no member-in-combat → preempt.
	if hostile, ok := findHostileInZone(mob); ok {
		if hostile.Character.RoomId == mob.Character.RoomId {
			return PlanResult{Command: "attack " + hostile.Character.Name, Status: StatusRunning}
		}
		return PlanResult{Command: "pathto " + strconv.Itoa(hostile.Character.RoomId), Status: StatusRunning}
	}

	// Zone calm with members present → hold.
	return PlanResult{Status: StatusSuccess}
}
```

- [ ] **Step 15.2: Write tests**

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestProtectionFaction_Registered(t *testing.T) {
	if LookupPlanner("protection-faction") == nil {
		t.Fatalf("protection-faction not registered")
	}
}

func TestProtectionFaction_NoFactionId_Failure(t *testing.T) {
	fn := LookupPlanner("protection-faction")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "protection-faction"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestProtectionFaction_NoMembersInZone_Failure(t *testing.T) {
	fn := LookupPlanner("protection-faction")
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	g := &goals.Goal{Type: "protection-faction", Params: map[string]any{"faction_id": "nonexistent"}}
	res := fn(mob, g)
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure (no members in zone)", res.Status)
	}
}
```

- [ ] **Step 15.3: Run + commit**

Run: `go test ./internal/planners/ -run "TestProtectionFaction_" -v` → PASS.

```bash
git add internal/planners/protection_faction.go internal/planners/protection_faction_test.go
git commit -m "feat(planners): protection-faction planner (4.4)" -m "Member in combat in same room → attack aggressor. Member in combat
in zone → pathto member's room. Hostile in zone with no member-in-
combat → pathto hostile + attack. Calm zone → Success (hold). No
members in zone → Failure.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 16 — `befriend` planner

**Files:**
- Create: `internal/planners/befriend.go`
- Create: `internal/planners/befriend_test.go`

- [ ] **Step 16.1: Create befriend.go**

```go
package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const (
	befriendCooldownKey    = "plan:befriend:cooldown_round"
	befriendDefaultThreshold = 60
	// Default cooldown between interactions if no config knob exists.
	// (Reads BefriendInteractionCooldown if defined; falls back to 30.)
	befriendDefaultCooldown = 30
)

func init() {
	RegisterPlanner("befriend", befriendPlanner)
}

// befriendPlanner: raise opinion with named target via positive social
// interactions.
//   - Opinion >= threshold → Success.
//   - Cooldown active → Running (waiting).
//   - Target same room → emit social action (rotated), set cooldown.
//   - Target same zone different room → pathto.
//   - Target out of zone → Failure.
func befriendPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	kind := goalParamStringOr(goal, "target_kind", "")
	id := goalParamIntOr(goal, "target_id", 0)
	threshold := goalParamIntOr(goal, "opinion_threshold", befriendDefaultThreshold)
	if kind == "" || id == 0 {
		return PlanResult{Status: StatusFailure}
	}

	// Opinion satisfied → Success. (Mob→mob opinions return 0 per 4.3
	// limitation; planner only meaningfully satisfies for player targets.)
	if kind == "player" && opinions.Get(int(mob.MobId), id) >= threshold {
		return PlanResult{Status: StatusSuccess}
	}

	// Cooldown active?
	nowRound := util.GetRoundCount()
	if cd := uint64(mobMiscIntOr(mob, befriendCooldownKey, 0)); nowRound < cd {
		return PlanResult{Status: StatusRunning} // waiting; no action this tick
	}

	targetRoom := resolveTargetRoomId(kind, id)
	if targetRoom == 0 {
		return PlanResult{Status: StatusFailure}
	}
	if r := rooms.LoadRoom(targetRoom); r == nil || r.Zone != mob.Character.Zone {
		return PlanResult{Status: StatusFailure}
	}

	// Same room → interaction + cooldown.
	if targetRoom == mob.Character.RoomId {
		cooldownLen := uint64(befriendInteractionCooldown())
		mobSetMisc(mob, befriendCooldownKey, int(nowRound+cooldownLen))
		return PlanResult{Command: pickSocialEmote(), Status: StatusRunning}
	}

	// Different room same zone → close distance.
	return PlanResult{Command: "pathto " + strconv.Itoa(targetRoom), Status: StatusRunning}
}

// befriendInteractionCooldown reads the config knob (if defined) or
// returns the default.
//
// TODO-ADAPT: if a BefriendInteractionCooldown knob doesn't exist yet,
// add it to configs.Balance (default 30) as part of this task. Otherwise
// fall back to the constant.
func befriendInteractionCooldown() int {
	// Cheapest path: just use the constant. Add config knob later if
	// gameplay tuning needs it.
	_ = configs.GetBalanceConfig()
	return befriendDefaultCooldown
}
```

- [ ] **Step 16.2: Write tests**

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestBefriend_Registered(t *testing.T) {
	if LookupPlanner("befriend") == nil {
		t.Fatalf("befriend not registered")
	}
}

func TestBefriend_NoParams_Failure(t *testing.T) {
	fn := LookupPlanner("befriend")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "befriend"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestBefriend_TargetOutOfZone_Failure(t *testing.T) {
	fn := LookupPlanner("befriend")
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	g := &goals.Goal{Type: "befriend", Params: map[string]any{
		"target_kind": "player", "target_id": 99999,
	}}
	res := fn(mob, g)
	// User doesn't exist → resolveTargetRoomId returns 0 → Failure.
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}
```

- [ ] **Step 16.3: Run + commit**

Run: `go test ./internal/planners/ -run "TestBefriend_" -v` → PASS.

```bash
git add internal/planners/befriend.go internal/planners/befriend_test.go
git commit -m "feat(planners): befriend planner (4.4)" -m "Opinion >= threshold → Success. Cooldown active → Running (waiting).
Target in same room → social emote + set cooldown. Same zone different
room → pathto. Out of zone → Failure.

Cooldown prevents emote-spam every tick. opinions.Get returns 0 for
mob→mob (4.3 catalog limitation) so planner only meaningfully
satisfies for player targets.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 17 — `befriend-faction` planner

**Files:**
- Create: `internal/planners/befriend_faction.go`
- Create: `internal/planners/befriend_faction_test.go`

- [ ] **Step 17.1: Create befriend_faction.go**

```go
package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const (
	befriendFactionFocusKey    = "plan:befriend-faction:focus_mob_instance_id"
	befriendFactionCooldownKey = "plan:befriend-faction:cooldown_round"
)

func init() {
	RegisterPlanner("befriend-faction", befriendFactionPlanner)
}

// befriendFactionPlanner: get into proximity with faction members.
// Picks one member as focus (sticky), pathing to / emoting near them.
// Actual rep accumulation is via 4.5's reactive counter-writing hook.
func befriendFactionPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	factionId := goalParamStringOr(goal, "faction_id", "")
	if factionId == "" {
		return PlanResult{Status: StatusFailure}
	}

	// Resolve or pick a focus member.
	focusId := mobMiscIntOr(mob, befriendFactionFocusKey, 0)
	var focus *mobs.Mob
	if focusId != 0 {
		focus = mobs.GetInstance(focusId)
	}
	// Stale focus (gone or wrong zone) → re-pick.
	if focus == nil || focus.Character.Zone != mob.Character.Zone {
		picked, ok := findFactionMemberInZone(mob, factionId, false)
		if !ok {
			return PlanResult{Status: StatusFailure}
		}
		focus = picked
		mobSetMisc(mob, befriendFactionFocusKey, focus.InstanceId)
	}

	// Cooldown active?
	nowRound := util.GetRoundCount()
	if cd := uint64(mobMiscIntOr(mob, befriendFactionCooldownKey, 0)); nowRound < cd {
		return PlanResult{Status: StatusRunning}
	}

	// Same room → emote + cooldown.
	if focus.Character.RoomId == mob.Character.RoomId {
		cooldownLen := uint64(befriendInteractionCooldown())
		mobSetMisc(mob, befriendFactionCooldownKey, int(nowRound+cooldownLen))
		return PlanResult{Command: pickSocialEmote(), Status: StatusRunning}
	}

	// Different room same zone → pathto.
	return PlanResult{Command: "pathto " + strconv.Itoa(focus.Character.RoomId), Status: StatusRunning}
}
```

- [ ] **Step 17.2: Write tests**

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestBefriendFaction_Registered(t *testing.T) {
	if LookupPlanner("befriend-faction") == nil {
		t.Fatalf("befriend-faction not registered")
	}
}

func TestBefriendFaction_NoFactionId_Failure(t *testing.T) {
	fn := LookupPlanner("befriend-faction")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "befriend-faction"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestBefriendFaction_NoMembersInZone_Failure(t *testing.T) {
	fn := LookupPlanner("befriend-faction")
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	g := &goals.Goal{Type: "befriend-faction", Params: map[string]any{"faction_id": "nonexistent"}}
	res := fn(mob, g)
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure (no members)", res.Status)
	}
}
```

- [ ] **Step 17.3: Run + commit**

Run: `go test ./internal/planners/ -run "TestBefriendFaction_" -v` → PASS.

```bash
git add internal/planners/befriend_faction.go internal/planners/befriend_faction_test.go
git commit -m "feat(planners): befriend-faction planner (4.4)" -m "Pick a focus member in zone (sticky via MiscData), pathto + emote
on cooldown. Actual rep counter writes ship in 4.5's reactive hook;
4.4's planner just gets the mob into proximity.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 18 — `mastery-skill` planner

**Files:**
- Create: `internal/planners/mastery_skill.go`
- Create: `internal/planners/mastery_skill_test.go`

- [ ] **Step 18.1: Create mastery_skill.go**

```go
package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	RegisterPlanner("mastery-skill", masterySkillPlanner)
}

// masterySkillPlanner: dispatch to per-context training action based
// on the skill's TrainingContext. Per-skill mapping lives in
// skillTrainingTable.
func masterySkillPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	skillName := goalParamStringOr(goal, "skill_name", "")
	targetRank := goalParamIntOr(goal, "target_rank", 0)
	if skillName == "" || targetRank == 0 {
		return PlanResult{Status: StatusFailure}
	}

	currentRank := mobSkillRank(mob, skillName)
	if currentRank >= targetRank {
		return PlanResult{Status: StatusSuccess}
	}

	switch SkillTrainingContextOf(skillName) {
	case TrainingCombat:
		// Auto-aggro mob in room → attack. Else wander.
		if hostile, ok := findHostileInRoom(mob); ok {
			return PlanResult{Command: "attack " + hostile.Character.Name, Status: StatusRunning}
		}
		return PlanResult{Command: "wander", Status: StatusRunning}
	case TrainingCrafting:
		// At a station with a known recipe? craft it. Else pathto station.
		recipeId := pickKnownRecipeForSkill(mob, skillName)
		if recipeId == "" {
			return PlanResult{Status: StatusFailure}
		}
		// TODO-ADAPT: stationName comes from crafting.GetRecipe(recipeId).Station.
		stationName := craftingStationForRecipe(recipeId)
		if stationName == "" || mobInStationRoom(mob, stationName) {
			return PlanResult{Command: "craft " + recipeId, Status: StatusRunning}
		}
		room, ok := findCraftingStationInZone(mob, stationName)
		if !ok {
			return PlanResult{Status: StatusFailure}
		}
		return PlanResult{Command: "pathto " + strconv.Itoa(room), Status: StatusRunning}
	case TrainingForaging:
		return PlanResult{Command: "forage", Status: StatusRunning}
	case TrainingSocial:
		// Anyone in room? emote. Else wander.
		if roomHasObserver(mob) {
			return PlanResult{Command: pickSocialEmote(), Status: StatusRunning}
		}
		return PlanResult{Command: "wander", Status: StatusRunning}
	case TrainingSkullduggery:
		// No autonomous theft in 4.4 (too easy to misfire). Wander.
		return PlanResult{Command: "wander", Status: StatusRunning}
	}
	return PlanResult{Status: StatusFailure} // TrainingUnknown
}

// mobSkillRank reads the mob's current rank for a skill. Wraps the
// 4.3 catalog's identical helper. TODO-ADAPT to use the actual call.
func mobSkillRank(mob *mobs.Mob, skillName string) int {
	// TODO-ADAPT: mob.Character.GetSkillLevel(skills.SkillTag(skillName))
	return 0
}

func findHostileInRoom(mob *mobs.Mob) (*mobs.Mob, bool) {
	for _, instId := range mobs.GetAllMobInstanceIds() {
		inst := mobs.GetInstance(instId)
		if inst == nil || inst.Character.RoomId != mob.Character.RoomId || inst.InstanceId == mob.InstanceId {
			continue
		}
		if inst.AutoAggro {
			return inst, true
		}
	}
	return nil, false
}

func roomHasObserver(mob *mobs.Mob) bool {
	// TODO-ADAPT: rooms.LoadRoom(mob.Character.RoomId).GetPlayers()
	// or similar — count > 0 means there's an audience for the emote.
	// Cheapest stub: always true (gives the social emote a chance).
	return true
}

func craftingStationForRecipe(recipeId string) string {
	// TODO-ADAPT: crafting.GetRecipe(recipeId).Station
	return ""
}
```

- [ ] **Step 18.2: Write tests**

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestMasterySkill_Registered(t *testing.T) {
	if LookupPlanner("mastery-skill") == nil {
		t.Fatalf("mastery-skill not registered")
	}
}

func TestMasterySkill_NoParams_Failure(t *testing.T) {
	fn := LookupPlanner("mastery-skill")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "mastery-skill"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestMasterySkill_UnknownSkill_Failure(t *testing.T) {
	fn := LookupPlanner("mastery-skill")
	g := &goals.Goal{Type: "mastery-skill", Params: map[string]any{
		"skill_name": "made-up-skill", "target_rank": 30,
	}}
	res := fn(&mobs.Mob{}, g)
	// Unknown skill → TrainingUnknown → Failure.
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestMasterySkill_ForagingSkill_ForageCommand(t *testing.T) {
	fn := LookupPlanner("mastery-skill")
	g := &goals.Goal{Type: "mastery-skill", Params: map[string]any{
		"skill_name": "foraging", "target_rank": 30,
	}}
	res := fn(&mobs.Mob{}, g)
	if res.Command != "forage" {
		t.Errorf("command=%q, want forage", res.Command)
	}
	if res.Status != StatusRunning {
		t.Errorf("status=%v, want Running", res.Status)
	}
}
```

- [ ] **Step 18.3: Run + commit**

Run: `go test ./internal/planners/ -run "TestMasterySkill_" -v` → PASS.

```bash
git add internal/planners/mastery_skill.go internal/planners/mastery_skill_test.go
git commit -m "feat(planners): mastery-skill planner (4.4)" -m "Dispatch per skill's TrainingContext (from skillTrainingTable):
Combat → attack hostile in room else wander. Crafting → craft known
recipe at station else pathto. Foraging → forage. Social → emote.
Skullduggery → wander (no autonomous theft in 4.4). Unknown → Failure.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 19 — `mastery-equip` planner

**Files:**
- Create: `internal/planners/mastery_equip.go`
- Create: `internal/planners/mastery_equip_test.go`

- [ ] **Step 19.1: Create mastery_equip.go**

```go
package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const masteryEquipShopRoomKey = "plan:mastery-equip:target_shop_room"

func init() {
	RegisterPlanner("mastery-equip", masteryEquipPlanner)
}

// masteryEquipPlanner: upgrade a slot to rarity tier ≥ target.
//   - Current slot meets tier → Success.
//   - Shop in zone stocks suitable item + at shop → buy then (next tick) wear.
//   - Shop in zone stocks suitable item + not at shop → pathto sticky room.
//   - No shop sells suitable item → Failure (4.5 might seed wealth-gold).
func masteryEquipPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	slot := goalParamStringOr(goal, "slot", "")
	minTier := goalParamIntOr(goal, "min_rarity_tier", 0)
	if slot == "" || minTier == 0 {
		return PlanResult{Status: StatusFailure}
	}

	if mobSlotMeetsTier(mob, slot, minTier) {
		return PlanResult{Status: StatusSuccess}
	}

	// Resolve sticky shop room.
	shopRoom := mobMiscIntOr(mob, masteryEquipShopRoomKey, 0)
	if shopRoom == 0 {
		// TODO-ADAPT: extend findShopInZoneSelling with a slot filter
		// (or add a sibling findShopInZoneSellingForSlot). Cheapest
		// stub: just look for any shop and hope it has slot items.
		room, ok := findShopInZoneSelling(mob, "", 0)
		if !ok {
			return PlanResult{Status: StatusFailure}
		}
		shopRoom = room
		mobSetMisc(mob, masteryEquipShopRoomKey, shopRoom)
	}

	// At shop → buy. (Item name selection requires a shop-stock query
	// that depends on shops API; for 4.4 a generic "buy <slot>" may not
	// work — implementer should verify the shop's `buy` command syntax
	// at this stage.)
	if mob.Character.RoomId == shopRoom {
		return PlanResult{Command: "buy " + slot, Status: StatusRunning}
	}

	// Not at shop → pathto.
	return PlanResult{Command: "pathto " + strconv.Itoa(shopRoom), Status: StatusRunning}
}

// mobSlotMeetsTier returns true if the item equipped in slot has
// rarity_tier ≥ minTier. Untagged items use the engine fallback (tier 50).
//
// TODO-ADAPT: wire against 4.3 catalog's mobSlotRarityTier (separate
// package — duplicate or re-export via a sub-package).
func mobSlotMeetsTier(mob *mobs.Mob, slot string, minTier int) bool {
	// TODO-ADAPT.
	return false
}
```

- [ ] **Step 19.2: Write tests**

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestMasteryEquip_Registered(t *testing.T) {
	if LookupPlanner("mastery-equip") == nil {
		t.Fatalf("mastery-equip not registered")
	}
}

func TestMasteryEquip_NoParams_Failure(t *testing.T) {
	fn := LookupPlanner("mastery-equip")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "mastery-equip"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestMasteryEquip_NoShopInZone_Failure(t *testing.T) {
	fn := LookupPlanner("mastery-equip")
	g := &goals.Goal{Type: "mastery-equip", Params: map[string]any{
		"slot": "weapon", "min_rarity_tier": 60,
	}}
	res := fn(&mobs.Mob{}, g)
	// findShopInZoneSelling stub returns false → Failure branch.
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure (no shop)", res.Status)
	}
}
```

- [ ] **Step 19.3: Run + commit**

Run: `go test ./internal/planners/ -run "TestMasteryEquip_" -v` → PASS.

```bash
git add internal/planners/mastery_equip.go internal/planners/mastery_equip_test.go
git commit -m "feat(planners): mastery-equip planner (4.4)" -m "Slot meets tier → Success. Otherwise resolve sticky shop room with
slot-appropriate stock, pathto, buy. TODO-ADAPT: shop-stock filter
for the slot is currently a stub — wire against shops API.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 20 — `visit-zone` planner

**Files:**
- Create: `internal/planners/visit_zone.go`
- Create: `internal/planners/visit_zone_test.go`

- [ ] **Step 20.1: Create visit_zone.go**

```go
package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const visitZoneNextHopKey = "plan:visit-zone:next_hop_zone"

func init() {
	RegisterPlanner("visit-zone", visitZonePlanner)
}

// visitZonePlanner: walk to an exit-room leading toward target zone.
// Multi-hop uses a simple "any unvisited adjacent zone" heuristic.
func visitZonePlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	target := goalParamStringOr(goal, "target_zone", "")
	if target == "" {
		return PlanResult{Status: StatusFailure}
	}

	// Already visited / currently in target → Success (predicate will fire).
	if mob.VisitedZones[target] {
		return PlanResult{Status: StatusSuccess}
	}
	if mob.Character.Zone == target {
		return PlanResult{Status: StatusSuccess}
	}

	// Target is adjacent? Walk into it directly.
	for _, adj := range zoneAdjacentTo(mob.Character.Zone) {
		if adj == target {
			exitRoom, ok := exitRoomToward(mob, target)
			if !ok {
				return PlanResult{Status: StatusFailure}
			}
			return PlanResult{Command: "pathto " + strconv.Itoa(exitRoom), Status: StatusRunning}
		}
	}

	// Multi-hop: pick "any unvisited adjacent zone" heuristic.
	hop := mobMiscStringOr(mob, visitZoneNextHopKey, "")
	if hop == "" {
		for _, adj := range zoneAdjacentTo(mob.Character.Zone) {
			if !mob.VisitedZones[adj] {
				hop = adj
				break
			}
		}
		if hop == "" {
			return PlanResult{Status: StatusFailure}
		}
		mobSetMisc(mob, visitZoneNextHopKey, hop)
	}
	exitRoom, ok := exitRoomToward(mob, hop)
	if !ok {
		return PlanResult{Status: StatusFailure}
	}
	return PlanResult{Command: "pathto " + strconv.Itoa(exitRoom), Status: StatusRunning}
}

// exitRoomToward returns the room id in mob's current zone that has an
// exit leading INTO the target zone. ok=false if no such room.
//
// TODO-ADAPT: walk rooms in mob's zone, inspect each room's Exits map,
// find an exit whose destination room's Zone == targetZone. Return
// the source room id.
func exitRoomToward(mob *mobs.Mob, targetZone string) (int, bool) {
	// TODO-ADAPT.
	return 0, false
}

// mobMiscStringOr reads a string MiscData value with a fallback default.
// Pair with mobMiscIntOr (added in Task 9). Move to helpers.go if not
// already there.
func mobMiscStringOr(mob *mobs.Mob, key string, def string) string {
	raw := mob.Character.GetMiscData(key)
	if s, ok := raw.(string); ok {
		return s
	}
	return def
}
```

- [ ] **Step 20.2: Write tests**

```go
package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestVisitZone_Registered(t *testing.T) {
	if LookupPlanner("visit-zone") == nil {
		t.Fatalf("visit-zone not registered")
	}
}

func TestVisitZone_NoTargetZone_Failure(t *testing.T) {
	fn := LookupPlanner("visit-zone")
	res := fn(&mobs.Mob{}, &goals.Goal{Type: "visit-zone"})
	if res.Status != StatusFailure {
		t.Errorf("status=%v, want Failure", res.Status)
	}
}

func TestVisitZone_AlreadyVisited_Success(t *testing.T) {
	fn := LookupPlanner("visit-zone")
	mob := &mobs.Mob{}
	mob.VisitedZones = map[string]bool{"stillwater": true}
	g := &goals.Goal{Type: "visit-zone", Params: map[string]any{"target_zone": "stillwater"}}
	res := fn(mob, g)
	if res.Status != StatusSuccess {
		t.Errorf("status=%v, want Success", res.Status)
	}
}

func TestVisitZone_InTargetZone_Success(t *testing.T) {
	fn := LookupPlanner("visit-zone")
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	g := &goals.Goal{Type: "visit-zone", Params: map[string]any{"target_zone": "stillwater"}}
	res := fn(mob, g)
	if res.Status != StatusSuccess {
		t.Errorf("status=%v, want Success", res.Status)
	}
}
```

- [ ] **Step 20.3: Run + commit**

Run: `go test ./internal/planners/ -run "TestVisitZone_" -v` → PASS.

```bash
git add internal/planners/visit_zone.go internal/planners/visit_zone_test.go
git commit -m "feat(planners): visit-zone planner (4.4)" -m "Already visited / currently in target → Success. Target adjacent →
pathto exit room leading into target. Multi-hop → 'any unvisited
adjacent zone' heuristic (sticky via MiscData). exitRoomToward
TODO-ADAPT — walk rooms in zone + inspect exit destinations.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 21 — Per-archetype YAML edits (insert `try_goal_planner`)

Per spec §5. 18 non-boss archetype YAMLs get `try_goal_planner` inserted at an author-chosen position in the tree.

**Files:**
- Modify: 18 archetype YAMLs in `_datafiles/world/dogmud/behaviors/archetypes/`

**Per spec §5 placement guidance, summary:**

| Archetype | Placement guidance |
|-----------|-------------------|
| ambusher, combat_passive, defensive_caster, generic_fighter, melee_self_buff, predator, pure_caster, scout, support_caster, tank_taunter, leader, prey | AFTER reflex (flee-if-alpha-dead, pack-cohesion); BEFORE wander/idle. Typically after existing `try_combat` so default reactive combat still works. |
| forager | BEFORE the forage selector loop; AFTER any patrol step. |
| noncombat_shopkeeper, noncombat_passive, noncombat_questgiver | Near top of tree. These are mostly idle today; goal-driven behavior IS their primary action. Shopkeeper specifically: AFTER any schedule step. |
| lookout | AFTER patrol step. |

**Skipped:** boss_chrysalis_phantom, boss_edrin, boss_rhett, boss_soren, boss_sylara (no seeded defaults; would always be no-op).

- [ ] **Step 21.1: Read each non-boss archetype YAML in turn**

For each of the 18 non-boss archetype files, read the file, identify the right insertion point per guidance above, and insert a `try_goal_planner` action node. The exact YAML shape depends on the surrounding tree structure — match the existing pattern.

Example for a `selector`-style archetype tree:

```yaml
tree:
  type: selector
  children:
    - type: action
      do: try_combat              # existing reactive combat
    - type: action
      do: try_goal_planner        # NEW chunk 4.4 — strategic goal pursuit
    - type: action
      do: wander                  # existing idle
```

For archetypes whose tree is a single action or a flat sequence, place the new action immediately before whichever existing action represents "idle / wander / default fallback".

- [ ] **Step 21.2: Verify each edit doesn't break the YAML parse**

After all 18 edits, run:

Run: `go build ./...`
Expected: clean build.

Run: `timeout 60 go run . 2>&1 | grep -iE "panic|loadedcount" | head -25`
Expected: archetypes load without panic; all 23 archetype `loadedCount`s appear.

Stop with Ctrl+C.

- [ ] **Step 21.3: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/
git commit -m "content(archetypes): insert try_goal_planner into 18 non-boss archetypes (4.4)" -m "Per spec §5: each non-boss archetype's tree gets try_goal_planner
inserted at an author-chosen position. Combat archetypes after reflex
reactions, before idle/wander. Forager before forage loop, after
patrol. Noncombatants near top. Lookout after patrol.

Boss archetypes skipped (no seeded defaults; insertion would be no-op).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 22 — `internal/planners/context.md`

Per-package documentation per project convention.

**Files:**
- Create: `internal/planners/context.md`

- [ ] **Step 22.1: Author context.md**

Match the voice and structure of `internal/goals/context.md` (written in 4.3) and `internal/goals/catalog/context.md`. Factual, dense, no marketing.

Cover (~100-150 lines):
- Package purpose: chunk-4.4 per-goal-type planners; turns 4.2-selected goals into tactical actions.
- Activation: regular import from `main.go` fires per-planner `init()` registrations + makes `ClearPlanState` callable.
- File layout: `planners.go` (framework), `state.go` (cleanup), `helpers.go` (subsystem adapters), `skill_training_table.go` (mastery-skill dispatch), 13 `<type>.go` files (one per planner) + tests alongside.
- The 13 planners as a table (name + one-line description).
- How to add a new planner: file naming, `init()` registration, PlanFn signature, MiscData key convention (`plan:<type>:`), test pattern.
- MiscData convention: `plan:<goal_type>:<key>` prefix — wiped on goal switch by `ClearPlanState` registered into `goals.Recompute` via main.go.
- Helper adapter pattern: TODO-ADAPT shims wrap subsystems (shops, factions, room/zone, items). Local-impact when subsystem APIs change.
- Out-of-scope: reactive goal generation (4.5), satisfaction sweep (4.6), planner visualization, general-purpose planner.
- Reference the spec: `docs/superpowers/specs/2026-05-27-mob-aliveness-4.4-strategic-tactical-translation-design.md`.

- [ ] **Step 22.2: Commit**

```bash
git add internal/planners/context.md
git commit -m "docs: context.md for internal/planners (4.4)" -m "Per project convention: matches internal/goals/context.md +
internal/goals/catalog/context.md voice and structure. Covers
the 13-planner catalog, framework files, activation via main.go
import, MiscData key convention, and what's out of scope.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 23 — Smoke + roadmap + patch notes

Pre-push SOP. Mark the chunk done. Engineered smoke for the new observable behavior.

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`
- Modify: `PATCH_NOTES.md`
- Verify: `_datafiles/config.yaml`

- [ ] **Step 23.1: Wipe instance saves**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

- [ ] **Step 23.2: Boot test**

Run: `timeout 90 go run . 2>&1 | grep -iE "panic|loadedcount|started|fatal" | head -40`

Expect:
- `loadedCount` lines for all standard loads (mobs, rooms, archetypes).
- `MainWorker state="Started"`.
- No panics.

Stop with Ctrl+C.

- [ ] **Step 23.3: Live engineered smoke #1 — survival**

Reboot. Connect as admin. Find a combat-archetype mob with a seeded `survival` goal (any mob whose archetype was edited in 4.3 Task 21 should qualify — e.g., a thief in stillwater).

```
admin <mob_name> health 10
```

(or a similar admin command to set HP low — verify the exact admin command). Tail the server log; expect a `flee <exit>` command issued by the mob next round, plus a `goals.switch` if the survival ContextScore now exceeds whatever was current.

- [ ] **Step 23.4: Live engineered smoke #2 — revenge-mob**

Reboot. Connect as admin (let's say userId=1, character "Adminuser" in zone "stillwater"). Pick a combat mob in stillwater (e.g., a thief, MobId=200).

```
goal add 200 revenge-mob 80 target_kind=player target_id=1
```

Move admin to the same room as the mob. Expect mob to `attack adminuser`. Move admin to an adjacent room. Expect mob to `pathto <admin's room id>`.

- [ ] **Step 23.5: Live engineered smoke #3 — wealth-gold sell loop**

Reboot. Find a thief (auto-seeded with `wealth-gold target=500` per 4.3 Task 21). Drop some loot in the thief's room (or admin-give an item with value > 0). Tail the log over a few rounds; expect:

1. Thief picks up loot (or has it).
2. `goal current <thief>` shows `wealth-gold` as current (or `survival` if HP is low).
3. After a few rounds: `pathto` to a vendor room, then `sell all`, then gold rises.

(This smoke depends on the shop helpers being wired in Task 6; if `findShopInZoneBuying` is still a stub, the planner will `wander` instead — note this as a partial-wire condition for the chunk.)

- [ ] **Step 23.6: Update `MOB_ALIVENESS_ROADMAP.md`**

Find the 4.4 row in the chunks table. Flip status to `Done`. Update size from `L` to `XL` (per brainstorming decision). Update rollup from `25 / 42 done • 0 in progress • 17 not started` to `26 / 42 done • 0 in progress • 16 not started`.

Find the 4.4 detail block (lines starting `### 4.4 Strategic→tactical translation`). Add a `**Shipped:**` paragraph similar to 4.2/4.3's, summarizing the 13 planners + framework + per-archetype edits + MiscData convention + boot wiring.

- [ ] **Step 23.7: Append `PATCH_NOTES.md` entry**

Insert at the top, above the 4.3 entry:

```markdown
## 2026-05-27 — Mob aliveness chunk 4.4: strategic → tactical translation

NPCs now actually pursue the goals 4.2 selects from 4.3's catalog.
Combat-capable mobs flee when HP drops below the survival threshold.
Thieves wander to vendors and sell loot when seeded with wealth-gold.
Named NPCs path to and attack revenge targets, defend protection
targets, walk among faction members for befriend / protection / revenge
goals. Foragers move toward unvisited zones for visit-zone goals.
Crafters seek stations to produce known recipes.

13 hand-authored per-type planners in the new internal/planners/
package, dispatched via one new behavior-tree action `try_goal_planner`
inserted into 18 non-boss archetype trees at the priority position
each archetype's author chose. Planners are pure Go functions returning
one command + status per tick; intermediate state lives in
mob.Character.MiscData under a `plan:<goal_type>:` key prefix that's
wiped automatically on goal switch.

Boot wiring registers planners.ClearPlanState as a callback into
goals.Recompute (mirrors 4.2's SetWeightsLookup and 4.3's
SetArchetypeDefaultsLookup patterns). Bridges the goals → planners
direction without an import cycle.

Permanent-stuck-goal detection (planner-perpetually-fails) is deferred
to 4.6's pruning sweep; reactive seeding of coexisting goals (e.g.,
craft-item triggers a wealth-item for missing materials) is deferred
to 4.5. Cross-zone pursuit, plan visualization admin command, and
schedule-aware planning are all out of 4.4 scope.

No schema change. Player-facing impact: noticeable NPC liveness.
```

- [ ] **Step 23.8: Verify `Logging.LogToFile: false`**

Confirm `_datafiles/config.yaml` has `LogToFile: false`. (Should already be set from prior pushes.)

- [ ] **Step 23.9: Full test suite**

Run: `go test ./...`
Expected: PASS across the board.

- [ ] **Step 23.10: Commit roadmap + patch notes**

```bash
git add MOB_ALIVENESS_ROADMAP.md PATCH_NOTES.md _datafiles/config.yaml
git commit -m "chore(roadmap): mark aliveness 4.4 strategic→tactical Done (26/42)" -m "- Roadmap: 4.4 status -> Done, size L -> XL (per brainstorming).
- Roadmap rollup: 25/42 -> 26/42.
- PATCH_NOTES: chunk 4.4 entry.
- Config LogToFile=false already set (verified).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Self-Review

**Spec coverage:** every section of the 4.4 spec maps to tasks:
- §1 Architecture → Tasks 1-7 (framework + supporting infrastructure).
- §2 API surface → Tasks 1 (PlanFn types), 2 (ClearPlanState), 3 (SetPlanStateClear callback), 4 (boot wiring), 5 (try_goal_planner action).
- §3 The 13 planners → Tasks 8-20 (one per type).
- §4 Supporting infrastructure → Task 6 (helpers.go) + per-planner local helpers as needed.
- §5 Btree integration / per-archetype edits → Task 21.
- §6 Plan-state cleanup → Tasks 2 + 3 (ClearPlanState + callback wiring).
- §7 Edge cases → covered across per-planner branches + Task 5's panic-recovery + Task 3's idempotent-callback test.
- §8 Testing strategy & rollout → distributed across all per-planner tasks + Task 23.

**Placeholder scan:** every TODO-ADAPT comment is a documented adaptation point with explicit "use codegraph to verify" guidance — same pattern as the 4.3 plan that the implementer already validated. Per-planner tests are minimal (registration + branch happy/sad paths) because deeper integration coverage requires live mob/zone/shop fixtures that belong in Task 23 smoke.

**Type consistency:** `PlanFn(*mobs.Mob, *goals.Goal) PlanResult`, `PlanResult{Command, Status}`, `BTreeStatus` (StatusFailure/StatusSuccess/StatusRunning), `RegisterPlanner(string, PlanFn)`, `LookupPlanner(string) PlanFn`, `PlanStateClearFn(*mobs.Mob)`, `SetPlanStateClear(fn)` consistent across Tasks 1-5. MiscData key prefix `plan:` consistent across all per-planner tasks. `goalParamIntOr` / `goalParamStringOr` / `mobMiscIntOr` / `mobSetMisc` / `mobMiscStringOr` helpers introduced in Task 8/9/20 and reused consistently.

**Scope:** 23 tasks, single feature branch (per CLAUDE.md), XL chunk. Per-planner tasks are slightly more complex than 4.3 catalog tasks (planners have multi-branch logic + MiscData; catalog types are mostly Predicate + ContextScore), but each is still bite-sized at ~5 steps.





