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
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)

	// Seed a default biome so GetVisibility() / GetBiome() don't return nil
	// when rooms are created with no explicit Biome field. Without this,
	// any code path that calls room.GetBiome().IsDark() panics.
	rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		`default`: {
			BiomeId:      `default`,
			Name:         `Default`,
			Symbol:       `•`,
			LitArea:      true,
			Description:  `A default biome used in tests.`,
			MovementCost: 1.0,
		},
	})

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

// grantHiddenBuff adds buff 9 to the character AND advances the Awareness
// state machine to Hidden so that char.IsHidden() returns true.
// Callers must have seeded hiddenBuffSpec (or equivalent) before calling.
// Uses a fatal error if AddBuff fails so tests get a clear message.
func grantHiddenBuff(t *testing.T, char *characters.Character) {
	t.Helper()
	if err := char.AddBuff(9, false); err != nil {
		t.Fatalf("grantHiddenBuff: AddBuff(9) failed: %v", err)
	}
	// Sync Awareness machine to Hidden state so char.IsHidden() returns true.
	if char.Awareness == nil {
		char.Awareness = awareness.NewMachine()
	}
	r := state.TransitionReason{Trigger: "test_setup"}
	char.Awareness.ForceVisible(r) // reset regardless of current state
	_ = char.Awareness.TransitionToConcealing(awareness.ConcealingData{}, r)
	char.Awareness.ResolveConcealment(true, r)
}
