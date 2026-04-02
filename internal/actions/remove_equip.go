package actions

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// RemoveEquipResult is the result of a RemoveEquipment call.
type RemoveEquipResult struct {
	Item    items.Item
	Found   bool
	Removed bool // true when RemoveFromBody succeeded and item was stored/dropped
	Err     error
}

// RemoveEquipment removes a worn item from the actor's body and stores it in
// their backpack (falling back to dropping it on the floor if the backpack is
// full). It cancels hidden buffs, fires the EquipmentChange event, and calls
// Validate(). Cursed-item checks, messaging, and all-remove loops remain in
// the callers.
func RemoveEquipment(actor Actor, itemName string) RemoveEquipResult {
	char := actor.GetCharacter()
	room := actor.GetRoom()

	matchItem, found := char.FindOnBody(itemName)
	if !found || matchItem.ItemId < 1 {
		return RemoveEquipResult{Found: false}
	}

	char.CancelBuffsWithFlag(buffs.Hidden)

	if !char.RemoveFromBody(matchItem) {
		// RemoveFromBody failed — item is still on body
		return RemoveEquipResult{Item: matchItem, Found: true, Removed: false}
	}

	if !char.StoreItem(matchItem) {
		// Backpack full — drop to floor as safety net
		room.AddItem(matchItem, false)
	}

	events.AddToQueue(events.EquipmentChange{
		UserId:        actor.GetUserId(),
		MobInstanceId: actor.GetMobInstanceId(),
		ItemsRemoved:  []items.Item{matchItem},
	})

	char.Validate()

	return RemoveEquipResult{Item: matchItem, Found: true, Removed: true}
}
