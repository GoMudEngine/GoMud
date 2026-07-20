package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/util"

	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* item 				(All)
* item.create		(Create a new item)
* item.spawn		(Spawn a new item in the room)
 */
func Item(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(rest)

	if len(args) < 1 {
		infoOutput, _ := templates.Process("admincommands/help/command.item", nil, user.UserId)
		user.SendText(messaging.CategorySystem, infoOutput)
		return true, nil
	}

	// create a new item
	if args[0] == `create` {

		if !user.HasRolePermission(`item.create`) {
			user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">item.create</ansi> permission`)
			return true, nil
		}

		return item_Create(strings.TrimSpace(rest[6:]), user, room, flags)
	}

	// spawn an existing item
	if args[0] == `spawn` {

		if !user.HasRolePermission(`item.spawn`) {
			user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">item.create</ansi> permission`)
			return true, nil
		}

		return item_Spawn(strings.TrimSpace(rest[5:]), user, room, flags)
	}

	// List existing items
	if args[0] == `list` {

		if !user.HasRolePermission(`item.spawn`) {
			user.SendText(messaging.CategorySystem, `you do not have <ansi fg="command">item.create</ansi> permission`)
			return true, nil
		}

		return item_List(strings.TrimSpace(rest[4:]), user, room, flags)
	}

	return true, nil
}

