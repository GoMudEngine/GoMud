package mobs

// SeedMobsForTest replaces all global mob registries with the supplied test
// data and returns a cleanup function that restores the originals.
// Intended for cross-package integration tests (hooks, commands).
func SeedMobsForTest(specs map[int]*Mob, instances map[int]*Mob) func() {
	origMobs := mobs
	origAllMobNames := allMobNames
	origMobInstances := mobInstances
	origMobsHatePlayers := mobsHatePlayers
	origMobNameCache := mobNameCache
	origRecentlyDied := recentlyDied
	origInstanceCounter := instanceCounter

	mobs = specs
	mobInstances = instances

	// Rebuild derived caches from specs
	names := make([]string, 0, len(specs))
	cache := make(map[MobId]string, len(specs))
	for id, m := range specs {
		names = append(names, m.Character.Name)
		cache[MobId(id)] = m.Character.Name
	}
	allMobNames = names
	mobNameCache = cache

	mobsHatePlayers = map[string]map[int]int{}
	recentlyDied = map[int]int{}
	instanceCounter = 200

	return func() {
		mobs = origMobs
		allMobNames = origAllMobNames
		mobInstances = origMobInstances
		mobsHatePlayers = origMobsHatePlayers
		mobNameCache = origMobNameCache
		recentlyDied = origRecentlyDied
		instanceCounter = origInstanceCounter
	}
}
