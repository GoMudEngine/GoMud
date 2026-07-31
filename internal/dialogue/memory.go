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
	Greeted         bool // this mob instance has greeted this player (5a; in-process, so once per boot)
	RecentTopics    []string
}

// memoryCache keys: (mobInstanceId << 32 | uint32(userId)) → *PlayerMemory
var memoryCache = map[uint64]*PlayerMemory{}

// memorySweepIdleRounds is how long a (mob instance, player) pair may go
// untouched before its conversation memory is dropped by SweepMemories.
//
// The cache grows by one entry per pair that has ever spoken and, before this
// existed, was only ever emptied by an explicit ResetMemory — so it grew
// monotonically with uptime and player count, holding entries for mob instances
// that despawned days earlier. The state is already per-instance and lost on
// restart, so dropping an idle entry costs nothing a respawn would not.
const memorySweepIdleRounds uint64 = 20000

func memKey(mobInstanceId, userId int) uint64 {
	return uint64(mobInstanceId)<<32 | uint64(uint32(userId))
}

// SweepMemories drops conversation memories untouched for longer than
// memorySweepIdleRounds and returns how many were removed. Safe to call on a
// timer; entries still in use are refreshed by UpdateMemory.
func SweepMemories() int {
	now := util.GetRoundCount()
	removed := 0

	for key, mem := range memoryCache {
		// LastVisitRound == 0 means the entry was created by GetMemory but no
		// node has been visited yet. Leave it; it is one round old at most.
		if mem.LastVisitRound == 0 {
			continue
		}
		if now-mem.LastVisitRound > memorySweepIdleRounds {
			delete(memoryCache, key)
			removed++
		}
	}

	return removed
}

// ForgetMobInstance drops every player's memory of one mob instance. Call it
// when a mob despawns — the instance id may be reused by a later spawn, and a
// new mob should not inherit a stranger's conversation history.
func ForgetMobInstance(mobInstanceId int) int {
	removed := 0
	prefix := uint64(mobInstanceId) << 32

	for key := range memoryCache {
		if key&0xFFFFFFFF00000000 == prefix {
			delete(memoryCache, key)
			removed++
		}
	}

	return removed
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
	delta := gametime.PeriodLength(expiryPeriod)
	if delta == 0 {
		return false
	}
	return util.GetRoundCount()-mem.LastVisitRound > delta
}

// ResetMemory wipes a player's state for the given mob instance.
func ResetMemory(mobInstanceId, userId int) {
	key := memKey(mobInstanceId, userId)
	delete(memoryCache, key)
}
