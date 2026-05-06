package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// MobDeathFactionRep bumps faction rep downward for every player
// who participated in killing a mob with at least one defined
// faction. Participation = direct damage (PlayerDamage map) OR
// being in the same room as the death AND in the killer's party
// (covers pure-support healers).
//
// Magnitude: Balance.FactionMemberKillRep (default -10) per
// affected (player, faction) pair. Saturates Hostile after ~10
// kills of the same faction.
func MobDeathFactionRep(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}

	if len(evt.PlayerDamage) == 0 {
		return events.Continue
	}

	mob := mobs.GetInstance(evt.InstanceId)
	if mob == nil {
		return events.Continue
	}

	factionIds := factions.FactionsForMob(mob)
	if len(factionIds) == 0 {
		return events.Continue
	}

	delta := int(configs.GetBalanceConfig().FactionMemberKillRep)

	// Build set of users to bump: direct damagers + party members in
	// the death room.
	toBump := make(map[int]struct{}, len(evt.PlayerDamage))
	for userId := range evt.PlayerDamage {
		toBump[userId] = struct{}{}
	}
	for damagerUserId := range evt.PlayerDamage {
		party := parties.Get(damagerUserId)
		if party == nil {
			continue
		}
		for _, memberId := range party.UserIds {
			if _, already := toBump[memberId]; already {
				continue
			}
			u := users.GetByUserId(memberId)
			if u == nil {
				continue
			}
			if u.Character.RoomId == evt.RoomId {
				toBump[memberId] = struct{}{}
			}
		}
	}

	for userId := range toBump {
		for _, fid := range factionIds {
			factions.BumpRep(fid, userId, delta)
		}
	}

	return events.Continue
}
