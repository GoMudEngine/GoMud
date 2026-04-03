package usercommands

import (
	"errors"
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/prompt"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Character(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !room.IsCharacterRoom {
		return false, fmt.Errorf(`not in a IsCharacterRoom`)
	}

	altNames := []string{}
	nameToAlt := map[string]characters.Character{}

	for _, char := range characters.LoadAlts(user.UserId) {
		altNames = append(altNames, char.Name)
		nameToAlt[char.Name] = char
	}

	c := configs.GetGamePlayConfig()

	if c.MaxAltCharacters == 0 {
		user.SendText(`<ansi fg="203">Alt character are disabled on this server.</ansi>`)
		return true, errors.New(`alt characters disabled`)
	}

	if user.Character.GetTotalSkillRanks() < 20 && len(nameToAlt) < 1 {
		user.SendText(`<ansi fg="203">You must gain at least 20 total skill ranks to access character alts.</ansi>`)
		return true, errors.New(`20 skill ranks minimum`)
	}

	// Form a set of all mobs currently charmed (and possibly hired)
	hiredOutChars := map[string]characters.Character{}
	for _, mobInstanceId := range user.Character.GetCharmIds() {
		mob := mobs.GetInstance(mobInstanceId)
		if mob == nil {
			continue
		}
		hiredOutChars[mob.Character.Name] = mob.Character
	}

	menuOptions := []string{`new`}

	cmdPrompt, isNew := user.StartPrompt(`character`, rest)

	if isNew {

		if len(altNames) > 0 {
			menuOptions = append(menuOptions, `view`)
			menuOptions = append(menuOptions, `change`)
			menuOptions = append(menuOptions, `delete`)
			menuOptions = append(menuOptions, `hire`)
		}

		if len(nameToAlt) > 0 {
			altTblTxt := getAltTable(nameToAlt, hiredOutChars, user.UserId)
			user.SendText(``)
			user.SendText(altTblTxt)
		}

	}

	menuOptions = append(menuOptions, `quit`)

	question := cmdPrompt.Ask(`What would you like to do?`, menuOptions, `quit`)
	if !question.Done {
		return true, nil
	}

	if question.Response == `quit` {
		user.ClearPrompt()
		return true, nil
	}

	switch question.Response {
	case `new`:
		return cmdCharacterNew(user, room, cmdPrompt, nameToAlt, altNames, c)
	case `delete`:
		return cmdCharacterDelete(user, cmdPrompt, altNames, nameToAlt, hiredOutChars)
	case `change`:
		return cmdCharacterChange(user, room, cmdPrompt, altNames, nameToAlt, hiredOutChars, flags)
	case `view`:
		return cmdCharacterView(user, room, cmdPrompt, altNames, nameToAlt, hiredOutChars, flags)
	case `hire`:
		return cmdCharacterHire(user, room, cmdPrompt, altNames, nameToAlt, hiredOutChars)
	}

	return true, nil
}

// selectAltByName prompts for an alt name and returns the match and raw response.
// done=false means the question hasn't been answered yet.
func selectAltByName(cmdPrompt *prompt.Prompt, altNames []string, questionText string) (match string, rawResponse string, done bool) {
	question := cmdPrompt.Ask(questionText, []string{})
	if !question.Done {
		return ``, ``, false
	}

	m, closeMatch := util.FindMatchIn(question.Response, altNames...)
	if m == `` {
		m = closeMatch
	}

	return m, question.Response, true
}

