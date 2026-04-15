package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Consider(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(rest)

	// Looking AT something?
	if len(args) > 0 {
		lookAt := args[0]

		//
		// look for any mobs, players, npcs
		//

		playerId, mobId := room.FindByName(lookAt)
		if playerId == user.UserId {
			playerId = 0
		}

		if playerId > 0 || mobId > 0 {

			// Track perception use when considering a target
			user.Character.OnStatUse("perception", user.UserId)

			ratio := 0.0

			considerType := "mob"
			considerName := "nobody"

			if playerId > 0 {
				u := users.GetByUserId(playerId)

				p1 := combat.PowerScore(*user.Character)
				p2 := combat.PowerScore(*u.Character)
				if p2 > 0 {
					ratio = p1 / p2
				}
				considerType = "user"
				considerName = u.Character.Name

			} else if mobId > 0 {

				m := mobs.GetInstance(mobId)
				if m == nil {
					user.SendText("You don't see them here.")
					return true, nil
				}

				p1 := combat.PowerScore(*user.Character)
				p2 := combat.PowerScore(m.Character)
				if p2 > 0 {
					ratio = p1 / p2
				}
				considerType = "mob"
				considerName = m.Character.Name
			}

			prediction := `<ansi fg="red-bold">You will not survive this fight</ansi>`
			if ratio > 4 {
				prediction = `<ansi fg="blue-bold">They pose no threat to you</ansi>`
			} else if ratio > 3 {
				prediction = `<ansi fg="green">You hold a clear advantage</ansi>`
			} else if ratio > 2 {
				prediction = `<ansi fg="green">The odds favor you</ansi>`
			} else if ratio > 1 {
				prediction = `<ansi fg="yellow">An even contest — tread carefully</ansi>`
			} else if ratio > 0.5 {
				prediction = `<ansi fg="red-bold">They have the upper hand</ansi>`
			} else if ratio > 0 {
				prediction = `<ansi fg="red-bold">You are severely outmatched</ansi>`
			}

			user.SendText(
				fmt.Sprintf(`You consider <ansi fg="%sname">%s</ansi>...`, considerType, considerName),
			)
			user.SendText(
				fmt.Sprintf(`Your instincts tell you: %s`, prediction),
			)
		}
	}

	return true, nil
}
