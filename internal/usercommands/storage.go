package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Storage(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if !room.IsStorage {

		user.SendText(`You are not at a storage location.` + term.CRLFStr)

		if len(room.Containers) > 0 {
			cName := ``
			for k := range room.Containers {
				cName = k
				break
			}
			user.SendText(fmt.Sprintf(`Maybe you meant to use the <ansi fg="command">put</ansi> command to <ansi fg="command">put</ansi> something into the <ansi fg="container">%s</ansi>?`, cName) + term.CRLFStr)
		}

		return true, nil
	}

	// Display storage list (no args, or bare "storage remove" which historically
	// showed the list before args were required).
	if rest == `` || rest == `remove` {

		slots := user.ItemStorage.GetSlots()
		slotNames := []string{}
		for _, slot := range slots {
			name := slot.Item.NameComplex()
			if slot.Count > 1 {
				name = fmt.Sprintf(`%s <ansi fg="uses-left">(x%d)</ansi>`, name, slot.Count)
			}
			slotNames = append(slotNames, name)
		}

		storageTxt, _ := templates.Process("character/storage", slotNames, user.UserId)
		user.SendText(storageTxt)

		return true, nil
	}

	if rest == `add` {
		user.SendText(`add what?` + term.CRLFStr)
		return true, nil
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 2 || (args[0] != `add` && args[0] != `remove`) {
		user.SendText(`Try <ansi fg="command">help storage</ansi> for more information about storage.` + term.CRLFStr)
		return true, nil
	}

	action := args[0]
	remaining := args[1:]

	// Parse optional leading quantity: "5 iron-ore", "all iron-ore", "all"
	qty := 1        // default: move 1
	qtyAll := false // true when user said "all"
	nameArgs := remaining

	if len(remaining) > 0 {
		if remaining[0] == `all` {
			qtyAll = true
			nameArgs = remaining[1:] // may be empty (add all) or have a name (add all iron-ore)
		} else if n, err := strconv.Atoi(remaining[0]); err == nil && n > 0 {
			qty = n
			nameArgs = remaining[1:]
		}
	}

	itemName := strings.Join(nameArgs, ` `)

	// ------------------------------------------------------------------ ADD
	if action == `add` {

		storageCap := room.StorageCapacity
		if storageCap <= 0 {
			storageCap = 20
		}
		if user.ItemStorage.SlotCount() >= storageCap {
			user.SendText(`Your storage is full.`)
			return true, nil
		}

		// storage add all — deposit everything from backpack AND component bag
		if qtyAll && itemName == `` {
			allItems := append([]items.Item{}, user.Character.GetAllBackpackItems()...)
			allItems = append(allItems, user.Character.ComponentItems...)

			deposited := 0
			for _, itm := range allItems {
				if user.ItemStorage.SlotCount() >= storageCap {
					break
				}
				if storageAddQuiet(user, itm) {
					deposited++
				}
			}
			if deposited > 0 {
				user.SendText(fmt.Sprintf(`You placed %d item(s) into storage.`, deposited))
			}
			return true, nil
		}

		// storage add all iron-ore — deposit all matching units
		if qtyAll {
			deposited := 0
			for {
				if user.ItemStorage.SlotCount() >= storageCap {
					break
				}
				itm, found := storageCarriedFind(user, itemName)
				if !found {
					break
				}
				if !storageAddQuiet(user, itm) {
					break
				}
				deposited++
			}
			switch deposited {
			case 0:
				user.SendText(fmt.Sprintf(`You don't have a %s to add to storage.%s`, itemName, term.CRLFStr))
			case 1:
				user.SendText(fmt.Sprintf(`You placed the <ansi fg="itemname">%s</ansi> into storage.`, itemName))
			default:
				user.SendText(fmt.Sprintf(`You placed %d <ansi fg="itemname">%s</ansi> into storage.`, deposited, itemName))
			}
			return true, nil
		}

		// storage add [N] iron-ore
		if itemName == `` {
			user.SendText(`add what?` + term.CRLFStr)
			return true, nil
		}

		deposited := 0
		for i := 0; i < qty; i++ {
			if user.ItemStorage.SlotCount() >= storageCap {
				user.SendText(`Your storage is full.`)
				break
			}
			itm, found := storageCarriedFind(user, itemName)
			if !found {
				if deposited > 0 {
					user.SendText(fmt.Sprintf(`You only had %d to deposit.`, deposited))
				} else {
					user.SendText(fmt.Sprintf(`You don't have a %s to add to storage.%s`, itemName, term.CRLFStr))
				}
				break
			}
			storageAddQuiet(user, itm)
			deposited++
		}
		switch deposited {
		case 0:
			// error already sent above
		case 1:
			user.SendText(fmt.Sprintf(`You placed the <ansi fg="itemname">%s</ansi> into storage.`, itemName))
		default:
			user.SendText(fmt.Sprintf(`You placed %d <ansi fg="itemname">%s</ansi> into storage.`, deposited, itemName))
		}

		return true, nil
	}

	// --------------------------------------------------------------- REMOVE
	// storage remove all — retrieve everything
	if qtyAll && itemName == `` {
		retrieved := 0
		for _, slot := range user.ItemStorage.GetSlots() {
			for i := 0; i < slot.Count; i++ {
				if !storageRemoveQuiet(user, slot.Item) {
					user.SendText(`You can't carry the rest.`)
					break
				}
				retrieved++
			}
		}
		if retrieved > 0 {
			user.SendText(fmt.Sprintf(`You retrieved %d item(s) from storage.`, retrieved))
		}
		return true, nil
	}

	// Support "all.itemname" diku syntax as sugar for "all iron-ore"
	if strings.HasPrefix(itemName, `all.`) {
		qtyAll = true
		itemName = itemName[4:]
	}

	// storage remove all.iron-ore (or "all iron-ore")
	if qtyAll {
		retrieved := 0
		for {
			itm, found := user.ItemStorage.FindItem(itemName)
			if !found {
				break
			}
			if !storageRemoveQuiet(user, itm) {
				if retrieved > 0 {
					user.SendText(`You can't carry the rest.`)
				}
				break
			}
			retrieved++
		}
		switch retrieved {
		case 0:
			user.SendText(fmt.Sprintf(`You don't have a %s in storage.`, itemName))
		case 1:
			user.SendText(fmt.Sprintf(`You removed the <ansi fg="itemname">%s</ansi> from storage.`, itemName))
		default:
			user.SendText(fmt.Sprintf(`You retrieved %d <ansi fg="itemname">%s</ansi> from storage.`, retrieved, itemName))
		}
		return true, nil
	}

	// Numeric slot index: "storage remove 3"
	if itmIdx, err := strconv.Atoi(itemName); err == nil && itmIdx > 0 {
		slots := user.ItemStorage.GetSlots()
		itmIdx-- // convert to 0-based
		if itmIdx >= len(slots) {
			user.SendText(`You don't have that item in storage.`)
			return true, nil
		}
		storageRemoveOne(user, slots[itmIdx].Item)
		return true, nil
	}

	// storage remove [N] iron-ore
	retrieved := 0
	for i := 0; i < qty; i++ {
		itm, found := user.ItemStorage.FindItem(itemName)
		if !found {
			if retrieved > 0 {
				user.SendText(fmt.Sprintf(`You can only retrieve %d from storage.`, retrieved))
			} else {
				user.SendText(fmt.Sprintf(`You don't have a %s in storage.`, itemName))
			}
			break
		}
		if !storageRemoveOne(user, itm) {
			if retrieved == 0 {
				user.SendText(`You can't carry that!`)
			} else {
				user.SendText(`You can't carry the rest.`)
			}
			break
		}
		retrieved++
	}
	if retrieved > 1 {
		user.SendText(fmt.Sprintf(`You retrieved %d <ansi fg="itemname">%s</ansi> from storage.`, retrieved, itemName))
	}

	return true, nil
}

// storageCarriedFind searches the player's backpack AND component bag for the
// first item matching itemName.
func storageCarriedFind(user *users.UserRecord, itemName string) (items.Item, bool) {
	itm, found := user.Character.FindInBackpack(itemName)
	if found {
		return itm, true
	}
	if len(user.Character.ComponentItems) > 0 {
		close, full := items.FindMatchIn(itemName, user.Character.ComponentItems...)
		if full.ItemId != 0 {
			return full, true
		}
		if close.ItemId != 0 {
			return close, true
		}
	}
	return items.Item{}, false
}

// storageAddQuiet transfers one unit of itm from the player to storage.
// No player-facing message is sent; the caller is expected to print a
// summary. Returns false if RemoveItem fails (item not on player).
func storageAddQuiet(user *users.UserRecord, itm items.Item) bool {
	if !user.Character.RemoveItem(itm) {
		return false
	}
	user.ItemStorage.AddItem(itm)
	events.AddToQueue(events.ItemOwnership{
		UserId: user.UserId,
		Item:   itm,
		Gained: false,
	})
	return true
}

// storageRemoveQuiet transfers one unit of itm from storage to the player.
// No player-facing message is sent; the caller is expected to print a
// summary. Returns false if the player cannot carry the item.
func storageRemoveQuiet(user *users.UserRecord, itm items.Item) bool {
	if !user.Character.StoreItem(itm) {
		return false
	}
	events.AddToQueue(events.ItemOwnership{
		UserId: user.UserId,
		Item:   itm,
		Gained: true,
	})
	user.ItemStorage.RemoveItem(itm)
	return true
}

// storageRemoveOne transfers one unit of itm from storage to the player and
// sends a confirmation message. Returns false if carry capacity blocks it.
func storageRemoveOne(user *users.UserRecord, itm items.Item) bool {
	if !storageRemoveQuiet(user, itm) {
		return false
	}
	user.SendText(fmt.Sprintf(`You removed the <ansi fg="itemname">%s</ansi> from storage.`, itm.DisplayName()))
	return true
}
