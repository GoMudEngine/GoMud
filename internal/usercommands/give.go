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
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Give(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	rest = util.StripPrepositions(rest)

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 2 {
		user.SendText(messaging.CategorySystem, `Give what? To whom? (<ansi fg="command">give {object-name} {receiver-name}</ansi>)`)
		return true, nil
	}

	// Split args into "<object>" and "<recipient>". The recipient may be a
	// multi-word name (e.g. "smith rusk"), so we try progressively longer
	// recipient phrases from the right and pick the first split where BOTH
	// the object (an item in the backpack, or a gold amount) AND the
	// recipient resolve. Falls back to the last single token as recipient
	// (legacy behavior) so existing error messages still fire for typos.
	giveWhat, giveWho := splitGiveArgs(args, user, room)

	var giveItem items.Item = items.Item{}
	var giveGoldAmount int = 0

	if amount, isGold, ok := parseGoldPhrase(giveWhat); isGold {

		if !ok {
			// Either unparseable or negative. The old code accepted a
			// negative here and rejected it with its own message; keep a
			// single message now that one parser owns both cases.
			user.SendText(messaging.CategorySystem, "That isn't an amount of gold you can give.")
			return true, nil
		}

		giveGoldAmount = amount

		if giveGoldAmount > user.Character.Gold {
			user.SendText(messaging.CategorySystem, "You don't have that much gold to give.")
			return true, nil
		}

	} else {

		var found bool = false

		// Check whether the user has an item in their inventory that matches
		giveItem, found = user.Character.FindInBackpack(giveWhat)

		if !found {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`You don't have a %s to give. (<ansi fg="command">give {object-name} {receiver-name}</ansi>)`, giveWhat))
			return true, nil
		}

	}

	target, err := actions.ResolveTargetActor(room, giveWho)
	if err == nil {
		if target.IsPlayer() {

			targetUser := target.(*actions.UserActor).User

			user.Character.CancelBuffsWithFlag(buffs.Hidden)

			// Swap the item location
			if giveItem.ItemId > 0 {
				userActor := &actions.UserActor{User: user, Room: room}
				result := actions.GiveItemToChar(userActor, giveWhat, targetUser.Character, targetUser.UserId, 0)
				if result.Err != nil {
					user.SendText(messaging.CategorySystem, "Something went wrong.")
					return true, nil
				}

				user.SendText(messaging.CategorySystem,
					fmt.Sprintf(`You give the <ansi fg="item">%s</ansi> to <ansi fg="username">%s</ansi>.`, result.Item.DisplayName(), targetUser.Character.Name),
				)
				targetUser.SendText(messaging.CategorySystem,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> gives you their <ansi fg="item">%s</ansi>.`, user.Character.Name, result.Item.DisplayName()),
				)
				room.SendTextVisual(messaging.CategoryLoot,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> gives <ansi fg="username">%s</ansi> a <ansi fg="itemname">%s</ansi>.`, user.Character.Name, targetUser.Character.Name, result.Item.NameSimple()),
					user.UserId,
					targetUser.UserId)

			} else if giveGoldAmount > 0 {

				if targetUser.UserId == user.UserId {

					user.SendText(messaging.CategorySystem,
						fmt.Sprintf(`You count out <ansi fg="gold">%d gold</ansi> and put it back in your pocket.`, giveGoldAmount),
					)
					room.SendTextVisual(messaging.CategoryLoot,
						fmt.Sprintf(`<ansi fg="username">%s</ansi> counts out some <ansi fg="gold">gold</ansi> and put it back in their pocket.`, user.Character.Name),
						user.UserId)

				} else {
					userActor := &actions.UserActor{User: user, Room: room}
					if err := actions.GiveGoldToChar(userActor, giveGoldAmount, targetUser.Character); err != nil {
						user.SendText(messaging.CategorySystem, "Something went wrong.")
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

					user.SendText(messaging.CategorySystem,
						fmt.Sprintf(`You give <ansi fg="gold">%d gold</ansi> to <ansi fg="username">%s</ansi>.`, giveGoldAmount, targetUser.Character.Name),
					)
					targetUser.SendText(messaging.CategorySystem,
						fmt.Sprintf(`<ansi fg="username">%s</ansi> gives you <ansi fg="gold">%d gold</ansi>.`, user.Character.Name, giveGoldAmount),
					)
					room.SendTextVisual(messaging.CategoryLoot,
						fmt.Sprintf(`<ansi fg="username">%s</ansi> gives <ansi fg="username">%s</ansi> some <ansi fg="gold">gold</ansi>.`, user.Character.Name, targetUser.Character.Name),
						user.UserId,
						targetUser.UserId)
				}
			} else {
				user.SendText(messaging.CategorySystem, "Something went wrong.")
			}

			return true, nil
		}

		//
		// Mob target
		//
		m := target.(*actions.MobActor).Mob

		// MUST be before the transfer below: give.go hands the item over before
		// any handler fires and cannot take it back (see CLAUDE.md). A sleeping
		// Guard Captain Velk was accepting quest items and advancing quests.
		if actions.RefuseMobIfAsleep(m, user) {
			return true, nil
		}

		user.Character.CancelBuffsWithFlag(buffs.Hidden)

		// Swap the item location
		if giveItem.ItemId > 0 || giveGoldAmount > 0 {

			if giveGoldAmount > 0 {
				userActor := &actions.UserActor{User: user, Room: room}
				if err := actions.GiveGoldToChar(userActor, giveGoldAmount, &m.Character); err != nil {
					user.SendText(messaging.CategorySystem, "Something went wrong.")
					return true, nil
				}

				events.AddToQueue(events.EquipmentChange{
					UserId:     user.UserId,
					GoldChange: -giveGoldAmount,
				})

				user.SendText(messaging.CategorySystem,
					fmt.Sprintf(`You give <ansi fg="gold">%d gold</ansi> to <ansi fg="username">%s</ansi>.`, giveGoldAmount, m.Character.Name),
				)
				room.SendTextVisual(messaging.CategoryLoot,
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

					user.SendText(messaging.CategorySystem,
						fmt.Sprintf(`You give the <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, giveItem.DisplayName(), m.Character.Name),
					)
					room.SendTextVisual(messaging.CategoryLoot,
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
					user.SendText(messaging.CategorySystem, "Something went wrong.")
					return true, nil
				}
				// Update giveItem so onGive scripting below has the live value.
				giveItem = result.Item

				user.SendText(messaging.CategorySystem,
					fmt.Sprintf(`You give the <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, giveItem.DisplayName(), m.Character.Name),
				)
				room.SendTextVisual(messaging.CategoryLoot,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> gave their <ansi fg="item">%s</ansi> to <ansi fg="mobname">%s</ansi>.`, user.Character.Name, giveItem.DisplayName(), m.Character.Name),
					user.UserId,
				)

				// Chunk 4.5: notify seeders of every item give to a mob.
				// GiftOffered: fires unconditionally (no consumers in 4.5;
				//   reserved for future analytics / tutorial rules).
				// GiftAccepted: fires immediately because there is no clean
				//   "mob decided to keep" btree hook. Mobs that run the
				//   equip-if-better path may subsequently return the item;
				//   rule 7's per-pair cooldown (100 rounds) absorbs the noise.
				events.AddToQueue(events.GiftOffered{
					UserId:        user.UserId,
					MobInstanceId: m.InstanceId,
					ItemId:        giveItem.ItemId,
				})
				events.AddToQueue(events.GiftAccepted{
					UserId:        user.UserId,
					MobInstanceId: m.InstanceId,
					ItemId:        giveItem.ItemId,
				})

			}

			// Behavior tree: try before JS
			if behaviortree.TryMobBehavior(m.InstanceId, behaviortree.EventContext{
				EventType: "player_give",
				UserId:    user.UserId,
				ItemId:    giveItem.ItemId,
				ItemUUID:  giveItem.UUID,
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
			user.SendText(messaging.CategorySystem, "Something went wrong.")
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
			user.SendText(messaging.CategorySystem, "Who???")
			return true, nil
		}

		if giveGoldAmount > 0 {
			room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`What would %s do with <ansi fg="gold">%d gold</ansi>?`, petUser.Character.Pet.DisplayName(), giveGoldAmount))
			return true, nil
		}

		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You give the <ansi fg="itemname">%s</ansi> to %s.`, giveItem.DisplayName(), petUser.Character.Pet.DisplayName()))
		room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`<ansi fg="username">%s</ansi> gives their <ansi fg="itemname">%s</ansi> to %s...`, user.Character.Name, giveItem.DisplayName(), petUser.Character.Pet.DisplayName()), user.UserId)

		user.Character.RemoveItem(giveItem)

		events.AddToQueue(events.ItemOwnership{
			UserId: user.UserId,
			Item:   giveItem,
			Gained: false,
		})

		if len(petUser.Character.Pet.Items) >= petUser.Character.Pet.Capacity || !petUser.Character.Pet.StoreItem(giveItem) {
			room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`%s throws the <ansi fg="itemname">%s</ansi> onto the ground.`, petUser.Character.Pet.DisplayName(), giveItem.DisplayName()))
			room.AddItem(giveItem, false)
		}

		return true, nil
	}

	user.SendText(messaging.CategorySystem, `Who??? (<ansi fg="command">give {object-name} {receiver-name}</ansi>)`)

	return true, nil
}

