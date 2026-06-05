# DOGMud NPC Management System Context

## Overview

The DOGMud mobs system provides comprehensive NPC (Non-Player Character) management with support for AI behaviors, behavior trees, conversation systems, pathfinding, shop management, and complex social dynamics. It features a dual-layer architecture with immutable mob specifications and mutable mob instances, supporting dynamic spawning, behavioral patterns, and sophisticated interaction systems.

**DOGMud Differences from upstream GoMud:**
- Mobs no longer use Level for stat initialization — stats and skills defined directly in YAML
- No `AutoTrain()` or `StatPoints = Level` in mob creation
- Mana removed — mobs use the same three resource pools as players (Health, Stamina, Conviction)
- Species system replaces races for NPC types
- Mobs and players are mechanically identical (same stat/skill resolution)

## Architecture

The mobs system is built around several key components:

### Core Components

**Mob Specifications:**
- Immutable blueprint definitions for all NPC types
- YAML-based storage with automatic loading and validation
- Zone-based organization with hierarchical file structure
- Character integration for stats, equipment, and abilities

**Mob Instances:**
- Runtime instances with unique IDs and state management
- Dynamic spawning and despawning with memory management
- Behavioral state tracking and command scheduling
- Temporary data storage for scripting and AI systems

**AI Behavior System:**
- Activity-based idle command execution
- Combat command selection and execution
- Conversation system integration with multi-mob interactions
- Pathfinding and movement planning

**Social Dynamics:**
- Group-based allegiances and hostilities
- Race-based hatred and alliance systems
- Player relationship tracking and memory
- Alignment-based conflict resolution

## Key Features

### 1. **Dynamic Instance Management**
- Unique instance IDs for each spawned mob
- Automatic stat calculation and equipment validation
- Stats come from species base stats + mob YAML `statpool` bonus (no zone-level autoscaling)
- Stats and skills defined directly in YAML (no level-based scaling)
- Memory management with automatic cleanup

### 2. **Behavioral AI System**
- Activity level-based command frequency
- Idle, angry, and combat command sets
- Boredom tracking and player interaction memory
- Conversation participation with other NPCs
- Behavior tree combat decisions (see `internal/behaviortree/context.md`)

### 2b. **Mob Behavior (chunk 2.6 update)**

