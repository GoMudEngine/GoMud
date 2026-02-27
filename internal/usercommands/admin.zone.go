package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
* Role Permissions:
* zone 				(All)
 */
func Zone(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	handled := true

	// args should look like one of the following:
	// info <optional room id>
	// <move to room id>
	args := util.SplitButRespectQuotes(rest)

	if len(args) == 0 {
		// send some sort of help info?
		infoOutput, _ := templates.Process("admincommands/help/command.zone", nil, user.UserId)
		user.SendText(infoOutput)

		return handled, nil
	}

	zoneCmd := strings.ToLower(args[0])
	args = args[1:]

	// Interactive Editing
	if zoneCmd == `edit` {
		return zone_Edit(``, user, room, flags)
	}

	zoneConfig := rooms.GetZoneConfig(room.Zone)
	if zoneConfig == nil {
		user.SendText(fmt.Sprintf(`Couldn't find zone info for <ansi fg="red">%s</ansi>`, room.Zone))
		return true, nil
	}

	if zoneCmd == `info` {

		user.SendText(``)
		user.SendText(fmt.Sprintf(`<ansi fg="yellow-bold">Zone Config for:    <ansi fg="red">%s</ansi></ansi>`, room.Zone))
		user.SendText(fmt.Sprintf(`   <ansi fg="yellow-bold">Root Room Id:</ansi>    <ansi fg="red">%d</ansi>`, zoneConfig.RoomId))

		user.SendText(``)

		return true, nil
	}

	// Everthing after this point requires additional args
	if len(args) < 1 {
		user.SendText(`Not enough arguments provided.`)
		return true, nil
	}

	if zoneCmd == `set` {

		setWhat := args[0]

		args = args[1:]

		if setWhat == `autoscale` {
			user.SendText(`Autoscaling has been removed. Use per-mob statpool values instead.`)
			return true, nil
		}

	}

	return true, nil
}

func zone_Edit(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	originalZoneConfig := rooms.GetZoneConfig(room.Zone)
	if originalZoneConfig == nil {
		user.SendText(`Could not find zone config.`)
		return true, nil
	}

	// Make a copy that we'll edit
	editZoneConfig := *originalZoneConfig

	allZoneMutators := []string{}
	for _, roomMut := range editZoneConfig.Mutators {
		allZoneMutators = append(allZoneMutators, roomMut.MutatorId)
	}

	cmdPrompt, _ := user.StartPrompt(`zone edit`, rest)

	selectedMutatorList := []string{}
	if muts, ok := cmdPrompt.Recall(`mutators`); ok {
		selectedMutatorList = muts.([]string)
	} else {
		if len(selectedMutatorList) == 0 {
			selectedMutatorList = append(selectedMutatorList, allZoneMutators...)
		}
	}

	// Keep track of the state
	cmdPrompt.Store(`mutators`, selectedMutatorList)

	selectedMutatorLookup := map[string]bool{}
	for _, mutId := range selectedMutatorList {
		selectedMutatorLookup[mutId] = true
	}

	mutatorOptions := []templates.NameDescription{}

	for _, mutId := range mutators.GetAllMutatorIds() {
		marked := false
		if _, ok := selectedMutatorLookup[mutId]; ok {
			marked = true
		}

		mutatorOptions = append(mutatorOptions, templates.NameDescription{Id: mutId, Marked: marked, Name: mutId})

	}

	sort.SliceStable(mutatorOptions, func(i, j int) bool {
		return mutatorOptions[i].Name < mutatorOptions[j].Name
	})

	question := cmdPrompt.Ask(`Select a mutator to add/remove, or nothing to continue:`, []string{}, `0`)
	if !question.Done {
		tplTxt, _ := templates.Process("tables/numbered-list-doubled", mutatorOptions, user.UserId)
		user.SendText(tplTxt)
		return true, nil
	}

	if question.Response != `0` {

		mutatorSelected := ``

		if restNum, err := strconv.Atoi(question.Response); err == nil {
			if restNum > 0 && restNum <= len(mutatorOptions) {
				mutatorSelected = mutatorOptions[restNum-1].Id.(string)
			}
		}

		if mutatorSelected == `` {
			for _, b := range mutatorOptions {
				if strings.EqualFold(b.Name, question.Response) {
					mutatorSelected = b.Id.(string)
					break
				}
			}
		}

		if mutatorSelected == `` {

			user.SendText("Invalid selection.")
			question.RejectResponse()

			tplTxt, _ := templates.Process("tables/numbered-list-doubled", mutatorOptions, user.UserId)
			user.SendText(tplTxt)
			return true, nil
		}

		if _, ok := selectedMutatorLookup[mutatorSelected]; ok {

			delete(selectedMutatorLookup, mutatorSelected)
			for idx, mutId := range selectedMutatorList {
				if mutId == mutatorSelected {
					selectedMutatorList = append(selectedMutatorList[0:idx], selectedMutatorList[idx+1:]...)
					break
				}
			}

		} else {

			selectedMutatorList = append(selectedMutatorList, mutatorSelected)
			selectedMutatorLookup[mutatorSelected] = true

		}

		cmdPrompt.Store(`mutators`, selectedMutatorList)

		question.RejectResponse()

		for idx, data := range mutatorOptions {
			_, data.Marked = selectedMutatorLookup[data.Id.(string)]
			mutatorOptions[idx] = data
		}

		tplTxt, _ := templates.Process("tables/numbered-list-doubled", mutatorOptions, user.UserId)
		user.SendText(tplTxt)
		return true, nil

	}

	//
	// Music Options
	//
	{

		question := cmdPrompt.Ask(`Should the zone have music?`, []string{`yes`, `no`}, util.BoolYN(editZoneConfig.MusicFile != ``))
		if !question.Done {
			return true, nil
		}

		if question.Response == `yes` {

			relativeString := configs.GetFilePathsConfig().WebCDNLocation.String()
			if len(relativeString) > 0 {
				user.SendText(`   <ansi fg="red">Note:</ansi> Music file path must be relative to: <ansi fg="red">` + relativeString + `</ansi>`)
			}

			question := cmdPrompt.Ask(`Zone music file path?`, []string{editZoneConfig.MusicFile}, editZoneConfig.MusicFile)
			if !question.Done {
				return true, nil
			}
			editZoneConfig.MusicFile = question.Response

		} else {
			editZoneConfig.MusicFile = ``
		}

	}

	//
	// Done editing. Save results
	//
	editZoneConfig.Mutators = mutators.MutatorList{}
	for _, mutId := range selectedMutatorList {
		editZoneConfig.Mutators = append(editZoneConfig.Mutators, mutators.Mutator{MutatorId: mutId})
	}

	rooms.SaveZoneConfig(&editZoneConfig)

	user.SendText(``)
	user.SendText(`Changes saved.`)
	user.SendText(``)

	user.ClearPrompt()

	return true, nil
}
