package characters

import (
	"github.com/GoMudEngine/GoMud/internal/items"
)

// HandSlot describes one arm slot with its label and a pointer to the
// equipment field.
type HandSlot struct {
	Label   string
	ItemPtr *items.Item
}

// HandPair groups two adjacent arm slots. For half-pairs (odd arm count),
// Second.ItemPtr is nil.
type HandPair struct {
	First  HandSlot
	Second HandSlot
}

// GetHandPairs returns the available hand pairs for this character.
// Pair A is always present. Pairs B and C require extra-arms mutation.
// Odd mutation levels produce a "half pair" where only the First slot
// exists (Second.ItemPtr is nil) — no 2H allowed, only 1H/shield.
func (c *Character) GetHandPairs() []HandPair {
	pairs := []HandPair{
		{
			First:  HandSlot{"wielded", &c.Equipment.Weapon},
			Second: HandSlot{"offhand", &c.Equipment.Offhand},
		},
	}
	if c.ExtraArms >= 1 {
		p := HandPair{
			First: HandSlot{"extra arm 1", &c.Equipment.ExtraArm1},
		}
		if c.ExtraArms >= 2 {
			p.Second = HandSlot{"extra arm 2", &c.Equipment.ExtraArm2}
		}
		pairs = append(pairs, p)
	}
	if c.ExtraArms >= 3 {
		p := HandPair{
			First: HandSlot{"extra arm 3", &c.Equipment.ExtraArm3},
		}
		if c.ExtraArms >= 4 {
			p.Second = HandSlot{"extra arm 4", &c.Equipment.ExtraArm4}
		}
		pairs = append(pairs, p)
	}
	return pairs
}

// Is2H returns true if the item in this slot is a 2-handed weapon that
// occupies both slots of the pair.
func (s HandSlot) Is2H(c *Character) bool {
	if s.ItemPtr == nil || s.ItemPtr.ItemId < 1 {
		return false
	}
	return c.HandsRequired(*s.ItemPtr) >= 2
}

// IsEmpty returns true if the slot has no item or is a nil half-pair slot.
func (s HandSlot) IsEmpty() bool {
	return s.ItemPtr == nil || s.ItemPtr.ItemId < 1 || s.ItemPtr.IsDisabled()
}

// IsHalfPair returns true if the second slot of a pair doesn't exist
// (odd number of extra arms).
func (p HandPair) IsHalfPair() bool {
	return p.Second.ItemPtr == nil
}

// PairIsFree returns true if both slots in the pair are empty.
// A half-pair always returns false (can't fit a 2H).
func (p HandPair) PairIsFree() bool {
	if p.IsHalfPair() {
		return false
	}
	return p.First.IsEmpty() && p.Second.IsEmpty()
}

// PairOccupantCount returns how many non-empty items are in the pair.
func (p HandPair) PairOccupantCount() int {
	count := 0
	if !p.First.IsEmpty() {
		count++
	}
	if !p.IsHalfPair() && !p.Second.IsEmpty() {
		count++
	}
	return count
}

// FindFirstEmptySlot scans all pairs for the first empty slot.
// If isShield is true, skips the Weapon slot (arm 1) since shields
// can't go in the primary weapon hand.
func (c *Character) FindFirstEmptySlot(pairs []HandPair, isShield bool) *HandSlot {
	for pi := range pairs {
		p := &pairs[pi]
		// Skip first slot of first pair (Weapon) for shields
		if pi == 0 && isShield {
			if !p.IsHalfPair() && p.Second.IsEmpty() {
				return &p.Second
			}
			continue
		}
		// Don't place into a slot whose pair-partner holds a 2H
		// (the second slot of a 2H pair is implicitly consumed)
		if p.First.Is2H(c) {
			continue
		}
		if p.First.IsEmpty() {
			return &p.First
		}
		if !p.IsHalfPair() && p.Second.IsEmpty() {
			return &p.Second
		}
	}
	return nil
}

// FindFirstFreePair scans pairs for the first where both slots are empty
// and the pair is not a half-pair. Used for placing 2H weapons.
func FindFirstFreePair(pairs []HandPair) *HandPair {
	for i := range pairs {
		if pairs[i].PairIsFree() {
			return &pairs[i]
		}
	}
	return nil
}

// FindCheapestPairToDisplace finds the full pair with fewest occupants
// to make room for a 2H weapon. Skips half-pairs.
func FindCheapestPairToDisplace(pairs []HandPair) *HandPair {
	var best *HandPair
	bestCount := 3 // higher than max (2)
	for i := range pairs {
		if pairs[i].IsHalfPair() {
			continue
		}
		count := pairs[i].PairOccupantCount()
		if count < bestCount {
			bestCount = count
			best = &pairs[i]
		}
	}
	return best
}

// BestParryRating returns the highest ParryRating across all equipped
// items in weapon/arm slots.
func (c *Character) BestParryRating() int {
	best := 0
	for _, slot := range c.getWeaponAndArmItems() {
		if slot.ItemId < 1 {
			continue
		}
		if pr := slot.GetSpec().ParryRating; pr > best {
			best = pr
		}
	}
	return best
}

// BestBlockRating returns the highest BlockRating across all equipped
// items in weapon/arm slots.
func (c *Character) BestBlockRating() int {
	best := 0
	for _, slot := range c.getWeaponAndArmItems() {
		if slot.ItemId < 1 {
			continue
		}
		if br := slot.GetSpec().BlockRating; br > best {
			best = br
		}
	}
	return best
}

// HasAnyShield returns true if any arm slot holds a shield-type item.
func (c *Character) HasAnyShield() bool {
	slots := c.getWeaponAndArmItems()
	for _, slot := range slots {
		if slot.ItemId < 1 {
			continue
		}
		spec := slot.GetSpec()
		if spec.Type == items.Offhand && (spec.DamageReduction > 0 || spec.Subtype == items.Wearable) {
			return true
		}
	}
	return false
}

// getWeaponAndArmItems returns item copies from all weapon/arm slots.
func (c *Character) getWeaponAndArmItems() []items.Item {
	slots := []items.Item{
		c.Equipment.Weapon,
		c.Equipment.Offhand,
	}
	if c.ExtraArms >= 1 {
		slots = append(slots, c.Equipment.ExtraArm1)
	}
	if c.ExtraArms >= 2 {
		slots = append(slots, c.Equipment.ExtraArm2)
	}
	if c.ExtraArms >= 3 {
		slots = append(slots, c.Equipment.ExtraArm3)
	}
	if c.ExtraArms >= 4 {
		slots = append(slots, c.Equipment.ExtraArm4)
	}
	return slots
}
