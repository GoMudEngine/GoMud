package mobcommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Show(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	rest = util.StripPrepositions(rest)

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 2 {
		return true, nil
	}

	var showItem items.Item = items.Item{}
	var found bool = false

	var targetName string = args[len(args)-1]
	args = args[:len(args)-1]
	var objectName string = strings.Join(args, " ")

	// Check whether the user has an item in their inventory that matches
	showItem, found = mob.Character.FindInBackpack(objectName)

	if !found {
		return true, nil
	}

	target, err := actions.ResolveTargetActor(room, targetName)
	if err != nil {
		return true, nil
	}

	if target.IsPlayer() {

		targetUser := target.(*actions.UserActor).User
		mob.Character.CancelBuffsWithFlag(buffs.Hidden)

		// Swap the item location
		if showItem.ItemId > 0 {

			// Tell the Showee
			targetUser.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> shows you their <ansi fg="item">%s</ansi>.`, mob.Character.Name, showItem.DisplayName()),
			)

			targetUser.SendText(
				"\n" + showItem.GetLongDescription() + "\n",
			)

			// Tell the rest of the room
			room.SendTextVisual(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> shows their <ansi fg="item">%s</ansi> to <ansi fg="username">%s</ansi>.`, mob.Character.Name, showItem.DisplayName(), targetUser.Character.Name),
				targetUser.UserId)

		}

		return true, nil
	}

	//
	// Mob target
	//
	targetMob := target.(*actions.MobActor).Mob
	mob.Character.CancelBuffsWithFlag(buffs.Hidden)

	if showItem.ItemId > 0 {

		room.SendTextVisual(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> shows their <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, mob.Character.Name, showItem.DisplayName(), targetMob.Character.Name),
		)

	}

	return true, nil
}
