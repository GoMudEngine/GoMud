package planners

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const wealthItemShopRoomKey = "plan:wealth-item:target_shop_room"

func init() {
	RegisterPlanner("wealth-item", wealthItemPlanner)
}

// wealthItemPlanner acquires a specific item by buying at a vendor that
// stocks it, or wandering to find loot if no vendor in the zone sells it.
//
// Branches per tick:
//   - tag == "" && itemId == 0 → Failure (misconfigured goal)
//   - item already in backpack or equipment → Success
//   - shop in zone sells it + mob at shop → buy <desc>
//   - shop in zone sells it + mob not at shop → resolve sticky
//     target_shop_room MiscData + pathto
//   - no shop in zone sells it → wander (loot / forage chance)
func wealthItemPlanner(mob *mobs.Mob, goal *goals.Goal) PlanResult {
	if mob == nil {
		return PlanResult{Status: StatusFailure}
	}
	tag := goalParamStringOr(goal, "item_tag", "")
	itemId := goalParamIntOr(goal, "item_id", 0)
	if tag == "" && itemId == 0 {
		return PlanResult{Status: StatusFailure}
	}

	// Item already held → success.
	if mobHasItem(mob, tag, itemId) {
		return PlanResult{Status: StatusSuccess}
	}

	// Resolve (and cache) the vendor room that stocks the target item.
	shopRoom := mobMiscIntOr(mob, wealthItemShopRoomKey, 0)
	if shopRoom == 0 {
		room, ok := findShopInZoneSelling(mob, tag, itemId)
		if !ok {
			// No vendor stocks it → wander (loot/forage chance).
			return PlanResult{Command: "wander", Status: StatusRunning}
		}
		shopRoom = room
		mobSetMisc(mob, wealthItemShopRoomKey, shopRoom)
	}

	// Already at the vendor → issue buy command.
	if mob.Character.RoomId == shopRoom {
		desc := tag
		if desc == "" {
			desc = strconv.Itoa(itemId)
		}
		return PlanResult{Command: "buy " + desc, Status: StatusRunning}
	}

	// Not at vendor yet → pathfind there.
	return PlanResult{Command: "pathto " + strconv.Itoa(shopRoom), Status: StatusRunning}
}

// mobHasItem checks the mob's backpack and equipped slots for an item
// matching tag (ComponentTag) or id (ItemId).
//
// Mirrors the catalog package's helper of the same name
// (internal/goals/catalog/wealth_item.go:64). Duplicated here to keep
// the planners package free of a goals/catalog import. If churn grows,
// extract both copies to a shared sub-package.
func mobHasItem(mob *mobs.Mob, tag string, itemId int) bool {
	if mob == nil {
		return false
	}
	// Backpack
	for _, it := range mob.Character.Items {
		if matchesPlannerItem(it.ItemId, it.GetSpec().ComponentTag, tag, itemId) {
			return true
		}
	}
	// Equipment slots — GetAllItems() is verified at
	// internal/characters/worn.go:143; returns []items.Item.
	for _, it := range mob.Character.Equipment.GetAllItems() {
		if matchesPlannerItem(it.ItemId, it.GetSpec().ComponentTag, tag, itemId) {
			return true
		}
	}
	return false
}

// matchesPlannerItem returns true if the item identified by (gotId,
// gotTag) satisfies the planner's target (wantTag, wantId). Either
// criterion alone is sufficient.
func matchesPlannerItem(gotId int, gotTag string, wantTag string, wantId int) bool {
	if wantId > 0 && gotId == wantId {
		return true
	}
	if wantTag != "" && gotTag == wantTag {
		return true
	}
	return false
}
