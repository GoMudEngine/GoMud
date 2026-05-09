package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestMobRoomChange_KnowledgeObservers_ForagerTriggers(t *testing.T) {
	room := rooms.LoadRoom(1)
	if room == nil {
		t.Skip("fixture room missing")
	}

	foragerTemplateId := 50
	observerTemplateId := 99

	foragerMob := mobs.NewMobByIdFresh(mobs.MobId(foragerTemplateId), 0)
	observer := mobs.NewMobByIdFresh(mobs.MobId(observerTemplateId), 0)
	if foragerMob == nil || observer == nil {
		t.Skip("forager or observer template not registered in test fixtures")
	}
	foragerMob.InstanceId = 5001
	observer.InstanceId = 5002

	room.AddMob(observer.InstanceId)
	defer room.RemoveMob(observer.InstanceId)

	evt := events.RoomChange{
		MobInstanceId: foragerMob.InstanceId,
		FromRoomId:    0,
		ToRoomId:      room.RoomId,
	}
	MobRoomChangeKnowledgeObservers(evt)

	if knowledge.Get(observerTemplateId, knowledge.MobSubject(foragerTemplateId)) == nil {
		t.Errorf("observer should have a knowledge record of the forager")
	}
}

func TestMobRoomChange_KnowledgeObservers_NonForagerSilent(t *testing.T) {
	room := rooms.LoadRoom(1)
	if room == nil {
		t.Skip("fixture room missing")
	}

	regularTemplateId := 100
	observerTemplateId := 92
	regular := mobs.NewMobByIdFresh(mobs.MobId(regularTemplateId), 0)
	observer := mobs.NewMobByIdFresh(mobs.MobId(observerTemplateId), 0)
	if regular == nil || observer == nil {
		t.Skip("template not registered")
	}
	regular.InstanceId = 5101
	observer.InstanceId = 5102

	room.AddMob(observer.InstanceId)
	defer room.RemoveMob(observer.InstanceId)

	evt := events.RoomChange{
		MobInstanceId: regular.InstanceId,
		ToRoomId:      room.RoomId,
	}
	MobRoomChangeKnowledgeObservers(evt)

	if knowledge.Get(observerTemplateId, knowledge.MobSubject(regularTemplateId)) != nil {
		t.Errorf("non-forager mob should not trigger knowledge writes")
	}
}
