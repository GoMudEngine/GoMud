package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// TransferItemToFloor moves an item from a character's inventory to the room floor.
// Returns an error if the item cannot be removed from the character's inventory.
func TransferItemToFloor(item items.Item, char *characters.Character, userId int, mobInstanceId int, room *rooms.Room) error {
	// Remove item from character
	if !char.RemoveItem(item) {
		return fmt.Errorf("item not found in inventory")
	}

	// Add item to room floor (always succeeds)
	room.AddItem(item, false)

	// Fire ItemOwnership event for loss
	events.AddToQueue(events.ItemOwnership{
		UserId:        userId,
		MobInstanceId: mobInstanceId,
		Item:          item,
		Gained:        false,
	})

	return nil
}

// TransferItemToBackpack moves an item from a source (e.g., room floor) to a character's backpack.
// Parameters:
//   - item: the item being transferred
//   - toChar: the character receiving the item
//   - toUserId: user ID of the receiving character (0 for mobs)
//   - toMobInstanceId: mob instance ID of the receiving character (0 for players)
//   - removeFrom: function to remove the item from the source
//   - rollback: function to restore the item to the source if storage fails
//
// Returns an error if capacity is exceeded.
func TransferItemToBackpack(item items.Item, toChar *characters.Character, toUserId int, toMobInstanceId int, removeFrom func(items.Item), rollback func(items.Item)) error {
	// Remove from source
	removeFrom(item)

	// Try to store in destination
	if !toChar.StoreItem(item) {
		// Restore to source on failure
		rollback(item)
		return fmt.Errorf("inventory capacity exceeded")
	}

	// Fire ItemOwnership event for gain
	events.AddToQueue(events.ItemOwnership{
		UserId:        toUserId,
		MobInstanceId: toMobInstanceId,
		Item:          item,
		Gained:        true,
	})

	return nil
}

// TransferItemBetweenChars moves an item from one character to another.
// Rolls back if the destination cannot accept the item.
// Returns an error if the item cannot be removed from the source or stored in the destination.
func TransferItemBetweenChars(item items.Item, fromChar *characters.Character, fromUserId int, fromMobInstanceId int, toChar *characters.Character, toUserId int, toMobInstanceId int) error {
	// Remove from source
	if !fromChar.RemoveItem(item) {
		return fmt.Errorf("item not found in source inventory")
	}

	// Try to store in destination
	if !toChar.StoreItem(item) {
		// Rollback: restore to source
		if !fromChar.StoreItem(item) {
			// This should never happen since we just removed it
			return fmt.Errorf("critical error: failed to rollback item to source")
		}
		return fmt.Errorf("destination inventory capacity exceeded")
	}

	// Fire ItemOwnership events
	events.AddToQueue(events.ItemOwnership{
		UserId:        fromUserId,
		MobInstanceId: fromMobInstanceId,
		Item:          item,
		Gained:        false,
	})

	events.AddToQueue(events.ItemOwnership{
		UserId:        toUserId,
		MobInstanceId: toMobInstanceId,
		Item:          item,
		Gained:        true,
	})

	return nil
}

// TransferGold moves gold from one character to another.
// Validates that the source has sufficient gold.
// Returns an error if the amount is invalid or the source has insufficient funds.
func TransferGold(amount int, fromChar *characters.Character, toChar *characters.Character) error {
	if amount <= 0 {
		return fmt.Errorf("transfer amount must be positive")
	}

	if fromChar.Gold < amount {
		return fmt.Errorf("insufficient gold")
	}

	fromChar.Gold -= amount
	toChar.Gold += amount

	return nil
}

// FloorPickupGold moves gold from the room floor to a character.
// Validates that the amount is positive and available on the floor.
// Returns an error if the amount is invalid or insufficient gold is on the floor.
func FloorPickupGold(amount int, room *rooms.Room, char *characters.Character) error {
	if amount <= 0 {
		return fmt.Errorf("pickup amount must be positive")
	}

	if room.Gold < amount {
		return fmt.Errorf("insufficient gold on floor")
	}

	room.Gold -= amount
	char.Gold += amount

	return nil
}

// FloorDropGold moves gold from a character to the room floor.
// Validates that the character has sufficient gold and the amount is positive.
// Returns an error if the amount is invalid or the character has insufficient gold.
func FloorDropGold(amount int, char *characters.Character, room *rooms.Room) error {
	if amount <= 0 {
		return fmt.Errorf("drop amount must be positive")
	}

	if char.Gold < amount {
		return fmt.Errorf("insufficient gold in inventory")
	}

	char.Gold -= amount
	room.Gold += amount

	return nil
}
