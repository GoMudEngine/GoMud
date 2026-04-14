package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/util"

	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* mob 				(All)
* mob.create		(Create new mobs)
* mob.spawn			(Spawn a mob in the room)
 */
func Mob(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(rest)

	if len(args) < 1 {
		infoOutput, _ := templates.Process("admincommands/help/command.mob", nil, user.UserId)
		user.SendText(infoOutput)
		return true, nil
	}

	// Create a new mob
	if args[0] == `create` {

		if !user.HasRolePermission(`mob.create`) {
			user.SendText(`you do not have <ansi fg="command">mob.create</ansi> permission`)
			return true, nil
		}

		return mob_Create(strings.TrimSpace(rest[6:]), user, room, flags)
	}

	// Spawn a mob instance
	if args[0] == `spawn` {

		if !user.HasRolePermission(`mob.spawn`) {
			user.SendText(`you do not have <ansi fg="command">mob.spawn</ansi> permission`)
			return true, nil
		}

		return mob_Spawn(strings.TrimSpace(rest[5:]), user, room, flags)
	}

	// List existing mobs
	if args[0] == `list` {

		if !user.HasRolePermission(`mob.spawn`) {
			user.SendText(`you do not have <ansi fg="command">mob.spawn</ansi> permission`)
			return true, nil
		}

		return mob_List(strings.TrimSpace(rest[4:]), user, room, flags)
	}

	return true, nil
}

