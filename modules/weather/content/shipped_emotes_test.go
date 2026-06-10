package content

import (
	"os"
	"testing"
	"unicode/utf8"

	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

// Validates the shipped DOGMud weather emote tables parse and obey the
// severe-types-always-audible-indoors rule.
func TestShippedDogmudEmoteTables(t *testing.T) {
	fsys := os.DirFS("../../../_datafiles/world/dogmud")
	tables, err := LoadEmotes(fsys, "weather/emotes")
	if err != nil {
		t.Fatalf("LoadEmotes: %v", err)
	}
	if len(tables) != 8 {
		t.Fatalf("expected 8 emote tables, got %d", len(tables))
	}

	for wt, table := range tables {
		if len(table.Outdoor["default"]) == 0 {
			t.Errorf("%s: outdoor default pool must not be empty", wt)
		}
		for biome, lines := range table.Outdoor {
			for _, l := range lines {
				if utf8.RuneCountInString(l) > 80 {
					t.Errorf("%s/outdoor/%s: line exceeds 80 chars: %q", wt, biome, l)
				}
			}
		}
		for biome, pool := range table.Indoor {
			for _, l := range append(append([]string{}, pool.Mild...), pool.Strong...) {
				if utf8.RuneCountInString(l) > 80 {
					t.Errorf("%s/indoor/%s: line exceeds 80 chars: %q", wt, biome, l)
				}
			}
		}
	}

	for _, severe := range []sim.WeatherType{"storm", "blizzard", "dust"} {
		pool := tables[severe].Indoor["default"]
		if len(pool.Strong) == 0 {
			t.Errorf("%s: must have non-empty strong indoor pool", severe)
		}
	}
}
