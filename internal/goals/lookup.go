package goals

import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
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

// PlanStateClearFn wipes any planner-owned intermediate state on a mob.
// Registered once at boot from main.go; invoked by Recompute on every
// goal switch. nil = no cleanup (safe default for tests).
//
// Chunk 4.4.
type PlanStateClearFn func(mob *mobs.Mob)

var planStateClear PlanStateClearFn // guarded by lookupMu

// SetPlanStateClear registers the plan-state cleanup callback. Called
// once at boot from main.go to bridge goals → planners without an
// import cycle. Pass nil to unregister (tests use this for isolation).
//
// Chunk 4.4.
func SetPlanStateClear(fn PlanStateClearFn) {
	lookupMu.Lock()
	planStateClear = fn
	lookupMu.Unlock()
}

// invokePlanStateClear calls the registered callback under panic
// recovery — mirrors invokeContextScore (4.2). A panic in the callback
// must not compromise the goal switch itself.
func invokePlanStateClear(mob *mobs.Mob) {
	lookupMu.RLock()
	fn := planStateClear
	lookupMu.RUnlock()
	if fn == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			mudlog.Warn("goals.planStateClear panic",
				"mob_id", mob.MobId,
				"panic", fmt.Sprintf("%v", r))
		}
	}()
	fn(mob)
}
