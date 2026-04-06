package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// enchantSlotCandidate pairs a slot label with the item occupying it.
type enchantSlotCandidate struct {
	SlotLabel string
	Item      items.Item
}

// resolveEnchantSlot finds the equipment slot to enchant based on the recipe's
// target_type and the player's optional slot specifier (e.g. "weapon#2", "2.ring").
//
// Returns the slot label (for GetSlotPointer), the item pointer, and an error
// message string. On success errMsg is empty.
func resolveEnchantSlot(equipment *characters.Worn, targetType string, specifier string) (slotLabel string, item *items.Item, errMsg string) {

	// Build ordered candidate list for this target type
	candidates := buildSlotCandidates(equipment, targetType)

	if len(candidates) == 0 {
		return "", nil, fmt.Sprintf(
			"You don't have a %s equipped to enchant.",
			strings.ReplaceAll(targetType, "_", " "))
	}

	// No specifier — auto-select if unambiguous
	if specifier == "" {
		if len(candidates) == 1 {
			c := candidates[0]
			ptr := equipment.GetSlotPointer(c.SlotLabel)
			return c.SlotLabel, ptr, ""
		}
		// Multiple candidates — tell the player to specify
		lines := "Which slot? You have multiple options:\n"
		for i, c := range candidates {
			lines += fmt.Sprintf("  %s#%d — %s\n",
				slotSpecName(targetType), i+1, c.Item.DisplayName())
		}
		return "", nil, lines
	}

	// Parse the specifier for #N suffix or N. prefix
	baseName, idx := util.GetMatchNumber(specifier)

	// Validate the base name matches the target type's player-facing name
	expectedName := slotSpecName(targetType)
	if baseName != "" && !strings.EqualFold(baseName, expectedName) {
		return "", nil, fmt.Sprintf(
			"That recipe targets %s slots, not %s.",
			expectedName, baseName)
	}

	// idx is 1-based
	if idx < 1 || idx > len(candidates) {
		return "", nil, fmt.Sprintf(
			"There is no %s#%d equipped.", expectedName, idx)
	}

	c := candidates[idx-1]
	ptr := equipment.GetSlotPointer(c.SlotLabel)
	if ptr == nil || ptr.ItemId < 1 {
		return "", nil, "That equipment slot is empty."
	}
	return c.SlotLabel, ptr, ""
}

// slotSpecName returns the player-facing name for a target type.
// This is what the player types as the slot specifier.
func slotSpecName(targetType string) string {
	switch targetType {
	case "offhand":
		return "shield"
	default:
		return targetType
	}
}

// buildSlotCandidates returns an ordered list of equipped items matching the
// given target type. Only slots with an item whose ItemSpec.Type matches are
// included.
func buildSlotCandidates(eq *characters.Worn, targetType string) []enchantSlotCandidate {
	var candidates []enchantSlotCandidate

	// Define the slot scan order per target type.
	type slotEntry struct {
		label string
		item  items.Item
	}

	var scanOrder []slotEntry

	switch targetType {
	case "weapon":
		scanOrder = []slotEntry{
			{"wielded", eq.Weapon},
			{"offhand", eq.Offhand},
			{"extra arm 1", eq.ExtraArm1},
			{"extra arm 2", eq.ExtraArm2},
			{"extra arm 3", eq.ExtraArm3},
			{"extra arm 4", eq.ExtraArm4},
		}
	case "offhand": // shields
		scanOrder = []slotEntry{
			{"offhand", eq.Offhand},
			{"extra arm 1", eq.ExtraArm1},
			{"extra arm 2", eq.ExtraArm2},
			{"extra arm 3", eq.ExtraArm3},
			{"extra arm 4", eq.ExtraArm4},
		}
	case "head":
		scanOrder = []slotEntry{{"worn - head", eq.Head}}
	case "neck":
		scanOrder = []slotEntry{{"worn - neck", eq.Neck}}
	case "shoulders":
		scanOrder = []slotEntry{{"worn - shoulders", eq.Shoulders}}
	case "body":
		scanOrder = []slotEntry{{"worn - body", eq.Body}}
	case "back":
		scanOrder = []slotEntry{{"worn - back", eq.Back}}
	case "belt":
		scanOrder = []slotEntry{{"worn - belt", eq.Belt}}
	case "wrist":
		scanOrder = []slotEntry{
			{"worn - wrist", eq.Wrist1},
			{"worn - wrist2", eq.Wrist2},
			{"extra wrist 1", eq.ExtraWrist1},
			{"extra wrist 2", eq.ExtraWrist2},
			{"extra wrist 3", eq.ExtraWrist3},
			{"extra wrist 4", eq.ExtraWrist4},
		}
	case "gloves":
		scanOrder = []slotEntry{{"worn - gloves", eq.Gloves}}
	case "ring":
		scanOrder = []slotEntry{
			{"worn - ring", eq.Ring},
			{"worn - ring2", eq.Ring2},
		}
	case "legs":
		scanOrder = []slotEntry{{"worn - legs", eq.Legs}}
	case "feet":
		scanOrder = []slotEntry{{"worn - feet", eq.Feet}}
	case "tail":
		scanOrder = []slotEntry{{"worn - tail", eq.Tail}}
	}

	for _, s := range scanOrder {
		if s.item.ItemId < 1 {
			continue
		}
		spec := s.item.GetSpec()
		if string(spec.Type) == targetType {
			candidates = append(candidates, enchantSlotCandidate{
				SlotLabel: s.label,
				Item:      s.item,
			})
		}
	}

	return candidates
}
