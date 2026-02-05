package races

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
	races map[int]*Race = map[int]*Race{}
)

const (
	Small  Size = "small"  // Something like a mouse, dog
	Medium Size = "medium" // Something like a human
	Large  Size = "large"  // Something like a troll, ogre, dragon, kraken, or leviathan (or bigger).
)

type Race struct {
	RaceId           int
	Name             string
	Description      string
	DefaultAlignment int8
	BuffIds          []int // Permabuffs this race always has
	Size             Size
	TNLScale         float32
	UnarmedName      string
	Tameable         bool
	Damage           items.Damage
	Selectable       bool
	AngryCommands    []string         // randomly chosen to queue when they are angry/entering combat.
	KnowsFirstAid    bool             // Whether they can apply aid to other players.
	Stats            stats.Statistics // Base stats for this race.
	DisabledSlots    []string         `yaml:"disabledslots,omitempty"`
}

func GetRaces() []Race {
	ret := []Race{}
	for _, r := range races {
		ret = append(ret, *r)
	}
	return ret
}

func GetRace(raceId int) *Race {
	return races[raceId]
}

func FindRace(name string) (Race, bool) {

	name = strings.ToLower(name)

	closeMatch := -1
	for idx, r := range races {
		testName := strings.ToLower(r.Name)
		if strings.HasPrefix(testName, name) {
			return *r, true
		} else if strings.Contains(testName, name) {
			closeMatch = idx
		}
	}
	// close matches
	if closeMatch > -1 {
		return *races[closeMatch], true
	}

	return Race{}, false
}

func (r *Race) Id() int {
	return r.RaceId
}

func (r *Race) Validate() error {
	if r.Name == "" {
		return errors.New("race has no name")
	}
	if r.Description == "" {
		return errors.New("race has no description")
	}
	if r.Size == "" {
		return errors.New("race has no size")
	}
	r.Size = Size(strings.ToLower(string(r.Size))) // Sometimes a mismatching CaSe value is provided.

	// Recalculate stats, based on level one because this is actually the baseline for the race
	r.Stats.Strength.Recalculate(1)
	r.Stats.Dexterity.Recalculate(1)
	r.Stats.Perception.Recalculate(1)
	r.Stats.Vitality.Recalculate(1)
	r.Stats.Willpower.Recalculate(1)
	r.Stats.Charisma.Recalculate(1)

	if r.Damage.Attacks < 1 && r.Damage.DiceCount > 0 && r.Damage.SideCount > 0 {
		r.Damage.Attacks = 1
	}

	// If a diceroll was specified, absorb that into the damage struct
	if r.Damage.DiceRoll != `` {
		r.Damage.InitDiceRoll(r.Damage.DiceRoll)
		r.Damage.FormatDiceRoll()
	}

	// Ensure TNLScale is never 0, default to 1.0
	if r.TNLScale == 0 {
		r.TNLScale = 1.0
	}

	return nil
}

func (r Race) GetEnabledSlots() []string {

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
		for _, slot := range r.DisabledSlots {
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

func (r *Race) Filename() string {
	filename := util.ConvertForFilename(r.Name)
	return fmt.Sprintf("%d-%s.yaml", r.RaceId, filename)
}

func (r *Race) Filepath() string {
	return r.Filename()
}

func (r *Race) Save() error {

	bytes, err := yaml.Marshal(r)
	if err != nil {
		return err
	}

	saveFilePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `races`, `/`, r.Filename())

	err = os.WriteFile(saveFilePath, bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

// file self loads due to init()
func LoadDataFiles() {

	start := time.Now()

	tmpRaces, err := fileloader.LoadAllFlatFiles[int, *Race](configs.GetFilePathsConfig().DataFiles.String() + `/races`)
	if err != nil {
		panic(err)
	}

	races = tmpRaces

	mudlog.Info("races.LoadDataFiles()", "loadedCount", len(races), "Time Taken", time.Since(start))

}
