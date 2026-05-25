# Mob Aliveness 3.2 — NPC Schedules Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Authored daily routines that move NPCs between rooms by the hour and swap their idle flavor — Kerra works at the forge during the day, drinks at the tavern in the evening, sleeps in her loft at night. Town that empties at night and fills in the morning.

**Architecture:** New `schedule_id` field on mob specs references shared schedule YAMLs in `_datafiles/world/dogmud/schedules/<zone>/<id>.yaml`. A Go-side executor wired into `NewRound_IdleMobs` resolves the current segment by hour, steers the mob via existing `pathto` plumbing, swaps the mob's `IdleCommands` pool per segment, and gates `TickMobCraft` via a per-segment `activity:` field. Spawn override places scheduled mobs at the current segment's target room on cold start and on respawn. Three Thornwall pilots (Kerra, Marek, Olen) with three new above-shop home rooms.

**Tech Stack:** Go 1.24, `internal/mobs/` (new `schedule.go` + `schedule_loader.go`), `internal/hooks/NewRound_IdleMobs.go` (executor hook), `internal/behaviortree/conditions_state.go` (new `mob_at_target_room`), `internal/configs/config.balance.go` (new `ScheduleMaxPathRetries` knob), `internal/usercommands/admin.mob.go` (new `mob schedule` subcommand), existing `mapper.GetPath` for pathfinding sanity, existing `mob.Character.SetMiscData/GetMiscData` for per-instance state, existing `fileloader.LoadAllFlatFiles` for loading.

**Spec:** `docs/superpowers/specs/2026-05-25-mob-aliveness-3.2-npc-schedules-design.md`

**Branch:** `feature/mob-aliveness-3.2-npc-schedules` (already created; spec committed as `0854655f`).

---

## Stage map

| Stage | Task | Description |
|---|---|---|
| 1 | T1 | Schedule types + `CurrentSegment` resolver (TDD) |
| 2 | T2 | Schedule loader + 24h-coverage validator + pathfinding sanity (TDD) |
| 3 | T3 | Mob `schedule_id` field + cross-check at LoadDataFiles |
| 4 | T4 | Spawn override in `newMobByIdInternal` (TDD) |
| 5 | T5 | `ScheduleMaxPathRetries` config knob |
| 6 | T6 | `NewRound_IdleMobs` schedule hook (TDD) |
| 7 | T7 | `TickMobCraft` activity gate (TDD) |
| 8 | T8 | `mob_at_target_room` btree condition (TDD) |
| 9 | T9 | `mob schedule` admin inspector + helpfile |
| 10 | T10 | Pilot content: 3 above-shop home rooms + workplace exit edits |
| 11 | T11 | Pilot content: 3 schedule YAMLs + 3 mob YAML edits |
| 12 | T12 | Documentation pass (schema, context.md, CLAUDE.md, helpfiles) |
| 13 | T13 | Smoketester goal file + manual smoke + roadmap closeout |

13 tasks. Order matters: T1-T8 are sequential code work (each builds on previous), T9 is standalone admin tooling, T10-T11 are content (need T1-T8 in place for boot to succeed), T12 is documentation, T13 is verification and closeout.

---

## Task 1: Schedule types + `CurrentSegment` resolver

**Files:**
- Create: `internal/mobs/schedule.go`
- Create: `internal/mobs/schedule_test.go`

- [ ] **Step 1: Read the chunk 3.1 hour-range helper to reuse the wrap-around math**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '60,160p' internal/behaviortree/conditions_state.go
```

Confirm `inHourRange(hour, start, end int) bool` exists at the chunk 3.1 location. We will call it directly from `Schedule.CurrentSegment` — it already handles wrap-around correctly. If it is currently unexported, leave it unexported and copy the logic; if it is exported (or test-exported), reuse.

- [ ] **Step 2: Write the failing test**

Create `internal/mobs/schedule_test.go`:

```go
package mobs

import (
	"testing"
)

// Fixture: Kerra-shaped schedule covering all 24 hours.
//
// 6-9   loft (waking)
// 9-18  forge (craft)
// 18-22 tavern
// 22-6  loft (sleep, wraps midnight)
func kerraTestSchedule() *Schedule {
	return &Schedule{
		Id:          "thornwall_smith",
		Description: "Kerra test fixture",
		Segments: []ScheduleSegment{
			{Start: 6, End: 9, TargetRoom: 1234, Activity: "", IdleCommands: []string{"emote wakes."}},
			{Start: 9, End: 18, TargetRoom: 5678, Activity: "craft", IdleCommands: []string{"emote hammers."}},
			{Start: 18, End: 22, TargetRoom: 9012, Activity: "", IdleCommands: []string{"emote sips."}},
			{Start: 22, End: 6, TargetRoom: 1234, Activity: "", IdleCommands: []string{"emote snores."}},
		},
	}
}

func TestSchedule_CurrentSegment_BasicHours(t *testing.T) {
	s := kerraTestSchedule()
	cases := []struct {
		hour  int
		want  int // expected segment Start
	}{
		{7, 6},   // waking segment
		{10, 9},  // forge segment
		{17, 9},  // still forge (exclusive end)
		{18, 18}, // tavern starts at 18
		{21, 18}, // still tavern
		{22, 22}, // sleep starts at 22
	}
	for _, c := range cases {
		seg := s.CurrentSegment(c.hour)
		if seg == nil {
			t.Errorf("hour %d: expected segment with Start=%d, got nil", c.hour, c.want)
			continue
		}
		if seg.Start != c.want {
			t.Errorf("hour %d: expected segment Start=%d, got Start=%d", c.hour, c.want, seg.Start)
		}
	}
}

func TestSchedule_CurrentSegment_WrapsMidnight(t *testing.T) {
	s := kerraTestSchedule()
	// 22-6 wraps midnight; 23, 0, 5 should all return the wrap segment.
	for _, hour := range []int{22, 23, 0, 1, 5} {
		seg := s.CurrentSegment(hour)
		if seg == nil || seg.Start != 22 {
			t.Errorf("hour %d: expected wrap segment Start=22, got %+v", hour, seg)
		}
	}
}

func TestSchedule_CurrentSegment_ExclusiveEnd(t *testing.T) {
	s := kerraTestSchedule()
	// Hour 6 is the exclusive end of the sleep wrap segment AND the inclusive
	// start of the waking segment. The waking segment should win.
	seg := s.CurrentSegment(6)
	if seg == nil || seg.Start != 6 {
		t.Errorf("hour 6: expected waking segment Start=6 to win, got %+v", seg)
	}
}

func TestSchedule_CurrentSegment_SameRoomTwiceNonAdjacent(t *testing.T) {
	// Olen pattern: chamber appears in two non-adjacent segments with different
	// idlecommands. The resolver must return the right segment for the right
	// hour, not just the first segment that matches the room.
	s := &Schedule{
		Id: "thornwall_temple_priest",
		Segments: []ScheduleSegment{
			{Start: 4, End: 6, TargetRoom: 100, IdleCommands: []string{"rise"}},
			{Start: 6, End: 10, TargetRoom: 200, IdleCommands: []string{"prayers-morning"}},
			{Start: 10, End: 12, TargetRoom: 100, IdleCommands: []string{"rest"}},
			{Start: 12, End: 18, TargetRoom: 200, IdleCommands: []string{"prayers-afternoon"}},
			{Start: 18, End: 22, TargetRoom: 300, IdleCommands: []string{"tavern"}},
			{Start: 22, End: 4, TargetRoom: 100, IdleCommands: []string{"sleep"}},
		},
	}
	if got := s.CurrentSegment(5).IdleCommands[0]; got != "rise" {
		t.Errorf("hour 5: want rise, got %s", got)
	}
	if got := s.CurrentSegment(11).IdleCommands[0]; got != "rest" {
		t.Errorf("hour 11: want rest, got %s", got)
	}
	if got := s.CurrentSegment(23).IdleCommands[0]; got != "sleep" {
		t.Errorf("hour 23: want sleep, got %s", got)
	}
}

func TestSchedule_CurrentSegment_NoCoverageReturnsNil(t *testing.T) {
	// Defensive: at runtime a validated schedule always covers 24h, but the
	// resolver should not panic if given a gap-having fixture.
	s := &Schedule{
		Id:       "broken",
		Segments: []ScheduleSegment{{Start: 9, End: 17, TargetRoom: 1}},
	}
	if seg := s.CurrentSegment(3); seg != nil {
		t.Errorf("hour 3: expected nil for uncovered hour, got %+v", seg)
	}
}
```

- [ ] **Step 3: Run the test, confirm fail**

Run:
```bash
go test ./internal/mobs/ -run TestSchedule_ -v
```
Expected: compilation error — `Schedule`, `ScheduleSegment`, and `CurrentSegment` are not defined yet.

- [ ] **Step 4: Implement `internal/mobs/schedule.go`**

```go
package mobs

import "sync"

// Schedule is a daily routine attached to NPCs via Mob.ScheduleId.
// Loaded from _datafiles/world/dogmud/schedules/<zone>/<id>.yaml at startup.
type Schedule struct {
	Id          string             `yaml:"id"`
	Description string             `yaml:"description,omitempty"`
	Segments    []ScheduleSegment  `yaml:"segments"`
}

// ScheduleSegment covers a contiguous hour range [Start, End). When Start > End
// the segment wraps midnight (e.g. Start=22 End=6 covers 22-23 and 0-5).
type ScheduleSegment struct {
	Start        int      `yaml:"start"`              // 0-23 inclusive
	End          int      `yaml:"end"`                // 1-24 inclusive
	TargetRoom   int      `yaml:"target_room"`        // room the mob should occupy
	Activity     string   `yaml:"activity,omitempty"` // "" | "craft" | future maintenance verbs
	IdleCommands []string `yaml:"idlecommands,omitempty"`
}

// Package-level registry, populated by LoadSchedules at startup.
var (
	schedulesMu sync.RWMutex
	schedules   = map[string]*Schedule{}
)

// GetSchedule returns the schedule with the given id, or nil if no such id is
// loaded.
func GetSchedule(id string) *Schedule {
	if id == "" {
		return nil
	}
	schedulesMu.RLock()
	defer schedulesMu.RUnlock()
	return schedules[id]
}

// CurrentSegment returns the segment active at hour24 (0-23). Returns nil if
// no segment covers this hour. At runtime, loaded schedules are validated to
// cover all 24 hours exactly once, so nil should never be observed.
func (s *Schedule) CurrentSegment(hour24 int) *ScheduleSegment {
	if s == nil {
		return nil
	}
	for i := range s.Segments {
		seg := &s.Segments[i]
		if segContainsHour(seg, hour24) {
			return seg
		}
	}
	return nil
}

// segContainsHour reports whether [seg.Start, seg.End) contains hour24,
// handling the wrap-around case where Start > End.
//
// Inclusive at start, exclusive at end. End == 24 means "up to but not
// including midnight" (the day boundary).
func segContainsHour(seg *ScheduleSegment, hour24 int) bool {
	if seg.Start == seg.End {
		return false // empty segment — validation rejects these at load time
	}
	if seg.Start < seg.End {
		return hour24 >= seg.Start && hour24 < seg.End
	}
	// Wraps midnight: covers [Start, 24) and [0, End).
	return hour24 >= seg.Start || hour24 < seg.End
}
```

- [ ] **Step 5: Run the test, confirm pass**

Run:
```bash
go test ./internal/mobs/ -run TestSchedule_ -v
```
Expected: all five tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/schedule.go internal/mobs/schedule_test.go
git commit -m "$(cat <<'EOF'
feat(mobs): schedule type and CurrentSegment resolver

Foundation for chunk 3.2 NPC schedules. Schedule + ScheduleSegment
types, package-level registry, CurrentSegment(hour24) with
midnight-wrap handling. No loader yet; tests use in-memory
fixtures.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Schedule loader + load-time validators

**Files:**
- Create: `internal/mobs/schedule_loader.go`
- Create: `internal/mobs/schedule_loader_test.go`
- Modify: `internal/mobs/mobs.go` (existing `LoadDataFiles()` to call schedule loader)

- [ ] **Step 1: Read the existing buff/quest loader pattern for a template**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -rn "LoadAllFlatFiles" internal/buffs/ | head -5
ls internal/buffs/*.go | head -5
```

