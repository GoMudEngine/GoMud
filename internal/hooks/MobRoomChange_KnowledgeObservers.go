package hooks

// MobRoomChange_KnowledgeObservers.go — Stage 1.4 / Task 14
//
// Fires knowledge.RecordRoutineObservers whenever a forager or caravan
// mob enters a room, so that every NPC bystander records a witnessed
// observation of the moving entity.

import (
	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// MobRoomChangeKnowledgeObservers is an events.RoomChange listener that
// records bystander knowledge whenever a forager or caravan mob enters a
// room. Only mob moves are handled (MobInstanceId != 0); player moves and
// events with no recognizable template are silently ignored.
func MobRoomChangeKnowledgeObservers(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok {
		return events.Continue
	}
	// Player move — not our concern.
	if evt.MobInstanceId == 0 {
		return events.Continue
	}
	mob := mobs.GetInstance(evt.MobInstanceId)
	if mob == nil {
		return events.Continue
	}
	templateId := int(mob.MobId)
	if !forager.IsForagerMob(templateId) && !caravan.IsCaravanMob(templateId) {
		return events.Continue
	}
	room := rooms.LoadRoom(evt.ToRoomId)
	if room == nil {
		return events.Continue
	}
	knowledge.RecordRoutineObservers(evt.MobInstanceId, templateId, room)
	return events.Continue
}
