package keywords

// SeedKeywordsForTest replaces the global loadedKeywords with an empty but
// non-nil Aliases struct so that keyword lookups don't panic in tests.
// Returns a cleanup function that restores the original.
func SeedKeywordsForTest() func() {
	orig := loadedKeywords
	loadedKeywords = &Aliases{
		Help:               map[string]map[string][]string{},
		HelpAliases:        map[string][]string{},
		CommandAliases:     map[string][]string{},
		DirectionAliases:   map[string]string{},
		MapLegendOverrides: map[string]map[string]string{},
		helpTopics:         map[string]HelpTopic{},
		helpAliases:        map[string]string{},
		commandAliases:     map[string]string{},
		mapLegendOverrides: map[string]map[rune]string{},
	}
	return func() {
		loadedKeywords = orig
	}
}
