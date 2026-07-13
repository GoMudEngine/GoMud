package devtools

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// magnitudeBounds are generous sane ranges per effect type — tight enough to
// catch a fat-finger, loose enough for playtest tuning. Values are LEVEL-1 base
// magnitudes (the deepening curve scales them at ranks 2-4). Buff-id effect
// types (aura_*, on_hit_buff, on_reflect_buff) are intentionally OMITTED — their
// "value" is a buff id, not a magnitude.
var magnitudeBounds = map[string][2]float64{
	"reflect_damage":              {1, 30},
	"natural_armor":               {-25, 25},
	"magical_damage_reduction":    {0, 0.5},
	"conviction_damage_reduction": {0, 0.5},
	"dodge_modifier":              {-25, 35},
	"health_multiplier":           {-0.3, 0.4},
	"stat_multiplier":             {-0.3, 0.4},
	"spell_power":                 {-0.3, 0.6},
	"stealth_bonus":               {-30, 50},
	"movement_speed":              {-0.3, 0.3},
	"health_regen_multiplier":     {-0.3, 0.6},
	"stamina_regen_multiplier":    {-0.4, 0.4},
	"conviction_cost_multiplier":  {-0.4, 0.4},
	"stat_flat":                   {-30, 20},
}

// TestMutationMagnitudesInBounds catches accidental order-of-magnitude outliers
// in the authored per-rank base magnitudes (6e consistency guard).
func TestMutationMagnitudesInBounds(t *testing.T) {
	dir := filepath.Join(dataRoot(t), "mutations")
	files, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
	typeRe := regexp.MustCompile(`^\s*-?\s*type:\s*([a-z_]+)`)
	valRe := regexp.MustCompile(`^\s*value:\s*(-?[0-9.]+)`)
	for _, f := range files {
		body, _ := os.ReadFile(f)
		var curType string
		for _, ln := range strings.Split(string(body), "\n") {
			if m := typeRe.FindStringSubmatch(ln); m != nil {
				curType = m[1]
				continue
			}
			if m := valRe.FindStringSubmatch(ln); m != nil && curType != "" {
				v, _ := strconv.ParseFloat(m[1], 64)
				if b, ok := magnitudeBounds[curType]; ok && (v < b[0] || v > b[1]) {
					t.Errorf("%s: %s value %.3f out of sane bounds [%.2f,%.2f]",
						filepath.Base(f), curType, v, b[0], b[1])
				}
			}
		}
	}
}
