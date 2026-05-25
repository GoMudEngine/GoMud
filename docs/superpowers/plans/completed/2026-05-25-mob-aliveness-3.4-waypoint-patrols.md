# Mob Aliveness 3.4 — Waypoint Patrols Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Authored multi-room patrol routes for NPCs. Standalone (mob has `patrol_id`) OR composed with chunk 3.2 schedules via `activity: patrol`. Two loop shapes (strict + yo-yo) with per-waypoint dwell. Combat interrupts, resume to same target waypoint.

**Architecture:** Mirrors chunk 3.2's pattern almost exactly — new `Patrol` type + per-zone YAML files + loader with DI for world validation + executor that runs in `NewRound_IdleMobs` after the schedule branch. Per-mob runtime state stored in MiscData (waypoint index, direction, dwell remaining, path retry count). Composition with schedules works via a single MiscData breadcrumb: schedule executor stamps `active_patrol_id` for the patrol executor to consume the same tick.

**Tech Stack:** Go 1.24. New files in `internal/mobs/` (patrol type + loader) and `internal/hooks/` (executor). Modifications to `internal/mobs/mobs.go` (PatrolId field), `internal/mobs/schedule.go` (ScheduleSegment.PatrolId), `internal/mobs/schedule_loader.go` (activity: patrol recognition), `internal/hooks/NewRound_IdleMobs_schedule.go` (stamps active_patrol_id), `internal/hooks/NewRound_IdleMobs.go` (patrol branch wiring), `internal/usercommands/admin.mob.go` (inspector extension), `main.go` (DI wiring). Content YAMLs in `_datafiles/world/dogmud/patrols/thornwall_city/` and existing schedules / mobs / rooms folders.

**Spec:** `docs/superpowers/specs/2026-05-25-mob-aliveness-3.4-waypoint-patrols-design.md`

**Branch:** `feature/mob-aliveness-3.4-waypoint-patrols` (already created; spec committed as `4f78794c`).

---

## Stage map

| Stage | Task | Description |
|---|---|---|
| 1 | T1 | `Patrol` + `PatrolWaypoint` types + registry + `NextWaypoint` helper (TDD) |
| 2 | T2 | `LoadPatrols` + standalone validators + DI for world checks |
| 3 | T3 | `main.go` wires `SetPatrolWorldValidator` |
| 4 | T4 | `PatrolId` field on Mob + load-time cross-check |
| 5 | T5 | `ScheduleSegment.PatrolId` field + schedule loader recognizes `activity: patrol` + optional `target_room` |
| 6 | T6 | Schedule executor stamps `active_patrol_id` on patrol-activity segment ticks |
| 7 | T7 | Spawn override falls back to patrol's first waypoint when segment has no `target_room` |
| 8 | T8 | `patrolTickPlan` + `applyPatrolPlan` (TDD) |
| 9 | T9 | `NewRound_IdleMobs` wires the patrol branch after the schedule branch |
| 10 | T10 | `mob schedule` admin inspector shows patrol state |
| 11 | T11 | Pilot content — patrol YAML + schedule YAML + barracks room + mob YAML edit |
| 12 | T12 | Documentation pass (schema, context.md, CLAUDE.md, helpfile) |
| 13 | T13 | Smoketester goal file + roadmap closeout |

13 tasks. Sequential ordering preferred — T1 is foundation for T2/T4/T8; T5 needs T2 done; T6 needs T5; T7 needs T2; T8/T9 need T1+T4; T10 needs T1+T4; T11 needs T2+T5+T7+T9; T12+T13 documentation/closeout.

---

## Task 1: `Patrol` + `PatrolWaypoint` types + `NextWaypoint` helper

**Files:**
- Create: `internal/mobs/patrol.go`
- Create: `internal/mobs/patrol_test.go`

- [ ] **Step 1: Read the chunk 3.2 schedule type for the template**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,120p' internal/mobs/schedule.go
```

Confirm the struct + registry + helper pattern. T1 mirrors this almost line-for-line, swapping `Schedule` for `Patrol` and adding loop-shape semantics.

- [ ] **Step 2: Write the failing tests**

Create `internal/mobs/patrol_test.go`:

```go
package mobs

import (
	"testing"
)

// Fixture: 4-waypoint patrol covering all combinations.
//
// rooms: 100 → 101 → 102 → 103
// dwell: 5 / 10 / 5 / 0
func fourWaypointPatrol(loopShape string) *Patrol {
	return &Patrol{
		Id:          "test_patrol",
		Description: "4-waypoint fixture",
		LoopShape:   loopShape,
		Waypoints: []PatrolWaypoint{
			{Room: 100, DwellRounds: 5},
			{Room: 101, DwellRounds: 10},
			{Room: 102, DwellRounds: 5},
			{Room: 103, DwellRounds: 0},
		},
	}
}

func TestPatrol_NextWaypoint_StrictLoop(t *testing.T) {
	p := fourWaypointPatrol("strict")
	cases := []struct {
		currentIdx int
		direction  int
		wantIdx    int
		wantDir    int
	}{
		{0, +1, 1, +1},
		{1, +1, 2, +1},
		{2, +1, 3, +1},
		{3, +1, 0, +1}, // wrap to start
	}
	for _, c := range cases {
		gotIdx, gotDir := p.NextWaypoint(c.currentIdx, c.direction)
		if gotIdx != c.wantIdx || gotDir != c.wantDir {
			t.Errorf("strict from idx=%d dir=%+d: want (%d, %+d), got (%d, %+d)",
				c.currentIdx, c.direction, c.wantIdx, c.wantDir, gotIdx, gotDir)
		}
	}
}

func TestPatrol_NextWaypoint_YoYoForwardAndReverse(t *testing.T) {
	p := fourWaypointPatrol("yo-yo")
	// Forward up to the end, then flip and reverse back.
	cases := []struct {
		currentIdx int
		direction  int
		wantIdx    int
		wantDir    int
	}{
		{0, +1, 1, +1},
		{1, +1, 2, +1},
		{2, +1, 3, +1},
		{3, +1, 2, -1}, // at last, flip to reverse
		{2, -1, 1, -1},
		{1, -1, 0, -1},
		{0, -1, 1, +1}, // at first going reverse, flip to forward
	}
	for _, c := range cases {
		gotIdx, gotDir := p.NextWaypoint(c.currentIdx, c.direction)
		if gotIdx != c.wantIdx || gotDir != c.wantDir {
			t.Errorf("yo-yo from idx=%d dir=%+d: want (%d, %+d), got (%d, %+d)",
				c.currentIdx, c.direction, c.wantIdx, c.wantDir, gotIdx, gotDir)
		}
	}
}

func TestPatrol_NextWaypoint_StrictEmptyDirectionIgnored(t *testing.T) {
	// Strict loops ignore direction — always +1, always wraps end → start.
	p := fourWaypointPatrol("strict")
	// Even if direction is -1 (shouldn't happen for strict, defensive), strict
	// behaves as forward.
	gotIdx, gotDir := p.NextWaypoint(3, -1)
	if gotIdx != 0 || gotDir != +1 {
		t.Errorf("strict with dir=-1: want (0, +1), got (%d, %+d)", gotIdx, gotDir)
	}
}

func TestPatrol_NextWaypoint_DefaultsToStrictWhenLoopShapeEmpty(t *testing.T) {
	p := fourWaypointPatrol("") // empty loop_shape → defaults to strict
	gotIdx, gotDir := p.NextWaypoint(3, +1)
	if gotIdx != 0 || gotDir != +1 {
		t.Errorf("empty loop_shape with idx=3 dir=+1: want (0, +1), got (%d, %+d)",
			gotIdx, gotDir)
	}
}

func TestGetPatrol_EmptyReturnsNil(t *testing.T) {
	if got := GetPatrol(""); got != nil {
		t.Errorf("GetPatrol(\"\"): want nil, got %+v", got)
	}
}

func TestGetPatrol_UnknownReturnsNil(t *testing.T) {
	if got := GetPatrol("definitely_not_real"); got != nil {
		t.Errorf("unknown patrol id: want nil, got %+v", got)
	}
}
```

- [ ] **Step 3: Run the test, confirm fail**

```bash
go test ./internal/mobs/ -run "TestPatrol_|TestGetPatrol_" -v
```

Expected: compilation error — `Patrol`, `PatrolWaypoint`, `GetPatrol`, `NextWaypoint` not defined.

- [ ] **Step 4: Implement `internal/mobs/patrol.go`**

```go
package mobs

import "sync"

// Patrol is a looped multi-room route attached to NPCs via Mob.PatrolId
// or via a schedule segment with activity: patrol + patrol_id.
// Loaded from _datafiles/world/dogmud/patrols/<zone>/<id>.yaml at startup.
type Patrol struct {
	Id          string           `yaml:"id"`
	Description string           `yaml:"description,omitempty"`
	LoopShape   string           `yaml:"loop_shape,omitempty"` // "strict" (default) | "yo-yo"
	Waypoints   []PatrolWaypoint `yaml:"waypoints"`
}

// PatrolWaypoint is one stop on a patrol route.
type PatrolWaypoint struct {
	Room        int `yaml:"room"`
	DwellRounds int `yaml:"dwell_rounds,omitempty"` // 0 = move on immediately
}

// Package-level registry, populated by LoadPatrols at startup.
var (
	patrolsMu sync.RWMutex
	patrols   = map[string]*Patrol{}
)

// GetPatrol returns the patrol with the given id, or nil if no such id is
// loaded.
func GetPatrol(id string) *Patrol {
	if id == "" {
		return nil
	}
	patrolsMu.RLock()
	defer patrolsMu.RUnlock()
	return patrols[id]
}

