package buffs

// SeedBuffsForTest replaces the global buffs map with the supplied test data
// and returns a cleanup function that restores the original.
// Intended for cross-package integration tests (hooks, commands).
func SeedBuffsForTest(buffMap map[int]*BuffSpec) func() {
	orig := buffs
	buffs = buffMap
	return func() {
		buffs = orig
	}
}
