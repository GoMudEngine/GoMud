package sim

import "testing"

func TestClimateForFallsBackToDefault(t *testing.T) {
	c := DefaultClimate()
	if _, ok := c["default"]; !ok {
		t.Fatal("DefaultClimate must include a 'default' profile")
	}
	// A biome with no profile returns the default profile.
	got := c.For("no-such-biome")
	if got.SpawnWeight != c["default"].SpawnWeight || len(got.Weather) != len(c["default"].Weather) {
		t.Error("For(unknown) should return the default profile")
	}
	// A known biome returns its own profile.
	if _, ok := c["tundra"]; ok {
		if _, hasSnow := c.For("tundra").Weather["snow"]; !hasSnow {
			t.Error("tundra profile should include snow")
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxActiveFronts <= 0 {
		t.Error("MaxActiveFronts must be positive")
	}
	if cfg.HistoryLen <= 0 {
		t.Error("HistoryLen must be positive")
	}
	if cfg.SpawnChance < 0 || cfg.SpawnChance > 1 {
		t.Error("SpawnChance must be in [0,1]")
	}
	if cfg.CoverageFalloff <= 0 || cfg.CoverageFalloff > 1 {
		t.Error("CoverageFalloff must be in (0,1]")
	}
	if cfg.MinProjected <= 0 || cfg.MinProjected > 1 {
		t.Error("MinProjected must be in (0,1]")
	}
	if cfg.MaxFrontRadius < 0 {
		t.Error("MaxFrontRadius must be >= 0")
	}
}
