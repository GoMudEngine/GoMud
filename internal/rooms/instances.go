package rooms

import (
	"sync"
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

// Package-level singleton.
var instanceRegistry = NewInstanceRegistry()

// GetInstanceRegistry returns the global instance registry.
func GetInstanceRegistry() *InstanceRegistry {
	return instanceRegistry
}
