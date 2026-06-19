package sim

// WeatherInfluence is the terrain → front-dynamics half of the feedback loop:
// how a biome modifies a front passing through it each tick.
type WeatherInfluence struct {
	IntensityDelta     float64 `json:"intensityDelta"`
	MoistureDelta      float64 `json:"moistureDelta"`
	MovementResistance float64 `json:"movementResistance"` // 0..1; higher = lingers
}

// ClimateProfile is one biome's weather data: valid weather types + spawn
// weights (biome → weather), the influence it exerts on passing fronts
// (weather ← biome), and how often new fronts originate here.
type ClimateProfile struct {
	Weather     map[WeatherType]float64 `json:"weather"`
	Influence   WeatherInfluence        `json:"influence"`
	SpawnWeight float64                 `json:"spawnWeight"`

	// Track names the season cycle this biome follows (seasons package);
	// "" = no seasons for this biome. Carried as data — Step ignores it.
	Track string `json:"track,omitempty"`
}

// Climate maps biome id -> profile. Use For() to resolve with default fallback.
type Climate map[string]ClimateProfile

// For returns the profile for a biome, or the "default" profile if the biome
// has none. If even "default" is absent, returns a zero profile.
func (c Climate) For(biome string) ClimateProfile {
	if p, ok := c[biome]; ok {
		return p
	}
	return c["default"]
}

// Config holds simulation tuning knobs.
type Config struct {
	MaxActiveFronts int     // global front budget
	SpawnChance     float64 // per-tick chance to spawn when under budget (0..1)
	HistoryLen      int     // bounded front path length (no-backtrack window)
	FrontHardAge    int     // hard age cap; older fronts die regardless

	// Area coverage: a front projects onto zones within MaxFrontRadius hops of
	// its center; the intensity it projects falls off by CoverageFalloff per hop,
	// and a zone is only covered while the projected value stays >= MinProjected.
	// Net effect: stronger fronts naturally cover a larger area.
	CoverageFalloff float64 // 0..1 multiplier per hop (e.g. 0.5 = halve each hop)
	MinProjected    float64 // minimum projected intensity for a zone to be covered
	MaxFrontRadius  int     // hard cap on coverage radius (hops)
}

// DefaultConfig returns sensible simulation defaults.
func DefaultConfig() Config {
	return Config{
		MaxActiveFronts: 8,
		SpawnChance:     0.25,
		HistoryLen:      4,
		FrontHardAge:    48,
		CoverageFalloff: 0.5,
		MinProjected:    0.15,
		MaxFrontRadius:  2,
	}
}

