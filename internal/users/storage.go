package users

import "github.com/GoMudEngine/GoMud/internal/items"

// StorageSlot holds one logical "stack" of items in the bank.
// Stackable item types (components, food, potions, etc.) accumulate
// Count here so that 50 iron ore occupies a single slot.
// Non-stackable types always have Count == 1.
type StorageSlot struct {
	Item  items.Item `yaml:"item"`
	Count int        `yaml:"count"` // >= 1
}

// Storage is the per-player bank inventory.
//
// The canonical field is Slots. The legacy Items field is kept with
// omitempty for one release so that existing YAML files continue to
// deserialise cleanly. The migration path in
// characters/migrations.go folds Items into Slots on first load and
// sets Items to nil so the next save writes only Slots.
type Storage struct {
	Slots []StorageSlot `yaml:"slots,omitempty"`

	// Legacy field — read on load for migration, then cleared.
	// DO NOT remove until the next release after migration ships.
	Items []items.Item `yaml:"items,omitempty"`
}

// SlotCount returns the number of occupied slots (stacks), which is
// what the room's storagecapacity is measured against.
func (s *Storage) SlotCount() int {
	return len(s.Slots)
}

// GetItems returns one copy of each item in storage, one per logical
// unit (i.e. a slot with Count 3 contributes three separate Item
// values). This is the legacy-compatible view used by callers that
// iterate over individual items.
func (s *Storage) GetItems() []items.Item {
	out := make([]items.Item, 0, len(s.Slots))
	for _, slot := range s.Slots {
		for i := 0; i < slot.Count; i++ {
			out = append(out, slot.Item)
		}
	}
	return out
}

// GetSlots returns a copy of the slot list. Prefer this over GetItems
// when you need to reason about stacks (e.g. display, fee billing).
func (s *Storage) GetSlots() []StorageSlot {
	return append([]StorageSlot{}, s.Slots...)
}

// FindItem searches for the first item matching itemName, using the
// same fuzzy-match logic as items.FindMatchIn.
func (s *Storage) FindItem(itemName string) (items.Item, bool) {
	if itemName == `` {
		return items.Item{}, false
	}

	// Build a flat list of representative items (one per slot).
	reps := make([]items.Item, 0, len(s.Slots))
	for _, slot := range s.Slots {
		reps = append(reps, slot.Item)
	}

	closeMatch, fullMatch := items.FindMatchIn(itemName, reps...)

	if fullMatch.ItemId != 0 {
		return fullMatch, true
	}
	if closeMatch.ItemId != 0 {
		return closeMatch, true
	}
	return items.Item{}, false
}

// AddItem deposits one unit of itm into storage. If the item's spec
// is stackable and an identical stack already exists (per SameStack),
// the existing slot's Count is incremented. Otherwise a new slot is
// appended. Returns false if the item is invalid.
func (s *Storage) AddItem(i items.Item) bool {
	if i.ItemId < 1 {
		return false
	}

	spec := i.GetSpec()

	if spec.IsStackable() {
		// Look for an existing matching stack.
		for idx := range s.Slots {
			if items.SameStack(s.Slots[idx].Item, i) {
				s.Slots[idx].Count++
				return true
			}
		}
	}

	// New slot (also the path for non-stackable types).
	s.Slots = append(s.Slots, StorageSlot{Item: i, Count: 1})
	return true
}

// RemoveItem removes one logical unit of itm from storage. For
// stackable items it decrements the stack Count; when Count reaches
// zero the slot is dropped. For non-stackable items the exact slot
// (matched by UUID via items.Item.Equals) is removed. Returns false
// if no matching item was found.
func (s *Storage) RemoveItem(i items.Item) bool {
	spec := i.GetSpec()

	if spec.IsStackable() {
		// Decrement the first matching stack.
		for idx := range s.Slots {
			if items.SameStack(s.Slots[idx].Item, i) {
				s.Slots[idx].Count--
				if s.Slots[idx].Count <= 0 {
					s.Slots = append(s.Slots[:idx], s.Slots[idx+1:]...)
				}
				return true
			}
		}
		return false
	}

	// Non-stackable: match by UUID (original Equals semantics).
	for idx := range s.Slots {
		if s.Slots[idx].Item.Equals(i) {
			s.Slots = append(s.Slots[:idx], s.Slots[idx+1:]...)
			return true
		}
	}
	return false
}

// RemoveSlot removes the entire slot at position idx (0-based) and
// returns the slot that was removed. Panics if idx is out of range.
// Used by fee forfeiture to drop a whole stack at once.
func (s *Storage) RemoveSlot(idx int) StorageSlot {
	slot := s.Slots[idx]
	s.Slots = append(s.Slots[:idx], s.Slots[idx+1:]...)
	return slot
}
