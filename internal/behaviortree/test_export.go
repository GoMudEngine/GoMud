package behaviortree

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
