package mobs

// CombatMemory tracks who the mob was fighting across flee/re-engage cycles.
type CombatMemory struct {
	TargetUserId   int    // Player they were fighting
	TargetMobId    int    // Or mob they were fighting
	LastSeenRoomId int    // Where the target was last seen
	LastSeenRound  uint64 // When they last saw the target
	Grudge         bool   // Should they pursue?
}

// SetCombatMemory creates a new CombatMemory for a mob that just entered combat.
func SetCombatMemory(targetUserId int, targetMobId int, roomId int, round uint64) *CombatMemory {
	return &CombatMemory{
		TargetUserId:   targetUserId,
		TargetMobId:    targetMobId,
		LastSeenRoomId: roomId,
		LastSeenRound:  round,
		Grudge:         true,
	}
}

// CombatMemoryExpired returns true if the memory has expired based on round count.
func CombatMemoryExpired(mem *CombatMemory, currentRound uint64, maxDuration int) bool {
	if mem == nil {
		return true
	}
	return currentRound-mem.LastSeenRound > uint64(maxDuration)
}

// UpdateCombatMemoryLocation updates where and when the target was last seen.
func UpdateCombatMemoryLocation(mem *CombatMemory, roomId int, round uint64) {
	if mem == nil {
		return
	}
	mem.LastSeenRoomId = roomId
	mem.LastSeenRound = round
}
