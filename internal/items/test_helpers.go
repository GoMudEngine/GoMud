package items

// SeedItemsForTest replaces the global items map with the supplied test data
// and returns a cleanup function that restores the original.
// Intended for cross-package integration tests (hooks, commands).
func SeedItemsForTest(specs map[int]*ItemSpec) func() {
	orig := items
	items = specs
	return func() {
		items = orig
	}
}
