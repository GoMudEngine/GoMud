package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Use(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	containerName := room.FindContainerByName(rest)
	if containerName != `` {
		if c, exists := room.Containers[containerName]; exists && c.Hidden {
			if user == nil || !user.Character.HasDiscovery(room.RoomId, containerName) {
				containerName = ``
			}
		}
	}
	if containerName != `` {

		container := room.Containers[containerName]

		if len(container.Recipes) > 0 {

			if container.Lock.IsLocked() {
				user.SendTextLegacy(``)
				user.SendTextLegacy(fmt.Sprintf(`The <ansi fg="container">%s</ansi> is locked.`, containerName))
				user.SendTextLegacy(``)
				return true, nil
			}

			recipeReadyItemId := container.RecipeReady()

			if recipeReadyItemId == 0 {
				user.SendTextLegacy("")
				user.SendTextLegacy(fmt.Sprintf(`The <ansi fg="container">%s</ansi> seems to be missing something.`, containerName))
				user.SendTextLegacy("")
				return true, nil
			}

			for _, removeItem := range container.Recipes[recipeReadyItemId] {
				if matchItem, found := container.FindItemById(removeItem); found {
					container.RemoveItem(matchItem)
				}
			}

			newItem := items.New(recipeReadyItemId)

			container.AddItem(newItem)
			room.Containers[containerName] = container

			room.PlaySound(`change`, `other`)

			user.SendTextLegacy(``)
			user.SendTextLegacy(fmt.Sprintf(`The <ansi fg="container">%s</ansi> produces a <ansi fg="itemname">%s</ansi>!`, containerName, newItem.DisplayName()))
			user.SendTextLegacy(``)

			room.SendTextVisualLegacy(fmt.Sprintf(`<ansi fg="username">%s</ansi> does something with the <ansi fg="container">%s</ansi>.`, user.Character.Name, containerName), user.UserId)

			return true, nil

		}

	}

	// Check whether the user has an item in their inventory that matches
	matchItem, found := user.Character.FindInBackpack(rest)

	if !found {
		user.SendTextLegacy(fmt.Sprintf(`You don't have a "%s" to use.`, rest))
	} else {

		itemSpec := matchItem.GetSpec()

		if itemSpec.Subtype != items.Usable {
			user.SendTextLegacy(
				fmt.Sprintf(`You can't use <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()))
			return true, nil
		}

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		user.SendTextLegacy(fmt.Sprintf(`You use the <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()))
		room.SendTextVisualLegacy(fmt.Sprintf(`<ansi fg="username">%s</ansi> uses their <ansi fg="itemname">%s</ansi>.`, user.Character.Name, matchItem.DisplayName()), user.UserId)

		// YAML-driven use effects (replaces JS onUse for simple items)
		if itemSpec.OnUseTrainSkill != "" {
			trainAmount := itemSpec.OnUseTrainAmount
			if trainAmount < 1 {
				trainAmount = 1
			}
			user.Character.TrainSkill(itemSpec.OnUseTrainSkill, trainAmount)
			if itemSpec.OnUseUserText != "" {
				user.SendTextLegacy(itemSpec.OnUseUserText)
			}
			if itemSpec.OnUseRoomText != "" {
				room.SendTextVisualLegacy(itemSpec.OnUseRoomText, user.UserId)
			}
		}

		// If no more uses, will be lost, so trigger event
		if usesLeft := user.Character.UseItem(matchItem); usesLeft < 1 {

			events.AddToQueue(events.ItemOwnership{
				UserId: user.UserId,
				Item:   matchItem,
				Gained: false,
			})

		}

		for _, buffId := range itemSpec.BuffIds {
			user.AddBuff(buffId, `item`)
		}
	}

	return true, nil
}