// NextWaypoint returns the next (waypoint index, direction) given the
// current index and direction. Direction is meaningful only for yo-yo
// patrols; strict patrols always loop forward.
//
// For strict: increments idx, wraps to 0 at the end, always returns dir=+1.
// For yo-yo: increments by dir, flips dir at endpoints (0 or last).
//
// Caller is responsible for handling the "current dwell expired, time to
// advance" decision; this function is purely about computing the next
// position in the route.
func (p *Patrol) NextWaypoint(currentIdx, currentDir int) (nextIdx, nextDir int) {
	if p == nil || len(p.Waypoints) == 0 {
		return 0, +1
	}

	loop := p.LoopShape
	if loop == "" {
		loop = "strict"
	}

	if loop != "yo-yo" {
		// strict (default + any unrecognized value)
		next := currentIdx + 1
		if next >= len(p.Waypoints) {
			next = 0
		}
		return next, +1
	}

	// yo-yo
	if currentDir == 0 {
		currentDir = +1
	}
	next := currentIdx + currentDir
	if next >= len(p.Waypoints) {
		// At the last waypoint going forward — flip to reverse.
		return len(p.Waypoints) - 2, -1
	}
	if next < 0 {
		// At the first waypoint going reverse — flip to forward.
		return 1, +1
	}
	return next, currentDir
}
```

- [ ] **Step 5: Run the test, confirm pass**

```bash
go test ./internal/mobs/ -run "TestPatrol_|TestGetPatrol_" -v
```

Expected: all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/patrol.go internal/mobs/patrol_test.go
git commit -m "$(cat <<'EOF'
feat(mobs): Patrol type + NextWaypoint helper

Foundation for chunk 3.4 waypoint patrols. Patrol + PatrolWaypoint
types, package-level registry, GetPatrol(id) accessor, and the
NextWaypoint(idx, dir) loop helper handling both strict
(wrap-to-start) and yo-yo (flip at endpoints) shapes. No loader
yet; tests use in-memory fixtures.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: `LoadPatrols` + validators + DI for world checks

**Files:**
- Create: `internal/mobs/patrol_loader.go`
- Create: `internal/mobs/patrol_loader_test.go`
- Modify: `internal/mobs/mobs.go` (wire `LoadPatrols()` into `LoadDataFiles()` after `LoadSchedules()`)

- [ ] **Step 1: Re-read the chunk 3.2 schedule loader for the template**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,160p' internal/mobs/schedule_loader.go
```

The patrol loader mirrors this near-exactly: DI struct + SetXxxWorldValidator function + LoadXxx() that walks files / parses / validates / panics-on-error. Match the style.

- [ ] **Step 2: Write the failing tests**

Create `internal/mobs/patrol_loader_test.go`:

```go
package mobs

import (
	"strings"
	"testing"
)

func TestValidatePatrol_OK(t *testing.T) {
	p := fourWaypointPatrol("strict")
	if err := validatePatrolStandalone(p); err != nil {
		t.Errorf("4-waypoint fixture should validate, got: %v", err)
	}
}

func TestValidatePatrol_EmptyWaypoints(t *testing.T) {
	p := &Patrol{Id: "broken", Waypoints: nil}
	err := validatePatrolStandalone(p)
	if err == nil || !strings.Contains(err.Error(), "waypoints") {
		t.Errorf("expected waypoints error, got: %v", err)
	}
}

func TestValidatePatrol_NegativeDwell(t *testing.T) {
	p := &Patrol{
		Id: "broken",
		Waypoints: []PatrolWaypoint{
			{Room: 100, DwellRounds: -1},
		},
	}
	err := validatePatrolStandalone(p)
	if err == nil || !strings.Contains(err.Error(), "dwell") {
		t.Errorf("expected dwell error, got: %v", err)
	}
}

func TestValidatePatrol_ZeroRoom(t *testing.T) {
	p := &Patrol{
		Id: "broken",
		Waypoints: []PatrolWaypoint{
			{Room: 0, DwellRounds: 5},
		},
	}
	err := validatePatrolStandalone(p)
	if err == nil || !strings.Contains(err.Error(), "room") {
		t.Errorf("expected room error, got: %v", err)
	}
}

func TestValidatePatrol_BadLoopShape(t *testing.T) {
	p := fourWaypointPatrol("not-a-real-shape")
	err := validatePatrolStandalone(p)
	if err == nil || !strings.Contains(err.Error(), "loop_shape") {
		t.Errorf("expected loop_shape error, got: %v", err)
	}
}

func TestValidatePatrol_SingleWaypoint_WarnsButValidates(t *testing.T) {
	p := &Patrol{
		Id: "lonely",
		Waypoints: []PatrolWaypoint{
			{Room: 100, DwellRounds: 5},
		},
	}
	// Single-waypoint patrols are warn-only (mob stands and dwells forever).
	// Validator returns nil; warning goes to logs.
	if err := validatePatrolStandalone(p); err != nil {
		t.Errorf("single-waypoint patrol should validate (warn-only), got: %v", err)
	}
}

// World-aware tests require a rooms registry; integration via T13 boot smoke.
func TestValidatePatrolAgainstWorld_Stub(t *testing.T) {
	t.Skip("requires rooms + mapper fixtures — covered by boot smoke in T13")
}
```

- [ ] **Step 3: Run the test, confirm fail**

```bash
go test ./internal/mobs/ -run "TestValidatePatrol" -v
```

Expected: compile error — `validatePatrolStandalone` not defined.

- [ ] **Step 4: Implement `internal/mobs/patrol_loader.go`**

```go
package mobs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// PatrolWorldValidator is injected at startup (after both rooms and mapper
// are initialized) to perform world-aware patrol validation without
// creating an import cycle. mobs ← rooms ← mobs would be a cycle. Same
// pattern as scheduleWorldValidator from chunk 3.2.
//
// roomExists(id) must return true if the room is in the registry.
// hasPath(from, to) must return true if the mapper can route between the rooms.
// Call SetPatrolWorldValidator before LoadPatrols if world-aware checks
// are desired; omitting it skips the world-aware pass silently.
var patrolWorldValidator struct {
	roomExists func(id int) bool
	hasPath    func(from, to int) bool
}

// SetPatrolWorldValidator wires in the room-existence and pathfinding
// checks for patrol validation. Must be called before LoadPatrols.
func SetPatrolWorldValidator(roomExists func(int) bool, hasPath func(from, to int) bool) {
	patrolWorldValidator.roomExists = roomExists
	patrolWorldValidator.hasPath = hasPath
}

// LoadPatrols walks _datafiles/world/dogmud/patrols/**/*.yaml, parses
// each file into a *Patrol, validates it, and registers it in the
// package-level patrols map. Duplicate ids and validation failures cause
// a panic. If the patrols directory does not exist, the function logs
// and returns without error — patrols are optional content.
func LoadPatrols() {
	start := time.Now()

	dataPath := configs.GetFilePathsConfig().DataFiles.String() + `/patrols`

	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		mudlog.Info("mobs.LoadPatrols()", "loadedCount", 0,
			"note", "patrols directory does not exist — skipping",
			"Time Taken", time.Since(start))
		return
	}

	tmp := map[string]*Patrol{}

	walkErr := filepath.Walk(dataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("patrol: read %s: %w", path, readErr)
		}

		var p Patrol
		if unmarshalErr := yaml.Unmarshal(data, &p); unmarshalErr != nil {
			return fmt.Errorf("patrol: unmarshal %s: %w", path, unmarshalErr)
		}

		// Filename ↔ id check.
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		expected := util.ConvertForFilename(p.Id)
		if base != expected {
			return fmt.Errorf(
				"patrol: filename mismatch: file %q has base name %q but expected %q (derived from id %q)",
				path, base, expected, p.Id,
			)
		}

		if valErr := validatePatrolStandalone(&p); valErr != nil {
			return fmt.Errorf("patrol %q (%s): %w", p.Id, path, valErr)
		}

		// Warn on degenerate single-waypoint patrols.
		if len(p.Waypoints) == 1 {
			mudlog.Warn("mobs.LoadPatrols()",
				"patrolId", p.Id,
				"warning", "single-waypoint patrol — mob will stand and dwell forever (likely author error)")
		}

		if _, dup := tmp[p.Id]; dup {
			return fmt.Errorf("patrol: duplicate id %q in %s", p.Id, path)
		}

		pp := p // capture
		tmp[p.Id] = &pp
		return nil
	})

	if walkErr != nil {
		panic(fmt.Sprintf("mobs.LoadPatrols() failed: %v", walkErr))
	}

	// World-aware validation: confirm rooms exist and inter-waypoint
	// pairs are reachable via the mapper. Only runs when both validator
	// functions are wired (matches chunk 3.2 schedule pattern).
	if patrolWorldValidator.roomExists != nil && patrolWorldValidator.hasPath != nil {
		for _, p := range tmp {
			if valErr := validatePatrolAgainstWorld(p,
				patrolWorldValidator.roomExists,
				patrolWorldValidator.hasPath,
			); valErr != nil {
				panic(fmt.Sprintf(
					"mobs.LoadPatrols() world validation failed for patrol %q: %v",
					p.Id, valErr,
				))
			}
		}
	}

	patrolsMu.Lock()
	patrols = tmp
	patrolsMu.Unlock()

	mudlog.Info("mobs.LoadPatrols()", "loadedCount", len(tmp), "Time Taken", time.Since(start))
}

// validatePatrolStandalone checks a patrol for internal consistency
// without touching the rooms registry or filesystem. Suitable for unit
// tests.
//
// Rules:
//   - At least one waypoint.
//   - Each waypoint's Room is non-zero.
//   - Each waypoint's DwellRounds is non-negative.
//   - LoopShape is "" / "strict" / "yo-yo".
func validatePatrolStandalone(p *Patrol) error {
	if len(p.Waypoints) == 0 {
		return errors.New("patrol has no waypoints")
	}

	if p.LoopShape != "" && p.LoopShape != "strict" && p.LoopShape != "yo-yo" {
		return fmt.Errorf("invalid loop_shape %q (want \"\" | \"strict\" | \"yo-yo\")", p.LoopShape)
	}

	for i, w := range p.Waypoints {
		if w.Room == 0 {
			return fmt.Errorf("waypoint %d: room is 0 (must be a valid room id)", i)
		}
		if w.DwellRounds < 0 {
			return fmt.Errorf("waypoint %d: dwell_rounds=%d is negative", i, w.DwellRounds)
		}
	}

	return nil
}

// validatePatrolAgainstWorld checks every target room exists and that
// consecutive waypoint pairs are reachable via the mapper. For strict
// loops, also checks the wrap (last → first). Injected functions break
// the import cycle.
func validatePatrolAgainstWorld(p *Patrol, roomExists func(int) bool, hasPath func(from, to int) bool) error {
	for i, w := range p.Waypoints {
		if !roomExists(w.Room) {
			return fmt.Errorf("waypoint %d: room %d does not exist in rooms registry", i, w.Room)
		}
	}

	// Inter-waypoint pathfinding sanity.
	for i := 0; i < len(p.Waypoints)-1; i++ {
		from := p.Waypoints[i].Room
		to := p.Waypoints[i+1].Room
		if from == to {
			continue
		}
		if !hasPath(from, to) {
			return fmt.Errorf("no path from waypoint %d (room %d) to waypoint %d (room %d)",
				i, from, i+1, to)
		}
	}

	// For strict loops, also check the wrap (last → first).
	loop := p.LoopShape
	if loop == "" {
		loop = "strict"
	}
	if loop == "strict" && len(p.Waypoints) > 1 {
		from := p.Waypoints[len(p.Waypoints)-1].Room
		to := p.Waypoints[0].Room
		if from != to && !hasPath(from, to) {
			return fmt.Errorf("strict-loop wrap: no path from last waypoint (room %d) to first (room %d)",
				from, to)
		}
	}

	return nil
}
```

