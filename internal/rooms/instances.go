package rooms

import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ZoneInstance tracks an active instanced zone.
type ZoneInstance struct {
	InstanceId      int   // ephemeral chunk ID (any ephemeral room ID works)
	TemplateZone    string // zone name that was cloned
	GoldPaid        int   // gold amount (for future scaling system)
	AuthorizedUsers []int // userId snapshot at creation
	OwnerUserId     int   // who paid
	CreatedRound    uint64 // for timer tracking
	PortalDuration  string // from zone config (e.g. "30m")
	DeathPolicy     string // "rejoin" or "ejected"
	AllowRecall     bool   // from zone config
	OverworldRoomId int    // room where portal was created
	EntryRoomId     int    // ephemeral entry room ID
	RoomIdMap       map[int]int // original → ephemeral room ID mapping
}

// IsAuthorized checks if a user is on the access list.
func (zi *ZoneInstance) IsAuthorized(userId int) bool {
	for _, uid := range zi.AuthorizedUsers {
		if uid == userId {
			return true
		}
	}
	return false
}

// RevokeAccess removes a user from the authorized list.
// Used by "ejected" death policy.
func (zi *ZoneInstance) RevokeAccess(userId int) {
	filtered := make([]int, 0, len(zi.AuthorizedUsers))
	for _, uid := range zi.AuthorizedUsers {
		if uid != userId {
			filtered = append(filtered, uid)
		}
	}
	zi.AuthorizedUsers = filtered
}

// InstanceRegistry is a thread-safe registry of active zone instances.
type InstanceRegistry struct {
	mu        sync.RWMutex
	instances []*ZoneInstance
	roomIndex map[int]*ZoneInstance // ephemeral roomId → instance
}

// NewInstanceRegistry creates a new instance registry.
func NewInstanceRegistry() *InstanceRegistry {
	return &InstanceRegistry{
		instances: make([]*ZoneInstance, 0),
		roomIndex: make(map[int]*ZoneInstance),
	}
}

// Add registers an instance and indexes all of its room IDs.
func (ir *InstanceRegistry) Add(inst *ZoneInstance) {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	ir.instances = append(ir.instances, inst)

	// Index the entry room
	ir.roomIndex[inst.EntryRoomId] = inst

	// Index all room IDs from the map
	for _, ephemeralId := range inst.RoomIdMap {
		ir.roomIndex[ephemeralId] = inst
	}
}

// Remove deregisters an instance and cleans up the index.
func (ir *InstanceRegistry) Remove(inst *ZoneInstance) {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	// Remove from instances slice
	filtered := make([]*ZoneInstance, 0, len(ir.instances))
	for _, i := range ir.instances {
		if i != inst {
			filtered = append(filtered, i)
		}
	}
	ir.instances = filtered

	// Clean up room index
	delete(ir.roomIndex, inst.EntryRoomId)
	for _, ephemeralId := range inst.RoomIdMap {
		delete(ir.roomIndex, ephemeralId)
	}
}

// FindByRoomId looks up an instance by any of its ephemeral room IDs.
// Returns nil if not found.
func (ir *InstanceRegistry) FindByRoomId(roomId int) *ZoneInstance {
	ir.mu.RLock()
	defer ir.mu.RUnlock()

	return ir.roomIndex[roomId]
}

// All returns a snapshot of all active instances.
func (ir *InstanceRegistry) All() []*ZoneInstance {
	ir.mu.RLock()
	defer ir.mu.RUnlock()

	result := make([]*ZoneInstance, len(ir.instances))
	copy(result, ir.instances)
	return result
}

