package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
)

// GiveItemResult is the result of a GiveItemToChar call.
type GiveItemResult struct {
	Item  items.Item
	Found bool
	Err   error
}

// GiveItemToChar searches the actor's backpack for an item matching itemName,
// then atomically transfers it to toChar using TransferItemBetweenChars.
// If no matching item is found, Found is false and no transfer occurs.
func GiveItemToChar(actor Actor, itemName string, toChar *characters.Character, toUserId int, toMobInstanceId int) GiveItemResult {
	fromChar := actor.GetCharacter()

	matchItem, found := fromChar.FindInBackpack(itemName)
	if !found {
		return GiveItemResult{Found: false}
	}

	err := TransferItemBetweenChars(
		matchItem,
		fromChar,
		actor.GetUserId(),
		actor.GetMobInstanceId(),
		toChar,
		toUserId,
		toMobInstanceId,
	)

	return GiveItemResult{Item: matchItem, Found: true, Err: err}
}

// GiveGoldToChar transfers the given amount of gold from the actor's character
// to the target character. Delegates to TransferGold for validation.
func GiveGoldToChar(actor Actor, amount int, to *characters.Character) error {
	return TransferGold(amount, actor.GetCharacter(), to)
}
