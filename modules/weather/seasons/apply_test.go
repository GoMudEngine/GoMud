package seasons

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

func testClimate() sim.Climate {
	return sim.Climate{
		"tundra": {
			Weather:     map[sim.WeatherType]float64{"snow": 2, "clear": 4, "heatwave": 1},
			Influence:   sim.WeatherInfluence{IntensityDelta: -0.05, MovementResistance: 0.2},
			SpawnWeight: 1.0,
			Track:       "temperate",
		},
		"desert": { // unbound — must pass through untouched
			Weather:     map[sim.WeatherType]float64{"dust": 3},
			SpawnWeight: 0.7,
		},
	}
}

func TestEffectiveClimateFullSeason(t *testing.T) {
	tracks := loadOne(t, temperateYAML)
	// Day 200 = mid-summer, blend 1.0; summer multiplies storm only (x1.4),
	// snow/heatwave/clear untouched by summer.
	eff := EffectiveClimate(testClimate(), tracks, CalendarPos{DayOfYear: 200, DaysPerYear: 365})
	if eff["tundra"].Weather["snow"] != 2 || eff["tundra"].Weather["clear"] != 4 {
		t.Errorf("summer should not change snow/clear: %+v", eff["tundra"].Weather)
	}
	// Mid-winter (day 20, blend 1.0): snow x3, heatwave x0.
	eff = EffectiveClimate(testClimate(), tracks, CalendarPos{DayOfYear: 20, DaysPerYear: 365})
	if eff["tundra"].Weather["snow"] != 6 {
		t.Errorf("winter snow: want 6, got %v", eff["tundra"].Weather["snow"])
	}
	if eff["tundra"].Weather["heatwave"] != 0 {
		t.Errorf("winter heatwave: want 0, got %v", eff["tundra"].Weather["heatwave"])
	}
	if got := eff["tundra"].SpawnWeight; math.Abs(got-0.9) > 1e-9 {
		t.Errorf("winter spawn weight: want 0.9, got %v", got)
	}
	if got := eff["tundra"].Influence.IntensityDelta; math.Abs(got-(-0.07)) > 1e-9 {
		t.Errorf("winter influence: want -0.07 (-0.05 biome + -0.02 season), got %v", got)
	}
}

func TestEffectiveClimateBlends(t *testing.T) {
	tracks := loadOne(t, temperateYAML)
	// Day 338: winter, blend 0.5. Autumn (prev) has no snow multiplier (=1.0),
	// winter has 3.0 -> effective 2.0 -> weight 2*2 = 4.
	eff := EffectiveClimate(testClimate(), tracks, CalendarPos{DayOfYear: 338, DaysPerYear: 365})
	if got := eff["tundra"].Weather["snow"]; math.Abs(got-4) > 1e-9 {
		t.Errorf("blended snow: want 4, got %v", got)
	}
}

func TestEffectiveClimateLeavesUnboundAndBase(t *testing.T) {
	base := testClimate()
	tracks := loadOne(t, temperateYAML)
	eff := EffectiveClimate(base, tracks, CalendarPos{DayOfYear: 20, DaysPerYear: 365})
	// Unbound biome untouched.
	if eff["desert"].Weather["dust"] != 3 || eff["desert"].SpawnWeight != 0.7 {
		t.Errorf("desert must pass through: %+v", eff["desert"])
	}
	// Base climate not mutated (deep copy).
	if base["tundra"].Weather["snow"] != 2 {
		t.Errorf("base climate mutated: %v", base["tundra"].Weather["snow"])
	}
	// Unknown track behaves as unbound.
	base2 := sim.Climate{"x": {Weather: map[sim.WeatherType]float64{"rain": 1}, Track: "nope"}}
	eff2 := EffectiveClimate(base2, tracks, CalendarPos{DayOfYear: 20, DaysPerYear: 365})
	if eff2["x"].Weather["rain"] != 1 {
		t.Errorf("unknown track must pass through: %+v", eff2["x"])
	}
}

const shatteringYAML = `track: stillness
seasons:
  - name: calm
    months: [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
  - name: shattering
    months: [11, 12]
    transitionDays: 2
    baseWeightScale: 0.0
    weatherWeightAdditions: { glassrain: 8 }
`

