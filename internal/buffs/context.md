# GoMud Buffs System Context

## Overview

The GoMud buffs system provides comprehensive temporary status effects for characters with support for stat modifications, behavioral flags, round-based triggers, and duration management. It features a dual-layer architecture with immutable buff specifications and mutable buff instances, supporting complex timing mechanics, permanent buffs, and sophisticated flag-based behavior modification.

## Architecture

The buffs system is built around several key components:

### Core Components

**Buff Specifications (BuffSpec):**
- Immutable blueprint definitions for all buff types
- YAML-based storage with automatic loading and validation
- Time-based trigger rate calculations with game time integration
- Stat modification definitions and behavioral flags
- Config-driven tick behaviors with stat scaling

**Buff Instances (Buff):**
- Runtime instances with unique state tracking
- Round-based trigger counters and expiration management
- Source tracking for buff origin identification
- Permanent buff support for equipment and racial effects
- Start event queuing for delayed activation

**Buffs Collection (Buffs):**
- Efficient collection management with flag indexing
- Fast lookup maps for buff IDs and flags
- Automatic validation and rebuilding of internal indexes
- Batch operations for triggering and pruning

## Key Features

### 1. **Comprehensive Flag System**
- Behavioral modification flags (combat, movement, fleeing restrictions)
- Death prevention and revival mechanics
- Equipment interaction flags (permanent gear, curse removal)
- Status effect flags (poison, accuracy, stealth, vision enhancement)
- Environmental interaction flags (water cancellation, light emission)

### 2. **Advanced Timing System**
- Round-based trigger intervals with game time integration
- Flexible trigger rates using time string parsing
- Unlimited duration support for permanent effects
- Precise expiration tracking and automatic cleanup
- Trigger counting with configurable limits

### 3. **Stat Modification Integration**
- Dynamic stat bonuses and penalties
- Cumulative effects from multiple buffs
- Integration with character stat system
- Racial and equipment stat modifications
- Combat effectiveness modifiers

## Buff Structure

### Buff Specification Structure
```go
type BuffSpec struct {
    BuffId        int               // Unique identifier
    Name          string            // Display name
    Description   string            // Description text
    Secret        bool              // Hidden from player view
    TriggerNow    bool              // Immediate trigger on application
    TriggerRate   string            // Time-based trigger frequency
    RoundInterval int               // Calculated round interval
    TriggerCount  int               // Total trigger limit
    StatMods      statmods.StatMods // Stat modifications
    Flags         []Flag            // Behavioral flags
}
```

### Buff Instance Structure
```go
type Buff struct {
    BuffId         int    // Reference to BuffSpec
    Source         string // Origin identifier (spell, item, area)
    OnStartWaiting bool   // Pending start event
    PermaBuff      bool   // Permanent buff flag
    RoundCounter   int    // Elapsed rounds
    TriggersLeft   int    // Remaining triggers
}
```

### Buffs Collection Structure
```go
type Buffs struct {
    List      []*Buff           // Active buff instances
    buffFlags map[Flag][]int    // Flag to buff index mapping
    buffIds   map[int]int       // BuffId to index mapping
}
```

## Flag System

### Behavioral Flags
```go
const (
    // Combat and Movement Restrictions
    NoCombat       Flag = "no-combat"        // Prevents combat initiation
    NoMovement     Flag = "no-go"            // Prevents movement
    NoFlee         Flag = "no-flee"          // Prevents fleeing combat
    
    // Cancellation Conditions
    CancelIfCombat Flag = "cancel-on-combat" // Removes buff when combat starts
    CancelOnAction Flag = "cancel-on-action" // Removes buff on any action
    CancelOnWater  Flag = "cancel-on-water"  // Removes buff in water
    
    // Death and Revival
    ReviveOnDeath  Flag = "revive-on-death"  // Prevents death once
    
    // Equipment Interaction
    PermaGear      Flag = "perma-gear"       // Equipment cannot be removed
    RemoveCurse    Flag = "remove-curse"     // Allows cursed item removal
    
    // Status Effects
    Poison         Flag = "poison"           // Harmful poison effect
    Drunk          Flag = "drunk"            // Intoxication effects
    Hidden         Flag = "hidden"           // Stealth/invisibility
    Accuracy       Flag = "accuracy"         // Enhanced hit chance
    Blink          Flag = "blink"            // Dodge enhancement
    
    // Sensory Enhancement
    EmitsLight     Flag = "lightsource"      // Provides illumination
    SuperHearing   Flag = "superhearing"     // Enhanced hearing
    NightVision    Flag = "nightvision"      // See in darkness
    SeeHidden      Flag = "see-hidden"       // Detect hidden entities
    SeeNouns       Flag = "see-nouns"        // Enhanced object identification
    
    // Environmental Status
    Warmed         Flag = "warmed"           // Temperature regulation
    Hydrated       Flag = "hydrated"         // Hydration status
    Thirsty        Flag = "thirsty"          // Dehydration status
)
```

