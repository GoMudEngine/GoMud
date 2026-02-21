package devtools

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
)

// TestGenerateGrid3x2 builds a 3×2 grid in a temp directory and verifies:
//   - 6 YAML files are created
//   - CheckZoneConsistency reports 0 issues
func TestGenerateGrid3x2(t *testing.T) {
	const (
		zoneName = "test_grid"
		width    = 3
		height   = 2
		firstId  = 1000 // arbitrary starting ID for the test
	)

	tmpDir := t.TempDir()
	zoneDir := filepath.Join(tmpDir, "test_grid") + string(os.PathSeparator)
	if err := os.MkdirAll(zoneDir, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Build the grid rooms without touching engine configs
	grid := buildGrid(zoneName, width, height, firstId)
	if len(grid) != width*height {
		t.Fatalf("expected %d rooms, got %d", width*height, len(grid))
	}

	// Write them to the temp dir
	if err := writeRoomsToDir(grid, zoneDir); err != nil {
		t.Fatalf("writeRoomsToDir: %v", err)
	}

	// Verify file count
	entries, err := os.ReadDir(zoneDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	yamlCount := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".yaml" {
			yamlCount++
		}
	}
	if yamlCount != width*height {
		t.Errorf("expected %d YAML files, got %d", width*height, yamlCount)
	}

	// Consistency check — all exits should be bidirectional within the zone
	report, issueCount, err := checkZoneConsistencyFromDir(zoneDir, zoneName)
	if err != nil {
		t.Fatalf("consistency check error: %v", err)
	}
	if issueCount != 0 {
		t.Errorf("expected 0 issues in generated grid, got %d\nReport:\n%s", issueCount, report)
	}
}

// TestBuildGridExits verifies that corner and edge rooms have the right number of exits.
func TestBuildGridExits(t *testing.T) {
	const (
		zoneName = "exit_test"
		width    = 3
		height   = 3
		firstId  = 2000
	)

	grid := buildGrid(zoneName, width, height, firstId)

	// Build a quick lookup
	byId := make(map[int]map[string]exit.RoomExit)
	for _, r := range grid {
		byId[r.RoomId] = r.Exits
	}

	tests := []struct {
		col, row    int
		expectedDir int // expected number of exits
	}{
		{0, 0, 2}, // top-left corner: east + south
		{1, 0, 3}, // top edge: west + east + south
		{2, 0, 2}, // top-right corner: west + south
		{0, 1, 3}, // left edge: north + east + south
		{1, 1, 4}, // center: all 4
		{2, 1, 3}, // right edge: north + west + south
		{0, 2, 2}, // bottom-left corner: north + east
		{1, 2, 3}, // bottom edge: north + west + east
		{2, 2, 2}, // bottom-right corner: north + west
	}

	cellId := func(col, row int) int { return firstId + row*width + col }

	for _, tt := range tests {
		id := cellId(tt.col, tt.row)
		exits, ok := byId[id]
		if !ok {
			t.Errorf("room at (%d,%d) id=%d not found", tt.col, tt.row, id)
			continue
		}
		if got := len(exits); got != tt.expectedDir {
			keys := make([]string, 0, len(exits))
			for k := range exits {
				keys = append(keys, k)
			}
			t.Errorf("(%d,%d) expected %d exits, got %d: %v",
				tt.col, tt.row, tt.expectedDir, got, fmt.Sprintf("%v", keys))
		}
	}
}
