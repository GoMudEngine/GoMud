package rooms

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

// init wires up mudlog for the rooms test binary. Several code paths
// exercised by these tests (e.g. TryEphemeralCleanup) call mudlog.Info
// unconditionally, which panics if the slog instance is nil. Mirrors the
// pattern in actions_test.go, hooks_test.go, etc.
func init() {
	mudlog.SetupLogger(nil, "", "", false)
}

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

// ─── CheckPortalTimers cleanup chain ────────────────────────────────────────
//
// These tests lock the TTL-triggered cleanup chain landed in
// `feat(rooms): TTL-triggered instance cleanup chain in CheckPortalTimers`.
// The chain runs: boot players → Remove(inst) → btree eviction → ephemeral
// cleanup. Tests use a fresh InstanceRegistry per case (the production
// CheckPortalTimers is a method on *InstanceRegistry, so no need to mutate
// the package-level singleton). The package-level btreeStateEvictor is
// snapshot+restored via defer where touched.

func TestCheckPortalTimers_TtlExpiryDeregisters(t *testing.T) {
	ir := NewInstanceRegistry()

	ephEntry := ephemeralRoomIdMinimum + 100
	ephExtra := ephemeralRoomIdMinimum + 101
	inst := &ZoneInstance{
		InstanceId:      ephEntry,
		TemplateZone:    "TestZone",
		AuthorizedUsers: []int{},
		OwnerUserId:     0,
		CreatedRound:    0, // long past
		PortalDuration:  "1m",
		OverworldRoomId: 1,
		EntryRoomId:     ephEntry,
		RoomIdMap: map[int]int{
			1: ephEntry,
			2: ephExtra,
		},
	}
	ir.Add(inst)

	ir.CheckPortalTimers()

	// Expired instance is removed from the registry.
	assert.Nil(t, ir.FindByRoomId(inst.EntryRoomId))
	assert.Nil(t, ir.FindByRoomId(ephExtra))
	assert.Len(t, ir.All(), 0)
}

func TestCheckPortalTimers_TtlExpiryEvictsBtreeState(t *testing.T) {
	// Snapshot/restore the package-level evictor.
	origEvictor := btreeStateEvictor
	defer SetBTreeStateEvictor(origEvictor)

	evicted := []int{}
	SetBTreeStateEvictor(func(roomId int) {
		evicted = append(evicted, roomId)
	})

	ir := NewInstanceRegistry()

	ephEntry := ephemeralRoomIdMinimum + 200
	ephExtra := ephemeralRoomIdMinimum + 201
	inst := &ZoneInstance{
		InstanceId:      ephEntry,
		TemplateZone:    "TestZone",
		AuthorizedUsers: []int{},
		OwnerUserId:     0,
		CreatedRound:    0,
		PortalDuration:  "1m",
		OverworldRoomId: 1,
		EntryRoomId:     ephEntry,
		RoomIdMap: map[int]int{
			1: ephEntry,
			2: ephExtra,
		},
	}
	ir.Add(inst)

	ir.CheckPortalTimers()

	// The registered btree-eviction callback fires once per ephemeral
	// room id in the instance's RoomIdMap.
	assert.Len(t, evicted, len(inst.RoomIdMap))
	assert.Contains(t, evicted, ephEntry)
	assert.Contains(t, evicted, ephExtra)
}