// checkHiredOut checks if a character is currently hired out as a mob. Returns true if hired out (and sends message).
func checkHiredOut(user *users.UserRecord, char characters.Character, hiredOutChars map[string]characters.Character) bool {
	if friend, ok := hiredOutChars[char.Name]; ok && friend.Description == char.Description {
		user.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is currently hired out.`, char.Name))
		user.ClearPrompt()
		return true
	}
	return false
}

func cmdCharacterNew(user *users.UserRecord, room *rooms.Room, cmdPrompt *prompt.Prompt, nameToAlt map[string]characters.Character, altNames []string, c configs.GamePlay) (bool, error) {

	if len(altNames) >= int(c.MaxAltCharacters) {
		user.SendText(`<ansi fg="203">You already have too many alts.</ansi>`)
		user.SendText(`<ansi fg="203">You'll need to delete one to create a new one.</ansi>`)

		// Reject the menu response so they can pick again
		// Re-calling Ask with the same question text returns the existing question object
		cmdPrompt.Ask(`What would you like to do?`, []string{`quit`}, `quit`).RejectResponse()
		return true, nil
	}

	question := cmdPrompt.Ask(`Are you SURE? (Your current character will be saved here to change back to later)`, []string{`yes`, `no`}, `no`)
	if !question.Done {
		return true, nil
	}

	if question.Response == `no` {
		user.ClearPrompt()
		return true, nil
	}

	newAlts := []characters.Character{}
	for _, char := range nameToAlt {
		newAlts = append(newAlts, char)
	}
	newAlts = append(newAlts, *user.Character)
	characters.SaveAlts(user.UserId, newAlts)

	// Send them back to start with a fresh/empty character
	user.Character = characters.New()
	user.Character.Name = user.TempName()

	room.RemovePlayer(user.UserId)
	rooms.MoveToRoom(user.UserId, -1)

	return true, nil
}

func cmdCharacterDelete(user *users.UserRecord, cmdPrompt *prompt.Prompt, altNames []string, nameToAlt map[string]characters.Character, hiredOutChars map[string]characters.Character) (bool, error) {

	if len(nameToAlt) > 0 {
		altTblTxt := getAltTable(nameToAlt, hiredOutChars, user.UserId)
		user.SendText(``)
		user.SendText(altTblTxt)
	}

	match, rawResponse, done := selectAltByName(cmdPrompt, altNames, `Enter the name of the character you wish to delete:`)
	if !done {
		return true, nil
	}

	if match != `` {

		delChar := nameToAlt[match]

		if checkHiredOut(user, delChar, hiredOutChars) {
			return true, nil
		}

		question := cmdPrompt.Ask(`<ansi fg="red">Are you SURE you want to delete <ansi fg="username">`+delChar.Name+`</ansi>?</ansi>`, []string{`yes`, `no`}, `no`)
		if !question.Done {
			return true, nil
		}

		if question.Response == `no` {
			user.SendText(`<ansi fg="203">Okay. Aborting.</ansi>`)
			user.ClearPrompt()
			return true, nil
		}

		newAlts := []characters.Character{}
		for _, char := range nameToAlt {
			if char.Name != match {
				newAlts = append(newAlts, char)
			}
		}
		characters.SaveAlts(user.UserId, newAlts)

		user.EventLog.Add(`char`, `Deleted alt character: <ansi fg="username">`+match+`</ansi>`)

		user.SendText(`<ansi fg="username">` + match + `</ansi> <ansi fg="red">is deleted.</ansi>`)
		user.ClearPrompt()
		return true, nil

	}

	user.SendText(`<ansi fg="203">No character with the name <ansi fg="username">` + rawResponse + `</ansi> found.</ansi>`)

	user.ClearPrompt()
	return true, nil
}

