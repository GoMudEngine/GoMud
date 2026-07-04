package enchantments

// SeedEnchantmentsForTest replaces the package-private allEnchantments
// registry with the supplied test data and returns a cleanup function that
// restores the original. Intended for cross-package integration tests
// (characters, hooks) that need GetTierReservePct/GetEnchantment to resolve
// without loading the full YAML data set. Mirrors items.SeedItemsForTest and
// mutations.SeedMutationsForTest.
func SeedEnchantmentsForTest(defs map[string]*EnchantmentDef) func() {
	orig := allEnchantments
	allEnchantments = defs
	return func() {
		allEnchantments = orig
	}
}
