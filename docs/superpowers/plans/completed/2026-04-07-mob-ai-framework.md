# Mob AI Framework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an event-driven mob AI framework with YAML-configurable tactics, per-mob reaction speed, tactical discipline, and combat memory — enabling mobs to react to combat events within 0.5-2 seconds instead of waiting for the 4-second round tick.

**Architecture:** New `internal/mobai/` package with 5 files (reactor, tactics, memory, triggers, actions). The reactor listens on `NewTurn` events (100ms tick) to process queued reactions. Combat hooks emit new `MobAISignal` events when combat-relevant things happen. Mob YAML gains `reaction_delay`, `tactical_discipline`, `tactics`, and `tactic_preset` fields.

**Tech Stack:** Go, YAML mob definitions, existing event system.

---

### Task 1: Add Mob YAML Fields and Data Structures

**Files:**
- Modify: `internal/mobs/mobs.go` (add fields to Mob struct)
- Create: `internal/mobai/types.go` (shared types)

- [ ] **Step 1: Create types file**

Create `internal/mobai/types.go`:

```go
package mobai

// TacticRule is a single priority-ordered behavior rule defined in mob YAML.
type TacticRule struct {
	Trigger  string `yaml:"trigger"`  // e.g. "target_casting", "health_below:30"
	Action   string `yaml:"action"`   // e.g. "trip", "cast conviction-spike", "flee"
	Priority int    `yaml:"priority"` // Higher = evaluated first
}

// CombatMemory tracks who the mob was fighting across flee/re-engage cycles.
type CombatMemory struct {
	TargetUserId   int    // Player they were fighting
	TargetMobId    int    // Or mob they were fighting
	LastSeenRoomId int    // Where the target was last seen
	LastSeenRound  uint64 // When they last saw the target
	Grudge         bool   // Should they pursue?
}

// PendingReaction is a queued tactical reaction waiting to fire.
type PendingReaction struct {
	MobInstanceId int
	Action        string // Command to execute
	FireTurn      uint64 // Turn number when this should execute
}
```

- [ ] **Step 2: Add fields to Mob struct**

In `internal/mobs/mobs.go`, add to the Mob struct after the `Archetype` field (around line 86):

```go
// Mob AI framework fields
ReactionDelay      float64       `yaml:"reaction_delay,omitempty"`       // Seconds before executing a reactive tactic (default 1.5)
TacticalDiscipline float64      `yaml:"tactical_discipline,omitempty"`  // 0.0-1.0, how reliably mob follows tactics (default 0.5)
TacticPreset       string       `yaml:"tactic_preset,omitempty"`        // Named preset: "aggressive_melee", "defensive_caster", "ambusher"
Tactics            []mobai.TacticRule `yaml:"tactics,omitempty"`         // Per-mob tactic overrides
CombatMemory       *mobai.CombatMemory `yaml:"-"`                      // Runtime combat memory (not persisted)
lastReactionTurn   uint64                                               // Cooldown: last turn a reaction fired
```

Note: This will require `import "github.com/GoMudEngine/GoMud/internal/mobai"` in mobs.go.

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 4: Commit**

```bash
git add internal/mobai/types.go internal/mobs/mobs.go
git commit -m "feat: add mob AI types and YAML fields (reaction_delay, tactics, combat memory)"
```

---

### Task 2: Add Config Knobs

**Files:**
- Modify: `internal/configs/config.balance.go` (add fields + validation)
- Modify: `_datafiles/config.yaml` (add entries)

- [ ] **Step 1: Add fields to BalanceConfig**

In `internal/configs/config.balance.go`, add:

```go
CombatMemoryDuration   ConfigInt   `yaml:"CombatMemoryDuration"`   // Rounds before combat memory expires (default 300)
MobAIEnabled           ConfigBool  `yaml:"MobAIEnabled"`           // Global toggle for reactive AI system (default true)
MobReactionDelayMin    ConfigFloat `yaml:"MobReactionDelayMin"`    // Min reaction delay in seconds (default 0.25)
MobReactionDelayMax    ConfigFloat `yaml:"MobReactionDelayMax"`    // Max reaction delay in seconds (default 4.0)
```

- [ ] **Step 2: Add validation**

```go
if b.CombatMemoryDuration < 1 {
	b.CombatMemoryDuration = 300
}
if b.MobAIEnabled == 0 {
	b.MobAIEnabled = 1 // default true
}
if b.MobReactionDelayMin <= 0 {
	b.MobReactionDelayMin = 0.25
}
if b.MobReactionDelayMax <= 0 {
	b.MobReactionDelayMax = 4.0
}
```

- [ ] **Step 3: Add config entries**

In `_datafiles/config.yaml`:

