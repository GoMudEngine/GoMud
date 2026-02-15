package items

import (
	"fmt"
)

var (
	defenseMessages map[DefenseType]*DefenseMessageGroup = map[DefenseType]*DefenseMessageGroup{}
)

// DefenseType identifies the type of defensive action
type DefenseType string

const (
	DefenseDodge DefenseType = "dodge"
	DefenseParry DefenseType = "parry"
	DefenseBlock DefenseType = "block"
)

type DefenseMessageGroup struct {
	OptionId DefenseType      `yaml:"optionid"`
	Options  DefenseIntensity `yaml:"options"`
}

type DefenseIntensity map[Intensity]DefenseOptions

type DefenseOptions struct {
	Together DefenseTogetherMessages `yaml:"together"`
}

type DefenseTogetherMessages struct {
	ToDefender MessageOptions `yaml:"todefender"`
	ToAttacker MessageOptions `yaml:"toattacker"`
	ToRoom     MessageOptions `yaml:"toroom"`
}

// Presumably to ensure the datafile hasn't messed something up.
func (d *DefenseMessageGroup) Id() DefenseType {
	return d.OptionId
}

// Presumably to ensure the datafile hasn't messed something up.
func (d *DefenseMessageGroup) Validate() error {

	// Make sure all important options are present.
	optionsToCheck := []Intensity{Weak, Normal, Heavy}
	for _, option := range optionsToCheck {
		if _, ok := d.Options[option]; !ok {
			return fmt.Errorf("missing option[`%s`] for %s", option, d.OptionId)
		}
	}

	return nil
}

func (d *DefenseMessageGroup) Filepath() string {
	return fmt.Sprintf("%s.yaml", d.OptionId)
}

// GetDefenseMessage returns the appropriate defense message based on defense type and intensity
func GetDefenseMessage(defenseType DefenseType, zScore float64) DefenseOptions {

	var intensity Intensity
	// Map z-score to intensity:
	// High z-score = easy defense (opponent barely came close)
	// Low z-score = narrow defense (opponent almost hit)
	if zScore >= 2.0 {
		intensity = Heavy // Easy/decisive defense
	} else if zScore >= 0.5 {
		intensity = Normal // Standard defense
	} else {
		intensity = Weak // Narrow/close defense
	}

	// Check whether this defense type has any messages
	if defenseMsgOptions, ok := defenseMessages[defenseType]; ok {
		if defenseMsgOptions, ok := defenseMsgOptions.Options[intensity]; ok {
			return defenseMsgOptions
		}
	}

	// Return empty if not found (caller should fallback to generic messages)
	return DefenseOptions{}
}
