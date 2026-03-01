package mobs

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

// ─── SelectPackAlpha ────────────────────────────────────────────────────────

func TestSelectPackAlpha_HighestStatPool(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	mobInstances[200] = &Mob{
		MobId: 1, InstanceId: 200, StatPool: 100,
		Groups: []string{"canine"},
		Character: characters.Character{Name: "Wolf A", RoomId: 10},
	}
	mobInstances[201] = &Mob{
		MobId: 1, InstanceId: 201, StatPool: 150,
		Groups: []string{"canine"},
		Character: characters.Character{Name: "Wolf B", RoomId: 10},
	}
	mobInstances[202] = &Mob{
		MobId: 1, InstanceId: 202, StatPool: 120,
		Groups: []string{"canine"},
		Character: characters.Character{Name: "Wolf C", RoomId: 10},
	}

	alphaId := SelectPackAlpha([]int{200, 201, 202})
	assert.Equal(t, 201, alphaId, "highest StatPool should be alpha")
}

func TestSelectPackAlpha_TieBreakByInstanceId(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	mobInstances[200] = &Mob{
		MobId: 1, InstanceId: 200, StatPool: 100,
		Character: characters.Character{Name: "Wolf A", RoomId: 10},
	}
	mobInstances[201] = &Mob{
		MobId: 1, InstanceId: 201, StatPool: 100,
		Character: characters.Character{Name: "Wolf B", RoomId: 10},
	}

	alphaId := SelectPackAlpha([]int{200, 201})
	assert.Equal(t, 200, alphaId, "lowest InstanceId should win tie")
}

func TestSelectPackAlpha_LessThanTwo(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	assert.Equal(t, 0, SelectPackAlpha([]int{100}))
	assert.Equal(t, 0, SelectPackAlpha([]int{}))
	assert.Equal(t, 0, SelectPackAlpha(nil))
}

func TestSelectPackAlpha_NilInstance(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	mobInstances[200] = &Mob{
		MobId: 1, InstanceId: 200, StatPool: 100,
		Character: characters.Character{Name: "Wolf", RoomId: 10},
	}

	alphaId := SelectPackAlpha([]int{200, 999})
	assert.Equal(t, 200, alphaId)
}

// ─── GetPackMembersInRoom ───────────────────────────────────────────────────

func TestGetPackMembersInRoom(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	mobInstances[200] = &Mob{
		MobId: 1, InstanceId: 200, Groups: []string{"canine"},
		Character: characters.Character{Name: "Wolf A", RoomId: 10},
	}
	mobInstances[201] = &Mob{
		MobId: 1, InstanceId: 201, Groups: []string{"canine"},
		Character: characters.Character{Name: "Wolf B", RoomId: 10},
	}
	mobInstances[202] = &Mob{
		MobId: 2, InstanceId: 202, Groups: []string{"ursine"},
		Character: characters.Character{Name: "Bear", RoomId: 10},
	}

	members := GetPackMembersInRoom(mobInstances[200], []int{200, 201, 202})
	assert.Len(t, members, 2, "should find 2 canine members")
	assert.Contains(t, members, 200)
	assert.Contains(t, members, 201)
}

func TestGetPackMembersInRoom_NoGroups(t *testing.T) {
	mob := &Mob{Groups: []string{}}
	members := GetPackMembersInRoom(mob, []int{100, 101})
	assert.Nil(t, members)
}

// ─── HandleAlphaDeath ──────────────────────────────────────────────────────

