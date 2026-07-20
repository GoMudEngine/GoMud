package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// TestMobRoomChange_KnowledgeObservers_ForagerTriggers verifies that when a
// registered forager mob (template 371 = Tova) enters a room, every other NPC
// in the room gets a knowledge record written for that forager.
func TestMobRoomChange_KnowledgeObservers_ForagerTriggers(t *testing.T) {
	// Sanity: confirm the template IDs we rely on are actually registered.
	if !forager.IsForagerMob(371) {
		t.Fatal("test setup error: template 371 is not a registered forager")
	}
	if forager.IsForagerMob(101) {
		t.Fatal("test setup error: template 101 should NOT be a registered forager")
	}

	const roomId = 5500
	room, cleanupRoom := seedHookRoom(t, roomId)
	defer cleanupRoom()

	foragerTemplateId := 371  // Tova (Marsh Forager)
	observerTemplateId := 101 // non-forager citizen used as observer

	foragerSpec := &mobs.Mob{
		MobId: mobs.MobId(foragerTemplateId),
		Character: characters.Character{
			Name:  "Tova",
			Buffs: buffs.New(),
		},
	}
	foragerInst := &mobs.Mob{
		MobId:      mobs.MobId(foragerTemplateId),
		InstanceId: 5001,
		Character: characters.Character{
			Name:   "Tova",
			RoomId: roomId,
			Buffs:  buffs.New(),
		},
	}
	observerSpec := &mobs.Mob{
		MobId: mobs.MobId(observerTemplateId),
		Character: characters.Character{
			Name:  "citizen",
			Buffs: buffs.New(),
		},
	}
	observerInst := &mobs.Mob{
		MobId:      mobs.MobId(observerTemplateId),
		InstanceId: 5002,
		Character: characters.Character{
			Name:   "citizen",
			RoomId: roomId,
			Buffs:  buffs.New(),
		},
	}

	cleanupMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{foragerTemplateId: foragerSpec, observerTemplateId: observerSpec},
		map[int]*mobs.Mob{5001: foragerInst, 5002: observerInst},
	)
	defer cleanupMobs()

	// Observer is present in the room; forager is arriving.
	room.AddMob(observerInst.InstanceId)
	defer room.RemoveMob(observerInst.InstanceId)

	evt := events.RoomChange{
		MobInstanceId: foragerInst.InstanceId,
		FromRoomId:    0,
		ToRoomId:      room.RoomId,
	}
	MobRoomChangeKnowledgeObservers(evt)

	rec := knowledge.Get(observerTemplateId, knowledge.MobSubject(foragerTemplateId))
	if rec == nil {
		t.Errorf("observer (template %d) should have a knowledge record of forager (template %d)",
			observerTemplateId, foragerTemplateId)
	}
}

// TestMobRoomChange_KnowledgeObservers_NonForagerSilent verifies that when a
// non-forager, non-caravan mob enters a room, no knowledge records are written.
func TestMobRoomChange_KnowledgeObservers_NonForagerSilent(t *testing.T) {
	// Confirm neither template is a forager (belt-and-suspenders for test validity).
	if forager.IsForagerMob(102) {
		t.Fatal("test setup error: template 102 should NOT be a registered forager")
	}
	if forager.IsForagerMob(103) {
		t.Fatal("test setup error: template 103 should NOT be a registered forager")
	}

	const roomId = 5501
	room, cleanupRoom := seedHookRoom(t, roomId)
	defer cleanupRoom()

	regularTemplateId := 102  // non-forager mover
	observerTemplateId := 103 // non-forager bystander

	regularSpec := &mobs.Mob{
		MobId: mobs.MobId(regularTemplateId),
		Character: characters.Character{
			Name:  "wanderer",
			Buffs: buffs.New(),
		},
	}
	regularInst := &mobs.Mob{
		MobId:      mobs.MobId(regularTemplateId),
		InstanceId: 5101,
		Character: characters.Character{
			Name:   "wanderer",
			RoomId: roomId,
			Buffs:  buffs.New(),
		},
	}
	observerSpec := &mobs.Mob{
		MobId: mobs.MobId(observerTemplateId),
		Character: characters.Character{
			Name:  "bystander",
			Buffs: buffs.New(),
		},
	}
	observerInst := &mobs.Mob{
		MobId:      mobs.MobId(observerTemplateId),
		InstanceId: 5102,
		Character: characters.Character{
			Name:   "bystander",
			RoomId: roomId,
			Buffs:  buffs.New(),
		},
	}

	cleanupMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{regularTemplateId: regularSpec, observerTemplateId: observerSpec},
		map[int]*mobs.Mob{5101: regularInst, 5102: observerInst},
	)
	defer cleanupMobs()

	room.AddMob(observerInst.InstanceId)
	defer room.RemoveMob(observerInst.InstanceId)

	evt := events.RoomChange{
		MobInstanceId: regularInst.InstanceId,
		ToRoomId:      room.RoomId,
	}
	MobRoomChangeKnowledgeObservers(evt)

	if knowledge.Get(observerTemplateId, knowledge.MobSubject(regularTemplateId)) != nil {
		t.Errorf("non-forager mob should not trigger knowledge writes for bystanders")
	}
}
