package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
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

		target, err := actions.ResolveTargetActor(room, lookAt, actions.ResolveTargetOptions{
			ExcludeUserId: user.UserId,
		})
		if err != nil {
			// Pre-migration silently no-oped on no-match (no else branch).
			// On ErrTargetVanished (stale mob ID) the original code DID
			// message "You don't see them here." — preserve that.
			if err == actions.ErrTargetVanished {
				user.SendText("You don't see them here.")
			}
			return true, nil
		}

		// Track perception use when considering a target
		user.Character.OnStatUse("perception", user.UserId)

		p1 := combat.PowerScore(*user.Character)
		var p2 float64
		var considerType, considerName string
		if target.IsPlayer() {
			u := target.(*actions.UserActor).User
			p2 = combat.PowerScore(*u.Character)
			considerType = "user"
			considerName = u.Character.Name
		} else {
			m := target.(*actions.MobActor).Mob
			p2 = combat.PowerScore(m.Character)
			considerType = "mob"
			considerName = m.Character.Name
		}

		ratio := 0.0
		if p2 > 0 {
			ratio = p1 / p2
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

	return true, nil
}