- [ ] **Step 5: Wire `LoadPatrols` into `mobs.LoadDataFiles()`**

In `internal/mobs/mobs.go`, find where `LoadSchedules()` is called (chunk 3.2 wiring). Add `LoadPatrols()` immediately after:

```go
LoadSchedules()
LoadPatrols()
```

Both loaders are independent of each other but both depend on rooms being loaded. The ordering doesn't matter between them; placing LoadPatrols after LoadSchedules just for readability.

- [ ] **Step 6: Run tests, confirm pass**

```bash
go test ./internal/mobs/ -v
```

Expected: green. Empty patrols directory at runtime means `LoadPatrols()` logs and returns; no validator fires.

- [ ] **Step 7: Build, confirm clean**

```bash
go build ./...
```

Expected: clean.

- [ ] **Step 8: Commit**

```bash
git add internal/mobs/patrol.go internal/mobs/patrol_test.go internal/mobs/patrol_loader.go internal/mobs/patrol_loader_test.go internal/mobs/mobs.go
git commit -m "$(cat <<'EOF'
feat(mobs): patrol YAML loader + standalone validators

Walks _datafiles/world/dogmud/patrols/**/*.yaml, parses each
file into a *Patrol, validates each (waypoints non-empty,
loop_shape valid, rooms non-zero, dwell non-negative), then
registers by id. World-aware validation (room existence +
mapper pathfinding) uses dependency injection via
SetPatrolWorldValidator to break the rooms ← mobs import cycle.

Single-waypoint patrols emit a warn-only log (degenerate but
not invalid).

LoadPatrols() called from LoadDataFiles() immediately after
LoadSchedules() — mirrors the chunk 3.2 wiring.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: `main.go` wires `SetPatrolWorldValidator`

**Files:**
- Modify: `main.go` (alongside the existing `SetScheduleWorldValidator` wiring)

- [ ] **Step 1: Re-read the existing schedule wiring**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '1147,1170p' main.go
```

Confirm the `SetScheduleWorldValidator` block placement (after `rooms.LoadDataFiles()`, before `mobs.LoadDataFiles()`).

- [ ] **Step 2: Add the patrol wiring**

In `main.go`, immediately after the `mobs.SetScheduleWorldValidator(...)` block and before `mobs.LoadDataFiles()`, add:

```go
mobs.SetPatrolWorldValidator(
	func(roomId int) bool {
		return rooms.LoadRoom(roomId) != nil
	},
	func(from, to int) bool {
		_, err := mapper.GetPath(from, to)
		return err == nil
	},
)
```

`mapper` and `rooms` are already imported in main.go (chunk 3.2 wiring confirmed this).

- [ ] **Step 3: Build, confirm clean**

```bash
go build ./...
```

Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "$(cat <<'EOF'
feat(mobs): wire SetPatrolWorldValidator in main.go

Injects room-existence + pathfinding callbacks for patrol
loader's world-aware validation, mirroring the chunk 3.2
SetScheduleWorldValidator wiring. Same cycle-break reason:
mobs cannot directly import rooms because rooms imports mobs.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: `PatrolId` field on Mob + load-time cross-check

**Files:**
- Modify: `internal/mobs/mobs.go` (add `PatrolId` field to Mob struct, add cross-check loop in `LoadDataFiles`)

- [ ] **Step 1: Add `PatrolId` field to the Mob struct**

In `internal/mobs/mobs.go`, find the `ScheduleId` field added in chunk 3.2. Add `PatrolId` immediately after:

```go
ScheduleId string `yaml:"schedule_id,omitempty"` // chunk 3.2: daily routine reference
PatrolId   string `yaml:"patrol_id,omitempty"`   // chunk 3.4: patrol route reference
```

- [ ] **Step 2: Add a cross-check pass to `LoadDataFiles`**

In `internal/mobs/mobs.go` `LoadDataFiles()`, find the existing chunk 3.2 `schedule_id` cross-check loop. Add a peer block for patrols immediately after:

```go
// Cross-check: every mob's patrol_id must resolve to a loaded patrol.
mobsMu.RLock()
for _, mob := range mobs {
	if mob.PatrolId == "" {
		continue
	}
	if GetPatrol(mob.PatrolId) == nil {
		mobsMu.RUnlock()
		panic(fmt.Errorf("mob %d (%s): patrol_id %q does not resolve to a loaded patrol",
			mob.MobId, mob.Character.Name, mob.PatrolId))
	}
}
mobsMu.RUnlock()
```

`fmt` is already imported.

- [ ] **Step 3: Confirm build and tests still pass**

```bash
go build ./...
go test ./internal/mobs/ -v
```

Expected: clean. Existing mob YAMLs have no `patrol_id`, so the cross-check is a no-op.

- [ ] **Step 4: Commit**

```bash
git add internal/mobs/mobs.go
git commit -m "$(cat <<'EOF'
feat(mobs): patrol_id field on Mob + load-time cross-check

Adds the patrol_id YAML field. LoadDataFiles cross-checks every
patrol_id resolves to a loaded patrol and panics on miss.
Mirrors the chunk 3.2 schedule_id pattern.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: `ScheduleSegment.PatrolId` + schedule loader recognizes `activity: patrol`

**Files:**
- Modify: `internal/mobs/schedule.go` (add `PatrolId` field to `ScheduleSegment`)
- Modify: `internal/mobs/schedule_loader.go` (relax target_room check for patrol activity; require patrol_id when activity is patrol)
- Modify: `internal/mobs/schedule_loader_test.go` (test new validation paths)

- [ ] **Step 1: Add `PatrolId` to `ScheduleSegment`**

In `internal/mobs/schedule.go`, find the `ScheduleSegment` struct definition (around line 15). Add the field:

```go
type ScheduleSegment struct {
	Start        int      `yaml:"start"`
	End          int      `yaml:"end"`
	TargetRoom   int      `yaml:"target_room,omitempty"`     // chunk 3.4: now optional for activity: patrol segments
	Activity     string   `yaml:"activity,omitempty"`
	PatrolId     string   `yaml:"patrol_id,omitempty"`       // chunk 3.4: required when activity is "patrol"
	IdleCommands []string `yaml:"idlecommands,omitempty"`
}
```

Note: the spec's `yaml:"target_room"` tag becomes `yaml:"target_room,omitempty"` so it can be legitimately omitted from patrol segments. The validation logic enforces presence for non-patrol activities.

- [ ] **Step 2: Update the schedule loader to handle the new field**

In `internal/mobs/schedule_loader.go`, find the segment validation loop (the `target_room is 0` check and the activity warning loop). Modify:

The existing `target_room is 0` check needs to be conditional — skip it when `activity: patrol`. Find this block in `validateScheduleStandalone`:

```go
if seg.TargetRoom == 0 {
	return fmt.Errorf("schedule %q segment %d: target_room is 0 (must be a valid room id)",
		s.Id, i)
}
```

Change to:

```go
if seg.TargetRoom == 0 && seg.Activity != "patrol" {
	return fmt.Errorf("schedule %q segment %d: target_room is 0 (must be a valid room id, except for activity: patrol)",
		s.Id, i)
}
```

Then add a new validation: when `activity: patrol`, `PatrolId` must be non-empty. Add this after the bounds/target_room block:

```go
if seg.Activity == "patrol" && seg.PatrolId == "" {
	return fmt.Errorf("schedule %q segment %d: activity is \"patrol\" but patrol_id is empty",
		s.Id, i)
}
```

Find the activity warning block (the one that warns on unknown activities). Add a warning for `patrol_id` set on a non-patrol segment:

```go
if seg.PatrolId != "" && seg.Activity != "patrol" {
	mudlog.Warn("mobs.LoadSchedules()",
		"scheduleId", s.Id,
		"segment", i,
		"warning", "patrol_id set but activity is not \"patrol\" — field has no effect")
}
```

Also: the cross-check for `patrol_id` resolution (every segment's `patrol_id` must reference a loaded patrol) belongs in the post-load step. Add to the existing world-aware validation block in `LoadSchedules` (or as a separate loop after both schedules and patrols load — your choice; the cleanest is a tiny loop right after the world-aware patrol pass).

Add to `LoadSchedules()` near the bottom (after `schedulesMu.Lock(); schedules = tmp; schedulesMu.Unlock();` and before the final `mudlog.Info`):

```go
// Chunk 3.4: cross-check schedule segment patrol_id references.
schedulesMu.RLock()
for _, s := range schedules {
	for i, seg := range s.Segments {
		if seg.PatrolId == "" {
			continue
		}
		if GetPatrol(seg.PatrolId) == nil {
			schedulesMu.RUnlock()
			panic(fmt.Sprintf(
				"mobs.LoadSchedules() schedule %q segment %d: patrol_id %q does not resolve to a loaded patrol (patrols loaded yet?)",
				s.Id, i, seg.PatrolId))
		}
	}
}
schedulesMu.RUnlock()
```

**Important ordering note:** the cross-check assumes `LoadPatrols()` ran BEFORE `LoadSchedules()` so patrols are in the registry. Confirm the order in `LoadDataFiles` (T2 put `LoadPatrols()` after `LoadSchedules()` — REVERSE THIS for T5, or do the cross-check elsewhere).

**Adjustment:** Move `LoadPatrols()` to BEFORE `LoadSchedules()` in `LoadDataFiles()`. The schedule loader needs patrols available for cross-check. Patrols are independent so loading them first is fine.

- [ ] **Step 3: Add the tests**

Append to `internal/mobs/schedule_loader_test.go`:

```go
func TestValidateSchedule_PatrolSegmentWithoutTargetRoom_OK(t *testing.T) {
	s := &Schedule{
		Id: "patrol_schedule_test",
		Segments: []ScheduleSegment{
			{Start: 0, End: 24, Activity: "patrol", PatrolId: "some_patrol",
				IdleCommands: []string{"emote watches."}},
		},
	}
	if err := validateScheduleStandalone(s); err != nil {
		t.Errorf("patrol segment without target_room should validate, got: %v", err)
	}
}

