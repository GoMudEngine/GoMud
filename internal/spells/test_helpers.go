package spells

// SeedSpellsForTest replaces the global allSpells map with the supplied test
// data and returns a cleanup function that restores the original.
// Intended for cross-package integration tests (hooks, commands).
func SeedSpellsForTest(spellMap map[string]*SpellData) func() {
	orig := allSpells
	allSpells = spellMap
	return func() {
		allSpells = orig
	}
}