func cmdCharacterChange(user *users.UserRecord, room *rooms.Room, cmdPrompt *prompt.Prompt, altNames []string, nameToAlt map[string]characters.Character, hiredOutChars map[string]characters.Character, flags events.EventFlag) (bool, error) {

	if len(nameToAlt) > 0 {
		altTblTxt := getAltTable(nameToAlt, hiredOutChars, user.UserId)
		user.SendText(``)
		user.SendText(altTblTxt)
	}

	match, rawResponse, done := selectAltByName(cmdPrompt, altNames, `Enter the name of the character you wish to change to:`)
	if !done {
		return true, nil
	}

	if match != `` {

		char := nameToAlt[match]

		if checkHiredOut(user, char, hiredOutChars) {
			return true, nil
		}

		question := cmdPrompt.Ask(`<ansi fg="51">Are you SURE you want to change to <ansi fg="username">`+char.Name+`</ansi>?</ansi>`, []string{`yes`, `no`}, `no`)
		if !question.Done {
			return true, nil
		}

		if question.Response == `no` {
			user.SendText(`<ansi fg="203">Okay. Aborting.</ansi>`)
			user.ClearPrompt()
			return true, nil
		}

		oldName := user.Character.Name

		succes := user.SwapToAlt(match)
		if !succes {
			user.SendText(`<ansi fg="203">Something went wrong.</ansi>`)
			user.ClearPrompt()
			return true, nil
		}

		newRoom := rooms.LoadRoom(user.Character.RoomId)
		if newRoom == nil {
			user.Character.RoomId = 0
			newRoom = rooms.LoadRoom(user.Character.RoomId)
		}

		// Remove from old room
		room.RemovePlayer(user.UserId)
		// add to new room
		newRoom.AddPlayer(user.UserId)

		users.SaveUser(*user)

		user.EventLog.Add(`char`, `Changed from <ansi fg="username">`+oldName+`</ansi> to alt character: <ansi fg="username">`+char.Name+`</ansi>`)

		user.SendText(term.CRLFStr + `You dematerialize as <ansi fg="username">` + oldName + `</ansi>. and rematerialize as <ansi fg="username">` + char.Name + `</ansi>!` + term.CRLFStr)
		room.SendText(`<ansi fg="username">`+oldName+`</ansi> vanishes, and <ansi fg="username">`+char.Name+`</ansi> appears in a shower of sparks!`, user.UserId)

		user.ClearPrompt()
		return true, nil

	}

	user.SendText(`<ansi fg="203">No character with the name <ansi fg="username">` + rawResponse + `</ansi> found.</ansi>`)

	user.ClearPrompt()
	return true, nil
}

func cmdCharacterView(user *users.UserRecord, room *rooms.Room, cmdPrompt *prompt.Prompt, altNames []string, nameToAlt map[string]characters.Character, hiredOutChars map[string]characters.Character, flags events.EventFlag) (bool, error) {

	if len(nameToAlt) > 0 {
		altTblTxt := getAltTable(nameToAlt, hiredOutChars, user.UserId)
		user.SendText(``)
		user.SendText(altTblTxt)
	}

	match, rawResponse, done := selectAltByName(cmdPrompt, altNames, `Enter the name of the character you wish to view:`)
	if !done {
		return true, nil
	}

	if match != `` {

		char := nameToAlt[match]

		if checkHiredOut(user, char, hiredOutChars) {
			return true, nil
		}

		char.Validate()

		tmpChar := user.Character
		user.Character = &char

		Status(``, user, room, flags)

		user.Character = tmpChar

		m := mobs.NewMobById(59, user.Character.RoomId)
		m.Character = char
		room.AddMob(m.InstanceId)
		m.Character.Charm(user.UserId, -1, `suicide vanish`)

		user.ClearPrompt()
		return true, nil

	}

	user.SendText(`<ansi fg="203">No character with the name <ansi fg="username">` + rawResponse + `</ansi> found.</ansi>`)

	user.ClearPrompt()
	return true, nil
}

