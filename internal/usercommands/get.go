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

func Get(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Can't pick things up if you can't see
	if room.GetVisibility() < 1 && !user.Character.HasFlagFromAnySource(buffs.NightVision) {
		user.SendText(messaging.CategorySystem, "You can't see anything to pick up!")
		return true, nil
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {
		user.SendText(messaging.CategorySystem, "Get what?")
		return true, nil
	}

	// Sealed crate short-circuit — players can't `get` from it.
	if len(args) > 0 && room.MatchesSealedCrate(strings.ToLower(args[len(args)-1])) {
		user.SendText(messaging.CategorySystem, `The shipping crate is sealed and bound for the caravan; you can't get into it.`)
		return true, nil
	}

	if args[0] == "all" {

		// get all bag/case/pouch — move everything from component bag to backpack
		if len(args) >= 2 {
			allLastArg := args[len(args)-1]
			if allLastArg == "bag" || allLastArg == "case" || allLastArg == "pouch" || allLastArg == "components" {
				if len(user.Character.ComponentItems) == 0 {
					user.SendText(messaging.CategorySystem, `Your component bag is empty.`)
				} else {
					ct := len(user.Character.ComponentItems)
					user.Character.Items = append(user.Character.Items, user.Character.ComponentItems...)
					user.Character.ComponentItems = nil
					user.SendText(messaging.CategorySystem, fmt.Sprintf(`You move %d item(s) from your component bag to your backpack.`, ct))
				}
				return true, nil
			}
			if allLastArg == "bandolier" || allLastArg == "potions" {
				if len(user.Character.PotionItems) == 0 {
					user.SendText(messaging.CategorySystem, `Your bandolier is empty.`)
				} else {
					ct := len(user.Character.PotionItems)
					user.Character.Items = append(user.Character.Items, user.Character.PotionItems...)
					user.Character.PotionItems = nil
					user.SendText(messaging.CategorySystem, fmt.Sprintf(`You move %d potion(s) from your bandolier to your backpack.`, ct))
				}
				return true, nil
			}
		}

		// get all <container> — grab everything from a specific container
		if len(args) >= 2 {
			cName := room.FindContainerByName(args[len(args)-1])
			if cName != `` {
				if c, exists := room.Containers[cName]; exists && c.Hidden {
					if user == nil || !user.Character.HasDiscovery(room.RoomId, cName) {
						cName = ``
					}
				}
			}
			if cName != `` {
				container := room.Containers[cName]
				if container.Gold > 0 {
					Get(fmt.Sprintf("gold %s", cName), user, room, flags)
				}
				if len(container.Items) > 0 {
					iCopies := append([]items.Item{}, container.Items...)
					for _, item := range iCopies {
						Get(fmt.Sprintf("%s %s", item.Name(), cName), user, room, flags)
					}
				}
				if container.Gold < 1 && len(container.Items) < 1 {
					user.SendText(messaging.CategorySystem, fmt.Sprintf(`There's nothing in the <ansi fg="container">%s</ansi>.`, cName))
				}
				return true, nil
			}
		}

		// get all — grab everything from the floor
		if room.Gold > 0 {
			Get(`gold`, user, room, flags)
		}

		if len(room.Items) > 0 {
			iCopies := append([]items.Item{}, room.Items...)

			for _, item := range iCopies {
				Get(item.Name(), user, room, flags)
			}
		}

		return true, nil
	}

	// Handle "get all.item" — pick up all matching items from the floor
	{
		itemName, matchNum := util.GetMatchNumber(args[0])
		if matchNum == -1 {
			picked := 0
			for {
				matchItem, found := room.FindOnFloor(itemName, false)
				if !found {
					break
				}

				if matchItem.HasAdjective(`exploding`) {
					break
				}

				user.Character.CancelBuffsWithFlag(buffs.Hidden)

				if user.Character.StoreItem(matchItem) {
					room.RemoveItem(matchItem, false)

					events.AddToQueue(events.ItemOwnership{
						UserId: user.UserId,
						Item:   matchItem,
						Gained: true,
					})

					picked++
				} else {
					user.SendText(messaging.CategorySystem, 
						fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi> - you're already overloaded!`, matchItem.DisplayName()),
					)
					break
				}
			}
			if picked == 0 {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`You don't see any "%s" to pick up.`, itemName))
			} else {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`You pick up %d item(s).`, picked))
				room.SendTextVisual(messaging.CategoryLoot, 
					fmt.Sprintf(`<ansi fg="username">%s</ansi> picks up some items.`, user.Character.Name),
					user.UserId,
				)
				sendEncumbranceWarning(user)
			}
			return true, nil
		}
	}

	getFromStash := false
	containerName := ``
	petUserId := 0
	corpseIdx := -1

	if len(args) >= 2 {
		// Detect "stash" or "from stash" at end and remove it
		if args[len(args)-1] == "stash" {
			getFromStash = true
			if args[len(args)-2] == "from" {
				rest = strings.Join(args[0:len(args)-2], " ")
			} else {
				rest = strings.Join(args[0:len(args)-1], " ")
			}
		}

		if args[len(args)-1] == "ground" {
			getFromStash = false
			if args[len(args)-2] == "from" {
				rest = strings.Join(args[0:len(args)-2], " ")
			} else {
				rest = strings.Join(args[0:len(args)-1], " ")
			}
		}

		// Check for "get X from bag/case/pouch/bandolier" — pull from component bag or bandolier
		lastArg := args[len(args)-1]
		isBagGet := lastArg == "bag" || lastArg == "case" || lastArg == "pouch" || lastArg == "components"
		isBandolierGet := lastArg == "bandolier" || lastArg == "potions"

		if isBagGet || isBandolierGet {
			if len(args) >= 3 && args[len(args)-2] == "from" {
				rest = strings.Join(args[0:len(args)-2], " ")
			} else {
				rest = strings.Join(args[0:len(args)-1], " ")
			}

			var sourceItems *[]items.Item
			var label string
			if isBagGet {
				sourceItems = &user.Character.ComponentItems
				label = "component bag"
			} else {
				sourceItems = &user.Character.PotionItems
				label = "bandolier"
			}

			closeMatch, exactMatch := items.FindMatchIn(rest, *sourceItems...)
			foundItem := exactMatch
			if foundItem.ItemId == 0 {
				foundItem = closeMatch
			}

			if foundItem.ItemId == 0 {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`You don't see a "%s" in your %s.`, rest, label))
				return true, nil
			}

			// Remove from source and add to backpack
			for j := len(*sourceItems) - 1; j >= 0; j-- {
				if (*sourceItems)[j].Equals(foundItem) {
					*sourceItems = append((*sourceItems)[:j], (*sourceItems)[j+1:]...)
					break
				}
			}
			user.Character.Items = append(user.Character.Items, foundItem)
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`You move the <ansi fg="itemname">%s</ansi> from your %s to your backpack.`, foundItem.DisplayName(), label))
			return true, nil
		}

		containerName = room.FindContainerByName(args[len(args)-1])
		if containerName != `` {
			if c, exists := room.Containers[containerName]; exists && c.Hidden {
				if user == nil || !user.Character.HasDiscovery(room.RoomId, containerName) {
					containerName = ``
				}
			}
		}
		if containerName != `` {
			getFromStash = false
			if args[len(args)-2] == "from" {
				rest = strings.Join(args[0:len(args)-2], " ")
			} else {
				rest = strings.Join(args[0:len(args)-1], " ")
			}
		}

		//
		// Look for any pets in the room
		//
		petUserId = room.FindByPetName(args[len(args)-1])
		if petUserId == 0 && args[len(args)-1] == `pet` && user.Character.Pet.Exists() {
			petUserId = user.UserId
		}
		if petUserId > 0 {

			if petUserId != user.UserId {
				user.SendText(messaging.CategorySystem, `You can't do that!`)
				return true, nil
			}

			getFromStash = false
			if petUser := users.GetByUserId(petUserId); petUser != nil {

				if args[len(args)-2] == "from" {
					rest = strings.Join(args[0:len(args)-2], " ")
				} else {
					rest = strings.Join(args[0:len(args)-1], " ")
				}
			}
		}

		//
		// "get <item|gold> from <corpse name>" — corpse loot container.
		// A corpse name is multi-word ("grey wolf corpse"), so unlike room
		// Containers we split on the "from" keyword and treat everything
		// after it as the corpse search name. Only attempt this if nothing
		// else (room container, pet) already claimed the trailing target.
		//
		if containerName == `` && petUserId == 0 {
			fromIdx := -1
			for i, a := range args {
				if a == "from" {
					fromIdx = i
					break
				}
			}
			if fromIdx > 0 && fromIdx < len(args)-1 {
				corpseSearch := strings.Join(args[fromIdx+1:], " ")
				if idx := room.FindCorpseIndex(corpseSearch); idx >= 0 {
					corpseIdx = idx
					getFromStash = false
					rest = strings.Join(args[0:fromIdx], " ")
				}
			}
		}
	}

	//
	// Corpse loot: operate on a POINTER into room.Corpses so mutations to
	// the loot container (item removal, gold drawdown) persist in place.
	//
	if corpseIdx >= 0 {
		corpse := &room.Corpses[corpseIdx]

		// Ownership/mode gate — Task 3 stub always allows; Tasks 8/10 gate.
		if !canLootCorpse(user, corpse) {
			user.SendText(messaging.CategorySystem, `This isn't your kill.`)
			return true, nil
		}

		goldName := `gold`
		if args[0] == goldName || (len(args[0]) < 5 && goldName[0:len(args[0])-1] == args[0]) {

			if corpse.Loot.Gold < 1 {
				user.SendText(messaging.CategorySystem, "There's no gold to grab.")
			} else {
				user.Character.CancelBuffsWithFlag(buffs.Hidden) // No longer sneaking

				amt := corpse.Loot.Gold
				corpse.Loot.Gold -= amt
				grantCorpseGold(user, amt)

				user.SendText(messaging.CategorySystem,
					fmt.Sprintf(`You take <ansi fg="gold">%d gold</ansi> from the <ansi fg="mob-corpse">%s</ansi>.`, amt, corpse.DisplayName()),
				)
				room.SendTextVisual(messaging.CategoryLoot,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> loots some <ansi fg="gold">gold</ansi> from the <ansi fg="mob-corpse">%s</ansi>.`, user.Character.Name, corpse.DisplayName()),
					user.UserId,
				)
			}

			return true, nil
		}

		matchItem, found := corpse.Loot.FindItem(rest)
		if !found {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`You don't see a %s in the <ansi fg="mob-corpse">%s</ansi>.`, rest, corpse.DisplayName()))
			return true, nil
		}

		user.Character.CancelBuffsWithFlag(buffs.Hidden) // No longer sneaking

		if user.Character.StoreItem(matchItem) {
			events.AddToQueue(events.ItemOwnership{
				UserId: user.UserId,
				Item:   matchItem,
				Gained: true,
			})
			corpse.Loot.RemoveItem(matchItem)

			user.SendText(messaging.CategorySystem,
				fmt.Sprintf(`You take the <ansi fg="itemname">%s</ansi> from the <ansi fg="mob-corpse">%s</ansi>.`, matchItem.DisplayName(), corpse.DisplayName()),
			)
			room.SendTextVisual(messaging.CategoryLoot,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> loots the <ansi fg="itemname">%s</ansi> from the <ansi fg="mob-corpse">%s</ansi>...`, user.Character.Name, matchItem.DisplayName(), corpse.DisplayName()),
				user.UserId,
			)

			sendEncumbranceWarning(user)
		} else {
			user.SendText(messaging.CategorySystem,
				fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi> - you're already overloaded!`, matchItem.DisplayName()),
			)
		}

		return true, nil
	}

	if petUserId == user.UserId {

		matchItem, found := user.Character.Pet.FindItem(rest)
		if !found {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`You don't see a %s carried by %s.`, rest, user.Character.Pet.DisplayName()))
		} else {

			if user.Character.Pet.RemoveItem(matchItem) {
				if !user.Character.StoreItem(matchItem) {
					user.Character.Pet.StoreItem(matchItem)
					user.SendText(messaging.CategorySystem, 
						fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi> - you're already overloaded!`, matchItem.DisplayName()),
					)
				} else {

					events.AddToQueue(events.ItemOwnership{
						UserId: user.UserId,
						Item:   matchItem,
						Gained: true,
					})

					user.SendText(messaging.CategorySystem, 
						fmt.Sprintf(`You remove a <ansi fg="itemname">%s</ansi> from %s.`, matchItem.DisplayName(), user.Character.Pet.DisplayName()),
					)
					room.SendTextVisual(messaging.CategoryLoot, 
						fmt.Sprintf(`<ansi fg="username">%s</ansi> removes a <ansi fg="itemname">%s</ansi> from %s...`, user.Character.Name, matchItem.DisplayName(), user.Character.Pet.DisplayName()),
						user.UserId,
					)

					// Check encumbrance and warn player
					sendEncumbranceWarning(user)
				}

			}
		}

		return true, nil

	}

	if containerName != `` {
		container := room.Containers[containerName]

		goldName := `gold`
		if args[0] == goldName || (len(args[0]) < 5 && goldName[0:len(args[0])-1] == args[0]) {

			if container.Gold < 1 {
				user.SendText(messaging.CategorySystem, "There's no gold to grab.")
			} else {

				user.Character.CancelBuffsWithFlag(buffs.Hidden) // No longer sneaking

				goldAmt := container.Gold
				user.Character.Gold += goldAmt
				container.Gold -= goldAmt
				room.Containers[containerName] = container

				events.AddToQueue(events.EquipmentChange{
					UserId:     user.UserId,
					GoldChange: -goldAmt,
				})

				user.SendText(messaging.CategorySystem, 
					fmt.Sprintf(`You pick up <ansi fg="gold">%d gold</ansi> from the <ansi fg="container">%s</ansi>.`, goldAmt, containerName),
				)
				room.SendTextVisual(messaging.CategoryLoot, 
					fmt.Sprintf(`<ansi fg="username">%s</ansi> picks up some <ansi fg="gold">gold</ansi> from the <ansi fg="container">%s</ansi>.`, user.Character.Name, containerName),
					user.UserId,
				)
			}

			return true, nil
		}

		matchItem, found := container.FindItem(rest)

		if !found {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`You don't see a %s in the <ansi fg="container">%s</ansi>.`, rest, containerName))
		} else {

			user.Character.CancelBuffsWithFlag(buffs.Hidden) // No longer sneaking

			// Trigger onFound event
			if user.Character.StoreItem(matchItem) {

				events.AddToQueue(events.ItemOwnership{
					UserId: user.UserId,
					Item:   matchItem,
					Gained: true,
				})

				// Swap the item location
				container.RemoveItem(matchItem)
				room.Containers[containerName] = container

				user.SendText(messaging.CategorySystem, 
					fmt.Sprintf(`You take the <ansi fg="itemname">%s</ansi> from the <ansi fg="container">%s</ansi>.`, matchItem.DisplayName(), containerName),
				)
				room.SendTextVisual(messaging.CategoryLoot, 
					fmt.Sprintf(`<ansi fg="username">%s</ansi> picks up the <ansi fg="itemname">%s</ansi> from the <ansi fg="container">%s</ansi>...`, user.Character.Name, matchItem.DisplayName(), containerName),
					user.UserId,
				)

				// Check encumbrance and warn player
				sendEncumbranceWarning(user)

				return true, nil

			} else {
				user.SendText(messaging.CategorySystem, 
					fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi> - you're already overloaded!`, matchItem.DisplayName()),
				)
			}

		}

	} else {

		goldName := `gold`
		if args[0] == goldName || (len(args[0]) < 5 && goldName[0:len(args[0])-1] == args[0]) {

			if room.Gold < 1 {
				user.SendText(messaging.CategorySystem, "There's no gold to grab.")
			} else {

				user.Character.CancelBuffsWithFlag(buffs.Hidden) // No longer sneaking

				goldAmt := room.Gold
				if err := actions.GetGoldFromFloor(&actions.UserActor{User: user, Room: room}, goldAmt); err == nil {
					events.AddToQueue(events.EquipmentChange{
						UserId:     user.UserId,
						GoldChange: -goldAmt,
					})

					user.SendText(messaging.CategorySystem, 
						fmt.Sprintf(`You pick up <ansi fg="gold">%d gold</ansi>.`, goldAmt),
					)
					room.SendTextVisual(messaging.CategoryLoot, 
						fmt.Sprintf(`<ansi fg="username">%s</ansi> picks up some <ansi fg="gold">gold</ansi>.`, user.Character.Name),
						user.UserId,
					)
				}
			}

			return true, nil
		}

		// Check whether the user has an item in their inventory that matches
		var matchItem items.Item
		var found bool

		// First try the requested source (floor or stash)
		{
			// Peek at the item before transferring so we can guard against exploding items
			peekItem, peekFound := room.FindOnFloor(rest, getFromStash)
			if peekFound && peekItem.HasAdjective(`exploding`) {
				user.SendText(messaging.CategorySystem, `You can't pick that up, it's about to explode!`)
				return true, nil
			}
			if peekFound {
				result := actions.GetItemFromFloor(&actions.UserActor{User: user, Room: room}, rest, getFromStash)
				if result.Found {
					matchItem = result.Item
					found = true
					if result.Err != nil {
						// Capacity exceeded — item was rolled back to floor
						user.SendText(messaging.CategorySystem, 
							fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi> - you're already overloaded!`, matchItem.DisplayName()),
						)
						return true, nil
					}
				}
			}
		}

		// Check if user is specifying an item they stashed (auto-detect)
		if !found && !getFromStash {
			stashItemMatch, stashFound := room.FindOnFloor(rest, true)
			if stashFound && stashItemMatch.StashedBy == user.UserId {
				getFromStash = true
				result := actions.GetItemFromFloor(&actions.UserActor{User: user, Room: room}, rest, true)
				if result.Found {
					found = true
					matchItem = result.Item
					if result.Err != nil {
						user.SendText(messaging.CategorySystem, 
							fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi> - you're already overloaded!`, matchItem.DisplayName()),
						)
						return true, nil
					}
					// Clear stash owner tag on the in-inventory copy
					for i, inv := range user.Character.Items {
						if inv.Equals(matchItem) && inv.StashedBy == user.UserId {
							user.Character.Items[i].StashedBy = 0
							matchItem = user.Character.Items[i]
							break
						}
					}
				}
			}
		}

		if found {
			user.Character.CancelBuffsWithFlag(buffs.Hidden) // No longer sneaking

			if getFromStash {
				user.SendText(messaging.CategorySystem, 
					fmt.Sprintf(`You dig out the <ansi fg="itemname">%s</ansi> from where it was stashed.`, matchItem.DisplayName()),
				)
				room.SendTextVisual(messaging.CategoryLoot, 
					fmt.Sprintf(`<ansi fg="username">%s</ansi> digs around in the area and picks something up...`, user.Character.Name),
					user.UserId,
				)
			} else {
				user.SendText(messaging.CategorySystem, 
					fmt.Sprintf(`You pick up the <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()),
				)
				room.SendTextVisual(messaging.CategoryLoot, 
					fmt.Sprintf(`<ansi fg="username">%s</ansi> picks up the <ansi fg="itemname">%s</ansi>...`, user.Character.Name, matchItem.DisplayName()),
					user.UserId,
				)
			}

			// Check encumbrance and warn player
			sendEncumbranceWarning(user)

			return true, nil
		}

		//
		// Look for any nouns in the room info
		//
		foundNoun, _ := room.FindNoun(rest)
		if len(foundNoun) > 0 {

			user.SendText(messaging.CategorySystem, fmt.Sprintf(`You can't get the <ansi fg="noun">%s</ansi>`, foundNoun))
			room.SendTextVisual(messaging.CategoryLoot, fmt.Sprintf(`<ansi fg="username">%s</ansi> is grasping at the air.`, user.Character.Name), user.UserId)

			return true, nil
		}

	}

	if _, corpseFound := room.FindCorpse(rest); corpseFound {
		user.SendText(messaging.CategorySystem, `You can't pick up corpses. What would people think?`)
		return true, nil
	}

	containerName = room.FindContainerByName(rest)
	if containerName != `` {
		if c, exists := room.Containers[containerName]; exists && c.Hidden {
			if user == nil || !user.Character.HasDiscovery(room.RoomId, containerName) {
				containerName = ``
			}
		}
	}
	if containerName != `` {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You can't pick up the <ansi fg="container">%s</ansi>. Try looking at it.`, containerName))
	} else {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("You don't see a %s around.", rest))
	}

	return true, nil
}

// sendEncumbranceWarning checks the player's encumbrance level and sends appropriate warnings
func sendEncumbranceWarning(user *users.UserRecord) {
	weight := user.Character.GetCarriedWeight()
	capacity := user.Character.CarryCapacity()
	encPct := weight / capacity

	if encPct >= 2.0 {
		user.SendText(messaging.CategorySystem, `<ansi fg="red-bold">You are severely overloaded!</ansi> Your combat and movement are heavily penalized.`)
	} else if encPct >= 1.5 {
		user.SendText(messaging.CategorySystem, `<ansi fg="red">You are heavily encumbered.</ansi> Your combat and movement are significantly penalized.`)
	} else if encPct >= 1.0 {
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">You are encumbered.</ansi> Your combat and movement are penalized.`)
	} else if encPct >= 0.75 {
		user.SendText(messaging.CategorySystem, `<ansi fg="yellow">You are carrying a moderate load.</ansi>`)
	}
	// No message for light load or unencumbered
}

// canLootCorpse reports whether user is entitled to take loot from corpse.
//
// Kill-ownership gate (corpse-loot redesign Task 8): delegates to the pure,
// unit-tested Corpse.LootAllowed, feeding it the current round. Loot-mode
// round-robin/leaderhold gating (Task 10) layers on top of this.
func canLootCorpse(user *users.UserRecord, corpse *rooms.Corpse) bool {
	return corpse.LootAllowed(user.UserId, util.GetRoundCount())
}

// grantCorpseGold credits gold looted from a corpse to the looter.
//
// TEMPORARY STUB (corpse-loot redesign Task 3): straight to the character's
// purse. Task 11 routes it to a shared party gold pool when in a party.
func grantCorpseGold(user *users.UserRecord, amt int) {
	if amt <= 0 {
		return
	}
	user.Character.Gold += amt
	events.AddToQueue(events.EquipmentChange{
		UserId:     user.UserId,
		GoldChange: -amt,
	})
}
