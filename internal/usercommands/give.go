package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Give(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	rest = util.StripPrepositions(rest)

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 2 {
		user.SendText(`Give what? To whom? (<ansi fg="command">give {object-name} {receiver-name}</ansi>)`)
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

		if giveGoldAmount < 0 {
			user.SendText("You can't give a negative amount of gold.")
			return true, nil
		}

		if giveGoldAmount > user.Character.Gold {
			user.SendText("You don't have that much gold to give.")
			return true, nil
		}

	} else {

		var found bool = false

		// Check whether the user has an item in their inventory that matches
		giveItem, found = user.Character.FindInBackpack(giveWhat)

		if !found {
			user.SendText(fmt.Sprintf(`You don't have a %s to give. (<ansi fg="command">give {object-name} {receiver-name}</ansi>)`, giveWhat))
			return true, nil
		}

	}

	playerId, mobId := room.FindByName(giveWho)

	if playerId > 0 {

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		targetUser := users.GetByUserId(playerId)

		// Swap the item location
		if giveItem.ItemId > 0 {
			userActor := &actions.UserActor{User: user, Room: room}
			result := actions.GiveItemToChar(userActor, giveWhat, targetUser.Character, targetUser.UserId, 0)
			if result.Err != nil {
				user.SendText("Something went wrong.")
				return true, nil
			}

			user.SendText(
				fmt.Sprintf(`You give the <ansi fg="item">%s</ansi> to <ansi fg="username">%s</ansi>.`, result.Item.DisplayName(), targetUser.Character.Name),
			)
			targetUser.SendText(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> gives you their <ansi fg="item">%s</ansi>.`, user.Character.Name, result.Item.DisplayName()),
			)
			room.SendTextVisual(
				fmt.Sprintf(`<ansi fg="username">%s</ansi> gives <ansi fg="username">%s</ansi> a <ansi fg="itemname">%s</ansi>.`, user.Character.Name, targetUser.Character.Name, result.Item.NameSimple()),
				user.UserId,
				targetUser.UserId)

		} else if giveGoldAmount > 0 {

			if targetUser.UserId == user.UserId {

				user.SendText(
					fmt.Sprintf(`You count out <ansi fg="gold">%d gold</ansi> and put it back in your pocket.`, giveGoldAmount),
				)
				room.SendTextVisual(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> counts out some <ansi fg="gold">gold</ansi> and put it back in their pocket.`, user.Character.Name),
					user.UserId)

			} else {
				userActor := &actions.UserActor{User: user, Room: room}
				if err := actions.GiveGoldToChar(userActor, giveGoldAmount, targetUser.Character); err != nil {
					user.SendText("Something went wrong.")
					return true, nil
				}

				events.AddToQueue(events.EquipmentChange{
					UserId:     targetUser.UserId,
					GoldChange: giveGoldAmount,
				})

				events.AddToQueue(events.EquipmentChange{
					UserId:     user.UserId,
					GoldChange: -giveGoldAmount,
				})

				user.SendText(
					fmt.Sprintf(`You give <ansi fg="gold">%d gold</ansi> to <ansi fg="username">%s</ansi>.`, giveGoldAmount, targetUser.Character.Name),
				)
				targetUser.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> gives you <ansi fg="gold">%d gold</ansi>.`, user.Character.Name, giveGoldAmount),
				)
				room.SendTextVisual(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> gives <ansi fg="username">%s</ansi> some <ansi fg="gold">gold</ansi>.`, user.Character.Name, targetUser.Character.Name),
					user.UserId,
					targetUser.UserId)
			}
		} else {
			user.SendText("Something went wrong.")
		}

		return true, nil

	}

	//
	// Look for an NPC
	//
	if mobId > 0 {

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		m := mobs.GetInstance(mobId)

		if m != nil {

			// Swap the item location
			if giveItem.ItemId > 0 || giveGoldAmount > 0 {

				if giveGoldAmount > 0 {
					userActor := &actions.UserActor{User: user, Room: room}
					if err := actions.GiveGoldToChar(userActor, giveGoldAmount, &m.Character); err != nil {
						user.SendText("Something went wrong.")
						return true, nil
					}

					events.AddToQueue(events.EquipmentChange{
						UserId:     user.UserId,
						GoldChange: -giveGoldAmount,
					})

					user.SendText(
						fmt.Sprintf(`You give <ansi fg="gold">%d gold</ansi> to <ansi fg="username">%s</ansi>.`, giveGoldAmount, m.Character.Name),
					)
					room.SendTextVisual(
						fmt.Sprintf(`<ansi fg="username">%s</ansi> gave some gold to <ansi fg="mobname">%s</ansi>.`, user.Character.Name, m.Character.Name),
						user.UserId,
					)
				} else {

					// Check quest engine first — it may intercept the give before
					// the item is transferred to the mob.
					bridge := questengine.NewGameBridge(user, room.RoomId)
					qResult := questengine.GetEngine().Notify("item_give", questengine.EventDetails{
						UserId: user.UserId,
						RoomId: room.RoomId,
						MobId:  int(m.MobId),
						ItemId: giveItem.ItemId,
					}, bridge, bridge)

					if qResult.Handled && qResult.ConsumeItem {
						// Quest engine consumed the item — remove from player only,
						// do NOT transfer to mob and do NOT fire onGive script.
						user.Character.RemoveItem(giveItem)

						user.SendText(
							fmt.Sprintf(`You give the <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, giveItem.DisplayName(), m.Character.Name),
						)
						room.SendTextVisual(
							fmt.Sprintf(`<ansi fg="username">%s</ansi> gave their <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, user.Character.Name, giveItem.DisplayName(), m.Character.Name),
							user.UserId,
						)

						events.AddToQueue(events.ItemOwnership{
							UserId: user.UserId,
							Item:   giveItem,
							Gained: false,
						})

						return true, nil
					}

					// Normal flow — transfer item to mob via atomic transfer.
					userActor := &actions.UserActor{User: user, Room: room}
					result := actions.GiveItemToChar(userActor, giveWhat, &m.Character, 0, m.InstanceId)
					if result.Err != nil {
						user.SendText("Something went wrong.")
						return true, nil
					}
					// Update giveItem so onGive scripting below has the live value.
					giveItem = result.Item

					user.SendText(
						fmt.Sprintf(`You give the <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, giveItem.DisplayName(), m.Character.Name),
					)
					room.SendTextVisual(
						fmt.Sprintf(`<ansi fg="username">%s</ansi> gave their <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, user.Character.Name, giveItem.DisplayName(), m.Character.Name),
						user.UserId,
					)

				}

				// Behavior tree: try before JS
				if behaviortree.TryMobBehavior(m.InstanceId, behaviortree.EventContext{
					EventType: "player_give",
					UserId:    user.UserId,
					ItemId:    giveItem.ItemId,
					RoomId:    room.RoomId,
				}) {
					return true, nil
				}

				if giveGoldAmount > 0 {
					m.Command(`emote counts his gold coins and chuckles a bit.`)
				} else {
					m.Command(fmt.Sprintf(`emote considers the <ansi fg="itemname">%s</ansi> for a moment.`, giveItem.DisplayName()))
					m.Command(fmt.Sprintf(`gearup !%d`, giveItem.ItemId))
				}
			} else {
				user.SendText("Something went wrong.")
			}

		}

		return true, nil
	}

	//
	// Look for any pets in the room
	//
	petUserId := room.FindByPetName(giveWho)
	if petUserId == 0 && giveWho == `pet` && user.Character.Pet.Exists() {
		petUserId = user.UserId
	}
	if petUserId > 0 {

		petUser := users.GetByUserId(petUserId)
		if petUser == nil {
			user.SendText("Who???")
			return true, nil
		}

		if giveGoldAmount > 0 {
			room.SendTextVisual(fmt.Sprintf(`What would %s do with <ansi fg="gold">%d gold</ansi>?`, petUser.Character.Pet.DisplayName(), giveGoldAmount))
			return true, nil
		}

		user.SendText(fmt.Sprintf(`You give the <ansi fg="itemname">%s</ansi> to %s.`, giveItem.DisplayName(), petUser.Character.Pet.DisplayName()))
		room.SendTextVisual(fmt.Sprintf(`<ansi fg="username">%s</ansi> gives their <ansi fg="itemname">%s</ansi> to %s...`, user.Character.Name, giveItem.DisplayName(), petUser.Character.Pet.DisplayName()), user.UserId)

		user.Character.RemoveItem(giveItem)

		events.AddToQueue(events.ItemOwnership{
			UserId: user.UserId,
			Item:   giveItem,
			Gained: false,
		})

		if len(petUser.Character.Pet.Items) >= petUser.Character.Pet.Capacity || !petUser.Character.Pet.StoreItem(giveItem) {
			room.SendTextVisual(fmt.Sprintf(`%s throws the <ansi fg="itemname">%s</ansi> onto the ground.`, petUser.Character.Pet.DisplayName(), giveItem.DisplayName()))
			room.AddItem(giveItem, false)
		}

		return true, nil
	}

	user.SendText(`Who??? (<ansi fg="command">give {object-name} {receiver-name}</ansi>)`)

	return true, nil
}