func TestEffectiveClimateEsotericSeason(t *testing.T) {
	tracks := loadOne(t, shatteringYAML)
	base := sim.Climate{"plateau": {
		Weather:     map[sim.WeatherType]float64{"clear": 5, "rain": 2},
		SpawnWeight: 1.0,
		Track:       "stillness",
	}}
	// Month 11 starts at day 305; transitionDays=2 => day 307 is fully shattering.
	eff := EffectiveClimate(base, tracks, CalendarPos{DayOfYear: 307, DaysPerYear: 365})
	w := eff["plateau"].Weather
	if w["clear"] != 0 || w["rain"] != 0 {
		t.Errorf("baseWeightScale 0 must suppress normal weather: %+v", w)
	}
	if w["glassrain"] != 8 {
		t.Errorf("addition must introduce the new type: %+v", w)
	}
	// Mid-window (day 306, blend 0.5): base halved, addition half-strength.
	eff = EffectiveClimate(base, tracks, CalendarPos{DayOfYear: 306, DaysPerYear: 365})
	w = eff["plateau"].Weather
	if math.Abs(w["clear"]-2.5) > 1e-9 || math.Abs(w["glassrain"]-4) > 1e-9 {
		t.Errorf("esoteric blend wrong: %+v", w)
	}
	// Back in calm (day 100): no glassrain entry leaks.
	eff = EffectiveClimate(base, tracks, CalendarPos{DayOfYear: 100, DaysPerYear: 365})
	if v, ok := eff["plateau"].Weather["glassrain"]; ok && v != 0 {
		t.Errorf("glassrain must not persist outside its season: %v", v)
	}
}

func TestEffectiveClimateAdditionsInBothSeasons(t *testing.T) {
	yaml := `track: ashen
seasons:
  - name: smolder
    months: [1, 2, 3, 4, 5, 6]
    weatherWeightAdditions: { ashfall: 2 }
  - name: eruption
    months: [7, 8, 9, 10, 11, 12]
    transitionDays: 4
    weatherWeightAdditions: { ashfall: 10 }
`
	tracks := loadOne(t, yaml)
	base := sim.Climate{"slopes": {
		Weather: map[sim.WeatherType]float64{"clear": 5},
		Track:   "ashen",
	}}
	// Month 7 starts day 183; day 185 -> daysInto 2 -> blend 0.5:
	// ashfall = lerp(2, 10, 0.5) = 6 (dedup branch: key in BOTH seasons' additions).
	eff := EffectiveClimate(base, tracks, CalendarPos{DayOfYear: 185, DaysPerYear: 365})
	if got := eff["slopes"].Weather["ashfall"]; math.Abs(got-6) > 1e-9 {
		t.Errorf("dual addition mid-window: want 6, got %v", got)
	}
	// And the down-ramp: entering smolder (day 1, blend 0 with no window ->
	// smolder has transitionDays 0 => blend 1 immediately => ashfall 2).
	eff = EffectiveClimate(base, tracks, CalendarPos{DayOfYear: 1, DaysPerYear: 365})
	if got := eff["slopes"].Weather["ashfall"]; math.Abs(got-2) > 1e-9 {
		t.Errorf("post-window addition: want 2, got %v", got)
	}
}

func TestZoneSeasons(t *testing.T) {
	g := &sim.Graph{Nodes: map[string]sim.ZoneNode{
		"Frost": {Zone: "Frost", Biome: "tundra"},
		"Dune":  {Zone: "Dune", Biome: "desert"},
	}}
	zs := ZoneSeasons(g, testClimate(), loadOne(t, temperateYAML), CalendarPos{DayOfYear: 20, DaysPerYear: 365})
	got, ok := zs["Frost"]
	if !ok || got.Track != "temperate" || got.Season != "winter" || got.Blend != 1.0 {
		t.Errorf("Frost: %+v ok=%v", got, ok)
	}
	if _, ok := zs["Dune"]; ok {
		t.Error("unbound zone must be absent from the map")
	}
}