func TestHandleAlphaDeath(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	deadMob := &Mob{InstanceId: 300, Groups: []string{"canine"}}

	survivor1 := &Mob{
		MobId: 1, InstanceId: 200, Groups: []string{"canine"},
		PackAlphaId: 300, IsPackAlpha: false,
		Character: characters.Character{Name: "Wolf A", RoomId: 10},
	}
	survivor2 := &Mob{
		MobId: 1, InstanceId: 201, Groups: []string{"canine"},
		PackAlphaId: 300, IsPackAlpha: false,
		Character: characters.Character{Name: "Wolf B", RoomId: 10},
	}
	bystander := &Mob{
		MobId: 2, InstanceId: 202, Groups: []string{"ursine"},
		Character: characters.Character{Name: "Bear", RoomId: 10},
	}
	mobInstances[200] = survivor1
	mobInstances[201] = survivor2
	mobInstances[202] = bystander

	// Use internal function with explicit scatter rounds
	handleAlphaDeathWithScatter(deadMob, []int{200, 201, 202}, 3)

	assert.Equal(t, 0, survivor1.PackAlphaId, "pack state cleared")
	assert.False(t, survivor1.IsPackAlpha, "pack alpha cleared")
	assert.Equal(t, 3, survivor1.ScatterRounds, "scatter rounds set")

	assert.Equal(t, 0, survivor2.PackAlphaId, "pack state cleared")
	assert.Equal(t, 3, survivor2.ScatterRounds, "scatter rounds set")

	// Bystander unaffected
	assert.Equal(t, 0, bystander.ScatterRounds, "bystander unaffected")
}

func TestHandleAlphaDeath_NilMob(t *testing.T) {
	handleAlphaDeathWithScatter(nil, []int{100}, 2)
}

func TestHandleAlphaDeath_NoGroups(t *testing.T) {
	deadMob := &Mob{InstanceId: 300, Groups: []string{}}
	handleAlphaDeathWithScatter(deadMob, []int{100}, 2)
}

// ─── ScatterRounds Decrement ───────────────────────────────────────────────

func TestScatterRoundsDecrement(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	mob := &Mob{
		MobId: 1, InstanceId: 200, Groups: []string{"canine"},
		ScatterRounds: 3,
		Character: characters.Character{Name: "Wolf", RoomId: 10},
	}
	mobInstances[200] = mob

	// Use internal function directly (bypasses config check)
	tickPackRoamingWithMaxSize(-1)
	assert.Equal(t, 2, mob.ScatterRounds)

	tickPackRoamingWithMaxSize(-1)
	assert.Equal(t, 1, mob.ScatterRounds)

	tickPackRoamingWithMaxSize(-1)
	assert.Equal(t, 0, mob.ScatterRounds)

	tickPackRoamingWithMaxSize(-1)
	assert.Equal(t, 0, mob.ScatterRounds, "should not go negative")
}

// ─── TickPackRoaming ───────────────────────────────────────────────────────

func TestTickPackRoaming_ElectsAlpha(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	wolf1 := &Mob{
		MobId: 1, InstanceId: 200, StatPool: 100,
		Groups:    []string{"canine"},
		Character: characters.Character{Name: "Wolf A", RoomId: 10},
	}
	wolf2 := &Mob{
		MobId: 1, InstanceId: 201, StatPool: 150,
		Groups:    []string{"canine"},
		Character: characters.Character{Name: "Wolf B", RoomId: 10},
	}
	mobInstances[200] = wolf1
	mobInstances[201] = wolf2

	tickPackRoamingWithMaxSize(-1)

	assert.True(t, wolf2.IsPackAlpha, "highest StatPool should be alpha")
	assert.Equal(t, 0, wolf2.PackAlphaId, "alpha doesn't follow anyone")
	assert.Equal(t, 201, wolf1.PackAlphaId, "follower should reference alpha")
	assert.False(t, wolf1.IsPackAlpha, "follower is not alpha")
}

func TestTickPackRoaming_ClearsStaleAlpha(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	wolf := &Mob{
		MobId: 1, InstanceId: 200, StatPool: 100,
		Groups:      []string{"canine"},
		PackAlphaId: 999, // stale reference
		Character:   characters.Character{Name: "Wolf", RoomId: 10},
	}
	mobInstances[200] = wolf

	tickPackRoamingWithMaxSize(-1)

	assert.Equal(t, 0, wolf.PackAlphaId, "stale alpha ref should be cleared")
	assert.False(t, wolf.IsPackAlpha, "lone mob is not alpha")
}

