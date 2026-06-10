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
}