### Flag Usage Patterns
```go
// Check for specific behavioral flags
func (bs *Buffs) HasFlag(action Flag, expire bool) bool {
    if action != All {
        if _, ok := bs.buffFlags[action]; !ok {
            return false
        }
    }
    
    found := false
    for index, b := range bs.List {
        if b.Expired() {
            continue
        }
        
        bSpec := GetBuffSpec(b.BuffId)
        for _, flag := range bSpec.Flags {
            if flag == action || action == All {
                found = true
                
                // Optionally expire the buff when checked
                if expire {
                    if b.BuffId == 0 { // Special buff 0 handling
                        bs.List = append(bs.List[:index], bs.List[index+1:]...)
                    } else {
                        b.TriggersLeft = TriggersLeftExpired
                        bs.List[index] = b
                    }
                    break
                }
                
                return found
            }
        }
    }
    
    return found
}

// Get all buff IDs with specific flag
func (bs *Buffs) GetBuffIdsWithFlag(action Flag) []int {
    buffIds := []int{}
    for _, idx := range bs.buffFlags[action] {
        buffIds = append(buffIds, bs.List[idx].BuffId)
    }
    return buffIds
}
```

## Timing and Trigger System

### Round-Based Triggers
```go
// Trigger buffs based on round intervals
func (bs *Buffs) Trigger(buffId ...int) (triggeredBuffs []*Buff) {
    for idx, b := range bs.List {
        // Handle specific buff triggering if requested
        if len(buffId) > 0 {
            found := false
            for _, id := range buffId {
                if b.BuffId == id {
                    found = true
                    break
                }
            }
            if !found {
                continue
            }
        }
        
        if buffInfo := GetBuffSpec(b.BuffId); buffInfo != nil {
            if b.TriggersLeft > 0 {
                b.RoundCounter++
                
                // Check if it's time to trigger
                if b.RoundCounter%buffInfo.RoundInterval == 0 {
                    triggeredBuffs = append(triggeredBuffs, b)
                    
                    // Decrement triggers unless unlimited
                    if b.TriggersLeft != TriggersLeftUnlimited {
                        b.TriggersLeft--
                    } else {
                        // Reset counter to prevent overflow
                        b.RoundCounter = 0
                    }
                }
                
                bs.List[idx] = b
            }
        }
    }
    
    return triggeredBuffs
}
```

### Duration Calculations
```go
// Calculate remaining and total duration
func GetDurations(buff *Buff, spec *BuffSpec) (roundsLeft int, totalRounds int) {
    totalRounds = spec.TriggerCount * spec.RoundInterval
    roundsLeft = totalRounds - buff.RoundCounter
    return roundsLeft, totalRounds
}

// Check if buff has expired
func (b *Buff) Expired() bool {
    return b.TriggersLeft <= TriggersLeftExpired
}
```

### Time String Processing
```go
// Validate and convert time strings to round intervals
func (b *BuffSpec) Validate() error {
    // Special handling for logout/meditation buff
    if b.BuffId == 0 {
        b.TriggerCount = int(configs.GetNetworkConfig().LogoutRounds)
    }
    
    // Convert time string to round interval using game time system
    b.RoundInterval = int(validationCalculator.AddPeriod(b.TriggerRate) - validationRound)
    
    if b.TriggerCount < 1 {
        return fmt.Errorf("buffId %d (%s) has TriggerCount < 1", b.BuffId, b.Name)
    }
    
    if b.RoundInterval < 1 {
        return fmt.Errorf("buffId %d (%s) has RoundInterval < 1. Is %s valid?", 
                         b.BuffId, b.Name, b.TriggerRate)
    }
    
    return nil
}
```

## Stat Modification System

### Individual Buff Stat Modifications
```go
// Get stat modification from single buff
func (b *Buff) StatMod(statName string) int {
    if b.Expired() {
        return 0
    }
    
    if buffInfo := GetBuffSpec(b.BuffId); buffInfo != nil {
        return buffInfo.StatMods.Get(statName)
    }
    
    return 0
}
```

### Cumulative Stat Modifications
```go
// Calculate total stat modification from all active buffs
func (bs *Buffs) StatMod(statName string) int {
    buffAmt := 0
    for _, b := range bs.List {
        buffAmt += b.StatMod(statName)
    }
    return buffAmt
}
```

