package content

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/GoMudEngine/GoMud/modules/weather/sim"
	"gopkg.in/yaml.v2"
)

// Table holds the ambient lines for one weather type, keyed by biome with a
// "default" fallback, split outdoor/indoor (spec §9.4). Lines are uniform
// random picks (the spec's per-line weights are an unneeded refinement for
// shipped defaults; builders wanting bias can repeat a line).
type Table struct {
	Weather string              `yaml:"weather"`
	Outdoor map[string][]string `yaml:"outdoor"`
	Indoor  map[string][]string `yaml:"indoor"`
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

// Pick selects one ambient line for (weather, biome, indoor), or "" when
// nothing matches. Fallbacks: exact biome -> "default" biome. Indoor never
// falls back to outdoor — silence beats wrong prose. roll(n) must return a
// value in [0,n); pass the engine's util.Rand (or a stub in tests) — NEVER the
// sim RNG, which must stay isolated from presentation randomness.
// An out-of-range roll result is clamped to the first line rather than panicking.
func (ts Tables) Pick(weather sim.WeatherType, biome string, indoor bool, roll func(int) int) string {
	t, ok := ts[weather]
	if !ok {
		return ""
	}
	section := t.Outdoor
	if indoor {
		section = t.Indoor
	}
	lines := section[biome]
	if len(lines) == 0 {
		lines = section["default"]
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
