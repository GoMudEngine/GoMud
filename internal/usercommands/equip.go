package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Equip(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if rest == "all" {
		return Gearup(``, user, room, flags)
	}

	if rest == "" {
		user.SendText(`Wear WHAT?`)
		return true, nil
	}

	// Check for "arm1" / "arm2" / "arm3" / "arm4" suffix for extra arm slot targeting
	targetArmSlot := 0
	restLower := strings.ToLower(rest)
	if strings.HasSuffix(restLower, " arm1") {
		targetArmSlot = 1
		rest = strings.TrimSpace(rest[:len(rest)-5])
	} else if strings.HasSuffix(restLower, " arm2") {
		targetArmSlot = 2
		rest = strings.TrimSpace(rest[:len(rest)-5])
	} else if strings.HasSuffix(restLower, " arm3") {
		targetArmSlot = 3
		rest = strings.TrimSpace(rest[:len(rest)-5])
	} else if strings.HasSuffix(restLower, " arm4") {
		targetArmSlot = 4
		rest = strings.TrimSpace(rest[:len(rest)-5])
	}

	// Check whether the user has an item in their inventory that matches
	matchItem, found := user.Character.FindInBackpack(rest)

	if !found {
		user.SendText(fmt.Sprintf(`You don't have a "%s" to wear.`, rest))
	} else {

		iSpec := matchItem.GetSpec()
		if iSpec.Type != items.Weapon && iSpec.Subtype != items.Wearable {
			user.SendText(
				fmt.Sprintf(`Your <ansi fg="item">%s</ansi> doesn't look very fashionable.`, matchItem.DisplayName()),
			)
			return true, nil
		}

		// Handle direct extra arm slot equip
		if targetArmSlot > 0 {
			if iSpec.Type != items.Weapon {
				user.SendText(`You can only wield weapons in extra arms.`)
				return true, nil
			}
			if user.Character.ExtraArms < targetArmSlot {
				user.SendText(fmt.Sprintf(`You don't have enough extra arms for arm slot %d.`, targetArmSlot))
				return true, nil
			}
			handsReq := user.Character.HandsRequired(matchItem)
			if handsReq > 1 {
				user.SendText(`You can only wield one-handed weapons in extra arms.`)
				return true, nil
			}

			var oldItem items.Item
			switch targetArmSlot {
			case 1:
				oldItem = user.Character.Equipment.ExtraArm1
				user.Character.Equipment.ExtraArm1 = matchItem
			case 2:
				oldItem = user.Character.Equipment.ExtraArm2
				user.Character.Equipment.ExtraArm2 = matchItem
			case 3:
				oldItem = user.Character.Equipment.ExtraArm3
				user.Character.Equipment.ExtraArm3 = matchItem
			case 4:
				oldItem = user.Character.Equipment.ExtraArm4
				user.Character.Equipment.ExtraArm4 = matchItem
			}

			user.Character.CancelBuffsWithFlag(buffs.Hidden)
			user.Character.RemoveItem(matchItem)

			if oldItem.ItemId > 0 {
				user.SendText(fmt.Sprintf(`You remove your <ansi fg="item">%s</ansi> from extra arm %d and return it to your backpack.`, oldItem.DisplayName(), targetArmSlot))
				user.Character.StoreItem(oldItem)
			}

			user.SendText(fmt.Sprintf(`You wield your <ansi fg="item">%s</ansi> in extra arm %d.`, matchItem.DisplayName(), targetArmSlot))
			room.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> wields their <ansi fg="item">%s</ansi> in an extra arm.`, user.Character.Name, matchItem.DisplayName()),
				user.UserId,
			)

			user.Character.Validate(true)
			events.AddToQueue(events.EquipmentChange{
				UserId:       user.UserId,
				ItemsWorn:    []items.Item{matchItem},
				ItemsRemoved: []items.Item{oldItem},
			})

			// Quest engine: command notification
			bridge := questengine.NewGameBridge(user, room.RoomId)
			questengine.GetEngine().Notify("command", questengine.EventDetails{
				UserId:  user.UserId,
				RoomId:  room.RoomId,
				Command: "equip",
			}, bridge, bridge)

			return true, nil
		}

		// Delegate core equip logic to shared action.
		actor := &actions.UserActor{User: user, Room: room}
		result := actions.EquipItem(actor, rest)

		if result.Equipped {

			for _, oldItem := range result.DisplacedItems {
				if oldItem.ItemId != 0 {
					user.SendText(
						fmt.Sprintf(`You remove your <ansi fg="item">%s</ansi> and return it to your backpack.`, oldItem.DisplayName()),
					)
					room.SendText(
						fmt.Sprintf(`<ansi fg="username">%s</ansi> removes their <ansi fg="item">%s</ansi> and stores it away.`, user.Character.Name, oldItem.DisplayName()),
						user.UserId,
					)
				}
			}

			if result.Item.GetSpec().Subtype == items.Wearable {
				user.SendText(
					fmt.Sprintf(`You wear your <ansi fg="item">%s</ansi>.`, result.Item.DisplayName()),
				)
				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> puts on their <ansi fg="item">%s</ansi>.`, user.Character.Name, result.Item.DisplayName()),
					user.UserId,
				)
			} else {
				user.SendText(
					fmt.Sprintf(`You wield your <ansi fg="item">%s</ansi>. You're feeling dangerous.`, result.Item.DisplayName()),
				)
				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> wields their <ansi fg="item">%s</ansi>.`, user.Character.Name, result.Item.DisplayName()),
					user.UserId,
				)
			}

			// Trigger any outstanding buff onStart events
			if len(result.Item.GetSpec().WornBuffIds) > 0 {
				for _, buff := range user.Character.Buffs.List {
					if buff.OnStartWaiting {
						if _, err := scripting.TryBuffScriptEvent(`onStart`, user.UserId, 0, buff.BuffId); err == nil {
							user.Character.TrackBuffStarted(buff.BuffId)
						}
					}
				}
			}

			// Quest engine: command notification
			bridge := questengine.NewGameBridge(user, room.RoomId)
			questengine.GetEngine().Notify("command", questengine.EventDetails{
				UserId:  user.UserId,
				RoomId:  room.RoomId,
				Command: "equip",
			}, bridge, bridge)

		} else if result.Found {
			failureReason := result.FailureReason
			if len(failureReason) <= 1 {
				failureReason = fmt.Sprintf(`You can't figure out how to equip the <ansi fg="item">%s</ansi>.`, matchItem.DisplayName())
			}
			user.SendText(failureReason)
		}

	}

	return true, nil
}
