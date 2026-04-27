package behaviortree

import "testing"

// TestBanditBtreeYAMLsLoad verifies that all four hand-written bandit
// behavior tree files in _datafiles/world/dogmud/behaviors/north_road/
// parse cleanly. A YAML edit that introduces a structural error (mismatched
// indentation, unknown action name, etc.) would manifest in-game as the
// btree silently failing to fire — this catches it at test time instead.
func TestBanditBtreeYAMLsLoad(t *testing.T) {
	files := []string{
		"../../_datafiles/world/dogmud/behaviors/north_road/283-bandit_lookout.yaml",
		"../../_datafiles/world/dogmud/behaviors/north_road/284-bandit_fighter.yaml",
		"../../_datafiles/world/dogmud/behaviors/north_road/285-bandit_caster.yaml",
		"../../_datafiles/world/dogmud/behaviors/north_road/286-soren.yaml",
	}
	for _, f := range files {
		t.Run(f, func(t *testing.T) {
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
