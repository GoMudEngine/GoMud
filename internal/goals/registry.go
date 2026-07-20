package goals

import (
	"fmt"
	"slices"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

var (
	registryMu   sync.RWMutex
	typeRegistry = map[string]GoalTypeMeta{}
)

// RegisterGoalType is called from each goal-type package's init().
// 4.1 ships with no callers; 4.3 fills the registry. Re-registration
// of an existing type overwrites and logs a warning (defensive — in
// practice each package registers once).
func RegisterGoalType(goalType string, meta GoalTypeMeta) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := typeRegistry[goalType]; exists {
		mudlog.Warn("goals.RegisterGoalType: overwriting existing registration",
			"type", goalType)
	}
	typeRegistry[goalType] = meta
}

// lookupMeta returns the registered metadata for goalType. Internal —
// store.go and registry validation are the only callers.
func lookupMeta(goalType string) (GoalTypeMeta, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	m, ok := typeRegistry[goalType]
	return m, ok
}

// ValidateSymmetry walks every registered type's ConflictsWith list
// and verifies that each target also declares the source. Returns a
// slice of warning strings (empty if all pairs are symmetric).
//
// Called from cmd/main at boot after all packages have registered;
// soft check (does not panic, does not block startup).
func ValidateSymmetry() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	var warnings []string
	for srcType, srcMeta := range typeRegistry {
		for _, dstType := range srcMeta.ConflictsWith {
			dstMeta, ok := typeRegistry[dstType]
			if !ok {
				// Forward-declared conflict targeting an unregistered
				// type — not a symmetry issue, but worth noting.
				continue
			}
			if !slices.Contains(dstMeta.ConflictsWith, srcType) {
				warnings = append(warnings, fmt.Sprintf(
					"goals: conflict %q→%q is one-sided (%q has no reverse declaration)",
					srcType, dstType, dstType))
			}
		}
	}
	return warnings
}

// LookupGoalType returns the registered metadata for a goal type and
// whether it was found. Exported variant of the internal lookupMeta —
// catalog tests and future consumers may need to inspect the registry.
// Chunk 4.3.
func LookupGoalType(name string) (GoalTypeMeta, bool) {
	return lookupMeta(name)
}

// ContextScoreFor returns the live context-score multiplier for goal g
// against mob. It delegates to effectiveContextMod, which:
//   - returns 1.0 when no ContextScore is registered for g.Type
//   - returns 0  when the registered ContextScore panics
//   - returns 1.0 when mob is nil and the registered fn is nil
//
// When mob is nil and a ContextScore IS registered, the fn receives nil —
// well-authored ContextScore functions guard for nil mob and return 1.0.
// The panic-recovery in invokeContextScore converts any nil-deref to 0.
//
// Chunk 5.3 — used by the `goal scores` admin diagnostic.
func ContextScoreFor(g *Goal, mob *mobs.Mob) float64 {
	return effectiveContextMod(g, mob)
}

// WeightFor returns the archetype weight for goal g against mob.
// Resolves via the registered WeightsLookupFn (wired at boot by main.go).
// Returns 1.0 when no lookup is registered, when mob is nil, or when
// the goal type is absent from the archetype's weight map.
//
// Chunk 5.3 — used by the `goal scores` admin diagnostic.
func WeightFor(g *Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 1.0
	}
	weights := resolveWeights(mob)
	if w, ok := weights[g.Type]; ok {
		return w
	}
	return 1.0
}
