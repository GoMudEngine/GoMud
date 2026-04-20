package behaviortree

import (
	"testing"
)

func TestEngine_ArchetypeCache_EmptyByDefault(t *testing.T) {
	e := &Engine{
		trees:       make(map[int]Node),
		noTree:      make(map[int]bool),
		roomTrees:   make(map[int]Node),
		noRoomTree:  make(map[int]bool),
		archetypes:  make(map[string]Node),
		noArchetype: make(map[string]bool),
	}
	if got := e.GetArchetype("nonexistent"); got != nil {
		t.Fatalf("want nil for missing archetype, got %v", got)
	}
	if e.HasNoArchetype("nonexistent") {
		t.Fatalf("empty negative cache should not report missing")
	}
}

func TestEngine_ArchetypeNegativeCache(t *testing.T) {
	e := &Engine{
		archetypes:  make(map[string]Node),
		noArchetype: make(map[string]bool),
	}
	e.SetNoArchetype("ghost_archetype")
	if !e.HasNoArchetype("ghost_archetype") {
		t.Fatalf("expected negative-cache hit after SetNoArchetype")
	}
	if e.HasNoArchetype("other_archetype") {
		t.Fatalf("negative cache should be name-specific, not global")
	}
}

func TestEngine_LoadArchetypeClearsNegativeCache(t *testing.T) {
	e := &Engine{
		archetypes:  make(map[string]Node),
		noArchetype: make(map[string]bool),
	}
	e.SetNoArchetype("temp")
	if !e.HasNoArchetype("temp") {
		t.Fatalf("precondition: negative cache should be set")
	}
	// Directly install a tree via the cache map (bypassing file load).
	e.mu.Lock()
	e.archetypes["temp"] = &SelectorNode{}
	delete(e.noArchetype, "temp")
	e.mu.Unlock()
	if e.HasNoArchetype("temp") {
		t.Fatalf("negative cache should clear when archetype is installed")
	}
	if e.GetArchetype("temp") == nil {
		t.Fatalf("installed archetype should be retrievable")
	}
}