// splitGiveArgs separates the give command's arguments into an object phrase
// ("iron sword", "50 gold") and a recipient phrase ("rusk", "smith rusk").
// It tries recipient phrases of increasing length from the right and returns
// the first split where the object resolves (item in backpack or a valid gold
// amount) AND the recipient resolves (player, mob, or pet in the room). If no
// split fully resolves, it falls back to treating the last single token as the
// recipient — preserving legacy behavior and the downstream not-found errors.
func splitGiveArgs(args []string, user *users.UserRecord, room *rooms.Room) (giveWhat, giveWho string) {
	for k := 1; k < len(args); k++ {
		who := strings.Join(args[len(args)-k:], " ")
		what := strings.Join(args[:len(args)-k], " ")
		if giveObjectResolves(what, user) && giveTargetResolves(who, user, room) {
			return what, who
		}
	}
	return strings.Join(args[:len(args)-1], " "), args[len(args)-1]
}

// parseGoldPhrase is the single gold parser for both resolution and execution.
// It reports whether `what` is a gold phrase ("50 gold", "50gold") and, if so,
// the amount.
//
// Resolution and execution used to slice the amount differently (len-4 here
// versus len-5 in Give), so the compact form resolved as 50 and transferred 5.
// Both callers now go through this function; do not reintroduce a second
// parse. `isGold` is true for any phrase ending in "gold" even when the amount
// is unparseable or negative, so callers can reject it with a gold-specific
// message instead of falling through to item lookup.
func parseGoldPhrase(what string) (amount int, isGold bool, ok bool) {
	if len(what) <= 4 || what[len(what)-4:] != "gold" {
		return 0, false, false
	}
	amt, err := strconv.ParseInt(strings.TrimSpace(what[:len(what)-4]), 10, 32)
	if err != nil || amt < 0 {
		return 0, true, false
	}
	return int(amt), true, true
}

// giveObjectResolves reports whether the object phrase names something the
// player can give: a valid (non-negative) gold amount, or an item in their
// backpack.
func giveObjectResolves(what string, user *users.UserRecord) bool {
	if _, isGold, ok := parseGoldPhrase(what); isGold {
		return ok
	}
	_, found := user.Character.FindInBackpack(what)
	return found
}

// giveTargetResolves reports whether the recipient phrase names a player, mob,
// or pet present in the room (or the player's own pet via the literal "pet").
func giveTargetResolves(who string, user *users.UserRecord, room *rooms.Room) bool {
	if playerId, mobInstanceId := room.FindByName(who); playerId > 0 || mobInstanceId > 0 {
		return true
	}
	if room.FindByPetName(who) > 0 {
		return true
	}
	if who == "pet" && user.Character.Pet.Exists() {
		return true
	}
	return false
}