// DefaultClimate returns the built-in climate for the standard biomes. Builders
// can replace/extend this (file-based overrides land in M3). Influence sign
// convention: positive IntensityDelta feeds a front, negative saps it.
func DefaultClimate() Climate {
	return Climate{
		"default": {
			Weather:     map[WeatherType]float64{"clear": 6, "overcast": 3, "rain": 2, "fog": 1},
			Influence:   WeatherInfluence{IntensityDelta: -0.02, MoistureDelta: 0, MovementResistance: 0.1},
			SpawnWeight: 1.0,
		},
		"plains": {
			Weather:     map[WeatherType]float64{"clear": 5, "overcast": 3, "rain": 3, "storm": 2},
			Influence:   WeatherInfluence{IntensityDelta: 0.02, MoistureDelta: 0, MovementResistance: 0.05},
			SpawnWeight: 1.2,
			Track:       "temperate",
		},
		"forest": {
			Weather:     map[WeatherType]float64{"clear": 4, "overcast": 4, "rain": 4, "fog": 3},
			Influence:   WeatherInfluence{IntensityDelta: -0.01, MoistureDelta: 0.02, MovementResistance: 0.15},
			SpawnWeight: 1.0,
			Track:       "temperate",
		},
		"mountain": {
			Weather:     map[WeatherType]float64{"overcast": 4, "snow": 4, "storm": 2, "fog": 3},
			Influence:   WeatherInfluence{IntensityDelta: -0.15, MoistureDelta: -0.10, MovementResistance: 0.5},
			SpawnWeight: 0.8,
			Track:       "temperate",
		},
		"desert": {
			Weather:     map[WeatherType]float64{"clear": 7, "dust": 3, "heatwave": 2},
			Influence:   WeatherInfluence{IntensityDelta: -0.05, MoistureDelta: -0.08, MovementResistance: 0.1},
			SpawnWeight: 0.7,
			// desert: unbound (no seasonal cycle) — matches the standalone.
		},
		"tundra": {
			Weather:     map[WeatherType]float64{"clear": 5, "overcast": 4, "snow": 6, "blizzard": 2, "fog": 2},
			Influence:   WeatherInfluence{IntensityDelta: -0.05, MoistureDelta: -0.02, MovementResistance: 0.2},
			SpawnWeight: 1.0,
			Track:       "temperate",
		},
		"swamp": {
			Weather:     map[WeatherType]float64{"overcast": 4, "rain": 5, "fog": 5, "storm": 2},
			Influence:   WeatherInfluence{IntensityDelta: 0.01, MoistureDelta: 0.05, MovementResistance: 0.2},
			SpawnWeight: 1.1,
			Track:       "temperate",
		},
		"ocean": {
			Weather:     map[WeatherType]float64{"clear": 3, "overcast": 4, "rain": 4, "storm": 4, "fog": 2},
			Influence:   WeatherInfluence{IntensityDelta: 0.06, MoistureDelta: 0.08, MovementResistance: 0.02},
			SpawnWeight: 1.5,
			Track:       "temperate",
		},

		// --- Stock-world / DOGMud biome ids. These previously fell through to
		// the "default" profile (a coverage gap): DOGMud's world uses these ids
		// (mountains/cliffs/snow/shore/water/farmland/land/road/city), not the
		// archetype keys above. Bound to "temperate" except indoor/desert.
		"mountains": { // = mountain archetype
			Weather:     map[WeatherType]float64{"overcast": 4, "snow": 4, "storm": 2, "fog": 3},
			Influence:   WeatherInfluence{IntensityDelta: -0.15, MoistureDelta: -0.10, MovementResistance: 0.5},
			SpawnWeight: 0.8,
			Track:       "temperate",
		},
		"cliffs": { // exposed high ground: mountain-lite, windier storms
			Weather:     map[WeatherType]float64{"clear": 3, "overcast": 4, "storm": 3, "fog": 3},
			Influence:   WeatherInfluence{IntensityDelta: -0.08, MoistureDelta: -0.04, MovementResistance: 0.3},
			SpawnWeight: 0.9,
			Track:       "temperate",
		},
		"snow": { // = tundra archetype
			Weather:     map[WeatherType]float64{"clear": 5, "overcast": 4, "snow": 6, "blizzard": 2, "fog": 2},
			Influence:   WeatherInfluence{IntensityDelta: -0.05, MoistureDelta: -0.02, MovementResistance: 0.2},
			SpawnWeight: 1.0,
			Track:       "temperate",
		},
		"shore": { // coastal: ocean-fed but calmer
			Weather:     map[WeatherType]float64{"clear": 4, "overcast": 4, "rain": 4, "storm": 3, "fog": 3},
			Influence:   WeatherInfluence{IntensityDelta: 0.04, MoistureDelta: 0.06, MovementResistance: 0.05},
			SpawnWeight: 1.3,
			Track:       "temperate",
		},
		"water": { // = ocean archetype
			Weather:     map[WeatherType]float64{"clear": 3, "overcast": 4, "rain": 4, "storm": 4, "fog": 2},
			Influence:   WeatherInfluence{IntensityDelta: 0.06, MoistureDelta: 0.08, MovementResistance: 0.02},
			SpawnWeight: 1.5,
			Track:       "temperate",
		},
		"farmland": { // = plains archetype
			Weather:     map[WeatherType]float64{"clear": 5, "overcast": 3, "rain": 3, "storm": 2},
			Influence:   WeatherInfluence{IntensityDelta: 0.02, MoistureDelta: 0, MovementResistance: 0.05},
			SpawnWeight: 1.2,
			Track:       "temperate",
		},
		"land": { // generic open ground = plains archetype
			Weather:     map[WeatherType]float64{"clear": 5, "overcast": 3, "rain": 3, "storm": 2},
			Influence:   WeatherInfluence{IntensityDelta: 0.02, MoistureDelta: 0, MovementResistance: 0.05},
			SpawnWeight: 1.2,
			Track:       "temperate",
		},
		"road": { // travelled open ground: plains-lite, low spawn pressure
			Weather:     map[WeatherType]float64{"clear": 6, "overcast": 3, "rain": 2, "fog": 2},
			Influence:   WeatherInfluence{IntensityDelta: 0, MoistureDelta: 0, MovementResistance: 0.05},
			SpawnWeight: 0.8,
			Track:       "temperate",
		},
		"city": { // urban: mild, fog-prone, storms dampened
			Weather:     map[WeatherType]float64{"clear": 5, "overcast": 4, "rain": 3, "fog": 3, "storm": 1},
			Influence:   WeatherInfluence{IntensityDelta: -0.03, MoistureDelta: -0.02, MovementResistance: 0.15},
			SpawnWeight: 0.9,
			Track:       "temperate",
		},
		// "fort" is indoor:true in DOGMud (biomes.go) — left UNBOUND so it
		// reports no outdoor seasonal ambience (the standalone bound it
		// temperate, but its fort was outdoor).
		"fort": {
			Weather:     map[WeatherType]float64{"clear": 5, "overcast": 4, "rain": 3, "fog": 3, "storm": 1},
			Influence:   WeatherInfluence{IntensityDelta: -0.03, MoistureDelta: -0.02, MovementResistance: 0.15},
			SpawnWeight: 0.9,
		},
		"jungle": { // dense tropical canopy — monsoon-cycled (spec §3.2)
			Weather:     map[WeatherType]float64{"rain": 5, "storm": 3, "fog": 4, "overcast": 3, "clear": 2},
			Influence:   WeatherInfluence{IntensityDelta: 0.03, MoistureDelta: 0.06, MovementResistance: 0.25},
			SpawnWeight: 1.3,
			Track:       "monsoon",
		},
	}
}
