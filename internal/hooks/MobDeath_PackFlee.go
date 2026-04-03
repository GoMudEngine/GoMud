package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// PackFlee triggers remaining pack members to flee when one of their species dies.
func PackFlee(e events.Event) events.ListenerReturn {

	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}

	// Quest engine: notify all players who damaged this mob
	for userId := range evt.PlayerDamage {
		if u := users.GetByUserId(userId); u != nil {
			bridge := questengine.NewGameBridge(u, evt.RoomId)
			questengine.GetEngine().Notify("mob_death", questengine.EventDetails{
				UserId: userId,
				RoomId: evt.RoomId,
				MobId:  evt.MobId,
			}, bridge, bridge)
		}
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
		room.SendText(
			fmt.Sprintf(`<ansi fg="yellow">Sensing the death of their packmate, the remaining %s scatter!</ansi>`, speciesName),
		)
	}

	// Stage 42.8: If dead mob was pack alpha or member, scatter the pack
	if mobs.PackRoamingEnabled() {
		mobs.HandleAlphaDeath(deadSpec, mobIds)
	}

	return events.Continue
}
