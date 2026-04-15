package actions

import (
	"github.com/GoMudEngine/GoMud/internal/items"
)

// GetItemResult is the result of a GetItemFromFloor call.
type GetItemResult struct {
	Item  items.Item
	Found bool
	Err   error
}

// GetItemFromFloor searches the room floor (or stash) for an item matching
// itemName, then atomically moves it into the actor's backpack using
// TransferItemToBackpack. The stash flag mirrors room.FindOnFloor /
// room.RemoveItem / room.AddItem semantics — pass true to search the stash.
func GetItemFromFloor(actor Actor, itemName string, stash bool) GetItemResult {
	room := actor.GetRoom()

	matchItem, found := room.FindOnFloor(itemName, stash)
	if !found {
		return GetItemResult{Found: false}
	}

	char := actor.GetCharacter()
	err := TransferItemToBackpack(
		matchItem,
		char,
		actor.GetUserId(),
		actor.GetMobInstanceId(),
		func(i items.Item) { room.RemoveItem(i, stash) },
		func(i items.Item) { room.AddItem(i, stash) },
	)

	return GetItemResult{Item: matchItem, Found: true, Err: err}
}

// GetGoldFromFloor moves all gold currently on the room floor into the actor's
// character wallet. It delegates to FloorPickupGold which validates the amount.
func GetGoldFromFloor(actor Actor, amount int) error {
	return FloorPickupGold(amount, actor.GetRoom(), actor.GetCharacter())
}
