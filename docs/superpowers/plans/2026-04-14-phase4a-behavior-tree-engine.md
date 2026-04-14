# Phase 4a: Behavior Tree Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a YAML-driven behavior tree engine for mob AI, replacing JS scripts and the dialogue system with one composable, declarative system.

**Architecture:** New `internal/behaviortree/` package with tree evaluator, node registry, and YAML loader. Trees evaluate event-driven (immediate) or idle-tick (round-based). Actions are instant or perception-delayed via a delayed action queue. Three proof-of-concept mobs validate the architecture.

**Tech Stack:** Go, YAML, existing event/rooms/mobs/characters packages

**Spec:** `docs/superpowers/specs/2026-04-14-phase4a-behavior-tree-engine-design.md`

---

## File Structure

```
internal/behaviortree/
├── types.go          — Node interface, result enum, event context struct
├── engine.go         — Tree evaluator, event dispatch, delayed action queue
├── loader.go         — YAML parsing, tree construction from definitions
├── conditions.go     — All condition node implementations
├── actions.go        — All action node implementations  
├── decorators.go     — Decorator node implementations
├── structural.go     — Selector, sequence nodes
├── state.go          — BehaviorState (per-mob-instance key/value store)
├── engine_test.go    — Unit tests for tree evaluation
├── conditions_test.go — Unit tests for conditions
├── actions_test.go   — Unit tests for actions
```

---

### Task 1: Core Types and Interfaces

**Files:**
- Create: `internal/behaviortree/types.go`
- Create: `internal/behaviortree/state.go`

- [ ] **Step 1: Create types.go with core interfaces**

```go
package behaviortree

// Result is the return value of a node evaluation.
type Result int

const (
    Success Result = iota
    Failure
    Running
)

// EventContext carries information about the triggering event.
type EventContext struct {
    EventType string         // "player_ask", "mob_idle", "player_give", etc.
    UserId    int            // Triggering player (0 if none)
    MobId     int            // Triggering mob instance (0 if none)
    Text      string         // For ask/say events — the text spoken
    ItemId    int            // For give/show events — the item
    RoomId    int            // Room where event occurred
    Extra     map[string]any // Extensible context
}

// Node is the interface all behavior tree nodes implement.
type Node interface {
    // Evaluate runs this node with the given context.
    // mob is the owning mob, event is the triggering context.
    Evaluate(ctx *EvalContext) Result
}

// EvalContext bundles everything a node needs during evaluation.
type EvalContext struct {
    Event      EventContext
    MobState   *BehaviorState
    // These will be populated by the engine from the mob instance:
    MobId      int    // Mob template ID
    InstanceId int    // Mob instance ID  
    RoomId     int    // Current room
    MobName    string // Mob's display name
}

// NodeDef is the raw YAML definition of a node, parsed before
// being compiled into a Node.
type NodeDef struct {
    Type     string    `yaml:"type"`               // selector, sequence, condition, action, decorator
    Event    string    `yaml:"event,omitempty"`     // Only evaluate on this event type
    Children []NodeDef `yaml:"children,omitempty"`  // Child nodes
    // Condition fields
    Check    string    `yaml:"check,omitempty"`     // Condition type name
    // Action fields
    Do       string    `yaml:"do,omitempty"`        // Action type name
    // Decorator fields
    Mod      string    `yaml:"mod,omitempty"`       // Decorator type name
    Child    *NodeDef  `yaml:"child,omitempty"`     // Single child (decorators)
    // Generic parameter map for condition/action/decorator-specific config
    Params   map[string]any `yaml:",inline"`
}

// TreeDef is the top-level YAML structure.
type TreeDef struct {
    Tree NodeDef `yaml:"tree"`
}
```

- [ ] **Step 2: Create state.go**