// CheckPortalTimers runs the per-tick instance lifecycle pass:
//   - Broadcasts 5-minute and 1-minute warning messages to players
//     inside instances whose portal is about to expire.
//   - On TTL expiry, runs the consolidated cleanup chain: boot any
//     remaining players to OverworldRoomId with a flavor message,
//     deregister the instance, evict each ephemeral room's btree
//     state via the registered callback, and free the ephemeral
//     chunk via TryEphemeralCleanup.
//
// Runs inside util.LockMud() (called from world.go's per-tick loop),
// so concurrent player movement is serialized against the chain.
//
// The TryEphemeralCleanup call here overlaps with the existing
// RoomChange_CleanupEphemeralRooms hook (which fires when a player
// leaves an ephemeral room with no remaining players). Both paths
// are correct: the hook handles "last player left, no TTL yet"
// (typical), and this TTL chain handles "TTL expired, players may or
// may not still be inside" (the leak case). The function self-
// protects against double-free via its instance-active and
// players-present guards.
func (ir *InstanceRegistry) CheckPortalTimers() {
	// Phase A: under RLock, snapshot the expired instances and emit
	// the 5-minute / 1-minute warnings for in-TTL ones.
	ir.mu.RLock()

	if len(ir.instances) == 0 {
		ir.mu.RUnlock()
		return
	}

	c := configs.GetConfig()
	currentRound := util.GetRoundCount()

	fiveMinRounds := uint64(c.Timing.MinutesToRounds(5))
	oneMinRounds := uint64(c.Timing.MinutesToRounds(1))

	var expired []*ZoneInstance
	for _, inst := range ir.instances {
		if inst.PortalDuration == `` {
			continue
		}

		g := gametime.GetDate(inst.CreatedRound)
		expiryRound := g.AddPeriod(inst.PortalDuration)

		if currentRound >= expiryRound {
			expired = append(expired, inst)
			continue
		}

		remainingRounds := expiryRound - currentRound

		var msg string
		switch remainingRounds {
		case fiveMinRounds:
			msg = `<ansi fg="yellow">The portal flickers — it won't hold much longer.</ansi>`
		case oneMinRounds:
			msg = `<ansi fg="red">The portal is barely a shimmer now. Leave soon or find your own way out.</ansi>`
		}

		if msg == `` {
			continue
		}

		for _, ephId := range inst.RoomIdMap {
			if room := LoadRoom(ephId); room != nil {
				room.SendText(messaging.CategoryWarning, msg)
			}
		}
	}
	// Drop the RLock BEFORE Phase B — Remove(inst) takes its own
	// write lock, which would deadlock against an outer RLock.
	ir.mu.RUnlock()

	// Phase B: process each expired instance OUTSIDE the RLock.
	const collapseMsg = `<ansi fg="red">The portal's shimmer collapses around you — the instance unravels.</ansi>`
	for _, inst := range expired {
		// 1. Boot phase — O(1) populated-room check via roomsWithUsers.
		//    Send the flavor message BEFORE MoveToRoom so players see
		//    it in the ephemeral room context, not after teleport.
		for _, ephId := range inst.RoomIdMap {
			if roomManager.roomsWithUsers[ephId] == 0 {
				continue
			}
			room := LoadRoom(ephId)
			if room == nil {
				continue
			}
			for _, userId := range room.GetPlayers() {
				if u := users.GetByUserId(userId); u != nil {
					u.SendText(messaging.CategoryWarning, collapseMsg)
				}
				MoveToRoom(userId, inst.OverworldRoomId)
			}
		}

		// 2. Deregister phase (Remove takes its own ir.mu.Lock).
		ir.Remove(inst)

		// 3. Btree eviction phase — callback-mediated to avoid an
		//    internal/rooms → internal/behaviortree import cycle.
		//    inst.EntryRoomId is stored in RoomIdMap as a self-mapping
		//    by CreateZoneInstance, so iterating RoomIdMap covers it.
		if btreeStateEvictor != nil {
			for _, ephId := range inst.RoomIdMap {
				btreeStateEvictor(ephId)
			}
		}

		// 4. Ephemeral chunk free phase. TryEphemeralCleanup self-
		//    protects (returns []int{} if any room has players or an
		//    active instance). After steps 1+2 both gates are clear.
		TryEphemeralCleanup(inst.EntryRoomId)
	}
}

// Package-level singleton.
var instanceRegistry = NewInstanceRegistry()

