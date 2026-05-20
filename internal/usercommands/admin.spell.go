package usercommands

import (
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/util"

	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
* Role Permissions:
* spell 			(All)
* spell.list		(List all spells)
 */
func Spell(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(rest)

	if len(args) < 1 {
		infoOutput, _ := templates.Process("admincommands/help/command.spell", nil, user.UserId)
		user.SendTextLegacy(infoOutput)
		return true, nil
	}

	// List existing spells
	if args[0] == `list` {

		if !user.HasRolePermission(`spell.list`) {
			user.SendTextLegacy(`you do not have <ansi fg="command">spell.list</ansi> permission`)
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
	user.SendTextLegacy(tplTxt)

	return true, nil
}