```yaml
  CombatMemoryDuration: 300       # Rounds before mob forgets combat target
  MobAIEnabled: true              # Global toggle for reactive mob AI
  MobReactionDelayMin: 0.25       # Minimum reaction delay (seconds)
  MobReactionDelayMax: 4.0        # Maximum reaction delay (seconds)
```

- [ ] **Step 4: Verify build and commit**

```bash
go build ./...
git add internal/configs/config.balance.go _datafiles/config.yaml
git commit -m "feat: add mob AI config knobs (memory duration, reaction delay range)"
```

---

### Task 3: Implement Combat Memory

**Files:**
- Create: `internal/mobai/memory.go`
- Create: `internal/mobai/memory_test.go`

- [ ] **Step 1: Write tests**

Create `internal/mobai/memory_test.go`:

```go
package mobai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetMemory(t *testing.T) {
	mem := SetMemory(5, 0, 100, 1000)
	assert.Equal(t, 5, mem.TargetUserId)
	assert.Equal(t, 100, mem.LastSeenRoomId)
	assert.Equal(t, uint64(1000), mem.LastSeenRound)
	assert.True(t, mem.Grudge)
}

func TestMemoryExpired(t *testing.T) {
	mem := SetMemory(5, 0, 100, 1000)
	assert.False(t, MemoryExpired(mem, 1200, 300))
	assert.True(t, MemoryExpired(mem, 1301, 300))
}

func TestMemoryExpired_Nil(t *testing.T) {
	assert.True(t, MemoryExpired(nil, 1000, 300))
}

func TestUpdateMemoryLocation(t *testing.T) {
	mem := SetMemory(5, 0, 100, 1000)
	UpdateMemoryLocation(mem, 200, 1500)
	assert.Equal(t, 200, mem.LastSeenRoomId)
	assert.Equal(t, uint64(1500), mem.LastSeenRound)
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/mobai/... -run TestSetMemory -v`
Expected: FAIL — functions undefined.

- [ ] **Step 3: Implement memory functions**

Create `internal/mobai/memory.go`:

```go
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

// ClearMemory resets a mob's combat memory.
func ClearMemory(mem **CombatMemory) {
	*mem = nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/mobai/... -v`
Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/mobai/memory.go internal/mobai/memory_test.go
git commit -m "feat: implement combat memory for mob AI"
```

---

### Task 4: Implement Trigger Evaluation

**Files:**
- Create: `internal/mobai/triggers.go`
- Create: `internal/mobai/triggers_test.go`

- [ ] **Step 1: Write tests**

Create `internal/mobai/triggers_test.go`:

```go
package mobai

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/stretchr/testify/assert"
)

func TestEvalTrigger_CombatStart(t *testing.T) {
	ctx := &TriggerContext{CombatJustStarted: true}
	assert.True(t, EvalTrigger("combat_start", ctx))
	ctx.CombatJustStarted = false
	assert.False(t, EvalTrigger("combat_start", ctx))
}

func TestEvalTrigger_TargetCasting(t *testing.T) {
	target := &characters.Character{}
	ctx := &TriggerContext{Target: target}
	assert.False(t, EvalTrigger("target_casting", ctx))
	target.SetCast(3, characters.SpellAggroInfo{SpellId: "test"})
	assert.True(t, EvalTrigger("target_casting", ctx))
}

func TestEvalTrigger_HealthBelow(t *testing.T) {
	mob := &characters.Character{}
	mob.Health = 20
	mob.HealthMax.Value = 100
	ctx := &TriggerContext{MobChar: mob}
	assert.True(t, EvalTrigger("health_below:30", ctx))
	assert.False(t, EvalTrigger("health_below:10", ctx))
}

func TestEvalTrigger_MultipleTargets(t *testing.T) {
	ctx := &TriggerContext{EnemyCount: 3}
	assert.True(t, EvalTrigger("multiple_targets", ctx))
	ctx.EnemyCount = 1
	assert.False(t, EvalTrigger("multiple_targets", ctx))
}

func TestEvalTrigger_AfterAction(t *testing.T) {
	ctx := &TriggerContext{LastAction: "trip"}
	assert.True(t, EvalTrigger("after_action:trip", ctx))
	assert.False(t, EvalTrigger("after_action:bash", ctx))
}

func TestEvalTrigger_NoAggro(t *testing.T) {
	ctx := &TriggerContext{HasAggro: false, HasMemory: true}
	assert.True(t, EvalTrigger("no_aggro", ctx))
	ctx.HasAggro = true
	assert.False(t, EvalTrigger("no_aggro", ctx))
}

func TestEvalTrigger_Unknown(t *testing.T) {
	ctx := &TriggerContext{}
	assert.False(t, EvalTrigger("nonexistent_trigger", ctx))
}
```

- [ ] **Step 2: Implement triggers**

Create `internal/mobai/triggers.go`:

```go
package mobai