func mob_List(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {

	mobList := []templates.NameDescription{}
	longestName := 0

	for _, mob := range mobs.GetAllMobInfo() {

		entry := templates.NameDescription{
			Id:   mob.MobId,
			Name: mob.Character.Name,
		}

		// If searching for matches
		if len(rest) > 0 {
			if !strings.Contains(rest, `*`) {
				rest += `*`
			}
			if !util.StringWildcardMatch(strings.ToLower(entry.Name), rest) {
				continue
			}
		}

		if len(entry.Name) > longestName {
			longestName = len(entry.Name)
		}

		mobList = append(mobList, entry)
	}

	sort.SliceStable(mobList, func(i, j int) bool {
		return strings.ToLower(mobList[i].Name) < strings.ToLower(mobList[j].Name)
	})

	numWidth := len(strconv.Itoa(len(mobList)))
	colWidth := 1 + numWidth + 2 + longestName + 1

	user.SendText(``)
	sw := user.ClientSettings().Display.GetScreenWidth()
	strOut := templates.DynamicList(mobList, colWidth, sw, numWidth, longestName)
	user.SendText(strOut)
	user.SendText(``)

	return true, nil
}

func mob_Spawn(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	c := configs.GetLootGoblinConfig()

	// special handling of loot goblin
	if rest == `loot goblin` && c.RoomId != 0 {
		if gRoom := rooms.LoadRoom(int(c.RoomId)); gRoom != nil { // loot goblin room
			user.SendText(`Somewhere in the realm, a <ansi fg="mobname">loot goblin</ansi> appears!`)
			mudlog.Info(`Loot Goblin Spawn`, `roundNumber`, util.GetRoundCount(), `forced`, true)
			gRoom.Prepare(false) // Make sure the loot goblin spawns.
		}
		return true, nil
	}

	mobId := mobs.MobIdByName(rest)

	if mobId < 1 {
		mobIdInt, _ := strconv.Atoi(rest)
		mobId = mobs.MobId(mobs.MobId(mobIdInt))
	}

	if mobId > 0 {
		if mob := mobs.NewMobById(mobId, room.RoomId); mob != nil {
			room.AddMob(mob.InstanceId)

			user.SendText(
				fmt.Sprintf(`You wave your hands around and <ansi fg="mobname">%s</ansi> appears in the air and falls to the ground.`, mob.Character.Name),
			)
			room.SendTextVisual(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> waves their hands around and <ansi fg="mobname">%s</ansi> appears in the air and falls to the ground.`, user.Character.Name, mob.Character.Name),
				user.UserId,
			)

			return true, nil
		}
	}

	user.SendText(
		fmt.Sprintf(`Mob <ansi fg="mobname">%s</ansi> not found.`, rest),
	)

	return true, nil
}

func mob_Create(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	var newMob = mobs.Mob{}

	if len(rest) > 0 {
		if mobId, err := strconv.Atoi(rest); err == nil {
			newMob = *(mobs.GetMobSpec(mobs.MobId(mobId)))
		}
		if newMob.MobId == 0 {
			if mobId := mobs.MobIdByName(rest); mobId != 0 {
				newMob = *(mobs.GetMobSpec(mobId))
			}
		}
	}

	// Get if already exists, otherwise create new
	cmdPrompt, isNew := user.StartPrompt(`mob create`, rest)

	if isNew {
		user.SendText(``)
		user.SendText(fmt.Sprintf(`Lets get a little info first.%s`, term.CRLFStr))
	}

	//
	// Name Selection
	//
	question := cmdPrompt.Ask(`What will the mob be called?`, []string{newMob.Character.Name}, newMob.Character.Name)
	if !question.Done {
		return true, nil
	}

	if question.Response == `` {
		user.SendText("Aborting...")
		user.ClearPrompt()
		return true, nil
	}

	newMob.Character.Name = question.Response

	//
	// Species Selection
	//
	allSpecies := species.GetAllSpecies()

	speciesOptions := []templates.NameDescription{}
	for _, r := range allSpecies {
		speciesOptions = append(speciesOptions, templates.NameDescription{
			Name:        r.Name,
			Description: r.Description,
		})
	}

	sort.SliceStable(speciesOptions, func(i, j int) bool {
		return speciesOptions[i].Name < speciesOptions[j].Name
	})

	raceName := ``
	if newMob.Character.SpeciesId > 0 {
		if r := species.GetSpecies(newMob.Character.SpeciesId); r != nil {
			raceName = r.Name
		}
	}

	question = cmdPrompt.Ask(`What species will the mob be?`, []string{raceName}, raceName)
	if !question.Done {
		tplTxt, _ := templates.Process("tables/numbered-list", speciesOptions, user.UserId)
		user.SendText(tplTxt)
		user.SendText(`  <ansi fg="black-bold">Enter <ansi fg="command">help {speciesname}</ansi> or <ansi fg="command">help {number}</ansi> for details.</ansi>`)
		user.SendText(``)
		return true, nil
	}

	if question.Response == `` {
		user.SendText("Aborting...")
		user.ClearPrompt()
		return true, nil
	}

	respLower := strings.ToLower(question.Response)
	if len(respLower) >= 5 && respLower[0:5] == `help ` {
		helpCmd := `race`
		helpRest := respLower[5:]

		if restNum, err := strconv.Atoi(helpRest); err == nil {
			if restNum > 0 && restNum <= len(speciesOptions) {
				helpRest = speciesOptions[restNum-1].Name
			} else {
				helpCmd = `races`
				helpRest = ``
			}
		}

		question.RejectResponse()
		return Help(helpCmd+` `+helpRest, user, room, flags)
	}

	raceNameSelection := question.Response
	if restNum, err := strconv.Atoi(raceNameSelection); err == nil {
		if restNum > 0 && restNum <= len(speciesOptions) {
			raceNameSelection = speciesOptions[restNum-1].Name
		}
	}

	for _, r := range allSpecies {
		if strings.EqualFold(r.Name, raceNameSelection) {
			newMob.Character.SpeciesId = r.SpeciesId
			break
		}
	}

	if newMob.Character.SpeciesId == 0 {
		question.RejectResponse()

		tplTxt, _ := templates.Process("tables/numbered-list", speciesOptions, user.UserId)
		user.SendText(tplTxt)
		user.SendText(`  <ansi fg="black-bold">Enter <ansi fg="command">help {speciesname}</ansi> or <ansi fg="command">help {number}</ansi> for details.</ansi>`)
		user.SendText(``)

		return true, nil
	}

	//
	// Zone Selection
	//
	allZones := rooms.GetAllZoneNames()

	zoneOptions := []templates.NameDescription{}
	for _, z := range allZones {
		zoneOptions = append(zoneOptions, templates.NameDescription{
			Name:        z,
			Description: ``,
		})
	}

	sort.SliceStable(zoneOptions, func(i, j int) bool {
		return zoneOptions[i].Name < zoneOptions[j].Name
	})

	question = cmdPrompt.Ask(`What zone is this mob from?`, []string{newMob.Zone}, newMob.Zone)
	if !question.Done {
		tplTxt, _ := templates.Process("tables/numbered-list-doubled", zoneOptions, user.UserId)
		user.SendText(tplTxt)
		return true, nil
	}

	if question.Response == `` {
		user.SendText("Aborting...")
		user.ClearPrompt()
		return true, nil
	}

	zoneNameSelection := question.Response
	if restNum, err := strconv.Atoi(zoneNameSelection); err == nil {
		if restNum > 0 && restNum <= len(zoneOptions) {
			zoneNameSelection = zoneOptions[restNum-1].Name
		}
	}

	for _, z := range allZones {
		if strings.EqualFold(z, zoneNameSelection) {
			newMob.Zone = z
			break
		}
	}

	if newMob.Zone == `` {
		question.RejectResponse()

		tplTxt, _ := templates.Process("tables/numbered-list", zoneOptions, user.UserId)
		user.SendText(tplTxt)
		return true, nil
	}

	//
	// Description
	//
	question = cmdPrompt.Ask(`Enter a description for the mob:`, []string{newMob.Character.GetDescription()}, newMob.Character.GetDescription())
	if !question.Done {
		return true, nil
	}

	if question.Response != `` {
		newMob.Character.Description = question.Response
	}

	//
	// Max Wander
	//
	question = cmdPrompt.Ask(`How far can this mob wander (-1 = none. 0 = unlimted)?`, []string{strconv.Itoa(newMob.MaxWander)}, strconv.Itoa(newMob.MaxWander))
	if !question.Done {
		return true, nil
	}

	newMob.MaxWander, _ = strconv.Atoi(question.Response)

	//
	// Hostile?
	//
	defaultYN := `n`
	if newMob.Hostile {
		defaultYN = `y`
	}

	question = cmdPrompt.Ask(`Is this mob hostile?`, []string{`y`, `n`}, defaultYN)
	if !question.Done {
		return true, nil
	}

	newMob.Hostile = question.Response == `y`

	//
	// Quest Script?
	//
	question = cmdPrompt.Ask(`Create with a sample script?`, []string{`y`, `n`}, `n`)
	if !question.Done {
		return true, nil
	}

	scriptType := ``
	scriptTemplate := ``
	if question.Response == `y` {

		scriptOptions := []templates.NameDescription{}
		for about, _ := range mobs.SampleScripts {
			scriptOptions = append(scriptOptions, templates.NameDescription{
				Name: about,
			})
		}

		sort.SliceStable(scriptOptions, func(i, j int) bool {
			return scriptOptions[i].Name < scriptOptions[j].Name
		})

		question = cmdPrompt.Ask(`Which sample script?`, []string{})
		if !question.Done {
			tplTxt, _ := templates.Process("tables/numbered-list", scriptOptions, user.UserId)
			user.SendText(tplTxt)
			return true, nil
		}

		scriptType = question.Response

		if restNum, err := strconv.Atoi(scriptType); err == nil {
			if restNum > 0 && restNum <= len(scriptOptions) {
				scriptType = scriptOptions[restNum-1].Name
			}
		}

		if _, ok := mobs.SampleScripts[scriptType]; !ok {
			question.RejectResponse()

			tplTxt, _ := templates.Process("tables/numbered-list", scriptOptions, user.UserId)
			user.SendText(tplTxt)
			return true, nil
		}

		scriptTemplate = mobs.SampleScripts[scriptType]
	}

	//
	// Confirm?
	//
	question = cmdPrompt.Ask(`Does this look correct?`, []string{`y`, `n`}, `n`)
	if !question.Done {

		user.SendText(`  <ansi fg="yellow-bold">Name:</ansi>    <ansi fg="white-bold">` + newMob.Character.Name + `</ansi>`)
		user.SendText(`  <ansi fg="yellow-bold">Species:</ansi> <ansi fg="white-bold">` + strconv.Itoa(newMob.Character.SpeciesId) + ` (` + raceNameSelection + `)</ansi>`)
		user.SendText(`  <ansi fg="yellow-bold">Zone:</ansi>    <ansi fg="white-bold">` + newMob.Zone + `</ansi>`)
		user.SendText(`  <ansi fg="yellow-bold">Desc:</ansi>    <ansi fg="white-bold">` + newMob.Character.Description + `</ansi>`)
		user.SendText(`  <ansi fg="yellow-bold">Wander:</ansi>  <ansi fg="white-bold">` + strconv.Itoa(newMob.MaxWander) + `</ansi>`)
		user.SendText(`  <ansi fg="yellow-bold">Hostile:</ansi> <ansi fg="white-bold">` + strconv.FormatBool(newMob.Hostile) + `</ansi>`)
		user.SendText(`  <ansi fg="yellow-bold">Script:</ansi>  <ansi fg="white-bold">` + scriptType + ` (` + scriptTemplate + `)</ansi>`)

		return true, nil
	}

	user.ClearPrompt()

	if question.Response != `y` {
		user.SendText("Aborting...")
		return true, nil
	}

	mobId, err := mobs.CreateNewMobFile(newMob, scriptTemplate)

	if err != nil {
		user.SendText("Error: " + err.Error())
		return true, nil
	}

	mobInst := mobs.GetMobSpec(mobId)

	user.SendText(``)
	user.SendText(`  <ansi bg="red" fg="white-bold">MOB CREATED</ansi>`)
	user.SendText(``)
	user.SendText(`  <ansi fg="yellow-bold">File Path:</ansi>   <ansi fg="white-bold">` + mobInst.Filepath() + `</ansi>`)
	if scriptTemplate != `` {
		user.SendText(`  <ansi fg="yellow-bold">Script Path:</ansi> <ansi fg="white-bold">` + mobInst.GetScriptPath() + `</ansi>`)
	}
	user.SendText(``)
	user.SendText(`  <ansi fg="black-bold">note: Try <ansi fg="command">mob spawn ` + mobInst.Character.Name + `</ansi> to test it.`)

	return true, nil
}