```go
package behaviortree

// BehaviorState is a per-mob-instance key/value store for behavior
// tree state. Persists for the mob's lifetime, reset on respawn.
type BehaviorState struct {
    data map[string]any
}

func NewBehaviorState() *BehaviorState {
    return &BehaviorState{data: make(map[string]any)}
}

func (s *BehaviorState) Get(key string) any {
    if s.data == nil {
        return nil
    }
    return s.data[key]
}

func (s *BehaviorState) GetString(key string) string {
    v, _ := s.Get(key).(string)
    return v
}

func (s *BehaviorState) GetInt(key string) int {
    switch v := s.Get(key).(type) {
    case int:
        return v
    case float64:
        return int(v)
    }
    return 0
}

func (s *BehaviorState) Set(key string, value any) {
    if s.data == nil {
        s.data = make(map[string]any)
    }
    s.data[key] = value
}

func (s *BehaviorState) Delete(key string) {
    delete(s.data, key)
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/behaviortree/...`

- [ ] **Step 4: Commit**

```bash
git add internal/behaviortree/types.go internal/behaviortree/state.go
git commit -m "feat: add behavior tree core types, interfaces, and state

Node interface, Result enum, EventContext, NodeDef for YAML parsing,
BehaviorState per-mob key/value store.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 2: Structural Nodes (Selector + Sequence)

**Files:**
- Create: `internal/behaviortree/structural.go`
- Create: `internal/behaviortree/engine_test.go`

- [ ] **Step 1: Write tests for selector and sequence**

Create `internal/behaviortree/engine_test.go`:

```go
package behaviortree

import "testing"

// mockNode returns a fixed result.
type mockNode struct {
    result Result
    called bool
}

func (n *mockNode) Evaluate(ctx *EvalContext) Result {
    n.called = true
    return n.result
}

func TestSelector_ReturnsFirstSuccess(t *testing.T) {
    fail := &mockNode{result: Failure}
    pass := &mockNode{result: Success}
    skip := &mockNode{result: Success}

    sel := &SelectorNode{Children: []Node{fail, pass, skip}}
    result := sel.Evaluate(&EvalContext{})

    if result != Success {
        t.Errorf("expected Success, got %v", result)
    }
    if !fail.called || !pass.called {
        t.Error("first two children should have been called")
    }
    if skip.called {
        t.Error("third child should NOT have been called")
    }
}

func TestSelector_AllFail(t *testing.T) {
    sel := &SelectorNode{Children: []Node{
        &mockNode{result: Failure},
        &mockNode{result: Failure},
    }}
    if sel.Evaluate(&EvalContext{}) != Failure {
        t.Error("expected Failure when all children fail")
    }
}

func TestSequence_RunsAllOnSuccess(t *testing.T) {
    a := &mockNode{result: Success}
    b := &mockNode{result: Success}
    seq := &SequenceNode{Children: []Node{a, b}}

    if seq.Evaluate(&EvalContext{}) != Success {
        t.Error("expected Success")
    }
    if !a.called || !b.called {
        t.Error("both children should have been called")
    }
}

func TestSequence_StopsOnFailure(t *testing.T) {
    a := &mockNode{result: Success}
    b := &mockNode{result: Failure}
    c := &mockNode{result: Success}
    seq := &SequenceNode{Children: []Node{a, b, c}}

    if seq.Evaluate(&EvalContext{}) != Failure {
        t.Error("expected Failure")
    }
    if c.called {
        t.Error("third child should NOT have been called")
    }
}

func TestEventFilter_SkipsMismatch(t *testing.T) {
    inner := &mockNode{result: Success}
    filtered := &EventFilterNode{
        EventType: "player_ask",
        Child:     inner,
    }
    // Wrong event — should return Failure without calling child
    ctx := &EvalContext{Event: EventContext{EventType: "mob_idle"}}
    if filtered.Evaluate(ctx) != Failure {
        t.Error("expected Failure for mismatched event")
    }
    if inner.called {
        t.Error("child should not have been called")
    }
}

