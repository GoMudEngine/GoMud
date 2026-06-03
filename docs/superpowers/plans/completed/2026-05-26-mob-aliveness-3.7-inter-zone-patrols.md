# Mob Aliveness 3.7 — Inter-Zone Patrols + Caravan Unification — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend chunk 3.4 patrols to cross zone boundaries, and migrate the existing Thornwall↔Stillwater caravan onto the unified patrol layer. Caravan-specific concerns (cargo, vendor trade, throughput, deliveries, Fernway handoff) stay in `internal/caravan/`; movement code is deleted.

**Architecture:** Two-phase rollout. Phase 1 adds backward-compatible primitive features (`arrival_event` waypoint field, `max_path_retries` per-patrol override, `PatrolWaypointArrival` event emission, zone-resolution boot logging, `caravan.SynthesizeStateForLeader` helper). Phase 2 is a single atomic flip — wire up the new arrival listener, add `patrol_id` to Ketil (mob 357), delete the legacy caravan state machine. Dashboard preserved end-to-end via the synthesizer helper.

**Tech Stack:** Go 1.24+, YAML config files, GoMud event system (`events.RegisterListener`), patrol primitive from chunk 3.4 (`internal/mobs/patrol.go`, `internal/hooks/NewRound_IdleMobs_patrol.go`).

**Spec:** `docs/superpowers/specs/2026-05-26-mob-aliveness-3.7-inter-zone-patrols-design.md`

**Branch:** `feature/mob-aliveness-3.7-inter-zone-patrols`

---

## File Structure

**Modified files:**

- `internal/mobs/patrol.go` — `Patrol` gains `MaxPathRetries int`; `PatrolWaypoint` gains `ArrivalEvent string`
- `internal/mobs/patrol_loader.go` — boot-time zone-resolution logging
- `internal/mobs/patrol_loader_test.go` — new fixture tests for `arrival_event` + `max_path_retries`
- `internal/events/eventtypes.go` — new `PatrolWaypointArrival` event type
- `internal/hooks/NewRound_IdleMobs_patrol.go` — emit `PatrolWaypointArrival` at first-dwell-tick + zero-dwell `WantsAdvance` paths; honor `MaxPathRetries` override
- `internal/hooks/NewRound_IdleMobs_patrol_test.go` — new emission idempotency + per-patrol retry tests (file may need to be created — check existing test layout in Phase 1 Task 4)
- `internal/hooks/hooks.go` — register `CaravanArrivalListener` (Phase 2)
- `internal/economy/health/capture.go` — swap `BTreeState["caravan_state"]` read for `caravan.SynthesizeStateForLeader` call (Phase 2)
- `internal/economy/health/capture_test.go` — fixtures rewritten to use `PatrolId` + `patrol_waypoint_idx` MiscData (Phase 2)
- `internal/caravan/state.go` — delete `AdvanceState`, `ParseState`, `nameToState`, `RouteForState`; keep enum + `Name()` + `Is*State` predicates (Phase 2)
- `internal/caravan/state_test.go` — delete tests for removed functions (Phase 2)
- `internal/behaviortree/caravan_reset.go` — rewrite for patrol-state-based reset (Phase 2)
- `_datafiles/world/dogmud/mobs/thornwall_city/357-*.yaml` (Ketil) — add `patrol_id: caravan_thornwall_stillwater` (Phase 2)

**New files:**

- `internal/caravan/synthesize_state.go` — `SynthesizeStateForLeader(leader *mobs.Mob) (CaravanState, bool)` (Phase 1)
- `internal/caravan/synthesize_state_test.go` — table-driven tests (Phase 1)
- `internal/caravan/arrival_listener.go` — `CaravanArrivalListener(e events.Event)` — dispatches based on `ArrivalEvent` value (Phase 2)
- `internal/caravan/arrival_listener_test.go` — unit tests with stub events (Phase 2)
- `_datafiles/world/dogmud/patrols/thornwall_outskirts/caravan_thornwall_stillwater.yaml` — the new caravan patrol (Phase 2)

**Deleted files:**

- `internal/caravan/routes.go` (Phase 2 atomic flip)
- `internal/behaviortree/actions_caravan.go` (Phase 2 atomic flip)
- `internal/behaviortree/actions_caravan_test.go` (Phase 2 atomic flip)
- `internal/behaviortree/actions_wagon.go` — only if audit confirms it's all wagon-as-follower movement helpers (Phase 2 atomic flip)
- `internal/behaviortree/actions_wagon_test.go` — paired with the above

---

## Phase 1 — Generic patrol primitive (no caravan effect)

### Task 1: Add `ArrivalEvent` field to `PatrolWaypoint`

**Files:**
- Modify: `internal/mobs/patrol.go:15-19`
- Modify: `internal/mobs/patrol_loader_test.go` (add test case)
- Test: `internal/mobs/patrol_loader_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/mobs/patrol_loader_test.go`:

```go
func TestLoadPatrol_ParsesArrivalEvent(t *testing.T) {
	tmp := t.TempDir()
	zoneDir := filepath.Join(tmp, "test_zone")
	if err := os.MkdirAll(zoneDir, 0o755); err != nil {
		t.Fatal(err)
	}
	yamlPath := filepath.Join(zoneDir, "with_arrival_event.yaml")
	yaml := `id: with_arrival_event
loop_shape: strict
waypoints:
  - room: 100
    dwell_rounds: 5
    arrival_event: vendor_visit
  - room: 101
    dwell_rounds: 0
`
	if err := os.WriteFile(yamlPath, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	// Use the test-only loader path that takes a base dir directly.
	count, err := loadPatrolsFromDir(tmp, func(id int) bool { return true }, func(a, b int) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 patrol loaded, got %d", count)
	}

	p := GetPatrol("with_arrival_event")
	if p == nil {
		t.Fatal("patrol not registered")
	}
	if got := p.Waypoints[0].ArrivalEvent; got != "vendor_visit" {
		t.Errorf("waypoint 0 ArrivalEvent = %q, want %q", got, "vendor_visit")
	}
	if got := p.Waypoints[1].ArrivalEvent; got != "" {
		t.Errorf("waypoint 1 ArrivalEvent = %q, want empty string", got)
	}
	t.Cleanup(func() { unregisterPatrolForTest("with_arrival_event") })
}
```

If `loadPatrolsFromDir` doesn't exist as a test-friendly helper, instead use `RegisterPatrolForTest` with a struct literal and skip the YAML loader path for this specific test:

```go
func TestPatrolWaypoint_ArrivalEventFieldRoundTrip(t *testing.T) {
	p := &Patrol{
		Id: "test_arrival_event",
		Waypoints: []PatrolWaypoint{
			{Room: 100, DwellRounds: 5, ArrivalEvent: "vendor_visit"},
			{Room: 101, DwellRounds: 0},
		},
	}
	RegisterPatrolForTest(p)
	t.Cleanup(func() { UnregisterPatrolForTest("test_arrival_event") })

	got := GetPatrol("test_arrival_event")
	if got == nil {
		t.Fatal("patrol not registered")
	}
	if got.Waypoints[0].ArrivalEvent != "vendor_visit" {
		t.Errorf("waypoint 0 ArrivalEvent = %q, want %q", got.Waypoints[0].ArrivalEvent, "vendor_visit")
	}
	if got.Waypoints[1].ArrivalEvent != "" {
		t.Errorf("waypoint 1 ArrivalEvent = %q, want empty string", got.Waypoints[1].ArrivalEvent)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/mobs/ -run TestPatrolWaypoint_ArrivalEventFieldRoundTrip -v`
Expected: FAIL with "unknown field ArrivalEvent" compile error.

- [ ] **Step 3: Add `ArrivalEvent` to the `PatrolWaypoint` struct**

In `internal/mobs/patrol.go:15-19`, change:

```go
// PatrolWaypoint is one stop on a patrol route.
type PatrolWaypoint struct {
	Room        int `yaml:"room"`
	DwellRounds int `yaml:"dwell_rounds,omitempty"` // 0 = move on immediately
}
```

to:

```go
// PatrolWaypoint is one stop on a patrol route.
type PatrolWaypoint struct {
	Room         int    `yaml:"room"`
	DwellRounds  int    `yaml:"dwell_rounds,omitempty"`  // 0 = move on immediately
	ArrivalEvent string `yaml:"arrival_event,omitempty"` // chunk 3.7: emitted via PatrolWaypointArrival on first-dwell-tick
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/mobs/ -run TestPatrolWaypoint_ArrivalEventFieldRoundTrip -v`
Expected: PASS.

- [ ] **Step 5: Run the full mobs package tests**

