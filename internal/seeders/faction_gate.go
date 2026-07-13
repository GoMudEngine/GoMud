package seeders

// registeredFactionIds returns the subset of groups that are registered
// factions, per the isFaction predicate. Kept pure + injectable so the gate
// decision is unit-testable without loading the faction registry or the live
// mob-instance map. Callers pass factions.GetDefinition(g) != nil as isFaction.
func registeredFactionIds(groups []string, isFaction func(string) bool) []string {
	out := make([]string, 0, len(groups))
	for _, g := range groups {
		if isFaction(g) {
			out = append(out, g)
		}
	}
	return out
}