func TestValidateSchedule_PatrolSegmentWithoutPatrolId_Fails(t *testing.T) {
	s := &Schedule{
		Id: "broken_patrol_schedule",
		Segments: []ScheduleSegment{
			{Start: 0, End: 24, TargetRoom: 100, Activity: "patrol", PatrolId: "",
				IdleCommands: []string{"x"}},
		},
	}
	err := validateScheduleStandalone(s)
	if err == nil || !strings.Contains(err.Error(), "patrol_id is empty") {
		t.Errorf("expected patrol_id-empty error, got: %v", err)
	}
}

func TestValidateSchedule_NonPatrolSegmentWithTargetRoomZero_Fails(t *testing.T) {
	// Regression: target_room must still be required for non-patrol activities.
	s := &Schedule{
		Id: "broken",
		Segments: []ScheduleSegment{
			{Start: 0, End: 24, TargetRoom: 0, Activity: "", IdleCommands: []string{"x"}},
		},
	}
	err := validateScheduleStandalone(s)
	if err == nil || !strings.Contains(err.Error(), "target_room is 0") {
		t.Errorf("expected target_room=0 error for non-patrol activity, got: %v", err)
	}
}
```

- [ ] **Step 4: Run tests + build**

```bash
go test ./internal/mobs/ -v
go build ./...
```

Expected: green.

- [ ] **Step 5: Commit**

```bash
git add internal/mobs/schedule.go internal/mobs/schedule_loader.go internal/mobs/schedule_loader_test.go internal/mobs/mobs.go
git commit -m "$(cat <<'EOF'
feat(mobs): schedule recognizes activity: patrol + optional target_room

ScheduleSegment gains a PatrolId field. Schedule loader:
- target_room becomes optional when activity is "patrol"
- patrol_id is required when activity is "patrol"
- patrol_id set on a non-patrol segment emits a warn (no effect)
- post-load cross-check: every segment's patrol_id must resolve
  to a loaded patrol (LoadPatrols moved before LoadSchedules so
  patrols are available for the check)

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Schedule executor stamps `active_patrol_id`

**Files:**
- Modify: `internal/hooks/NewRound_IdleMobs_schedule.go` (stamp `active_patrol_id` when current segment is `activity: patrol`)
- Modify: `internal/hooks/NewRound_IdleMobs_schedule_test.go` (test the stamp)

- [ ] **Step 1: Re-read the existing schedule executor**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '40,165p' internal/hooks/NewRound_IdleMobs_schedule.go
```

The executor has two entry points: `scheduleTickPlan` (pure decision) and `applySchedulePlan` (side-effect applier). T6 stamps the MiscData inside `applySchedulePlan` when the current segment's activity is `patrol`.

- [ ] **Step 2: Write the failing test**

Append to `internal/hooks/NewRound_IdleMobs_schedule_test.go`:

```go
func TestApplySchedulePlan_StampsActivePatrolId_OnPatrolSegment(t *testing.T) {
	mobs.RegisterScheduleForTest(&mobs.Schedule{
		Id: "guard_sched",
		Segments: []mobs.ScheduleSegment{
			{Start: 0, End: 12, Activity: "patrol", PatrolId: "guard_patrol",
				IdleCommands: []string{"watches."}},
			{Start: 12, End: 24, TargetRoom: 9999, Activity: "",
				IdleCommands: []string{"sleeps."}},
		},
	})
	defer mobs.UnregisterScheduleForTest("guard_sched")

	mob := &mobs.Mob{ScheduleId: "guard_sched"}
	mob.Character.RoomId = 1000

	plan := scheduleTickPlan(mob, 8) // patrol segment

	applySchedulePlan(mob, plan)

	got := mob.Character.GetMiscData("active_patrol_id")
	if got == nil || got.(string) != "guard_patrol" {
		t.Errorf("expected active_patrol_id=guard_patrol, got %v", got)
	}
}

func TestApplySchedulePlan_DoesNotStampActivePatrolId_OnNonPatrolSegment(t *testing.T) {
	mobs.RegisterScheduleForTest(&mobs.Schedule{
		Id: "non_patrol_sched",
		Segments: []mobs.ScheduleSegment{
			{Start: 0, End: 24, TargetRoom: 9999, Activity: "",
				IdleCommands: []string{"x"}},
		},
	})
	defer mobs.UnregisterScheduleForTest("non_patrol_sched")

	mob := &mobs.Mob{ScheduleId: "non_patrol_sched"}
	mob.Character.RoomId = 1000

	plan := scheduleTickPlan(mob, 8)

	applySchedulePlan(mob, plan)

	if got := mob.Character.GetMiscData("active_patrol_id"); got != nil && got.(string) != "" {
		t.Errorf("expected no active_patrol_id stamp on non-patrol segment, got %v", got)
	}
}
```

- [ ] **Step 3: Run, confirm fail**

```bash
go test ./internal/hooks/ -run TestApplySchedulePlan_.*ActivePatrolId -v
```

Expected: fail (active_patrol_id not stamped).

- [ ] **Step 4: Implement the stamp**

In `internal/hooks/NewRound_IdleMobs_schedule.go`, inside `applySchedulePlan`, add the stamp at the end of the function (after all existing apply logic). The stamp depends on the current segment's activity, which the function already has access to via `scheduleTickPlan`'s output — but `schedulePlan` doesn't currently carry the segment's activity/patrol_id. Need to thread them through.

Add fields to `schedulePlan` struct (find at the top of the file):

```go
type schedulePlan struct {
	// ... existing fields ...

	// Chunk 3.4: current segment patrol context. Empty for non-patrol
	// segments. applySchedulePlan stamps `active_patrol_id` MiscData
	// so the patrol executor (NewRound_IdleMobs_patrol.go) can consume
	// it the same tick.
	ActivePatrolId string
}
```

In `scheduleTickPlan`, after the segment is resolved (just before the return), populate it:

```go
if seg.Activity == "patrol" {
	plan.ActivePatrolId = seg.PatrolId
}
```

In `applySchedulePlan`, at the END of the function (after all existing logic):

```go
// Chunk 3.4: stamp active_patrol_id for the patrol executor to consume.
// Empty string is fine — patrol executor reads-and-clears each tick,
// so an empty stamp simply means "no active patrol context this tick."
mob.Character.SetMiscData("active_patrol_id", plan.ActivePatrolId)
```

- [ ] **Step 5: Run, confirm pass**

```bash
go test ./internal/hooks/ -v
go build ./...
```

Expected: green.

- [ ] **Step 6: Commit**

```bash
git add internal/hooks/NewRound_IdleMobs_schedule.go internal/hooks/NewRound_IdleMobs_schedule_test.go
git commit -m "$(cat <<'EOF'
feat(hooks): schedule executor stamps active_patrol_id

When the current segment is activity: patrol with a patrol_id,
applySchedulePlan stamps the id into MiscData for the patrol
executor (T9) to consume the same tick. Empty stamp on
non-patrol segments — the patrol executor reads-and-clears
each tick, so empty means "no patrol context."

schedulePlan gains an ActivePatrolId field populated in
scheduleTickPlan.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Spawn override falls back to patrol's first waypoint

**Files:**
- Modify: `internal/mobs/schedule.go` (extend `applyScheduleSpawnOverride` to handle patrol fallback)
- Modify: `internal/mobs/schedule_spawn_test.go` (new test for patrol fallback)

- [ ] **Step 1: Re-read the existing spawn override**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -n "applyScheduleSpawnOverride" internal/mobs/schedule.go
sed -n '75,100p' internal/mobs/schedule.go
```

Confirm the function shape. Currently: returns segment's TargetRoom, falls back to homeRoomId on unknown segment.

- [ ] **Step 2: Write the failing test**

Append to `internal/mobs/schedule_spawn_test.go`:

```go
func TestApplyScheduleSpawnOverride_PatrolSegmentFallsBackToFirstWaypoint(t *testing.T) {
	// Patrol segment with no target_room — spawn should land at patrol's
	// first waypoint.
	RegisterScheduleForTest(&Schedule{
		Id: "guard_sched",
		Segments: []ScheduleSegment{
			{Start: 0, End: 24, Activity: "patrol", PatrolId: "guard_patrol",
				IdleCommands: []string{"watches."}},
		},
	})
	defer UnregisterScheduleForTest("guard_sched")

	registerPatrolForTest(&Patrol{
		Id:        "guard_patrol",
		Waypoints: []PatrolWaypoint{{Room: 4200, DwellRounds: 5}, {Room: 4201, DwellRounds: 5}},
	})
	defer unregisterPatrolForTest("guard_patrol")

	got := applyScheduleSpawnOverride("guard_sched", 9999 /* home */, 8 /* hour */)
	if got != 4200 {
		t.Errorf("expected first waypoint 4200, got %d", got)
	}
}

