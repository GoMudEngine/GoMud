package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// PackFlee triggers remaining pack members to flee when one of their species dies.
func PackFlee(e events.Event) events.ListenerReturn {

	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}

	// Stage 1 NPC party system: if the dead mob was a party leader,
	// dissolve the party (which fires PartyDissolved event for member
	// btrees to react). This supersedes the species-based pack flee
	// for that case — the dissolution handler is responsible for any
	// post-leader-death behavior.
	if p := parties.GetByMobInstanceId(evt.InstanceId); p != nil {
		// If the dead mob raised the active help call, clear it so any
		// in-flight responders stop trekking to the now-empty rally room.
		// This runs BEFORE leader-dissolution so it fires whether the
		// caller was the leader or a regular member.
		if p.HelpCallerInstanceId == evt.InstanceId {
			p.HelpRoomId = 0
			p.HelpCallerInstanceId = 0
		}
		if p.Leader != nil && p.Leader.GetMobInstanceId() == evt.InstanceId {
			p.Dissolve("leader_died")
			return events.Continue
		}
		// Non-leader member died — remove them from party state but
		// let the existing species-based pack-flee logic still fire.
		// The party continues with remaining members; the leader's
		// btree handles any tactical retreat decisions.
		p.RemoveActorByMobInstanceId(evt.InstanceId)
		// fall through to species-based pack-flee logic
	}

	// Load the dead mob's spec to get its species
	deadSpec := mobs.GetMobSpec(mobs.MobId(evt.MobId))
	if deadSpec == nil || deadSpec.Character.SpeciesId == 0 {
		return events.Continue
	}

	room := rooms.LoadRoom(evt.RoomId)
	if room == nil {
		return events.Continue
	}

	// Check all living mobs in the room
	mobIds := room.GetMobs(rooms.FindAll)
	fleeCount := 0
	for _, mobInstId := range mobIds {
		mob := mobs.GetInstance(mobInstId)
		if mob == nil {
			continue
		}
		// Skip the dead mob's instance (should already be gone, but be safe)
		if mob.InstanceId == evt.InstanceId {
			continue
		}

		// Charmed/companion mobs don't flee with wild packs
		if mob.Character.IsCharmed() {
			continue
		}

		// Non-combatant mobs (merchants, quest NPCs) never scatter
		if mob.IsNonCombatant() {
			continue
		}

		// Mobs with pack_flee_immune hold their ground
		if mob.PackFleeImmune {
			continue
		}

		// Check alliance: same MobId or same species
		if mob.MobId != mobs.MobId(evt.MobId) {
			if mob.Character.SpeciesId == 0 ||
				deadSpec.Character.SpeciesId != mob.Character.SpeciesId {
				continue
			}
		}

		// Queue flee command on this mob
		mob.Command("flee")
		fleeCount++
	}

	if fleeCount > 0 {
		speciesName := deadSpec.Character.Species()
		if speciesName == "" {
			speciesName = "creatures"
		}
		sendVisualRoomText(room, messaging.CategoryMobEmote,
			fmt.Sprintf(`<ansi fg="yellow">Sensing the death of their packmate, the remaining %s scatter!</ansi>`, speciesName),
		)
	}

	// Stage 42.8: If dead mob was pack alpha or member, scatter the pack
	if mobs.PackRoamingEnabled() {
		mobs.HandleAlphaDeath(deadSpec, mobIds)
	}

	return events.Continue
}
