package itemvalue

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutations"
)

// canonicalRank returns the sort order for a SlotName, used as
// a tiebreaker when two slots produce equal Score. Lower rank
// = preferred. Weapon=0 wins over Offhand=1 when scores tie
// (so a wand placed on an empty-handed mob lands in main hand).
func canonicalRank(s SlotName) int {
	for i, n := range canonicalSlotOrder {
		if n == s {
			return i
		}
	}
	return len(canonicalSlotOrder)
}

var canonicalSlotOrder = []SlotName{
	SlotWeapon, SlotOffhand,
	SlotExtraArm1, SlotExtraArm2, SlotExtraArm3, SlotExtraArm4,
	SlotHead, SlotNeck, SlotShoulders, SlotBody, SlotBack, SlotBelt,
	SlotWrist1, SlotWrist2,
	SlotExtraWrist1, SlotExtraWrist2, SlotExtraWrist3, SlotExtraWrist4,
	SlotGloves,
	SlotRing, SlotRing2,
	SlotLegs, SlotFeet,
	SlotTail, SlotComponentBag,
}

// extraArmsLevel returns the active level of the Extra Arms
// mutation on the character (0..4). 0 means no mutation.
//
// Adapted from plan: extra-arms is a single mutation with
// level 1-4, not separate per-level mutations.
func extraArmsLevel(char *characters.Character) int {
	lvl := mutations.GetMutationLevel(char.Mutations, "extra-arms")
	if lvl > 4 {
		lvl = 4 // cap at 4 per spec
	}
	return lvl
}

// hasTailMutation reports whether the character has the Tail
// mutation active.
func hasTailMutation(char *characters.Character) bool {
	return mutations.HasMutation(char.Mutations, "tail")
}

// compatibleSlotsFor returns the list of SlotNames where the
// candidate could be placed, respecting character mutations.
// Returns nil/empty list when the item is not equippable.
func compatibleSlotsFor(spec items.ItemSpec, char *characters.Character) []SlotName {
	switch spec.Type {
	case items.Weapon:
		if spec.Hands == items.TwoHanded {
			return []SlotName{SlotWeapon}
		}
		return []SlotName{SlotWeapon, SlotOffhand}
	case items.Offhand:
		return []SlotName{SlotOffhand}
	case items.Head:
		return []SlotName{SlotHead}
	case items.Neck:
		return []SlotName{SlotNeck}
	case items.Shoulders:
		return []SlotName{SlotShoulders}
	case items.Body:
		return []SlotName{SlotBody}
	case items.Back:
		return []SlotName{SlotBack}
	case items.Belt:
		return []SlotName{SlotBelt}
	case items.Wrist:
		slots := []SlotName{SlotWrist1, SlotWrist2}
		level := extraArmsLevel(char)
		if level >= 1 {
			slots = append(slots, SlotExtraWrist1)
		}
		if level >= 2 {
			slots = append(slots, SlotExtraWrist2)
		}
		if level >= 3 {
			slots = append(slots, SlotExtraWrist3)
		}
		if level >= 4 {
			slots = append(slots, SlotExtraWrist4)
		}
		return slots
	case items.Gloves:
		return []SlotName{SlotGloves}
	case items.Ring:
		return []SlotName{SlotRing, SlotRing2}
	case items.Legs:
		return []SlotName{SlotLegs}
	case items.Feet:
		return []SlotName{SlotFeet}
	case items.Tail:
		if hasTailMutation(char) {
			return []SlotName{SlotTail}
		}
		return nil
	case items.ComponentBag:
		return []SlotName{SlotComponentBag}
	}
	return nil
}

// itemInSlot returns the currently-equipped item in a given
// slot (or the zero-value items.Item if the slot is empty).
func itemInSlot(slot SlotName, char *characters.Character) items.Item {
	e := &char.Equipment
	switch slot {
	case SlotWeapon:
		return e.Weapon
	case SlotOffhand:
		return e.Offhand
	case SlotExtraArm1:
		return e.ExtraArm1
	case SlotExtraArm2:
		return e.ExtraArm2
	case SlotExtraArm3:
		return e.ExtraArm3
	case SlotExtraArm4:
		return e.ExtraArm4
	case SlotHead:
		return e.Head
	case SlotNeck:
		return e.Neck
	case SlotShoulders:
		return e.Shoulders
	case SlotBody:
		return e.Body
	case SlotBack:
		return e.Back
	case SlotBelt:
		return e.Belt
	case SlotWrist1:
		return e.Wrist1
	case SlotWrist2:
		return e.Wrist2
	case SlotExtraWrist1:
		return e.ExtraWrist1
	case SlotExtraWrist2:
		return e.ExtraWrist2
	case SlotExtraWrist3:
		return e.ExtraWrist3
	case SlotExtraWrist4:
		return e.ExtraWrist4
	case SlotGloves:
		return e.Gloves
	case SlotRing:
		return e.Ring
	case SlotRing2:
		return e.Ring2
	case SlotLegs:
		return e.Legs
	case SlotFeet:
		return e.Feet
	case SlotTail:
		return e.Tail
	case SlotComponentBag:
		return e.ComponentBag
	}
	return items.Item{}
}

