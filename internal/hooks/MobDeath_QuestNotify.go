package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// MobDeathQuestNotify fires quest engine mob_death notifications for all
// players who damaged the dying mob. This is separate from PackFlee so
// quest triggers work regardless of pack/species behavior.
func MobDeathQuestNotify(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.MobDeath)
	if !ok {
		return events.Continue
	}

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

	return events.Continue
}
