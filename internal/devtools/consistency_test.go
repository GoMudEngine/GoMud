package devtools

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCheckZoneConsistency writes two rooms to a temp directory:
//   - Room 1 has a "north" exit to room 2
//   - Room 2 has NO "south" exit back to room 1
//
// Expects exactly 1 issue (missing reverse exit).
func TestCheckZoneConsistency(t *testing.T) {

	tmpDir := t.TempDir()
	zoneDir := filepath.Join(tmpDir, "test_zone") + string(os.PathSeparator)
	if err := os.MkdirAll(zoneDir, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	room1YAML := `roomid: 1
zone: test_zone
title: Room One
description: First room.
exits:
  north:
    roomid: 2
`
	room2YAML := `roomid: 2
zone: test_zone
title: Room Two
description: Second room.
exits: {}
`

	writeFile := func(name, content string) {
		if err := os.WriteFile(filepath.Join(zoneDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	writeFile("1.yaml", room1YAML)
	writeFile("2.yaml", room2YAML)

	// Override the zone dir path by monkey-patching is not possible here, so we
	// call the internal helper directly with the temp directory.
	report, issueCount, err := checkZoneConsistencyFromDir(zoneDir, "test_zone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expect 2 issues: (1) missing south reverse exit on room 2,
	// and (2) room 1 is an orphan because nothing exits to it.
	if issueCount != 2 {
		t.Errorf("expected 2 issues, got %d\nReport:\n%s", issueCount, report)
	}
}

// TestCheckZoneConsistencyClean verifies that a properly linked pair of rooms
// produces zero issues.
func TestCheckZoneConsistencyClean(t *testing.T) {

	tmpDir := t.TempDir()
	zoneDir := filepath.Join(tmpDir, "clean_zone") + string(os.PathSeparator)
	if err := os.MkdirAll(zoneDir, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	room1YAML := `roomid: 10
zone: clean_zone
title: Room Ten
description: First room.
exits:
  north:
    roomid: 11
`
	room2YAML := `roomid: 11
zone: clean_zone
title: Room Eleven
description: Second room.
exits:
  south:
    roomid: 10
`

	writeFile := func(name, content string) {
		if err := os.WriteFile(filepath.Join(zoneDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	writeFile("10.yaml", room1YAML)
	writeFile("11.yaml", room2YAML)

	report, issueCount, err := checkZoneConsistencyFromDir(zoneDir, "clean_zone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issueCount != 0 {
		t.Errorf("expected 0 issues, got %d\nReport:\n%s", issueCount, report)
	}
}
