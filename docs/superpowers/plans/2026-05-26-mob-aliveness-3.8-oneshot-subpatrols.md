# Mob Aliveness 3.8 — One-Shot Sub-Patrols (Caravan Runner + Forager Delivery) — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a one-shot sub-patrol engine primitive and apply it to two consumers — Lars (mob 359) as caravan depot-to-vendor runner, and Marsh/Steppe foragers' Delivering phase.

**Architecture:** New `loop_shape: oneshot` patrol mode runs waypoints once and emits `events.PatrolCompleted` on the final dwell. Outer state machines (caravan arrival listener, forager `forager_step` btree action) call `mobs.StartOneshotPatrol(mob, patrolId)` to dispatch a delivery walk and advance their own state on the completion event. Caravan main route truncates from 22 → 4 waypoints; Lars carries cargo wagon↔vendors. Foragers' StateDelivering collapses from internal vendor-room loop to "start patrol, wait, advance."

**Tech Stack:** Go 1.24+, YAML patrol files, GoMud event system (`events.RegisterListener`), patrol primitive from chunks 3.4 + 3.7 (`internal/mobs/patrol.go`, `internal/hooks/NewRound_IdleMobs_patrol.go`).

**Spec:** `docs/superpowers/specs/2026-05-26-mob-aliveness-3.8-oneshot-subpatrols-design.md`

**Branch:** `feature/mob-aliveness-3.8-oneshot-subpatrols`

---

## File Structure

**Modified files (engine — Phase 1):**

- `internal/mobs/patrol.go` — `Patrol` accepts `loop_shape: oneshot`; new `StartOneshotPatrol` + `ClearOneshotPatrol` helpers
- `internal/mobs/patrol_test.go` — tests for the new helpers + parsing
- `internal/events/eventtypes.go` — new `PatrolCompleted` event
- `internal/hooks/NewRound_IdleMobs_patrol.go` — new `WantsComplete` plan branch; emits event + clears patrol
- `internal/hooks/NewRound_IdleMobs_patrol_test.go` — test for the new emission path

**Modified files (forager — Phase 2):**

- `internal/forager/territory.go` — `ForagerProfile` gains `DeliveryPatrolId` field; Marsh + Steppe profiles populated
- `internal/behaviortree/actions_forager.go` — `tickForagerDeliveringTown` rewrites to delegate to oneshot patrol; 5.4 sanctuary-fallback safety
- `internal/hooks/hooks.go` — register forager arrival + completion listeners

**New files (forager — Phase 2):**

- `internal/forager/arrival_listener.go` — consumes `forager_vendor` arrival events, fires `SellToVendor`
- `internal/forager/completion_listener.go` — advances state machine from `StateDelivering` to next state on `PatrolCompleted`
- `internal/forager/arrival_listener_test.go`
- `internal/forager/completion_listener_test.go`
- `_datafiles/world/dogmud/patrols/stillwater_marsh/marsh_forager_delivery.yaml`
- `_datafiles/world/dogmud/patrols/ironwind_steppe/steppe_forager_delivery.yaml`

**Modified files (caravan — Phase 3):**

- `_datafiles/world/dogmud/mobs/thornwall_city/359-lars.yaml` — strength bump, runner group tag, Ketil's-son flavor
- `_datafiles/world/dogmud/patrols/thornwall_outskirts/caravan_thornwall_stillwater.yaml` — 22 → 4 waypoints
- `internal/caravan/arrival_listener.go` — wp0/wp2 dispatch starts runner circuit; vendor handler swaps wagon → runner source; 5.3 stranded-cargo safety
- `internal/caravan/synthesize_state.go` — read Lars's patrol state to distinguish `*Route` from `*Dwell`
- `internal/caravan/synthesize_state_test.go` — extend table tests
- `internal/caravan/wagon.go` — add `RunnerMobId = 359` constant + `FindRunnerInRoom` helper
- `internal/hooks/hooks.go` — register caravan runner-completion listener

**New files (caravan — Phase 3):**

- `internal/caravan/cargo_handoff.go` — `TransferCargoToRunner`, `TransferAllCargoBack`
- `internal/caravan/cargo_handoff_test.go`
- `internal/caravan/runner_completion_listener.go` — on `PatrolCompleted` for runner circuits, return cargo to wagon
- `internal/caravan/runner_completion_listener_test.go`
- `_datafiles/world/dogmud/patrols/thornwall_outskirts/thornwall_runner_circuit.yaml`
- `_datafiles/world/dogmud/patrols/thornwall_outskirts/stillwater_runner_circuit.yaml`

**Documentation (Phase 4):**

- `PATCH_NOTES.md` — dated 3.8 entry
- `MOB_ALIVENESS_ROADMAP.md` — mark 3.8 Done with shipped summary

---

## Phase 1 — Engine primitive (no behavior change)

### Task 1: `loop_shape: oneshot` parsing

**Files:**
- Modify: `internal/mobs/patrol.go:7-19` (Patrol struct)
- Modify: `internal/mobs/patrol_test.go` (add round-trip test)

- [ ] **Step 1: Write the failing test**

Add to `internal/mobs/patrol_test.go`:

```go
func TestPatrol_OneshotLoopShape_RoundTrips(t *testing.T) {
	p := &Patrol{
		Id:        "test_oneshot",
		LoopShape: "oneshot",
		Waypoints: []PatrolWaypoint{
			{Room: 100, DwellRounds: 3},
			{Room: 101, DwellRounds: 1},
		},
	}
	RegisterPatrolForTest(p)
	t.Cleanup(func() { UnregisterPatrolForTest("test_oneshot") })

	got := GetPatrol("test_oneshot")
	if got == nil {
		t.Fatal("patrol not registered")
	}
	if got.LoopShape != "oneshot" {
		t.Errorf("LoopShape = %q, want %q", got.LoopShape, "oneshot")
	}
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `go test ./internal/mobs/ -run TestPatrol_OneshotLoopShape_RoundTrips -v`
Expected: PASS (the struct already has `LoopShape string` — this test just verifies the new value is stored. No struct change needed yet.)

- [ ] **Step 3: Add validation note in the LoopShape field comment**

In `internal/mobs/patrol.go:7-13`, change:

```go
type Patrol struct {
	Id             string           `yaml:"id"`
	Description    string           `yaml:"description,omitempty"`
	LoopShape      string           `yaml:"loop_shape,omitempty"`        // "strict" (default) | "yo-yo"
	MaxPathRetries int              `yaml:"max_path_retries,omitempty"` // chunk 3.7: override global ScheduleMaxPathRetries; 0 = use global default
	Waypoints      []PatrolWaypoint `yaml:"waypoints"`
}
```

to:

```go
type Patrol struct {
	Id             string           `yaml:"id"`
	Description    string           `yaml:"description,omitempty"`
	LoopShape      string           `yaml:"loop_shape,omitempty"`        // "strict" (default) | "yo-yo" | "oneshot" (chunk 3.8)
	MaxPathRetries int              `yaml:"max_path_retries,omitempty"` // chunk 3.7: override global ScheduleMaxPathRetries; 0 = use global default
	Waypoints      []PatrolWaypoint `yaml:"waypoints"`
}
```

- [ ] **Step 4: Run the full mobs package tests**

Run: `go test ./internal/mobs/...`
Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/mobs/patrol.go internal/mobs/patrol_test.go
git commit -m "feat(patrol): document loop_shape: oneshot (3.8 schema prep)

Chunk 3.8 adds a third loop shape — oneshot patrols walk waypoints
once, emit events.PatrolCompleted on the final dwell, and clear
themselves. The Patrol struct's LoopShape field already accepts
arbitrary strings; this commit just documents the new accepted
value and adds a round-trip test. Executor support lands next.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Define `events.PatrolCompleted`

**Files:**
- Modify: `internal/events/eventtypes.go`

- [ ] **Step 1: Locate where to add the type**

Run: `grep -n 'PatrolWaypointArrival' internal/events/eventtypes.go`. Note that line — place the new event right after `PatrolWaypointArrival` to keep patrol events grouped.

- [ ] **Step 2: Add the event struct**

In `internal/events/eventtypes.go`, immediately after the `PatrolWaypointArrival` definition + `Type()` method, add:

```go
// PatrolCompleted fires once when a oneshot patrol exhausts its last
// waypoint's dwell. Consumers (outer state machines) read this to
// advance their own state. The patrol executor clears the mob's
// PatrolId before emitting, so the mob is back in the "no patrol"
// state when the listener sees the event. Chunk 3.8.
type PatrolCompleted struct {
	MobInstanceId int
	PatrolId      string
	RoomId        int
}

func (e PatrolCompleted) Type() string { return "PatrolCompleted" }
```

- [ ] **Step 3: Verify build**

Run: `go build ./internal/events/...`
Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/events/eventtypes.go
git commit -m "feat(events): define PatrolCompleted event type (3.8 prep)

Fires when a oneshot patrol exhausts its last waypoint's dwell.
No emitter yet; the patrol executor wires emission in a follow-up
commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: `StartOneshotPatrol` and `ClearOneshotPatrol` helpers

**Files:**
- Modify: `internal/mobs/patrol.go` (add helpers near GetPatrol)
- Modify: `internal/mobs/patrol_test.go` (new tests)

- [ ] **Step 1: Write the failing tests**

Add to `internal/mobs/patrol_test.go`:

```go
func TestStartOneshotPatrol_AssignsAndResetsMiscData(t *testing.T) {
	RegisterPatrolForTest(&Patrol{
		Id:        "test_oneshot_assign",
		LoopShape: "oneshot",
		Waypoints: []PatrolWaypoint{{Room: 100, DwellRounds: 2}},
	})
	t.Cleanup(func() { UnregisterPatrolForTest("test_oneshot_assign") })

	mob := &Mob{InstanceId: 90100}
	// Pre-seed dirty MiscData to verify reset.
	mob.Character.SetMiscData("patrol_waypoint_idx", 7)
	mob.Character.SetMiscData("patrol_direction", -1)
	mob.Character.SetMiscData("patrol_dwell_remaining", 99)
	mob.Character.SetMiscData("patrol_path_fail_count", 5)

	ok := StartOneshotPatrol(mob, "test_oneshot_assign")
	if !ok {
		t.Fatal("StartOneshotPatrol returned false for valid oneshot patrol")
	}
	if mob.PatrolId != "test_oneshot_assign" {
		t.Errorf("PatrolId = %q, want %q", mob.PatrolId, "test_oneshot_assign")
	}
	if v, _ := mob.Character.GetMiscData("patrol_waypoint_idx").(int); v != 0 {
		t.Errorf("patrol_waypoint_idx = %d, want 0", v)
	}
	if v, _ := mob.Character.GetMiscData("patrol_direction").(int); v != 1 {
		t.Errorf("patrol_direction = %d, want 1", v)
	}
	if v, _ := mob.Character.GetMiscData("patrol_dwell_remaining").(int); v != 0 {
		t.Errorf("patrol_dwell_remaining = %d, want 0", v)
	}
	if v, _ := mob.Character.GetMiscData("patrol_path_fail_count").(int); v != 0 {
		t.Errorf("patrol_path_fail_count = %d, want 0", v)
	}
}

