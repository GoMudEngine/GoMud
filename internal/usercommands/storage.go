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
			for k, _ := range room.Containers {
				cName = k
				break
			}
			user.SendText(fmt.Sprintf(`Maybe you meant to use the <ansi fg="command">put</ansi> command to <ansi fg="command">put</ansi> something into the <ansi fg="container">%s</ansi>?`, cName) + term.CRLFStr)
		}

		return true, nil
	}

	itemsInStorage := user.ItemStorage.GetItems()

	if rest == `` || rest == `remove` {

		itemNames := []string{}
		for _, item := range itemsInStorage {
			itemNames = append(itemNames, item.NameComplex())
		}

		storageTxt, _ := templates.Process("character/storage", itemNames, user.UserId)
		user.SendText(storageTxt)

		return true, nil
	}

	if rest == `add` || rest == `remove` {
		user.SendText(fmt.Sprintf(`%s what?%s`, rest, term.CRLFStr))
		return true, nil
	}

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) < 2 || (args[0] != `add` && args[0] != `remove`) {
		user.SendText(`Try <ansi fg="command">help storage</ansi> for more information about storage.` + term.CRLFStr)
		return true, nil
	}

	action := args[0]
	itemName := strings.Join(args[1:], ` `)

	if action == `add` {

		storageCap := room.StorageCapacity
		if storageCap <= 0 {
			storageCap = 20
		}
		spaceLeft := storageCap - len(itemsInStorage)
		if spaceLeft < 1 {
			user.SendText(`Your storage is full.`)
			return true, nil
		}

		if itemName == `all` {

			for _, itm := range user.Character.GetAllBackpackItems() {
				Storage(fmt.Sprintf(`add !%d`, itm.ItemId), user, room, flags)

				spaceLeft--
				if spaceLeft < 0 {
					break
				}
			}

			return true, nil
		}

		itm, found := user.Character.FindInBackpack(itemName)

		if !found {
			user.SendText(fmt.Sprintf(`You don't have a %s to add to storage.%s`, itemName, term.CRLFStr))
			return true, nil
		}

		user.Character.RemoveItem(itm)
		user.ItemStorage.AddItem(itm)

		events.AddToQueue(events.ItemOwnership{
			UserId: user.UserId,
			Item:   itm,
			Gained: false,
		})

		user.SendText(fmt.Sprintf(`You placed the <ansi fg="itemname">%s</ansi> into storage.`, itm.DisplayName()))

	} else if action == `remove` {

		if itemName == `all` {

			for _, itm := range user.ItemStorage.GetItems() {
				Storage(fmt.Sprintf(`remove !%d`, itm.ItemId), user, room, flags)
			}

			return true, nil
		}

		var itm items.Item
		var found bool = false
		itmIdx, _ := strconv.Atoi(itemName)

		if itmIdx > 0 {
			itmIdx -= 1
			for i, storageItm := range itemsInStorage {
				if itmIdx == i {
					itm = storageItm
					found = true
					break
				}
			}

		} else {
			itm, found = user.ItemStorage.FindItem(itemName)
		}

		if !found {
			user.SendText(fmt.Sprintf(`You don't have a %s in storage.`, itemName))
			return true, nil
		}

		if user.Character.StoreItem(itm) {

			events.AddToQueue(events.ItemOwnership{
				UserId: user.UserId,
				Item:   itm,
				Gained: true,
			})

			user.ItemStorage.RemoveItem(itm)

			user.SendText(fmt.Sprintf(`You removed the <ansi fg="itemname">%s</ansi> from storage.`, itm.DisplayName()))

		} else {
			user.SendText(`You can't carry that!`)
		}

	}

	return true, nil
}
