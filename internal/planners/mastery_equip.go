package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const masteryEquipShopRoomKey = "plan:mastery-equip:target_shop_room"

// equipRarityFallbackTier is the tier-50 engine fallback applied to any item
// whose ItemSpec.RarityTier == 0 (untagged). Mirrors the constant in the
// catalog's mastery_equip.go (internal/goals/catalog); duplicated here
// because the two packages cannot share it without an import cycle.
const equipRarityFallbackTier = 50

func init() {
	RegisterPlanner("mastery-equip", masteryEquipPlanner)
}

// masteryEquipPlanner: upgrade a slot to rarity tier ≥ target.
//   - Nil mob or missing required params → Failure.
//   - Slot already meets tier → Success.
//   - Shop in zone stocks any item + at shop → buy (slot-name as item hint)
//     then (next tick) wear.
//   - Shop in zone stocks any item + not at shop → pathto sticky room.
//   - No shop found → Failure.
//
// Note: findShopInZoneSelling is called with ("", 0) — "find any shop" —
// because a slot-aware stock filter is not available in 4.4. The mob
// navigates to the shop and issues "buy <slot>" as a best-effort hint;
// the actual item purchased depends on shop stock at that moment.
func masteryEquipPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	slot := goalParamStringOr(goal, "slot", "")
	minTier := goalParamIntOr(goal, "min_rarity_tier", 0)
	if slot == "" || minTier == 0 {
		return PlanResult{Status: StatusFailure}
	}

	// Predicate already satisfied — nothing to do.
	if mobSlotMeetsTier(mob, slot, minTier) {
		return PlanResult{Status: StatusSuccess}
	}

	// Resolve sticky shop room. Once found, remember it across ticks so the
	// mob doesn't re-scan every round.
	shopRoom := mobMiscIntOr(mob, masteryEquipShopRoomKey, 0)
	if shopRoom == 0 {
		room, ok := findShopInZoneSelling(mob, "", 0)
		if !ok {
			return PlanResult{Status: StatusFailure}
		}
		shopRoom = room
		mobSetMisc(mob, masteryEquipShopRoomKey, shopRoom)
	}

	// At the shop → issue a buy command using the slot name as the item hint.
	if mob.Character.RoomId == shopRoom {
		return PlanResult{Command: "buy " + slot, Status: StatusRunning}
	}

	// Not at the shop yet → navigate there.
	return PlanResult{Command: "pathto " + strconv.Itoa(shopRoom), Status: StatusRunning}
}

// ─── Slot-tier helpers ────────────────────────────────────────────────────────
//
// These duplicate the equivalent helpers in internal/goals/catalog/mastery_equip.go.
// They cannot be shared without either creating a common sub-package or an
// import cycle. The duplication is intentional and explicitly called out in the
// 4.4 plan. If the catalog's mapping changes, update both sites.

// mobSlotMeetsTier returns true if the item currently equipped in the named
// slot has rarity_tier ≥ minTier. Untagged items (RarityTier == 0) get the
// engine fallback tier (equipRarityFallbackTier = 50). An empty slot returns
// false regardless of minTier.
func mobSlotMeetsTier(mob *mobs.Mob, slot string, minTier int) bool {
	tier := equipMobSlotRarityTier(mob, slot)
	return tier >= minTier
}

// equipMobSlotRarityTier returns the rarity tier of the item equipped in slot.
// Returns 0 for an empty slot. Untagged real items get the engine fallback
// (equipRarityFallbackTier = 50).
func equipMobSlotRarityTier(mob *mobs.Mob, slot string) int {
	it := equipMobSlotItem(mob, slot)
	if it.ItemId == 0 {
		return 0 // empty slot
	}
	tier := it.GetSpec().RarityTier
	if tier == 0 {
		return equipRarityFallbackTier
	}
	return tier
}

// equipMobSlotItem returns the item equipped in the named slot, or a zero-
// value items.Item (ItemId == 0) for unknown or empty slots.
func equipMobSlotItem(mob *mobs.Mob, slot string) items.Item {
	ptr := mob.Character.Equipment.GetSlotPointer(equipSlotLabel(slot))
	if ptr == nil {
		return items.Item{}
	}
	return *ptr
}

// equipSlotLabel converts the short author-facing slot name used in goal
// params to the internal label string expected by Worn.GetSlotPointer.
// Mirrors slotLabel in internal/goals/catalog/mastery_equip.go.
func equipSlotLabel(slot string) string {
	switch slot {
	case "weapon":
		return "wielded"
	case "offhand":
		return "offhand"
	case "head":
		return "worn - head"
	case "neck":
		return "worn - neck"
	case "shoulders":
		return "worn - shoulders"
	case "body":
		return "worn - body"
	case "back":
		return "worn - back"
	case "belt":
		return "worn - belt"
	case "wrist", "wrist1":
		return "worn - wrist"
	case "wrist2":
		return "worn - wrist2"
	case "gloves":
		return "worn - gloves"
	case "ring":
		return "worn - ring"
	case "ring2":
		return "worn - ring2"
	case "legs":
		return "worn - legs"
	case "feet":
		return "worn - feet"
	case "tail":
		return "worn - tail"
	case "componentbag":
		return "worn - componentbag"
	}
	return "" // unknown → GetSlotPointer returns nil
}