func TestApplyScheduleSpawnOverride_PatrolSegmentWithTargetRoomPrefersTarget(t *testing.T) {
	// If a patrol segment happens to set both target_room AND patrol_id,
	// the target_room wins (it's the explicit override). This shouldn't
	// happen in well-authored YAML but the precedence matters.
	RegisterScheduleForTest(&Schedule{
		Id: "guard_sched2",
		Segments: []ScheduleSegment{
			{Start: 0, End: 24, TargetRoom: 8888, Activity: "patrol", PatrolId: "guard_patrol2",
				IdleCommands: []string{"x"}},
		},
	})
	defer UnregisterScheduleForTest("guard_sched2")

	registerPatrolForTest(&Patrol{
		Id:        "guard_patrol2",
		Waypoints: []PatrolWaypoint{{Room: 4200, DwellRounds: 5}},
	})
	defer unregisterPatrolForTest("guard_patrol2")

	got := applyScheduleSpawnOverride("guard_sched2", 9999, 8)
	if got != 8888 {
		t.Errorf("expected explicit target_room 8888 to win, got %d", got)
	}
}
```

Need test helpers for patrol registration. Add to `internal/mobs/patrol.go` (peer to the schedule test helpers):

```go
// registerPatrolForTest / unregisterPatrolForTest are test-only helpers
// for injecting patrols into the package-level registry.
func registerPatrolForTest(p *Patrol) {
	patrolsMu.Lock()
	defer patrolsMu.Unlock()
	patrols[p.Id] = p
}

func unregisterPatrolForTest(id string) {
	patrolsMu.Lock()
	defer patrolsMu.Unlock()
	delete(patrols, id)
}

// Exported test-only helpers for cross-package tests (hooks tests).
// Matches the RegisterScheduleForTest pattern from chunk 3.2.
func RegisterPatrolForTest(p *Patrol)     { registerPatrolForTest(p) }
func UnregisterPatrolForTest(id string)   { unregisterPatrolForTest(id) }
```

- [ ] **Step 3: Run, confirm fail**

```bash
go test ./internal/mobs/ -run TestApplyScheduleSpawnOverride_Patrol -v
```

Expected: fail (fallback not implemented).

- [ ] **Step 4: Implement the patrol fallback**

In `internal/mobs/schedule.go`, find `applyScheduleSpawnOverride`. The current logic returns `seg.TargetRoom` if found. Modify to handle the patrol fallback:

```go
func applyScheduleSpawnOverride(scheduleId string, homeRoomId int, hour24 int) int {
	if scheduleId == "" {
		return homeRoomId
	}
	s := GetSchedule(scheduleId)
	if s == nil {
		return homeRoomId
	}
	seg := s.CurrentSegment(hour24)
	if seg == nil {
		return homeRoomId
	}

	// Explicit target_room wins (works for all activities).
	if seg.TargetRoom != 0 {
		return seg.TargetRoom
	}

	// Chunk 3.4: patrol segment without explicit target_room falls back
	// to the patrol's first waypoint so guards start at the beginning of
	// their beat instead of at the barracks.
	if seg.Activity == "patrol" && seg.PatrolId != "" {
		if p := GetPatrol(seg.PatrolId); p != nil && len(p.Waypoints) > 0 {
			return p.Waypoints[0].Room
		}
	}

	// Defensive: no target, no patrol fallback → home.
	return homeRoomId
}
```

- [ ] **Step 5: Run, confirm pass**

```bash
go test ./internal/mobs/ -v
go build ./...
```

Expected: green.

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/schedule.go internal/mobs/schedule_spawn_test.go internal/mobs/patrol.go
git commit -m "$(cat <<'EOF'
feat(mobs): spawn override falls back to patrol first waypoint

applyScheduleSpawnOverride extended to handle patrol segments
without explicit target_room: falls back to the patrol's first
waypoint so a guard at server-boot during their dayshift segment
appears at the start of their beat instead of at the barracks.

Adds patrol test helpers (registerPatrolForTest +
RegisterPatrolForTest cross-package export).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: `patrolTickPlan` + `applyPatrolPlan`

**Files:**
- Create: `internal/hooks/NewRound_IdleMobs_patrol.go`
- Create: `internal/hooks/NewRound_IdleMobs_patrol_test.go`
- Modify: `internal/hooks/spell_foldrecall.go` (add `getMiscDataString` peer helper if not present)

- [ ] **Step 1: Re-read the existing schedule executor pattern**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,165p' internal/hooks/NewRound_IdleMobs_schedule.go
```

Match the structure: pure decision plan + side-effect applier.

- [ ] **Step 2: Add `getMiscDataString` helper if missing**

```bash
grep -n "getMiscDataString" internal/hooks/*.go
```

If not present, add to `internal/hooks/spell_foldrecall.go` next to `getMiscDataInt`:

```go
// getMiscDataString retrieves a string stored in MiscData; returns "" for
// unset keys or non-string values.
func getMiscDataString(char *characters.Character, key string) string {
	v := char.GetMiscData(key)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
```

- [ ] **Step 3: Write the failing tests**

Create `internal/hooks/NewRound_IdleMobs_patrol_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func registerTestPatrol(t *testing.T) {
	t.Helper()
	mobs.RegisterPatrolForTest(&mobs.Patrol{
		Id:        "test_patrol",
		LoopShape: "strict",
		Waypoints: []mobs.PatrolWaypoint{
			{Room: 100, DwellRounds: 5},
			{Room: 101, DwellRounds: 0},
			{Room: 102, DwellRounds: 10},
		},
	})
	t.Cleanup(func() { mobs.UnregisterPatrolForTest("test_patrol") })
}

func TestPatrolTickPlan_WantsPathWhenNotAtTarget(t *testing.T) {
	registerTestPatrol(t)

	mob := &mobs.Mob{}
	mob.Character.RoomId = 999 // not at any waypoint

	plan := patrolTickPlan(mob, "test_patrol")

	if !plan.HasPatrol {
		t.Fatalf("expected HasPatrol=true")
	}
	if !plan.WantsPath {
		t.Errorf("expected WantsPath=true when away from target, got %+v", plan)
	}
	if plan.TargetRoom != 100 {
		t.Errorf("expected initial target=100 (waypoint[0]), got %d", plan.TargetRoom)
	}
}

func TestPatrolTickPlan_WantsDwellWaitAtTarget(t *testing.T) {
	registerTestPatrol(t)

	mob := &mobs.Mob{}
	mob.Character.RoomId = 100 // at waypoint[0]
	mob.Character.SetMiscData("patrol_dwell_remaining", 3)

	plan := patrolTickPlan(mob, "test_patrol")

	if !plan.WantsDwellWait {
		t.Errorf("expected WantsDwellWait=true at target with dwell>0, got %+v", plan)
	}
}

func TestPatrolTickPlan_WantsAdvanceAtTargetWithZeroDwell(t *testing.T) {
	registerTestPatrol(t)

	mob := &mobs.Mob{}
	mob.Character.RoomId = 101 // at waypoint[1] which has DwellRounds=0
	mob.Character.SetMiscData("patrol_waypoint_idx", 1)
	mob.Character.SetMiscData("patrol_dwell_remaining", 0)

	plan := patrolTickPlan(mob, "test_patrol")

	if !plan.WantsAdvance {
		t.Errorf("expected WantsAdvance=true at target with dwell=0, got %+v", plan)
	}
	if plan.NextWaypointIdx != 2 {
		t.Errorf("expected NextWaypointIdx=2, got %d", plan.NextWaypointIdx)
	}
}

func TestPatrolTickPlan_WantsHomeFallbackAfterMaxRetries(t *testing.T) {
	registerTestPatrol(t)

	mob := &mobs.Mob{}
	mob.Character.RoomId = 999 // not at target
	mob.Character.SetMiscData("patrol_path_fail_count", 99) // way past default 20

	plan := patrolTickPlan(mob, "test_patrol")

	if !plan.WantsHomeFallback {
		t.Errorf("expected WantsHomeFallback=true after max retries, got %+v", plan)
	}
}

func TestApplyPatrolPlan_IncrementsRetryOnWantsPath(t *testing.T) {
	registerTestPatrol(t)

	mob := &mobs.Mob{}
	mob.Character.RoomId = 999

	plan := patrolTickPlan(mob, "test_patrol")
	applyPatrolPlan(mob, plan)

	got := mob.Character.GetMiscData("patrol_path_fail_count")
	if got == nil || got.(int) != 1 {
		t.Errorf("expected patrol_path_fail_count=1 after WantsPath apply, got %v", got)
	}
}

func TestApplyPatrolPlan_ResetsRetryAtTarget(t *testing.T) {
	registerTestPatrol(t)

	mob := &mobs.Mob{}
	mob.Character.RoomId = 100 // at waypoint[0]
	mob.Character.SetMiscData("patrol_path_fail_count", 5)
	mob.Character.SetMiscData("patrol_dwell_remaining", 3) // will WantsDwellWait

	plan := patrolTickPlan(mob, "test_patrol")
	applyPatrolPlan(mob, plan)

	got := mob.Character.GetMiscData("patrol_path_fail_count")
	if got == nil || got.(int) != 0 {
		t.Errorf("expected patrol_path_fail_count reset to 0 at target, got %v", got)
	}
}
```

- [ ] **Step 4: Run, confirm fail**

```bash
go test ./internal/hooks/ -run TestPatrolTickPlan_ -v
```

Expected: compile error — `patrolTickPlan` not defined.

- [ ] **Step 5: Implement `internal/hooks/NewRound_IdleMobs_patrol.go`**

