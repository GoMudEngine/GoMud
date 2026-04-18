package mobcommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Give(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	rest = util.StripPrepositions(rest)

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 2 {
		return true, nil
	}

	var giveWho string = args[len(args)-1]
	args = args[:len(args)-1]
	var giveWhat string = strings.Join(args, " ")

	var giveItem items.Item = items.Item{}
	var giveGoldAmount int = 0

	if len(giveWhat) > 4 && giveWhat[len(giveWhat)-4:] == "gold" {

		g, _ := strconv.ParseInt(giveWhat[0:len(giveWhat)-5], 10, 32)
		giveGoldAmount = int(g)

		if giveGoldAmount > mob.Character.Gold {
			return true, nil
		}

	} else {

		var found bool = false

		// Check whether the user has an item in their inventory that matches
		giveItem, found = mob.Character.FindInBackpack(giveWhat)

		if !found {
			return true, nil
		}

	}

	target, err := actions.ResolveTargetActor(room, giveWho)
	if err != nil {
		return true, nil
	}

	if target.IsPlayer() {

		targetUser := target.(*actions.UserActor).User
		mob.Character.CancelBuffsWithFlag(buffs.Hidden)

		// Swap the item location
		if giveItem.ItemId > 0 {
			mobActor := &actions.MobActor{Mob: mob, Room: room}
			result := actions.GiveItemToChar(mobActor, giveWhat, targetUser.Character, targetUser.UserId, 0)
			if result.Err != nil {
				return true, nil
			}

			targetUser.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> gives you their <ansi fg="item">%s</ansi>.`, mob.Character.Name, result.Item.DisplayName()),
			)

		} else if giveGoldAmount > 0 {

			mobActor := &actions.MobActor{Mob: mob, Room: room}
			if err := actions.GiveGoldToChar(mobActor, giveGoldAmount, targetUser.Character); err != nil {
				return true, nil
			}

			targetUser.SendText(
				fmt.Sprintf(`<ansi fg="mobname">%s</ansi> gives you <ansi fg="gold">%d gold</ansi>.`, mob.Character.Name, giveGoldAmount),
			)

		}

		return true, nil
	}

	//
	// Mob target
	//
	m := target.(*actions.MobActor).Mob
	mob.Character.CancelBuffsWithFlag(buffs.Hidden)

	// Swap the item location
	if giveItem.ItemId > 0 {
		mobActor := &actions.MobActor{Mob: mob, Room: room}
		result := actions.GiveItemToChar(mobActor, giveWhat, &m.Character, 0, m.InstanceId)
		if result.Err != nil {
			return true, nil
		}

		room.SendTextVisual(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> gave their <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, mob.Character.Name, result.Item.DisplayName(), m.Character.Name),
		)
	} else if giveGoldAmount > 0 {

		mobActor := &actions.MobActor{Mob: mob, Room: room}
		if err := actions.GiveGoldToChar(mobActor, giveGoldAmount, &m.Character); err != nil {
			return true, nil
		}

		room.SendTextVisual(
			fmt.Sprintf(`<ansi fg="mobname">%s</ansi> gave some gold to <ansi fg="mobname">%s</ansi>.`, mob.Character.Name, m.Character.Name),
		)
	}

	return true, nil
}
