package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// topDamager returns the userId that dealt the most damage in playerDamage,
// tie-broken by the LOWER userId for determinism (maps range in random order,
// so a plain "first" pick is nondeterministic). Returns 0 for an empty map.
// Mirrors the highest-damager selection in MobDeath_BountyClaim.go.
func topDamager(playerDamage map[int]int) int {
	topUser, topDamage := 0, -1
	for userId, dmg := range playerDamage {
		if dmg > topDamage || (dmg == topDamage && userId < topUser) {
			topUser, topDamage = userId, dmg
		}
	}
	return topUser
}

// payoutPartyGold settles the shared party gold pool and credits each ONLINE
// member their share, emitting a gold-sync event so prompts/GMCP refresh.
// Shares for offline/unloaded members (and an offline leader's remainder) are
// re-pooled rather than destroyed, so gold is conserved. Used on player logout
// (PlayerDespawn) where there is no interactive command context.
func payoutPartyGold(p *parties.Party) {
	payouts := p.SettleGold() // pure: pool now 0
	if len(payouts) == 0 {
		return
	}
	var reserved int
	for uid, amt := range payouts {
		if amt <= 0 {
			continue
		}
		u := users.GetByUserId(uid)
		if u == nil {
			reserved += amt // keep absent members' shares in the pool
			continue
		}
		u.Character.Gold += amt
		events.AddToQueue(events.EquipmentChange{
			UserId:     uid,
			GoldChange: -amt,
		})
	}
	p.GoldPool += reserved // conserve: undistributed shares stay pooled
}

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
	// Use the highest-damager's party deterministically (maps range randomly),
	// so the stamped mode matches the party item assignment uses.
	top := topDamager(playerDamage)
	if top == 0 {
		return "ffa"
	}
	party := parties.Get(top)
	if party == nil || party.LootMode == "" {
		return "ffa"
	}
	return party.LootMode
}

// lootTimeoutRound returns the round at which corpse ownership opens to
// free-for-all: now advanced by the configured real-time duration.
func lootTimeoutRound(now uint64, dur string) uint64 {
	return gametime.GetDate(now).AddPeriod(dur)
}

// killerParty returns the highest-damager's party, or nil when the killer is
// solo. Uses the same deterministic top-damager selection as corpseLootMode so
// the stamped loot mode and per-item assignment agree on one party.
func killerParty(playerDamage map[int]int) *parties.Party {
	top := topDamager(playerDamage)
	if top == 0 {
		return nil
	}
	return parties.Get(top)
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
