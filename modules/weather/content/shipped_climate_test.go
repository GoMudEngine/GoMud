package content

import (
	"os"
	"testing"
)

// Validates the shipped DOGMud climate profiles: all 17 biomes covered,
// parseable, indoor biomes have zero spawn weight.
func TestShippedDogmudClimateProfiles(t *testing.T) {
	fsys := os.DirFS("../../../_datafiles/world/dogmud")
	climate, err := LoadClimate(fsys, "weather/climate")
	if err != nil {
		t.Fatalf("LoadClimate: %v", err)
	}

	biomes := []string{"water", "shore", "cliffs", "desert", "snow", "mountains",
		"swamp", "forest", "farmland", "land", "road", "city",
		"cave", "dungeon", "house", "fort", "spiderweb"}
	for _, b := range biomes {
		p, ok := climate[b]
		if !ok {
			t.Errorf("missing climate profile for biome %q", b)
			continue
		}
		if len(p.Weather) == 0 {
			t.Errorf("%s: empty weather weights", b)
		}
	}

	for _, indoor := range []string{"cave", "dungeon", "house", "fort", "spiderweb"} {
		if w := climate[indoor].SpawnWeight; w != 0 {
			t.Errorf("%s: indoor biome must have spawnWeight 0, got %v", indoor, w)
		}
	}

	// Pin values that differ from the module's built-in defaults so a missing
	// file can't silently fall back (forest/desert/swamp exist in defaults too).
	// desert: DOGMud file adds overcast weight 1; DefaultClimate "desert" has none (0).
	if climate["desert"].Weather["overcast"] != 1 {
		t.Errorf("desert: expected overcast weight 1 from the DOGMud climate file, got %v",
			climate["desert"].Weather["overcast"])
	}
	// swamp: DOGMud file has fog weight 3; DefaultClimate "swamp" has fog weight 5.
	if climate["swamp"].Weather["fog"] != 3 {
		t.Errorf("swamp: expected fog weight 3 from the DOGMud climate file, got %v",
			climate["swamp"].Weather["fog"])
	}
	// forest: DOGMud file has spawnWeight 0.9; DefaultClimate "forest" has spawnWeight 1.0.
	if climate["forest"].SpawnWeight != 0.9 {
		t.Errorf("forest: expected spawnWeight 0.9 from the DOGMud climate file, got %v",
			climate["forest"].SpawnWeight)
	}
}