func TestCheckPortalTimers_TtlExpiryBootsPlayers(t *testing.T) {
	// Snapshot/restore the package-level evictor (chain calls it; we don't
	// care about its side effects here, just keep the package state clean).
	origEvictor := btreeStateEvictor
	defer SetBTreeStateEvictor(origEvictor)
	SetBTreeStateEvictor(func(int) {})

	overworldId := 4242
	ephEntry := ephemeralRoomIdMinimum + 300

	overworldRoom := &Room{
		RoomId:  overworldId,
		Zone:    "Overworld",
		Title:   "Town Square",
		players: []int{},
	}
	ephRoom := &Room{
		RoomId:  ephEntry,
		Zone:    "TestZone",
		Title:   "Ephemeral Hall",
		players: []int{},
	}

	cleanupRooms := SeedRoomsForTest(
		map[int]*Room{
			overworldId: overworldRoom,
			ephEntry:    ephRoom,
		},
		map[string]*ZoneConfig{
			"Overworld": {Name: "Overworld", RoomId: overworldId, RoomIds: map[int]struct{}{overworldId: {}}},
			"TestZone":  {Name: "TestZone", RoomId: ephEntry, RoomIds: map[int]struct{}{ephEntry: {}}},
		},
	)
	defer cleanupRooms()

	// Seed a real user inside the ephemeral room. NewTestUser sets
	// RoomId=1 by default; align it with the ephemeral room so the
	// MoveToRoom currentRoom lookup matches.
	const userId = 99
	u := users.NewTestUser(userId, "tester", "Tester", uint64(userId))
	u.Character.RoomId = ephEntry
	cleanupUsers := users.SeedUsersForTest(map[int]*users.UserRecord{userId: u})
	defer cleanupUsers()

	ephRoom.AddPlayer(userId)
	MarkRoomOccupancy(ephEntry, 1, 0)

	ir := NewInstanceRegistry()
	inst := &ZoneInstance{
		InstanceId:      ephEntry,
		TemplateZone:    "TestZone",
		AuthorizedUsers: []int{userId},
		OwnerUserId:     userId,
		CreatedRound:    0,
		PortalDuration:  "1m",
		OverworldRoomId: overworldId,
		EntryRoomId:     ephEntry,
		RoomIdMap: map[int]int{
			1: ephEntry,
		},
	}
	ir.Add(inst)

	ir.CheckPortalTimers()

	// Boot phase moved the player to the overworld room.
	assert.Equal(t, overworldId, u.Character.RoomId,
		"player should be moved out of expired ephemeral room")

	// roomsWithUsers no longer tracks the drained ephemeral room
	// (MoveToRoom deletes the entry when the last player leaves).
	assert.Equal(t, 0, roomManager.roomsWithUsers[ephEntry])

	// And the overworld destination now tracks the moved-in player.
	assert.Equal(t, 1, roomManager.roomsWithUsers[overworldId])

	// Instance is deregistered after the chain.
	assert.Nil(t, ir.FindByRoomId(ephEntry))
}

func TestCheckPortalTimers_NotExpiredNoOp(t *testing.T) {
	origEvictor := btreeStateEvictor
	defer SetBTreeStateEvictor(origEvictor)

	evicted := []int{}
	SetBTreeStateEvictor(func(roomId int) {
		evicted = append(evicted, roomId)
	})

	ir := NewInstanceRegistry()

	ephEntry := ephemeralRoomIdMinimum + 400
	inst := &ZoneInstance{
		InstanceId:      ephEntry,
		TemplateZone:    "TestZone",
		AuthorizedUsers: []int{},
		OwnerUserId:     0,
		CreatedRound:    util.GetRoundCount(),
		PortalDuration:  "999 real hours",
		OverworldRoomId: 1,
		EntryRoomId:     ephEntry,
		RoomIdMap: map[int]int{
			1: ephEntry,
		},
	}
	ir.Add(inst)

	ir.CheckPortalTimers()

	// Not-yet-expired instance is untouched: still findable, registry intact.
	assert.NotNil(t, ir.FindByRoomId(inst.EntryRoomId))
	assert.Len(t, ir.All(), 1)

	// No btree eviction calls — the chain only runs for expired instances.
	assert.Empty(t, evicted)
}

// ─── CreateZoneInstanceWithOpts: SuppressReturnPortal ────────────────────────

