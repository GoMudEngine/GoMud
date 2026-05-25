package mobs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// ScheduleWorldValidator is injected at startup (after both rooms and mapper
// are initialized) to perform world-aware validation without creating an
// import cycle. mobs ← rooms ← mobs would be a cycle.
//
// roomExists(id) must return true if the room is in the registry.
// hasPath(from, to) must return true if the mapper can route between the rooms.
// Call SetScheduleWorldValidator before LoadSchedules if world-aware checks
// are desired; omitting it skips the world-aware pass silently.
var scheduleWorldValidator struct {
	roomExists func(id int) bool
	hasPath    func(from, to int) bool
}

// SetScheduleWorldValidator wires in the room-existence and pathfinding
// checks for schedule validation. Must be called before LoadSchedules.
// Callers: internal/server or whatever bootstraps the world after rooms and
// mapper are initialized.
func SetScheduleWorldValidator(roomExists func(int) bool, hasPath func(from, to int) bool) {
	scheduleWorldValidator.roomExists = roomExists
	scheduleWorldValidator.hasPath = hasPath
}

// LoadSchedules walks _datafiles/world/dogmud/schedules/**/*.yaml, parses
// each file into a *Schedule, validates it, and registers it in the
// package-level schedules map. Duplicate ids and validation failures cause
// a panic. If the schedules directory does not exist, the function logs and
// returns without error — schedules are optional content.
func LoadSchedules() {
	start := time.Now()

	dataPath := configs.GetFilePathsConfig().DataFiles.String() + `/schedules`

	// Directory is optional — no schedules yet is fine.
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		mudlog.Info("mobs.LoadSchedules()", "loadedCount", 0,
			"note", "schedules directory does not exist — skipping",
			"Time Taken", time.Since(start))
		return
	}

	tmp := map[string]*Schedule{}

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
			return fmt.Errorf("schedule: read %s: %w", path, readErr)
		}

		var s Schedule
		if unmarshalErr := yaml.Unmarshal(data, &s); unmarshalErr != nil {
			return fmt.Errorf("schedule: unmarshal %s: %w", path, unmarshalErr)
		}

		// Filename ↔ id consistency check: base name (without extension) must
		// exactly equal ConvertForFilename(id). HasSuffix would pass a file like
		// "my_thornwall_smith.yaml" for id "thornwall_smith" — reject that.
		base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		expected := util.ConvertForFilename(s.Id)
		if base != expected {
			return fmt.Errorf(
				"schedule: filename mismatch: file %q has base name %q but expected %q (derived from id %q)",
				path, base, expected, s.Id,
			)
		}

		if valErr := validateScheduleStandalone(&s); valErr != nil {
			return fmt.Errorf("schedule %q (%s): %w", s.Id, path, valErr)
		}

		// Warn on activity/idlecommands issues (non-fatal).
		for i, seg := range s.Segments {
			if seg.Activity != "" && seg.Activity != "craft" {
				mudlog.Warn("mobs.LoadSchedules()",
					"scheduleId", s.Id,
					"segment", i,
					"activity", seg.Activity,
					"warning", "unknown activity value (expected '' or 'craft')")
			}
			if len(seg.IdleCommands) == 0 {
				mudlog.Warn("mobs.LoadSchedules()",
					"scheduleId", s.Id,
					"segment", i,
					"warning", "segment has no idlecommands")
			}
		}

		if _, dup := tmp[s.Id]; dup {
			return fmt.Errorf("schedule: duplicate id %q in %s", s.Id, path)
		}

		sp := s // capture
		tmp[s.Id] = &sp
		return nil
	})

	if walkErr != nil {
		panic(fmt.Sprintf("mobs.LoadSchedules() failed: %v", walkErr))
	}

	// World-aware validation: confirm rooms exist and consecutive segments are
	// reachable via the mapper. Only runs when the validator has been wired in.
	if scheduleWorldValidator.roomExists != nil && scheduleWorldValidator.hasPath != nil {
		for _, s := range tmp {
			if valErr := validateScheduleAgainstWorld(s,
				scheduleWorldValidator.roomExists,
				scheduleWorldValidator.hasPath,
			); valErr != nil {
				panic(fmt.Sprintf(
					"mobs.LoadSchedules() world validation failed for schedule %q: %v",
					s.Id, valErr,
				))
			}
		}
	}

	schedulesMu.Lock()
	schedules = tmp
	schedulesMu.Unlock()

	mudlog.Info("mobs.LoadSchedules()", "loadedCount", len(tmp), "Time Taken", time.Since(start))
}

