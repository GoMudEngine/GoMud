package content

import (
	"testing"
	"testing/fstest"

	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

const tundraYAML = `biome: tundra
weather:
  snow: 9
  clear: 1
influence:
  intensityDelta: -0.07
  moistureDelta: -0.03
  movementResistance: 0.4
spawnWeight: 0.5
`

func TestParseClimate(t *testing.T) {
	biome, p, err := ParseClimate([]byte(tundraYAML))
	if err != nil {
		t.Fatal(err)
	}
	if biome != "tundra" {
		t.Errorf("biome: %q", biome)
	}
	if p.Weather[sim.WeatherType("snow")] != 9 || p.Weather[sim.WeatherType("clear")] != 1 {
		t.Errorf("weights: %+v", p.Weather)
	}
	if p.Influence.IntensityDelta != -0.07 || p.Influence.MovementResistance != 0.4 {
		t.Errorf("influence: %+v", p.Influence)
	}
	if p.SpawnWeight != 0.5 {
		t.Errorf("spawnWeight: %v", p.SpawnWeight)
	}
}

func TestParseClimateRejectsMissingBiome(t *testing.T) {
	if _, _, err := ParseClimate([]byte("weather:\n  clear: 1\n")); err == nil {
		t.Fatal("a climate file without 'biome' must be rejected")
	}
}

func TestLoadClimateMergesOverDefaults(t *testing.T) {
	fsys := fstest.MapFS{
		"climate/tundra.yaml": {Data: []byte(tundraYAML)},
	}
	c, err := LoadClimate(fsys, "climate")
	if err != nil {
		t.Fatal(err)
	}
	if c["tundra"].Weather[sim.WeatherType("snow")] != 9 {
		t.Error("override profile not applied")
	}
	if _, ok := c["ocean"]; !ok {
		t.Error("non-overridden default profiles must survive the merge")
	}
}

func TestLoadClimateMissingDirIsDefaults(t *testing.T) {
	c, err := LoadClimate(fstest.MapFS{}, "climate")
	if err != nil {
		t.Fatal(err)
	}
	if len(c) != len(sim.DefaultClimate()) {
		t.Error("missing dir should return pure defaults")
	}
}
