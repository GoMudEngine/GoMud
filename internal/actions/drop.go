package actions

import (
	"github.com/GoMudEngine/GoMud/internal/items"
)

// DropItemResult is the result of a DropItem call.
type DropItemResult struct {
	Item  items.Item
	Found bool
	Err   error
}

// DropItem removes an item from the actor's backpack and places it on the
// room floor. It handles the FindInBackpack → TransferItemToFloor sequence
// and fires the ItemOwnership event. Messaging and actor-specific guards
// (gold paths, all-drop loops, grenade handling) remain in the callers.
func DropItem(actor Actor, itemName string) DropItemResult {
	char := actor.GetCharacter()
	room := actor.GetRoom()

	matchItem, found := char.FindInBackpack(itemName)
	if !found {
		return DropItemResult{Found: false}
	}

	err := TransferItemToFloor(matchItem, char, actor.GetUserId(), actor.GetMobInstanceId(), room)
	return DropItemResult{Item: matchItem, Found: true, Err: err}
}