Mob behavior is driven entirely by the behavior tree (btree) system —
see `internal/behaviortree/context.md`. The legacy `internal/mobai/`
tactics engine was removed in chunk 2.6; the Mob struct no longer
carries `Tactics`, `TacticPreset`, `ReactionDelay`, or
`TacticalDiscipline` fields. Mob YAMLs no longer support
`tactic_preset:`, `tactics:`, `reaction_delay:`, or
`tactical_discipline:` keys (they're silently ignored if present).

Note: the `CombatMemory` substrate (grudge tracking across flee /
re-engage cycles) was preserved from the deleted engine and now lives
at `internal/mobs/combat_memory.go`. Mob struct still carries the
`CombatMemory *CombatMemory` field and the SetCombatMemory /
CombatMemoryExpired helpers.

Design: `docs/superpowers/specs/2026-05-12-mob-aliveness-2.6-sunset-tactics-engine-design.md`

### 2c. **Mob Lifecycle — Presence Machine (chunk 5)**

`BoredomCounter uint8` and `PreventIdle bool` are **deleted** in chunk 5
(T12). Their roles are taken over by the Presence state machine on
`Character.Presence *presence.Machine` and a new per-mob field
`Character.LastTargetFoundRound uint64`.

**BoredomCounter → Presence:** The boredom semantics now live in
`NewRound_PresenceTick.go`. Each round, the hook computes
`roundNow - mob.Character.LastTargetFoundRound` and fires
`Active→Dormant` when the delta exceeds `PresenceMobDormantAfterRounds`
(default 30 rounds). `Dormant→Despawning` fires after an additional
`PresenceMobDespawnAfterRounds` (default 60 rounds). Both transitions
are vetoed for essential mobs (`IsEssential() || IsCharmed()`).

**PreventIdle → Active veto:** The one-tick suppression that
`PreventIdle` provided (blocking idle command dispatch for a single
round) is subsumed by the Active state itself. A mob in Active state is
always eligible for idle commands; the veto that `PreventIdle` guarded
(skipping an idle tick after a state change) is no longer needed because
the Presence machine's transition itself provides the state boundary.

**Essential-mob veto (T5):** `hooks.Presence_MobVetoes.go` registers
`Active→Dormant` and `Active→Despawning` vetoes that return `ErrVetoed`
when `mob.IsEssential() || mob.Character.IsCharmed()`. Shopkeepers,
foragers, caravan crew, and charmed companions never leave Active.

**Spawning state:** `NewMobPresence()` starts in `Spawning`. On the
next `NewRound_PresenceTick`, the mob advances `Spawning→Active`. This
gives one-round initialization budget before the mob begins receiving
idle ticks and combat-eligible checks.

See `internal/state/presence/context.md` for the full Presence state
list, transition tables, and config knobs.

### 3. **Social and Combat Dynamics**
- Group-based allegiance system
- Race and alignment-based hostility
- Player attack tracking and memory
- Shop ownership and trading behavior

### 4. **Pathfinding and Movement**
- Pre-calculated path following system
- Waypoint-based navigation
- Wandering behavior with distance limits
- Room-based movement constraints

## Mob Structure

### Core Mob Properties
```go
type Mob struct {
    MobId           MobId                    // Unique mob type identifier
    Zone            string                   // Zone this mob belongs to
    InstanceId      int                      // Unique runtime instance ID
    HomeRoomId      int                      // Starting/home room
    Character       characters.Character     // Character stats and properties
    
    // Behavior Properties
    ActivityLevel   int                      // 1-100% activity frequency
    Hostile         bool                     // Attack players on sight
    MaxWander       int                      // Maximum rooms from home
    WanderCount     int                      // Current wander distance
    PreventIdle     bool                     // Disable idle behavior
    
    // AI Command Sets
    IdleCommands    []string                 // Commands executed when idle
    AngryCommands   []string                 // Commands when entering combat
    CombatCommands  []string                 // Commands during combat
    
    // Social Properties
    Groups          []string                 // Group allegiances
    Hates           []string                 // Groups/races this mob hates
    QuestFlags      []string                 // Quest flags for interactions
    
    // Economy
    ItemDropChance  int                      // Chance to drop items on death
    BuffIds         []int                    // Permanent buffs on spawn
    
    // Scripting
    ScriptTag       string                   // Custom script identifier
    
    // Runtime State
    LastIdleCommand uint8                    // Track last idle command used
    BoredomCounter  uint8                    // Rounds since seeing players
    tempDataStore   map[string]any           // Temporary data storage for AI
    conversationId  int                      // Active conversation ID
    Path            PathQueue                // Movement pathfinding queue
    lastCommandTurn uint64                   // Command scheduling tracking
    playersAttacked map[int]struct{}         // Players this mob has attacked
}
```

### Mob Creation and Spawning
```go
// Create new mob instance from specification
// NOTE: In DOGMud, mobs no longer use Level for stat initialization.
// Stats come from species base stats + the mob's statpool bonus.
// The forceStatPool parameter overrides the mob YAML statpool if > 0.
// Zone-level autoscaling was removed in Phase 21.
func NewMobById(mobId MobId, homeRoomId int, forceStatPool int) *Mob {
    if spec, ok := mobs[int(mobId)]; ok {
        instanceCounter++

        // Create copy of mob specification
        mob := *spec
        mob.HomeRoomId = homeRoomId
        mob.Character.RoomId = homeRoomId
        mob.InstanceId = instanceCounter
        mob.Character.PlayerDamage = make(map[int]int)

        // Initialize resource pools from stats
        mob.Character.Health = mob.Character.HealthMax.Value
        mob.Character.Stamina = mob.Character.StaminaMax.Value
        mob.Character.Conviction = mob.Character.ConvictionMax.Value

        // Apply permanent buffs
        mob.Character.SetPermaBuffs(mob.BuffIds)

        // Validate all equipment
        mob.validateEquipment()

        // Store instance
        mobInstances[mob.InstanceId] = &mob
        return &mob
    }
    return nil
}
```

## AI Behavior System

### Command Execution and Scheduling
```go
// Schedule commands with timing
func (m *Mob) Command(inputTxt string, waitSeconds ...float64) {
    readyTurn := util.GetTurnCount()
    turnDelay := uint64(0)
    
    // Ensure sequential command execution
    if readyTurn > m.lastCommandTurn {
        m.lastCommandTurn = readyTurn
    } else {
        readyTurn = m.lastCommandTurn
    }
    
    if len(waitSeconds) > 0 {
        turnDelay = uint64(float64(configs.GetTimingConfig().SecondsToTurns(1)) * waitSeconds[0])
    }
    
    // Handle multiple commands separated by semicolons
    for i, cmd := range strings.Split(inputTxt, ";") {
        m.lastCommandTurn = readyTurn + turnDelay + uint64(i)
        
        events.AddToQueue(events.Input{
            MobInstanceId: m.InstanceId,
            InputText:     cmd,
            ReadyTurn:     m.lastCommandTurn,
        })
    }
}

// Sleep functionality
func (m *Mob) Sleep(seconds int) {
    m.Command("noop", float64(seconds))
}
```

### Behavioral Command Selection
```go
// Get idle command based on mob or race defaults
func (m *Mob) GetIdleCommand() string {
    // 1% chance to do nothing (prevents required empty commands)
    if util.Rand(100) == 0 {
        return ""
    }
    
    // Check mob-specific idle commands
    if len(m.IdleCommands) > 0 {
        return m.IdleCommands[util.Rand(len(m.IdleCommands))]
    }
    
    return ""
}

// Get angry command when entering combat
func (m *Mob) GetAngryCommand() string {
    // Check mob-specific angry commands
    if len(m.AngryCommands) > 0 {
        return m.AngryCommands[util.Rand(len(m.AngryCommands))]
    }
    
    // Fall back to race-based commands
    if r := races.GetRace(m.Character.RaceId); r != nil {
        if len(r.AngryCommands) > 0 {
            return r.AngryCommands[util.Rand(len(r.AngryCommands))]
        }
    }
    
    return ""
}
```

## Social Dynamics System

### Group Allegiances and Hostilities

`IsGuardMob(groups []string) bool` (package func) reports whether a group list
includes the law-enforcement `"guard"` marker (lifted from `internal/hooks` 2026-06-05
so `internal/seeders` can share it). NB: distinct from a guard *faction* id and from
the combat-stance `Character.IsGuard()` predicate.

```go
// Check if two mobs are allies
func (r *Mob) ConsidersAnAlly(m *Mob) bool {
    if m.MobId == r.MobId {
        return true // Same mob type = ally
    }
    
    if len(m.Groups) == 0 && len(r.Groups) == 0 {
        return true // No factions = neutral allies
    }
    
    // Check for shared group membership
    for _, targetGroup := range r.Groups {
        for _, testGroup := range m.Groups {
            if testGroup == targetGroup {
                return true
            }
        }
    }
    
    return false
}

// Check race-based hatred
func (r *Mob) HatesRace(raceName string) bool {
    raceName = strings.ToLower(raceName)
    for _, hateGroup := range r.Hates {
        if hateGroup == raceName {
            return true
        }
    }
    return false
}

// Check alignment-based hostility
func (r *Mob) HatesAlignment(otherAlignment int8) bool {
    // Neutral alignment = no hatred
    if characters.AlignmentToString(r.Character.Alignment) == "neutral" || 
       characters.AlignmentToString(otherAlignment) == "neutral" {
        return false
    }
    
    // Same side = no hatred
    if (r.Character.Alignment > 0 && otherAlignment > 0) ||
       (r.Character.Alignment < 0 && otherAlignment < 0) {
        return false
    }
    
    // Check alignment difference threshold
    delta := int(math.Abs(float64(r.Character.Alignment) - float64(otherAlignment)))
    return delta > characters.AlignmentAggroThreshold
}
```

### Player Relationship Tracking
```go
// Track player attacks for memory system
func (m *Mob) PlayerAttacked(userId int) {
    if m.playersAttacked == nil {
        m.playersAttacked = map[int]struct{}{}
    }
    m.playersAttacked[userId] = struct{}{}
}

func (m *Mob) HasAttackedPlayer(userId int) bool {
    if m.playersAttacked == nil {
        return false
    }
    _, ok := m.playersAttacked[userId]
    return ok
}

// Global hostility tracking
func MakeHostile(groupName string, userId int, rounds int) {
    if _, ok := mobsHatePlayers[groupName]; !ok {
        mobsHatePlayers[groupName] = make(map[int]int)
    }
    
    if mobsHatePlayers[groupName][userId] < rounds {
        mobsHatePlayers[groupName][userId] = rounds
    }
}

func IsHostile(groupName string, userId int) bool {
    if group, ok := mobsHatePlayers[groupName]; ok {
        _, hostile := group[userId]
        return hostile
    }
    return false
}
```

## Conversation System Integration

### Multi-Mob Conversations
```go
// Check if mob is in conversation
func (m *Mob) InConversation() bool {
    return m.conversationId > 0
}

// Set conversation participation
func (m *Mob) SetConversation(id int) {
    m.conversationId = id
}

// Execute conversation actions
func (m *Mob) Converse() {
    mobInst1, mobInst2, actions := conversations.GetNextActions(m.conversationId)
    
    var mob1, mob2 *Mob
    
    // Determine which mob is which in the conversation
    if mobInst1 == int(m.InstanceId) {
        mob1 = m
        mob2 = GetInstance(mobInst2)
    } else {
        mob1 = GetInstance(mobInst1)
        mob2 = m
    }
    
    // Execute conversation actions
    for _, act := range actions {
        if len(act) >= 3 {
            target := act[0:3]
            cmd := act[3:]
            
            // Replace mob references in commands
            cmd = strings.ReplaceAll(cmd, " #1 ", " "+mob1.ShorthandId()+" ")
            cmd = strings.ReplaceAll(cmd, " #2 ", " "+mob2.ShorthandId()+" ")
            
            if target == "#1 " {
                mob1.Command(cmd)
            } else {
                mob2.Command(cmd, 1)
            }
        }
    }
    
    // Clean up completed conversations
    if conversations.IsComplete(m.conversationId) {
        conversations.Destroy(m.conversationId)
        mob1.SetConversation(0)
        mob2.SetConversation(0)
    }
}
```

## Pathfinding and Movement

### Path Queue System
```go
type PathQueue struct {
    roomQueue   []PathRoom
    currentRoom PathRoom
}

// Path management
func (p *PathQueue) SetPath(path []PathRoom) {
    p.roomQueue = path
    p.currentRoom = nil
}

func (p *PathQueue) Next() PathRoom {
    if len(p.roomQueue) == 0 {
        return nil
    }
    p.currentRoom = p.roomQueue[0]
    p.roomQueue = p.roomQueue[1:]
    return p.currentRoom
}

// Get remaining waypoints
func (p *PathQueue) Waypoints() []int {
    wpList := []int{}
    if p.currentRoom != nil && p.currentRoom.Waypoint() {
        wpList = append(wpList, p.currentRoom.RoomId())
    }
    
    for _, r := range p.roomQueue {
        if r.Waypoint() {
            wpList = append(wpList, r.RoomId())
        }
    }
    
    return wpList
}
```

## Shop and Trading System

### NPC Merchant Behavior
```go
// Check if mob has shop
func (m *Mob) HasShop() bool {
    return len(m.Character.Shop) > 0
}

// Calculate sell price for items
func (m *Mob) GetSellPrice(item items.Item) int {
    if item.IsSpecial() {
        return 0 // Don't buy special items
    }
    
    itemType := item.GetSpec().Type
    itemSubtype := item.GetSpec().Subtype
    value := 0
    likesType := false
    likesSubtype := false
    newAddition := true
    priceScale := 0.0
    
    currentSaleItems := m.Character.Shop.GetInstock()
    
    // Check existing inventory for pricing
    for _, stockItm := range currentSaleItems {
        if stockItm.ItemId == item.ItemId {
            newAddition = false
            likesType = true
            likesSubtype = true
            value = stockItm.Price
            // Reduce price based on current stock
            priceScale = 1.0 - (float64(stockItm.Quantity) / 20)
            break
        }
        
        // Check for type/subtype preferences
        tmpItm := items.New(stockItm.ItemId)
        if tmpItm.GetSpec().Type == itemType {
            likesType = true
            priceScale += 0.5
        }
        if tmpItm.GetSpec().Subtype == itemSubtype {
            likesSubtype = true
            priceScale += 0.5
        }
    }
    
    // Limit inventory variety
    if newAddition && len(currentSaleItems) >= 20 {
        return 0
    }
    
    if value == 0 {
        value = item.GetSpec().Value
    }
    
    // Apply price scaling (max 25% of item value)
    priceScale = math.Max(0, math.Min(priceScale * 0.25, 1.0))
    return int(math.Ceil(float64(value) * priceScale))
}
```

## Temporary Data Storage

```go
// Runtime data storage for AI systems
func (m *Mob) SetTempData(key string, value any) {
    if m.tempDataStore == nil {
        m.tempDataStore = make(map[string]any)
    }
    
    if value == nil {
        delete(m.tempDataStore, key)
        return
    }
    m.tempDataStore[key] = value
}

func (m *Mob) GetTempData(key string) any {
    if m.tempDataStore == nil {
        m.tempDataStore = make(map[string]any)
    }
    
    if value, ok := m.tempDataStore[key]; ok {
        return value
    }
    return nil
}
```

## Crafter Mob System

Mobs with `crafter: true` autonomously craft items into their shop inventory.
Crafting fires once per material restock cycle (every `CrafterMaterialRestockRate`
rounds, default 200 ≈ 13 minutes real time). On each restock tick, materials arrive
and the mob immediately attempts one craft.

### Crafter Mob Fields
```go
// YAML-configured fields on Mob struct
Crafter                 bool     `yaml:"crafter,omitempty"`
CrafterSkill            string   `yaml:"crafterskill,omitempty"`
CrafterRecipeIds        []string `yaml:"crafterrecipeids,omitempty"`
CrafterRestockMaterials []int    `yaml:"crafterrestockmaterials,omitempty"`

// Transient runtime fields (not persisted)
crafterLastRestockRound uint64   `yaml:"-"`
```

### Crafting Algorithm (`crafter.go`)
```go
func TickMobCraft(mob *Mob) *CraftResult
```
1. Guard: skip if `!Crafter`, `!CrafterEnabled` config, or mob is in combat
2. Wait for restock tick: `roundCount - lastRestock >= CrafterMaterialRestockRate`
3. Restock materials: add items from `CrafterRestockMaterials` to mob's backpack
4. Pick eligible recipe: filter `CrafterRecipeIds` by skill match, skill minimum,
   and available ingredients; pick one at random
5. Roll success via `crafting.CalcSuccessChance(skillLevel, recipe.SkillMinimum)`
6. Consume ingredients regardless of outcome
7. On success: `Shop.StockItem(output.ItemId)` + `OnSkillUse()` for progression
8. Return `CraftResult` struct for the caller (hook) to emit room messages and
   world events

### CraftResult Struct
```go
type CraftResult struct {
    Success      bool
    RecipeName   string
    OutputItemId int
    SkillMinimum int
    MobName      string
    Zone         string
}
```

The caller (`MobIdle_HandleIdleMobs.go`) handles room flavor text and emits
`MobCraftedRare` world events when `SkillMinimum >= CrafterRareThreshold`.

### Config (in `config.balance.go`)
- `CrafterEnabled` (default true) — master toggle
- `CrafterMaterialRestockRate` (default 200 rounds) — restock/craft interval
- `CrafterRareThreshold` (default 3) — skill minimum for rare craft events

### Example Crafter Mob YAML
```yaml
crafter: true
crafterskill: blacksmithing
crafterrecipeids:
  - iron-dagger
  - iron-short-sword
  - iron-buckler
crafterrestockmaterials:
  - 40001  # iron ingot
  - 40002  # leather strip
  - 40003  # wooden plank
character:
  skills:
    blacksmithing: 10
  shop:
    - itemid: 40001
      quantity: 0
      quantitymax: 0
      price: 1
```

Active crafter mobs: Blacksmith Kerra (97), Apothecary Voss (98), Jeweler Tess (108),
Weaver Maren (110) — all in Thornwall City.

---

## Pack Scaling System

Mobs sharing a group tag gain Training bonuses when they survive consecutive
rounds together. Death of any member resets the counter.

### Pack Scaling Fields
```go
// On Mob struct (transient, not persisted in YAML)
PackBonusTotal int `yaml:"-"`
```

### Algorithm (`pack_scaling.go`)
```go
func TickPackSurvival() []PackBonus
```
1. Build `map[groupTag][]instanceId` for all live mobs with groups
2. Only count groups with >= 2 members
3. If member count dropped since last tick → reset survival counter
4. Else increment counter; if >= `PackSurvivalRounds` (default 10):
   - Reset counter
   - Add `PackBonusTrainingPts` (default 1) to a stat based on archetype
   - Cap at `PackMaxBonus` (default 5) total
   - Return `PackBonus` struct for caller to emit room message + world event

### PackBonus Struct
```go
type PackBonus struct {
    GroupTag  string
    MemberIds []int
    ReachedMax bool
}
```

### Stat Selection by Archetype
- `"fighting"` archetype → strength Training
- `"casting"` archetype → willpower Training
- Default → vitality Training

### Config (in `config.balance.go`)
- `PackScalingEnabled` (default true)
- `PackSurvivalRounds` (default 10) — rounds together before bonus
- `PackBonusTrainingPts` (default 1) — Training points per bonus
- `PackMaxBonus` (default 5) — max total bonus per mob

Called from `NewRound_MobRoundTick.go` before the per-mob loop. The hook handles
room messaging and world event emission (avoids import cycle with `rooms` package).

---

## Mob Mutation Acquisition

Mobs accumulate `MutationProgress` during combat, using the same threshold system
as players but at a reduced rate (`MobMutationRate`, default 0.3 = 30% of player rate).

### How It Works
- Processed in `NewRound_MobRoundTick.go` inside the per-mob loop (after buff
  triggers, before `Validate()`)
- Guard: `MobMutationEnabled` config + mob must be in combat (`Aggro != nil`)
- Progress: `+= MutationProgressGainPerRound * MobMutationRate`
- Threshold: same `MutationBaseProgress * MutationProgressScale^load` formula
- On acquire: uses `mutations.GetWeightedPool()` and `RollAcquisition()`
- On deepen: uses `mutations.RollDeepening()`
- Room flavor text emitted for mutation events
- World events emitted with significance based on mutation rarity:
  - Rarity >= 8 → Global
  - Rarity >= 5 → Regional
  - Rarity < 5 → Local
  - Deepening to level 3 → bump one tier

### Config (in `config.balance.go`)
- `MobMutationEnabled` (default false — opt-in)
- `MobMutationRate` (default 0.3)

Persistence handled automatically — `MobInstanceData` saves `Mutations` and
`MutationProgress` fields.

---

## Special Mob Types and Behaviors

### Tameable Mobs
```go
// Check if mob can be tamed by players
func (m *Mob) IsTameable() bool {
    if m.HasShop() {
        return false // Merchants can't be tamed
    }
    if len(m.ScriptTag) > 0 {
        return false // Tagged mobs can't be tamed
    }
    if r := races.GetRace(m.Character.RaceId); r != nil {
        if !r.Tameable {
            return false // Race doesn't allow taming
        }
    }
    return true
}
```

### Persistent vs Temporary Mobs
```go
// Check if mob should despawn when room unloads
func (m *Mob) Despawns() bool {
    if m.HasShop() {
        return false // Merchants are persistent
    }
    return true // Most mobs despawn with room
}
```

## Memory and Performance Management

### Instance Tracking
```go
// Memory usage reporting
func GetMemoryUsage() map[string]util.MemoryResult {
    ret := map[string]util.MemoryResult{}
    
    ret["mobs"] = util.MemoryResult{util.MemoryUsage(mobs), len(mobs)}
    ret["allMobNames"] = util.MemoryResult{util.MemoryUsage(allMobNames), len(allMobNames)}
    ret["mobInstances"] = util.MemoryResult{util.MemoryUsage(mobInstances), len(mobInstances)}
    ret["mobsHatePlayers"] = util.MemoryResult{util.MemoryUsage(mobsHatePlayers), len(mobsHatePlayers)}
    
    return ret
}

// Recent death tracking
func TrackRecentDeath(instanceId int) {
    recentlyDied[instanceId] = int(util.GetRoundCount())
}

func RecentlyDied(instanceId int) bool {
    // Automatic cleanup of old entries
    if len(recentlyDied) > 30 {
        roundNow := int(util.GetRoundCount())
        for k, v := range recentlyDied {
            if roundNow-v > 15 {
                delete(recentlyDied, k)
            }
        }
    }
    
    _, ok := recentlyDied[instanceId]
    return ok
}
```

### Hostility Management
```go
// Reduce hostility over time
func ReduceHostility() {
    for groupName, group := range mobsHatePlayers {
        for userId, rounds := range group {
            rounds--
            if rounds < 1 {
                delete(mobsHatePlayers[groupName], userId)
            } else {
                mobsHatePlayers[groupName][userId] = rounds
            }
        }
        
        // Clean up empty groups
        if len(mobsHatePlayers[groupName]) < 1 {
            delete(mobsHatePlayers, groupName)
        }
    }
}
```

## File Organization and Persistence

### Zone-Based File Structure
```go
// Automatic file organization by zone
func (m *Mob) Filepath() string {
    zone := ZoneNameSanitize(m.Zone)
    return util.FilePath(zone, "/", m.Filename())
}

func (m *Mob) Filename() string {
    if name, ok := mobNameCache[m.MobId]; ok {
        return fmt.Sprintf("%d-%s.yaml", m.Id(), util.ConvertForFilename(name))
    }
    // Fallback to character name
    filename := util.ConvertForFilename(m.Character.Name)
    return fmt.Sprintf("%d-%s.yaml", m.Id(), filename)
}

// Zone name sanitization
func ZoneNameSanitize(zone string) string {
    if zone == "" {
        return ""
    }
    zone = strings.ReplaceAll(zone, " ", "_")
    return strings.ToLower(zone)
}
```

### Data Loading and Validation
```go
// Load all mob specifications from files
func LoadDataFiles() {
    start := time.Now()
    
    tmpMobs, err := fileloader.LoadAllFlatFiles[int, *Mob](
        configs.GetFilePathsConfig().DataFiles.String() + "/mobs"
    )
    if err != nil {
        panic(err)
    }
    
    mobs = tmpMobs
    clear(mobNameCache)
    
    // Build name cache and validation
    for _, mob := range mobs {
        mob.Character.CacheDescription()
        allMobNames = append(allMobNames, mob.Character.Name)
        mobNameCache[mob.MobId] = mob.Character.Name
    }
    
    mudlog.Info("mobs.LoadDataFiles()", 
        "loadedCount", len(mobs), 
        "Time Taken", time.Since(start))
}
```

## Integration Patterns

### Event System Integration
```go
// Buff application through events
func (m *Mob) AddBuff(buffId int, source string) {
    events.AddToQueue(events.Buff{
        MobInstanceId: m.InstanceId,
        BuffId:        buffId,
        Source:        source,
    })
}

// Command execution through Input events
// All mob commands go through the same event system as player commands
```

### Character System Integration
```go
// Mobs use the same character system as players
type Mob struct {
    Character characters.Character  // Full character integration
}

// Stats and skills defined directly in YAML — no level-based scaling
// Equipment bonuses and stat modifications
// Same skill/stat resolution as players (no special cases)
```

## Usage Examples

### Creating and Managing Mob Instances
```go
// Spawn mob in specific room (stats come from species + statpool)
// forceStatPool=0 means use the mob YAML's statpool value
mob := mobs.NewMobById(mobs.MobId(123), roomId, 0)
if mob != nil {
    // Mob spawned successfully
    room.AddMob(mob.InstanceId)
}

// Schedule mob commands
mob.Command("say Hello there!")
mob.Command("emote waves", 2.0) // Wait 2 seconds
mob.Command("look; smile", 1.0) // Multiple commands
```

### AI Behavior Implementation
```go
// Idle behavior processing
if mob.ActivityLevel > util.Rand(100) {
    idleCmd := mob.GetIdleCommand()
    if idleCmd != "" {
        mob.Command(idleCmd)
    }
}

// Combat initiation
if mob.Hostile && playerInRoom {
    angryCmd := mob.GetAngryCommand()
    if angryCmd != "" {
        mob.Command(angryCmd)
    }
    // Start combat...
}
```

### Social Dynamics
```go
// Check relationships before combat
func shouldAttack(attacker *Mob, target *Mob) bool {
    if attacker.ConsidersAnAlly(target) {
        return false
    }
    
    if attacker.HatesMob(target) {
        return true
    }
    
    return attacker.Hostile
}
```

## Dependencies

- `internal/characters` - Character system integration for stats and equipment
- `internal/events` - Event system for command scheduling and buff application
- `internal/conversations` - Multi-mob conversation system
- `internal/items` - Item system for equipment and inventory management
- `internal/species` - Species system for default behaviors and restrictions
- `internal/buffs` - Status effect system for permanent and temporary effects
- `internal/configs` - Configuration management for file paths and timing
- `internal/crafting` - Recipe lookup and ingredient management for crafter mobs
- `internal/skills` - Skill tag types for crafter recipe filtering
- `internal/util` - Utility functions for randomization, file operations, and validation
- `internal/fileloader` - YAML file loading and validation system

## Mob Creation and File Management

### New Mob Creation System
```go
// Create new mob file
func CreateNewMobFile(newMobInfo Mob) (MobId, error) {
    newMobInfo.MobId = getNextMobId()
    
    if newMobInfo.MobId == 0 {
        return 0, errors.New("Could not find a new mob id to assign.")
    }
    
    // Validate mob configuration
    if err := newMobInfo.Validate(); err != nil {
        return 0, err
    }
    
    // Save to file system with optional careful save mode
    saveModes := []fileloader.SaveOption{}
    if configs.GetFilePathsConfig().CarefulSaveFiles {
        saveModes = append(saveModes, fileloader.SaveCareful)
    }
    
    if err := fileloader.SaveFlatFile[*Mob](
        configs.GetFilePathsConfig().DataFiles.String()+"/mobs", 
        &newMobInfo, 
        saveModes...
    ); err != nil {
        return 0, err
    }
    
    // Update in-memory cache
    allMobNames = append(allMobNames, newMobInfo.Character.Name)
    mobNameCache[newMobInfo.MobId] = newMobInfo.Character.Name
    mobs[newMobInfo.Id()] = &newMobInfo
    
    return newMobInfo.MobId, nil
}

// Automatic ID assignment
func getNextMobId() MobId {
    lowestFreeId := MobId(0)
    for _, mInfo := range mobs {
        if mInfo.MobId >= lowestFreeId {
            lowestFreeId = mInfo.MobId + 1
        }
    }
    return lowestFreeId
}
```

### File System Integration
- **Automatic ID Assignment**: Sequential ID allocation to prevent conflicts
- **Careful Save Mode**: Optional backup creation during file operations
- **Cache Synchronization**: Immediate update of in-memory caches after creation

This comprehensive mob system provides sophisticated NPC management with AI behaviors, social dynamics, file management capabilities, and seamless integration with all other game systems.

---

## Shop Persistence & Living Economy

### Shop State Files
Shop economic state (stock levels, NPC gold, restock timers) persists
separately from mob instance data:

- **Path:** `_datafiles/world/dogmud/shops/{zone}/{mobid}-room{roomid}.yaml`
- **NOT** the same as `mobs.instances/` — do not include in instance save
  cleanup SOP
- Deleting a shop file resets that merchant to template defaults (500g
  starting gold, base stock levels from mob YAML)

### Initialization
`RegisterMobShop(mob *Mob)` is called at mob spawn time. It loads the
persisted `ShopInventory` if the file exists; otherwise it initializes
from the mob's `Character.Shop` seed data (the legacy `[]ShopItem` slice).

### Non-Combatant Flag
Mobs with `non_combatant: true` in their YAML are protected from attack,
theft, and harmful spell targeting. Intended for shop NPCs and other
peaceful characters the player should never be able to kill outright.

### Dynamic Pricing
Prices range from 0.25x (overstocked) to 5.0x (out of stock), driven by
the `ShopAbundanceThreshold` config knob and normalized per item by restock
quantity.

Config knobs (in `config.balance.go`):
- `ShopBuyRatio` — fraction of item value paid when buying player items
- `ShopPriceFloor` / `ShopPriceCeiling` — dynamic price multiplier bounds
- `ShopAbundanceThreshold` — stock level that triggers floor pricing
- `ShopMaterialReserve` — units of each material the NPC holds back
- `ShopGoldReserveRatio` — fraction of StartingGold the NPC won't spend
- `BarterMaxDiscount` / `BarterMaxBonus` — barter skill effect bounds

### NPC Craft Decision Priority
When a crafter mob evaluates what to do on a restock tick, it ranks
candidates in this order:

1. **Self-gear upgrade** — craft equipment that is strictly better than
   what the mob currently wears (`IsUpgrade()` comparison)
2. **Profitable stock** — craft items where product value > material cost
3. **Profitable salvage** — break down overstock into scarce materials

The `ShopMaterialReserve` limit prevents NPCs from consuming all materials
on self-gear. The `ShopGoldReserveRatio × StartingGold` floor prevents
NPCs from spending themselves into poverty on gear upgrades.

---

## Schedules (chunk 3.2)

Mobs with `schedule_id:` set follow daily routines authored in
`_datafiles/world/dogmud/schedules/<zone>/<id>.yaml`. See
`docs/schemas/schedule.md` for the full schema.

- `schedule.go`: `Schedule`, `ScheduleSegment`, `GetSchedule`,
  `CurrentSegment`, `applyScheduleSpawnOverride`, test helpers
  (`RegisterScheduleForTest`, `UnregisterScheduleForTest`).
- `schedule_loader.go`: `LoadSchedules`, `validateScheduleStandalone`,
  `validateScheduleAgainstWorld`, `SetScheduleWorldValidator` (DI
  injection used in main.go to break the `mobs ← rooms` import cycle).
  Called from `LoadDataFiles` after mob templates load.
- Spawn override: `newMobByIdInternal` calls
  `applyScheduleSpawnOverride` to place scheduled mobs at the
  current segment's target room.
- Crafter activity gate: `TickMobCraft` returns nil when a
  scheduled mob's current segment activity != "craft".

---

## Sleeping (chunk 3.3)

- `sleeper.go`: `OnSleeperWoken(c *characters.Character)` — central
  wake-event hook. Stamps `schedule_wake_round` MiscData for
  scheduled mobs so the schedule executor's grace cooldown can
  suppress re-sleep. No-op for players and unscheduled mobs.
- Schedule executor (`internal/hooks/NewRound_IdleMobs_schedule.go`)
  recognizes `activity: sleeping` segments. On entry: `mob.Command("sleep")`
  once at target. On exit: `CancelBuffsWithFlag(buffs.Sleeping)`.
- Grace cooldown: after a forced wake the executor reads
  `schedule_wake_round` from MiscData and suppresses re-sleep for
  `ScheduleWakeGraceRounds` rounds (config, default 50).

---

## Patrols (chunk 3.4)

Mobs with `patrol_id:` set follow waypoint patrols authored in
`_datafiles/world/dogmud/patrols/<zone>/<id>.yaml`. See
`docs/schemas/patrol.md` for the full schema.

- `patrol.go`: `Patrol`, `PatrolWaypoint`, `GetPatrol`,
  `NextWaypoint`, test helpers.
- `patrol_loader.go`: `LoadPatrols`, `validatePatrolStandalone`,
  `validatePatrolAgainstWorld`, `SetPatrolWorldValidator` (DI
  injection used in main.go to break the mobs ← rooms import
  cycle). Called from `LoadDataFiles` before `LoadSchedules`
  (schedule loader cross-checks segment patrol_ids).
- Spawn override: `applyScheduleSpawnOverride` falls back to
  the patrol's first waypoint when a patrol segment has no
  `target_room`.