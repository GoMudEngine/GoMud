package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/itemvalue"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const (
	upgradePendingEquipKey = "plan:upgrade-gear:pending_equip"
	upgradeSellShopRoomKey = "plan:upgrade-gear:sell_shop_room"
)

func init() {
	RegisterPlanner("upgrade-gear", upgradeGearPlanner)
}

// upgradeGearPlanner: survey-worst-slot equipment shopping. Each tick the mob
// (idle, out of combat by selection contract):
//  1. Just bought last tick → gearup to wear it, clear flag.
//  2. Affordable in-stock upgrade → at shop: buy it (+ flag equip next tick);
//     else pathto the shop.
//  3. Upgrade exists but unaffordable → save up via the wealth-gold sell loop.
//  4. Nothing in stock anywhere → idle (Running, empty command).
func upgradeGearPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}

	// (1) Equip what we bought last tick.
	if mobMiscIntOr(mob, upgradePendingEquipKey, 0) == 1 {
		mobSetMisc(mob, upgradePendingEquipKey, 0)
		return PlanResult{Command: "gearup", Status: StatusRunning}
	}

	b := configs.GetBalanceConfig()
	reserve := goalParamIntOr(goal, "reserve", int(b.MobUpgradeGoldReserve))
	minDelta := float64(b.MobUpgradeMinDelta)
	profile := itemvalue.ProfileFor(mob.Archetype, mob.BehaviorArchetype)
	budget := mob.Character.Gold - reserve

	// (2) Affordable upgrade?
	if budget > 0 {
		if cand, ok := scanZoneUpgrades(mob, profile, budget, true, minDelta); ok {
			// Entering the buy flow: drop any stale save-up sell-shop room.
			// This goal's predicate is always false, so ClearPlanState only
			// fires on a goal SWITCH — within a continuous run across multiple
			// buy/sell cycles the sticky would otherwise persist a stale vendor.
			mobSetMisc(mob, upgradeSellShopRoomKey, 0)
			if mob.Character.RoomId == cand.ShopRoom {
				mobSetMisc(mob, upgradePendingEquipKey, 1)
				return PlanResult{Command: "buy " + cand.ItemName, Status: StatusRunning}
			}
			return PlanResult{Command: "pathto " + strconv.Itoa(cand.ShopRoom), Status: StatusRunning}
		}
	}

	// (3) Upgrade exists but unaffordable → save up by selling loot.
	if _, exists := scanZoneUpgrades(mob, profile, 0, false, minDelta); exists {
		if mobHasSellableItems(mob) {
			if mobInVendorRoom(mob) {
				return PlanResult{Command: "sell all", Status: StatusRunning}
			}
			room := mobMiscIntOr(mob, upgradeSellShopRoomKey, 0)
			if room == 0 {
				if r, ok := findShopInZoneBuying(mob); ok {
					room = r
					mobSetMisc(mob, upgradeSellShopRoomKey, room)
				}
			}
			if room != 0 {
				return PlanResult{Command: "pathto " + strconv.Itoa(room), Status: StatusRunning}
			}
		}
		// No buying vendor found (or nothing to sell) → wander hoping for loot.
		return PlanResult{Command: "wander", Status: StatusRunning}
	}

	// (4) Nothing in stock anywhere in zone → idle.
	return PlanResult{Status: StatusRunning}
}
