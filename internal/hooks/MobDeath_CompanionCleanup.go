package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// CompanionCleanup removes a dead companion from the owner's Companions list
// and notifies the player when their companion falls in battle.
func CompanionCleanup(e events.Event) events.ListenerReturn {

	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}

	// Check if the dead mob's instance was a companion before it was destroyed.
	// We rely on the CharmedMobs tracking on the player character to find the owner.
	allUsers := users.GetAllActiveUsers()
	for _, user := range allUsers {
		comp := user.Character.GetCompanionByInstanceId(evt.InstanceId)
		if comp == nil {
			continue
		}

		// We found the owner. Remove the companion from the list.
		companionName := comp.Name
		if companionName == "" {
			if spec := mobs.GetMobSpec(mobs.MobId(comp.MobId)); spec != nil {
				companionName = spec.Character.Name
			} else {
				companionName = "companion"
			}
		}

		user.Character.RemoveCompanion(evt.InstanceId)
		user.Character.TrackCharmed(evt.InstanceId, false)

		user.SendTextLegacy(fmt.Sprintf(
			`<ansi fg="red">Your %s has fallen.</ansi>`,
			companionName,
		))

		break
	}

	return events.Continue
}