```go
package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// patrolPlan describes what the patrol executor wants to do for a
// mob this tick. Extracted so the decision logic is unit-testable
// without driving the full IdleMobs loop.
type patrolPlan struct {
	HasPatrol         bool
	WantsDwellWait    bool   // mob at current target waypoint AND dwell > 0
	WantsPath         bool   // mob not at current target waypoint
	TargetRoom        int
	WantsAdvance      bool   // dwell expired (or 0); advance this tick
	NextWaypointIdx   int
	NextDirection     int    // +1 / -1; only meaningful for yo-yo
	NextDwellRounds   int    // dwell for the new waypoint after advance
	WantsHomeFallback bool   // after MaxPathRetries
	FailureMessage    string
}

// patrolTickPlan computes the desired tick action for a patrol mob.
// Pure over its inputs (mob.Character.RoomId, MiscData, patrolId) —
// safe to call from tests with stubbed registry state.
func patrolTickPlan(mob *mobs.Mob, patrolId string) patrolPlan {
	plan := patrolPlan{}
	if mob == nil || patrolId == "" {
		return plan
	}
	p := mobs.GetPatrol(patrolId)
	if p == nil || len(p.Waypoints) == 0 {
		return plan
	}
	plan.HasPatrol = true

	idx := getMiscDataInt(&mob.Character, "patrol_waypoint_idx")
	if idx < 0 || idx >= len(p.Waypoints) {
		idx = 0 // first-tick or stale-after-patrol-shrink: reset to start
	}
	dir := getMiscDataInt(&mob.Character, "patrol_direction")
	if dir == 0 {
		dir = +1
	}
	dwellRemaining := getMiscDataInt(&mob.Character, "patrol_dwell_remaining")
	failCount := getMiscDataInt(&mob.Character, "patrol_path_fail_count")

	currentWaypoint := &p.Waypoints[idx]

	if mob.Character.RoomId == currentWaypoint.Room {
		// At target. Dwell or advance?
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

	// Not at target — path or fallback.
	maxRetries := int(configs.GetBalanceConfig().ScheduleMaxPathRetries)
	if maxRetries > 0 && failCount >= maxRetries {
		plan.WantsHomeFallback = true
		plan.FailureMessage = fmt.Sprintf(
			"patrol mob %d (%s) unreachable waypoint room %d after %d retries; falling back to home",
			mob.MobId, mob.Character.Name, currentWaypoint.Room, failCount)
		return plan
	}
	plan.WantsPath = true
	plan.TargetRoom = currentWaypoint.Room
	return plan
}

// applyPatrolPlan mutates the mob (updates MiscData, queues commands)
// based on the plan. Side-effecting; not pure.
func applyPatrolPlan(mob *mobs.Mob, plan patrolPlan) {
	if !plan.HasPatrol {
		return
	}

	switch {
	case plan.WantsHomeFallback:
		mudlog.Warn("patrol", "msg", plan.FailureMessage)
		mob.Command("pathto home")
		mob.Character.SetMiscData("patrol_path_fail_count", 0)
		return

	case plan.WantsAdvance:
		mob.Character.SetMiscData("patrol_waypoint_idx", plan.NextWaypointIdx)
		mob.Character.SetMiscData("patrol_direction", plan.NextDirection)
		mob.Character.SetMiscData("patrol_dwell_remaining", plan.NextDwellRounds)
		mob.Character.SetMiscData("patrol_path_fail_count", 0)
		return

	case plan.WantsDwellWait:
		current := getMiscDataInt(&mob.Character, "patrol_dwell_remaining")
		if current > 0 {
			mob.Character.SetMiscData("patrol_dwell_remaining", current-1)
		}
		// At-target → reset retry counter.
		mob.Character.SetMiscData("patrol_path_fail_count", 0)
		return

	case plan.WantsPath:
		// Queue pathto if no path is in flight (matches schedule executor pattern).
		if mob.Path.Len() == 0 && mob.Path.Current() == nil {
			mob.Command(fmt.Sprintf("pathto %d", plan.TargetRoom))
		}
		fails := getMiscDataInt(&mob.Character, "patrol_path_fail_count")
		mob.Character.SetMiscData("patrol_path_fail_count", fails+1)
		return
	}
}
```

- [ ] **Step 6: Run tests + build**

```bash
go test ./internal/hooks/ -v
go build ./...
```

Expected: green.

- [ ] **Step 7: Commit**

```bash
git add internal/hooks/NewRound_IdleMobs_patrol.go internal/hooks/NewRound_IdleMobs_patrol_test.go internal/hooks/spell_foldrecall.go
git commit -m "$(cat <<'EOF'
feat(hooks): patrolTickPlan + applyPatrolPlan

Pure decision helper + side-effecting applier for the chunk 3.4
patrol executor. Decides per-tick whether the mob wants to dwell
at its current waypoint, advance to the next (using NextWaypoint
from chunk 3.4 T1), path toward an out-of-position waypoint, or
fall back to pathto home after ScheduleMaxPathRetries failures.

Adds getMiscDataString helper next to getMiscDataInt.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: `NewRound_IdleMobs` wires the patrol branch

**Files:**
- Modify: `internal/hooks/NewRound_IdleMobs.go` (insert patrol branch after schedule branch)

- [ ] **Step 1: Re-read the IdleMobs hook**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '65,90p' internal/hooks/NewRound_IdleMobs.go
```

Find where the chunk 3.2 schedule branch was inserted. The patrol branch goes AFTER it (so the schedule's `active_patrol_id` stamp is visible).

- [ ] **Step 2: Add the patrol branch**

In `internal/hooks/NewRound_IdleMobs.go`, immediately after the chunk 3.2 schedule branch, add:

```go
// Chunk 3.4: patrol executor. Reads active_patrol_id stamped by the
// schedule branch above (if the current schedule segment has
// activity: patrol), otherwise falls back to the mob's own PatrolId
// for standalone patrols. Reads-and-clears the stamp so it doesn't
// linger across ticks.
{
	var activePatrolId string
	if id := getMiscDataString(&mob.Character, "active_patrol_id"); id != "" {
		activePatrolId = id
		mob.Character.SetMiscData("active_patrol_id", "")
	}
	if activePatrolId == "" && mob.PatrolId != "" {
		activePatrolId = mob.PatrolId
	}
	if activePatrolId != "" {
		plan := patrolTickPlan(mob, activePatrolId)
		applyPatrolPlan(mob, plan)
	}
}
```

- [ ] **Step 3: Build, confirm clean**

```bash
go build ./...
go test ./internal/hooks/ -v
```

Expected: green.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/NewRound_IdleMobs.go
git commit -m "$(cat <<'EOF'
feat(hooks): IdleMobs wires the patrol branch after schedule branch

Patrol executor runs per tick. Reads active_patrol_id stamped
by the schedule branch (composed via activity: patrol segments),
falls back to mob.PatrolId for standalone patrols. Reads-and-
clears the stamp so it doesn't linger across ticks.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: `mob schedule` admin inspector shows patrol state

**Files:**
- Modify: `internal/usercommands/admin.mob.go` (extend `mob_Schedule` to render patrol state when present)

- [ ] **Step 1: Re-read the chunk 3.2 inspector**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -n "func mob_Schedule" internal/usercommands/admin.mob.go
sed -n '220,310p' internal/usercommands/admin.mob.go
```

Find the function and its output format. Append the patrol block at the end of the output.

- [ ] **Step 2: Add the patrol state output**

In `internal/usercommands/admin.mob.go`, find `mob_Schedule`. Just before the function's final `return true, nil` and after the existing schedule output is sent, add a patrol block:

```go
// Chunk 3.4: also show patrol state if the mob has one (either via
// PatrolId directly or via the active schedule segment's patrol_id).
activePatrolId := m.PatrolId
if sched != nil {
	if curSeg := sched.CurrentSegment(gametime.GetDate().Hour24); curSeg != nil && curSeg.Activity == "patrol" {
		activePatrolId = curSeg.PatrolId
	}
}
if activePatrolId != "" {
	p := mobs.GetPatrol(activePatrolId)
	if p == nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(
			"  (patrol_id %q does not resolve)", activePatrolId))
	} else {
		idx := m.Character.GetMiscData("patrol_waypoint_idx")
		dir := m.Character.GetMiscData("patrol_direction")
		dwell := m.Character.GetMiscData("patrol_dwell_remaining")
		fails := m.Character.GetMiscData("patrol_path_fail_count")

		loopShape := p.LoopShape
		if loopShape == "" {
			loopShape = "strict"
		}

		out := fmt.Sprintf(
			"Patrol state:\n"+
				"  patrol_id:           %s\n"+
				"  loop_shape:          %s\n"+
				"  current waypoint:    %v (room %d)\n"+
				"  direction:           %v\n"+
				"  dwell remaining:     %v rounds\n"+
				"  path retries:        %v",
			activePatrolId, loopShape,
			idx, p.Waypoints[normalizeIdxForDisplay(idx, len(p.Waypoints))].Room,
			dir, dwell, fails)
		user.SendText(messaging.CategorySystem, out)
	}
}
```

Add the helper `normalizeIdxForDisplay` somewhere near the bottom of the file:

```go
// normalizeIdxForDisplay clamps an interface{} (likely int) waypoint
// index into a safe slice index for display purposes only. Returns 0
// for nil / non-int / out-of-range values.
func normalizeIdxForDisplay(v any, length int) int {
	if length == 0 {
		return 0
	}
	if i, ok := v.(int); ok && i >= 0 && i < length {
		return i
	}
	return 0
}
```

The `gametime` import should already be present from chunk 3.2 work.

- [ ] **Step 3: Build, confirm clean**

```bash
go build ./...
```

Expected: clean.

- [ ] **Step 4: Update helpfile**

In `_datafiles/world/dogmud/templates/admincommands/help/command.mob.template`, find the `mob schedule` block (added in chunk 3.2). Append a sentence noting patrol state:

```
For mobs with a patrol (standalone or via an active schedule segment),
the inspector also shows the patrol's current waypoint, direction
(for yo-yo loops), dwell remaining, and path retry count.
```

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/admin.mob.go _datafiles/world/dogmud/templates/admincommands/help/command.mob.template
git commit -m "$(cat <<'EOF'
feat(admin): mob schedule inspector shows patrol state

Extends mob schedule <instId> to also render patrol state when
the mob has one (standalone via PatrolId, or composed via the
active schedule segment's patrol_id). Shows current waypoint,
loop_shape, direction, dwell remaining, and path retry count.
Helpfile updated.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Pilot content — Thornwall city guard

**Files:**
- Create: `_datafiles/world/dogmud/patrols/thornwall_city/thornwall_market_beat.yaml`
- Create: `_datafiles/world/dogmud/schedules/thornwall_city/thornwall_city_guard_dayshift.yaml`
- Create: `_datafiles/world/dogmud/rooms/thornwall_city/<barracks_id>.yaml` (new "above-shop" room for the guard barracks)
- Modify: an existing Thornwall room — add `up` exit to the new barracks
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/106-city_guard.yaml` (add `schedule_id`)
- Run: `python tools/id_inventory.py --zone thornwall_city --type rooms --alloc rooms 2` for the new room id

