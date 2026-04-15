package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Eat(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Check whether the user has an item in their inventory that matches
	matchItem, found := user.Character.FindInBackpack(rest)

	if !found {
		user.SendText(fmt.Sprintf(`You don't have a "%s" to eat.`, rest))
	} else {

		itemSpec := matchItem.GetSpec()

		if itemSpec.Subtype != items.Edible {
			user.SendText(
				fmt.Sprintf(`You can't eat <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()),
			)
			return true, nil
		}

		// Check if food has spoiled
		if itemSpec.Aging.HasAging() && matchItem.CraftedRound > 0 {
			currentRound := util.GetRoundCount()
			var elapsed uint64
			if currentRound >= matchItem.CraftedRound {
				elapsed = currentRound - matchItem.CraftedRound
			}
			effSpeed := items.CalcEffectiveAgingSpeed(1.0, matchItem.CraftSkill) // food has no bottle
			phase, _ := items.GetAgingPhase(elapsed, itemSpec.Aging, effSpeed)
			if phase == items.PhaseSpoiled {
				user.SendText(`<ansi fg="red">The food has gone bad! It reeks of decay and is clearly inedible.</ansi>`)
				return true, nil
			}
		}

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		user.SendText(fmt.Sprintf(`You eat some of the <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()))
		room.SendTextVisual(fmt.Sprintf(`<ansi fg="username">%s</ansi> eats some <ansi fg="itemname">%s</ansi>.`, user.Character.Name, matchItem.DisplayName()), user.UserId)

		// If no more uses, will be lost, so trigger event
		if usesLeft := user.Character.UseItem(matchItem); usesLeft < 1 {

			events.AddToQueue(events.ItemOwnership{
				UserId: user.UserId,
				Item:   matchItem,
				Gained: false,
			})

		}

		for _, buffId := range itemSpec.BuffIds {
			user.AddBuff(buffId, `food`)
		}

	}

	return true, nil
}
