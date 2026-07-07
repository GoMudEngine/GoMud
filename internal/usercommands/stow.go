package usercommands

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Stow deposits all crafting components (component bag + is_component backpack
// items) into the room's storage, leaving gear/consumables/quest items behind.
// Shorthand for the component-only case of `storage add all`.
func Stow(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	if !room.IsStorage {
		user.SendText(messaging.CategorySystem, `There is nowhere to stow anything here.`)
		return true, nil
	}
	storageCap := room.StorageCapacity
	if storageCap <= 0 {
		storageCap = 20
	}

	comps := append([]items.Item{}, user.Character.ComponentItems...)
	for _, itm := range user.Character.GetAllBackpackItems() {
		if itm.GetSpec().IsComponent {
			comps = append(comps, itm)
		}
	}

	stowed := 0
	full := false
	for _, itm := range comps {
		if user.ItemStorage.SlotCount() >= storageCap {
			full = true
			break
		}
		if storageAddQuiet(user, itm) {
			stowed++
		}
	}
	switch {
	case stowed == 0 && full:
		user.SendText(messaging.CategorySystem, `Storage is full.`)
	case stowed == 0:
		user.SendText(messaging.CategorySystem, `You have no crafting components to stow.`)
	case full:
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You stow %d component(s); storage filled up before the rest.`, stowed))
	default:
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You stow %d crafting component(s).`, stowed))
	}
	return true, nil
}
