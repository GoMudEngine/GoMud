package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Show(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	rest = util.StripPrepositions(rest)

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 2 {
		user.SendText(messaging.CategorySystem, "Show what? To whom?")
		return true, nil
	}

	var showItem items.Item = items.Item{}
	var found bool = false

	var targetName string = args[len(args)-1]
	args = args[:len(args)-1]
	var objectName string = strings.Join(args, " ")

	// Check whether the user has an item in their inventory that matches
	showItem, found = user.Character.FindInBackpack(objectName)

	if !found {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("You don't have a %s to show.", objectName))
		return true, nil
	}

	target, err := actions.ResolveTargetActor(room, targetName)
	if err != nil {
		user.SendText(messaging.CategorySystem, "Who???")
		return true, nil
	}

	user.Character.CancelBuffsWithFlag(buffs.Hidden)

	if showItem.ItemId == 0 {
		user.SendText(messaging.CategorySystem, "Something went wrong.")
		return true, nil
	}

	if target.IsPlayer() {

		targetUser := target.(*actions.UserActor).User

		// Tell the shower
		user.SendText(messaging.CategorySystem,
			fmt.Sprintf(`You show the <ansi fg="item">%s</ansi> to <ansi fg="username">%s</ansi>.`, showItem.DisplayName(), targetUser.Character.Name),
		)

		// Tell the Showee
		targetUser.SendText(messaging.CategorySystem,
			fmt.Sprintf(`<ansi fg="username">%s</ansi> shows you their <ansi fg="item">%s</ansi>.`, user.Character.Name, showItem.DisplayName()),
		)

		targetUser.SendText(messaging.CategorySystem,
			"\n"+showItem.GetLongDescription()+"\n",
		)

		// Tell the rest of the room
		room.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="username">%s</ansi> shows their <ansi fg="item">%s</ansi> to <ansi fg="username">%s</ansi>.`, user.Character.Name, showItem.DisplayName(), targetUser.Character.Name),
			targetUser.UserId,
			user.UserId)

	} else {

		targetMob := target.(*actions.MobActor).Mob

		user.SendText(messaging.CategorySystem,
			fmt.Sprintf(`You show the <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, showItem.DisplayName(), targetMob.Character.Name),
		)

		room.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="username">%s</ansi> shows their <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, user.Character.Name, showItem.DisplayName(), targetMob.Character.Name),
			user.UserId,
		)
	}

	return true, nil
}