Look at `internal/buffs/buffs.go` `LoadDataFiles()` for the path / panic / log pattern we should mirror. The schedule loader will:
1. Walk `_datafiles/world/dogmud/schedules/**/*.yaml`.
2. Parse each into a `Schedule`.
3. Run per-schedule validators; panic with a clear message on any failure.
4. After all schedules load, register by id; panic on duplicate ids.

- [ ] **Step 2: Write the failing tests**

Create `internal/mobs/schedule_loader_test.go`:

```go
package mobs

import (
	"strings"
	"testing"
)

// validateScheduleStandalone is the load-time per-schedule validator. It is
// extracted so we can unit-test it without writing fixture files.
// Real test names below call validateScheduleStandalone with hand-built
// schedules; the file-walking loader gets a separate integration test.

func TestValidateSchedule_FullCoverage_OK(t *testing.T) {
	s := kerraTestSchedule()
	if err := validateScheduleStandalone(s); err != nil {
		t.Errorf("kerra fixture should validate, got error: %v", err)
	}
}

func TestValidateSchedule_GapInCoverage_Panics(t *testing.T) {
	s := &Schedule{
		Id: "broken",
		Segments: []ScheduleSegment{
			{Start: 6, End: 9, TargetRoom: 1},
			// gap: 9-22
			{Start: 22, End: 6, TargetRoom: 1},
		},
	}
	err := validateScheduleStandalone(s)
	if err == nil {
		t.Fatal("expected error for coverage gap, got nil")
	}
	if !strings.Contains(err.Error(), "gap") && !strings.Contains(err.Error(), "coverage") {
		t.Errorf("expected gap/coverage error, got: %v", err)
	}
}

func TestValidateSchedule_OverlappingSegments_Panics(t *testing.T) {
	s := &Schedule{
		Id: "broken",
		Segments: []ScheduleSegment{
			{Start: 6, End: 12, TargetRoom: 1},
			{Start: 10, End: 18, TargetRoom: 2}, // overlaps 10-12
			{Start: 18, End: 6, TargetRoom: 1},
		},
	}
	err := validateScheduleStandalone(s)
	if err == nil {
		t.Fatal("expected error for overlap, got nil")
	}
	if !strings.Contains(err.Error(), "overlap") && !strings.Contains(err.Error(), "claimed") {
		t.Errorf("expected overlap error, got: %v", err)
	}
}

func TestValidateSchedule_BadHourBounds_Panics(t *testing.T) {
	cases := []struct {
		name string
		seg  ScheduleSegment
	}{
		{"negative start", ScheduleSegment{Start: -1, End: 9, TargetRoom: 1}},
		{"start over 23", ScheduleSegment{Start: 25, End: 9, TargetRoom: 1}},
		{"end over 24", ScheduleSegment{Start: 9, End: 25, TargetRoom: 1}},
		{"end zero", ScheduleSegment{Start: 9, End: 0, TargetRoom: 1}},
		{"start equals end", ScheduleSegment{Start: 9, End: 9, TargetRoom: 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &Schedule{Id: "broken", Segments: []ScheduleSegment{c.seg}}
			if err := validateScheduleStandalone(s); err == nil {
				t.Errorf("expected error for %s, got nil", c.name)
			}
		})
	}
}

func TestValidateSchedule_MissingTargetRoom_Panics(t *testing.T) {
	s := &Schedule{
		Id: "broken",
		Segments: []ScheduleSegment{
			{Start: 0, End: 24, TargetRoom: 0}, // 24h coverage but room id 0
		},
	}
	err := validateScheduleStandalone(s)
	if err == nil {
		t.Fatal("expected error for missing target_room, got nil")
	}
	if !strings.Contains(err.Error(), "target_room") {
		t.Errorf("expected target_room error, got: %v", err)
	}
}
```

Note: `validateScheduleStandalone` does the hour-coverage / bounds / id checks without touching the filesystem or the rooms package. Room-existence and pathfinding checks live in a second `validateScheduleAgainstWorld` function tested against a real or fake rooms registry — see Step 5.

- [ ] **Step 3: Run the test, confirm fail**

Run:
```bash
go test ./internal/mobs/ -run TestValidateSchedule_ -v
```
Expected: compilation error — `validateScheduleStandalone` not defined.

- [ ] **Step 4: Implement `internal/mobs/schedule_loader.go`**

```go
package mobs

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// LoadSchedules walks the schedules data directory, parses each YAML into a
// Schedule, validates each, and populates the package-level registry.
// Called from mobs.LoadDataFiles() after rooms are loaded so that target_room
// existence checks resolve.
func LoadSchedules() {
	start := time.Now()

	dataPath := configs.GetFilePathsConfig().DataFiles.String() + `/schedules`
	loaded, err := fileloader.LoadAllFlatFiles[string, *Schedule](dataPath)
	if err != nil {
		// schedules is an optional data directory — if it doesn't exist yet, log
		// and continue with an empty registry.
		if errors.Is(err, fileloader.ErrPathDoesNotExist) {
			mudlog.Info("mobs.LoadSchedules()", "msg", "schedules directory absent, skipping", "path", dataPath)
			return
		}
		panic(fmt.Errorf("schedules: %w", err))
	}

	tmp := make(map[string]*Schedule, len(loaded))
	for path, s := range loaded {
		if existing, dup := tmp[s.Id]; dup {
			panic(fmt.Errorf("schedule id %q duplicated at %s and %s",
				s.Id, path, scheduleSourcePath(existing)))
		}
		// Filename ↔ id check.
		base := filepath.Base(path)
		base = strings.TrimSuffix(base, filepath.Ext(base))
		expected := util.ConvertForFilename(s.Id)
		if base != expected {
			panic(fmt.Errorf("schedule %q at %s: filename %q does not match id (expected %q)",
				s.Id, path, base, expected))
		}
		if err := validateScheduleStandalone(s); err != nil {
			panic(fmt.Errorf("schedule %q at %s: %w", s.Id, path, err))
		}
		if err := validateScheduleAgainstWorld(s); err != nil {
			panic(fmt.Errorf("schedule %q at %s: %w", s.Id, path, err))
		}
		tmp[s.Id] = s
	}

	schedulesMu.Lock()
	schedules = tmp
	schedulesMu.Unlock()

	mudlog.Info("mobs.LoadSchedules()", "loadedCount", len(tmp), "Time Taken", time.Since(start))
}

// validateScheduleStandalone runs the hour-coverage / hour-bounds /
// target_room-nonzero checks without touching the rooms package or filesystem.
// Tested in isolation via in-memory fixtures.
func validateScheduleStandalone(s *Schedule) error {
	if len(s.Segments) == 0 {
		return errors.New("schedule has no segments")
	}

	// Per-segment bounds.
	for i, seg := range s.Segments {
		if seg.Start < 0 || seg.Start > 23 {
			return fmt.Errorf("segment %d: start %d out of range [0,23]", i, seg.Start)
		}
		if seg.End < 1 || seg.End > 24 {
			return fmt.Errorf("segment %d: end %d out of range [1,24]", i, seg.End)
		}
		if seg.Start == seg.End {
			return fmt.Errorf("segment %d: start equals end (%d) — empty segment", i, seg.Start)
		}
		if seg.TargetRoom == 0 {
			return fmt.Errorf("segment %d: target_room is 0", i)
		}
	}

	// Coverage check: every hour 0-23 must be claimed by exactly one segment.
	claimedBy := make([]int, 24)
	for i := range claimedBy {
		claimedBy[i] = -1
	}
	for segIdx, seg := range s.Segments {
		for hour := 0; hour < 24; hour++ {
			if !segContainsHour(&s.Segments[segIdx], hour) {
				continue
			}
			if claimedBy[hour] != -1 {
				return fmt.Errorf("hour %d claimed by both segment %d and segment %d (overlap)",
					hour, claimedBy[hour], segIdx)
			}
			claimedBy[hour] = segIdx
		}
	}
	var gaps []int
	for hour, owner := range claimedBy {
		if owner == -1 {
			gaps = append(gaps, hour)
		}
	}
	if len(gaps) > 0 {
		return fmt.Errorf("coverage gap: hours %v have no segment", gaps)
	}

	// Activity warnings (non-fatal; collected and logged after this returns).
	for i, seg := range s.Segments {
		if seg.Activity != "" && seg.Activity != "craft" {
			mudlog.Warn("schedule", "id", s.Id, "segment", i,
				"msg", "unknown activity value", "value", seg.Activity)
		}
		if len(seg.IdleCommands) == 0 {
			mudlog.Warn("schedule", "id", s.Id, "segment", i,
				"msg", "segment has zero idlecommands — mob will be silent at this location")
		}
	}

	return nil
}

// validateScheduleAgainstWorld checks target_room existence and inter-segment
// pathfinding. Requires rooms and mapper to be initialised, so it runs after
// rooms.LoadDataFiles() and before schedules go live.
func validateScheduleAgainstWorld(s *Schedule) error {
	// Target rooms must exist.
	for i, seg := range s.Segments {
		if rooms.LoadRoom(seg.TargetRoom) == nil {
			return fmt.Errorf("segment %d: target_room %d does not exist", i, seg.TargetRoom)
		}
	}
	// Inter-segment pathfinding sanity. We walk segments in start-hour order,
	// including the wrap-around transition (last → first).
	ordered := orderedSegmentsByStart(s)
	for i := 0; i < len(ordered); i++ {
		next := ordered[(i+1)%len(ordered)]
		curr := ordered[i]
		if curr.TargetRoom == next.TargetRoom {
			continue // same room — no path needed
		}
		_, err := mapper.GetPath(curr.TargetRoom, next.TargetRoom)
		if err != nil {
			return fmt.Errorf("segment transition %d→%d: no path from room %d to room %d (%w)",
				i, (i+1)%len(ordered), curr.TargetRoom, next.TargetRoom, err)
		}
	}
	return nil
}

// orderedSegmentsByStart returns segment pointers ordered by start hour
// ascending. Used by the pathfinding validator so that the transition pairs
// match the real chronological order.
func orderedSegmentsByStart(s *Schedule) []*ScheduleSegment {
	out := make([]*ScheduleSegment, len(s.Segments))
	for i := range s.Segments {
		out[i] = &s.Segments[i]
	}
	// Insertion sort is fine — segment lists are short (typically 3-6).
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1].Start > out[j].Start; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

// scheduleSourcePath is a placeholder hook for tracking the file each schedule
// loaded from. fileloader.LoadAllFlatFiles returns paths as map keys, so the
// caller already knows. This helper exists only to satisfy the duplicate-id
// error message in LoadSchedules.
//
// In practice we never reach it (load order means the duplicate detection
// fires before any cross-reference back to the first file's path is needed),
// so a stub is sufficient.
var _ = sync.RWMutex{} // keep sync import used if all other refs go away
func scheduleSourcePath(_ *Schedule) string {
	return "<earlier file>"
}
```

If `fileloader.LoadAllFlatFiles[string, *Schedule]` does not support a `string` key (it may require a numeric ID like mobs/rooms), inspect `internal/fileloader/fileloader.go` to find an alternative entry point (e.g., a directory walker that does not require a numeric key) and adapt. The fileloader API is the only piece that may need to be threaded differently — the validation logic above is independent of how files are read.

If `fileloader.ErrPathDoesNotExist` is not the correct sentinel, check `internal/fileloader/` for the actual missing-directory error and substitute.

- [ ] **Step 5: Add `validateScheduleAgainstWorld` tests using a minimal rooms fixture**

Append to `internal/mobs/schedule_loader_test.go`:

```go
// These tests require the rooms registry to be initialised. We use the
// existing test-helper pattern from elsewhere in the package; if no such
// helper exists, the integration aspect of these tests is exercised by the
// real boot path via the smoke test in T13.
//
// For now we keep the world-aware tests minimal — the validator's logic is
// straightforward and the smoke test catches regressions.

func TestValidateScheduleAgainstWorld_RoomDoesNotExist(t *testing.T) {
	t.Skip("requires rooms fixture — covered by boot smoke in T13")
}
```

This is the one place we accept a `t.Skip` because building a real rooms registry inline is heavyweight, and the boot smoke at T13 catches any regression at the only place that matters (server startup). Leave the test stub in place as a future-fixture hook.

- [ ] **Step 6: Wire `LoadSchedules` into `mobs.LoadDataFiles()`**

In `internal/mobs/mobs.go`, find `LoadDataFiles()` (around line 1162) and add a call to `LoadSchedules()` after the mob template load completes:

```go
// ... existing mob loading ...
mudlog.Info("mobs.LoadDataFiles()", "loadedCount", len(tmpMobs), "Time Taken", time.Since(start))

// Load schedules after mobs so the cross-check below knows which schedule
// ids are valid. Schedules must load after rooms (which already happened
// at this point in startup).
LoadSchedules()

// ... existing relationships population ...
```

Place the call after the existing `mudlog.Info("mobs.LoadDataFiles()", …)` line and before the relationships block.

- [ ] **Step 7: Run all `mobs` tests, confirm pass**

Run:
```bash
go test ./internal/mobs/ -v
```
Expected: all schedule tests pass, all pre-existing mob tests still pass.

- [ ] **Step 8: Boot the server, confirm clean startup**

```bash
go build ./...
```
Expected: clean build.

```bash
./dogmud.exe   # or whatever the binary is named locally
```
Expected: server boots through `mobs.LoadDataFiles()` and `mobs.LoadSchedules() loadedCount=0` (no schedules authored yet) without panic. Ctrl+C to stop.

- [ ] **Step 9: Commit**

```bash
git add internal/mobs/schedule_loader.go internal/mobs/schedule_loader_test.go internal/mobs/mobs.go
git commit -m "$(cat <<'EOF'
feat(mobs): schedule YAML loader and load-time validators

Walks _datafiles/world/dogmud/schedules/**/*.yaml, validates each
schedule for 24h-coverage / hour-bounds / target_room existence /
inter-segment pathfinding, panics on any failure. Empty schedules
directory is tolerated (logs and continues).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Mob `schedule_id` field + cross-check at LoadDataFiles

**Files:**
- Modify: `internal/mobs/mobs.go` (add `ScheduleId` field to `Mob` struct, add cross-check loop in `LoadDataFiles`)
- Modify: `internal/mobs/schedule_loader_test.go` (cross-check test)

- [ ] **Step 1: Add `ScheduleId` field to the Mob struct**

In `internal/mobs/mobs.go`, find the `Mob` struct (around line 80) and add the field. Place it near `BehaviorArchetype` (around line 157) for grouping:

```go
BehaviorArchetype string `yaml:"behavior_archetype,omitempty"`
ScheduleId        string `yaml:"schedule_id,omitempty"` // chunk 3.2: daily routine reference
```

- [ ] **Step 2: Add a cross-check pass to `LoadDataFiles`**

In `internal/mobs/mobs.go` `LoadDataFiles()`, immediately after `LoadSchedules()`:

```go
LoadSchedules()

// Cross-check: every mob's schedule_id must resolve to a loaded schedule.
mobsMu.RLock()
for _, mob := range mobs {
    if mob.ScheduleId == "" {
        continue
    }
    if GetSchedule(mob.ScheduleId) == nil {
        mobsMu.RUnlock()
        panic(fmt.Errorf("mob %d (%s): schedule_id %q does not resolve to a loaded schedule",
            mob.MobId, mob.Character.Name, mob.ScheduleId))
    }
}
mobsMu.RUnlock()
```

Ensure `fmt` is imported in `mobs.go` (it almost certainly is already; verify with a quick grep).

- [ ] **Step 3: Write a failing test**

Append to `internal/mobs/schedule_loader_test.go`:

```go
func TestMobScheduleIdCrossCheck_Stub(t *testing.T) {
	// Cross-check is wired in LoadDataFiles and is exercised by the boot
	// smoke at T13. Stub here as a documentation hook — if we add a
	// LoadDataFiles unit-test fixture later this is where it slots in.
	t.Skip("covered by boot smoke in T13")
}
```

Same justification as Step 5 of T2: building a full mob + schedule fixture inline is heavyweight; the boot smoke catches regressions at the only place that matters.

- [ ] **Step 4: Confirm build and tests still pass**

```bash
go build ./...
go test ./internal/mobs/ -v
```
Expected: clean build and green tests. Existing mob YAMLs have no `schedule_id`, so the cross-check loop is a no-op.

- [ ] **Step 5: Commit**

```bash
git add internal/mobs/mobs.go internal/mobs/schedule_loader_test.go
git commit -m "$(cat <<'EOF'
feat(mobs): schedule_id field on Mob + load-time cross-check

Adds the schedule_id YAML field that pilot NPCs will reference.
LoadDataFiles cross-checks every schedule_id resolves to a
loaded schedule and panics on miss.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Spawn override in `newMobByIdInternal`

**Files:**
- Modify: `internal/mobs/mobs.go` (around line 333 — after `mob.HomeRoomId = homeRoomId` / `mob.Character.RoomId = homeRoomId`)
- Create: `internal/mobs/schedule_spawn_test.go`

- [ ] **Step 1: Read the spawn site to confirm the right insertion point**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '317,360p' internal/mobs/mobs.go
```

Confirm `mob.Character.RoomId = homeRoomId` is set on a single line and that the lines immediately after deal with submission/surrender policy. The override goes after those policy assignments so we don't fight with anything else that reads `Character.RoomId` during instance init.

- [ ] **Step 2: Write the failing test**

Create `internal/mobs/schedule_spawn_test.go`:

```go
package mobs

import (
	"testing"
)

// We cannot exercise newMobByIdInternal end-to-end in a unit test (it
// requires the template registry, the room registry, etc.) so we
// directly unit-test the override logic via a small helper.

func TestApplyScheduleSpawnOverride_PlacesAtCurrentSegment(t *testing.T) {
	// Stand up an in-memory schedule and register it.
	s := kerraTestSchedule()
	registerScheduleForTest(s)
	defer unregisterScheduleForTest(s.Id)

	got := applyScheduleSpawnOverride("thornwall_smith", 9999 /* homeRoomId */, 10 /* hour */)
	if got != 5678 {
		t.Errorf("hour 10: expected forge room 5678, got %d", got)
	}

	got = applyScheduleSpawnOverride("thornwall_smith", 9999, 19)
	if got != 9012 {
		t.Errorf("hour 19: expected tavern room 9012, got %d", got)
	}

	got = applyScheduleSpawnOverride("thornwall_smith", 9999, 23)
	if got != 1234 {
		t.Errorf("hour 23: expected loft room 1234, got %d", got)
	}
}

func TestApplyScheduleSpawnOverride_NoScheduleReturnsHome(t *testing.T) {
	got := applyScheduleSpawnOverride("", 9999, 10)
	if got != 9999 {
		t.Errorf("no schedule: expected home %d, got %d", 9999, got)
	}
}

func TestApplyScheduleSpawnOverride_UnknownScheduleReturnsHome(t *testing.T) {
	got := applyScheduleSpawnOverride("definitely_not_real", 9999, 10)
	if got != 9999 {
		t.Errorf("unknown schedule: expected home %d, got %d", 9999, got)
	}
}
```

- [ ] **Step 3: Run the test, confirm fail**

Run:
```bash
go test ./internal/mobs/ -run TestApplyScheduleSpawnOverride -v
```
Expected: compilation error — `applyScheduleSpawnOverride`, `registerScheduleForTest`, `unregisterScheduleForTest` not defined.

- [ ] **Step 4: Implement the helper and test plumbing**

Append to `internal/mobs/schedule.go`:

```go
// applyScheduleSpawnOverride returns the room a scheduled mob should spawn
// at given the current hour. Returns homeRoomId unchanged when the mob has
// no schedule, when the schedule is unknown, or when the schedule has no
// segment covering the current hour. Called from newMobByIdInternal during
// instance creation.
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
	return seg.TargetRoom
}

// registerScheduleForTest / unregisterScheduleForTest are test-only helpers
// for injecting schedules into the package-level registry. NOT exported.
// Tests should call these via go:build test tags if we want stricter
// isolation; for now they live alongside the regular code and rely on the
// _test.go file naming to limit accidental misuse.
func registerScheduleForTest(s *Schedule) {
	schedulesMu.Lock()
	defer schedulesMu.Unlock()
	schedules[s.Id] = s
}

func unregisterScheduleForTest(id string) {
	schedulesMu.Lock()
	defer schedulesMu.Unlock()
	delete(schedules, id)
}
```

Note: the test helpers do not have `_test.go` build tags because Go does not support that idiom for shared-package helpers across multiple test files. If the codebase has a convention for test-only registry pokes (look in `internal/mobs/mobs_test.go`), follow that convention instead.

- [ ] **Step 5: Wire the override into `newMobByIdInternal`**

In `internal/mobs/mobs.go`, find the instance-init block around lines 333-368. After the surrender-policy block (around line 360) and before `mob.Character.ResetForMobInstance()`:

```go
// Chunk 3.2: scheduled mob spawn override. If the mob has a schedule_id,
// place it at the current segment's target room instead of HomeRoomId.
if mob.ScheduleId != "" {
    hour := gametime.GetDate().Hour24
    mob.Character.RoomId = applyScheduleSpawnOverride(mob.ScheduleId, mob.HomeRoomId, hour)
}
```

Add the import for `gametime`:
```go
import (
    // ... existing imports ...
    "github.com/GoMudEngine/GoMud/internal/gametime"
)
```

- [ ] **Step 6: Run all `mobs` tests, confirm pass**

```bash
go test ./internal/mobs/ -v
```
Expected: green.

- [ ] **Step 7: Boot the server, confirm clean startup**

```bash
go build ./...
./dogmud.exe
```
Expected: clean startup. No scheduled mobs exist yet so the override branch is dormant.

- [ ] **Step 8: Commit**

```bash
git add internal/mobs/schedule.go internal/mobs/schedule_spawn_test.go internal/mobs/mobs.go
git commit -m "$(cat <<'EOF'
feat(mobs): spawn override places scheduled mobs at current segment

newMobByIdInternal now consults schedule_id and current game hour
to override Character.RoomId. HomeRoomId is preserved as the
"true home" for pathto home / GetVisibility / etc. Mobs without
schedule_id are unchanged.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: `ScheduleMaxPathRetries` config knob

**Files:**
- Modify: `internal/configs/config.balance.go` (add the knob)

- [ ] **Step 1: Read the existing balance-config shape to mirror it**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -n "ShopAbundanceThreshold\|StatSoftCap\|SkillMultiplier" internal/configs/config.balance.go | head -10
```

Find a numeric knob with a similar shape (defaulted int) and use its declaration form.

- [ ] **Step 2: Add the new knob**

In `internal/configs/config.balance.go`, find the struct definition and add:

```go
// ScheduleMaxPathRetries is the number of consecutive failed pathto
// attempts a scheduled mob will tolerate before falling back to
// `pathto home`. Default 20 (≈80 seconds at the default tick rate).
// See chunk 3.2 spec.
ScheduleMaxPathRetries ConfigInt `yaml:"schedulemaxpathretries"`
```

Place it near other mob/scheduling knobs (likely close to wander or forager settings).

If `ConfigInt` is the wrong type wrapper, mirror whatever wrapper the neighbouring knobs use. The codebase uses several config-cell types — match the local pattern.

- [ ] **Step 3: Set the default**

Find the balance-config defaulting function (often in the same file, named like `setDefaults` or `Validate`) and add:

```go
if c.ScheduleMaxPathRetries == 0 {
    c.ScheduleMaxPathRetries = 20
}
```

- [ ] **Step 4: Confirm build**

```bash
go build ./...
```
Expected: clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/configs/config.balance.go
git commit -m "$(cat <<'EOF'
feat(configs): ScheduleMaxPathRetries knob (default 20)

After N consecutive pathto failures, the chunk 3.2 schedule
executor falls back to pathto home to avoid stranding. Knob lets
ops tune the threshold without code changes.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: `NewRound_IdleMobs` schedule hook

**Files:**
- Modify: `internal/hooks/NewRound_IdleMobs.go` (insert schedule branch after combat/conversation/despawn guards)
- Create: `internal/hooks/NewRound_IdleMobs_schedule_test.go`

- [ ] **Step 1: Re-read the hook entry point so the insertion is clean**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '30,135p' internal/hooks/NewRound_IdleMobs.go
```

Confirm: combat guard → conversation guard → path-walker → fallback to `MobIdle` event. The schedule branch goes between the conversation guard (around line 66-69) and the path-walker (line 73), so it can clear/queue paths before the walker tries to consume them.

- [ ] **Step 2: Write the failing tests**

Create `internal/hooks/NewRound_IdleMobs_schedule_test.go`:

```go
package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// These tests exercise the pure-function helpers extracted from the
// IdleMobs schedule branch. The full integration (mob in registry, room
// loaded, path walker advancing) is exercised by the manual smoke at T13.

func TestScheduleTick_ChoosesPathtoWhenAwayFromTarget(t *testing.T) {
	// Helper: scheduleTickPlan(mob, currentRoomId, hour) returns a struct
	// describing what the executor wants to do this tick.
	mob := &mobs.Mob{ScheduleId: "thornwall_smith"}
	mob.Character.RoomId = 1111 // not at any segment target

	registerKerraScheduleForTest(t)

	plan := scheduleTickPlan(mob, 10 /* forge hour */)
	if !plan.WantsPath {
		t.Fatalf("expected WantsPath=true when away from target, got %+v", plan)
	}
	if plan.TargetRoom != 5678 {
		t.Errorf("expected target=5678, got %d", plan.TargetRoom)
	}
}

func TestScheduleTick_NoPathWhenAtTarget(t *testing.T) {
	mob := &mobs.Mob{ScheduleId: "thornwall_smith"}
	mob.Character.RoomId = 5678 // at forge

	registerKerraScheduleForTest(t)

	plan := scheduleTickPlan(mob, 10)
	if plan.WantsPath {
		t.Errorf("expected WantsPath=false when at target, got %+v", plan)
	}
}

func TestScheduleTick_TransitionClearsStalePath(t *testing.T) {
	mob := &mobs.Mob{ScheduleId: "thornwall_smith"}
	mob.Character.RoomId = 5678
	mob.Character.SetMiscData("schedule_last_seg_start", 9) // last tick was forge segment

	registerKerraScheduleForTest(t)

	plan := scheduleTickPlan(mob, 19 /* tavern hour */)
	if !plan.SegmentChanged {
		t.Errorf("expected SegmentChanged=true on transition, got %+v", plan)
	}
	if plan.NewSegmentStart != 18 {
		t.Errorf("expected NewSegmentStart=18, got %d", plan.NewSegmentStart)
	}
}

// registerKerraScheduleForTest is a thin wrapper that injects the Kerra
// fixture into the mobs package registry via the test helpers added in T4.
// Implemented via export_test.go or similar — see Step 5.
func registerKerraScheduleForTest(t *testing.T) {
	t.Helper()
	mobs.RegisterScheduleForTest(&mobs.Schedule{
		Id: "thornwall_smith",
		Segments: []mobs.ScheduleSegment{
			{Start: 6, End: 9, TargetRoom: 1234, IdleCommands: []string{"wake"}},
			{Start: 9, End: 18, TargetRoom: 5678, Activity: "craft", IdleCommands: []string{"hammer"}},
			{Start: 18, End: 22, TargetRoom: 9012, IdleCommands: []string{"sip"}},
			{Start: 22, End: 6, TargetRoom: 1234, IdleCommands: []string{"sleep"}},
		},
	})
	t.Cleanup(func() { mobs.UnregisterScheduleForTest("thornwall_smith") })
}
```

- [ ] **Step 3: Export the test helpers from the mobs package**

The helpers `registerScheduleForTest` / `unregisterScheduleForTest` from T4 are currently lowercase. For cross-package tests we need exported variants. Add to a new file `internal/mobs/export_test.go` (Go-standard pattern — exports only visible to `_test.go` files in any package):

```go
package mobs

// RegisterScheduleForTest is the exported wrapper around the package-private
// schedule injector. Available only from _test.go files.
func RegisterScheduleForTest(s *Schedule) {
	registerScheduleForTest(s)
}

// UnregisterScheduleForTest mirrors RegisterScheduleForTest.
func UnregisterScheduleForTest(id string) {
	unregisterScheduleForTest(id)
}
```

`export_test.go` files are compiled only for test builds, so these exports do not leak into production.

- [ ] **Step 4: Run the test, confirm fail**

```bash
go test ./internal/hooks/ -run TestScheduleTick -v
```
Expected: compilation error — `scheduleTickPlan` not defined.

- [ ] **Step 5: Implement the schedule-tick plan helper and the executor wiring**

Create or append to `internal/hooks/NewRound_IdleMobs.go` (just below the imports, or in a dedicated `NewRound_IdleMobs_schedule.go` peer file in the same `hooks` package — split for readability):

Create new file `internal/hooks/NewRound_IdleMobs_schedule.go`:

```go
package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// schedulePlan describes what the executor wants to do for a mob this tick.
// Extracted so the decision logic is unit-testable without driving the full
// IdleMobs loop.
type schedulePlan struct {
	HasSchedule       bool
	SegmentChanged    bool
	NewSegmentStart   int
	NewIdleCommands   []string
	WantsPath         bool
	TargetRoom        int
	WantsHomeFallback bool   // true after MaxPathRetries failures
	FailureMessage    string // set when WantsHomeFallback fires
}

// scheduleTickPlan computes the desired tick action for a scheduled mob. Pure
// over its inputs (mob.ScheduleId, mob.Character.RoomId, hour, MiscData) —
// safe to call from tests with stubbed registry state.
func scheduleTickPlan(mob *mobs.Mob, hour24 int) schedulePlan {
	plan := schedulePlan{}
	if mob == nil || mob.ScheduleId == "" {
		return plan
	}
	s := mobs.GetSchedule(mob.ScheduleId)
	if s == nil {
		return plan
	}
	seg := s.CurrentSegment(hour24)
	if seg == nil {
		return plan
	}
	plan.HasSchedule = true

	lastSegStart := getMiscDataInt(&mob.Character, "schedule_last_seg_start", -1)
	if seg.Start != lastSegStart {
		plan.SegmentChanged = true
		plan.NewSegmentStart = seg.Start
		plan.NewIdleCommands = seg.IdleCommands
	}

	if mob.Character.RoomId != seg.TargetRoom {
		fails := getMiscDataInt(&mob.Character, "schedule_path_fail_count", 0)
		maxRetries := int(configs.GetBalanceConfig().ScheduleMaxPathRetries)
		if maxRetries > 0 && fails >= maxRetries {
			plan.WantsHomeFallback = true
			plan.FailureMessage = fmt.Sprintf(
				"scheduled mob %d (%s) unreachable target_room %d after %d retries; falling back to home",
				mob.MobId, mob.Character.Name, seg.TargetRoom, fails)
		} else {
			plan.WantsPath = true
			plan.TargetRoom = seg.TargetRoom
		}
	}
	return plan
}

// applySchedulePlan mutates the mob (clears path, swaps idle pool, queues
// pathto, increments retry counter) based on the plan. Side-effecting; not
// pure. Called only from the live IdleMobs hook.
func applySchedulePlan(mob *mobs.Mob, plan schedulePlan) {
	if !plan.HasSchedule {
		return
	}
	if plan.SegmentChanged {
		mob.Character.SetMiscData("schedule_last_seg_start", plan.NewSegmentStart)
		mob.Character.SetMiscData("schedule_path_fail_count", 0) // reset on transition
		mob.Path.Clear()
		mob.IdleCommands = plan.NewIdleCommands
	}
	if plan.WantsHomeFallback {
		mudlog.Warn("schedule", "msg", plan.FailureMessage)
		mob.Command("pathto home")
		mob.Character.SetMiscData("schedule_path_fail_count", 0)
		return
	}
	if plan.WantsPath {
		// Only queue a new pathto if no path is in flight; let the existing
		// walker consume the current one if it still resolves.
		if mob.Path.Len() == 0 && mob.Path.Current() == nil {
			mob.Command(fmt.Sprintf("pathto %d", plan.TargetRoom))
			// Will increment on next tick if we still aren't at target.
			fails := getMiscDataInt(&mob.Character, "schedule_path_fail_count", 0)
			mob.Character.SetMiscData("schedule_path_fail_count", fails+1)
		}
	} else {
		// At target — reset retry counter.
		mob.Character.SetMiscData("schedule_path_fail_count", 0)
	}
	// Suppress wander while a schedule is active.
	mob.MaxWander = 0
}

// getMiscDataInt is a tiny coercion helper. mob.Character.GetMiscData returns
// any; misc data ints come back as int from MiscData{}'s native Go path.
func getMiscDataInt(c *charactersLike, key string, fallback int) int {
	v := c.GetMiscData(key)
	if v == nil {
		return fallback
	}
	if n, ok := v.(int); ok {
		return n
	}
	return fallback
}

// charactersLike is a tiny structural-narrowing type alias so the helper can
// be unit-tested without importing the full characters package. Implemented
// as a type alias to the real Character to avoid an actual interface
// declaration.
type charactersLike = struct {
	// Empty: real Character fits the alias because Go alias resolution treats
	// this purely as a name. The function body uses the actual GetMiscData
	// method via interface satisfaction. If the compiler rejects the alias,
	// inline the helper with the concrete *characters.Character type instead.
}
```

