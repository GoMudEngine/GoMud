package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/util"

	"github.com/GoMudEngine/GoMud/internal/users"
)

func Ask(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Core "useful" commands
	usefulCommands := []string{
		`attack`,
		`give`,
		`get`,
		`drop`,
		`equip`,
		`remove`,
	}

	// Additional commands that are more for fun
	allowedCommands := []string{
		`say`,
		`look`,
		`emote`,
		`throw`,
		`eat`,
		`drink`,
	}

	// args should look like one of the following:
	// target buffId - put buff on target if in the room
	// buffId - put buff on self
	// search searchTerm - search for buff by name, display results
	args := util.SplitButRespectQuotes(rest)

	if len(args) < 2 {

		for _, mId := range room.GetMobs(rooms.FindCharmed) {
			mob := mobs.GetInstance(mId)
			if mob == nil {
				continue
			}
			if mob.Character.IsCharmed(user.UserId) {

				mob.Command(fmt.Sprintf(`say I can do a few useful things, such as %s`,
					fmt.Sprintf(`<ansi fg="command">%s</ansi>`, strings.Join(usefulCommands, `</ansi>, <ansi fg="command">`))))

				mob.Command(fmt.Sprintf(`say I can do some other stuff, like %s`,
					fmt.Sprintf(`<ansi fg="command">%s</ansi>`, strings.Join(allowedCommands, `</ansi>, <ansi fg="command">`))))

				return true, nil
			}
		}

		user.SendText(`You must <ansi fg="command">ask</ansi> <ansi fg="mobname">someone</ansi> <ansi fg="yellow">something</ansi>`)
		return true, nil
	}

	allowedCommands = append(allowedCommands, usefulCommands...)

	searchName := args[0]

	// Only ask charmed players or mobs to do stuff
	_, mobId := room.FindByName(searchName)

	if mobId > 0 {

		mob := mobs.GetInstance(mobId)
		if mob == nil {
			user.SendText(`Nobody found by that name`)
			return true, nil
		}

		args = args[1:]

		if !mob.Character.IsCharmed() {
			room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> asks <ansi fg="mobname">%s</ansi> about "%s"`, user.Character.Name, mob.Character.Name, strings.Join(args, ` `)), user.UserId)
		}

		// players may type "ask <mob> to <do something>"
		if len(args) > 1 && strings.ToLower(args[0]) == `to` {
			args = args[1:]
		}
		if len(args) > 1 && strings.ToLower(args[0]) == `about` {
			args = args[1:]
		}

		if mob.Character.IsCharmed(user.UserId) {

			mobCmd := args[0]
			askRest := strings.Join(args[1:], ` `)

			// If an alias was entered, conovert it
			mobCmd = keywords.TryCommandAlias(mobCmd)

			if mobCmd == `attack` {
				if pid, _ := room.FindByName(askRest); pid > 0 {

					if configs.GetGamePlayConfig().PVP != `enabled` {

						mob.Command(`emote shakes their head.`)
						mob.Command(`say PVP is currently disabled.`)

						return true, nil
					}
				}
			}

			// Check if actual command is allowed
			for _, allowedCmd := range allowedCommands {
				if mobCmd == allowedCmd {

					mob.Command(fmt.Sprintf(`%s %s`, mobCmd, askRest))

					return true, nil
				}
			}
		}

		rest = strings.Join(args, ` `)
		if handled, err := scripting.TryMobScriptEvent(`onAsk`, mobId, user.UserId, `user`, map[string]any{"askText": rest}); err == nil {
			if !handled {
				mob.Command(`emote shakes their head.`)
			}
		}

		// Track charisma use when asking an NPC
		user.Character.OnStatUse("charisma", user.UserId)

		room.SendTextToExits(`You hear someone talking.`, true)

	} else {

		user.SendText(`ask who what?`)

	}

	return true, nil
}
