package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/itemvalue"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// EquipBestFloorItem scans the floor items in mob's room, scores
// each via itemvalue.ItemValueDelta, and equips the best
// positive-scoring upgrade if any. Returns true if a swap occurred.
//
// No-op (returns false) if any of:
//   - !itemvalue.CanScanFloorLoot(&mob.Character, mob.BehaviorArchetype)
//   - mob is in combat (Character.IsInCombat())
//   - room is nil or has no floor items
//   - no floor item scores as an upgrade for this mob+profile
//
// On successful pickup, emits a room broadcast distinct from
// give-equip phrasing ("picks up X and dons/wields it"). Displaced
// items go to mob's backpack per actions.EquipItem default (charmed
// mobs don't reach this path).
func EquipBestFloorItem(mob *mobs.Mob, room *rooms.Room) bool {
	if !itemvalue.CanScanFloorLoot(&mob.Character, mob.BehaviorArchetype) {
		return false
	}
	if mob.Character.IsInCombat() {
		return false // busy fighting
	}
	if room == nil || len(room.Items) == 0 {
		return false // nothing on floor
	}

	profile := itemvalue.ProfileFor(mob.Archetype, mob.BehaviorArchetype)

	// Find the floor item with the highest positive delta score.
	var bestItem items.Item
	bestScore := 0.0
	for _, floorItem := range room.Items {
		delta := itemvalue.ItemValueDelta(&mob.Character, profile, floorItem)
		if delta.Score > bestScore {
			bestScore = delta.Score
			bestItem = floorItem
		}
	}
	if bestItem.ItemId == 0 {
		return false // nothing was an upgrade
	}

	// Remove from floor and into backpack so EquipItem can find it.
	room.RemoveItem(bestItem, false)
	mob.Character.StoreItem(bestItem)

	actor := &actions.MobActor{Mob: mob, Room: room}
	result := actions.EquipItem(actor, bestItem.Name())
	if !result.Equipped {
		// Edge case: ItemValueDelta thought slot was compatible
		// but EquipItem refused (rare). Item is still in backpack;
		// mob effectively "picked it up" without equipping.
		return false
	}

	// Room broadcast: distinct from give-equip phrasing to signal
	// the loot-pickup origin.
	spec := result.Item.GetSpec()
	if spec.Subtype == items.Wearable {
		room.SendTextVisual(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> picks up <ansi fg="item">%s</ansi> and dons it.`,
			mob.Character.Name, result.Item.DisplayName()))
	} else {
		room.SendTextVisual(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> picks up <ansi fg="item">%s</ansi> and wields it.`,
			mob.Character.Name, result.Item.DisplayName()))
	}

	return true
}