### Buff Value Calculation
```go
// Calculate relative power/value of a buff for balance
func (b *BuffSpec) GetValue() int {
    val := 0
    
    // Sum absolute values of all stat modifications
    for _, v := range b.StatMods {
        val += int(math.Abs(float64(v)))
    }
    
    // Add value for frequency (more frequent = more valuable)
    freqVal := max(5-b.RoundInterval, 0)
    val += freqVal
    
    // Add value for flags (5 points per flag)
    val += len(b.Flags) * 5
    
    // Multiply by trigger count for total effect
    if b.TriggerCount > 0 {
        val *= b.TriggerCount
    }
    
    return val
}
```

## Buff Management Operations

### Adding Buffs
```go
// Add new buff or refresh existing buff
func (bs *Buffs) AddBuff(buffId int, isPermanent bool) bool {
    if buffInfo := GetBuffSpec(buffId); buffInfo != nil {
        newBuff := Buff{
            BuffId:       buffInfo.BuffId,
            RoundCounter: 0,
            PermaBuff:    false,
            TriggersLeft: buffInfo.TriggerCount,
        }
        
        // Handle permanent buffs (from equipment/race)
        if isPermanent {
            newBuff.TriggersLeft = TriggersLeftUnlimited
            newBuff.PermaBuff = true
        }
        
        // Check if buff already exists
        if idx, ok := bs.buffIds[buffId]; ok {
            // Refresh existing buff
            bs.List[idx].TriggersLeft = newBuff.TriggersLeft
            bs.List[idx].PermaBuff = newBuff.PermaBuff
            return true
        }
        
        // Add new buff
        bs.List = append(bs.List, &newBuff)
        listIndex := len(bs.List) - 1
        bs.buffIds[buffId] = listIndex
        
        // Update flag indexes
        for _, flag := range buffInfo.Flags {
            if _, ok := bs.buffFlags[flag]; !ok {
                bs.buffFlags[flag] = []int{}
            }
            bs.buffFlags[flag] = append(bs.buffFlags[flag], listIndex)
        }
        
        return true
    }
    
    return false
}
```

### Removing Buffs
```go
// Remove specific buff by ID
func (bs *Buffs) RemoveBuff(buffId int) bool {
    if index, ok := bs.buffIds[buffId]; ok {
        bs.List[index].TriggersLeft = TriggersLeftExpired
        return true
    }
    return false
}

// Mark buff as started (no longer waiting for start event)
func (bs *Buffs) Started(buffId int) {
    if idx, ok := bs.buffIds[buffId]; ok {
        bs.List[idx].OnStartWaiting = false
    }
}
```

### Pruning Expired Buffs
```go
// Remove all expired buffs and rebuild indexes
func (bs *Buffs) Prune() (prunedBuffs []*Buff) {
    if len(bs.List) == 0 {
        return prunedBuffs
    }
    
    didPrune := false
    
    // Iterate backwards to safely remove items
    for i := len(bs.List) - 1; i >= 0; i-- {
        b := bs.List[i]
        prune := false
        
        buffInfo := GetBuffSpec(b.BuffId)
        if buffInfo == nil || b.Expired() {
            prune = true
        }
        
        if prune {
            prunedBuffs = append(prunedBuffs, b)
            bs.List = append(bs.List[:i], bs.List[i+1:]...)
            didPrune = true
        }
    }
    
    // Rebuild lookup indexes if any buffs were pruned
    if didPrune {
        bs.Validate(true)
    }
    
    return prunedBuffs
}
```

## Collection Validation and Indexing

### Index Management
```go
// Validate and rebuild internal indexes
func (bs *Buffs) Validate(forceRebuild ...bool) {
    if bs.buffFlags == nil {
        bs.buffFlags = make(map[Flag][]int)
    }
    if bs.buffIds == nil {
        bs.buffIds = make(map[int]int)
    }
    
    // Rebuild if size mismatch or forced
    if (len(bs.List) != len(bs.buffIds)) || (len(forceRebuild) > 0 && forceRebuild[0]) {
        bs.buffIds = make(map[int]int)
        bs.buffFlags = make(map[Flag][]int)
        
        // Rebuild all indexes
        for idx, b := range bs.List {
            bs.buffIds[b.BuffId] = idx
            
            bSpec := GetBuffSpec(b.BuffId)
            if bSpec == nil {
                mudlog.Warn("buffs.Validate()", "buffId", b.BuffId, "error", "invalid buffId")
                continue
            }
            
            // Index all flags for this buff
            for _, flag := range bSpec.Flags {
                if _, ok := bs.buffFlags[flag]; !ok {
                    bs.buffFlags[flag] = []int{}
                }
                bs.buffFlags[flag] = append(bs.buffFlags[flag], idx)
            }
        }
    }
}
```

