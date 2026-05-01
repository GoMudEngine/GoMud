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
