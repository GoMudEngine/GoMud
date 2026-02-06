package usercommands

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Help(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	var helpTxt string
	var err error = nil

	args := util.SplitButRespectQuotes(rest)

	if len(args) == 0 {

		type helpCommand struct {
			Command string
			Type    string
			Missing bool
		}

		type commandLists struct {
			Commands map[string][]helpCommand
			Skills   map[string][]helpCommand
			Admin    map[string][]helpCommand
		}

		helpCommandList := commandLists{
			Commands: make(map[string][]helpCommand),
			Skills:   make(map[string][]helpCommand),
			Admin:    make(map[string][]helpCommand),
		}

		for _, command := range keywords.GetAllHelpTopicInfo() {

			category := command.Category
			if category == `all` {
				category = ``
			}

			templateFile := `help/` + keywords.TryHelpAlias(command.Command)

			if command.AdminOnly {
				if user.HasRolePermission(command.Command, true) {
					helpCommandList.Admin[category] = append(
						helpCommandList.Admin[category],
						helpCommand{Command: command.Command, Type: "command-admin", Missing: false},
					)
				}
				continue
			}

			hlpCmd := helpCommand{Command: command.Command, Type: command.Type, Missing: !templates.Exists(templateFile)}

			if command.Type == `skill` {
				helpCommandList.Skills[category] = append(helpCommandList.Skills[category], hlpCmd)
				continue
			}

			helpCommandList.Commands[category] = append(helpCommandList.Commands[category], hlpCmd)

		}

		helpTxt, err = templates.Process("help/help", helpCommandList, user.UserId)
		if err != nil {
			helpTxt = err.Error()
		}
	} else {

		helpTxt, err = GetHelpContents(rest)
		if err != nil {
			user.SendText(fmt.Sprintf(`No help found for "%s"`, rest))
			return true, err
		}

	}

	user.SendText(helpTxt)

	return true, nil
}

func getSpeciesOptions(speciesRequest string) []species.Species {

	allSpecies := species.GetAllSpecies()
	sort.Slice(allSpecies, func(i, j int) bool {
		return allSpecies[i].SpeciesId < allSpecies[j].SpeciesId
	})

	speciesNames := strings.Split(speciesRequest, ` `)

	getAllSpecies := false
	if speciesRequest == `all` {
		getAllSpecies = true
	}

	speciesOptions := []species.Species{}
	for _, sp := range allSpecies {

		if len(speciesRequest) == 0 {
			if !sp.Selectable && !getAllSpecies {
				continue
			}
		} else if len(speciesNames) > 0 {
			lowerName := strings.ToLower(sp.Name)
			found := false
			for _, rName := range speciesNames {
				if strings.Contains(lowerName, strings.ToLower(rName)) {
					found = true
					break
				}
			}
			if !getAllSpecies && !found {
				continue
			}
		}
		speciesOptions = append(speciesOptions, sp)
	}

	return speciesOptions
}

func GetHelpContents(input string) (string, error) {

	args := util.SplitButRespectQuotes(input)

	helpName := args[0]
	helpRest := ``

	args = args[1:]
	if len(args) > 0 {
		helpRest = strings.Join(args, ` `)
	}

	// replace any non alpha/numeric characters in "rest"
	if fullSearchAlias := keywords.TryHelpAlias(input); fullSearchAlias != input {
		helpName = fullSearchAlias
	} else {
		helpName = regexp.MustCompile(`[^a-zA-Z0-9\\-]+`).ReplaceAllString(helpName, ``)
		helpName = keywords.TryHelpAlias(helpName)
	}

	var helpVars any = nil

	if helpName == `emote` {
		helpVars = emoteAliases
	}

	if helpName == `races` {
		helpVars = getSpeciesOptions(helpRest)
	}

	if helpName == `spell` {
		sData := spells.GetSpell(helpRest)
		if sData == nil {
			sData = spells.FindSpellByName(helpRest)
		}

		if sData == nil {
			helpName = `spells`
		} else {
			helpVars = *sData
		}
	}

	return templates.Process("help/"+helpName, helpVars, 0)
}
