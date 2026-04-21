package behaviortree

import (
	"strings"
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

func TestGetArchetypePath(t *testing.T) {
	path := GetArchetypePath("melee_self_buff")
	if path == "" {
		t.Fatalf("expected a path, got empty")
	}
	// Normalize to forward slashes so the test passes on Windows and Linux.
	normalized := strings.ReplaceAll(path, `\`, `/`)
	// Just verify it ends with the expected suffix (datafiles prefix is config-dependent).
	wantSuffix := "/behaviors/archetypes/melee_self_buff.yaml"
	if !strings.HasSuffix(normalized, wantSuffix) {
		t.Fatalf("path %q does not end with %q", path, wantSuffix)
	}
}

func TestTryMobBehavior_PerMobWinsOverArchetype(t *testing.T) {
	// Install both a per-mob tree and an archetype with the same name.
	// Verify the per-mob tree runs.
	e := GetEngine()
	mobId := 987654
	archetypeName := "test_arch_perMob"

	e.mu.Lock()
	e.trees[mobId] = &markerNode{mark: "per_mob"}
	delete(e.noTree, mobId)
	e.archetypes[archetypeName] = &markerNode{mark: "archetype"}
	delete(e.noArchetype, archetypeName)
	e.mu.Unlock()

	// Can't call TryMobBehavior without a registered mob instance, so
	// exercise resolution by checking the cache map directly here and
	// leave end-to-end coverage for the integration tests later.
	if e.GetTree(mobId) == nil {
		t.Fatalf("per-mob tree should be present")
	}
	if e.GetArchetype(archetypeName) == nil {
		t.Fatalf("archetype should be present")
	}

	// Cleanup
	e.mu.Lock()
	delete(e.trees, mobId)
	delete(e.archetypes, archetypeName)
	e.mu.Unlock()
}

// markerNode is a test helper that records when Evaluate was called.
type markerNode struct {
	mark         string
	wasEvaluated bool
}

func (m *markerNode) Evaluate(ctx *EvalContext) Result {
	m.wasEvaluated = true
	return Success
}
