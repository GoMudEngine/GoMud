package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedTargetResolutionRegistries seeds the cross-package registries needed
// to exercise rooms.FindByName + ResolveTargetActor end-to-end. Returns a
// combined cleanup function. The fixture mirrors the minimal surface area
// of hooks/seedAllRegistries trimmed to what target resolution touches:
//
//   - users registry: 1=alice (display name "Alice"), 2=bob ("Bob"),
//     both placed in room 1
//   - mobs registry: spec 1=Skeleton, instance 100 (name "Skeleton")
//     placed in room 1; second instance 101 (name "Wraith") also in room 1
//     for tests that need multiple mobs.
//   - rooms: room 1 ("Town Square") with both players + both mobs
func seedTargetResolutionRegistries(t *testing.T) func() {
	t.Helper()

	// Users
	u1 := users.NewTestUser(1, "alice", "Alice", 1001)
	u1.Character.RoomId = 1
	u2 := users.NewTestUser(2, "bob", "Bob", 1002)
	u2.Character.RoomId = 1
	cleanupUsers := users.SeedUsersForTest(map[int]*users.UserRecord{
		1: u1,
		2: u2,
	})

	// Mobs
	mobSpecs := map[int]*mobs.Mob{
		1: {
			MobId: 1,
			Zone:  "TestZone",
			Character: characters.Character{
				Name: "Skeleton",
			},
		},
		2: {
			MobId: 2,
			Zone:  "TestZone",
			Character: characters.Character{
				Name: "Wraith",
			},
		},
	}
	mobInstances := map[int]*mobs.Mob{
		100: {
			MobId:      1,
			InstanceId: 100,
			HomeRoomId: 1,
			Character: characters.Character{
				Name:      "Skeleton",
				RoomId:    1,
				Health:    50,
				Buffs:     buffs.New(),
				Cooldowns: map[string]int{},
			},
		},
		101: {
			MobId:      2,
			InstanceId: 101,
			HomeRoomId: 1,
			Character: characters.Character{
				Name:      "Wraith",
				RoomId:    1,
				Health:    50,
				Buffs:     buffs.New(),
				Cooldowns: map[string]int{},
			},
		},
	}
	cleanupMobs := mobs.SeedMobsForTest(mobSpecs, mobInstances)

	// Rooms
	room1 := &rooms.Room{
		RoomId:      1,
		Zone:        "TestZone",
		Title:       "Town Square",
		Description: "A bustling town square.",
		Exits: map[string]exit.RoomExit{
			"north": {RoomId: 2},
		},
		Biome: "city",
	}
	cleanupRooms := rooms.SeedRoomsForTest(
		map[int]*rooms.Room{1: room1},
		map[string]*rooms.ZoneConfig{
			"TestZone": {
				Name:    "TestZone",
				RoomId:  1,
				RoomIds: map[int]struct{}{1: {}},
			},
		},
	)
	room1.AddPlayer(1)
	room1.AddPlayer(2)
	room1.AddMob(100)
	room1.AddMob(101)
	rooms.MarkRoomOccupancy(1, 2, 2)

	return func() {
		cleanupRooms()
		cleanupMobs()
		cleanupUsers()
	}
}

// TestResolveTargetActor_PlayerMatch: name resolves to a player; helper
// returns a *UserActor wrapping the correct user.
func TestResolveTargetActor_PlayerMatch(t *testing.T) {
	cleanup := seedTargetResolutionRegistries(t)
	defer cleanup()

	room := rooms.LoadRoom(1)
	require.NotNil(t, room)

	target, err := ResolveTargetActor(room, "alice")

	require.NoError(t, err)
	require.NotNil(t, target)
	assert.True(t, target.IsPlayer(), "expected UserActor (IsPlayer=true)")
	assert.Equal(t, 1, target.GetUserId(), "expected to resolve user id 1")
	assert.Equal(t, "Alice", target.GetName())

	// Concrete-type sanity: the helper returns a *UserActor, so asserting
	// on the concrete type should succeed for downstream user-only paths.
	_, ok := target.(*UserActor)
	assert.True(t, ok, "expected concrete *UserActor for player match")

	// Room is wired through so target.GetRoom() / SendRoomText work
	// without callers having to re-resolve.
	assert.Equal(t, room, target.GetRoom(), "expected target.GetRoom() to return the resolution room")
}

