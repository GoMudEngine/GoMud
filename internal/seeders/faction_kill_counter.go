package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const ruleNameFactionKillCounter = "faction_kill_counter"

func init() {
	Register(ruleNameFactionKillCounter, factionKillCounter, "MobDeath")
}

// factionKillCounter: on MobDeath, if the killer is a mob, bump
// faction_kills_inflicted:<factionId> on the killer for each faction
// the victim belongs to. Read by the 4.3 catalog's revenge-faction
// Predicate (internal/goals/catalog/revenge_faction.go).
//
// Player-as-killer events (KillerMobInstanceId == 0) are skipped —
// mob MiscData counters are only meaningful as mob-vs-mob attribution.
// Future rules wanting player-kill faction attribution can walk
// events.MobDeath.PlayerDamage instead.
func factionKillCounter(event events.Event) {
	md, ok := event.(events.MobDeath)
	if !ok {
		return
	}
	killerKind, killerId := resolveKillerFromMobDeath(event)
	if killerKind != "mob" || killerId == 0 {
		return // player-as-killer does not write to mob MiscData counters
	}

	// Look up victim — may already be removed from instance map at the
	// time this rule fires (death cleanup races). Nil-guard is safe:
	// if victim is gone we simply can't read its faction list.
	victim := mobs.GetInstance(md.InstanceId)
	if victim == nil {
		return
	}
	victimFactions := factions.FactionsForMob(victim)
	if len(victimFactions) == 0 {
		return
	}

	killer := mobs.GetInstance(killerId)
	if killer == nil {
		return // killer already despawned
	}

	for _, fid := range victimFactions {
		bumpMiscInt(killer, "faction_kills_inflicted:"+fid, 1)
	}
}