// btreeStateEvictor is set at startup by main.go to wire the
// rooms→behaviortree dependency direction without an import cycle.
// nil-safe: a nil evictor is a no-op (used in tests that don't care
// about btree state).
var btreeStateEvictor func(roomId int)

// SetBTreeStateEvictor registers the per-room btree state eviction
// callback. Called once at startup from main.go with
// behaviortree.EvictRoomBTreeState. Safe to leave unregistered in
// tests that don't exercise btree state.
func SetBTreeStateEvictor(fn func(int)) {
	btreeStateEvictor = fn
}

// GetInstanceRegistry returns the global instance registry.
func GetInstanceRegistry() *InstanceRegistry {
	return instanceRegistry
}

// ScaleSpawnStatPools multiplies each spawn's StatPool by goldPaid.
// Template stat pools act as multipliers (1=trash, 2=tough, 3=boss).
// Spawns with StatPool 0 default to 1x. Cap of 0 means uncapped.
func ScaleSpawnStatPools(spawns []SpawnInfo, goldPaid int, cap int) {
	for i := range spawns {
		mult := spawns[i].StatPool
		if mult < 1 {
			mult = 1
		}
		scaled := goldPaid * mult
		if cap > 0 && scaled > cap {
			scaled = cap
		}
		spawns[i].StatPool = scaled
	}
}

// ZoneInstanceOpts holds optional behavioural overrides for CreateZoneInstanceWithOpts.
// The zero value reproduces the behaviour of the plain CreateZoneInstance call.
type ZoneInstanceOpts struct {
	// SuppressReturnPortal, when true, skips adding the "return portal"
	// temporary exit to the entry room. Use this for jail cells or any
	// instanced room that must have no escape exit.
	SuppressReturnPortal bool
}

