package stats

import "math"

const (
	BaseModFactor         = 0.3333333334 // How much of a scaling to aply to levels before multiplying by racial stat
	NaturalGainsModFactor = 0.5          // Free stats gained per level modded by this.
)

type Statistics struct {
	Strength   StatInfo `yaml:"strength,omitempty"`   // Muscular strength (damage)
	Dexterity  StatInfo `yaml:"dexterity,omitempty"`  // Speed and agility (dodging)
	Perception StatInfo `yaml:"perception,omitempty"` // Awareness and intelligence (noticing things, memory, deduction)
	Vitality   StatInfo `yaml:"vitality,omitempty"`   // Health and stamina (health capacity)
	Willpower  StatInfo `yaml:"willpower,omitempty"`  // Mental fortitude and conviction
	Charisma   StatInfo `yaml:"charisma,omitempty"`   // Force of personality and social influence
}

// When saving to a file, we don't need to write all the properties that we calculate.
// Just keep track of "Training" because that's not calculated.
type StatInfo struct {
	Training int `yaml:"training,omitempty"` // How much it's been trained with Training Points spending
	Value    int `yaml:"-"`                  // Final calculated value
	ValueAdj int `yaml:"-"`                  // Final calculated value (Adjusted)
	Racial   int `yaml:"-"`                  // Value provided by racial benefits
	Base     int `yaml:"base,omitempty"`     // Base stat value
	Mods     int `yaml:"-"`                  // How much it's modded by equipment, spells, etc.
}

func (si *StatInfo) SetMod(mod ...int) {
	if len(mod) == 0 {
		si.Mods = 0
		return
	}
	si.Mods = 0
	for _, m := range mod {
		si.Mods += m
	}
}

func (si *StatInfo) GainsForLevel(level int) int {
	if level < 1 {
		level = 1
	}
	levelScale := float64(level-1) * BaseModFactor
	basePoints := int(levelScale * float64(si.Base))

	// every x levels we get natural gains
	freeStatPoints := int(float64(level) * NaturalGainsModFactor)

	return basePoints + freeStatPoints
}

func (si *StatInfo) Recalculate(level int) {
	si.Racial = si.GainsForLevel(level)
	si.Value = si.Racial + si.Training + si.Mods
	si.ValueAdj = si.Value
	if si.ValueAdj >= 105 {
		overage := si.ValueAdj - 100
		si.ValueAdj = 100 + int(math.Round(math.Sqrt(float64(overage))*2))
	}
}
