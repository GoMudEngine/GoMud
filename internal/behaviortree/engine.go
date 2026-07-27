package behaviortree

import (
	"fmt"
	"sync"
	"time"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// Engine manages behavior tree loading, caching, and evaluation.
type Engine struct {
	mu                    sync.RWMutex
	trees                 map[int]Node                  // mobId → compiled root node
	noTree                map[int]bool                  // mobId → no behavior file exists on disk
	roomTrees             map[int]Node                  // roomId → compiled root node
	noRoomTree            map[int]bool                  // roomId → no behavior file exists on disk
	archetypes            map[string]Node               // archetype name → compiled root node
	noArchetype           map[string]bool               // archetype name → no archetype file exists on disk
	archetypeGoalWeights  map[string]map[string]float64 // chunk 4.2 — per-archetype goal-type weight multipliers
	archetypeDefaultGoals map[string][]GoalDefault      // chunk 4.3 — per-archetype default goals
	queue                 []DelayedAction
}

type DelayedAction struct {
	ExecuteAt time.Time
	Action    func()
}

var globalEngine *Engine

func init() {
	globalEngine = &Engine{
		trees:                 make(map[int]Node),
		noTree:                make(map[int]bool),
		roomTrees:             make(map[int]Node),
		noRoomTree:            make(map[int]bool),
		archetypes:            make(map[string]Node),
		noArchetype:           make(map[string]bool),
		archetypeGoalWeights:  make(map[string]map[string]float64),
		archetypeDefaultGoals: make(map[string][]GoalDefault),
	}
	// Register the attack rejection callback so FireAttackRejected can fire btree events
	mobs.AttackRejectedTryMobBehavior = func(mobInstanceId int, ctx mobs.EventContext) bool {
		return TryMobBehavior(mobInstanceId, EventContext{
			EventType: ctx.EventType,
			UserId:    ctx.UserId,
			MobId:     ctx.MobId,
		})
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

// LoadArchetype loads and caches a behavior tree (chunk-4.2
// goal_weights map, and chunk-4.3 default_goals list) for an
// archetype name. Clears any negative-cache entry so newly added
// files are picked up at runtime.
func (e *Engine) LoadArchetype(name string, path string) error {
	tree, weights, defaults, err := LoadArchetypeYAMLFromFile(path)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.archetypes[name] = tree
	if e.archetypeGoalWeights == nil {
		e.archetypeGoalWeights = map[string]map[string]float64{}
	}
	e.archetypeGoalWeights[name] = weights
	if e.archetypeDefaultGoals == nil {
		e.archetypeDefaultGoals = map[string][]GoalDefault{}
	}
	e.archetypeDefaultGoals[name] = defaults
	delete(e.noArchetype, name)
	e.mu.Unlock()
	return nil
}

// EvictArchetype removes an archetype from every cache and sets the
// negative entry — the delete-side of the 5d editor's hot-reload contract.
func (e *Engine) EvictArchetype(name string) {
	e.mu.Lock()
	delete(e.archetypes, name)
	delete(e.archetypeGoalWeights, name)
	delete(e.archetypeDefaultGoals, name)
	e.noArchetype[name] = true
	e.mu.Unlock()
}

// EvictTree removes a per-mob tree from the cache and sets the negative —
// after a delete, "no tree" is the truth (archetype fallback).
func (e *Engine) EvictTree(mobId int) {
	e.mu.Lock()
	delete(e.trees, mobId)
	e.noTree[mobId] = true
	e.mu.Unlock()
}

// EvictRoomTree removes a room tree from the cache and sets the negative.
func (e *Engine) EvictRoomTree(roomId int) {
	e.mu.Lock()
	delete(e.roomTrees, roomId)
	e.noRoomTree[roomId] = true
	e.mu.Unlock()
}

// GetArchetypeGoalWeights returns the cached goal_weights map for the
// named archetype, or an empty map if the archetype is unknown or
// declared no weights. Safe to call from any goroutine; returns a
// copy so callers can't mutate the cache.
func (e *Engine) GetArchetypeGoalWeights(name string) map[string]float64 {
	e.mu.RLock()
	defer e.mu.RUnlock()
	raw := e.archetypeGoalWeights[name]
	if len(raw) == 0 {
		return map[string]float64{}
	}
	out := make(map[string]float64, len(raw))
	for k, v := range raw {
		out[k] = v
	}
	return out
}

// GetArchetypeDefaultGoals returns the cached default_goals list for
// the named archetype, or an empty slice if the archetype is unknown
// or declared no defaults. Returns a shallow copy. Chunk 4.3.
func (e *Engine) GetArchetypeDefaultGoals(name string) []GoalDefault {
	e.mu.RLock()
	defer e.mu.RUnlock()
	raw := e.archetypeDefaultGoals[name]
	if len(raw) == 0 {
		return []GoalDefault{}
	}
	out := make([]GoalDefault, len(raw))
	copy(out, raw)
	return out
}

// HasNoArchetype reports whether the negative cache has recorded that
// the named archetype has no file on disk. Callers should check this
// before os.Stat.
func (e *Engine) HasNoArchetype(name string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.noArchetype[name]
}

// SetNoArchetype records that the named archetype has no file on disk,
// suppressing future os.Stat calls until the engine is restarted or
// a tree is successfully loaded via LoadArchetype.
func (e *Engine) SetNoArchetype(name string) {
	e.mu.Lock()
	e.noArchetype[name] = true
	e.mu.Unlock()
}

// GetArchetype returns the cached archetype tree by name, or nil.
func (e *Engine) GetArchetype(name string) Node {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.archetypes[name]
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
		safeExecuteDelayed("behaviortree.delayed_action", da.Action)
	}
}

// safeExecuteDelayed runs a delayed-action closure with panic recovery so a
// single misbehaving action (e.g., one closing over a mob/user/room that has
// since been destroyed) does not crash the engine round tick. Panics are
// logged at mudlog.Error with the supplied name as the operation label.
func safeExecuteDelayed(name string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			mudlog.Error(name, "error", fmt.Sprintf("delayed action panicked: %v", r))
		}
	}()
	fn()
}