- [ ] **Step 1: Reserve a room ID**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
python tools/id_inventory.py --zone thornwall_city --type rooms --alloc rooms 2
```

Record the first id (e.g., `5104`) as `<BARRACKS_ID>`. The second is slack.

- [ ] **Step 2: Pick the patrol waypoints + the barracks anchor room**

Read existing Thornwall rooms to pick a sensible 4-5 waypoint perimeter route through the market/civic district. The market squares + bank + gate + tavern area is the natural fit.

```bash
ls _datafiles/world/dogmud/rooms/thornwall_city/ | head -30
```

Find rooms that form a connected loop. Goal: 4-5 waypoints, all reachable from each other.

For the barracks, pick a constabulary / guard-hut room that doesn't already have an `up` exit. Add the barracks above it via the chunk 3.3 "above-shop" pattern.

Record the picks as `<WAYPOINT_1>` ... `<WAYPOINT_5>` and `<BARRACKS_ANCHOR_ROOM>`.

- [ ] **Step 3: Create the patrol YAML**

```yaml
id: thornwall_market_beat
description: "City guard's market square perimeter beat."
loop_shape: strict
waypoints:
  - room: <WAYPOINT_1>
    dwell_rounds: 5
  - room: <WAYPOINT_2>
    dwell_rounds: 10
  - room: <WAYPOINT_3>
    dwell_rounds: 8
  - room: <WAYPOINT_4>
    dwell_rounds: 5
  - room: <WAYPOINT_5>
    dwell_rounds: 3
```

- [ ] **Step 4: Create the barracks room YAML**

`_datafiles/world/dogmud/rooms/thornwall_city/<BARRACKS_ID>.yaml`:

```yaml
roomid: <BARRACKS_ID>
zone: Thornwall City
title: Guard Barracks Above the Constabulary
description: A spare upstairs room over the constabulary. Three cots line
  one wall; a rack of patched cloaks and helmets stands by the door. A
  small table holds half-finished mugs of tea and a sand-clock. The boots
  by the door are arranged neatly, ready to be pulled on at the next
  shift call.
biome: city
mapsymbol: h
maplegend: Home
coord:
  x: <ANCHOR_X>
  y: <ANCHOR_Y>
  z: 1
exits:
  down:
    roomid: <BARRACKS_ANCHOR_ROOM>
```

Replace the coord placeholders with the anchor room's x/y.

- [ ] **Step 5: Add the `up` exit to the anchor room**

Edit `_datafiles/world/dogmud/rooms/thornwall_city/<BARRACKS_ANCHOR_ROOM>.yaml`. Find the `exits:` block. Add:

```yaml
  up:
    roomid: <BARRACKS_ID>
```

- [ ] **Step 6: Create the dayshift schedule YAML**

`_datafiles/world/dogmud/schedules/thornwall_city/thornwall_city_guard_dayshift.yaml`:

```yaml
id: thornwall_city_guard_dayshift
description: "City guard's day shift — patrol the market beat 6-22, sleep at the barracks 22-6."
segments:
  - start: 6
    end: 22
    activity: patrol
    patrol_id: thornwall_market_beat
    idlecommands:
      - say All clear here.
      - emote scans the square.
      - emote nods to a passing merchant.
  - start: 22
    end: 6
    target_room: <BARRACKS_ID>
    activity: sleeping
    idlecommands:
      - emote sleeps soundly under a coarse blanket.
      - emote shifts in his sleep, murmuring.
```

- [ ] **Step 7: Update the mob YAML**

In `_datafiles/world/dogmud/mobs/thornwall_city/106-city_guard.yaml`, add:

```yaml
schedule_id: thornwall_city_guard_dayshift
```

Place it near `behavior_archetype` if present, matching the chunk 3.2/3.3 pilot pattern.

- [ ] **Step 8: Clean up any stale instance saves**

```bash
ls _datafiles/world/dogmud/rooms.instances/thornwall_city/ 2>/dev/null | grep -E "^<BARRACKS_ANCHOR_ROOM>"
ls _datafiles/world/dogmud/mobs.instances/thornwall_city/ 2>/dev/null | grep "^106-"
```

Delete any matches per the CLAUDE.md instance-save SOP.

- [ ] **Step 9: Boot and smoke**

Boot the server. Confirm:
- `mobs.LoadPatrols() loadedCount=1`
- No validation panics
- City guard (mob 106) appears at the market beat during day hours
- Use `mob schedule <inst>` to confirm both schedule + patrol state render

- [ ] **Step 10: Commit**

```bash
git add _datafiles/world/dogmud/patrols/ _datafiles/world/dogmud/schedules/thornwall_city/thornwall_city_guard_dayshift.yaml _datafiles/world/dogmud/rooms/thornwall_city/<BARRACKS_ID>.yaml _datafiles/world/dogmud/rooms/thornwall_city/<BARRACKS_ANCHOR_ROOM>.yaml _datafiles/world/dogmud/mobs/thornwall_city/106-city_guard.yaml
git commit -m "$(cat <<'EOF'
feat(content): Thornwall city guard pilot — patrol + dayshift schedule

City guard (mob 106) gains a day/night routine:
- 6-22: patrols the market beat (4-5 waypoints around market,
  bank, gate, tavern)
- 22-6: sleeps at a new guard barracks above the constabulary

Adds the new barracks room (mapsymbol h Home), an up exit from
the constabulary, the thornwall_market_beat patrol YAML, the
thornwall_city_guard_dayshift schedule YAML, and schedule_id
on the mob.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Documentation pass

**Files:**
- Create: `docs/schemas/patrol.md`
- Modify: `docs/schemas/schedule.md`
- Modify: `internal/mobs/context.md`
- Modify: `internal/hooks/context.md`
- Modify: `internal/configs/context.md`
- Modify: `CLAUDE.md`

- [ ] **Step 1: Create patrol schema doc**

`docs/schemas/patrol.md`:

```markdown
# Patrol YAML schema (chunk 3.4)

Patrols are looped multi-room routes attached to NPCs via the
`patrol_id` field on mob specs, OR via a schedule segment with
`activity: patrol` + `patrol_id`. Each patrol references a file
at `_datafiles/world/dogmud/patrols/<zone>/<id>.yaml` (filename
= `ConvertForFilename(id)`).

## Required fields

```yaml
id: <string>                      # must match the filename
description: <string>             # short prose; used in admin debug output
waypoints: [<waypoint>, ...]      # at least one
loop_shape: <"strict" | "yo-yo">  # default "strict" if omitted
```

## Waypoint fields

```yaml
- room: <int>                     # target room id; must exist
  dwell_rounds: <int>             # rounds to stay at this waypoint before
                                  # moving on; default 0 (move immediately)
```

## Loop semantics

**Strict (default):** A → B → C → D → A → B → … After the last
waypoint, the next target is the first waypoint.

**Yo-yo:** A → B → C → D → C → B → A → B → … Direction flips at
endpoints. State carried in per-mob MiscData (`patrol_direction`).

## Composition with schedules (chunk 3.2)

A schedule segment can opt in to running a patrol for its
duration:

```yaml
- start: 6
  end: 22
  # target_room is optional for activity: patrol segments
  activity: patrol
  patrol_id: thornwall_market_beat
  idlecommands:
    - say All clear here.
```

When the schedule executor enters a patrol segment, it stamps
`active_patrol_id` MiscData. The patrol executor (which runs
after the schedule executor on each idle tick) consumes the
stamp and drives the patrol.

If a patrol segment also sets `target_room` (legal but
redundant), the `target_room` wins for spawn-override
placement. The patrol's first waypoint serves as the fallback
when no `target_room` is set.

## Validation (load-time, panics)

- Filename must equal `ConvertForFilename(id)`.
- At least one waypoint.
- Each waypoint's `room` exists in the rooms registry.
- Each waypoint's `dwell_rounds >= 0`.
- `loop_shape` is empty, `"strict"`, or `"yo-yo"`.
- Inter-waypoint pathfinding resolves for every consecutive
  pair (and the wrap pair for `strict`).
- Mob `patrol_id` references resolve.
- Schedule segment `patrol_id` references resolve.

## Validation (load-time, warn-only)

- Single-waypoint patrol — degenerate.
- Schedule segment with `patrol_id` set but `activity` is not
  `patrol` — field has no effect.
- Cross-zone waypoint — out of scope for chunk 3.4 (will be
  handled by chunk 3.7).

## Runtime

The patrol executor runs in `NewRound_IdleMobs`. Per tick, for
mobs with an active patrol context, it consults the mob's
current waypoint index + direction (in MiscData) and decides
whether to dwell, advance, path toward the current target, or
fall back to `pathto home` after `ScheduleMaxPathRetries`
(default 20) consecutive path failures.

Combat interrupts patrols via the existing IdleMobs combat
guard. On the next idle tick after combat ends, the patrol
executor sees the same `patrol_waypoint_idx` and resumes
pathing.

## Future: cross-zone + caravan unification (chunk 3.7)

Cross-zone waypoint references and caravan-movement
unification (caravans become a yo-yo patrol with cargo + vendor
semantics layered on top) are deferred to chunk 3.7. The 3.4
loop_shape choices were made with future caravan migration in
mind.
```

- [ ] **Step 2: Extend schedule.md schema doc**

In `docs/schemas/schedule.md`, find the activity vocabulary section. Add a new entry:

```markdown
- `patrol` — When the segment is active, the schedule executor
  stamps the segment's `patrol_id` into MiscData; the patrol
  executor consumes it and drives the patrol. `target_room`
  becomes optional for patrol segments (the patrol's first
  waypoint serves as the spawn-override anchor when omitted).
  Requires `patrol_id` to be set; loader panics if it's empty
  or unresolved. See `docs/schemas/patrol.md`.
```

Also note in the segment-fields section that `target_room` is
optional for patrol activity.

- [ ] **Step 3: Update internal context.md files**

`internal/mobs/context.md` — append after the Schedules section:

