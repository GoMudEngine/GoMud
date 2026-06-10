package weather

import (
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

// EmoteModeModule / EmoteModeTagOnly are the two §9.4 delivery modes.
const (
	EmoteModeModule  = "module"   // the module emits ambient lines itself
	EmoteModeTagOnly = "tag-only" // mutator tags only; the world's scripts react
)

// Config is the resolved module configuration (keys live under
// Modules.weather.* and default from files/data-overlays/config.yaml). Keys
// are flat (BuffsEnabled, not Buffs.Enabled) because plugin config lookup
// reads flattened scalar leaves.
type Config struct {
	Enabled            bool
	IncludeSecretExits bool
	RebuildGraphOnBoot bool

	Seed               uint64  // 0 = derive a stable seed from the world's zone names
	TickEveryGameHours int     // weather-simulation cadence in game hours (>= 1)
	MaxActiveFronts    int     // global front budget (>= 1)
	SpawnRateScale     float64 // multiplier on the default spawn chance
	EmoteMode          string  // EmoteModeModule | EmoteModeTagOnly
	EmoteEveryRounds   int     // ambient emote cadence in rounds (jittered ±25%, >= 5)
	BuffsEnabled       bool    // false strips buff ids from weather mutator specs
	Persist            bool    // save/restore fronts + RNG across reboots
}

// getter abstracts plugin.Config.Get for testability.
type getter func(string) any

func asBool(v any) bool { b, _ := v.(bool); return b }

func boolOr(v any, def bool) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return def
}

func intOr(v any, def int) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
			return i
		}
	}
	return def
}

func floatOr(v any, def float64) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(n), 64); err == nil {
			return f
		}
	}
	return def
}

func stringOr(v any, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}

// buildConfig resolves config from a getter, applying defaults and sanity
// clamps so a partial or hand-mangled overlay still yields a usable module.
func buildConfig(get getter) Config {
	seed := intOr(get("Seed"), 0)
	if seed < 0 {
		seed = 0 // negative would wrap via uint64(); treat as "derive from world"
	}

	c := Config{
		Enabled:            asBool(get("Enabled")),
		IncludeSecretExits: boolOr(get("IncludeSecretExits"), true),
		RebuildGraphOnBoot: asBool(get("RebuildGraphOnBoot")),

		Seed:               uint64(seed),
		TickEveryGameHours: intOr(get("TickEveryGameHours"), 1),
		MaxActiveFronts:    intOr(get("MaxActiveFronts"), 8),
		SpawnRateScale:     floatOr(get("SpawnRateScale"), 1.0),
		EmoteMode:          strings.ToLower(stringOr(get("EmoteMode"), EmoteModeModule)),
		EmoteEveryRounds:   intOr(get("EmoteEveryRounds"), 20),
		BuffsEnabled:       boolOr(get("BuffsEnabled"), true),
		Persist:            boolOr(get("Persist"), true),
	}
	if c.TickEveryGameHours < 1 {
		c.TickEveryGameHours = 1
	}
	if c.MaxActiveFronts < 1 {
		c.MaxActiveFronts = 1
	}
	if c.EmoteEveryRounds < 5 {
		c.EmoteEveryRounds = 5
	}
	if c.SpawnRateScale < 0 {
		c.SpawnRateScale = 0
	}
	if c.EmoteMode != EmoteModeModule && c.EmoteMode != EmoteModeTagOnly {
		c.EmoteMode = EmoteModeModule
	}
	return c
}

// simConfig maps module config onto the simulation's tuning knobs.
func (c Config) simConfig() sim.Config {
	sc := sim.DefaultConfig()
	sc.MaxActiveFronts = c.MaxActiveFronts
	sc.SpawnChance *= c.SpawnRateScale
	if sc.SpawnChance > 1 {
		sc.SpawnChance = 1
	}
	return sc
}

// loadConfig reads the module's live config via the plugin API.
func loadConfig(p *plugins.Plugin) Config {
	return buildConfig(func(k string) any { return p.Config.Get(k) })
}
