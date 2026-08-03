package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Equip(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if refuseWhileBusy(user, `change equipment`) {
		return true, nil
	}

	if rest == "all" {
		return Gearup(``, user, room, flags)
	}

	if rest == "" {
		user.SendText(messaging.CategorySystem, `Wear WHAT?`)
		return true, nil
	}

	// Check for arm#N / N.arm suffix for arm slot targeting (arms 1-6).
	// Also supports legacy "arm1" through "arm6" plain suffix.
	targetArmSlot := 0
	restLower := strings.ToLower(rest)
	words := strings.Fields(restLower)
	if len(words) >= 2 {
		lastWord := words[len(words)-1]
		armBase, armNum := util.GetMatchNumber(lastWord)
		if armBase == "arm" && armNum >= 1 && armNum <= 6 {
			targetArmSlot = armNum
		} else if len(lastWord) == 4 && strings.HasPrefix(lastWord, "arm") {
			// Legacy: "arm1" through "arm6" (no separator)
			if n := lastWord[3] - '0'; n >= 1 && n <= 6 {
				targetArmSlot = int(n)
			}
		}
		if targetArmSlot > 0 {
			// Strip the last word from rest
			lastSpaceIdx := strings.LastIndex(rest, " ")
			rest = strings.TrimSpace(rest[:lastSpaceIdx])
		}
	}

	// Check whether the user has an item in their inventory that matches.
	// Equippable-first: prefer something that can actually be worn/wielded
	// (`wear stillwater` → the pendant, not the raw pearl), falling back to
	// the unfiltered match so the fashionable-flavor rejection still fires
	// when the only match truly isn't equipment.
	matchItem, found := user.Character.FindInBackpackWhere(rest, func(it items.Item) bool {
		spec := it.GetSpec()
		return spec.Type == items.Weapon || spec.Subtype == items.Wearable
	})
	if !found {
		matchItem, found = user.Character.FindInBackpack(rest)
	}

	if !found {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You don't have a "%s" to wear.`, rest))
	} else {

		iSpec := matchItem.GetSpec()
		if iSpec.Type != items.Weapon && iSpec.Subtype != items.Wearable {
			user.SendText(messaging.CategorySystem,
				fmt.Sprintf(`Your <ansi fg="item">%s</ansi> doesn't look very fashionable.`, matchItem.DisplayName()),
			)
			return true, nil
		}

		// Handle direct arm slot equip (arms 1-6)
		if targetArmSlot > 0 {
			// Only weapons and shields (offhand) can go in arm slots
			if iSpec.Type != items.Weapon && iSpec.Type != items.Offhand {
				user.SendText(messaging.CategorySystem, `You can only wield weapons or shields in arm slots.`)
				return true, nil
			}
			// Shields cannot go in arm 1 (primary weapon hand)
			if iSpec.Type == items.Offhand && targetArmSlot == 1 {
				user.SendText(messaging.CategorySystem, `You can't put a shield in your primary weapon hand (arm 1).`)
				return true, nil
			}

			// Validate the character has this arm
			pairs := user.Character.GetHandPairs()
			pairIdx := (targetArmSlot - 1) / 2
			slotInPair := (targetArmSlot - 1) % 2

			if pairIdx >= len(pairs) {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`You don't have arm %d.`, targetArmSlot))
				return true, nil
			}
			pair := &pairs[pairIdx]

			// Check if targeting the second slot of a half-pair
			if slotInPair == 1 && pair.IsHalfPair() {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`You don't have arm %d.`, targetArmSlot))
				return true, nil
			}

			handsReq := user.Character.HandsRequired(matchItem)

			// 2H weapons must go in odd-numbered arms (first slot of a pair)
			if handsReq >= 2 {
				if slotInPair != 0 {
					user.SendText(messaging.CategorySystem, `A two-handed weapon needs a pair of arms. Try arm 1, 3, or 5.`)
					return true, nil
				}
				if pair.IsHalfPair() {
					user.SendText(messaging.CategorySystem, `That arm doesn't have a partner for a two-handed weapon.`)
					return true, nil
				}
			}

			// Determine which slots to displace
			var displaced []items.Item
			targetSlot := &pair.First
			if slotInPair == 1 {
				targetSlot = &pair.Second
			}

			// Check for cursed items before displacement
			if !targetSlot.IsEmpty() && targetSlot.ItemPtr.IsCursed() {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`Your <ansi fg="item">%s</ansi> is cursed and can't be removed!`, targetSlot.ItemPtr.DisplayName()))
				return true, nil
			}

			// For 2H, also check/displace the second slot
			if handsReq >= 2 {
				if !pair.Second.IsEmpty() && pair.Second.ItemPtr.IsCursed() {
					user.SendText(messaging.CategorySystem, fmt.Sprintf(`Your <ansi fg="item">%s</ansi> is cursed and can't be removed!`, pair.Second.ItemPtr.DisplayName()))
					return true, nil
				}
				// Also check if the first slot holds a 2H (its partner is implicitly occupied)
				if !pair.First.IsEmpty() && pair.First.ItemPtr.IsCursed() {
					user.SendText(messaging.CategorySystem, fmt.Sprintf(`Your <ansi fg="item">%s</ansi> is cursed and can't be removed!`, pair.First.ItemPtr.DisplayName()))
					return true, nil
				}
			}

			// If the partner slot holds a 2H weapon, we need to displace it too
			if slotInPair == 1 && pair.First.Is2H(user.Character) {
				if pair.First.ItemPtr.IsCursed() {
					user.SendText(messaging.CategorySystem, fmt.Sprintf(`Your <ansi fg="item">%s</ansi> is cursed and can't be removed!`, pair.First.ItemPtr.DisplayName()))
					return true, nil
				}
				displaced = append(displaced, *pair.First.ItemPtr)
				*pair.First.ItemPtr = items.Item{}
			}

			// Displace current occupant(s)
			if !targetSlot.IsEmpty() {
				displaced = append(displaced, *targetSlot.ItemPtr)
				*targetSlot.ItemPtr = items.Item{}
			}
			if handsReq >= 2 && !pair.Second.IsEmpty() {
				displaced = append(displaced, *pair.Second.ItemPtr)
				*pair.Second.ItemPtr = items.Item{}
			}

			// Place the new item
			user.Character.CancelBuffsWithFlag(buffs.Hidden)
			user.Character.RemoveItem(matchItem)
			*targetSlot.ItemPtr = matchItem

			// Return displaced items to backpack
			for _, old := range displaced {
				if old.ItemId > 0 {
					user.SendText(messaging.CategorySystem, fmt.Sprintf(`You remove your <ansi fg="item">%s</ansi> and return it to your backpack.`, old.DisplayName()))
					user.Character.StoreItem(old)
				}
			}

			// Build wield/wear message
			armLabel := pair.First.Label
			if slotInPair == 1 {
				armLabel = pair.Second.Label
			}
			if iSpec.Type == items.Offhand {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`You equip your <ansi fg="item">%s</ansi> in your %s.`, matchItem.DisplayName(), armLabel))
			} else {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`You wield your <ansi fg="item">%s</ansi> in your %s.`, matchItem.DisplayName(), armLabel))
			}
			room.SendTextVisual(messaging.CategoryEquipment,
				fmt.Sprintf(`<ansi fg="username">%s</ansi> equips their <ansi fg="item">%s</ansi>.`, user.Character.Name, matchItem.DisplayName()),
				user.UserId,
			)

			user.Character.Validate(true)
			events.AddToQueue(events.EquipmentChange{
				UserId:       user.UserId,
				ItemsWorn:    []items.Item{matchItem},
				ItemsRemoved: displaced,
			})

			// Trigger any outstanding buff onStart events
			if len(iSpec.WornBuffIds) > 0 {
				for _, buff := range user.Character.Buffs.List {
					if buff.OnStartWaiting {
						user.Character.TrackBuffStarted(buff.BuffId)
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

			return true, nil
		}

		// Delegate core equip logic to shared action.
		actor := &actions.UserActor{User: user, Room: room}
		result := actions.EquipItem(actor, rest)

		if result.Equipped {

			for _, oldItem := range result.DisplacedItems {
				if oldItem.ItemId != 0 {
					user.SendText(messaging.CategorySystem,
						fmt.Sprintf(`You remove your <ansi fg="item">%s</ansi> and return it to your backpack.`, oldItem.DisplayName()),
					)
					room.SendTextVisual(messaging.CategoryEquipment,
						fmt.Sprintf(`<ansi fg="username">%s</ansi> removes their <ansi fg="item">%s</ansi> and stores it away.`, user.Character.Name, oldItem.DisplayName()),
						user.UserId,
					)
				}
			}

			if result.Item.GetSpec().Subtype == items.Wearable {
				user.SendText(messaging.CategorySystem,
					fmt.Sprintf(`You wear your <ansi fg="item">%s</ansi>.`, result.Item.DisplayName()),
				)
				room.SendTextVisual(messaging.CategoryEquipment,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> puts on their <ansi fg="item">%s</ansi>.`, user.Character.Name, result.Item.DisplayName()),
					user.UserId,
				)
			} else {
				user.SendText(messaging.CategorySystem,
					fmt.Sprintf(`You wield your <ansi fg="item">%s</ansi>. You're feeling dangerous.`, result.Item.DisplayName()),
				)
				room.SendTextVisual(messaging.CategoryEquipment,
					fmt.Sprintf(`<ansi fg="username">%s</ansi> wields their <ansi fg="item">%s</ansi>.`, user.Character.Name, result.Item.DisplayName()),
					user.UserId,
				)
			}

			// Trigger any outstanding buff onStart events
			if len(result.Item.GetSpec().WornBuffIds) > 0 {
				for _, buff := range user.Character.Buffs.List {
					if buff.OnStartWaiting {
						user.Character.TrackBuffStarted(buff.BuffId)
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
			user.SendText(messaging.CategorySystem, failureReason)
		}

	}

	return true, nil
}
