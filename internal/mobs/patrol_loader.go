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

// patrolWorldValidator is injected at startup (after both rooms and mapper
// are initialized) to perform world-aware patrol validation without
// creating an import cycle. mobs ← rooms ← mobs would be a cycle. Same
// pattern as scheduleWorldValidator from chunk 3.2.
var patrolWorldValidator struct {
	roomExists func(id int) bool
	hasPath    func(from, to int) bool
	roomZone   func(id int) string // chunk 3.7: resolve room id → zone name for boot log
}

// SetPatrolWorldValidator wires in the room-existence, pathfinding, and
// zone-resolution checks for patrol validation. Must be called before
// LoadPatrols.
func SetPatrolWorldValidator(roomExists func(int) bool, hasPath func(from, to int) bool, roomZone func(int) string) {
	patrolWorldValidator.roomExists = roomExists
	patrolWorldValidator.hasPath = hasPath
	patrolWorldValidator.roomZone = roomZone
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

			// Chunk 3.7: log resolved zones for human-readable boot-log
			// sanity-checking of cross-zone patrols. No validation panics
			// — zone resolution failures just log an empty zone name. Room
			// existence is already validated above.
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
