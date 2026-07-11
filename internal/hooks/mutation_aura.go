package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// auraRecipients returns the player ids in a room that an aura owner projects
// onto: everyone present except the owner.
func auraRecipients(playerIds []int, ownerId int) []int {
	out := make([]int, 0, len(playerIds))
	for _, id := range playerIds {
		if id != ownerId {
			out = append(out, id)
		}
	}
	return out
}

// applyRoomAllyAuras applies each in-combat ally-aura owner's buff to the other
// players in the room. Buffs go through the user AddBuff wrapper (start text +
// GMCP; silent on refresh), and are short-lived so they lapse when the aura
// owner leaves or the fight ends.
func applyRoomAllyAuras(room *rooms.Room) {
	playerIds := room.GetPlayers()
	if len(playerIds) < 2 {
		return // an aura needs an owner and at least one ally
	}
	for _, ownerId := range playerIds {
		owner := users.GetByUserId(ownerId)
		if owner == nil || !owner.Character.IsInCombat() {
			continue
		}
		buffIds := mutations.GetAllyAuraBuffs(owner.Character.Mutations)
		if len(buffIds) == 0 {
			continue
		}
		// Project only onto the owner's PARTY — never strangers or PvP foes.
		// A solo owner (no party) has no allies to embolden.
		party := parties.Get(ownerId)
		if party == nil {
			continue
		}
		for _, rid := range auraRecipients(playerIds, ownerId) {
			if !party.IsMember(rid) {
				continue
			}
			ally := users.GetByUserId(rid)
			if ally == nil {
				continue
			}
			for _, buffId := range buffIds {
				ally.AddBuff(buffId, "aura")
			}
		}
	}
}
