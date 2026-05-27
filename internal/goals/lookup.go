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
