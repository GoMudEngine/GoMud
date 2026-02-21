package devtools

import (
	"fmt"
	"os"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// GenerateGrid creates a width×height grid of interconnected rooms in zoneName.
// Adjacent rooms are wired with bidirectional north/south/east/west exits.
//
// If the zone does not yet exist it is created via rooms.CreateZone, which
// writes zone-config.yaml, creates the instances folder, and registers the
// zone with the running engine — preventing the "No zone-config.yaml" startup
// panic. CreateZone also places one starter room in the zone; the grid rooms
// are appended after it.
//
// Returns the first and last grid room IDs, or an error.
func GenerateGrid(zoneName string, width, height int) (firstId, lastId int, err error) {

	if err = rooms.ValidateZoneName(zoneName); err != nil {
		return 0, 0, fmt.Errorf("invalid zone name: %w", err)
	}
	if width < 1 || height < 1 {
		return 0, 0, fmt.Errorf("width and height must be at least 1 (got %d×%d)", width, height)
	}

	// Ensure the zone has a zone-config.yaml and is registered with the engine.
	// CreateZone returns "zone already exists" when the zone is already set up —
	// that is fine; we just continue.
	if _, createErr := rooms.CreateZone(zoneName); createErr != nil {
		if !strings.Contains(createErr.Error(), "already exists") {
			return 0, 0, fmt.Errorf("failed to initialise zone %q: %w", zoneName, createErr)
		}
	}

	// GetNextRoomId now reflects any room created by CreateZone above.
	firstId = rooms.GetNextRoomId()

	dirPath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), "/rooms/", rooms.ZoneToFolder(zoneName))

	grid := buildGrid(zoneName, width, height, firstId)

	if writeErr := writeRoomsToDir(grid, dirPath); writeErr != nil {
		return 0, 0, writeErr
	}

	lastId = firstId + width*height - 1
	rooms.SetNextRoomId(firstId + width*height)

	return firstId, lastId, nil
}

// buildGrid constructs the Room slice for a width×height grid starting at firstId.
// It does not touch the filesystem or engine state.
func buildGrid(zoneName string, width, height, firstId int) []rooms.Room {
	total := width * height
	grid := make([]rooms.Room, 0, total)

	titleZone := toTitleCase(zoneName)

	cellId := func(col, row int) int {
		return firstId + row*width + col
	}

	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			id := cellId(col, row)
			r := rooms.Room{
				RoomId:      id,
				Zone:        zoneName,
				Title:       fmt.Sprintf("%s - Room %d", titleZone, id),
				Description: fmt.Sprintf("A room in %s. This area has not yet been described.", zoneName),
				Exits:       make(map[string]exit.RoomExit),
			}

			if col+1 < width {
				r.Exits["east"] = exit.RoomExit{RoomId: cellId(col+1, row)}
			}
			if col-1 >= 0 {
				r.Exits["west"] = exit.RoomExit{RoomId: cellId(col-1, row)}
			}
			if row+1 < height {
				r.Exits["south"] = exit.RoomExit{RoomId: cellId(col, row+1)}
			}
			if row-1 >= 0 {
				r.Exits["north"] = exit.RoomExit{RoomId: cellId(col, row-1)}
			}

			grid = append(grid, r)
		}
	}

	return grid
}

// writeRoomsToDir marshals each room to YAML and writes it to dirPath/<roomId>.yaml.
// dirPath must already exist.
func writeRoomsToDir(rs []rooms.Room, dirPath string) error {
	for _, r := range rs {
		data, err := yaml.Marshal(r)
		if err != nil {
			return fmt.Errorf("failed to marshal room %d: %w", r.RoomId, err)
		}
		filePath := fmt.Sprintf("%s%d.yaml", dirPath, r.RoomId)
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return fmt.Errorf("failed to write room %d: %w", r.RoomId, err)
		}
	}
	return nil
}

// toTitleCase converts snake_case or space-separated names to Title Case.
func toTitleCase(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