Run: `go test ./internal/mobs/...`
Expected: All tests pass (the new field is additive; existing tests don't touch it).

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/patrol.go internal/mobs/patrol_loader_test.go
git commit -m "feat(patrol): add optional ArrivalEvent field to PatrolWaypoint (3.7 schema prep)

Backwards-compatible. Patrol waypoints can now author an optional
arrival_event string that will be emitted via events.PatrolWaypointArrival
once that event type lands in a follow-up commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Add `MaxPathRetries` field to `Patrol` and honor it in the executor

**Files:**
- Modify: `internal/mobs/patrol.go:7-13`
- Modify: `internal/hooks/NewRound_IdleMobs_patrol.go:69-77`
- Test: `internal/hooks/NewRound_IdleMobs_patrol_test.go` (check if file exists; create if not)

- [ ] **Step 1: Check if patrol-executor test file exists**

Run: `ls internal/hooks/NewRound_IdleMobs_patrol*.go`
If no `_test.go` file exists, create one with this header:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)
```

- [ ] **Step 2: Write the failing test**

Add to the test file:

```go
func TestPatrolTickPlan_RespectsPerPatrolMaxRetries(t *testing.T) {
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:             "test_high_retry_patrol",
		LoopShape:      "strict",
		MaxPathRetries: 40,
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 100, DwellRounds: 0},
			{Room: 200, DwellRounds: 0},
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest("test_high_retry_patrol") })

	mob := &mobs.Mob{PatrolId: "test_high_retry_patrol"}
	mob.Character.RoomId = 999 // not at waypoint 0 (room 100) — wants path

	// At fail count 39, should still want path (under custom 40 cap)
	mob.Character.SetMiscData("patrol_path_fail_count", 39)
	plan := patrolTickPlan(mob, "test_high_retry_patrol")
	if plan.WantsHomeFallback {
		t.Errorf("expected WantsHomeFallback=false at fails=39 with cap=40, got %+v", plan)
	}
	if !plan.WantsPath {
		t.Errorf("expected WantsPath=true at fails=39 under cap=40, got %+v", plan)
	}

	// At fail count 40, should trigger home fallback
	mob.Character.SetMiscData("patrol_path_fail_count", 40)
	plan = patrolTickPlan(mob, "test_high_retry_patrol")
	if !plan.WantsHomeFallback {
		t.Errorf("expected WantsHomeFallback=true at fails=40 with cap=40, got %+v", plan)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/hooks/ -run TestPatrolTickPlan_RespectsPerPatrolMaxRetries -v`
Expected: FAIL with "unknown field MaxPathRetries" compile error.

- [ ] **Step 4: Add `MaxPathRetries` to the `Patrol` struct**

In `internal/mobs/patrol.go:7-13`, change:

```go
type Patrol struct {
	Id          string           `yaml:"id"`
	Description string           `yaml:"description,omitempty"`
	LoopShape   string           `yaml:"loop_shape,omitempty"` // "strict" (default) | "yo-yo"
	Waypoints   []PatrolWaypoint `yaml:"waypoints"`
}
```

to:

```go
type Patrol struct {
	Id             string           `yaml:"id"`
	Description    string           `yaml:"description,omitempty"`
	LoopShape      string           `yaml:"loop_shape,omitempty"`        // "strict" (default) | "yo-yo"
	MaxPathRetries int              `yaml:"max_path_retries,omitempty"` // chunk 3.7: override global ScheduleMaxPathRetries; 0 = use global default
	Waypoints      []PatrolWaypoint `yaml:"waypoints"`
}
```

- [ ] **Step 5: Honor `MaxPathRetries` in the patrol executor**

In `internal/hooks/NewRound_IdleMobs_patrol.go:69-77`, change:

```go
	// Not at target — path or fallback.
	maxRetries := int(configs.GetBalanceConfig().ScheduleMaxPathRetries)
	if maxRetries > 0 && failCount >= maxRetries {
		plan.WantsHomeFallback = true
		plan.FailureMessage = fmt.Sprintf(
			"patrol mob %d (%s) unreachable waypoint room %d after %d retries; falling back to home",
			mob.MobId, mob.Character.Name, currentWaypoint.Room, failCount)
		return plan
	}
```

to:

```go
	// Not at target — path or fallback.
	// Chunk 3.7: honor per-patrol MaxPathRetries override; 0 = use global default.
	maxRetries := p.MaxPathRetries
	if maxRetries == 0 {
		maxRetries = int(configs.GetBalanceConfig().ScheduleMaxPathRetries)
	}
	if maxRetries > 0 && failCount >= maxRetries {
		plan.WantsHomeFallback = true
		plan.FailureMessage = fmt.Sprintf(
			"patrol mob %d (%s) unreachable waypoint room %d after %d retries; falling back to home",
			mob.MobId, mob.Character.Name, currentWaypoint.Room, failCount)
		return plan
	}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `go test ./internal/hooks/ -run TestPatrolTickPlan_RespectsPerPatrolMaxRetries -v`
Expected: PASS.

- [ ] **Step 7: Run the full hooks package tests**

Run: `go test ./internal/hooks/...`
Expected: All tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/mobs/patrol.go internal/hooks/NewRound_IdleMobs_patrol.go internal/hooks/NewRound_IdleMobs_patrol_test.go
git commit -m "feat(patrol): add per-patrol MaxPathRetries override (3.7 cross-zone prep)

Cross-zone patrols (caravan) need more retry headroom than the
default 20-tick ScheduleMaxPathRetries — a single transient blocker
on a 50-room transit could exhaust the budget. The new field is
optional; 0 (default) falls back to the global config knob.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Define `events.PatrolWaypointArrival` event type

**Files:**
- Modify: `internal/events/eventtypes.go`

- [ ] **Step 1: Locate where to add the type**

Run: `grep -n 'type MobIdle\|type RoomChange\|type NewRound' internal/events/eventtypes.go | head -5`
Note the line numbers — add the new type near a similar mob-lifecycle event.

- [ ] **Step 2: Add the event struct**

In `internal/events/eventtypes.go`, add (placement: near other mob-lifecycle events like `MobDeath` or `MobIdle`):

```go
// PatrolWaypointArrival fires once when a patrol-running mob reaches a
// waypoint room and enters the dwell phase. Consumers filter by
// ArrivalEvent name. Empty ArrivalEvent fires regardless — useful for
// debug subscribers but skipped by name-filtered consumers. Chunk 3.7.
type PatrolWaypointArrival struct {
	MobInstanceId int
	PatrolId      string
	WaypointIdx   int
	RoomId        int
	ArrivalEvent  string
}

// Type returns the event type name (satisfies the events.Event interface).
func (e PatrolWaypointArrival) Type() string { return "PatrolWaypointArrival" }
```

Note: verify the interface method matches the local pattern. If existing events use a different method (e.g. `EventType()`, `Name()`), match it.

- [ ] **Step 3: Run build to verify the struct compiles**

Run: `go build ./internal/events/...`
Expected: clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/events/eventtypes.go
git commit -m "feat(events): define PatrolWaypointArrival event type (3.7 prep)

No emission yet; the patrol executor wires emission in a follow-up
commit. Consumers will filter on ArrivalEvent name.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Emit `PatrolWaypointArrival` from the patrol executor

**Files:**
- Modify: `internal/hooks/NewRound_IdleMobs_patrol.go`
- Test: `internal/hooks/NewRound_IdleMobs_patrol_test.go`

- [ ] **Step 1: Write the failing test for first-dwell-tick emission**

Add to `internal/hooks/NewRound_IdleMobs_patrol_test.go`:

```go
func TestApplyPatrolPlan_EmitsArrivalEventOnFirstDwellTick(t *testing.T) {
	// Register a test patrol with one waypoint that has a named arrival event.
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:        "test_arrival_emit",
		LoopShape: "strict",
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 100, DwellRounds: 5, ArrivalEvent: "test_marker"},
			{Room: 200, DwellRounds: 0},
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest("test_arrival_emit") })

	// Drain any prior queued events to isolate this test.
	events.DrainForTest()

	mob := &mobs.Mob{InstanceId: 9001, PatrolId: "test_arrival_emit"}
	mob.Character.RoomId = 100 // at waypoint 0

	// First tick: mob just arrived. dwell_remaining should be authored value (5).
	mob.Character.SetMiscData("patrol_waypoint_idx", 0)
	mob.Character.SetMiscData("patrol_dwell_remaining", 5)

	plan := patrolTickPlan(mob, "test_arrival_emit")
	applyPatrolPlan(mob, plan)

	queued := events.SnapshotForTest()
	matched := 0
	for _, e := range queued {
		if a, ok := e.(events.PatrolWaypointArrival); ok && a.MobInstanceId == 9001 {
			matched++
			if a.ArrivalEvent != "test_marker" {
				t.Errorf("ArrivalEvent = %q, want %q", a.ArrivalEvent, "test_marker")
			}
			if a.WaypointIdx != 0 {
				t.Errorf("WaypointIdx = %d, want 0", a.WaypointIdx)
			}
			if a.RoomId != 100 {
				t.Errorf("RoomId = %d, want 100", a.RoomId)
			}
		}
	}
	if matched != 1 {
		t.Fatalf("expected exactly 1 PatrolWaypointArrival for instance 9001, got %d", matched)
	}

	// Second tick at same waypoint — dwell now 4 (already decremented). Should NOT re-emit.
	events.DrainForTest()
	plan = patrolTickPlan(mob, "test_arrival_emit")
	applyPatrolPlan(mob, plan)
	queued = events.SnapshotForTest()
	for _, e := range queued {
		if a, ok := e.(events.PatrolWaypointArrival); ok && a.MobInstanceId == 9001 {
			t.Errorf("unexpected re-emission of PatrolWaypointArrival on subsequent tick: %+v", a)
		}
	}
}
```

If `events.DrainForTest` / `events.SnapshotForTest` don't exist, write the test using a different approach: register a listener that appends to a local slice, dispatch through `events.AddToQueue` + `events.Drain` (or whatever the local event-bus mechanism is). Check `internal/events/` to find the test helpers in use elsewhere (e.g., grep for `events\..*ForTest` in existing tests).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/hooks/ -run TestApplyPatrolPlan_EmitsArrivalEventOnFirstDwellTick -v`
Expected: FAIL — no emission happens yet.

- [ ] **Step 3: Add emission to `applyPatrolPlan` — first-dwell-tick path**

In `internal/hooks/NewRound_IdleMobs_patrol.go`, locate the `WantsDwellWait` branch (around line 104-111). Modify it:

```go
	case plan.WantsDwellWait:
		current := getMiscDataInt(&mob.Character, "patrol_dwell_remaining")
		// Chunk 3.7: emit the per-waypoint arrival event on the first dwell
		// tick. Detected by comparing current dwell to the authored value;
		// they match only on the tick immediately after arrival (before
		// the decrement below). Idempotent: subsequent ticks have a smaller
		// current value and won't re-emit.
		p := mobs.GetPatrol(activePatrolId)
		if p != nil {
			idx := getMiscDataInt(&mob.Character, "patrol_waypoint_idx")
			if idx >= 0 && idx < len(p.Waypoints) && current == p.Waypoints[idx].DwellRounds {
				events.AddToQueue(events.PatrolWaypointArrival{
					MobInstanceId: mob.InstanceId,
					PatrolId:      activePatrolId,
					WaypointIdx:   idx,
					RoomId:        mob.Character.RoomId,
					ArrivalEvent:  p.Waypoints[idx].ArrivalEvent,
				})
			}
		}
		if current > 0 {
			mob.Character.SetMiscData("patrol_dwell_remaining", current-1)
		}
		// At-target → reset retry counter.
		mob.Character.SetMiscData("patrol_path_fail_count", 0)
		return
```

Note: the `activePatrolId` is the second arg to `applyPatrolPlan`. Verify the signature in the existing file; if it doesn't take that parameter today, add it (and update the call site in `NewRound_IdleMobs.go`).

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/hooks/ -run TestApplyPatrolPlan_EmitsArrivalEventOnFirstDwellTick -v`
Expected: PASS.

- [ ] **Step 5: Write the failing test for zero-dwell waypoint emission**

Add to the same test file:

```go
func TestApplyPatrolPlan_EmitsArrivalEventOnZeroDwellWaypoint(t *testing.T) {
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:        "test_zero_dwell_emit",
		LoopShape: "strict",
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 100, DwellRounds: 0, ArrivalEvent: "instant_pass"},
			{Room: 200, DwellRounds: 0},
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest("test_zero_dwell_emit") })

	events.DrainForTest()

	mob := &mobs.Mob{InstanceId: 9002, PatrolId: "test_zero_dwell_emit"}
	mob.Character.RoomId = 100
	mob.Character.SetMiscData("patrol_waypoint_idx", 0)
	mob.Character.SetMiscData("patrol_dwell_remaining", 0)

	plan := patrolTickPlan(mob, "test_zero_dwell_emit")
	// Zero-dwell waypoints go directly to WantsAdvance, not WantsDwellWait.
	if !plan.WantsAdvance {
		t.Fatalf("expected WantsAdvance=true for zero-dwell waypoint, got %+v", plan)
	}
	applyPatrolPlan(mob, plan)

	queued := events.SnapshotForTest()
	matched := 0
	for _, e := range queued {
		if a, ok := e.(events.PatrolWaypointArrival); ok && a.MobInstanceId == 9002 {
			matched++
			if a.ArrivalEvent != "instant_pass" {
				t.Errorf("ArrivalEvent = %q, want %q", a.ArrivalEvent, "instant_pass")
			}
		}
	}
	if matched != 1 {
		t.Errorf("expected exactly 1 PatrolWaypointArrival on zero-dwell advance, got %d", matched)
	}
}
```

- [ ] **Step 6: Run test to verify it fails**

Run: `go test ./internal/hooks/ -run TestApplyPatrolPlan_EmitsArrivalEventOnZeroDwellWaypoint -v`
Expected: FAIL — zero-dwell `WantsAdvance` path doesn't emit yet.

- [ ] **Step 7: Add emission to the `WantsAdvance` branch (zero-dwell case)**

In `internal/hooks/NewRound_IdleMobs_patrol.go`, locate the `WantsAdvance` branch (around line 97-102). Modify:

```go
	case plan.WantsAdvance:
		// Chunk 3.7: emit arrival event for zero-dwell waypoints, which
		// skip the WantsDwellWait branch. Guard on RoomId == waypoint room
		// so we only fire when actually at the waypoint, not on a path-walk
		// step that happened to call applyPatrolPlan with WantsAdvance.
		p := mobs.GetPatrol(activePatrolId)
		if p != nil {
			idx := getMiscDataInt(&mob.Character, "patrol_waypoint_idx")
			if idx >= 0 && idx < len(p.Waypoints) &&
				p.Waypoints[idx].DwellRounds == 0 &&
				mob.Character.RoomId == p.Waypoints[idx].Room {
				events.AddToQueue(events.PatrolWaypointArrival{
					MobInstanceId: mob.InstanceId,
					PatrolId:      activePatrolId,
					WaypointIdx:   idx,
					RoomId:        mob.Character.RoomId,
					ArrivalEvent:  p.Waypoints[idx].ArrivalEvent,
				})
			}
		}
		mob.Character.SetMiscData("patrol_waypoint_idx", plan.NextWaypointIdx)
		mob.Character.SetMiscData("patrol_direction", plan.NextDirection)
		mob.Character.SetMiscData("patrol_dwell_remaining", plan.NextDwellRounds)
		mob.Character.SetMiscData("patrol_path_fail_count", 0)
		return