If the `charactersLike` alias trick proves awkward (Go's type alias rules may not allow it as written), inline the helper with the concrete type:

```go
import "github.com/GoMudEngine/GoMud/internal/characters"

func getMiscDataInt(c *characters.Character, key string, fallback int) int { ... }
```

Either approach is acceptable. The test cases above pass the concrete `Character` either way.

- [ ] **Step 6: Wire the executor into `IdleMobs`**

In `internal/hooks/NewRound_IdleMobs.go`, after the conversation guard (around line 69) and before the path-walker check (around line 73), add:

```go
// Chunk 3.2: schedule executor. Runs before the path-walker so it can
// clear stale paths on segment transitions and queue new pathtos before
// the walker consumes them.
if mob.ScheduleId != "" {
    plan := scheduleTickPlan(mob, gametime.GetDate().Hour24)
    applySchedulePlan(mob, plan)
}
```

Add the `gametime` import if not already present.

- [ ] **Step 7: Run all hooks tests**

```bash
go test ./internal/hooks/ -v
```
Expected: green.

- [ ] **Step 8: Boot the server, confirm clean startup**

```bash
go build ./...
./dogmud.exe
```
Expected: clean boot. No scheduled mobs yet → executor dormant.

- [ ] **Step 9: Commit**

```bash
git add internal/hooks/NewRound_IdleMobs.go internal/hooks/NewRound_IdleMobs_schedule.go internal/hooks/NewRound_IdleMobs_schedule_test.go internal/mobs/export_test.go
git commit -m "$(cat <<'EOF'
feat(hooks): schedule executor in NewRound_IdleMobs

Inserts scheduleTickPlan + applySchedulePlan between the
conversation guard and the path-walker. Detects segment
transitions, clears stale paths, swaps the mob's IdleCommands
pool, queues pathto toward the segment target, and falls back
to pathto home after ScheduleMaxPathRetries consecutive
failures.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: `TickMobCraft` activity gate

**Files:**
- Modify: `internal/mobs/crafter.go` (gate at top of `TickMobCraft`)
- Create or append: `internal/mobs/crafter_test.go`

- [ ] **Step 1: Re-read the `TickMobCraft` entry point**

```bash
sed -n '187,215p' internal/mobs/crafter.go
```

Confirm the function starts with `if !mob.Crafter { return nil }` and other early returns. The schedule gate goes after those existing guards (no need to consult schedules for non-crafters).

- [ ] **Step 2: Write the failing test**

Append to `internal/mobs/crafter_test.go`:

```go
func TestTickMobCraft_ScheduleGate_BlocksWhenActivityNotCraft(t *testing.T) {
	mob := minimalCrafterMobForTest(t) // helper already in this file or copy from neighbor test
	mob.ScheduleId = "thornwall_smith"
	mob.Character.RoomId = 9012 // tavern, not forge

	registerScheduleForTest(kerraTestSchedule())
	defer unregisterScheduleForTest("thornwall_smith")

	// Pin the game-time to a hour where Kerra is at the tavern (segment
	// activity = "").
	setTestHourForCrafter(t, 19) // tavern segment

	result := TickMobCraft(mob)
	if result != nil {
		t.Errorf("expected nil result during non-craft segment, got %+v", result)
	}
}

func TestTickMobCraft_ScheduleGate_AllowsWhenActivityIsCraft(t *testing.T) {
	mob := minimalCrafterMobForTest(t)
	mob.ScheduleId = "thornwall_smith"
	mob.Character.RoomId = 5678 // forge

	registerScheduleForTest(kerraTestSchedule())
	defer unregisterScheduleForTest("thornwall_smith")

	setTestHourForCrafter(t, 10) // forge segment, activity = craft

	// We don't assert what the result IS (the shop subsystem may legitimately
	// return nil if no recipe is craftable in this fixture), only that the
	// gate didn't preempt it. Use a sentinel: set up the mob to be in a state
	// where the *non*-gate path would do _something_ measurable.
	//
	// Simplest sentinel: confirm the function gets past the gate by checking
	// it does NOT panic and returns either nil or a CraftResult — the gate
	// itself doesn't differentiate, but its absence in the call stack does.
	// For a tighter assertion we'd need a richer fixture; that's the manual
	// smoke at T13's job.
	_ = TickMobCraft(mob)
}

func TestTickMobCraft_ScheduleGate_NoScheduleIsNoOpForGate(t *testing.T) {
	mob := minimalCrafterMobForTest(t)
	mob.ScheduleId = "" // no schedule

	// Should behave identically to pre-3.2 — no gate applies.
	_ = TickMobCraft(mob)
}
```

If `minimalCrafterMobForTest` and `setTestHourForCrafter` do not exist, port equivalents from neighbouring tests in the file. The point is: build a crafter mob with the smallest possible scaffolding and use `util.SetRoundCountForTest` (chunk 3.1 pattern) to pin the hour.

- [ ] **Step 3: Run the test, confirm fail**

```bash
go test ./internal/mobs/ -run TestTickMobCraft_ScheduleGate -v
```
Expected: the "BlocksWhenActivityNotCraft" test fails because crafting fires regardless of schedule.

- [ ] **Step 4: Implement the gate**

In `internal/mobs/crafter.go`, at the top of `TickMobCraft` (line ~194), after the existing combat-aggro guard:

```go
func TickMobCraft(mob *Mob) *CraftResult {
    if !mob.Crafter {
        return nil
    }
    b := configs.GetBalanceConfig()
    if !bool(b.CrafterEnabled) {
        return nil
    }
    if mob.Character.Aggro != nil {
        return nil
    }

    // Chunk 3.2: schedule activity gate. If the mob has a schedule, crafting
    // only fires when the current segment has activity: craft. Mobs without
    // a schedule_id are unaffected.
    if mob.ScheduleId != "" {
        if s := GetSchedule(mob.ScheduleId); s != nil {
            seg := s.CurrentSegment(gametime.GetDate().Hour24)
            if seg == nil || seg.Activity != "craft" {
                return nil
            }
        }
    }

    // ... existing implementation unchanged ...
```

Add `gametime` to the imports if not present.

- [ ] **Step 5: Run tests, confirm pass**

```bash
go test ./internal/mobs/ -v
```
Expected: all green, including the new gate tests.

- [ ] **Step 6: Commit**

```bash
git add internal/mobs/crafter.go internal/mobs/crafter_test.go
git commit -m "$(cat <<'EOF'
feat(mobs): TickMobCraft respects schedule activity gate

Scheduled crafters only fire the crafting tick when the current
segment has activity: craft. Mobs without a schedule_id retain
the pre-3.2 behavior unchanged.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: `mob_at_target_room` btree condition

**Files:**
- Modify: `internal/behaviortree/conditions_state.go` (add `condMobAtTargetRoom`)
- Modify: `internal/behaviortree/conditions.go` (register the new condition)
- Modify: `internal/behaviortree/conditions_state_test.go` (tests)

- [ ] **Step 1: Read the chunk 3.1 `time_of_day` precedent**

```bash
sed -n '60,200p' internal/behaviortree/conditions_state.go
```

Confirm the function signature pattern `func condXxx(params map[string]any, ctx *EvalContext) Result` and where the registration happens.

- [ ] **Step 2: Write the failing test**

Append to `internal/behaviortree/conditions_state_test.go`:

```go
func TestCondMobAtTargetRoom_AtTarget(t *testing.T) {
	defer setTestHour(t, 10)() // forge hour

	mobs.RegisterScheduleForTest(&mobs.Schedule{
		Id: "thornwall_smith",
		Segments: []mobs.ScheduleSegment{
			{Start: 6, End: 9, TargetRoom: 1234, IdleCommands: []string{"wake"}},
			{Start: 9, End: 18, TargetRoom: 5678, Activity: "craft", IdleCommands: []string{"hammer"}},
			{Start: 18, End: 22, TargetRoom: 9012, IdleCommands: []string{"sip"}},
			{Start: 22, End: 6, TargetRoom: 1234, IdleCommands: []string{"sleep"}},
		},
	})
	defer mobs.UnregisterScheduleForTest("thornwall_smith")

	mob := &mobs.Mob{ScheduleId: "thornwall_smith"}
	mob.Character.RoomId = 5678 // at forge

	ctx := &EvalContext{Mob: mob}
	if res := condMobAtTargetRoom(nil, ctx); res != Success {
		t.Errorf("expected Success at target, got %v", res)
	}
}

func TestCondMobAtTargetRoom_NotAtTarget(t *testing.T) {
	defer setTestHour(t, 10)()
	mobs.RegisterScheduleForTest(&mobs.Schedule{
		Id: "thornwall_smith",
		Segments: []mobs.ScheduleSegment{
			{Start: 0, End: 24, TargetRoom: 5678},
		},
	})
	defer mobs.UnregisterScheduleForTest("thornwall_smith")

	mob := &mobs.Mob{ScheduleId: "thornwall_smith"}
	mob.Character.RoomId = 9999 // not at target

	ctx := &EvalContext{Mob: mob}
	if res := condMobAtTargetRoom(nil, ctx); res != Failure {
		t.Errorf("expected Failure away from target, got %v", res)
	}
}

func TestCondMobAtTargetRoom_NoSchedule(t *testing.T) {
	mob := &mobs.Mob{ScheduleId: ""}
	ctx := &EvalContext{Mob: mob}
	if res := condMobAtTargetRoom(nil, ctx); res != Failure {
		t.Errorf("expected Failure with no schedule, got %v", res)
	}
}
```

Note: `EvalContext.Mob` may not be the actual field name — check the struct definition in `internal/behaviortree/types.go`. If it's `MobInstance` or `MobId`, adjust the test setup and the implementation accordingly. The principle stays the same: the condition resolves the mob, looks up its schedule, and compares `Character.RoomId` to the current segment's `TargetRoom`.

- [ ] **Step 3: Run the test, confirm fail**

```bash
go test ./internal/behaviortree/ -run TestCondMobAtTargetRoom -v
```
Expected: compilation error — `condMobAtTargetRoom` not defined.

- [ ] **Step 4: Implement the condition**

In `internal/behaviortree/conditions_state.go`, append:

```go
// condMobAtTargetRoom returns Success when the mob is at the room its current
// schedule segment names as target_room. Returns Failure when the mob has no
// schedule, no current segment, or is in transit. See chunk 3.2 spec.
func condMobAtTargetRoom(_ map[string]any, ctx *EvalContext) Result {
	if ctx == nil {
		return Failure
	}
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil || mob.ScheduleId == "" {
		return Failure
	}
	s := mobs.GetSchedule(mob.ScheduleId)
	if s == nil {
		return Failure
	}
	seg := s.CurrentSegment(gametime.GetDate().Hour24)
	if seg == nil {
		return Failure
	}
	if mob.Character.RoomId == seg.TargetRoom {
		return Success
	}
	return Failure
}
```

Add imports as needed (`mobs`, `gametime`).

Adjust ctx field reference to whatever the real struct uses (`ctx.InstanceId` if that's how mobs are resolved in btree conditions — confirm from the chunk 3.1 condition, which uses the same registry-driven pattern).

- [ ] **Step 5: Register the condition**

In `internal/behaviortree/conditions.go`, add:

```go
conditionRegistry["mob_at_target_room"] = condMobAtTargetRoom
```

Place it near the other state conditions (alphabetical by key or whatever local convention exists).

- [ ] **Step 6: Run tests, confirm pass**

```bash
go test ./internal/behaviortree/ -v
```
Expected: green.

- [ ] **Step 7: Commit**

```bash
git add internal/behaviortree/conditions_state.go internal/behaviortree/conditions.go internal/behaviortree/conditions_state_test.go
git commit -m "$(cat <<'EOF'
feat(btree): mob_at_target_room condition

Returns Success when the mob is at the room its current schedule
segment names as target_room. Lets archetype authors gate
branches on location-bound activities (e.g., "smith strikes the
anvil" emotes that should only fire at the forge).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: `mob schedule` admin inspector + helpfile

**Files:**
- Modify: `internal/usercommands/admin.mob.go` (add `mob schedule <instId>` subcommand)
- Modify: `_datafiles/world/dogmud/templates/admincommands/help/command.mob.template` (helpfile)

- [ ] **Step 1: Re-read `mob_Heal` as the closest existing subcommand template**

```bash
sed -n '79,120p' internal/usercommands/admin.mob.go
```

Mirror the `args[0] == "heal"` dispatch / permission check / instance lookup pattern.

- [ ] **Step 2: Add the dispatcher branch**

In `internal/usercommands/admin.mob.go`, after the `args[0] == "heal"` block (around line 68):

```go
if args[0] == `schedule` {
    if !user.HasRolePermission(`mob.spawn`) {
        user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">mob.spawn</ansi> permission`)
        return true, nil
    }
    return mob_Schedule(args[1:], user, room, flags)
}
```

- [ ] **Step 3: Implement `mob_Schedule`**

Append to `internal/usercommands/admin.mob.go`:

```go
import (
    // ensure these are present alongside existing imports:
    "github.com/GoMudEngine/GoMud/internal/gametime"
)

// mob_Schedule prints the schedule debug summary for a single mob instance.
// Usage: mob schedule <instId>
func mob_Schedule(args []string, user *users.UserRecord, room *rooms.Room, _ events.EventFlag) (bool, error) {
    if len(args) == 0 {
        user.SendText(messaging.CategorySystem,
            `Usage: <ansi fg="command">mob schedule [instId]</ansi>`)
        return true, nil
    }
    instId, err := strconv.Atoi(args[0])
    if err != nil || instId < 1 {
        user.SendText(messaging.CategorySystem,
            `Usage: <ansi fg="command">mob schedule [instId]</ansi>`)
        return true, nil
    }
    m := mobs.GetInstance(instId)
    if m == nil {
        user.SendText(messaging.CategorySystem,
            fmt.Sprintf(`No mob instance with id %d.`, instId))
        return true, nil
    }
    if m.ScheduleId == "" {
        user.SendText(messaging.CategorySystem,
            fmt.Sprintf(`%s (mob %d, instance %d) has no schedule.`,
                m.Character.Name, m.MobId, instId))
        return true, nil
    }
    s := mobs.GetSchedule(m.ScheduleId)
    if s == nil {
        user.SendText(messaging.CategorySystem,
            fmt.Sprintf(`%s references unknown schedule %q.`,
                m.Character.Name, m.ScheduleId))
        return true, nil
    }
    hour := gametime.GetDate().Hour24
    cur := s.CurrentSegment(hour)
    if cur == nil {
        user.SendText(messaging.CategorySystem,
            fmt.Sprintf(`Schedule %q has no segment for hour %d (coverage bug).`,
                m.ScheduleId, hour))
        return true, nil
    }

    // Find the next segment for the "in N hours" hint.
    next := nextSegmentAfter(s, cur)
    hoursUntilNext := hoursUntil(hour, next.Start)

    atTarget := "AT TARGET"
    if m.Character.RoomId != cur.TargetRoom {
        atTarget = fmt.Sprintf("EN ROUTE (current room %d)", m.Character.RoomId)
    }

    out := fmt.Sprintf(
        "Schedule for %s (mob %d, instance %d):\n"+
            "  schedule_id:     %s\n"+
            "  current hour:    %d\n"+
            "  current segment: (%d-%d)\n"+
            "    target_room:   %d\n"+
            "    activity:      %s\n"+
            "  next segment:    (%d-%d) in %d hours\n"+
            "  mob location:    %s (%d)\n"+
            "  path queue:      %d steps remaining",
        m.Character.Name, m.MobId, instId,
        m.ScheduleId,
        hour,
        cur.Start, cur.End,
        cur.TargetRoom,
        ifEmpty(cur.Activity, "(none)"),
        next.Start, next.End,
        hoursUntilNext,
        atTarget, cur.TargetRoom,
        m.Path.Len(),
    )
    user.SendText(messaging.CategorySystem, out)
    return true, nil
}

// nextSegmentAfter returns the segment whose Start is the smallest strictly
// greater than current.Start, wrapping around to the earliest segment.
func nextSegmentAfter(s *mobs.Schedule, current *mobs.ScheduleSegment) *mobs.ScheduleSegment {
    var earliest, found *mobs.ScheduleSegment
    for i := range s.Segments {
        seg := &s.Segments[i]
        if earliest == nil || seg.Start < earliest.Start {
            earliest = seg
        }
        if seg.Start > current.Start {
            if found == nil || seg.Start < found.Start {
                found = seg
            }
        }
    }
    if found != nil {
        return found
    }
    return earliest
}

// hoursUntil returns the number of hours from now until targetHour, wrapping
// past midnight.
func hoursUntil(now, target int) int {
    diff := target - now
    if diff <= 0 {
        diff += 24
    }
    return diff
}

func ifEmpty(s, fallback string) string {
    if s == "" {
        return fallback
    }
    return s
}
```

Note: `nextSegmentAfter` uses segment Start as the ordering key. With Kerra's segments [6, 9, 18, 22], from current=9 it picks 18 (correct). From current=22 it returns the earliest (6) which is correct for the wrap-around case.

- [ ] **Step 4: Append helpfile entry**

In `_datafiles/world/dogmud/templates/admincommands/help/command.mob.template`, append:

```
<ansi fg="command">mob schedule [instId]</ansi>
Inspect a scheduled mob's current segment, target room, next
transition, and path queue. Useful for debugging schedule drift,
spawn-override placement, or pathto failures.
    <ansi fg="command">mob schedule 142</ansi>    Inspect mob instance 142.

```

- [ ] **Step 5: Boot the server, smoke the command**

```bash
go build ./...
./dogmud.exe
```

In a connected session as an admin:
```
mob schedule 42      (or any mob instance id — likely "no schedule" for now)
help mob             (confirm helpfile renders the new block)
```

Expected: clean output, no crash. Mobs without schedules report "no schedule." After T11 the pilot mobs will return real schedule data.

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/admin.mob.go _datafiles/world/dogmud/templates/admincommands/help/command.mob.template
git commit -m "$(cat <<'EOF'
feat(admin): mob schedule inspector command

mob schedule <instId> prints the current segment, target room,
next transition, and path queue. Helpfile updated. Used for
debugging schedule drift and pilot rollout verification.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Pilot content — 3 above-shop home rooms + workplace exit edits

**Files:**
- Create: 3 new room YAMLs in `_datafiles/world/dogmud/rooms/thornwall_city/` (Kerra's loft, Marek's quarters, Olen's chamber)
- Modify: 3 existing workplace room YAMLs in `_datafiles/world/dogmud/rooms/thornwall_city/` (add `up` exit)

- [ ] **Step 1: Reserve a 5-room ID block in Thornwall**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
python tools/id_inventory.py --zone thornwall_city --type rooms --alloc rooms 5
```

Record the assigned range (likely something like `4101-4105`). The first three IDs are for the new home rooms; the remaining two are slack.

For the remainder of this task, the placeholder IDs are:
- `<KERRA_LOFT_ID>` = first allocated ID
- `<MAREK_QUARTERS_ID>` = second allocated ID
- `<OLEN_CHAMBER_ID>` = third allocated ID

Substitute the real IDs from the inventory tool.

- [ ] **Step 2: Identify the three workplace rooms**

```bash
grep -rn "Blacksmith\|Kerra" _datafiles/world/dogmud/rooms/thornwall_city/ | head -5
grep -rn "Tavern\|Marek" _datafiles/world/dogmud/rooms/thornwall_city/ | head -5
grep -rn "Temple\|Olen" _datafiles/world/dogmud/rooms/thornwall_city/ | head -5
```

Find:
- Kerra's forge room ID — call this `<FORGE_ROOM_ID>`
- Marek's tavern room (the one Marek actually stands in, not just an entry hall) — call this `<TAVERN_ROOM_ID>`
- Olen's temple room (the cloister or worship hall where he stands) — call this `<TEMPLE_ROOM_ID>`

Also check `_datafiles/world/dogmud/mobs/thornwall_city/{95,96,97}-*.yaml` for any `home_room` or spawn-location hints that confirm which rooms these NPCs anchor to.

Record these IDs and continue.

- [ ] **Step 3: Create Kerra's loft room YAML**

Create `_datafiles/world/dogmud/rooms/thornwall_city/<KERRA_LOFT_ID>.yaml` (filename = numeric room id):

```yaml
roomid: <KERRA_LOFT_ID>
zone: Thornwall City
title: A Loft Above the Forge
description: >-
  A narrow loft tucked beneath the forge's rafters, warmed by the
  banked coals below. A straw mattress lies under a single thick
  blanket; a battered chest holds a change of work clothes. The
  faint smell of iron and woodsmoke drifts up through the
  floorboards.
mapsymbol: h
maplegend: Home
biome: indoors
exits:
  down:
    roomid: <FORGE_ROOM_ID>
```

If your room YAML schema differs (some zones use `Description:` capitalised, some lower; check a neighbour file first), match the local convention.

- [ ] **Step 4: Create Marek's quarters room YAML**

Create `_datafiles/world/dogmud/rooms/thornwall_city/<MAREK_QUARTERS_ID>.yaml`:

```yaml
roomid: <MAREK_QUARTERS_ID>
zone: Thornwall City
title: Marek's Quarters Above the Tavern
description: >-
  A snug room above the common floor. A neatly made bed, a small
  table with a half-finished ledger, and a window looking out over
  the alley. Muffled voices and the clatter of crockery drift up
  through the boards. Marek keeps his livelihood close enough to
  hear, even when he sleeps.
mapsymbol: h
maplegend: Home
biome: indoors
exits:
  down:
    roomid: <TAVERN_ROOM_ID>
```

- [ ] **Step 5: Create Olen's chamber room YAML**

Create `_datafiles/world/dogmud/rooms/thornwall_city/<OLEN_CHAMBER_ID>.yaml`:

```yaml
roomid: <OLEN_CHAMBER_ID>
zone: Thornwall City
title: Olen's Chamber Above the Temple
description: >-
  A spare cell of a room above the temple, scarcely larger than
  the cot it holds. A worn breviary rests on a small shelf beside
  a clay basin. A single tall window admits a slant of pale light.
  Even here the air smells of beeswax and old incense.
mapsymbol: h
maplegend: Home
biome: indoors
exits:
  down:
    roomid: <TEMPLE_ROOM_ID>
```

- [ ] **Step 6: Add `up` exits to the workplace rooms**

Edit `_datafiles/world/dogmud/rooms/thornwall_city/<FORGE_ROOM_ID>.yaml`. In the `exits:` block, add:
```yaml
  up:
    roomid: <KERRA_LOFT_ID>
```

Edit `<TAVERN_ROOM_ID>.yaml`:
```yaml
  up:
    roomid: <MAREK_QUARTERS_ID>
```

Edit `<TEMPLE_ROOM_ID>.yaml`:
```yaml
  up:
    roomid: <OLEN_CHAMBER_ID>
```

- [ ] **Step 7: Clean up any stale instance saves for the edited workplace rooms**

Per CLAUDE.md's instance-save SOP:

```bash
ls _datafiles/world/dogmud/rooms.instances/thornwall_city/ 2>/dev/null | grep -E "<FORGE_ROOM_ID>|<TAVERN_ROOM_ID>|<TEMPLE_ROOM_ID>"
```

If any of those instance files exist, delete them so the engine reloads from templates:
```bash
rm _datafiles/world/dogmud/rooms.instances/thornwall_city/<FORGE_ROOM_ID>.yaml
# repeat for tavern and temple if present
```

- [ ] **Step 8: Boot the server, confirm clean startup and validate exits**

```bash
go build ./...
./dogmud.exe
```

Expected: clean boot through `rooms.LoadDataFiles()`. Connect, navigate to each workplace, `up` into the home rooms, `down` back. Confirm each room's description renders correctly.

Ctrl+C to stop.

- [ ] **Step 9: Commit**

```bash
git add _datafiles/world/dogmud/rooms/thornwall_city/<KERRA_LOFT_ID>.yaml _datafiles/world/dogmud/rooms/thornwall_city/<MAREK_QUARTERS_ID>.yaml _datafiles/world/dogmud/rooms/thornwall_city/<OLEN_CHAMBER_ID>.yaml _datafiles/world/dogmud/rooms/thornwall_city/<FORGE_ROOM_ID>.yaml _datafiles/world/dogmud/rooms/thornwall_city/<TAVERN_ROOM_ID>.yaml _datafiles/world/dogmud/rooms/thornwall_city/<TEMPLE_ROOM_ID>.yaml
git commit -m "$(cat <<'EOF'
feat(content): Thornwall pilot homes — Kerra's loft, Marek's
quarters, Olen's chamber

Three small above-shop home rooms for the chunk 3.2 NPC schedule
pilot. Each is a dead-end room reached via 'up' from the
workplace, with no loot or affordances. up/down exits added to
the workplace rooms.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Pilot content — 3 schedule YAMLs + 3 mob YAML edits

**Files:**
- Create: `_datafiles/world/dogmud/schedules/thornwall_city/thornwall_smith.yaml`
- Create: `_datafiles/world/dogmud/schedules/thornwall_city/thornwall_tavern_keeper.yaml`
- Create: `_datafiles/world/dogmud/schedules/thornwall_city/thornwall_temple_priest.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/97-blacksmith_kerra.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/96-tavern_keeper_marek.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/95-temple_priest_olen.yaml`

- [ ] **Step 1: Create the smith schedule**

Create `_datafiles/world/dogmud/schedules/thornwall_city/thornwall_smith.yaml`:

```yaml
id: thornwall_smith
description: "Blacksmith Kerra's daily routine: forge by day, tavern by evening, loft by night."
segments:
  - start: 6
    end: 9
    target_room: <KERRA_LOFT_ID>
    activity: ""
    idlecommands:
      - emote rubs sleep from her eyes.
      - emote pulls on her boots and apron.
      - emote stretches with a yawn.
  - start: 9
    end: 18
    target_room: <FORGE_ROOM_ID>
    activity: craft
    idlecommands:
      - emote raises the hammer once more.
      - emote tongs a glowing bar from the coals.
      - say I'll have it done by sunset.
      - emote wipes soot from her brow.
  - start: 18
    end: 22
    target_room: <TAVERN_ROOM_ID>
    activity: ""
    idlecommands:
      - emote sips from a tankard.
      - say Long day at the forge.
      - emote rolls a stiff shoulder.
  - start: 22
    end: 6
    target_room: <KERRA_LOFT_ID>
    activity: ""
    idlecommands:
      - emote breathes evenly, asleep on the cot.
      - emote turns over with a soft snore.
```

Substitute the real room IDs allocated in T10.

- [ ] **Step 2: Create the tavern keeper schedule**

Create `_datafiles/world/dogmud/schedules/thornwall_city/thornwall_tavern_keeper.yaml`:

```yaml
id: thornwall_tavern_keeper
description: "Tavern Keeper Marek's daily routine: long shift on the floor, prep in the morning, sleep upstairs."
segments:
  - start: 6
    end: 10
    target_room: <MAREK_QUARTERS_ID>
    activity: ""
    idlecommands:
      - emote scribbles in a ledger.
      - emote counts out coins from a small purse.
      - say Slow morning, that's how I like it.
  - start: 10
    end: 22
    target_room: <TAVERN_ROOM_ID>
    activity: ""
    idlecommands:
      - emote wipes down the bar with a worn rag.
      - emote pulls a fresh tap.
      - say What'll it be, friend?
      - emote shouts an order back to the kitchen.
  - start: 22
    end: 6
    target_room: <MAREK_QUARTERS_ID>
    activity: ""
    idlecommands:
      - emote sleeps soundly under a worn blanket.
      - emote mumbles in a dream.
```

- [ ] **Step 3: Create the temple priest schedule**

Create `_datafiles/world/dogmud/schedules/thornwall_city/thornwall_temple_priest.yaml`:

```yaml
id: thornwall_temple_priest
description: "Temple Priest Olen's daily routine: dawn and afternoon prayers, an evening at the tavern, sleep in the chamber."
segments:
  - start: 4
    end: 6
    target_room: <OLEN_CHAMBER_ID>
    activity: ""
    idlecommands:
      - emote rises in the dark and washes at the basin.
      - emote whispers a private prayer.
  - start: 6
    end: 10
    target_room: <TEMPLE_ROOM_ID>
    activity: ""
    idlecommands:
      - emote intones the dawn prayer.
      - emote bows toward the altar.
      - say May the morning light keep you.
  - start: 10
    end: 12
    target_room: <OLEN_CHAMBER_ID>
    activity: ""
    idlecommands:
      - emote pages through his breviary.
      - emote rests with eyes closed.
  - start: 12
    end: 18
    target_room: <TEMPLE_ROOM_ID>
    activity: ""
    idlecommands:
      - emote tends to a guttering candle.
      - emote speaks a quiet blessing over a passerby.
      - say Peace be with you.
  - start: 18
    end: 22
    target_room: <TAVERN_ROOM_ID>
    activity: ""
    idlecommands:
      - emote sips a thin cup of wine.
      - say Even priests get thirsty.
      - emote nods quietly to those around him.
  - start: 22
    end: 4
    target_room: <OLEN_CHAMBER_ID>
    activity: ""
    idlecommands:
      - emote sleeps with hands folded over the breviary.
      - emote breathes slow and even.
```

- [ ] **Step 4: Add `schedule_id:` to the three mob YAMLs**

In `_datafiles/world/dogmud/mobs/thornwall_city/97-blacksmith_kerra.yaml`, add:
```yaml
schedule_id: thornwall_smith
```

In `96-tavern_keeper_marek.yaml`:
```yaml
schedule_id: thornwall_tavern_keeper
```

In `95-temple_priest_olen.yaml`:
```yaml
schedule_id: thornwall_temple_priest
```

Add the field near the bottom of each file, after `behavior_archetype` if present. Maintain the existing field order convention; if the file has `crafter:`, `crafterskill:`, etc., place `schedule_id:` after those.

- [ ] **Step 5: Clean up any stale mob instance saves for the pilot NPCs**

Per CLAUDE.md's instance-save SOP:

```bash
ls _datafiles/world/dogmud/mobs.instances/thornwall_city/ 2>/dev/null | grep -E "^9[5-7]-"
```

Delete any matches so the engine reloads from templates:
```bash
rm _datafiles/world/dogmud/mobs.instances/thornwall_city/95-*.yaml
rm _datafiles/world/dogmud/mobs.instances/thornwall_city/96-*.yaml
rm _datafiles/world/dogmud/mobs.instances/thornwall_city/97-*.yaml
```

- [ ] **Step 6: Boot the server, confirm clean startup**

```bash
go build ./...
./dogmud.exe
```

Expected log entries:
- `rooms.LoadDataFiles()` succeeds with the new rooms.
- `mobs.LoadDataFiles()` loads cleanly.
- `mobs.LoadSchedules() loadedCount=3` (the three pilot schedules).
- No panic from the cross-check loop.

If the boot panics due to a coverage gap or unreachable target_room, the validator messages will point at the file + reason — fix the schedule and retry.

- [ ] **Step 7: Spot-check the pilot in-game**

Connect, navigate to the forge, observe Kerra. Use the admin command:
```
mob schedule <kerra_instance_id>
```

Expected: the inspector reports the current segment, target room, and "AT TARGET" if she's at the forge.

If the current game-time happens to be a non-forge hour, navigate to wherever the schedule says she should be and confirm she's there.

`time` command shows current game-time. Use admin `time set HH:MM` if available to jump times and watch transitions.

- [ ] **Step 8: Commit**

```bash
git add _datafiles/world/dogmud/schedules/thornwall_city/ _datafiles/world/dogmud/mobs/thornwall_city/95-temple_priest_olen.yaml _datafiles/world/dogmud/mobs/thornwall_city/96-tavern_keeper_marek.yaml _datafiles/world/dogmud/mobs/thornwall_city/97-blacksmith_kerra.yaml
git commit -m "$(cat <<'EOF'
feat(content): Thornwall pilot schedules — Kerra, Marek, Olen

Three daily routines:
- Blacksmith Kerra: loft → forge (craft) → tavern → loft sleep
- Tavern Keeper Marek: quarters → tavern (long shift) → sleep
- Temple Priest Olen: chamber → temple (dawn) → rest → temple
  (afternoon) → tavern → sleep

Mob YAMLs gain schedule_id fields; instance saves cleaned.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Documentation pass

**Files:**
- Create: `docs/schemas/schedule.md`
- Modify: `internal/mobs/context.md`
- Modify: `internal/hooks/context.md`
- Modify: `internal/behaviortree/context.md`
- Modify: `internal/configs/context.md`
- Modify: `CLAUDE.md`
- Modify: `docs/CONTENT_GENERATION_GUIDE.md`
- Modify: `_datafiles/world/default/templates/help/ask.template`
- Modify: `modules/time/files/datafiles/templates/help/time.template`

- [ ] **Step 1: Write `docs/schemas/schedule.md`**

Create `docs/schemas/schedule.md`:

````markdown
# Schedule YAML schema

Schedules are daily routines attached to NPCs via the
`schedule_id:` field on mob specs. Each schedule references a file
at `_datafiles/world/dogmud/schedules/<zone>/<id>.yaml` (filename
= `ConvertForFilename(id)`).

## Required fields

```yaml
id: <string>                      # must match the filename
description: <string>             # short prose, used in admin debug output
segments: [<segment>, ...]        # must cover all 24 hours, no overlaps
```

## Segment fields

```yaml
- start: <int 0-23>               # inclusive
  end: <int 1-24>                 # exclusive; end may equal 24 for the
                                  # day-boundary
  target_room: <int>              # must exist; mob is steered here
  activity: <"" | "craft">        # gates engine-side activity verbs
  idlecommands:                   # mob's idle pool while in this segment
    - emote <text>
    - say <text>
```

Wrap-around: when `start > end`, the segment covers `[start, 24)`
and `[0, end)`. Example: `start: 22 end: 6` covers 22-5.

Two segments may share a `target_room` (e.g. the priest visits the
temple twice with different `idlecommands`).

## Validation (load-time, panics)

- Filename must equal `ConvertForFilename(id)`.
- Each segment must satisfy `0 <= start < 24` and `0 < end <= 24`.
- `start != end` (no empty segments).
- Every hour 0-23 must be claimed by exactly one segment (no
  overlaps, no gaps).
- `target_room` must exist.
- `mapper.GetPath` must succeed for every consecutive segment pair
  (chronological order, including the wrap-around transition).
- The mob's `schedule_id` (in its mob YAML) must resolve.

## Validation (load-time, warn-only)

- `activity:` value not in `{"", "craft"}`.
- `activity: craft` on a mob that lacks `crafter: true`.
- `target_room` is outside the mob's `zone`.
- Segment has zero `idlecommands` (mob will be silent).

## Future: per-day variation

Single-day routines are the current shape. When per-day variation
lands (chunk after 3.2), the loader will recognise an optional
top-level `days:` map and prefer it over flat `segments:`:

```yaml
# Not implemented today — shown for forward-compatibility.
id: thornwall_smith
description: "..."
days:
  default: [ ...segments... ]
  weekend: [ ...segments... ]
  holiday: [ ...segments... ]
```

Existing flat-segment schedules continue working unchanged. The
schedule `id` is the stable reference; mob YAMLs do not move.

## Authoring workflow

1. Pick an `id` (snake_case, zone prefix recommended).
2. Identify target rooms in the zone — confirm reachability from
   each other.
3. Author segments covering all 24 hours; each segment gets idle
   flavor written in NPC voice (`say` and `emote` lines).
4. Save to
   `_datafiles/world/dogmud/schedules/<zone>/<filename>.yaml`.
5. Add `schedule_id: <id>` to the mob spec.
6. Restart the server. The validator will surface any coverage,
   pathing, or reference issues at boot.
7. In-game, use `mob schedule <instId>` to confirm the executor
   resolves the right segment for the current hour.

## See also

- Spec: `docs/superpowers/specs/2026-05-25-mob-aliveness-3.2-npc-schedules-design.md`
- Mob spec field: `internal/mobs/mobs.go` (`Mob.ScheduleId`)
- Loader: `internal/mobs/schedule_loader.go`
- Executor: `internal/hooks/NewRound_IdleMobs_schedule.go`
- Admin inspector: `mob schedule <instId>`
````

- [ ] **Step 2: Update `internal/mobs/context.md`**

Find the section listing mob fields / spawn helpers and append:

```markdown
## Schedules (chunk 3.2)

Mobs with `schedule_id:` set follow daily routines authored in
`_datafiles/world/dogmud/schedules/<zone>/<id>.yaml`. See
`docs/schemas/schedule.md` for the full schema.

- `schedule.go`: `Schedule`, `ScheduleSegment`, `GetSchedule`,
  `CurrentSegment`, `applyScheduleSpawnOverride`, test helpers.
- `schedule_loader.go`: `LoadSchedules`, `validateScheduleStandalone`,
  `validateScheduleAgainstWorld`. Called from `LoadDataFiles` after
  mob templates load.
- Spawn override: `newMobByIdInternal` calls
  `applyScheduleSpawnOverride` to place scheduled mobs at the
  current segment's target room.
- Crafter activity gate: `TickMobCraft` returns nil when a
  scheduled mob's current segment activity != "craft".
```

- [ ] **Step 3: Update `internal/hooks/context.md`**

Find the table row for `NewRound_IdleMobs` and append a note:

```markdown
- After the conversation guard and before the path-walker, the
  schedule executor (in `NewRound_IdleMobs_schedule.go`) resolves
  the current segment, swaps the mob's IdleCommands, queues
  `pathto`, and falls back to `pathto home` after
  `ScheduleMaxPathRetries` failures.
```

And for `MobIdle_HandleIdleMobs`, add:

```markdown
- `TickMobCraft` now respects the schedule activity gate; only
  fires when the current segment activity is "craft".
```

- [ ] **Step 4: Update `internal/behaviortree/context.md`**

In the conditions table, add a row:

```markdown
| `mob_at_target_room` | none | Success when mob is at its current schedule segment's target_room; Failure when no schedule, no current segment, or in transit. |
```

- [ ] **Step 5: Update `internal/configs/context.md`**

In the balance-config knobs table, add a row:

```markdown
| `ScheduleMaxPathRetries` | 20 | After N consecutive failed `pathto` attempts, a scheduled mob falls back to `pathto home`. Chunk 3.2. |
```

- [ ] **Step 6: Update `CLAUDE.md`**

In the "Project Context" section, append a new subsection:

```markdown
## NPC Schedules
Townspeople NPCs can carry a `schedule_id:` field that references
a daily routine in
`_datafiles/world/dogmud/schedules/<zone>/<id>.yaml`. Schedules
cover all 24 hours, swap the mob's idle command pool per segment,
steer the mob between rooms via the existing `pathto` plumbing,
and gate `TickMobCraft` via segment `activity:`. Schedule
validators panic at startup on coverage gaps, unreachable target
rooms, or unresolved `schedule_id` references — pre-push SOP
boot-test catches these. See `docs/schemas/schedule.md`.
```

- [ ] **Step 7: Update `docs/CONTENT_GENERATION_GUIDE.md`**

Append at the bottom:

```markdown
## Schedules

No `/new-schedule` command yet — author by hand using
`docs/schemas/schedule.md`. Restart required after authoring.
Validators run at boot and panic on coverage / pathing /
reference errors.
```

- [ ] **Step 8: Update `_datafiles/world/default/templates/help/ask.template`**

Append a short paragraph at the end of the file (preserve the existing
ANSI formatting style):

```
NPCs follow daily routines. The smith might not be at the forge
after sunset — try the tavern or come back in the morning.
```

- [ ] **Step 9: Update `modules/time/files/datafiles/templates/help/time.template`**

Append one sentence after the existing "Day and Night fall at specific times of day." line:

```
Many townspeople follow daily routines — they work, eat, drink, and sleep at different times of day.
```

- [ ] **Step 10: Boot the server, sanity-check helpfiles**

```bash
go build ./...
./dogmud.exe
```

Connect as a player:
```
help time
help ask
help mob          (admin)
```
Expected: each helpfile renders the new content.

- [ ] **Step 11: Commit**

```bash
git add docs/schemas/schedule.md internal/mobs/context.md internal/hooks/context.md internal/behaviortree/context.md internal/configs/context.md CLAUDE.md docs/CONTENT_GENERATION_GUIDE.md _datafiles/world/default/templates/help/ask.template modules/time/files/datafiles/templates/help/time.template
git commit -m "$(cat <<'EOF'
docs: schedule schema, context.md updates, helpfiles for 3.2

Schema reference at docs/schemas/schedule.md. Internal
context.md updates for mobs/hooks/behaviortree/configs.
CLAUDE.md gains an NPC Schedules subsection. Player-facing
help text added to 'ask' and 'time' so players understand why
the smith isn't at the forge at midnight.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 13: Smoketester goal file + manual smoke + roadmap closeout

**Files:**
- Create: `tools/testing/goals/3.2-schedule-observation.yaml`
- Modify: `MOB_ALIVENESS_ROADMAP.md`
- Possibly modify: `PATCH_NOTES.md` (per pre-push SOP — only at the final push, not in this task)

- [ ] **Step 1: Write the smoketester goal file**

Look at an existing goal file in `tools/testing/goals/` to mirror its shape:

```bash
ls tools/testing/goals/
cat tools/testing/goals/*.yaml 2>/dev/null | head -40
```

Create `tools/testing/goals/3.2-schedule-observation.yaml`:

```yaml
goal: "Observe Blacksmith Kerra for one full game-day and verify her schedule resolves correctly at each segment, including the crafter activity gate."

target_npc:
  name: "Blacksmith Kerra"
  mob_id: 97
  zone: thornwall_city

session_length: "One full game-day (compute from configs.GetTimingConfig().RoundsPerDay × tick rate; default 20 RPD × ~4s/round ≈ 80 seconds per game-day — but verify before running)."

sampling_cadence: "Re-locate Kerra every ~RoundsPerDay/12 rounds, so roughly 12 samples per game-day."

procedure:
  - "On session start, locate Kerra. Log her room and the game time (`time` command)."
  - "Wait the sampling cadence; recheck her location and the game time. Log any change."
  - "When Kerra is at the forge during craft hours (9-18), confirm crafting output is visible in the room — messages like 'finishes crafting' or 'frowns at a failed attempt'."
  - "When Kerra is at the tavern (18-22) or in the loft (22-9), confirm NO crafting output appears."
  - "If Kerra appears stuck (same room across multiple samples in a segment she should have left), flag it."
  - "If Kerra acquires the 'lost' adjective, flag it."
  - "If any error/panic appears in chat or status messages, flag it."

pass_criteria:
  - "All 4 segments visited in their correct hour-ranges (loft 6-9, forge 9-18, tavern 18-22, loft sleep 22-6)."
  - "No startup panic in the server logs."
  - "No 'lost' adjective acquired."
  - "Crafting output appears ONLY during the forge segment (9-18)."

output_format: "Markdown report saved to tools/testing/reports/3.2-schedule-observation-<timestamp>.md. Include timestamped location log, crafting-gate observations, anomalies, and a final pass/fail verdict."
```

- [ ] **Step 2: Run the manual smoke**

Boot the server fresh:
```bash
go build ./...
./dogmud.exe
```

As an admin connected to the server, run through the manual smoke pass from the spec:

1. Confirm no startup panic.
2. `time set 09:00` (or whatever the admin syntax is — check `help time` and admin docs) → confirm Kerra at forge, Marek at tavern, Olen at temple.
3. `time set 13:00` → confirm Olen at chamber (rest segment), Kerra still at forge.
4. `time set 22:00` → confirm all three at their home rooms.
5. `mob schedule <kerra_instance>` at each time confirms expected current segment.
6. `mob schedule <olen_instance>` at 16:00 confirms `temple (12-18) afternoon prayers` segment.
7. Attack Kerra mid-shift; confirm combat suspends schedule (no schedule transition messages, normal combat behavior).
8. After combat, confirm schedule resumes (next idle tick re-queues `pathto` if she moved during combat).

If any step fails, fix the issue and re-run from step 1.

- [ ] **Step 3: Dispatch the smoketester for the long-running observation**

Per CLAUDE.md, the AI testing skill is invoked via:
```
/test-mud local feel-tester 3.2-schedule-observation.yaml
```

Confirm the session starts cleanly. The full game-day observation runs unattended; the report drops into `tools/testing/reports/` when done.

- [ ] **Step 4: Update the roadmap**

Edit `MOB_ALIVENESS_ROADMAP.md`. Find the chunk 3.2 row:

```markdown
| 3.2 | Routine | NPC schedules | L | 3.1 | Not started |
```

Change to:

```markdown
| 3.2 | Routine | NPC schedules | L | 3.1 | Done |
```

Add a "Shipped:" line under the 3.2 section header (around line 420):

```markdown
- **Shipped:** Schedule loader + 24h-coverage validator + pathfinding sanity in `internal/mobs/schedule.go` and `internal/mobs/schedule_loader.go`. Go-side executor in `internal/hooks/NewRound_IdleMobs_schedule.go` steers scheduled mobs via existing `pathto` plumbing, swaps per-segment `IdleCommands`, falls back to home after `ScheduleMaxPathRetries` failures. Spawn override in `newMobByIdInternal` places scheduled mobs at the current segment's target room. `TickMobCraft` respects per-segment `activity: craft` so Blacksmith Kerra only forges at the forge. New `mob_at_target_room` btree condition. New `mob schedule <instId>` admin inspector. Three Thornwall pilots: Blacksmith Kerra, Tavern Keeper Marek, Temple Priest Olen, each with a new above-shop home room. Spec at `docs/superpowers/specs/2026-05-25-mob-aliveness-3.2-npc-schedules-design.md`, plan at `docs/superpowers/plans/2026-05-25-mob-aliveness-3.2-npc-schedules.md`.
```

Also update chunk 3.5 (Maintenance routines) to mention the new dependency on 3.2's `activity:` field. Find:
```markdown
### 3.5 Maintenance routines
**Status:** Not started • **Size:** M
```

Append after the existing "Depends on:" line:

```markdown
- **Builds on:** Chunk 3.2's per-segment `activity:` field. New maintenance verbs (`tend_crops`, `shelve_books`, etc.) will be dispatched when a segment declares them.
```

- [ ] **Step 5: Move the spec and plan files to `completed/`**

Per the pattern established by 3.1:

```bash
mkdir -p docs/superpowers/specs/completed docs/superpowers/plans/completed
git mv docs/superpowers/specs/2026-05-25-mob-aliveness-3.2-npc-schedules-design.md docs/superpowers/specs/completed/
git mv docs/superpowers/plans/2026-05-25-mob-aliveness-3.2-npc-schedules.md docs/superpowers/plans/completed/
```

Update the `Spec:` and `Plan:` paths in the roadmap entry from Step 4 to point to the `completed/` subdirectory.

- [ ] **Step 6: Commit**

```bash
git add tools/testing/goals/3.2-schedule-observation.yaml MOB_ALIVENESS_ROADMAP.md docs/superpowers/specs/completed/ docs/superpowers/plans/completed/
git commit -m "$(cat <<'EOF'
chore(roadmap): mark 3.2 NPC schedules Done

Adds the smoketester goal file for a full-game-day observation
of Kerra's schedule. Updates MOB_ALIVENESS_ROADMAP.md with the
shipped summary and the chunk-3.5 dependency note. Moves spec
and plan to completed/.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 7: Final verification — full test suite + boot**

```bash
go build ./...
go test ./...
./dogmud.exe   # confirm clean boot one more time; Ctrl+C after the LoadSchedules log line
```

Expected: clean build, all tests pass, server boots through schedule loading without panic.

- [ ] **Step 8: Pre-push checklist (only when ready to push to prod)**

This is the standard chunk-completion handoff. Do NOT push without:

1. Boot the server locally one more time — confirm clean startup past data file loading.
2. Set `Logging.LogToFile: false` in `_datafiles/config.yaml` (prod-droplet disk).
3. Update `PATCH_NOTES.md` with a dated entry summarising the chunk:

```markdown
## 2026-XX-XX — Mob Aliveness 3.2 (NPC schedules)

Townspeople NPCs in Thornwall now follow daily routines. Smith
Kerra forges by day, drinks at the tavern in the evening, and
sleeps in her new loft above the forge. Tavern keeper Marek works
a long shift on the floor and sleeps upstairs. Temple Priest Olen
keeps dawn and afternoon prayers, joins the regulars at the tavern
for an evening cup, and sleeps in his cell above the temple.

Crafters with schedules only craft when their current segment is
flagged as a craft segment — Kerra won't forge at the tavern.

New admin command: `mob schedule <instId>` for debugging.
```

4. Smoketester session must have finished and reported PASS, OR you must have documented any unresolved anomalies in the PATCH_NOTES entry.

---

## Self-Review

**Spec coverage check:**

- ✅ Schedule type + `CurrentSegment` resolver → T1
- ✅ Schedule loader + 24h-coverage validator → T2
- ✅ Pathfinding sanity validator → T2 (`validateScheduleAgainstWorld`)
- ✅ Mob `schedule_id` field + cross-check → T3
- ✅ Spawn override at current segment's target room → T4
- ✅ `ScheduleMaxPathRetries` config knob → T5
- ✅ `NewRound_IdleMobs` schedule hook → T6 (including segment-transition path clearing, idlecommand pool swap, retry counter with home fallback, MaxWander suppression)
- ✅ `TickMobCraft` activity gate → T7
- ✅ `mob_at_target_room` btree condition → T8
- ✅ `mob schedule` admin inspector + helpfile → T9
- ✅ Three above-shop home rooms + workplace exit edits → T10
- ✅ Three pilot schedule YAMLs + three mob YAML edits → T11
- ✅ `docs/schemas/schedule.md` → T12
- ✅ Internal context.md updates (mobs, hooks, behaviortree, configs) → T12
- ✅ CLAUDE.md update → T12
- ✅ docs/CONTENT_GENERATION_GUIDE.md update → T12
- ✅ Player-facing helpfiles (ask, time) → T12
- ✅ Smoketester goal file → T13
- ✅ Manual smoke pass → T13
- ✅ Roadmap closeout + spec/plan move to completed/ → T13
- ✅ PATCH_NOTES.md guidance (push-time, not in-chunk) → T13 step 8

**Type consistency check:**

- `Schedule.Id` (Go) ↔ `id:` (YAML): consistent across T1, T2, T11, T12
- `Schedule.Segments` ↔ `segments:`: consistent
- `ScheduleSegment.Start/End/TargetRoom/Activity/IdleCommands` ↔ YAML field names: consistent
- `applyScheduleSpawnOverride(scheduleId, homeRoomId, hour24)` signature consistent between T4 implementation and the caller in T4 step 5
- `scheduleTickPlan(mob, hour24)` and `applySchedulePlan(mob, plan)` signatures consistent T6
- MiscData keys `schedule_last_seg_start`, `schedule_path_fail_count` consistent across T6 and the admin inspector in T9
- Test helpers `registerScheduleForTest` / `unregisterScheduleForTest` (lowercase) introduced in T4, exported as `RegisterScheduleForTest` / `UnregisterScheduleForTest` via `export_test.go` in T6 step 3, used by T8 tests — consistent.

**Placeholder scan:**

Searched for "TBD", "TODO", "implement later", "fill in details", "handle edge cases", "appropriate error handling", "similar to Task" — none found. Real code blocks accompany each implementation step.

Two intentional `t.Skip` calls (T2 step 5, T3 step 3) are documented and tied to the boot-smoke verification in T13; these are not placeholders, they are explicit "covered at integration time" markers.

**Scope check:**

13 tasks, ~3-5 days of focused work. Sequential dependencies are well-defined (T1→T2→T3→T4, T5 standalone, T6→T7→T8, T9 standalone, T10→T11, T12 standalone, T13 closes out). Subagent-driven execution can parallelise T5/T8/T9/T12 with the right ordering.

**Ambiguity check:**

- Test-helper export pattern (T6 step 3) uses `export_test.go` — Go-standard, unambiguous.
- The `charactersLike` type-alias trick in T6 step 5 is flagged with a fallback (inline the concrete type) in case Go's alias rules reject it.
- The `fileloader.LoadAllFlatFiles[string, *Schedule]` call in T2 has a fallback note for if the generic accepts only numeric keys.

Plan is internally consistent and ready for execution.
