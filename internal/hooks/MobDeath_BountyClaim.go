package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// MobDeathBountyClaim auto-claims open mob-target bounties when the
// mob dies. The killer is the highest entry in evt.PlayerDamage
// (companion damage already rolls up to owners via combat.go's
// GetCharmedUserId path). Faction-issued bounties also bump rep.
func MobDeathBountyClaim(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}
	if len(evt.PlayerDamage) == 0 {
		return events.Continue
	}

	// Use the mob TEMPLATE — instance is destroyed by the time this
	// queued event fires (suicide.go calls DestroyInstance after
	// queuing MobDeath).
	spec := mobs.GetMobSpec(mobs.MobId(evt.MobId))
	if spec == nil {
		return events.Continue
	}
	target := knowledge.MobSubject(int(spec.MobId))

	open := bounties.OpenForTarget(target)
	if len(open) == 0 {
		return events.Continue
	}

	// Highest-damager wins.
	killerUserId, topDamage := 0, 0
	for userId, dmg := range evt.PlayerDamage {
		if dmg > topDamage {
			killerUserId, topDamage = userId, dmg
		}
	}
	if killerUserId == 0 {
		return events.Continue
	}
	killer := knowledge.PlayerSubject(killerUserId)
	user := users.GetByUserId(killerUserId)
	if user == nil {
		return events.Continue
	}

	for _, b := range open {
		claimed, ok := bounties.TryClaim(b.Id, killer)
		if !ok {
			continue
		}
		// Pay gold.
		user.Character.Gold += claimed.GoldReward

		// Faction rep bump (only when issuer is a faction).
		if claimed.Issuer.Type == bounties.IssuerFaction {
			factions.BumpRep(claimed.Issuer.Id, killerUserId, claimed.RepReward)
		}

		user.SendText(messaging.CategoryLoot, fmt.Sprintf(
			"You collect a bounty: %dg.\r\n",
			claimed.GoldReward,
		))
	}

	return events.Continue
}
