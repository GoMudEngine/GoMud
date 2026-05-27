package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

const wealthGoldShopRoomKey = "plan:wealth-gold:target_shop_room"

func init() {
	RegisterPlanner("wealth-gold", wealthGoldPlanner)
}

// wealthGoldPlanner is a sell-loot loop. The mob navigates to a vendor
// in its zone, sells sellable inventory, and repeats until the gold
// target is met.
//
// Branches per tick:
//   - target <= 0 or mob nil → Failure (misconfigured goal)
//   - mob.Gold >= target → Success (predicate satisfied)
//   - has sellable items + in vendor room → sell all
//   - has sellable items + not at vendor → resolve sticky
//     target_shop_room from MiscData + pathto
//   - no sellable items → wander (waiting for loot drop / forage)
func wealthGoldPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	target := goalParamIntOr(goal, "target", 0)
	if target <= 0 {
		return PlanResult{Status: StatusFailure}
	}

	// Target met → predicate satisfies next tick.
	if mob.Character.Gold >= target {
		return PlanResult{Status: StatusSuccess}
	}

	hasSellable := mobHasSellableItems(mob)

	// Has sellable + currently in a vendor room → sell all.
	if hasSellable && mobInVendorRoom(mob) {
		return PlanResult{Command: "sell all", Status: StatusRunning}
	}

	// Has sellable + not at vendor → resolve target_shop_room (sticky
	// across ticks via MiscData) and pathto.
	if hasSellable {
		shopRoom := mobMiscIntOr(mob, wealthGoldShopRoomKey, 0)
		if shopRoom == 0 {
			room, ok := findShopInZoneBuying(mob)
			if !ok {
				// No vendor found in zone → wander hoping for a better
				// zone or loot opportunity.
				return PlanResult{Command: "wander", Status: StatusRunning}
			}
			shopRoom = room
			mobSetMisc(mob, wealthGoldShopRoomKey, shopRoom)
		}
		return PlanResult{
			Command: "pathto " + strconv.Itoa(shopRoom),
			Status:  StatusRunning,
		}
	}

	// No sellable items → wander to find loot.
	return PlanResult{Command: "wander", Status: StatusRunning}
}

// mobHasSellableItems returns true if mob has at least one inventory item
// with a non-zero Value and no quest-token flag.
//
// Items with QuestToken set are quest-gated and must not be sold.
// Items with Value == 0 (e.g. raw materials with no vendor value) are
// excluded to avoid wasting a trip to the vendor.
func mobHasSellableItems(mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	for i := range mob.Character.Items {
		spec := mob.Character.Items[i].GetSpec()
		if spec.QuestToken != "" {
			continue
		}
		if spec.Value > 0 {
			return true
		}
	}
	return false
}

// mobInVendorRoom returns true if mob's current room hosts a registered
// shop in the same zone.
//
// Implementation: walks shops.AllShops(), filters to the mob's zone,
// and checks if any shop's RoomId matches mob.Character.RoomId.
func mobInVendorRoom(mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	for _, shop := range shops.AllShops() {
		if shop.Zone != mob.Character.Zone {
			continue
		}
		if shop.RoomId == mob.Character.RoomId {
			return true
		}
	}
	return false
}
