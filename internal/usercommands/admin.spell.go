package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/util"

	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* spell 				(All)
* spell.create			(Create new spells)
* spell.list			(List all spells)
 */
func Spell(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(rest)

	if len(args) < 1 {
		infoOutput, _ := templates.Process("admincommands/help/command.spell", nil, user.UserId)
		user.SendText(infoOutput)
		return true, nil
	}

	// Create a new spell
	if args[0] == `create` {

		if !user.HasRolePermission(`spell.create`) {
			user.SendText(`you do not have <ansi fg="command">spell.create</ansi> permission`)
			return true, nil
		}

		return spell_Create(strings.TrimSpace(rest[6:]), user, room, flags)
	}

	// List existing spells
	if args[0] == `list` {

		if !user.HasRolePermission(`spell.list`) {
			user.SendText(`you do not have <ansi fg="command">spell.list</ansi> permission`)
			return true, nil
		}

		return spell_List(strings.TrimSpace(rest[4:]), user, room, flags)
	}

	return true, nil
}

func spell_List(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	spellNames := []templates.NameDescription{}

	for _, spellInfo := range spells.GetAllSpells() {

		// If searching for matches
		if len(rest) > 0 {
			if !strings.Contains(rest, `*`) {
				rest += `*`
			}
			if !util.StringWildcardMatch(strings.ToLower(spellInfo.Name), rest) && !util.StringWildcardMatch(strings.ToLower(spellInfo.Description), rest) {
				continue
			}
		}

		spellNames = append(spellNames, templates.NameDescription{
			Name:        spellInfo.Name,
			Description: spellInfo.Description,
		})
	}

	sort.SliceStable(spellNames, func(i, j int) bool {
		return spellNames[i].Name < spellNames[j].Name
	})

	tplTxt, _ := templates.Process("tables/numbered-list", spellNames, user.UserId)
	user.SendText(tplTxt)

	return true, nil
}