import (
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
)

// TriggerContext holds the current state used to evaluate triggers.
// Built fresh each time the AI reactor evaluates a mob.
type TriggerContext struct {
	MobChar            *characters.Character
	Target             *characters.Character
	EnemyCount         int
	HasAggro           bool
	HasMemory          bool
	CombatJustStarted  bool
	LastAction         string // Last successfully completed action
	PlayerEntered      bool   // A player just entered the room
	IsHidden           bool   // Mob is currently hidden
	ActiveBuffIds      []int  // Mob's active buff IDs
}

// EvalTrigger checks whether a trigger string matches the current context.
func EvalTrigger(trigger string, ctx *TriggerContext) bool {
	// Handle parameterized triggers (e.g. "health_below:30")
	parts := strings.SplitN(trigger, ":", 2)
	triggerName := parts[0]
	triggerParam := ""
	if len(parts) > 1 {
		triggerParam = parts[1]
	}

	switch triggerName {
	case "combat_start":
		return ctx.CombatJustStarted

	case "target_casting":
		return ctx.Target != nil && ctx.Target.GetCastRoundsRemaining() > 0

	case "target_prone":
		return ctx.Target != nil && ctx.Target.CombatPosition == characters.PositionProne

	case "target_grappled":
		return ctx.Target != nil && ctx.Target.CombatPosition == characters.PositionGrappled

	case "health_below":
		if ctx.MobChar == nil || triggerParam == "" {
			return false
		}
		pct, err := strconv.Atoi(triggerParam)
		if err != nil {
			return false
		}
		if ctx.MobChar.HealthMax.Value <= 0 {
			return false
		}
		currentPct := (ctx.MobChar.Health * 100) / ctx.MobChar.HealthMax.Value
		return currentPct < pct

	case "multiple_targets":
		return ctx.EnemyCount > 1

	case "single_target":
		return ctx.EnemyCount == 1

	case "target_fled":
		return !ctx.HasAggro && ctx.HasMemory

	case "not_hidden":
		return !ctx.IsHidden

	case "no_aggro":
		return !ctx.HasAggro && ctx.HasMemory

	case "after_action":
		return ctx.LastAction == triggerParam

	case "player_entered":
		return ctx.PlayerEntered

	case "has_buff":
		if triggerParam == "" {
			return false
		}
		buffId, err := strconv.Atoi(triggerParam)
		if err != nil {
			return false
		}
		for _, id := range ctx.ActiveBuffIds {
			if id == buffId {
				return true
			}
		}
		return false

	case "missing_buff":
		if triggerParam == "" {
			return false
		}
		buffId, err := strconv.Atoi(triggerParam)
		if err != nil {
			return false
		}
		for _, id := range ctx.ActiveBuffIds {
			if id == buffId {
				return false
			}
		}
		return true

	default:
		return false
	}
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/mobai/... -v`
Expected: All PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/mobai/triggers.go internal/mobai/triggers_test.go
git commit -m "feat: implement trigger evaluation for mob AI tactics"
```

---

### Task 5: Implement Tactic Evaluation and Presets

**Files:**
- Create: `internal/mobai/tactics.go`
- Create: `internal/mobai/tactics_test.go`

- [ ] **Step 1: Write tests**

Create `internal/mobai/tactics_test.go`:

```go
package mobai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateTactics_HighestPriorityWins(t *testing.T) {
	tactics := []TacticRule{
		{Trigger: "combat_start", Action: "cast shield", Priority: 5},
		{Trigger: "combat_start", Action: "trip", Priority: 10},
	}
	ctx := &TriggerContext{CombatJustStarted: true}
	action := EvaluateTactics(tactics, ctx)
	assert.Equal(t, "trip", action)
}

func TestEvaluateTactics_NoMatch(t *testing.T) {
	tactics := []TacticRule{
		{Trigger: "target_casting", Action: "trip", Priority: 10},
	}
	ctx := &TriggerContext{CombatJustStarted: true} // not casting
	action := EvaluateTactics(tactics, ctx)
	assert.Equal(t, "", action)
}

func TestGetPreset_Known(t *testing.T) {
	preset := GetPreset("aggressive_melee")
	assert.Greater(t, len(preset), 0)
}

func TestGetPreset_Unknown(t *testing.T) {
	preset := GetPreset("nonexistent")
	assert.Nil(t, preset)
}

func TestMergeTactics_OverrideAndExtend(t *testing.T) {
	preset := []TacticRule{
		{Trigger: "target_prone", Action: "kick", Priority: 10},
	}
	custom := []TacticRule{
		{Trigger: "health_below:20", Action: "flee", Priority: 15},
	}
	merged := MergeTactics(preset, custom)
	assert.Equal(t, 2, len(merged))
}
```

- [ ] **Step 2: Implement tactics**

Create `internal/mobai/tactics.go`:

```go
package mobai

import "sort"

// Presets maps named presets to their tactic rule lists.
var presets = map[string][]TacticRule{
	"aggressive_melee": {
		{Trigger: "target_prone", Action: "kick", Priority: 10},
		{Trigger: "target_casting", Action: "bash", Priority: 9},
		{Trigger: "target_grappled", Action: "submit", Priority: 8},
	},
	"defensive_caster": {
		{Trigger: "missing_buff:2", Action: "cast chrysalis-cocoon", Priority: 10},
		{Trigger: "multiple_targets", Action: "cast conviction-barrage", Priority: 9},
		{Trigger: "health_below:30", Action: "flee", Priority: 8},
		{Trigger: "single_target", Action: "cast conviction-spike", Priority: 5},
	},
	"ambusher": {
		{Trigger: "after_action:surprise-strike", Action: "flee", Priority: 10},
		{Trigger: "no_aggro", Action: "track_memory", Priority: 9},
		{Trigger: "not_hidden", Action: "hide", Priority: 8},
		{Trigger: "target_casting", Action: "trip", Priority: 7},
	},
	"tank": {
		{Trigger: "target_casting", Action: "bash", Priority: 10},
		{Trigger: "target_prone", Action: "kick", Priority: 9},
		{Trigger: "health_below:20", Action: "call_for_help", Priority: 8},
	},
}

// GetPreset returns the tactic rules for a named preset, or nil.
func GetPreset(name string) []TacticRule {
	p, ok := presets[name]
	if !ok {
		return nil
	}
	// Return a copy
	result := make([]TacticRule, len(p))
	copy(result, p)
	return result
}

// MergeTactics combines preset and custom tactics. Custom rules are
// appended (higher priority custom rules will naturally sort first).
func MergeTactics(preset []TacticRule, custom []TacticRule) []TacticRule {
	merged := make([]TacticRule, 0, len(preset)+len(custom))
	merged = append(merged, preset...)
	merged = append(merged, custom...)
	return merged
}

// EvaluateTactics checks all rules against the current context and
// returns the action string for the highest-priority matching rule.
// Returns "" if no rule matches.
func EvaluateTactics(tactics []TacticRule, ctx *TriggerContext) string {
	if len(tactics) == 0 {
		return ""
	}

	// Sort by priority descending
	sorted := make([]TacticRule, len(tactics))
	copy(sorted, tactics)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})

	for _, rule := range sorted {
		if EvalTrigger(rule.Trigger, ctx) {
			return rule.Action
		}
	}

	return ""
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./internal/mobai/... -v`
Expected: All PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/mobai/tactics.go internal/mobai/tactics_test.go
git commit -m "feat: implement tactic evaluation and presets for mob AI"
```

---

### Task 6: Implement Special Actions

**Files:**
- Create: `internal/mobai/actions.go`

- [ ] **Step 1: Implement special action handlers**

Create `internal/mobai/actions.go`:

```go
package mobai

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ExecuteAction handles both direct mob commands and special framework
// actions. Returns true if the action was handled.
func ExecuteAction(mob *mobs.Mob, action string, room *rooms.Room) bool {
	switch action {
	case "retarget_strongest":
		return doRetargetStrongest(mob, room)
	case "call_for_help":
		return doCallForHelp(mob, room)
	case "track_memory":
		return doTrackMemory(mob)
	default:
		// Check for recall:<roomid> pattern
		if len(action) > 7 && action[:7] == "recall:" {
			return doRecall(mob, action[7:])
		}
		// Default: treat as a mob command
		mob.Command(action, 0)
		return true
	}
}

// doRetargetStrongest switches aggro to the highest-power target in the room.
func doRetargetStrongest(mob *mobs.Mob, room *rooms.Room) bool {
	if room == nil {
		return false
	}

	bestPower := 0
	bestUserId := 0

	for _, uid := range room.GetPlayers() {
		u := users.GetByUserId(uid)
		if u == nil || u.Character.Health <= 0 {
			continue
		}
		// Simple power estimate: sum of stats
		power := u.Character.Stats.Strength.ValueAdj +
			u.Character.Stats.Dexterity.ValueAdj +
			u.Character.Stats.Willpower.ValueAdj
		if power > bestPower {
			bestPower = power
			bestUserId = uid
		}
	}

	if bestUserId > 0 && (mob.Character.Aggro == nil || mob.Character.Aggro.UserId != bestUserId) {
		mob.Command(fmt.Sprintf("attack !%d", bestUserId), 0)
		return true
	}
	return false
}

// doCallForHelp alerts allied mobs in adjacent rooms to come help.
func doCallForHelp(mob *mobs.Mob, room *rooms.Room) bool {
	if room == nil || mob.Character.Aggro == nil {
		return false
	}

	// Get adjacent rooms
	for _, exit := range room.Exits {
		adjRoom := rooms.LoadRoom(exit.RoomId)
		if adjRoom == nil {
			continue
		}
		// Find allied mobs in adjacent rooms
		for _, adjMobId := range adjRoom.GetMobs(rooms.FindAll) {
			adjMob := mobs.GetInstance(adjMobId)
			if adjMob == nil || adjMob.Character.Aggro != nil {
				continue // already fighting
			}
			if adjMob.IsNonCombatant() {
				continue
			}
			// Same zone = allied
			if adjMob.Zone == mob.Zone {
				// Path to the caller's room and engage their target
				adjMob.Command(fmt.Sprintf("pathto room %d", mob.Character.RoomId), 0)
				if mob.Character.Aggro.UserId > 0 {
					adjMob.Command(fmt.Sprintf("attack !%d", mob.Character.Aggro.UserId), 1.5)
				}
			}
		}
	}

	// Emit a room message
	room.SendText(fmt.Sprintf(
		`<ansi fg="mobname">%s</ansi> calls out for reinforcements!`,
		mob.Character.Name))

	return true
}

// doTrackMemory uses combat memory to path toward the remembered target.
func doTrackMemory(mob *mobs.Mob) bool {
	if mob.CombatMemory == nil || !mob.CombatMemory.Grudge {
		return false
	}

	targetRoomId := mob.CombatMemory.LastSeenRoomId
	if targetRoomId <= 0 || targetRoomId == mob.Character.RoomId {
		return false
	}

	mob.Command(fmt.Sprintf("pathto room %d", targetRoomId), 0)
	return true
}

// doRecall paths the mob to a specific room by ID.
func doRecall(mob *mobs.Mob, roomIdStr string) bool {
	mob.Command(fmt.Sprintf("pathto room %s", roomIdStr), 0)
	return true
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 3: Commit**

```bash
git add internal/mobai/actions.go
git commit -m "feat: implement special actions (retarget, call_for_help, track_memory, recall)"
```

---

### Task 7: Implement the Reactor (Core Event Loop)

**Files:**
- Create: `internal/mobai/reactor.go`
- Create: `internal/mobai/reactor_test.go`

This is the central piece — it listens for events, builds trigger context, evaluates tactics, and queues reactions with the appropriate delay.

- [ ] **Step 1: Create the MobAISignal event type**

In `internal/events/eventtypes.go`, add:

```go
// MobAISignal is emitted when something combat-relevant happens that
// mobs should potentially react to (damage taken, player entered, etc.).
type MobAISignal struct {
	MobInstanceId int
	SignalType    string // "damage_taken", "combat_start", "target_fled", "action_complete", "player_entered"
	Detail        string // e.g. action name for "action_complete"
	RoomId        int
}

func (m MobAISignal) Type() string { return "MobAISignal" }
```

- [ ] **Step 2: Write reactor tests**

Create `internal/mobai/reactor_test.go`:

```go
package mobai

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEffectiveReactionDelay(t *testing.T) {
	assert.Equal(t, 1.5, GetEffectiveReactionDelay(0, 0.25, 4.0))   // default
	assert.Equal(t, 0.5, GetEffectiveReactionDelay(0.5, 0.25, 4.0)) // custom
	assert.Equal(t, 0.25, GetEffectiveReactionDelay(0.1, 0.25, 4.0)) // clamped low
	assert.Equal(t, 4.0, GetEffectiveReactionDelay(5.0, 0.25, 4.0))  // clamped high
}

func TestShouldFollowTactic(t *testing.T) {
	// With discipline 1.0, should always follow
	follows := 0
	for i := 0; i < 100; i++ {
		if ShouldFollowTactic(1.0) {
			follows++
		}
	}
	assert.Equal(t, 100, follows)

	// With discipline 0.0, should never follow
	follows = 0
	for i := 0; i < 100; i++ {
		if ShouldFollowTactic(0.0) {
			follows++
		}
	}
	assert.Equal(t, 0, follows)
}
```

- [ ] **Step 3: Implement the reactor**

Create `internal/mobai/reactor.go`:

```go
package mobai

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// pendingReactions holds queued reactions waiting to fire.
var pendingReactions []PendingReaction

// GetEffectiveReactionDelay returns the mob's reaction delay, clamped
// to the configured min/max range. Returns 1.5 if not set.
func GetEffectiveReactionDelay(mobDelay float64, minDelay float64, maxDelay float64) float64 {
	if mobDelay <= 0 {
		mobDelay = 1.5
	}
	if mobDelay < minDelay {
		mobDelay = minDelay
	}
	if mobDelay > maxDelay {
		mobDelay = maxDelay
	}
	return mobDelay
}

// ShouldFollowTactic rolls against tactical discipline.
// Returns true if the mob should execute the tactic.
func ShouldFollowTactic(discipline float64) bool {
	if discipline <= 0 {
		return false
	}
	if discipline >= 1.0 {
		return true
	}
	return util.Rand(100) < int(discipline*100)
}

// HandleMobAISignal processes a combat-relevant signal for a mob.
// Evaluates the mob's tactics and queues a reaction if appropriate.
func HandleMobAISignal(e events.Event) events.ListenerReturn {
	if !bool(configs.GetBalanceConfig().MobAIEnabled) {
		return events.Continue
	}

	signal, ok := e.(events.MobAISignal)
	if !ok {
		return events.Continue
	}

	mob := mobs.GetInstance(signal.MobInstanceId)
	if mob == nil || mob.Character.Health <= 0 {
		return events.Continue
	}

	// Check reaction cooldown
	currentTurn := util.GetTurnCount()
	bal := configs.GetBalanceConfig()
	delay := GetEffectiveReactionDelay(
		mob.ReactionDelay,
		float64(bal.MobReactionDelayMin),
		float64(bal.MobReactionDelayMax),
	)
	cooldownTurns := uint64(delay * float64(configs.GetTimingConfig().TurnsPerSecond()))
	if currentTurn-mob.GetLastReactionTurn() < cooldownTurns {
		return events.Continue
	}

	// No tactics defined? Skip.
	tactics := ResolveTactics(mob)
	if len(tactics) == 0 {
		return events.Continue
	}

	// Build trigger context
	ctx := BuildTriggerContext(mob, signal)

	// Evaluate tactics
	action := EvaluateTactics(tactics, ctx)
	if action == "" {
		return events.Continue
	}

	// Roll discipline
	discipline := mob.TacticalDiscipline
	if discipline <= 0 {
		discipline = 0.5
	}
	if !ShouldFollowTactic(discipline) {
		return events.Continue
	}

	// Queue the reaction with delay
	fireTurn := currentTurn + uint64(delay*float64(configs.GetTimingConfig().TurnsPerSecond()))
	pendingReactions = append(pendingReactions, PendingReaction{
		MobInstanceId: mob.InstanceId,
		Action:        action,
		FireTurn:      fireTurn,
	})

	mob.SetLastReactionTurn(currentTurn)

	return events.Continue
}

// ProcessPendingReactions fires any queued reactions whose delay has elapsed.
// Called on every NewTurn (100ms tick).
func ProcessPendingReactions(e events.Event) events.ListenerReturn {
	if len(pendingReactions) == 0 {
		return events.Continue
	}

	currentTurn := util.GetTurnCount()
	remaining := make([]PendingReaction, 0, len(pendingReactions))

	for _, pr := range pendingReactions {
		if currentTurn < pr.FireTurn {
			remaining = append(remaining, pr)
			continue
		}

		// Fire the reaction
		mob := mobs.GetInstance(pr.MobInstanceId)
		if mob == nil || mob.Character.Health <= 0 {
			continue
		}

		room := rooms.LoadRoom(mob.Character.RoomId)
		ExecuteAction(mob, pr.Action, room)
	}

	pendingReactions = remaining
	return events.Continue
}

// ResolveTactics returns the effective tactic list for a mob,
// merging preset + custom tactics.
func ResolveTactics(mob *mobs.Mob) []TacticRule {
	var base []TacticRule
	if mob.TacticPreset != "" {
		base = GetPreset(mob.TacticPreset)
	}
	if len(mob.Tactics) > 0 {
		if base == nil {
			return mob.Tactics
		}
		return MergeTactics(base, mob.Tactics)
	}
	return base
}

// BuildTriggerContext assembles the current state for trigger evaluation.
func BuildTriggerContext(mob *mobs.Mob, signal events.MobAISignal) *TriggerContext {
	ctx := &TriggerContext{
		MobChar:   &mob.Character,
		HasAggro:  mob.Character.Aggro != nil,
		HasMemory: mob.CombatMemory != nil && mob.CombatMemory.Grudge,
		IsHidden:  mob.Character.HasBuffFlag(9), // buff 9 = hidden
	}

	// Populate active buff IDs
	ctx.ActiveBuffIds = mob.Character.GetActiveBuffIds()

	// Signal-specific context
	switch signal.SignalType {
	case "combat_start":
		ctx.CombatJustStarted = true
	case "action_complete":
		ctx.LastAction = signal.Detail
	case "player_entered":
		ctx.PlayerEntered = true
	}

	// Resolve target
	if mob.Character.Aggro != nil {
		if mob.Character.Aggro.UserId > 0 {
			if u := users.GetByUserId(mob.Character.Aggro.UserId); u != nil {
				ctx.Target = u.Character
			}
		} else if mob.Character.Aggro.MobInstanceId > 0 {
			if tm := mobs.GetInstance(mob.Character.Aggro.MobInstanceId); tm != nil {
				ctx.Target = &tm.Character
			}
		}
	}

	// Count enemies in room
	if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
		ctx.EnemyCount = len(room.GetPlayers())
	}

	return ctx
}
```

- [ ] **Step 4: Add accessor methods to Mob for lastReactionTurn**

In `internal/mobs/mobs.go`, add methods:

```go
func (m *Mob) GetLastReactionTurn() uint64 {
	return m.lastReactionTurn
}

func (m *Mob) SetLastReactionTurn(turn uint64) {
	m.lastReactionTurn = turn
}
```

Also add discipline variance and growth fields + initialization. In the Mob struct (from Task 1), the `TacticalDiscipline` field is the YAML base value. Add a runtime effective discipline:

```go
effectiveDiscipline float64 // Runtime discipline (base ± variance, grows over time)
disciplineInitialized bool  // Whether effective discipline has been set
```

Add methods:

```go
// GetEffectiveDiscipline returns the mob's current discipline, initializing
// on first call with ±0.1 random variance from the base YAML value.
func (m *Mob) GetEffectiveDiscipline() float64 {
	if !m.disciplineInitialized {
		base := m.TacticalDiscipline
		if base <= 0 {
			base = 0.5
		}
		// ±0.1 variance (uniform random)
		variance := (float64(util.Rand(21)) - 10.0) / 100.0 // -0.10 to +0.10
		m.effectiveDiscipline = base + variance
		if m.effectiveDiscipline < 0 {
			m.effectiveDiscipline = 0
		}
		if m.effectiveDiscipline > 1.0 {
			m.effectiveDiscipline = 1.0
		}
		m.disciplineInitialized = true
	}
	return m.effectiveDiscipline
}

// GrowDiscipline nudges the mob's effective discipline toward 1.0.
// Called after a successful tactic execution.
func (m *Mob) GrowDiscipline(amount float64) {
	m.effectiveDiscipline += amount
	if m.effectiveDiscipline > 1.0 {
		m.effectiveDiscipline = 1.0
	}
}
```

The `amount` for growth should be small — e.g. 0.01 per successful tactic. A mob starting at 0.4 would reach 0.5 after ~10 successful tactic uses, reaching 1.0 after ~60. This is configurable by what value we pass to `GrowDiscipline`.

Then in the reactor (`reactor.go`), replace `mob.TacticalDiscipline` with `mob.GetEffectiveDiscipline()`, and after a tactic fires successfully, call `mob.GrowDiscipline(0.01)`.
```

- [ ] **Step 5: Run tests and verify build**

Run: `go build ./... && go test ./internal/mobai/... -v`
Expected: Clean build, all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/mobai/reactor.go internal/mobai/reactor_test.go \
       internal/events/eventtypes.go internal/mobs/mobs.go
git commit -m "feat: implement mob AI reactor with event-driven reaction queuing"
```

---

### Task 8: Emit MobAISignals from Combat Hooks

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat.go` (emit combat_start)
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go` (emit damage_taken, action_complete)
- Modify: `internal/hooks/hooks.go` (register listeners)

- [ ] **Step 1: Register AI listeners in hooks.go**

In `internal/hooks/hooks.go`, add:

```go
// Mob AI reactor
events.RegisterListener(events.MobAISignal{}, mobai.HandleMobAISignal)
events.RegisterListener(events.NewTurn{}, mobai.ProcessPendingReactions)
```

Add `"github.com/GoMudEngine/GoMud/internal/mobai"` to imports.

- [ ] **Step 2: Emit combat_start signal**

In `internal/hooks/NewRound_DoCombat.go`, when a mob first enters combat (aggro is set), emit:

```go
events.AddToQueue(events.MobAISignal{
	MobInstanceId: mob.InstanceId,
	SignalType:    "combat_start",
	RoomId:        mob.Character.RoomId,
})
```

Find the right location: after aggro validation succeeds and the mob is confirmed to be in combat. This should fire on the first round a mob has aggro.

Add a flag or check to prevent re-emitting every round. The simplest approach: only emit when the mob's `CombatMemory` is nil (first round of combat):

```go
if mob.CombatMemory == nil && mob.Character.Aggro != nil {
	mob.CombatMemory = mobai.SetMemory(
		mob.Character.Aggro.UserId,
		mob.Character.Aggro.MobInstanceId,
		mob.Character.RoomId,
		evt.RoundNumber,
	)
	events.AddToQueue(events.MobAISignal{
		MobInstanceId: mob.InstanceId,
		SignalType:    "combat_start",
		RoomId:        mob.Character.RoomId,
	})
}
```

- [ ] **Step 3: Emit damage_taken signal**

In the mob damage handlers (where `mob.Character.Health -= dmg`), add after damage is applied:

```go
events.AddToQueue(events.MobAISignal{
	MobInstanceId: mob.InstanceId,
	SignalType:    "damage_taken",
	RoomId:        mob.Character.RoomId,
})
```

Find the appropriate locations in `NewRound_DoCombat_helpers.go` — the `handleMobVsPlayer` and similar functions where mob HP is reduced.

- [ ] **Step 4: Emit player_entered signal**

In the `RoomChange` event handler or `go.go` command, when a player enters a room with mobs, emit signals for each mob:

Find where `events.RoomChange` is processed. For each mob in the destination room that has tactics defined:

```go
events.AddToQueue(events.MobAISignal{
	MobInstanceId: mobId,
	SignalType:    "player_entered",
	RoomId:        toRoomId,
})
```

- [ ] **Step 5: Update combat memory on aggro changes**

In `ValidateAggro` / `RetargetOrEnd`, update combat memory location when the mob sees its target:

```go
if mob.CombatMemory != nil {
	mobai.UpdateMemoryLocation(mob.CombatMemory, mob.Character.RoomId, util.GetRoundCount())
}
```

- [ ] **Step 6: Decay combat memory on round tick**

In `NewRound_MobRoundTick.go`, add memory expiration check:

```go
if mob.CombatMemory != nil {
	bal := configs.GetBalanceConfig()
	if mobai.MemoryExpired(mob.CombatMemory, roundCount, int(bal.CombatMemoryDuration)) {
		mob.CombatMemory = nil
	}
}
```

- [ ] **Step 7: Verify build and tests**

Run: `go build ./... && go test ./internal/hooks/... ./internal/mobai/... -count=1`
Expected: Clean build, all tests pass.

- [ ] **Step 8: Commit**

```bash
git add internal/hooks/hooks.go internal/hooks/NewRound_DoCombat.go \
       internal/hooks/NewRound_DoCombat_helpers.go \
       internal/hooks/NewRound_MobRoundTick.go
git commit -m "feat: emit MobAISignals from combat hooks and register reactor listeners"
```

---

### Task 9: Integrate Reactor with Existing Combat AI

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go` (handleMobAIDecision)

The reactive AI should take priority over the old round-tick AI. If a mob has tactics and already queued a reaction this round, skip the legacy AI decision.

- [ ] **Step 1: Update handleMobAIDecision**

In `handleMobAIDecision`, add an early return if the mob has pending reactions or recently reacted:

```go
func handleMobAIDecision(mob *mobs.Mob, c configs.Config) bool {
	// If mob has reactive AI tactics and recently reacted, skip legacy AI
	if len(mobai.ResolveTactics(mob)) > 0 {
		bal := configs.GetBalanceConfig()
		delay := mobai.GetEffectiveReactionDelay(
			mob.ReactionDelay,
			float64(bal.MobReactionDelayMin),
			float64(bal.MobReactionDelayMax),
		)
		cooldownTurns := uint64(delay * float64(configs.GetTimingConfig().TurnsPerSecond()))
		if util.GetTurnCount()-mob.GetLastReactionTurn() < cooldownTurns*2 {
			return true // Reactive AI is handling this mob
		}
	}

	// ... existing legacy AI code unchanged ...
```

This means mobs WITH tactics use the reactive system; mobs WITHOUT tactics fall through to the existing `ChooseSpecialMove` / `ChooseCastAction` / `CombatCommands` path.

- [ ] **Step 2: Add mobai import**

Add `"github.com/GoMudEngine/GoMud/internal/mobai"` to imports in `NewRound_DoCombat_helpers.go`.

- [ ] **Step 3: Verify build and tests**

Run: `go build ./... && go test ./internal/hooks/... -count=1`
Expected: Clean build, all tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/NewRound_DoCombat_helpers.go
git commit -m "feat: integrate reactive AI with legacy combat AI (tactics take priority)"
```

---

### Task 10: Final Integration Test

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -count=1 2>&1 | tail -30`
Expected: All tests pass.

- [ ] **Step 2: Verify no import cycles**

Run: `go build ./...`
Expected: Clean build, no import cycle errors.

- [ ] **Step 3: Commit any cleanup**

```bash
git commit -m "chore: final cleanup for mob AI framework"
```