// CreateZoneInstanceWithOpts is identical to CreateZoneInstance but accepts an
// additional ZoneInstanceOpts argument that lets callers suppress optional
// behaviours (e.g. the return-portal exit). Existing callers that invoke
// CreateZoneInstance are unaffected — that function delegates here with a
// zero-value opts struct, which preserves the original behaviour.
func CreateZoneInstanceWithOpts(
	zoneName string,
	goldPaid int,
	ownerUserId int,
	authorizedUsers []int,
	overworldRoomId int,
	opts ZoneInstanceOpts,
) (*ZoneInstance, error) {

	// 1. Look up and validate the zone config.
	zCfg := GetZoneConfig(zoneName)
	if zCfg == nil {
		return nil, fmt.Errorf("CreateZoneInstance: zone %q not found", zoneName)
	}
	if !zCfg.Instanced {
		return nil, fmt.Errorf("CreateZoneInstance: zone %q is not marked as instanced", zoneName)
	}

	// 2. Clone the zone into ephemeral rooms.
	//    For the oasis zone, clone only the entry room and generate the
	//    cube procedurally. Other zones use the standard full-zone clone.
	var roomIdMap map[int]int
	var ephemeralEntryId int
	var err error

	if zoneName == "Instance Planar Oasis" {
		// Clone only the entry room (Oasis Threshold).
		roomIdMap, err = CreateEphemeralRoomIds(zCfg.EntryRoom)
		if err != nil {
			return nil, fmt.Errorf("CreateZoneInstance: failed to clone entry room for %q: %w", zoneName, err)
		}

		var ok bool
		ephemeralEntryId, ok = roomIdMap[zCfg.EntryRoom]
		if !ok {
			return nil, fmt.Errorf("CreateZoneInstance: entry room %d not in cloned rooms for %q", zCfg.EntryRoom, zoneName)
		}

		// Remove the template north exit (pointed to room 5004) since the
		// cube generator will add a temporary north exit to the cube entry.
		entryRoom := LoadRoom(ephemeralEntryId)
		if entryRoom != nil {
			delete(entryRoom.Exits, "north")
		}

		// Generate the 5x5x5 wrapping cube.
		cubeRoomIds, _, cubeErr := GenerateOasisCube(
			ephemeralEntryId,
			zoneName,
			goldPaid,
			ephemeralEntryId, // instanceId
			zCfg.AllowRecall,
			zCfg.DeathPolicy,
		)
		if cubeErr != nil {
			return nil, fmt.Errorf("CreateZoneInstance: cube generation failed for %q: %w", zoneName, cubeErr)
		}

		// Add all cube rooms to the roomIdMap so the instance registry
		// indexes them and cleanup works.
		for _, cubeId := range cubeRoomIds {
			roomIdMap[cubeId] = cubeId
		}
	} else {
		roomIdMap, err = CreateEphemeralZone(zoneName)
		if err != nil {
			return nil, fmt.Errorf("CreateZoneInstance: failed to clone zone %q: %w", zoneName, err)
		}

		var ok bool
		ephemeralEntryId, ok = roomIdMap[zCfg.EntryRoom]
		if !ok {
			return nil, fmt.Errorf("CreateZoneInstance: entry room %d not in cloned zone %q", zCfg.EntryRoom, zoneName)
		}
	}

	// 3. Build the ZoneInstance struct.
	inst := &ZoneInstance{
		InstanceId:      ephemeralEntryId,
		TemplateZone:    zoneName,
		GoldPaid:        goldPaid,
		AuthorizedUsers: authorizedUsers,
		OwnerUserId:     ownerUserId,
		CreatedRound:    util.GetRoundCount(),
		PortalDuration:  zCfg.PortalDuration,
		DeathPolicy:     zCfg.DeathPolicy,
		AllowRecall:     zCfg.AllowRecall,
		OverworldRoomId: overworldRoomId,
		EntryRoomId:     ephemeralEntryId,
		RoomIdMap:       roomIdMap,
	}

	// 4. Add a return portal in the ephemeral entry room pointing back to the
	//    overworld. Skipped when opts.SuppressReturnPortal is true (e.g. jail
	//    cells that must not have an escape exit). Expires is set very long;
	//    actual cleanup is handled by the instance cleanup system when all
	//    players leave.
	if !opts.SuppressReturnPortal {
		entryRoom := LoadRoom(ephemeralEntryId)
		if entryRoom == nil {
			return nil, fmt.Errorf("CreateZoneInstance: could not load ephemeral entry room %d", ephemeralEntryId)
		}

		returnExit := exit.TemporaryRoomExit{
			RoomId:       overworldRoomId,
			Title:        "Return Portal",
			UserId:       0, // system-created
			SpawnedRound: inst.CreatedRound,
			Expires:      "999 real hours",
		}
		entryRoom.AddTemporaryExit("return portal", returnExit)
	}

	// 5. Scale mob stat pools based on gold paid.
	cap := int(configs.GetBalanceConfig().InstanceStatPoolCap)
	for _, ephId := range roomIdMap {
		if room := LoadRoom(ephId); room != nil {
			ScaleSpawnStatPools(room.SpawnInfo, goldPaid, cap)
		}
	}

	// 6. Stamp instance metadata on every ephemeral room for scripting access.
	for _, ephemeralId := range roomIdMap {
		room := LoadRoom(ephemeralId)
		if room == nil {
			continue
		}
		room.SetTempData("instance_id", inst.InstanceId)
		room.SetTempData("allow_recall", inst.AllowRecall)
		room.SetTempData("death_policy", inst.DeathPolicy)
		room.SetTempData("gold_paid", inst.GoldPaid)
	}

	// 7. Register the instance in the global registry.
	instanceRegistry.Add(inst)

	return inst, nil
}

// CreateZoneInstance clones a zone template into ephemeral rooms, wires up a
// return portal in the entry room, stamps instance metadata on every ephemeral
// room, and registers the instance in the global registry.
//
// This is a convenience wrapper around CreateZoneInstanceWithOpts with a
// zero-value opts struct, preserving all prior behaviour for existing callers.
func CreateZoneInstance(
	zoneName string,
	goldPaid int,
	ownerUserId int,
	authorizedUsers []int,
	overworldRoomId int,
) (*ZoneInstance, error) {
	return CreateZoneInstanceWithOpts(zoneName, goldPaid, ownerUserId, authorizedUsers, overworldRoomId, ZoneInstanceOpts{})
}
