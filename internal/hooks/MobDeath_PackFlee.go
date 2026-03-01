package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// PackFlee triggers remaining pack members to flee when one of their group dies.
func PackFlee(e events.Event) events.ListenerReturn {

	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}

	// Load the dead mob's spec to get its groups
	deadSpec := mobs.GetMobSpec(mobs.MobId(evt.MobId))
	if deadSpec == nil || len(deadSpec.Groups) == 0 {
		return events.Continue
	}

	room := rooms.LoadRoom(evt.RoomId)
	if room == nil {
		return events.Continue
	}

	// Build a set of the dead mob's groups for fast lookup
	deadGroups := make(map[string]bool, len(deadSpec.Groups))
	for _, g := range deadSpec.Groups {
		deadGroups[g] = true
	}

	// Find the first overlapping group name for the scatter message
	scatterGroup := deadSpec.Groups[0]

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

		// Check if any groups overlap
		hasOverlap := false
		for _, g := range mob.Groups {
			if deadGroups[g] {
				hasOverlap = true
				break
			}
		}
		if !hasOverlap {
			continue
		}

		// Queue flee command on this mob
		mob.Command("flee")
		fleeCount++
	}

	if fleeCount > 0 {
		room.SendText(
			fmt.Sprintf(`<ansi fg="yellow">Sensing the death of their packmate, the remaining %s scatter!</ansi>`, scatterGroup),
		)
	}

	// Stage 42.8: If dead mob was pack alpha or member, scatter the pack
	if mobs.PackRoamingEnabled() {
		mobs.HandleAlphaDeath(deadSpec, mobIds)
	}

	return events.Continue
}
