package behaviortree

import (
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}

// seedTestMob seeds a single mob spec at templateId and a single instance
// at instanceId placed in homeRoomId. Returns a cleanup function.
func seedTestMob(t *testing.T, templateId int, instanceId int, homeRoomId int, name string) func() {
	t.Helper()
	spec := &mobs.Mob{
		MobId: mobs.MobId(templateId),
		Character: characters.Character{
			Name:   name,
			RoomId: homeRoomId,
			Buffs:  buffs.New(),
		},
	}
	instance := &mobs.Mob{
		MobId:      mobs.MobId(templateId),
		InstanceId: instanceId,
		HomeRoomId: homeRoomId,
		Character: characters.Character{
			Name:   name,
			RoomId: homeRoomId,
			Buffs:  buffs.New(),
		},
	}
	return mobs.SeedMobsForTest(
		map[int]*mobs.Mob{templateId: spec},
		map[int]*mobs.Mob{instanceId: instance},
	)
}

// seedTwoMobs registers two mob templates + two instances in one
// SeedMobsForTest call, avoiding the single-mob limitation of
// seedTestMob. Returns a single cleanup function.
func seedTwoMobs(t *testing.T, roomId int,
	template1, instance1 int, name1 string,
	template2, instance2 int, name2 string,
) func() {
	t.Helper()
	specs := map[int]*mobs.Mob{
		template1: {MobId: mobs.MobId(template1), Character: characters.Character{
			Name: name1, RoomId: roomId, Buffs: buffs.New(),
		}},
		template2: {MobId: mobs.MobId(template2), Character: characters.Character{
			Name: name2, RoomId: roomId, Buffs: buffs.New(),
		}},
	}
	instances := map[int]*mobs.Mob{
		instance1: {MobId: mobs.MobId(template1), InstanceId: instance1, HomeRoomId: roomId,
			Character: characters.Character{Name: name1, RoomId: roomId, Buffs: buffs.New()}},
		instance2: {MobId: mobs.MobId(template2), InstanceId: instance2, HomeRoomId: roomId,
			Character: characters.Character{Name: name2, RoomId: roomId, Buffs: buffs.New()}},
	}
	return mobs.SeedMobsForTest(specs, instances)
}

// seedTestUser seeds a single user (UserId, username, charName, RoomId).
// Returns a cleanup function.
func seedTestUser(t *testing.T, userId int, username string, charName string, roomId int) func() {
	t.Helper()
	u := users.NewTestUser(userId, username, charName, uint64(userId))
	u.Character.RoomId = roomId
	return users.SeedUsersForTest(map[int]*users.UserRecord{userId: u})
}

// seedTestRoom seeds a single room with the supplied id and zone.
// Returns a cleanup function.
func seedTestRoom(t *testing.T, roomId int, zone string) func() {
	t.Helper()
	r := &rooms.Room{
		RoomId: roomId,
		Zone:   zone,
		Title:  "Test Room",
		Exits:  map[string]exit.RoomExit{},
	}
	return rooms.SeedRoomsForTest(
		map[int]*rooms.Room{roomId: r},
		map[string]*rooms.ZoneConfig{},
	)
}