func TestCreateZoneInstanceWithOpts_SuppressReturnPortal(t *testing.T) {
	// ── Filesystem fixture ────────────────────────────────────────────────────
	// CreateEphemeralZone calls LoadRoomTemplate which reads YAML from disk.
	// We write a minimal room YAML into a temp dir and point DataFiles at it.
	const (
		zoneName    = "Test Instance Zone"
		entryRoomId = 9001
		overworldId = 1
		zoneFolder  = "test_instance_zone"
	)

	tempDir := t.TempDir()
	roomsDir := filepath.Join(tempDir, "rooms", zoneFolder)
	if err := os.MkdirAll(roomsDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	templateRoom := &Room{
		RoomId:      entryRoomId,
		Zone:        zoneName,
		Title:       "Test Cell",
		Description: "A bare cell used only in tests.",
		Exits:       map[string]exit.RoomExit{},
		SpawnInfo:   []SpawnInfo{},
	}
	roomYAML, err := yaml.Marshal(templateRoom)
	if err != nil {
		t.Fatalf("yaml.Marshal: %v", err)
	}
	roomFile := filepath.Join(roomsDir, "9001.yaml")
	if err := os.WriteFile(roomFile, roomYAML, 0644); err != nil {
		t.Fatalf("write room file: %v", err)
	}

	// Override the DataFiles config to the temp dir so LoadRoomTemplate finds
	// the file. Restore on test exit.
	prevDataFiles := configs.GetFilePathsConfig().DataFiles.String()
	if err := configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": tempDir}); err != nil {
		t.Fatalf("AddOverlayOverrides: %v", err)
	}
	defer func() {
		_ = configs.AddOverlayOverrides(map[string]any{"FilePaths.DataFiles": prevDataFiles})
	}()

	// ── Room manager fixture ───────────────────────────────────────────────────
	// SeedRoomsForTest replaces the global roomManager. The zone config tells
	// CreateZoneInstance which room IDs belong to the zone (for
	// CreateEphemeralZone) and what the entry room is.
	zCfg := &ZoneConfig{
		Name:           zoneName,
		Instanced:      true,
		EntryRoom:      entryRoomId,
		DeathPolicy:    "rejoin",
		PortalDuration: "30 real minutes",
		AllowRecall:    true,
		RoomIds:        map[int]struct{}{entryRoomId: {}},
	}
	cleanupRooms := SeedRoomsForTest(
		map[int]*Room{},
		map[string]*ZoneConfig{zoneName: zCfg},
	)
	defer cleanupRooms()

	// Seed the room path in the file cache so GetFilePath finds it without
	// a filesystem walk (the walk uses the configs.DataFiles path, which we
	// overrode, but the cache short-circuits it entirely).
	roomManager.roomIdToFileCache[entryRoomId] = filepath.Join(zoneFolder, "9001.yaml")
	defer delete(roomManager.roomIdToFileCache, entryRoomId)

	// Reset ephemeral state so the test is isolated and re-runnable.
	defer func() {
		for i := range ephemeralRoomChunks {
			ephemeralRoomChunks[i] = nil
		}
		originalRoomIdLookups = map[int]int{}
	}()

	// ── sub-case A: default CreateZoneInstance → return portal present ────────

	instA, err := CreateZoneInstance(zoneName, 100, 1, []int{1}, overworldId)
	if err != nil {
		t.Fatalf("CreateZoneInstance (default) failed: %v", err)
	}

	entryA := LoadRoom(instA.EntryRoomId)
	if entryA == nil {
		t.Fatalf("could not load ephemeral entry room %d", instA.EntryRoomId)
	}
	_, hasPortalA := entryA.ExitsTemp["return portal"]
	assert.True(t, hasPortalA, "default instance: entry room should have 'return portal' exit")

	// Tear down instance A so the ephemeral chunk is free for instance B.
	GetInstanceRegistry().Remove(instA)
	TryEphemeralCleanup(instA.EntryRoomId)

	// Re-seed the room cache entry (TryEphemeralCleanup evicts the room from
	// memory; the cache entry was deleted by removeRoomFromMemory).
	roomManager.roomIdToFileCache[entryRoomId] = filepath.Join(zoneFolder, "9001.yaml")

	// ── sub-case B: CreateZoneInstanceWithOpts(SuppressReturnPortal=true) ─────

	instB, err := CreateZoneInstanceWithOpts(
		zoneName, 100, 1, []int{1}, overworldId,
		ZoneInstanceOpts{SuppressReturnPortal: true},
	)
	if err != nil {
		t.Fatalf("CreateZoneInstanceWithOpts (SuppressReturnPortal) failed: %v", err)
	}

	entryB := LoadRoom(instB.EntryRoomId)
	if entryB == nil {
		t.Fatalf("could not load ephemeral entry room %d (suppressed)", instB.EntryRoomId)
	}
	_, hasPortalB := entryB.ExitsTemp["return portal"]
	assert.False(t, hasPortalB, "SuppressReturnPortal=true: entry room must NOT have a 'return portal' exit")

	// Tear down instance B.
	GetInstanceRegistry().Remove(instB)
	TryEphemeralCleanup(instB.EntryRoomId)
}
