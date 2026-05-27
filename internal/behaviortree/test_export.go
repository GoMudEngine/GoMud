package behaviortree

import "testing"

// File: test_export.go
//
// Test-only helpers exposed for cross-package integration tests
// (e.g., hooks-package routing tests need to install a recording
// behavior tree to detect mob_hurt dispatch).
//
// These functions are kept in a non-_test.go file because Go does
// not export _test.go symbols across package boundaries. They are
// nevertheless intended for test use only — production code paths
// load trees via LoadTree (file-based) and never call these.

// LoadArchetypeForTest loads a behavior-tree archetype from the given path
// and installs it in the engine under the given name. Cleans up on test end.
//
// path may be absolute or relative to the process working directory (which
// for Go tests is the package directory). The canonical production path for
// an archetype named "melee_self_buff" in the dogmud world is:
//
//	../../_datafiles/world/dogmud/behaviors/archetypes/melee_self_buff.yaml
//
// (relative from internal/behaviortree/).
func LoadArchetypeForTest(t *testing.T, name string, path string) {
	t.Helper()
	if err := GetEngine().LoadArchetype(name, path); err != nil {
		t.Fatalf("LoadArchetypeForTest(%s, %s): %v", name, path, err)
	}
	t.Cleanup(func() {
		e := GetEngine()
		e.mu.Lock()
		delete(e.archetypes, name)
		delete(e.noArchetype, name)
		delete(e.archetypeGoalWeights, name)
		delete(e.archetypeDefaultGoals, name)
		e.mu.Unlock()
	})
}

// DrainAllDelayedActionsForTest immediately executes all pending delayed
// actions in the engine's queue, regardless of their scheduled time.
// This is used in integration tests to force perception-delayed BTree actions
// (e.g., cast_best_in_category) to fire synchronously.
//
// FOR TEST USE ONLY.
func DrainAllDelayedActionsForTest(t *testing.T) {
	t.Helper()
	e := GetEngine()
	e.mu.Lock()
	all := make([]DelayedAction, len(e.queue))
	copy(all, e.queue)
	e.queue = e.queue[:0]
	e.mu.Unlock()
	for _, da := range all {
		safeExecuteDelayed("test.DrainAllDelayedActionsForTest", da.Action)
	}
}

// SetMobTreeForTest installs a behavior tree node directly into the
// engine's mob-tree cache, bypassing file loading. Use only from tests.
// Returns a cleanup function that removes the installed tree (and any
// negative-cache entry that may have been set by an earlier load
// attempt for the same mobId).
func (e *Engine) SetMobTreeForTest(mobId int, n Node) func() {
	e.mu.Lock()
	prev, hadPrev := e.trees[mobId]
	prevNoTree := e.noTree[mobId]
	e.trees[mobId] = n
	delete(e.noTree, mobId)
	e.mu.Unlock()
	return func() {
		e.mu.Lock()
		if hadPrev {
			e.trees[mobId] = prev
		} else {
			delete(e.trees, mobId)
		}
		if prevNoTree {
			e.noTree[mobId] = true
		} else {
			delete(e.noTree, mobId)
		}
		e.mu.Unlock()
	}
}