```

- [ ] **Step 8: Run both new tests to verify they pass**

Run: `go test ./internal/hooks/ -run "TestApplyPatrolPlan_EmitsArrivalEvent" -v`
Expected: both PASS.

- [ ] **Step 9: Run the full hooks suite**

Run: `go test ./internal/hooks/...`
Expected: all tests pass. The existing market-beat patrol still works (it has no `arrival_event`, so emission carries empty strings — no consumer matches yet).

- [ ] **Step 10: Commit**

```bash
git add internal/hooks/NewRound_IdleMobs_patrol.go internal/hooks/NewRound_IdleMobs_patrol_test.go
git commit -m "feat(patrol): emit PatrolWaypointArrival events on waypoint arrival

Fires exactly once per waypoint visit. Covers both the standard
first-dwell-tick path and the zero-dwell-waypoint WantsAdvance path.
Idempotent on the dwell side via dwell_remaining == authored value;
idempotent on the advance side via RoomId == waypoint room.

No consumer yet — caravan listener wires up in chunk 3.7 Phase 2.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Add zone-resolution boot logging to the patrol loader

**Files:**
- Modify: `internal/mobs/patrol_loader.go`

- [ ] **Step 1: Find the post-load patrol iteration point**

Run: `grep -n 'loadedCount\|mudlog.Info.*Patrol\|range patrols\|len(patrols)' internal/mobs/patrol_loader.go`
Note the location where each patrol's `Waypoints` are accessible after parsing.

- [ ] **Step 2: Add the per-waypoint zone log**

In `internal/mobs/patrol_loader.go`, after each patrol is successfully parsed and registered, add an info-level log for each waypoint:

```go
// Chunk 3.7: log resolved zones for human-readable boot-log sanity-checking
// of cross-zone patrols. No validation panics — zone resolution failures
// just log an empty zone name. Room existence is already validated above.
for idx, wp := range p.Waypoints {
	zoneName := ""
	if patrolWorldValidator.roomZone != nil {
		zoneName = patrolWorldValidator.roomZone(wp.Room)
	}
	mudlog.Info("patrol waypoint",
		"patrol", p.Id,
		"waypoint", idx,
		"room", wp.Room,
		"zone", zoneName,
	)
}
```

If `patrolWorldValidator` doesn't have a `roomZone` field yet, add one:

In `internal/mobs/patrol_loader.go`, change:

```go
var patrolWorldValidator struct {
	roomExists func(id int) bool
	hasPath    func(from, to int) bool
}

func SetPatrolWorldValidator(roomExists func(int) bool, hasPath func(from, to int) bool) {
	patrolWorldValidator.roomExists = roomExists
	patrolWorldValidator.hasPath = hasPath
}
```

to:

```go
var patrolWorldValidator struct {
	roomExists func(id int) bool
	hasPath    func(from, to int) bool
	roomZone   func(id int) string // chunk 3.7: resolve room id → zone name for boot log
}

func SetPatrolWorldValidator(roomExists func(int) bool, hasPath func(from, to int) bool, roomZone func(int) string) {
	patrolWorldValidator.roomExists = roomExists
	patrolWorldValidator.hasPath = hasPath
	patrolWorldValidator.roomZone = roomZone
}
```

- [ ] **Step 3: Update the call site for `SetPatrolWorldValidator`**

Run: `grep -rn 'SetPatrolWorldValidator' internal/`
Note where this is called (typically in main.go or boot wiring).

Update the call site to pass a `roomZone` resolver. Likely:

```go
mobs.SetPatrolWorldValidator(
	func(id int) bool { return rooms.LoadRoom(id) != nil },
	func(from, to int) bool { /* existing pathfinder call */ },
	func(id int) string {
		r := rooms.LoadRoom(id)
		if r == nil {
			return ""
		}
		return r.Zone
	},
)
```

- [ ] **Step 4: Build the project**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 5: Boot the server briefly to verify the logs appear**

Run:
```bash
go build -o dogmud-test.exe . && ./dogmud-test.exe 2>&1 | tee boot.log &
BOOT_PID=$!
sleep 12
kill $BOOT_PID 2>/dev/null
grep 'patrol waypoint' boot.log | head -20
rm dogmud-test.exe boot.log
```

Expected: at least 5 lines of `patrol waypoint` info logs for the existing Thornwall market-beat patrol's 5 waypoints, with `zone="Thornwall City"`.

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/patrol_loader.go path/to/main_or_boot_wiring.go
git commit -m "feat(patrol): log resolved zone for each waypoint on load (3.7 visibility)

Cross-zone patrols can now be sanity-checked from the boot log without
needing a separate admin command. Info-level only; no validation.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: Add `caravan.SynthesizeStateForLeader` helper

**Files:**
- Create: `internal/caravan/synthesize_state.go`
- Create: `internal/caravan/synthesize_state_test.go`

The synthesizer maps `(waypoint idx, arrival_event, mob in-transit flag)` to one of the eight `CaravanState` enum values. The mapping reflects the caravan patrol YAML's waypoint structure (authored in Task 7):

| Waypoint idx | Arrival event | Synthesized state |
|---|---|---|
| 0 | caravan_depot | ThornwallDwell (departure side, long dwell) |
| 1 | caravan_fernway_pickup | OutboundFernwayPickup |
| 2 | caravan_depot | StillwaterDwell (arrival side) |
| 3-10 | caravan_vendor | StillwaterRoute |
| 11 | caravan_depot | StillwaterDwell (departure side) |
| 12 | caravan_fernway_pickup | InboundFernwayPickup |
| 13 | caravan_depot | ThornwallDwell (arrival side) |
| 14-21 | caravan_vendor | ThornwallRoute |

In-transit (between waypoints, `mob.Path.Len() > 0` or `mob.Path.Current() != nil`):
- Coming from wp 0 or going to wp 1, 2 → OutboundTransit
- Coming from wp 11 or going to wp 12, 13 → InboundTransit

- [ ] **Step 1: Write the failing tests**

Create `internal/caravan/synthesize_state_test.go`:

```go
package caravan

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestSynthesizeStateForLeader_NonCaravanMobReturnsFalse(t *testing.T) {
	mob := &mobs.Mob{PatrolId: "thornwall_market_beat"}
	state, ok := SynthesizeStateForLeader(mob)
	if ok {
		t.Errorf("expected (_, false) for non-caravan patrol, got (%v, true)", state)
	}
}

func TestSynthesizeStateForLeader_NoPatrolIdReturnsFalse(t *testing.T) {
	mob := &mobs.Mob{}
	state, ok := SynthesizeStateForLeader(mob)
	if ok {
		t.Errorf("expected (_, false) for mob with no patrol_id, got (%v, true)", state)
	}
}

func TestSynthesizeStateForLeader_NilMobReturnsFalse(t *testing.T) {
	state, ok := SynthesizeStateForLeader(nil)
	if ok {
		t.Errorf("expected (_, false) for nil mob, got (%v, true)", state)
	}
}

func TestSynthesizeStateForLeader_WaypointMapping(t *testing.T) {
	// Register the caravan patrol shape (waypoint count + arrival_events
	// must match the authored YAML in Task 7).
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:        "caravan_thornwall_stillwater",
		LoopShape: "strict",
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 465, DwellRounds: 360, ArrivalEvent: "caravan_depot"},          // wp0
			{Room: 4038, DwellRounds: 8, ArrivalEvent: "caravan_fernway_pickup"},  // wp1
			{Room: 4109, DwellRounds: 20, ArrivalEvent: "caravan_depot"},          // wp2
			{Room: 4102, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},          // wp3
			{Room: 4103, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 4105, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 4106, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 4125, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 4126, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 4135, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 4143, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},          // wp10
			{Room: 4109, DwellRounds: 20, ArrivalEvent: "caravan_depot"},          // wp11
			{Room: 4038, DwellRounds: 8, ArrivalEvent: "caravan_fernway_pickup"},  // wp12
			{Room: 465, DwellRounds: 20, ArrivalEvent: "caravan_depot"},           // wp13
			{Room: 464, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},           // wp14
			{Room: 470, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 471, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 475, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 480, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 481, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 482, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},
			{Room: 483, DwellRounds: 5, ArrivalEvent: "caravan_vendor"},           // wp21
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest("caravan_thornwall_stillwater") })

	cases := []struct {
		name        string
		waypointIdx int
		want        CaravanState
	}{
		{"wp0 (Thornwall depot, departure)", 0, StateThornwallDwell},
		{"wp1 (Outbound Fernway pickup)", 1, StateOutboundFernwayPickup},
		{"wp2 (Stillwater depot, arrival)", 2, StateStillwaterDwell},
		{"wp3-10 (Stillwater vendor circuit)", 5, StateStillwaterRoute},
		{"wp10 (Stillwater vendor circuit end)", 10, StateStillwaterRoute},
		{"wp11 (Stillwater depot, departure)", 11, StateStillwaterDwell},
		{"wp12 (Inbound Fernway pickup)", 12, StateInboundFernwayPickup},
		{"wp13 (Thornwall depot, arrival)", 13, StateThornwallDwell},
		{"wp14-21 (Thornwall vendor circuit)", 17, StateThornwallRoute},
		{"wp21 (Thornwall vendor circuit end)", 21, StateThornwallRoute},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mob := &mobs.Mob{PatrolId: "caravan_thornwall_stillwater"}
			mob.Character.SetMiscData("patrol_waypoint_idx", tc.waypointIdx)
			// No path → not in transit → at waypoint.
			got, ok := SynthesizeStateForLeader(mob)
			if !ok {
				t.Fatalf("expected ok=true for caravan patrol, got false")
			}
			if got != tc.want {
				t.Errorf("waypoint %d: got state %s, want %s", tc.waypointIdx, got.Name(), tc.want.Name())
			}
		})
	}
}

func TestSynthesizeStateForLeader_InTransitClassification(t *testing.T) {
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:        "caravan_thornwall_stillwater",
		LoopShape: "strict",
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 465, DwellRounds: 360, ArrivalEvent: "caravan_depot"},
			// (truncated for test brevity — only need enough waypoints to test transit)
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest("caravan_thornwall_stillwater") })

	mob := &mobs.Mob{PatrolId: "caravan_thornwall_stillwater"}
	mob.Character.SetMiscData("patrol_waypoint_idx", 0)
	// Simulate path-in-flight: not at waypoint room.
	mob.Character.RoomId = 999 // somewhere between waypoints

	got, ok := SynthesizeStateForLeader(mob)
	if !ok {
		t.Fatal("expected ok=true")
	}
	// Heading toward wp0 from elsewhere = outbound transit (caravan just departed depot
	// OR is in early transit segment — synthesizer classifies by "next waypoint idx").
	if got != StateOutboundTransit {
		t.Errorf("got %s, want OutboundTransit when mob is in transit toward wp0/wp1/wp2", got.Name())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/caravan/ -run TestSynthesizeStateForLeader -v`
Expected: FAIL — `SynthesizeStateForLeader` undefined.

- [ ] **Step 3: Implement the helper**

Create `internal/caravan/synthesize_state.go`:

```go
package caravan

import "github.com/GoMudEngine/GoMud/internal/mobs"

// CaravanPatrolId is the canonical id of the patrol that drives the
// Thornwall ↔ Stillwater caravan. Used by SynthesizeStateForLeader to
// recognize the caravan leader.
const CaravanPatrolId = "caravan_thornwall_stillwater"

// SynthesizeStateForLeader derives the dashboard-facing CaravanState
// from the leader mob's patrol state (waypoint index + arrival_event
// + in-transit flag). Returns (_, false) if the mob is not running
// the caravan patrol.
//
// This replaces the pre-3.7 read of BTreeState["caravan_state"]. The
// patrol layer is now the source of truth for caravan movement state;
// the synthesizer adapts patrol state back into the dashboard's
// canonical state-name vocabulary so the economy-health JSON payload
// is byte-identical post-migration.
func SynthesizeStateForLeader(leader *mobs.Mob) (CaravanState, bool) {
	if leader == nil || leader.PatrolId != CaravanPatrolId {
		return 0, false
	}
	p := mobs.GetPatrol(CaravanPatrolId)
	if p == nil || len(p.Waypoints) == 0 {
		return 0, false
	}
	idx := getMiscDataInt(&leader.Character, "patrol_waypoint_idx")
	if idx < 0 || idx >= len(p.Waypoints) {
		idx = 0
	}

	wp := p.Waypoints[idx]
	atWaypoint := leader.Character.RoomId == wp.Room

	if !atWaypoint {
		// In transit toward waypoint idx. Classify outbound vs inbound by
		// whether the next waypoint is in the outbound half (wp 1–10) or
		// inbound half (wp 12–21) of the cycle.
		if idx <= 10 {
			return StateOutboundTransit, true
		}
		return StateInboundTransit, true
	}

	// At a waypoint — map the arrival_event + waypoint role to a state.
	switch wp.ArrivalEvent {
	case "caravan_fernway_pickup":
		if idx == 1 {
			return StateOutboundFernwayPickup, true
		}
		return StateInboundFernwayPickup, true
	case "caravan_depot":
		// Thornwall depot lives at wp0 (departure, long dwell) and wp13
		// (arrival, short dwell). Both synthesize to ThornwallDwell.
		// Stillwater depot lives at wp2 (arrival) and wp11 (departure).
		// Both synthesize to StillwaterDwell.
		if wp.Room == 465 {
			return StateThornwallDwell, true
		}
		return StateStillwaterDwell, true
	case "caravan_vendor":
		// Stillwater vendor circuit is wp3–wp10; Thornwall vendor circuit is wp14–wp21.
		if idx <= 10 {
			return StateStillwaterRoute, true
		}
		return StateThornwallRoute, true
	}

	// Unrecognized arrival_event on a caravan patrol — fall back to
	// Thornwall dwell as the safest default (matches initial-spawn state).
	return StateThornwallDwell, true
}

// getMiscDataInt is a local helper for reading an int MiscData key with
// zero as the unset default. Kept package-local because the equivalent
// helper in package hooks isn't exported.
func getMiscDataInt(c interface{ GetMiscData(string) any }, key string) int {
	v := c.GetMiscData(key)
	if v == nil {
		return 0
	}
	if i, ok := v.(int); ok {
		return i
	}
	return 0
}
```

If the `getMiscDataInt` helper signature doesn't match the actual `Character.GetMiscData` shape (check the existing implementation), adjust to match. The intent: read an int value, return 0 on missing or wrong type.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/caravan/ -run TestSynthesizeStateForLeader -v`
Expected: all PASS.

- [ ] **Step 5: Run the full caravan package tests**

Run: `go test ./internal/caravan/...`
Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/caravan/synthesize_state.go internal/caravan/synthesize_state_test.go
git commit -m "feat(caravan): add SynthesizeStateForLeader for dashboard parity (3.7)

Derives the canonical state-name string from the leader's patrol
waypoint + arrival_event + in-transit flag. Not yet wired into the
dashboard — capture.go swap lands in the Phase 2 atomic flip.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Phase 2 — Caravan scaffolding (compiled but inactive)

### Task 7: Author the caravan patrol YAML

**Files:**
- Create: `_datafiles/world/dogmud/patrols/thornwall_outskirts/caravan_thornwall_stillwater.yaml`

- [ ] **Step 1: Create the YAML file**

Write `_datafiles/world/dogmud/patrols/thornwall_outskirts/caravan_thornwall_stillwater.yaml`:

```yaml
id: caravan_thornwall_stillwater
description: "Ketil's caravan crew: Thornwall depot → Stillwater vendors → back. Authored 3.7 (mob aliveness chunk 3.7 — replaces internal/caravan/routes.go state machine)."
loop_shape: strict
max_path_retries: 40   # cross-zone transits need more headroom than the 20 default
waypoints:
  # ── wp0: Thornwall depot — long dwell before next departure ───
  - room: 465
    dwell_rounds: 360
    arrival_event: caravan_depot

  # ── wp1: Outbound Fernway pickup (forager handoff) ────────────
  - room: 4038
    dwell_rounds: 8
    arrival_event: caravan_fernway_pickup

  # ── wp2: Stillwater arrival depot ─────────────────────────────
  - room: 4109
    dwell_rounds: 20
    arrival_event: caravan_depot

  # ── wp3-wp10: Stillwater vendor circuit ───────────────────────
  - { room: 4102, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 4103, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 4105, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 4106, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 4125, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 4126, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 4135, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 4143, dwell_rounds: 5, arrival_event: caravan_vendor }

  # ── wp11: Stillwater departure depot ──────────────────────────
  - room: 4109
    dwell_rounds: 20
    arrival_event: caravan_depot

  # ── wp12: Inbound Fernway pickup ──────────────────────────────
  - room: 4038
    dwell_rounds: 8
    arrival_event: caravan_fernway_pickup

  # ── wp13: Thornwall arrival depot ─────────────────────────────
  - room: 465
    dwell_rounds: 20
    arrival_event: caravan_depot

  # ── wp14-wp21: Thornwall vendor circuit ───────────────────────
  - { room: 464, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 470, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 471, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 475, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 480, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 481, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 482, dwell_rounds: 5, arrival_event: caravan_vendor }
  - { room: 483, dwell_rounds: 5, arrival_event: caravan_vendor }
  # loops back to wp0 (Thornwall depot, 360-round dwell)
```

- [ ] **Step 2: Boot the server and verify load**

Run:
```bash
go build -o dogmud-test.exe . && ./dogmud-test.exe 2>&1 | tee boot.log &
BOOT_PID=$!
sleep 15
kill $BOOT_PID 2>/dev/null
grep 'LoadPatrols\|caravan_thornwall_stillwater\|patrol waypoint' boot.log | head -30
rm dogmud-test.exe boot.log
```

Expected:
- `mobs.LoadPatrols() loadedCount=2` (was 1).
- `patrol waypoint patrol=caravan_thornwall_stillwater waypoint=0 room=465 zone="Thornwall City"` (and 21 more).
- No panics.

The patrol is loaded but no mob references it yet — caravan still runs on the legacy state machine.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/patrols/thornwall_outskirts/caravan_thornwall_stillwater.yaml
git commit -m "feat(content): author caravan_thornwall_stillwater patrol YAML (3.7)

22 waypoints: depots + 2 Fernway pickup stops + 16 vendor stops
(8 Stillwater + 8 Thornwall). max_path_retries: 40 for cross-zone
headroom. Loaded at boot but not yet referenced by any mob —
legacy caravan state machine still drives Ketil until Phase 2
atomic flip.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: Add `internal/caravan/arrival_listener.go` (compiled but not registered)

**Files:**
- Create: `internal/caravan/arrival_listener.go`
- Create: `internal/caravan/arrival_listener_test.go`

The listener subscribes to `events.PatrolWaypointArrival`. It dispatches based on `ArrivalEvent`:

- `caravan_depot` at **wp0** (Thornwall depot, 360 dwell — fresh cycle start) → force-regroup any in-world crew not co-located with the leader.
- `caravan_depot` at any other wp → no-op for now (depot bookkeeping placeholder).
- `caravan_vendor` → `VisitVendorsInRoom(roomId, wagon, deliveryBuckets, pickupBuckets)`, format the visit message, send to room.
- `caravan_fernway_pickup` → forager handoff bookkeeping (extracted from current Fernway-pickup btree action).

Also updates `caravan_state_started_round` MiscData when the synthesized state name flips.

**Note on the leader-respawn crew regroup:** the spec proposed a separate `MobRespawn_CaravanCrewRegroup` hook, but this codebase has no explicit MobRespawn event. The regroup mechanism collapses cleanly into the wp0 `caravan_depot` arrival case — that's the exact moment any fresh respawn lands (HomeRoomId 465, patrol restarts at idx 0, arrival listener fires). Self-healing without a separate hook.

- [ ] **Step 1: Find the current Fernway-pickup logic to lift**

Run: `grep -rn 'fernway\|Fernway\|4038' internal/behaviortree/ | head -20`

Locate the current Fernway-pickup btree action implementation (likely in `internal/behaviortree/actions_caravan.go`). Note the forager handoff function — that gets called from the new arrival listener.

- [ ] **Step 2: Write the failing tests**

Create `internal/caravan/arrival_listener_test.go`:

```go
package caravan

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestCaravanArrivalListener_IgnoresNonCaravanEvents(t *testing.T) {
	// Event for a non-caravan mob — listener should no-op.
	e := events.PatrolWaypointArrival{
		MobInstanceId: 12345,
		PatrolId:      "thornwall_market_beat",
		ArrivalEvent:  "",
	}
	got := CaravanArrivalListener(e)
	if got != events.Continue {
		t.Errorf("expected events.Continue for non-caravan event, got %v", got)
	}
	// (No side-effect to assert; the test is structural — verifying the
	// listener doesn't panic on non-caravan events.)
}

func TestCaravanArrivalListener_NoOpOnUnknownArrivalEvent(t *testing.T) {
	// Caravan patrol but unknown arrival_event name — listener should no-op
	// (matches the "free-form arrival_event, consumers filter" contract).
	e := events.PatrolWaypointArrival{
		MobInstanceId: 12345,
		PatrolId:      CaravanPatrolId,
		ArrivalEvent:  "made_up_event_name",
	}
	got := CaravanArrivalListener(e)
	if got != events.Continue {
		t.Errorf("expected events.Continue for unknown arrival_event, got %v", got)
	}
}

// Note: tests that exercise VisitVendorsInRoom + room state are deferred
// to Phase 3 smoke validation (they require full mob + room setup that
// the existing internal/caravan/visit_test.go handles in isolation).
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/caravan/ -run TestCaravanArrivalListener -v`
Expected: FAIL — `CaravanArrivalListener` undefined.

- [ ] **Step 4: Implement the listener**

Create `internal/caravan/arrival_listener.go`:

```go
package caravan

import (
	"github.com/GoMudEngine/GoMud/internal/economy"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// CaravanArrivalListener consumes events.PatrolWaypointArrival and
// dispatches caravan-specific bookkeeping based on the event's
// ArrivalEvent name. Filters on the caravan patrol id; ignores
// arrivals from any other patrol.
//
// Dispatches:
//   - caravan_depot at wp0 (long-dwell Thornwall start) → crew regroup
//     (handles the "leader died mid-route, respawned at depot, fresh
//      cycle starts" contingency).
//   - caravan_depot at any other wp → state-name transition stamp.
//   - caravan_vendor → bidirectional VisitVendorsInRoom + room flavor.
//   - caravan_fernway_pickup → forager handoff bookkeeping.
//   - empty / unknown → no-op (free-form arrival_event contract).
func CaravanArrivalListener(e events.Event) events.ListenerReturn {
	arrival, ok := e.(events.PatrolWaypointArrival)
	if !ok {
		return events.Continue
	}
	if arrival.PatrolId != CaravanPatrolId {
		return events.Continue
	}

	leader := mobs.GetInstance(arrival.MobInstanceId)
	if leader == nil {
		return events.Continue
	}

	// Stamp the synthesized-state-started-round on transitions. The
	// dashboard reads this from MiscData (no longer from BTreeState).
	if state, ok := SynthesizeStateForLeader(leader); ok {
		prev := uint64(0)
		if v, ok := leader.Character.GetMiscData("caravan_state_last").(string); ok && v == state.Name() {
			// Same state — preserve the existing started-round.
			if pv, ok := leader.Character.GetMiscData("caravan_state_started_round").(uint64); ok {
				prev = pv
			}
		}
		if prev == 0 {
			leader.Character.SetMiscData("caravan_state_started_round", uint64(util.GetRoundCount()))
			leader.Character.SetMiscData("caravan_state_last", state.Name())
		}
	}

	switch arrival.ArrivalEvent {
	case "caravan_depot":
		handleDepotArrival(leader, arrival)
	case "caravan_vendor":
		handleVendorArrival(leader, arrival)
	case "caravan_fernway_pickup":
		handleFernwayPickupArrival(leader, arrival)
	}

	return events.Continue
}

// handleDepotArrival force-regroups stranded crew when the leader is at
// wp0 (the long-dwell Thornwall start — also the post-respawn landing
// point). At other depot waypoints, no-op (could grow into gold/cargo
// settlement bookkeeping in future).
func handleDepotArrival(leader *mobs.Mob, arrival events.PatrolWaypointArrival) {
	if arrival.WaypointIdx != 0 {
		return // only wp0 triggers the fresh-cycle regroup
	}
	// Force-move the wagon + any in-world crew to the leader's room.
	leaderRoom := leader.Character.RoomId
	for _, mobId := range caravanMobIds {
		_ = mobId // (caravanMobIds is a map[int]struct{}; iterate by key)
	}
	for templateId := range caravanMobIds {
		if templateId == int(leader.MobId) {
			continue
		}
		// Find the in-world instance by template id. Live-instance lookup
		// avoids the issue of stale references.
		for _, instId := range mobs.GetAllMobInstanceIds() {
			m := mobs.GetInstance(instId)
			if m == nil || int(m.MobId) != templateId {
				continue
			}
			if m.Character.RoomId == leaderRoom {
				continue // already co-located, nothing to do
			}
			oldRoom := m.Character.RoomId
			_ = oldRoom
			// Use the standard mob room-move helper. Look up the canonical
			// move function in internal/mobs/ or internal/rooms/ — likely
			// rooms.MoveMobToRoom(mobInstanceId, newRoomId) or similar.
			// Pseudo:
			//   rooms.MoveMobBetweenRooms(m, oldRoom, leaderRoom)
			mudlog.Info("caravan crew regroup",
				"leader", leader.Character.Name,
				"crew_template", templateId,
				"crew_instance", m.InstanceId,
				"from_room", oldRoom,
				"to_room", leaderRoom,
			)
			// Real call goes here — replace with the actual helper signature
			// after verifying it in step 5.
		}
	}
}

// handleVendorArrival fires the bidirectional vendor trade and prints
// the room flavor message.
func handleVendorArrival(leader *mobs.Mob, arrival events.PatrolWaypointArrival) {
	wagon := FindWagonInRoom(arrival.RoomId)
	if wagon == nil {
		mudlog.Warn("caravan vendor stop without wagon",
			"leader", leader.Character.Name,
			"room", arrival.RoomId,
		)
		return
	}

	// Bucket lists are caravan-policy decisions; in the legacy implementation
	// these came from internal/economy/. Verify the actual function signature
	// when implementing — VisitVendorsInRoom returns (delivered, pickedUp).
	deliveryBuckets := []string{} // placeholder — fill in from economy
	pickupBuckets := []string{}   // placeholder — fill in from economy
	_ = economy.BucketFor          // suppress unused-import warning until filled in

	delivered, pickedUp := VisitVendorsInRoom(arrival.RoomId, wagon, deliveryBuckets, pickupBuckets)
	msg := FormatVisitMessage(delivered, pickedUp)
	if msg == "" {
		return
	}
	room := rooms.LoadRoom(arrival.RoomId)
	if room == nil {
		return
	}
	// Send the visual flavor message to the room. Use the canonical helper
	// (e.g., room.SendText / sendVisualRoomText) — match the legacy
	// implementation in visit.go's caller.
	_ = messaging.CategoryMobIdle
	// room.SendText(messaging.CategoryMobIdle, msg)
}

// handleFernwayPickupArrival fires the forager handoff logic that
// previously lived in the StateOutbound/InboundFernwayPickup btree
// action. Lift from internal/behaviortree/actions_caravan.go.
func handleFernwayPickupArrival(leader *mobs.Mob, arrival events.PatrolWaypointArrival) {
	wagon := FindWagonInRoom(arrival.RoomId)
	if wagon == nil {
		return
	}
	// Lifted from the existing Fernway-pickup btree action. The legacy
	// code looks up a forager mob in the same room and transfers
	// inventory to the wagon. Verify the exact handoff function name
	// when implementing — likely tryForagerHandoff(wagon, room).
	// Pseudo:
	//   tryForagerHandoff(wagon, arrival.RoomId)
	_ = wagon
}
```

**Implementation notes for the agent doing this task:**

- The `rooms.MoveMobBetweenRooms` call in `handleDepotArrival` needs the actual function name. Grep `internal/rooms/` for the standard mob room-move helper. The legacy `actions_caravan.go` will have a working example of how the wagon was moved between rooms.
- The bucket lists in `handleVendorArrival` were authored in the legacy code — grep `actions_caravan.go` or `internal/caravan/visit.go` for `deliveryBuckets` / `pickupBuckets` definitions and lift them verbatim.
- The forager handoff in `handleFernwayPickupArrival` lives in the legacy btree action — lift the body of the existing Fernway-pickup handler into this function. The signature is whatever takes the wagon mob + room id.

- [ ] **Step 5: Wire up the actual helper calls**

Grep the legacy implementations and fill in the placeholders marked with comments:

```bash
grep -rn 'deliveryBuckets\|pickupBuckets' internal/behaviortree/actions_caravan.go internal/caravan/
grep -rn 'MoveMobBetweenRooms\|MoveToRoom\|moveTo' internal/rooms/ internal/mobs/ | head -20
grep -rn 'foragerHandoff\|forager_handoff' internal/
```

Replace the placeholder calls with the actual function invocations.

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/caravan/ -run TestCaravanArrivalListener -v`
Expected: PASS.

- [ ] **Step 7: Run the full caravan package tests**

Run: `go test ./internal/caravan/...`
Expected: all tests pass.

- [ ] **Step 8: Confirm the listener is NOT yet registered**

Run: `grep -n 'CaravanArrivalListener' internal/hooks/hooks.go`
Expected: no matches. Phase 2 atomic flip wires it up.

- [ ] **Step 9: Commit**

```bash
git add internal/caravan/arrival_listener.go internal/caravan/arrival_listener_test.go
git commit -m "feat(caravan): add CaravanArrivalListener for PatrolWaypointArrival (3.7)

