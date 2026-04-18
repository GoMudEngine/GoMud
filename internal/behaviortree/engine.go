package behaviortree

import (
	"sync"
	"time"
)

// Engine manages behavior tree loading, caching, and evaluation.
type Engine struct {
	mu         sync.RWMutex
	trees      map[int]Node // mobId → compiled root node
	noTree     map[int]bool // mobId → no behavior file exists on disk
	roomTrees  map[int]Node // roomId → compiled root node
	noRoomTree map[int]bool // roomId → no behavior file exists on disk
	queue      []DelayedAction
}

type DelayedAction struct {
	ExecuteAt time.Time
	Action    func()
}

var globalEngine *Engine

func init() {
	globalEngine = &Engine{
		trees:      make(map[int]Node),
		noTree:     make(map[int]bool),
		roomTrees:  make(map[int]Node),
		noRoomTree: make(map[int]bool),
	}
}

func GetEngine() *Engine {
	return globalEngine
}

// LoadTree loads and caches a behavior tree for a mob type.
// Clears any negative-cache entry so newly added files are picked up at runtime.
func (e *Engine) LoadTree(mobId int, path string) error {
	node, err := LoadTreeFromFile(path)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.trees[mobId] = node
	delete(e.noTree, mobId)
	e.mu.Unlock()
	return nil
}

// Negative cache (noTree / noRoomTree) — design note.
//
// The negative cache records mob/room ids whose behavior tree YAML does
// not exist on disk (or whose load failed at file-stat time). Once set,
// an entry only clears on a successful subsequent LoadTree / LoadRoomTree.
//
// This is correct ONLY because behavior tree files are static for the
// process lifetime — there is no hot-reload of YAML on disk change. If
// hot-reload is ever added, the negative cache becomes a stale-cache bug:
// a file appearing on disk after the negative entry is set will be
// invisible until the engine restarts.
//
// TODO(hot-reload): bust cache on file change if/when hot-reload is added.

// HasNoTree reports whether the negative cache has recorded that mobId has no
// behavior tree file on disk. Callers should check this before os.Stat.
func (e *Engine) HasNoTree(mobId int) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.noTree[mobId]
}

// SetNoTree records that mobId has no behavior tree file on disk, suppressing
// future os.Stat calls for this template until the engine is restarted or a
// tree is successfully loaded via LoadTree.
func (e *Engine) SetNoTree(mobId int) {
	e.mu.Lock()
	e.noTree[mobId] = true
	e.mu.Unlock()
}

// GetTree returns the cached tree for a mob type, or nil.
func (e *Engine) GetTree(mobId int) Node {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.trees[mobId]
}

// LoadRoomTree loads and caches a behavior tree for a room.
// Clears any negative-cache entry so newly added files are picked up at runtime.
func (e *Engine) LoadRoomTree(roomId int, path string) error {
	node, err := LoadTreeFromFile(path)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.roomTrees[roomId] = node
	delete(e.noRoomTree, roomId)
	e.mu.Unlock()
	return nil
}

// HasNoRoomTree reports whether the negative cache has recorded that roomId has
// no behavior tree file on disk. Callers should check this before os.Stat.
func (e *Engine) HasNoRoomTree(roomId int) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.noRoomTree[roomId]
}

// SetNoRoomTree records that roomId has no behavior tree file on disk,
// suppressing future os.Stat calls until the engine is restarted or a tree is
// successfully loaded via LoadRoomTree.
func (e *Engine) SetNoRoomTree(roomId int) {
	e.mu.Lock()
	e.noRoomTree[roomId] = true
	e.mu.Unlock()
}

// GetRoomTree returns the cached tree for a room, or nil.
func (e *Engine) GetRoomTree(roomId int) Node {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.roomTrees[roomId]
}

// EvaluateEvent triggers immediate tree evaluation for a mob instance.
func (e *Engine) EvaluateEvent(mobId int, instanceId int, event EventContext, state *BehaviorState) {
	tree := e.GetTree(mobId)
	if tree == nil {
		return
	}
	ctx := &EvalContext{
		Event:      event,
		MobState:   state,
		MobId:      mobId,
		InstanceId: instanceId,
		RoomId:     event.RoomId,
	}
	tree.Evaluate(ctx)
}

// QueueDelayed adds an action to execute after a delay.
func (e *Engine) QueueDelayed(delay time.Duration, action func()) {
	e.mu.Lock()
	e.queue = append(e.queue, DelayedAction{
		ExecuteAt: time.Now().Add(delay),
		Action:    action,
	})
	e.mu.Unlock()
}

// DrainQueue executes all delayed actions whose time has come.
// Called once per round tick.
func (e *Engine) DrainQueue() {
	e.mu.Lock()
	now := time.Now()
	remaining := make([]DelayedAction, 0, len(e.queue))
	var ready []DelayedAction
	for _, da := range e.queue {
		if now.After(da.ExecuteAt) || now.Equal(da.ExecuteAt) {
			ready = append(ready, da)
		} else {
			remaining = append(remaining, da)
		}
	}
	e.queue = remaining
	e.mu.Unlock()

	for _, da := range ready {
		da.Action()
	}
}
