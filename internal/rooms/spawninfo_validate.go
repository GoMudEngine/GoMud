package rooms

import "fmt"

// spawnValidators injects the existence checks so the spawn policy is
// testable without a loaded world.
type spawnValidators struct {
	mobExists  func(id int) bool
	itemExists func(id int) bool
	buffExists func(id int) bool
	periodOK   func(period string) bool
	containers map[string]struct{} // container nouns present in THIS room
}

// ValidateSpawnEntry enforces the spawn-entry rules an author can get wrong.
//
// A spawn entry spawns exactly ONE of: a mob, an item, or gold. The YAML
// shape allows all three at once, which behaves unpredictably; reject it at
// save so a contradictory entry cannot reach a room file.
func ValidateSpawnEntry(s SpawnInfo, v spawnValidators) error {
	kinds := 0
	if s.MobId != 0 {
		kinds++
	}
	if s.ItemId != 0 {
		kinds++
	}
	if s.Gold != 0 {
		kinds++
	}
	if kinds == 0 {
		return fmt.Errorf("a spawn entry must spawn a mob, an item, or gold")
	}
	if kinds > 1 {
		return fmt.Errorf("a spawn entry must spawn exactly one of mob / item / gold")
	}

	if s.MobId != 0 && !v.mobExists(s.MobId) {
		return fmt.Errorf("mob %d does not exist", s.MobId)
	}
	if s.ItemId != 0 && !v.itemExists(s.ItemId) {
		return fmt.Errorf("item %d does not exist", s.ItemId)
	}

	if s.Container != "" {
		if s.MobId != 0 {
			return fmt.Errorf("a mob cannot spawn into a container")
		}
		if _, ok := v.containers[s.Container]; !ok {
			return fmt.Errorf("this room has no container named %q", s.Container)
		}
	}

	for _, b := range s.BuffIds {
		if !v.buffExists(b) {
			return fmt.Errorf("buff %d does not exist", b)
		}
	}

	// An unparseable period is worse than an error at runtime: AddPeriod
	// returns the current round, so the spawn returns immediately and nobody
	// notices until the world feels wrong.
	if s.RespawnRate != "" && !v.periodOK(s.RespawnRate) {
		return fmt.Errorf("respawn rate %q is not a period the engine understands (e.g. \"5 real minutes\")", s.RespawnRate)
	}
	return nil
}