### Query Operations
```go
// Check if specific buff exists
func (bs *Buffs) HasBuff(buffId int) bool {
    if _, ok := bs.buffIds[buffId]; ok {
        return true
    }
    return false
}

// Get remaining triggers for buff
func (bs *Buffs) TriggersLeft(buffId int) int {
    if idx, ok := bs.buffIds[buffId]; ok {
        return bs.List[idx].TriggersLeft
    }
    return 0
}

// Get all active buffs (optionally filtered by ID)
func (bs *Buffs) GetBuffs(buffId ...int) []*Buff {
    retBuffs := []*Buff{}
    for _, b := range bs.List {
        if !b.Expired() {
            if len(buffId) > 0 {
                // Filter by specific buff IDs
                for _, id := range buffId {
                    if b.BuffId == id {
                        retBuffs = append(retBuffs, b)
                        break
                    }
                }
            } else {
                // Return all active buffs
                retBuffs = append(retBuffs, b)
            }
        }
    }
    return retBuffs
}
```

## Display and Visibility
```go
// Get visible name and description (handles secret buffs)
func (b *BuffSpec) VisibleNameDesc() (name, description string) {
    if b.Secret {
        return "Mysterious Affliction", "Unknown"
    }
    return b.Name, b.Description
}

// Get buff display name
func (bs *Buff) Name() string {
    if sp := GetBuffSpec(bs.BuffId); sp != nil {
        return sp.Name
    }
    return ""
}
```

## Data Management and Search

### Buff Discovery
```go
// Search buffs by name or description
func SearchBuffs(searchTerm string) []int {
    searchTerm = strings.TrimSpace(strings.ToLower(searchTerm))
    results := make([]int, 0, 2)
    
    for _, buff := range buffs {
        if strings.Contains(strings.ToLower(buff.Name), searchTerm) ||
           strings.Contains(strings.ToLower(buff.Description), searchTerm) {
            results = append(results, buff.BuffId)
        }
    }
    
    return results
}

// Get all available buff IDs
func GetAllBuffIds() []int {
    results := make([]int, 0, len(buffs))
    for _, buff := range buffs {
        results = append(results, buff.BuffId)
    }
    return results
}
```

### File Management
```go
// Generate filename for buff specification
func (b *BuffSpec) Filename() string {
    filename := util.ConvertForFilename(b.Name)
    return fmt.Sprintf("%d-%s.yaml", b.BuffId, filename)
}

// Load all buff specifications from files
func LoadDataFiles() {
    start := time.Now()
    
    tmpBuffs, err := fileloader.LoadAllFlatFiles[int, *BuffSpec](
        string(configs.GetFilePathsConfig().DataFiles) + "/buffs"
    )
    if err != nil {
        panic(err)
    }
    
    buffs = tmpBuffs
    
    mudlog.Info("buffSpec.LoadDataFiles()", 
        "loadedCount", len(buffs), 
        "Time Taken", time.Since(start))
}
```

## Integration Patterns

### Character System Integration
```go
// Buffs integrate with character stats and behavior
- character.Buffs.StatMod("strength")     // Stat modifications
- character.Buffs.HasFlag(buffs.NoCombat) // Behavioral restrictions
- character.Buffs.Trigger()               // Round-based processing
- character.Buffs.Prune()                 // Cleanup expired buffs
```

### Combat System Integration
```go
// Combat checks buff flags for behavior modification
if sourceChar.HasBuffFlag(buffs.Accuracy) {
    critChance *= 2 // Double crit chance
}

if targetChar.HasBuffFlag(buffs.Blink) {
    critChance /= 2 // Half crit chance against blink
}

if !sourceChar.HasBuffFlag(buffs.Hidden) {
    // Send visible combat messages
}
```

### Event System Integration
```go
// Buffs trigger events for start, effect, and end
events.AddToQueue(events.Buff{
    MobInstanceId: mobInstanceId,
    BuffId:        buffId,
    Source:        source,
})
```

## Usage Examples

### Basic Buff Management
```go
// Create new buff collection
buffs := buffs.New()

// Add temporary buff
buffs.AddBuff(poisonBuffId, false)

// Add permanent buff (from equipment)
buffs.AddBuff(strengthBuffId, true)

// Check for specific behavior
if buffs.HasFlag(buffs.NoCombat, false) {
    user.SendText("You cannot engage in combat right now.")
    return
}

// Process round-based triggers
triggeredBuffs := buffs.Trigger()
for _, buff := range triggeredBuffs {
    // Handle buff effects
    processBuff(buff)
}

// Clean up expired buffs
prunedBuffs := buffs.Prune()
for _, buff := range prunedBuffs {
    // Send buff expiration messages
    notifyBuffExpired(buff)
}
```