// slotOf returns the slot in which the given equipped item
// currently resides, or "" if the item is not currently
// equipped on char.
func slotOf(item items.Item, char *characters.Character) SlotName {
	if item.ItemId == 0 {
		return ""
	}
	e := &char.Equipment
	pairs := []struct {
		slot SlotName
		got  items.Item
	}{
		{SlotWeapon, e.Weapon},
		{SlotOffhand, e.Offhand},
		{SlotExtraArm1, e.ExtraArm1},
		{SlotExtraArm2, e.ExtraArm2},
		{SlotExtraArm3, e.ExtraArm3},
		{SlotExtraArm4, e.ExtraArm4},
		{SlotHead, e.Head},
		{SlotNeck, e.Neck},
		{SlotShoulders, e.Shoulders},
		{SlotBody, e.Body},
		{SlotBack, e.Back},
		{SlotBelt, e.Belt},
		{SlotWrist1, e.Wrist1},
		{SlotWrist2, e.Wrist2},
		{SlotExtraWrist1, e.ExtraWrist1},
		{SlotExtraWrist2, e.ExtraWrist2},
		{SlotExtraWrist3, e.ExtraWrist3},
		{SlotExtraWrist4, e.ExtraWrist4},
		{SlotGloves, e.Gloves},
		{SlotRing, e.Ring},
		{SlotRing2, e.Ring2},
		{SlotLegs, e.Legs},
		{SlotFeet, e.Feet},
		{SlotTail, e.Tail},
		{SlotComponentBag, e.ComponentBag},
	}
	for _, p := range pairs {
		if p.got.ItemId == item.ItemId &&
			p.got.Uses == item.Uses &&
			p.got.EnchantType == item.EnchantType &&
			p.got.EnchantTier == item.EnchantTier {
			return p.slot
		}
	}
	return ""
}

// displacedItemsForSlot returns the items that would be
// unequipped when placing candidateSpec at targetSlot.
// - 2H weapon in Weapon: displaces both Weapon AND Offhand.
// - 1H in Offhand while current Weapon is 2H: displaces the 2H weapon.
// - Otherwise: the current item in that slot, or empty.
func displacedItemsForSlot(char *characters.Character, targetSlot SlotName, candidateSpec items.ItemSpec) []items.Item {
	var displaced []items.Item

	if targetSlot == SlotWeapon && candidateSpec.Type == items.Weapon && candidateSpec.Hands == items.TwoHanded {
		if char.Equipment.Weapon.ItemId > 0 {
			displaced = append(displaced, char.Equipment.Weapon)
		}
		if char.Equipment.Offhand.ItemId > 0 {
			displaced = append(displaced, char.Equipment.Offhand)
		}
		return displaced
	}

	if targetSlot == SlotOffhand && char.Equipment.Weapon.ItemId > 0 &&
		char.Equipment.Weapon.GetSpec().Hands == items.TwoHanded {
		displaced = append(displaced, char.Equipment.Weapon)
		return displaced
	}

	current := itemInSlot(targetSlot, char)
	if current.ItemId > 0 {
		displaced = append(displaced, current)
	}
	return displaced
}

// placementBonus returns the additional score a spec earns
// when placed at slot, evaluated against pre-swap char state.
// Used SYMMETRICALLY for candidate and displaced. DualWieldBonus
// is conditional on the pre-swap main hand having a 1H weapon.
func placementBonus(profile WeightProfile, spec items.ItemSpec, slot SlotName, char *characters.Character) float64 {
	bonus := 0.0

	if spec.Hands == items.TwoHanded {
		bonus += profile.TwoHandedBonus
	}

	if slot == SlotOffhand {
		if spec.Type == items.Weapon {
			if char.Equipment.Weapon.ItemId > 0 &&
				char.Equipment.Weapon.GetSpec().Hands != items.TwoHanded {
				bonus += profile.DualWieldBonus
			}
		} else if spec.Type == items.Offhand {
			bonus += profile.ShieldBonus
		}
	}

	return bonus
}

// ItemValueDelta returns the net effect of equipping candidate
// over char's current loadout under the given profile. Smart
// slot selection picks the optimal placement. Returns
// SwapDelta{Score: 0, Slot: "", Displaced: nil} when candidate
// is not equippable on this character.
func ItemValueDelta(char *characters.Character, profile WeightProfile, candidate items.Item) SwapDelta {
	candidateSpec := candidate.GetSpec()
	candidateRaw := ItemValue(candidateSpec, profile)

	slots := compatibleSlotsFor(candidateSpec, char)
	if len(slots) == 0 {
		return SwapDelta{}
	}

	best := SwapDelta{Slot: "", Displaced: nil}
	bestSet := false
	bestRank := -1

	for _, slot := range slots {
		displaced := displacedItemsForSlot(char, slot, candidateSpec)

		candidateAt := candidateRaw + placementBonus(profile, candidateSpec, slot, char)

		displacedTotal := 0.0
		for _, d := range displaced {
			dSpec := d.GetSpec()
			currentSlot := slotOf(d, char)
			displacedTotal += ItemValue(dSpec, profile) +
				placementBonus(profile, dSpec, currentSlot, char)
		}

		netScore := candidateAt - displacedTotal

		// Encumbrance tier penalty: filled in by Task 6.
		// For now, no encumbrance adjustment.

		rank := canonicalRank(slot)

		if !bestSet ||
			netScore > best.Score ||
			(netScore == best.Score && rank < bestRank) {
			best = SwapDelta{
				Score:     netScore,
				Slot:      slot,
				Displaced: displaced,
			}
			bestRank = rank
			bestSet = true
		}
	}

	return best
}
