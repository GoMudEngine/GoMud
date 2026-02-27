package dialogue

import (
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// PlayerMemory tracks per-player conversation state for a specific mob instance.
type PlayerMemory struct {
	VisitCount      int
	LastVisitRound  uint64
	UnlockedNodes   map[string]bool
	CurrentRootSeen bool
	RecentTopics    []string
}

// memoryCache keys: (mobInstanceId << 32 | uint32(userId)) → *PlayerMemory
var memoryCache = map[uint64]*PlayerMemory{}

func memKey(mobInstanceId, userId int) uint64 {
	return uint64(mobInstanceId)<<32 | uint64(uint32(userId))
}

// GetMemory returns the player's memory for this mob instance, creating it if absent.
func GetMemory(mobInstanceId, userId int) *PlayerMemory {
	key := memKey(mobInstanceId, userId)
	if mem, ok := memoryCache[key]; ok {
		return mem
	}
	mem := &PlayerMemory{
		UnlockedNodes: map[string]bool{},
	}
	memoryCache[key] = mem
	return mem
}

// UpdateMemory records a successful node visit and updates unlocks and topic history.
func UpdateMemory(mobInstanceId, userId int, nodeId string, unlocks []string, topic string) {
	mem := GetMemory(mobInstanceId, userId)
	mem.VisitCount++
	mem.LastVisitRound = util.GetRoundCount()
	if nodeId != "" {
		mem.UnlockedNodes[nodeId] = true
	}
	for _, u := range unlocks {
		mem.UnlockedNodes[u] = true
	}
	if topic != "" {
		mem.RecentTopics = append(mem.RecentTopics, topic)
		if len(mem.RecentTopics) > 5 {
			mem.RecentTopics = mem.RecentTopics[len(mem.RecentTopics)-5:]
		}
	}
}

// IsExpired returns true if enough game-time has elapsed to expire this memory.
func IsExpired(mem *PlayerMemory, expiryPeriod string) bool {
	if expiryPeriod == "" || mem.LastVisitRound == 0 {
		return false
	}
	baseline := gametime.GetDate(1000000)
	expiryRound := baseline.AddPeriod(expiryPeriod)
	delta := expiryRound - 1000000
	return util.GetRoundCount()-mem.LastVisitRound > delta
}

// ResetMemory wipes a player's state for the given mob instance.
func ResetMemory(mobInstanceId, userId int) {
	key := memKey(mobInstanceId, userId)
	delete(memoryCache, key)
}
