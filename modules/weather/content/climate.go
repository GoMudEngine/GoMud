package content

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/GoMudEngine/GoMud/modules/weather/sim"
	"gopkg.in/yaml.v2"
)

// climateFile mirrors the on-disk climate schema (spec §7.3). Field names are
// camelCase in the files, so explicit yaml tags are required (yaml.v2 would
// otherwise expect all-lowercase keys).
type climateFile struct {
	Biome     string             `yaml:"biome"`
	Weather   map[string]float64 `yaml:"weather"`
	Influence struct {
		IntensityDelta     float64 `yaml:"intensityDelta"`
		MoistureDelta      float64 `yaml:"moistureDelta"`
		MovementResistance float64 `yaml:"movementResistance"`
	} `yaml:"influence"`
	SpawnWeight float64 `yaml:"spawnWeight"`
}

// ParseClimate parses one climate profile file into its biome id and profile.
func ParseClimate(b []byte) (string, sim.ClimateProfile, error) {
	var cf climateFile
	if err := yaml.Unmarshal(b, &cf); err != nil {
		return "", sim.ClimateProfile{}, err
	}
	if cf.Biome == "" {
		return "", sim.ClimateProfile{}, fmt.Errorf("climate file missing required 'biome' key")
	}
	p := sim.ClimateProfile{
		Weather: make(map[sim.WeatherType]float64, len(cf.Weather)),
		Influence: sim.WeatherInfluence{
			IntensityDelta:     cf.Influence.IntensityDelta,
			MoistureDelta:      cf.Influence.MoistureDelta,
			MovementResistance: cf.Influence.MovementResistance,
		},
		SpawnWeight: cf.SpawnWeight,
	}
	for k, v := range cf.Weather {
		p.Weather[sim.WeatherType(k)] = v
	}
	return cf.Biome, p, nil
}

// LoadClimate returns sim.DefaultClimate() overlaid with every *.yaml climate
// file found under dir in fsys. A file replaces the default profile for its
// biome wholesale (omitted keys become zero values — including spawnWeight, so
// a profile that should spawn fronts must say so). A missing dir simply means
// no overrides. The first malformed file aborts with an error; the caller
// decides whether to fail soft.
func LoadClimate(fsys fs.FS, dir string) (sim.Climate, error) {
	climate := sim.DefaultClimate()
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return climate, nil // dir not shipped — pure defaults
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		b, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		if err != nil {
			return climate, fmt.Errorf("%s: %w", e.Name(), err)
		}
		biome, p, err := ParseClimate(b)
		if err != nil {
			return climate, fmt.Errorf("%s: %w", e.Name(), err)
		}
		climate[biome] = p
	}
	return climate, nil
}