func TestEventFilter_PassesMatch(t *testing.T) {
    inner := &mockNode{result: Success}
    filtered := &EventFilterNode{
        EventType: "player_ask",
        Child:     inner,
    }
    ctx := &EvalContext{Event: EventContext{EventType: "player_ask"}}
    if filtered.Evaluate(ctx) != Success {
        t.Error("expected Success for matching event")
    }
    if !inner.called {
        t.Error("child should have been called")
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -v ./internal/behaviortree/ -run TestSelector`
Expected: compilation failure

- [ ] **Step 3: Implement structural nodes**

Create `internal/behaviortree/structural.go`:

```go
package behaviortree

// SelectorNode tries children in order, returns Success on first
// success, Failure if all fail. Like an OR gate.
type SelectorNode struct {
    Children []Node
}

func (n *SelectorNode) Evaluate(ctx *EvalContext) Result {
    for _, child := range n.Children {
        result := child.Evaluate(ctx)
        if result == Success || result == Running {
            return result
        }
    }
    return Failure
}

// SequenceNode runs children in order, returns Failure on first
// failure, Success if all succeed. Like an AND gate.
type SequenceNode struct {
    Children []Node
}

func (n *SequenceNode) Evaluate(ctx *EvalContext) Result {
    for _, child := range n.Children {
        result := child.Evaluate(ctx)
        if result == Failure {
            return Failure
        }
        if result == Running {
            return Running
        }
    }
    return Success
}

// EventFilterNode wraps a child and only evaluates it when the
// event type matches. Returns Failure on mismatch (skips branch).
type EventFilterNode struct {
    EventType string
    Child     Node
}

func (n *EventFilterNode) Evaluate(ctx *EvalContext) Result {
    if ctx.Event.EventType != n.EventType {
        return Failure
    }
    return n.Child.Evaluate(ctx)
}
```

- [ ] **Step 4: Run tests**

Run: `go test -v ./internal/behaviortree/...`
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/behaviortree/structural.go internal/behaviortree/engine_test.go
git commit -m "feat: add selector, sequence, and event filter nodes

Core structural nodes for behavior trees with full test coverage.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 3: Condition Nodes

**Files:**
- Create: `internal/behaviortree/conditions.go`
- Create: `internal/behaviortree/conditions_test.go`

- [ ] **Step 1: Write tests for key conditions**

Create `internal/behaviortree/conditions_test.go` with tests for:
- `keyword_match` — matches "quest" against ["quest", "task"]
- `keyword_match` — fails on "hello" against ["quest", "task"]
- `state_equals` — checks BehaviorState value
- `random_chance` — 100% always succeeds, 0% always fails
- `round_mod` — succeeds when round % N == 0

Each test constructs the condition node directly and calls Evaluate
with an appropriate EvalContext.

- [ ] **Step 2: Implement conditions**

Create `internal/behaviortree/conditions.go` with a condition registry
pattern:

```go
package behaviortree

import (
    "strings"
)

// ConditionFunc is the signature for condition evaluators.
type ConditionFunc func(params map[string]any, ctx *EvalContext) Result

// conditionRegistry maps condition names to their evaluators.
var conditionRegistry = map[string]ConditionFunc{}

// RegisterCondition adds a condition to the registry.
func RegisterCondition(name string, fn ConditionFunc) {
    conditionRegistry[name] = fn
}

// ConditionNode wraps a registered condition function.
type ConditionNode struct {
    CheckName string
    Params    map[string]any
}

func (n *ConditionNode) Evaluate(ctx *EvalContext) Result {
    fn, ok := conditionRegistry[n.CheckName]
    if !ok {
        return Failure
    }
    return fn(n.Params, ctx)
}

func init() {
    RegisterCondition("keyword_match", condKeywordMatch)
    RegisterCondition("player_has_quest", condPlayerHasQuest)
    RegisterCondition("player_missing_quest", condPlayerMissingQuest)
    RegisterCondition("player_has_item", condPlayerHasItem)
    RegisterCondition("player_has_gold", condPlayerHasGold)
    RegisterCondition("player_has_flag", condPlayerHasFlag)
    RegisterCondition("mob_in_combat", condMobInCombat)
    RegisterCondition("mob_health_below", condMobHealthBelow)
    RegisterCondition("mob_at_home", condMobAtHome)
    RegisterCondition("time_of_day", condTimeOfDay)
    RegisterCondition("round_mod", condRoundMod)
    RegisterCondition("random_chance", condRandomChance)
    RegisterCondition("state_equals", condStateEquals)
    RegisterCondition("players_in_room", condPlayersInRoom)
    RegisterCondition("item_matches", condItemMatches)
}
```

Implement each condition function. They extract parameters from the
`params` map and check against the `EvalContext`. For conditions that
need game state (player quest tokens, mob health, room data), the
implementer must read the existing Go APIs:

- **Player quest state:** `users.GetByUserId(ctx.Event.UserId)` →
  `user.Character.HasQuest(questId)` or similar. Read
  `internal/characters/character.go` for quest methods.
- **Mob state:** `mobs.GetInstance(ctx.InstanceId)` →
  `mob.Character.Health`, `mob.Character.Aggro`, etc.
- **Room state:** `rooms.LoadRoom(ctx.RoomId)` →
  `room.GetPlayers()`, etc.
- **Time of day:** Search for `IsDay()` or `GetTimeOfDay()` in
  `internal/gametime/`.
- **Round number:** `util.GetRoundCount()`

The `keyword_match` condition splits `ctx.Event.Text` into words and
checks if any match the `keywords` list (case-insensitive).

The `item_matches` condition checks `ctx.Event.ItemId` against the
`item_id` parameter.

- [ ] **Step 3: Run tests**

Run: `go test -v ./internal/behaviortree/... -run TestCond`
Expected: all tests PASS

- [ ] **Step 4: Commit**

```bash
git add internal/behaviortree/conditions.go internal/behaviortree/conditions_test.go
git commit -m "feat: add condition node registry with 15 condition types

keyword_match, quest/item/gold/flag checks, combat state, time,
randomness, behavior state. Registry pattern for extensibility.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 4: Action Nodes

**Files:**
- Create: `internal/behaviortree/actions.go`

- [ ] **Step 1: Implement action node registry**

Same registry pattern as conditions:

```go
package behaviortree

type ActionFunc func(params map[string]any, ctx *EvalContext) Result

var actionRegistry = map[string]ActionFunc{}

func RegisterAction(name string, fn ActionFunc) {
    actionRegistry[name] = fn
}

type ActionNode struct {
    DoName string
    Params map[string]any
}

func (n *ActionNode) Evaluate(ctx *EvalContext) Result {
    fn, ok := actionRegistry[n.DoName]
    if !ok {
        return Failure
    }
    return fn(n.Params, ctx)
}
```

Register all action types in `init()`:
- `respond` — send user_text/room_text with token substitution + hints
- `say` — mob says text to room
- `emote` — mob emotes text to room
- `grant_quest` — give quest token to triggering player
- `set_quest_flag` — set quest flag on player
- `give_item` — give item to player
- `take_item` — take item from player
- `give_gold` — give gold to player
- `take_gold` — take gold from player
- `move` — move mob to room
- `attack` — start combat
- `flee` — flee from combat
- `cast` — cast a spell
- `spawn_mob` — spawn mob in room
- `add_temp_exit` — add temporary room exit
- `set_state` — set BehaviorState value
- `command` — execute arbitrary mob command (escape hatch)

Each action function:
1. Extracts params from the map
2. Looks up the mob instance via `mobs.GetInstance(ctx.InstanceId)`
3. Looks up the user via `users.GetByUserId(ctx.Event.UserId)` if needed
4. Performs the action using existing Go APIs
5. Returns Success or Failure

**IMPORTANT:** Actions that are reaction-delayed (respond, say, emote,
move, attack, flee, cast) should check the spec's timing model. For
Phase 4a, implement them as instant first — reaction delay is wired in
Task 6 (engine). The action functions themselves just do the work;
the engine handles when to call them.

Use `textutil.SubstituteTokens` for `{source}` and `{target}` in
text fields. Use `rooms.Room.SendTextVisual` for room broadcasts
(darkness-aware, added in the bug fix session).

For `respond` with `hints`: send the hints as narrator text after the
NPC's speech. Read how the existing dialogue system formats hints
(search for `hints` in `internal/hooks/` or `internal/usercommands/talk.go`).

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/behaviortree/...`

- [ ] **Step 3: Commit**

```bash
git add internal/behaviortree/actions.go
git commit -m "feat: add action node registry with 17 action types

respond, say, emote, quest/item/gold actions, movement, combat,
spawning, state management. Uses existing Go APIs.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 5: Decorator Nodes

**Files:**
- Create: `internal/behaviortree/decorators.go`

- [ ] **Step 1: Implement decorator nodes**

```go
package behaviortree

import (
    "github.com/GoMudEngine/GoMud/internal/util"
)

// CooldownDecorator skips its child if it was last run within N rounds.
type CooldownDecorator struct {
    Rounds   int
    StateKey string // unique key for tracking last-run round in BehaviorState
    Child    Node
}

func (d *CooldownDecorator) Evaluate(ctx *EvalContext) Result {
    lastRun := ctx.MobState.GetInt(d.StateKey)
    currentRound := int(util.GetRoundCount())
    if currentRound-lastRun < d.Rounds {
        return Failure
    }
    result := d.Child.Evaluate(ctx)
    if result == Success {
        ctx.MobState.Set(d.StateKey, currentRound)
    }
    return result
}

// RepeatDecorator runs its child N times.
type RepeatDecorator struct {
    Times int
    Child Node
}

func (d *RepeatDecorator) Evaluate(ctx *EvalContext) Result {
    for i := 0; i < d.Times; i++ {
        result := d.Child.Evaluate(ctx)
        if result == Failure {
            return Failure
        }
    }
    return Success
}

// InvertDecorator flips Success↔Failure.
type InvertDecorator struct {
    Child Node
}

func (d *InvertDecorator) Evaluate(ctx *EvalContext) Result {
    result := d.Child.Evaluate(ctx)
    if result == Success {
        return Failure
    }
    if result == Failure {
        return Success
    }
    return Running
}

// RandomDecorator runs its child with N% probability.
type RandomDecorator struct {
    Percent int
    Child   Node
}

func (d *RandomDecorator) Evaluate(ctx *EvalContext) Result {
    if util.Rand(100) >= d.Percent {
        return Failure
    }
    return d.Child.Evaluate(ctx)
}

// DelayDecorator waits N rounds using BehaviorState to track start.
type DelayDecorator struct {
    Rounds   int
    StateKey string
    Child    Node
}

func (d *DelayDecorator) Evaluate(ctx *EvalContext) Result {
    startRound := ctx.MobState.GetInt(d.StateKey)
    currentRound := int(util.GetRoundCount())
    if startRound == 0 {
        ctx.MobState.Set(d.StateKey, currentRound)
        return Running
    }
    if currentRound-startRound < d.Rounds {
        return Running
    }
    ctx.MobState.Delete(d.StateKey)
    return d.Child.Evaluate(ctx)
}
```

- [ ] **Step 2: Verify compilation and commit**

```bash
git add internal/behaviortree/decorators.go
git commit -m "feat: add decorator nodes — cooldown, repeat, invert, random, delay

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 6: YAML Loader

**Files:**
- Create: `internal/behaviortree/loader.go`

- [ ] **Step 1: Implement YAML loader that compiles NodeDefs into Nodes**

The loader reads a `TreeDef` from YAML and recursively compiles each
`NodeDef` into a concrete `Node`:

```go
package behaviortree

import (
    "fmt"
    "os"
    "gopkg.in/yaml.v2"
)

// LoadTreeFromFile reads a behavior tree YAML file and returns the
// compiled root node.
func LoadTreeFromFile(path string) (Node, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    return LoadTreeFromBytes(data)
}

func LoadTreeFromBytes(data []byte) (Node, error) {
    var def TreeDef
    if err := yaml.Unmarshal(data, &def); err != nil {
        return nil, fmt.Errorf("parse error: %w", err)
    }
    return compileNode(def.Tree, "root")
}
```

The `compileNode` function switches on `NodeDef.Type`:
- `"selector"` → compile children, wrap in `SelectorNode`
- `"sequence"` → compile children, wrap in `SequenceNode`
- `"condition"` → create `ConditionNode` with `Check` and remaining params
- `"action"` → create `ActionNode` with `Do` and remaining params
- `"decorator"` → switch on `Mod`, create appropriate decorator wrapping compiled `Child`

If a node has an `Event` field, wrap it in an `EventFilterNode`.

The `Params` field uses `yaml:",inline"` to capture all extra YAML
keys not covered by the fixed fields (Type, Event, Children, etc.).
The implementer needs to extract known fields (Check, Do, Mod,
keywords, quest, text, etc.) from the inline params and pass the
rest to the condition/action node.

**IMPORTANT:** Read the `NodeDef` struct from Task 1. The `Params`
field with `yaml:",inline"` captures all unknown keys. The loader
must separate structural fields (type, event, children, check, do,
mod, child) from content params (keywords, quest, text, item_id,
etc.) when constructing nodes.

- [ ] **Step 2: Write a test that loads a simple YAML tree**

```go
func TestLoadTree_SimpleSelector(t *testing.T) {
    yaml := `
tree:
  type: selector
  children:
    - type: condition
      check: random_chance
      percent: 0
    - type: action
      do: set_state
      key: test
      value: passed
`
    node, err := LoadTreeFromBytes([]byte(yaml))
    if err != nil {
        t.Fatalf("load error: %v", err)
    }
    state := NewBehaviorState()
    ctx := &EvalContext{MobState: state, Event: EventContext{EventType: "mob_idle"}}
    node.Evaluate(ctx)
    if state.GetString("test") != "passed" {
        t.Error("expected state 'test' to be 'passed'")
    }
}
```

- [ ] **Step 3: Run tests, verify, commit**

```bash
git add internal/behaviortree/loader.go
git commit -m "feat: add YAML loader that compiles behavior tree definitions

Parses TreeDef YAML, recursively compiles NodeDefs into Node
instances. Handles event filtering, conditions, actions, decorators.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 7: Engine — Tree Evaluation and Delayed Action Queue

**Files:**
- Create: `internal/behaviortree/engine.go`

- [ ] **Step 1: Implement the engine**

The engine manages:
1. A cache of loaded behavior trees (mob ID → compiled root Node)
2. Event dispatch — receives events and evaluates the matching mob's tree
3. Idle tick — called once per round for each mob with a tree
4. Delayed action queue — actions with reaction delay are queued and
   executed after delay expires

```go
package behaviortree

import (
    "sync"
    "time"
)

// Engine manages behavior tree loading, caching, and evaluation.
type Engine struct {
    mu    sync.RWMutex
    trees map[int]Node // mobId → compiled root node
    queue []DelayedAction
}

type DelayedAction struct {
    ExecuteAt time.Time
    Action    func()
}

var globalEngine *Engine

func init() {
    globalEngine = &Engine{
        trees: make(map[int]Node),
    }
}

func GetEngine() *Engine {
    return globalEngine
}

// LoadTree loads and caches a behavior tree for a mob type.
func (e *Engine) LoadTree(mobId int, path string) error {
    node, err := LoadTreeFromFile(path)
    if err != nil {
        return err
    }
    e.mu.Lock()
    e.trees[mobId] = node
    e.mu.Unlock()
    return nil
}

// GetTree returns the cached tree for a mob type, or nil.
func (e *Engine) GetTree(mobId int) Node {
    e.mu.RLock()
    defer e.mu.RUnlock()
    return e.trees[mobId]
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
```

- [ ] **Step 2: Verify compilation and commit**

```bash
git add internal/behaviortree/engine.go
git commit -m "feat: add behavior tree engine with caching and delayed action queue

Global engine singleton, tree loading/caching, event-driven evaluation,
time-based delayed action queue for perception-scaled reactions.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 8: Add BehaviorState to Mob + Config for Reaction Delays

**Files:**
- Modify: `internal/mobs/mobs.go`
- Modify: `_datafiles/config.yaml` (or config Go struct)

- [ ] **Step 1: Add BehaviorState to Mob struct**

In `internal/mobs/mobs.go`, add a `BehaviorState` field to the `Mob`
struct (or to `Character` — check which makes sense). Read the Mob
struct to find the right place. It should be a `yaml:"-"` field
(not persisted) since it resets on respawn.

```go
BTreeState *behaviortree.BehaviorState `yaml:"-"`
```

Initialize it in `NewMobById` after the deep copy block:
```go
mob.BTreeState = behaviortree.NewBehaviorState()
```

- [ ] **Step 2: Add reaction delay config**

Search `internal/configs/` for the Balance config struct. Add:

```go
MobReactionBase            float64 `yaml:"MobReactionBase,omitempty"`
MobReactionPerceptionScale int     `yaml:"MobReactionPerceptionScale,omitempty"`
MobReactionMin             float64 `yaml:"MobReactionMin,omitempty"`
MobReactionMax             float64 `yaml:"MobReactionMax,omitempty"`
```

Add defaults in the config initialization (base: 3.0, scale: 100,
min: 0.25, max: 3.5).

- [ ] **Step 3: Verify compilation and commit**

```bash
git add internal/mobs/mobs.go internal/configs/
git commit -m "feat: add BehaviorState to Mob, reaction delay config

Per-instance behavior state resets on spawn. Config knobs for
perception-scaled reaction delays.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 9: Wire Engine into Mob Event Hooks

**Files:**
- Modify: `internal/hooks/` — multiple hook files

- [ ] **Step 1: Wire mob event dispatch to behavior tree engine**

Find where mob events are dispatched to JS scripts:
- `onIdle` — in `MobIdle_HandleIdleMobs.go`
- `onAsk` — search for `TryMobScriptEvent("onAsk"` in hooks
- `onGive` — search for `TryMobScriptEvent("onGive"`
- `onPlayerEnter` — search for player enter events dispatched to mobs
- `onHurt` — search for mob hurt events

At each dispatch point, BEFORE the JS `TryMobScriptEvent` call, add
behavior tree evaluation:

```go
if mob.BTreeState != nil {
    behaviortree.GetEngine().EvaluateEvent(
        int(mob.MobId),
        mob.InstanceId,
        behaviortree.EventContext{
            EventType: "player_ask",  // or appropriate event type
            UserId:    userId,
            RoomId:    mob.Character.RoomId,
            Text:      askText,       // for ask/say events
        },
        mob.BTreeState,
    )
}
```

Also wire `DrainQueue()` into the round tick so delayed actions execute.

- [ ] **Step 2: Wire tree loading at mob spawn**

In `NewMobById` (or wherever mobs are initialized), check if a
behavior YAML file exists for the mob and load it:

```go
// Check for behavior tree file
btreePath := behaviortree.GetBehaviorPath(mob.MobId, mob.Zone, mob.Character.Name)
if _, err := os.Stat(btreePath); err == nil {
    behaviortree.GetEngine().LoadTree(int(mob.MobId), btreePath)
}
```

Add a `GetBehaviorPath` function to the behaviortree package that
constructs the path: `_datafiles/world/{world}/mobs/{zone}/behaviors/{mobid}-{name}.yaml`

- [ ] **Step 3: Verify compilation and commit**

```bash
git add internal/hooks/ internal/mobs/ internal/behaviortree/
git commit -m "feat: wire behavior tree engine into mob event dispatch

Behavior trees evaluate on mob events (ask, give, idle, etc.) before
JS scripts. Delayed action queue drained each round tick. Trees
auto-loaded at mob spawn from behaviors/ YAML files.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 10: Proof of Concept — Temple Priest Olen

**Files:**
- Create: `_datafiles/world/dogmud/mobs/thornwall_city/behaviors/95-temple_priest_olen.yaml`
- Delete: `_datafiles/world/dogmud/mobs/thornwall_city/scripts/95-temple_priest_olen.js.bak` (if exists)
- Modify: relevant dialogue YAML (migrate into behavior tree)

- [ ] **Step 1: Read existing Olen JS and dialogue**

Read the Olen JS script and dialogue YAML to understand all his
behaviors: quest dialogue, idle emotes, item reception (tithe ledger),
quest gating.

- [ ] **Step 2: Create behavior tree YAML**

Translate all of Olen's behavior into a single behavior tree YAML
file. The tree should handle:
- Quest 9 dialogue (tithe audit) — ask triggers, quest gating
- Receiving the tithe ledger (player_give event)
- Idle emotes with cooldown
- Hints for discoverability

Follow the format from the spec. Include comments explaining each
branch.

- [ ] **Step 3: Delete old JS and dialogue files**

Remove Olen's JS script. Migrate his dialogue YAML entries into the
behavior tree.

- [ ] **Step 4: Test manually**

Start server, go to Olen's room, test:
- `ask olen quest` — should get quest dialogue
- `ask olen tithe` — should get quest details
- Give ledger to Olen — should advance quest
- Idle emotes should fire periodically

- [ ] **Step 5: Commit**

```bash
git add _datafiles/world/dogmud/mobs/thornwall_city/behaviors/
git rm [old files]
git commit -m "feat: migrate Temple Priest Olen to behavior tree

First proof-of-concept mob. Quest dialogue, item reception, idle
emotes all in one YAML behavior tree.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 11: Proof of Concept — Bandit Leader

**Files:**
- Create: `_datafiles/world/dogmud/mobs/marches_spur_road/behaviors/254-bandit_leader.yaml`
- Delete: `_datafiles/world/dogmud/mobs/marches_spur_road/scripts/254-bandit_leader.js`

- [ ] **Step 1: Read existing Bandit Leader JS**

Understand: timed aggro countdown, negotiation system, combat tactics
(hit-and-run with sneak/hide).

- [ ] **Step 2: Create behavior tree YAML**

The tree should handle:
- Player enters room → start countdown (state tracking)
- Countdown expires → attack
- Player negotiates (say/give gold) → cancel countdown
- In combat: periodic flee + sneak + re-engage (state machine)

- [ ] **Step 3: Delete old JS, test manually, commit**

```bash
git commit -m "feat: migrate Bandit Leader to behavior tree

Combat AI with timed aggro, negotiation, hit-and-run tactics.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 12: Proof of Concept — Barmaid Dal

**Files:**
- Create: `_datafiles/world/dogmud/mobs/thornwall_city/behaviors/117-barmaid_dal.yaml`
- Delete: `_datafiles/world/dogmud/mobs/thornwall_city/scripts/117-barmaid_dal.js`

- [ ] **Step 1: Read existing Barmaid Dal JS**

Understand: room cycling routine (moves between rooms on timer),
NPC-NPC interactions, dialogue.

- [ ] **Step 2: Create behavior tree YAML**

The tree should handle:
- Timed room cycling (move between rooms every N rounds)
- Idle emotes in each room
- Player dialogue (talk/ask triggers)
- State tracking for current position in routine

- [ ] **Step 3: Delete old JS, test manually, commit**

```bash
git commit -m "feat: migrate Barmaid Dal to behavior tree

Timed routine with room cycling, idle emotes, dialogue.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```

---

### Task 13: Documentation and Content Creation Support

**Files:**
- Create: `docs/schemas/behavior.md`
- Create: `_datafiles/world/dogmud/mobs/thornwall_city/context.md` (or update existing)
- Modify: `.claude/commands/new-mob.md`

- [ ] **Step 1: Create behavior tree schema doc**

`docs/schemas/behavior.md` — full reference for all node types,
conditions, actions, decorators. YAML format examples. Event types.
Discoverability SOP.

- [ ] **Step 2: Create/update zone context.md files**

Add style guide, ordering conventions, and syntax reference for
behavior tree YAML files. This is what subagents read before
generating content.

- [ ] **Step 3: Update /new-mob to generate behavior tree templates**

When generating a new mob with dialogue or AI, the slash command
should generate a behavior tree template alongside the mob YAML.

- [ ] **Step 4: Create AI tester goals file**

Create `tools/testing/goals/phase4a-behavior.yaml` for smoke testing
the three migrated mobs.

- [ ] **Step 5: Commit**

```bash
git add docs/schemas/behavior.md _datafiles/world/dogmud/mobs/*/context.md .claude/commands/new-mob.md tools/testing/goals/
git commit -m "docs: add behavior tree schema, context guides, testing goals

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>"
```