func spell_Create(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	var newSpell = spells.SpellData{}

	if len(rest) > 0 {
		if spellId := spells.FindSpell(rest); spellId != `` {
			newSpell = *(spells.GetSpell(spellId))
		}
	}

	// Get if already exists, otherwise create new
	cmdPrompt, isNew := user.StartPrompt(`spell create`, rest)

	if isNew {
		user.SendText(``)
		user.SendText(fmt.Sprintf(`Lets get a little info first.%s`, term.CRLFStr))
	}

	//
	// Name Selection
	//
	{
		question := cmdPrompt.Ask(`What will the spell be called?`, []string{newSpell.Name}, newSpell.Name)
		if !question.Done {
			return true, nil
		}

		if question.Response == `` {
			user.SendText("Aborting...")
			user.ClearPrompt()
			return true, nil
		}

		newSpell.Name = question.Response
	}

	//
	// Id Selection
	//
	{
		question := cmdPrompt.Ask(`What will the spell's unique id be (single-word)? This can also be used as an alias to cast the spell.`, []string{newSpell.SpellId}, newSpell.SpellId)
		if !question.Done {
			return true, nil
		}

		if question.Response == `` {
			user.SendText("Aborting...")
			user.ClearPrompt()
			return true, nil
		}

		newSpell.SpellId = question.Response
	}

	//
	// Description
	//
	{
		question := cmdPrompt.Ask(`Enter a SHORT description for the spell:`, []string{newSpell.Description}, newSpell.Description)
		if !question.Done {
			return true, nil
		}

		newSpell.Description = question.Response
	}

	//
	// Spell Type
	//
	{
		typeOptions := []templates.NameDescription{}

		typeOptions = append(typeOptions, templates.NameDescription{
			Name:        string(spells.Neutral),
			Description: `Does not harm or benefit a target`,
		})

		typeOptions = append(typeOptions, templates.NameDescription{
			Name:        string(spells.HelpSingle),
			Description: `Is beneficial for a single target`,
		})

		typeOptions = append(typeOptions, templates.NameDescription{
			Name:        string(spells.HelpMulti),
			Description: `Is beneficial for a group`,
		})

		typeOptions = append(typeOptions, templates.NameDescription{
			Name:        string(spells.HelpArea),
			Description: `Is beneficial for an entire room`,
		})

		typeOptions = append(typeOptions, templates.NameDescription{
			Name:        string(spells.HarmSingle),
			Description: `Is harmful for a single target`,
		})

		typeOptions = append(typeOptions, templates.NameDescription{
			Name:        string(spells.HarmMulti),
			Description: `Is harmful for a group`,
		})

		typeOptions = append(typeOptions, templates.NameDescription{
			Name:        string(spells.HarmArea),
			Description: `Is harmful for an entire room`,
		})

		sort.SliceStable(typeOptions, func(i, j int) bool {
			return typeOptions[i].Name < typeOptions[j].Name
		})

		question := cmdPrompt.Ask(`What kind of spell affect is it?`, []string{string(newSpell.Type)}, string(newSpell.Type))
		if !question.Done {
			tplTxt, _ := templates.Process("tables/numbered-list", typeOptions, user.UserId)
			user.SendText(tplTxt)
			return true, nil
		}

		if question.Response == `` {
			question.RejectResponse()

			tplTxt, _ := templates.Process("tables/numbered-list", typeOptions, user.UserId)
			user.SendText(tplTxt)
			return true, nil
		}

		newSpell.Type = spells.SpellType(question.Response)
	}

	//
	// Cost
	//
	{
		question := cmdPrompt.Ask(`Conviction cost of spell:`, []string{strconv.Itoa(newSpell.Cost)}, strconv.Itoa(newSpell.Cost))
		if !question.Done {
			return true, nil
		}

		newSpell.Cost, _ = strconv.Atoi(question.Response)
	}

	//
	// Rounds
	//
	{
		question := cmdPrompt.Ask(`Wait Rounds:`, []string{strconv.Itoa(newSpell.WaitRounds)}, strconv.Itoa(newSpell.WaitRounds))
		if !question.Done {
			return true, nil
		}

		newSpell.WaitRounds, _ = strconv.Atoi(question.Response)
	}

	//
	// Rounds
	//
	{
		question := cmdPrompt.Ask(`Casting Difficulty (0-100):`, []string{strconv.Itoa(newSpell.Difficulty)}, strconv.Itoa(newSpell.Difficulty))
		if !question.Done {
			return true, nil
		}

		newSpell.Difficulty, _ = strconv.Atoi(question.Response)
	}

	//
	// Confirm?
	//
	{
		question := cmdPrompt.Ask(`Does this look correct?`, []string{`y`, `n`}, `n`)
		if !question.Done {

			user.SendText(`  <ansi fg="yellow-bold">SpellId:</ansi>     <ansi fg="white-bold">` + newSpell.SpellId + `</ansi>`)
			user.SendText(`  <ansi fg="yellow-bold">Spell Name:</ansi>  <ansi fg="white-bold">` + newSpell.Name + `</ansi>`)
			user.SendText(`  <ansi fg="yellow-bold">Desc:</ansi>        <ansi fg="white-bold">` + newSpell.Description + `</ansi>`)
			user.SendText(`  <ansi fg="yellow-bold">Type:</ansi>        <ansi fg="white-bold">` + string(newSpell.Type) + `</ansi>`)
			user.SendText(`  <ansi fg="yellow-bold">Cost:</ansi>        <ansi fg="white-bold">` + strconv.Itoa(newSpell.Cost) + `</ansi>`)
			user.SendText(`  <ansi fg="yellow-bold">Wait:</ansi>        <ansi fg="white-bold">` + strconv.Itoa(newSpell.WaitRounds) + ` rounds</ansi>`)
			user.SendText(`  <ansi fg="yellow-bold">Difficulty:</ansi>  <ansi fg="white-bold">` + strconv.Itoa(newSpell.Difficulty) + `%</ansi>`)

			return true, nil
		}

		user.ClearPrompt()

		if question.Response != `y` {
			user.SendText("Aborting...")
			return true, nil
		}
	}

	spellId, err := spells.CreateNewSpellFile(newSpell)

	if err != nil {
		user.SendText("Error: " + err.Error())
		return true, nil
	}

	spellSpec := spells.GetSpell(spellId)

	user.SendText(``)
	user.SendText(`  <ansi bg="red" fg="white-bold">SPELL CREATED</ansi>`)
	user.SendText(``)
	user.SendText(`  <ansi fg="yellow-bold">File Path:</ansi>   <ansi fg="white-bold">` + spellSpec.Filepath() + `</ansi>`)
	user.SendText(`  <ansi fg="yellow-bold">Script Path:</ansi> <ansi fg="white-bold">` + spellSpec.GetScriptPath() + `</ansi>`)
	user.SendText(``)
	user.SendText(`  <ansi fg="black-bold">note: Try <ansi fg="command">???</ansi> to test it.`)

	return true, nil
}
