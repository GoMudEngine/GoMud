package goals

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// WeightsLookupFn returns the per-mob archetype goal-weights map.
// Implemented by the boot wiring in main.go as a thin adapter over
// behaviortree.GetEngine().GetArchetypeGoalWeights(mob.BehaviorArchetype).
//
// Chunk 4.2 introduced this seam to keep the goals package free of
// any behaviortree import (would create a cycle — behaviortree already
// imports a lot, including rooms).
type WeightsLookupFn func(mob *mobs.Mob) map[string]float64

var (
	lookupMu      sync.RWMutex
	weightsLookup WeightsLookupFn // nil → no archetype weights applied (all 1.0)
)

// SetWeightsLookup registers the archetype-weights resolver. Called once
// at boot from main.go after behaviortree is wired up. Pass nil to
// unregister (tests use this for isolation).
func SetWeightsLookup(fn WeightsLookupFn) {
	lookupMu.Lock()
	weightsLookup = fn
	lookupMu.Unlock()
}

// resolveWeights returns the weights map for a mob, or nil if no
// lookup is registered. Internal — called by Recompute.
func resolveWeights(mob *mobs.Mob) map[string]float64 {
	lookupMu.RLock()
	fn := weightsLookup
	lookupMu.RUnlock()
	if fn == nil {
		return nil
	}
	return fn(mob)
}

// ArchetypeDefaultsLookupFn returns the archetype's default goal list
// for the given mob. Registered once at boot from main.go as a thin
// adapter over behaviortree.GetEngine().GetArchetypeDefaultGoals
// (returning the goals-package mirror type). Chunk 4.3.
type ArchetypeDefaultsLookupFn func(mob *mobs.Mob) []GoalDefault

var archetypeDefaultsLookup ArchetypeDefaultsLookupFn // guarded by lookupMu

// SetArchetypeDefaultsLookup registers the archetype-defaults resolver.
// Pass nil to unregister (tests use this for isolation). Chunk 4.3.
func SetArchetypeDefaultsLookup(fn ArchetypeDefaultsLookupFn) {
	lookupMu.Lock()
	archetypeDefaultsLookup = fn
	lookupMu.Unlock()
}

// resolveArchetypeDefaults returns the archetype defaults for a mob,
// or nil if no lookup is registered. Internal — called by the lazy-
// seed path in loadOrLazyInit. Chunk 4.3.
func resolveArchetypeDefaults(mob *mobs.Mob) []GoalDefault {
	lookupMu.RLock()
	fn := archetypeDefaultsLookup
	lookupMu.RUnlock()
	if fn == nil {
		return nil
	}
	return fn(mob)
}