// validateScheduleStandalone checks a schedule for internal consistency
// without touching the rooms registry or filesystem. Suitable for unit tests.
//
// Rules:
//   - Every ScheduleSegment must have Start in [0, 23] and End in [1, 24].
//   - Start != End (zero-duration segments are disallowed).
//   - target_room must be non-zero.
//   - Segments must cover all 24 hours exactly once (no gaps, no overlaps).
func validateScheduleStandalone(s *Schedule) error {
	if len(s.Segments) == 0 {
		return fmt.Errorf("schedule %q has no segments", s.Id)
	}

	// Per-segment bounds check.
	for i, seg := range s.Segments {
		if seg.Start < 0 || seg.Start > 23 {
			return fmt.Errorf("schedule %q segment %d: start=%d is out of range [0, 23]",
				s.Id, i, seg.Start)
		}
		if seg.End < 1 || seg.End > 24 {
			return fmt.Errorf("schedule %q segment %d: end=%d is out of range [1, 24]",
				s.Id, i, seg.End)
		}
		if seg.Start == seg.End {
			return fmt.Errorf("schedule %q segment %d: start equals end (%d), zero-duration segment",
				s.Id, i, seg.Start)
		}
		if seg.TargetRoom == 0 {
			return fmt.Errorf("schedule %q segment %d: target_room is 0 (must be a valid room id)",
				s.Id, i)
		}
	}

	// Coverage check: walk all 24 hours and ensure each hour is claimed by
	// exactly one segment.
	hourOwner := [24]int{} // value = 1-based segment index that claims this hour; 0 = unclaimed
	for i, seg := range s.Segments {
		for h := 0; h < 24; h++ {
			if !segContainsHour(&seg, h) {
				continue
			}
			if hourOwner[h] != 0 {
				prev := hourOwner[h] - 1
				return fmt.Errorf(
					"schedule %q: overlap at hour %d — claimed by segment %d (start=%d end=%d) and segment %d (start=%d end=%d)",
					s.Id, h,
					prev, s.Segments[prev].Start, s.Segments[prev].End,
					i, seg.Start, seg.End,
				)
			}
			hourOwner[h] = i + 1
		}
	}

	for h, owner := range hourOwner {
		if owner == 0 {
			return fmt.Errorf("schedule %q: coverage gap at hour %d (no segment covers this hour)", s.Id, h)
		}
	}

	return nil
}

// validateScheduleAgainstWorld checks that every target_room exists in the
// rooms registry, and that consecutive segment room pairs are reachable via
// the mapper. A missing path between consecutive segments is a hard error —
// all schedule transitions must be walkable.
//
// roomExists and hasPath are injected to avoid import cycles.
func validateScheduleAgainstWorld(s *Schedule, roomExists func(int) bool, hasPath func(from, to int) bool) error {
	if len(s.Segments) == 0 {
		return nil
	}

	// Check every target_room exists.
	for i, seg := range s.Segments {
		if !roomExists(seg.TargetRoom) {
			return fmt.Errorf("segment %d: target_room %d does not exist in rooms registry", i, seg.TargetRoom)
		}
	}

	// Sort a copy of segments by Start for pathfinding adjacency checks.
	sorted := make([]ScheduleSegment, len(s.Segments))
	copy(sorted, s.Segments)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Start < sorted[j].Start
	})

	n := len(sorted)
	for i := 0; i < n; i++ {
		fromRoom := sorted[i].TargetRoom
		toRoom := sorted[(i+1)%n].TargetRoom
		if fromRoom == toRoom {
			continue
		}
		if !hasPath(fromRoom, toRoom) {
			return fmt.Errorf("segment transition %d→%d: no path from room %d to room %d",
				i, (i+1)%n, fromRoom, toRoom)
		}
	}

	return nil
}
