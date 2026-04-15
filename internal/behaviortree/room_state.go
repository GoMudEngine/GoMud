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
