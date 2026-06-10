package mutators

// SeedSpecsForTest registers mutator specs for tests, returning a cleanup
// func that restores the prior registry. Mirrors rooms.SeedBiomesForTest.
func SeedSpecsForTest(specs ...MutatorSpec) func() {
	prev := allMutators
	next := map[string]*MutatorSpec{}
	for k, v := range prev {
		next[k] = v
	}
	for i := range specs {
		s := specs[i]
		next[s.MutatorId] = &s
	}
	allMutators = next
	return func() { allMutators = prev }
}
