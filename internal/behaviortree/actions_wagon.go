package behaviortree

// actions_wagon.go — Stage 3.4 wagon-specific btree actions.
//
// distribute_cargo_to_hostiles fires on the wagon's mob_death event.
// Walks all mobs in the wagon's current room, identifies hostiles
// (those that HatesMob the wagon — bandits with hates: [caravan]),
// and distributes wagon items round-robin into their inventories
// until either the wagon is empty or all hostiles are at carry
// capacity.
//
// Items that don't fit drop as standard wagon-corpse loot via the
// engine's normal mob-death corpse-creation path — players can still
// recover scraps from the wreckage even if all bandits filled up.
//
// Returns Failure if there are no hostiles in the room (lets the
// standard corpse-drop run unmodified).

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func init() {
	actionRegistry["distribute_cargo_to_hostiles"] = actDistributeCargoToHostiles
}

func actDistributeCargoToHostiles(params map[string]any, ctx *EvalContext) Result {
	wagon := mobs.GetInstance(ctx.InstanceId)
	if wagon == nil {
		return Failure
	}
	if len(wagon.Character.Items) == 0 {
		return Success // nothing to distribute, but not a failure
	}
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}

	// Find hostile mobs in the room — those that HatesMob the wagon.
	// (Bandits at North Road 4052 have hates: [caravan]; the wagon's
	// Groups include "caravan" and "merchant_train".)
	var hostiles []*mobs.Mob
	for _, instId := range room.GetMobs(rooms.FindAll) {
		if instId == wagon.InstanceId {
			continue
		}
		m := mobs.GetInstance(instId)
		if m == nil {
			continue
		}
		if m.HatesMob(wagon) {
			hostiles = append(hostiles, m)
		}
	}
	if len(hostiles) == 0 {
		// No hostiles — let standard corpse-drop run unmodified.
		return Failure
	}

	// Round-robin distribution. Walk wagon items in reverse so
	// RemoveItem is index-safe.
	h := 0
	for i := len(wagon.Character.Items) - 1; i >= 0; i-- {
		item := wagon.Character.Items[i]
		placed := false
		// Try every hostile before giving up on this item.
		for tries := 0; tries < len(hostiles); tries++ {
			target := hostiles[h]
			h = (h + 1) % len(hostiles)
			if target.Character.StoreItem(item) {
				wagon.Character.RemoveItem(item)
				placed = true
				break
			}
		}
		if !placed {
			// All hostiles at cap — remaining items stay in
			// wagon.Items and drop as standard corpse loot.
			break
		}
	}

	return Success
}