// TestResolveTargetActor_MobMatch: name resolves to a mob; helper returns
// a *MobActor wrapping the correct mob.
func TestResolveTargetActor_MobMatch(t *testing.T) {
	cleanup := seedTargetResolutionRegistries(t)
	defer cleanup()

	room := rooms.LoadRoom(1)
	require.NotNil(t, room)

	target, err := ResolveTargetActor(room, "skeleton")

	require.NoError(t, err)
	require.NotNil(t, target)
	assert.False(t, target.IsPlayer(), "expected MobActor (IsPlayer=false)")
	assert.Equal(t, 100, target.GetMobInstanceId(), "expected to resolve instance 100")
	assert.Equal(t, "Skeleton", target.GetName())

	// Concrete-type sanity for mob-only branches.
	_, ok := target.(*MobActor)
	assert.True(t, ok, "expected concrete *MobActor for mob match")

	// Room is wired through (see PlayerMatch test for the same promise).
	assert.Equal(t, room, target.GetRoom(), "expected target.GetRoom() to return the resolution room")
}

// TestResolveTargetActor_NotFound: no match in room → ErrTargetNotFound.
func TestResolveTargetActor_NotFound(t *testing.T) {
	cleanup := seedTargetResolutionRegistries(t)
	defer cleanup()

	room := rooms.LoadRoom(1)
	require.NotNil(t, room)

	target, err := ResolveTargetActor(room, "nobodyhere")

	assert.Nil(t, target)
	assert.ErrorIs(t, err, ErrTargetNotFound)
}

// TestResolveTargetActor_StalePlayerId: the room's player list still
// references a user id whose entry has been removed from the user
// registry. FindByName returns the id (via the @id resolver, which
// short-circuits the GetByUserId lookup when FindAll is in effect),
// and the helper's downstream GetByUserId returns nil → ErrTargetVanished.
//
// We use the @id syntax intentionally because the name-based path
// dereferences the user record without nil-checking inside FindByName
// itself, which would crash before reaching the helper. The race we model
// here is the genuine "registry mutated between FindByName and helper's
// pointer fetch" case — @id flows the id through without a name-table
// lookup, exposing the helper's nil guard.
func TestResolveTargetActor_StalePlayerId(t *testing.T) {
	cleanup := seedTargetResolutionRegistries(t)
	defer cleanup()

	room := rooms.LoadRoom(1)
	require.NotNil(t, room)

	// Sanity: with the user present, the @id lookup should succeed.
	target, err := ResolveTargetActor(room, "@1")
	require.NoError(t, err)
	require.NotNil(t, target)
	require.Equal(t, 1, target.GetUserId())

	// Now drop the user from the registry while leaving the room's player
	// list (containing user id 1) intact — the registry mutation race the
	// helper guards against.
	resealUsers := users.SeedUsersForTest(map[int]*users.UserRecord{
		2: users.NewTestUser(2, "bob", "Bob", 1002),
	})
	defer resealUsers()

	target2, err2 := ResolveTargetActor(room, "@1")
	assert.Nil(t, target2)
	assert.ErrorIs(t, err2, ErrTargetVanished,
		"helper must convert nil GetByUserId result to ErrTargetVanished")
}