func TestStartOneshotPatrol_RejectsNonOneshot(t *testing.T) {
	RegisterPatrolForTest(&Patrol{
		Id:        "test_strict_patrol",
		LoopShape: "strict",
		Waypoints: []PatrolWaypoint{{Room: 100, DwellRounds: 2}},
	})
	t.Cleanup(func() { UnregisterPatrolForTest("test_strict_patrol") })

	mob := &Mob{InstanceId: 90101}
	ok := StartOneshotPatrol(mob, "test_strict_patrol")
	if ok {
		t.Errorf("StartOneshotPatrol should return false for strict patrol, got true")
	}
	if mob.PatrolId != "" {
		t.Errorf("PatrolId should remain empty after rejection, got %q", mob.PatrolId)
	}
}

func TestStartOneshotPatrol_RejectsUnknownPatrol(t *testing.T) {
	mob := &Mob{InstanceId: 90102}
	ok := StartOneshotPatrol(mob, "does_not_exist")
	if ok {
		t.Errorf("StartOneshotPatrol should return false for unknown patrol, got true")
	}
}

func TestClearOneshotPatrol_ClearsAllPatrolMiscData(t *testing.T) {
	mob := &Mob{InstanceId: 90103, PatrolId: "test_patrol"}
	mob.Character.SetMiscData("patrol_waypoint_idx", 5)
	mob.Character.SetMiscData("patrol_direction", -1)
	mob.Character.SetMiscData("patrol_dwell_remaining", 3)
	mob.Character.SetMiscData("patrol_path_fail_count", 2)

	ClearOneshotPatrol(mob)

	if mob.PatrolId != "" {
		t.Errorf("PatrolId = %q, want empty", mob.PatrolId)
	}
	if v, _ := mob.Character.GetMiscData("patrol_waypoint_idx").(int); v != 0 {
		t.Errorf("patrol_waypoint_idx = %d, want 0", v)
	}
	if v, _ := mob.Character.GetMiscData("patrol_direction").(int); v != 0 {
		t.Errorf("patrol_direction = %d, want 0", v)
	}
	if v, _ := mob.Character.GetMiscData("patrol_dwell_remaining").(int); v != 0 {
		t.Errorf("patrol_dwell_remaining = %d, want 0", v)
	}
	if v, _ := mob.Character.GetMiscData("patrol_path_fail_count").(int); v != 0 {
		t.Errorf("patrol_path_fail_count = %d, want 0", v)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/mobs/ -run "TestStartOneshotPatrol|TestClearOneshotPatrol" -v`
Expected: FAIL — `StartOneshotPatrol` / `ClearOneshotPatrol` undefined.

- [ ] **Step 3: Implement the helpers**

In `internal/mobs/patrol.go`, add after the existing `GetPatrol` function:

```go
// StartOneshotPatrol assigns a oneshot patrol to a mob at runtime and
// resets the patrol MiscData so the executor begins from waypoint 0 on
// the next idle tick. Returns false if the patrol id doesn't resolve or
// isn't loop_shape: oneshot.
//
// Used by outer state machines (caravan arrival listener, forager
// forager_step btree action) to dispatch a sub-patrol when their
// routine reaches a delivery phase. The executor emits
// events.PatrolCompleted and calls ClearOneshotPatrol when the
// terminal waypoint's dwell expires. Chunk 3.8.
func StartOneshotPatrol(mob *Mob, patrolId string) bool {
	if mob == nil {
		return false
	}
	p := GetPatrol(patrolId)
	if p == nil || p.LoopShape != "oneshot" {
		return false
	}
	mob.PatrolId = patrolId
	mob.Character.SetMiscData("patrol_waypoint_idx", 0)
	mob.Character.SetMiscData("patrol_direction", 1)
	mob.Character.SetMiscData("patrol_dwell_remaining", 0)
	mob.Character.SetMiscData("patrol_path_fail_count", 0)
	return true
}

// ClearOneshotPatrol clears mob.PatrolId and resets the four patrol
// MiscData keys. Called by the executor on PatrolCompleted; also
// exposed for explicit cancellation (e.g., outer state machine aborts
// mid-circuit). Chunk 3.8.
func ClearOneshotPatrol(mob *Mob) {
	if mob == nil {
		return
	}
	mob.PatrolId = ""
	mob.Character.SetMiscData("patrol_waypoint_idx", 0)
	mob.Character.SetMiscData("patrol_direction", 0)
	mob.Character.SetMiscData("patrol_dwell_remaining", 0)
	mob.Character.SetMiscData("patrol_path_fail_count", 0)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/mobs/ -run "TestStartOneshotPatrol|TestClearOneshotPatrol" -v`
Expected: all PASS.

- [ ] **Step 5: Run full mobs suite**

Run: `go test ./internal/mobs/...`
Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/patrol.go internal/mobs/patrol_test.go
git commit -m "feat(patrol): StartOneshotPatrol + ClearOneshotPatrol runtime helpers (3.8)

Outer state machines (caravan listener, forager btree) use these
to dispatch a oneshot sub-patrol and tear it down. Start resets
the four patrol MiscData keys so the executor walks from waypoint 0.
Clear is called by the executor on PatrolCompleted; also callable
for explicit cancellation. Rejects non-oneshot or unknown patrols.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: `WantsComplete` executor branch

**Files:**
- Modify: `internal/hooks/NewRound_IdleMobs_patrol.go` (add new plan flag + branch)
- Modify: `internal/hooks/NewRound_IdleMobs_patrol_test.go` (add emission test)

- [ ] **Step 1: Read the patrol executor**

Read `internal/hooks/NewRound_IdleMobs_patrol.go` end-to-end. Note:
- The `patrolPlan` struct (top of file)
- `patrolTickPlan` — pure function computing the plan
- `applyPatrolPlan` — side-effecting consumer
- How `activePatrolId` is threaded (T4 of chunk 3.7 already added it as a parameter)

- [ ] **Step 2: Write the failing test**

Add to `internal/hooks/NewRound_IdleMobs_patrol_test.go`:

```go
func TestApplyPatrolPlan_OneshotTerminal_EmitsCompletedAndClears(t *testing.T) {
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:        "test_oneshot_complete",
		LoopShape: "oneshot",
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 100, DwellRounds: 1},
			{Room: 200, DwellRounds: 1},
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest("test_oneshot_complete") })

	events.DrainQueuedPatrolWaypointArrivalsForTest(0) // drain any leftover

	mob := &mobs.Mob{InstanceId: 9100, PatrolId: "test_oneshot_complete"}
	// At terminal waypoint (idx 1), dwell already expired.
	mob.Character.RoomId = 200
	mob.Character.SetMiscData("patrol_waypoint_idx", 1)
	mob.Character.SetMiscData("patrol_dwell_remaining", 0)

	plan := patrolTickPlan(mob, "test_oneshot_complete")
	if !plan.WantsComplete {
		t.Fatalf("expected WantsComplete=true at oneshot terminal, got %+v", plan)
	}
	applyPatrolPlan(mob, plan, "test_oneshot_complete")

	// PatrolId should be cleared.
	if mob.PatrolId != "" {
		t.Errorf("expected PatrolId cleared, got %q", mob.PatrolId)
	}

	// PatrolCompleted event should be queued exactly once for mob 9100.
	completed := events.DrainQueuedPatrolCompletedForTest(mob.InstanceId)
	if len(completed) != 1 {
		t.Fatalf("expected 1 PatrolCompleted event, got %d", len(completed))
	}
	if completed[0].PatrolId != "test_oneshot_complete" {
		t.Errorf("PatrolId in event = %q, want %q",
			completed[0].PatrolId, "test_oneshot_complete")
	}
	if completed[0].RoomId != 200 {
		t.Errorf("RoomId in event = %d, want 200", completed[0].RoomId)
	}
}

func TestApplyPatrolPlan_StrictPatrol_DoesNotComplete(t *testing.T) {
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:        "test_strict_no_complete",
		LoopShape: "strict",
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 100, DwellRounds: 1},
			{Room: 200, DwellRounds: 1},
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest("test_strict_no_complete") })

	mob := &mobs.Mob{InstanceId: 9101, PatrolId: "test_strict_no_complete"}
	mob.Character.RoomId = 200 // at last waypoint
	mob.Character.SetMiscData("patrol_waypoint_idx", 1)
	mob.Character.SetMiscData("patrol_dwell_remaining", 0)

	plan := patrolTickPlan(mob, "test_strict_no_complete")
	if plan.WantsComplete {
		t.Errorf("expected WantsComplete=false for strict patrol at last waypoint, got true")
	}
}
```

- [ ] **Step 3: Add `DrainQueuedPatrolCompletedForTest` to events package**

Look at the existing `DrainQueuedPatrolWaypointArrivalsForTest` in `internal/events/events.go` (added in chunk 3.7 Task 4). Copy that pattern.

In `internal/events/events.go`, add after the patrol-waypoint-arrival drain helper:

```go
// DrainQueuedPatrolCompletedForTest pops and returns every queued
// PatrolCompleted event whose MobInstanceId matches. Mirrors the
// drain pattern used by other test helpers in this file. Pass 0
// to drain all (for a global reset between tests). Chunk 3.8.
func DrainQueuedPatrolCompletedForTest(instanceId int) []PatrolCompleted {
	queueMu.Lock()
	defer queueMu.Unlock()
	var matched []PatrolCompleted
	var kept []Event
	for _, e := range queue {
		pc, ok := e.(PatrolCompleted)
		if !ok {
			kept = append(kept, e)
			continue
		}
		if instanceId == 0 || pc.MobInstanceId == instanceId {
			matched = append(matched, pc)
			continue
		}
		kept = append(kept, e)
	}
	queue = kept
	return matched
}
```

Verify the field names by reading the existing `DrainQueuedPatrolWaypointArrivalsForTest` and `queue` declaration first. Match exactly.

- [ ] **Step 4: Run tests to verify they fail**

Run: `go test ./internal/hooks/ -run "TestApplyPatrolPlan_Oneshot|TestApplyPatrolPlan_Strict" -v`
Expected: FAIL — `WantsComplete` field doesn't exist on `patrolPlan` yet.

- [ ] **Step 5: Add `WantsComplete` field to `patrolPlan`**

In `internal/hooks/NewRound_IdleMobs_patrol.go`, find the `patrolPlan` struct (top of file) and add a new field:

```go
type patrolPlan struct {
	HasPatrol         bool
	WantsDwellWait    bool
	WantsPath         bool
	TargetRoom        int
	WantsAdvance      bool
	NextWaypointIdx   int
	NextDirection     int
	NextDwellRounds   int
	WantsHomeFallback bool
	FailureMessage    string
	WantsComplete     bool // chunk 3.8: oneshot patrol reached its terminal waypoint
}
```

- [ ] **Step 6: Detect oneshot completion in `patrolTickPlan`**

In `patrolTickPlan`, after the "at target waypoint" early-out where existing code sets `WantsDwellWait` or `WantsAdvance`, add the oneshot-terminal check.

Find this section (will look roughly like):

```go
	if mob.Character.RoomId == currentWaypoint.Room {
		if dwellRemaining > 0 {
			plan.WantsDwellWait = true
			return plan
		}
		// Advance.
		nextIdx, nextDir := p.NextWaypoint(idx, dir)
		plan.WantsAdvance = true
		plan.NextWaypointIdx = nextIdx
		plan.NextDirection = nextDir
		plan.NextDwellRounds = p.Waypoints[nextIdx].DwellRounds
		return plan
	}
