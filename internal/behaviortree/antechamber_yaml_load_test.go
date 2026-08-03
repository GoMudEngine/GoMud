package behaviortree

import (
	"path/filepath"
	"testing"
)

// TestAntechamberBtreeYAMLsLoad verifies every hand-written behavior tree
// in the newcomer tutorial (Dewey's per-mob tree incl. the flee re-nag
// branch, plus the six room exit-gate/handoff trees) parses and compiles
// cleanly. Btrees load lazily and a structural error (unknown condition,
// bad decorator mod, mis-indent) fails silently in-game — this catches it
// at test time instead. Same rationale as TestBanditBtreeYAMLsLoad.
func TestAntechamberBtreeYAMLsLoad(t *testing.T) {
	patterns := []string{
		"../../_datafiles/world/dogmud/behaviors/newcomer_antechamber/*.yaml",
		"../../_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/*.yaml",
	}
	var files []string
	for _, p := range patterns {
		matches, err := filepath.Glob(p)
		if err != nil {
			t.Fatalf("glob %s: %v", p, err)
		}
		files = append(files, matches...)
	}
	if len(files) == 0 {
		t.Fatal("no antechamber behavior files found — path drift?")
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			node, err := LoadTreeFromFile(f)
			if err != nil {
				t.Fatalf("load failed: %v", err)
			}
			if node == nil {
				t.Fatal("loaded a nil tree")
			}
		})
	}
}