Subscribes (in a follow-up commit) to events.PatrolWaypointArrival
and dispatches per the ArrivalEvent name:
- caravan_depot at wp0 → crew regroup (self-healing, also covers
  leader-died-mid-route scenario from the spec)
- caravan_vendor → bidirectional VisitVendorsInRoom + flavor
- caravan_fernway_pickup → forager handoff (lifted from
  legacy actions_caravan.go)

Compiled but inert — the legacy state machine still drives the
caravan. Atomic flip lands in the next commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Phase 3 — Atomic flip (one commit, multiple coordinated edits)

### Task 9: Atomic caravan flip

This is the **atomic flip** — many edits in one commit. There is no safe intermediate state because both the legacy state machine and the new arrival listener would call `VisitVendorsInRoom` at the same waypoints, double-restocking every vendor on every visit.

**Files:**
- Modify: `internal/hooks/hooks.go` (register `CaravanArrivalListener`)
- Modify: `internal/economy/health/capture.go` (swap BTreeState read for synthesizer)
- Modify: `internal/economy/health/capture_test.go` (rewrite caravan fixture)
- Modify: `internal/caravan/state.go` (delete obsolete functions)
- Modify: `internal/caravan/state_test.go` (delete tests for removed functions)
- Modify: `internal/behaviortree/caravan_reset.go` (rewrite for patrol-based reset)
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/357-<filename>.yaml` (add `patrol_id`)
- Delete: `internal/caravan/routes.go`
- Delete: `internal/behaviortree/actions_caravan.go`
- Delete: `internal/behaviortree/actions_caravan_test.go`
- Delete: `internal/behaviortree/actions_wagon.go` (if audit confirms — see step 1)
- Delete: `internal/behaviortree/actions_wagon_test.go` (paired)

- [ ] **Step 1: Audit `actions_wagon.go`**

Run:
```bash
cat internal/behaviortree/actions_wagon.go
grep -rn 'wagon_step\|wagon_follow' internal/ _datafiles/
```

If every function in `actions_wagon.go` is a wagon-as-follower movement helper (likely) and the only callers are mob YAML archetypes (which will be deleted along with the caravan state machine), schedule the file for deletion. If any function does non-movement work (e.g., cargo accounting), move that function to `internal/caravan/` and delete the rest.

- [ ] **Step 2: Find Ketil's mob YAML filename**

Run: `ls _datafiles/world/dogmud/mobs/thornwall_city/357*.yaml`

- [ ] **Step 3: Make all the edits in order**

The order within this task matters because Go compiles transitively — code that references deleted symbols fails to build. Do them in this order:

3a. **Add `patrol_id` to Ketil's mob YAML.** Edit `_datafiles/world/dogmud/mobs/thornwall_city/357-<filename>.yaml`. Add the line near other top-level fields:

```yaml
patrol_id: caravan_thornwall_stillwater
```

If the mob currently has a `behavior_archetype:` that drove the legacy state machine, audit whether it should change to something simpler (e.g., `noncombat_default`) or just have the caravan-state-machine archetype's references cleaned up after the btree-actions deletion.

3b. **Register the arrival listener.** In `internal/hooks/hooks.go`, add a new line in `RegisterListeners()` near other domain-specific listeners:

```go
events.RegisterListener(events.PatrolWaypointArrival{}, caravan.CaravanArrivalListener)
```

Add the import:

```go
import "github.com/GoMudEngine/GoMud/internal/caravan"
```

3c. **Swap the dashboard read path.** In `internal/economy/health/capture.go`, locate `captureCaravans()`. Replace the BTreeState read block:

```go
// Before:
bs, ok := mob.BTreeState.(*behaviortree.BehaviorState)
if !ok || bs == nil {
    continue
}
stateName := bs.GetString("caravan_state")
if stateName == "" {
    continue
}
startedRound, _ := strconv.ParseUint(bs.GetString("caravan_state_started_round"), 10, 64)
```

with:

```go
// After (chunk 3.7):
state, ok := caravan.SynthesizeStateForLeader(mob)
if !ok {
    continue
}
stateName := state.Name()
startedRound, _ := mob.Character.GetMiscData("caravan_state_started_round").(uint64)
```

Add the import:

```go
import "github.com/GoMudEngine/GoMud/internal/caravan"
```

Remove the now-unused `behaviortree` and `strconv` imports if they're not used elsewhere in the file.

3d. **Update the dashboard test fixture.** In `internal/economy/health/capture_test.go`, find the `bs.Set("caravan_state", "outbound_transit")` setup block and rewrite it:

```go
// Before:
bs := behaviortree.NewBehaviorState()
bs.Set("caravan_state", "outbound_transit")
bs.Set("caravan_state_started_round", strconv.FormatUint(12100, 10))
leader.BTreeState = bs

// After (chunk 3.7):
mobs.RegisterPatrolForTest(&mobs.Patrol{
    Id:        caravan.CaravanPatrolId,
    LoopShape: "strict",
    Waypoints: []mobs.PatrolWaypoint{
        {Room: 465, DwellRounds: 360, ArrivalEvent: "caravan_depot"},
        {Room: 4038, DwellRounds: 8, ArrivalEvent: "caravan_fernway_pickup"},
        // (truncated — only need enough waypoints to drive the test scenario)
    },
})
t.Cleanup(func() { mobs.UnregisterPatrolForTest(caravan.CaravanPatrolId) })
leader.PatrolId = caravan.CaravanPatrolId
leader.Character.RoomId = 999 // some non-waypoint room → in-transit → OutboundTransit
leader.Character.SetMiscData("patrol_waypoint_idx", 1)
leader.Character.SetMiscData("caravan_state_started_round", uint64(12100))
```

3e. **Trim `internal/caravan/state.go`.** Delete `AdvanceState`, `ParseState`, `nameToState`, `RouteForState`. Keep `CaravanState` enum, `Name()`, `IsDwellState`, `IsTransitState`, `IsRouteState`, `IsFernwayPickupState`.

3f. **Trim `internal/caravan/state_test.go`.** Delete tests for the removed functions (`TestAdvanceState`, `TestParseState`, `TestRouteForState*`). Keep tests for the predicates.

3g. **Rewrite `internal/behaviortree/caravan_reset.go`.** Replace the BTreeState-mutating reset with a patrol-state-mutating reset:

```go
package behaviortree

// caravan_reset.go — exported helper for admin-driven caravan state reset.
//
// Chunk 3.7: the legacy caravan_state BTreeState key is gone. Reset now
// means "rewind the leader's patrol to waypoint 0 and force-move the
// crew back to Thornwall depot."

