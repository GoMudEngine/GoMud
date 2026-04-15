# Phase 4c: Room, Spell & Buff Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate all 30 remaining dogmud JS scripts to Go/YAML/behavior trees, introduce room behavior trees as a first-class system, simplify the death flow, and eliminate the shadow realm detour.

**Architecture:** Three tiers — (1) build room behavior tree engine infrastructure (EvalContext changes, TryRoomBehavior, room event wiring, new conditions/actions), (2) migrate tutorial and non-tutorial room scripts to room behavior trees, (3) move spell/buff logic to Go hooks, simplify death, delete stubs.

**Tech Stack:** Go, YAML, existing behaviortree/hooks/rooms/characters packages

**Spec:** `docs/superpowers/specs/2026-04-15-phase4c-room-spell-buff-migration-design.md`

## File Structure

```
internal/behaviortree/
├── types.go          — add Intercepted field to EvalContext, Command/Rest/Direction fields to EventContext
├── actions.go        — add ~9 new actions, static delay support in Evaluate
├── conditions.go     — add 3 new conditions
├── helpers.go        — add TryRoomBehavior, GetRoomBehaviorPath
├── engine.go         — add room tree cache + room negative cache
├── room_state.go     — room BehaviorState storage (NEW)
├── context.md        — update docs with room behavior tree system

internal/hooks/
├── NewRound_UserRoundTick.go  — wire room_idle event
├── spell_resolution.go        — wire Go spell hooks for fold-anchor/fold-recall/purge-affliction
├── spell_foldanchor.go        — fold-anchor Go hook (NEW)
├── spell_foldrecall.go        — fold-recall Go hook (NEW)
├── spell_purgeaffliction.go   — purge-affliction Go hook (NEW)
├── buff_meditating.go         — meditating buff Go hook (NEW)

internal/usercommands/
├── go.go             — wire room_enter and room_exit events
├── usercommands.go   — wire room_command event (before command dispatch)
├── suicide.go        — simplify death flow

internal/rooms/
├── rooms.go          — add BTreeState field to Room struct

_datafiles/world/dogmud/behaviors/rooms/
├── sanctum_basin/102.yaml   (NEW)
├── sanctum_basin/106.yaml   (NEW)
├── sanctum_basin/108.yaml   (NEW)
├── sanctum_basin/109.yaml   (NEW)
├── sanctum_basin/111.yaml   (NEW)
├── sanctum_basin/113.yaml   (NEW)
├── sanctum_basin/114.yaml   (NEW)
├── sanctum_basin/116.yaml   (NEW)
├── sanctum_basin/120.yaml   (NEW)
├── dustwalk_road/407.yaml   (NEW)
├── ashwick/4023.yaml        (NEW)

_datafiles/world/dogmud/behaviors/
├── thornwall_city/315-sable.yaml  (NEW)
```

---

## Task 1: Extend EvalContext for room events + command interception

**Goal:** Add `Intercepted bool` to `EvalContext` and `Command`, `Rest`, `Direction` to `EventContext` so room behavior trees can inspect and intercept player commands.

**Files:** `internal/behaviortree/types.go`

- [ ] **Step 1: Add fields to EventContext**

  Open `internal/behaviortree/types.go`. Add three new fields to `EventContext` after the `Extra` field:

  ```go
  // Replace the existing EventContext struct
  ```

  Apply this edit to `internal/behaviortree/types.go`:

  **Find:**
  ```go
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
  ```

  **Replace with:**
  ```go
  // EventContext carries information about the triggering event.
  type EventContext struct {
  	EventType string         // "player_ask", "mob_idle", "player_give", etc.
  	UserId    int            // Triggering player (0 if none)
  	MobId     int            // Triggering mob instance (0 if none)
  	Text      string         // For ask/say events — the text spoken
  	ItemId    int            // For give/show events — the item
  	RoomId    int            // Room where event occurred
  	Extra     map[string]any // Extensible context
  	Command   string         // For room_command events — the parsed command verb
  	Rest      string         // For room_command events — arguments after the verb
  	Direction string         // For room_enter/room_exit — the exit name used
  }
  ```

- [ ] **Step 2: Add Intercepted field to EvalContext**

  In the same file, add `Intercepted bool` to `EvalContext`.

  **Find:**
  ```go
  // EvalContext bundles everything a node needs during evaluation.
  type EvalContext struct {
  	Event      EventContext
  	MobState   *BehaviorState
  	MobId      int    // Mob template ID
  	InstanceId int    // Mob instance ID
  	RoomId     int    // Current room
  	MobName    string // Mob's display name
  }
  ```

  **Replace with:**
  ```go
  // EvalContext bundles everything a node needs during evaluation.
  type EvalContext struct {
  	Event       EventContext
  	MobState    *BehaviorState
  	MobId       int    // Mob template ID
  	InstanceId  int    // Mob instance ID
  	RoomId      int    // Current room
  	MobName     string // Mob's display name
  	Intercepted bool   // Set by room trees to block command dispatch
  }
  ```

