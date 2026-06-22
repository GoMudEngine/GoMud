package caravan

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func TestFindWagonInRoom_ReturnsWagon(t *testing.T) {
	wagon := &mobs.Mob{
		MobId:      mobs.MobId(WagonMobId),
		InstanceId: 91374,
		HomeRoomId: 9999,
		Zone:       "TestZone",
	}
	wagon.Character.Name = "TestWagon"
	wagon.Character.Buffs = buffs.New()
	wagon.Character.RoomId = 9999

	r := &rooms.Room{
		RoomId: 9999,
		Zone:   "TestZone",
		Title:  "Test Room",
		Exits:  map[string]exit.RoomExit{},
	}
	cleanRoom := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{9999: r},
		map[string]*rooms.ZoneConfig{},
	)
	defer cleanRoom()

	mobs.SetInstanceForTest(wagon.InstanceId, wagon)
	defer mobs.SetInstanceForTest(wagon.InstanceId, nil)
	r.AddMob(wagon.InstanceId)

	got := FindWagonInRoom(9999)
	if got == nil {
		t.Fatalf("FindWagonInRoom(9999): got nil, want wagon mob %d", WagonMobId)
	}
	if int(got.MobId) != WagonMobId {
		t.Errorf("FindWagonInRoom(9999): got mobId %d, want %d", got.MobId, WagonMobId)
	}
}

func TestFindWagonInRoom_AbsentReturnsNil(t *testing.T) {
	// Room exists but has no mobs.
	r := &rooms.Room{
		RoomId: 9998,
		Zone:   "TestZone",
		Title:  "Empty Room",
		Exits:  map[string]exit.RoomExit{},
	}
	cleanRoom := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{9998: r},
		map[string]*rooms.ZoneConfig{},
	)
	defer cleanRoom()

	if got := FindWagonInRoom(9998); got != nil {
		t.Errorf("FindWagonInRoom(9998) on empty room: got %v, want nil", got)
	}
}

func TestFindWagonInRoom_UnknownRoomReturnsNil(t *testing.T) {
	if got := FindWagonInRoom(424242); got != nil {
		t.Errorf("FindWagonInRoom(unknown room): got %v, want nil", got)
	}
}

func TestFindWagonInRoom_NonWagonMobReturnsNil(t *testing.T) {
	// A non-wagon mob (different MobId) should not match.
	other := &mobs.Mob{
		MobId:      mobs.MobId(1),
		InstanceId: 91001,
		HomeRoomId: 9997,
		Zone:       "TestZone",
	}
	other.Character.Name = "NotAWagon"
	other.Character.Buffs = buffs.New()
	other.Character.RoomId = 9997

	r := &rooms.Room{
		RoomId: 9997,
		Zone:   "TestZone",
		Title:  "Test Room",
		Exits:  map[string]exit.RoomExit{},
	}
	cleanRoom := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{9997: r},
		map[string]*rooms.ZoneConfig{},
	)
	defer cleanRoom()

	mobs.SetInstanceForTest(other.InstanceId, other)
	defer mobs.SetInstanceForTest(other.InstanceId, nil)
	r.AddMob(other.InstanceId)

	if got := FindWagonInRoom(9997); got != nil {
		t.Errorf("FindWagonInRoom(9997) with non-wagon mob: got mobId %d, want nil", got.MobId)
	}
}

func TestFindMobByTemplateInRoom_FindsByTemplate(t *testing.T) {
	// An NP import runner (Dobb, template 9304) co-located in a room.
	dobb := &mobs.Mob{
		MobId:      mobs.MobId(9304),
		InstanceId: 99304,
		HomeRoomId: 9990,
		Zone:       "TestZone",
	}
	dobb.Character.Name = "Dobb"
	dobb.Character.Buffs = buffs.New()
	dobb.Character.RoomId = 9990

	r := &rooms.Room{
		RoomId: 9990,
		Zone:   "TestZone",
		Title:  "Test Depot",
		Exits:  map[string]exit.RoomExit{},
	}
	cleanRoom := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{9990: r},
		map[string]*rooms.ZoneConfig{},
	)
	defer cleanRoom()

	mobs.SetInstanceForTest(dobb.InstanceId, dobb)
	defer mobs.SetInstanceForTest(dobb.InstanceId, nil)
	r.AddMob(dobb.InstanceId)

	got := FindMobByTemplateInRoom(9990, 9304)
	if got == nil {
		t.Fatal("FindMobByTemplateInRoom(9990, 9304): got nil, want Dobb")
	}
	if int(got.MobId) != 9304 {
		t.Errorf("got mobId %d, want 9304", got.MobId)
	}
	// A template id NOT present must return nil.
	if other := FindMobByTemplateInRoom(9990, 12345); other != nil {
		t.Errorf("FindMobByTemplateInRoom(9990, 12345): got %v, want nil", other)
	}
}

func TestFindMobByTemplateInRoom_NilRoomReturnsNil(t *testing.T) {
	if m := FindMobByTemplateInRoom(424243, 9304); m != nil {
		t.Errorf("expected nil for missing room, got %v", m)
	}
}
