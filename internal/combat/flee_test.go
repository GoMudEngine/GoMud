package combat

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedFleeFixture wires the minimal room + mob/user registries the
// ResolveFleeBlockers helper needs to walk. Returns a cleanup func
// the test must defer.
func seedFleeFixture(t *testing.T, fleer *characters.Character,
	blockerMobs map[int]*mobs.Mob, blockerUsers map[int]*users.UserRecord) func() {
	t.Helper()

	room := &rooms.Room{
		RoomId: 1,
		Zone:   "TestZone",
		Title:  "Test Room",
		Exits:  map[string]exit.RoomExit{"north": {RoomId: 2}},
	}
	north := &rooms.Room{RoomId: 2, Zone: "TestZone", Title: "North"}

	cleanupRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{1: room, 2: north},
		map[string]*rooms.ZoneConfig{
			"TestZone": {Name: "TestZone", RoomId: 1, RoomIds: map[int]struct{}{1: {}, 2: {}}},
		},
	)

	cleanupMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{1: {MobId: 1, Zone: "TestZone"}},
		blockerMobs,
	)
	for id := range blockerMobs {
		room.AddMob(id)
	}

	if blockerUsers == nil {
		blockerUsers = map[int]*users.UserRecord{}
	}
	cleanupUsers := users.SeedUsersForTest(blockerUsers)
	for id := range blockerUsers {
		room.AddPlayer(id)
	}

	// Place the fleer in the same room.
	fleer.RoomId = 1

	return func() {
		cleanupRooms()
		cleanupMobs()
		cleanupUsers()
	}
}

// newFleerMob builds a mob instance suitable for the fleer side of
// the helper. Stats default to mid-tier; caller adjusts as needed.
func newFleerMob(id int, name string, dex int, skullduggery int) *mobs.Mob {
	m := &mobs.Mob{
		MobId:      1,
		InstanceId: id,
		HomeRoomId: 1,
		Character: characters.Character{
			Name:      name,
			RoomId:    1,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
			Skills:    map[string]int{string(skills.Skullduggery): skullduggery},
		},
	}
	m.Character.MobInstanceId = id
	m.Character.Stats.Dexterity.ValueAdj = dex
	m.Character.HealthMax.Value = 100
	m.Character.Health = 100
	return m
}

// newBlockerMob builds a mob whose aggro targets the given fleer
// (either UserId or MobInstanceId).
func newBlockerMob(id int, name string, dex int, unarmed int, targetUid, targetMid int) *mobs.Mob {
	m := &mobs.Mob{
		MobId:      1,
		InstanceId: id,
		HomeRoomId: 1,
		Character: characters.Character{
			Name:      name,
			RoomId:    1,
			Buffs:     buffs.New(),
			Cooldowns: map[string]int{},
			Skills:    map[string]int{string(skills.UnarmedCombat): unarmed},
			Aggro: &characters.Aggro{
				Type:          characters.DefaultAttack,
				UserId:        targetUid,
				MobInstanceId: targetMid,
			},
		},
	}
	m.Character.MobInstanceId = id
	m.Character.Stats.Dexterity.ValueAdj = dex
	m.Character.HealthMax.Value = 100
	m.Character.Health = 100
	return m
}

func TestResolveFleeBlockers_NoOpposers(t *testing.T) {
	fleer := newFleerMob(100, "fleer", 100, 25)
	cleanup := seedFleeFixture(t, &fleer.Character,
		map[int]*mobs.Mob{100: fleer},
		nil,
	)
	defer cleanup()

	blocker := ResolveFleeBlockers(&fleer.Character, rooms.LoadRoom(1))
	assert.Nil(t, blocker, "no combatants in room → no blocker")
}

func TestResolveFleeBlockers_MobBlockerWins(t *testing.T) {
	fleer := newFleerMob(100, "weakling", 20, 0)
	// Strong mob blocker targeting the fleer's MobInstanceId.
	blockerMob := newBlockerMob(200, "bouncer", 200, 50, 0, 100)
	cleanup := seedFleeFixture(t, &fleer.Character,
		map[int]*mobs.Mob{100: fleer, 200: blockerMob},
		nil,
	)
	defer cleanup()

	blocker := ResolveFleeBlockers(&fleer.Character, rooms.LoadRoom(1))
	require.NotNil(t, blocker, "strong mob blocker should win opposed roll")
	assert.Equal(t, "bouncer", blocker.Name)
	assert.Equal(t, 200, blocker.MobInstanceId)
	assert.False(t, blocker.IsPlayer())
}

func TestResolveFleeBlockers_PlayerBlockerWins(t *testing.T) {
	// Fleer is a mob targeted by a player blocker — this is the
	// PvM-from-mob-side case. (Pre-refactor, the player-flee path
	// had a shadow bug that would have failed the inverse case
	// (player fleer / player blocker); the shared helper fixes that
	// naturally by reading the fleer's identity, not the iterated
	// blocker's.)
	fleer := newFleerMob(100, "weak-mob", 20, 0)
	blockerUser := users.NewTestUser(1, "alice", "Aliceia", 1001)
	blockerUser.Character.Stats.Dexterity.ValueAdj = 200
	blockerUser.Character.Skills = map[string]int{string(skills.UnarmedCombat): 50}
	// Player needs an aggro target pointing at the fleer's mob id.
	blockerUser.Character.SetAggro(0, 100, characters.DefaultAttack)

	cleanup := seedFleeFixture(t, &fleer.Character,
		map[int]*mobs.Mob{100: fleer},
		map[int]*users.UserRecord{1: blockerUser},
	)
	defer cleanup()

	blocker := ResolveFleeBlockers(&fleer.Character, rooms.LoadRoom(1))
	require.NotNil(t, blocker, "strong player blocker should win opposed roll")
	assert.Equal(t, "Aliceia", blocker.Name)
	assert.Equal(t, 1, blocker.UserId)
	assert.True(t, blocker.IsPlayer())
}

func TestResolveFleeBlockers_NonTargetingCombatantIgnored(t *testing.T) {
	// Strong mob in the room but its aggro points at a DIFFERENT
	// target — the helper should NOT count it as a blocker.
	fleer := newFleerMob(100, "weakling", 20, 0)
	otherMob := newBlockerMob(200, "uninterested", 200, 50, 0, 999) // targets mob 999
	cleanup := seedFleeFixture(t, &fleer.Character,
		map[int]*mobs.Mob{100: fleer, 200: otherMob},
		nil,
	)
	defer cleanup()

	blocker := ResolveFleeBlockers(&fleer.Character, rooms.LoadRoom(1))
	assert.Nil(t, blocker, "combatant not targeting the fleer should not block")
}

func TestResolveFleeBlockers_NilInputs(t *testing.T) {
	assert.Nil(t, ResolveFleeBlockers(nil, nil))
	c := &characters.Character{}
	assert.Nil(t, ResolveFleeBlockers(c, nil))
}
