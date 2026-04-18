package mutations

// SeedMutationsForTest replaces the package-private allMutations map with the
// supplied test data and returns a cleanup function that restores the original.
// Intended for cross-package integration tests (behaviortree, hooks).
func SeedMutationsForTest(mutMap map[string]*MutationSpec) func() {
	orig := allMutations
	allMutations = mutMap
	return func() {
		allMutations = orig
	}
}
