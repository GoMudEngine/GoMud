package keywords

import (
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var (
	loadedKeywords *Aliases
	fileSystems    []fileloader.ReadableGroupFS
)

type HelpTopic struct {
	Command   string
	Type      string // command/skill
	Category  string
	AdminOnly bool
}

type Aliases struct {
	Help               map[string]map[string][]string `yaml:"help"`
	HelpAliases        map[string][]string            `yaml:"help-aliases"`
	CommandAliases     map[string][]string            `yaml:"command-aliases"`
	DirectionAliases   map[string]string              `yaml:"direction-aliases"`
	MapLegendOverrides map[string]map[string]string   `yaml:"legend-overrides"`

	// helpTopics
	// [skill/command/admin]
	// helpTopics[`skill`][`character`][`alignment`] = `alignment`
	helpTopics map[string]HelpTopic
	// Organized help aliases
	helpAliases map[string]string
	// Organized command aliases
	commandAliases map[string]string
	// Converted strings to runes
	mapLegendOverrides map[string]map[rune]string
}

// Presumably to ensure the datafile hasn't messed something up.
func (a *Aliases) Validate() error {

	mergeAliases := []Aliases{*a}

	OLPath := util.FilePath(`data-overlays`, `/`, a.Filepath())
	for _, f := range fileSystems {

		for fsub := range f.AllFileSubSystems {

			if b, err := fsub.ReadFile(OLPath); err == nil {
				a := Aliases{}
				if err = yaml.Unmarshal(b, &a); err == nil {
					mergeAliases = append(mergeAliases, a)
				}
			}

		}
	}

	//
	// Unroll the data into structures that are quickly searched
	//

	a.helpTopics = map[string]HelpTopic{}
	a.helpAliases = map[string]string{}
	a.commandAliases = map[string]string{}
	a.mapLegendOverrides = map[string]map[rune]string{}

	for _, ma := range mergeAliases {

		// helpGroup = commands/skills/admin
		for helpGroup, helpTypes := range ma.Help {

			helpGroup = strings.ToLower(helpGroup)

			// helpType = configuration/character/shops/quests/combat
			for helpCategory, helpList := range helpTypes {
				helpCategory = strings.ToLower(helpCategory)
				for _, helpCommand := range helpList {

					helpCommand = strings.ToLower(helpCommand)

					entry := HelpTopic{
						Command:   helpCommand,
						Type:      helpGroup,
						Category:  helpCategory,
						AdminOnly: (helpGroup == `admin`),
					}

					a.helpTopics[helpCommand] = entry
				}
			}
		}

		for helpTopic, aliasList := range ma.HelpAliases {
			helpTopic = strings.ToLower(helpTopic)
			for _, alias := range aliasList {
				a.helpAliases[alias] = helpTopic
			}
		}

		for command, aliasList := range ma.CommandAliases {
			command = strings.ToLower(command)
			for _, alias := range aliasList {
				a.commandAliases[alias] = command
			}
		}

		for alias, direction := range ma.DirectionAliases {
			a.commandAliases[strings.ToLower(alias)] = direction
		}

		for area, overrides := range ma.MapLegendOverrides {

			area := strings.ToLower(area)
			if _, ok := a.mapLegendOverrides[area]; !ok {
				a.mapLegendOverrides[area] = map[rune]string{}
			}

			for symbol, name := range overrides {
				a.mapLegendOverrides[area][[]rune(symbol)[0]] = name
			}
		}

	}

	return nil
}

func (a *Aliases) Filename() string {
	return `keywords.yaml`
}

func (a *Aliases) Filepath() string {
	return a.Filename()
}

func GetAllHelpTopics() []string {
	helpTopics := []string{}

	for helpCommand, _ := range loadedKeywords.helpTopics {
		helpTopics = append(helpTopics, helpCommand)
	}

	sort.Slice(helpTopics, func(i, j int) bool {
		return helpTopics[i] < helpTopics[j]
	})

	return helpTopics
}

func GetAllHelpTopicInfo() []HelpTopic {
	helpTopics := []HelpTopic{}
	for _, helpTopicInfo := range loadedKeywords.helpTopics {
		helpTopics = append(helpTopics, helpTopicInfo)
	}

	sort.Slice(helpTopics, func(i, j int) bool {
		return helpTopics[i].Command < helpTopics[j].Command
	})

	return helpTopics
}

func GetAllCommandAliases() map[string]string {

	ret := map[string]string{}
	for alias, command := range loadedKeywords.commandAliases {
		ret[alias] = command
	}

	return ret
}

func GetAllHelpAliases() map[string]string {

	ret := map[string]string{}
	for alias, command := range loadedKeywords.helpAliases {
		ret[alias] = command
	}

	return ret
}

func GetAllLegendAliases(area ...string) map[rune]string {

	ret := map[rune]string{}

	for symbol, name := range loadedKeywords.mapLegendOverrides[`*`] {
		ret[symbol] = name
	}

	if len(area) > 0 {
		for symbol, name := range loadedKeywords.mapLegendOverrides[strings.ToLower(area[0])] {
			ret[symbol] = name
		}
	}

	return ret
}

func TryDirectionAlias(input string) string {
	if alias, ok := loadedKeywords.DirectionAliases[strings.ToLower(input)]; ok {
		return alias
	}

	return input
}

func TryCommandAlias(input string) string {
	if alias, ok := loadedKeywords.commandAliases[strings.ToLower(input)]; ok {
		return alias
	}

	return input
}

func TryHelpAlias(input string) string {
	if alias, ok := loadedKeywords.helpAliases[strings.ToLower(input)]; ok {
		return alias
	}
	return input
}

// Loads the ansi aliases from the config file
// Only if the file has been modified since the last load
func LoadAliases(f ...fileloader.ReadableGroupFS) {

	if len(f) > 0 {
		fileSystems = append(fileSystems, f...)
	}

	dataPath := string(configs.GetFilePathsConfig().DataFiles) + `/keywords.yaml`
	tmpLoadedKeywords, err := fileloader.LoadFlatFile[*Aliases](dataPath)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+dataPath))
	}

	loadedKeywords = tmpLoadedKeywords

}
