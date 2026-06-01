package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/itemvalue"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
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

	// Eligibility gate: if this mob can never wear bought gear (non-combat
	// archetype, or animal species with a disabled weapon slot), don't shop.
	// Defensive — the shipped archetypes (thief, guard_captain) all pass; this
	// guards against future seeding of upgrade-gear onto an ineligible mob.
	if !itemvalue.CanEquipFromGive(&mob.Character, mob.BehaviorArchetype) {
		return PlanResult{Status: StatusRunning}
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

	// TEMP 5.3dbg — remove after diagnosis
	mudlog.Info("[5.3dbg] upgradeGearPlanner entry",
		"mob", mob.Character.Name, "mob_id", mob.MobId,
		"gold", mob.Character.Gold, "reserve", reserve,
		"budget", budget, "minDelta", minDelta)

	// (2) Affordable upgrade?
	if budget > 0 {
		cand, ok := scanZoneUpgrades(mob, profile, budget, true, minDelta)
		// TEMP 5.3dbg — remove after diagnosis
		mudlog.Info("[5.3dbg] upgradeGearPlanner affordable-scan",
			"mob", mob.Character.Name, "mob_id", mob.MobId,
			"ok", ok,
			"item", cand.ItemName, "delta", cand.Delta, "price", cand.Price, "shop_room", cand.ShopRoom)
		if ok {
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
	saveUpCand, exists := scanZoneUpgrades(mob, profile, 0, false, minDelta)
	// TEMP 5.3dbg — remove after diagnosis
	mudlog.Info("[5.3dbg] upgradeGearPlanner saveup-scan",
		"mob", mob.Character.Name, "mob_id", mob.MobId,
		"ok", exists,
		"item", saveUpCand.ItemName, "delta", saveUpCand.Delta, "price", saveUpCand.Price)
	if exists {
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
	// TEMP 5.3dbg — remove after diagnosis
	mudlog.Info("[5.3dbg] upgradeGearPlanner idle: no candidate, no surplus",
		"mob", mob.Character.Name, "mob_id", mob.MobId)
	return PlanResult{Status: StatusRunning}
}
