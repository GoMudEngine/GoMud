package rooms

import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/gametime"
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

// CleanupEmptyInstances removes registry entries for instances whose
// ephemeral rooms have been destroyed, and removes their overworld portals.
func (ir *InstanceRegistry) CleanupEmptyInstances() {
	ir.mu.Lock()
	defer ir.mu.Unlock()

	remaining := make([]*ZoneInstance, 0, len(ir.instances))
	for _, inst := range ir.instances {
		// Check if any room in the instance still exists in memory
		alive := false
		for _, ephId := range inst.RoomIdMap {
			if LoadRoom(ephId) != nil {
				alive = true
				break
			}
		}
		if alive {
			remaining = append(remaining, inst)
		} else {
			// Clean up room index entries
			for _, ephId := range inst.RoomIdMap {
				delete(ir.roomIndex, ephId)
			}
			// Remove the entry portal from the overworld room
			if owRoom := LoadRoom(inst.OverworldRoomId); owRoom != nil {
				for k, v := range owRoom.ExitsTemp {
					if v.RoomId == inst.EntryRoomId {
						delete(owRoom.ExitsTemp, k)
						break
					}
				}
			}
		}
	}
	ir.instances = remaining
}

// CheckPortalTimers broadcasts warning messages to players inside instances
// when the portal timer is running low (5 minutes and 1 minute remaining).
func (ir *InstanceRegistry) CheckPortalTimers() {
	ir.mu.RLock()
	defer ir.mu.RUnlock()

	if len(ir.instances) == 0 {
		return
	}

	c := configs.GetConfig()
	currentRound := util.GetRoundCount()

	fiveMinRounds := uint64(c.Timing.MinutesToRounds(5))
	oneMinRounds := uint64(c.Timing.MinutesToRounds(1))

	for _, inst := range ir.instances {
		if inst.PortalDuration == `` {
			continue
		}

		g := gametime.GetDate(inst.CreatedRound)
		expiryRound := g.AddPeriod(inst.PortalDuration)

		if currentRound >= expiryRound {
			continue // already expired — cleanup handled elsewhere
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
				room.SendText(msg)
			}
		}
	}
}

// Package-level singleton.
var instanceRegistry = NewInstanceRegistry()

// GetInstanceRegistry returns the global instance registry.
func GetInstanceRegistry() *InstanceRegistry {
	return instanceRegistry
}

// CreateZoneInstance clones a zone template into ephemeral rooms, wires up a
// return portal in the entry room, stamps instance metadata on every ephemeral
// room, and registers the instance in the global registry.
func CreateZoneInstance(
	zoneName string,
	goldPaid int,
	ownerUserId int,
	authorizedUsers []int,
	overworldRoomId int,
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
	roomIdMap, err := CreateEphemeralZone(zoneName)
	if err != nil {
		return nil, fmt.Errorf("CreateZoneInstance: failed to clone zone %q: %w", zoneName, err)
	}

	// 3. Resolve the ephemeral entry room.
	ephemeralEntryId, ok := roomIdMap[zCfg.EntryRoom]
	if !ok {
		return nil, fmt.Errorf("CreateZoneInstance: entry room %d not in cloned zone %q", zCfg.EntryRoom, zoneName)
	}

	// 4. Build the ZoneInstance struct.
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

	// 5. Add a return portal in the ephemeral entry room pointing back to the
	//    overworld. Expires is set very long; actual cleanup is handled by
	//    the instance cleanup system when all players leave.
	entryRoom := LoadRoom(ephemeralEntryId)
	if entryRoom == nil {
		return nil, fmt.Errorf("CreateZoneInstance: could not load ephemeral entry room %d", ephemeralEntryId)
	}

	returnExit := exit.TemporaryRoomExit{
		RoomId:       overworldRoomId,
		Title:        "Return Portal",
		UserId:       0, // system-created
		SpawnedRound: inst.CreatedRound,
		Expires:      "999h",
	}
	entryRoom.AddTemporaryExit("return portal", returnExit)

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
