package behaviortree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetArchetypeGoalWeights_ParsedFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "weighted_archetype.yaml")
	yaml := []byte(`tree:
  type: action
  do: set_state
  key: goal_weights_test
  value: ok
goal_weights:
  revenge: 1.5
  wealth: 0.8
  protection: 0.7
`)
	if err := os.WriteFile(path, yaml, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	e := newEngineForTest()
	if err := e.LoadArchetype("weighted_archetype", path); err != nil {
		t.Fatalf("LoadArchetype: %v", err)
	}
	got := e.GetArchetypeGoalWeights("weighted_archetype")
	if got["revenge"] != 1.5 {
		t.Errorf("revenge: got %f, want 1.5", got["revenge"])
	}
	if got["wealth"] != 0.8 {
		t.Errorf("wealth: got %f, want 0.8", got["wealth"])
	}
	if got["protection"] != 0.7 {
		t.Errorf("protection: got %f, want 0.7", got["protection"])
	}
}

func TestGetArchetypeGoalWeights_AbsentField_EmptyMap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no_weights.yaml")
	yaml := []byte(`tree:
  type: action
  do: set_state
  key: no_weights_test
  value: ok
`)
	if err := os.WriteFile(path, yaml, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	e := newEngineForTest()
	if err := e.LoadArchetype("no_weights", path); err != nil {
		t.Fatalf("LoadArchetype: %v", err)
	}
	got := e.GetArchetypeGoalWeights("no_weights")
	if len(got) != 0 {
		t.Errorf("len=%d, want 0", len(got))
	}
}

func TestGetArchetypeGoalWeights_UnknownArchetype_EmptyMap(t *testing.T) {
	e := newEngineForTest()
	got := e.GetArchetypeGoalWeights("never_loaded")
	if len(got) != 0 {
		t.Errorf("len=%d, want 0", len(got))
	}
}

// newEngineForTest returns a fresh Engine for isolated goal-weights tests.
func newEngineForTest() *Engine {
	return &Engine{
		trees:                 map[int]Node{},
		noTree:                map[int]bool{},
		roomTrees:             map[int]Node{},
		noRoomTree:            map[int]bool{},
		archetypes:            map[string]Node{},
		noArchetype:           map[string]bool{},
		archetypeGoalWeights:  map[string]map[string]float64{},
		archetypeDefaultGoals: map[string][]GoalDefault{},
	}
}