func item_List(rest string, user *users.UserRecord, _ *rooms.Room, _ events.EventFlag) (bool, error) {

	itmNames := []templates.NameDescription{}
	longestName := 0

	for _, itm := range items.GetAllItemSpecs() {

		entry := templates.NameDescription{
			Id:   itm.ItemId,
			Name: itm.Name,
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

		itmNames = append(itmNames, entry)
	}

	sort.SliceStable(itmNames, func(i, j int) bool {
		return strings.ToLower(itmNames[i].Name) < strings.ToLower(itmNames[j].Name)
	})

	numWidth := len(strconv.Itoa(len(itmNames)))
	colWidth := 1 + numWidth + 2 + longestName + 1

	user.SendText(messaging.CategorySystem, ``)
	sw := user.ClientSettings().Display.GetScreenWidth()
	strOut := templates.DynamicList(itmNames, colWidth, sw, numWidth, longestName)
	user.SendText(messaging.CategorySystem, strOut)
	user.SendText(messaging.CategorySystem, ``)

	return true, nil
}

func item_Spawn(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	itemId := items.FindItem(rest)
	if itemId != 0 {

		itm := items.New(itemId)
		if itm.ItemId > 0 {
			room.AddItem(itm, false)

			user.SendText(messaging.CategorySystem,
				fmt.Sprintf(`You wave your hands around and <ansi fg="item">%s</ansi> appears from thin air and falls to the ground.`, itm.DisplayName()),
			)
			room.SendTextVisual(messaging.CategoryMobEmote,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> waves their hands around and <ansi fg="item">%s</ansi> appears from thin air and falls to the ground.`, user.Character.Name, itm.DisplayName()),
				user.UserId,
			)

			return true, nil
		}

	}

	user.SendText(messaging.CategorySystem,
		fmt.Sprintf(`Item <ansi fg="itemname">%s</ansi> not found.`, rest),
	)

	return true, nil
}

func item_Create(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	var newItemSpec = items.ItemSpec{}

	if len(rest) > 0 {
		if itemId := items.FindItem(rest); itemId > 0 {
			newItemSpec = *(items.GetItemSpec(itemId))
		}
	}

	// Get if already exists, otherwise create new
	cmdPrompt, isNew := user.StartPrompt(`item create`, rest)

	if isNew {
		user.SendText(messaging.CategorySystem, ``)
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`Lets get a little info first.%s`, term.CRLFStr))
	}

	//
	// Name Selection
	//
	{

		question := cmdPrompt.Ask(`What will the item be called?`, []string{newItemSpec.Name}, newItemSpec.Name)
		if !question.Done {
			return true, nil
		}

		newItemSpec.Name = question.Response

	}

	//
	// Type Selection
	//
	{
		allTypes := items.ItemTypes()

		typeOptions := []templates.NameDescription{}
		for _, t := range allTypes {
			typeOptions = append(typeOptions, templates.NameDescription{
				Name:        t.Type,
				Description: t.Description,
			})
		}

		question := cmdPrompt.Ask(`What Type of item will it be?`, []string{string(newItemSpec.Type)}, string(newItemSpec.Type))
		if !question.Done {
			tplTxt, _ := templates.Process("tables/numbered-list", typeOptions, user.UserId)
			user.SendText(messaging.CategorySystem, tplTxt)
			return true, nil
		}

		typeNameSelection := question.Response
		if restNum, err := strconv.Atoi(typeNameSelection); err == nil {
			if restNum > 0 && restNum <= len(typeOptions) {
				typeNameSelection = typeOptions[restNum-1].Name
			}
		}

		for _, t := range allTypes {
			if strings.EqualFold(t.Type, typeNameSelection) {
				newItemSpec.Type = items.ItemType(t.Type)
			}
		}

		if newItemSpec.Type == `` {
			question.RejectResponse()

			tplTxt, _ := templates.Process("tables/numbered-list", typeOptions, user.UserId)
			user.SendText(messaging.CategorySystem, tplTxt)

			return true, nil
		}
	}

	//
	// Damage (if weapon)
	//
	if newItemSpec.Type == items.Weapon {

		question := cmdPrompt.Ask(`What damage does this weapon do (Example: 1d4)?`, []string{newItemSpec.Damage.DiceRoll}, newItemSpec.Damage.DiceRoll)
		if !question.Done {
			return true, nil
		}

		if question.Response != `` {
			newItemSpec.Damage.InitDiceRoll(question.Response)
			newItemSpec.Damage.DiceRoll = newItemSpec.Damage.FormatDiceRoll()
		}

		// Damage multiplier for unified pipeline
		dmDefault := fmt.Sprintf(`%.2f`, newItemSpec.DamageMultiplier)
		if newItemSpec.DamageMultiplier <= 0 {
			dmDefault = `1.00`
		}
		dmQuestion := cmdPrompt.Ask(`Damage multiplier (0.15=fists, 1.0=iron, 2.5=legendary)?`, []string{dmDefault}, dmDefault)
		if !dmQuestion.Done {
			return true, nil
		}
		if val, err := strconv.ParseFloat(dmQuestion.Response, 64); err == nil && val > 0 {
			newItemSpec.DamageMultiplier = val
		}
	}

	//
	// Mitigation (if armor/equipment)
	//
	isArmorType := newItemSpec.Type == items.Head || newItemSpec.Type == items.Body ||
		newItemSpec.Type == items.Legs || newItemSpec.Type == items.Feet ||
		newItemSpec.Type == items.Gloves || newItemSpec.Type == items.Neck ||
		newItemSpec.Type == items.Ring || newItemSpec.Type == items.Belt ||
		newItemSpec.Type == items.Offhand
	if isArmorType {
		pmDefault := strconv.Itoa(newItemSpec.PhysicalMitigation)
		pmQ := cmdPrompt.Ask(`Physical mitigation % (plate:12-18, leather:4-8, cloth:1-3)?`, []string{pmDefault}, pmDefault)
		if !pmQ.Done {
			return true, nil
		}
		newItemSpec.PhysicalMitigation, _ = strconv.Atoi(pmQ.Response)

		mmDefault := strconv.Itoa(newItemSpec.MagicalMitigation)
		mmQ := cmdPrompt.Ask(`Magical mitigation % (robes:5-12, amulet:3-8, plate:0-2)?`, []string{mmDefault}, mmDefault)
		if !mmQ.Done {
			return true, nil
		}
		newItemSpec.MagicalMitigation, _ = strconv.Atoi(mmQ.Response)

		cmDefault := strconv.Itoa(newItemSpec.ConvictionMitigation)
		cmQ := cmdPrompt.Ask(`Conviction mitigation % (willpower items:3-8, most armor:0)?`, []string{cmDefault}, cmDefault)
		if !cmQ.Done {
			return true, nil
		}
		newItemSpec.ConvictionMitigation, _ = strconv.Atoi(cmQ.Response)
	}

	//
	// Target room/exit/container (If key)
	//
	if newItemSpec.Type == items.Key {

		roomIdStr := ``
		roomExitStr := ``

		keyParts := strings.Split(newItemSpec.KeyLockId, `-`)
		if len(keyParts) == 2 {
			roomIdStr = keyParts[0]
			roomExitStr = keyParts[1]
		}

		question := cmdPrompt.Ask(`What Room Id will this key be used in?`, []string{roomIdStr}, roomIdStr)
		if !question.Done {
			return true, nil
		}

		if question.Response == `` {
			user.SendText(messaging.CategorySystem, "Aborting...")
			user.ClearPrompt()
			return true, nil
		}

		roomId, _ := strconv.Atoi(question.Response)
		if roomId == 0 {
			question.RejectResponse()
			return true, nil
		}

		question = cmdPrompt.Ask(`What exit name or container will this open?`, []string{roomExitStr}, roomExitStr)
		if !question.Done {
			return true, nil
		}

		if question.Response == `` {
			user.SendText(messaging.CategorySystem, "Aborting...")
			user.ClearPrompt()
			return true, nil
		}

		newItemSpec.KeyLockId = fmt.Sprintf(`%d-%s`, roomId, strings.ToLower(question.Response))
	}

	//
	// SubType Selection
	//
	{
		allSubTypes := items.ItemSubtypes()

		subTypeOptions := []templates.NameDescription{}
		for _, t := range allSubTypes {
			subTypeOptions = append(subTypeOptions, templates.NameDescription{
				Name:        t.Type,
				Description: t.Description,
			})
		}

		question := cmdPrompt.Ask(`What Subtype of item will it be?`, []string{string(newItemSpec.Subtype)}, string(newItemSpec.Subtype))
		if !question.Done {
			tplTxt, _ := templates.Process("tables/numbered-list", subTypeOptions, user.UserId)
			user.SendText(messaging.CategorySystem, tplTxt)
			return true, nil
		}

		if question.Response == `` {
			user.SendText(messaging.CategorySystem, "Aborting...")
			user.ClearPrompt()
			return true, nil
		}

		typeNameSelection := question.Response
		if restNum, err := strconv.Atoi(typeNameSelection); err == nil {
			if restNum > 0 && restNum <= len(subTypeOptions) {
				typeNameSelection = subTypeOptions[restNum-1].Name
			}
		}

		for _, t := range allSubTypes {
			if strings.EqualFold(t.Type, typeNameSelection) {
				newItemSpec.Subtype = items.ItemSubType(t.Type)
			}
		}

		if newItemSpec.Subtype == `` {
			question.RejectResponse()

			tplTxt, _ := templates.Process("tables/numbered-list", subTypeOptions, user.UserId)
			user.SendText(messaging.CategorySystem, tplTxt)

			return true, nil
		}
	}

	//
	// Maximum Uses
	//
	{
		defaultYN := `n`
		if newItemSpec.Uses > 0 {
			defaultYN = `y`
		}
		question := cmdPrompt.Ask(`Will this item have a maximum number of uses?`, []string{`y`, `n`}, defaultYN)
		if !question.Done {
			return true, nil
		}

		if question.Response == `y` {
			question := cmdPrompt.Ask(`How many uses will this item have?`, []string{strconv.Itoa(newItemSpec.Uses)}, strconv.Itoa(newItemSpec.Uses))
			if !question.Done {
				return true, nil
			}

			newItemSpec.Uses, _ = strconv.Atoi(question.Response)
		}

	}

	//
	// Description
	//
	{
		qToken := newItemSpec.QuestToken
		qTokenDefault := qToken
		if qToken == `` {
			qToken = `_`
		}

		question := cmdPrompt.Ask(`Quest token given if acquired (if any):`, []string{qTokenDefault}, qToken)
		if !question.Done {
			return true, nil
		}

		if question.Response != `_` {

			if quests.GetQuest(question.Response) == nil {
				user.SendText(messaging.CategorySystem, `Invalid quest token`)
				question.RejectResponse()
				return true, nil
			}

			newItemSpec.QuestToken = question.Response
		}
	}

	//
	// Name Selection
	//
	{
		question := cmdPrompt.Ask(`Gold value override:`, []string{strconv.Itoa(newItemSpec.Value)}, strconv.Itoa(newItemSpec.Value))
		if !question.Done {
			return true, nil
		}
		newItemSpec.Value, _ = strconv.Atoi(question.Response)
	}

	//
	// Description
	//
	{
		question := cmdPrompt.Ask(`Enter a description for the item:`, []string{newItemSpec.Description}, newItemSpec.Description)
		if !question.Done {
			return true, nil
		}

		newItemSpec.Description = question.Response
	}

	//
	// Confirm?
	//
	{
		question := cmdPrompt.Ask(`Does this look correct?`, []string{`y`, `n`}, `n`)
		if !question.Done {

			user.SendText(messaging.CategorySystem, `  <ansi fg="yellow-bold">Name:</ansi>        <ansi fg="white-bold">`+newItemSpec.Name+`</ansi>`)
			user.SendText(messaging.CategorySystem, `  <ansi fg="yellow-bold">Desc:</ansi>        <ansi fg="white-bold">`+newItemSpec.Description+`</ansi>`)
			user.SendText(messaging.CategorySystem, `  <ansi fg="yellow-bold">Type:</ansi>        <ansi fg="white-bold">`+string(newItemSpec.Type)+`</ansi>`)

			if newItemSpec.Type == items.Key {
				user.SendText(messaging.CategorySystem, `  <ansi fg="yellow-bold">KeyId:</ansi>       <ansi fg="white-bold">`+newItemSpec.KeyLockId+`</ansi>`)
			}
			if newItemSpec.Type == items.Weapon {
				user.SendText(messaging.CategorySystem, `  <ansi fg="yellow-bold">Damage:</ansi>      <ansi fg="white-bold">`+newItemSpec.Damage.DiceRoll+`</ansi>`)
				user.SendText(messaging.CategorySystem, `  <ansi fg="yellow-bold">DmgMult:</ansi>     <ansi fg="white-bold">`+fmt.Sprintf(`%.2f`, newItemSpec.DamageMultiplier)+`</ansi>`)
			}
			if newItemSpec.PhysicalMitigation > 0 || newItemSpec.MagicalMitigation > 0 || newItemSpec.ConvictionMitigation > 0 {
				user.SendText(messaging.CategorySystem, `  <ansi fg="yellow-bold">Physical:</ansi>    <ansi fg="white-bold">`+strconv.Itoa(newItemSpec.PhysicalMitigation)+`%</ansi>`)
				user.SendText(messaging.CategorySystem, `  <ansi fg="yellow-bold">Magical:</ansi>     <ansi fg="white-bold">`+strconv.Itoa(newItemSpec.MagicalMitigation)+`%</ansi>`)
				user.SendText(messaging.CategorySystem, `  <ansi fg="yellow-bold">Conviction:</ansi>  <ansi fg="white-bold">`+strconv.Itoa(newItemSpec.ConvictionMitigation)+`%</ansi>`)
			}

			if newItemSpec.Uses > 0 {
				user.SendText(messaging.CategorySystem, `  <ansi fg="yellow-bold">Uses:</ansi>        <ansi fg="white-bold">`+strconv.Itoa(newItemSpec.Uses)+`</ansi>`)
			}

			if newItemSpec.Value > 0 {
				user.SendText(messaging.CategorySystem, `  <ansi fg="yellow-bold">Value:</ansi>       <ansi fg="white-bold">`+strconv.Itoa(newItemSpec.Value)+`</ansi>`)
			}

			if newItemSpec.QuestToken != `` {
				user.SendText(messaging.CategorySystem, `  <ansi fg="yellow-bold">Quest Token:</ansi> <ansi fg="white-bold">`+newItemSpec.QuestToken+`</ansi>`)
			}

			user.SendText(messaging.CategorySystem, `  <ansi fg="yellow-bold">SubType:</ansi>     <ansi fg="white-bold">`+string(newItemSpec.Subtype)+`</ansi>`)

			return true, nil
		}

		user.ClearPrompt()

		if question.Response != `y` {
			user.SendText(messaging.CategorySystem, "Aborting...")
			return true, nil
		}
	}

	newItemId, err := items.CreateNewItemFile(newItemSpec)

	if err != nil {
		user.SendText(messaging.CategorySystem, "Error: "+err.Error())
		return true, nil
	}

	itemInst := items.GetItemSpec(newItemId)

	user.SendText(messaging.CategorySystem, ``)
	user.SendText(messaging.CategorySystem, `  <ansi bg="red" fg="white-bold">ITEM CREATED</ansi>`)
	user.SendText(messaging.CategorySystem, ``)
	user.SendText(messaging.CategorySystem, `  <ansi fg="yellow-bold">File Path:</ansi>   <ansi fg="white-bold">`+itemInst.Filepath()+`</ansi>`)
	user.SendText(messaging.CategorySystem, ``)
	user.SendText(messaging.CategorySystem, `  <ansi fg="black-bold">note: Try <ansi fg="command">item spawn `+newItemSpec.Name+`</ansi> to test it.`)

	return true, nil
}
