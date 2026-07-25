package rooms

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

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

	// An unrecognised period is worse than an error at runtime: AddPeriod does
	// not fail, it falls through to a failover that treats the quantity as
	// rounds — so a typo yields a spawn returning in about four seconds, and
	// nobody notices until the world feels wrong. See RealPeriodOK.
	if s.RespawnRate != "" && !v.periodOK(s.RespawnRate) {
		return fmt.Errorf("respawn rate %q is not a period the engine understands (e.g. \"5 real minutes\")", s.RespawnRate)
	}
	return nil
}

// periodUnitPrefixes are the unit prefixes gametime.AddPeriod actually
// branches on (it compares the first three characters), plus the two it
// matches in full.
//
// "rou" (rounds) is deliberately included even though AddPeriod has no branch
// for it: an unrecognised unit falls through to a generic failover that treats
// the quantity as a count of ROUNDS. That failover is why "600 rounds" — the
// idiom already used across the world — works.
//
// It is also why validation here cannot simply ask the parser. AddPeriod never
// errors and never leaves the round unchanged: "banana" takes the same
// failover and yields ONE round, roughly four seconds. A typo'd respawn rate
// is therefore not a loud failure but a spawn that returns almost instantly.
// Checking the vocabulary is the only way to catch it.
var periodUnitPrefixes = []string{"yea", "mon", "wee", "day", "dai", "hou", "min", "noo", "mid", "rou"}

// periodModifiers are the words AddPeriod accepts between quantity and unit.
var periodModifiers = map[string]bool{"real": true, "irl": true, "game": true, "gametime": true}

// RealPeriodOK reports whether a respawn-rate string names a time unit the
// engine understands. Empty is legal and means the engine default.
func RealPeriodOK(period string) bool {
	if period == "" {
		return true
	}
	parts := strings.Fields(strings.ToLower(period))
	if len(parts) == 0 || len(parts) > 3 {
		return false
	}

	// Drop the real/irl/game modifier and any leading quantity; whatever
	// remains must be the unit.
	rest := []string{}
	for i, p := range parts {
		if periodModifiers[p] {
			continue
		}
		if i == 0 && len(parts) > 1 {
			if n, err := strconv.Atoi(p); err != nil || n < 1 {
				return false // a leading quantity that is not a positive number
			}
			continue
		}
		rest = append(rest, p)
	}
	if len(rest) != 1 {
		return false
	}

	unit := rest[0]
	if unit == "sunrise" || unit == "sunrises" || unit == "sunset" || unit == "sunsets" {
		return true
	}
	if len(unit) < 3 {
		return false
	}
	return slices.Contains(periodUnitPrefixes, unit[0:3])
}

// ValidateSpawnEntryLive is the production wiring, checked against the real
// registries and the given room's containers.
func ValidateSpawnEntryLive(s SpawnInfo, containers map[string]Container) error {
	set := map[string]struct{}{}
	for name := range containers {
		set[name] = struct{}{}
	}
	return ValidateSpawnEntry(s, spawnValidators{
		mobExists:  func(id int) bool { return mobs.GetMobSpec(mobs.MobId(id)) != nil },
		itemExists: func(id int) bool { return items.GetItemSpec(id) != nil },
		buffExists: func(id int) bool { return buffs.GetBuffSpec(id) != nil },
		periodOK:   RealPeriodOK,
		containers: set,
	})
}
