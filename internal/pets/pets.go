package pets

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/statmods"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Pet struct {
	Name          string            `yaml:"name,omitempty"`          // Name of the pet (player provided hopefully)
	NameStyle     string            `yaml:"namestyle,omitempty"`     // Optional color pattern to apply
	Type          string            `yaml:"type"`                    // type of pet
	Food          Food              `yaml:"food,omitempty"`          // how much food the pet has
	LastMealRound uint8             `yaml:"lastmealround,omitempty"` // When the pet was last fed
	Damage        items.Damage      `yaml:"damage,omitempty"`        // When the pet was last fed
	StatMods      statmods.StatMods `yaml:"statmods,omitempty"`      // stat mods the pet provides
	BuffIds       []int             `yaml:"buffids,omitempty"`       // Permabuffs this pet affords the player
	Capacity      int               `yaml:"capacity,omitempty"`      // How many items this mob can carry
	Items         []items.Item      `yaml:"items,omitempty"`         // Items held by this pet
}

var (
	petTypes = map[string]*Pet{}
)

func (p *Pet) StatMod(statName string) int {
	return p.StatMods.Get(statName)
}

func (p *Pet) Exists() bool {
	return p.Type != ``
}

func (p *Pet) DisplayName() string {

	name := p.Name
	if name == `` {
		name = p.Type
	}

	if len(p.NameStyle) > 0 {
		patternName := p.NameStyle
		if patternName[0:1] == `:` {
			patternName = patternName[1:]
		}
		return colorpatterns.ApplyColorPattern(name, patternName)
	}

	return fmt.Sprintf(`<ansi fg="petname">%s</ansi>`, name)
}

func (p *Pet) StoreItem(i items.Item) bool {

	if p.Capacity < 1 {
		return false
	}

	if i.ItemId < 1 {
		return false
	}
	i.Validate()
	p.Items = append(p.Items, i)
	return true
}

func (p *Pet) RemoveItem(i items.Item) bool {

	for j := len(p.Items) - 1; j >= 0; j-- {
		if p.Items[j].Equals(i) {
			p.Items = append(p.Items[:j], p.Items[j+1:]...)
			return true
		}
	}
	return false
}

func (p *Pet) GetBuffs() []int {
	return append([]int{}, p.BuffIds...)
}

func (p *Pet) FindItem(itemName string) (items.Item, bool) {

	if itemName == `` {
		return items.Item{}, false
	}

	closeMatchItem, matchItem := items.FindMatchIn(itemName, p.Items...)

	if matchItem.ItemId != 0 {
		return matchItem, true
	}

	if closeMatchItem.ItemId != 0 {
		return closeMatchItem, true
	}

	return items.Item{}, false
}

func (p *Pet) GetDiceRoll() (attacks int, dCount int, dSides int, bonus int, buffOnCrit []int) {
	return p.Damage.Attacks, p.Damage.DiceCount, p.Damage.SideCount, p.Damage.BonusDamage, p.Damage.CritBuffIds
}

func GetPetCopy(petId string) Pet {
	if petInfo, ok := petTypes[petId]; ok {
		return *petInfo
	}
	return Pet{}
}

func GetPetSpec(petId string) Pet {
	if petInfo, ok := petTypes[petId]; ok {
		return *petInfo
	}
	return Pet{}
}

func (p *Pet) Filename() string {
	filename := util.ConvertForFilename(p.Type)
	return fmt.Sprintf("%s.yaml", filename)
}

func (p *Pet) Filepath() string {
	return p.Filename()
}

func (p *Pet) Save() error {
	fileName := strings.ToLower(p.Name)

	bytes, err := yaml.Marshal(p)
	if err != nil {
		return err
	}

	saveFilePath := util.FilePath(configs.GetFilePathsConfig().DataFiles.String(), `/`, `pets`, `/`, fmt.Sprintf("%s.yaml", fileName))

	err = os.WriteFile(saveFilePath, bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (p *Pet) Id() string {
	return p.Type
}

func (p *Pet) Validate() error {

	if p.BuffIds == nil {
		p.BuffIds = []int{}
	}

	if p.Items == nil {
		p.Items = []items.Item{}
	}

	p.Damage.InitDiceRoll(p.Damage.DiceRoll)
	p.Damage.FormatDiceRoll()

	return nil
}

// file self loads due to init()
func LoadDataFiles() {

	start := time.Now()

	dataPath := configs.GetFilePathsConfig().DataFiles.String() + `/pets`
	tmpPetTypes, err := fileloader.LoadAllFlatFiles[string, *Pet](dataPath)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+dataPath))
	}

	petTypes = tmpPetTypes

	mudlog.Info("pets.LoadDataFiles()", "loadedCount", len(petTypes), "Time Taken", time.Since(start))
}
