package species

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/stats"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

type Size string

var (
	allSpecies map[int]*Species = map[int]*Species{}
)

const (
	Small  Size = "small"  // Something like a mouse, dog
	Medium Size = "medium" // Something like a human
	Large  Size = "large"  // Something like a troll, ogre, dragon, kraken, or leviathan (or bigger).
)

type Species struct {
	SpeciesId        int              `yaml:"speciesid"`
	Name             string
	Description      string
	DefaultAlignment int8
	BuffIds          []int // Permabuffs this species always has
	Size             Size
	TNLScale         float32
	UnarmedName      string
	Tameable         bool
	Damage           items.Damage
	Selectable       bool
	AngryCommands    []string         // randomly chosen to queue when they are angry/entering combat.
	KnowsFirstAid    bool             // Whether they can apply aid to other players.
	Stats            stats.Statistics // Base stats for this species.
	DisabledSlots    []string         `yaml:"disabledslots,omitempty"`
}

func GetAllSpecies() []Species {
	ret := []Species{}
	for _, s := range allSpecies {
		ret = append(ret, *s)
	}
	return ret
}

func GetSpecies(speciesId int) *Species {
	return allSpecies[speciesId]
}

func FindSpecies(name string) (Species, bool) {

	name = strings.ToLower(name)

	closeMatch := -1
	for idx, s := range allSpecies {
		testName := strings.ToLower(s.Name)
		if strings.HasPrefix(testName, name) {
			return *s, true
		} else if strings.Contains(testName, name) {
			closeMatch = idx
		}
	}
	// close matches
	if closeMatch > -1 {
		return *allSpecies[closeMatch], true
	}

	return Species{}, false
}

func (s *Species) Id() int {
	return s.SpeciesId
}

func (s *Species) Validate() error {
	if s.Name == "" {
		return errors.New("species has no name")
	}
	if s.Description == "" {
		return errors.New("species has no description")
	}
	if s.Size == "" {
		return errors.New("species has no size")
	}
	s.Size = Size(strings.ToLower(string(s.Size))) // Sometimes a mismatching CaSe value is provided.

	// Recalculate stats, based on level one because this is actually the baseline for the species
	s.Stats.Strength.Recalculate(1)
	s.Stats.Dexterity.Recalculate(1)
	s.Stats.Perception.Recalculate(1)
	s.Stats.Vitality.Recalculate(1)
	s.Stats.Willpower.Recalculate(1)
	s.Stats.Charisma.Recalculate(1)

	if s.Damage.Attacks < 1 && s.Damage.DiceCount > 0 && s.Damage.SideCount > 0 {
		s.Damage.Attacks = 1
	}

	// If a diceroll was specified, absorb that into the damage struct
	if s.Damage.DiceRoll != `` {
		s.Damage.InitDiceRoll(s.Damage.DiceRoll)
		s.Damage.FormatDiceRoll()
	}

	// Ensure TNLScale is never 0, default to 1.0
	if s.TNLScale == 0 {
		s.TNLScale = 1.0
	}

	return nil
}

func (s Species) GetEnabledSlots() []string {

	ret := []string{}
	slots := []string{
		string(items.Weapon),
		string(items.Offhand),
		string(items.Head),
		string(items.Neck),
		string(items.Body),
		string(items.Belt),
		string(items.Gloves),
		string(items.Ring),
		string(items.Legs),
		string(items.Feet),
	}

	for _, slotName := range slots {
		add := true
		for _, slot := range s.DisabledSlots {
			if slotName == slot {
				add = false
				break
			}
		}
		if add {
			ret = append(ret, slotName)
		}
	}

	return ret
}

func (s *Species) Filename() string {
	filename := util.ConvertForFilename(s.Name)
	return fmt.Sprintf("%d-%s.yaml", s.SpeciesId, filename)
}

func (s *Species) Filepath() string {
	return s.Filename()
}

func (s *Species) Save() error {

	bytes, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	saveFilePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `species`, `/`, s.Filename())

	err = os.WriteFile(saveFilePath, bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

// file self loads due to init()
func LoadDataFiles() {

	start := time.Now()

	tmpSpecies, err := fileloader.LoadAllFlatFiles[int, *Species](configs.GetFilePathsConfig().DataFiles.String() + `/species`)
	if err != nil {
		panic(err)
	}

	allSpecies = tmpSpecies

	mudlog.Info("species.LoadDataFiles()", "loadedCount", len(allSpecies), "Time Taken", time.Since(start))

}