```

Insert a oneshot-terminal check BEFORE the `Advance` block:

```go
	if mob.Character.RoomId == currentWaypoint.Room {
		if dwellRemaining > 0 {
			plan.WantsDwellWait = true
			return plan
		}
		// Chunk 3.8: oneshot terminal — at the last waypoint of a
		// oneshot patrol, with dwell expired, signal completion
		// instead of advancing.
		if p.LoopShape == "oneshot" && idx == len(p.Waypoints)-1 {
			plan.WantsComplete = true
			return plan
		}
		// Advance.
		nextIdx, nextDir := p.NextWaypoint(idx, dir)
		plan.WantsAdvance = true
		plan.NextWaypointIdx = nextIdx
		plan.NextDirection = nextDir
		plan.NextDwellRounds = p.Waypoints[nextIdx].DwellRounds
		return plan
	}
```

- [ ] **Step 7: Handle `WantsComplete` in `applyPatrolPlan`**

In `applyPatrolPlan`, find the switch over plan flags. Add a new case at the top (before the existing branches):

```go
	switch {
	case plan.WantsComplete:
		events.AddToQueue(events.PatrolCompleted{
			MobInstanceId: mob.InstanceId,
			PatrolId:      activePatrolId,
			RoomId:        mob.Character.RoomId,
		})
		mobs.ClearOneshotPatrol(mob)
		return

	case plan.WantsHomeFallback:
		// ... existing code unchanged
```

- [ ] **Step 8: Run tests to verify they pass**

Run: `go test ./internal/hooks/ -run "TestApplyPatrolPlan_Oneshot|TestApplyPatrolPlan_Strict" -v`
Expected: both PASS.

- [ ] **Step 9: Run full hooks suite**

Run: `go test ./internal/hooks/...`
Expected: all tests pass (existing market beat + caravan main patrol behavior unaffected).

- [ ] **Step 10: Commit**

```bash
git add internal/hooks/NewRound_IdleMobs_patrol.go internal/hooks/NewRound_IdleMobs_patrol_test.go internal/events/events.go
git commit -m "feat(patrol): WantsComplete branch emits PatrolCompleted on oneshot terminal

When a oneshot patrol reaches its final waypoint and dwell expires,
the executor emits events.PatrolCompleted and calls
mobs.ClearOneshotPatrol. Strict/yo-yo patrols are unaffected
(WantsComplete only fires when loop_shape == oneshot AND idx ==
last waypoint).

Adds DrainQueuedPatrolCompletedForTest matching the existing patrol-
arrival drain helper.

No live consumers yet — caravan + forager wiring follows in
Phases 2 and 3.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Phase 2 — Forager migration

### Task 5: `DeliveryPatrolId` field on ForagerProfile

**Files:**
- Modify: `internal/forager/territory.go` (struct + Marsh/Steppe profiles)
- Modify: `internal/forager/territory_test.go` (if it exists — add field round-trip)

- [ ] **Step 1: Check if territory_test.go exists**

Run: `ls internal/forager/territory_test.go`. If it exists, you'll extend it. If not, the test goes in a new file (skip the test for this task; rely on the boot-load smoke).

- [ ] **Step 2: Add the field**

In `internal/forager/territory.go`, modify the `ForagerProfile` struct:

```go
type ForagerProfile struct {
	Kind             ForagerKind
	MobId            int
	Name             string
	SanctuaryRoom    int
	TerritoryRooms   []int
	PreyWhitelist    []int
	VendorRooms      []int
	MeetingRoom      int
	Buckets          []string
	DeliveryPatrolId string // chunk 3.8: oneshot patrol id for the Delivering state; empty = use legacy single-room handoff (Kessa)
}
```

- [ ] **Step 3: Populate for Marsh + Steppe**

In the `profiles` map, add `DeliveryPatrolId` to the two foragers with vendor circuits:

```go
var profiles = map[int]*ForagerProfile{
	371: { // Marsh Forager (Tova)
		Kind:             KindMarsh,
		MobId:            371,
		Name:             "Tova",
		SanctuaryRoom:    4123,
		TerritoryRooms:   []int{ /* unchanged */ },
		PreyWhitelist:    []int{ /* unchanged */ },
		VendorRooms:      []int{4102, 4103, 4105, 4106, 4125, 4126, 4135, 4143},
		Buckets:          []string{"stillwater", "base", "overlap"},
		DeliveryPatrolId: "marsh_forager_delivery", // chunk 3.8
	},
	372: { // Steppe Forager (Halix)
		Kind:             KindSteppe,
		MobId:            372,
		Name:             "Halix",
		SanctuaryRoom:    3040,
		TerritoryRooms:   []int{ /* unchanged */ },
		PreyWhitelist:    []int{ /* unchanged */ },
		VendorRooms:      []int{464, 470, 471, 475, 480, 481, 482, 483, 507},
		Buckets:          []string{"base", "overlap"},
		DeliveryPatrolId: "steppe_forager_delivery", // chunk 3.8
	},
	373: { // Fernway Forager (Kessa) — DeliveryPatrolId stays "" (single-room meeting handoff)
		Kind:           KindFernway,
		MobId:          373,
		Name:           "Kessa",
		SanctuaryRoom:  4197,
		TerritoryRooms: []int{ /* unchanged */ },
		PreyWhitelist:  []int{ /* unchanged */ },
		VendorRooms:    nil,
		MeetingRoom:    4038,
		Buckets:        []string{"fernway"},
		// DeliveryPatrolId intentionally empty — Kessa keeps her existing single-room handoff path.
	},
}
```

Preserve the existing `TerritoryRooms` and `PreyWhitelist` values verbatim — they're long; just don't change them.

- [ ] **Step 4: Verify build**

Run: `go build ./internal/forager/...`
Expected: clean build.

- [ ] **Step 5: Run forager tests**

Run: `go test ./internal/forager/...`
Expected: existing tests pass (new field is additive; default `""` for any tests that don't set it).

- [ ] **Step 6: Commit**

```bash
git add internal/forager/territory.go
git commit -m "feat(forager): DeliveryPatrolId on ForagerProfile (3.8 prep)

Marsh (Tova) and Steppe (Halix) get their oneshot delivery patrol
ids populated. Fernway (Kessa) intentionally empty — she keeps her
single-room meeting handoff at room 4038, which doesn't need a
sub-patrol.

Patrols themselves authored + listeners wired in follow-up
commits.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Author forager-delivery oneshot YAMLs

**Files:**
- Create: `_datafiles/world/dogmud/patrols/stillwater_marsh/marsh_forager_delivery.yaml`
- Create: `_datafiles/world/dogmud/patrols/ironwind_steppe/steppe_forager_delivery.yaml`

- [ ] **Step 1: Create the Marsh forager-delivery YAML**

Write `_datafiles/world/dogmud/patrols/stillwater_marsh/marsh_forager_delivery.yaml`:

```yaml
id: marsh_forager_delivery
description: "Tova's vendor circuit through Stillwater: deliver foraged goods, return to sanctuary. Chunk 3.8 oneshot."
loop_shape: oneshot
max_path_retries: 40
waypoints:
  - { room: 4102, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 4103, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 4105, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 4106, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 4125, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 4126, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 4135, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 4143, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 4123, dwell_rounds: 1, arrival_event: "" }
```

- [ ] **Step 2: Create the Steppe forager-delivery YAML**

Write `_datafiles/world/dogmud/patrols/ironwind_steppe/steppe_forager_delivery.yaml`:

```yaml
id: steppe_forager_delivery
description: "Halix's vendor circuit through Thornwall: deliver foraged goods, return to sanctuary. Chunk 3.8 oneshot."
loop_shape: oneshot
max_path_retries: 40
waypoints:
  - { room: 464, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 470, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 471, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 475, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 480, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 481, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 482, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 483, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 507, dwell_rounds: 3, arrival_event: forager_vendor }
  - { room: 3040, dwell_rounds: 1, arrival_event: "" }
```

- [ ] **Step 3: Boot test (validate patrols load)**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go build -o dogmud-t6.exe . && ./dogmud-t6.exe > t6-boot.log 2>&1 &
BOOT_PID=$!
sleep 15
kill $BOOT_PID 2>/dev/null
grep -E 'LoadPatrols|panic|FATAL|marsh_forager_delivery|steppe_forager_delivery' t6-boot.log | head -30
rm dogmud-t6.exe t6-boot.log
```

Expected:
- `mobs.LoadPatrols() loadedCount=4` (was 2 — added the 2 new patrols).
- 10 lines of `patrol waypoint patrol="marsh_forager_delivery"` (9 vendor stops + 1 sanctuary).
- 10 lines of `patrol waypoint patrol="steppe_forager_delivery"`.
- No panics.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/patrols/stillwater_marsh/marsh_forager_delivery.yaml \
        _datafiles/world/dogmud/patrols/ironwind_steppe/steppe_forager_delivery.yaml
git commit -m "feat(content): author marsh + steppe forager-delivery oneshot patrols (3.8)

Two oneshot patrols matching the vendor circuits in
ForagerProfile.VendorRooms for Tova (Marsh) and Halix (Steppe).
Each ends at the forager's sanctuary room with a 1-round dwell.

Loaded at boot but not yet referenced — wiring lands when the
arrival + completion listeners ship.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: Forager arrival listener

**Files:**
- Create: `internal/forager/arrival_listener.go`
- Create: `internal/forager/arrival_listener_test.go`
- Modify: `internal/hooks/hooks.go` (register listener)

- [ ] **Step 1: Find the existing SellToVendor signature**

Run: `grep -n 'func SellToVendor' internal/forager/`. Note the exact signature. The listener will call this for each `forager_vendor` arrival.

- [ ] **Step 2: Write the failing tests**

Create `internal/forager/arrival_listener_test.go`:

```go
package forager

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestForagerArrivalListener_IgnoresNonForagerVendorEvents(t *testing.T) {
	e := events.PatrolWaypointArrival{
		MobInstanceId: 12345,
		PatrolId:      "marsh_forager_delivery",
		ArrivalEvent:  "",
	}
	got := ForagerArrivalListener(e)
	if got != events.Continue {
		t.Errorf("expected events.Continue for non-forager_vendor event, got %v", got)
	}
}

func TestForagerArrivalListener_IgnoresUnknownMob(t *testing.T) {
	e := events.PatrolWaypointArrival{
		MobInstanceId: 999999, // doesn't exist
		PatrolId:      "marsh_forager_delivery",
		ArrivalEvent:  "forager_vendor",
		RoomId:        4102,
	}
	got := ForagerArrivalListener(e)
	if got != events.Continue {
		t.Errorf("expected events.Continue when mob missing, got %v", got)
	}
}

func TestForagerArrivalListener_IgnoresNonForagerMob(t *testing.T) {
	// Need a live mob instance that isn't a registered forager.
	// Use SeedMobsForTest to inject a fake mob.
	// (Adapt to whatever helper the package uses — match the
	//  pattern in internal/caravan/arrival_listener_test.go.)
	// ...this test is optional for the core path.
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/forager/ -run TestForagerArrivalListener -v`
Expected: FAIL — `ForagerArrivalListener` undefined.

- [ ] **Step 4: Implement the listener**

Create `internal/forager/arrival_listener.go`:

```go
package forager

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// ForagerArrivalListener consumes events.PatrolWaypointArrival for
// the forager-delivery oneshot patrols. When a forager arrives at a
// vendor waypoint (arrival_event: forager_vendor), fire the per-vendor
// sell handoff (SellToVendor) using items from the forager's own
// inventory.
//
// Non-forager_vendor events, non-forager mobs, and unknown instances
// are silently ignored — the listener is registered globally and
// fields every PatrolWaypointArrival regardless of source. Chunk 3.8.
func ForagerArrivalListener(e events.Event) events.ListenerReturn {
	pwa, ok := e.(events.PatrolWaypointArrival)
	if !ok {
		return events.Continue
	}
	if pwa.ArrivalEvent != "forager_vendor" {
		return events.Continue
	}
	forager := mobs.GetInstance(pwa.MobInstanceId)
	if forager == nil {
		return events.Continue
	}
	profile := ProfileFor(int(forager.MobId))
	if profile == nil {
		return events.Continue
	}
	SellToVendor(forager, pwa.RoomId, profile.Buckets)
	return events.Continue
}
```

If `SellToVendor`'s actual signature differs from `(mob, roomId, buckets)`, adapt to match the real signature.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/forager/ -run TestForagerArrivalListener -v`
Expected: PASS for the implemented cases.

- [ ] **Step 6: Register the listener in hooks.go**

Find the `RegisterListeners` function in `internal/hooks/hooks.go`. Add (alongside other event registrations):

```go
events.RegisterListener(events.PatrolWaypointArrival{}, forager.ForagerArrivalListener)
```

Add the import for `internal/forager` if not already present.

- [ ] **Step 7: Run full builds + tests**

Run: `go build ./... && go test ./internal/forager/... ./internal/hooks/...`
Expected: build clean, tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/forager/arrival_listener.go internal/forager/arrival_listener_test.go internal/hooks/hooks.go
git commit -m "feat(forager): ForagerArrivalListener consumes forager_vendor events (3.8)

When a forager reaches a vendor waypoint on a marsh_forager_delivery
or steppe_forager_delivery oneshot patrol, fire SellToVendor using
items from the forager's own inventory. Non-forager events ignored.

Listener registered globally. Dormant until forager_step btree
action starts assigning the oneshot patrols (next commit).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: Forager completion listener

**Files:**
- Create: `internal/forager/completion_listener.go`
- Create: `internal/forager/completion_listener_test.go`
- Modify: `internal/hooks/hooks.go` (register listener)

- [ ] **Step 1: Find how state transitions are read/written**

Run: `grep -n 'transitionForager\|keyForagerState\|forager_state' internal/behaviortree/actions_forager.go | head -20`. The listener needs to read+write the forager's BTreeState `forager_state` key to advance the state machine.

The pattern is: read current state from `BTreeState["forager_state"]`, then call `transitionForager(mobState, nextState)` to write the new state + reset the started-round counter.

But the completion listener lives in `internal/forager/` (not `internal/behaviortree/`), so it can't call `transitionForager` directly without an import cycle. Two options:

A. **Set BTreeState directly** — write the new state name + reset key into BTreeState from the listener. This duplicates `transitionForager`'s body but keeps the dependency direction clean.

B. **Move `transitionForager` into the forager package** — but the function operates on `*behaviortree.BehaviorState`, which is a behaviortree-package type. That'd require restructuring.

Option A is simpler. The listener does:

```go
bs, ok := mob.BTreeState.(behaviortree.BehaviorStater) // use the interface, not the concrete type
```

Wait — but `behaviortree.BehaviorState` is the actual storage. The listener in `internal/forager/` needs to import behaviortree to call methods on BehaviorState. That's a cycle if behaviortree already imports forager.

Run: `grep -n 'GoMudEngine/GoMud/internal/forager' internal/behaviortree/*.go`. If behaviortree imports forager, we can't do the reverse.

If there's a cycle, use option A via the `MobState` interface that BehaviorState satisfies. Look at how other cross-package listeners handle this.

**Take whichever approach the existing code supports without a cycle.** If both options fail, escalate.

- [ ] **Step 2: Write the failing tests**

Create `internal/forager/completion_listener_test.go`:

```go
package forager

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestForagerCompletionListener_IgnoresUnknownPatrolId(t *testing.T) {
	e := events.PatrolCompleted{
		MobInstanceId: 12345,
		PatrolId:      "thornwall_market_beat", // not a forager patrol
		RoomId:        100,
	}
	got := ForagerCompletionListener(e)
	if got != events.Continue {
		t.Errorf("expected events.Continue for unknown patrol, got %v", got)
	}
}

func TestForagerCompletionListener_IgnoresUnknownMob(t *testing.T) {
	e := events.PatrolCompleted{
		MobInstanceId: 999999,
		PatrolId:      "marsh_forager_delivery",
		RoomId:        4123,
	}
	got := ForagerCompletionListener(e)
	if got != events.Continue {
		t.Errorf("expected events.Continue when mob missing, got %v", got)
	}
}
```

Plus a state-transition test using a seeded mob (mirror the pattern from existing forager tests).

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/forager/ -run TestForagerCompletionListener -v`
Expected: FAIL — `ForagerCompletionListener` undefined.

- [ ] **Step 4: Implement the listener**

Create `internal/forager/completion_listener.go`:

```go
package forager

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// foragerDeliveryPatrols is the set of patrol ids that, when completed,
// should advance a forager's StateDelivering to the next state.
var foragerDeliveryPatrols = map[string]struct{}{
	"marsh_forager_delivery":  {},
	"steppe_forager_delivery": {},
}

// ForagerCompletionListener consumes events.PatrolCompleted for the
// forager-delivery oneshot patrols. When the patrol finishes (forager
// is back at sanctuary), advance the forager's state machine from
// StateDelivering to StateStoring (if storage_chest_room configured)
// or StateRecalling.
//
// Non-forager patrols, unknown mobs, and unknown patrol ids are
// silently ignored. Chunk 3.8.
func ForagerCompletionListener(e events.Event) events.ListenerReturn {
	pc, ok := e.(events.PatrolCompleted)
	if !ok {
		return events.Continue
	}
	if _, isForagerPatrol := foragerDeliveryPatrols[pc.PatrolId]; !isForagerPatrol {
		return events.Continue
	}
	forager := mobs.GetInstance(pc.MobInstanceId)
	if forager == nil {
		return events.Continue
	}

	// Decide next state: Storing if forager has a chest AND carries
	// unsold items, otherwise Recalling.
	nextState := StateRecalling
	if forager.StorageChestRoom > 0 && len(forager.Character.Items) > 0 {
		nextState = StateStoring
	}

	// Write directly to BTreeState rather than calling
	// behaviortree.transitionForager (avoids import cycle).
	// The forager_step btree action will pick up the new state on
	// its next tick.
	if bs := forager.BTreeStateString(); bs != nil {
		bs.Set("forager_state", nextState.Name())
		bs.Set("forager_state_started_round", strconv.FormatUint(util.GetRoundCount(), 10))
		bs.Set("forager_visit_index", "0") // reset legacy counter for safety
	}

	return events.Continue
}
```

If `BTreeStateString()` isn't the right accessor, find the actual method used by other listeners that mutate BTreeState — match it. The key names (`forager_state`, `forager_state_started_round`, `forager_visit_index`) match `keyForagerState` etc. in `actions_forager.go`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/forager/ -run TestForagerCompletionListener -v`
Expected: PASS.

- [ ] **Step 6: Register the listener in hooks.go**

Add to `RegisterListeners`:

```go
events.RegisterListener(events.PatrolCompleted{}, forager.ForagerCompletionListener)
```

- [ ] **Step 7: Run full tests**

Run: `go test ./internal/forager/... ./internal/hooks/...`
Expected: all pass.

- [ ] **Step 8: Commit**

```bash
git add internal/forager/completion_listener.go internal/forager/completion_listener_test.go internal/hooks/hooks.go
git commit -m "feat(forager): ForagerCompletionListener advances state on PatrolCompleted

When a forager-delivery oneshot patrol finishes, advance the
forager's BTreeState[forager_state] from delivering to either
storing (chest + unsold items) or recalling. Listener writes
BTreeState directly to avoid an internal/forager <-> internal/
behaviortree import cycle. Chunk 3.8.

Listener registered globally. forager_step rewrite in next
commit will start producing the events this listener consumes.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: Rewrite `tickForagerDeliveringTown` + sanctuary fallback

**Files:**
- Modify: `internal/behaviortree/actions_forager.go` (the `tickForagerDeliveringTown` function)

- [ ] **Step 1: Read the current implementation**

Read `internal/behaviortree/actions_forager.go` lines 252-288 (`tickForagerDelivering` + `tickForagerDeliveringTown`). Note the existing `keyVisitIndex` counter, the `pathto target` per vendor logic, and the `npcVisitVendorsInRoom` call.

- [ ] **Step 2: Replace `tickForagerDeliveringTown` with the oneshot dispatch**

Replace the existing `tickForagerDeliveringTown` function with:

```go
func tickForagerDeliveringTown(
	p *forager.ForagerProfile,
	mob *mobs.Mob,
	ctx *EvalContext,
) Result {
	// Chunk 3.8 5.4 sanctuary-fallback safety: if the forager has
	// somehow ended up at the sanctuary while StateDelivering with
	// no active oneshot patrol (e.g., patrol home-fallback fired
	// and never produced a PatrolCompleted), advance state
	// directly. Cargo, if any, carries through to Storing or
	// Recalling.
	if mob.Character.RoomId == p.SanctuaryRoom && mob.PatrolId == "" {
		if mob.StorageChestRoom > 0 && len(mob.Character.Items) > 0 {
			ctx.MobState.Set(keyStoringTurns, "0")
			transitionForager(ctx.MobState, forager.StateStoring)
		} else {
			transitionForager(ctx.MobState, forager.StateRecalling)
		}
		return Success
	}

	// If a oneshot delivery patrol is already running, the executor
	// will drive movement and the arrival listener will fire vendor
	// sells. Nothing for this tick to do.
	if mob.PatrolId == p.DeliveryPatrolId && mob.PatrolId != "" {
		return Success
	}

	// First entry to StateDelivering — start the oneshot patrol.
	if p.DeliveryPatrolId == "" {
		// Defensive: shouldn't happen for KindMarsh/KindSteppe since
		// territory.go populates these. Fall through to Recalling.
		transitionForager(ctx.MobState, forager.StateRecalling)
		return Success
	}
	if !mobs.StartOneshotPatrol(mob, p.DeliveryPatrolId) {
		// Patrol id didn't resolve or isn't oneshot — log + give up.
		transitionForager(ctx.MobState, forager.StateRecalling)
		return Success
	}
	return Success
}
```

The `keyVisitIndex` counter is no longer used by this path — the patrol layer drives waypoint progression. The `npcVisitVendorsInRoom` call is now driven from `internal/forager/arrival_listener.go` instead.

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 4: Run forager + behaviortree tests**

Run: `go test ./internal/forager/... ./internal/behaviortree/...`
Expected: all pass. Some existing forager-delivering tests may need adjustment — they likely assert the old `keyVisitIndex` behavior. Update them to the new "start patrol, exit Success" expectation, OR delete them if they tested deleted behavior.

- [ ] **Step 5: Commit**

```bash
git add internal/behaviortree/actions_forager.go
git commit -m "feat(forager): StateDelivering delegates to oneshot patrol (3.8)

tickForagerDeliveringTown shrinks from a per-vendor walk loop to:
1. Sanctuary-fallback safety (5.4): if at sanctuary with no active
   patrol, advance state directly.
2. If a oneshot delivery patrol is already running, return Success
   (the executor + arrival listener handle the work).
3. Otherwise start the oneshot patrol via mobs.StartOneshotPatrol.

The deleted body owned: per-vendor pathto loop, keyVisitIndex
counter, npcVisitVendorsInRoom call. Movement is now patrol-layer;
vendor sells fire from arrival_listener.go.

KindFernway (Kessa) still routes through tickForagerDeliveringFernway
unchanged — she has no DeliveryPatrolId and uses the meeting-room
sealed crate handoff.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: Forager Phase 2 boot + smoke

**Files:** (no code changes — validation only)

- [ ] **Step 1: Wipe instance saves per the smoke SOP**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

- [ ] **Step 2: Build + boot**

```bash
go build -o dogmud-p2.exe . && ./dogmud-p2.exe > p2-boot.log 2>&1 &
BOOT_PID=$!
sleep 15
kill $BOOT_PID 2>/dev/null
grep -E 'panic|FATAL|LoadPatrols|marsh_forager_delivery|steppe_forager_delivery' p2-boot.log | head -20
rm dogmud-p2.exe p2-boot.log
```

Expected:
- `mobs.LoadPatrols() loadedCount=4`
- No panics.
- 10 lines per forager patrol showing waypoint zone resolutions.

- [ ] **Step 3: Hand off to user for in-game smoke**

Smoke checklist (user runs in another session):

1. Find Tova or Halix in their territory. Watch them wander+forage until carry-cap triggers StateTravelingToDropoff.
2. They reach the first vendor room. Verify trade flavor fires (`<ansi fg="yellow">...</ansi>`).
3. They walk the 8-9 vendor rooms in sequence. Each fires trade flavor.
4. They arrive at their sanctuary (4123 for Tova, 3040 for Halix). Verify the BTreeState transitions to either `storing` or `recalling`.
5. Watch the cycle through StateRecalling + StateResting. Cycle restarts after rest.
6. `/admin/economy/` dashboard: forager card shows `delivering` → `storing`/`recalling` state names per the existing dashboard plumbing.

If any step fails, escalate before continuing to Phase 3.

- [ ] **Step 4: No commit (validation task)**

Phase 2 is shippable as-is once smoke confirms. The remaining phases don't depend on Phase 2 smoke success structurally — they can land in parallel branches — but per project SOP, smoke first.

---

## Phase 3 — Caravan runner-delivery flip

### Task 11: Lars (mob 359) YAML edits

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/359-lars.yaml`

- [ ] **Step 1: Read the current YAML**

Read `_datafiles/world/dogmud/mobs/thornwall_city/359-lars.yaml`. Note current values, especially `strength.training: 18`.

- [ ] **Step 2: Bump strength + add runner group**

Edit the YAML. Three changes:

1. Add `runner` to the groups list:

```yaml
groups:
  - caravan
  - merchant_train
  - runner
```

2. Bump strength training. The wagon (mob 374) carries roughly a smith's-shop-worth of cargo each leg. Target: Lars carries comparable. With `CarryCapacityMultiplier = 0.65`, strength 60 → 39 lb capacity, strength 90 → ~58 lb. Start with **60**, tune at smoke time:

```yaml
  stats:
    strength:
      training: 60
    dexterity:
      training: 30
    vitality:
      training: 18
    perception:
      training: 25
    willpower:
      training: 12
    charisma:
      training: 10
```

3. Update the description to establish Lars as Ketil's son canonically:

```yaml
character:
  name: Lars
  description: |
    Ketil's son — lean and sharp-eyed, in dark traveling leathers, a
    hunting bow over one shoulder and a quiver of fletched arrows at
    his hip. A long knife rides on his belt for close work. He has
    the still, economical stance of someone who learned early that
    small motions waste less energy than large ones. He shoulders the
    runs between caravan and shopkeepers when the wagon is parked,
    moving with the practiced rhythm of a kid who grew up on the
    trade road.
```

(Leave equipment/spells/tactics/skills unchanged.)

- [ ] **Step 3: Verify boot still loads**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go build -o dogmud-t11.exe . && ./dogmud-t11.exe > t11-boot.log 2>&1 &
BOOT_PID=$!
sleep 12
kill $BOOT_PID 2>/dev/null
grep -E 'panic|FATAL|mobs.LoadDataFiles' t11-boot.log | head -5
rm dogmud-t11.exe t11-boot.log
```

Expected: clean load, `loadedCount=226`.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/mobs/thornwall_city/359-lars.yaml
git commit -m "content(lars): strength bump + Ketil's-son flavor + runner group (3.8 prep)

Strength training 18 -> 60 so Lars can carry a wagon-load worth of
cargo on his depot-to-vendor circuit. Description updated to make
his relation to Ketil canon (was implicit, now explicit). Groups
gain 'runner' tag for future filtering.

Cargo handoff + runner-circuit dispatch lands in follow-up commits.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 12: Author caravan runner-circuit YAMLs

**Files:**
- Create: `_datafiles/world/dogmud/patrols/thornwall_outskirts/thornwall_runner_circuit.yaml`
- Create: `_datafiles/world/dogmud/patrols/thornwall_outskirts/stillwater_runner_circuit.yaml`

- [ ] **Step 1: Create the Thornwall runner-circuit YAML**

Write `_datafiles/world/dogmud/patrols/thornwall_outskirts/thornwall_runner_circuit.yaml`:

```yaml
id: thornwall_runner_circuit
description: "Lars's vendor-delivery circuit in Thornwall — wagon parks at the depot, Lars runs the goods out. Chunk 3.8 oneshot."
loop_shape: oneshot
max_path_retries: 40
waypoints:
  - { room: 464, dwell_rounds: 3, arrival_event: caravan_vendor }
  - { room: 470, dwell_rounds: 3, arrival_event: caravan_vendor }
  - { room: 471, dwell_rounds: 3, arrival_event: caravan_vendor }
  - { room: 475, dwell_rounds: 3, arrival_event: caravan_vendor }
  - { room: 480, dwell_rounds: 3, arrival_event: caravan_vendor }
  - { room: 481, dwell_rounds: 3, arrival_event: caravan_vendor }
  - { room: 482, dwell_rounds: 3, arrival_event: caravan_vendor }
  - { room: 483, dwell_rounds: 3, arrival_event: caravan_vendor }
  - { room: 465, dwell_rounds: 1, arrival_event: "" }
```

- [ ] **Step 2: Create the Stillwater runner-circuit YAML**

Write `_datafiles/world/dogmud/patrols/thornwall_outskirts/stillwater_runner_circuit.yaml`:

```yaml
id: stillwater_runner_circuit
description: "Lars's vendor-delivery circuit in Stillwater — wagon parks at the depot, Lars runs the goods out. Chunk 3.8 oneshot."
loop_shape: oneshot
max_path_retries: 40
waypoints:
  - { room: 4102, dwell_rounds: 3, arrival_event: caravan_vendor }
  - { room: 4103, dwell_rounds: 3, arrival_event: caravan_vendor }
  - { room: 4105, dwell_rounds: 3, arrival_event: caravan_vendor }
  - { room: 4106, dwell_rounds: 3, arrival_event: caravan_vendor }
  - { room: 4125, dwell_rounds: 3, arrival_event: caravan_vendor }
  - { room: 4126, dwell_rounds: 3, arrival_event: caravan_vendor }
  - { room: 4135, dwell_rounds: 3, arrival_event: caravan_vendor }
  - { room: 4143, dwell_rounds: 3, arrival_event: caravan_vendor }
  - { room: 4109, dwell_rounds: 1, arrival_event: "" }
```

- [ ] **Step 3: Boot test**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go build -o dogmud-t12.exe . && ./dogmud-t12.exe > t12-boot.log 2>&1 &
BOOT_PID=$!
sleep 15
kill $BOOT_PID 2>/dev/null
grep -E 'LoadPatrols|panic|thornwall_runner_circuit|stillwater_runner_circuit' t12-boot.log | head -10
rm dogmud-t12.exe t12-boot.log
```

Expected: `loadedCount=6` (4 from before + 2 new). 10 waypoint log lines per new patrol. No panics.

- [ ] **Step 4: Commit**

```bash
git add _datafiles/world/dogmud/patrols/thornwall_outskirts/thornwall_runner_circuit.yaml \
        _datafiles/world/dogmud/patrols/thornwall_outskirts/stillwater_runner_circuit.yaml
git commit -m "feat(content): author Thornwall + Stillwater runner-circuit YAMLs (3.8)

Two oneshot patrols matching the existing vendor lists. Each ends
at its depot (terminal wp). 3-round dwell per vendor matches the
forager-delivery cadence; final 1-round terminal is just the
arrival marker before PatrolCompleted fires.

Loaded at boot but not yet assigned to any mob. Lars wiring lands
with the caravan atomic flip in T15.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 13: Cargo handoff helpers

**Files:**
- Create: `internal/caravan/cargo_handoff.go`
- Create: `internal/caravan/cargo_handoff_test.go`
- Modify: `internal/caravan/wagon.go` (add `RunnerMobId` constant + `FindRunnerInRoom` helper)

- [ ] **Step 1: Add RunnerMobId + FindRunnerInRoom**

In `internal/caravan/wagon.go`, near the existing `WagonMobId` / `LeaderMobId` constants, add:

```go
// RunnerMobId is the mob template ID of the caravan runner — Ketil's
// son Lars, who carries cargo wagon ↔ vendor during depot stops.
// Chunk 3.8.
const RunnerMobId = 359
```

And add to `caravanMobIds` (the existing set):

Search for the existing `caravanMobIds` map; it should already include 359. If not (or if it was a `var` set instead), add it.

Run: `grep -n 'caravanMobIds' internal/caravan/wagon.go`. The set was authored in 3.7 and likely already has 359 (since Lars is part of the crew). If yes, no change needed. If no, add it.

Add `FindRunnerInRoom`:

```go
// FindRunnerInRoom returns the runner mob (RunnerMobId) co-located in
// the given room, or nil if Lars is not present. Mirrors
// FindWagonInRoom. Chunk 3.8.
func FindRunnerInRoom(roomId int) *mobs.Mob {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return nil
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		if int(m.MobId) == RunnerMobId {
			return m
		}
	}
	return nil
}
```

- [ ] **Step 2: Write the failing tests**

Create `internal/caravan/cargo_handoff_test.go`:

```go
package caravan

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func newCargoTestMob(t *testing.T, instId, mobId int, name string) *mobs.Mob {
	t.Helper()
	m := &mobs.Mob{
		MobId:      mobs.MobId(mobId),
		InstanceId: instId,
	}
	m.Character.Name = name
	m.Character.Buffs = buffs.New()
	return m
}

func TestTransferCargoToRunner_MovesMatchingBucketItems(t *testing.T) {
	wagon := newCargoTestMob(t, 80374, WagonMobId, "wagon")
	runner := newCargoTestMob(t, 80359, RunnerMobId, "Lars")

	// Wagon has 3 items. Two match outbound buckets, one doesn't.
	wagon.Character.Items = []items.Item{
		// (real items.Item construction — adapt to the constructor
		//  that exists. Bucket assignment uses economy.BucketFor;
		//  test should use items whose bucket is deterministic.)
	}
	// ... seed wagon items per the testing helpers your codebase has.

	moved := TransferCargoToRunner(wagon, runner, []string{"thornwall", "fernway"})
	if moved == 0 {
		t.Errorf("expected at least one item moved; got 0")
	}

	// Assert wagon lost the matching items.
	// Assert runner gained them.
}

func TestTransferAllCargoBack_EmptiesRunner(t *testing.T) {
	wagon := newCargoTestMob(t, 80375, WagonMobId, "wagon")
	runner := newCargoTestMob(t, 80360, RunnerMobId, "Lars")

	// Runner has items.
	// (seed runner items per testing helpers)

	preCount := len(runner.Character.Items)
	moved := TransferAllCargoBack(runner, wagon)
	if moved != preCount {
		t.Errorf("expected %d items moved back, got %d", preCount, moved)
	}
	if len(runner.Character.Items) != 0 {
		t.Errorf("runner should be empty after TransferAllCargoBack, has %d items",
			len(runner.Character.Items))
	}
}
```

If items.Item is hard to construct in tests (real items require an item-spec registry), look at how `visit_test.go` seeds mock items — match that pattern. If the pattern doesn't exist, the tests above are sketches; the implementer should write minimal-but-real tests in the same style as the existing visit tests, or escalate.

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/caravan/ -run "TestTransferCargoToRunner|TestTransferAllCargoBack" -v`
Expected: FAIL — functions undefined.

- [ ] **Step 4: Implement the helpers**

Create `internal/caravan/cargo_handoff.go`:

```go
package caravan

import (
	"slices"

	"github.com/GoMudEngine/GoMud/internal/economy"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// TransferCargoToRunner moves all items from the wagon's inventory
// whose bucket (per economy.BucketFor) is in the outboundBuckets list
// to the runner's inventory. Returns the number of items moved.
// Used at depot arrival in the caravan arrival listener. Chunk 3.8.
func TransferCargoToRunner(wagon, runner *mobs.Mob, outboundBuckets []string) int {
	if wagon == nil || runner == nil || len(outboundBuckets) == 0 {
		return 0
	}
	moved := 0
	// Iterate in reverse so RemoveItem is index-safe.
	for i := len(wagon.Character.Items) - 1; i >= 0; i-- {
		item := wagon.Character.Items[i]
		bucket := economy.BucketFor(item.ItemId)
		if bucket == "" || !slices.Contains(outboundBuckets, bucket) {
			continue
		}
		if !runner.Character.StoreItem(item) {
			break // runner at carry cap; stop transferring
		}
		wagon.Character.RemoveItem(item)
		moved++
	}
	return moved
}

// TransferAllCargoBack moves every item from the runner's inventory
// back to the wagon. No bucket filtering — what didn't sell goes home.
// Called on PatrolCompleted by the runner-completion listener.
// Returns the number of items moved. Chunk 3.8.
func TransferAllCargoBack(runner, wagon *mobs.Mob) int {
	if runner == nil || wagon == nil {
		return 0
	}
	moved := 0
	for i := len(runner.Character.Items) - 1; i >= 0; i-- {
		item := runner.Character.Items[i]
		if !wagon.Character.StoreItem(item) {
			break // wagon at carry cap; stop transferring
		}
		runner.Character.RemoveItem(item)
		moved++
	}
	return moved
}
```

If `economy.BucketFor` doesn't exist by that name, find the actual function used in `internal/caravan/visit.go` (it was lifted from there in 3.7) and use the same call.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/caravan/ -run "TestTransferCargoToRunner|TestTransferAllCargoBack" -v`
Expected: PASS.

- [ ] **Step 6: Run the full caravan suite**

Run: `go test ./internal/caravan/...`
Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add internal/caravan/cargo_handoff.go internal/caravan/cargo_handoff_test.go internal/caravan/wagon.go
git commit -m "feat(caravan): cargo-handoff helpers + RunnerMobId constant (3.8)

- TransferCargoToRunner: bucket-filtered wagon -> runner move; mirrors
  VisitVendorsInRoom's filter shape.
- TransferAllCargoBack: unfiltered runner -> wagon empty-out, called
  on PatrolCompleted.
- RunnerMobId = 359 constant + FindRunnerInRoom helper, siblings to
  the existing WagonMobId / FindWagonInRoom plumbing.

No live callers yet — caravan atomic flip in T15 wires these into
the arrival listener.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 14: Runner completion listener

**Files:**
- Create: `internal/caravan/runner_completion_listener.go`
- Create: `internal/caravan/runner_completion_listener_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/caravan/runner_completion_listener_test.go`:

```go
package caravan

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestCaravanRunnerCompletionListener_IgnoresUnknownPatrol(t *testing.T) {
	e := events.PatrolCompleted{
		MobInstanceId: 12345,
		PatrolId:      "thornwall_market_beat",
		RoomId:        100,
	}
	got := CaravanRunnerCompletionListener(e)
	if got != events.Continue {
		t.Errorf("expected events.Continue for unknown patrol, got %v", got)
	}
}

func TestCaravanRunnerCompletionListener_IgnoresMissingRunner(t *testing.T) {
	e := events.PatrolCompleted{
		MobInstanceId: 999999,
		PatrolId:      "thornwall_runner_circuit",
		RoomId:        465,
	}
	got := CaravanRunnerCompletionListener(e)
	if got != events.Continue {
		t.Errorf("expected events.Continue when runner missing, got %v", got)
	}
}
```

(A test that exercises the actual cargo-back transfer requires the full mob + room fixture pattern. Defer to the smoke if the unit-test fixture is too heavy.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/caravan/ -run TestCaravanRunnerCompletionListener -v`
Expected: FAIL — listener undefined.

- [ ] **Step 3: Implement the listener**

Create `internal/caravan/runner_completion_listener.go`:

```go
package caravan

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// runnerCircuitPatrols is the set of patrol ids that, when completed,
// trigger the runner -> wagon cargo return.
var runnerCircuitPatrols = map[string]struct{}{
	"thornwall_runner_circuit":  {},
	"stillwater_runner_circuit": {},
}

// CaravanRunnerCompletionListener consumes events.PatrolCompleted for
// the runner-circuit oneshot patrols. When Lars finishes his circuit
// (back at the depot terminal waypoint), transfer any residual cargo
// from his inventory back to the wagon.
//
// Non-runner patrols, missing instances, and runner-without-wagon
// cases are silently ignored. Chunk 3.8.
func CaravanRunnerCompletionListener(e events.Event) events.ListenerReturn {
	pc, ok := e.(events.PatrolCompleted)
	if !ok {
		return events.Continue
	}
	if _, isRunnerCircuit := runnerCircuitPatrols[pc.PatrolId]; !isRunnerCircuit {
		return events.Continue
	}
	runner := mobs.GetInstance(pc.MobInstanceId)
	if runner == nil || int(runner.MobId) != RunnerMobId {
		return events.Continue
	}
	wagon := FindWagonInRoom(runner.Character.RoomId)
	if wagon == nil {
		// Wagon not co-located — caravan already departed or some
		// other anomaly. Cargo stays on Lars; the 5.3 depot-arrival
		// safety in handleDepotArrival will catch this on the next
		// caravan depot visit.
		return events.Continue
	}
	TransferAllCargoBack(runner, wagon)
	return events.Continue
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/caravan/ -run TestCaravanRunnerCompletionListener -v`
Expected: PASS.

- [ ] **Step 5: Commit (listener file only — registration lands in T15)**

```bash
git add internal/caravan/runner_completion_listener.go internal/caravan/runner_completion_listener_test.go
git commit -m "feat(caravan): runner-completion listener returns residual cargo to wagon

On PatrolCompleted for thornwall_runner_circuit or
stillwater_runner_circuit, find the wagon co-located with Lars and
transfer his remaining inventory back. If wagon is not co-located
(caravan departed early, etc.), no-op — the 5.3 stranded-cargo
safety in handleDepotArrival catches that case on next visit.

Listener is compiled but not yet registered. Registration + the
arrival-listener dispatch changes land atomically in T15.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 15: Atomic caravan flip

**Files (atomic commit):**
- Modify: `internal/caravan/arrival_listener.go` (dispatch + 5.3 safety + vendor handler source swap)
- Modify: `internal/caravan/synthesize_state.go` (4-waypoint mapping + Lars-state check)
- Modify: `internal/caravan/synthesize_state_test.go` (update fixture)
- Modify: `_datafiles/world/dogmud/patrols/thornwall_outskirts/caravan_thornwall_stillwater.yaml` (22 → 4 waypoints)
- Modify: `internal/hooks/hooks.go` (register `CaravanRunnerCompletionListener`)

This is the single high-risk atomic change. All pieces land in one commit because the listener changes assume the truncated YAML, the synthesizer changes assume Lars's patrol-state is readable, and the YAML truncation changes the dashboard's pre-existing waypoint-to-state mapping.

- [ ] **Step 1: Truncate the caravan main YAML**

Replace `_datafiles/world/dogmud/patrols/thornwall_outskirts/caravan_thornwall_stillwater.yaml` with:

```yaml
id: caravan_thornwall_stillwater
description: "Ketil's caravan crew: Thornwall depot → Stillwater depot → back. Chunk 3.8: vendor stops are now Lars's runner sub-patrols, not main-route waypoints."
loop_shape: strict
max_path_retries: 40
waypoints:
  # ── wp0: Thornwall depot — long dwell; Lars runs Thornwall vendor circuit ──
  - room: 465
    dwell_rounds: 360
    arrival_event: caravan_depot

  # ── wp1: Outbound Fernway pickup (forager handoff) ────────────────────────
  - room: 4038
    dwell_rounds: 8
    arrival_event: caravan_fernway_pickup

  # ── wp2: Stillwater depot — medium dwell; Lars runs Stillwater circuit ────
  - room: 4109
    dwell_rounds: 180
    arrival_event: caravan_depot

  # ── wp3: Inbound Fernway pickup ───────────────────────────────────────────
  - room: 4038
    dwell_rounds: 8
    arrival_event: caravan_fernway_pickup

  # loops back to wp0
```

- [ ] **Step 2: Update the caravan arrival listener**

In `internal/caravan/arrival_listener.go`:

Find `handleDepotArrival`. Replace the existing body with:

```go
func handleDepotArrival(leader *mobs.Mob, arrival events.PatrolWaypointArrival) {
	// Chunk 3.8: at wp0 (Thornwall, 360 dwell) and wp2 (Stillwater,
	// 180 dwell), kick off Lars's runner circuit for that town.
	switch arrival.WaypointIdx {
	case 0:
		startRunnerCircuit(leader, arrival, "thornwall_runner_circuit", []string{"stillwater", "fernway"})
	case 2:
		startRunnerCircuit(leader, arrival, "stillwater_runner_circuit", []string{"thornwall", "fernway"})
	}

	// Chunk 3.8 5.3 stranded-cargo safety: if Lars is in the depot
	// room carrying cargo (e.g., home-fallback from a stranded
	// circuit), transfer it back to the wagon now.
	lars := FindRunnerInRoom(leader.Character.RoomId)
	wagon := FindWagonInRoom(leader.Character.RoomId)
	if lars != nil && wagon != nil && len(lars.Character.Items) > 0 && lars.PatrolId == "" {
		TransferAllCargoBack(lars, wagon)
	}
}

// startRunnerCircuit transfers outbound-bucket cargo from wagon -> Lars
// and assigns him the oneshot runner-circuit patrol. No-op if Lars is
// not in the depot, the wagon is not in the depot, or Lars already has
// a patrol assigned.
func startRunnerCircuit(leader *mobs.Mob, arrival events.PatrolWaypointArrival, circuitPatrolId string, outboundBuckets []string) {
	lars := FindRunnerInRoom(arrival.RoomId)
	if lars == nil {
		mudlog.Warn("caravan depot without runner",
			"leader", leader.Character.Name,
			"room", arrival.RoomId,
			"circuit", circuitPatrolId,
		)
		return
	}
	if lars.PatrolId != "" {
		// Already on a circuit — don't double-start.
		return
	}
	wagon := FindWagonInRoom(arrival.RoomId)
	if wagon == nil {
		mudlog.Warn("caravan depot without wagon",
			"leader", leader.Character.Name,
			"room", arrival.RoomId,
		)
		return
	}
	TransferCargoToRunner(wagon, lars, outboundBuckets)
	mobs.StartOneshotPatrol(lars, circuitPatrolId)
}
```

Find `handleVendorArrival`. Replace its body so the source-mob is now Lars (not the wagon):

```go
func handleVendorArrival(leader *mobs.Mob, arrival events.PatrolWaypointArrival) {
	// Chunk 3.8: the source mob is Lars (runner), not the wagon.
	// The wagon is parked back at the depot. Find Lars by his template
	// id; he's expected to be co-located at this vendor waypoint
	// because the event fired off his oneshot patrol.
	lars := FindRunnerInRoom(arrival.RoomId)
	if lars == nil {
		mudlog.Warn("caravan vendor stop without runner",
			"leader", leader.Character.Name,
			"room", arrival.RoomId,
		)
		return
	}

	deliveryBuckets, pickupBuckets := bucketsForWaypointIdx(arrival.WaypointIdx, arrival.PatrolId)
	delivered, pickedUp := VisitVendorsInRoom(arrival.RoomId, lars, deliveryBuckets, pickupBuckets)
	msg := FormatVisitMessage(delivered, pickedUp)
	if msg == "" {
		return
	}
	room := rooms.LoadRoom(arrival.RoomId)
	if room == nil {
		return
	}
	room.SendText(messaging.CategoryMobIdle, msg)
}
```

If `bucketsForWaypointIdx` takes only one arg today (the leader's waypoint idx, not the runner's), refactor it to also take the patrol id so it can distinguish runner-circuit patrols from the legacy caravan main route. Adapt the signature to match what's needed.

- [ ] **Step 3: Update the synthesizer**

In `internal/caravan/synthesize_state.go`, rewrite `SynthesizeStateForLeader` to handle the new 4-waypoint shape + Lars's patrol state:

```go
func SynthesizeStateForLeader(leader *mobs.Mob) (CaravanState, bool) {
	if leader == nil || leader.PatrolId != CaravanPatrolId {
		return 0, false
	}
	p := mobs.GetPatrol(CaravanPatrolId)
	if p == nil || len(p.Waypoints) == 0 {
		return 0, false
	}
	idx := miscDataInt(&leader.Character, "patrol_waypoint_idx")
	if idx < 0 || idx >= len(p.Waypoints) {
		idx = 0
	}

	wp := p.Waypoints[idx]
	atWaypoint := leader.Character.RoomId == wp.Room

	if !atWaypoint {
		// Chunk 3.8: with 4 waypoints, in-transit classification is:
		//   heading toward wp1 or wp2 → outbound
		//   heading toward wp3 or wp0 → inbound
		if idx == 1 || idx == 2 {
			return StateOutboundTransit, true
		}
		return StateInboundTransit, true
	}

	// At a waypoint — dispatch on arrival_event.
	switch wp.ArrivalEvent {
	case "caravan_fernway_pickup":
		if idx == 1 {
			return StateOutboundFernwayPickup, true
		}
		return StateInboundFernwayPickup, true
	case "caravan_depot":
		// wp0 = Thornwall, wp2 = Stillwater. Check if Lars has an
		// active runner-circuit patrol — if so, report *Route
		// (Lars is mid-delivery); otherwise *Dwell.
		larsActive := isRunnerCircuitActive()
		if idx == 0 {
			if larsActive {
				return StateThornwallRoute, true
			}
			return StateThornwallDwell, true
		}
		// idx == 2
		if larsActive {
			return StateStillwaterRoute, true
		}
		return StateStillwaterDwell, true
	}

	// Unrecognized arrival_event — defensive fallback.
	return StateThornwallDwell, true
}

// isRunnerCircuitActive reports whether Lars (mob 359) has an active
// runner-circuit oneshot patrol. Returns false if Lars isn't currently
// instanced (mid-respawn or wiped).
func isRunnerCircuitActive() bool {
	for _, instId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(instId)
		if m == nil || int(m.MobId) != RunnerMobId {
			continue
		}
		_, isRunner := runnerCircuitPatrols[m.PatrolId]
		return isRunner
	}
	return false
}
```

- [ ] **Step 4: Update the synthesizer tests**

In `internal/caravan/synthesize_state_test.go`, the fixture currently registers a 22-waypoint test patrol. Update the `registerTestCaravanPatrol` helper to register a 4-waypoint patrol matching the new shape:

```go
func registerTestCaravanPatrol(t *testing.T) {
	t.Helper()
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:        CaravanPatrolId,
		LoopShape: "strict",
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 465, DwellRounds: 360, ArrivalEvent: "caravan_depot"},          // wp0 Thornwall
			{Room: 4038, DwellRounds: 8, ArrivalEvent: "caravan_fernway_pickup"},  // wp1 outbound Fernway
			{Room: 4109, DwellRounds: 180, ArrivalEvent: "caravan_depot"},         // wp2 Stillwater
			{Room: 4038, DwellRounds: 8, ArrivalEvent: "caravan_fernway_pickup"},  // wp3 inbound Fernway
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest(CaravanPatrolId) })
}
```

Update `TestSynthesizeStateForLeader_WaypointMapping`'s case list:

```go
cases := []struct {
	name        string
	waypointIdx int
	want        CaravanState
}{
	{"wp0 (Thornwall depot)", 0, StateThornwallDwell},
	{"wp1 (Outbound Fernway pickup)", 1, StateOutboundFernwayPickup},
	{"wp2 (Stillwater depot)", 2, StateStillwaterDwell},
	{"wp3 (Inbound Fernway pickup)", 3, StateInboundFernwayPickup},
}
```

(Lars-active tests can be added but require seeding a runner mob instance — defer or add a focused one if straightforward.)

Update `TestSynthesizeStateForLeader_InTransitOutbound` and `_InTransitInbound` to use waypoint indices in the new ranges (1 or 2 for outbound, 3 or 0 for inbound — heading-toward-wp0 from elsewhere is inbound).

- [ ] **Step 5: Register the runner completion listener**

In `internal/hooks/hooks.go`'s `RegisterListeners`, add:

```go
events.RegisterListener(events.PatrolCompleted{}, caravan.CaravanRunnerCompletionListener)
```

- [ ] **Step 6: Build + test**

Run: `go build ./... && go test ./...`
Expected: build clean, all tests pass.

If `bucketsForWaypointIdx`-style functions break because of the 22→4 waypoint reduction, update them — runner-circuit patrols are distinct from the caravan main patrol, so the bucket logic likely needs a "which patrol id" parameter.

- [ ] **Step 7: Boot test**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go build -o dogmud-t15.exe . && ./dogmud-t15.exe > t15-boot.log 2>&1 &
BOOT_PID=$!
sleep 15
kill $BOOT_PID 2>/dev/null
grep -E 'panic|FATAL|LoadPatrols|loadedCount=' t15-boot.log | head -10
rm dogmud-t15.exe t15-boot.log
```

Expected:
- No panics.
- `mobs.LoadPatrols() loadedCount=6` (market beat + truncated caravan + 2 caravan runner + 2 forager delivery).
- Truncated caravan main shows 4 waypoint zone-resolutions (not 22).

- [ ] **Step 8: Stage and commit atomically**

```bash
git add internal/caravan/arrival_listener.go \
        internal/caravan/synthesize_state.go \
        internal/caravan/synthesize_state_test.go \
        _datafiles/world/dogmud/patrols/thornwall_outskirts/caravan_thornwall_stillwater.yaml \
        internal/hooks/hooks.go

git commit -m "feat(caravan): flip onto runner-delivery (22 -> 4 waypoints, Lars runs circuits)

Atomic change:
- Truncate caravan main YAML from 22 waypoints to 4 (depots +
  Fernway pickups). Vendor stops now live on Lars's runner-circuit
  oneshot patrols, not on the main route.
- handleDepotArrival at wp0/wp2 transfers wagon -> Lars cargo and
  starts the runner circuit. 5.3 stranded-cargo safety: if Lars is
  at the depot with cargo and no active patrol, transfer back.
- handleVendorArrival reads from FindRunnerInRoom (Lars), not the
  wagon. Wagon stays at the depot.
- Synthesizer rewrites to 4-waypoint shape; Lars's active patrol
  state determines whether dashboard shows *Route (mid-circuit) or
  *Dwell (idle at depot).
- CaravanRunnerCompletionListener registered to return cargo on
  PatrolCompleted.

Dashboard JSON enum unchanged (StillwaterRoute / ThornwallRoute /
*Dwell / *Transit / *FernwayPickup) — semantics evolve, schema
preserved.

Single atomic commit because the listener changes, synthesizer
changes, and YAML truncation are interdependent.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 16: Phase 3 boot + smoke

**Files:** (validation only)

- [ ] **Step 1: Wipe instance saves + boot**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
go build -o dogmud-p3.exe . && ./dogmud-p3.exe
```

Leave running for the smoke session.

- [ ] **Step 2: Run the 12-step smoke checklist**

From the spec (§Testing strategy):

1. `LoadPatrols loadedCount=6` per boot log.
2. Caravan at Thornwall depot, Lars co-located.
3. Lars's Thornwall circuit starts: cargo transfers wagon → Lars, Lars walks 8 vendor rooms with trade flavor at each.
4. Lars returns to depot, `PatrolCompleted` fires, residual cargo Lars → wagon.
5. Caravan departs Thornwall after 360-round dwell. Lars rejoins party.
6. Caravan reaches Stillwater. Stillwater circuit fires.
7. Caravan returns to Thornwall. Cycle restarts.
8. Forager Tova: wander+forage unchanged; on `StateDelivering`, sub-patrol starts; vendor stops fire `SellToVendor`; on `PatrolCompleted`, state advances to `StateStoring`.
9. Force stranding: admin command stops Lars mid-circuit. Watch home-fallback. Verify cargo-return on next depot arrival (5.3 safety).
10. Kill Lars mid-circuit. Verify cargo drops, Lars respawns at depot, party-follow re-engages.
11. `/admin/economy/` dashboard: caravan state cycles through `ThornwallDwell` → `ThornwallRoute` → `ThornwallDwell` → `OutboundTransit` → … etc.
12. Sanity-check Dal's `thornwall_barmaid` schedule still fires (3.6 conversation pilot).

- [ ] **Step 3: Clean up + hand off**

```bash
taskkill //F //IM dogmud-p3.exe 2>/dev/null
rm dogmud-p3.exe
```

If any step fails, escalate. Phase 3 is the highest-risk piece of 3.8.

---

## Phase 4 — Documentation + cleanup

### Task 17: PATCH_NOTES + roadmap update + memory archival

**Files:**
- Modify: `PATCH_NOTES.md`
- Modify: `MOB_ALIVENESS_ROADMAP.md`
- Modify: `~/.claude/projects/.../memory/MEMORY.md` (mark resolved memories)

- [ ] **Step 1: Add dated entry to PATCH_NOTES.md**

Prepend a new section after the top header:

```markdown
## 2026-MM-DD — Mob aliveness 3.8 (caravan runner + forager delivery on oneshot patrols)

**Caravan runs are richer now.** Ketil's caravan rolls into Thornwall
or Stillwater depot, parks the wagon, and Lars (Ketil's son) walks
the goods out to each vendor while the rest of the crew rests. The
wagon never gets dragged into an alchemy shop again. What doesn't
sell comes back to the wagon when Lars returns.

**Foragers stop getting hung up on the delivery loop.** Marsh
(Tova) and Steppe (Halix) foragers now use the same patrol-layer
machinery for their vendor circuit — retry-then-home-fallback,
combat-interrupt-and-resume, and standardized waypoint dwell.
Fernway forager (Kessa) keeps her single-stop sealed-crate handoff.

**Under the hood:** new `loop_shape: oneshot` patrol mode, new
`events.PatrolCompleted` event, `mobs.StartOneshotPatrol` /
`ClearOneshotPatrol` runtime helpers. Caravan main route shrinks
from 22 waypoints to 4 (depots + Fernway pickups). Vendor stops
live entirely on Lars's runner-circuit sub-patrols.

**Looking ahead:** attacking the caravan crew or wagon will carry
severe consequences once Town Justice (chunk 5.1) lands —
massive Thornwall guard faction rep loss, murder records, etc.
Don't roleplay yourself into a permanent rep hole on the way to
the next bandit ambush.
```

(Replace `2026-MM-DD` with today's date.)

- [ ] **Step 2: Mark 3.8 Done in MOB_ALIVENESS_ROADMAP.md**

Find the 3.8 chunk brief in `MOB_ALIVENESS_ROADMAP.md`. Update **Status** from `Not started` to `Done (2026-MM-DD)`. Add a **Shipped:** subsection mirroring the format of 3.4/3.6/3.7. Also update the table row at the top of the roadmap (status column) and the rollup count.

- [ ] **Step 3: Resolve memory items**

In the user's `MEMORY.md`, change `caravan-runner-delivery-flavor` priority from `Medium` to `Done` and update the description to reflect that 3.8 shipped the resurrection.

If any forager-stranding memories were 3.8-applicable (e.g., the chunk 2.9 fatigue-cadence followup), update those similarly if 3.8 closed them.

- [ ] **Step 4: Commit**

```bash
git add PATCH_NOTES.md MOB_ALIVENESS_ROADMAP.md
git commit -m "docs(3.8): PATCH_NOTES + roadmap status for one-shot sub-patrols

Mark chunk 3.8 (Caravan runner-delivery + foragers-on-patrols) Done.
Patch notes summarize the player-visible flavor change (Lars runs
the depot-to-vendor circuit) plus the engine surface (oneshot
patrol mode + PatrolCompleted event + runtime helpers).

Forward-looking note in patch notes flagging future Town Justice
consequences for attacking the caravan crew/wagon (chunk 5.1).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

Memory file updates are personal-config, not committed to the repo.

---

## Self-Review

The plan as written:

- **Spec coverage:** every spec In-scope item maps to a task. Engine primitive (T1-T4), forager consumer (T5-T10), caravan consumer (T11-T16), docs (T17).
- **Placeholder scan:** no TBD / TODO markers. Each step has the actual code or command.
- **Type consistency:** `StartOneshotPatrol` / `ClearOneshotPatrol` used consistently across T3, T4, T9. `TransferCargoToRunner` / `TransferAllCargoBack` consistent between T13 (defined) and T14/T15 (called). `RunnerMobId` defined in T13 and read in T13/T14/T15. `CaravanPatrolId` carried over from chunk 3.7.
- **Atomicity boundaries:** T15 is the single atomic commit. T1-T14 land independently. T16-T17 are validation/docs.

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-26-mob-aliveness-3.8-oneshot-subpatrols.md`. Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — Execute tasks in this session using `executing-plans`, batch execution with checkpoints.

Which approach?
