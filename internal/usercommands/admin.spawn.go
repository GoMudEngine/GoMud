package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
* Role Permissions:
* spawn				(All)
 */
func Spawn(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {
		// send some sort of help info?
		infoOutput, _ := templates.Process("admincommands/help/command.spawn", nil, user.UserId)
		user.SendText(messaging.CategorySystem, infoOutput)
		return true, nil
	}

	spawnType := args[0]
	args = args[1:]

	spawnTarget := ``
	if len(args) == 1 {
		spawnTarget = args[0]
		args = args[1:]
	} else {
		spawnTarget = strings.Join(args, ` `)
		args = []string{}

	}

	if len(spawnTarget) > 0 {

		if spawnType == `container` {

			containerName := room.SpawnTempContainer(spawnTarget, "3 rounds", 0)

			user.SendText(messaging.CategorySystem, 
				fmt.Sprintf(`You wave your hands around and <ansi fg="container">%s</ansi> appears from thin air and falls to the ground.`, containerName),
			)
			room.SendTextVisual(messaging.CategoryMobEmote, 
				fmt.Sprintf(`<ansi fg="username">%s</ansi> waves their hands around and <ansi fg="container">%s</ansi> appears from thin air and falls to the ground.`, user.Character.Name, containerName),
				user.UserId,
			)

			return true, nil
		}

		if spawnType == `gold` || spawnTarget == `gold` {

			goldAmt := 0
			if spawnType == `gold` {
				goldAmt, _ = strconv.Atoi(spawnTarget)
			} else {
				goldAmt, _ = strconv.Atoi(spawnType)
			}

			if goldAmt < 1 {
				goldAmt = 1
			}

			room.Gold += goldAmt

			user.SendText(messaging.CategorySystem, 
				fmt.Sprintf(`You wave your hands around and <ansi fg="gold">%d gold</ansi> appears from thin air and falls to the ground.`, goldAmt),
			)
			room.SendTextVisual(messaging.CategoryMobEmote, 
				fmt.Sprintf(`<ansi fg="username">%s</ansi> waves their hands around and <ansi fg="gold">%d gold</ansi> appears from thin air and falls to the ground.`, user.Character.Name, goldAmt),
				user.UserId,
			)

			return true, nil
		}

	}

	user.SendText(messaging.CategorySystem, 
		"You wave your hands around pathetically.",
	)
	room.SendTextVisual(messaging.CategoryMobEmote, 
		fmt.Sprintf(`<ansi fg="username">%s</ansi> waves their hands around pathetically.`, user.Character.Name),
		user.UserId,
	)

	return true, nil
}
