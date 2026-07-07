package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// computeCorpseOwners returns userIds allowed to loot before the free-for-all
// timeout: the damagers plus their same-room party members. Empty when no
// player damaged it (mob/environment kill -> anyone loots).
//
// Mirrors buildKillerSet in MobDeath_FactionRep.go: seed the set from the
// direct damagers, then expand each damager's party with same-room members
// (so pure-support healers who dealt no damage can still loot).
func computeCorpseOwners(playerDamage map[int]int, roomId int) []int {
	if len(playerDamage) == 0 {
		return nil
	}

	set := make(map[int]struct{}, len(playerDamage))
	for userId := range playerDamage {
		set[userId] = struct{}{}
	}

	for damagerUserId := range playerDamage {
		party := parties.Get(damagerUserId)
		if party == nil {
			continue
		}
		for _, memberId := range party.UserIds {
			if _, already := set[memberId]; already {
				continue
			}
			u := users.GetByUserId(memberId)
			if u == nil {
				continue
			}
			if u.Character.RoomId == roomId {
				set[memberId] = struct{}{}
			}
		}
	}

	out := make([]int, 0, len(set))
	for userId := range set {
		out = append(out, userId)
	}
	return out
}

// corpseLootMode returns the killer's party loot mode, or "ffa" if the killer
// is solo / not in a party (or no player damaged the mob).
func corpseLootMode(playerDamage map[int]int) string {
	for userId := range playerDamage {
		party := parties.Get(userId)
		if party == nil {
			continue
		}
		if party.LootMode == "" {
			return "ffa"
		}
		return party.LootMode
	}
	return "ffa"
}

// lootTimeoutRound returns the round at which corpse ownership opens to
// free-for-all: now advanced by the configured real-time duration.
func lootTimeoutRound(now uint64, dur string) uint64 {
	return gametime.GetDate(now).AddPeriod(dur)
}

// killerParty returns the party a damager belongs to (first found), or nil when
// the killer is solo. Mirrors the first-damager selection corpseLootMode uses.
func killerParty(playerDamage map[int]int) *parties.Party {
	for userId := range playerDamage {
		if party := parties.Get(userId); party != nil {
			return party
		}
	}
	return nil
}

// assignCorpseLoot maps each loot item's stable UID to the userId it is reserved
// for, per the killer party's loot mode. Returns nil (no per-item reservations
// = free-for-all among owners) for ffa, solo kills, or when there are no members
// to deal to. Advances the party's round-robin cursor as a side effect.
//
//   - "leaderhold": every item -> the party leader.
//   - "roundrobin": items dealt across the same-room owner-members (join order),
//     starting at party.RRCursor, which is then advanced.
//   - "ffa"/"": nil.
func assignCorpseLoot(loot rooms.Container, owners []int, playerDamage map[int]int) map[string]int {
	party := killerParty(playerDamage)
	if party == nil {
		return nil
	}

	mode := party.LootMode
	if mode == "" {
		mode = "ffa"
	}

	switch mode {

	case "leaderhold":
		if len(loot.Items) == 0 {
			return nil
		}
		assignments := make(map[string]int, len(loot.Items))
		for _, it := range loot.Items {
			assignments[it.UUID.String()] = party.LeaderUserId
		}
		return assignments

	case "roundrobin":
		// members = party join order, filtered to the corpse's owner set (the
		// members actually present in the room, per computeCorpseOwners).
		ownerSet := make(map[int]struct{}, len(owners))
		for _, id := range owners {
			ownerSet[id] = struct{}{}
		}
		members := make([]int, 0, len(party.UserIds))
		for _, id := range party.UserIds {
			if _, ok := ownerSet[id]; ok {
				members = append(members, id)
			}
		}
		if len(members) == 0 || len(loot.Items) == 0 {
			return nil
		}

		assignees, newCursor := rooms.RoundRobinOrder(len(loot.Items), members, party.RRCursor)
		party.RRCursor = newCursor

		assignments := make(map[string]int, len(loot.Items))
		for i, it := range loot.Items {
			assignments[it.UUID.String()] = assignees[i]
		}
		return assignments

	default: // ffa
		return nil
	}
}