func TestTickPackRoaming_DifferentRooms(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	wolf1 := &Mob{
		MobId: 1, InstanceId: 200, StatPool: 100,
		Groups:    []string{"canine"},
		Character: characters.Character{Name: "Wolf A", RoomId: 10},
	}
	wolf2 := &Mob{
		MobId: 1, InstanceId: 201, StatPool: 150,
		Groups:    []string{"canine"},
		Character: characters.Character{Name: "Wolf B", RoomId: 20}, // different room
	}
	mobInstances[200] = wolf1
	mobInstances[201] = wolf2

	tickPackRoamingWithMaxSize(-1)

	assert.False(t, wolf1.IsPackAlpha)
	assert.Equal(t, 0, wolf1.PackAlphaId)
	assert.False(t, wolf2.IsPackAlpha)
	assert.Equal(t, 0, wolf2.PackAlphaId)
}

func TestTickPackRoaming_ScatteredMobsSkipped(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	wolf1 := &Mob{
		MobId: 1, InstanceId: 200, StatPool: 100,
		Groups: []string{"canine"}, ScatterRounds: 2,
		Character: characters.Character{Name: "Wolf A", RoomId: 10},
	}
	wolf2 := &Mob{
		MobId: 1, InstanceId: 201, StatPool: 150,
		Groups: []string{"canine"}, ScatterRounds: 2,
		Character: characters.Character{Name: "Wolf B", RoomId: 10},
	}
	mobInstances[200] = wolf1
	mobInstances[201] = wolf2

	tickPackRoamingWithMaxSize(-1)

	// Scattered mobs should not form packs
	assert.False(t, wolf1.IsPackAlpha)
	assert.Equal(t, 0, wolf1.PackAlphaId)
	assert.False(t, wolf2.IsPackAlpha)
	assert.Equal(t, 0, wolf2.PackAlphaId)

	// But scatter rounds should have decremented
	assert.Equal(t, 1, wolf1.ScatterRounds)
	assert.Equal(t, 1, wolf2.ScatterRounds)
}

func TestTickPackRoaming_RespectsMaxSize(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	alpha := &Mob{
		MobId: 1, InstanceId: 200, StatPool: 200,
		Groups:    []string{"canine"},
		Character: characters.Character{Name: "Alpha", RoomId: 10},
	}
	follower1 := &Mob{
		MobId: 1, InstanceId: 201, StatPool: 100,
		Groups:    []string{"canine"},
		Character: characters.Character{Name: "Wolf B", RoomId: 10},
	}
	follower2 := &Mob{
		MobId: 1, InstanceId: 202, StatPool: 80,
		Groups:    []string{"canine"},
		Character: characters.Character{Name: "Wolf C", RoomId: 10},
	}
	mobInstances[200] = alpha
	mobInstances[201] = follower1
	mobInstances[202] = follower2

	// MaxSize = 1: only one follower allowed
	tickPackRoamingWithMaxSize(1)

	assert.True(t, alpha.IsPackAlpha)
	// One follower should have the alpha, one should not
	followCount := 0
	if follower1.PackAlphaId == 200 {
		followCount++
	}
	if follower2.PackAlphaId == 200 {
		followCount++
	}
	assert.Equal(t, 1, followCount, "only 1 follower should be in pack")
}

// ─── MovePackFollowers ─────────────────────────────────────────────────────

func TestMovePackFollowers_NilAlpha(t *testing.T) {
	MovePackFollowers(nil, "north", []int{100})
}

func TestMovePackFollowers_MaxWanderBreaksOff(t *testing.T) {
	cleanup := seedRegistry()
	defer cleanup()

	alpha := &Mob{
		MobId: 1, InstanceId: 200, IsPackAlpha: true,
		Groups:    []string{"canine"},
		Character: characters.Character{Name: "Alpha Wolf", RoomId: 10},
	}
	follower := &Mob{
		MobId: 1, InstanceId: 201, PackAlphaId: 200,
		MaxWander: 3, WanderCount: 4,
		Groups:    []string{"canine"},
		Character: characters.Character{Name: "Follower Wolf", RoomId: 10},
	}
	mobInstances[200] = alpha
	mobInstances[201] = follower

	MovePackFollowers(alpha, "north", []int{200, 201})

	assert.Equal(t, 0, follower.PackAlphaId, "should break off due to MaxWander")
}
