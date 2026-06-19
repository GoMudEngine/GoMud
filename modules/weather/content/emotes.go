package content

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/GoMudEngine/GoMud/modules/weather/sim"
	"gopkg.in/yaml.v2"
)

// StrongFeltThreshold is the felt-intensity at which weather becomes
// perceptible indoors (drumming on roofs, wind in the eaves). Below it,
// indoor rooms get the mild pool — usually empty, i.e. silence.
const StrongFeltThreshold = 0.5

// IndoorPool holds intensity-banded indoor lines for one biome key.
// Mild plays below StrongFeltThreshold (usually empty: light weather
// doesn't register through walls); Strong plays at/above it.
type IndoorPool struct {
	Mild   []string `yaml:"mild"`
	Strong []string `yaml:"strong"`
}

// TableSection is one outdoor/indoor pair of biome-keyed lines. Indoor is
// felt-banded (IndoorPool) to match the base section's schema. Used for
// per-season weather variants and for seasonal-ambience tables.
type TableSection struct {
	Outdoor map[string][]string   `yaml:"outdoor"`
	Indoor  map[string]IndoorPool `yaml:"indoor"`
}

// Table holds the ambient lines for one weather type, keyed by biome with a
// "default" fallback, split outdoor/indoor (spec §9.4). Outdoor lines are
// uniform random picks; indoor lines are intensity-banded (see IndoorPool).
// The spec's per-line weights are an unneeded refinement for shipped defaults;
// builders wanting bias can repeat a line.
type Table struct {
	Weather string                `yaml:"weather"`
	Outdoor map[string][]string   `yaml:"outdoor"`
	Indoor  map[string]IndoorPool `yaml:"indoor"`
	// Seasonal holds optional per-season variants, keyed by season NAME
	// (matching across tracks by design — "winter" is temperate's winter).
	// Missing seasons/sections fall through to the base lines (spec §6).
	Seasonal map[string]TableSection `yaml:"seasonal"`
}

// Tables maps weather type -> emote table.
type Tables map[sim.WeatherType]Table

// ParseEmoteTable parses one emote table file.
func ParseEmoteTable(b []byte) (Table, error) {
	var t Table
	if err := yaml.Unmarshal(b, &t); err != nil {
		return Table{}, err
	}
	if t.Weather == "" {
		return Table{}, fmt.Errorf("emote table missing required 'weather' key")
	}
	return t, nil
}

// LoadEmotes loads every *.yaml emote table under dir in fsys, keyed by the
// table's weather type. A missing dir yields empty tables (silence). The first
// malformed file aborts with an error; the caller decides whether to fail soft.
// On duplicate weather keys, the later file in sorted filename order wins.
func LoadEmotes(fsys fs.FS, dir string) (Tables, error) {
	tables := Tables{}
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return tables, nil
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		b, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		if err != nil {
			return tables, fmt.Errorf("%s: %w", e.Name(), err)
		}
		t, err := ParseEmoteTable(b)
		if err != nil {
			return tables, fmt.Errorf("%s: %w", e.Name(), err)
		}
		tables[sim.WeatherType(t.Weather)] = t
	}
	return tables, nil
}

// Pick selects one ambient line for (weather, biome, indoor, felt, season),
// or "" when nothing matches. Lookup order: the season's variant section
// (when season != "" and a variant exists) -> the base section; within a
// section, exact biome -> "default" biome. season "" skips the variant layer
// (seasons off / unbound zone). Indoor never falls back to outdoor — silence
// beats wrong prose — and is felt-banded: felt < StrongFeltThreshold picks
// Mild (usually empty), otherwise Strong. roll(n) must return a value in
// [0,n); pass util.Rand — NEVER the sim RNG. Out-of-range roll results clamp
// to the first line.
func (ts Tables) Pick(weather sim.WeatherType, biome string, indoor bool, felt float64, season string, roll func(int) int) string {
	t, ok := ts[weather]
	if !ok {
		return ""
	}

	var lines []string
	if season != "" {
		if v, ok := t.Seasonal[season]; ok {
			lines = bandedSectionLines(v.Outdoor, v.Indoor, biome, indoor, felt)
		}
	}
	if len(lines) == 0 {
		lines = bandedSectionLines(t.Outdoor, t.Indoor, biome, indoor, felt)
	}

	if len(lines) == 0 {
		return ""
	}
	i := roll(len(lines))
	if i < 0 || i >= len(lines) {
		i = 0
	}
	return lines[i]
}

// bandedSectionLines resolves biome -> "default" within one outdoor/indoor
// pair. Outdoor is a flat list; indoor is felt-banded (Mild below
// StrongFeltThreshold, else Strong). Indoor never falls back to outdoor.
func bandedSectionLines(outdoor map[string][]string, indoor map[string]IndoorPool, biome string, useIndoor bool, felt float64) []string {
	if useIndoor {
		pool, ok := indoor[biome]
		if !ok || (len(pool.Mild) == 0 && len(pool.Strong) == 0) {
			pool = indoor["default"]
		}
		if felt >= StrongFeltThreshold {
			return pool.Strong
		}
		return pool.Mild
	}
	lines := outdoor[biome]
	if len(lines) == 0 {
		lines = outdoor["default"]
	}
	return lines
}

// SeasonalKey identifies one (track, season) ambience table.
type SeasonalKey struct{ Track, Season string }

// SeasonalTables holds the seasonal-ambience emote tables — the persistent
// voice of a season in CALM weather (the weather tables' Seasonal variants
// cover weathered moments). Loaded from weather/emotes/seasons/.
type SeasonalTables map[SeasonalKey]TableSection

// seasonalEmoteFile mirrors the on-disk schema.
type seasonalEmoteFile struct {
	Track   string                `yaml:"track"`
	Season  string                `yaml:"season"`
	Outdoor map[string][]string   `yaml:"outdoor"`
	Indoor  map[string]IndoorPool `yaml:"indoor"`
}

// LoadSeasonalEmotes loads every *.yaml under dir, keyed by (track, season).
// Missing dir = empty tables; the first malformed file aborts with an error
// (caller fails soft). Requires both 'track' and 'season' keys. On duplicate
// (track, season) keys, the later file in sorted filename order wins.
func LoadSeasonalEmotes(fsys fs.FS, dir string) (SeasonalTables, error) {
	out := SeasonalTables{}
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return out, nil
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		b, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		if err != nil {
			return out, fmt.Errorf("%s: %w", e.Name(), err)
		}
		var f seasonalEmoteFile
		if err := yaml.Unmarshal(b, &f); err != nil {
			return out, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if f.Track == "" || f.Season == "" {
			return out, fmt.Errorf("%s: missing required 'track' or 'season' key", e.Name())
		}
		out[SeasonalKey{f.Track, f.Season}] = TableSection{Outdoor: f.Outdoor, Indoor: f.Indoor}
	}
	return out, nil
}

// Pick selects one seasonal-ambience line for the zone's exact (track,
// season); "" when no table or no matching lines. Same biome/indoor banding
// and roll contract as the weather tables.
func (st SeasonalTables) Pick(track, season, biome string, indoor bool, felt float64, roll func(int) int) string {
	sec, ok := st[SeasonalKey{track, season}]
	if !ok {
		return ""
	}
	lines := bandedSectionLines(sec.Outdoor, sec.Indoor, biome, indoor, felt)
	if len(lines) == 0 {
		return ""
	}
	i := roll(len(lines))
	if i < 0 || i >= len(lines) {
		i = 0
	}
	return lines[i]
}
