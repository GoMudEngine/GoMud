package items

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/util"
)

var (
	attackMessages map[ItemSubType]*WeaponAttackMessageGroup = map[ItemSubType]*WeaponAttackMessageGroup{}
)

type WeaponAttackMessageGroup struct {
	OptionId ItemSubType `yaml:"optionid"`
	Options  AttackTypes `yaml:"options"`
}

type AttackTypes map[Intensity]AttackOptions

type AttackOptions struct {
	Together TogetherMessages `yaml:"together"`
	Separate SeparateMessages `yaml:"separate"`
}

type TogetherMessages struct {
	ToAttacker MessageOptions `yaml:"toattacker"`
	ToDefender MessageOptions `yaml:"todefender"`
	ToRoom     MessageOptions `yaml:"toroom"`
}

type SeparateMessages struct {
	ToAttacker     MessageOptions `yaml:"toattacker"`
	ToDefender     MessageOptions `yaml:"todefender"`
	ToAttackerRoom MessageOptions `yaml:"toattackerroom"`
	ToDefenderRoom MessageOptions `yaml:"todefenderroom"`
}

type MessageOptions []ItemMessage

func (am ItemMessage) SetTokenValue(tokenName TokenName, tokenValue string) ItemMessage {
	return ItemMessage(strings.Replace(string(am), string(tokenName), tokenValue, -1))
}

func (mo MessageOptions) Get(seedNum ...int) ItemMessage {

	if ct := len(mo); ct > 0 {

		if len(seedNum) == 0 || seedNum[0] == 0 {
			return mo[util.Rand(ct)]
		}

		if seedNum[0] == 0 {
			return mo[0]
		}

		return mo[seedNum[0]%len(mo)]
	}

	return ItemMessage("")
}

// Presumably to ensure the datafile hasn't messed something up.
func (w *WeaponAttackMessageGroup) Id() ItemSubType {
	return w.OptionId
}

// Presumably to ensure the datafile hasn't messed something up.
func (w *WeaponAttackMessageGroup) Validate() error {

	// Make sure all important options are present.
	optionsToCheck := []Intensity{Prepare, Wait, Miss, Weak, Normal, Heavy, Critical, Fumble}
	for _, option := range optionsToCheck {
		if _, ok := w.Options[option]; !ok {
			return fmt.Errorf("missing option[`%s`] for %s", option, w.OptionId)
		}
	}

	return nil
}

func (w *WeaponAttackMessageGroup) Filepath() string {
	return fmt.Sprintf("%s.yaml", w.OptionId)
}

func GetPreAttackMessage(subType ItemSubType, messageType Intensity) AttackOptions {

	// Check whether this item subtype has any attack messages
	if attackMsgOptions, ok := attackMessages[subType]; ok {
		if attackMsgOptions, ok := attackMsgOptions.Options[messageType]; ok {
			// return a random message
			return attackMsgOptions
		}
	}
	// default to generic.
	return GetPreAttackMessage(Generic, messageType)
}

func GetAttackMessage(subType ItemSubType, pctDamage int) AttackOptions {

	var intensity Intensity
	if pctDamage >= 101 {
		intensity = Critical
	} else if pctDamage >= 75 {
		intensity = Heavy
	} else if pctDamage >= 30 {
		intensity = Normal
	} else if pctDamage >= 1 {
		intensity = Weak
	} else {
		intensity = Miss
	}

	// Check whether this item subtype has any attack messages
	if attackMsgOptions, ok := attackMessages[subType]; ok {
		if attackMsgOptions, ok := attackMsgOptions.Options[intensity]; ok {
			// return a random message
			return attackMsgOptions
		}
	}
	// default to generic.
	return GetAttackMessage(Generic, pctDamage)
}
