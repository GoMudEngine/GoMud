package behaviortree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetArchetypeDefaultGoals_ParsedFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "defaults_archetype.yaml")
	yaml := []byte(`tree:
  type: action
  do: set_state
default_goals:
  - type: survival
    priority: 80
  - type: wealth-gold
    priority: 40
    params:
      target: 500
`)
	if err := os.WriteFile(path, yaml, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	e := newEngineForTest()
	if err := e.LoadArchetype("defaults_archetype", path); err != nil {
		t.Fatalf("LoadArchetype: %v", err)
	}
	got := e.GetArchetypeDefaultGoals("defaults_archetype")
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if got[0].Type != "survival" || got[0].Priority != 80 {
		t.Errorf("got[0]=%+v, want survival/80", got[0])
	}
	if got[1].Type != "wealth-gold" || got[1].Priority != 40 {
		t.Errorf("got[1]=%+v, want wealth-gold/40", got[1])
	}
}

func TestGetArchetypeDefaultGoals_AbsentField_EmptyList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no_defaults.yaml")
	yaml := []byte(`tree:
  type: action
  do: set_state
`)
	if err := os.WriteFile(path, yaml, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	e := newEngineForTest()
	if err := e.LoadArchetype("no_defaults", path); err != nil {
		t.Fatalf("LoadArchetype: %v", err)
	}
	got := e.GetArchetypeDefaultGoals("no_defaults")
	if len(got) != 0 {
		t.Errorf("len=%d, want 0", len(got))
	}
}

func TestGetArchetypeDefaultGoals_UnknownArchetype_EmptyList(t *testing.T) {
	e := newEngineForTest()
	got := e.GetArchetypeDefaultGoals("never_loaded")
	if len(got) != 0 {
		t.Errorf("len=%d, want 0", len(got))
	}
}
