package actions

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
)

// EquipItemResult is the result of an EquipItem call.
type EquipItemResult struct {
	Item           items.Item
	DisplacedItems []items.Item
	Found          bool
	Equipped       bool
	FailureReason  string
}

// EquipItem takes a named item from the actor's backpack and equips it.
// Displaced items (swapped-out gear) are stored back to the backpack, or
// dropped to the floor if the backpack is full — no item loss allowed.
// CancelBuffsWithFlag(Hidden), Validate(), and EquipmentChange are all
// handled here. Messaging, arm-slot logic, buff onStart triggers, and
// quest-engine notifications remain in the callers.
func EquipItem(actor Actor, itemName string) EquipItemResult {
	char := actor.GetCharacter()

	matchItem, found := char.FindInBackpack(itemName)
	if !found {
		return EquipItemResult{Found: false}
	}

	// Tentatively remove from backpack so Wear() can place it on the body.
	char.RemoveItem(matchItem)

	displaced, newItemWorn, failureReason := char.Wear(matchItem)
	if !newItemWorn {
		// Put it back — equip failed.
		char.StoreItem(matchItem)
		return EquipItemResult{
			Item:          matchItem,
			Found:         true,
			Equipped:      false,
			FailureReason: failureReason,
		}
	}

	char.Awareness.TransitionToRevealing(state.TransitionReason{
		Trigger: awareness.TriggerForceVisible,
	})

	// Return displaced gear to backpack; drop to floor on overflow.
	for _, di := range displaced {
		if di.ItemId != 0 {
			if !char.StoreItem(di) {
				actor.GetRoom().AddItem(di, false)
			}
		}
	}

	char.Validate()

	events.AddToQueue(events.EquipmentChange{
		UserId:        actor.GetUserId(),
		MobInstanceId: actor.GetMobInstanceId(),
		ItemsWorn:     []items.Item{matchItem},
		ItemsRemoved:  displaced,
	})

	return EquipItemResult{
		Item:           matchItem,
		DisplacedItems: displaced,
		Found:          true,
		Equipped:       true,
	}
}

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

	char.Awareness.TransitionToRevealing(state.TransitionReason{
		Trigger: awareness.TriggerForceVisible,
	})

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
