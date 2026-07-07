package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/parties"
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