- [ ] **Step 3: Verify build**

  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 4: Commit**

  ```bash
  git add internal/behaviortree/types.go
  git commit -m "$(cat <<'EOF'
  feat(behaviortree): extend EvalContext for room events + interception

  Add Command, Rest, Direction fields to EventContext for room behavior
  tree events. Add Intercepted bool to EvalContext so room trees can
  block command dispatch.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

## Task 2: Add static delay support to ActionNode.Evaluate

**Goal:** Allow behavior tree actions to specify a fixed `delay` parameter (in seconds) that takes precedence over perception-scaled delay. This is needed for scripted NPC dialogue sequences in room behavior trees where timing must be deterministic.

**Files:** `internal/behaviortree/actions.go`

- [ ] **Step 1: Add static delay check before perception delay**

  In `internal/behaviortree/actions.go`, modify `ActionNode.Evaluate` to check for a `delay` float64 param first. When present, it queues the action with that exact delay, bypassing perception scaling.

  **Find:**
  ```go
  func (n *ActionNode) Evaluate(ctx *EvalContext) Result {
  	if delayedActions[n.Name] {
  		delay := calcReactionDelay(ctx.InstanceId)
  		if delay > 0 {
  			params := n.Params
  			fn := n.Fn
  			evalCtx := &EvalContext{
  				Event:      ctx.Event,
  				MobState:   ctx.MobState,
  				MobId:      ctx.MobId,
  				InstanceId: ctx.InstanceId,
  				RoomId:     ctx.RoomId,
  				MobName:    ctx.MobName,
  			}
  			GetEngine().QueueDelayed(delay, func() {
  				fn(params, evalCtx)
  			})
  			return Success
  		}
  	}
  	return n.Fn(n.Params, ctx)
  }
  ```

  **Replace with:**
  ```go
  func (n *ActionNode) Evaluate(ctx *EvalContext) Result {
  	// Check for explicit static delay (seconds). Takes precedence over
  	// perception-scaled delay. Used for scripted dialogue sequences where
  	// timing must be deterministic.
  	if staticDelay, ok := n.Params["delay"]; ok {
  		var delaySec float64
  		switch v := staticDelay.(type) {
  		case float64:
  			delaySec = v
  		case int:
  			delaySec = float64(v)
  		}
  		if delaySec > 0 {
  			params := n.Params
  			fn := n.Fn
  			evalCtx := &EvalContext{
  				Event:       ctx.Event,
  				MobState:    ctx.MobState,
  				MobId:       ctx.MobId,
  				InstanceId:  ctx.InstanceId,
  				RoomId:      ctx.RoomId,
  				MobName:     ctx.MobName,
  				Intercepted: ctx.Intercepted,
  			}
  			dur := time.Duration(delaySec * float64(time.Second))
  			GetEngine().QueueDelayed(dur, func() {
  				fn(params, evalCtx)
  			})
  			return Success
  		}
  	}

  	if delayedActions[n.Name] {
  		delay := calcReactionDelay(ctx.InstanceId)
  		if delay > 0 {
  			params := n.Params
  			fn := n.Fn
  			evalCtx := &EvalContext{
  				Event:       ctx.Event,
  				MobState:    ctx.MobState,
  				MobId:       ctx.MobId,
  				InstanceId:  ctx.InstanceId,
  				RoomId:      ctx.RoomId,
  				MobName:     ctx.MobName,
  				Intercepted: ctx.Intercepted,
  			}
  			GetEngine().QueueDelayed(delay, func() {
  				fn(params, evalCtx)
  			})
  			return Success
  		}
  	}
  	return n.Fn(n.Params, ctx)
  }
  ```

- [ ] **Step 2: Add `time` to imports if not present**

  Check the import block in `actions.go`. It currently imports `fmt` and several internal packages but NOT `time`. Add `"time"` to the import block.

  **Find:**
  ```go
  import (
  	"fmt"

  	"github.com/GoMudEngine/GoMud/internal/characters"
  ```

  **Replace with:**
  ```go
  import (
  	"fmt"
  	"time"

  	"github.com/GoMudEngine/GoMud/internal/characters"
  ```

- [ ] **Step 3: Add `delay` to knownFields in loader**

  The YAML loader's `cleanParams` strips known fields so they don't leak into `Params`. Add `"delay"` to the `knownFields` map in the loader so it IS passed through as a param (it's consumed by `ActionNode.Evaluate`, not the YAML parser). Actually — `delay` is NOT a structural YAML field, it's an inline param that should end up in `Params`. Since `knownFields` only strips structural fields (`type`, `event`, `children`, etc.), `delay` will naturally flow into `Params` via the `",inline"` tag. **No change needed here.**

- [ ] **Step 4: Verify build**

  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 5: Commit**

  ```bash
  git add internal/behaviortree/actions.go
  git commit -m "$(cat <<'EOF'
  feat(behaviortree): add static delay support to ActionNode.Evaluate

  Actions can now specify a delay param (float64 seconds) that queues
  execution with a fixed delay, bypassing perception-scaled timing.
  Used for scripted NPC dialogue sequences in room behavior trees.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

## Task 3: Add room behavior tree infrastructure

**Goal:** Add `TryRoomBehavior`, `GetRoomBehaviorPath`, room tree cache, room negative cache, and room `BehaviorState` storage. This parallels the mob behavior tree system but keyed on roomId.

**Files:**
- `internal/behaviortree/engine.go` — add room tree/negative cache maps and methods
- `internal/behaviortree/helpers.go` — add `TryRoomBehavior` and `GetRoomBehaviorPath`
- `internal/behaviortree/room_state.go` (NEW) — room BehaviorState storage

### Step Group A: Engine room tree cache

- [ ] **Step 1: Add room tree maps to Engine struct and init**

  In `internal/behaviortree/engine.go`, add `roomTrees` and `noRoomTree` maps to the `Engine` struct and initialize them in `init()`.

  **Find:**
  ```go
  // Engine manages behavior tree loading, caching, and evaluation.
  type Engine struct {
  	mu     sync.RWMutex
  	trees  map[int]Node // mobId → compiled root node
  	noTree map[int]bool // mobId → no behavior file exists on disk
  	queue  []DelayedAction
  }
  ```

  **Replace with:**
  ```go
  // Engine manages behavior tree loading, caching, and evaluation.
  type Engine struct {
  	mu         sync.RWMutex
  	trees      map[int]Node // mobId → compiled root node
  	noTree     map[int]bool // mobId → no behavior file exists on disk
  	roomTrees  map[int]Node // roomId → compiled root node
  	noRoomTree map[int]bool // roomId → no behavior file exists on disk
  	queue      []DelayedAction
  }
  ```

  **Find:**
  ```go
  func init() {
  	globalEngine = &Engine{
  		trees:  make(map[int]Node),
  		noTree: make(map[int]bool),
  	}
  }
  ```

  **Replace with:**
  ```go
  func init() {
  	globalEngine = &Engine{
  		trees:      make(map[int]Node),
  		noTree:     make(map[int]bool),
  		roomTrees:  make(map[int]Node),
  		noRoomTree: make(map[int]bool),
  	}
  }
  ```

- [ ] **Step 2: Add room tree cache methods to Engine**

  Append the following methods after the existing `GetTree` method in `internal/behaviortree/engine.go`:

  ```go
  // LoadRoomTree loads and caches a behavior tree for a room.
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

  // HasNoRoomTree reports whether the negative cache has recorded that roomId
  // has no behavior tree file on disk.
  func (e *Engine) HasNoRoomTree(roomId int) bool {
  	e.mu.RLock()
  	defer e.mu.RUnlock()
  	return e.noRoomTree[roomId]
  }

  // SetNoRoomTree records that roomId has no behavior tree file on disk.
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
  ```

- [ ] **Step 3: Verify build**

  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

### Step Group B: Room state storage

- [ ] **Step 4: Create room_state.go**

  Create `internal/behaviortree/room_state.go` with a package-level map for room BehaviorState storage. Rooms can't import behaviortree (circular), so state is stored here and accessed by roomId.

  ```go
  package behaviortree

  import "sync"

  // roomStates stores per-room BehaviorState instances. Keyed by roomId.
  // This lives in the behaviortree package (not rooms) to avoid circular
  // imports — rooms cannot import behaviortree.
  var (
  	roomStateMu sync.RWMutex
  	roomStates  = make(map[int]*BehaviorState)
  )

  // EnsureRoomBTreeState lazily initializes and returns the BehaviorState
  // for a given room. Thread-safe.
  func EnsureRoomBTreeState(roomId int) *BehaviorState {
  	roomStateMu.RLock()
  	state, ok := roomStates[roomId]
  	roomStateMu.RUnlock()
  	if ok && state != nil {
  		return state
  	}
  	roomStateMu.Lock()
  	defer roomStateMu.Unlock()
  	// Double-check after acquiring write lock
  	if state, ok = roomStates[roomId]; ok && state != nil {
  		return state
  	}
  	state = NewBehaviorState()
  	roomStates[roomId] = state
  	return state
  }
  ```

- [ ] **Step 5: Verify build**

  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

### Step Group C: TryRoomBehavior and path helper

- [ ] **Step 6: Add GetRoomBehaviorPath to helpers.go**

  In `internal/behaviortree/helpers.go`, add the path helper after the existing `GetBehaviorPath` function. Room trees are stored at `{dataFiles}/behaviors/rooms/{zoneSafe}/{roomId}.yaml`.

  Add `"strconv"` to the import block if not already present (it already is — verify).

  Add after the `GetBehaviorPath` function:

  ```go
  // GetRoomBehaviorPath constructs the filesystem path to a room's behavior
  // tree YAML. Path: {dataFiles}/behaviors/rooms/{zone}/{roomId}.yaml
  func GetRoomBehaviorPath(roomId int, zone string) string {
  	dataFiles := configs.GetFilePathsConfig().DataFiles.String()
  	zoneSafe := rooms.ZoneNameSanitize(zone)
  	return util.FilePath(dataFiles, `/`, `behaviors`, `/`, `rooms`, `/`, zoneSafe, `/`,
  		strconv.Itoa(roomId)+`.yaml`)
  }
  ```

  Also add `"github.com/GoMudEngine/GoMud/internal/rooms"` to the import block. Wait — `helpers.go` already imports `rooms` indirectly? Let me check. Looking at the existing imports in `helpers.go`:

  ```go
  import (
  	"fmt"
  	"os"
  	"strconv"
  	"time"

  	"github.com/GoMudEngine/GoMud/internal/configs"
  	"github.com/GoMudEngine/GoMud/internal/mobs"
  	"github.com/GoMudEngine/GoMud/internal/mudlog"
  	"github.com/GoMudEngine/GoMud/internal/util"
  )
  ```

  The `rooms` package is NOT imported in `helpers.go`. Add it:

  **Find:**
  ```go
  import (
  	"fmt"
  	"os"
  	"strconv"
  	"time"

  	"github.com/GoMudEngine/GoMud/internal/configs"
  	"github.com/GoMudEngine/GoMud/internal/mobs"
  	"github.com/GoMudEngine/GoMud/internal/mudlog"
  	"github.com/GoMudEngine/GoMud/internal/util"
  )
  ```

  **Replace with:**
  ```go
  import (
  	"fmt"
  	"os"
  	"strconv"
  	"time"

  	"github.com/GoMudEngine/GoMud/internal/configs"
  	"github.com/GoMudEngine/GoMud/internal/mobs"
  	"github.com/GoMudEngine/GoMud/internal/mudlog"
  	"github.com/GoMudEngine/GoMud/internal/rooms"
  	"github.com/GoMudEngine/GoMud/internal/util"
  )
  ```

  **Note:** Verify `helpers.go` does not already import `rooms` — if it does (e.g., from another edit), skip this import addition. The `behaviortree` package already imports `rooms` in `conditions.go` and `actions.go`, so there is no circular import risk.

- [ ] **Step 7: Add TryRoomBehavior to helpers.go**

  Add after `GetRoomBehaviorPath`:

  ```go
  // TryRoomBehavior is the main entry point for room behavior tree dispatch.
  // Returns true if a room_command event was intercepted (ctx.Intercepted),
  // or if any other event type was handled (result == Success).
  func TryRoomBehavior(roomId int, event EventContext) bool {
  	room := rooms.LoadRoom(roomId)
  	if room == nil {
  		return false
  	}

  	// Lazy-load tree if not cached
  	tree := GetEngine().GetRoomTree(roomId)
  	if tree == nil {
  		if GetEngine().HasNoRoomTree(roomId) {
  			return false
  		}
  		path := GetRoomBehaviorPath(roomId, room.Zone)
  		if _, err := os.Stat(path); err != nil {
  			GetEngine().SetNoRoomTree(roomId)
  			return false
  		}
  		if err := GetEngine().LoadRoomTree(roomId, path); err != nil {
  			mudlog.Error("TryRoomBehavior", "error",
  				fmt.Sprintf("failed to load room behavior tree for room %d: %v", roomId, err))
  			return false
  		}
  		tree = GetEngine().GetRoomTree(roomId)
  		if tree == nil {
  			return false
  		}
  	}

  	state := EnsureRoomBTreeState(roomId)
  	event.RoomId = roomId

  	ctx := &EvalContext{
  		Event:    event,
  		MobState: state, // reuse MobState field for room state
  		RoomId:   roomId,
  	}
  	result := tree.Evaluate(ctx)

  	// For room_command events, the tree signals interception via ctx.Intercepted.
  	// For all other events, Success means the tree handled it.
  	if event.EventType == "room_command" {
  		return ctx.Intercepted
  	}
  	return result == Success
  }
  ```

- [ ] **Step 8: Verify build**

  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 9: Commit**

  ```bash
  git add internal/behaviortree/engine.go internal/behaviortree/helpers.go internal/behaviortree/room_state.go
  git commit -m "$(cat <<'EOF'
  feat(behaviortree): add room behavior tree infrastructure

  Add room tree cache (roomTrees/noRoomTree) to Engine with load/get/set
  methods. Add TryRoomBehavior entry point and GetRoomBehaviorPath helper.
  Add room_state.go for per-room BehaviorState storage (stored in
  behaviortree package to avoid circular imports with rooms).

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

## Task 4: Wire room events

**Goal:** Wire `room_enter`, `room_exit`, `room_command`, and `room_idle` events to call `TryRoomBehavior` at the appropriate points in the game loop.

**Files:**
- `internal/usercommands/go.go` — wire room_enter and room_exit
- `internal/usercommands/usercommands.go` — wire room_command
- `internal/hooks/NewRound_UserRoundTick.go` — wire room_idle

### Step Group A: Wire room_enter and room_exit in go.go

- [ ] **Step 1: Wire room_exit before MoveToRoom**

  In `internal/usercommands/go.go`, fire `room_exit` just before `rooms.MoveToRoom` is called. Find the block around line 257 where `MoveToRoom` is called.

  **Find:**
  ```go
		if err := rooms.MoveToRoom(user.UserId, destRoom.RoomId); err != nil {
			user.SendText("Oops, couldn't move there!")
		} else {

			scripting.TryRoomScriptEvent(`onExit`, user.UserId, originRoomId)
  ```

  **Replace with:**
  ```go
		// Fire room behavior tree exit event before moving
		behaviortree.TryRoomBehavior(originRoomId, behaviortree.EventContext{
			EventType: "room_exit",
			UserId:    user.UserId,
			Direction: exitName,
		})

		if err := rooms.MoveToRoom(user.UserId, destRoom.RoomId); err != nil {
			user.SendText("Oops, couldn't move there!")
		} else {

			scripting.TryRoomScriptEvent(`onExit`, user.UserId, originRoomId)
  ```

  Verify that `behaviortree` is already imported in `go.go` (it is — see the import block at line 7).

- [ ] **Step 2: Wire room_enter after MoveToRoom succeeds**

  In the same file, fire `room_enter` after the quest engine notification and the onExit script call. Add it right after the `scripting.TryRoomScriptEvent('onExit', ...)` line.

  **Find:**
  ```go
			scripting.TryRoomScriptEvent(`onExit`, user.UserId, originRoomId)

			// Quest engine: room_enter notification
  ```

  **Replace with:**
  ```go
			scripting.TryRoomScriptEvent(`onExit`, user.UserId, originRoomId)

			// Fire room behavior tree enter event in destination room
			behaviortree.TryRoomBehavior(destRoom.RoomId, behaviortree.EventContext{
				EventType: "room_enter",
				UserId:    user.UserId,
				Direction: exitName,
			})

			// Quest engine: room_enter notification
  ```

- [ ] **Step 3: Verify build**

  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

### Step Group B: Wire room_command in usercommands.go

- [ ] **Step 4: Wire room_command before TryRoomScripts**

  In `internal/usercommands/usercommands.go`, insert a `TryRoomBehavior` call for `room_command` events just before the existing `TryRoomScripts` call. If the behavior tree intercepts, skip the JS room scripts entirely.

  **Find:**
  ```go
		if !skipScript {
			// Instead of calling scripting.TryRoomCommand directly,
			// use our new function that sends GMCP notifications for blocked directions
			handled, err := TryRoomScripts(cmd+` `+rest, alias, rest, userId)
			if handled {
				return true, err
			}
		}
  ```

  **Replace with:**
  ```go
		if !skipScript {
			// Room behavior tree: check before JS scripts
			if behaviortree.TryRoomBehavior(user.Character.RoomId, behaviortree.EventContext{
				EventType: "room_command",
				UserId:    userId,
				Command:   alias,
				Rest:      rest,
			}) {
				return true, nil
			}

			// Instead of calling scripting.TryRoomCommand directly,
			// use our new function that sends GMCP notifications for blocked directions
			handled, err := TryRoomScripts(cmd+` `+rest, alias, rest, userId)
			if handled {
				return true, err
			}
		}
  ```

  Verify that `behaviortree` is already imported in `usercommands.go`. Check the import block — if not present, add `"github.com/GoMudEngine/GoMud/internal/behaviortree"`.

- [ ] **Step 5: Verify build**

  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

### Step Group C: Wire room_idle in UserRoundTick

- [ ] **Step 6: Wire room_idle alongside existing TryRoomIdleEvent**

  In `internal/hooks/NewRound_UserRoundTick.go`, add a `TryRoomBehavior` call for `room_idle` alongside the existing `TryRoomIdleEvent` call. If the behavior tree handles idle, suppress idle messages the same way the JS script does.

  **Find:**
  ```go
			allowIdleMessages := true
			if handled, err := scripting.TryRoomIdleEvent(roomId); err == nil {
				if handled { // For this event, handled represents whether to reject the move.
					allowIdleMessages = false
				}
			}
  ```

  **Replace with:**
  ```go
			allowIdleMessages := true

			// Room behavior tree idle event (checked before JS scripts)
			if behaviortree.TryRoomBehavior(roomId, behaviortree.EventContext{
				EventType: "room_idle",
			}) {
				allowIdleMessages = false
			}

			if allowIdleMessages {
				if handled, err := scripting.TryRoomIdleEvent(roomId); err == nil {
					if handled {
						allowIdleMessages = false
					}
				}
			}
  ```

  Add `"github.com/GoMudEngine/GoMud/internal/behaviortree"` to the import block in `NewRound_UserRoundTick.go`.

  **Find:**
  ```go
  import (
  	"fmt"
  	"math"
  	"strconv"
  	"strings"

  	"github.com/GoMudEngine/GoMud/internal/buffs"
  	"github.com/GoMudEngine/GoMud/internal/configs"
  ```

  **Replace with:**
  ```go
  import (
  	"fmt"
  	"math"
  	"strconv"
  	"strings"

  	"github.com/GoMudEngine/GoMud/internal/behaviortree"
  	"github.com/GoMudEngine/GoMud/internal/buffs"
  	"github.com/GoMudEngine/GoMud/internal/configs"
  ```

- [ ] **Step 7: Verify build**

  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 8: Commit**

  ```bash
  git add internal/usercommands/go.go internal/usercommands/usercommands.go internal/hooks/NewRound_UserRoundTick.go
  git commit -m "$(cat <<'EOF'
  feat(behaviortree): wire room_enter, room_exit, room_command, room_idle events

  Fire TryRoomBehavior at four lifecycle points:
  - room_exit: before MoveToRoom in go.go
  - room_enter: after MoveToRoom succeeds in go.go
  - room_command: before TryRoomScripts in usercommands.go (intercepts)
  - room_idle: before TryRoomIdleEvent in UserRoundTick (suppresses idle msgs)

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

## Task 5: Add new conditions and actions

**Goal:** Add the 3 new conditions and 9 new actions needed by room behavior trees.

**Files:**
- `internal/behaviortree/conditions.go` — add `command_matches`, `command_rest_contains`, `mob_in_room`
- `internal/behaviortree/actions.go` — add `mob_say`, `mob_emote`, `grant_mutation`, `send_user_text`, `send_room_text`, `intercept`, `remove_buff`, `move_player`

**Note:** `give_gold` already exists as `actGiveGold` in `actions.go`. No need to add it.

### Step Group A: New conditions

- [ ] **Step 1: Register new conditions**

  In `internal/behaviortree/conditions.go`, add the three new conditions to the registry in `init()`.

  **Find:**
  ```go
  	conditionRegistry["multiple_enemies"] = condMultipleEnemies
  }
  ```

  **Replace with:**
  ```go
  	conditionRegistry["multiple_enemies"] = condMultipleEnemies

  	// Room behavior tree conditions
  	conditionRegistry["command_matches"] = condCommandMatches
  	conditionRegistry["command_rest_contains"] = condCommandRestContains
  	conditionRegistry["mob_in_room"] = condMobInRoom
  }
  ```

- [ ] **Step 2: Implement condCommandMatches**

  Add at the bottom of `internal/behaviortree/conditions.go`:

  ```go
  // condCommandMatches checks if the event's Command field matches any of the
  // given command names. Used in room_command event handlers.
  // params: commands ([]string) — list of command names to match against
  func condCommandMatches(params map[string]any, ctx *EvalContext) Result {
  	cmdsRaw, ok := params["commands"].([]any)
  	if !ok || len(cmdsRaw) == 0 {
  		// Also support a single "command" string param
  		single := getStringParam(params, "command")
  		if single != "" && strings.EqualFold(ctx.Event.Command, single) {
  			return Success
  		}
  		return Failure
  	}
  	cmdLower := strings.ToLower(ctx.Event.Command)
  	for _, c := range cmdsRaw {
  		if s, ok := c.(string); ok && strings.ToLower(s) == cmdLower {
  			return Success
  		}
  	}
  	return Failure
  }
  ```

- [ ] **Step 3: Implement condCommandRestContains**

  Add at the bottom of `internal/behaviortree/conditions.go`:

  ```go
  // condCommandRestContains checks if the event's Rest field contains the given
  // substring (case-insensitive). Used to match arguments in room_command events.
  // params: text (string) — substring to search for in Rest
  func condCommandRestContains(params map[string]any, ctx *EvalContext) Result {
  	text := getStringParam(params, "text")
  	if text == "" {
  		return Failure
  	}
  	if strings.Contains(strings.ToLower(ctx.Event.Rest), strings.ToLower(text)) {
  		return Success
  	}
  	return Failure
  }
  ```

- [ ] **Step 4: Implement condMobInRoom**

  Add at the bottom of `internal/behaviortree/conditions.go`:

  ```go
  // condMobInRoom checks if a mob with the given template ID exists in the room.
  // params: mob_id (int) — the mob template ID to look for
  func condMobInRoom(params map[string]any, ctx *EvalContext) Result {
  	mobId := getIntParam(params, "mob_id")
  	if mobId == 0 {
  		return Failure
  	}
  	room := rooms.LoadRoom(ctx.RoomId)
  	if room == nil {
  		return Failure
  	}
  	for _, instId := range room.GetMobs(rooms.FindAll) {
  		m := mobs.GetInstance(instId)
  		if m != nil && int(m.MobId) == mobId {
  			return Success
  		}
  	}
  	return Failure
  }
  ```

- [ ] **Step 5: Verify build**

  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

### Step Group B: New actions

- [ ] **Step 6: Register new actions**

  In `internal/behaviortree/actions.go`, add the new actions to the registry in `init()`.

  **Find:**
  ```go
  	actionRegistry["grant_quest_to_user"] = actGrantQuest // alias for grant_quest
  }
  ```

  **Replace with:**
  ```go
  	actionRegistry["grant_quest_to_user"] = actGrantQuest // alias for grant_quest

  	// Room behavior tree actions
  	actionRegistry["mob_say"] = actMobSay
  	actionRegistry["mob_emote"] = actMobEmote
  	actionRegistry["grant_mutation"] = actGrantMutation
  	actionRegistry["send_user_text"] = actSendUserText
  	actionRegistry["send_room_text"] = actSendRoomText
  	actionRegistry["intercept"] = actIntercept
  	actionRegistry["remove_buff"] = actRemoveBuff
  	actionRegistry["move_player"] = actMovePlayer
  }
  ```

- [ ] **Step 7: Implement actMobSay and actMobEmote**

  Add at the bottom of `internal/behaviortree/actions.go`:

  ```go
  // actMobSay finds a mob in the current room by template ID and makes it say text.
  // params: mob_id (int), text (string)
  func actMobSay(params map[string]any, ctx *EvalContext) Result {
  	mobId := getIntParam(params, "mob_id")
  	if mobId == 0 {
  		return Failure
  	}
  	text := getStringParam(params, "text")
  	if text == "" {
  		return Failure
  	}
  	room := rooms.LoadRoom(ctx.RoomId)
  	if room == nil {
  		return Failure
  	}
  	for _, instId := range room.GetMobs(rooms.FindAll) {
  		m := mobs.GetInstance(instId)
  		if m != nil && int(m.MobId) == mobId {
  			m.Command("say " + text)
  			return Success
  		}
  	}
  	return Failure
  }

  // actMobEmote finds a mob in the current room by template ID and makes it emote.
  // params: mob_id (int), text (string)
  func actMobEmote(params map[string]any, ctx *EvalContext) Result {
  	mobId := getIntParam(params, "mob_id")
  	if mobId == 0 {
  		return Failure
  	}
  	text := getStringParam(params, "text")
  	if text == "" {
  		return Failure
  	}
  	room := rooms.LoadRoom(ctx.RoomId)
  	if room == nil {
  		return Failure
  	}
  	for _, instId := range room.GetMobs(rooms.FindAll) {
  		m := mobs.GetInstance(instId)
  		if m != nil && int(m.MobId) == mobId {
  			m.Command("emote " + text)
  			return Success
  		}
  	}
  	return Failure
  }
  ```

- [ ] **Step 8: Implement actGrantMutation**

  Add at the bottom of `internal/behaviortree/actions.go`. This rolls a random mutation and grants it, matching the quest engine's `GiveMutation` behavior.

  First, add `"github.com/GoMudEngine/GoMud/internal/mutations"` to the import block in `actions.go`:

  **Find:**
  ```go
  	"github.com/GoMudEngine/GoMud/internal/items"
  	"github.com/GoMudEngine/GoMud/internal/mobs"
  	"github.com/GoMudEngine/GoMud/internal/rooms"
  ```

  **Replace with:**
  ```go
  	"github.com/GoMudEngine/GoMud/internal/items"
  	"github.com/GoMudEngine/GoMud/internal/mobs"
  	"github.com/GoMudEngine/GoMud/internal/mutations"
  	"github.com/GoMudEngine/GoMud/internal/rooms"
  ```

  Then add the implementation:

  ```go
  // actGrantMutation rolls a random mutation from the weighted pool and grants
  // it to the triggering user at level 1. If mutation_id is specified, grants
  // that specific mutation instead of rolling randomly.
  // params: mutation_id (string, optional)
  func actGrantMutation(params map[string]any, ctx *EvalContext) Result {
  	user := users.GetByUserId(ctx.Event.UserId)
  	if user == nil {
  		return Failure
  	}
  	if user.Character.Mutations == nil {
  		user.Character.Mutations = make(map[string]int)
  	}

  	mutId := getStringParam(params, "mutation_id")
  	if mutId == "" {
  		// Roll a random mutation
  		pool := mutations.GetWeightedPool(user.Character.Mutations)
  		if len(pool) == 0 {
  			return Failure
  		}
  		mutId = mutations.RollAcquisition(pool)
  		if mutId == "" {
  			return Failure
  		}
  	}

  	if _, exists := user.Character.Mutations[mutId]; !exists {
  		user.Character.Mutations[mutId] = 1
  		user.Character.Validate()
  		return Success
  	}
  	return Failure // already owned
  }
  ```

- [ ] **Step 9: Implement actSendUserText and actSendRoomText**

  Add at the bottom of `internal/behaviortree/actions.go`:

  ```go
  // actSendUserText sends text directly to the triggering user.
  // params: text (string)
  func actSendUserText(params map[string]any, ctx *EvalContext) Result {
  	user := users.GetByUserId(ctx.Event.UserId)
  	if user == nil {
  		return Failure
  	}
  	text := getStringParam(params, "text")
  	if text == "" {
  		return Failure
  	}
  	user.SendText(text)
  	return Success
  }

  // actSendRoomText sends text to all players in the room, optionally
  // excluding the triggering user.
  // params: text (string), exclude_user (bool, default false)
  func actSendRoomText(params map[string]any, ctx *EvalContext) Result {
  	room := rooms.LoadRoom(ctx.RoomId)
  	if room == nil {
  		return Failure
  	}
  	text := getStringParam(params, "text")
  	if text == "" {
  		return Failure
  	}
  	excludeStr := getStringParam(params, "exclude_user")
  	if excludeStr == "true" && ctx.Event.UserId > 0 {
  		room.SendText(text, ctx.Event.UserId)
  	} else {
  		room.SendText(text)
  	}
  	return Success
  }
  ```

- [ ] **Step 10: Implement actIntercept**

  Add at the bottom of `internal/behaviortree/actions.go`:

  ```go
  // actIntercept sets ctx.Intercepted to true, signaling that the room behavior
  // tree has consumed this command and it should not be dispatched further.
  func actIntercept(params map[string]any, ctx *EvalContext) Result {
  	ctx.Intercepted = true
  	return Success
  }
  ```

- [ ] **Step 11: Implement actRemoveBuff**

  Add at the bottom of `internal/behaviortree/actions.go`:

  ```go
  // actRemoveBuff removes a buff from the triggering user.
  // params: buff_id (int)
  func actRemoveBuff(params map[string]any, ctx *EvalContext) Result {
  	user := users.GetByUserId(ctx.Event.UserId)
  	if user == nil {
  		return Failure
  	}
  	buffId := getIntParam(params, "buff_id")
  	if buffId == 0 {
  		return Failure
  	}
  	user.Character.RemoveBuff(buffId)
  	return Success
  }
  ```

- [ ] **Step 12: Implement actMovePlayer**

  Add at the bottom of `internal/behaviortree/actions.go`:

  ```go
  // actMovePlayer teleports the triggering user to a different room.
  // params: room_id (int)
  func actMovePlayer(params map[string]any, ctx *EvalContext) Result {
  	userId := ctx.Event.UserId
  	if userId == 0 {
  		return Failure
  	}
  	roomId := getIntParam(params, "room_id")
  	if roomId == 0 {
  		return Failure
  	}
  	if err := rooms.MoveToRoom(userId, roomId); err != nil {
  		return Failure
  	}
  	return Success
  }
  ```

- [ ] **Step 13: Add mob_say and mob_emote to delayedActions**

  These actions produce visible output and should respect delay timing (either static or perception-scaled).

  **Find:**
  ```go
  var delayedActions = map[string]bool{
  	"respond":     true,
  	"say":         true,
  	"emote":       true,
  	"attack":      true,
  	"flee":        true,
  	"cast":        true,
  	"move":        true,
  	"add_buff":    true,
  	"command_mob": true,
  }
  ```

  **Replace with:**
  ```go
  var delayedActions = map[string]bool{
  	"respond":     true,
  	"say":         true,
  	"emote":       true,
  	"attack":      true,
  	"flee":        true,
  	"cast":        true,
  	"move":        true,
  	"add_buff":    true,
  	"command_mob": true,
  	"mob_say":     true,
  	"mob_emote":   true,
  }
  ```

- [ ] **Step 14: Verify build**

  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 15: Commit**

  ```bash
  git add internal/behaviortree/conditions.go internal/behaviortree/actions.go
  git commit -m "$(cat <<'EOF'
  feat(behaviortree): add room behavior tree conditions and actions

  Conditions: command_matches, command_rest_contains, mob_in_room
  Actions: mob_say, mob_emote, grant_mutation, send_user_text,
  send_room_text, intercept, remove_buff, move_player

  These enable room behavior trees to inspect player commands, interact
  with mobs in the room, send text, grant mutations, and intercept
  command dispatch.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

## Task 6: Migrate Sanctum Basin Tutorial Rooms (9 rooms)

**Goal:** Replace all 9 Sanctum Basin room JS scripts with room behavior tree YAML files. Each room gets its own YAML behavior tree in `_datafiles/world/dogmud/behaviors/rooms/sanctum_basin/`.

**Prerequisites:** Tasks 1-5 complete (room behavior tree engine, event wiring, conditions/actions)

**Available conditions:** `player_has_quest`, `player_missing_quest`, `player_has_item`, `command_matches`, `command_rest_contains`, `mob_in_room`, `state_equals`, `state_greater_than`, `players_in_room`

**Available actions:** `mob_say`, `mob_emote`, `grant_quest`, `give_item`, `give_gold`, `send_user_text`, `send_room_text`, `intercept`, `set_room_locked`, `set_state`, `increment_state`, `decrement_state`, `grant_mutation`, `move_player`, `remove_buff`

### Step Group A: Room 102 — Basin Gate (Warden)

- [ ] **Step 1: Create room 102 behavior tree YAML**

  Create `_datafiles/world/dogmud/behaviors/rooms/sanctum_basin/102.yaml`:

  ```yaml
  # Basin Gate - Room 102
  # Basin Warden (mob 56) checks quest state and gives appropriate dialogue.
  # Gate starts locked (difficulty 255). Unlocked when player reaches 1-cave.
  # Events: room_enter, room_command

  tree:
    type: selector
    children:

      # ── ENTER: already graduated — brief acknowledgment ──
      - type: sequence
        event: room_enter
        children:
          - type: condition
            check: player_has_quest
            quest: "1-end"
          - type: condition
            check: mob_in_room
            mob_id: 56
          - type: action
            do: set_room_locked
            direction: south
            locked: "false"
          - type: action
            do: mob_emote
            mob_id: 56
            text: gives a brief nod of recognition.
            delay: 0.5
          - type: action
            do: mob_say
            mob_id: 56
            text: Safe travels out there.
            delay: 1.5

      # ── ENTER: has warden speech but hasn't left yet ──
      - type: sequence
        event: room_enter
        children:
          - type: condition
            check: player_has_quest
            quest: "1-warden"
          - type: condition
            check: mob_in_room
            mob_id: 56
          - type: action
            do: set_room_locked
            direction: south
            locked: "false"
          - type: action
            do: mob_emote
            mob_id: 56
            text: nods toward the open gate.
            delay: 0.5

      # ── ENTER: first-time graduation — warden speech ──
      - type: sequence
        event: room_enter
        children:
          - type: condition
            check: player_has_quest
            quest: "1-cave"
          - type: condition
            check: player_missing_quest
            quest: "1-warden"
          - type: condition
            check: mob_in_room
            mob_id: 56
          - type: action
            do: mob_say
            mob_id: 56
            text: >-
              Your record is complete. Six trials, six instructors
              -- and the cave.
            delay: 1.0
          - type: action
            do: mob_say
            mob_id: 56
            text: >-
              The Basin Warden has one function: to ensure that no
              one leaves Sanctum Basin unprepared. You are prepared.
            delay: 2.5
          - type: action
            do: mob_say
            mob_id: 56
            text: >-
              The gate is open. Whatever you find south of here
              -- remember what you learned in the basin.
            delay: 4.0
          - type: action
            do: mob_say
            mob_id: 56
            text: >-
              One last thing: the world is larger than what six
              instructors can cover. Type <ansi fg="command">help</ansi>
              to see what documentation is available -- there is
              more to learn out there than what we teach here.
            delay: 5.5
          - type: action
            do: mob_say
            mob_id: 56
            text: >-
              The south road reaches Confluence within a day's walk.
              Follow the river north from there and you will find
              New Plymouth -- the largest settlement in this part of
              Gaius. That is where most people head first.
            delay: 7.0
          - type: action
            do: mob_emote
            mob_id: 56
            text: steps aside and gestures toward the road south.
            delay: 9.0
          - type: action
            do: set_room_locked
            direction: south
            locked: "false"
          - type: action
            do: grant_quest
            quest: "1-warden"
          - type: action
            do: grant_quest
            quest: "1-end"

      # ── ENTER: hasn't started trials ──
      - type: sequence
        event: room_enter
        children:
          - type: condition
            check: player_missing_quest
            quest: "1-start"
          - type: condition
            check: mob_in_room
            mob_id: 56
          - type: action
            do: mob_say
            mob_id: 56
            text: Hold.
            delay: 1.0
          - type: action
            do: mob_say
            mob_id: 56
            text: >-
              You have not begun the Sanctum Trials. Speak with the
              Chrysalis Priest in the Academy Hall -- north to Town
              Square, then north and east.
            delay: 2.0

      # ── ENTER: in progress — short check-in ──
      - type: sequence
        event: room_enter
        children:
          - type: condition
            check: player_has_quest
            quest: "1-start"
          - type: condition
            check: player_missing_quest
            quest: "1-cave"
          - type: condition
            check: mob_in_room
            mob_id: 56
          - type: action
            do: mob_emote
            mob_id: 56
            text: >-
              glances at you and returns their gaze to the road.
            delay: 0.5
          - type: action
            do: mob_say
            mob_id: 56
            text: Not yet. Finish your trials and come back.
            delay: 1.5
          - type: action
            do: mob_say
            mob_id: 56
            text: >-
              Type <ansi fg="command">quest</ansi> if you have
              lost your way.
            delay: 2.5

      # ── COMMAND: talk — graduated ──
      - type: sequence
        event: room_command
        children:
          - type: condition
            check: command_matches
            command: talk
          - type: condition
            check: mob_in_room
            mob_id: 56
          - type: condition
            check: player_has_quest
            quest: "1-end"
          - type: action
            do: mob_say
            mob_id: 56
            text: The world is wide. Come back if you need to.
          - type: action
            do: intercept

      # ── COMMAND: talk — cave done ──
      - type: sequence
        event: room_command
        children:
          - type: condition
            check: command_matches
            command: talk
          - type: condition
            check: mob_in_room
            mob_id: 56
          - type: condition
            check: player_has_quest
            quest: "1-cave"
          - type: action
            do: mob_say
            mob_id: 56
            text: The gate is open. Move when you are ready.
          - type: action
            do: intercept

      # ── COMMAND: talk — trials in progress ──
      - type: sequence
        event: room_command
        children:
          - type: condition
            check: command_matches
            command: talk
          - type: condition
            check: mob_in_room
            mob_id: 56
          - type: condition
            check: player_has_quest
            quest: "1-start"
          - type: action
            do: mob_say
            mob_id: 56
            text: >-
              Your trials are not complete. Type
              <ansi fg="command">quest</ansi> to see what remains.
          - type: action
            do: intercept

      # ── COMMAND: talk — hasn't started ──
      - type: sequence
        event: room_command
        children:
          - type: condition
            check: command_matches
            command: talk
          - type: condition
            check: mob_in_room
            mob_id: 56
          - type: action
            do: mob_say
            mob_id: 56
            text: >-
              Find the Chrysalis Priest in the Academy Hall. North
              to Town Square, then north and east.
          - type: action
            do: intercept
  ```

- [ ] **Step 2: Delete room 102 JS**

  ```bash
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/rooms/sanctum_basin/102.js
  ```

### Step Group B: Rooms 106, 108, 109, 111, 116 — Command Detect + Quest Grant

These five rooms share a common pattern: onEnter introduces a lesson, onCommand detects a specific command, onIdle waits N ticks then grants the quest step. Each gets its own YAML file. Due to the size of these files, see the full YAML content inline below.

For rooms 106, 108, 109, 111, 116: create YAML files following the exact same pattern as Room 102 above, translating the JS logic line-by-line. The key pattern for each:

1. `room_enter` sequence: check prerequisite quest + missing arrival quest + mob present, then NPC dialogue with delays, then `grant_quest`
2. `room_command` sequences: detect the target command, set state counter to 1
3. `room_command` talk sequences: NPC responses based on quest state, with `intercept`
4. `room_idle` sequences: check state counter reaching threshold, NPC reaction dialogue, `grant_quest`, reset counter
5. `room_idle` increment sequences: increment counter when >0

**The full YAML for each room is provided in the detailed spec. Due to the size of these files (each 100-300 lines), they are written during implementation using the JS as the source of truth. The JS files have already been read and analyzed above.**

- [ ] **Step 3: Create room 106 YAML** — West Meadow (Guide Fen, mob 54). Two command tracks: forage→wilderness_forage, track→wilderness_track. Use `forage_pending` and `track_pending` state counters, threshold 2.

- [ ] **Step 4: Delete room 106 JS**

- [ ] **Step 5: Create room 108 YAML** — Market Street (Merchant Adela, mob 63). Two command tracks: buy→shopping_buy, equip→shopping_equip. Use `buy_pending` and `equip_pending` state counters, threshold 2.

- [ ] **Step 6: Delete room 108 JS**

- [ ] **Step 7: Create room 109 YAML** — The Forge (Blacksmith Korvath, mob 52). Gives items 40001+40002 on enter. Craft detection with 5-tick threshold. Success/failure selector based on `player_has_item` 10009.

- [ ] **Step 8: Delete room 109 JS**

- [ ] **Step 9: Create room 111 YAML** — Alchemist's Workshop (Alchemist Yenna, mob 53). Gives items 40004×2+40006 on enter. Craft detection with 5-tick threshold. Success/failure selector based on `player_has_item` 30036.

- [ ] **Step 10: Delete room 111 JS**

- [ ] **Step 11: Create room 116 YAML** — Observatory (Elder Saris, mob 55). Cast detection for `cast` or `conviction-spike` commands. 2-tick threshold.

- [ ] **Step 12: Delete room 116 JS**

### Step Group C: Room 114 — Training Yard (Mob Death Tracking)

- [ ] **Step 13: Create room 114 YAML** — Training Yard (Combat Trainer, mob 51). Monitors training dummy (mob 65) death via `mob_in_room` with `negate: true`. Grants `1-combat_defeat` when dummy is gone.

- [ ] **Step 13b: Add negate support to ConditionNode**

  In `internal/behaviortree/types.go`, add `Negate bool` field to `ConditionNode` struct and modify `Evaluate` to invert the result when `Negate` is true.

  **Find:**
  ```go
  type ConditionNode struct {
  	Params map[string]any
  	Fn     ConditionFunc
  }
  ```

  **Replace with:**
  ```go
  type ConditionNode struct {
  	Params map[string]any
  	Fn     ConditionFunc
  	Negate bool
  }
  ```

  **Find:**
  ```go
  func (n *ConditionNode) Evaluate(ctx *EvalContext) Result {
  	return n.Fn(n.Params, ctx)
  }
  ```

  **Replace with:**
  ```go
  func (n *ConditionNode) Evaluate(ctx *EvalContext) Result {
  	result := n.Fn(n.Params, ctx)
  	if n.Negate {
  		if result == Success {
  			return Failure
  		}
  		return Success
  	}
  	return result
  }
  ```

  In `internal/behaviortree/loader.go`, extract `negate` from params when building condition nodes:

  **Find:**
  ```go
  		return &ConditionNode{
  			Params: cleanParams(raw),
  			Fn:     fn,
  		}, nil
  ```

  **Replace with:**
  ```go
  		params := cleanParams(raw)
  		negate := false
  		if v, ok := params["negate"]; ok {
  			if b, ok := v.(bool); ok {
  				negate = b
  			}
  			delete(params, "negate")
  		}
  		return &ConditionNode{
  			Params: params,
  			Fn:     fn,
  			Negate: negate,
  		}, nil
  ```

- [ ] **Step 14: Delete room 114 JS**

### Step Group D: Room 120 — Boss Cave (Mob Death Tracking)

- [ ] **Step 15: Create room 120 YAML** — Boss Cave (Aberrant, mob 69). Atmospheric text on enter. Tracks `boss_engaged` state. Grants `1-cave` when aberrant is gone (`mob_in_room` + `negate: true`).

- [ ] **Step 16: Delete room 120 JS**

### Step Group E: Room 113 — Academy Hall (Ceremony)

- [ ] **Step 17: Create room 113 YAML** — Academy Hall (Chrysalis Priest, mob 50). The largest room tree:
  - Three `room_enter` branches: silent rite (ceremony running), fallback (no priest), full ceremony
  - Full ceremony: lock 4 exits, 18 timed priest lines (1.0s to 33.5s delays), grant mutation + 10 gold + quest tokens, start ceremony_ticks counter
  - `room_command` movement block during ceremony, mosaic get/take block, mosaic/map look handler (full ASCII map), talk/greet responses
  - `room_idle` unlock timer: unlock 4 exits at tick 6, increment counter when >0

  **Full YAML content** (all 18 priest lines with exact delays from JS):

  ```yaml
  # Academy Hall - Room 113
  # Chrysalis Priest (mob 50) delivers the Awakening Rite ceremony.
  # State keys: ceremony_ticks (0 = idle, >0 = counting to unlock)

  tree:
    type: selector
    children:

      # ── ENTER: ceremony already running — silent rite ──
      - type: sequence
        event: room_enter
        children:
          - type: condition
            check: player_missing_quest
            quest: "1-mutation"
          - type: condition
            check: state_greater_than
            key: ceremony_ticks
            value: 0
          - type: action
            do: grant_mutation
          - type: action
            do: give_gold
            amount: 10
          - type: action
            do: grant_quest
            quest: "1-start"
          - type: action
            do: grant_quest
            quest: "1-mutation"

      # ── ENTER: priest absent fallback ──
      - type: sequence
        event: room_enter
        children:
          - type: condition
            check: player_missing_quest
            quest: "1-mutation"
          - type: condition
            check: mob_in_room
            mob_id: 50
            negate: true
          - type: action
            do: grant_mutation
          - type: action
            do: give_gold
            amount: 10
          - type: action
            do: grant_quest
            quest: "1-start"
          - type: action
            do: grant_quest
            quest: "1-mutation"

      # ── ENTER: full ceremony ──
      - type: sequence
        event: room_enter
        children:
          - type: condition
            check: player_missing_quest
            quest: "1-mutation"
          - type: condition
            check: mob_in_room
            mob_id: 50
          - type: action
            do: set_room_locked
            direction: north
            locked: "true"
          - type: action
            do: set_room_locked
            direction: south
            locked: "true"
          - type: action
            do: set_room_locked
            direction: east
            locked: "true"
          - type: action
            do: set_room_locked
            direction: west
            locked: "true"
          - type: action
            do: mob_say
            mob_id: 50
            text: Be still for a moment.
            delay: 1.0
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              Type <ansi fg="command">look</ansi> at any time to
              examine your surroundings. Type
              <ansi fg="command">help</ansi> if you are unsure what
              commands are available -- there is documentation for
              most things.
            delay: 2.5
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              You are newly arrived in Gaius. You may not yet
              understand what that means.
            delay: 5.0
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              The world here is not like any world you have come
              from. Gaius is made of belief -- belief shapes the
              land, the creatures, the rules of cause and effect.
            delay: 6.5
          - type: action
            do: mob_say
            mob_id: 50
            text: And it shapes people.
            delay: 8.5
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              The Chrysalis has been cataloguing that shaping for
              four hundred years. We know what to expect. You may
              not.
            delay: 10.0
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              The Chrysalis is the name we give to the form between
              what was and what will be. Every person who has passed
              through this hall was once standing where you are
              standing.
            delay: 12.0
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              In Gaius, that becoming is not a metaphor. The body
              remembers stress, conflict, effort. Over time, it
              changes. We call these changes Mutations.
            delay: 14.5
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              Every Mutation is the body saying yes to something.
              Yes to strength, yes to perception, yes to endurance.
              There are costs -- the Chrysalis does not lie about
              that -- but a Priest does not dwell on costs. We dwell
              on what you are becoming.
            delay: 16.5
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              The first Mutation is always a gift -- or a warning.
              The Chrysalis performs the Awakening Rite to initiate
              it before the body does something unexpected on its
              own.
            delay: 18.5
          - type: action
            do: mob_emote
            mob_id: 50
            text: >-
              steps forward and places two fingers lightly on your
              forehead. The air around you hums very quietly.
            delay: 20.5
          - type: action
            do: mob_emote
            mob_id: 50
            text: withdraws their hand. The hum fades.
            delay: 22.5
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              It has begun. You will feel the change over time --
              in combat, in effort, in moments of stress. It is
              part of you now.
            delay: 24.0
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              You can type <ansi fg="command">mutations</ansi> at
              any time to see what the change has wrought.
            delay: 25.5
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              Now. Before you leave this place, you have six trials
              to complete. Each instructor teaches a different
              skill.
            delay: 27.0
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              First: visit the Merchant on Market Street, one step
              south of here. She will show you how commerce works
              here.
            delay: 28.5
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              After that, find the Combat Trainer in the Training
              Yard -- go east, just beyond here. Then the Forge to
              the east, the Alchemist west of the well, the
              Wilderness Guide in the West Meadow, and finally the
              Elder at the Observatory above.
            delay: 30.0
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              When all six are done, the Basin Warden at the south
              gate will let you pass. Type
              <ansi fg="command">quest</ansi> at any time if you
              lose your place.
            delay: 32.0
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              There is a map of the wider world set into the floor.
              Type <ansi fg="command">look mosaic</ansi> to examine
              it before you leave.
            delay: 33.5
          - type: action
            do: grant_mutation
          - type: action
            do: give_gold
            amount: 10
          - type: action
            do: grant_quest
            quest: "1-start"
          - type: action
            do: grant_quest
            quest: "1-mutation"
          - type: action
            do: set_state
            key: ceremony_ticks
            value: 1

      # ── COMMAND: movement during ceremony ──
      - type: sequence
        event: room_command
        children:
          - type: condition
            check: state_greater_than
            key: ceremony_ticks
            value: 0
          - type: condition
            check: command_matches
            commands: [north, south, east, west, n, s, e, w, go]
          - type: condition
            check: mob_in_room
            mob_id: 50
          - type: action
            do: mob_say
            mob_id: 50
            text: The Rite is not yet complete. A moment more.
          - type: action
            do: intercept

      # ── COMMAND: get/take mosaic or map ──
      - type: sequence
        event: room_command
        children:
          - type: condition
            check: command_matches
            commands: [get, take, grab]
          - type: condition
            check: command_rest_contains
            text: mosaic
          - type: action
            do: send_user_text
            text: >-
              The mosaic is set into the floor. It is not going
              anywhere.
          - type: action
            do: intercept

      - type: sequence
        event: room_command
        children:
          - type: condition
            check: command_matches
            commands: [get, take, grab]
          - type: condition
            check: command_rest_contains
            text: map
          - type: action
            do: send_user_text
            text: >-
              The mosaic is set into the floor. It is not going
              anywhere.
          - type: action
            do: intercept

      # ── COMMAND: look mosaic — world map ──
      - type: sequence
        event: room_command
        children:
          - type: condition
            check: command_matches
            commands: [look, examine]
          - type: condition
            check: command_rest_contains
            text: mosaic
          - type: action
            do: send_user_text
            text: ""
          - type: action
            do: send_user_text
            text: "+------------------------------------------------------+"
          - type: action
            do: send_user_text
            text: "|  THE WINDWARD MARCHES                                |"
          - type: action
            do: send_user_text
            text: "|  (partial -- much of Gaius remains uncharted)        |"
          - type: action
            do: send_user_text
            text: "+------------------------------------------------------+"
          - type: action
            do: send_user_text
            text: "|                                                      |"
          - type: action
            do: send_user_text
            text: "|  [NEW PLYMOUTH *]               [? ? ?]             |"
          - type: action
            do: send_user_text
            text: "|        |         \\                                   |"
          - type: action
            do: send_user_text
            text: "|   [TIDEMARK]      \\         [STILLWATER]            |"
          - type: action
            do: send_user_text
            text: "|        |           \\              |                 |"
          - type: action
            do: send_user_text
            text: "|   [GREENFORD]   [AMBER VALLEY]   |                 |"
          - type: action
            do: send_user_text
            text: "|         \\              \\          |                 |"
          - type: action
            do: send_user_text
            text: "|          \\---------[CONFLUENCE]                    |"
          - type: action
            do: send_user_text
            text: "|                         |                           |"
          - type: action
            do: send_user_text
            text: "|                   . . . . . . .                     |"
          - type: action
            do: send_user_text
            text: "|                  .   unmapped   .                   |"
          - type: action
            do: send_user_text
            text: "|                   . . . . . . .                     |"
          - type: action
            do: send_user_text
            text: "|                    /~~~~~~~~~~~\\                    |"
          - type: action
            do: send_user_text
            text: "|                   |  SANCTUM    |                   |"
          - type: action
            do: send_user_text
            text: "|                   |   BASIN     |  <- you are here  |"
          - type: action
            do: send_user_text
            text: "|                    \\~~~~~~~~~~~/                    |"
          - type: action
            do: send_user_text
            text: "+------------------------------------------------------+"
          - type: action
            do: send_user_text
            text: "| * city  ~ plateau rim  . unmapped  \\ road           |"
          - type: action
            do: send_user_text
            text: "+------------------------------------------------------+"
          - type: action
            do: send_user_text
            text: ""
          - type: action
            do: intercept

      # ── COMMAND: look map (without "mosaic") ──
      - type: sequence
        event: room_command
        children:
          - type: condition
            check: command_matches
            commands: [look, examine]
          - type: condition
            check: command_rest_contains
            text: map
          - type: action
            do: send_user_text
            text: ""
          - type: action
            do: send_user_text
            text: "+------------------------------------------------------+"
          - type: action
            do: send_user_text
            text: "|  THE WINDWARD MARCHES                                |"
          - type: action
            do: send_user_text
            text: "|  (partial -- much of Gaius remains uncharted)        |"
          - type: action
            do: send_user_text
            text: "+------------------------------------------------------+"
          - type: action
            do: send_user_text
            text: "|                                                      |"
          - type: action
            do: send_user_text
            text: "|  [NEW PLYMOUTH *]               [? ? ?]             |"
          - type: action
            do: send_user_text
            text: "|        |         \\                                   |"
          - type: action
            do: send_user_text
            text: "|   [TIDEMARK]      \\         [STILLWATER]            |"
          - type: action
            do: send_user_text
            text: "|        |           \\              |                 |"
          - type: action
            do: send_user_text
            text: "|   [GREENFORD]   [AMBER VALLEY]   |                 |"
          - type: action
            do: send_user_text
            text: "|         \\              \\          |                 |"
          - type: action
            do: send_user_text
            text: "|          \\---------[CONFLUENCE]                    |"
          - type: action
            do: send_user_text
            text: "|                         |                           |"
          - type: action
            do: send_user_text
            text: "|                   . . . . . . .                     |"
          - type: action
            do: send_user_text
            text: "|                  .   unmapped   .                   |"
          - type: action
            do: send_user_text
            text: "|                   . . . . . . .                     |"
          - type: action
            do: send_user_text
            text: "|                    /~~~~~~~~~~~\\                    |"
          - type: action
            do: send_user_text
            text: "|                   |  SANCTUM    |                   |"
          - type: action
            do: send_user_text
            text: "|                   |   BASIN     |  <- you are here  |"
          - type: action
            do: send_user_text
            text: "|                    \\~~~~~~~~~~~/                    |"
          - type: action
            do: send_user_text
            text: "+------------------------------------------------------+"
          - type: action
            do: send_user_text
            text: "| * city  ~ plateau rim  . unmapped  \\ road           |"
          - type: action
            do: send_user_text
            text: "+------------------------------------------------------+"
          - type: action
            do: send_user_text
            text: ""
          - type: action
            do: intercept

      # ── COMMAND: talk/greet — already shopping ──
      - type: sequence
        event: room_command
        children:
          - type: condition
            check: command_matches
            commands: [talk, greet]
          - type: condition
            check: mob_in_room
            mob_id: 50
          - type: condition
            check: player_has_quest
            quest: "1-shopping_arrive"
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              Your Awakening is behind you. The trials are ahead.
              Type <ansi fg="command">quest</ansi> to see what
              remains.
          - type: action
            do: intercept

      # ── COMMAND: talk/greet — mutation done ──
      - type: sequence
        event: room_command
        children:
          - type: condition
            check: command_matches
            commands: [talk, greet]
          - type: condition
            check: mob_in_room
            mob_id: 50
          - type: condition
            check: player_has_quest
            quest: "1-mutation"
          - type: action
            do: mob_say
            mob_id: 50
            text: >-
              Visit the Merchant on Market Street -- east to Town
              Square, then south. She will show you how commerce
              works here.
          - type: action
            do: intercept

      # ── COMMAND: talk/greet — not started ──
      - type: sequence
        event: room_command
        children:
          - type: condition
            check: command_matches
            commands: [talk, greet]
          - type: condition
            check: mob_in_room
            mob_id: 50
          - type: action
            do: mob_say
            mob_id: 50
            text: Step closer. The Awakening Rite will not wait.
          - type: action
            do: intercept

      # ── IDLE: ceremony unlock at tick 6 ──
      - type: sequence
        event: room_idle
        children:
          - type: condition
            check: state_equals
            key: ceremony_ticks
            value: 6
          - type: action
            do: set_room_locked
            direction: north
            locked: "false"
          - type: action
            do: set_room_locked
            direction: south
            locked: "false"
          - type: action
            do: set_room_locked
            direction: east
            locked: "false"
          - type: action
            do: set_room_locked
            direction: west
            locked: "false"
          - type: action
            do: set_state
            key: ceremony_ticks
            value: 0

      # ── IDLE: increment ceremony counter ──
      - type: sequence
        event: room_idle
        children:
          - type: condition
            check: state_greater_than
            key: ceremony_ticks
            value: 0
          - type: action
            do: increment_state
            key: ceremony_ticks
  ```

- [ ] **Step 18: Delete room 113 JS**

  ```bash
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/rooms/sanctum_basin/113.js
  ```

### Step Group F: Build and Commit

- [ ] **Step 19: Create directory structure**

  ```bash
  mkdir -p /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/behaviors/rooms/sanctum_basin
  ```

- [ ] **Step 20: Verify build**

  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 21: Commit**

  ```bash
  git add _datafiles/world/dogmud/behaviors/rooms/sanctum_basin/ internal/behaviortree/types.go internal/behaviortree/loader.go
  git rm _datafiles/world/dogmud/rooms/sanctum_basin/102.js _datafiles/world/dogmud/rooms/sanctum_basin/106.js _datafiles/world/dogmud/rooms/sanctum_basin/108.js _datafiles/world/dogmud/rooms/sanctum_basin/109.js _datafiles/world/dogmud/rooms/sanctum_basin/111.js _datafiles/world/dogmud/rooms/sanctum_basin/113.js _datafiles/world/dogmud/rooms/sanctum_basin/114.js _datafiles/world/dogmud/rooms/sanctum_basin/116.js _datafiles/world/dogmud/rooms/sanctum_basin/120.js
  git commit -m "$(cat <<'EOF'
  feat: migrate 9 Sanctum Basin tutorial rooms to behavior trees

  Replace all JS room scripts (102, 106, 108, 109, 111, 113, 114,
  116, 120) with room behavior tree YAML files. Add negate support
  to ConditionNode for checking mob absence (mob_in_room + negate).

  Full ceremony (room 113) includes all 18 priest lines with static
  delays, exit locking, mutation grant, and timed unlock.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

## Task 7: Migrate Non-Tutorial Rooms (5 rooms)

**Goal:** Migrate rooms 407, 4023, 1, 75, and -1. Rooms 1 and 75 convert to room nouns (no behavior tree needed). Room -1 is just deleted (death flow change in Task 8 eliminates it).

### Step Group A: Room 407 — Abandoned Campsite

- [ ] **Step 1: Create room 407 behavior tree YAML**

  Create `_datafiles/world/dogmud/behaviors/rooms/dustwalk_road/407.yaml`:

  ```yaml
  # Abandoned Campsite - Room 407
  # Grants quest step 4-investigate when player on quest 4 enters.

  tree:
    type: selector
    children:

      - type: sequence
        event: room_enter
        children:
          - type: condition
            check: player_has_quest
            quest: "4-start"
          - type: condition
            check: player_missing_quest
            quest: "4-investigate"
          - type: action
            do: grant_quest
            quest: "4-investigate"
          - type: action
            do: send_user_text
            text: >-
              The scattered debris and fresh boot prints confirm
              it -- this is where the bandits have been operating.
              There may be more evidence deeper in the gully.
  ```

- [ ] **Step 2: Delete room 407 JS**

  ```bash
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/rooms/dustwalk_road/407.js
  ```

### Step Group B: Room 4023 — Maren's Cottage

- [ ] **Step 3: Create room 4023 behavior tree YAML**

  Create `_datafiles/world/dogmud/behaviors/rooms/ashwick/4023.yaml`:

  ```yaml
  # Maren's Cottage - Room 4023
  # Push/move/pull stone reveals hidden letter (item 40041).

  tree:
    type: selector
    children:

      # ── Push stone, already has letter ──
      - type: sequence
        event: room_command
        children:
          - type: condition
            check: command_matches
            commands: [push, move, pull, shift, press]
          - type: condition
            check: command_rest_contains
            text: stone
          - type: condition
            check: player_has_item
            item_id: 40041
          - type: action
            do: send_user_text
            text: >-
              The loose stone shifts aside, revealing the empty
              cavity behind it. Whatever was hidden here has
              already been found.
          - type: action
            do: intercept

      # ── Push stone, discover the letter ──
      - type: sequence
        event: room_command
        children:
          - type: condition
            check: command_matches
            commands: [push, move, pull, shift, press]
          - type: condition
            check: command_rest_contains
            text: stone
          - type: action
            do: give_item
            item_id: 40041
          - type: action
            do: send_user_text
            text: >-
              You push the loose stone aside. It grinds against its
              neighbors and shifts inward, revealing a small cavity
              in the wall behind it. Inside, a folded letter rests
              against the smooth stone -- hidden deliberately,
              waiting for someone to look.
          - type: action
            do: send_room_text
            text: >-
              Someone pushes aside a loose stone near the hearth,
              revealing a hidden cavity.
            exclude_user: "true"
          - type: action
            do: intercept
  ```

- [ ] **Step 4: Delete room 4023 JS**

  ```bash
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/rooms/ashwick/4023.js
  ```

### Step Group C: Room 1 — Startland Town Square (noun)

- [ ] **Step 5: Add map noun to room 1 YAML**

  Edit `_datafiles/world/dogmud/rooms/startland/1.yaml` — add `nouns` field at the end:

  **Find:**
  ```yaml
  - A <ansi fg="mobname">guard</ansi> brushes off some of the snow that has accumulated
    on the <ansi fg="itemname">sign</ansi>.
  ```

  **Replace with:**
  ```yaml
  - A <ansi fg="mobname">guard</ansi> brushes off some of the snow that has accumulated
    on the <ansi fg="itemname">sign</ansi>.
  nouns:
    sign: >-
      A plain wooden sign nailed to a post near the square. A hand-drawn
      map is tacked to the front of it. Type <ansi fg="command">look
      map</ansi> to examine the map more closely.
    map: |
      You look at the map nailed to the sign.

      +--------------------------------------------------+
      |             MAP OF STARTLAND                      |
      +--------------------------------------------------+
      |                                                    |
      |        [Graveyard]                                |
      |             |                                      |
      |  [Garden]--[North Road]--[Stable]                 |
      |             |                                      |
      |  [Bakery]--[Town Square]--[Tailor]                |
      |             |                                      |
      |        [South Road]                               |
      |             |                                      |
      |        [The Fray]                                 |
      |                                                    |
      +--------------------------------------------------+
  ```

- [ ] **Step 6: Delete room 1 JS**

  ```bash
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/rooms/startland/1.js
  ```

### Step Group D: Shadow Realm Cleanup

- [ ] **Step 7: Delete room 75 JS**

  ```bash
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/rooms/shadow_realm/75.js
  ```

- [ ] **Step 8: Delete room -1 JS**

  ```bash
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/rooms/shadow_realm/-1.js
  ```

### Step Group E: Build and Commit

- [ ] **Step 9: Create directory structures**

  ```bash
  mkdir -p /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/behaviors/rooms/dustwalk_road
  mkdir -p /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/behaviors/rooms/ashwick
  ```

- [ ] **Step 10: Verify build**

  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 11: Commit**

  ```bash
  git add _datafiles/world/dogmud/behaviors/rooms/dustwalk_road/407.yaml _datafiles/world/dogmud/behaviors/rooms/ashwick/4023.yaml _datafiles/world/dogmud/rooms/startland/1.yaml
  git rm _datafiles/world/dogmud/rooms/dustwalk_road/407.js _datafiles/world/dogmud/rooms/ashwick/4023.js _datafiles/world/dogmud/rooms/startland/1.js _datafiles/world/dogmud/rooms/shadow_realm/75.js _datafiles/world/dogmud/rooms/shadow_realm/-1.js
  git commit -m "$(cat <<'EOF'
  feat: migrate non-tutorial room scripts (407, 4023, 1, 75, -1)

  Room 407: behavior tree for quest 4 investigate step.
  Room 4023: behavior tree for push-stone letter discovery.
  Room 1: convert dynamic map to static room noun.
  Rooms 75, -1: delete shadow realm JS (eliminated by death flow).

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

## Task 8: Simplify Death Flow

**Goal:** Modify `suicide.go` to restore pools to full instead of 5%, move players to their home room instead of DeathRecoveryRoom, and remove the shadow realm detour.

**Files:** `internal/usercommands/suicide.go`, `_datafiles/world/dogmud/buffs/24-death_recovery.js`

- [ ] **Step 1: Modify death pool restoration and room destination**

  In `internal/usercommands/suicide.go`:

  **Find:**
  ```go
  	// Set all pools to 5% of max so the player can regen up in the shadow realm
  	// instead of arriving deep in the negatives and getting stuck.
  	user.Character.Health = user.Character.HealthMax.Value / 20
  	if user.Character.Health < 1 {
  		user.Character.Health = 1
  	}
  	user.Character.Stamina = user.Character.StaminaMax.Value / 20
  	if user.Character.Stamina < 1 {
  		user.Character.Stamina = 1
  	}
  	user.Character.Conviction = user.Character.ConvictionMax.Value / 20
  	if user.Character.Conviction < 1 {
  		user.Character.Conviction = 1
  	}
  	events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})

  	clear(user.Character.PlayerDamage)

  	// Check if player died in an instanced zone with ejected death policy
  	if rooms.IsEphemeralRoomId(user.Character.RoomId) {
  		if inst := rooms.GetInstanceRegistry().FindByRoomId(user.Character.RoomId); inst != nil {
  			if inst.DeathPolicy == "ejected" {
  				inst.RevokeAccess(user.UserId)
  				user.SendText(`<ansi fg="red">You have been expelled from the instance. There is no return.</ansi>`)
  			}
  		}
  	}

  	rooms.MoveToRoom(user.UserId, int(configs.GetSpecialRoomsConfig().DeathRecoveryRoom))
  ```

  **Replace with:**
  ```go
  	// Restore all pools to full — the old shadow realm regen loop is removed.
  	user.Character.Health = user.Character.HealthMax.Value
  	user.Character.Stamina = user.Character.StaminaMax.Value
  	user.Character.Conviction = user.Character.ConvictionMax.Value
  	events.AddToQueue(events.CharacterVitalsChanged{UserId: user.UserId})

  	clear(user.Character.PlayerDamage)

  	// Check if player died in an instanced zone with ejected death policy
  	if rooms.IsEphemeralRoomId(user.Character.RoomId) {
  		if inst := rooms.GetInstanceRegistry().FindByRoomId(user.Character.RoomId); inst != nil {
  			if inst.DeathPolicy == "ejected" {
  				inst.RevokeAccess(user.UserId)
  				user.SendText(`<ansi fg="red">You have been expelled from the instance. There is no return.</ansi>`)
  			}
  		}
  	}

  	// Resolve home room: check player setting, default to startland (room 0)
  	homeRoom := 0
  	if home, ok := user.Character.Settings["home"]; ok && home == "thornwall" {
  		homeRoom = 468
  	}
  	rooms.MoveToRoom(user.UserId, homeRoom)

  	user.SendText(`<ansi fg="cyan">You awaken in familiar surroundings, shaken but whole.</ansi>`)
  ```

- [ ] **Step 2: Delete death_recovery buff JS**

  ```bash
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/buffs/24-death_recovery.js
  ```

- [ ] **Step 3: Verify build**

  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 4: Commit**

  ```bash
  git add internal/usercommands/suicide.go
  git rm _datafiles/world/dogmud/buffs/24-death_recovery.js
  git commit -m "$(cat <<'EOF'
  feat: simplify death flow — full restore + home room

  Replace shadow realm detour with direct respawn: pools restored to
  full, player moved to home room (thornwall 468 or startland 0).
  Delete death_recovery buff JS (no longer needed).

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

## Task 9: Spell Go Hooks + Stub Deletion

**Goal:** Create Go hooks for fold-anchor, fold-recall, and purge-affliction spells. Wire them in spell resolution before JS dispatch. Delete JS files.

**Files:**
- `internal/hooks/spell_foldanchor.go` (NEW)
- `internal/hooks/spell_foldrecall.go` (NEW)
- `internal/hooks/spell_purgeaffliction.go` (NEW)
- `internal/hooks/spell_resolution.go` — add dispatch
- `_datafiles/world/dogmud/spells/fold-anchor.js` — delete
- `_datafiles/world/dogmud/spells/fold-recall.js` — delete

- [ ] **Step 1: Create spell_foldanchor.go**

  Create `internal/hooks/spell_foldanchor.go`:

  ```go
  package hooks

  import (
  	"fmt"

  	"github.com/GoMudEngine/GoMud/internal/rooms"
  	"github.com/GoMudEngine/GoMud/internal/users"
  )

  // resolveFoldAnchor handles the fold-anchor spell's onMagic phase.
  func resolveFoldAnchor(user *users.UserRecord) {
  	roomId := user.Character.RoomId
  	user.Character.SetMiscData("fold-anchor-room", roomId)

  	user.SendText(`A Chrysalis anchor locks into place here. ` +
  		`Cast <ansi fg="command">fold-recall</ansi> from elsewhere to return.`)

  	if room := rooms.LoadRoom(roomId); room != nil {
  		room.SendText(fmt.Sprintf(
  			`A faint shimmer marks where <ansi fg="username">%s</ansi> has set an anchor.`,
  			user.Character.GetCharacterName(true)), user.UserId)
  	}
  }
  ```

- [ ] **Step 2: Create spell_foldrecall.go**

  Create `internal/hooks/spell_foldrecall.go`:

  ```go
  package hooks

  import (
  	"fmt"

  	"github.com/GoMudEngine/GoMud/internal/rooms"
  	"github.com/GoMudEngine/GoMud/internal/users"
  )

  // validateFoldRecall checks whether fold-recall can be cast.
  // Returns false to abort casting.
  func validateFoldRecall(user *users.UserRecord) bool {
  	currentRoomId := user.Character.RoomId

  	if room := rooms.LoadRoom(currentRoomId); room != nil {
  		if v, ok := room.GetTempData("allow_recall"); ok {
  			if allowed, ok := v.(bool); ok && !allowed {
  				user.SendText(`Something about this place prevents you from recalling.`)
  				return false
  			}
  		}
  	}

  	anchorRoom := getAnchorRoom(user)
  	if anchorRoom <= 0 {
  		user.SendText(`You reach for the Veil, but there is no anchor to pull you. ` +
  			`Set one first with <ansi fg="command">cast fold-anchor</ansi>.`)
  		return false
  	}

  	if anchorRoom == currentRoomId {
  		user.SendText(`You are already standing on your anchor.`)
  		return false
  	}

  	return true
  }

  // resolveFoldRecall handles the fold-recall spell's onMagic phase.
  func resolveFoldRecall(user *users.UserRecord) {
  	anchorRoom := getAnchorRoom(user)
  	currentRoomId := user.Character.RoomId

  	if anchorRoom <= 0 || anchorRoom == currentRoomId {
  		user.SendText(`The fold collapses — no valid anchor found.`)
  		return
  	}

  	user.Character.EndAggro()
  	name := user.Character.GetCharacterName(true)

  	if room := rooms.LoadRoom(currentRoomId); room != nil {
  		room.SendText(fmt.Sprintf(
  			`<ansi fg="username">%s</ansi> folds through the Veil and vanishes!`,
  			name), user.UserId)
  	}

  	rooms.MoveToRoom(user.UserId, anchorRoom)
  	user.SendText(`You fold through the Veil and arrive at your anchor point!`)

  	if room := rooms.LoadRoom(anchorRoom); room != nil {
  		room.SendText(fmt.Sprintf(
  			`<ansi fg="username">%s</ansi> folds through the Veil and appears!`,
  			name), user.UserId)
  	}
  }

  // getAnchorRoom retrieves the stored fold-anchor room from misc data.
  func getAnchorRoom(user *users.UserRecord) int {
  	val := user.Character.GetMiscData("fold-anchor-room")
  	if val == nil {
  		return 0
  	}
  	switch v := val.(type) {
  	case int:
  		return v
  	case float64:
  		return int(v)
  	case string:
  		var n int
  		fmt.Sscanf(v, "%d", &n)
  		return n
  	}
  	return 0
  }
  ```

- [ ] **Step 3: Create spell_purgeaffliction.go**

  Create `internal/hooks/spell_purgeaffliction.go`:

  ```go
  package hooks

  import (
  	"fmt"

  	"github.com/GoMudEngine/GoMud/internal/buffs"
  	"github.com/GoMudEngine/GoMud/internal/mobs"
  	"github.com/GoMudEngine/GoMud/internal/rooms"
  	"github.com/GoMudEngine/GoMud/internal/users"
  )

  // resolvePurgeAffliction cancels all poison-flagged buffs on targets.
  func resolvePurgeAffliction(user *users.UserRecord, targetUserIds []int, targetMobIds []int) {
  	casterName := user.Character.GetCharacterName(true)

  	for _, targetUserId := range targetUserIds {
  		target := users.GetByUserId(targetUserId)
  		if target == nil {
  			continue
  		}
  		target.Character.CancelBuffsWithFlag(buffs.Poison)

  		if targetUserId == user.UserId {
  			user.SendText(`<ansi fg="cyan">Chrysalis energy pulses through you, burning away toxins.</ansi>`)
  			if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
  				room.SendText(fmt.Sprintf(
  					`<ansi fg="cyan"><ansi fg="username">%s</ansi> glows briefly as toxins are purged.</ansi>`,
  					casterName), user.UserId)
  			}
  		} else {
  			user.SendText(fmt.Sprintf(
  				`<ansi fg="cyan">You purge the afflictions from <ansi fg="username">%s</ansi>.</ansi>`,
  				target.Character.GetCharacterName(true)))
  			target.SendText(fmt.Sprintf(
  				`<ansi fg="cyan"><ansi fg="username">%s</ansi> purges the toxins from your body.</ansi>`,
  				casterName))
  		}
  	}

  	for _, mobInstId := range targetMobIds {
  		mob := mobs.GetInstance(mobInstId)
  		if mob == nil {
  			continue
  		}
  		mob.Character.CancelBuffsWithFlag(buffs.Poison)
  		user.SendText(fmt.Sprintf(
  			`<ansi fg="cyan">You purge the afflictions from %s.</ansi>`,
  			mob.Character.GetCharacterName(true)))
  	}
  }
  ```

- [ ] **Step 4: Wire Go hooks in spell_resolution.go**

  In `internal/hooks/spell_resolution.go`:

  **Find:**
  ```go
  	spellAggro := characters.SpellAggroInfo{
  		SpellId:              cs.SpellId,
  		SpellRest:            cs.SpellRest,
  		TargetUserIds:        cs.TargetUserIds,
  		TargetMobInstanceIds: cs.TargetMobInstanceIds,
  	}
  	scripting.TrySpellScriptEvent("onMagic", user.UserId, 0, spellAggro)
  ```

  **Replace with:**
  ```go
  	// Go-native spell hooks — dispatch before JS scripts.
  	goHookHandled := false
  	if spellData != nil {
  		switch spellData.SpellId {
  		case "fold-anchor":
  			resolveFoldAnchor(user)
  			goHookHandled = true
  		case "fold-recall":
  			resolveFoldRecall(user)
  			goHookHandled = true
  		case "purge-affliction":
  			resolvePurgeAffliction(user, cs.TargetUserIds, cs.TargetMobInstanceIds)
  			goHookHandled = true
  		}
  	}

  	if !goHookHandled {
  		spellAggro := characters.SpellAggroInfo{
  			SpellId:              cs.SpellId,
  			SpellRest:            cs.SpellRest,
  			TargetUserIds:        cs.TargetUserIds,
  			TargetMobInstanceIds: cs.TargetMobInstanceIds,
  		}
  		scripting.TrySpellScriptEvent("onMagic", user.UserId, 0, spellAggro)
  	}
  ```

- [ ] **Step 5: Wire fold-recall validation in cast initiation**

  Search for `TrySpellScriptEvent("onCast"` to find the cast validation point. Add before the JS call:

  ```go
  if spellData.SpellId == "fold-recall" {
  	if !validateFoldRecall(user) {
  		return // abort casting
  	}
  }
  ```

- [ ] **Step 6: Delete all spell JS files (hooks + stubs + vestigial)**

  Delete the 3 hook scripts (replaced by Go), 2 stubs (no logic), and
  1 vestigial spell (chrysalis-aid — death system handles respawn):

  ```bash
  rm _datafiles/world/dogmud/spells/fold-anchor.js
  rm _datafiles/world/dogmud/spells/fold-recall.js
  rm _datafiles/world/dogmud/spells/purge-affliction.js
  rm _datafiles/world/dogmud/spells/heal.js
  rm _datafiles/world/dogmud/spells/identify.js
  rm _datafiles/world/dogmud/spells/chrysalis-aid.js
  ```

- [ ] **Step 7: Delete chrysalis-aid spell YAML + add player migration**

  Delete the spell definition:

  ```bash
  rm _datafiles/world/dogmud/spells/chrysalis-aid.yaml
  ```

  Add a player migration in `internal/usercommands/suicide.go` (or
  wherever existing migrations run — grep for
  `migration-alchemy-potions-done` to find the migration runner).
  The migration removes chrysalis-aid from player spellbooks on
  first login:

  Find the migration runner file. Add a new migration block:

  ```go
  if _, ok := user.Character.GetMiscData("migration-chrysalis-aid-removed").(string); !ok {
      delete(user.Character.Spellbook, "chrysalis-aid")
      user.Character.SetMiscData("migration-chrysalis-aid-removed", "1")
  }
  ```

  Also grep for `chrysalis-aid` in mob YAML files and NPC spell
  lists — remove any references.

- [ ] **Step 8: Verify build**

  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 9: Commit**

  ```bash
  git add internal/hooks/spell_foldanchor.go internal/hooks/spell_foldrecall.go \
    internal/hooks/spell_purgeaffliction.go internal/hooks/spell_resolution.go
  git rm _datafiles/world/dogmud/spells/*.js
  git rm _datafiles/world/dogmud/spells/chrysalis-aid.yaml
  git commit -m "$(cat <<'EOF'
  feat: migrate spells to Go hooks, delete chrysalis-aid

  fold-anchor, fold-recall, purge-affliction moved to Go hooks.
  chrysalis-aid deleted (vestigial — death system handles respawn).
  heal.js, identify.js stubs deleted. Player migration prunes
  chrysalis-aid from spellbooks.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

## Task 10: Buff Text Fields + Stub Deletion

**Goal:** Migrate remaining buff JS files to YAML text fields and delete all buff JS files. The buff text infrastructure already exists in `BuffSpec` and is wired in `Buff_ApplyBuffs.go` and `NewTurn_PruneBuffs.go`.

- [ ] **Step 1: Add text fields to meditating buff YAML**

  Edit `_datafiles/world/dogmud/buffs/0-meditating.yaml`:

  **Find:**
  ```yaml
  flags:
  - cancel-on-action
  - cancel-on-combat
  ```

  **Replace with:**
  ```yaml
  flags:
  - cancel-on-action
  - cancel-on-combat
  start_user_text: >-
    You sit down and begin your meditation.
    Your meditation must complete without interruption to quit
    gracefully.
  start_room_text: >-
    {source} sits down and begins to meditate.
  trigger_user_text: >-
    You continue your meditation.
  trigger_room_text: >-
    {source} continues meditating.
  ```

- [ ] **Step 2: Add text fields to illumination buff YAML**

  Edit `_datafiles/world/dogmud/buffs/1-illumination.yaml`:

  **Find:**
  ```yaml
  flags:
    - lightsource
  ```

  **Replace with:**
  ```yaml
  flags:
    - lightsource
  start_user_text: A warm glow surrounds you.
  start_room_text: A warm glow surrounds {source}.
  end_user_text: Your glowing fades away.
  end_room_text: The glow surrounding {source} fades away.
  ```

- [ ] **Step 3: Add text fields to stunned buff YAML**

  Edit `_datafiles/world/dogmud/buffs/2-stunned.yaml`:

  **Find:**
  ```yaml
  flags:
    - no-combat
  ```

  **Replace with:**
  ```yaml
  flags:
    - no-combat
  start_user_text: >-
    <ansi fg="yellow">You are stunned! You cannot attack.</ansi>
  start_room_text: "{source} staggers, dazed!"
  end_user_text: >-
    <ansi fg="green">You shake off the stun.</ansi>
  ```

- [ ] **Step 4: Add text fields to blinded buff YAML**

  Edit `_datafiles/world/dogmud/buffs/3-blinded.yaml`:

  **Find:**
  ```yaml
  statmods:
    perception: -40
  ```

  **Replace with:**
  ```yaml
  statmods:
    perception: -40
  start_user_text: >-
    <ansi fg="yellow">Your vision goes dark — you have been
    blinded!</ansi>
  start_room_text: "{source} claws at their eyes, blinded!"
  end_user_text: >-
    <ansi fg="green">Your vision slowly returns to normal.</ansi>
  ```

- [ ] **Step 5: Add text fields to hidden buff YAML**

  Edit `_datafiles/world/dogmud/buffs/9-hidden.yaml`:

  **Find:**
  ```yaml
  flags:
    - hidden
    - cancel-on-combat
  ```

  **Replace with:**
  ```yaml
  flags:
    - hidden
    - cancel-on-combat
  start_user_text: You feel sneaky.
  start_room_text: "{source} disappears into the shadows."
  end_user_text: You no longer feel sneaky.
  end_room_text: "{source} emerges from the shadows."
  ```

- [ ] **Step 6: Delete all dogmud buff JS files**

  ```bash
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/buffs/0-meditating.js
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/buffs/1-illumination.js
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/buffs/2-stunned.js
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/buffs/3-blinded.js
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/buffs/9-hidden.js
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/buffs/48-clarity_tonic.js
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/buffs/49-fire_resistance.js
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/buffs/51-berserker_elixir.js
  ```

- [ ] **Step 7: Verify build**

  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 8: Commit**

  ```bash
  git add _datafiles/world/dogmud/buffs/0-meditating.yaml _datafiles/world/dogmud/buffs/1-illumination.yaml _datafiles/world/dogmud/buffs/2-stunned.yaml _datafiles/world/dogmud/buffs/3-blinded.yaml _datafiles/world/dogmud/buffs/9-hidden.yaml
  git rm _datafiles/world/dogmud/buffs/0-meditating.js _datafiles/world/dogmud/buffs/1-illumination.js _datafiles/world/dogmud/buffs/2-stunned.js _datafiles/world/dogmud/buffs/3-blinded.js _datafiles/world/dogmud/buffs/9-hidden.js _datafiles/world/dogmud/buffs/48-clarity_tonic.js _datafiles/world/dogmud/buffs/49-fire_resistance.js _datafiles/world/dogmud/buffs/51-berserker_elixir.js
  git commit -m "$(cat <<'EOF'
  feat: migrate buff JS to YAML text fields + delete stubs

  Move start/end/trigger text for meditating, illumination, stunned,
  blinded, and hidden buffs to YAML text fields. Delete all remaining
  dogmud buff JS files.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---

## Task 11: Sable Portal Vendor + Documentation

**Goal:** Migrate Sable (mob 315) portal vendor from JS to behavior tree. Update documentation.

**Files:**
- `internal/behaviortree/actions.go` — add `create_instance_portal` action
- `_datafiles/world/dogmud/behaviors/thornwall_city/315-sable.yaml` (NEW)
- `_datafiles/world/dogmud/mobs/thornwall_city/scripts/315-sable.js` — delete
- `internal/behaviortree/context.md` — add room behavior tree docs

- [ ] **Step 1: Add create_instance_portal action**

  Register in `internal/behaviortree/actions.go` init():

  ```go
  	actionRegistry["create_instance_portal"] = actCreateInstancePortal
  ```

  Implementation (add at bottom of file):

  ```go
  // actCreateInstancePortal parses ask text for "<zone> <gold>",
  // validates gold, creates an instanced zone, and adds a temporary
  // exit. Specialized for portal vendor NPCs.
  // params: zone_map (map), min_gold (int), exit_duration (string)
  func actCreateInstancePortal(params map[string]any, ctx *EvalContext) Result {
  	user := users.GetByUserId(ctx.Event.UserId)
  	if user == nil {
  		return Failure
  	}

  	text := strings.ToLower(strings.TrimSpace(ctx.Event.Text))
  	parts := strings.Fields(text)
  	if len(parts) < 2 {
  		return Failure
  	}
  	zoneName := parts[0]
  	goldAmount := 0
  	fmt.Sscanf(parts[1], "%d", &goldAmount)
  	if goldAmount <= 0 {
  		return Failure
  	}

  	zoneMapRaw, ok := params["zone_map"].(map[string]any)
  	if !ok {
  		return Failure
  	}
  	templateZone := ""
  	for k, v := range zoneMapRaw {
  		if k == zoneName {
  			if s, ok := v.(string); ok {
  				templateZone = s
  			}
  		}
  	}
  	if templateZone == "" {
  		return Failure
  	}

  	// Find the mob for dialogue
  	var mob *mobs.Mob
  	room := rooms.LoadRoom(ctx.RoomId)
  	if room != nil && ctx.MobId > 0 {
  		for _, instId := range room.GetMobs(rooms.FindAll) {
  			m := mobs.GetInstance(instId)
  			if m != nil && int(m.MobId) == ctx.MobId {
  				mob = m
  				break
  			}
  		}
  	}

  	minGold := getIntParam(params, "min_gold")
  	if minGold == 0 {
  		minGold = 100
  	}
  	if goldAmount < minGold {
  		if mob != nil {
  			mob.Command(fmt.Sprintf(
  				"say That barely covers the runes. I need at least %d gold.",
  				minGold))
  		}
  		return Success
  	}
  	if user.Character.Gold < goldAmount {
  		if mob != nil {
  			mob.Command("say You do not have that much gold.")
  		}
  		return Success
  	}

  	user.Character.Gold -= goldAmount

  	entryRoomId := rooms.CreateInstance(
  		templateZone, goldAmount, ctx.Event.UserId, ctx.RoomId)
  	if entryRoomId <= 0 {
  		user.Character.Gold += goldAmount
  		if mob != nil {
  			mob.Command("say Something went wrong. The planes resist me. " +
  				"Your gold is returned.")
  		}
  		return Success
  	}

  	exitDuration := getStringParam(params, "exit_duration")
  	if exitDuration == "" {
  		exitDuration = "30 real minutes"
  	}
  	exitKey := fmt.Sprintf("%s-%d", zoneName, ctx.Event.UserId)

  	if room != nil {
  		added := room.AddTemporaryExit(
  			exitKey, exitKey+" rift", entryRoomId, exitDuration)
  		if !added {
  			user.Character.Gold += goldAmount
  			if mob != nil {
  				mob.Command("say The archway is already sustaining too " +
  					"many rifts. Your gold is returned.")
  			}
  			return Success
  		}
  		if mob != nil {
  			mob.Command(fmt.Sprintf(
  				"say The rift is open. Type %s to enter.", exitKey))
  			mob.Command("say It will hold for a time. Do not tarry.")
  		}
  		room.SendText("The stone archway flares with energy. " +
  			"A shimmering portal appears.")
  	}

  	return Success
  }
  ```

  **Import check:** Verify `rooms.CreateInstance` and `room.AddTemporaryExit` signatures match. Adjust if needed.

- [ ] **Step 2: Create Sable behavior tree YAML**

  Create `_datafiles/world/dogmud/behaviors/thornwall_city/315-sable.yaml`:

  ```yaml
  # Sable - Portal Vendor (mob 315)
  # Opens rifts to instanced zones for gold.

  tree:
    type: selector
    children:

      # ── ASK: portal/portals/zones/rift ──
      - type: sequence
        event: player_ask
        children:
          - type: condition
            check: keyword_match
            keywords:
              - portal
              - portals
              - zones
              - instance
              - instances
              - rift
              - rifts
          - type: action
            do: say
            text: >-
              I can open rifts to dangerous places, for a price.
          - type: action
            do: say
            text: >-
              Currently I can reach the Arena and the Planar Oasis.
          - type: action
            do: say
            text: >-
              Tell me the place and how much gold you wish to
              invest. The more gold, the more dangerous -- and
              rewarding.
          - type: action
            do: send_user_text
            text: >-
              <ansi fg="181">  [Try: ask sable arena 200 or ask
              sable oasis 300]</ansi>

      # ── ASK: arena (info) ──
      - type: sequence
        event: player_ask
        children:
          - type: condition
            check: keyword_match
            keywords: [arena]
          - type: action
            do: say
            text: >-
              The Arena is a brutal proving ground. Death means
              expulsion.
          - type: action
            do: say
            text: >-
              You cannot recall from within. Tell me how much
              gold to invest.

      # ── ASK: oasis (info) ──
      - type: sequence
        event: player_ask
        children:
          - type: condition
            check: keyword_match
            keywords: [oasis]
          - type: action
            do: say
            text: >-
              The Planar Oasis shimmers between worlds. Dangerous,
              but you may return if you fall.
          - type: action
            do: say
            text: >-
              Recall magic still works there. Tell me how much
              gold to invest.

      # ── ASK: greetings ──
      - type: sequence
        event: player_ask
        children:
          - type: condition
            check: keyword_match
            keywords: [hello, hi, help, who, what, quest]
          - type: action
            do: say
            text: >-
              I am Sable. I open rifts to places best left
              undisturbed.
          - type: action
            do: say
            text: >-
              For the right price in gold, I can send you and
              your party somewhere dangerous.
          - type: action
            do: say
            text: Ask me about portals if you want to know more.
          - type: action
            do: send_user_text
            text: >-
              <ansi fg="181">  [Try: ask sable portals]</ansi>

      # ── ASK: "<zone> <gold>" — create portal (catch-all) ──
      - type: sequence
        event: player_ask
        children:
          - type: action
            do: create_instance_portal
            zone_map:
              arena: "Instance Arena"
              oasis: "Instance Planar Oasis"
            min_gold: 100
            exit_duration: "30 real minutes"
  ```

- [ ] **Step 3: Delete Sable JS**

  ```bash
  rm /c/Users/Calabe\ Davis/workspace/DOGMud/_datafiles/world/dogmud/mobs/thornwall_city/scripts/315-sable.js
  ```

- [ ] **Step 4: Update context.md with room behavior tree docs**

  Append to `internal/behaviortree/context.md`:

  ```markdown
  ## Room Behavior Trees

  Room behavior trees provide event-driven scripting for rooms,
  replacing JS room scripts. They respond to player lifecycle events.

  ### File Convention

  ```
  _datafiles/world/dogmud/behaviors/rooms/{zone}/{roomId}.yaml
  ```

  ### Event Types

  | Event | Fired When | Context Fields |
  |-------|-----------|----------------|
  | `room_enter` | Player enters room | UserId, Direction |
  | `room_exit` | Player leaves room | UserId, Direction |
  | `room_command` | Player issues command | UserId, Command, Rest |
  | `room_idle` | Once per tick per room | (none) |

  ### Command Interception

  For `room_command` events, include an `intercept` action to consume
  the command and prevent normal dispatch. Used for NPC talk responses,
  blocking movement during ceremonies, and custom interactions.

  ### Room-Specific Conditions

  | Condition | Params | Description |
  |-----------|--------|-------------|
  | `command_matches` | `command`/`commands` | Command verb matches |
  | `command_rest_contains` | `text` | Rest contains substring |
  | `mob_in_room` | `mob_id`, `negate` | Mob template in room |

  ### Room-Specific Actions

  | Action | Params | Description |
  |--------|--------|-------------|
  | `mob_say` | `mob_id`, `text`, `delay` | Room mob speaks |
  | `mob_emote` | `mob_id`, `text`, `delay` | Room mob emotes |
  | `send_user_text` | `text` | Text to triggering player |
  | `send_room_text` | `text`, `exclude_user` | Text to room |
  | `intercept` | — | Block command dispatch |
  | `grant_mutation` | `mutation_id` (opt) | Roll + grant mutation |
  | `move_player` | `room_id` | Teleport player |
  | `remove_buff` | `buff_id` | Remove buff from player |

  ### Reference Example

  See `_datafiles/world/dogmud/behaviors/rooms/sanctum_basin/113.yaml`
  for the most complex room tree (Academy Hall ceremony).
  ```

- [ ] **Step 5: Verify build**

  ```bash
  cd /c/Users/Calabe\ Davis/workspace/DOGMud && go build ./...
  ```

- [ ] **Step 6: Commit**

  ```bash
  git add _datafiles/world/dogmud/behaviors/thornwall_city/315-sable.yaml internal/behaviortree/actions.go internal/behaviortree/context.md
  git rm _datafiles/world/dogmud/mobs/thornwall_city/scripts/315-sable.js
  git commit -m "$(cat <<'EOF'
  feat: migrate Sable portal vendor to behavior tree + update docs

  Add create_instance_portal action for portal vendor NPCs. Convert
  Sable (mob 315) from JS to behavior tree. Update context.md with
  room behavior tree documentation.

  Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
  EOF
  )"
  ```

---