```markdown
## Patrols (chunk 3.4)

Mobs with `patrol_id:` set follow waypoint patrols authored in
`_datafiles/world/dogmud/patrols/<zone>/<id>.yaml`. See
`docs/schemas/patrol.md` for the full schema.

- `patrol.go`: `Patrol`, `PatrolWaypoint`, `GetPatrol`,
  `NextWaypoint`, test helpers.
- `patrol_loader.go`: `LoadPatrols`, `validatePatrolStandalone`,
  `validatePatrolAgainstWorld`, `SetPatrolWorldValidator` (DI
  injection used in main.go to break the mobs ← rooms import
  cycle). Called from `LoadDataFiles` before `LoadSchedules`
  (schedule loader cross-checks segment patrol_ids).
- Spawn override: `applyScheduleSpawnOverride` falls back to
  the patrol's first waypoint when a patrol segment has no
  `target_room`.
```

`internal/hooks/context.md` — append in the IdleMobs section:

```markdown
- `NewRound_IdleMobs_patrol.go`: `patrolTickPlan` (pure decision)
  + `applyPatrolPlan` (side effects). Runs in IdleMobs AFTER the
  schedule branch, so a schedule-stamped `active_patrol_id`
  (from an `activity: patrol` segment) is visible. Reads-and-
  clears the stamp; falls back to `mob.PatrolId` for standalone
  patrols.
- `NewRound_IdleMobs_schedule.go`: stamps `active_patrol_id`
  MiscData in `applySchedulePlan` when the current segment has
  `activity: patrol`.
```

`internal/configs/context.md` — add a note:

```markdown
- `ScheduleMaxPathRetries`: also governs patrol path retries
  (chunk 3.4 reuses the same threshold; no separate knob).
```

`CLAUDE.md` — append a "Patrols" subsection near the existing
"NPC Schedules" / "Sleep Mechanics" subsections:

```markdown
### NPC Patrols

Patrol routes (multi-room loops) are authored at
`_datafiles/world/dogmud/patrols/<zone>/<id>.yaml`. A mob can
reference one directly via `patrol_id:` (always-on patrol), or
a schedule segment can opt in via `activity: patrol` +
`patrol_id:` (patrol runs during the segment only). Two loop
shapes: `strict` (loop back to start) and `yo-yo` (flip
direction at endpoints). Per-waypoint `dwell_rounds`. Combat
interrupts patrols; the executor resumes to the same target
waypoint on the next idle tick. Path retries use the chunk 3.2
`ScheduleMaxPathRetries` knob, falling back to `pathto home`
after the threshold. See `docs/schemas/patrol.md`.

Inter-zone patrols and caravan unification onto the patrol
layer are deferred to chunk 3.7.
```

- [ ] **Step 4: Build, confirm clean (sanity)**

```bash
go build ./...
```

Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add docs/schemas/patrol.md docs/schemas/schedule.md internal/mobs/context.md internal/hooks/context.md internal/configs/context.md CLAUDE.md
git commit -m "$(cat <<'EOF'
docs: patrol schema + context.md updates + CLAUDE.md

New docs/schemas/patrol.md with full waypoint schema, loop
semantics, schedule composition, validation rules. schedule.md
extended with activity: patrol entry. mobs / hooks / configs
context.md note the new patrol executor + DI wiring + retry
knob reuse. CLAUDE.md gains an NPC Patrols subsection.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 13: Smoketester + roadmap closeout

**Files:**
- Create: `tools/testing/goals/3.4-patrol-observation.yaml`
- Modify: `MOB_ALIVENESS_ROADMAP.md`
- Move: spec and plan to `completed/` subdirectories

- [ ] **Step 1: Author smoketester goal**

`tools/testing/goals/3.4-patrol-observation.yaml`:

```yaml
description: "Observe Thornwall city guard for ~2 full game-day patrol loops + day/night transition + combat interrupt."

goals:
  - "Find the city guard (mob 106) in Thornwall. Use 'mob schedule <inst>' to see patrol state."
  - "Watch the guard for one full patrol loop. Confirm all waypoints visited in order."
  - "Confirm dwell timing at the longer-dwell waypoints (10 / 8 round dwells should be visibly longer than 3 / 5)."
  - "Use 'time set 23' to trigger the night transition. Confirm the guard heads to the barracks and sleeps (uses chunk 3.3 mechanics)."
  - "Use 'time set 7' to trigger the day transition. Confirm the guard resumes patrolling."
  - "Mid-patrol, attack the guard. Disengage or 'flee'. Confirm: next idle tick, guard resumes pathing toward the same waypoint."

pass_criteria:
  - "All patrol waypoints visited in order during a full loop"
  - "Day/night schedule transitions work (patrol → barracks sleep → patrol)"
  - "Combat interrupt + resume to same waypoint works"
  - "No 'lost' adjective acquired on the guard"
  - "Admin 'mob schedule' inspector renders patrol state correctly"

notes:
  - "If 'time set <hour>' admin command isn't available, observe in real-time."
  - "Double-spawn fix (commit 31cbc3b1) should prevent guard duplication on schedule transitions."
```

- [ ] **Step 2: Update roadmap**

Edit `MOB_ALIVENESS_ROADMAP.md`. Find the chunk 3.4 progress tracker row:

```markdown
| 3.4 | Routine | Waypoint patrols | M | — | Not started |
```

Change to:

```markdown
| 3.4 | Routine | Waypoint patrols | M | — | Done |
```

Find the detailed 3.4 section. Append:

```markdown
- **Shipped:** Patrol primitive — multi-room routes with strict +
  yo-yo loop shapes, per-waypoint dwell, combat interrupt with
  resume-to-same-waypoint, retry-then-pathto-home fallback (reuses
  chunk 3.2 `ScheduleMaxPathRetries`). Two integration paths:
  standalone (`patrol_id` on mob) and composed (`activity: patrol`
  segment via chunk 3.2 schedules). New
  `internal/mobs/patrol.go` + `patrol_loader.go`, new
  `internal/hooks/NewRound_IdleMobs_patrol.go`. Schedule schema
  gains `target_room`-optional for patrol segments and a
  `patrol_id` field; spawn override falls back to the patrol's
  first waypoint when a patrol segment has no explicit target.
  Admin `mob schedule <instId>` inspector extended to render
  patrol state. Pilot: Thornwall city guard (mob 106) with a
  6-22 patrol of the market beat + 22-6 sleep at a new guard
  barracks room. Spec at
  `docs/superpowers/specs/completed/2026-05-25-mob-aliveness-3.4-waypoint-patrols-design.md`,
  plan at
  `docs/superpowers/plans/completed/2026-05-25-mob-aliveness-3.4-waypoint-patrols.md`.
```

Note chunk 3.7's now-satisfied dependency: append to the 3.7 entry:

```markdown
- **3.4 satisfied:** Single-zone patrol primitive shipped in
  3.4. 3.7 lifts the single-zone restriction and migrates
  caravan movement onto the shared layer.
```

- [ ] **Step 3: Move spec and plan to completed/**

```bash
git mv docs/superpowers/specs/2026-05-25-mob-aliveness-3.4-waypoint-patrols-design.md docs/superpowers/specs/completed/
git mv docs/superpowers/plans/2026-05-25-mob-aliveness-3.4-waypoint-patrols.md docs/superpowers/plans/completed/
```

- [ ] **Step 4: Final verification**

```bash
go build ./...
go test ./...
```

Expected: clean across the board.

- [ ] **Step 5: Commit**

```bash
git add tools/testing/goals/3.4-patrol-observation.yaml MOB_ALIVENESS_ROADMAP.md docs/superpowers/specs/completed/ docs/superpowers/plans/completed/
git commit -m "$(cat <<'EOF'
chore(roadmap): mark 3.4 waypoint patrols Done

Smoketester goal file authored at
tools/testing/goals/3.4-patrol-observation.yaml. Spec + plan
moved to completed/. Chunk 3.7's "depends on 3.4" line updated
to note the single-zone primitive is now available.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Self-Review

**Spec coverage check:**

- ✅ Patrol type + NextWaypoint helper → T1
- ✅ Patrol loader + standalone validators → T2
- ✅ DI wiring for world checks (main.go) → T3
- ✅ Mob `patrol_id` field + cross-check → T4
- ✅ Schedule loader recognizes `activity: patrol` + optional `target_room` + `patrol_id` field on ScheduleSegment + cross-check → T5
- ✅ Schedule executor stamps `active_patrol_id` → T6
- ✅ Spawn override falls back to patrol's first waypoint → T7
- ✅ `patrolTickPlan` + `applyPatrolPlan` → T8
- ✅ IdleMobs wires patrol branch → T9
- ✅ Admin `mob schedule` inspector extension → T10
- ✅ Thornwall city guard pilot (patrol + schedule + barracks) → T11
- ✅ Documentation (schema, context.md, CLAUDE.md, helpfile) → T12
- ✅ Smoketester goal + roadmap closeout → T13

**Type consistency check:**

- `Patrol` / `PatrolWaypoint` shapes consistent across T1, T2, T5, T7, T8, T10, T11
- `GetPatrol(id) *Patrol` accessor used identically in T7, T8, T10
- `RegisterPatrolForTest(*Patrol)` test helper consistent in T7, T8
- `patrolPlan` struct fields consistent T8 → T9
- MiscData keys: `patrol_waypoint_idx`, `patrol_direction`, `patrol_dwell_remaining`, `patrol_path_fail_count`, `active_patrol_id` used identically in T6, T7, T8, T9, T10
- Config knob `ScheduleMaxPathRetries` (chunk 3.2 carryover) used in T8 only
- Schedule schema change: `ScheduleSegment.PatrolId` field defined in T5 and consumed in T6, T7, T8

**Placeholder scan:**

Searched for "TBD", "TODO", "implement later", "appropriate error handling", "similar to Task" — none found. T11 uses `<BARRACKS_ID>` / `<WAYPOINT_N>` as explicit placeholders that the implementer resolves via `tools/id_inventory.py` and zone-map reading; documented inline as resolved-during-implementation, not silent placeholders.

**Scope check:** 13 tasks, comparable to chunk 3.2 (13) and 3.3 (18). M-sized chunk. Sequential dispatch order is fine; some pairs (T5+T6 schedule changes; T8+T9 executor + wiring) could be combined if running short on subagent overhead, but kept separate for cleaner review.

Plan is internally consistent and ready for execution.
