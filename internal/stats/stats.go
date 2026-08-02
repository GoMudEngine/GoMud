package stats

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

func (si *StatInfo) Recalculate() {
	si.Racial = si.Base
	si.Value = si.Racial + si.Training + si.Mods
	si.ValueAdj = si.Value
}
