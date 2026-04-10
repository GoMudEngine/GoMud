package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstanceRegistry_CreateAndLookup(t *testing.T) {
	registry := NewInstanceRegistry()

	// Create a ZoneInstance
	inst := &ZoneInstance{
		InstanceId:      1001,
		TemplateZone:    "Arena",
		GoldPaid:        100,
		AuthorizedUsers: []int{10, 20, 30},
		OwnerUserId:     10,
		CreatedRound:    5000,
		PortalDuration:  "30m",
		DeathPolicy:     "rejoin",
		AllowRecall:     true,
		OverworldRoomId: 100,
		EntryRoomId:     2000,
		RoomIdMap: map[int]int{
			101: 2001,
			102: 2002,
			103: 2003,
		},
	}

	// Add to registry
	registry.Add(inst)

	// Find by entry room ID
	found := registry.FindByRoomId(inst.EntryRoomId)
	assert.NotNil(t, found)
	assert.Equal(t, inst.InstanceId, found.InstanceId)
	assert.Equal(t, inst.TemplateZone, found.TemplateZone)

	// Find by first mapped room ID
	found = registry.FindByRoomId(2001)
	assert.NotNil(t, found)
	assert.Equal(t, inst.InstanceId, found.InstanceId)

	// Find by second mapped room ID
	found = registry.FindByRoomId(2002)
	assert.NotNil(t, found)
	assert.Equal(t, inst.InstanceId, found.InstanceId)

	// Find by third mapped room ID
	found = registry.FindByRoomId(2003)
	assert.NotNil(t, found)
	assert.Equal(t, inst.InstanceId, found.InstanceId)

	// Unknown room returns nil
	found = registry.FindByRoomId(9999)
	assert.Nil(t, found)

	// Check authorization
	assert.True(t, inst.IsAuthorized(10))
	assert.True(t, inst.IsAuthorized(20))
	assert.True(t, inst.IsAuthorized(30))
	assert.False(t, inst.IsAuthorized(40))
}

func TestInstanceRegistry_RevokeAccess(t *testing.T) {
	inst := &ZoneInstance{
		InstanceId:      1001,
		TemplateZone:    "Arena",
		AuthorizedUsers: []int{10, 20, 30},
		OwnerUserId:     10,
		RoomIdMap:       map[int]int{},
	}

	// Check initial authorization
	assert.True(t, inst.IsAuthorized(10))
	assert.True(t, inst.IsAuthorized(20))
	assert.True(t, inst.IsAuthorized(30))

	// Revoke access for user 20
	inst.RevokeAccess(20)

	// Check authorization after revoke
	assert.True(t, inst.IsAuthorized(10))
	assert.False(t, inst.IsAuthorized(20))
	assert.True(t, inst.IsAuthorized(30))

	// Verify the slice was actually filtered
	assert.Len(t, inst.AuthorizedUsers, 2)
	assert.Equal(t, []int{10, 30}, inst.AuthorizedUsers)

	// Revoke another user
	inst.RevokeAccess(10)
	assert.False(t, inst.IsAuthorized(10))
	assert.False(t, inst.IsAuthorized(20))
	assert.True(t, inst.IsAuthorized(30))
	assert.Len(t, inst.AuthorizedUsers, 1)

	// Revoke the last user
	inst.RevokeAccess(30)
	assert.False(t, inst.IsAuthorized(30))
	assert.Len(t, inst.AuthorizedUsers, 0)

	// Revoking a user not in the list should be a no-op
	inst.RevokeAccess(999)
	assert.Len(t, inst.AuthorizedUsers, 0)
}

func TestInstanceRegistry_Remove(t *testing.T) {
	registry := NewInstanceRegistry()

	inst := &ZoneInstance{
		InstanceId:      1001,
		TemplateZone:    "Arena",
		AuthorizedUsers: []int{10, 20},
		OwnerUserId:     10,
		EntryRoomId:     2000,
		RoomIdMap: map[int]int{
			101: 2001,
			102: 2002,
		},
	}

	// Add instance
	registry.Add(inst)

	// Verify it can be found
	found := registry.FindByRoomId(2001)
	assert.NotNil(t, found)

	// Remove instance
	registry.Remove(inst)

	// Verify it cannot be found by entry room
	found = registry.FindByRoomId(2000)
	assert.Nil(t, found)

	// Verify it cannot be found by mapped rooms
	found = registry.FindByRoomId(2001)
	assert.Nil(t, found)

	found = registry.FindByRoomId(2002)
	assert.Nil(t, found)

	// Verify instances slice is empty
	all := registry.All()
	assert.Len(t, all, 0)
}

