package goals

import (
	"fmt"
	"sync"

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
			if !sliceContains(dstMeta.ConflictsWith, srcType) {
				warnings = append(warnings, fmt.Sprintf(
					"goals: conflict %q→%q is one-sided (%q has no reverse declaration)",
					srcType, dstType, dstType))
			}
		}
	}
	return warnings
}

func sliceContains(s []string, needle string) bool {
	for _, x := range s {
		if x == needle {
			return true
		}
	}
	return false
}

// LookupGoalType returns the registered metadata for a goal type and
// whether it was found. Exported variant of the internal lookupMeta —
// catalog tests and future consumers may need to inspect the registry.
// Chunk 4.3.
func LookupGoalType(name string) (GoalTypeMeta, bool) {
	return lookupMeta(name)
}
