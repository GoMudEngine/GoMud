package devtools

import (
	"fmt"
	"os"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mapper"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// CheckZoneConsistency scans all room YAML files in a zone and reports:
//   - Exits pointing to invalid destinations
//   - Missing reverse (reciprocal) exits for intra-zone connections
//   - Orphaned rooms (no room has an exit leading to them)
//
// Returns a formatted report string, the total issue count, and any fatal error.
func CheckZoneConsistency(zoneName string) (report string, issueCount int, err error) {
	dirPath := util.FilePath(
		configs.GetFilePathsConfig().DataFiles.String(),
		"/rooms/",
		rooms.ZoneToFolder(zoneName),
	)
	return checkZoneConsistencyFromDir(dirPath, zoneName)
}

// checkZoneConsistencyFromDir is the testable core: it scans dirPath directly.
func checkZoneConsistencyFromDir(dirPath, zoneName string) (report string, issueCount int, err error) {

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", 0, fmt.Errorf("cannot read zone directory %q: %w", dirPath, err)
	}

	type exitRoomRef struct {
		RoomId int `yaml:"roomid"`
	}
	type roomData struct {
		RoomId int                    `yaml:"roomid"`
		Zone   string                 `yaml:"zone"`
		Exits  map[string]exitRoomRef `yaml:"exits"`
	}

	zoneRooms := make(map[int]roomData)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		filePath := dirPath + entry.Name()
		raw, readErr := os.ReadFile(filePath)
		if readErr != nil {
			return "", 0, fmt.Errorf("cannot read file %q: %w", filePath, readErr)
		}

		var rd roomData
		if yamlErr := yaml.Unmarshal(raw, &rd); yamlErr != nil {
			return "", 0, fmt.Errorf("cannot parse file %q: %w", filePath, yamlErr)
		}

		if rd.RoomId == 0 {
			continue // skip zone-config.yaml and similar non-room files
		}

		zoneRooms[rd.RoomId] = rd
	}

	if len(zoneRooms) == 0 {
		return fmt.Sprintf("Zone %q: no rooms found.\n", zoneName), 0, nil
	}

	// Build the set of room IDs referenced by at least one exit within the zone
	referenced := make(map[int]bool)
	for _, rm := range zoneRooms {
		for _, ex := range rm.Exits {
			referenced[ex.RoomId] = true
		}
	}

	var issues []string

	for id, rm := range zoneRooms {
		for dir, ex := range rm.Exits {
			if ex.RoomId == 0 {
				issues = append(issues, fmt.Sprintf("Room %d exit %s → 0: invalid destination", id, dir))
				continue
			}

			dest, exists := zoneRooms[ex.RoomId]
			if !exists {
				// Cross-zone exits are intentional — skip
				continue
			}

			// Use delta arithmetic to find the reciprocal direction.
			// This avoids non-determinism from GetReciprocalExit when multiple exit
			// names share the same (x,y,z) delta (e.g. "south" and "south-gap").
			dx, dy, dz := mapper.GetDelta(dir)
			if dx == 0 && dy == 0 && dz == 0 {
				continue // non-compass exit; reciprocal check not applicable
			}
			revX, revY, revZ := -dx, -dy, -dz

			hasRev := false
			for destDir := range dest.Exits {
				ddx, ddy, ddz := mapper.GetDelta(destDir)
				if ddx == revX && ddy == revY && ddz == revZ {
					hasRev = true
					break
				}
			}

			if !hasRev {
				issues = append(issues, fmt.Sprintf(
					"Room %d exit %s → %d: missing reverse exit",
					id, dir, ex.RoomId,
				))
			}
		}

		// Orphan check: no room in the zone has an exit leading to this room
		if !referenced[id] {
			issues = append(issues, fmt.Sprintf("Room %d: orphan (no incoming connections)", id))
		}
	}

	issueCount = len(issues)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== Zone Consistency Report: %s ===\n", zoneName))
	sb.WriteString(fmt.Sprintf("Rooms checked: %d\n", len(zoneRooms)))
	if issueCount == 0 {
		sb.WriteString("No issues found.\n")
	} else {
		sb.WriteString(fmt.Sprintf("Issues found: %d\n\n", issueCount))
		for _, issue := range issues {
			sb.WriteString("  - ")
			sb.WriteString(issue)
			sb.WriteString("\n")
		}
	}

	return sb.String(), issueCount, nil
}
