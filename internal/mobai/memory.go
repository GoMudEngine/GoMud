package mobai

// SetMemory creates a new CombatMemory for a mob that just entered combat.
func SetMemory(targetUserId int, targetMobId int, roomId int, round uint64) *CombatMemory {
	return &CombatMemory{
		TargetUserId:   targetUserId,
		TargetMobId:    targetMobId,
		LastSeenRoomId: roomId,
		LastSeenRound:  round,
		Grudge:         true,
	}
}

// MemoryExpired returns true if the memory has expired based on round count.
func MemoryExpired(mem *CombatMemory, currentRound uint64, maxDuration int) bool {
	if mem == nil {
		return true
	}
	return currentRound-mem.LastSeenRound > uint64(maxDuration)
}

// UpdateMemoryLocation updates where and when the target was last seen.
func UpdateMemoryLocation(mem *CombatMemory, roomId int, round uint64) {
	if mem == nil {
		return
	}
	mem.LastSeenRoomId = roomId
	mem.LastSeenRound = round
}
