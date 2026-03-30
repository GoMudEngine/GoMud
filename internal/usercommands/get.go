package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Get(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Can't pick things up if you can't see
	if room.GetVisibility() < 1 && !user.Character.HasFlagFromAnySource(buffs.NightVision) {
		user.SendText("You can't see anything to pick up!")
		return true, nil
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {
		user.SendText("Get what?")
		return true, nil
	}

	if args[0] == "all" {

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
					user.SendText(fmt.Sprintf(`There's nothing in the <ansi fg="container">%s</ansi>.`, cName))
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
					user.SendText(
						fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi> - you're already overloaded!`, matchItem.DisplayName()),
					)
					break
				}
			}
			if picked == 0 {
				user.SendText(fmt.Sprintf(`You don't see any "%s" to pick up.`, itemName))
			} else {
				user.SendText(fmt.Sprintf(`You pick up %d item(s).`, picked))
				room.SendText(
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
				user.SendText(`You can't do that!`)
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
	}

	if petUserId == user.UserId {

		matchItem, found := user.Character.Pet.FindItem(rest)
		if !found {
			user.SendText(fmt.Sprintf(`You don't see a %s carried by %s.`, rest, user.Character.Pet.DisplayName()))
		} else {

			if user.Character.Pet.RemoveItem(matchItem) {
				if !user.Character.StoreItem(matchItem) {
					user.Character.Pet.StoreItem(matchItem)
					user.SendText(
						fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi> - you're already overloaded!`, matchItem.DisplayName()),
					)
				} else {

					events.AddToQueue(events.ItemOwnership{
						UserId: user.UserId,
						Item:   matchItem,
						Gained: true,
					})

					user.SendText(
						fmt.Sprintf(`You remove a <ansi fg="itemname">%s</ansi> from %s.`, matchItem.DisplayName(), user.Character.Pet.DisplayName()),
					)
					room.SendText(
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
				user.SendText("There's no gold to grab.")
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

				user.SendText(
					fmt.Sprintf(`You pick up <ansi fg="gold">%d gold</ansi> from the <ansi fg="container">%s</ansi>.`, goldAmt, containerName),
				)
				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> picks up some <ansi fg="gold">gold</ansi> from the <ansi fg="container">%s</ansi>.`, user.Character.Name, containerName),
					user.UserId,
				)
			}

			return true, nil
		}

		matchItem, found := container.FindItem(rest)

		if !found {
			user.SendText(fmt.Sprintf(`You don't see a %s in the <ansi fg="container">%s</ansi>.`, rest, containerName))
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

				user.SendText(
					fmt.Sprintf(`You take the <ansi fg="itemname">%s</ansi> from the <ansi fg="container">%s</ansi>.`, matchItem.DisplayName(), containerName),
				)
				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> picks up the <ansi fg="itemname">%s</ansi> from the <ansi fg="container">%s</ansi>...`, user.Character.Name, matchItem.DisplayName(), containerName),
					user.UserId,
				)

				// Check encumbrance and warn player
				sendEncumbranceWarning(user)

				return true, nil

			} else {
				user.SendText(
					fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi> - you're already overloaded!`, matchItem.DisplayName()),
				)
			}

		}

	} else {

		goldName := `gold`
		if args[0] == goldName || (len(args[0]) < 5 && goldName[0:len(args[0])-1] == args[0]) {

			if room.Gold < 1 {
				user.SendText("There's no gold to grab.")
			} else {

				user.Character.CancelBuffsWithFlag(buffs.Hidden) // No longer sneaking

				goldAmt := room.Gold
				user.Character.Gold += goldAmt
				room.Gold -= goldAmt

				events.AddToQueue(events.EquipmentChange{
					UserId:     user.UserId,
					GoldChange: -goldAmt,
				})

				user.SendText(
					fmt.Sprintf(`You pick up <ansi fg="gold">%d gold</ansi>.`, goldAmt),
				)
				room.SendText(
					fmt.Sprintf(`<ansi fg="username">%s</ansi> picks up some <ansi fg="gold">gold</ansi>.`, user.Character.Name),
					user.UserId,
				)
			}

			return true, nil
		}

		// Check whether the user has an item in their inventory that matches
		matchItem, found := room.FindOnFloor(rest, getFromStash)

		// Check if user is specifying an item they stashed
		if !found && !getFromStash {
			stashItemMatch, stashFound := room.FindOnFloor(rest, true)
			if stashFound && stashItemMatch.StashedBy == user.UserId {
				found = true
				getFromStash = true
				matchItem = stashItemMatch
			}
		}

		if found {

			if matchItem.HasAdjective(`exploding`) {
				user.SendText(`You can't pick that up, it's about to explode!`)
				return true, nil
			}

			user.Character.CancelBuffsWithFlag(buffs.Hidden) // No longer sneaking

			// If it was in the stash, remove the stash owner tag
			if getFromStash {
				matchItem.StashedBy = 0
			}

			if user.Character.StoreItem(matchItem) {

				// Swap the item location
				room.RemoveItem(matchItem, getFromStash)

				events.AddToQueue(events.ItemOwnership{
					UserId: user.UserId,
					Item:   matchItem,
					Gained: true,
				})

				if getFromStash {
					user.SendText(
						fmt.Sprintf(`You dig out the <ansi fg="itemname">%s</ansi> from where it was stashed.`, matchItem.DisplayName()),
					)
					room.SendText(
						fmt.Sprintf(`<ansi fg="username">%s</ansi> digs around in the area and picks something up...`, user.Character.Name),
						user.UserId,
					)
				} else {
					user.SendText(
						fmt.Sprintf(`You pick up the <ansi fg="itemname">%s</ansi>.`, matchItem.DisplayName()),
					)
					room.SendText(
						fmt.Sprintf(`<ansi fg="username">%s</ansi> picks up the <ansi fg="itemname">%s</ansi>...`, user.Character.Name, matchItem.DisplayName()),
						user.UserId,
					)
				}

				// Check encumbrance and warn player
				sendEncumbranceWarning(user)

			} else {
				user.SendText(
					fmt.Sprintf(`You can't carry the <ansi fg="itemname">%s</ansi> - you're already overloaded!`, matchItem.DisplayName()),
				)
			}

			return true, nil
		}

		//
		// Look for any nouns in the room info
		//
		foundNoun, _ := room.FindNoun(rest)
		if len(foundNoun) > 0 {

			user.SendText(fmt.Sprintf(`You can't get the <ansi fg="noun">%s</ansi>`, foundNoun))
			room.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is grasping at the air.`, user.Character.Name), user.UserId)

			return true, nil
		}

	}

	if _, corpseFound := room.FindCorpse(rest); corpseFound {
		user.SendText(`You can't pick up corpses. What would people think?`)
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
		user.SendText(fmt.Sprintf(`You can't pick up the <ansi fg="container">%s</ansi>. Try looking at it.`, containerName))
	} else {
		user.SendText(fmt.Sprintf("You don't see a %s around.", rest))
	}

	return true, nil
}

// sendEncumbranceWarning checks the player's encumbrance level and sends appropriate warnings
func sendEncumbranceWarning(user *users.UserRecord) {
	weight := user.Character.GetCarriedWeight()
	capacity := user.Character.CarryCapacity()
	encPct := weight / capacity

	if encPct >= 2.0 {
		user.SendText(`<ansi fg="red-bold">You are severely overloaded!</ansi> Your combat and movement are heavily penalized.`)
	} else if encPct >= 1.5 {
		user.SendText(`<ansi fg="red">You are heavily encumbered.</ansi> Your combat and movement are significantly penalized.`)
	} else if encPct >= 1.0 {
		user.SendText(`<ansi fg="yellow">You are encumbered.</ansi> Your combat and movement are penalized.`)
	} else if encPct >= 0.75 {
		user.SendText(`<ansi fg="yellow">You are carrying a moderate load.</ansi>`)
	}
	// No message for light load or unencumbered
}
