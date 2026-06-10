package mutators

import "testing"

// SeedSpecsForTest registers mutator specs into the global registry for the
// duration of a test, restoring the prior state on cleanup.
func SeedSpecsForTest(t *testing.T, specs ...MutatorSpec) {
	t.Helper()
	prev := allMutators
	next := make(map[string]*MutatorSpec, len(prev)+len(specs))
	for k, v := range prev {
		next[k] = v
	}
	for i := range specs {
		s := specs[i]
		next[s.MutatorId] = &s
	}
	allMutators = next
	t.Cleanup(func() { allMutators = prev })
}
