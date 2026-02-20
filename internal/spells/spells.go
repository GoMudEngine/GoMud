package spells

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

type SpellType string

type SpellData struct {
	SpellId     string     `yaml:"spellid,omitempty"`
	Name        string     `yaml:"name,omitempty"`
	Description string     `yaml:"description,omitempty"`
	Type        SpellType  `yaml:"type,omitempty"`
	Schools     []string   `yaml:"schools,omitempty"` // Can have multiple school tags
	Cost        int        `yaml:"cost,omitempty"`    // Conviction cost
	HealthCost  int        `yaml:"healthcost,omitempty"` // Optional Health cost for life-force magic
	WaitRounds  int        `yaml:"waitrounds,omitempty"`
	Difficulty  int        `yaml:"difficulty,omitempty"` // Augments final success chance by this %
	PrimaryStat         string `yaml:"primarystat,omitempty"`          // Stat used for spell rolls and progression
	BaseFolds           int    `yaml:"base_folds,omitempty"`           // 0 = default to 4
	TargetDefenseType   string `yaml:"target_defense_type,omitempty"`  // "physical", "mental", "" = none
	ComponentTag        string `yaml:"component_tag,omitempty"`        // Required item component tag (e.g. "stone")
	EffectType          string `yaml:"effect_type,omitempty"`          // "damage"|"heal"|"buff"|"tame"|"shield"
	EffectMagnitude     int    `yaml:"effect_magnitude,omitempty"`     // Base damage/heal amount
	BuffIds             []int  `yaml:"buff_ids,omitempty"`             // Buff IDs to apply (for "buff" effect type)
}

const (
	WaitRoundsDefault = 3

	Neutral    SpellType = "neutral"    // Neutral, no expected actor target, use on
	HarmSingle SpellType = "harmsingle" // Harmful, defaults to current aggro - magic missile etc
	HarmMulti  SpellType = "harmmulti"  // Harmful, defaults to all aggro mobs - chain lightning etc
	HelpSingle SpellType = "helpsingle" // Helpful, defaults on self - heal etc
	HelpMulti  SpellType = "helpmulti"  // Helpful, defaults on party - mass heal etc
	HarmArea   SpellType = "harmarea"   // Hits everyone in the room, even if hidden or friendly
	HelpArea   SpellType = "helparea"   // Hits everyone in the room, even if hidden

	// DOG Spell Schools
	SchoolElemental   = "elemental"   // Fire, ice, lightning, earth, wind - offensive elemental magic
	SchoolEnhancement = "enhancement" // Buffs, shields, enchantments - augmentation magic
	SchoolMental      = "mental"      // Illusions, charms, telepathy - mind-affecting magic (Psionics skill)
	SchoolVital       = "vital"       // Healing, curing, life/death manipulation - vital force magic
)

var (
	allSpells = map[string]*SpellData{}
)

func (s SpellType) HelpOrHarmString() string {
	switch s {
	case Neutral:
		return `Neutral`
	case HelpSingle, HelpMulti, HelpArea:
		return `Helpful`
	case HarmSingle, HarmMulti, HarmArea:
		return `Harmful`
	}
	return `Unknown`
}

func (s SpellType) TargetTypeString(short ...bool) string {
	// Return a short version
	if len(short) > 0 && short[0] {
		switch s {
		case Neutral:
			return `Unknown`
		case HelpSingle, HarmSingle:
			return `Single`
		case HelpMulti, HarmMulti:
			return `Group`
		case HelpArea, HarmArea:
			return `Area`
		}
		return `Unknown`
	}
	// Regular handling
	switch s {
	case Neutral:
		return `Unknown`
	case HelpSingle:
		return `Single Target`
	case HarmSingle:
		return `Single Target`
	case HelpMulti:
		return `Group Target`
	case HarmMulti:
		return `Group Target`
	case HelpArea, HarmArea:
		return `Area Target`
	}
	return `Unknown`
}

// Finds a match for a spell by name or id
func FindSpell(spellName string) string {
	if sd, ok := allSpells[spellName]; ok {
		return sd.SpellId
	}
	for _, spellInfo := range allSpells {
		if strings.ToLower(spellInfo.Name) == spellName {
			return spellInfo.SpellId
		}
	}
	return ``
}

func GetSpell(spellId string) *SpellData {
	if sd, ok := allSpells[spellId]; ok {
		return sd
	}
	return nil
}

func FindSpellByName(spellName string) *SpellData {

	var closestMatch *SpellData = nil

	spellName = strings.ToLower(spellName)
	for _, spellData := range allSpells {

		testName := strings.ToLower(spellData.Name)

		if testName == spellName {
			return spellData
		}

		if closestMatch == nil && strings.HasPrefix(strings.ToLower(spellData.Name), spellName) {
			closestMatch = spellData
		}

	}
	return closestMatch
}

func GetAllSpells() map[string]*SpellData {
	retSpellBook := make(map[string]*SpellData)
	for k, v := range allSpells {
		retSpellBook[k] = v
	}
	return retSpellBook
}

func (s *SpellData) Id() string {
	return s.SpellId
}

// SpellData implements the Filepath method from the Loadable interface.
func (s *SpellData) Filepath() string {
	return util.FilePath(fmt.Sprintf("%s.yaml", s.SpellId))
}

func (s *SpellData) Validate() error {

	if s.Difficulty < 0 {
		s.Difficulty = 0
	} else if s.Difficulty > 100 {
		s.Difficulty = 100
	}

	return nil
}

func (s *SpellData) GetDifficulty() int {
	return s.Difficulty
}

// GetSchoolsString returns a comma-separated string of spell schools
func (s *SpellData) GetSchoolsString() string {
	if len(s.Schools) == 0 {
		return "Unknown"
	}
	return strings.Join(s.Schools, ", ")
}

// GetTotalConvictionCost returns the conviction cost, optionally scaled by a multiplier
func (s *SpellData) GetTotalConvictionCost(multiplier float64) int {
	if multiplier <= 0 {
		multiplier = 1.0
	}
	return int(float64(s.Cost) * multiplier)
}

// GetTotalHealthCost returns the health cost, optionally scaled by a multiplier
func (s *SpellData) GetTotalHealthCost(multiplier float64) int {
	if multiplier <= 0 {
		multiplier = 1.0
	}
	return int(float64(s.HealthCost) * multiplier)
}

func (s *SpellData) GetScript() string {

	scriptPath := s.GetScriptPath()

	// Load the script into a string
	if _, err := os.Stat(scriptPath); err == nil {
		if bytes, err := os.ReadFile(scriptPath); err == nil {
			return string(bytes)
		}
	}

	return ``
}

func (s *SpellData) GetScriptPath() string {
	// Load any script for the room
	return strings.Replace(string(configs.GetFilePathsConfig().DataFiles)+`/spells/`+s.Filepath(), `.yaml`, `.js`, 1)
}

func LoadSpellFiles() {

	start := time.Now()

	tmpAllSpells, err := fileloader.LoadAllFlatFiles[string, *SpellData](string(configs.GetFilePathsConfig().DataFiles) + `/spells`)
	if err != nil {
		panic(err)
	}

	allSpells = tmpAllSpells

	mudlog.Info("spells.loadAllSpells()", "loadedCount", len(allSpells), "Time Taken", time.Since(start))

}