func TestInstanceRegistry_MultipleInstances(t *testing.T) {
	registry := NewInstanceRegistry()

	// Create two instances
	inst1 := &ZoneInstance{
		InstanceId:      1001,
		TemplateZone:    "Arena",
		AuthorizedUsers: []int{10},
		OwnerUserId:     10,
		EntryRoomId:     2000,
		RoomIdMap: map[int]int{
			101: 2001,
		},
	}

	inst2 := &ZoneInstance{
		InstanceId:      1002,
		TemplateZone:    "Oasis",
		AuthorizedUsers: []int{20},
		OwnerUserId:     20,
		EntryRoomId:     3000,
		RoomIdMap: map[int]int{
			201: 3001,
		},
	}

	// Add both
	registry.Add(inst1)
	registry.Add(inst2)

	// Check All() returns both
	all := registry.All()
	assert.Len(t, all, 2)

	// Verify both are findable by their room IDs
	found := registry.FindByRoomId(2001)
	assert.NotNil(t, found)
	assert.Equal(t, 1001, found.InstanceId)

	found = registry.FindByRoomId(3001)
	assert.NotNil(t, found)
	assert.Equal(t, 1002, found.InstanceId)

	// Remove first instance
	registry.Remove(inst1)

	// Verify only second remains
	all = registry.All()
	assert.Len(t, all, 1)
	assert.Equal(t, 1002, all[0].InstanceId)

	// Verify first is no longer findable
	found = registry.FindByRoomId(2001)
	assert.Nil(t, found)

	// Verify second is still findable
	found = registry.FindByRoomId(3001)
	assert.NotNil(t, found)
}

func TestInstanceRegistry_GetGlobalSingleton(t *testing.T) {
	registry1 := GetInstanceRegistry()
	registry2 := GetInstanceRegistry()

	// Both should be the same instance
	assert.True(t, registry1 == registry2)
}

func TestInstanceRegistry_EmptyLookup(t *testing.T) {
	registry := NewInstanceRegistry()

	// Lookup in empty registry should return nil
	found := registry.FindByRoomId(9999)
	assert.Nil(t, found)

	// All() should return empty slice
	all := registry.All()
	assert.Len(t, all, 0)
}

func TestInstanceRegistry_AllReturnsSnapshot(t *testing.T) {
	registry := NewInstanceRegistry()

	inst1 := &ZoneInstance{
		InstanceId:      1001,
		TemplateZone:    "Arena",
		AuthorizedUsers: []int{10},
		OwnerUserId:     10,
		EntryRoomId:     2000,
		RoomIdMap:       map[int]int{},
	}

	inst2 := &ZoneInstance{
		InstanceId:      1002,
		TemplateZone:    "Oasis",
		AuthorizedUsers: []int{20},
		OwnerUserId:     20,
		EntryRoomId:     3000,
		RoomIdMap:       map[int]int{},
	}

	registry.Add(inst1)
	snapshot1 := registry.All()
	assert.Len(t, snapshot1, 1)

	registry.Add(inst2)
	snapshot2 := registry.All()
	assert.Len(t, snapshot2, 2)

	// snapshot1 should not have changed
	assert.Len(t, snapshot1, 1)
}

func TestZoneInstance_IsAuthorizedEmpty(t *testing.T) {
	inst := &ZoneInstance{
		InstanceId:      1001,
		TemplateZone:    "Arena",
		AuthorizedUsers: []int{},
		OwnerUserId:     10,
		RoomIdMap:       map[int]int{},
	}

	// Empty auth list should reject all
	assert.False(t, inst.IsAuthorized(10))
	assert.False(t, inst.IsAuthorized(20))
}

func TestZoneInstance_RevokeAccessNotInList(t *testing.T) {
	inst := &ZoneInstance{
		InstanceId:      1001,
		TemplateZone:    "Arena",
		AuthorizedUsers: []int{10, 20},
		OwnerUserId:     10,
		RoomIdMap:       map[int]int{},
	}

	// Revoking a user not in the list should not panic or error
	inst.RevokeAccess(999)

	// List should be unchanged
	assert.Len(t, inst.AuthorizedUsers, 2)
	assert.True(t, inst.IsAuthorized(10))
	assert.True(t, inst.IsAuthorized(20))
}

func TestScaleSpawnStatPools(t *testing.T) {
	spawns := []SpawnInfo{
		{MobId: 1, StatPool: 1},
		{MobId: 2, StatPool: 2},
		{MobId: 3, StatPool: 3},
		{MobId: 4, StatPool: 0},
	}
	ScaleSpawnStatPools(spawns, 500, 50000)
	assert.Equal(t, 500, spawns[0].StatPool)
	assert.Equal(t, 1000, spawns[1].StatPool)
	assert.Equal(t, 1500, spawns[2].StatPool)
	assert.Equal(t, 500, spawns[3].StatPool)
}

func TestScaleSpawnStatPools_Cap(t *testing.T) {
	spawns := []SpawnInfo{
		{MobId: 1, StatPool: 3},
	}
	ScaleSpawnStatPools(spawns, 20000, 50000)
	assert.Equal(t, 50000, spawns[0].StatPool)
}

func TestScaleSpawnStatPools_NoCap(t *testing.T) {
	spawns := []SpawnInfo{
		{MobId: 1, StatPool: 3},
	}
	ScaleSpawnStatPools(spawns, 20000, 0)
	assert.Equal(t, 60000, spawns[0].StatPool)
}