func cmdCharacterHire(user *users.UserRecord, room *rooms.Room, cmdPrompt *prompt.Prompt, altNames []string, nameToAlt map[string]characters.Character, hiredOutChars map[string]characters.Character) (bool, error) {

	match, rawResponse, done := selectAltByName(cmdPrompt, altNames, `Enter the name of the character you wish to hire:`)
	if !done {
		return true, nil
	}

	if match != `` {

		char := nameToAlt[match]

		// Do they already have this mob hired??
		if friend, ok := hiredOutChars[char.Name]; ok && friend.Description == char.Description {
			user.SendText(fmt.Sprintf(`<ansi fg="mobname">%s</ansi> is already hired out.`, char.Name))
			user.ClearPrompt()
			return true, nil
		}

		char.Validate()

		gearValue := char.GetGearValue()

		charValue := gearValue + (250 * char.GetTotalSkillRanks())

		mudlog.Debug(`Hire Alt`, `UserId`, user.UserId, `alt-name`, char.Name, `gear-value`, gearValue, `skill-ranks`, char.GetTotalSkillRanks(), `total`, charValue)

		question := cmdPrompt.Ask(fmt.Sprintf(`<ansi fg="51">The price to hire <ansi fg="username">%s</ansi> is <ansi fg="gold">%d gold</ansi>. Are you sure?</ansi>`, char.Name, charValue), []string{`yes`, `no`}, `no`)
		if !question.Done {
			return true, nil
		}

		if question.Response != `yes` {
			user.ClearPrompt()
			return true, nil
		}

		if user.Character.Gold < charValue {
			user.SendText(fmt.Sprintf(`You only have <ansi fg="gold">%d gold</ansi> and it would cost <ansi fg="gold">%d gold</ansi> to hire <ansi fg="username">%s</ansi>.`, charValue, charValue, char.Name))
			user.ClearPrompt()
			return true, nil
		}

		// Prevent follower overage
		maxCharmed := user.Character.GetMaxCharmedCreatures()
		if len(hiredOutChars) >= maxCharmed {
			user.SendText(fmt.Sprintf(`You can only have %d mobs following you at a time.`, maxCharmed))
			user.ClearPrompt()
			return true, nil
		}

		user.Character.Gold -= charValue

		m := mobs.NewMobById(59, user.Character.RoomId)
		m.Character = char

		// To prevent dupes/exploits, clear vulnerable copied data
		m.Character.Items = []items.Item{}   // Clear items
		m.Character.Gold = 0                 // Clear gold
		m.Character.Bank = 0                 // Clear bank
		m.Character.Shop = characters.Shop{} // Clear shop

		m.Character.AddBuff(36, true) // Give a perma-gear buff, so that items can't be removed.

		room.AddMob(m.InstanceId)

		// Anti-recursion: strip any companions the hired mob had before charming.
		for _, subId := range m.Character.GetCharmIds() {
			if subMob := mobs.GetInstance(subId); subMob != nil {
				subMob.Character.RemoveCharm()
				if subRoom := rooms.LoadRoom(subMob.Character.RoomId); subRoom != nil {
					subRoom.RemoveMob(subId)
				}
				mobs.DestroyInstance(subId)
			}
		}
		m.Character.CharmedMobs = nil

		m.Character.Charm(user.UserId, -1, `suicide vanish`)
		user.Character.TrackCharmed(m.InstanceId, true)

		user.EventLog.Add(`char`, `Hired an alt character to help you out: <ansi fg="username">`+m.Character.Name+`</ansi>`)

		user.SendText(`<ansi fg="username">` + m.Character.Name + `</ansi> appears to help you out!`)
		room.SendText(`<ansi fg="username">`+m.Character.Name+`</ansi> appears to help <ansi fg="username">`+user.Character.Name+`</ansi>!`, user.UserId)

		m.Command(`emote waves sheepishly.`, 2)

		user.ClearPrompt()
		return true, nil

	}

	user.SendText(`<ansi fg="203">No character with the name <ansi fg="username">` + rawResponse + `</ansi> found.</ansi>`)

	user.ClearPrompt()
	return true, nil
}

func getAltTable(nameToAlt map[string]characters.Character, charmedChars map[string]characters.Character, viewingUserId int) string {

	headers := []string{"Name", "Species", "Title", "Status"}
	rows := [][]string{}

	for _, char := range nameToAlt {

		allRanks := char.GetAllSkillRanks()
		raceName := `Unknown`
		if speciesInfo := species.GetSpecies(char.SpeciesId); speciesInfo != nil {
			raceName = speciesInfo.Name
		}

		mobBusy := ``
		if c, ok := charmedChars[char.Name]; ok {
			if c.Description == char.Description {
				mobBusy = `<ansi fg="210">busy</ansi>`
			}
		}

		rows = append(rows, []string{
			fmt.Sprintf(`<ansi fg="username">%s</ansi>`, char.Name),
			raceName,
			skills.GetTitle(char.Mutations, allRanks, char.Stats),
			mobBusy,
		})

	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i][0] < rows[j][0]
	})

	altTableData := templates.GetTable(fmt.Sprintf(`Your alt characters (%d/%d)`, len(nameToAlt), configs.GetGamePlayConfig().MaxAltCharacters), headers, rows)
	tplTxt, _ := templates.Process("tables/generic", altTableData, viewingUserId)

	return tplTxt
}
