package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

const ruleNameWitnessOfTheftToRevenge = "witness_of_theft_to_revenge"
const theftVictimRevengePriority = 90
const theftWitnessRevengePriority = 60

// OnTheft is the public entry point called from the steal-action handler
// when a player successfully steals from a mob. Seeds a revenge-mob goal
// into the victim (priority 90) and into every other mob sharing the same
// room (priority 60, since witnesses care less than the direct victim).
//
// No event subscription is used here because events.ItemOwnership lacks a
// theft subtype to filter on — this is the architectural-exception pattern
// (same as rule 3 / craft_materials_to_wealth_item): the caller in
// actions/steal.go invokes OnTheft directly at the success point.
//
// Skips silently when thiefUserId == 0 or victimMob == nil.
func OnTheft(thiefUserId int, victimMob *mobs.Mob, item items.Item) {
	if thiefUserId == 0 || victimMob == nil {
		return
	}

	// Seed revenge into the direct victim at high priority.
	seedRevengeGoalIfAbsent(victimMob, "player", thiefUserId, theftVictimRevengePriority)

	// Seed revenge into every other mob sharing the victim's room at
	// a lower witness priority.
	room := rooms.LoadRoom(victimMob.Character.RoomId)
	if room == nil {
		return
	}
	for _, witnessInstId := range room.GetMobs() {
		if witnessInstId == victimMob.InstanceId {
			continue
		}
		witness := mobs.GetInstance(witnessInstId)
		if witness == nil {
			continue
		}
		seedRevengeGoalIfAbsent(witness, "player", thiefUserId, theftWitnessRevengePriority)
	}
}