### Stat Modification Usage
```go
// Calculate total stat bonuses from all buffs
strengthBonus := character.Buffs.StatMod("strength")
dexBonus := character.Buffs.StatMod("dexterity")
healthBonus := character.Buffs.StatMod("health")

// Apply to character stats
character.Stats.Strength.ValueAdj += strengthBonus
character.Stats.Dexterity.ValueAdj += dexBonus
character.HealthMax.Value += healthBonus
```

### Flag-Based Behavior Control
```go
// Check movement restrictions
if character.Buffs.HasFlag(buffs.NoMovement, false) {
    user.SendText("You are unable to move.")
    return
}

// Check combat restrictions with expiration
if character.Buffs.HasFlag(buffs.CancelOnAction, true) {
    user.SendText("Your concentration is broken!")
    // Buff automatically expired by HasFlag call
}

// Environmental interactions
if character.Buffs.HasFlag(buffs.EmitsLight, false) {
    room.LightLevel += 1 // Provide illumination
}
```

## Dependencies

- `internal/statmods` - Stat modification system integration
- `internal/configs` - Configuration management for file paths and timing
- `internal/gametime` - Game time system for trigger rate calculations
- `internal/fileloader` - YAML file loading and validation system
- `internal/util` - Utility functions for file operations and validation
- `internal/mudlog` - Logging system for debugging and monitoring

This comprehensive buffs system provides sophisticated temporary status
effects with precise timing control, behavioral modification, stat
integration, and config-driven effect behaviors.

---

## DOGMud chunk-4d buffs

Two new buffs added in chunk 4d (T9 + T10). Neither is a regen potion or
combat potion — they are combat consequence buffs applied by the submission
outcome resolver (`internal/combat/submission_outcome.go`).

| ID | Name | Duration | Source | Effect |
|----|------|----------|--------|--------|
| 83 | Broken Limb | ~3600 rounds (~1 hr play) | Cripple submission outcome via `applyBrokenLimbBuff` | Reduces combat effectiveness for the afflicted limb's weapon role; persists across respawn; cannot be dispelled early by normal means |
| 84 | Submission Stunned | 1 round | Crit submission tier (mercy policy only) via `applyStunnedBuff` | Brief combat stagger; auto-clears at the end of the following round |

**Buff 83 (Broken Limb)** is the first persistent, non-dispellable
combat debuff players commonly encounter. Triggered by a cripple-policy
submission where the sub type targets a joint (armbar, kimura). Choke-class
subs (RNC, Triangle, Anaconda, Guillotine) do NOT trigger buff 83 because
they have no body-part target — the policy degrades to subdue instead.

**Buff 84 (Submission Stunned)** is a 1-round stagger applied to the
recipient when a mercy-policy submission lands a crit roll
(`SubTierCrit`). Only fires on mercy policy because subdue/cripple/lethal
send the defender through the death cascade and the buff would be a no-op.

See `internal/combat/context.md` "Submission System" for the full
context in which these buffs are applied.

---

## DOGMud chunk-3.3 buffs (Sleeping)

### New flags

| Flag | String value | Purpose |
|------|-------------|---------|
| `Sleeping` | `"sleeping"` | Bearer is asleep — gates regen boost, first-hit-crit, room rendering. Chunk 3.3. |
| `CancelOnDamage` | `"cancel-on-damage"` | Buff cancels when any damage is applied to bearer. Wired in damage pipeline. Chunk 3.3. |

### New buff

| ID | Name | Duration | Source | Effect |
|----|------|----------|--------|--------|
| 15 | Sleeping | Unlimited (until woken) | `actions.Sleep`, sleep user command, sleep mob command, schedule executor | Applies `Sleeping` + `CancelOnDamage` + `NoCombat` + `NoMovement` flags; triggers `SleepRegenMultiplier` (5×) regen; forces first-hit-crit on all attackers for the round the buff is active. Cancelled by damage, failed steal, shout-in-room, light source entering room, `stand`, or schedule segment end. |

### Usage pattern

```go
// Check if character is asleep
if c.HasBuffFlag(buffs.Sleeping) { ... }

// Wake a sleeper (central hook)
mobs.OnSleeperWoken(c)

// Cancel all sleeping buffs (schedule exit path)
c.CancelBuffsWithFlag(buffs.Sleeping)
```