// TestResolveTargetActor_StaleMobId: FindByName returns a mob ID but the
// instance is gone from the registry. → ErrTargetVanished.
//
// We use the #instanceId syntax intentionally because the name-based path
// dereferences the mob record without nil-checking inside findMobByName
// itself (mob.Character.IsCharmed()), which would crash before reaching
// the helper. The race we model here is the genuine "registry mutated
// between FindByName and helper's GetInstance call" case — #id flows the
// id through the mob list without a name-table dereference, exposing the
// helper's nil guard.
func TestResolveTargetActor_StaleMobId(t *testing.T) {
	cleanup := seedTargetResolutionRegistries(t)
	defer cleanup()

	room := rooms.LoadRoom(1)
	require.NotNil(t, room)

	// Sanity: present mob resolves cleanly via #id.
	target, err := ResolveTargetActor(room, "#100")
	require.NoError(t, err)
	require.NotNil(t, target)
	require.Equal(t, 100, target.GetMobInstanceId())

	// Destroy the instance while leaving the room's mob list (containing
	// instance id 100) intact — the registry mutation race the helper
	// guards against.
	mobs.DestroyInstance(100)

	target2, err2 := ResolveTargetActor(room, "#100")
	assert.Nil(t, target2)
	assert.ErrorIs(t, err2, ErrTargetVanished,
		"helper must convert nil GetInstance result to ErrTargetVanished")
}

// TestResolveTargetActor_ExcludeSelf: a user matches but is excluded via
// ExcludeUserId option. With no other user matches → ErrTargetNotFound.
// (The helper does not currently differentiate "not found" from "self
// excluded"; ErrTargetSelfExcluded is reserved for a future revision.)
func TestResolveTargetActor_ExcludeSelf(t *testing.T) {
	cleanup := seedTargetResolutionRegistries(t)
	defer cleanup()

	room := rooms.LoadRoom(1)
	require.NotNil(t, room)

	// Without exclusion: alice resolves cleanly.
	target, err := ResolveTargetActor(room, "alice")
	require.NoError(t, err)
	require.NotNil(t, target)

	// With exclusion of user 1: alice is filtered out and there is no
	// other "alice" in the room, so the helper returns ErrTargetNotFound.
	target2, err2 := ResolveTargetActor(room, "alice", ResolveTargetOptions{
		ExcludeUserId: 1,
	})
	assert.Nil(t, target2)
	assert.ErrorIs(t, err2, ErrTargetNotFound,
		"exclude-self with no other matches returns ErrTargetNotFound today; "+
			"ErrTargetSelfExcluded is reserved for a future revision when "+
			"callers need to differentiate.")
}

// TestResolveTargetActor_FindFlagFiltering: passing FindFighting only
// returns combatants. With a non-fighting mob in the room and no
// fighting mobs, name resolution returns ErrTargetNotFound. After
// setting Aggro, the same query resolves successfully.
func TestResolveTargetActor_FindFlagFiltering(t *testing.T) {
	cleanup := seedTargetResolutionRegistries(t)
	defer cleanup()

	room := rooms.LoadRoom(1)
	require.NotNil(t, room)

	// Skeleton has no Aggro; FindFighting filter must hide it.
	target, err := ResolveTargetActor(room, "skeleton", ResolveTargetOptions{
		FindFlags: []rooms.FindFlag{rooms.FindFighting},
	})
	assert.Nil(t, target)
	assert.ErrorIs(t, err, ErrTargetNotFound,
		"non-fighting mob must be filtered out by FindFighting")

	// Set the skeleton's Aggro on user 2 so it qualifies as
	// FindFightingPlayer, then re-query.
	skeleton := mobs.GetInstance(100)
	require.NotNil(t, skeleton)
	skeleton.Character.SetAggro(2, 0, characters.DefaultAttack)

	target2, err2 := ResolveTargetActor(room, "skeleton", ResolveTargetOptions{
		FindFlags: []rooms.FindFlag{rooms.FindFighting},
	})
	require.NoError(t, err2)
	require.NotNil(t, target2)
	assert.False(t, target2.IsPlayer())
	assert.Equal(t, 100, target2.GetMobInstanceId())
}
