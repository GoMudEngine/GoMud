package mobcommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCallForHelp_FiresHeardCallforhelpOnRoutineMatch verifies the new
// pack-tactics path: mobs in adjacent rooms whose routine matches the
// caller's (or is linked to it) receive a heard_callforhelp btree event.
// Non-matching mobs do not receive the event.
func TestCallForHelp_FiresHeardCallforhelpOnRoutineMatch(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Caller in room 1 with a routine.
	caller := mobs.GetInstance(100)
	require.NotNil(t, caller)
	caller.Routine = "bandit_camp_guard"

	// Seed two mobs in room 2 (already exists from seedAllRegistries):
	// one matching routine, one not.
	matcher := &mobs.Mob{
		InstanceId: 2000,
		Routine:    "bandit_camp_guard",
		Character: characters.Character{
			RoomId: 2,
			Health: 50,
		},
	}
	nonMatcher := &mobs.Mob{
		InstanceId: 2001,
		Routine:    "temple_service",
		Character: characters.Character{
			RoomId: 2,
			Health: 50,
		},
	}
	seed2 := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		2000: matcher,
		2001: nonMatcher,
	})
	defer seed2()

	room2 := rooms.LoadRoom(2)
	require.NotNil(t, room2)
	room2.AddMob(2000)
	room2.AddMob(2001)

	// Room 1 should already have exit to room 2 from seedAllRegistries.
	room1 := rooms.LoadRoom(1)
	require.NotNil(t, room1)
	require.NotNil(t, room1.Exits["north"])
	require.Equal(t, 2, room1.Exits["north"].RoomId)

	// Spy on the btree event dispatcher.
	recorded := map[int]string{}
	orig := dispatchEventFn
	t.Cleanup(func() { dispatchEventFn = orig })
	dispatchEventFn = func(instId int, ctx behaviortree.EventContext) bool {
		recorded[instId] = ctx.EventType
		return true
	}

	_, _ = CallForHelp("", caller, room1)

	assert.Equal(t, "heard_callforhelp", recorded[2000],
		"matching-routine mob should receive heard_callforhelp")
	_, fired := recorded[2001]
	assert.False(t, fired,
		"non-matching-routine mob should not receive heard_callforhelp")
}

// TestCallForHelp_FiresHeardCallforhelpOnRoutineLinksMatch verifies that
// mobs receiving a callforhelp event through RoutineLinks also get
// heard_callforhelp fired.
func TestCallForHelp_FiresHeardCallforhelpOnRoutineLinksMatch(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	// Caller in room 1 with a routine.
	caller := mobs.GetInstance(100)
	require.NotNil(t, caller)
	caller.Routine = "bandit_camp_guard"

	// Seed a mob in room 2 (already exists from seedAllRegistries) linked via RoutineLinks.
	linkedMob := &mobs.Mob{
		InstanceId:   2000,
		Routine:      "bandit_lookout_north",
		RoutineLinks: []string{"bandit_camp_guard"},
		Character: characters.Character{
			RoomId: 2,
			Health: 50,
		},
	}
	seed2 := mobs.SeedMobsForTest(nil, map[int]*mobs.Mob{
		2000: linkedMob,
	})
	defer seed2()

	room2 := rooms.LoadRoom(2)
	require.NotNil(t, room2)
	room2.AddMob(2000)

	// Room 1 should already have exit to room 2 from seedAllRegistries.
	room1 := rooms.LoadRoom(1)
	require.NotNil(t, room1)
	require.NotNil(t, room1.Exits["north"])
	require.Equal(t, 2, room1.Exits["north"].RoomId)

	// Spy on the btree event dispatcher.
	recorded := map[int]string{}
	orig := dispatchEventFn
	t.Cleanup(func() { dispatchEventFn = orig })
	dispatchEventFn = func(instId int, ctx behaviortree.EventContext) bool {
		recorded[instId] = ctx.EventType
		return true
	}

	_, _ = CallForHelp("", caller, room1)

	assert.Equal(t, "heard_callforhelp", recorded[2000],
		"routine-linked mob should receive heard_callforhelp")
}
