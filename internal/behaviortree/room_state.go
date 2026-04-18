package behaviortree

import "sync"

var (
	roomStateMu sync.RWMutex
	roomStates  = make(map[int]*BehaviorState)
)

// EnsureRoomBTreeState lazily initializes and returns the BehaviorState for a
// room. State persists for the lifetime of the process (rooms don't respawn).
func EnsureRoomBTreeState(roomId int) *BehaviorState {
	roomStateMu.RLock()
	state, ok := roomStates[roomId]
	roomStateMu.RUnlock()
	if ok && state != nil {
		return state
	}
	roomStateMu.Lock()
	defer roomStateMu.Unlock()
	if state, ok = roomStates[roomId]; ok && state != nil {
		return state
	}
	state = NewBehaviorState()
	roomStates[roomId] = state
	return state
}

// EvictRoomBTreeState removes the cached BehaviorState for the given room.
// No-op if the room has no cached state. The next EnsureRoomBTreeState call
// for this roomId will allocate a fresh BehaviorState.
//
// Intended caller: rooms-package teardown for ephemeral zone instances
// (e.g., InstanceRegistry.Remove for portal-spawned rooms). Static rooms
// must NOT be evicted — their state is meant to live for the process
// lifetime; eviction loses any accumulated state (cooldown counters,
// delay rounds, etc.).
//
// 1.8 ships only the API. Wire-up from the rooms package is captured in
// project_rooms_package_audit_needed.md and belongs to a future rooms-
// package audit pass.
func EvictRoomBTreeState(roomId int) {
	roomStateMu.Lock()
	defer roomStateMu.Unlock()
	delete(roomStates, roomId)
}