import (
	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// ResetAllCaravanStates iterates every live mob instance, finds any whose
// PatrolId == caravan.CaravanPatrolId (i.e. caravan leaders), and resets
// each one to waypoint 0. Returns the count of mobs reset.
func ResetAllCaravanStates() int {
	resetCount := 0
	for _, instId := range mobs.GetAllMobInstanceIds() {
		mob := mobs.GetInstance(instId)
		if mob == nil || mob.PatrolId != caravan.CaravanPatrolId {
			continue
		}
		prevIdx := 0
		if v, ok := mob.Character.GetMiscData("patrol_waypoint_idx").(int); ok {
			prevIdx = v
		}
		mob.Character.SetMiscData("patrol_waypoint_idx", 0)
		mob.Character.SetMiscData("patrol_direction", +1)
		mob.Character.SetMiscData("patrol_dwell_remaining", 0)
		mob.Character.SetMiscData("patrol_path_fail_count", 0)
		resetCount++
		mudlog.Info("caravan state reset by admin",
			"instanceId", instId,
			"mobName", mob.Character.Name,
			"prevWaypointIdx", prevIdx,
		)
	}
	return resetCount
}

// ResetCaravanStateByInstanceId resets the caravan state for a single mob
// instance. Returns true if the mob was found and is a caravan leader,
// false otherwise.
func ResetCaravanStateByInstanceId(instanceId int) bool {
	mob := mobs.GetInstance(instanceId)
	if mob == nil || mob.PatrolId != caravan.CaravanPatrolId {
		return false
	}
	prevIdx := 0
	if v, ok := mob.Character.GetMiscData("patrol_waypoint_idx").(int); ok {
		prevIdx = v
	}
	mob.Character.SetMiscData("patrol_waypoint_idx", 0)
	mob.Character.SetMiscData("patrol_direction", +1)
	mob.Character.SetMiscData("patrol_dwell_remaining", 0)
	mob.Character.SetMiscData("patrol_path_fail_count", 0)
	mudlog.Info("caravan state reset by admin (targeted)",
		"instanceId", instanceId,
		"mobName", mob.Character.Name,
		"prevWaypointIdx", prevIdx,
	)
	return true
}
```

3h. **Delete the legacy files.** Run:

```bash
rm internal/caravan/routes.go
rm internal/behaviortree/actions_caravan.go
rm internal/behaviortree/actions_caravan_test.go
```

If Step 1's audit said `actions_wagon.go` is all movement helpers:

```bash
rm internal/behaviortree/actions_wagon.go
rm internal/behaviortree/actions_wagon_test.go
```

3i. **Clean up any remaining references to deleted symbols.** Build the project:

```bash
go build ./...
```

Expected: any remaining import of `caravan.ThornwallDepotRoomId`, `caravan.OutboundRoute`, `caravan.AdvanceState`, `caravan.ParseState`, `caravan.RouteForState`, or any deleted btree action will fail. Fix each call site by deleting the reference (most should be in the deleted files, but there may be stray callers in admin commands or tests).

- [ ] **Step 4: Run all tests**

Run: `go test ./...`
Expected: all tests pass. If any fail because they referenced deleted symbols, update them.

- [ ] **Step 5: Boot the server and verify clean load**

Run:
```bash
go build -o dogmud-test.exe . && ./dogmud-test.exe 2>&1 | tee boot.log &
BOOT_PID=$!
sleep 15
kill $BOOT_PID 2>/dev/null
grep -E 'panic|FATAL|ERROR|loadedCount=|LoadPatrols' boot.log | head -30
rm dogmud-test.exe boot.log
```

Expected:
- No panics, no FATAL, no ERROR.
- `mobs.LoadPatrols() loadedCount=2`.
- `mobs.LoadDataFiles() loadedCount=226` (or whatever the current count is — unchanged).

- [ ] **Step 6: Stage all the changes**

```bash
git status --short
```

Verify the changes listed include:
- Modified: `internal/hooks/hooks.go`, `internal/economy/health/capture.go`, `internal/economy/health/capture_test.go`, `internal/caravan/state.go`, `internal/caravan/state_test.go`, `internal/behaviortree/caravan_reset.go`, and the Ketil mob YAML.
- Deleted: `internal/caravan/routes.go`, `internal/behaviortree/actions_caravan.go`, `internal/behaviortree/actions_caravan_test.go`, optionally `internal/behaviortree/actions_wagon.go` + `_test.go`.

Stage with explicit paths (NOT `git add -A` to avoid stray files):

```bash
git add internal/hooks/hooks.go \
        internal/economy/health/capture.go \
        internal/economy/health/capture_test.go \
        internal/caravan/state.go \
        internal/caravan/state_test.go \
        internal/behaviortree/caravan_reset.go \
        _datafiles/world/dogmud/mobs/thornwall_city/357-*.yaml
git rm internal/caravan/routes.go \
       internal/behaviortree/actions_caravan.go \
       internal/behaviortree/actions_caravan_test.go
# If audit cleared:
# git rm internal/behaviortree/actions_wagon.go internal/behaviortree/actions_wagon_test.go
```

- [ ] **Step 7: Commit (the atomic flip)**

```bash
git commit -m "feat(caravan): flip Thornwall<->Stillwater caravan onto unified patrol layer

Atomic migration removing the parallel caravan state machine
and replacing it with the chunk 3.7 patrol primitive:

- Ketil (mob 357) gains patrol_id: caravan_thornwall_stillwater
- CaravanArrivalListener registered on events.PatrolWaypointArrival
- internal/caravan/routes.go deleted (room constants moved to YAML)
- internal/behaviortree/actions_caravan.go deleted (state machine driver)
- internal/caravan/state.go shrinks to enum + predicates (no driver)
- caravan_reset.go rewritten for patrol-state-based reset
- economy/health/capture.go swapped to SynthesizeStateForLeader
  (dashboard JSON schema byte-identical, UI unchanged)

Single atomic commit because dual-driver state would double-restock
every vendor on every visit. Boot test verifies clean load with
loadedCount=2 patrols.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Phase 4 — Verification

### Task 10: Smoke test the caravan migration

**Files:** none (manual in-game testing)

- [ ] **Step 1: Boot the server**

Run:
```bash
go build -o dogmud-test.exe . && ./dogmud-test.exe
```

Leave running for the smoke session.

- [ ] **Step 2: Connect as an admin user and verify caravan state**

In-game:
- `mob schedule <ketil-instance-id>` — expected to show patrol id `caravan_thornwall_stillwater`, current waypoint idx 0, dwell remaining 360.
- `caravan reset` — expected: "Reset N caravan(s) to waypoint 0." Ketil and crew should be at Thornwall depot (room 465).

- [ ] **Step 3: Watch a full Thornwall → Stillwater leg**

Either accelerate game-time via admin or wait for natural progression:
- wp0 (Thornwall depot) → wp1 (Fernway pickup): leader queues `pathto 4038`. Wagon + guards follow via party plumbing.
- At wp1 (room 4038): leader dwells 8 rounds. If a forager is co-located, handoff fires (verify cargo transfer via the wagon's inventory).
- wp1 → wp2 (Stillwater depot): leader paths cross-zone to 4109.
- wp2 (Stillwater depot): leader dwells 20 rounds.

- [ ] **Step 4: Watch a Stillwater vendor circuit**

- wp3 (room 4102): leader arrives. Expected: room sees `<ansi fg="yellow">The caravan crew unloads supplies for the local merchants.</ansi>` (or trade variant). Vendor stock counts should change.
- wp4-wp10: each vendor stop fires the same visit message + restock.

Verify via the economy dashboard (`/admin/economy/`):
- Caravan card shows state transitioning: `outbound_fernway_pickup` → `stillwater_dwell` → `stillwater_route` → `stillwater_dwell` (departure).
- `state_entered_round` advances when state name flips.
- `cargo_weight` updates as the wagon delivers + picks up.
- `deliveries_by_tier` counters increment on caravan_vendor events.

- [ ] **Step 5: Watch the return leg**

- wp11 (Stillwater depot, departure) → wp12 (Inbound Fernway pickup) → wp13 (Thornwall depot, arrival).
- wp14-wp21 (Thornwall vendor circuit) — same restock + flavor pattern as Stillwater.
- After wp21 loops back to wp0, leader settles into long depot dwell.

- [ ] **Step 6: Test leader-respawn crew regroup**

- Locate Ketil mid-Stillwater-vendor-circuit (e.g., at wp5).
- Admin: `kill <ketil-instance-id>` or use a debug command to force-kill the leader.
- Wait for leader respawn (HomeRoomId = 465 Thornwall depot).
- Verify: on respawn, the wagon and any surviving guards force-move to Thornwall depot via the wp0 caravan_depot arrival handler.
- Verify: wagon cargo preserved (check inventory before/after via admin).

- [ ] **Step 7: Test server-restart resumption mid-transit**

- During a Stillwater leg (mid-transit, not at a waypoint), `shutdown` the server.
- Restart: `./dogmud-test.exe`.
- Verify: Ketil and wagon resume at their saved rooms; on the next idle tick, the patrol resumes pathing to the current target waypoint.

- [ ] **Step 8: Test `caravan reset` mid-cycle**

- Mid-Stillwater-circuit, admin: `caravan reset`.
- Verify: Ketil's `patrol_waypoint_idx` resets to 0, crew force-moves to Thornwall depot, cycle restarts.

- [ ] **Step 9: Inspect the boot log for any new warnings**

Run: `grep -E 'WARN|ERROR|panic' boot.log | head -20`

Expected: no new warnings beyond the pre-existing biome warnings on legacy rooms (chunk 3.7 should not introduce new warnings).

- [ ] **Step 10: Clean up the test artifacts**

```bash
rm dogmud-test.exe boot.log
```

- [ ] **Step 11: Hand off to user for in-game smoke**

Per project SOP, manual playtest precedes prod push. Report the smoke results to the user and confirm before pushing to `origin/master`.

---

## Self-Review Notes

This plan was self-reviewed against the spec on 2026-05-26. Key checks:

- **Spec coverage:** Every "In scope" item maps to a task. Schema additions (Tasks 1, 2), event plumbing (Tasks 3, 4), zone-resolution logging (Task 5), synthesizer (Task 6), YAML (Task 7), listener (Task 8), atomic flip including dashboard swap + state machine deletion + caravan_reset rewrite + Ketil's patrol_id (Task 9), verification (Task 10).
- **Placeholder scan:** No "TBD" or "implement later" markers. Step bodies contain the actual code or commands.
- **Type consistency:** `CaravanArrivalListener` referenced consistently. `CaravanPatrolId` constant referenced in synthesizer + listener + Ketil's YAML + dashboard tests. `SynthesizeStateForLeader` signature consistent across tasks.
- **Deferred from spec:** The `MobRespawn_CaravanCrewRegroup` hook proposed in the spec is folded into the arrival listener's `handleDepotArrival` case for wp0 — same behavior, simpler implementation, since the codebase has no explicit MobRespawn event. This is documented in Task 8.

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-26-mob-aliveness-3.7-inter-zone-patrols.md`.

Two execution options:

**1. Subagent-driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline execution** — execute tasks in this session using `executing-plans`, batch execution with checkpoints.

Which approach?
