package species

// SeedSpeciesForTest replaces the global allSpecies map with the supplied test
// data and returns a cleanup function that restores the original.
// Intended for cross-package integration tests (hooks, commands).
func SeedSpeciesForTest(specs map[int]*Species) func() {
	orig := allSpecies
	allSpecies = specs
	return func() {
		allSpecies = orig
	}
}
